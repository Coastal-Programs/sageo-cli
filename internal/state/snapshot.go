// Frozen per-run snapshots of sageo state.
//
// Every `sageo run` invocation writes a timestamped directory under
// .sageo/snapshots/<ts>/ containing a frozen copy of state.json, the
// recommendations for that run, the rendered HTML report, and a
// metadata.json describing the invocation. The previous run is never
// overwritten — snapshots are the foundation for historical comparison,
// calibration, and longitudinal analysis.
//
// .sageo/state.json at the top of the project is always a COPY of the
// latest snapshot's state.json (Option A from the design doc). Copying
// rather than symlinking keeps the layout portable: zip archives,
// Windows filesystems, and rsync all handle it the same. The 2× disk
// usage per run is negligible for JSON at this size.
//
// Atomicity: each snapshot is written to .sageo/snapshots/<ts>.tmp/
// first, then renamed to its final directory (atomic on POSIX). The
// index.json pointer and the top-level state.json copy are written with
// the same write-temp-then-rename discipline. If any step fails the
// previous snapshot stays intact.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/version"
)

const (
	// SnapshotsDirName is the directory under .sageo/ that holds frozen runs.
	SnapshotsDirName = "snapshots"
	// SnapshotIndexFile is the ordered index of snapshot directories.
	SnapshotIndexFile = "index.json"

	snapshotStateFile  = "state.json"
	snapshotRecsFile   = "recommendations.json"
	snapshotReportFile = "report.html"
	snapshotMetaFile   = "metadata.json"

	// snapshotTimestampFormat is ISO-8601 with colons swapped for dashes so
	// it is safe on all filesystems. Lexicographic sort order matches
	// chronological order.
	snapshotTimestampFormat = "2006-01-02T15-04-05Z"
)

// SnapshotMeta captures the invocation context of a single `sageo run`.
type SnapshotMeta struct {
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
	StagesRun       []string  `json:"stages_run,omitempty"`
	TotalCostUSD    float64   `json:"total_cost_usd"`
	PipelineVersion string    `json:"pipeline_version"`
	GitCommit       string    `json:"git_commit,omitempty"`
	Outcome         string    `json:"outcome,omitempty"`
	FailedStage     string    `json:"failed_stage,omitempty"`
	Error           string    `json:"error,omitempty"`
	// Migrated marks a snapshot synthesised from a pre-snapshot state.json
	// during the one-time legacy migration.
	Migrated bool `json:"migrated,omitempty"`
}

// Snapshot is a single frozen run on disk.
type Snapshot struct {
	Timestamp time.Time    `json:"timestamp"`
	Dir       string       `json:"dir"`
	Meta      SnapshotMeta `json:"meta"`

	// state is loaded lazily by State().
	state *State
}

// snapshotIndex is the on-disk index.json format.
type snapshotIndex struct {
	Snapshots []snapshotIndexEntry `json:"snapshots"`
}

type snapshotIndexEntry struct {
	Timestamp string       `json:"timestamp"`
	Dir       string       `json:"dir"` // relative to .sageo/snapshots
	Meta      SnapshotMeta `json:"meta"`
}

// sageoPath returns baseDir/.sageo.
func sageoPath(baseDir string) string {
	return filepath.Join(baseDir, DirName)
}

func snapshotsRoot(baseDir string) string {
	return filepath.Join(sageoPath(baseDir), SnapshotsDirName)
}

func indexPath(baseDir string) string {
	return filepath.Join(sageoPath(baseDir), SnapshotIndexFile)
}

// FormatSnapshotTimestamp formats t for use as a snapshot directory name.
func FormatSnapshotTimestamp(t time.Time) string {
	return t.UTC().Format(snapshotTimestampFormat)
}

// parseSnapshotTimestamp parses a snapshot directory name back to time.
func parseSnapshotTimestamp(name string) (time.Time, error) {
	return time.Parse(snapshotTimestampFormat, name)
}

