package commands

import (
	"bytes"
	"strings"
	"testing"
)

// TestPreflightGSCWarning_FiresWhenUnconfigured verifies the pre-flight
// warning emits a loud, actionable stderr message when the user is about
// to run the full pipeline with no active GSC property. This is the
// guard against the silent-skip failure mode that produced 20
// unknown-tier recommendations on the baysidebuilderswa.com.au run.
func TestPreflightGSCWarning_FiresWhenUnconfigured(t *testing.T) {
	var buf bytes.Buffer
	preflightGSCWarningWithProperty(&buf, map[string]bool{}, map[string]bool{}, "")

	out := buf.String()
	for _, want := range []string{
		"WARNING",
		"no GSC property configured",
		"priority_tier=unknown",
		"sageo gsc sites use",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("expected warning to contain %q, got:\n%s", want, out)
		}
	}
}

// TestPreflightGSCWarning_SilentWhenSkipped verifies --skip gsc
// suppresses the warning (user has explicitly opted out).
func TestPreflightGSCWarning_SilentWhenSkipped(t *testing.T) {
	var buf bytes.Buffer
	preflightGSCWarningWithProperty(&buf, map[string]bool{stageGSC: true}, map[string]bool{}, "")
	if buf.Len() != 0 {
		t.Errorf("expected no output with --skip gsc, got: %s", buf.String())
	}
}

// TestPreflightGSCWarning_SilentWhenOnlyExcludesGSC verifies --only
// without gsc also suppresses the warning.
func TestPreflightGSCWarning_SilentWhenOnlyExcludesGSC(t *testing.T) {
	var buf bytes.Buffer
	preflightGSCWarningWithProperty(&buf, map[string]bool{}, map[string]bool{"crawl": true, "psi": true}, "")
	if buf.Len() != 0 {
		t.Errorf("expected no output when --only excludes gsc, got: %s", buf.String())
	}
}

// TestPreflightGSCWarning_SilentWhenConfigured verifies the warning
// does not fire when gsc_property is already set.
func TestPreflightGSCWarning_SilentWhenConfigured(t *testing.T) {
	var buf bytes.Buffer
	preflightGSCWarningWithProperty(&buf, map[string]bool{}, map[string]bool{}, "https://example.com/")
	if buf.Len() != 0 {
		t.Errorf("expected no output when gsc_property is set, got: %s", buf.String())
	}
}
