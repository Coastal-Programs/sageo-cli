# AI Citation Signals — Cross-Engine Synthesis (2026)

> Synthesis of primary-source research across Google AI Overviews, ChatGPT Search, Claude, Perplexity, and Schema.org.
> Source files: `docs/research/sources/*.md`
> Date: 2026-04-22

## Executive summary

The four major AI engines disagree on what they cite, and they publish wildly different amounts of guidance about it. Any advice that claims to work "across all AI search engines" is almost certainly overfitting.

1. **Only Google and Microsoft have publicly confirmed that schema markup influences their AI surfaces.** OpenAI, Anthropic, and Perplexity have said nothing (per `schema-and-technical.md` §3). Schema is a plausible lever for ChatGPT/Claude/Perplexity — not a proven one.
2. **The biggest evidence-backed on-page lever is direct-answer formatting at the top of the page.** ~44% of ChatGPT citations come from the first 30% of an article (per `perplexity-and-industry.md` §B.1.2), with multiple independent studies converging.
3. **Brand mentions and off-site entity presence outperform raw backlink volume** as predictors of AI citation, confirmed by Ahrefs (75K brands), SE Ranking (129K domains), and Growth Memo (per `perplexity-and-industry.md` §B.1.1).
4. **Freshness matters for ChatGPT, Perplexity, and AI Overviews** — cited URLs average ~26% fresher than organic SERPs. Less evidence for Claude (per `perplexity-and-industry.md` §B.1.3).
5. **Crawler access is table-stakes.** Each engine has a named bot (Googlebot, OAI-SearchBot, ClaudeBot/Claude-SearchBot, PerplexityBot) that must be allowed in robots.txt; blocking them removes eligibility (per `openai-chatgpt-search.md` §3, `anthropic-claude.md` §3, `perplexity-and-industry.md` §A.3).
6. **Google top-10 ranking is helpful for AI Overviews but nearly irrelevant elsewhere** — only ~12% of URLs cited by ChatGPT/Perplexity/Copilot rank in Google's top 10 (per `perplexity-and-industry.md` §B.2.1).
7. **Most "XX% uplift" vendor claims lack accessible methodology.** Treat FAQPage's "28% more citations", HowTo boosts, `llms.txt` benefits, and Q&A schema uplifts as unverified until primary data is released.

(≈295 words)

## Signals matrix

| Signal | Google (AIO/AIM) | OpenAI (ChatGPT Search) | Anthropic (Claude) | Perplexity |
|---|---|---|---|---|
| TL;DR / direct-answer block at top of page | likely¹ | likely¹ | unclear¹ | likely¹ |
| Lists / tables / structured formatting | likely²·¹ | likely¹ | no evidence | likely¹ |
| Q&A / FAQ-style visible structure | unclear²·¹ | likely¹ | no evidence | likely¹ |
| Explicit author byline + credentials | likely³ | no evidence | no evidence | likely¹ |
| Visible publish / updated dates | likely³ | no evidence | likely⁴ | likely¹ |
| Content freshness (<12 months) | likely¹·³ | likely¹ | unclear⁴ | confirmed¹·¹⁰ |
| Outbound citations to authoritative sources | unclear¹ | no evidence | no evidence | likely¹ |
| Entity consistency (Wikipedia/Wikidata, NAP) | confirmed²·⁵ | unclear¹ | no evidence | likely¹ |
| Schema: Article / NewsArticle | confirmed²·⁵ | unclear⁵·⁶ | no evidence | likely⁵ |
| Schema: FAQPage | unclear²·⁵ | unclear¹·⁵ | no evidence | unclear¹·⁵ |
| Schema: HowTo | unclear⁵ | no evidence | no evidence | no evidence |
| Schema: Organization | confirmed²·⁵ | unclear⁶ | no evidence | likely¹·⁵ |
| Schema: Person | likely³·⁵ | no evidence | no evidence | likely⁵ |
| Schema: Product / Review | confirmed²·⁵ | likely⁵ | no evidence | likely⁵ |
| Schema: BreadcrumbList | confirmed²·⁵ | no evidence | no evidence | likely⁵ |
| Schema: LocalBusiness | confirmed²·⁵ | unclear⁵ | no evidence | likely⁵ |
| Semantic HTML hierarchy (proper H1/H2/H3) | likely²·⁵ | no evidence | likely⁴·⁵ | likely⁵·¹ |
| Page speed / Core Web Vitals | likely⁵ | no evidence | no evidence | no evidence |
| Mobile-friendliness | likely⁵ | no evidence | no evidence | no evidence |
| Inbound backlinks from authoritative domains | likely¹ | likely¹ | no evidence | likely¹ |
| Direct traffic / brand-search volume | likely¹ | likely¹ | no evidence | likely¹ |
| JavaScript-rendered content (risk) | unclear² | no evidence | confirmed⁴ (web-fetch no JS) | likely¹ (crawler no JS) |
| Structured data via @graph / @id patterns | likely⁵ | no evidence | no evidence | no evidence |
| Appearing in Google top-10 for the query | confirmed¹·⁹ (coupled) | no evidence (~12% overlap)¹ | no evidence | likely¹ (~33% overlap) |
| `llms.txt` file | no evidence² | no evidence⁸ | no evidence⁴ | no evidence |

