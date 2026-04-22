package commands

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// captureStdout replaces os.Stdout during fn() and returns what was written.
// output.PrintSuccess writes to os.Stdout directly; cobra's SetOut does not
// reroute it, so tests must intercept at the OS level.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		done <- buf.String()
	}()
	fn()
	_ = w.Close()
	os.Stdout = orig
	return <-done
}

// chdir helper that restores the original cwd when the test ends.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func seedSnapshots(t *testing.T, dir string, n int) {
	t.Helper()
	st, err := state.Init(dir, "https://example.com")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Now().UTC().Add(-time.Duration(n) * time.Hour)
	for i := 0; i < n; i++ {
		_, err := state.CreateSnapshot(dir, st, state.SnapshotMeta{
			CompletedAt:  base.Add(time.Duration(i) * time.Hour),
			TotalCostUSD: 0.01 * float64(i+1),
			StagesRun:    []string{"crawl", "audit"},
			Outcome:      "success",
		}, nil)
		if err != nil {
			t.Fatalf("snap %d: %v", i, err)
		}
	}
}

func TestSnapshotsListCmd(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	seedSnapshots(t, dir, 3)

	format := "json"
	verbose := false
	cmd := NewSnapshotsCmd(&format, &verbose)
	cmd.SetArgs([]string{"list"})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	var env struct {
		Success  bool             `json:"success"`
		Data     []map[string]any `json:"data"`
		Metadata map[string]any   `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("decode: %v: %s", err, out)
	}
	if !env.Success {
		t.Errorf("expected success=true: %s", out)
	}
	if len(env.Data) != 3 {
		t.Errorf("expected 3 rows, got %d", len(env.Data))
	}
}

func TestSnapshotsPathCmd(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	seedSnapshots(t, dir, 2)

	format := "json"
	verbose := false
	cmd := NewSnapshotsCmd(&format, &verbose)
	cmd.SetArgs([]string{"path", "latest"})

	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	p := strings.TrimSpace(buf.String())
	if p == "" {
		t.Fatal("expected path output")
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("path does not exist on disk: %s: %v", p, err)
	}
}

func TestSnapshotsPruneDryRun(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	seedSnapshots(t, dir, 5)

	format := "json"
	verbose := false
	cmd := NewSnapshotsCmd(&format, &verbose)
	cmd.SetArgs([]string{"prune", "--keep", "2", "--within", "0"})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	snaps, _ := state.ListSnapshots(dir)
	if len(snaps) != 5 {
		t.Errorf("dry-run removed snapshots: %d remaining", len(snaps))
	}

	var env struct {
		Metadata map[string]any `json:"metadata"`
	}
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("decode: %v: %s", err, out)
	}
	if env.Metadata["dry_run"] != true {
		t.Errorf("expected dry_run=true in metadata: %+v", env.Metadata)
	}
	if env.Metadata["would_prune"] == nil {
		t.Errorf("expected would_prune in metadata: %+v", env.Metadata)
	}
}

func TestSnapshotsPruneConfirm(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	seedSnapshots(t, dir, 5)

	format := "json"
	verbose := false
	cmd := NewSnapshotsCmd(&format, &verbose)
	cmd.SetArgs([]string{"prune", "--keep", "2", "--within", "0", "--confirm"})

	captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	snaps, _ := state.ListSnapshots(dir)
	if len(snaps) != 2 {
		t.Errorf("expected 2 remaining after prune --confirm, got %d", len(snaps))
	}
}

func TestSnapshotsShowCmd(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)
	seedSnapshots(t, dir, 2)

	format := "json"
	verbose := false
	cmd := NewSnapshotsCmd(&format, &verbose)
	cmd.SetArgs([]string{"show", "latest"})

	out := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})
	if !strings.Contains(out, "timestamp") {
		t.Errorf("expected 'timestamp' in output: %s", out)
	}
}

// TestSnapshotsFlags verifies CLI flag wiring for the prune subcommand.
func TestSnapshotsFlags(t *testing.T) {
	format := "json"
	verbose := false
	cmd := NewSnapshotsCmd(&format, &verbose)

	prune, _, err := cmd.Find([]string{"prune"})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"keep", "within", "confirm"} {
		if prune.Flag(f) == nil {
			t.Errorf("prune missing --%s", f)
		}
	}
}
