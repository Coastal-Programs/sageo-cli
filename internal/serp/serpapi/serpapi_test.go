package serpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/serp"
)

func TestEstimate(t *testing.T) {
	a := New("test-key")
	est, err := a.Estimate(serp.AnalyzeRequest{Query: "test"})
	if err != nil {
		t.Fatalf("estimate failed: %v", err)
	}
	if est.Amount != 0.01 {
		t.Fatalf("expected 0.01, got %v", est.Amount)
	}
	if est.Currency != "USD" {
		t.Fatalf("expected USD, got %s", est.Currency)
	}
}

func TestAnalyze(t *testing.T) {
	mockResp := map[string]any{
		"organic_results": []map[string]any{
			{
				"position": 1,
				"title":    "Test Result",
				"link":     "https://example.com/page",
				"snippet":  "A test snippet",
			},
			{
				"position": 2,
				"title":    "Another Result",
				"link":     "https://other.com/page",
				"snippet":  "Another snippet",
			},
		},
		"search_information": map[string]any{
			"total_results":        "12345",
			"time_taken_displayed": 0.42,
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "test-key" {
			t.Error("expected api_key=test-key")
		}
		if r.URL.Query().Get("q") != "seo tools" {
			t.Error("expected q=seo tools")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(mockResp)
	}))
	defer srv.Close()

	a := New("test-key", WithBaseURL(srv.URL))
	resp, err := a.Analyze(serp.AnalyzeRequest{Query: "seo tools"})
	if err != nil {
		t.Fatalf("analyze failed: %v", err)
	}

	if resp.Query != "seo tools" {
		t.Fatalf("expected query 'seo tools', got %q", resp.Query)
	}
	if len(resp.OrganicResults) != 2 {
		t.Fatalf("expected 2 results, got %d", len(resp.OrganicResults))
	}
	if resp.OrganicResults[0].Domain != "example.com" {
		t.Fatalf("expected domain 'example.com', got %q", resp.OrganicResults[0].Domain)
	}
	if resp.TotalResults != 12345 {
		t.Fatalf("expected 12345 total results, got %d", resp.TotalResults)
	}
}

func TestAnalyzeServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New("test-key", WithBaseURL(srv.URL))
	_, err := a.Analyze(serp.AnalyzeRequest{Query: "fail"})
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

func TestDryRunNoNetworkCall(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	a := New("test-key", WithBaseURL(srv.URL))

	// Estimate does not make any HTTP calls
	_, err := a.Estimate(serp.AnalyzeRequest{Query: "dry run test"})
	if err != nil {
		t.Fatalf("estimate failed: %v", err)
	}

	if callCount != 0 {
		t.Fatalf("expected 0 HTTP calls during estimate, got %d", callCount)
	}
}

func TestName(t *testing.T) {
	a := New("key")
	if a.Name() != "serpapi" {
		t.Fatalf("expected 'serpapi', got %q", a.Name())
	}
}

func TestVerifyKeySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("api_key") != "valid-key" {
			t.Errorf("expected api_key=valid-key, got %q", r.URL.Query().Get("api_key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"account_email":      "user@example.com",
			"plan_name":          "Free",
			"searches_per_month": 100,
			"this_month_usage":   5,
		})
	}))
	defer srv.Close()

	a := New("valid-key", WithAccountURL(srv.URL))
	if err := a.VerifyKey(); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestVerifyKeyInvalidKey(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	a := New("bad-key", WithAccountURL(srv.URL))
	err := a.VerifyKey()
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if got := err.Error(); got != "serpapi: invalid API key" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestVerifyKeyForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	a := New("bad-key", WithAccountURL(srv.URL))
	err := a.VerifyKey()
	if err == nil {
		t.Fatal("expected error for forbidden key")
	}
	if got := err.Error(); got != "serpapi: invalid API key" {
		t.Fatalf("unexpected error message: %s", got)
	}
}

func TestVerifyKeyServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New("key", WithAccountURL(srv.URL))
	err := a.VerifyKey()
	if err == nil {
		t.Fatal("expected error for server error")
	}
	expected := "serpapi account endpoint returned status 500"
	if got := err.Error(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestVerifyKeyMalformedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	a := New("key", WithAccountURL(srv.URL))
	err := a.VerifyKey()
	if err == nil {
		t.Fatal("expected error for malformed body")
	}
	if got := err.Error(); len(got) == 0 {
		t.Fatal("expected non-empty error message")
	}
}

func TestVerifyKeyMissingEmail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"plan_name": "Free",
		})
	}))
	defer srv.Close()

	a := New("key", WithAccountURL(srv.URL))
	err := a.VerifyKey()
	if err == nil {
		t.Fatal("expected error when account_email is missing")
	}
	expected := "serpapi: invalid API key (no account email in response)"
	if got := err.Error(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestVerifyKeyTransportError(t *testing.T) {
	a := New("key", WithAccountURL("http://127.0.0.1:1"))
	err := a.VerifyKey()
	if err == nil {
		t.Fatal("expected error for transport failure")
	}
}
