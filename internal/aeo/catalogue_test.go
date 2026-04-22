package aeo

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
)

// mockHTTP implements dataforseo.HTTPClient for tests.
type mockHTTP struct {
	mu     sync.Mutex
	byPath map[string]string // URL.Path -> JSON body
	calls  int32
	err    error
}

func (m *mockHTTP) Do(req *http.Request) (*http.Response, error) {
	atomic.AddInt32(&m.calls, 1)
	if m.err != nil {
		return nil, m.err
	}
	m.mu.Lock()
	body, ok := m.byPath[req.URL.Path]
	m.mu.Unlock()
	if !ok {
		body = `{"status_code":40400,"status_message":"Not found."}`
	}
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}, nil
}

func okResponse(modelName, displayName string, isDefault bool) string {
	return `{
		"status_code": 20000,
		"status_message": "Ok.",
		"tasks": [{
			"status_code": 20000,
			"status_message": "Ok.",
			"result": [{
				"items": [
					{"model_name": "` + modelName + `", "model_display_name": "` + displayName + `", "is_default": ` + boolStr(isDefault) + `}
				]
			}]
		}]
	}`
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestEngineToPath(t *testing.T) {
	cases := map[string]string{
		"chatgpt":    "chat_gpt",
		"claude":     "claude",
		"gemini":     "gemini",
		"perplexity": "perplexity",
		"unknown":    "",
	}
	for in, want := range cases {
		if got := EngineToPath(in); got != want {
			t.Fatalf("EngineToPath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFetchModels_HappyPathPerEngine(t *testing.T) {
	mock := &mockHTTP{
		byPath: map[string]string{
			"/v3/ai_optimization/chat_gpt/llm_responses/models":   okResponse("gpt-5", "GPT-5", true),
			"/v3/ai_optimization/claude/llm_responses/models":     okResponse("claude-sonnet-4-6", "Claude Sonnet 4.6", true),
			"/v3/ai_optimization/gemini/llm_responses/models":     okResponse("gemini-3-pro", "Gemini 3 Pro", true),
			"/v3/ai_optimization/perplexity/llm_responses/models": okResponse("sonar-pro", "Sonar Pro", true),
		},
	}
	client := dataforseo.New("u", "p", dataforseo.WithHTTPClient(mock))

	for _, engine := range SupportedEngines {
		models, err := FetchModels(client, engine)
		if err != nil {
			t.Fatalf("FetchModels(%s) error: %v", engine, err)
		}
		if len(models) != 1 {
			t.Fatalf("FetchModels(%s): got %d models, want 1", engine, len(models))
		}
		if models[0].Engine != engine {
			t.Fatalf("FetchModels(%s): engine = %q", engine, models[0].Engine)
		}
		if !models[0].IsDefault {
			t.Fatalf("FetchModels(%s): expected IsDefault=true", engine)
		}
		if models[0].FetchedAt.IsZero() {
			t.Fatalf("FetchModels(%s): FetchedAt not set", engine)
		}
	}
}

func TestFetchModels_TaskError(t *testing.T) {
	mock := &mockHTTP{
		byPath: map[string]string{
			"/v3/ai_optimization/chat_gpt/llm_responses/models": `{
				"status_code": 20000,
				"status_message": "Ok.",
				"tasks": [{
					"status_code": 40501,
					"status_message": "Invalid Field."
				}]
			}`,
		},
	}
	client := dataforseo.New("u", "p", dataforseo.WithHTTPClient(mock))

	_, err := FetchModels(client, "chatgpt")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "40501") {
		t.Fatalf("expected 40501 in error, got: %v", err)
	}
}

func TestFetchModels_EnvelopeError(t *testing.T) {
	mock := &mockHTTP{
		byPath: map[string]string{
			"/v3/ai_optimization/chat_gpt/llm_responses/models": `{"status_code":40100,"status_message":"Auth failed."}`,
		},
	}
	client := dataforseo.New("u", "p", dataforseo.WithHTTPClient(mock))

	_, err := FetchModels(client, "chatgpt")
	if err == nil || !strings.Contains(err.Error(), "40100") {
		t.Fatalf("expected 40100 error, got: %v", err)
	}
}

func TestFetchModels_UnsupportedEngine(t *testing.T) {
	client := dataforseo.New("u", "p")
	_, err := FetchModels(client, "llama")
	if err == nil {
		t.Fatal("expected error for unsupported engine")
	}
}

func TestFetchAllModels(t *testing.T) {
	mock := &mockHTTP{
		byPath: map[string]string{
			"/v3/ai_optimization/chat_gpt/llm_responses/models":   okResponse("gpt-5", "GPT-5", true),
			"/v3/ai_optimization/claude/llm_responses/models":     okResponse("claude-sonnet-4-6", "Claude Sonnet 4.6", true),
			"/v3/ai_optimization/gemini/llm_responses/models":     okResponse("gemini-3-pro", "Gemini 3 Pro", true),
			"/v3/ai_optimization/perplexity/llm_responses/models": okResponse("sonar-pro", "Sonar Pro", true),
		},
	}
	client := dataforseo.New("u", "p", dataforseo.WithHTTPClient(mock))

	all, err := FetchAllModels(client)
	if err != nil {
		t.Fatalf("FetchAllModels: %v", err)
	}
	if len(all) != len(SupportedEngines) {
		t.Fatalf("got %d engines, want %d", len(all), len(SupportedEngines))
	}
	if atomic.LoadInt32(&mock.calls) != int32(len(SupportedEngines)) {
		t.Fatalf("expected %d calls, got %d", len(SupportedEngines), mock.calls)
	}
}

func TestCacheRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := map[string][]Model{
		"chatgpt": {{Engine: "chatgpt", ModelName: "gpt-5", DisplayName: "GPT-5", IsDefault: true, FetchedAt: time.Now().UTC()}},
		"claude":  {{Engine: "claude", ModelName: "claude-sonnet-4-6", IsDefault: true, FetchedAt: time.Now().UTC()}},
	}
	if err := SaveCached(dir, in); err != nil {
		t.Fatalf("SaveCached: %v", err)
	}

	out, err := LoadCached(dir)
	if err != nil {
		t.Fatalf("LoadCached: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("got %d engines, want 2", len(out))
	}
	if out["chatgpt"][0].ModelName != "gpt-5" {
		t.Fatalf("unexpected chatgpt model: %+v", out["chatgpt"])
	}
}

func TestLoadCached_MissingReturnsEmpty(t *testing.T) {
	dir := t.TempDir()
	out, err := LoadCached(dir)
	if err != nil {
		t.Fatalf("LoadCached: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(out))
	}
}

func TestLoadCached_TTLExpiry(t *testing.T) {
	dir := t.TempDir()
	expired := cacheFile{
		FetchedAt: time.Now().UTC().Add(-8 * 24 * time.Hour),
		Models: map[string][]Model{
			"chatgpt": {{Engine: "chatgpt", ModelName: "gpt-5"}},
		},
	}
	body, _ := json.MarshalIndent(expired, "", "  ")
	path := filepath.Join(dir, CacheFilename)
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	out, err := LoadCached(dir)
	if err != nil {
		t.Fatalf("LoadCached: %v", err)
	}
	if len(out) != 0 {
		t.Fatalf("expected empty map on TTL expiry, got %d entries", len(out))
	}
}

func TestDefaultModelName(t *testing.T) {
	cat := map[string][]Model{
		"chatgpt": {
			{Engine: "chatgpt", ModelName: "gpt-4o-mini", IsDefault: false},
			{Engine: "chatgpt", ModelName: "gpt-5", IsDefault: true},
		},
		"claude": {
			{Engine: "claude", ModelName: "claude-haiku-4-5", IsDefault: false},
		},
	}
	name, err := DefaultModelName(cat, "chatgpt")
	if err != nil || name != "gpt-5" {
		t.Fatalf("DefaultModelName(chatgpt) = %q, %v", name, err)
	}
	name, err = DefaultModelName(cat, "claude")
	if err != nil || name != "claude-haiku-4-5" {
		t.Fatalf("DefaultModelName(claude) = %q, %v", name, err)
	}
	if _, err := DefaultModelName(cat, "gemini"); err == nil {
		t.Fatal("expected error for missing engine")
	}
}