**Legend:** `confirmed` = official primary-source statement; `likely` = multi-study empirical support; `unclear` = conflicting or weak evidence; `no evidence` = nothing published or studied.

**Footnote sources** (dedupe across source files):

1. Growth Memo / Kevin Indig — ski-ramp and vertical studies, 2026 (`perplexity-and-industry.md` §B.1.2, §B.1.3, §B.1.5).
2. Google Search Central, "AI features and your website" — `developers.google.com/search/docs/appearance/ai-features` (`google-ai-overviews.md` §1–5).
3. Google Search Central, "Creating helpful, reliable, people-first content" — `developers.google.com/search/docs/fundamentals/creating-helpful-content` (`google-ai-overviews.md` §4).
4. Anthropic, "Web search tool" and "Web fetch tool" Messages API docs — `docs.anthropic.com/en/docs/agents-and-tools/tool-use/web-search-tool` (`anthropic-claude.md` §1, §5).
5. Google Search Central, "Structured data search gallery" + Semrush/Averi/Search Engine Land empirical studies (`schema-and-technical.md` §1–3).
6. OpenAI, "Publishers and Developers — FAQ" — `help.openai.com/en/articles/12627856-publishers-and-developers-faq` (`openai-chatgpt-search.md` §2, §5). Note: OpenAI publishes no schema guidance.
7. *(reserved)*
8. Ahrefs / position.digital — llms.txt empirical nulls (`perplexity-and-industry.md` §B.3).
9. Ahrefs, "76% of AIO citations pull from the top 10" / 2026 update — `ahrefs.com/blog/ai-overview-citations-top-10/` (`perplexity-and-industry.md` §B.2.1).
10. Perplexity docs + Discovered Labs recency data (`perplexity-and-industry.md` §A.2).

## Per-engine notes

### Google (AI Overviews, Gemini)

Google is the only engine with extensive primary-source guidance — but the guidance is deliberately narrow. The AI Features doc states plainly: "There are no additional requirements to appear in AI Overviews or AI Mode, nor other special optimizations necessary" (per `google-ai-overviews.md` §1). Eligibility is indexed + snippet-eligible + meets Search technical requirements. There are no AI-specific schema types, no preferred content patterns, and no published list of ranking signals specifically for AI surfaces.

The two mechanisms Google has publicly named are (a) the **query fan-out** technique — "issuing multiple related searches across subtopics and data sources" — which produces a wider, more diverse set of citations than classic Search (per `google-ai-overviews.md` §1), and (b) the snippet controls (`nosnippet`, `data-nosnippet`, `max-snippet`, `noindex`) as the only levers publishers have to limit AI surface appearance. `Google-Extended` governs AI *training* on other surfaces, not AI Overviews (per `google-ai-overviews.md` §5).

On structured data, Google's May 2025 blog says schema "makes pages eligible for certain search features and rich results" but stops short of claiming AI-Overview impact (per `schema-and-technical.md` §2). E-E-A-T, the Helpful Content framework, and "match markup to visible content" remain the stated quality bar (per `google-ai-overviews.md` §4).

### OpenAI (ChatGPT Search)

**OpenAI publishes almost no publisher ranking guidance.** The only on-record mechanical description is that the search model is "a fine-tuned version of GPT-4o" and "leverages third-party search providers, as well as content provided directly by our partners" (per `openai-chatgpt-search.md` §1). There is no public documentation of ranking signals, E-E-A-T analogues, citation selection logic, or schema preference.

