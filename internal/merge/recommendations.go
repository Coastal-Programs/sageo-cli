package merge

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jakeschepis/sageo-cli/internal/recommendations"
	"github.com/jakeschepis/sageo-cli/internal/state"
)

// effortFor returns the hardcoded effort estimate (in minutes) for a ChangeType.
func effortFor(ct recommendations.ChangeType) int {
	switch ct {
	case recommendations.ChangeTitle, recommendations.ChangeMeta, recommendations.ChangeH1:
		return 10
	case recommendations.ChangeH2:
		return 15
	case recommendations.ChangeSchema, recommendations.ChangeIndexability:
		return 30
	case recommendations.ChangeInternalLink:
		return 15
	case recommendations.ChangeBody:
		return 120
	case recommendations.ChangeSpeed:
		return 240
	case recommendations.ChangeBacklink:
		return 60
	// AI-citation change types: small on-page edits, typically < 30 min.
	case recommendations.ChangeTLDR:
		return 25
	case recommendations.ChangeListFormat:
		return 30
	case recommendations.ChangeAuthorByline:
		return 20
	case recommendations.ChangeFreshness:
		return 15
	case recommendations.ChangeEntityConsistency:
		return 45
	default:
		return 30
	}
}

// mergedFindingID returns a stable identifier for a MergedFinding suitable
// for tracing a Recommendation back to its originating merged finding.
func mergedFindingID(f *MergedFinding) string {
	return fmt.Sprintf("%s:%s", f.Rule, f.URL)
}

// newRec builds a Recommendation with a stable ID and sane defaults.
func newRec(f *MergedFinding, targetURL, targetQuery string, ct recommendations.ChangeType) recommendations.Recommendation {
	return recommendations.Recommendation{
		ID:              recommendations.HashID(targetURL, targetQuery, ct),
		TargetURL:       targetURL,
		TargetQuery:     targetQuery,
		ChangeType:      ct,
		Priority:        f.PriorityScore,
		EffortMinutes:   effortFor(ct),
		MergedFindingID: mergedFindingID(f),
	}
}

// GenerateRecommendations converts MergedFindings into concrete
// Recommendation objects describing what to change on the site.
//
// Each rule dispatches to a dedicated generator that reads the relevant
// state (crawl findings, GSC row, PSI result, SERP result, backlinks) and
// may emit zero or more recommendations.
func GenerateRecommendations(st *state.State, findings []MergedFinding) []recommendations.Recommendation {
	if st == nil {
		return nil
	}

	// Index helpers (mirrors those built in Run).
	gscByKeyword := make(map[string]*state.GSCRow)
	if st.GSC != nil {
		for i := range st.GSC.TopKeywords {
			row := &st.GSC.TopKeywords[i]
			gscByKeyword[strings.ToLower(row.Key)] = row
		}
	}

	serpByQuery := make(map[string]*state.SERPQueryResult)
	if st.SERP != nil {
		for i := range st.SERP.Queries {
			q := &st.SERP.Queries[i]
			serpByQuery[strings.ToLower(q.Query)] = q
		}
	}

	findingsByURL := make(map[string][]state.Finding)
	for _, cf := range st.Findings {
		findingsByURL[cf.URL] = append(findingsByURL[cf.URL], cf)
	}

	var out []recommendations.Recommendation
	for i := range findings {
		f := &findings[i]
		switch f.Rule {
		case "ranking-but-not-clicking":
			out = append(out, recsRankingButNotClicking(f, findingsByURL)...)
		case "not-indexed":
			out = append(out, recsNotIndexed(f, findingsByURL)...)
		case "issues-on-high-traffic-page":
			out = append(out, recsIssuesOnHighTrafficPage(f, findingsByURL)...)
		case "thin-content-ranking-well":
			out = append(out, recsThinContentRankingWell(f, serpByQuery, gscByKeyword)...)
		case "schema-not-showing":
			out = append(out, recsSchemaNotShowing(f)...)
		case "slow-core-web-vitals":
			out = append(out, recsSlowCoreWebVitals(f, st)...)
		case "ai-overview-eating-clicks":
			out = append(out, recsAIOverviewEatingClicks(f, serpByQuery)...)
		case "featured-snippet-opportunity":
			out = append(out, recsFeaturedSnippetOpportunity(f)...)
		case "paa-content-opportunity":
			out = append(out, recsPAAContentOpportunity(f, serpByQuery)...)
		case "easy-win-keyword":
			out = append(out, recsEasyWinKeyword(f)...)
		case "informational-content-gap":
			out = append(out, recsInformationalContentGap(f, st)...)
		case "weak-backlink-profile":
			out = append(out, recsWeakBacklinkProfile(f, st)...)
		case "broken-backlinks-found":
			out = append(out, recsBrokenBacklinksFound(f, st)...)
		case "missing-author-signals":
			out = append(out, recsMissingAuthorSignals(f)...)
		}
	}
	return out
}

