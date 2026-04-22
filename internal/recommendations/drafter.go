package recommendations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/llm"
)

// Length limits enforced on LLM-generated copy. These are industry defaults
// that balance SERP truncation and on-page rendering.
const (
	MaxTitleChars = 60
	MaxMetaChars  = 155
	MaxH2Chars    = 80
)

// PageContext is the subset of on-page and SERP data supplied to the LLM
// when drafting concrete copy. Fields are best-effort — callers should fill
// what they have.
type PageContext struct {
	Title               string
	MetaDescription     string
	H1                  string
	H2s                 []string
	BodyExcerpt         string
	TopCompetitorTitles []string
	TopPAAQuestions     []string
	TargetKeywords      []string
}

// draftAttempts is the number of times we'll re-ask the LLM if the output
// violates length constraints. A single retry is usually enough.
const draftAttempts = 2

// Draft asks the provider for concrete copy for rec and writes it to
// rec.RecommendedValue. Callers provide the PageContext from state so the
// LLM has real evidence to work with.
func Draft(ctx context.Context, p llm.Provider, rec *Recommendation, page PageContext) error {
	if rec == nil {
		return fmt.Errorf("draft: recommendation is nil")
	}
	if p == nil {
		return fmt.Errorf("draft: provider is nil")
	}

	system, user, validator := buildPrompt(rec, page)
	if system == "" && user == "" {
		// Unsupported ChangeType — leave RecommendedValue untouched.
		return nil
	}

	var lastText string
	var lastErr error
	for attempt := 0; attempt < draftAttempts; attempt++ {
		userPrompt := user
		if attempt > 0 && lastText != "" {
			userPrompt = user + "\n\nYour previous answer violated a constraint:\n" + lastText + "\n\nTry again and respect the length limits exactly."
		}
		resp, err := p.Complete(ctx, llm.CompletionRequest{
			System:      system,
			User:        userPrompt,
			MaxTokens:   800,
			Temperature: 0.4,
		})
		if err != nil {
			return fmt.Errorf("draft: llm call: %w", err)
		}
		text := strings.TrimSpace(resp.Text)
		text = stripCodeFences(text)
		if err := validator(text); err != nil {
			lastText = text
			lastErr = err
			continue
		}
		rec.RecommendedValue = text
		// Every LLM-drafted value lands in the review queue. Callers must
		// explicitly approve / edit before the value is treated as
		// ready-to-ship. See internal/state.ReviewStatus.
		rec.ReviewStatus = ReviewPending
		rec.OriginalDraft = text
		rec.ReviewedAt = nil
		rec.ReviewedBy = ""
		rec.ReviewNotes = ""
		return nil
	}
	return fmt.Errorf("draft: output failed validation after %d attempts: %w", draftAttempts, lastErr)
}

type validatorFn func(string) error