The only concrete levers OpenAI documents are:
- **OAI-SearchBot** must be allowed in `robots.txt` to appear in ChatGPT search answers; GPTBot and ChatGPT-User are separate, independent controls (per `openai-chatgpt-search.md` §3).
- Referral traffic carries `utm_source=chatgpt.com`; robots.txt updates take ~24h to propagate (per `openai-chatgpt-search.md` §3, §6).
- The only structural advice OpenAI has published is WAI-ARIA tagging — and that is for the **Atlas browser's agent**, not for search ranking (per `openai-chatgpt-search.md` §2).

Schema.org, heading hierarchy, word count, direct-answer formatting, freshness, sitemaps — **all undocumented**. Third-party claims about "E-E-A-T for ChatGPT" have no primary-source basis (per `openai-chatgpt-search.md` §Gaps).

### Anthropic (Claude web search)

Anthropic publishes even less positive publisher guidance than OpenAI. "The crawler help article is the only publisher-facing document, and it is framed around opt-out, not inclusion" (per `anthropic-claude.md` TL;DR). Three crawlers are documented: `ClaudeBot` (training), `Claude-User` (user-directed fetches), and `Claude-SearchBot` (search indexing). All respect `robots.txt` and `Crawl-delay` (per `anthropic-claude.md` §3).

What can be *inferred* from the Messages API shape (per `anthropic-claude.md` §5): web-search citations include a `cited_text` capped at ~150 characters, an accurate `page_age`, and a required `title` — which favours self-contained sentences, accurate last-modified dates, and clean `<title>` tags. The web fetch tool explicitly does not render JavaScript (per `anthropic-claude.md` §5). Anthropic has made no public statement on `llms.txt`, schema.org, heading hierarchy, or any content-structure pattern.

The only publisher-facing commercial signal is the July 2025 Wiley/MCP partnership, which committed to "author attribution and citations" standards for scientific content (per `anthropic-claude.md` §4).

### Perplexity (+ industry empirical research)

Perplexity has published no formal ranking-factors document, but several large-N empirical studies across AI engines converge on patterns that strongly apply to Perplexity:

- **Perplexity is the outlier on SERP overlap**: ~1 in 3 of its citations rank in Google's top 10, versus ~12% for ChatGPT/Copilot/Gemini (Ahrefs, 15K queries; per `perplexity-and-industry.md` §A.2).
- **Strong recency bias**: content updated within 30 days gets ~3.2× more citations (Discovered Labs; per `perplexity-and-industry.md` §A.2).
- **Passage-level extraction favours direct-answer intros**: 44.2% of ChatGPT citations come from the first 30% of text (Growth Memo, 18,012 citations; per `perplexity-and-industry.md` §B.1.2). Similar patterns in Perplexity.
- **Entity density matters**: cited content averages ~20.6% entity density vs ~5–8% in normal prose (Growth Memo; per `perplexity-and-industry.md` §B.1.4).
- **Topical authority across fan-out queries** correlates 0.77 Spearman with AI Overview citation — deeper beats longer (Ahrefs; per `perplexity-and-industry.md` §B.1.5).
- **PerplexityBot does not reliably render JavaScript**; critical content must be server-rendered (per `perplexity-and-industry.md` §A.3 and `schema-and-technical.md` §5).

Cross-engine empirical findings that span multiple engines: brand mentions beat backlinks as a predictor; freshness matters for three of four engines; structured data correlates with citation but the causal mechanism is disputed (Search/Atlas Dec 2024 found no correlation; Semrush Jan 2026 found strong correlation) (per `perplexity-and-industry.md` §B.1.1, §B.1.3, §B.1.6).

## Conflicts & open questions