// ─── per-rule generators ─────────────────────────────────────────────────────

func recsRankingButNotClicking(f *MergedFinding, findingsByURL map[string][]state.Finding) []recommendations.Recommendation {
	rationale := "Page ranks (has GSC impressions) but receives no clicks — title/meta are the primary levers for CTR."
	if f.GSCData != nil {
		rationale = fmt.Sprintf(
			"Page has %.0f impressions at avg position %.1f but CTR is %.2f%% — rewrite title and meta to lift CTR.",
			f.GSCData.Impressions, f.GSCData.Position, f.GSCData.CTR*100,
		)
	}
	currentTitle, currentMeta := currentTitleMeta(findingsByURL[f.URL])

	title := newRec(f, f.URL, "", recommendations.ChangeTitle)
	title.CurrentValue = currentTitle
	title.Rationale = rationale
	title.Evidence = gscEvidence(f.GSCData)

	meta := newRec(f, f.URL, "", recommendations.ChangeMeta)
	meta.CurrentValue = currentMeta
	meta.Rationale = rationale
	meta.Evidence = gscEvidence(f.GSCData)

	return []recommendations.Recommendation{title, meta}
}

func recsNotIndexed(f *MergedFinding, findingsByURL map[string][]state.Finding) []recommendations.Recommendation {
	// Describe the blocker based on crawl findings on this URL.
	blocker := "Page absent from GSC and no noindex directive was found — likely blocked by robots.txt, canonical mismatch, or a crawl trap."
	for _, cf := range findingsByURL[f.URL] {
		switch {
		case strings.Contains(cf.Rule, "robots"):
			blocker = "robots.txt disallow is blocking indexing."
		case strings.Contains(cf.Rule, "canonical"):
			blocker = "Canonical tag points to a different URL, so Google may be consolidating signals away from this page."
		case strings.Contains(cf.Rule, "redirect"):
			blocker = "Page resolves through a redirect chain — fix the redirect target."
		}
	}
	r := newRec(f, f.URL, "", recommendations.ChangeIndexability)
	r.Rationale = blocker
	r.Evidence = []recommendations.Evidence{
		{Source: "crawl", Metric: "crawl_issues", Value: f.CrawlIssues, Description: "crawl rule hits on this URL"},
		{Source: "gsc", Metric: "impressions", Value: 0, Description: "page absent from GSC top-pages"},
	}
	return []recommendations.Recommendation{r}
}

func recsIssuesOnHighTrafficPage(f *MergedFinding, findingsByURL map[string][]state.Finding) []recommendations.Recommendation {
	seen := make(map[recommendations.ChangeType]bool)
	var out []recommendations.Recommendation
	for _, cf := range findingsByURL[f.URL] {
		ct, ok := changeTypeForCrawlRule(cf.Rule)
		if !ok || seen[ct] {
			continue
		}
		seen[ct] = true
		r := newRec(f, f.URL, "", ct)
		r.Rationale = fmt.Sprintf("High-traffic page has crawl finding %q that should be fixed to protect existing traffic.", cf.Rule)
		if s, sok := cf.Value.(string); sok {
			r.CurrentValue = s
		}
		r.Evidence = append(gscEvidence(f.GSCData), recommendations.Evidence{
			Source: "crawl", Metric: cf.Rule, Value: cf.Value, Description: cf.Why,
		})
		out = append(out, r)
	}
	return out
}

