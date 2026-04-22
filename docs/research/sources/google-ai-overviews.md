# Google AI Overviews & Gemini — Official Guidance

> Research pass: 2026-04-22
> Sources: primary only (Google official: developers.google.com/search, support.google.com, search.google)

## TL;DR

- **No special optimization exists.** Google's own AI Features doc states plainly: "There are no additional requirements to appear in AI Overviews or AI Mode, nor other special optimizations necessary." Eligibility = indexed + snippet-eligible + meets Search technical requirements.
- **No AI-specific markup.** Google explicitly tells site owners: "You don't need to create new machine readable files, AI text files, or markup to appear in these features" — and no AI-specific schema.org types.
- **Standard snippet controls govern AI surfaces.** `nosnippet`, `data-nosnippet`, `max-snippet`, and `noindex` are the only levers for limiting how content appears in AI Overviews / AI Mode (within Search). `Google-Extended` is a separate control for AI *training*, not Search AI surfaces.
- **"Query fan-out"** is the one mechanism Google has publicly named: AI Overviews and AI Mode issue multiple related sub-queries to assemble a response, which means a wider and more diverse set of supporting links than classic Search.
- **E-E-A-T and the Helpful Content guidance are unchanged** and remain Google's stated quality framework for AI surfaces — no AI-specific E-E-A-T rules have been published.

---

## 1. Eligibility for AI Overviews

Google's primary doc on this is `developers.google.com/search/docs/appearance/ai-features` (last updated 2025-12-31 per the page footer).

- **Core eligibility rule.** "To be eligible to be shown as a supporting link in AI Overviews or AI Mode, a page must be indexed and eligible to be shown in Google Search with a snippet, fulfilling the Search technical requirements." [1]
- **No additional technical requirements.** "There are no additional technical requirements." Indexing/serving is never guaranteed even when a page meets every requirement. [1]
- **AI Overviews trigger selectively.** "AI Overviews are only shown when our systems determine that it is additive to classic Search, and as such, often don't trigger." [1]
- **Query fan-out mechanism.** Both AI Overviews and AI Mode may use a "'query fan-out' technique — issuing multiple related searches across subtopics and data sources" — which Google says lets them "display a wider and more diverse set of helpful links" than classic Search. [1]
- **Different models per surface.** "AI Mode and AI Overviews may use different models and techniques, so the set of responses and links they show will vary." [1]
- **May 2025 blog post reinforces basics.** Google's Search Central blog "Top ways to ensure your content performs well in Google's AI experiences on Search" (2025-05) tells owners to "make sure your pages meet our technical requirements for Google Search" — Googlebot not blocked, HTTP 200, indexable content — and states "Meeting the technical requirements covers you for search generally, including AI formats." [2]

## 2. Structured data

- **No AI-specific schema is required.** The AI Features doc states: "You don't need to create new machine readable files, AI text files, or markup to appear in these features." [1] Google has not published a list of schema.org types specifically used for AI Overviews.
- **Structured data is still useful for feature eligibility generally.** From the same May 2025 blog: "Structured data is useful for sharing information about your content in a machine-readable way that our systems consider and makes pages eligible for certain search features and rich results." [2]
- **Match markup to visible content.** AI Features doc recommends "making sure that your structured data matches the visible text on the page." [1]
- **Follow general structured data guidelines.** The generative-AI-content guidance tells creators: "For structured data, also ensure compliance with the general guidelines, the specific policies for the individual search features, and validate the markup to ensure eligibility for Search features." [3]
- **No mention of Article, FAQ, HowTo, or QAPage being "preferred" for AI Overviews.** Google's official docs do not single out any schema type as AI-Overview-enhancing. (Note: Google retired broad FAQ rich results in August 2023; they remain valid markup but aren't a display-eligibility shortcut.)

## 3. Content structure

Google has **not published** format-specific guidance (e.g., "use TL;DR blocks," "use Q&A format," "put the answer in the first 100 words") for AI Overviews. The AI Features doc instead says existing SEO fundamentals apply:

- "Ensuring that crawling is allowed in robots.txt, and by any CDN or hosting infrastructure" [1]
- "Making your content easily findable through internal links on your website" [1]
- "Making sure that important content is available in textual form" [1]
- "Supporting your textual content with high-quality images and videos" on pages [1]