- **Does FAQPage schema help?** Averi reports 28% higher citations for FAQ-marked pages; SE Ranking found a slight *negative* correlation (3.6 vs 4.2 citations); Search Atlas found no schema-coverage effect at all. Likely conclusion: visible Q&A formatting drives most of the effect; JSON-LD FAQ schema adds marginal lift for Google only (per `perplexity-and-industry.md` §B.2.2).
- **Does SERP rank predict AI citation?** Tight for AI Overviews pre-Gemini 3 (76% from top 10); loose post-Gemini 3 (~38%); nearly irrelevant for ChatGPT/Perplexity/Copilot (~12% top-10 overlap) (per `perplexity-and-industry.md` §B.2.1).
- **Does word count matter?** Onely says 2,000+ words get cited 3× more; Ahrefs found Spearman ~0.04 correlation (effectively zero); Growth Memo found it varies by vertical — finance penalises long pages. "Write 2,000 words" is not supported as a universal rule (per `perplexity-and-industry.md` §B.2.4).
- **Do engines cite the same sources?** Only 13% overlap between AI Mode and AI Overviews; 11% between ChatGPT and Perplexity. Any "works across all AI engines" claim is suspect (per `perplexity-and-industry.md` §B.2.3).
- **Does `llms.txt` help?** No evidence from any engine; Google implicitly said *no* by telling site owners not to create new machine-readable files (per `google-ai-overviews.md` §1, `perplexity-and-industry.md` §B.3).
- **Does schema parse through LLMs directly, or only via Google's pipeline?** ZipTie.dev argues "LLMs tokenize JSON-LD as raw text rather than parsing it as structured data" — so Q&A schema's benefit is via Google's Knowledge Graph, not direct LLM extraction. Not independently verified (per `perplexity-and-industry.md` §B.2.2).
- **Is LLM output stable enough to deterministically optimise against?** AI Overview content changes 70% of the time for the same query; 45.5% of citations get replaced on regeneration (Ahrefs). Treat any single-query test as noise (per `perplexity-and-industry.md` §B.3).
- **Google's `QAPage`/`HowTo`/`Speakable` impact on AI Overviews** — Google has not published any ranked list of schema types preferred by AI Overviews. This is a silence, not a confirmation (per `google-ai-overviews.md` §Gaps).

## Recommendations for sageo's rule set

For the refine-rules task (task id `4c5bb98b`). Current ChangeTypes in `internal/recommendations/types.go`: `ChangeTitle`, `ChangeMeta`, `ChangeH1`, `ChangeH2`, `ChangeSchema`, `ChangeBody`, `ChangeInternalLink`, `ChangeSpeed`, `ChangeBacklink`, `ChangeIndexability`.

### Keep (well-supported)

- **`ChangeTitle`** — Anthropic's citation schema makes `title` load-bearing (per `anthropic-claude.md` §5); Google's Helpful Content doc calls out descriptive headings (per `google-ai-overviews.md` §4). Strong cross-engine support.
- **`ChangeH1` / `ChangeH2`** — Semantic heading hierarchy improves passage-level retrieval across RAG systems (per `schema-and-technical.md` §5); Perplexity's extraction is passage-based. Keep as distinct signals.
- **`ChangeBody`** — Direct-answer intros in the first 30% of text are the strongest empirically-supported on-page lever (per `perplexity-and-industry.md` §B.1.2). Body-content rewrites are high-leverage.
- **`ChangeSchema`** — Confirmed for Google AI Overviews and Bing Copilot; correlates with citation elsewhere (per `schema-and-technical.md` §2). Keep, but scope recommendations to Tier-1 types (Organization, Article, BreadcrumbList, Person).
- **`ChangeIndexability`** — Crawler access (Googlebot, OAI-SearchBot, ClaudeBot, PerplexityBot) is table-stakes across all four engines (per `openai-chatgpt-search.md` §3, `anthropic-claude.md` §3, `perplexity-and-industry.md` §A.3).

### Demote (weak evidence)

- **`ChangeMeta`** — No primary source across the five research files claims meta description directly influences AI citation. Google treats it as a snippet hint, not a ranking signal. Demote below title/H-tags/body.
- **`ChangeSpeed`** — Page speed / Core Web Vitals are Google ranking prerequisites but have **no direct AI-engine confirmation** (per `schema-and-technical.md` §5). Keep, but lower-priority than content and schema work.
- **`ChangeBacklink`** — Ahrefs found DR correlation with ChatGPT citations, but brand mentions outperform backlinks as a predictor (per `perplexity-and-industry.md` §B.1.1). Don't demote to zero — but lower-priority than the brand/entity work it's a proxy for.

### Add (missing from current set)

