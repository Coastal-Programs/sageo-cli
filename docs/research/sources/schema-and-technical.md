# Schema.org & Technical Signals for AI Citation

*Compiled April 2026. Evidence strengths reflect what primary sources confirm vs. what empirical studies suggest vs. what is speculation.*

## TL;DR

- **Only two AI engines have publicly confirmed they use schema markup**: Google (AI Overviews) and Microsoft (Bing Copilot). OpenAI, Anthropic, and Perplexity have made no public statement about schema as a ranking or extraction signal. <cite index="20-28,20-29,20-30,20-31">Two major platforms have confirmed that schema markup helps their AIs understand content. For these platforms, it is confirmed infrastructure, not speculation. Google AI Overviews: In April 2025, the Google Search team said that structured data gives an advantage in search results. Microsoft Bing Copilot: Fabrice Canel, principal product manager at Microsoft Bing, confirmed in March 2025 that schema markup helps Microsoft's LLMs understand content for Copilot.</cite>
- **Google's official position (May 2025) is narrower than the SEO industry claims**: schema makes pages eligible for rich results and is "considered" by their systems — it does not guarantee inclusion in AI Overviews. <cite index="9-2,9-3">Structured data is useful for sharing information about your content in a machine-readable way that our systems consider and makes pages eligible for certain search features and rich results. If you're using structured data, be sure to follow our guidelines, such as making sure that all the content in your markup is also visible on your web page and that you validate the structured data markup.</cite>
- **Google deprecated FAQ and HowTo rich results in August 2023** (FAQ restricted to government/health sites; HowTo removed from desktop and mobile). The schema types still work as semantic hints but produce no visual rich result for most sites. <cite index="5-15,5-16">In August 2023, Google announced a major change to FAQ structured data visibility. FAQ rich results are now only available for well-known, authoritative government and health websites, effectively removing FAQ rich snippets from search results for the vast majority of businesses</cite> <cite index="7-13">Google removed How-To rich results on desktop and mobile.</cite>
- **Empirical evidence on schema → AI citation is mixed**. Some 2025 tests show schema correlates with AI Overview inclusion; a December 2024 Search/Atlas study found no correlation. <cite index="20-36,20-37,20-38,20-39">A December 2024 study from Search/Atlas found no correlation between schema markup coverage and citation rates. Sites with comprehensive schema didn't consistently outperform sites with minimal or no schema markup. This doesn't mean schema is useless, it means schema alone doesn't drive citations. LLM systems appear to prioritize relevance, topical authority, and semantic clarity over whether content has structured markup.</cite>
- **The highest-evidence types for AI visibility** (ranked by quality of supporting evidence): `Organization`, `Article`/`NewsArticle`, `Product`, `BreadcrumbList`, `Person`, `LocalBusiness`. `FAQPage` has mixed-but-popular support. `HowTo`, `Speakable`, `Event`, `Recipe`, `VideoObject`, `ImageObject`, `Review`/`AggregateRating`, `Course` are rich-result features without direct AI-engine confirmation.
- **Non-schema technical signals** (semantic HTML, heading hierarchy, canonical tags, sitemaps, robots.txt) are **foundational** for crawlability and likely have more direct impact on AI citation than schema, per 2025 Google guidance.

---

## 1. Google's official supported types

Google's authoritative list of rich-result-eligible structured-data features, from the Structured Data Search Gallery. <cite index="6-1">Google Search Central: 'Intro to how structured data markup works' – https://developers.google.com/search/docs/appearance/structured-data/intro-structured-data​</cite>