func buildPrompt(rec *Recommendation, page PageContext) (system, user string, validate validatorFn) {
	evidence := formatEvidence(rec.Evidence)
	keywords := strings.Join(page.TargetKeywords, ", ")
	if rec.TargetQuery != "" {
		if keywords == "" {
			keywords = rec.TargetQuery
		} else {
			keywords = rec.TargetQuery + ", " + keywords
		}
	}

	base := fmt.Sprintf(`Target URL: %s
Target keywords: %s
Current title: %q
Current meta description: %q
Current H1: %q
Current H2s: %s
Body excerpt: %s
Top competitor titles: %s
People Also Ask: %s

Evidence supporting this recommendation:
%s`,
		rec.TargetURL,
		keywords,
		page.Title,
		page.MetaDescription,
		page.H1,
		joinOrNone(page.H2s),
		excerpt(page.BodyExcerpt, 600),
		joinOrNone(page.TopCompetitorTitles),
		joinOrNone(page.TopPAAQuestions),
		evidence,
	)

	switch rec.ChangeType {
	case ChangeTitle:
		return "You are an SEO copywriter. Return ONLY the new page title as plain text — no quotes, no prefix, no explanation.",
			fmt.Sprintf(`Write a new <title> tag for this page.

Constraints:
- Maximum %d characters (hard limit — stay under it).
- Front-load the primary keyword.
- Natural, human phrasing. No clickbait, no emojis.
- No trailing brand unless it fits in the character budget.

%s`, MaxTitleChars, base),
			lengthValidator("title", MaxTitleChars)

	case ChangeMeta:
		return "You are an SEO copywriter. Return ONLY the new meta description as plain text — no quotes, no prefix, no explanation.",
			fmt.Sprintf(`Write a new <meta name="description"> for this page.

Constraints:
- Maximum %d characters (hard limit — stay under it).
- Include the primary keyword once, naturally.
- End with a concrete call-to-action or benefit.
- Write in active voice, second person where natural.

%s`, MaxMetaChars, base),
			lengthValidator("meta description", MaxMetaChars)

	case ChangeH1:
		return "You are an SEO copywriter. Return ONLY the new H1 heading as plain text — no quotes, no prefix, no markdown.",
			fmt.Sprintf(`Write a new H1 for this page.

Constraints:
- Maximum %d characters.
- Contains the primary keyword.
- Distinct from the <title> (don't just duplicate it).

%s`, MaxTitleChars+20, base),
			lengthValidator("h1", MaxTitleChars+20)

	case ChangeH2:
		return "You are an SEO strategist. Return ONLY the new H2 heading as plain text — no quotes, no markdown, no explanation.",
			fmt.Sprintf(`Propose a new H2 subheading to add to this page. Choose the single most valuable subheading based on the People Also Ask questions and competitor titles.

Constraints:
- Maximum %d characters.
- Phrased as either a question or a descriptive noun phrase.
- Semantically complements the existing H2s rather than duplicating them.

%s`, MaxH2Chars, base),
			lengthValidator("h2", MaxH2Chars)

	case ChangeSchema:
		return "You are a technical SEO engineer. Return ONLY a valid JSON-LD object as a JSON literal — no markdown, no prose, no <script> tag.",
			fmt.Sprintf(`Produce a JSON-LD schema block for this page.

Constraints:
- Must be valid JSON (parseable).
- Must include "@context": "https://schema.org" and an appropriate "@type".
- Fill as many properties as the evidence supports; do not invent facts.
- Do NOT wrap in a <script> tag.

%s`, base),
			jsonValidator()

	case ChangeBody:
		return "You are an expert content writer. Return ONLY the new body paragraph(s) as plain text — no headings, no markdown, no preamble.",
			fmt.Sprintf(`Write 1–3 short paragraphs (200–450 words total) to add to this page. Address the People Also Ask questions and fill the topical gap versus competitors.

Constraints:
- Plain prose. No bullet lists, no markdown.
- Use the primary keyword once in the first sentence, variations thereafter.
- Specific, factual, not promotional.

%s`, base),
			wordCountValidator(150, 600)

	case ChangeInternalLink:
		return "You are an SEO strategist. Return ONLY the suggested anchor text as plain text — no quotes, no URL, no explanation.",
			fmt.Sprintf(`Propose the anchor text for a new internal link pointing to this page.

Constraints:
- Maximum 70 characters.
- Descriptive, keyword-aware, not generic ("click here" is forbidden).

%s`, base),
			lengthValidator("anchor", 70)

	case ChangeTLDR:
		// Evidence: 44.2% of ChatGPT citations come from the first 30% of
		// an article (Growth Memo 2026); direct-answer intros are the
		// strongest on-page lever for AI citation. The prompt therefore
		// enforces a 40-70 word budget, target query in the first sentence,
		// and declarative (not hedged) phrasing.
		return "You are writing a TL;DR block optimised for extraction by Google AI Overviews, ChatGPT Search, and Perplexity. Return ONLY the TL;DR paragraph as plain text — no heading, no markdown, no preamble.",
			fmt.Sprintf(`Write a 40-70 word TL;DR block to place at the very top of this page.

Constraints:
- 40-70 words, hard limit.
- First sentence must state the direct answer to the target query (%q) declaratively — no hedging, no "it depends", no throat-clearing.
- Second and third sentences add the two most important caveats or qualifiers.
- Plain prose, self-contained, citable as a single passage.
- No lists, no markdown, no citations.

%s`, rec.TargetQuery, base),
			wordCountValidator(40, 70)

	case ChangeListFormat:
		// Evidence: lists/tables mark "likely" across Google, ChatGPT, and
		// Perplexity in the signals matrix; passages formatted as lists
		// extract more reliably into AI answers. The prompt asks for a
		// concrete ordered or unordered list the author can paste in.
		return "You are restructuring a prose answer into a list/table format optimised for AI extraction. Return ONLY the list as plain text (one item per line, starting with \"- \"). No heading, no markdown code fences, no preamble.",
			fmt.Sprintf(`Propose a 3-7 item list that restructures the core answer on this page. Each item should be a self-contained claim extractable as a single sentence — the format that performs best for AI Overviews, ChatGPT Search, and Perplexity citation.

Constraints:
- 3-7 items, one per line, prefixed with "- ".
- Each item 8-25 words.
- Do not invent facts beyond the evidence and page context.
- No markdown, no numbering, no trailing commentary.

%s`, base),
			listValidator(3, 7)

	case ChangeAuthorByline:
		// Evidence: E-E-A-T (Google Helpful Content) + Perplexity trust
		// signals. The prompt produces a short visible byline string.
		return "You are drafting a visible author byline for E-E-A-T and AI-citation trust signals. Return ONLY the byline as plain text (single line). No markdown, no quotes.",
			fmt.Sprintf(`Write a short visible author byline for this page.

Constraints:
- Maximum 120 characters.
- Format: "By <Name>, <Credential or role>" — e.g. "By Jane Doe, MD, Board-Certified Endocrinologist".
- Use the real author if identifiable from the page / evidence; otherwise propose a credential-bearing placeholder clearly marked with <AUTHOR_NAME>.
- No marketing copy, no adjectives like "award-winning" unless the evidence supports it.

%s`, base),
			lengthValidator("byline", 120)

	case ChangeFreshness:
		// Evidence: AI-cited URLs ~26% fresher than organic SERPs; 30-day
		// refresh window drives 3.2× Perplexity citation lift. The prompt
		// returns a short, concrete refresh plan (what to update + visible
		// "Updated" string).
		return "You are planning a content-refresh edit for AI-citation freshness signals. Return ONLY the refresh plan as 2-4 short plain-text lines (one per action). No markdown, no preamble.",
			fmt.Sprintf(`Propose a 2-4 line refresh plan for this page to restore freshness signals for AI Overviews, ChatGPT Search, and Perplexity.

Constraints:
- 2-4 lines, one concrete action per line.
- Include an explicit visible "Updated: <Month YYYY>" string that the author should add near the byline.
- Name the specific sections/facts to update, based on the body excerpt and evidence.
- Do not propose wholesale rewrites — the goal is a visible, accurate freshness refresh, not a content overhaul.

%s`, base),
			lineCountValidator(2, 4)

	case ChangeEntityConsistency:
		// Evidence: brand mentions outperform backlinks as a predictor of
		// AI citation (Ahrefs 75K brands, SE Ranking 129K domains, Growth
		// Memo). The prompt returns a short consistency checklist the
		// author can execute.
		return "You are drafting an entity-consistency checklist for brand NAP and sameAs alignment across the page, Organization schema, and external sources. Return ONLY the checklist as 3-6 short plain-text lines (one per action). No markdown, no preamble.",
			fmt.Sprintf(`Propose a 3-6 line entity-consistency checklist for this page.

Constraints:
- 3-6 lines, one concrete action per line.
- Cover: (a) visible brand name/NAP on the page, (b) Organization schema sameAs links (Wikipedia, Wikidata, LinkedIn, Crunchbase), (c) alignment with any visible parent/child brand references.
- Do not invent external URLs — where a sameAs target is unknown, write "sameAs <FIND_URL for Wikipedia/Wikidata/LinkedIn>".

%s`, base),
			lineCountValidator(3, 6)

	default:
		// Non-copy change types (speed_fix, backlink_outreach,
		// indexability_fix) — drafting copy is not meaningful.
		return "", "", nil
	}
}

