# OpenAI: ChatGPT Search & SearchGPT — Publisher Guidance

**Research date:** 2026-04-22
**Scope:** Primary sources only (openai.com, platform.openai.com, developers.openai.com, help.openai.com).

---

## TL;DR

- OpenAI exposes **two independent robots.txt controls** that publishers care about for search: `OAI-SearchBot` (controls inclusion in ChatGPT search answers) and `GPTBot` (controls use of content in foundation-model training). A third agent, `ChatGPT-User`, handles user-initiated fetches and is not used to decide search inclusion. [1][5]
- **To appear in ChatGPT search results, a publisher must allow `OAI-SearchBot`.** Sites blocked to `OAI-SearchBot` will not appear in ChatGPT search answers, though they may still appear as navigational links. [1][5]
- OpenAI states ChatGPT search leverages **third-party search providers plus partner content**; the underlying model is a fine-tuned GPT-4o (as of launch in Oct 2024). [2]
- OpenAI publishes **almost no specific prescriptive guidance** on content structure (headings, lists, schema.org, etc.) for surfacing in ChatGPT search. The only structural advice published is about **ARIA / WAI-ARIA tagging** — and that is specifically for ChatGPT **Agent** in the Atlas browser, not for search ranking. [5]
- Referral traffic from ChatGPT search carries `utm_source=chatgpt.com`. [5]
- Robots.txt changes take ~24 hours to propagate into search behavior. [1]
- **Named publisher partners** announced at ChatGPT Search launch: Associated Press, Axel Springer, Condé Nast, Dotdash Meredith, Financial Times, GEDI, Hearst, Le Monde, News Corp, Prisa (El País), Reuters, The Atlantic, Time, Vox Media. OpenAI states "any website or publisher can choose to appear" — partnership is not required. [2]
- OpenAI has published **nothing I could find** on citation selection criteria, structured data / schema.org preferences, or freshness signals beyond generic statements.

---

## 1. Citation selection

**What OpenAI publishes:** Very little specificity. The only on-record mechanical description is:

> "The search model is a fine-tuned version of GPT-4o, post-trained using novel synthetic data generation techniques, including distilling outputs from OpenAI o1-preview." [2]

And:

> "ChatGPT search leverages third-party search providers, as well as content provided directly by our partners, to provide the information users are looking for." [2]

**The SearchGPT prototype page describes citation design intent** (not ranking):

> "SearchGPT is designed to help users connect with publishers by prominently citing and linking to them in searches. Responses have clear, in-line, named attribution and links…" [3]

**Help Center (publisher FAQ)** adds one concrete mechanism — indirect surfacing:

> "if we obtain the URL of a disallowed page from a third-party search provider or by crawling other pages and have signals the page is relevant to a user's query, we may surface just the link and page title in ChatGPT Atlas." [5]

**What is NOT published:** OpenAI does not publicly document
- how it ranks or selects among candidate sources,
- any authority/reputation signals,
- E-E-A-T-style heuristics,
- how many sources are typically cited, or
- how it deduplicates overlapping sources.

---

## 2. Content structure preferences

**What OpenAI publishes:** Effectively nothing prescriptive for ChatGPT Search ranking.

The only structural guidance aimed at publishers/webmasters on help.openai.com is for the **ChatGPT Atlas browser's agent**, not for search surfacing:

> "ChatGPT Atlas uses ARIA tags—the same labels and roles that support screen readers—to interpret page structure and interactive elements." [5]

> "To improve compatibility, follow WAI-ARIA best practices by adding descriptive roles, labels, and states to interactive elements like buttons, menus, and forms." [5]

**What is NOT published by OpenAI:**
- No guidance on heading hierarchy (H1/H2/H3)
- No guidance on list formatting, tables, or "direct answer" patterns
- No word-count or content-length preferences
- No formal E-E-A-T or quality rater doctrine
- No published preference for extractive-friendly passages

This is a notable gap: unlike Google's Search Central documentation, OpenAI publishes **no content-structure playbook** for writers targeting ChatGPT search.

---

## 3. Crawler directives (OAI-SearchBot, GPTBot, ChatGPT-User)

Primary source: `platform.openai.com/docs/bots` (a.k.a. `developers.openai.com/api/docs/bots`). [1]

### Independence of controls

> "OpenAI uses OAI-SearchBot and GPTBot robots.txt tags to enable webmasters to manage how their sites and content work with AI. Each setting is independent of the others…" [1]

> "a webmaster can allow OAI-SearchBot in order to appear in search results while disallowing GPTBot to indicate that crawled content should not be used for training OpenAI's generative AI foundation models." [1]

