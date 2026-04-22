# Anthropic / Claude — Web Search & Citation Behaviour

**Research scope:** Anthropic's official, primary-source guidance on how Claude selects, cites, and surfaces web content; the Messages API "citations" feature; crawler policy; and any publisher-facing guidance.
**As of:** April 2026.
**Approach:** Primary sources only (docs.anthropic.com, anthropic.com/news, support.anthropic.com). Secondary coverage (news, legal analysis) noted only where it contextualises a primary-source silence.

---

## TL;DR

- Claude's **web search tool** (server-side, API) <cite index="0-0">decides when to search based on the prompt, runs the searches server-side, and at the end of its turn returns a final response with cited sources</cite>. Citations are not optional — <cite index="0-0">"Citations are always enabled for web search"</cite>.
- Each web citation contains a `url`, `title`, `encrypted_index`, and up to **150 characters** of `cited_text`. Terms require that citations be displayed to end users when API output is shown directly.
- The **Messages API "citations" feature** (separate product) is for RAG over user-supplied documents, not web content. It chunks provided PDFs/text into sentences and returns structured citations. It does **not** govern how Claude surfaces the public web.
- Anthropic operates **three named crawlers**: `ClaudeBot` (training), `Claude-User` (user-directed fetches), `Claude-SearchBot` (search index). All respect `robots.txt` and the non-standard `Crawl-delay` directive.
- Anthropic publishes essentially **no SEO / publisher optimisation guidance**. There is no equivalent of Google Search Central. The crawler help article is the only publisher-facing document, and it is framed around opt-out, not inclusion.
- 2025–2026 public posture: web search shipped on API (May 2025) and consumer (March 2025); web fetch tool followed (Sept 2025); dynamic filtering for both landed in early 2026. Beyond a Wiley MCP partnership (July 2025), Anthropic has said little publicly about commercial publisher relationships — in contrast to OpenAI/Google which have announced many.

---

## 1. Web search + citation mechanism

### How Claude selects and cites sources (Messages API `web_search` server tool)

From the `web_search_20250305` / `web_search_20260209` tool reference: <cite index="0-0">"Claude decides when to search based on the prompt. The API executes the searches and provides Claude with the results. This process may repeat multiple times throughout a single request. At the end of its turn, Claude provides a final response with cited sources"</cite>.

Under the hood, Anthropic describes the loop as follows (May 2025 announcement): <cite index="2-0">"When Claude receives a request that would benefit from up-to-date information or specialized knowledge, it uses its reasoning capabilities to determine whether the web search tool would help provide a more accurate response. If searching the web would be beneficial, Claude generates a targeted search query, retrieves relevant results, analyzes them for key information, and provides a comprehensive answer with citations back to the source material."</cite>

Claude is agentic in this loop: <cite index="2-0">"Claude can also operate agentically and conduct multiple progressive searches, using earlier results to inform subsequent queries in order to do light research and generate a more comprehensive answer."</cite> Developers cap this via `max_uses`.

Anthropic does not document the **underlying search index** (e.g., which provider powers results), nor the ranking/relevance signals it considers. The docs describe what the model *receives* (a list of `web_search_result` blocks with `url`, `title`, `encrypted_content`, `page_age`) and what it *produces* (text blocks with attached `web_search_result_location` citations), but not how Anthropic's backend selects or orders those results.

### Dynamic filtering (2026)

The `web_search_20260209` version (released with Claude Opus 4.7 / Sonnet 4.6 / Mythos Preview) adds a filtering layer: <cite index="0-0">"Claude can write and execute code to filter search results before they reach the context window, keeping only relevant information and discarding the rest. This leads to more accurate responses while reducing token consumption."</cite> Anthropic lists use cases: <cite index="0-0">"Searching through technical documentation / Literature review and citation verification / Technical research / Response grounding and verification"</cite>. Dynamic filtering requires the `code_execution` tool.

### Citation schema (web search)

From the Messages API reference:

```
{
  "type": "web_search_result_location",
  "url": "...",
  "title": "...",
  "encrypted_index": "...",
  "cited_text": "up to 150 chars of the cited content"
}
```

<cite index="0-0">"Citations are always enabled for web search"</cite>. Per the docs, <cite index="0-0">"The web search citation fields cited_text, title, and url do not count towards input or output token usage."</cite>