// CreateSnapshot writes the current state + recommendations + report into a
// new timestamped directory under baseDir/.sageo/snapshots/ and updates the
// top-level state.json and index.json pointers atomically. It also runs the
// legacy migration on first use (see migrateLegacyState).
func CreateSnapshot(baseDir string, st *State, meta SnapshotMeta, reportHTML []byte) (*Snapshot, error) {
	if st == nil {
		return nil, errors.New("snapshot: state is nil")
	}

	if meta.PipelineVersion == "" {
		meta.PipelineVersion = version.Current()
	}
	if meta.CompletedAt.IsZero() {
		meta.CompletedAt = time.Now().UTC()
	}
	if meta.StartedAt.IsZero() {
		meta.StartedAt = meta.CompletedAt
	}

	if err := migrateLegacyState(baseDir); err != nil {
		return nil, fmt.Errorf("snapshot: legacy migration: %w", err)
	}

	ts := meta.CompletedAt.UTC()
	name := FormatSnapshotTimestamp(ts)

	root := snapshotsRoot(baseDir)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("snapshot: mkdir snapshots: %w", err)
	}

	// If a snapshot with this timestamp already exists (two runs in the
	// same second) disambiguate by appending a counter.
	finalDir := filepath.Join(root, name)
	if _, err := os.Stat(finalDir); err == nil {
		for i := 1; i < 1000; i++ {
			candidate := fmt.Sprintf("%s-%d", name, i)
			if _, err := os.Stat(filepath.Join(root, candidate)); os.IsNotExist(err) {
				name = candidate
				finalDir = filepath.Join(root, name)
				break
			}
		}
	}

	tmpDir := finalDir + ".tmp"
	// Clean up any stale tmp from a prior crash.
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return nil, fmt.Errorf("snapshot: mkdir tmp: %w", err)
	}

	// Write state.json.
	stateBytes, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("snapshot: marshal state: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, snapshotStateFile), stateBytes, 0o644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("snapshot: write state: %w", err)
	}

	// Write recommendations.json — kept separate so historical diffs
	// don't have to parse the full state.
	recsBytes, err := json.MarshalIndent(st.Recommendations, "", "  ")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("snapshot: marshal recs: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, snapshotRecsFile), recsBytes, 0o644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("snapshot: write recs: %w", err)
	}

	// Write report.html (optional — may be nil for e.g. failed runs).
	if len(reportHTML) > 0 {
		if err := os.WriteFile(filepath.Join(tmpDir, snapshotReportFile), reportHTML, 0o644); err != nil {
			_ = os.RemoveAll(tmpDir)
			return nil, fmt.Errorf("snapshot: write report: %w", err)
		}
	}

	// Write metadata.json.
	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("snapshot: marshal meta: %w", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, snapshotMetaFile), metaBytes, 0o644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("snapshot: write meta: %w", err)
	}

	// Atomic rename tmp → final.
	if err := os.Rename(tmpDir, finalDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return nil, fmt.Errorf("snapshot: rename: %w", err)
	}

	// Update index.json (append entry, write-tmp-then-rename).
	if err := appendToIndex(baseDir, snapshotIndexEntry{
		Timestamp: name,
		Dir:       name,
		Meta:      meta,
	}); err != nil {
		return nil, fmt.Errorf("snapshot: update index: %w", err)
	}

	// Update .sageo/state.json to be a COPY of the new snapshot's state.
	if err := copyFileAtomic(
		filepath.Join(finalDir, snapshotStateFile),
		Path(baseDir),
	); err != nil {
		return nil, fmt.Errorf("snapshot: update latest state.json: %w", err)
	}

	snap := &Snapshot{
		Timestamp: ts,
		Dir:       finalDir,
		Meta:      meta,
		state:     st,
	}
	return snap, nil
}

// copyFileAtomic copies src → dst via a sibling temp file + rename.
func copyFileAtomic(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(dst), filepath.Base(dst)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := io.Copy(tmp, in); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, dst)
}

// appendToIndex loads (or creates) index.json, appends entry, and rewrites atomically.
func appendToIndex(baseDir string, entry snapshotIndexEntry) error {
	idx, err := loadIndex(baseDir)
	if err != nil {
		return err
	}
	idx.Snapshots = append(idx.Snapshots, entry)
	return writeIndex(baseDir, idx)
}

func loadIndex(baseDir string) (*snapshotIndex, error) {
	data, err := os.ReadFile(indexPath(baseDir))
	if os.IsNotExist(err) {
		return &snapshotIndex{}, nil
	}
	if err != nil {
		return nil, err
	}
	var idx snapshotIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("parse index.json: %w", err)
	}
	return &idx, nil
}

func writeIndex(baseDir string, idx *snapshotIndex) error {
	if err := os.MkdirAll(sageoPath(baseDir), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	path := indexPath(baseDir)
	tmp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, path)
}

