package commands

import (
	"strings"
	"testing"
)

// TestPreflightGSCCheck_AbortsWhenUnconfigured verifies the pre-flight
// gate returns an error when the user is about to run the full pipeline
// with no active GSC property. This is the guard against the silent-skip
// failure mode that produced 20 unknown-tier recommendations on the
// baysidebuilderswa.com.au run.
func TestPreflightGSCCheck_AbortsWhenUnconfigured(t *testing.T) {
	err := preflightGSCCheck(map[string]bool{}, map[string]bool{}, "")
	if err == nil {
		t.Fatal("expected non-nil preflight error when gsc_property is empty")
	}
	if !strings.Contains(err.Error(), "no GSC property configured") {
		t.Errorf("unexpected error message: %q", err.Error())
	}
	hint := err.Hint()
	for _, want := range []string{
		"sageo auth login gsc",
		"sageo gsc sites use",
		"--skip gsc",
	} {
		if !strings.Contains(hint, want) {
			t.Errorf("expected hint to contain %q, got:\n%s", want, hint)
		}
	}
}

// TestPreflightGSCCheck_SilentWhenSkipped verifies --skip gsc suppresses
// the gate (user has explicitly opted out).
func TestPreflightGSCCheck_SilentWhenSkipped(t *testing.T) {
	if err := preflightGSCCheck(map[string]bool{stageGSC: true}, map[string]bool{}, ""); err != nil {
		t.Errorf("expected nil with --skip gsc, got: %v", err)
	}
}

// TestPreflightGSCCheck_SilentWhenOnlyExcludesGSC verifies --only without
// gsc also suppresses the gate.
func TestPreflightGSCCheck_SilentWhenOnlyExcludesGSC(t *testing.T) {
	if err := preflightGSCCheck(map[string]bool{}, map[string]bool{"crawl": true, "psi": true}, ""); err != nil {
		t.Errorf("expected nil when --only excludes gsc, got: %v", err)
	}
}

// TestPreflightGSCCheck_SilentWhenConfigured verifies the gate does not
// fire when gsc_property is already set.
func TestPreflightGSCCheck_SilentWhenConfigured(t *testing.T) {
	if err := preflightGSCCheck(map[string]bool{}, map[string]bool{}, "https://example.com/"); err != nil {
		t.Errorf("expected nil when gsc_property is set, got: %v", err)
	}
}
