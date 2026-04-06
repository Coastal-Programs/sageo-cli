package merge

import (
	"strings"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// findRule returns the first MergedFinding with the given rule or nil.
func findRule(findings []MergedFinding, rule string) *MergedFinding {
	for i := range findings {
		if findings[i].Rule == rule {
			return &findings[i]
		}
	}
	return nil
}

// ─── cross-source rule tests ──────────────────────────────────────────────────

// TestRankingButNotClicking: page has title-too-long finding, GSC shows 50
// impressions with 0 clicks → expect "ranking-but-not-clicking".
func TestRankingButNotClicking(t *testing.T) {
	const pageURL = "https://example.com/product"

	st := buildState(
		[]state.Finding{
			{Rule: "title-too-long", URL: pageURL, Verdict: "fail"},
		},
		[]state.GSCRow{
			{Key: pageURL, Impressions: 50, Clicks: 0, CTR: 0, Position: 8},
		},
	)

	results := Run(st)

	f := findRule(results, "ranking-but-not-clicking")
	if f == nil {
		t.Fatalf("expected finding %q, got: %v", "ranking-but-not-clicking", results)
	}
	if f.GSCData == nil {
		t.Fatal("expected GSCData to be populated on the finding")
	}
	if f.GSCData.Impressions != 50 {
		t.Errorf("GSCData.Impressions = %.0f, want 50", f.GSCData.Impressions)
	}
	if f.GSCData.Clicks != 0 {
		t.Errorf("GSCData.Clicks = %.0f, want 0", f.GSCData.Clicks)
	}
	if len(f.Sources) == 0 {
		t.Error("expected Sources to be non-empty")
	}
}

// TestNotIndexed: page is present in crawl findings (200 status, no noindex
// issues) but absent from GSC data → expect "not-indexed".
func TestNotIndexed(t *testing.T) {
	const pageURL = "https://example.com/about"

	st := buildState(
		[]state.Finding{
			// A crawl-level issue that is NOT a noindex directive.
			{Rule: "missing-meta-description", URL: pageURL, Verdict: "fail"},
		},
		// GSC data is present (LastPull will be set) but for a different URL.
		[]state.GSCRow{
			{Key: "https://example.com/other-page", Impressions: 100, Clicks: 20},
		},
	)

	results := Run(st)

	f := findRule(results, "not-indexed")
	if f == nil {
		t.Fatalf("expected finding %q, got: %v", "not-indexed", results)
	}
	// not-indexed findings have no live GSC metrics by definition.
	if f.GSCData != nil {
		t.Error("not-indexed finding should not carry GSCData")
	}
}

// TestIssuesOnHighTrafficPage: page has crawl issues AND GSC shows 10 clicks
// → expect "issues-on-high-traffic-page".
func TestIssuesOnHighTrafficPage(t *testing.T) {
	const pageURL = "https://example.com/popular"

	st := buildState(
		[]state.Finding{
			{Rule: "slow-ttfb", URL: pageURL, Verdict: "fail"},
		},
		[]state.GSCRow{
			{Key: pageURL, Impressions: 200, Clicks: 10, CTR: 0.05, Position: 6},
		},
	)

	results := Run(st)

	f := findRule(results, "issues-on-high-traffic-page")
	if f == nil {
		t.Fatalf("expected finding %q, got: %v", "issues-on-high-traffic-page", results)
	}
	if f.GSCData == nil || f.GSCData.Clicks != 10 {
		t.Errorf("expected GSCData.Clicks = 10, got %v", f.GSCData)
	}
	if len(f.CrawlIssues) == 0 {
		t.Error("expected CrawlIssues to be non-empty")
	}
}

// TestThinContentRankingWell: page has thin-content finding and GSC position < 10
// → expect "thin-content-ranking-well".
func TestThinContentRankingWell(t *testing.T) {
	const pageURL = "https://example.com/guide"

	st := buildState(
		[]state.Finding{
			{Rule: "thin-content", URL: pageURL, Verdict: "fail"},
		},
		[]state.GSCRow{
			// Keep impressions ≤ 10 and clicks = 0 so other rules don't fire.
			{Key: pageURL, Impressions: 5, Clicks: 0, Position: 5},
		},
	)

	results := Run(st)

	f := findRule(results, "thin-content-ranking-well")
	if f == nil {
		t.Fatalf("expected finding %q, got: %v", "thin-content-ranking-well", results)
	}
	if f.GSCData == nil || f.GSCData.Position != 5 {
		t.Errorf("expected GSCData.Position = 5, got %v", f.GSCData)
	}
}

// TestNoMergedFindings: page has no crawl issues and strong GSC metrics
// → expect zero merged findings.
func TestNoMergedFindings(t *testing.T) {
	st := buildState(
		// No crawl findings at all.
		[]state.Finding{},
		[]state.GSCRow{
			{Key: "https://example.com/healthy", Impressions: 500, Clicks: 80, CTR: 0.16, Position: 2},
		},
	)

	results := Run(st)
	if len(results) != 0 {
		t.Errorf("expected 0 merged findings, got %d: %v", len(results), results)
	}
}

// TestURLNormalizationMatching: crawl URL has www prefix, GSC URL does not.
// Both should normalize to the same key and produce merged findings.
func TestURLNormalizationMatching(t *testing.T) {
	const crawlURL = "https://www.example.com/contact"
	const gscURL = "https://example.com/contact"

	st := buildState(
		[]state.Finding{
			{Rule: "title-too-long", URL: crawlURL, Verdict: "fail"},
		},
		// GSC stores the URL without www.
		[]state.GSCRow{
			{Key: gscURL, Impressions: 60, Clicks: 0, CTR: 0, Position: 7},
		},
	)

	results := Run(st)

	// ranking-but-not-clicking should fire: impressions > 10, clicks == 0, crawl issue present,
	// and www vs. non-www normalization must align the two URLs.
	f := findRule(results, "ranking-but-not-clicking")
	if f == nil {
		t.Fatalf("expected finding %q after URL normalization, got: %v", "ranking-but-not-clicking", results)
	}

	// not-indexed must NOT fire — the page is present in GSC after normalization.
	if ni := findRule(results, "not-indexed"); ni != nil {
		t.Errorf("not-indexed should not fire when a GSC row exists for the normalized URL")
	}
}

// TestSlowCoreWebVitals: page has PSI score < 50 AND GSC impressions → expect
// "slow-core-web-vitals" merged finding.
func TestSlowCoreWebVitals(t *testing.T) {
	const pageURL = "https://example.com/slow"

	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{
			{Key: pageURL, Impressions: 200, Clicks: 5, CTR: 0.025, Position: 8},
		},
	)
	st.PSI = &state.PSIData{
		LastRun: "2025-01-01T00:00:00Z",
		Pages: []state.PSIResult{
			{URL: pageURL, PerformanceScore: 28, LCP: 6500, CLS: 0.05, Strategy: "mobile"},
		},
	}

	results := Run(st)

	f := findRule(results, "slow-core-web-vitals")
	if f == nil {
		t.Fatalf("expected finding %q, got: %v", "slow-core-web-vitals", results)
	}
	if f.GSCData == nil {
		t.Fatal("expected GSCData to be populated on the finding")
	}
	if f.GSCData.Impressions != 200 {
		t.Errorf("GSCData.Impressions = %.0f, want 200", f.GSCData.Impressions)
	}
	// Fix should mention LCP since it's above the 4000 ms poor threshold.
	if !strings.Contains(f.Fix, "LCP") {
		t.Errorf("Fix message should mention LCP; got: %s", f.Fix)
	}
}

