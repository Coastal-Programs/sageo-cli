package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jakeschepis/sageo-cli/internal/state"
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
				ForecastedLift: &state.Forecast{EstimatedMonthlyClicksDelta: 120, ConfidenceLow: 60, ConfidenceHigh: 200},
			},
			{
				ID: "rec-2", TargetURL: "https://example.com/b", ChangeType: state.ChangeMeta,
				Priority: 65, EffortMinutes: 5,
				ForecastedLift: &state.Forecast{EstimatedMonthlyClicksDelta: 40, ConfidenceLow: 20, ConfidenceHigh: 70},
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
	pages, size, err := RenderWithStats(minimalState(), &buf, Options{})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	if size == 0 {
		t.Fatal("expected non-zero size")
	}
	if !bytes.HasPrefix(buf.Bytes(), []byte("%PDF-")) {
		t.Fatalf("expected PDF magic header, got %q", buf.Bytes()[:8])
	}
	if pages < 1 {
		t.Fatalf("expected at least 1 page, got %d", pages)
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

func TestRender_PopulatedState_AppendixChangesPageCount(t *testing.T) {
	var without bytes.Buffer
	pagesWithout, _, err := RenderWithStats(populatedState(), &without, Options{})
	if err != nil {
		t.Fatalf("render without appendix: %v", err)
	}

	var with bytes.Buffer
	pagesWith, _, err := RenderWithStats(populatedState(), &with, Options{IncludeAppendix: true})
	if err != nil {
		t.Fatalf("render with appendix: %v", err)
	}

	if pagesWith <= pagesWithout {
		t.Fatalf("expected more pages with appendix (with=%d, without=%d)", pagesWith, pagesWithout)
	}
}

func TestRender_SectionHeadingsPresent(t *testing.T) {
	var buf bytes.Buffer
	opts := Options{IncludeAppendix: true}
	opts.disableCompression = true
	_, _, err := RenderWithStats(populatedState(), &buf, opts)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// gofpdf produces a text stream with page content as literal strings
	// enclosed in parentheses. We search the raw bytes for substrings of each
	// expected heading. Defensive against compression/encoding quirks.
	body := buf.String()
	mustContain := []string{
		"Executive Summary",
		"Recommendations",
		"Forecast Summary",
		"Appendix A",
	}
	for _, want := range mustContain {
		if !strings.Contains(body, want) {
			t.Errorf("expected section heading %q in PDF stream", want)
		}
	}
}

func TestParseHex(t *testing.T) {
	r, g, b := parseHex("#1E40AF", defaultBrandHex)
	if r != 0x1E || g != 0x40 || b != 0xAF {
		t.Fatalf("unexpected: %d %d %d", r, g, b)
	}
	// Invalid falls back.
	r2, g2, b2 := parseHex("not-a-hex", "#1E40AF")
	if r2 != 0x1E || g2 != 0x40 || b2 != 0xAF {
		t.Fatalf("fallback failed: %d %d %d", r2, g2, b2)
	}
}

func TestPriorityColor(t *testing.T) {
	if r, _, _ := priorityColor(95); r != 185 {
		t.Errorf("priority 95 should be red, got r=%d", r)
	}
	if r, _, _ := priorityColor(70); r != 202 {
		t.Errorf("priority 70 should be amber")
	}
	if r, _, _ := priorityColor(40); r != 107 {
		t.Errorf("priority 40 should be grey")
	}
}