- **`ChangeAuthor`** (or `ChangePersonSchema`) — Author byline + linked `Person` schema + `sameAs` to Wikipedia/Wikidata. Evidence: Google E-E-A-T (`google-ai-overviews.md` §4); Perplexity trust signals and Person schema value (`schema-and-technical.md` §5, Tier 1). Not covered by `ChangeSchema` at sufficient granularity.
- **`ChangeFreshness`** (visible publish/updated dates + accurate `dateModified`) — Evidence: Ahrefs 17M citations (25.7% fresher on AI than organic); Perplexity recency boost; ChatGPT 76.4% of top-cited pages updated in last 30 days (per `perplexity-and-industry.md` §B.1.3). Distinct from body rewrites.
- **`ChangeEntityConsistency`** (Organization `sameAs`, consistent NAP, Wikipedia/Wikidata presence) — Evidence: brand mentions are the strongest AI-citation correlate (per `perplexity-and-industry.md` §B.1.1); `@graph`/`@id` patterns recommended (per `schema-and-technical.md` §4).
- **`ChangeJSRendering`** (server-side rendering for critical content) — Evidence: PerplexityBot does not render JS reliably; Claude's web-fetch tool explicitly doesn't execute JS (per `anthropic-claude.md` §5, `perplexity-and-industry.md` §A.3). Distinct failure mode from `ChangeIndexability`.
- **`ChangeDirectAnswerIntro`** (rewrite first paragraph to lead with declarative answer) — Evidence: 44.2% of ChatGPT citations come from first 30% of page; direct declarative openers beat hedged ones across all verticals (per `perplexity-and-industry.md` §B.1.2). Arguably a subtype of `ChangeBody`; if kept under `ChangeBody`, ensure prompts explicitly surface this pattern.

### Drop (no evidence or disconfirmed)

- **Adding `llms.txt` files** — No engine has confirmed using it; Ahrefs found it does not matter; Google implicitly said not to create such files (per `google-ai-overviews.md` §1, `perplexity-and-industry.md` §B.3). Do **not** recommend as a ChangeType.
- **`ChangeHowToSchema` / `ChangeSpeakableSchema`** — Google removed HowTo rich results in Aug 2023; Speakable remains a news-publisher beta. Neither has any cross-engine AI-citation evidence (per `schema-and-technical.md` §1, §2). If `ChangeSchema` has sub-types, deprioritise these.

## Sources

Consolidated and deduplicated across all five source files. Each entry lists URL, title, date, type, and which source file(s) cite it.

### Official / primary

