package commands

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
)

func TestResolveAEOQuerySpecs(t *testing.T) {
	tests := []struct {
		name              string
		engine            string
		modelNameOverride string
		models            []string
		all               bool
		tier              string
		wantCount         int
		wantFirstEngine   string
		wantFirstModel    string
		wantErr           bool
	}{
		{
			name:            "default single path unchanged (flagship chatgpt)",
			tier:            "flagship",
			wantCount:       1,
			wantFirstEngine: "chatgpt",
			wantFirstModel:  "gpt-5",
		},
		{
			name:            "cheap tier uses legacy defaults",
			engine:          "claude",
			tier:            "cheap",
			wantCount:       1,
			wantFirstEngine: "claude",
			wantFirstModel:  "claude-haiku-4-5",
		},
		{
			name:              "model-name override",
			engine:            "chatgpt",
			modelNameOverride: "gpt-4o",
			tier:              "flagship",
			wantCount:         1,
			wantFirstEngine:   "chatgpt",
			wantFirstModel:    "gpt-4o",
		},
		{
			name:            "all flagship fans to 4",
			all:             true,
			tier:            "flagship",
			wantCount:       4,
			wantFirstEngine: "chatgpt",
			wantFirstModel:  "gpt-5",
		},
		{
			name:            "explicit models list infers engines",
			models:          []string{"gpt-5", "claude-sonnet-4-6", "gemini-3-pro", "sonar-pro"},
			tier:            "flagship",
			wantCount:       4,
			wantFirstEngine: "chatgpt",
			wantFirstModel:  "gpt-5",
		},
		{
			name:    "invalid engine errors",
			engine:  "bard",
			tier:    "flagship",
			wantErr: true,
		},
		{
			name:    "unknown model prefix errors",
			models:  []string{"llama-4"},
			tier:    "flagship",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			specs, err := resolveAEOQuerySpecs(tt.engine, tt.modelNameOverride, tt.models, tt.all, tt.tier)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(specs) != tt.wantCount {
				t.Fatalf("want %d specs, got %d (%+v)", tt.wantCount, len(specs), specs)
			}
			if specs[0].Engine != tt.wantFirstEngine {
				t.Errorf("want first engine %q, got %q", tt.wantFirstEngine, specs[0].Engine)
			}
			if specs[0].ModelName != tt.wantFirstModel {
				t.Errorf("want first model %q, got %q", tt.wantFirstModel, specs[0].ModelName)
			}
		})
	}
}

func TestInferEngineFromModelName(t *testing.T) {
	cases := map[string]string{
		"gpt-5":             "chatgpt",
		"gpt-4o-mini":       "chatgpt",
		"claude-sonnet-4-6": "claude",
		"gemini-3-pro":      "gemini",
		"sonar-pro":         "perplexity",
	}
	for in, want := range cases {
		got, err := inferEngineFromModelName(in)
		if err != nil {
			t.Errorf("%s: unexpected err %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%s: want %q, got %q", in, want, got)
		}
	}
	if _, err := inferEngineFromModelName("mistral-large"); err == nil {
		t.Errorf("expected error for unknown prefix")
	}
}

func TestFlagshipModelNameForEngine(t *testing.T) {
	cases := map[string]string{
		"chatgpt":    "gpt-5",
		"claude":     "claude-sonnet-4-6",
		"gemini":     "gemini-3-pro",
		"perplexity": "sonar-pro",
	}
	for e, want := range cases {
		got, err := flagshipModelNameForEngine(e)
		if err != nil {
			t.Fatalf("%s: %v", e, err)
		}
		if got != want {
			t.Errorf("%s: want %q, got %q", e, want, got)
		}
	}
}

// fanOutAEOQueriesServer returns a mock DataForSEO server that returns a
// canned successful envelope for chat_gpt and gemini, but a task-level error
// for claude (to exercise partial failure propagation).
func fanOutAEOQueriesServer(t *testing.T, hits *int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(hits, 1)
		path := r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(path, "/claude/"):
			// Envelope OK, task-level failure.
			_, _ = fmt.Fprint(w, `{"status_code":20000,"status_message":"ok","tasks":[{"status_code":40501,"status_message":"Invalid model_name","result":[]}]}`)
		default:
			_, _ = fmt.Fprintf(w, `{"status_code":20000,"status_message":"ok","tasks":[{"status_code":20000,"status_message":"ok","result":[{"model_name":"mock","items":[{"type":"message","sections":[{"type":"text","text":"hello from %s"}]}]}]}]}`, path)
		}
	}))
}

func TestFanOutAEOQueries_AllEnginesWithPartialFailure(t *testing.T) {
	var hits int64
	srv := fanOutAEOQueriesServer(t, &hits)
	defer srv.Close()

	client := dataforseo.New("u", "p", dataforseo.WithBaseURL(srv.URL))
	specs := []aeoQuerySpec{
		{Engine: "chatgpt", ModelName: "gpt-5"},
		{Engine: "claude", ModelName: "claude-sonnet-4-6"},
		{Engine: "gemini", ModelName: "gemini-3-pro"},
		{Engine: "perplexity", ModelName: "sonar-pro"},
	}
	outcomes := fanOutAEOQueries(client, "what is sageo?", specs, 4)

	if len(outcomes) != 4 {
		t.Fatalf("want 4 outcomes, got %d", len(outcomes))
	}
	if atomic.LoadInt64(&hits) != 4 {
		t.Errorf("want 4 HTTP hits, got %d", hits)
	}

	for _, o := range outcomes {
		switch o.Engine {
		case "claude":
			if o.Error == "" {
				t.Errorf("claude should have surfaced an error, got success")
			}
			if o.Response != "" {
				t.Errorf("claude response should be empty on error, got %q", o.Response)
			}
		default:
			if o.Error != "" {
				t.Errorf("%s unexpected error: %s", o.Engine, o.Error)
			}
			if !strings.Contains(o.Response, "hello from") {
				t.Errorf("%s missing response text: %q", o.Engine, o.Response)
			}
		}
		if o.CostUSD != perQueryCostUSD {
			t.Errorf("%s: want cost %v, got %v", o.Engine, perQueryCostUSD, o.CostUSD)
		}
	}
}

func TestFanOutAEOQueries_SinglePreservesShape(t *testing.T) {
	var hits int64
	srv := fanOutAEOQueriesServer(t, &hits)
	defer srv.Close()

	client := dataforseo.New("u", "p", dataforseo.WithBaseURL(srv.URL))
	outcomes := fanOutAEOQueries(client, "q", []aeoQuerySpec{{Engine: "chatgpt", ModelName: "gpt-4o-mini"}}, 4)
	if len(outcomes) != 1 {
		t.Fatalf("want 1 outcome, got %d", len(outcomes))
	}
	if outcomes[0].Error != "" {
		t.Fatalf("unexpected error: %s", outcomes[0].Error)
	}
	if outcomes[0].Engine != "chatgpt" || outcomes[0].ModelName != "gpt-4o-mini" {
		t.Errorf("outcome identity wrong: %+v", outcomes[0])
	}
}