> "If your site has allowed both bots, we may use the results from just one crawl for both use cases to avoid duplicative crawling." [1]

> "For search results, please note it can take ~24 hours from a site's robots.txt update for our systems to adjust." [1]

### OAI-SearchBot (search inclusion)

- Purpose: "used to surface websites in search results in ChatGPT's search features." [1]
- Consequence of opt-out: "Sites that are opted out of OAI-SearchBot will not be shown in ChatGPT search answers, though can still appear as navigational links." [1]
- User-agent string (as of Apr 2026): `Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36; compatible; OAI-SearchBot/1.3; +https://openai.com/searchbot` [1]
- Published IP ranges: `https://openai.com/searchbot.json` [1]

### GPTBot (training)

- Purpose: "used to make our generative AI foundation models more useful and safe… used to crawl content that may be used in training our generative AI foundation models." [1]
- Opt-out effect: "Disallowing GPTBot indicates a site's content should not be used in training generative AI foundation models." [1]
- User-agent string: `Mozilla/5.0 AppleWebKit/537.36 (KHTML, like Gecko); compatible; GPTBot/1.3; +https://openai.com/gptbot` [1]
- Published IP ranges: `https://openai.com/gptbot.json` [1]

### ChatGPT-User (user-initiated fetch)

- "ChatGPT-User is not used for crawling the web in an automatic fashion. Because these actions are initiated by a user, robots.txt rules may not apply." [1]
- Critically for search: "ChatGPT-User is not used to determine whether content may appear in Search. Please use OAI-SearchBot in robots.txt for managing Search opt outs and automatic crawl." [1]

### OAI-AdsBot

- Used only to validate landing pages for submitted ads. "the data collected by OAI-AdsBot is not used to train generative AI foundation models." [1]

### Key difference summary (OAI-SearchBot vs GPTBot)

| Aspect | OAI-SearchBot | GPTBot |
|---|---|---|
| Purpose | Index for ChatGPT Search | Crawl for model training |
| Opt-out removes from search? | Yes | No |
| Opt-out removes from training? | No (separate control) | Yes |
| Referenced at | openai.com/searchbot | openai.com/gptbot |

---

## 4. Publisher programs

**Named launch partners (ChatGPT Search, Oct 31, 2024 announcement):** Associated Press, Axel Springer, Condé Nast, Dotdash Meredith, Financial Times, GEDI, Hearst, Le Monde, News Corp, Prisa (El País), Reuters, The Atlantic, Time, Vox Media. [2] `[dated: 2024-10]`

**Universality claim** — from the same post:

> "Any website or publisher can choose to appear in ChatGPT search." [2] `[dated: 2024-10]`

**Feedback channel:** `publishers-feedback@openai.com` (stated on both the SearchGPT prototype page [3] and the ChatGPT Search launch post [2]).

**Search vs training separation is explicit:**

> "SearchGPT is about search and is separate from training OpenAI's generative AI foundation models. Sites can be surfaced in search results even if they opt out of generative AI training." [3] `[dated: 2024-07]`

**What is NOT published:**
- No public-tier structure for the publisher program
- No application/onboarding page for being added as a "partner"
- No public documentation of what partner content access differs from standard crawled content, beyond the generic "provided directly by our partners" phrasing in [2]

---

## 5. Structured data

**What OpenAI publishes about schema.org / JSON-LD / microdata: nothing that I could locate on platform.openai.com, openai.com/blog, or help.openai.com as of 2026-04-22.**

The closest adjacent guidance is:
- WAI-ARIA tagging (for the Atlas agent, not search) [5]
- The `noindex` meta tag is acknowledged: "If you do not want this to happen, use the noindex meta tag. Note, in order for our crawler to read a meta tag, it must be allowed to crawl the relevant page(s)." [5]

OpenAI has **not** published any of the following for ChatGPT Search:
- A schema.org type preference list (Article, NewsArticle, FAQPage, HowTo, etc.)
- Any guidance on OpenGraph / Twitter Card usage
- Any sitemap submission endpoint or mechanism
- Any equivalent of Google's "structured data guidelines"

This is an explicit gap and worth flagging to users of this research.

---

## 6. Recent statements (2025–2026)

### Feb 5, 2025 — ChatGPT Search becomes universal

From the updated launch post [2]:

> "Update on February 5, 2025: ChatGPT search is now available to everyone in regions where ChatGPT is available. No signup required."

And earlier:

> "Update on December 16, 2024: ChatGPT search is now available to all logged-in users in regions where ChatGPT is available."

