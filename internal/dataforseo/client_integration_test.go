//go:build integration
// +build integration

package dataforseo

import (
	"os"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
)

// TestVerifyCredentials_Live hits the real DataForSEO API.
// Requires SAGEO_DATAFORSEO_LOGIN + SAGEO_DATAFORSEO_PASSWORD.
func TestVerifyCredentials_Live(t *testing.T) {
	if os.Getenv("SAGEO_LIVE_TESTS") != "1" {
		t.Skip("set SAGEO_LIVE_TESTS=1 to run integration tests (real API calls, costs money)")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
		t.Skip("DataForSEO credentials not configured; skipping live test")
	}

	c := New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)
	if err := c.VerifyCredentials(); err != nil {
		t.Fatalf("live VerifyCredentials: %v", err)
	}
}
