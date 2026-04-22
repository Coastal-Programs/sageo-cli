package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDefaultReportOutputPath_InProject verifies that when cwd has a
// .sageo/ directory, the default output path lands under
// .sageo/reports/<timestamp>.html.
func TestDefaultReportOutputPath_InProject(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".sageo"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	var errBuf bytes.Buffer
	path := defaultReportOutputPath(dir, &errBuf)
	if !strings.Contains(path, filepath.Join(".sageo", "reports")) {
		t.Errorf("expected default path under .sageo/reports/, got %s", path)
	}
	if !strings.HasSuffix(path, ".html") {
		t.Errorf("expected .html suffix, got %s", path)
	}
	if errBuf.Len() != 0 {
		t.Errorf("did not expect stderr note inside project, got: %s", errBuf.String())
	}
}

// TestDefaultReportOutputPath_OutsideProject verifies fallback to cwd
// plus a stderr note when no .sageo/ directory exists.
func TestDefaultReportOutputPath_OutsideProject(t *testing.T) {
	dir := t.TempDir()
	var errBuf bytes.Buffer
	path := defaultReportOutputPath(dir, &errBuf)
	if path != "./sageo-report.html" {
		t.Errorf("expected fallback ./sageo-report.html, got %s", path)
	}
	if !strings.Contains(errBuf.String(), "no .sageo") {
		t.Errorf("expected stderr note about missing .sageo/, got: %s", errBuf.String())
	}
}
