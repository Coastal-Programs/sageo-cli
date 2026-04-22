package html

import (
	"bytes"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
	netxhtml "golang.org/x/net/html"
)

func minimalState() *state.State {
	return &state.State{
		Site:         "https://example.com",
		Score:        72,
		PagesCrawled: 10,
	}
}

func populatedState() *state.State {
	now := time.Now().UTC()
	return &state.State{
		Site:         "https://example.com",
		Score:        58,
		PagesCrawled: 42,
		Findings: []state.Finding{
			{Rule: "missing-title", URL: "https://example.com/a", Verdict: "fail", Why: "no title", Fix: "add title"},
			{Rule: "meta-too-long", URL: "https://example.com/b", Verdict: "warn", Why: "meta >160", Fix: "trim"},
		},
		BrandTerms: []string{"Example"},
		GSC: &state.GSCData{
			LastPull: now.Format(time.RFC3339),
			TopKeywords: []state.GSCRow{
				{Key: "example keyword", Clicks: 3, Impressions: 500, CTR: 0.006, Position: 15.2},
				{Key: "brand test", Clicks: 10, Impressions: 200, CTR: 0.05, Position: 4.1},
			},
		},
		PSI: &state.PSIData{
			LastRun: now.Format(time.RFC3339),
			Pages: []state.PSIResult{
				{URL: "https://example.com/", PerformanceScore: 0.62, LCP: 3800, CLS: 0.18, Strategy: "mobile"},
			},
		},
		SERP: &state.SERPData{
			LastRun: now.Format(time.RFC3339),
			Queries: []state.SERPQueryResult{
				{Query: "example keyword", HasAIOverview: true, OurPosition: 12, TopDomains: []string{"wikipedia.org", "google.com"}},
			},
		},
		Backlinks: &state.BacklinksData{
			LastRun:               now.Format(time.RFC3339),
			TotalBacklinks:        1200,
			TotalReferringDomains: 85,
			BrokenBacklinks:       4,
			SpamScore:             12,
		},
		AEO: &state.AEOData{
			LastRun: now.Format(time.RFC3339),
			Responses: []state.AEOPromptResult{
				{Prompt: "Best example tools?", Results: []state.AEOResponseResult{
					{Engine: "openai", ModelName: "gpt", Response: "Example is a great option", FetchedAt: now},
				}},
				{Prompt: "Who makes foo?", Results: []state.AEOResponseResult{
					{Engine: "anthropic", ModelName: "claude", Response: "Some vendor", FetchedAt: now},
				}},
			},
		},
		Recommendations: []state.Recommendation{
			{
				ID: "rec-1", TargetURL: "https://example.com/a", ChangeType: state.ChangeTitle,
				CurrentValue: "Old title", RecommendedValue: "New, better title",
				Rationale: "Improves CTR", Priority: 90, EffortMinutes: 15,
				Evidence:       []state.Evidence{{Source: "gsc", Metric: "ctr", Value: 0.01}},
				ForecastedLift: &state.Forecast{RawEstimate: 120, RawConfidenceLow: 60, RawConfidenceHigh: 200, PriorityTier: state.PriorityMedium},
			},
			{
				ID: "rec-2", TargetURL: "https://example.com/b", ChangeType: state.ChangeMeta,
				Priority: 65, EffortMinutes: 5,
				ForecastedLift: &state.Forecast{RawEstimate: 40, RawConfidenceLow: 20, RawConfidenceHigh: 70, PriorityTier: state.PriorityLow},
			},
			{
				ID: "rec-3", TargetURL: "https://example.com/c", ChangeType: state.ChangeSchema,
				Priority: 35,
			},
		},
	}
}

func TestRender_MinimalState(t *testing.T) {
	var buf bytes.Buffer
	size, err := RenderWithStats(minimalState(), &buf, Options{})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if size == 0 {
		t.Fatal("expected non-zero size")
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("<!DOCTYPE html>")) {
		t.Fatalf("expected <!DOCTYPE html> prefix, got %q", buf.Bytes()[:20])
	}
}