1. **Google Search Central — "AI features and your website"** — `https://developers.google.com/search/docs/appearance/ai-features`. Last updated 2025-12-31. *Official.* (google-ai-overviews.md, schema-and-technical.md)
2. **Google Search Central — "Top ways to ensure your content performs well in Google's AI experiences on Search"** — `https://developers.google.com/search/blog/2025/05/succeeding-in-ai-search`. 2025-05. *Official.* (google-ai-overviews.md, schema-and-technical.md)
3. **Google Search Central — "Google Search's Guidance on Generative AI Content"** — `https://developers.google.com/search/docs/fundamentals/using-gen-ai-content`. Updated 2025-12-10. *Official.* (google-ai-overviews.md)
4. **Google Search Central — "Creating helpful, reliable, people-first content"** — `https://developers.google.com/search/docs/fundamentals/creating-helpful-content`. *Official.* (google-ai-overviews.md, schema-and-technical.md)
5. **Google Search Central — "Structured data markup that Google Search supports" (Search Gallery)** — `https://developers.google.com/search/docs/appearance/structured-data/search-gallery`. Current April 2026. *Official.* (schema-and-technical.md)
6. **Google Search Central — "Changes to HowTo and FAQ rich results"** — `https://developers.google.com/search/blog/2023/08/howto-faq-changes`. 2023-08. *Official.* (schema-and-technical.md)
7. **Google Search Central — "Robots meta tag, data-nosnippet, and X-Robots-Tag specifications"** — `https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag`. *Official.* (google-ai-overviews.md)
8. **Google Search Central — "Overview of Google crawlers (Google-Extended)"** — `https://developers.google.com/search/docs/crawling-indexing/overview-google-crawlers`. *Official.* (google-ai-overviews.md)
9. **Google Search Central — "AI-generated content guidance"** — `https://developers.google.com/search/blog/2023/02/google-search-and-ai-content`. 2023-02. *Official.* (google-ai-overviews.md)
10. **OpenAI — "Overview of OpenAI Crawlers"** — `https://platform.openai.com/docs/bots`. *Official.* (openai-chatgpt-search.md)
11. **OpenAI — "Introducing ChatGPT search"** — `https://openai.com/index/introducing-chatgpt-search/`. 2024-10-31 (updated 2025-02-05). *Official.* (openai-chatgpt-search.md)
12. **OpenAI — "SearchGPT Prototype"** — `https://openai.com/index/searchgpt-prototype/`. 2024-07-25. *Official.* (openai-chatgpt-search.md)
13. **OpenAI — "Publishers and Developers — FAQ"** — `https://help.openai.com/en/articles/12627856-publishers-and-developers-faq`. Updated ~2026-04. *Official.* (openai-chatgpt-search.md)
14. **OpenAI — "Web search (API guide)"** — `https://platform.openai.com/docs/guides/tools-web-search`. *Official.* (openai-chatgpt-search.md)
15. **OpenAI — "Introducing Structured Outputs in the API"** — `https://openai.com/index/introducing-structured-outputs-in-the-api/`. 2024-08. *Official.* (schema-and-technical.md)
16. **Anthropic — "Web search tool" (Messages API docs)** — `https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/web-search-tool`. *Official.* (anthropic-claude.md)
17. **Anthropic — "Introducing web search on the Anthropic API"** — `https://www.anthropic.com/news/web-search-api`. 2025-05-07. *Official.* (anthropic-claude.md)
18. **Anthropic — "Citations" (Messages API docs)** — `https://docs.anthropic.com/en/docs/build-with-claude/citations`. *Official.* (anthropic-claude.md)
19. **Anthropic — "Introducing Citations on the Anthropic API"** — `https://www.anthropic.com/news/introducing-citations-api`. 2025-06-23. *Official.* (anthropic-claude.md)
20. **Anthropic — "Search results" (Messages API docs)** — `https://docs.anthropic.com/en/docs/build-with-claude/search-results`. *Official.* (anthropic-claude.md)
21. **Anthropic Support — "Does Anthropic crawl data from the web…"** — `https://support.anthropic.com/en/articles/8896518-does-anthropic-crawl-data-from-the-web-and-how-can-site-owners-block-the-crawler`. *Official.* (anthropic-claude.md)
22. **Anthropic — "Claude can now search the web"** — `https://www.anthropic.com/news/web-search`. 2025-03-20. *Official.* (anthropic-claude.md)
23. **Wiley / Anthropic MCP partnership announcement** — `https://newsroom.wiley.com/press-releases/press-release-details/2025/Wiley-Partners-with-Anthropic-...`. 2025-07-09. *Official (joint).* (anthropic-claude.md)
24. **Perplexity — "Perplexity Crawlers" official docs** — `https://docs.perplexity.ai/guides/bots`. *Official.* (perplexity-and-industry.md)
25. **Perplexity — "Introducing the Perplexity Publishers' Program"** — `https://www.perplexity.ai/hub/blog/introducing-the-perplexity-publishers-program`. 2024-07-30. *Official.* (perplexity-and-industry.md)
26. **Perplexity — "Perplexity Expands Publisher Program with 15 New Media Partners"** — `https://www.perplexity.ai/hub/blog/perplexity-expands-publisher-program-with-15-new-media-partners`. 2024-12-05. *Official.* (perplexity-and-industry.md)
27. **Schema.org — Releases** — `https://schema.org/docs/releases.html`. v29.3, 2025-09-04. *Official.* (schema-and-technical.md)

### Empirical studies (large-N)