| Schema type | Google support level | Rich result produced | Docs URL |
|---|---|---|---|
| Article (incl. NewsArticle, BlogPosting) | Fully supported | Article card with title + larger image | /search/docs/appearance/structured-data/article |
| Book actions | Fully supported | Book preview action | /search/docs/appearance/structured-data/book |
| BreadcrumbList | Fully supported | Breadcrumb trail in result | /search/docs/appearance/structured-data/breadcrumb |
| Carousel | Fully supported (requires pairing) | Horizontal gallery — must combine with Recipe/Course/Restaurant/Movie | /search/docs/appearance/structured-data/carousel |
| Course (Course list / CourseInfo) | Fully supported | Course carousel | /search/docs/appearance/structured-data/course-info |
| Dataset | Fully supported | Google Dataset Search | /search/docs/appearance/structured-data/dataset |
| DiscussionForumPosting | Fully supported (added Nov 2023) | Forum rich result | /search/docs/appearance/structured-data/discussion-forum |
| Education Q&A | Fully supported | Flashcard result | /search/docs/appearance/structured-data/qa-flashcard |
| EmployerAggregateRating | Fully supported | Employer rating in Jobs | /search/docs/appearance/structured-data/employer-rating |
| Event | Fully supported | Event list / carousel | /search/docs/appearance/structured-data/event |
| FactCheck (ClaimReview) | Fully supported | Fact-check label | /search/docs/appearance/structured-data/factcheck |
| FAQPage | **Heavily restricted since Aug 2023** — rich result only for authoritative gov/health sites | FAQ drop-down (restricted) | /search/docs/appearance/structured-data/faqpage |
| HowTo | **Removed Aug 2023** — no rich result on desktop or mobile | (none) | /search/docs/appearance/structured-data/how-to |
| ImageObject (image metadata) | Fully supported | Image licence / creator metadata | /search/docs/appearance/structured-data/image-license-metadata |
| JobPosting | Fully supported | Google Jobs experience | /search/docs/appearance/structured-data/job-posting |
| LocalBusiness | Fully supported | Knowledge panel, map pack | /search/docs/appearance/structured-data/local-business |
| Math solver / Practice problems | Fully supported | Step-by-step math carousel | /search/docs/appearance/structured-data/math-solvers |
| Movie | Fully supported | Movie carousel | /search/docs/appearance/structured-data/movie |
| Organization | Fully supported (expanded Nov 2023 for Logo + Org details) | Knowledge panel | /search/docs/appearance/structured-data/organization |
| Product (+ Offer, Variants, ReturnPolicy, ShippingDetails, LoyaltyProgram) | Fully supported — expanded multiple times 2024-25 | Product snippets, merchant listings | /search/docs/appearance/structured-data/product |
| ProfilePage | Fully supported (Nov 2023) | Profile result | /search/docs/appearance/structured-data/profile-page |
| QAPage | Fully supported | Q&A rich result | /search/docs/appearance/structured-data/qapage |
| Recipe | Fully supported | Recipe carousel, cards | /search/docs/appearance/structured-data/recipe |
| Review snippet (Review, AggregateRating) | Fully supported | Star ratings in result | /search/docs/appearance/structured-data/review-snippet |
| SoftwareApp | Fully supported | App rating card | /search/docs/appearance/structured-data/software-app |
| Speakable (SpeakableSpecification) | Beta / limited (news publishers) | Voice-assistant read-aloud | /search/docs/appearance/structured-data/speakable |
| Subscription / Paywalled content | Fully supported | Flexible Sampling eligibility | /search/docs/appearance/structured-data/paywalled-content |
| VacationRental | Fully supported (added Dec 2023) | Vacation rental result | /search/docs/appearance/structured-data/vacation-rental |
| VehicleListing | Fully supported (added Oct 2023) | Vehicle card | /search/docs/appearance/structured-data/vehicle-listing |
| VideoObject | Fully supported | Video rich result, key moments | /search/docs/appearance/structured-data/video |
| Certification | Added Apr 2025 (replaces EnergyConsumptionDetails) | Merchant certification badge | /search/docs/appearance/structured-data/certification |

Sources for this list: Google Search Central "Structured data markup that Google Search supports" page (April 2026). SpecialAnnouncement was deprecated on 31 July 2025. <cite index="46-3,46-4">Google has announced that the Special announcement search feature will be deprecated on July 31, 2025. This rich result was brought in originally to help organizations with urgent announcements surrounding COVID-19.</cite>

---

## 2. Schema types by evidence strength

Values: **confirmed** = official public statement from the engine; **likely** = strong empirical study support; **unclear** = mixed evidence; **no evidence** = no public statement and no empirical study found.