func TestRender_InvalidInputs(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(nil, &buf, Options{}); err == nil {
		t.Fatal("expected error for nil state")
	}
	if err := Render(minimalState(), nil, Options{}); err == nil {
		t.Fatal("expected error for nil writer")
	}
}

func TestRender_AppendixTogglesContent(t *testing.T) {
	var without bytes.Buffer
	if err := Render(populatedState(), &without, Options{}); err != nil {
		t.Fatalf("render without appendix: %v", err)
	}
	var with bytes.Buffer
	if err := Render(populatedState(), &with, Options{IncludeAppendix: true}); err != nil {
		t.Fatalf("render with appendix: %v", err)
	}
	if with.Len() <= without.Len() {
		t.Fatalf("expected appendix render to be larger (with=%d, without=%d)", with.Len(), without.Len())
	}
	if strings.Contains(without.String(), "Appendix A") {
		t.Error("appendix content should not appear when IncludeAppendix=false")
	}
	if !strings.Contains(with.String(), "Appendix A") {
		t.Error("expected Appendix A heading when IncludeAppendix=true")
	}
}

func TestRender_SectionHeadings(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(populatedState(), &buf, Options{IncludeAppendix: true}); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := buf.String()
	mustContain := []string{
		"SEO Performance Report",
		"Executive Summary",
		"What's broken",
		"Recommendations",
		"Forecast Summary",
		"Appendix A",
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("expected %q in rendered HTML", want)
		}
	}
}

func TestRender_NoExternalURLs(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(populatedState(), &buf, Options{IncludeAppendix: true}); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := buf.String()
	// Check for external resource references (not just any URL — we expect
	// the site URL to appear in the report body as content, so we specifically
	// check for CSS/JS/image/font loads).
	forbidden := []*regexp.Regexp{
		regexp.MustCompile(`<link[^>]+href=["']https?://`),
		regexp.MustCompile(`<link[^>]+href=["']//`),
		regexp.MustCompile(`<script[^>]+src=["']https?://`),
		regexp.MustCompile(`<script[^>]+src=["']//`),
		regexp.MustCompile(`@import\s+(url\()?["']?https?://`),
		regexp.MustCompile(`url\(["']?https?://[^)]+\)`),
		regexp.MustCompile(`//cdn\.`),
	}
	for _, re := range forbidden {
		if re.MatchString(body) {
			t.Errorf("external resource reference found: %s", re)
		}
	}
}

func TestRender_LogoDataURIAllowed(t *testing.T) {
	var buf bytes.Buffer
	logo := "data:image/png;base64,iVBORw0KGgo="
	if err := Render(populatedState(), &buf, Options{LogoDataURI: logo}); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(buf.String(), logo) {
		t.Error("expected data-URI logo to appear in output")
	}
}

func TestRender_ParsesAsHTML(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(populatedState(), &buf, Options{IncludeAppendix: true}); err != nil {
		t.Fatalf("render: %v", err)
	}
	if _, err := netxhtml.Parse(&buf); err != nil {
		t.Fatalf("output does not parse as HTML: %v", err)
	}
}

func TestRender_PriorityBadgeAccessibility(t *testing.T) {
	var buf bytes.Buffer
	if err := Render(populatedState(), &buf, Options{}); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := buf.String()
	// Every priority badge must include an aria-label so colour is never
	// the sole signal for priority.
	if !strings.Contains(body, `aria-label="Priority tier MEDIUM"`) {
		t.Error("expected aria-label on priority tier badge")
	}
}

func TestPriorityClass(t *testing.T) {
	cases := map[int]string{95: "priority-high", 70: "priority-med", 40: "priority-low"}
	for p, want := range cases {
		if got := priorityClass(p); got != want {
			t.Errorf("priorityClass(%d) = %s, want %s", p, got, want)
		}
	}
}

func TestFormatInt(t *testing.T) {
	cases := map[int]string{0: "0", 123: "123", 1234: "1,234", 1234567: "1,234,567", -1000: "-1,000"}
	for in, want := range cases {
		if got := formatInt(in); got != want {
			t.Errorf("formatInt(%d) = %s, want %s", in, got, want)
		}
	}
}