28. **Ahrefs — "76% of AI Overview Citations Pull From the Top 10"** — `https://ahrefs.com/blog/search-rankings-ai-citations/`. 2025-07. 1.9M citations. *Empirical.* (perplexity-and-industry.md, schema-and-technical.md)
29. **Ahrefs — "Update: 38% of AI Overview Citations Pull From The Top 10"** — `https://ahrefs.com/blog/ai-overview-citations-top-10/`. 2026-01. 4M URLs. *Empirical.* (perplexity-and-industry.md, schema-and-technical.md)
30. **Ahrefs — "Only 12% of AI Cited URLs Rank in Google's Top 10"** — `https://ahrefs.com/blog/ai-search-overlap/`. 2025-09. 15K queries. *Empirical.* (perplexity-and-industry.md)
31. **Ahrefs — "The 10 Most Mentioned Domains"** — `https://ahrefs.com/blog/top-10-most-cited-domains-ai-assistants/`. 2025-08. *Empirical.* (perplexity-and-industry.md)
32. **Ahrefs — "How to Earn LLM Citations"** — `https://ahrefs.com/blog/llm-citations/`. 2025-11. *Empirical + peer research.* (perplexity-and-industry.md)
33. **Ahrefs — "How to Rank in AI Overviews"** — `https://ahrefs.com/blog/how-to-rank-in-ai-overviews/`. 2026-01. Spearman/fan-out. *Empirical.* (perplexity-and-industry.md)
34. **Ahrefs / Search Engine Journal — "AI Mode & AI Overviews Cite Different URLs"** — `https://www.searchenginejournal.com/google-ai-mode-ai-overviews-cite-different-urls-per-ahrefs-report/563364/`. 2025-12. 540K pairs. *Empirical.* (perplexity-and-industry.md)
35. **Semrush — "Most-Cited Domains in AI: 3-Month Study"** — `https://www.semrush.com/blog/most-cited-domains-ai/`. 2025-11. *Empirical.* (perplexity-and-industry.md)
36. **Semrush — "How Do Technical SEO Factors Impact AI Search?"** — `https://www.semrush.com/blog/technical-seo-impact-on-ai-search-study/`. 2026-01. 5M URLs. *Empirical.* (perplexity-and-industry.md, schema-and-technical.md)
37. **Semrush — "AI Mode Comparison Study"** — `https://www.semrush.com/blog/ai-mode-comparison-study/`. 2025-07. *Empirical.* (perplexity-and-industry.md)
38. **Semrush — "Content Optimization Tool for AI Search"** — `https://www.semrush.com/blog/content-optimization-ai-search-study/`. 2026-01. *Empirical.* (perplexity-and-industry.md)
39. **Semrush — "AI Overviews Study"** — `https://www.semrush.com/blog/semrush-ai-overviews-study/`. Updated 2025-12. 10M+ keywords. *Empirical.* (perplexity-and-industry.md, schema-and-technical.md)
40. **Growth Memo (Kevin Indig) — "The science of how AI pays attention"** — `https://www.growth-memo.com/p/the-science-of-how-ai-pays-attention`. 2026-02. 18,012 citations. *Empirical.* (perplexity-and-industry.md)
41. **Growth Memo — "The science of what AI actually rewards"** — `https://www.growth-memo.com/p/the-science-of-what-ai-actually-rewards`. 2026-03. *Empirical.* (perplexity-and-industry.md)
42. **Growth Memo — "The ghost citation problem"** — `https://www.growth-memo.com/p/the-ghost-citation-problem`. 2026-04. *Empirical.* (perplexity-and-industry.md)
43. **Growth Memo — "How much can we influence AI responses?"** — `https://www.growth-memo.com/p/how-much-can-we-influence-ai-responses`. 2026-01. *Synthesis of peer research.* (perplexity-and-industry.md)
44. **Search Engine Land — "Schema and AI Overviews: Does structured data improve visibility?"** — `https://searchengineland.com/schema-ai-overviews-structured-data-visibility-462353`. 2025-09. *Empirical (small-N).* (schema-and-technical.md)
45. **Search/Atlas — Schema-coverage vs AI-citation correlation study** — Referenced via Search Engine Land / Averi. 2024-12. *Empirical.* (schema-and-technical.md, perplexity-and-industry.md)
46. **Search Engine Land — "How schema markup fits into AI search — without the hype"** — `https://searchengineland.com/schema-markup-ai-search-no-hype-472339`. 2026-03. *Synthesis.* (schema-and-technical.md)

### Expert frameworks / practitioner analyses

47. **iPullRank (Mike King) — "The AI Search Manual"** — `https://ipullrank.com/ai-search-manual`. 2025-08. *Expert.* (perplexity-and-industry.md)
48. **iPullRank — "How AI Mode Works"** — `https://ipullrank.com/how-ai-mode-works`. 2025-08. *Expert / patent analysis.* (perplexity-and-industry.md)
49. **iPullRank — "Probability AI Search / GEO"** — `https://ipullrank.com/probability-ai-search`. 2025-10. *Expert.* (perplexity-and-industry.md)
50. **Mike King — SEO Week 2025 keynote** — `https://ipullrank.com/seo-week-2025-mike-king`. *Expert talk summarising Princeton GEO paper.* (perplexity-and-industry.md)
51. **Aleyda Solis — "AI Search Content Optimization Checklist"** — `https://learningaisearch.com/`. 2025-06. *Expert framework.* (perplexity-and-industry.md)
52. **Unite.AI — Alexandr Yarats (Head of Search, Perplexity) interview** — `https://www.unite.ai/alexandr-yarats-head-of-search-at-perplexity-interview-series/`. *Expert interview.* (perplexity-and-industry.md)