| Schema type | Google AI Overviews | ChatGPT Search | Claude (w/ web) | Perplexity | Source |
|---|---|---|---|---|---|
| Organization | **confirmed** (entity grounding) | unclear (plausible via Bing index) | no evidence | likely (entity grounding) | Google May 2025 blog; BrightEdge analysis |
| Article / NewsArticle | **confirmed** (rich results + AI) | unclear | no evidence | likely | Google docs; SEL Sep 2025 controlled test |
| Product (+ Offer, AggregateRating) | **confirmed** (Merchant/Shopping) | likely (structured retrieval) | no evidence | likely (shopping mode) | Google docs; Perplexity shopping features |
| BreadcrumbList | **confirmed** (rich result) | no evidence | no evidence | likely (site-structure signal) | Google docs |
| Person | likely (author/E-E-A-T signal) | no evidence | no evidence | likely (authorship trust) | Google quality guidelines; Perplexity trust signals |
| LocalBusiness | **confirmed** (knowledge panel, maps) | unclear | no evidence | likely (local queries) | Google docs |
| FAQPage | unclear — restricted as rich result but still a semantic signal | likely (Q&A retrieval) | no evidence | likely (Q&A retrieval) | Google Aug 2023 deprecation; Frase/Discovered Labs 2025 |
| HowTo | unclear — rich result removed, semantic value remains | no evidence | no evidence | no evidence | Google Aug 2023 deprecation |
| Review / AggregateRating | **confirmed** (when authentic) | no evidence | no evidence | likely (commercial queries) | Google docs; Perplexity commercial signals |
| Event | **confirmed** (rich result) | no evidence | no evidence | no evidence | Google docs |
| Recipe | **confirmed** (rich result) | no evidence | no evidence | no evidence | Google docs |
| VideoObject | **confirmed** (video results, key moments) | no evidence | no evidence | likely (Perplexity shows videos) | Google docs |
| ImageObject (image metadata) | **confirmed** (creator, licence) | no evidence | no evidence | no evidence | Google docs |
| SpeakableSpecification | Limited beta (news, English-only) | no evidence | no evidence | no evidence | Google docs |
| SoftwareApplication | **confirmed** (app rich result) | no evidence | no evidence | no evidence | Google docs |

**Key evidence notes:**

- Google Search Central explicitly reiterates that schema makes pages *eligible* for rich results but makes no guarantee for AI Overview inclusion. <cite index="9-1,9-2,9-3">If you're using structured data, be sure to follow our guidelines, such as making sure that all the content in your markup is also visible on your web page and that you validate the structured data markup. Structured data is useful for sharing information about your content in a machine-readable way that our systems consider and makes pages eligible for certain search features and rich results. If you're using structured data, be sure to follow our guidelines, such as making sure that all the content in your markup is also visible on your web page and that you validate the structured data markup.</cite>
- A September 2025 Search Engine Land controlled test (three near-identical pages, varying only schema quality) found only the well-implemented-schema page appeared in an AI Overview. <cite index="31-1,31-2">A controlled test compared three nearly identical pages: one with strong schema, one with poor schema, and one with none. Only the page with well-implemented schema appeared in an AI Overview and achieved the best organic ranking.</cite> But n=3 — weak generalisability.
- ChatGPT Search: Discovered Labs (January 2026) cites research that FAQPage-marked pages are 3.2x more likely to appear in AI responses. <cite index="26-14,26-15">FAQ schema has emerged as one of the most powerful structured data types for AI search. Research shows pages with FAQPage markup are 3.2x more likely to appear in AI responses, and schema contributes up to 10% of ranking factors.</cite> Source of the "3.2x" figure is not peer-reviewed and circulates unattributed.
- Perplexity: practitioner-estimated ~10% ranking-weight for schema — not from Perplexity itself. <cite index="28-1,28-2">The primary ranking factors are content relevance (~30%), visual placement (~20%), domain authority (~15%), content freshness (~15%), source diversity (~10%), and structured data (~10%). These weights shift by query type—informational queries</cite>
- ChatGPT Search citations have low overlap with Google results — suggesting different selection signals. <cite index="38-25,38-26,38-27">ChatGPT Search primarily cites lower-ranking pages (position 21+) about 90% of the time. (Semrush, July 2025) Only 12% of URLs cited by ChatGPT, Perplexity, and Copilot rank in Google's top 10 search results. (Ahrefs, August 2025) 80% of LLM citations don't even rank in Google's top 100 for the original query.</cite>
- **No peer-reviewed study exists** on schema's impact on AI search citation. <cite index="20-9,20-10">To date, there are no peer-reviewed studies on schema's impact on AI search visibility, or controlled experiments on LLM citation behavior and schema markup. OpenAI, Anthropic, Perplexity, and other platforms besides Microsoft or Google haven't published their indexing methods.</cite>

