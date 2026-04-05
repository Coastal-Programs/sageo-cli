package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Config stores local Sageo CLI settings.
type Config struct {
	ActiveProvider string `json:"active_provider"`
	APIKey         string `json:"api_key"`
	BaseURL        string `json:"base_url"`
	OrganizationID string `json:"organization_id"`
}

var (
	resolvedConfigPath string
	resolvePathOnce    sync.Once
)

func resetPathCacheForTest() {
	resolvedConfigPath = ""
	resolvePathOnce = sync.Once{}
}

// Path returns the resolved config file path.
func Path() string {
	resolvePathOnce.Do(func() {
		if p := os.Getenv("SAGEO_CONFIG"); p != "" {
			clean := filepath.Clean(p)
			if filepath.IsAbs(clean) && strings.HasSuffix(clean, ".json") && !strings.Contains(clean, "..") {
				resolvedConfigPath = clean
				return
			}
		}

		home, err := os.UserHomeDir()
		if err != nil {
			resolvedConfigPath = "config.json"
			return
		}

		resolvedConfigPath = filepath.Join(home, ".config", "sageo", "config.json")
	})

	return resolvedConfigPath
}

// Load reads config from disk. Missing file returns an empty config.
func Load() (*Config, error) {
	data, err := os.ReadFile(Path())
	if os.IsNotExist(err) {
		cfg := NewDefault()
		cfg.applyEnvOverrides()
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg.applyEnvOverrides()
	return &cfg, nil
}

// NewDefault returns default placeholder values for scaffold phase.
func NewDefault() *Config {
	return &Config{
		ActiveProvider: "local",
		BaseURL:        "",
	}
}

// Save persists config to disk with restrictive permissions.
func (c *Config) Save() error {
	if err := os.MkdirAll(filepath.Dir(Path()), 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	body, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	if err := os.WriteFile(Path(), body, 0o600); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

// Set updates a named key.
func (c *Config) Set(key, value string) error {
	switch strings.ToLower(key) {
	case "active_provider", "active-provider", "provider":
		c.ActiveProvider = value
	case "api_key", "api-key", "apikey":
		c.APIKey = value
	case "base_url", "base-url":
		c.BaseURL = value
	case "organization_id", "organization-id", "org_id", "org-id":
		c.OrganizationID = value
	default:
		return fmt.Errorf("unknown config key: %s", key)
	}
	return nil
}

// Get reads a named key and redacts secrets where needed.
func (c *Config) Get(key string) (string, error) {
	switch strings.ToLower(key) {
	case "active_provider", "active-provider", "provider":
		return c.ActiveProvider, nil
	case "api_key", "api-key", "apikey":
		return redact(c.APIKey), nil
	case "base_url", "base-url":
		return c.BaseURL, nil
	case "organization_id", "organization-id", "org_id", "org-id":
		return c.OrganizationID, nil
	default:
		return "", fmt.Errorf("unknown config key: %s", key)
	}
}

// Redacted returns map output safe for display.
func (c *Config) Redacted() map[string]any {
	return map[string]any{
		"active_provider": c.ActiveProvider,
		"api_key":         redact(c.APIKey),
		"base_url":        c.BaseURL,
		"organization_id": c.OrganizationID,
	}
}

func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("SAGEO_PROVIDER"); v != "" {
		c.ActiveProvider = v
	}
	if v := os.Getenv("SAGEO_API_KEY"); v != "" {
		c.APIKey = v
	}
	if v := os.Getenv("SAGEO_BASE_URL"); v != "" {
		c.BaseURL = v
	}
	if v := os.Getenv("SAGEO_ORGANIZATION_ID"); v != "" {
		c.OrganizationID = v
	}
}

func redact(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "****" + value[len(value)-4:]
}