func recsThinContentRankingWell(f *MergedFinding, serpByQuery map[string]*state.SERPQueryResult, gscByKeyword map[string]*state.GSCRow) []recommendations.Recommendation {
	// Target word count = max(top-3 SERP competitors × 1.1, 800). We don't
	// store competitor word counts, so use 800 as the floor and note the
	// heuristic in the rationale.
	targetWords := 800
	r := newRec(f, f.URL, "", recommendations.ChangeBody)
	r.Rationale = fmt.Sprintf(
		"Page has fewer than 300 words but ranks in the top 10 — expand to at least %d words to defend and grow the ranking.",
		targetWords,
	)
	r.RecommendedValue = fmt.Sprintf("target_word_count=%d", targetWords)
	r.Evidence = gscEvidence(f.GSCData)
	return []recommendations.Recommendation{r}
}

// recsSchemaNotShowing recommends Tier-1 structured data types that are
// confirmed (not just correlated) to influence Google's AI surfaces.
//
// Per docs/research/ai-citation-signals-2026.md signals matrix:
//   - Organization / Article / BreadcrumbList / Product / LocalBusiness
//     are "confirmed" for Google AI Overviews.
//   - FAQPage is "unclear" across every engine (Averi +28%, SE Ranking
//     slight negative, Search Atlas no effect) — so we no longer emit it
//     as a generic schema-not-showing fix; it is reserved for the
//     AI-Overview rule where Google's pipeline specifically indexes Q&A
//     markup (research doc "Conflicts & open questions", FAQPage entry).
//   - HowTo is dropped: Google removed HowTo rich results in Aug 2023
//     and no cross-engine AI-citation evidence exists (research doc
//     "Drop").
func recsSchemaNotShowing(f *MergedFinding) []recommendations.Recommendation {
	schemaType := schemaTypeForURL(f.URL)
	r := newRec(f, f.URL, "", recommendations.ChangeSchema)
	r.RecommendedValue = schemaType
	r.Rationale = fmt.Sprintf(
		"Schema types declared but CTR is low — add Tier-1 %s structured data and validate with the Rich Results Test. %s is confirmed to influence Google AI Overviews (developers.google.com/search/docs/appearance/structured-data/search-gallery).",
		schemaType, schemaType,
	)
	r.Evidence = gscEvidence(f.GSCData)
	return []recommendations.Recommendation{r}
}

func recsSlowCoreWebVitals(f *MergedFinding, st *state.State) []recommendations.Recommendation {
	r := newRec(f, f.URL, "", recommendations.ChangeSpeed)
	r.Rationale = f.Fix
	// Look up PSI result for this URL to name the slowest metric.
	if st.PSI != nil {
		for i := range st.PSI.Pages {
			p := &st.PSI.Pages[i]
			if p.URL == f.URL {
				sm := slowestMetric(p)
				r.Rationale = fmt.Sprintf("Slowest metric: %s — %s", sm.name, sm.detail)
				r.RecommendedValue = sm.name
				r.Evidence = []recommendations.Evidence{
					{Source: "psi", Metric: "performance_score", Value: p.PerformanceScore},
					{Source: "psi", Metric: "lcp_ms", Value: p.LCP},
					{Source: "psi", Metric: "cls", Value: p.CLS},
				}
				break
			}
		}
	}
	return []recommendations.Recommendation{r}
}