---

## 3. Non-Google AI-engine statements on schema

### OpenAI (ChatGPT Search / SearchGPT)
- **No official public statement** that ChatGPT uses schema.org markup as a ranking or extraction signal.
- OpenAI's developer community has open questions (e.g. "Does ChatGPT's browsing tool extract JSON-LD?") with no authoritative answer. <cite index="13-1,13-2,13-3,13-4,13-5">Hi everyone, I'm trying to understand exactly how ChatGPT's browsing feature works under the hood. In particular: When using the browsing plugin, does the tool fetch both visible page content (e.g. , , etc.) and any blocks? If so, is there any official documentation or authoritative statement from OpenAI confirming that JSON-LD is scraped and merged into the text sent to the model?</cite>
- OpenAI's own "Structured Outputs" is an unrelated concept — it refers to constraining model output JSON to a developer-supplied JSON Schema, not ingesting Schema.org from the open web. <cite index="16-11,16-12">While JSON mode improves model reliability for generating valid JSON outputs, it does not guarantee that the model's response will conform to a particular schema. Today we're introducing Structured Outputs in the API, a new feature designed to ensure model-generated outputs will exactly match JSON Schemas provided by developers.</cite>
- ChatGPT Search is known to rely on the Bing index for web retrieval, so any schema benefit likely flows indirectly via Bing's use of structured data. <cite index="19-20">ChatGPT Search / SearchGPT (OpenAI): This AI search often uses Bing's index as its source.</cite>

### Anthropic (Claude)
- **No public statement** about schema.org. Claude's web-search capability (introduced in early 2025) is not documented to use or preserve JSON-LD.
- Evidence is limited to third-party inference: <cite index="19-40,19-41,19-42">Anthropic Claude: In early 2025, Claude introduced web search. That means Claude (when web-enabled) will pull real-time info from indexed sites. Again, the fundamentals apply: structured, high-quality content is more likely to be used.</cite>

### Perplexity
- **No formal documentation** on schema usage. All claims about schema weight in Perplexity's ranking (e.g. "~10%") are practitioner estimates, not Perplexity statements. <cite index="3-13,3-14">Perplexity: Independent analyses and practitioner experience suggest well-structured, authoritative content is more likely to be cited. No formal spec exists; implement clean schema and semantic HTML to maximize parsability.</cite>
- Perplexity's own guidance to site owners emphasises crawlability, freshness, and structured content — not specific schema types.

### Microsoft (Bing Copilot) — for contrast
- **Confirmed use of schema** for Copilot grounding. <cite index="20-30,20-31">Google AI Overviews: In April 2025, the Google Search team said that structured data gives an advantage in search results. Microsoft Bing Copilot: Fabrice Canel, principal product manager at Microsoft Bing, confirmed in March 2025 that schema markup helps Microsoft's LLMs understand content for Copilot.</cite>

**Bottom line for non-Google engines**: the practical advice (implement FAQPage, Article, Product, Organization schema) is based on extrapolation from Google + Bing, from practitioner experiments with small samples, and from the reasonable assumption that LLMs parse structured data more cleanly than unstructured HTML. This is defensible but not proven.

---

## 4. New / emerging schema types (2024-2026)

From Google's 2023-2026 structured-data changelog and schema.org releases (currently v29.3, September 2025 <cite index="42-3">• Schema.org • V29.3 | 2025-09-04</cite>):

| Type / feature | Introduced | Relevance to AI |
|---|---|---|
| `DiscussionForumPosting` | Nov 2023 (Google) | Recognises that AI engines (Google, ChatGPT, Perplexity) cite forum content heavily — especially Reddit. High potential relevance for community-content sites. |
| `ProfilePage` | Nov 2023 (Google) | Author-entity disambiguation; supports E-E-A-T signals. |
| `VehicleListing` | Oct 2023 | Vertical. |
| `VacationRental` | Dec 2023 | Vertical. |
| Product `Variant` | Feb 2024 | Product schema expansion for SKUs. |
| Org-level return policy (`MerchantReturnPolicy`) | Jun 2024 | Shopping signal. |
| `LoyaltyProgram` (on Product/Org) | Jun 2025 | Shopping signal. |
| `Certification` (replaces `EnergyConsumptionDetails`) | Apr 2025 | Merchant listing; broader country/product coverage. <cite index="46-1,46-2">They also announced that, starting in April 2025, they're replacing the EnergyConsumptionDetails type with the more robust Certification type, as the new type supports more countries and a broader scope of certifications. This follows the announcement made in October 2024 that the EnergyConsumptionDetails type would be replaced by the more robust Certification type.</cite> |
| `SpecialAnnouncement` | **Deprecated 31 Jul 2025** | No longer produces rich result. |
| Store widget / Shopping | Sep 2025 | New visual shopping surface. |

