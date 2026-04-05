package commands

import (
	"strings"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
)

func TestBuildLoginSummaryLines_NotConfigured(t *testing.T) {
	cfg := &config.Config{}

	lines := buildLoginSummaryLines(cfg)

	if len(lines) != 3 {
		t.Fatalf("expected 3 summary lines, got %d", len(lines))
	}

	for _, line := range lines {
		if !strings.Contains(line, "not configured") {
			t.Fatalf("expected line to show not configured, got %q", line)
		}
	}
}

func TestBuildLoginSummaryLines_Configured(t *testing.T) {
	cfg := &config.Config{
		GSCClientID:        "1234567890",
		GSCClientSecret:    "supersecretvalue",
		DataForSEOLogin:    "user@example.com",
		DataForSEOPassword: "password123",
		SERPAPIKey:         "serpapikeyvalue",
		SERPProvider:       "dataforseo",
	}

	lines := buildLoginSummaryLines(cfg)

	if len(lines) != 4 {
		t.Fatalf("expected 4 summary lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "configured (1234****)") {
		t.Fatalf("expected redacted GSC client id in first line, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "configured (user@example.com)") {
		t.Fatalf("expected DataForSEO login in second line, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "configured (serp****)") {
		t.Fatalf("expected redacted SerpAPI key in third line, got %q", lines[2])
	}
	if !strings.Contains(lines[3], "SERP provider: dataforseo") {
		t.Fatalf("expected serp provider line, got %q", lines[3])
	}
}

func TestServiceSummaryStatus(t *testing.T) {
	if got := serviceSummaryStatus(false, "value"); got != "not configured" {
		t.Fatalf("expected not configured, got %q", got)
	}
	if got := serviceSummaryStatus(true, ""); got != "configured" {
		t.Fatalf("expected configured, got %q", got)
	}
	if got := serviceSummaryStatus(true, "  abc  "); got != "configured (abc)" {
		t.Fatalf("expected configured with trimmed value, got %q", got)
	}
}
