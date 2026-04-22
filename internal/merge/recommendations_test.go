package merge

import (
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// findRec returns the first recommendation matching targetURL and changeType or nil.
func findRec(recs []recommendations.Recommendation, targetURL string, ct recommendations.ChangeType) *recommendations.Recommendation {
	for i := range recs {
		if recs[i].TargetURL == targetURL && recs[i].ChangeType == ct {
			return &recs[i]
		}
	}
	return nil
}

func countByType(recs []recommendations.Recommendation, ct recommendations.ChangeType) int {
	n := 0
	for _, r := range recs {
		if r.ChangeType == ct {
			n++
		}
	}
	return n
}

// TestRecsRankingButNotClicking asserts Title + Meta recs are emitted.
func TestRecsRankingButNotClicking(t *testing.T) {
	const url = "https://example.com/product"
	st := buildState(
		[]state.Finding{{Rule: "title-too-long", URL: url, Verdict: "fail", Value: "Existing Title"}},
		[]state.GSCRow{{Key: url, Impressions: 50, Clicks: 0, CTR: 0, Position: 8}},
	)
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	title := findRec(recs, url, recommendations.ChangeTitle)
	if title == nil {
		t.Fatalf("expected ChangeTitle rec for %s", url)
	}
	if title.Priority == 0 {
		t.Error("priority not copied from finding")
	}
	if title.EffortMinutes != 10 {
		t.Errorf("title effort = %d, want 10", title.EffortMinutes)
	}
	if findRec(recs, url, recommendations.ChangeMeta) == nil {
		t.Errorf("expected ChangeMeta rec for %s", url)
	}
}

func TestRecsNotIndexed(t *testing.T) {
	const url = "https://example.com/about"
	st := buildState(
		[]state.Finding{{Rule: "missing-meta-description", URL: url, Verdict: "fail"}},
		[]state.GSCRow{{Key: "https://example.com/other", Impressions: 100, Clicks: 20}},
	)
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	r := findRec(recs, url, recommendations.ChangeIndexability)
	if r == nil {
		t.Fatalf("expected ChangeIndexability rec for %s", url)
	}
	if r.EffortMinutes != 30 {
		t.Errorf("indexability effort = %d, want 30", r.EffortMinutes)
	}
}

func TestRecsIssuesOnHighTrafficPage(t *testing.T) {
	const url = "https://example.com/page"
	st := buildState(
		[]state.Finding{
			{Rule: "missing-title", URL: url},
			{Rule: "missing-h1", URL: url},
			{Rule: "thin-content", URL: url},
		},
		[]state.GSCRow{{Key: url, Impressions: 100, Clicks: 15, CTR: 0.15, Position: 3}},
	)
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	// We expect one rec per distinct ChangeType triggered by the findings.
	// Findings: missing-title→Title, missing-h1→H1, thin-content→Body.
	if r := findRec(recs, url, recommendations.ChangeTitle); r == nil {
		t.Error("expected ChangeTitle rec")
	}
	if r := findRec(recs, url, recommendations.ChangeH1); r == nil {
		t.Error("expected ChangeH1 rec")
	}
	if r := findRec(recs, url, recommendations.ChangeBody); r == nil {
		t.Error("expected ChangeBody rec")
	}
}

func TestRecsThinContentRankingWell(t *testing.T) {
	const url = "https://example.com/thin"
	st := buildState(
		[]state.Finding{{Rule: "thin-content", URL: url}},
		[]state.GSCRow{{Key: url, Impressions: 300, Clicks: 30, CTR: 0.1, Position: 5}},
	)
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	r := findRec(recs, url, recommendations.ChangeBody)
	if r == nil {
		t.Fatal("expected ChangeBody rec")
	}
	if r.EffortMinutes != 120 {
		t.Errorf("body effort = %d, want 120", r.EffortMinutes)
	}
}

func TestRecsSchemaNotShowing(t *testing.T) {
	const url = "https://example.com/faq"
	st := buildState(
		[]state.Finding{{Rule: "schema-present", URL: url, Value: "FAQPage"}},
		[]state.GSCRow{{Key: url, Impressions: 500, Clicks: 0, CTR: 0.01, Position: 6}},
	)
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	// schema-not-showing emits ChangeSchema with RecommendedValue populated.
	var r *recommendations.Recommendation
	for i := range recs {
		if recs[i].TargetURL == url && recs[i].ChangeType == recommendations.ChangeSchema && recs[i].RecommendedValue != "" {
			r = &recs[i]
			break
		}
	}
	if r == nil {
		t.Fatal("expected ChangeSchema rec with RecommendedValue")
	}
	if r.RecommendedValue != "FAQPage" {
		t.Errorf("schema recommendation = %q, want FAQPage", r.RecommendedValue)
	}
}

func TestRecsSlowCoreWebVitals(t *testing.T) {
	const url = "https://example.com/slow"
	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{{Key: url, Impressions: 200, Clicks: 5, CTR: 0.025, Position: 4}},
	)
	st.PSI = &state.PSIData{
		LastRun: "2025-01-01T00:00:00Z",
		Pages:   []state.PSIResult{{URL: url, PerformanceScore: 30, LCP: 5000, CLS: 0.05, Strategy: "mobile"}},
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	r := findRec(recs, url, recommendations.ChangeSpeed)
	if r == nil {
		t.Fatal("expected ChangeSpeed rec")
	}
	if r.EffortMinutes != 240 {
		t.Errorf("speed effort = %d, want 240", r.EffortMinutes)
	}
	if r.RecommendedValue != "LCP" {
		t.Errorf("slow metric = %q, want LCP", r.RecommendedValue)
	}
}

// TestRecsAIOverviewEatingClicks asserts the research-backed stack:
// ChangeTLDR (Growth Memo 44.2% first-30% finding) +
// ChangeListFormat (signals matrix "likely" across Google/ChatGPT/Perplexity) +
// ChangeSchema FAQPage (Google pipeline marginal lift) +
// ChangeAuthorByline (E-E-A-T + Perplexity trust signals) + H2s per PAA.
func TestRecsAIOverviewEatingClicks(t *testing.T) {
	const query = "what is seo"
	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{{Key: "https://example.com/seo", Impressions: 100, Clicks: 10}},
	)
	st.GSC.TopKeywords = []state.GSCRow{
		{Key: query, Impressions: 200, Clicks: 1, CTR: 0.005, Position: 4},
	}
	st.SERP = &state.SERPData{
		LastRun: "2025-01-01T00:00:00Z",
		Queries: []state.SERPQueryResult{
			{Query: query, HasAIOverview: true, RelatedQuestions: []string{"what is on-page seo", "what is off-page seo"}},
		},
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	if findRec(recs, query, recommendations.ChangeTLDR) == nil {
		t.Error("expected ChangeTLDR rec for AI overview query")
	}
	if findRec(recs, query, recommendations.ChangeListFormat) == nil {
		t.Error("expected ChangeListFormat rec for AI overview query")
	}
	schemaRec := findRec(recs, query, recommendations.ChangeSchema)
	if schemaRec == nil {
		t.Error("expected ChangeSchema rec")
	} else if schemaRec.RecommendedValue != "FAQPage" {
		t.Errorf("AI-overview schema recommendation = %q, want FAQPage", schemaRec.RecommendedValue)
	}
	if findRec(recs, query, recommendations.ChangeAuthorByline) == nil {
		t.Error("expected ChangeAuthorByline rec for AI overview query")
	}
	if countByType(recs, recommendations.ChangeH2) < 2 {
		t.Errorf("expected at least 2 H2 recs for PAA, got %d", countByType(recs, recommendations.ChangeH2))
	}
}

func TestRecsFeaturedSnippetOpportunity(t *testing.T) {
	const query = "define seo"
	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{{Key: "https://example.com/x", Impressions: 100, Clicks: 5}},
	)
	st.GSC.TopKeywords = []state.GSCRow{
		{Key: query, Impressions: 500, Clicks: 10, CTR: 0.02, Position: 4},
	}
	st.SERP = &state.SERPData{
		LastRun: "2025-01-01T00:00:00Z",
		Queries: []state.SERPQueryResult{
			{Query: query, OurPosition: 4, Features: []state.SERPFeatureRecord{{Type: "featured_snippet"}}},
		},
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	// Research: featured-snippet-opportunity now emits ChangeTLDR
	// (40-60 word definition block) rather than generic ChangeBody.
	r := findRec(recs, query, recommendations.ChangeTLDR)
	if r == nil {
		t.Fatalf("expected ChangeTLDR rec for featured snippet; got %+v", recs)
	}
	if r.RecommendedValue != "definition_tldr_40_60_words" {
		t.Errorf("featured-snippet TLDR RecommendedValue = %q, want definition_tldr_40_60_words", r.RecommendedValue)
	}
}

// TestRecsMissingAuthorSignals asserts the new rule emits ChangeAuthorByline
// plus a Person-schema recommendation.
func TestRecsMissingAuthorSignals(t *testing.T) {
	const url = "https://example.com/guide"
	st := buildState(
		[]state.Finding{{Rule: "missing-author-byline", URL: url, Verdict: "fail"}},
		[]state.GSCRow{{Key: url, Impressions: 150, Clicks: 12, CTR: 0.08, Position: 7}},
	)
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	byline := findRec(recs, url, recommendations.ChangeAuthorByline)
	if byline == nil {
		t.Fatalf("expected ChangeAuthorByline rec for %s; got %+v", url, recs)
	}
	if byline.EffortMinutes != 20 {
		t.Errorf("byline effort = %d, want 20", byline.EffortMinutes)
	}
	// Person schema companion.
	var personRec *recommendations.Recommendation
	for i := range recs {
		if recs[i].TargetURL == url && recs[i].ChangeType == recommendations.ChangeSchema && recs[i].RecommendedValue == "Person" {
			personRec = &recs[i]
			break
		}
	}
	if personRec == nil {
		t.Error("expected Person schema rec alongside byline")
	}
}

// TestRecsFreshnessAndEntityChangeTypesReachable is a fixture test that
// exercises the new ChangeType values through the drafter's effortFor
// table, ensuring the enum wiring is complete.
func TestNewChangeTypesHaveEfforts(t *testing.T) {
	cases := map[recommendations.ChangeType]int{
		recommendations.ChangeTLDR:              25,
		recommendations.ChangeListFormat:        30,
		recommendations.ChangeAuthorByline:      20,
		recommendations.ChangeFreshness:         15,
		recommendations.ChangeEntityConsistency: 45,
	}
	for ct, want := range cases {
		if got := effortFor(ct); got != want {
			t.Errorf("effortFor(%s) = %d, want %d", ct, got, want)
		}
	}
}

func TestRecsPAAContentOpportunity(t *testing.T) {
	const query = "seo tips"
	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{{Key: "https://example.com/x", Impressions: 100, Clicks: 5}},
	)
	st.GSC.TopKeywords = []state.GSCRow{
		{Key: query, Impressions: 100, Clicks: 5, CTR: 0.05, Position: 6},
	}
	st.SERP = &state.SERPData{
		LastRun: "2025-01-01T00:00:00Z",
		Queries: []state.SERPQueryResult{
			{Query: query, RelatedQuestions: []string{"q1", "q2", "q3", "q4", "q5", "q6"}},
		},
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	var paaHits int
	for _, r := range recs {
		if r.ChangeType == recommendations.ChangeH2 && r.TargetURL == query {
			paaHits++
		}
	}
	if paaHits == 0 {
		t.Fatal("expected PAA H2 recs")
	}
	if paaHits > 5 {
		t.Errorf("PAA H2 recs should cap at 5, got %d", paaHits)
	}
}

func TestRecsEasyWinKeyword(t *testing.T) {
	const keyword = "cheap widgets"
	st := buildState(
		[]state.Finding{},
		[]state.GSCRow{{Key: "https://example.com/x", Impressions: 100, Clicks: 1}},
	)
	st.GSC.TopKeywords = []state.GSCRow{
		{Key: keyword, Impressions: 500, Clicks: 1, CTR: 0.002, Position: 12},
	}
	st.Labs = &state.LabsData{
		LastRun:  "2025-01-01T00:00:00Z",
		Keywords: []state.LabsKeyword{{Keyword: keyword, SearchVolume: 500, Difficulty: 20}},
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	if findRec(recs, keyword, recommendations.ChangeTitle) == nil {
		t.Error("expected ChangeTitle rec for easy-win keyword")
	}
	if findRec(recs, keyword, recommendations.ChangeH1) == nil {
		t.Error("expected ChangeH1 rec for easy-win keyword")
	}
}

func TestRecsInformationalContentGap(t *testing.T) {
	const keyword = "How To Use SEO"
	st := &state.State{
		Site:      "https://example.com/",
		LastCrawl: "2025-01-01T00:00:00Z",
	}
	st.Labs = &state.LabsData{
		LastRun: "2025-01-01T00:00:00Z",
		Keywords: []state.LabsKeyword{
			{Keyword: keyword, SearchVolume: 500, Difficulty: 20, Intent: "informational"},
		},
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	if len(recs) == 0 {
		t.Fatal("expected at least one rec")
	}
	want := "https://example.com/how-to-use-seo"
	r := findRec(recs, want, recommendations.ChangeBody)
	if r == nil {
		t.Fatalf("expected new-page rec at %s, got %+v", want, recs)
	}
	if r.CurrentValue != "" {
		t.Errorf("new-page rec should have empty CurrentValue, got %q", r.CurrentValue)
	}
}

func TestRecsWeakBacklinkProfile(t *testing.T) {
	st := &state.State{
		Site:      "https://example.com",
		LastCrawl: "2025-01-01T00:00:00Z",
	}
	st.Labs = &state.LabsData{
		LastRun:     "2025-01-01T00:00:00Z",
		Keywords:    []state.LabsKeyword{{Keyword: "k", SearchVolume: 100, Difficulty: 50}},
		Competitors: []string{"comp.com"},
	}
	st.Backlinks = &state.BacklinksData{
		LastRun:               "2025-01-01T00:00:00Z",
		Target:                "example.com",
		TotalReferringDomains: 3,
		GapDomains:            []string{"a.com", "b.com", "c.com"},
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	if countByType(recs, recommendations.ChangeBacklink) < 3 {
		t.Errorf("expected ≥3 backlink recs from gap domains, got %d", countByType(recs, recommendations.ChangeBacklink))
	}
}

func TestRecsBrokenBacklinksFound(t *testing.T) {
	st := &state.State{
		Site:      "https://example.com",
		LastCrawl: "2025-01-01T00:00:00Z",
	}
	st.Backlinks = &state.BacklinksData{
		LastRun:         "2025-01-01T00:00:00Z",
		Target:          "example.com",
		BrokenBacklinks: 5,
	}
	findings := Run(st)
	recs := GenerateRecommendations(st, findings)

	if countByType(recs, recommendations.ChangeBacklink) == 0 {
		t.Error("expected ChangeBacklink rec for broken backlinks")
	}
}

// TestRecsIdempotent ensures running GenerateRecommendations twice produces
// the same IDs (so UpsertRecommendations won't duplicate rows).
func TestRecsIdempotent(t *testing.T) {
	const url = "https://example.com/product"
	st := buildState(
		[]state.Finding{{Rule: "title-too-long", URL: url, Verdict: "fail"}},
		[]state.GSCRow{{Key: url, Impressions: 50, Clicks: 0, CTR: 0, Position: 8}},
	)
	findings := Run(st)

	a := GenerateRecommendations(st, findings)
	b := GenerateRecommendations(st, findings)

	if len(a) != len(b) {
		t.Fatalf("different rec counts: %d vs %d", len(a), len(b))
	}
	idsA := make(map[string]bool, len(a))
	for _, r := range a {
		idsA[r.ID] = true
	}
	for _, r := range b {
		if !idsA[r.ID] {
			t.Errorf("rec ID %s appeared only in second run", r.ID)
		}
	}
}