// TestSlowCoreWebVitalsNoFire: PSI score < 50 but no GSC impressions →
// "slow-core-web-vitals" must NOT fire (no ranking potential to lose).
func TestSlowCoreWebVitalsNoFire(t *testing.T) {
	const pageURL = "https://example.com/ghost"

	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{
			{Key: "https://example.com/other", Impressions: 100, Clicks: 10},
		},
	)
	st.PSI = &state.PSIData{
		LastRun: "2025-01-01T00:00:00Z",
		Pages: []state.PSIResult{
			{URL: pageURL, PerformanceScore: 20, LCP: 7000, CLS: 0.3, Strategy: "mobile"},
		},
	}

	results := Run(st)

	if f := findRule(results, "slow-core-web-vitals"); f != nil {
		t.Errorf("slow-core-web-vitals should not fire when page has no GSC impressions; got: %v", f)
	}
}

// TestSlowCoreWebVitalsGoodScore: PSI score >= 50 → rule must not fire.
func TestSlowCoreWebVitalsGoodScore(t *testing.T) {
	const pageURL = "https://example.com/ok"

	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{
			{Key: pageURL, Impressions: 300, Clicks: 20, CTR: 0.07, Position: 4},
		},
	)
	st.PSI = &state.PSIData{
		LastRun: "2025-01-01T00:00:00Z",
		Pages: []state.PSIResult{
			{URL: pageURL, PerformanceScore: 72, LCP: 1800, CLS: 0.02, Strategy: "mobile"},
		},
	}

	results := Run(st)

	if f := findRule(results, "slow-core-web-vitals"); f != nil {
		t.Errorf("slow-core-web-vitals should not fire when performance score >= 50; got: %v", f)
	}
}