// recsAIOverviewEatingClicks emits the stack of changes most associated
// with being cited by (rather than buried beneath) a Google AI Overview.
//
// Evidence basis (docs/research/ai-citation-signals-2026.md):
//   - ChangeTLDR: 44.2% of ChatGPT citations come from the first 30% of
//     an article (Growth Memo, 18,012 citations, §B.1.2); direct-answer
//     intros are the strongest on-page lever.
//   - ChangeListFormat: lists/tables mark "likely" across Google / ChatGPT /
//     Perplexity in the signals matrix; Averi and Growth Memo show
//     list/table passages extract more reliably into AI answers.
//   - ChangeSchema (FAQPage): weakly supported by ChatGPT Search and
//     Perplexity ("unclear") but Google's own pipeline indexes Q&A markup
//     for AI Overviews (research doc "Conflicts & open questions",
//     FAQPage entry). Rationale notes the asymmetry.
//   - ChangeAuthorByline: E-E-A-T signal (google-ai-overviews.md §4) plus
//     Perplexity trust signals (schema-and-technical.md §5 Tier 1).
//   - H2s per PAA question remain useful for passage-level extraction
//     across Perplexity and ChatGPT.
func recsAIOverviewEatingClicks(f *MergedFinding, serpByQuery map[string]*state.SERPQueryResult) []recommendations.Recommendation {
	// f.URL here is the GSC keyword (the query), since the rule uses TopKeywords rows.
	query := f.URL
	sq := serpByQuery[strings.ToLower(query)]

	tldr := newRec(f, f.URL, query, recommendations.ChangeTLDR)
	tldr.RecommendedValue = "tldr_40_70_words_top_of_page"
	tldr.Rationale = "AI Overview is absorbing clicks — add a 40-70 word direct-answer block at the top of the page. 44.2% of ChatGPT citations come from the first 30% of an article (Growth Memo, 18,012 citations)."
	tldr.Evidence = gscEvidence(f.GSCData)

	lists := newRec(f, f.URL, query, recommendations.ChangeListFormat)
	lists.RecommendedValue = "convert_to_list_or_table"
	lists.Rationale = "Convert the core answer to a numbered list, bullet list, or comparison table — list/table passages extract more reliably into AI Overviews, ChatGPT Search, and Perplexity (signals matrix, Averi / Growth Memo)."

	schema := newRec(f, f.URL, query, recommendations.ChangeSchema)
	schema.RecommendedValue = "FAQPage"
	schema.Rationale = "Add FAQPage structured data. Evidence is weak for direct LLM extraction (ChatGPT/Perplexity “unclear” in signals matrix) but Google's pipeline indexes Q&A markup for AI Overviews — marginal lift on Google, neutral elsewhere."

	author := newRec(f, f.URL, query, recommendations.ChangeAuthorByline)
	author.RecommendedValue = "visible_byline_with_credentials_and_bio_link"
	author.Rationale = "Add a visible author byline with credentials and a linked bio (Person schema with sameAs to Wikipedia/Wikidata if available). AI Overviews weight E-E-A-T author signals (Google Helpful Content guidance); Perplexity trust signals reward named authorship."

	out := []recommendations.Recommendation{tldr, lists, schema, author}

	// Emit H2 recs per PAA question (up to 5) so the page covers the
	// ground the AI Overview is summarising — H2 subheadings anchor
	// passage-level extraction across Perplexity and ChatGPT.
	if sq != nil {
		limit := 5
		for i, q := range sq.RelatedQuestions {
			if i >= limit {
				break
			}
			h2 := newRec(f, f.URL, q, recommendations.ChangeH2)
			h2.RecommendedValue = q
			h2.Rationale = fmt.Sprintf("Add an H2 answering %q to be cited by the AI Overview.", q)
			out = append(out, h2)
		}
	}
	return out
}

// recsFeaturedSnippetOpportunity emits a ChangeTLDR recommendation
// specifically scoped to a 40-60 word definition-style answer.
//
// Featured Snippets are still rendered from a short passage near the top
// of a ranking page. Combined with the AI-Overview direct-answer research
// (perplexity-and-industry.md §B.1.2), the same TL;DR passage both wins
// the snippet and satisfies AI extraction, so ChangeTLDR is the correct
// emitted type rather than a generic ChangeBody rewrite.
func recsFeaturedSnippetOpportunity(f *MergedFinding) []recommendations.Recommendation {
	query := f.URL
	r := newRec(f, f.URL, query, recommendations.ChangeTLDR)
	r.RecommendedValue = "definition_tldr_40_60_words"
	r.Rationale = fmt.Sprintf(
		"You rank in the top 10 for %q and a Featured Snippet exists — add a definition-style TL;DR (40-60 words) at the top of the page that directly answers the query. The same passage also serves AI Overview / ChatGPT citation (44.2%% of ChatGPT citations come from the first 30%% of an article, Growth Memo 2026).",
		query,
	)
	return []recommendations.Recommendation{r}
}