### Synthesis / vendor analyses

53. **thedigitalbloom.com — "2025 AI Visibility Report"** — `https://thedigitalbloom.com/learn/2025-ai-citation-llm-visibility-report/`. 2025-12. *Synthesis.* (perplexity-and-industry.md)
54. **Onely — "LLM-Friendly Content: 12 Tips"** — `https://www.onely.com/blog/llm-friendly-content/`. 2025-12. *Synthesis.* (perplexity-and-industry.md)
55. **Onely — "How to Rank on Perplexity"** — `https://www.onely.com/blog/how-to-rank-on-perplexity/`. 2026-02. *Synthesis.* (perplexity-and-industry.md, schema-and-technical.md)
56. **Discovered Labs — "AI Citation Patterns"** — `https://discoveredlabs.com/blog/ai-citation-patterns-how-chatgpt-claude-and-perplexity-choose-sources`. 2025-12. *Synthesis.* (perplexity-and-industry.md)
57. **Discovered Labs — "Perplexity Optimization"** — `https://discoveredlabs.com/blog/perplexity-optimization-how-to-get-cited-linked-2026`. 2026-01. *Synthesis.* (schema-and-technical.md)
58. **ZipTie.dev — "FAQ Schema for AI Answers"** — `https://ziptie.dev/blog/faq-schema-for-ai-answers/`. 2026-03. *Expert synthesis.* (perplexity-and-industry.md)
59. **Averi — "LLM-Optimized Content Structures"** — `https://www.averi.ai/how-to/llm-optimized-content-structures-tables-faqs-snippets`. 2026-02. *Synthesis.* (perplexity-and-industry.md, schema-and-technical.md)
60. **Passionfruit — "Why AI Citations Come from Top 10 Rankings"** — `https://www.getpassionfruit.com/blog/why-ai-citations-lean-on-the-top-10`. 2025-11. *Synthesis.* (perplexity-and-industry.md)
61. **position.digital — "150+ AI SEO Statistics for 2026"** — `https://www.position.digital/blog/ai-seo-statistics/`. Updated 2026-04. *Synthesis aggregator.* (perplexity-and-industry.md, schema-and-technical.md)
62. **Otterly.AI — "Perplexity SEO 2026"** — `https://otterly.ai/blog/perplexity-seo/`. 2026-02. *Synthesis.* (perplexity-and-industry.md)
63. **erlin.ai — "Perplexity SEO: A Complete Guide"** — `https://www.erlin.ai/blog/perplexity-seo`. 2026-03. *Vendor explainer.* (perplexity-and-industry.md)
64. **BrightEdge — "Structured Data in the AI Search Era"** — `https://www.brightedge.com/blog/structured-data-ai-search-era`. *Synthesis.* (schema-and-technical.md)
65. **Frase — "Are FAQ Schemas Important for AI Search"** — `https://www.frase.io/blog/faq-schema-ai-search-geo-aeo`. 2025-11. *Synthesis.* (schema-and-technical.md)

### Journalism / legal context

66. **Digiday — "Perplexity's rev-share publisher program"** — `https://digiday.com/media/perplexitys-new-rev-share-publisher-program-is-live-but-not-all-pubs-are-sold/`. 2024-07-30. *Journalism.* (perplexity-and-industry.md)
67. **Digiday — "How Perplexity's new revenue model works"** — `https://digiday.com/media/how-perplexity-new-revenue-model-works-according-to-its-head-of-publisher-partnerships/`. 2025-08-26. *Journalism.* (perplexity-and-industry.md)
68. **Bloomberg — "Perplexity to Let Publishers Share in Revenue"** — `https://www.bloomberg.com/news/articles/2025-08-25/perplexity-to-let-publishers-share-in-revenue-from-ai-searches`. 2025-08-25. *Journalism.* (perplexity-and-industry.md)
69. **thekeyword.co — "$42.5M revenue-sharing program"** — `https://www.thekeyword.co/news/perplexity-introduces-42-5m-revenue-sharing-program-for-publishers`. 2025-09-03. *Journalism synthesis.* (perplexity-and-industry.md)
70. **Will Scott — "How AI Licensing Deals Determine Search Visibility in 2025"** — `https://willscott.me/2025/10/04/ai-licensing-deals-search-visibility-in-2025/`. 2025-10. *Secondary (block-rate figures).* (anthropic-claude.md)