**Worth tracking (not yet widely adopted but plausibly relevant for AI):**
- `@graph` + stable `@id` patterns — not a new type but increasingly recommended for knowledge-graph-like linking across pages. <cite index="20-24,20-25">When schema is implemented with stable values (@id) and a structure (@graph), it starts to behave like a small internal knowledge graph. AI systems won't have to guess who you are and how your content fits together, and will be able to follow explicit connections between your brand, your authors, and your topics.</cite>
- `about` / `mentions` properties on Article — link content to entity pages.
- `SpeakableSpecification` — remains a niche beta but could grow with voice AI.

**Nothing new and AI-specific has entered schema.org core in 2024-2026.** Schema.org continues ~quarterly releases with mostly property additions and vertical expansions.

---

## 5. Other technical signals

Beyond schema, these signals matter for AI citation — and based on the May 2025 Google blog, arguably matter *more*:

### Content accessibility fundamentals (high evidence)
- **Semantic HTML** (article, section, nav, aside; proper `<h1>`–`<h3>` hierarchy). LLM retrieval systems chunk content by headings and extract passages; a flat `<div>` soup degrades passage extraction. <cite index="2-22,2-23,2-24,2-25">AI models rely on semantic HTML to understand content hierarchy and relationships. Use proper heading structure (H1, H2, H3) logically, not for styling. Mark up lists with proper UL/OL tags, use semantic elements like article, section, aside, and nav, and implement proper table markup for data with thead, tbody, and scope attributes. Avoid div soup—use meaningful HTML5 elements that convey content structure.</cite>
- **Heading-as-question pattern**: questions in H2/H3 that mirror user queries improve passage-level citation, especially in Perplexity's RAG architecture. <cite index="25-18,25-19,25-20">Perplexity's RAG system extracts content at the passage level, evaluating whether each section provides a complete, quotable answer. Self-contained paragraphs function as independent units that can be extracted without surrounding context. Direct answer in first 40-60 words of each section · Self-contained paragraphs (2-4 sentences) making complete points with supporting evidence · Lists of 5-7 items with parallel structure and clear boundaries · Tables for comparison data (2.5x citation rate increase) Proper heading hierarchy (H1→H2→H3 sequential structure)</cite>
- **Alt text on images** — Google explicitly recommends image/video quality for multimodal AI Overviews. <cite index="9-4,9-5">Through the power of our AI, people can perform multimodal searches where they snap a photo or upload an image, ask a question about it and get a rich, comprehensive response with links to dive deeper. For success with this, support your textual content with high-quality images and videos on your pages, and ensure that your Merchant Center and Business Profile information is up-to-date.</cite>

### Crawlability / indexability (high evidence)
- **robots.txt** — must not block Googlebot, GPTBot (OpenAI), ClaudeBot (Anthropic), PerplexityBot. Each engine documents its own user agent; blocking any of them removes eligibility.
- **XML sitemap** — aids discovery. The September 2025 Search Engine Land test noted AI Overview inclusion happened even without a sitemap, so it's not strictly required but remains best practice.
- **Canonical tags** — prevent duplicate-source confusion; important when multiple URLs show the same content. No public AI-engine statement on canonicals but they feed into Google's indexing, which underpins AI Overviews.
- **hreflang** — for multilingual sites; AI Overviews are live in 40+ languages and 100+ countries, and language selection uses standard signals.
- **JavaScript rendering**: Perplexity and some AI crawlers do not execute JS reliably; critical content should be in server-rendered HTML. <cite index="28-15,28-16,28-17,28-18">Technical accessibility is a foundational element of any perplexity AI optimization strategy. Ensure Perplexity can crawl and understand your content. ... Ensure critical content renders without JavaScript dependency so Perplexity's crawler can parse and index every page element.</cite>