func recsPAAContentOpportunity(f *MergedFinding, serpByQuery map[string]*state.SERPQueryResult) []recommendations.Recommendation {
	query := f.URL
	sq := serpByQuery[strings.ToLower(query)]
	if sq == nil {
		return nil
	}
	limit := 5
	var out []recommendations.Recommendation
	for i, q := range sq.RelatedQuestions {
		if i >= limit {
			break
		}
		r := newRec(f, f.URL, q, recommendations.ChangeH2)
		r.RecommendedValue = q
		r.Rationale = fmt.Sprintf("Add an H2 answering the People Also Ask question %q.", q)
		out = append(out, r)
	}
	return out
}

func recsEasyWinKeyword(f *MergedFinding) []recommendations.Recommendation {
	// f.URL is the keyword string (from TopKeywords row).
	keyword := f.URL

	title := newRec(f, f.URL, keyword, recommendations.ChangeTitle)
	title.RecommendedValue = keyword
	title.Rationale = fmt.Sprintf("Easy-win keyword %q — incorporate verbatim into the page title.", keyword)
	title.Evidence = gscEvidence(f.GSCData)

	h1 := newRec(f, f.URL, keyword, recommendations.ChangeH1)
	h1.RecommendedValue = keyword
	h1.Rationale = fmt.Sprintf("Easy-win keyword %q — incorporate verbatim into the H1.", keyword)

	return []recommendations.Recommendation{title, h1}
}

func recsInformationalContentGap(f *MergedFinding, st *state.State) []recommendations.Recommendation {
	keyword := f.URL
	base := strings.TrimRight(st.Site, "/")
	if base == "" {
		base = ""
	}
	slug := slugify(keyword)
	target := base + "/" + slug

	r := newRec(f, target, keyword, recommendations.ChangeBody)
	r.CurrentValue = ""
	r.Rationale = fmt.Sprintf(
		"No existing page targets %q — create a new page at %s covering the query comprehensively.",
		keyword, target,
	)
	r.RecommendedValue = fmt.Sprintf("new_page_slug=%s", slug)
	return []recommendations.Recommendation{r}
}

func recsWeakBacklinkProfile(f *MergedFinding, st *state.State) []recommendations.Recommendation {
	if st.Backlinks == nil {
		return nil
	}
	// Top-N outreach targets from gap data.
	const topN = 5
	targets := st.Backlinks.GapDomains
	if len(targets) > topN {
		targets = targets[:topN]
	}
	if len(targets) == 0 {
		// No gap data — emit a single generic rec on the site itself.
		r := newRec(f, f.URL, "", recommendations.ChangeBacklink)
		r.Rationale = "Backlink profile is weak relative to keyword difficulty. Run `sageo backlinks gap` to identify outreach targets."
		return []recommendations.Recommendation{r}
	}
	var out []recommendations.Recommendation
	for _, domain := range targets {
		r := newRec(f, f.URL, domain, recommendations.ChangeBacklink)
		r.RecommendedValue = domain
		r.Rationale = fmt.Sprintf("Outreach target: %s — competitors have a link from this domain but you don't.", domain)
		r.Evidence = []recommendations.Evidence{
			{Source: "backlinks", Metric: "gap_domain", Value: domain},
		}
		out = append(out, r)
	}
	return out
}

func recsBrokenBacklinksFound(f *MergedFinding, st *state.State) []recommendations.Recommendation {
	r := newRec(f, f.URL, "", recommendations.ChangeBacklink)
	r.Rationale = f.Fix
	if st.Backlinks != nil {
		r.Evidence = []recommendations.Evidence{
			{Source: "backlinks", Metric: "broken_backlinks", Value: st.Backlinks.BrokenBacklinks},
		}
	}
	return []recommendations.Recommendation{r}
}