**Display obligation (developer-facing, not SEO-facing):** <cite index="0-0">"When displaying API outputs directly to end users, citations must be included to the original source. If you are making modifications to API outputs, including by reprocessing and/or combining them with your own material before displaying them to end users, display citations as appropriate based on consultation with your legal team."</cite>

### Domain controls (developer-facing)

Developers can scope the search with `allowed_domains`, `blocked_domains`, and `user_location`. Admins can enforce this organisation-wide: per the May 2025 announcement, <cite index="2-0">"Domain allow lists: Specify which domains Claude can search and retrieve information from, ensuring that results only come from approved sources. Domain block lists: Prevent Claude from accessing certain domains..."</cite>

### Web fetch tool (complement to search)

Released September 10, 2025. Differs from web search in three notable ways: (a) fetches *full* page content (including PDFs) rather than snippets; (b) citations are **opt-in**, not mandatory — <cite index="1-0">"Unlike web search where citations are always enabled, citations are optional for web fetch"</cite>; (c) by design, <cite index="1-0">"Claude is not allowed to dynamically construct URLs. Claude can only fetch URLs that have been explicitly provided by the user or that come from previous web search or web fetch results."</cite>

This matters for retrieval behaviour: the natural pipeline is **search → fetch**, with fetch citing passages (char-range) from the retrieved document.

---

## 2. Messages API "citations" feature

**Key finding: this feature is about user-supplied documents (RAG), not the public web.** It is often confused with web search citations because both live in the same API.

From the Citations doc: <cite index="3-0">"Claude is capable of providing detailed citations when answering questions about documents, helping you track and verify information sources in responses."</cite> The relevant input types are `document` (PDF/text) and `search_result` blocks that the developer passes in.

### How it works

<cite index="3-0">"For PDFs: Text is extracted as described in PDF Support and content is chunked into sentences... For plain text documents: Content is chunked into sentences that can be cited from. For custom content documents: Your provided content blocks are used as-is and no further chunking is done."</cite>

Citation granularity is therefore **sentence-level** for auto-chunked inputs. Developers wanting other granularity (bullets, transcripts) must use custom content blocks.

### Announced value proposition

From the June 2025 launch post: <cite index="4-0">"Claude can now provide detailed references to the exact sentences and passages it uses to generate responses, leading to more verifiable, trustworthy outputs."</cite> Anthropic reports <cite index="4-0">"Our internal evaluations show that Claude's built-in citation capabilities outperform most custom implementations, increasing recall accuracy by up to 15%."</cite>

### Related: `search_result` content blocks

A separate, newer primitive (listed as available on Opus 4.5/4.6/4.7, Sonnet 4.5/4.6, Haiku 4.5, etc.): developers can return `search_result` blocks from custom tools or supply them top-level. The docs position this as <cite index="5-0">"bringing web search-quality citations to your custom applications"</cite>. Required fields are `source` (URL or identifier), `title`, and a `content` array of text blocks.

### Does this affect how Claude surfaces *web* content?

**No, not directly.** The citations feature operates on content the developer injects. However, the `search_result` block is an important signal: Anthropic is standardising a single citation format (source + title + cited text + location) across web search, web fetch, and developer-supplied RAG. A publisher whose content is retrieved through **any** of these channels will be surfaced with that schema.

---

## 3. Crawler directives (ClaudeBot, Claude-User, Claude-SearchBot)

Anthropic's only public crawler document is the support article "Does Anthropic crawl data from the web…" (last updated ~April 2026 per the page stamp). Three named bots:

| Bot | Purpose (Anthropic's wording) | Effect of blocking |
|---|---|---|
| **ClaudeBot** | <cite index="6-0">"ClaudeBot helps enhance the utility and safety of our generative AI models by collecting web content that could potentially contribute to their training."</cite> | <cite index="6-0">"When a site restricts ClaudeBot access, it signals that the site's future materials should be excluded from our AI model training datasets."</cite> |
| **Claude-User** | <cite index="6-0">"Claude-User supports Claude AI users. When individuals ask questions to Claude, it may access websites using a Claude-User agent."</cite> | <cite index="6-0">"Disabling Claude-User on your site prevents our system from retrieving your content in response to a user query, which may reduce your site's visibility for user-directed web search."</cite> |
| **Claude-SearchBot** | <cite index="6-0">"Claude-SearchBot navigates the web to improve search result quality for users. It analyzes online content specifically to enhance the relevance and accuracy of search responses."</cite> | <cite index="6-0">"Disabling Claude-SearchBot on your site prevents our system from indexing your content for search optimization, which may reduce your site's visibility and accuracy in user search results."</cite> |

### Operating principles

Anthropic states four principles:

1. <cite index="6-0">"Our collection of data should be transparent."</cite>
2. <cite index="6-0">"Our crawling should not be intrusive or disruptive. We aim for minimal disruption by being thoughtful about how quickly we crawl the same domains and respecting Crawl-delay where appropriate."</cite>
3. <cite index="6-0">"Anthropic's Bots respect 'do not crawl' signals by honoring industry standard directives in robots.txt."</cite>
4. <cite index="6-0">"Anthropic's Bots respect anti-circumvention technologies (e.g., we will not attempt to bypass CAPTCHAs for the sites we crawl.)"</cite>

### What publishers can set

Only two knobs are documented: `robots.txt` `User-agent` / `Disallow`, and the non-standard `Crawl-delay`. Example given by Anthropic:

```
User-agent: ClaudeBot
Crawl-delay: 1
```

and, to fully block:

```
User-agent: ClaudeBot
Disallow: /
```

Anthropic explicitly warns against blocking by IP: <cite index="6-0">"Alternate methods like blocking IP address(es) from which Anthropic Bots operates may not work correctly or persistently guarantee an opt-out, as doing so impedes our ability to read your robots.txt file."</cite> Anthropic publishes a list of source IPs for verification (linked from that same page).

### Silences

- **No equivalent of Google's sitemaps / structured data / canonical-URL guidance.**
- **No documentation** of crawl frequency, freshness, JavaScript rendering (note: web fetch tool doesn't render JS — <cite index="1-0">"The web fetch tool currently does not support websites dynamically rendered via JavaScript."</cite>), or how often Claude-SearchBot re-indexes.
- **No documented mechanism** for reporting wrong/outdated attributions (beyond a generic support email).
- The old user agent `anthropic-ai` — which circulated in 2023–2024 — **is not listed** on the current page. Publishers should treat the three bots above as the canonical current list.

---

## 4. Publisher guidance

**Primary finding: Anthropic has published no positive ("how to be visible in Claude") guidance for publishers.**

Everything that reads as "publisher guidance" is framed as opt-out, not inclusion:

- The crawler help article tells publishers how to **block** bots, not how to be indexed well.
- The web-search blog post discusses **developer** controls (allow/block lists, `max_uses`), not publisher-side signals.
- Anthropic's newsroom has **no** "Search Central"-style hub.

The one substantive publisher-facing signal in 2025 is the **Wiley / MCP partnership** (July 9, 2025). Wiley announced it adopted MCP with Anthropic, and <cite index="7-3,7-4">"Beginning with a pilot program, and subject to definitive agreement, Wiley and Anthropic will work to ensure university partners have streamlined, enhanced access to their Wiley research content. Another key focus of the partnership is to establish standards for how AI tools properly integrate scientific journal content into results while providing appropriate context for users, including author attribution and citations."</cite> This is the first public instance of Anthropic publicly committing to citation/attribution standards with a publisher.

**Contrast with competitors (secondary-source context):** third-party trackers report <cite index="8-0">"60% average blocking Anthropic crawlers"</cite> among top US news sites by May 2025 — a higher block rate than Perplexity (56%), Gemini (58%), or OpenAI's crawlers. Anthropic has not publicly responded to this or published a publisher FAQ comparable to OpenAI's.

---

## 5. Content structure preferences

**Primary finding: Anthropic publishes no explicit content-structure guidance for web publishers.**

What *can* be inferred from the technical docs:

1. **Sentence-granular citations.** The Messages API citations feature chunks PDFs/text at the **sentence** level by default. While this applies to developer-supplied documents, it suggests Anthropic's citation model is built around quotable, self-contained sentences. The doc notes: <cite index="3-0">"sentence chunking would allow Claude to cite a single sentence or chain together multiple consecutive sentences to cite a paragraph (or longer)!"</cite>

2. **Titles are load-bearing.** Both `web_search_result` and `search_result` schemas require a `title`. The rendered citation Claude produces includes it. Pages with accurate, descriptive `<title>` tags will attribute cleanly; pages without won't.

3. **`page_age` is a documented field.** Each `web_search_result` returned to the model includes <cite index="0-0">"page_age: When the site was last updated"</cite>. This implies Claude is given freshness metadata to reason with — publishers who expose accurate last-modified information (HTTP headers, visible dates) give the model more to work with. Anthropic does not document *how* this field is computed.

4. **`cited_text` is capped at ~150 chars for web results.** This favours content where key claims are expressed in short, self-contained statements — any claim Claude cites must fit (or be truncatable to) ~150 characters in the returned citation payload.

5. **No JavaScript execution in web fetch.** <cite index="1-0">"The web fetch tool currently does not support websites dynamically rendered via JavaScript."</cite> Content that renders only client-side will not be fetched usefully through this path. (Note: the separate server-side *search* pipeline — Claude-SearchBot — is not documented as having this limitation, but Anthropic publishes nothing about its rendering behaviour.)

6. **Structured RAG example formats.** The `search_result` content block spec shows Anthropic's preferred input shape: `{source, title, content: [{type: "text", text: "..."}]}` with optional multi-block segmentation so Claude can cite a **specific** block via `start_block_index`/`end_block_index`. A publisher thinking about how to be retrieval-friendly for Anthropic's own ingestion pathway should probably think in terms of **discrete, self-contained text units with a stable URL and title** — the same pattern.

**Silences to be explicit about:** Anthropic does not publish guidance on headings (H1/H2 structure), schema.org markup, FAQ/HowTo structured data, hreflang, pagination, canonical tags, sitemaps, or `llms.txt`-style files.

---

## 6. Recent statements (2025–2026)

Chronological primary-source timeline:

- **March 20, 2025** — Consumer web search launches. <cite index="9-0">"You can now use Claude to search the internet to provide more up-to-date and relevant responses... When Claude incorporates information from the web into its responses, it provides direct citations so you can easily fact check sources."</cite> Initially US-paid-only, Claude 3.7 Sonnet. Expanded globally to all plans May 27, 2025 (same post, updated).
- **May 7, 2025** — Web search tool launches on the Anthropic API. Pricing <cite index="2-0">"$10 per 1,000 searches plus standard token costs"</cite>.
- **June 23, 2025** — Citations feature GA on the Messages API (<cite index="4-0">"Citations is generally available on the Anthropic API and Google Cloud's Vertex AI"</cite>). Amazon Bedrock added June 30, 2025.
- **July 9, 2025** — Wiley/Anthropic MCP partnership announced, with explicit attribution language (see §4).
- **September 10, 2025** — Web fetch tool launches (<cite index="2-0">"You can now add the web fetch tool to your requests and Claude will fetch and analyze content from any webpage URL"</cite> — update banner on the May 2025 blog post).
- **February 9, 2026** (implied by version strings `web_search_20260209` / `web_fetch_20260209`) — Dynamic filtering lands for both web search and web fetch tools on Claude Opus 4.7 / 4.6 and Sonnet 4.6.
- **April 2026** — Crawler support article's "Updated over 2 weeks ago" stamp indicates recent revision; current text lists `ClaudeBot`, `Claude-User`, `Claude-SearchBot` as the canonical three bots.

On attribution and publisher relationships specifically, Anthropic's on-the-record statements are limited to:

- The repeated product-doc line that citations <cite index="0-0">"must be included to the original source"</cite> when API output is shown directly.
- The Wiley partnership statement (§4).
- The crawler principles (§3).

Anthropic has **not** published statements on: (a) revenue-sharing with publishers for web-search citations, (b) how `Claude-SearchBot` ranks content, (c) whether citation frequency correlates with traffic referral, (d) what "appropriate context for users" concretely means outside the Wiley context.

---

## Gaps / open questions

1. **What search index powers `web_search`?** Anthropic documents the tool interface but not the underlying backend (Brave? Bing? First-party? Mixed?). Publishers optimising for Claude are effectively optimising for an undisclosed pipeline.
2. **How are results ranked inside the tool?** Anthropic publishes no ranking signals.
3. **Does `Claude-SearchBot` render JavaScript?** The web *fetch* tool doesn't — unclear whether the search-side crawler is different.
4. **Refresh cadence for `Claude-SearchBot`?** Undocumented.
5. **Does Claude preferentially cite higher-authority domains?** No public statement; behaviour is empirically biased towards Wikipedia, major news, docs sites, but Anthropic does not publish ranking signals.
6. **Any `llms.txt` or AI-specific structured-data support?** No — Anthropic has not endorsed `llms.txt` or any successor in primary sources.
7. **Remediation path for bad citations?** The crawler-page email address exists for bot malfunctions; there is no documented channel for "Claude misattributed my content."
8. **Will the "old" `anthropic-ai` user agent still fetch anything?** It is absent from the current support page, but not explicitly deprecated in primary sources.
9. **Any publisher revenue / licensing programme?** Silent in primary sources beyond the one Wiley announcement. Secondary sources describe Anthropic as a licensing "holdout" (music and book publisher amicus briefs, late 2025 / early 2026) — but that is litigation framing, not Anthropic's own statement.

---

## Sources

Primary sources only. Accessed April 2026.

1. **Anthropic — "Web search tool" (Messages API docs).** https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/web-search-tool — tool versions `web_search_20250305` and `web_search_20260209`; citation schema; dynamic filtering; display obligation. Accessed 2026-04-22.
2. **Anthropic — "Introducing web search on the Anthropic API"** (product announcement). https://www.anthropic.com/news/web-search-api — dated May 7, 2025; update banner for web fetch dated Sept 10, 2025. Accessed 2026-04-22.
3. **Anthropic — "Citations" (Messages API docs).** https://docs.anthropic.com/en/docs/build-with-claude/citations — document chunking, sentence-granularity, citation schema for RAG. Accessed 2026-04-22.
4. **Anthropic — "Introducing Citations on the Anthropic API"** (product announcement). https://www.anthropic.com/news/introducing-citations-api — dated June 23, 2025. Accessed 2026-04-22.
5. **Anthropic — "Search results" (Messages API docs).** https://docs.anthropic.com/en/docs/build-with-claude/search-results — `search_result` content-block schema, model availability. Accessed 2026-04-22.
6. **Anthropic Support — "Does Anthropic crawl data from the web, and how can site owners block the crawler?"** https://support.anthropic.com/en/articles/8896518-does-anthropic-crawl-data-from-the-web-and-how-can-site-owners-block-the-crawler — canonical crawler list (ClaudeBot, Claude-User, Claude-SearchBot), `robots.txt` / `Crawl-delay` guidance. Page stamp: "Updated over 2 weeks ago" as of 2026-04-22. Accessed 2026-04-22.
7. **Wiley newsroom — "Wiley Partners with Anthropic to Accelerate Responsible AI Integration Across Scholarly Research"** (joint announcement). https://newsroom.wiley.com/press-releases/press-release-details/2025/Wiley-Partners-with-Anthropic-to-Accelerate-Responsible-AI-Integration-Across-Scholarly-Research/default.aspx — MCP adoption, attribution/citation language. Dated July 9, 2025. Accessed 2026-04-22. *(Joint primary source; Anthropic quoted in the release.)*
8. **Will Scott — "How AI Licensing Deals Determine Search Visibility in 2025"**, Oct 2025. https://willscott.me/2025/10/04/ai-licensing-deals-search-visibility-in-2025/ — *secondary source, used only for the publisher block-rate figure cited in §4.* `[dated: 2025-10]` Accessed 2026-04-22.
9. **Anthropic — "Claude can now search the web"** (consumer product announcement). https://www.anthropic.com/news/web-search — dated March 20, 2025; update banner May 27, 2025 on global availability. `[dated: 2025-03]` Accessed 2026-04-22.

Additional docs consulted but not quoted:

- **Anthropic — "Web fetch tool."** https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/web-fetch-tool — JS-rendering limitation; URL constraints; optional citations.
- **Anthropic — "Tool use with Claude" (overview).** https://docs.anthropic.com/en/docs/agents-and-tools/tool-use/overview — server-side vs. client-side tool distinction that frames where `web_search` runs.
