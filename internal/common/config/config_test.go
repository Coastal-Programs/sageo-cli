package config

import (
	"path/filepath"
	"testing"
)

func TestPathUsesEnvOverride(t *testing.T) {
	resetPathCacheForTest()

	configPath := filepath.Join(t.TempDir(), "sageo-config.json")
	t.Setenv("SAGEO_CONFIG", configPath)

	got := Path()
	if got != configPath {
		t.Fatalf("expected %q, got %q", configPath, got)
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	resetPathCacheForTest()

	configPath := filepath.Join(t.TempDir(), "sageo-config.json")
	t.Setenv("SAGEO_CONFIG", configPath)

	cfg := NewDefault()
	cfg.ActiveProvider = "cloudflare"
	cfg.APIKey = "secret-token-value"
	cfg.BaseURL = "https://api.example.com"
	cfg.OrganizationID = "org_123"

	if err := cfg.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.ActiveProvider != cfg.ActiveProvider {
		t.Fatalf("expected active provider %q, got %q", cfg.ActiveProvider, loaded.ActiveProvider)
	}
	if loaded.APIKey != cfg.APIKey {
		t.Fatalf("expected api key %q, got %q", cfg.APIKey, loaded.APIKey)
	}
	if loaded.BaseURL != cfg.BaseURL {
		t.Fatalf("expected base url %q, got %q", cfg.BaseURL, loaded.BaseURL)
	}
	if loaded.OrganizationID != cfg.OrganizationID {
		t.Fatalf("expected organization id %q, got %q", cfg.OrganizationID, loaded.OrganizationID)
	}
}