The May 2025 blog reiterates the general "Helpful Content" framing without adding AI-specific format rules — Google tells creators to "focus on making unique, non-commodity content that visitors from Search and your own readers will find helpful and satisfying." [2]

**Assessment:** Claims circulating in the SEO community about TL;DR blocks, answer-first paragraphs, or Q&A schema being preferred for AI Overviews are **not supported by any primary Google source** we found. The closest Google comes is the general Helpful Content self-assessment: "Does the main heading or page title provide a descriptive, helpful summary of the content?" [4]

## 4. E-E-A-T, authorship, freshness

- **Same E-E-A-T framework applies.** Google's 2023 AI-content post (still linked from current docs) says: "Google's ranking systems aim to reward original, high-quality content that demonstrates qualities of what we call E-E-A-T: expertise, experience, authoritativeness, and trustworthiness." [5] This post is [dated: 2023-02] but Google continues to reference it from current pages.
- **"Who / How / Why" framing for authorship.** The Creating Helpful Content page was updated to add "thinking in terms of Who, How, and Why in relation to how content is produced." [5] Source page: `developers.google.com/search/docs/fundamentals/creating-helpful-content`. [4]
- **AI byline caution.** "Giving AI an author byline is probably not the best way to follow our recommendation to make clear to readers when AI is part of the content creation process." [5] [dated: 2023-02]
- **Expertise signals Google asks you to self-audit.** "Is this content written or reviewed by an expert or enthusiast who demonstrably knows the topic well?" and "Does the content present information in a way that makes you want to trust it, such as clear sourcing, evidence of the expertise involved, background about the author or the site that publishes it…" [4]
- **Freshness:** No AI-Overviews-specific freshness guidance exists. The Helpful Content doc warns against fake freshness: "Are you changing the date of pages to make them seem fresh when the content has not substantially changed?" [4] Google's separate "Byline dates" documentation covers how to correctly mark publication/modified dates but is not AI-Overview-specific.

## 5. Crawler directives

From `developers.google.com/search/docs/crawling-indexing/robots-meta-tag` and the AI Features doc:

- **Snippet controls that affect AI Overviews / AI Mode (within Search).** The AI Features doc: "Make use of nosnippet, data-nosnippet, max-snippet, or noindex to set your display preferences. More restrictive permissions will limit how your content is featured in our AI experiences." [1][2]
- **`nosnippet`** — do not show a text snippet or video preview. Applies per-page via meta tag or X-Robots-Tag. [6]
- **`max-snippet:[number]`** — cap the snippet length in characters. [6]
- **`data-nosnippet`** — HTML attribute on `span`, `div`, or `section` elements to exclude that specific passage from snippets. [6]
- **`noindex`** — removes the page from Search entirely, which also removes it from AI surfaces. [6]
- **Conflict rule.** "In the case of conflicting robots rules, the more restrictive rule applies. For example, if a page has both max-snippet:50 and nosnippet rules, the nosnippet rule will apply." [6]
- **Googlebot = the control for AI in Search.** "AI is built into Search and integral to how Search functions, which is why robots.txt directives for Googlebot is the control for site owners to manage access to how their sites are crawled for Search." [1]
- **`Google-Extended` is for AI *training* / grounding on other Google surfaces — NOT for AI Overviews/AI Mode in Search.** The AI Features doc directs site owners who want to "restrict the training and grounding of AI in some other Google systems" to use Google-Extended. [1] Google's Google-Extended documentation has historically stated that Google-Extended does not impact a site's inclusion or ranking in Google Search [7] [dated: 2024-02].

## 6. New signals (2025-2026)

Things Google has officially published/clarified in 2025–2026 that were not in the classic SEO playbook:

