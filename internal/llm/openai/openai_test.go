package openai

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
	var auth string
	http := stubHTTP{fn: func(r *http.Request) (*http.Response, error) {
		auth = r.Header.Get("Authorization")
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return newResp(200, `{
			"model":"gpt-5",
			"choices":[{"message":{"role":"assistant","content":"hi"}}],
			"usage":{"prompt_tokens":7,"completion_tokens":3,"total_tokens":10}
		}`), nil
	}}

	c, err := New(&config.Config{OpenAIAPIKey: "sk-test"}, WithHTTPClient(http))
	if err != nil {
		t.Fatalf("new: %v", err)
	}
	resp, err := c.Complete(context.Background(), llm.CompletionRequest{
		System: "sys", User: "u", MaxTokens: 33,
	})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}

	if resp.Text != "hi" {
		t.Errorf("text: %q", resp.Text)
	}
	if resp.InputTokens != 7 || resp.OutputTokens != 3 {
		t.Errorf("tokens: %d/%d", resp.InputTokens, resp.OutputTokens)
	}
	if resp.CostUSD != Cost(7, 3) {
		t.Errorf("cost: %v", resp.CostUSD)
	}
	if auth != "Bearer sk-test" {
		t.Errorf("auth header: %q", auth)
	}

	msgs, ok := captured["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("messages: %v", captured["messages"])
	}
	first, _ := msgs[0].(map[string]any)
	if first["role"] != "system" || first["content"] != "sys" {
		t.Errorf("first msg: %v", first)
	}
	if got := captured["max_completion_tokens"]; got != float64(33) {
		t.Errorf("max_completion_tokens: %v", got)
	}
	if got := captured["model"]; got != DefaultModel {
		t.Errorf("model: %v", got)
	}
}

func TestComplete_APIError(t *testing.T) {
	http := stubHTTP{fn: func(r *http.Request) (*http.Response, error) {
		return newResp(400, `{"error":{"type":"invalid_request_error","message":"bad"}}`), nil
	}}
	c, _ := New(&config.Config{OpenAIAPIKey: "x"}, WithHTTPClient(http))
	_, err := c.Complete(context.Background(), llm.CompletionRequest{User: "u"})
	if err == nil || !strings.Contains(err.Error(), "invalid_request_error") {
		t.Errorf("err: %v", err)
	}
}

func TestComplete_EmptyChoices(t *testing.T) {
	http := stubHTTP{fn: func(r *http.Request) (*http.Response, error) {
		return newResp(200, `{"model":"gpt-5","choices":[],"usage":{}}`), nil
	}}
	c, _ := New(&config.Config{OpenAIAPIKey: "x"}, WithHTTPClient(http))
	_, err := c.Complete(context.Background(), llm.CompletionRequest{User: "u"})
	if err == nil {
		t.Fatal("expected error for empty choices")
	}
}

func TestNew_MissingKey(t *testing.T) {
	if _, err := New(&config.Config{}); err == nil {
		t.Fatal("expected error without key")
	}
}

func TestCost(t *testing.T) {
	// 1M input = $1.25, 1M output = $10
	got := Cost(1_000_000, 1_000_000)
	if got != 11.25 {
		t.Errorf("cost: got %v want 11.25", got)
	}
}
