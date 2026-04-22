package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestRunCmdFlags verifies the CLI flag surface for `sageo run` is wired
// correctly (flag names, defaults, required positional arg).
func TestRunCmdFlags(t *testing.T) {
	format := "json"
	verbose := false
	cmd := NewRunCmd(&format, &verbose)

	flags := []struct {
		name string
		want string // default string representation
	}{
		{"budget", "0"},
		{"max-pages", "100"},
		{"prompts", ""},
		{"dry-run", "false"},
		{"approve", "false"},
		{"resume", "false"},
	}
	for _, f := range flags {
		got := cmd.Flag(f.name)
		if got == nil {
			t.Errorf("flag --%s missing", f.name)
			continue
		}
		if got.DefValue != f.want {
			t.Errorf("flag --%s default=%q, want %q", f.name, got.DefValue, f.want)
		}
	}
	// skip / only should exist as slices.
	for _, name := range []string{"skip", "only"} {
		if cmd.Flag(name) == nil {
			t.Errorf("flag --%s missing", name)
		}
	}
}

// TestRunCmdRequiresURL verifies the command rejects missing positional arg.
func TestRunCmdRequiresURL(t *testing.T) {
	format := "json"
	verbose := false
	cmd := NewRunCmd(&format, &verbose)
	cmd.SetArgs([]string{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error on missing URL arg")
	}
}

// TestLoadRunPromptsFile reads a prompts file from disk and skips blanks/comments.
func TestLoadRunPromptsFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "prompts.txt")
	body := "first prompt\n\n# comment\nsecond prompt\n"
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	prompts, err := loadRunPrompts(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(prompts) != 2 {
		t.Fatalf("prompts = %v", prompts)
	}
	if prompts[0] != "first prompt" || prompts[1] != "second prompt" {
		t.Fatalf("prompts = %v", prompts)
	}
}

// TestLoadRunPromptsMissing returns nil without error when path is blank.
func TestLoadRunPromptsMissing(t *testing.T) {
	p, err := loadRunPrompts("")
	if err != nil || p != nil {
		t.Fatalf("loadRunPrompts(\"\") = %v, %v", p, err)
	}
}

// TestToSet converts CLI slices into lookup sets, skipping blanks.
func TestToSet(t *testing.T) {
	if toSet(nil) != nil {
		t.Fatal("nil should stay nil")
	}
	got := toSet([]string{"a", "", "b", " c "})
	want := map[string]bool{"a": true, "b": true, "c": true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for k := range want {
		if !got[k] {
			t.Errorf("missing %q", k)
		}
	}
}
