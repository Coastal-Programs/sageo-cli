package llmmentions

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/dataforseo"
)

// mockServer returns a server that serves a canned body for the given path
// prefix. All other paths return a task-level error envelope to exercise the
// error-surfacing path.
func mockServer(t *testing.T, happyPath, happyBody string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log the request so tests can assert it.
		body, _ := io.ReadAll(r.Body)
		t.Logf("mock %s body=%s", r.URL.Path, string(body))
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, happyPath) {
			_, _ = w.Write([]byte(happyBody))
			return
		}
		// Task-level error envelope (envelope OK, task failed).
		_, _ = w.Write([]byte(`{"status_code":20000,"status_message":"ok","tasks":[{"status_code":40400,"status_message":"Not Found","result":[]}]}`))
	}))
}

func newTestClient(srv *httptest.Server) *Client {
	return NewClient(dataforseo.New("u", "p", dataforseo.WithBaseURL(srv.URL)))
}

func TestSearch_HappyPath(t *testing.T) {
	body := `{
		"status_code":20000,"status_message":"ok",
		"tasks":[{"status_code":20000,"status_message":"ok","result":[
			{"keyword":"sageo","total_count":2,"items":[
				{"question":"what is sageo?","answer":"An SEO CLI.","mentions_count":3,"ai_search_volume":120,"impressions":55000},
				{"question":"is sageo free?","answer":"Yes.","mentions_count":1,"ai_search_volume":30,"impressions":9000}
			]}
		]}]}`
	srv := mockServer(t, "/search/live", body)
	defer srv.Close()

	c := newTestClient(srv)
	got, err := c.Search(Request{Keyword: "sageo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.TotalCount != 2 || len(got.Items) != 2 {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.Items[0].MentionsCount != 3 || got.Items[0].AISearchVolume != 120 {
		t.Errorf("first item wrong: %+v", got.Items[0])
	}
}

func TestSearch_TaskLevelError(t *testing.T) {
	// Server always returns task-level error for any path.
	srv := mockServer(t, "/never-matches", "")
	defer srv.Close()

	c := newTestClient(srv)
	_, err := c.Search(Request{Keyword: "sageo"})
	if err == nil {
		t.Fatal("expected task-level error")
	}
	if !strings.Contains(err.Error(), "40400") {
		t.Errorf("want error to mention 40400, got %v", err)
	}
}

func TestTopPages_HappyPath(t *testing.T) {
	body := `{
		"status_code":20000,"status_message":"ok",
		"tasks":[{"status_code":20000,"status_message":"ok","result":[
			{"total_count":1,"items":[
				{"url":"https://example.com/a","domain":"example.com","mentions_count":5,"impressions":10000}
			]}
		]}]}`
	srv := mockServer(t, "/top_pages/live", body)
	defer srv.Close()

	got, err := newTestClient(srv).TopPages(Request{Keyword: "sageo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Domain != "example.com" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestTopPages_TaskLevelError(t *testing.T) {
	srv := mockServer(t, "/never-matches", "")
	defer srv.Close()
	_, err := newTestClient(srv).TopPages(Request{Keyword: "sageo"})
	if err == nil || !strings.Contains(err.Error(), "40400") {
		t.Fatalf("want 40400 task error, got %v", err)
	}
}

func TestTopDomains_HappyPath(t *testing.T) {
	body := `{
		"status_code":20000,"status_message":"ok",
		"tasks":[{"status_code":20000,"status_message":"ok","result":[
			{"total_count":2,"items":[
				{"domain":"wikipedia.org","mentions_count":42,"impressions":500000},
				{"domain":"example.com","mentions_count":8,"impressions":60000}
			]}
		]}]}`
	srv := mockServer(t, "/top_domains/live", body)
	defer srv.Close()

	got, err := newTestClient(srv).TopDomains(Request{Keyword: "sageo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got.Items) != 2 || got.Items[0].Domain != "wikipedia.org" {
		t.Fatalf("unexpected: %+v", got)
	}
}

func TestTopDomains_TaskLevelError(t *testing.T) {
	srv := mockServer(t, "/never-matches", "")
	defer srv.Close()
	_, err := newTestClient(srv).TopDomains(Request{Keyword: "sageo"})
	if err == nil || !strings.Contains(err.Error(), "40400") {
		t.Fatalf("want 40400 task error, got %v", err)
	}
}

func TestAggregatedMetrics_HappyPath(t *testing.T) {
	body := `{
		"status_code":20000,"status_message":"ok",
		"tasks":[{"status_code":20000,"status_message":"ok","result":[
			{"total_count":1,"total":{"location":[{"key":"2036","mentions":1220,"ai_search_volume":76366}]}}
		]}]}`
	srv := mockServer(t, "/aggregated_metrics/live", body)
	defer srv.Close()

	got, err := newTestClient(srv).AggregatedMetrics(Request{Keyword: "sageo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got.TotalCount != 1 {
		t.Fatalf("unexpected: %+v", got)
	}
	// `total` is raw JSON; ensure it decoded into the raw message.
	if len(got.Total) == 0 {
		t.Errorf("want non-empty Total raw json")
	}
}

func TestAggregatedMetrics_TaskLevelError(t *testing.T) {
	srv := mockServer(t, "/never-matches", "")
	defer srv.Close()
	_, err := newTestClient(srv).AggregatedMetrics(Request{Keyword: "sageo"})
	if err == nil || !strings.Contains(err.Error(), "40400") {
		t.Fatalf("want 40400 task error, got %v", err)
	}
}

func TestBuildBody_Defaults(t *testing.T) {
	body := buildBody(Request{Keyword: "sageo"})
	if len(body) != 1 {
		t.Fatalf("want 1 task, got %d", len(body))
	}
	b, _ := json.Marshal(body[0])
	s := string(b)
	for _, want := range []string{
		`"location_code":2036`,
		`"language_code":"en"`,
		`"platform":"google"`,
		`"keyword":"sageo"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("want body to contain %s, got %s", want, s)
		}
	}
}