// listValidator accepts a "- "-prefixed list with a bounded item count.
func listValidator(minItems, maxItems int) validatorFn {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("list: empty output")
		}
		n := 0
		for _, line := range strings.Split(s, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if !strings.HasPrefix(trimmed, "- ") {
				return fmt.Errorf("list: line %q missing \"- \" prefix", trimmed)
			}
			n++
		}
		if n < minItems {
			return fmt.Errorf("list: %d items below min %d", n, minItems)
		}
		if n > maxItems {
			return fmt.Errorf("list: %d items above max %d", n, maxItems)
		}
		return nil
	}
}

// lineCountValidator accepts a plain-text block with a bounded line count.
func lineCountValidator(minLines, maxLines int) validatorFn {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("lines: empty output")
		}
		n := 0
		for _, line := range strings.Split(s, "\n") {
			if strings.TrimSpace(line) != "" {
				n++
			}
		}
		if n < minLines {
			return fmt.Errorf("lines: %d below min %d", n, minLines)
		}
		if n > maxLines {
			return fmt.Errorf("lines: %d above max %d", n, maxLines)
		}
		return nil
	}
}

func lengthValidator(label string, maxChars int) validatorFn {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("%s: empty output", label)
		}
		if len(s) > maxChars {
			return fmt.Errorf("%s: %d chars exceeds max %d", label, len(s), maxChars)
		}
		return nil
	}
}

func jsonValidator() validatorFn {
	return func(s string) error {
		s = strings.TrimSpace(s)
		if s == "" {
			return fmt.Errorf("schema: empty output")
		}
		var v any
		if err := json.Unmarshal([]byte(s), &v); err != nil {
			return fmt.Errorf("schema: invalid json: %w", err)
		}
		return nil
	}
}

func wordCountValidator(minWords, maxWords int) validatorFn {
	return func(s string) error {
		n := len(strings.Fields(s))
		if n < minWords {
			return fmt.Errorf("body: %d words below min %d", n, minWords)
		}
		if n > maxWords {
			return fmt.Errorf("body: %d words above max %d", n, maxWords)
		}
		return nil
	}
}

func formatEvidence(ev []Evidence) string {
	if len(ev) == 0 {
		return "(none)"
	}
	var lines []string
	for _, e := range ev {
		line := fmt.Sprintf("- [%s] %s", e.Source, e.Metric)
		if e.Value != nil {
			line += fmt.Sprintf(" = %v", e.Value)
		}
		if e.Description != "" {
			line += " — " + e.Description
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, " | ")
}

func excerpt(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(none)"
	}
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// stripCodeFences removes triple-backtick fences that LLMs sometimes wrap
// around plain text even when told not to.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	// drop first line
	if idx := strings.Index(s, "\n"); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
