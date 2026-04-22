package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/llm"
)

type stubHTTP struct {
	fn func(*http.Request) (*http.Response, error)
}

func (s stubHTTP) Do(r *http.Request) (*http.Response, error) { return s.fn(r) }

func newResp(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
}

func TestComplete_RequestShapeAndParse(t *testing.T) {
	var captured map[string]any
	var capturedHeaders http.Header
	http := stubHTTP{fn: func(r *http.Request) (*http.Response, error) {
		capturedHeaders = r.Header.Clone()
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		return newResp(200, `{
			"model":"claude-sonnet-4-6",
			"content":[{"type":"text","text":"hello world"}],
			"usage":{"input_tokens":10,"output_tokens":5}
		}`), nil
	}}

	c, err := New(&config.Config{AnthropicAPIKey: "sk-test"}, WithHTTPClient(http), WithBaseURL("https://example.test"))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	resp, err := c.Complete(context.Background(), llm.CompletionRequest{
		System: "sys", User: "u", MaxTokens: 50, Temperature: 0.5,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if resp.Text != "hello world" {
		t.Errorf("text: %q", resp.Text)
	}
	if resp.InputTokens != 10 || resp.OutputTokens != 5 {
		t.Errorf("tokens: %d/%d", resp.InputTokens, resp.OutputTokens)
	}
	want := Cost(10, 5)
	if resp.CostUSD != want {
		t.Errorf("cost: got %v want %v", resp.CostUSD, want)
	}
	if resp.Model != "claude-sonnet-4-6" {
		t.Errorf("model: %q", resp.Model)
	}

	if capturedHeaders.Get("x-api-key") != "sk-test" {
		t.Error("missing x-api-key")
	}
	if capturedHeaders.Get("anthropic-version") != APIVersion {
		t.Error("missing anthropic-version")
	}
	if got := captured["model"]; got != DefaultModel {
		t.Errorf("model: %v", got)
	}
	if got := captured["system"]; got != "sys" {
		t.Errorf("system: %v", got)
	}
	msgs, _ := captured["messages"].([]any)
	if len(msgs) != 1 {
		t.Fatalf("messages: %v", msgs)
	}
	if got := captured["max_tokens"]; got != float64(50) {
		t.Errorf("max_tokens: %v", got)
	}
}

func TestComplete_APIError(t *testing.T) {
	http := stubHTTP{fn: func(r *http.Request) (*http.Response, error) {
		return newResp(401, `{"error":{"type":"authentication_error","message":"bad key"}}`), nil
	}}
	c, _ := New(&config.Config{AnthropicAPIKey: "x"}, WithHTTPClient(http))
	_, err := c.Complete(context.Background(), llm.CompletionRequest{User: "u"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "authentication_error") {
		t.Errorf("err: %v", err)
	}
}

func TestNew_MissingKey(t *testing.T) {
	if _, err := New(&config.Config{}); err == nil {
		t.Fatal("expected error without key")
	}
}

func TestCost(t *testing.T) {
	// 1M input = $3, 1M output = $15
	got := Cost(1_000_000, 1_000_000)
	if got != 18.0 {
		t.Errorf("cost: got %v want 18", got)
	}
	if Cost(0, 0) != 0 {
		t.Error("zero tokens != 0")
	}
}
