package recommendations

import (
	"context"
	"strings"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/llm"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// stubProvider implements llm.Provider and returns scripted responses.
type stubProvider struct {
	responses []string
	calls     int
	lastUser  string
	lastSys   string
}

func (s *stubProvider) Name() string { return "stub" }

func (s *stubProvider) Complete(_ context.Context, req llm.CompletionRequest) (llm.CompletionResponse, error) {
	s.lastUser = req.User
	s.lastSys = req.System
	idx := s.calls
	if idx >= len(s.responses) {
		idx = len(s.responses) - 1
	}
	text := s.responses[idx]
	s.calls++
	return llm.CompletionResponse{Text: text, InputTokens: 10, OutputTokens: 5, CostUSD: 0.01, Model: "stub"}, nil
}

func TestDraft_TitleSuccessAndPromptIncludesEvidence(t *testing.T) {
	p := &stubProvider{responses: []string{"Best Running Shoes for Beginners 2026"}}
	rec := &state.Recommendation{
		ID:          "t1",
		TargetURL:   "https://example.com/shoes",
		TargetQuery: "running shoes",
		ChangeType:  state.ChangeTitle,
		Evidence: []state.Evidence{
			{Source: "gsc", Metric: "position", Value: 11.2, Description: "just off page one"},
		},
	}
	if err := Draft(context.Background(), p, rec, PageContext{Title: "Old", TargetKeywords: []string{"running shoes"}}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if rec.RecommendedValue != "Best Running Shoes for Beginners 2026" {
		t.Errorf("value: %q", rec.RecommendedValue)
	}
	if !strings.Contains(p.lastUser, "position") || !strings.Contains(p.lastUser, "just off page one") {
		t.Errorf("prompt missing evidence: %s", p.lastUser)
	}
	if !strings.Contains(p.lastSys, "title") {
		t.Errorf("system prompt should mention title, got %q", p.lastSys)
	}
}

func TestDraft_TitleLengthRetry(t *testing.T) {
	tooLong := strings.Repeat("a", 80)
	ok := "A Good Title"
	p := &stubProvider{responses: []string{tooLong, ok}}
	rec := &state.Recommendation{ChangeType: state.ChangeTitle}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if p.calls != 2 {
		t.Errorf("expected 2 calls (retry), got %d", p.calls)
	}
	if rec.RecommendedValue != ok {
		t.Errorf("value: %q", rec.RecommendedValue)
	}
}

func TestDraft_MetaLengthEnforced(t *testing.T) {
	tooLong := strings.Repeat("x", 200)
	p := &stubProvider{responses: []string{tooLong, tooLong}}
	rec := &state.Recommendation{ChangeType: state.ChangeMeta}
	if err := Draft(context.Background(), p, rec, PageContext{}); err == nil {
		t.Fatal("expected failure after retries")
	}
	if rec.RecommendedValue != "" {
		t.Error("should not have been written")
	}
}

func TestDraft_SchemaValidJSON(t *testing.T) {
	good := `{"@context":"https://schema.org","@type":"Article","headline":"x"}`
	p := &stubProvider{responses: []string{good}}
	rec := &state.Recommendation{ChangeType: state.ChangeSchema}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if !strings.Contains(rec.RecommendedValue, "schema.org") {
		t.Errorf("value: %q", rec.RecommendedValue)
	}
}

func TestDraft_SchemaInvalidJSONFails(t *testing.T) {
	p := &stubProvider{responses: []string{"not json", "also not json"}}
	rec := &state.Recommendation{ChangeType: state.ChangeSchema}
	if err := Draft(context.Background(), p, rec, PageContext{}); err == nil {
		t.Fatal("expected failure on invalid json")
	}
}

func TestDraft_SchemaStripsCodeFences(t *testing.T) {
	fenced := "```json\n{\"@context\":\"https://schema.org\",\"@type\":\"Article\"}\n```"
	p := &stubProvider{responses: []string{fenced}}
	rec := &state.Recommendation{ChangeType: state.ChangeSchema}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if strings.Contains(rec.RecommendedValue, "```") {
		t.Errorf("code fences not stripped: %q", rec.RecommendedValue)
	}
}

func TestDraft_BodyWordCount(t *testing.T) {
	body := strings.Repeat("word ", 250)
	p := &stubProvider{responses: []string{body}}
	rec := &state.Recommendation{ChangeType: state.ChangeBody}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if rec.RecommendedValue == "" {
		t.Error("expected body to be set")
	}
}

func TestDraft_UnsupportedChangeTypeNoop(t *testing.T) {
	p := &stubProvider{responses: []string{"should not be called"}}
	rec := &state.Recommendation{ChangeType: state.ChangeSpeed, RecommendedValue: ""}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if p.calls != 0 {
		t.Errorf("expected no calls, got %d", p.calls)
	}
	if rec.RecommendedValue != "" {
		t.Error("expected RecommendedValue to remain empty")
	}
}

// TestDraft_TLDRWordCount asserts the new ChangeTLDR prompt enforces the
// 40-70 word budget from docs/research/ai-citation-signals-2026.md §B.1.2.
func TestDraft_TLDRWordCount(t *testing.T) {
	// 50 words — inside the 40-70 band.
	ok := strings.TrimSpace(strings.Repeat("seo is the practice of improving site visibility in search results. ", 5))
	p := &stubProvider{responses: []string{ok}}
	rec := &state.Recommendation{ChangeType: state.ChangeTLDR, TargetQuery: "what is seo"}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if rec.RecommendedValue == "" {
		t.Error("expected TLDR value to be set")
	}
	if !strings.Contains(p.lastUser, "what is seo") {
		t.Errorf("TLDR prompt missing target query: %s", p.lastUser)
	}
}

func TestDraft_TLDRRejectsTooShort(t *testing.T) {
	short := "Too short."
	p := &stubProvider{responses: []string{short, short}}
	rec := &state.Recommendation{ChangeType: state.ChangeTLDR, TargetQuery: "x"}
	if err := Draft(context.Background(), p, rec, PageContext{}); err == nil {
		t.Fatal("expected failure on too-short TLDR")
	}
}

func TestDraft_ListFormatValidatesDashPrefix(t *testing.T) {
	list := "- first item that is reasonably long enough to be a claim\n- second item that is also a complete claim\n- third item rounding out the list"
	p := &stubProvider{responses: []string{list}}
	rec := &state.Recommendation{ChangeType: state.ChangeListFormat}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if !strings.Contains(rec.RecommendedValue, "- ") {
		t.Errorf("list missing dash prefix: %q", rec.RecommendedValue)
	}
}

func TestDraft_ListFormatRejectsProse(t *testing.T) {
	prose := "This is just prose without any dashes at all."
	p := &stubProvider{responses: []string{prose, prose}}
	rec := &state.Recommendation{ChangeType: state.ChangeListFormat}
	if err := Draft(context.Background(), p, rec, PageContext{}); err == nil {
		t.Fatal("expected prose to fail list validator")
	}
}

func TestDraft_AuthorBylineLength(t *testing.T) {
	byline := "By Jane Doe, MD, Board-Certified Endocrinologist"
	p := &stubProvider{responses: []string{byline}}
	rec := &state.Recommendation{ChangeType: state.ChangeAuthorByline}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if rec.RecommendedValue != byline {
		t.Errorf("byline value: %q", rec.RecommendedValue)
	}
}

func TestDraft_FreshnessLineCount(t *testing.T) {
	plan := "Add \"Updated: April 2026\" string near the byline\nRefresh the statistics in the intro paragraph\nVerify the 2024 study link still resolves"
	p := &stubProvider{responses: []string{plan}}
	rec := &state.Recommendation{ChangeType: state.ChangeFreshness}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if !strings.Contains(rec.RecommendedValue, "Updated") {
		t.Errorf("freshness plan missing Updated string: %q", rec.RecommendedValue)
	}
}

func TestDraft_EntityConsistencyLineCount(t *testing.T) {
	checklist := "Verify NAP matches footer across all pages\nAdd sameAs for Wikipedia in Organization schema\nAdd sameAs for LinkedIn company page"
	p := &stubProvider{responses: []string{checklist}}
	rec := &state.Recommendation{ChangeType: state.ChangeEntityConsistency}
	if err := Draft(context.Background(), p, rec, PageContext{}); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if !strings.Contains(rec.RecommendedValue, "sameAs") {
		t.Errorf("entity checklist missing sameAs: %q", rec.RecommendedValue)
	}
}

func TestDraft_H2Prompt(t *testing.T) {
	p := &stubProvider{responses: []string{"How does X work?"}}
	rec := &state.Recommendation{
		ChangeType: state.ChangeH2,
		Evidence:   []state.Evidence{{Source: "serp", Metric: "paa", Description: "PAA gap"}},
	}
	pc := PageContext{TopPAAQuestions: []string{"How does X work?", "What is X?"}}
	if err := Draft(context.Background(), p, rec, pc); err != nil {
		t.Fatalf("draft: %v", err)
	}
	if !strings.Contains(p.lastUser, "How does X work?") {
		t.Errorf("prompt missing PAA context: %s", p.lastUser)
	}
}

func TestDraft_NilInputs(t *testing.T) {
	if err := Draft(context.Background(), nil, &state.Recommendation{}, PageContext{}); err == nil {
		t.Error("expected error for nil provider")
	}
	p := &stubProvider{responses: []string{""}}
	if err := Draft(context.Background(), p, nil, PageContext{}); err == nil {
		t.Error("expected error for nil rec")
	}
}
