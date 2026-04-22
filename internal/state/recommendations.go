package state

import "time"

// ChangeType enumerates the kinds of on-site changes a Recommendation can
// describe.
//
// Evidence weights for each ChangeType are drawn from
// docs/research/ai-citation-signals-2026.md ("Recommendations for sageo's
// rule set"). Types added after the 2026-04-22 synthesis cite the specific
// signals-matrix row or per-engine note that supports them.
type ChangeType string

const (
	// ChangeTitle — Anthropic's citation schema makes the page title
	// load-bearing (title is a required field in Messages API citation
	// objects) and Google's Helpful Content doc calls out descriptive
	// headings. Research doc "Keep (well-supported)" §ChangeTitle.
	ChangeTitle ChangeType = "title"

	// ChangeMeta — No primary source confirms meta description influences
	// AI citation; Google treats it as a snippet hint only. Kept but
	// demoted below title / H-tags / body. Research doc "Demote (weak
	// evidence)" §ChangeMeta.
	ChangeMeta ChangeType = "meta_description"

	// ChangeH1 — Semantic heading hierarchy improves passage-level
	// retrieval across RAG systems (schema-and-technical.md §5). Research
	// doc "Keep (well-supported)" §ChangeH1 / ChangeH2.
	ChangeH1 ChangeType = "h1"

	// ChangeH2 — Same basis as ChangeH1; H2s anchor the passage-level
	// extraction used by Perplexity and ChatGPT. Research doc
	// "Keep (well-supported)" §ChangeH1 / ChangeH2.
	ChangeH2 ChangeType = "h2_add"

	// ChangeSchema — Confirmed for Google AI Overviews and Bing Copilot;
	// correlates with citation elsewhere but causal mechanism disputed.
	// Scope to Tier-1 types (Organization, Article, BreadcrumbList,
	// Person). Research doc "Keep (well-supported)" §ChangeSchema.
	ChangeSchema ChangeType = "schema_add"

	// ChangeBody — Generic body expansion / rewrite. Kept for thin-content
	// and word-count gaps. Direct-answer formatting is expressed via
	// ChangeTLDR; list/table conversion via ChangeListFormat. Research doc
	// "Keep (well-supported)" §ChangeBody.
	ChangeBody ChangeType = "body_expand"

	// ChangeInternalLink — Internal linking is an orthodox SEO lever with
	// no direct AI-citation evidence but no disconfirming evidence either.
	// Kept for general site-architecture recommendations.
	ChangeInternalLink ChangeType = "internal_link_add"

	// ChangeSpeed — Page speed / Core Web Vitals are Google ranking
	// prerequisites but have no direct AI-engine confirmation. Kept but
	// lower-priority than content and schema work. Research doc
	// "Demote (weak evidence)" §ChangeSpeed.
	ChangeSpeed ChangeType = "speed_fix"

	// ChangeBacklink — DR correlates with ChatGPT citations, but brand
	// mentions outperform raw backlink volume as a predictor of AI
	// citation (perplexity-and-industry.md §B.1.1). Lower-priority than
	// the entity work it is a proxy for. Research doc "Demote (weak
	// evidence)" §ChangeBacklink.
	ChangeBacklink ChangeType = "backlink_outreach"

	// ChangeIndexability — Crawler access (Googlebot, OAI-SearchBot,
	// ClaudeBot/Claude-SearchBot, PerplexityBot) is table-stakes across
	// all four engines; blocking any one removes eligibility. Research
	// doc "Keep (well-supported)" §ChangeIndexability.
	ChangeIndexability ChangeType = "indexability_fix"

	// ChangeTLDR — Add a 40-70 word direct-answer block at the top of the
	// page. ~44.2% of ChatGPT citations come from the first 30% of an
	// article (Growth Memo, 18,012 citations; perplexity-and-industry.md
	// §B.1.2). The single strongest empirically-supported on-page lever
	// across ChatGPT, Perplexity, and AI Overviews. Research doc "Add
	// (missing from current set)" §ChangeDirectAnswerIntro — renamed to
	// ChangeTLDR for clarity about the emitted artefact.
	ChangeTLDR ChangeType = "tldr_add"

	// ChangeListFormat — Convert prose answers to lists, tables, or
	// definition blocks. Signals matrix row "Lists / tables / structured
	// formatting" is marked likely for Google AI Overviews, ChatGPT
	// Search, and Perplexity; Averi and Growth Memo show list/table
	// passages extract more reliably into AI answers. Signals matrix +
	// perplexity-and-industry.md §B.1.2.
	ChangeListFormat ChangeType = "list_format"

	// ChangeAuthorByline — Add a visible author name, credentials, and a
	// linked bio / Person schema with sameAs to Wikipedia or Wikidata.
	// Grounded in Google E-E-A-T (google-ai-overviews.md §4) and
	// Perplexity trust signals / Person schema value
	// (schema-and-technical.md §5, Tier 1). Research doc "Add" §ChangeAuthor.
	ChangeAuthorByline ChangeType = "author_byline"

	// ChangeFreshness — Add or update visible publish/updated dates and
	// accurate dateModified. AI-cited URLs average ~26% fresher than
	// organic SERPs (Ahrefs 17M citations); content updated within 30
	// days gets ~3.2× more Perplexity citations (Discovered Labs); 76.4%
	// of top-cited ChatGPT pages were updated in the last 30 days.
	// perplexity-and-industry.md §B.1.3 / §A.2; research doc "Add"
	// §ChangeFreshness.
	ChangeFreshness ChangeType = "freshness_refresh"

	// ChangeEntityConsistency — Align brand NAP (name / address / phone)
	// across the page, Organization schema sameAs links, and external
	// sources (Wikipedia, Wikidata, LinkedIn, Crunchbase). Brand mentions
	// outperform backlinks as a predictor of AI citation across Ahrefs
	// (75K brands), SE Ranking (129K domains), and Growth Memo studies
	// (perplexity-and-industry.md §B.1.1). @graph / @id patterns are the
	// standard implementation vehicle (schema-and-technical.md §4).
	// Research doc "Add" §ChangeEntityConsistency.
	ChangeEntityConsistency ChangeType = "entity_consistency"
)