### Authority / E-E-A-T adjacent (medium-high evidence)
- **Author bylines, credentials, and linked Person entity pages** — feed both Google's E-E-A-T and LLM trust signals.
- **`sameAs` links to Wikipedia/Wikidata** on Organization and Person schema — disambiguates the entity for knowledge-graph matching. <cite index="49-7,49-8">"sameAs": [ "https://en.wikipedia.org/wiki/Author_Name", "https://www.wikidata.org/wiki/Q12345", "https://twitter.com/authorhandle" ], // sameAs links this entity to authoritative external representations. // Google uses sameAs to confirm you are talking about a known // real-world entity, not just a name string.</cite>
- **Freshness** (datePublished, dateModified accurate and recent). Perplexity applies an explicit recency boost. <cite index="28-13">To earn Perplexity citations, focus on structure, authority, schema, cross-platform presence, and freshness—Perplexity applies a recency boost that favors updated content.</cite>

### Page-experience / performance (medium evidence)
- Core Web Vitals, HTTPS, mobile-friendliness — Google ranking prerequisites; no direct AI-engine confirmation but necessary for indexation.

---

## Recommendations for sageo

Prioritise recommending these schema types, ranked by evidence quality for AI citation:

### Tier 1 — Implement everywhere (confirmed value)
1. **Organization** (+ logo, sameAs to Wikipedia/Wikidata/LinkedIn, contactPoint). Core entity anchor. Confirmed by Google; entity grounding benefits every AI engine.
2. **Article / NewsArticle / BlogPosting** with author (linked Person), datePublished, dateModified, publisher. The single most broadly applicable type.
3. **BreadcrumbList** on every non-home page. Cheap to implement; confirmed Google rich result.
4. **Person** schema for authors (expertise, credentials, sameAs). Supports E-E-A-T and entity disambiguation.

### Tier 2 — Conditional on content type (strong empirical support)
5. **Product** (+ Offer, AggregateRating, Review) — e-commerce/SaaS pricing pages. Perplexity shopping and Google merchant both weight this heavily.
6. **LocalBusiness** — any site with a physical location. Confirmed for Google knowledge panel and maps.
7. **FAQPage** — for pages that literally present Q&As. Flag to users that Google rich-result value is restricted, but AI extraction value remains. Warn against stuffing FAQs onto pages without real Q&A content (against Google policy).
8. **VideoObject** / **ImageObject** with creator + licence — supports multimodal AI Overviews, called out explicitly by Google May 2025.

### Tier 3 — Vertical-specific
9. **Event**, **Recipe**, **Course**, **JobPosting**, **VehicleListing**, **VacationRental**, **SoftwareApplication** — only recommend when content genuinely matches.

### Do NOT over-recommend
- **HowTo** — no rich result in Google since Aug 2023. Still valid schema.org vocabulary and useful for semantic clarity, but don't oversell.
- **SpeakableSpecification** — niche, English-news-publisher beta.
- **QAPage** — only where the page structure is actually a single question+answers thread.

### Non-schema technical checks sageo should surface
- robots.txt allows `Googlebot`, `GPTBot`, `ClaudeBot`, `Google-Extended`, `PerplexityBot`, `OAI-SearchBot`, `ChatGPT-User`
- XML sitemap present and discoverable
- Canonical tag on every page
- hreflang for multilingual sites
- Heading hierarchy is single-H1 + logical H2/H3 nesting
- All images have alt text
- Critical content renders without JavaScript (server-side rendering or progressive enhancement)
- `@id` values on schema entities for cross-page linking (knowledge-graph pattern)
- Schema markup content matches visible page content (Google policy — mismatch can disqualify the page from rich results entirely)

### Principles to bake into sageo's recommendations
- Flag when a user asks to add schema for content that doesn't actually exist on the page (e.g. FAQ schema with no visible Q&A). Google's policy requires markup to match visible content. <cite index="9-3">If you're using structured data, be sure to follow our guidelines, such as making sure that all the content in your markup is also visible on your web page and that you validate the structured data markup.</cite>
- Encourage validation via Google Rich Results Test + Schema.org Validator.
- Prefer JSON-LD over Microdata/RDFa.
- Be honest: for ChatGPT, Claude, Perplexity, schema is a *plausible* but *unconfirmed* lever. Authority, freshness, and passage-level structure likely matter more.

---

## Sources