### 2025–2026 — Atlas browser & publisher FAQ

The `help.openai.com` "Publishers and Developers — FAQ" [5] (last updated ~mid-April 2026 based on "8 days ago" seen on 2026-04-22) is the most current publisher-facing OpenAI document. It codifies:
- OAI-SearchBot as the search-inclusion control.
- `utm_source=chatgpt.com` as the referral-tracking parameter.
- GPTBot disallow as the training opt-out signal, which OpenAI "respect[s] … for content acquired via users' interactions in Atlas."
- A caveat that opted-out URLs may still appear as link + page title if obtained from a third-party provider.

### User-agent version bumps

OAI-SearchBot's user-agent string advanced to `OAI-SearchBot/1.3` (from earlier 1.0/1.1 documented in 2024–2025 archives). GPTBot is at `GPTBot/1.3`. [1]

### Citation / attribution commitments

Publisher quotes in the launch post reflect OpenAI's public stance on attribution, e.g.:

> "ChatGPT search promises to better highlight and attribute information from trustworthy news sources…" — Pam Wasserstein, Vox Media [2]

OpenAI's own framing:

> "ChatGPT search connects people with original, high-quality content from the web and makes it part of their conversation." [2]

### Freshness

OpenAI has **not** published a specific freshness/recency doctrine for ChatGPT Search ranking. The only freshness-adjacent statement is generic: search is designed to give "fast, timely answers with links to relevant web sources" and uses real-time web retrieval. [2][3]

---

## Gaps / open questions

1. **Ranking signals.** OpenAI does not document how sources are selected or ranked once crawled. Any third-party claims about "E-E-A-T for ChatGPT" have no primary-source basis.
2. **Content structure.** No prescriptive on-page guidance — no formal word on headings, lists, tables, direct-answer patterns, or TL;DR sections.
3. **Schema / structured data.** Completely undocumented. Whether JSON-LD influences surfacing is unstated.
4. **Freshness model.** Unknown — no documented re-crawl cadence or recency-boost mechanism for news/evergreen content.
5. **Sitemap submission.** No OpenAI-operated submission endpoint; discovery appears to be purely organic plus third-party search providers.
6. **"Provided directly by our partners" pipeline.** The technical shape of partner content ingestion is not publicly described (feed format, schema, update cadence).
7. **Citation count / diversity logic.** OpenAI does not state how many citations are typically used or how duplicate sources are handled.
8. **Differences between ChatGPT Search, Atlas, and the API `web_search` tool.** Only partially documented; the crawler bots overview treats them uniformly, but the user-facing products' ranking behavior may differ.
9. **`/.well-known/` or `ai.txt`-style additions.** OpenAI honors only classic `robots.txt` + meta tags; no OpenAI-specific well-known file has been introduced.

---

## Sources

All URLs accessed **2026-04-22** unless otherwise noted. Items marked `[dated: YYYY-MM]` are >24 months old.

1. **Overview of OpenAI Crawlers** — `https://platform.openai.com/docs/bots` (also mirrored at `https://developers.openai.com/api/docs/bots`). Primary technical reference for OAI-SearchBot, GPTBot, ChatGPT-User, OAI-AdsBot. Accessed 2026-04-22.
2. **Introducing ChatGPT search** — `https://openai.com/index/introducing-chatgpt-search/`. OpenAI blog, published 2024-10-31; updated 2024-12-16 and 2025-02-05. `[dated: 2024-10]` (original post, though kept current with updates). Accessed 2026-04-22.
3. **SearchGPT Prototype** — `https://openai.com/index/searchgpt-prototype/`. OpenAI blog, 2024-07-25. `[dated: 2024-07]`. Accessed 2026-04-22.
4. **Web search (API guide)** — `https://platform.openai.com/docs/guides/tools-web-search`. Describes the API `web_search` tool (distinct from consumer ChatGPT Search). Accessed 2026-04-22.
5. **Publishers and Developers — FAQ** — `https://help.openai.com/en/articles/12627856-publishers-and-developers-faq`. Last updated ~2026-04 ("8 days ago" on 2026-04-22). The most current OpenAI publisher-facing doc. Accessed 2026-04-22.

### Checked and not found / not applicable
- `https://openai.com/searchgpt/` — 404 (prototype waitlist page has been removed since ChatGPT Search launch).
- `https://openai.com/chatgpt/search/` — 404.
- `https://help.openai.com/en/articles/9237897-chatgpt-search` — 403 (access forbidden from fetch).
- No dedicated schema.org / structured-data documentation found anywhere under openai.com, platform.openai.com, help.openai.com, or developers.openai.com.
