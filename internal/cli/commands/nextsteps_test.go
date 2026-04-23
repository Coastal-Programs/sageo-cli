package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintNextStepsFormat(t *testing.T) {
	var buf bytes.Buffer
	printNextSteps(&buf, []string{"sageo auth login gsc", "sageo gsc sites use https://example.com/"})

	out := buf.String()
	if !strings.HasPrefix(out, "\n") {
		t.Errorf("expected leading blank line, got %q", out)
	}
	if !strings.Contains(out, "Next steps:") {
		t.Errorf("missing heading, got %q", out)
	}
	if !strings.Contains(out, "  1. sageo auth login gsc") {
		t.Errorf("missing numbered step 1, got %q", out)
	}
	if !strings.Contains(out, "  2. sageo gsc sites use https://example.com/") {
		t.Errorf("missing numbered step 2, got %q", out)
	}
}

func TestPrintNextStepsEmpty(t *testing.T) {
	var buf bytes.Buffer
	printNextSteps(&buf, nil)
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty steps, got %q", buf.String())
	}
	printNextSteps(&buf, []string{})
	if buf.Len() != 0 {
		t.Errorf("expected no output for empty steps slice, got %q", buf.String())
	}
}

func TestPrintNextStepsNilWriter(t *testing.T) {
	// Should not panic.
	printNextSteps(nil, []string{"x"})
}