- **"Query fan-out" named publicly.** Google formally described the fan-out technique in the AI Features doc — multiple related sub-queries issued per user prompt, producing a broader set of citations than classic Search. [1]
- **AI surfaces counted inside the "Web" search type in Search Console.** "Sites appearing in AI features (such as AI Overviews and AI Mode) are included in the overall search traffic in Search Console… reported on in the Performance report, within the 'Web' search type." [1] There is no separate "AI Overviews" filter; traffic is blended.
- **Click-quality claim.** Google has repeatedly claimed (both in the AI Features doc and the May 2025 blog): "When people click from search results pages with AI Overviews, these clicks are higher quality (meaning, users are more likely to spend more time on the site)." [1][2] No raw data has been published alongside this claim.
- **Multimodal inputs as an optimization vector.** The May 2025 blog adds: "Through the power of our AI, people can perform multimodal searches where they snap a photo or upload an image… For success with this, support your textual content with high-quality images and videos on your pages, and ensure that your Merchant Center and Business Profile information is up-to-date." [2]
- **Generative-AI content policy clarifications.** The Generative AI content guidance (last updated 2025-12-10) warns that using AI to mass-produce pages "without adding value for users may violate Google's spam policy on scaled content abuse," and points to Search Quality Raters Guidelines sections 4.6.5 (scaled content abuse) and 4.6.6 (main content with little effort/originality/added value). [3]
- **AI-generated image metadata (Merchant Center).** "AI-generated images must contain metadata using the IPTC DigitalSourceType TrainedAlgorithmicMedia metadata. AI-generated product data such as title and description attributes must be specified separately and labeled as AI-generated." [3] This is the single clearest *new* disclosure-style signal Google has formalized.
- **Preferred sources (feature).** Google launched a "Preferred sources" documentation page under Ranking & search appearance, but it governs user personalization of news source preferences — not a publisher-side optimization for AI Overviews.

## Gaps / open questions

Google has **not** published primary-source guidance on:

- Any ranked list of schema.org types that specifically boost AI Overview eligibility.
- Whether `QAPage`, `HowTo`, `FAQPage`, `Article`, or `SpeakableSpecification` markup influence AI Overview citation odds.
- Any specific content-structure pattern (TL;DR, answer-first, Q&A blocks, tables, lists) being preferred by AI Overviews.
- Word-count, reading-level, or heading-depth thresholds — the Helpful Content doc explicitly says: "Are you writing to a particular word count because you've heard or read that Google has a preferred word count? (No, we don't.)" [4]
- Author-entity / `sameAs` / Knowledge Graph signals specifically mapped to AI Overview inclusion.
- Freshness windows or recrawl cadence for AI Overviews specifically.
- How Gemini's grounding in AI Mode scores source trust differently from classic Search ranking.
- A confirmed list of languages/locales where AI Overviews are live (the consumer help page says "over 120 countries and territories, and 11 languages" but doesn't enumerate). [8]
- Whether `llms.txt` or any similar AI-specific manifest is supported (Google has implicitly said *no* by telling site owners not to create new machine-readable files [1]).

## Sources

Primary Google sources only. Access date for all: **2026-04-22**.

1. **AI features and your website** — `https://developers.google.com/search/docs/appearance/ai-features` (page last updated 2025-12-31 per footer)
2. **Top ways to ensure your content performs well in Google's AI experiences on Search** — `https://developers.google.com/search/blog/2025/05/succeeding-in-ai-search` (published 2025-05)
3. **Google Search's Guidance on Generative AI Content on Your Website** — `https://developers.google.com/search/docs/fundamentals/using-gen-ai-content` (last updated 2025-12-10 per footer)
4. **Creating helpful, reliable, people-first content** — `https://developers.google.com/search/docs/fundamentals/creating-helpful-content`
5. **Google Search's guidance about AI-generated content** — `https://developers.google.com/search/blog/2023/02/google-search-and-ai-content` [dated: 2023-02]
6. **Robots meta tag, data-nosnippet, and X-Robots-Tag specifications** — `https://developers.google.com/search/docs/crawling-indexing/robots-meta-tag`
7. **Overview of Google crawlers and fetchers (user agents) — Google-Extended entry** — `https://developers.google.com/search/docs/crawling-indexing/overview-google-crawlers` (and `google-common-crawlers` reference page listed in the Google-Extended user-agent string)
8. **Find information in faster & easier ways with AI Overviews in Google Search** (consumer help) — `https://support.google.com/websearch/answer/14901683`
9. **Google AI Overviews product page** — `https://search.google/ways-to-search/ai-overviews/`

### Not consulted / out of scope

- blog.google product launch posts: not directly quoted because the developers.google.com documentation supersedes them for site-owner guidance.
- deepmind.google Gemini technical posts: no AI-Overview-specific site-owner guidance is published there.
- Google Search Central YouTube: no single video was quoted; the written Search Central docs and blog carry the authoritative wording and are cited instead.
