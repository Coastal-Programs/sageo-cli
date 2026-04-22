package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newTestProject creates a fresh .sageo project in a temp dir and returns the
// directory and the created state.
func newTestProject(t *testing.T) (string, *State) {
	t.Helper()
	dir := t.TempDir()
	st, err := Init(dir, "https://example.com")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	return dir, st
}

func TestCreateSnapshot_RoundTrip(t *testing.T) {
	dir, st := newTestProject(t)

	st.Score = 82.5
	st.PagesCrawled = 12

	meta := SnapshotMeta{
		StartedAt:    time.Now().UTC().Add(-time.Minute),
		CompletedAt:  time.Now().UTC(),
		StagesRun:    []string{"crawl", "audit"},
		TotalCostUSD: 0.05,
		Outcome:      "success",
	}
	snap, err := CreateSnapshot(dir, st, meta, []byte("<html>report</html>"))
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Files present
	for _, f := range []string{"state.json", "recommendations.json", "report.html", "metadata.json"} {
		if _, err := os.Stat(filepath.Join(snap.Dir, f)); err != nil {
			t.Errorf("missing %s: %v", f, err)
		}
	}

	// Top-level state.json is a copy of the snapshot's state.
	top, err := os.ReadFile(Path(dir))
	if err != nil {
		t.Fatalf("read top state: %v", err)
	}
	snapState, err := os.ReadFile(filepath.Join(snap.Dir, "state.json"))
	if err != nil {
		t.Fatalf("read snap state: %v", err)
	}
	if string(top) != string(snapState) {
		t.Error("top-level state.json is not a copy of the snapshot state.json")
	}

	// index.json lists it.
	snaps, err := ListSnapshots(dir)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(snaps) != 1 {
		t.Fatalf("expected 1 snapshot, got %d", len(snaps))
	}
	if snaps[0].Meta.TotalCostUSD != 0.05 {
		t.Errorf("meta cost mismatch: %v", snaps[0].Meta.TotalCostUSD)
	}
}

func TestCreateSnapshot_TwoDistinctRuns(t *testing.T) {
	dir, st := newTestProject(t)

	_, err := CreateSnapshot(dir, st, SnapshotMeta{
		CompletedAt: time.Now().UTC(),
		Outcome:     "success",
	}, nil)
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	// Ensure second snapshot gets a distinct timestamp or suffix.
	_, err = CreateSnapshot(dir, st, SnapshotMeta{
		CompletedAt: time.Now().UTC(),
		Outcome:     "success",
	}, nil)
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	snaps, err := ListSnapshots(dir)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 snapshots, got %d", len(snaps))
	}
	if snaps[0].Dir == snaps[1].Dir {
		t.Fatal("snapshots share a directory")
	}
}

func TestLoadSnapshot_LatestAndPrevious(t *testing.T) {
	dir, st := newTestProject(t)

	t1 := time.Now().UTC().Add(-2 * time.Hour)
	t2 := time.Now().UTC().Add(-1 * time.Hour)

	if _, err := CreateSnapshot(dir, st, SnapshotMeta{CompletedAt: t1, Outcome: "success"}, nil); err != nil {
		t.Fatalf("s1: %v", err)
	}
	if _, err := CreateSnapshot(dir, st, SnapshotMeta{CompletedAt: t2, Outcome: "success"}, nil); err != nil {
		t.Fatalf("s2: %v", err)
	}

	latest, err := LoadSnapshot(dir, "latest")
	if err != nil {
		t.Fatalf("latest: %v", err)
	}
	prev, err := LoadSnapshot(dir, "previous")
	if err != nil {
		t.Fatalf("previous: %v", err)
	}
	if !latest.Timestamp.After(prev.Timestamp) {
		t.Errorf("latest (%s) should be after previous (%s)", latest.Timestamp, prev.Timestamp)
	}

	// Prefix match.
	prefix := FormatSnapshotTimestamp(t2)[:10] // YYYY-MM-DD
	_, err = LoadSnapshot(dir, prefix)
	// With two snapshots on the same day the prefix is ambiguous — that's
	// fine, just assert we get a clean error either way.
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "ambiguous") && !strings.Contains(err.Error(), "matching") {
		t.Errorf("unexpected prefix error: %v", err)
	}
}