// recsMissingAuthorSignals recommends adding a visible author byline,
// credentials, a bio link, and Person schema to a page.
//
// Evidence (docs/research/ai-citation-signals-2026.md):
//   - Google E-E-A-T framework (google-ai-overviews.md §4) explicitly
//     weights author expertise signals.
//   - Signals matrix "Explicit author byline + credentials" is marked
//     "likely" for Google AI Overviews and Perplexity.
//   - Perplexity Tier-1 Person schema (schema-and-technical.md §5).
func recsMissingAuthorSignals(f *MergedFinding) []recommendations.Recommendation {
	byline := newRec(f, f.URL, "", recommendations.ChangeAuthorByline)
	byline.RecommendedValue = "visible_byline_with_credentials_and_bio_link"
	byline.Rationale = "Page has no visible author byline. Add author name + credentials + a linked bio page. AI Overviews weight E-E-A-T author signals and Perplexity trust signals reward named authorship (signals matrix row “Explicit author byline + credentials”: likely for Google AIO and Perplexity)."
	byline.Evidence = gscEvidence(f.GSCData)

	person := newRec(f, f.URL, "", recommendations.ChangeSchema)
	person.RecommendedValue = "Person"
	person.Rationale = "Add Person structured data for the byline author with sameAs links to Wikipedia/Wikidata/LinkedIn where available (schema-and-technical.md §5 Tier-1 Person)."

	return []recommendations.Recommendation{byline, person}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func gscEvidence(m *GSCMetrics) []recommendations.Evidence {
	if m == nil {
		return nil
	}
	return []recommendations.Evidence{
		{Source: "gsc", Metric: "impressions", Value: m.Impressions},
		{Source: "gsc", Metric: "clicks", Value: m.Clicks},
		{Source: "gsc", Metric: "ctr", Value: m.CTR},
		{Source: "gsc", Metric: "position", Value: m.Position},
	}
}

// currentTitleMeta pulls any existing title/meta text out of crawl findings
// for the given URL. Returns empty strings when not available.
func currentTitleMeta(fs []state.Finding) (title, meta string) {
	for _, cf := range fs {
		switch cf.Rule {
		case "title-too-long", "title-too-short":
			if s, ok := cf.Value.(string); ok {
				title = s
			}
		case "meta-too-long", "meta-too-short":
			if s, ok := cf.Value.(string); ok {
				meta = s
			}
		}
	}
	return title, meta
}

// changeTypeForCrawlRule maps a crawl rule name to the ChangeType that would
// address it. The second return is false when the rule doesn't map to a
// specific change (so no recommendation is emitted).
func changeTypeForCrawlRule(rule string) (recommendations.ChangeType, bool) {
	switch {
	case rule == "missing-title",
		rule == "title-too-long",
		rule == "title-too-short",
		rule == "duplicate-title":
		return recommendations.ChangeTitle, true
	case rule == "missing-meta-description",
		rule == "meta-too-long",
		rule == "meta-too-short":
		return recommendations.ChangeMeta, true
	case rule == "missing-h1", rule == "multiple-h1":
		return recommendations.ChangeH1, true
	case rule == "thin-content", rule == "low-word-count":
		return recommendations.ChangeBody, true
	case strings.HasPrefix(rule, "schema-"):
		return recommendations.ChangeSchema, true
	case rule == "noindex-detected", rule == "x-robots-noindex":
		return recommendations.ChangeIndexability, true
	}
	return "", false
}

// schemaTypeForURL applies a simple heuristic to recommend a schema type
// based on the URL path.
func schemaTypeForURL(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "/faq"), strings.Contains(lower, "/questions"):
		return "FAQPage"
	case strings.Contains(lower, "/contact"), strings.Contains(lower, "/location"), strings.Contains(lower, "/stores"):
		return "LocalBusiness"
	case strings.Contains(lower, "/product"), strings.Contains(lower, "/shop"):
		return "Product"
	default:
		return "Article"
	}
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a keyword into a URL-safe slug.
func slugify(s string) string {
	lower := strings.ToLower(s)
	slug := slugRe.ReplaceAllString(lower, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "new-page"
	}
	return slug
}

// SortByPriorityDesc returns recs sorted by Priority descending (stable).
func SortByPriorityDesc(recs []recommendations.Recommendation) []recommendations.Recommendation {
	out := make([]recommendations.Recommendation, len(recs))
	copy(out, recs)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Priority > out[j].Priority
	})
	return out
}
