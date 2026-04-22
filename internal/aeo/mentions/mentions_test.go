package mentions

import (
	"strings"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

func resp(engine, model, text string) state.AEOResponseResult {
	return state.AEOResponseResult{Engine: engine, ModelName: model, Response: text}
}

func promptResult(prompt string, results ...state.AEOResponseResult) state.AEOPromptResult {
	return state.AEOPromptResult{Prompt: prompt, Results: results}
}

func TestDetectInResponses_CaseInsensitiveAndWholeWord(t *testing.T) {
	responses := []state.AEOPromptResult{
		promptResult("what seo tools?",
			resp("chatgpt", "gpt-5", "Sageo is great. We love sageo. But massageo is not sageo."),
		),
	}
	matches := DetectInResponses(responses, []string{"Sageo"})
	if len(matches) != 1 {
		t.Fatalf("want 1 match, got %d", len(matches))
	}
	m := matches[0]
	if m.Term != "Sageo" {
		t.Errorf("term: want Sageo, got %s", m.Term)
	}
	// "Sageo", "sageo", "sageo" = 3 (the "massageo" substring should NOT match).
	if m.Count != 3 {
		t.Errorf("count: want 3 (whole-word, case-insensitive), got %d", m.Count)
	}
}

func TestDetectInResponses_ContextWindowCap(t *testing.T) {
	// Five sentences each containing "Sageo" → expect Count=5 but only 3
	// contexts kept.
	text := "Sageo one. Sageo two. Sageo three. Sageo four. Sageo five."
	responses := []state.AEOPromptResult{
		promptResult("p", resp("chatgpt", "gpt-5", text)),
	}
	matches := DetectInResponses(responses, []string{"sageo"})
	if len(matches) != 1 {
		t.Fatalf("want 1 match, got %d", len(matches))
	}
	if matches[0].Count != 5 {
		t.Errorf("count: want 5, got %d", matches[0].Count)
	}
	if len(matches[0].Contexts) != 3 {
		t.Errorf("contexts: want 3 (capped), got %d", len(matches[0].Contexts))
	}
	// First context includes neighbours: "Sageo one. Sageo two."
	if !strings.Contains(matches[0].Contexts[0], "Sageo one") {
		t.Errorf("first context missing target sentence: %q", matches[0].Contexts[0])
	}
}

func TestDetectInResponses_MultiTermDedupe(t *testing.T) {
	responses := []state.AEOPromptResult{
		promptResult("p1",
			resp("chatgpt", "gpt-5", "Try Sageo. Also try sageo.io for more."),
			resp("claude", "claude-sonnet-4-6", "I recommend Sageo highly."),
		),
	}
	matches := DetectInResponses(responses, []string{"Sageo", "sageo.io", "notthere"})
	// Expect:
	//  chatgpt × Sageo (count 2)        — "Sageo" + "sageo.io" (sageo stem, whole-word)
	//  chatgpt × sageo.io (count 1)
	//  claude  × Sageo (count 1)
	// "notthere" not matched at all.
	if len(matches) != 3 {
		t.Fatalf("want 3 deduped matches, got %d: %+v", len(matches), matches)
	}
	seen := map[string]int{}
	for _, m := range matches {
		k := m.Engine + "|" + m.Term
		seen[k] = m.Count
	}
	if seen["chatgpt|Sageo"] != 2 {
		t.Errorf("chatgpt|Sageo: want 2, got %d", seen["chatgpt|Sageo"])
	}
	if seen["chatgpt|sageo.io"] != 1 {
		t.Errorf("chatgpt|sageo.io: want 1, got %d", seen["chatgpt|sageo.io"])
	}
	if seen["claude|Sageo"] != 1 {
		t.Errorf("claude|Sageo: want 1, got %d", seen["claude|Sageo"])
	}
	for _, m := range matches {
		if m.Term == "notthere" {
			t.Errorf("zero-count term should not appear: %+v", m)
		}
	}
}

func TestDetectInResponses_EmptyInputs(t *testing.T) {
	if m := DetectInResponses(nil, []string{"x"}); len(m) != 0 {
		t.Errorf("nil responses: want 0 matches, got %d", len(m))
	}
	if m := DetectInResponses([]state.AEOPromptResult{promptResult("p", resp("c", "m", "hi"))}, nil); len(m) != 0 {
		t.Errorf("nil terms: want 0 matches, got %d", len(m))
	}
	if m := DetectInResponses([]state.AEOPromptResult{promptResult("p", resp("c", "m", "hi"))}, []string{"", "  "}); len(m) != 0 {
		t.Errorf("blank terms: want 0 matches, got %d", len(m))
	}
}