func TestLoadSnapshot_PreviousWithOnlyOne(t *testing.T) {
	dir, st := newTestProject(t)
	if _, err := CreateSnapshot(dir, st, SnapshotMeta{CompletedAt: time.Now().UTC()}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSnapshot(dir, "previous"); err == nil {
		t.Error("expected error when requesting previous with only one snapshot")
	}
}

func TestPruneSnapshots_KeepN(t *testing.T) {
	dir, st := newTestProject(t)

	base := time.Now().UTC().Add(-10 * time.Hour)
	for i := 0; i < 5; i++ {
		_, err := CreateSnapshot(dir, st, SnapshotMeta{
			CompletedAt: base.Add(time.Duration(i) * time.Hour),
			Outcome:     "success",
		}, nil)
		if err != nil {
			t.Fatalf("snap %d: %v", i, err)
		}
	}

	removed, err := PruneSnapshots(dir, 2, 0)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(removed) != 3 {
		t.Errorf("expected 3 removed, got %d", len(removed))
	}

	snaps, _ := ListSnapshots(dir)
	if len(snaps) != 2 {
		t.Errorf("expected 2 remaining, got %d", len(snaps))
	}
}

func TestPruneSnapshots_KeepWithin(t *testing.T) {
	dir, st := newTestProject(t)

	// Snapshot 1: very old (20 days ago).
	_, err := CreateSnapshot(dir, st, SnapshotMeta{
		CompletedAt: time.Now().UTC().Add(-20 * 24 * time.Hour),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Snapshot 2: recent (1 hour ago).
	_, err = CreateSnapshot(dir, st, SnapshotMeta{
		CompletedAt: time.Now().UTC().Add(-time.Hour),
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Keep nothing by count; keep anything within 7 days.
	removed, err := PruneSnapshots(dir, 0, 7*24*time.Hour)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if len(removed) != 1 {
		t.Errorf("expected 1 removed (the 20-day-old one), got %d", len(removed))
	}
}

func TestPruneSnapshots_CombinedKeepsUnion(t *testing.T) {
	dir, st := newTestProject(t)

	// 5 snapshots over 30 days.
	for i := 0; i < 5; i++ {
		age := time.Duration(30-6*i) * 24 * time.Hour
		if _, err := CreateSnapshot(dir, st, SnapshotMeta{
			CompletedAt: time.Now().UTC().Add(-age),
		}, nil); err != nil {
			t.Fatal(err)
		}
	}

	// keep-N=1 AND within=15d → union. Newest is ~0 days, also within.
	// Plus one within 15d threshold. Count distinct kept >= 1.
	removed, err := PruneSnapshots(dir, 1, 15*24*time.Hour)
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	remaining, _ := ListSnapshots(dir)
	if len(removed)+len(remaining) != 5 {
		t.Errorf("removed+remaining != 5: %d + %d", len(removed), len(remaining))
	}
	if len(remaining) < 1 {
		t.Errorf("expected at least 1 remaining")
	}
}

func TestAtomicity_TmpDirDoesNotPollute(t *testing.T) {
	dir, st := newTestProject(t)

	if _, err := CreateSnapshot(dir, st, SnapshotMeta{CompletedAt: time.Now().UTC()}, nil); err != nil {
		t.Fatal(err)
	}

	// Simulate a crash: drop a stale .tmp directory alongside.
	root := snapshotsRoot(dir)
	stale := filepath.Join(root, "2099-01-01T00-00-00Z.tmp")
	if err := os.MkdirAll(stale, 0o755); err != nil {
		t.Fatal(err)
	}
	// A subsequent list should ignore .tmp.
	snaps, err := ListSnapshots(dir)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, s := range snaps {
		if strings.HasSuffix(s.Dir, ".tmp") {
			t.Errorf("tmp directory leaked into list: %s", s.Dir)
		}
	}
}

func TestLegacyMigration(t *testing.T) {
	dir := t.TempDir()
	sageoDir := filepath.Join(dir, DirName)
	if err := os.MkdirAll(sageoDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a legacy state.json (no snapshots/ dir present).
	st := &State{
		Site:        "https://legacy.example",
		Initialized: time.Now().UTC().Format(time.RFC3339),
		LastCrawl:   time.Now().UTC().Format(time.RFC3339),
		Score:       77,
		Findings: []Finding{{
			Rule: "example", URL: "https://legacy.example/",
			Verdict: "warn", Why: "test",
		}},
	}
	data, _ := json.MarshalIndent(st, "", "  ")
	if err := os.WriteFile(Path(dir), data, 0o644); err != nil {
		t.Fatal(err)
	}
	// Back-date the file so the migrated snapshot is distinguishable
	// from the new one created below.
	old := time.Now().UTC().Add(-48 * time.Hour)
	if err := os.Chtimes(Path(dir), old, old); err != nil {
		t.Fatal(err)
	}

	// Now a first real run: CreateSnapshot should migrate then create a new one.
	loaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Score = 90
	_, err = CreateSnapshot(dir, loaded, SnapshotMeta{
		CompletedAt: time.Now().UTC(),
		Outcome:     "success",
	}, nil)
	if err != nil {
		t.Fatal(err)
	}

	snaps, err := ListSnapshots(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 2 {
		t.Fatalf("expected 2 (migrated + new), got %d", len(snaps))
	}

	// The oldest snapshot should be marked Migrated.
	oldest := snaps[len(snaps)-1]
	if !oldest.Meta.Migrated {
		t.Errorf("expected oldest snapshot to be flagged migrated, got %+v", oldest.Meta)
	}
}

func TestListSnapshots_Empty(t *testing.T) {
	dir := t.TempDir()
	snaps, err := ListSnapshots(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(snaps) != 0 {
		t.Errorf("expected 0 snapshots on fresh dir, got %d", len(snaps))
	}
}

func TestPrune_UpdatesIndexConsistently(t *testing.T) {
	dir, st := newTestProject(t)
	for i := 0; i < 3; i++ {
		if _, err := CreateSnapshot(dir, st, SnapshotMeta{
			CompletedAt: time.Now().UTC().Add(time.Duration(i) * time.Second),
		}, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := PruneSnapshots(dir, 1, 0); err != nil {
		t.Fatal(err)
	}
	idx, err := loadIndex(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Snapshots) != 1 {
		t.Errorf("index should have 1 entry, got %d", len(idx.Snapshots))
	}
}
