// Package mentions detects brand mentions in locally captured AEO responses.
//
// This is "Layer A" of brand mention detection: it scans the AI engine
// responses already stored in state (via `sageo aeo responses`) for a list of
// brand terms using case-insensitive whole-word matching. It is free and
// offline — no DataForSEO calls. Layer B is the DataForSEO LLM Mentions API,
// wrapped by package internal/aeo/llmmentions.
package mentions

import (
	"regexp"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/state"
)

// Match is a single (engine, model, prompt, term) brand mention aggregate. It
// reports the number of times `Term` occurs in the associated response and
// preserves up to 3 surrounding-sentence contexts for human review.
type Match struct {
	Engine    string   `json:"engine"`
	ModelName string   `json:"model_name"`
	Prompt    string   `json:"prompt"`
	Term      string   `json:"term"`
	Count     int      `json:"count"`
	Contexts  []string `json:"contexts,omitempty"`
}

// maxContextsPerMatch caps the number of surrounding-sentence contexts kept
// per Match to avoid unbounded growth on long, mention-dense responses.
const maxContextsPerMatch = 3

// sentenceBoundary detects gaps between sentences: one or more of . ! ?
// followed by whitespace. It deliberately does NOT split on dots that are
// part of a word like "sageo.io", so domain-style brand terms are preserved.
var sentenceBoundary = regexp.MustCompile(`[.!?]+\s+`)

// DetectInResponses scans each AEO response for each term using
// case-insensitive whole-word matching. Results are deduplicated per
// (engine, model_name, prompt, term) tuple; a term mentioned N times in a
// single response produces one Match with Count=N and up to three
// surrounding-sentence contexts.
func DetectInResponses(responses []state.AEOPromptResult, terms []string) []Match {
	cleanTerms := make([]string, 0, len(terms))
	patterns := make([]*regexp.Regexp, 0, len(terms))
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		cleanTerms = append(cleanTerms, t)
		patterns = append(patterns, regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(t)+`\b`))
	}
	if len(patterns) == 0 {
		return nil
	}

	var out []Match
	for _, prompt := range responses {
		for _, r := range prompt.Results {
			spans := splitSentences(r.Response)
			for i, re := range patterns {
				hits := re.FindAllStringIndex(r.Response, -1)
				if len(hits) == 0 {
					continue
				}
				out = append(out, Match{
					Engine:    r.Engine,
					ModelName: r.ModelName,
					Prompt:    prompt.Prompt,
					Term:      cleanTerms[i],
					Count:     len(hits),
					Contexts:  buildContexts(spans, hits),
				})
			}
		}
	}
	return out
}

// sentenceSpan tracks the byte range a sentence covers inside the source
// response so a match offset can be mapped back to the sentence index.
type sentenceSpan struct {
	text       string
	start, end int // [start, end) in the original response
}

// splitSentences returns sentence spans covering s. Each span records its
// byte offsets so match indices can be located. Splits on terminal
// punctuation followed by whitespace; does NOT split mid-word (so "sageo.io"
// is preserved inside a single span).
func splitSentences(s string) []sentenceSpan {
	if s == "" {
		return nil
	}
	boundaries := sentenceBoundary.FindAllStringIndex(s, -1)
	var spans []sentenceSpan
	cursor := 0
	for _, b := range boundaries {
		text := strings.TrimSpace(s[cursor:b[1]])
		if text != "" {
			spans = append(spans, sentenceSpan{text: text, start: cursor, end: b[1]})
		}
		cursor = b[1]
	}
	if cursor < len(s) {
		text := strings.TrimSpace(s[cursor:])
		if text != "" {
			spans = append(spans, sentenceSpan{text: text, start: cursor, end: len(s)})
		}
	}
	return spans
}

// buildContexts returns up to maxContextsPerMatch context windows: each
// window is the sentence containing a hit joined with its two neighbours.
// Hits that land in the same sentence collapse to a single context.
func buildContexts(spans []sentenceSpan, hits [][]int) []string {
	if len(spans) == 0 {
		return nil
	}
	var out []string
	seenIdx := map[int]bool{}
	for _, h := range hits {
		if len(out) >= maxContextsPerMatch {
			break
		}
		idx := sentenceIndexFor(spans, h[0])
		if idx < 0 || seenIdx[idx] {
			continue
		}
		seenIdx[idx] = true
		out = append(out, window(spans, idx))
	}
	return out
}

// sentenceIndexFor returns the span index that contains byte offset off, or
// -1 if none does.
func sentenceIndexFor(spans []sentenceSpan, off int) int {
	for i, sp := range spans {
		if off >= sp.start && off < sp.end {
			return i
		}
	}
	return -1
}

// window returns the sentence at index i plus its immediate neighbours,
// joined by a single space. It yields up to three sentences of context.
func window(spans []sentenceSpan, i int) string {
	start := i - 1
	if start < 0 {
		start = 0
	}
	end := i + 2 // exclusive
	if end > len(spans) {
		end = len(spans)
	}
	texts := make([]string, 0, end-start)
	for _, sp := range spans[start:end] {
		texts = append(texts, sp.text)
	}
	return strings.Join(texts, " ")
}

// ToStateMatches converts Matches to the on-disk state representation.
func ToStateMatches(in []Match) []state.LocalMentionMatch {
	out := make([]state.LocalMentionMatch, 0, len(in))
	for _, m := range in {
		out = append(out, state.LocalMentionMatch{
			Engine:    m.Engine,
			ModelName: m.ModelName,
			Prompt:    m.Prompt,
			Term:      m.Term,
			Count:     m.Count,
			Contexts:  m.Contexts,
		})
	}
	return out
}