Primary (official):
1. Google Search Central — "Structured data markup that Google Search supports" (Search Gallery). https://developers.google.com/search/docs/appearance/structured-data/search-gallery — current as of April 2026.
2. Google Search Central Blog — "Top ways to ensure your content performs well in Google's AI experiences on Search" (May 2025). https://developers.google.com/search/blog/2025/05/succeeding-in-ai-search
3. Google Search Central Blog — "Changes to HowTo and FAQ rich results" (August 2023). https://developers.google.com/search/blog/2023/08/howto-faq-changes [dated: 2023-08]
4. Schema.org — Releases page. https://schema.org/docs/releases.html (latest release v29.3, 2025-09-04)
5. Schema.org — How we work. https://schema.org/docs/howwework.html
6. OpenAI — "Introducing Structured Outputs in the API" (Aug 2024). https://openai.com/index/introducing-structured-outputs-in-the-api/ [dated: 2024-08] — relevant only to show OpenAI's "structured outputs" is a different concept.
7. OpenAI Developer Community thread — "Does ChatGPT's browsing tool extract JSON-LD schema?" (June 2025). https://community.openai.com/t/does-chatgpt-s-browsing-tool-extract-json-ld-schema-along-with-visible-html/1281878 — no official answer.

Empirical studies / analysis:
8. Search Engine Land — "How schema markup fits into AI search — without the hype" (March 2026). https://searchengineland.com/schema-markup-ai-search-no-hype-472339 — most balanced assessment of what is confirmed vs. speculation.
9. Search Engine Land — "Schema and AI Overviews: Does structured data improve visibility?" (September 2025). https://searchengineland.com/schema-ai-overviews-structured-data-visibility-462353 — small-n controlled test (three pages).
10. Ahrefs — AI Overviews CTR studies (April 2025, December 2025). Reported via PPC Land. https://ppc.land/googles-ai-summaries-now-swallow-58-of-clicks-that-once-went-to-websites/
11. Semrush — AI Overviews Study 10M+ keywords (December 2025). https://www.semrush.com/blog/semrush-ai-overviews-study/
12. Authoritas / BrightEdge / Seer Interactive / Conductor — various AIO CTR and citation-pattern studies, 2025. Synthesised in iPullRank "Everything We Know About AI Overviews" (June 2025).
13. Position.Digital — "150+ AI SEO Statistics for 2026" (April 2026). https://www.position.digital/blog/ai-seo-statistics/ — aggregates Ahrefs/Semrush/SE Ranking data.
14. Search/Atlas — December 2024 study finding no correlation between schema coverage and AI citation. Referenced via (8). [dated: 2024-12]

Practitioner / industry analyses (used only to corroborate):
15. BrightEdge — "Structured Data in the AI Search Era". https://www.brightedge.com/blog/structured-data-ai-search-era
16. Geneo — "Schema Markup & Structured Data Best Practices for GEO in AI Search (2025)". https://geneo.app/blog/schema-markup-structured-data-best-practices-geo-ai-search-2025/
17. SEOTuners — "Structured Data for AEO & GEO in 2025" (December 2025). https://seotuners.com/blog/seo/schema-for-aeo-geo-faq-how-to-entities-that-win/
18. Onely — "How to Rank on Perplexity" (February 2026). https://www.onely.com/blog/how-to-rank-on-perplexity/
19. FogTrail — Perplexity ranking factors (March 2026). https://fogtrail.ai/blog/tactics-to-rank-higher-on-perplexity
20. Discovered Labs — Perplexity Optimization (January 2026). https://discoveredlabs.com/blog/perplexity-optimization-how-to-get-cited-linked-2026
21. Frase — "Are FAQ Schemas Important for AI Search" (November 2025). https://www.frase.io/blog/faq-schema-ai-search-geo-aeo
22. Sitebulb — Structured Data change history. https://sitebulb.com/structured-data-history/
23. Stan Ventures — "John Mueller Clarifies Schema Changes Coming in 2026" (November 2025). https://www.stanventures.com/news/google-john-mueller-schema-update-2026-5719/

Gaps in the evidence base (stated plainly):
- No peer-reviewed study on schema → LLM citation.
- No public statement from OpenAI, Anthropic, or Perplexity on Schema.org usage.
- Most "Xx citation rate" claims trace back to unattributed analyst estimates and should not be treated as authoritative.
