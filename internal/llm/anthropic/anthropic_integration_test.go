//go:build integration
// +build integration

package anthropic

import (
	"context"
	"os"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/llm"
)

// TestComplete_Live hits the real Anthropic Messages API.
// Requires SAGEO_ANTHROPIC_API_KEY.
func TestComplete_Live(t *testing.T) {
	if os.Getenv("SAGEO_LIVE_TESTS") != "1" {
		t.Skip("set SAGEO_LIVE_TESTS=1 to run integration tests (real API calls, costs money)")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AnthropicAPIKey == "" {
		t.Skip("SAGEO_ANTHROPIC_API_KEY not set; skipping live test")
	}

	c, err := New(cfg)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := c.Complete(context.Background(), llm.CompletionRequest{
		System:      "You are a test responder. Reply with the single word: ok.",
		User:        "ping",
		MaxTokens:   16,
		Temperature: 0,
	})
	if err != nil {
		t.Fatalf("live Complete: %v", err)
	}
	if resp.Text == "" {
		t.Fatalf("empty response text")
	}
}