// ListSnapshots returns every snapshot in baseDir sorted newest-first.
// It prefers index.json but falls back to a directory scan if the index is
// missing or corrupt (e.g. after manual edits).
func ListSnapshots(baseDir string) ([]Snapshot, error) {
	root := snapshotsRoot(baseDir)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	}

	idx, err := loadIndex(baseDir)
	if err != nil || idx == nil || len(idx.Snapshots) == 0 {
		// Fallback: scan the directory.
		return scanSnapshots(baseDir)
	}

	out := make([]Snapshot, 0, len(idx.Snapshots))
	for _, e := range idx.Snapshots {
		dir := filepath.Join(root, e.Dir)
		if _, err := os.Stat(dir); err != nil {
			continue // drop stale index entries
		}
		ts, _ := parseSnapshotTimestamp(e.Timestamp)
		out = append(out, Snapshot{
			Timestamp: ts,
			Dir:       dir,
			Meta:      e.Meta,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	return out, nil
}

// scanSnapshots discovers snapshots by scanning the filesystem.
func scanSnapshots(baseDir string) ([]Snapshot, error) {
	root := snapshotsRoot(baseDir)
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Snapshot
	for _, e := range entries {
		if !e.IsDir() || strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		ts, err := parseSnapshotTimestamp(strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		if err != nil {
			// Try the whole name.
			ts, err = parseSnapshotTimestamp(e.Name())
			if err != nil {
				continue
			}
		}
		dir := filepath.Join(root, e.Name())
		meta, _ := readSnapshotMeta(dir)
		out = append(out, Snapshot{
			Timestamp: ts,
			Dir:       dir,
			Meta:      meta,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.After(out[j].Timestamp)
	})
	return out, nil
}

func readSnapshotMeta(dir string) (SnapshotMeta, error) {
	var meta SnapshotMeta
	data, err := os.ReadFile(filepath.Join(dir, snapshotMetaFile))
	if err != nil {
		return meta, err
	}
	return meta, json.Unmarshal(data, &meta)
}

// LoadSnapshot resolves a snapshot by reference. Supported references:
//   - "latest"   — newest snapshot
//   - "previous" — second newest snapshot
//   - <prefix>   — any timestamp-prefix match (e.g. "2026-04-22")
//   - full directory name
func LoadSnapshot(baseDir, ref string) (*Snapshot, error) {
	snaps, err := ListSnapshots(baseDir)
	if err != nil {
		return nil, err
	}
	if len(snaps) == 0 {
		return nil, fmt.Errorf("no snapshots found")
	}

	ref = strings.TrimSpace(ref)
	switch strings.ToLower(ref) {
	case "", "latest":
		s := snaps[0]
		if err := s.loadState(); err != nil {
			return nil, err
		}
		return &s, nil
	case "previous":
		if len(snaps) < 2 {
			return nil, fmt.Errorf("only one snapshot exists; no previous")
		}
		s := snaps[1]
		if err := s.loadState(); err != nil {
			return nil, err
		}
		return &s, nil
	}

	// Prefix/exact match on directory basename.
	var matches []Snapshot
	for _, s := range snaps {
		base := filepath.Base(s.Dir)
		if base == ref || strings.HasPrefix(base, ref) {
			matches = append(matches, s)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no snapshot matching %q", ref)
	case 1:
		s := matches[0]
		if err := s.loadState(); err != nil {
			return nil, err
		}
		return &s, nil
	default:
		names := make([]string, 0, len(matches))
		for _, m := range matches {
			names = append(names, filepath.Base(m.Dir))
		}
		return nil, fmt.Errorf("ambiguous reference %q matches %d snapshots: %s",
			ref, len(matches), strings.Join(names, ", "))
	}
}

// State returns the frozen state for this snapshot, loading from disk on first access.
func (s *Snapshot) State() (*State, error) {
	if s.state != nil {
		return s.state, nil
	}
	if err := s.loadState(); err != nil {
		return nil, err
	}
	return s.state, nil
}

func (s *Snapshot) loadState() error {
	data, err := os.ReadFile(filepath.Join(s.Dir, snapshotStateFile))
	if err != nil {
		return fmt.Errorf("read snapshot state: %w", err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return fmt.Errorf("parse snapshot state: %w", err)
	}
	s.state = &st
	return nil
}

// PruneSnapshots enforces retention: keep the last keepLastN snapshots plus
// anything within keepWithin of now. Returns the directory paths that were
// removed. A zero or negative keepLastN disables the count rule. A zero
// keepWithin disables the age rule. If both are disabled, nothing is pruned.
func PruneSnapshots(baseDir string, keepLastN int, keepWithin time.Duration) ([]string, error) {
	snaps, err := ListSnapshots(baseDir)
	if err != nil {
		return nil, err
	}
	if len(snaps) == 0 {
		return nil, nil
	}
	if keepLastN <= 0 && keepWithin <= 0 {
		return nil, nil
	}

	now := time.Now().UTC()
	keep := make(map[string]bool, len(snaps))

	// Rule 1: keep newest keepLastN.
	for i, s := range snaps {
		if keepLastN > 0 && i < keepLastN {
			keep[s.Dir] = true
		}
		if keepWithin > 0 && now.Sub(s.Timestamp) <= keepWithin {
			keep[s.Dir] = true
		}
	}

	var removed []string
	kept := make([]snapshotIndexEntry, 0, len(snaps))
	for _, s := range snaps {
		if keep[s.Dir] {
			kept = append(kept, snapshotIndexEntry{
				Timestamp: filepath.Base(s.Dir),
				Dir:       filepath.Base(s.Dir),
				Meta:      s.Meta,
			})
			continue
		}
		if err := os.RemoveAll(s.Dir); err != nil {
			return removed, fmt.Errorf("remove %s: %w", s.Dir, err)
		}
		removed = append(removed, s.Dir)
	}

	// Rewrite index.json so it matches reality. Sort oldest-first in the
	// index (append order) for consistency with append behaviour.
	sort.Slice(kept, func(i, j int) bool {
		return kept[i].Timestamp < kept[j].Timestamp
	})
	if err := writeIndex(baseDir, &snapshotIndex{Snapshots: kept}); err != nil {
		return removed, err
	}
	return removed, nil
}

// migrateLegacyState detects a pre-snapshot install (state.json present, but
// no snapshots/ directory) and promotes the existing state.json into a
// synthetic snapshot so history starts from the first upgrade. Subsequent
// runs are no-ops.
func migrateLegacyState(baseDir string) error {
	statePath := Path(baseDir)
	root := snapshotsRoot(baseDir)

	if _, err := os.Stat(statePath); err != nil {
		return nil // no pre-existing state, nothing to migrate
	}
	if _, err := os.Stat(root); err == nil {
		return nil // already migrated
	} else if !os.IsNotExist(err) {
		return err
	}

	// Heuristic: only migrate state files that contain actual prior-run
	// data. A freshly `sageo init`ed state has no crawl, findings, or
	// recommendations and doesn't need a synthetic legacy snapshot.
	if data, err := os.ReadFile(statePath); err == nil {
		var probe State
		if err := json.Unmarshal(data, &probe); err == nil {
			if probe.LastCrawl == "" && len(probe.Findings) == 0 &&
				len(probe.Recommendations) == 0 && len(probe.PipelineRuns) == 0 {
				return nil
			}
		}
	}

	// Use the state.json mtime as the snapshot timestamp so it's
	// chronologically correct relative to later runs.
	info, err := os.Stat(statePath)
	if err != nil {
		return err
	}
	ts := info.ModTime().UTC()

	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	name := FormatSnapshotTimestamp(ts)
	finalDir := filepath.Join(root, name)
	if _, err := os.Stat(finalDir); err == nil {
		name = name + "-legacy"
		finalDir = filepath.Join(root, name)
	}
	tmpDir := finalDir + ".tmp"
	_ = os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		return err
	}

	// Copy (not move) the legacy state into the snapshot. Keeping the
	// top-level state.json in place preserves the invariant that it is
	// always a copy of the latest snapshot.
	if err := copyFileAtomic(statePath, filepath.Join(tmpDir, snapshotStateFile)); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}

	// Derive a recommendations.json from the legacy state.
	var recs []Recommendation
	if data, err := os.ReadFile(statePath); err == nil {
		var st State
		if err := json.Unmarshal(data, &st); err == nil {
			recs = st.Recommendations
		}
	}
	recsBytes, _ := json.MarshalIndent(recs, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, snapshotRecsFile), recsBytes, 0o644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}

	meta := SnapshotMeta{
		StartedAt:       ts,
		CompletedAt:     ts,
		PipelineVersion: version.Current(),
		Outcome:         "migrated",
		Migrated:        true,
	}
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(tmpDir, snapshotMetaFile), metaBytes, 0o644); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}

	if err := os.Rename(tmpDir, finalDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return err
	}

	if err := writeIndex(baseDir, &snapshotIndex{
		Snapshots: []snapshotIndexEntry{{
			Timestamp: name,
			Dir:       name,
			Meta:      meta,
		}},
	}); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr,
		"[snapshot] migrated pre-snapshot state.json → .sageo/snapshots/%s/\n",
		name)
	return nil
}
