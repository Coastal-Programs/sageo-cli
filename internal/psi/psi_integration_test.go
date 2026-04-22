//go:build integration
// +build integration

package psi

import (
	"os"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
)

// TestRun_Live hits the real PageSpeed Insights API.
// Requires SAGEO_PSI_API_KEY (or a working unauthenticated quota).
func TestRun_Live(t *testing.T) {
	if os.Getenv("SAGEO_LIVE_TESTS") != "1" {
		t.Skip("set SAGEO_LIVE_TESTS=1 to run integration tests (real API calls, costs money)")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	c := NewClient(cfg.PSIAPIKey, nil)
	res, err := c.Run("https://example.com/", "mobile")
	if err != nil {
		t.Fatalf("live PSI Run: %v", err)
	}
	if res.URL == "" {
		t.Fatalf("empty result URL")
	}
}
