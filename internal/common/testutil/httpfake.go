// Package testutil provides helpers for writing hermetic unit tests against
// Sageo's external API clients.
//
// # Test safety contract
//
// Unit tests (default `go test ./...`) must NEVER make a real outbound network
// call. Any test that needs HTTP-level coverage must use either:
//
//   - An in-process httptest.Server wired into the client under test via the
//     factories below, OR
//   - A mock HTTPClient (see internal/dataforseo/client_test.go for the
//     pattern).
//
// Tests that genuinely require live API coverage (paid smoke tests) must be
// in a file tagged `//go:build integration` AND gated on
// `os.Getenv("SAGEO_LIVE_TESTS") == "1"`. See TESTING.md.
//
// # Usage
//
//	srv, client := testutil.NewFakeDataForSEO(t, func(w http.ResponseWriter, r *http.Request) {
//	    _, _ = w.Write([]byte(`{"status_code":20000,"tasks":[...]}`))
//	})
//	defer srv.Close()
//	// ... exercise client ...
package testutil

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
	"github.com/jakeschepis/sageo-cli/internal/llm/anthropic"
	"github.com/jakeschepis/sageo-cli/internal/llm/openai"
	"github.com/jakeschepis/sageo-cli/internal/psi"
)

// NewFakeDataForSEO returns a live *dataforseo.Client wired to a local
// httptest.Server running the supplied handler. The server is registered for
// automatic teardown via t.Cleanup.
func NewFakeDataForSEO(t *testing.T, handler http.HandlerFunc) (*dataforseo.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := dataforseo.New("test-login", "test-pass", dataforseo.WithBaseURL(srv.URL))
	return client, srv
}

// NewFakeAnthropic returns an *anthropic.Client pointed at a local
// httptest.Server. Credentials in the config are stubbed.
func NewFakeAnthropic(t *testing.T, handler http.HandlerFunc) (*anthropic.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := anthropic.New(
		&config.Config{AnthropicAPIKey: "sk-test"},
		anthropic.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewFakeAnthropic: %v", err)
	}
	return client, srv
}

// NewFakeOpenAI returns an *openai.Client pointed at a local httptest.Server.
func NewFakeOpenAI(t *testing.T, handler http.HandlerFunc) (*openai.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client, err := openai.New(
		&config.Config{OpenAIAPIKey: "sk-test"},
		openai.WithBaseURL(srv.URL),
	)
	if err != nil {
		t.Fatalf("NewFakeOpenAI: %v", err)
	}
	return client, srv
}

// NewFakePSI returns a *psi.Client whose outbound requests are transparently
// redirected to a local httptest.Server. PSI's base URL is a package constant,
// so we cannot override it with an option; instead we inject a custom
// HTTPClient that rewrites the request URL's scheme+host on the fly.
func NewFakePSI(t *testing.T, handler http.HandlerFunc) (*psi.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("NewFakePSI: parse server URL: %v", err)
	}

	client := psi.NewClient("test-key", &rewriteClient{
		inner:  srv.Client(),
		scheme: target.Scheme,
		host:   target.Host,
	})
	return client, srv
}

// rewriteClient rewrites the scheme+host of every outbound request to point
// at a test server, preserving path and query. Used only by NewFakePSI.
type rewriteClient struct {
	inner  *http.Client
	scheme string
	host   string
}

func (r *rewriteClient) Do(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = r.scheme
	req.URL.Host = r.host
	req.Host = r.host
	return r.inner.Do(req)
}