// Evidence captures a single data point supporting a Recommendation.
type Evidence struct {
	Source      string      `json:"source"` // "gsc" | "psi" | "serp" | "labs" | "backlinks" | "aeo" | "crawl" | "audit"
	Metric      string      `json:"metric"` // e.g. "position", "lcp_ms", "ctr"
	Value       interface{} `json:"value,omitempty"`
	Description string      `json:"description,omitempty"`
}

// Forecast is the estimated traffic impact of a Recommendation.
type Forecast struct {
	EstimatedMonthlyClicksDelta int    `json:"estimated_monthly_clicks_delta"`
	ConfidenceLow               int    `json:"confidence_low"`
	ConfidenceHigh              int    `json:"confidence_high"`
	Method                      string `json:"method,omitempty"`
}

// Recommendation is the atomic unit of "what to change on the site".
// The canonical definition lives here (rather than in
// internal/recommendations) so that State can embed it without creating an
// import cycle between the two packages. The recommendations package
// re-exports these types via type aliases.
type Recommendation struct {
	ID               string     `json:"id"` // stable hash of (TargetURL + TargetQuery + ChangeType)
	TargetURL        string     `json:"target_url"`
	TargetQuery      string     `json:"target_query,omitempty"`
	ChangeType       ChangeType `json:"change_type"`
	CurrentValue     string     `json:"current_value,omitempty"`
	RecommendedValue string     `json:"recommended_value,omitempty"`
	Rationale        string     `json:"rationale,omitempty"`
	Evidence         []Evidence `json:"evidence,omitempty"`
	Priority         int        `json:"priority"` // 1–100, reuse scoring from internal/merge
	EffortMinutes    int        `json:"effort_minutes,omitempty"`
	ForecastedLift   *Forecast  `json:"forecasted_lift,omitempty"`
	MergedFindingID  string     `json:"merged_finding_id,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}
