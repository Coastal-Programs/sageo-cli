# Perplexity + Industry Research — AI Citation Signals

_Compiled: 2026-04. Scope: Perplexity's official guidance + independent empirical research on AI citation patterns, 2024–2026._

---

## TL;DR

Top signals with repeated, multi-study empirical support (weighted toward large-N correlational studies):

1. **Brand mentions / off-site entity presence correlate more strongly with AI citation than backlinks.** Ahrefs (75K brands), SE Ranking (129K domains), and Kevin Indig's large-scale analyses all converge here; brand search volume shows a ~0.334 correlation with LLM citations ([thedigitalbloom.com, 2025-12](https://thedigitalbloom.com/learn/2025-ai-citation-llm-visibility-report/)).
2. **Top-of-page, direct-answer "BLUF" formatting wins.** ~44% of ChatGPT citations come from the first 30% of an article ([Growth Memo, 2026-02](https://www.growth-memo.com/p/the-science-of-how-ai-pays-attention), n=18,012 citations).
3. **Freshness matters — especially for ChatGPT and Perplexity.** 76.4% of ChatGPT's most-cited pages were updated in last 30 days (Digitaloft, via Onely); AI-cited URLs 25.7% fresher than organic SERPs (Ahrefs, 17M citations).
4. **Entity density and specific numbers beat generic prose.** Cited content averages 20.6% entity density vs ~5–8% in normal text; quantitative claims get ~40% higher citation rates ([Growth Memo](https://www.growth-memo.com/p/the-science-of-how-ai-pays-attention)).
5. **Google ranking still helps, but less than the SEO consensus assumes.** Only ~12% of AI assistant citations appear in Google's top 10 for the target query (Ahrefs, 15K long-tail queries); Perplexity is the outlier — ~1 in 3 of its citations rank in Google's top 10.

Weak/noisy signals: FAQ schema direct effect, TL;DR blocks, word count, llms.txt. See §B.3.

---

## Part A: Perplexity (official)

### 1. Citation selection

Perplexity has never published a formal "ranking factors" document. What we have is (a) an interview with Perplexity's head of search, (b) third-party teardowns, and (c) the company's public Publisher-Program language about source selection.

- Perplexity's head of search Alexandr Yarats told Unite.AI that Perplexity is explicitly **not** optimising ranked lists for click probability; it optimises for "helpfulness and factuality in answers" (summarised in [Otterly.ai, 2026-02](https://otterly.ai/blog/perplexity-seo/), _expert interview, not an empirical study_).
- Perplexity uses a retrieval-augmented pipeline: decompose query → fetch candidates → rerank → synthesise with inline citations. <cite index="19-42,19-43,19-44,19-45,19-46">Perplexity has built a proprietary web index using a crawler called PerplexityBot; this crawler respects your robots.txt — if you block it, Perplexity cannot cite you. The pre-built index is what Perplexity searches at query time; if PerplexityBot hasn't visited a page before a user asks a question, that page cannot appear in the answer.</cite> ([erlin.ai, 2026-03](https://www.erlin.ai/blog/perplexity-seo), _vendor explainer synthesising primary docs_).
- Perplexity's own publisher announcement states: <cite index="91-4,91-5,91-6,91-7">"Every day, people turn to Perplexity with a wide array of questions. Our ability to provide high-quality answers hinges on trusted, accurate sources covering the topics people care about most. From day one, we've included citations in each answer"</cite> ([perplexity.ai/hub, 2024-07](https://www.perplexity.ai/hub/blog/introducing-the-perplexity-publishers-program), _official source_).
- Perplexity also admitted in that post: <cite index="91-1,91-2">"We are also modifying our processes and products based on feedback from our publishing partners. Recently, we updated how our systems index and cite sources"</cite> — i.e. citation selection is a moving target, changed at publisher request. (Same source.)
- iPullRank's teardown (based on 59 factors inferred from Perplexity outputs, _correlational analysis rather than controlled study_) reports Perplexity blends lexical + semantic relevance, topical authority, entity prominence and "answer extractability", and favours pages that restate the query in a subheading with a concise answer beneath it ([iPullRank AI Search Manual, 2025-08](https://ipullrank.com/ai-search-manual/search-architecture)).

### 2. Content patterns Perplexity cites most

Empirical pattern data on Perplexity specifically (distinct from ChatGPT/AI Overviews):

- **Top domains (June 2025 ~953K Perplexity prompts, Ahrefs Brand Radar):** Wikipedia 12.5%, YouTube 16.1% (clear favourite), then Reddit/Quora at lower rates than in AI Overviews ([Ahrefs, 2025-08](https://ahrefs.com/blog/top-10-most-cited-domains-ai-assistants/), _empirical, large N_).
- **SE Ranking (2K keywords, 2025-04):** YouTube is the single most-cited domain on Perplexity at 11.1% — almost identical share to ChatGPT (11.3%) ([via Veza Digital, 2026-03](https://www.vezadigital.com/services/ai-content-strategy), _vendor blog citing SE Ranking study_).
- **Perplexity is the outlier on SERP overlap.** <cite index="76-4">"Perplexity is the outlier: nearly 1 in 3 of its citations point to pages that rank in the top 10 for the target query"</cite> ([Ahrefs, 2025-09](https://ahrefs.com/blog/ai-search-overlap/), _empirical, 15K long-tail queries_). ChatGPT/Copilot/Gemini overlap averages ~11%.
- **Perplexity has a strong recency bias.** Discovered Labs and others cite "content updated within last 30 days gets 3.2x more citations than older material" on Perplexity ([Discovered Labs, 2025-12](https://discoveredlabs.com/blog/ai-citation-patterns-how-chatgpt-claude-and-perplexity-choose-sources), _vendor blog aggregating third-party numbers; methodology not disclosed_).
- **Passage-level extraction matters more than page rank.** Multiple third-party analyses report Perplexity fetches ~10 pages per query but cites only 3–4 ([Onely, 2026-02](https://www.onely.com/blog/how-to-rank-on-perplexity/), _vendor blog, numbers attributed to "Brandlight AI analysis"_). Treat as directional rather than proven.

### 3. Crawler (PerplexityBot)

From Perplexity's official docs ([docs.perplexity.ai/guides/bots](https://docs.perplexity.ai/guides/bots), _official documentation_):

- **Two distinct user agents:**
  - `PerplexityBot` — the indexer. Full UA: `Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko; compatible; PerplexityBot/1.0; +https://perplexity.ai/perplexitybot)`. IP list: https://www.perplexity.com/perplexitybot.json. Perplexity says: <cite index="12-31,12-32">"PerplexityBot is a crawler designed to surface and link websites in Perplexity search results. Perplexity recommends allowing it in robots.txt and, if you use a WAF, permitting requests from its published IP ranges"</cite>. Explicitly **not** used to train foundation models (per docs).
  - `Perplexity-User` — fetches on behalf of a live user query. <cite index="12-33,12-34">"Perplexity-User supports user-requested fetches when someone asks Perplexity a question that requires visiting a page. Perplexity states this fetcher generally ignores robots.txt because the request is user-initiated"</cite>. IP list: https://www.perplexity.com/perplexity-user.json.
- **Directives:** Standard robots.txt `Disallow` applies to PerplexityBot but not (reliably) to Perplexity-User. Changes may take up to 24h to propagate.
- **WAF guidance:** Combine User-Agent string matching with the published IP allowlists. Cloudflare and AWS WAF instructions are in the official docs above.
- **JS rendering caveat (third-party, not official):** [erlin.ai, 2026-03](https://www.erlin.ai/blog/perplexity-seo) reports <cite index="19-35,19-36,19-37,19-38">"Before anything else, PerplexityBot must be able to access your site. Check your robots.txt file. Confirm PerplexityBot is not blocked. Confirm your content is not hidden behind JavaScript — Perplexity's crawler does not render JavaScript reliably, which means JS-only pages are effectively invisible."</cite> _Expert opinion; not verified in a controlled test I can find._
- **Cloudflare dispute:** Cloudflare has publicly accused Perplexity of bypassing `robots.txt` protections via rotating user agents; Perplexity denies this, arguing user-initiated fetches (Perplexity-User) aren't "crawling" ([thekeyword.co, 2025-09](https://www.thekeyword.co/news/perplexity-introduces-42-5m-revenue-sharing-program-for-publishers)). Worth tracking — it affects whether WAF blocks will reliably stop Perplexity traffic.

### 4. Publisher program

Timeline and structure, from Perplexity's own announcements + Bloomberg/Digiday coverage:

- **Launched 2024-07-30** with six partners: TIME, Der Spiegel, Fortune, Entrepreneur, The Texas Tribune, WordPress.com ([perplexity.ai/hub](https://www.perplexity.ai/hub/blog/introducing-the-perplexity-publishers-program), _official_).
- **Original model (2024):** Revenue share on ads sold against related-question follow-ups; publishers got free API access and a year of Enterprise Pro for all staff ([Digiday, 2024-07-30](https://digiday.com/media/perplexitys-new-rev-share-publisher-program-is-live-but-not-all-pubs-are-sold/), _reported journalism_). <cite index="8-3,8-4">The chief business officer Dmitry Shevelenko said he hopes Perplexity's new Publisher Program will get the AI company back in the media's good graces. At its core, the Publisher Program is a revenue-share deal between the AI company and publishers, based on revenue made from advertising on Perplexity's platform.</cite>
- **Expansion 2024-12:** Added 15 partners including Los Angeles Times, The Independent, ADWEEK, Prisa Media, Lee Enterprises. <cite index="3-7,3-8">"Since publicly launching this program in July, we have been pleasantly overwhelmed by the interest from all across the news media in learning more about this program. We've heard from over 100 publishers to learn more about how the program works"</cite>. Perplexity hired Jessica Chan (ex-LinkedIn) as Head of Publisher Partnerships. ([perplexity.ai/hub, 2024-12-05](https://www.perplexity.ai/hub/blog/perplexity-expands-publisher-program-with-15-new-media-partners))
- **2025-08-25 — Comet Plus:** $5/mo subscription tier launched alongside the Comet browser. <cite index="2-3,2-4,2-5">"Revenue from Perplexity's subscriptions (Pro, Max and the new $5 tier Comet Plus, first reported by Bloomberg) gets pooled. Perplexity keeps 20% of it, and the other 80% will go to participants in Perplexity's publisher program. Revenue is divvied up based on three categories: direct visits to publisher's sites by people browsing using Comet, when publisher content is cited as an answer search queries on Comet and when content is used to complete tasks by Comet's AI assistant."</cite> ([Digiday, 2025-08-26](https://digiday.com/media/how-perplexity-new-revenue-model-works-according-to-its-head-of-publisher-partnerships/))
- **Pool size:** <cite index="4-1,4-2,4-3">"Perplexity has allocated $42.5 million to share revenue with publishers. The revenue comes from Comet Plus, a $5 monthly subscription service. Publishers will receive 80% of subscription revenue, while Perplexity keeps the rest."</cite> ([thekeyword.co, 2025-09-03](https://www.thekeyword.co/news/perplexity-introduces-42-5m-revenue-sharing-program-for-publishers), _news synthesis_.)
- **Legal context:** Program rolled out alongside active lawsuits. <cite index="4-7,4-8,4-9">"The launch comes as Perplexity faces plagiarism claims and lawsuits. Publishers, including Forbes and Condé Nast, have accused the company of using their reporting in AI-generated summaries without proper attribution. News Corp.'s Dow Jones and the New York Post are currently suing Perplexity for copyright infringement after the startup failed to dismiss their case."</cite> (same source)
- **How to apply:** Publishers email `publishers@perplexity.ai`. No public SLA on citation prominence for partners — i.e. being in the program does not formally guarantee citations.

---

## Part B: Industry research

### 1. Signals with multi-study consensus

Ordered by strength of empirical support (largest-N, most independent confirmations first).

#### 1.1 Brand mentions / off-site entity presence > backlinks

- **Ahrefs (75K brands, 2025):** <cite index="22-22,22-23,22-24">"According to data in Ahrefs Brand Radar, YouTube is the most cited domain in AI Overviews today, and has grown 34% over the last six months. Some of our other studies reinforce the importance of YouTube for AI visibility. For example, our research of 75K brands revealed that mentions on YouTube—in video titles, transcripts, and descriptions—are the strongest correlating factor with AI Overview visibility."</cite> ([Ahrefs, 2026-01](https://ahrefs.com/blog/ai-overview-citations-top-10/), _large-N empirical_.)
- **Ahrefs' separate LLM citations analysis (top 1,000 ChatGPT-cited sites):** <cite index="26-38,26-39">"From a traditional SEO point of view, websites that get cited by AI tend to have stronger link profiles. When I analyzed the top 1,000 sites most frequently mentioned by ChatGPT, the data showed a clear pattern: AI favors websites with a Domain Rating (DR) above 60, and the majority of citations came from high-authority domains in the DR 80–100 range."</cite> ([Ahrefs, 2025-11](https://ahrefs.com/blog/llm-citations/)). Note: Ahrefs' own later framing softens this — DR matters, but only among a broader bundle of "authority" signals.
- **Brand search volume correlation:** <cite index="21-13,21-14">"Brand search volume—not backlinks—is the strongest predictor of AI citations (0.334 correlation). This means brand-building activities that seemed disconnected from SEO now directly impact AI visibility."</cite> ([thedigitalbloom.com, 2025-12](https://thedigitalbloom.com/learn/2025-ai-citation-llm-visibility-report/), _synthesis report citing multiple sources inc. Ahrefs/SE Ranking; original study methodology not shown_.)
- **SE Ranking (129K domains, 2025-11):** Referring domains, domain traffic, and content structure identified as top three ChatGPT-citation factors ([via milwaukee-webdesigner.com](https://milwaukee-webdesigner.com/resources/ai-citation-optimization-content-that-gets-cited-and-what-ai-engines-actually-want-from-your-website/), _vendor write-up; haven't found the SE Ranking original, so weight moderately_).
- **SE Ranking, brand-platform effect:** "Domains with millions of brand mentions on Quora and Reddit have roughly 4x higher chances of being cited than those with minimal activity" and "Domains with profiles on platforms like Trustpilot, G2, Capterra, Sitejabber, and Yelp have 3x higher chances" ([position.digital stats roundup, 2026-04](https://www.position.digital/blog/ai-seo-statistics/), _secondary citation of SE Ranking 2025-11_).

**Consensus:** Distributed, consistent off-site mentions outperform raw backlink volume as a predictor of AI citation. Independent confirmation from Ahrefs, SE Ranking, and Growth Memo.

#### 1.2 Direct-answer / "BLUF" intro formatting

- **Growth Memo "ski ramp" study (18,012 citations from 1.2M ChatGPT responses, 2026-02):** <cite index="35-1,35-2">"We analyzed 18,012 citations and found a 'ski ramp' distribution. 44.2% of all citations come from the first 30% of text (the intro)."</cite> The middle accounts for 31.1%, the conclusion 24.7%. Reported P-value effectively zero. ([Growth Memo, 2026-02](https://www.growth-memo.com/p/the-science-of-how-ai-pays-attention), _largest-N independent study on this question_.)
- **Growth Memo Part 3 (vertical-segmented study):** <cite index="40-6,40-7,40-8,40-9,40-10">"The one universal rule: open with a direct declarative statement. Not a question, not context-setting, not preamble. The form is '[X] is [Y]' or '[X] does [Z].' This is the only writing instruction that holds regardless of vertical, content type, or length. LLMs 'penalize' hedging in your intro. 'This may help teams understand' performs worse than 'Teams that do X see Y.' Remove qualifiers from your opening paragraph before any other optimization."</cite> ([Growth Memo, 2026-03](https://www.growth-memo.com/p/the-science-of-what-ai-actually-rewards))
- **Semrush content-optimization study (thousands of citations, 2026-01):** <cite index="94-12,94-13">"Based on our research, we found five content qualities that showed a strong positive correlation with AI citations, plus one that showed a negative correlation: ... In other words: Content that leads with clear answers, demonstrates expertise, and uses structured formatting gets cited more often."</cite> ([Semrush, 2026-01](https://www.semrush.com/blog/content-optimization-ai-search-study/), _empirical, methodology disclosed_.)
- **Princeton GEO paper (via Onely):** Pages leading with direct answers in the first 40–60 words cited significantly more often ([Onely, 2026-02](https://www.onely.com/blog/how-to-rank-on-perplexity/), _secondary citation of academic paper_.)

**Consensus:** Strong. Multiple independent studies with different methodologies converge.

#### 1.3 Freshness

- **Ahrefs (17M citations, 7 AI platforms, 2025):** Cited URLs averaged 25.7% fresher than traditional search results ([Ahrefs summary via position.digital](https://www.position.digital/blog/ai-seo-statistics/), _empirical, large N_).
- **Digitaloft (via Onely):** <cite index="25-22,25-23">"76.4% of ChatGPT's most-cited pages were updated in the last 30 days, according to Digitaloft research. URLs cited in AI results are 25.7% fresher on average than those in traditional search results."</cite>
- **Academic finding Ahrefs surfaced:** <cite index="26-35,26-36,26-37">"Across seven models, GPT-3.5-turbo, GPT-4o, GPT-4, LLaMA-3 8B/70B, and Qwen-2.5 7B/72B, 'fresh' passages are consistently promoted, shifting the Top-10's mean publication year forward by up to 4.78 years and moving individual items by as many as 95 ranks in our listwise reranking experiments. (…) We also observe that the preference of LLMs between two passages with an identical relevance level can be reversed by up to 25% on average after date injection in our pairwise preference experiments."</cite> ([Ahrefs, 2025-11](https://ahrefs.com/blog/llm-citations/)) — _this is a peer-style empirical study and the strongest evidence in this section._
- **Caveat:** Growth Memo Part 3 found freshness effect varies by vertical. Finance rewards shorter, denser pages; other verticals behave differently. Freshness is not a universal floor.

**Consensus:** Strong across ChatGPT, Perplexity, AI Overviews. Weaker for Claude (less web-retrieval usage).

#### 1.4 Entity density and quantitative specificity

- **Growth Memo entity-density finding:** <cite index="35-17,35-18">"Normal English text has an 'entity density' (that is, contains proper nouns like brands, tools, people) of ~5-8%. Heavily cited text has an entity density of 20.6%!"</cite> (same 1,000-page analysis).
- **Quantitative claims:** "Quantitative claims get 40% higher citation rates than qualitative statements. Pages focused on statistics receive 40% higher citation rates than regular blog posts" ([Onely, 2025-12](https://www.onely.com/blog/llm-friendly-content/), aggregating multiple studies). _Methodology varies; treat the 40% figure as a rough industry claim rather than a replicated constant._
- **GEO paper (Princeton/KDD 2024, cited by Mike King at SEO Week):** Citing sources, authoritative tone, and adding statistics were the three most impactful GEO interventions on Perplexity. <cite index="55-6,55-7,55-8,55-9">"What they did here is they tried a whole bunch of different things in Perplexity to see what was gonna actually make a difference for ranking. And so they tried all the things. They did, like, you know, the keyword stuffing, all your typical SEO things. But what they found is that, you know, citing sources, being more authoritative in how you speak, and also having statistics are the things that you that get you in there most."</cite> ([iPullRank / Mike King SEO Week 2025](https://ipullrank.com/seo-week-2025-mike-king), _academic paper summarised in a keynote_.)

**Consensus:** Moderate–strong. Multiple independent signals, though the 40% number is quoted more often than it is cleanly sourced.

#### 1.5 Topical authority / fan-out coverage > single-page optimisation

- **Ahrefs Spearman correlation:** <cite index="79-18,79-19,79-20">"Josh also found that the Spearman correlation between ranking for fan-out queries and being cited in an AI Overview is 0.77—i.e. Very strong. It's pretty clear: Ranking in AI Overviews is about building deep topical authority."</cite> ([Ahrefs, 2026-01](https://ahrefs.com/blog/how-to-rank-in-ai-overviews/), _empirical_.)
- **Ahrefs word-count finding (same study):** <cite index="79-16,79-17">"AI Overviews don't care how long your blog is—they care how well your content answers the query. Our research shows near-zero correlation (Spearman ~0.04) between word count and AI citations."</cite>
- **iPullRank (Mike King):** Query fan-out decomposes user queries into 3–30 synthetic sub-queries; pages that appear across many fan-out SERPs get cited even when they don't rank for the headline keyword ([iPullRank, 2025-08](https://ipullrank.com/probability-ai-search), _patent analysis + qualitative synthesis_.)

**Consensus:** Strong for Google-surface engines (AI Overviews, AI Mode). Less directly proven for ChatGPT/Perplexity but highly plausible given shared RAG architecture.

#### 1.6 Structured data: correlates with citation, but causation disputed

- **Semrush technical-SEO study (5M URLs, 2026-01):** <cite index="93-33,93-34">"Pages cited by AI show a clear pattern: they're far more likely to implement a few specific schema markup types. While this doesn't prove schema causes citations, the correlation is strong enough to keep an eye on."</cite> Organization, Article, BreadcrumbList, FAQ, and Site Links Search Box were the most common on cited pages ([Semrush, 2026-01](https://www.semrush.com/blog/technical-seo-impact-on-ai-search-study/)).
- **Averi synthesis:** "Approximately 65% of pages cited by AI Mode and 71% of pages cited by ChatGPT include structured data—it's clearly correlated with citation, even if the causal mechanism is debated" ([averi.ai, 2026-02](https://www.averi.ai/how-to/llm%E2%80%91optimized-content-structures-tables-faqs-snippets), _vendor synthesis_).

**Consensus:** Cited pages disproportionately have schema — but it is strongly confounded with domain authority, editorial quality, and having modern CMS. See §B.3 for the negative evidence.

---

### 2. Conflicting findings

#### 2.1 Is SERP ranking a prerequisite for AI citation?

- **Botify / Semrush-adjacent studies:** ~75–97% of AI Overview citations come from Google's top 10–20 results ([ALM Corp synthesis, 2025-12](https://almcorp.com/blog/ahrefs-ai-overviews-vs-ai-mode-analysis/), _aggregated citations_).
- **Ahrefs July 2025:** 76% of AIO cited URLs came from top 10.
- **Ahrefs January 2026 update:** That figure dropped to **~38%** after Gemini 3 shipped ([Ahrefs, 2026-01](https://ahrefs.com/blog/ai-overview-citations-top-10/), _same methodology, updated data_).
- **Cross-engine (Ahrefs, 15K queries):** <cite index="28-1">Only 12% of URLs cited by ChatGPT, Perplexity, and Copilot appear in Google's top 10</cite>.
- **Semrush 2025-07:** 90% of pages ChatGPT cites rank position 21 or lower (per milwaukee-webdesigner.com summary above).

**Resolution:** AI Overviews (Google's own feature) is tightly coupled to SERP rank, but that coupling has loosened with Gemini 3 and fan-out. Standalone LLMs (ChatGPT, Perplexity, Copilot) are loosely coupled at best. Treat "rank in Google top 10" as helpful for AI Overviews, largely irrelevant for other engines.

#### 2.2 FAQ schema: does it help?

- **Positive:** <cite index="90-26">"Pages using FAQPage schema see 28% higher citation rates than those without"</cite> ([averi.ai](https://www.averi.ai/how-to/faq-optimization-for-ai-search-getting-your-answers-cited), _claim; original study not shown_).
- **Positive-ish:** amicited.com claims 15–20% higher AI citations for question-type queries on pages with FAQ schema, from an unnamed "test" — _n and methodology undisclosed_.
- **Negative:** <cite index="81-30,81-31">"SE Ranking's analysis found that pages with FAQ schema averaged 3.6 citations in ChatGPT responses, while pages without FAQ schema averaged 4.2 citations a slight negative correlation. The effect is modest and may reflect content-type differences rather than schema itself, but it complicates the claim that FAQ schema universally improves AI citation."</cite>
- **Mechanistic argument against direct causation:** <cite index="81-3,81-4,81-5,81-6">"FAQ schema does not directly influence ChatGPT or Perplexity citation decisions. LLMs tokenize JSON-LD as raw text rather than parsing it as structured data. But FAQ schema indirectly improves AI citation probability through Google's Knowledge Graph pipeline and visible on-page Q&A content (which mirrors the schema) is directly extractable by every major AI platform. The most effective approach combines both layers: JSON-LD for Google's infrastructure, visible Q&A formatting for LLM extraction."</cite> ([ZipTie.dev, 2026-03](https://ziptie.dev/blog/faq-schema-for-ai-answers/), _vendor synthesis_).
- **Search Atlas counter-study:** Averi reports "A Search Atlas study analyzing LLM citation patterns found that schema markup alone does not influence how often LLMs cite web domains—domains with complete schema coverage performed no better than those with minimal or no schema across OpenAI, Gemini, and Perplexity" ([averi.ai](https://www.averi.ai/how-to/llm%E2%80%91optimized-content-structures-tables-faqs-snippets)).

**Resolution:** Likely conclusion: visible Q&A formatting drives most of the effect; JSON-LD FAQ schema is a secondary/indirect lift for Google-ecosystem engines, near-zero lift for ChatGPT/Perplexity direct parsing. Beware the 28%/40% numbers circulated in vendor blogs — they almost never cite the underlying study.

#### 2.3 Do engines agree on what they cite?

- **Ahrefs (540K query pairs, Dec 2025):** <cite index="80-4,80-5,80-6">"The authors looked at 730,000 query pairs for content similarity and 540,000 query pairs for citation and URL analysis. Ahrefs reports that AI Mode and AI Overviews cited the same URLs only 13% of the time. When comparing only the top three citations in each response, overlap increased to 16%."</cite> ([Search Engine Journal, 2025-12](https://www.searchenginejournal.com/google-ai-mode-ai-overviews-cite-different-urls-per-ahrefs-report/563364/))
- **thedigitalbloom.com (2025-12):** <cite index="21-20">"The 2025 analysis found that only 11% of domains are cited by both ChatGPT and Perplexity, indicating significant differences in how these platforms retrieve and select their source material."</cite>
- **Kevin Indig warning:** <cite index="32-18,32-19,32-20">"Your strategy must be LLM-specific. A Gemini-first strategy is different from a ChatGPT-first strategy. Any AI visibility report that aggregates across LLMs is misleading."</cite> ([Growth Memo, 2026-04](https://www.growth-memo.com/p/the-ghost-citation-problem), _opinion backed by his own data_).

**Resolution:** Engines agree on _what to say_ ~86% of the time (Ahrefs) but pick very different sources. Don't trust any signal claimed to work "across all AI search engines" without per-engine data.

#### 2.4 Word count / long-form content

- **Onely/vendor synthesis:** "Publish long-form content (2,000+ words) – Gets cited 3x more than short posts" ([Onely](https://www.onely.com/blog/llm-friendly-content/)).
- **Ahrefs (1.9M AI Overview citations):** near-zero correlation (Spearman ~0.04) between word count and AI citation (see §1.5). <cite index="73-15,73-16">"Content length alone has essentially no correlation with AI Overview citation frequency. Ahrefs' analysis found that 53% of all AI Overview citations go to pages under 1,000 words."</cite>
- **Growth Memo Part 3:** Vertical-dependent. CRM/SaaS rewards long pages (1.59×); Finance penalises them (0.86× word count).

**Resolution:** "Write 2,000 words" is not empirically supported as a universal rule. Density and coverage matter; length per se doesn't.

---

### 3. Weak or unsupported claims

Popular SEO advice that isn't well-supported empirically as of 2026-04:

- **"Add a TL;DR block and you'll be cited more."** I could not find a single large-N independent study isolating TL;DR blocks from the broader "answer-first formatting" effect. The Growth Memo ski-ramp finding supports intros in general, not TL;DR headers specifically. Treat TL;DR as a useful UX pattern that may or may not carry an incremental signal.
- **"Author bylines boost AI citations."** Frequently asserted (Aleyda Solis, Onely citing "Wellows research") as E-E-A-T-linked, but the underlying Wellows number — "100% of ranking AI-assisted content demonstrated clear E-E-A-T signals" — is selection bias dressed as causation. _Aleyda Solis treats it as best practice, not a proven uplift._ ([LearningAiSearch, 2026-03](https://learningaisearch.com/), _expert framework, not an empirical study_.)
- **"Outbound citations from your content increase your own citation rate."** Plausible and suggested by the Princeton GEO paper (via Mike King) and Aleyda's guidance, but not independently replicated at scale. Weight as weak.
- **"llms.txt boosts visibility."** <cite index="23-6">"LLMs.txt doesn't matter but domain authority does"</cite> ([position.digital citing Ahrefs, 2026](https://www.position.digital/blog/ai-seo-statistics/)). As of early 2026, llms.txt is aspirational — no major engine has confirmed using it for ranking.
- **"FAQPage schema alone drives 28/40/58% more citations."** See §2.2. The high numbers are in vendor blogs without accessible methodology.
- **Specific numbers like "videos with chapters show 187% higher selection rates and 6.4× higher citation rates"** ([Onely on Perplexity](https://www.onely.com/blog/how-to-rank-on-perplexity/)) — cited without an attributable primary study. Treat as marketing copy until proven.
- **"AI search visitors convert 23× / 5× / 4.4× better than organic"** — three different numbers from three different vendors (Ahrefs, Onely, Semrush), all real data from their own properties, but not generalisable. <cite index="78-22,78-23">"Here's the paradox confusing marketers: AI search drives less than 1% of referral traffic, yet visitors convert at dramatically higher rates. Ahrefs discovered visitors from AI search platforms generated 12.1% of signups despite accounting for only 0.5% of overall traffic"</cite> — that's Ahrefs' own site, not a cross-industry benchmark ([Passionfruit synthesis, 2025-11](https://www.getpassionfruit.com/blog/why-ai-citations-lean-on-the-top-10)).
- **"LLM outputs are stable enough to optimise against deterministically."** Kevin Indig's and iPullRank's research argues strongly against this. <cite index="33-17,33-18,33-19">"The growing body of research shows the extreme fragility of LLMs. They're highly sensitive to how information is presented. Minor stylistic changes that don't alter the product's actual utility can move a product from the bottom of the list to the #1 recommendation."</cite> ([Growth Memo, 2026-01](https://www.growth-memo.com/p/how-much-can-we-influence-ai-responses)). Plus Ahrefs: <cite index="97-4,97-5">"AI Overview content changes 70% of the time for the same query. And when it generates a new answer, 45.5% of citations get replaced with new ones."</cite>

---

## Sources

Methodology labels: **[Empirical]** = original dataset with N disclosed; **[Correlational]** = observational analysis without controlled intervention; **[Official]** = vendor/platform first-party; **[Expert]** = framework/opinion from a known practitioner; **[Synthesis]** = secondary aggregator.

### Perplexity — official / primary

1. Perplexity — "Perplexity Crawlers" official docs. https://docs.perplexity.ai/guides/bots (accessed 2026-04). **[Official]**
2. Perplexity — "Introducing the Perplexity Publishers' Program" (2024-07-30). https://www.perplexity.ai/hub/blog/introducing-the-perplexity-publishers-program **[Official]** `[dated: 2024-07]`
3. Perplexity — "Perplexity Expands Publisher Program with 15 New Media Partners" (2024-12-05). https://www.perplexity.ai/hub/blog/perplexity-expands-publisher-program-with-15-new-media-partners **[Official]** `[dated: 2024-12]`
4. Unite.AI — Alexandr Yarats (Head of Search, Perplexity) interview. https://www.unite.ai/alexandr-yarats-head-of-search-at-perplexity-interview-series/ **[Expert interview]**
5. Digiday — "How Perplexity's new revenue model works" (2025-08-26). https://digiday.com/media/how-perplexity-new-revenue-model-works-according-to-its-head-of-publisher-partnerships/ **[Journalism]**
6. Digiday — "Perplexity's new rev-share publisher program is live" (2024-07-30). https://digiday.com/media/perplexitys-new-rev-share-publisher-program-is-live-but-not-all-pubs-are-sold/ **[Journalism]** `[dated: 2024-07]`
7. Bloomberg — "Perplexity to Let Publishers Share in Revenue" (2025-08-25). https://www.bloomberg.com/news/articles/2025-08-25/perplexity-to-let-publishers-share-in-revenue-from-ai-searches **[Journalism]**
8. thekeyword.co — "$42.5M revenue-sharing program" summary (2025-09-03). https://www.thekeyword.co/news/perplexity-introduces-42-5m-revenue-sharing-program-for-publishers **[Journalism synthesis]**

### Industry research — empirical (large N)

9. Ahrefs — "76% of AI Overview Citations Pull From the Top 10" (2025-07). 1.9M citations from 1M AI Overviews. https://ahrefs.com/blog/search-rankings-ai-citations/ **[Empirical, large-N]**
10. Ahrefs — "Update: 38% of AI Overview Citations Pull From The Top 10" (2026-01). 4M AIO URLs / 863K keyword SERPs. https://ahrefs.com/blog/ai-overview-citations-top-10/ **[Empirical, large-N]**
11. Ahrefs — "Only 12% of AI Cited URLs Rank in Google's Top 10" (2025-09). 15K long-tail queries, four assistants. https://ahrefs.com/blog/ai-search-overlap/ **[Empirical]**
12. Ahrefs — "The 10 Most Mentioned Domains" (2025-08). 76.7M AIOs, 957K ChatGPT, 953K Perplexity prompts. https://ahrefs.com/blog/top-10-most-cited-domains-ai-assistants/ **[Empirical, large-N]**
13. Ahrefs — "How to Earn LLM Citations" (2025-11). Analyses + academic freshness-injection paper. https://ahrefs.com/blog/llm-citations/ **[Synthesis + peer research]**
14. Ahrefs — "How to Rank in AI Overviews" (2026-01). Spearman correlations, fan-out analysis. https://ahrefs.com/blog/how-to-rank-in-ai-overviews/ **[Empirical]**
15. Ahrefs / Search Engine Journal — "AI Mode & AI Overviews Cite Different URLs" (2025-12). 540K query pairs. https://www.searchenginejournal.com/google-ai-mode-ai-overviews-cite-different-urls-per-ahrefs-report/563364/ **[Empirical]**
16. Semrush — "The Most-Cited Domains in AI: A 3-Month Study" (2025-11). https://www.semrush.com/blog/most-cited-domains-ai/ **[Empirical]**
17. Semrush — "How Do Technical SEO Factors Impact AI Search?" (2026-01). 5M cited URLs. https://www.semrush.com/blog/technical-seo-impact-on-ai-search-study/ **[Empirical, large-N]**
18. Semrush — "AI Mode Comparison Study" (2025-07). https://www.semrush.com/blog/ai-mode-comparison-study/ **[Empirical]**
19. Semrush — "How We Built a Content Optimization Tool for AI Search" (2026-01). 13 content parameters tested. https://www.semrush.com/blog/content-optimization-ai-search-study/ **[Empirical]**
20. Semrush — "AI Overviews Study" (updated 2025-12). 10M+ keywords. https://www.semrush.com/blog/semrush-ai-overviews-study/ **[Empirical, large-N]**
21. Kevin Indig / Growth Memo — "The science of how AI pays attention" (2026-02). 18,012 citations from 1.2M ChatGPT responses. https://www.growth-memo.com/p/the-science-of-how-ai-pays-attention **[Empirical]**
22. Kevin Indig / Growth Memo — "The science of what AI actually rewards" (2026-03). Cross-vertical content-signals study. https://www.growth-memo.com/p/the-science-of-what-ai-actually-rewards **[Empirical]**
23. Kevin Indig / Growth Memo — "The ghost citation problem" (2026-04). 3,981 domains × 115 prompts × 14 countries × 4 engines. https://www.growth-memo.com/p/the-ghost-citation-problem **[Empirical]**
24. Kevin Indig / Growth Memo — "How much can we influence AI responses?" (2026-01). Reviews E-GEO, Kumar et al. 2024 manipulation paper. https://www.growth-memo.com/p/how-much-can-we-influence-ai-responses **[Synthesis of peer research]**
25. Kevin Indig / Growth Memo — "AI on Innovation part 2" (2024-09). 546K AI Overviews. https://www.growth-memo.com/p/ai-on-innovation-part-2 **[Empirical]** `[dated: 2024-09]`
26. SparkToro (Rand Fishkin) + Datos — "State of Search" quarterly reports (2025-2026). Clickstream panel of millions of desktop devices. https://sparktoro.com/blog/ **[Empirical, panel-based; usage-focused, not citation-focused]**

### Industry research — primary + expert frameworks

27. iPullRank (Mike King) — "The AI Search Manual" (2025-08). https://ipullrank.com/ai-search-manual and https://ipullrank.com/ai-search-manual/search-architecture **[Expert framework; analyses 59 Perplexity factors]**
28. iPullRank (Mike King) — "How AI Mode Works" (2025-08). https://ipullrank.com/how-ai-mode-works **[Expert / patent analysis]**
29. iPullRank (Mike King) — "AI Search: How Generative Engine Optimization Reshapes SEO" (2025-10). https://ipullrank.com/probability-ai-search **[Expert / patent analysis]**
30. Mike King — SEO Week 2025 keynote transcript. https://ipullrank.com/seo-week-2025-mike-king **[Expert talk summarising GEO paper by Princeton]**
31. Aleyda Solis — "AI Search Content Optimization Checklist" (2025-06). https://learningaisearch.com/ and slides https://speakerdeck.com/aleyda/the-ai-search-optimization-roadmap-by-aleyda-solis **[Expert framework]** `[dated: 2025-06]`
32. Aleyda Solis — PPC Land coverage of the checklist. https://ppc.land/seo-expert-releases-ai-search-content-optimization-checklist/ **[Journalism summarising expert framework]** `[dated: 2025-06]`

### Industry research — vendor studies / synthesis (weight lower unless N disclosed)

33. SE Ranking — 129K-domain ChatGPT citation study (2025-11). Primary link not confirmed; referenced via milwaukee-webdesigner.com. https://milwaukee-webdesigner.com/resources/ai-citation-optimization-content-that-gets-cited-and-what-ai-engines-actually-want-from-your-website/ **[Synthesis citing empirical source]**
34. SE Ranking — 2K-keyword YouTube/Reddit domain study (2025-04). Referenced via Veza Digital. https://www.vezadigital.com/services/ai-content-strategy **[Synthesis]**
35. Onely — "LLM-Friendly Content: 12 Tips" (2025-12). https://www.onely.com/blog/llm-friendly-content/ **[Synthesis]**
36. Onely — "How to Rank on Perplexity" (2026-02). https://www.onely.com/blog/how-to-rank-on-perplexity/ **[Synthesis / vendor; cites Brandlight AI]**
37. ZipTie.dev — "FAQ Schema for AI Answers" (2026-03). Dual-Layer Citation Model argument. https://ziptie.dev/blog/faq-schema-for-ai-answers/ **[Expert synthesis]**
38. Averi — "LLM-Optimized Content Structures" (2026-02). https://www.averi.ai/how-to/llm%E2%80%91optimized-content-structures-tables-faqs-snippets **[Synthesis; notes Search Atlas negative schema study]**
39. Passionfruit — "Why AI Citations Come from Top 10 Rankings" (2025-11). https://www.getpassionfruit.com/blog/why-ai-citations-lean-on-the-top-10 **[Synthesis]**
40. position.digital — "150+ AI SEO Statistics for 2026" (updated 2026-04). Statistics aggregator. https://www.position.digital/blog/ai-seo-statistics/ **[Synthesis — useful as a Rosetta stone of numbers, but verify originals]**
41. thedigitalbloom.com — "2025 AI Visibility Report" (2025-12). https://thedigitalbloom.com/learn/2025-ai-citation-llm-visibility-report/ **[Synthesis; 680M+ citations claimed but methodology not fully disclosed]**
42. Profound (via tryprofound.com) — "2025 A-list of GEO experts" (2025). https://www.tryprofound.com/resources/articles/top-experts-in-generative-engine-optimization **[Opinion / ranking]**
43. authoritytech.io — "How Perplexity Selects Sources" (2026-02). XGBoost L3 gate claim, etc. https://authoritytech.io/blog/how-perplexity-selects-sources-algorithm-2026 **[Synthesis; several specific algorithmic claims not verifiable against Perplexity docs — treat with skepticism]**

### Perplexity — third-party teardowns (context)

44. Otterly.AI — "Perplexity SEO 2026" (2026-02). https://otterly.ai/blog/perplexity-seo/ **[Vendor synthesis citing Yarats interview]**
45. erlin.ai — "Perplexity SEO: A Complete Guide to Getting Cited in 2026" (2026-03). https://www.erlin.ai/blog/perplexity-seo **[Vendor explainer; mix of claims]**
46. Wellows — "How to Rank in Perplexity AI" (2026-03). https://wellows.com/blog/how-to-rank-in-perplexity/ **[Vendor synthesis]**
47. GeoXylia — "Perplexity SEO Guide" (2026-03). https://www.geoxylia.com/blog/perplexity-seo-guide **[Vendor]**

---

_Last updated: 2026-04. If citing this document, please re-verify any numeric claim against its primary source — the AI-search research field refreshes monthly and some studies listed here will be superseded within a quarter._