func buildState(findings []state.Finding, topPages []state.GSCRow) *state.State {
	st := &state.State{
		Site:      "https://example.com",
		LastCrawl: "2025-01-01T00:00:00Z",
		Findings:  findings,
	}
	if len(topPages) > 0 {
		st.GSC = &state.GSCData{
			LastPull: "2025-01-01T00:00:00Z",
			TopPages: topPages,
		}
	}
	return st
}

func TestPriorityHighTraffic(t *testing.T) {
	// Page with crawl issues AND GSC clicks > 0 should be HIGH (90-100).
	st := buildState(
		[]state.Finding{
			{Rule: "missing-title", URL: "https://example.com/page1"},
		},
		[]state.GSCRow{
			{Key: "https://example.com/page1", Clicks: 15, Impressions: 100, CTR: 0.15, Position: 5},
		},
	)

	results := Run(st)
	if len(results) == 0 {
		t.Fatal("expected at least one merged finding")
	}

	// Find the issues-on-high-traffic-page finding.
	var found *MergedFinding
	for i, r := range results {
		if r.Rule == "issues-on-high-traffic-page" && r.URL == "https://example.com/page1" {
			found = &results[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected issues-on-high-traffic-page finding")
	}
	if found.Priority != "high" {
		t.Errorf("expected priority high, got %s", found.Priority)
	}
	if found.PriorityScore < 90 || found.PriorityScore > 100 {
		t.Errorf("expected priority score 90-100, got %d", found.PriorityScore)
	}
}

func TestPriorityNoGSC(t *testing.T) {
	// Page with crawl issues but no GSC data should be LOW.
	st := buildState(
		[]state.Finding{
			{Rule: "missing-title", URL: "https://example.com/orphan"},
		},
		nil, // no GSC data at all
	)

	// We need GSC to be non-nil but the page absent from TopPages.
	st.GSC = &state.GSCData{
		LastPull: "2025-01-01T00:00:00Z",
		TopPages: []state.GSCRow{}, // empty — page not in GSC
	}

	results := Run(st)
	if len(results) == 0 {
		t.Fatal("expected at least one merged finding")
	}

	var found *MergedFinding
	for i, r := range results {
		if r.URL == "https://example.com/orphan" {
			found = &results[i]
			break
		}
	}
	if found == nil {
		t.Fatal("expected finding for orphan page")
	}
	if found.Priority != "low" {
		t.Errorf("expected priority low, got %s", found.Priority)
	}
	if found.PriorityScore < 10 || found.PriorityScore > 49 {
		t.Errorf("expected priority score 10-49, got %d", found.PriorityScore)
	}
}

func TestSortOrder(t *testing.T) {
	// Create findings that produce different priority scores and verify
	// they come back sorted by PriorityScore descending.
	st := buildState(
		[]state.Finding{
			{Rule: "missing-title", URL: "https://example.com/high"},
			{Rule: "missing-title", URL: "https://example.com/low"},
			{Rule: "missing-h1", URL: "https://example.com/mid"},
		},
		[]state.GSCRow{
			// high: clicks > 0 → HIGH (90-100)
			{Key: "https://example.com/high", Clicks: 20, Impressions: 200, CTR: 0.10, Position: 3},
			// mid: impressions > 20, clicks == 0 → HIGH (80-89)
			{Key: "https://example.com/mid", Clicks: 0, Impressions: 50, CTR: 0.0, Position: 8},
			// low is absent from GSC → LOW (via not-indexed rule)
		},
	)

	results := Run(st)
	if len(results) < 3 {
		t.Fatalf("expected at least 3 findings, got %d", len(results))
	}

	// Verify descending order.
	for i := 1; i < len(results); i++ {
		if results[i].PriorityScore > results[i-1].PriorityScore {
			t.Errorf("findings not sorted: index %d (score %d) > index %d (score %d)",
				i, results[i].PriorityScore, i-1, results[i-1].PriorityScore)
		}
	}

	// The first finding should have a higher score than the last.
	if results[0].PriorityScore <= results[len(results)-1].PriorityScore {
		t.Errorf("first finding (score %d) should have higher score than last (score %d)",
			results[0].PriorityScore, results[len(results)-1].PriorityScore)
	}

	// Verify every finding has a priority label assigned.
	for _, r := range results {
		if r.Priority == "" {
			t.Errorf("finding %s on %s has empty priority", r.Rule, r.URL)
		}
		if r.PriorityScore == 0 {
			t.Errorf("finding %s on %s has zero priority score", r.Rule, r.URL)
		}
	}
}
