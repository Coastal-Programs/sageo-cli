# Sageo CLI — Agent Skill

## What Sageo Is

Sageo is a command-line SEO/AEO/GEO analysis tool distributed as a single Go binary. It crawls websites, runs rule-based SEO audits, queries Google Search Console, and optionally enriches findings with paid SERP, Labs, AEO, and GEO intelligence from DataForSEO or SerpAPI. All commands return structured JSON wrapped in a consistent envelope (`success`, `data`, `error`, `metadata`). It is designed to be scripted, piped, and driven by agents without a UI.

---

## Recommended Workflow

1. **`sageo init --url <site>`** — Create `.sageo/state.json` in the current directory to track a site. Run once per project.

2. **`sageo audit run --url <site>`** — Crawl the site and run the full SEO audit. Saves score, page count, and all findings into `.sageo/state.json`. This is the primary data-gathering step and is free.

3. **`sageo status`** — Check what data exists in state: score, pages crawled, findings count, history, and which data sources are present (`sources_used`) vs missing (`sources_missing`). Read this before deciding what to run next.

4. **`sageo gsc sites list`** then **`sageo gsc sites use <property>`** — List the GSC properties the authenticated Google account has access to, then set the active property. Requires prior OAuth login (`sageo auth login gsc`).

5. **`sageo gsc query pages`** / **`sageo gsc query keywords`** — Pull page-level or keyword-level search performance data (clicks, impressions, CTR, position) from GSC for the active property. Defaults to the last 28 days. Use `--query` on `pages` to filter to a specific keyword, and `--page` on `keywords` to drill into which queries drive traffic to a specific URL.

5a. **`sageo gsc query trends`** — After pulling pages or keywords, check trends to see whether traffic is growing or declining over the date range. Useful for spotting seasonal drops or recent algorithm effects.

5b. **`sageo gsc query devices`** — If mobile performance looks different from desktop, run this to get the split by device type and compare CTR and position across devices.

5c. **`sageo gsc query countries`** / **`sageo gsc query appearances`** — Optionally break down traffic by country or by rich result type (web, image, video) to find untapped segments.

6. **`sageo gsc opportunities`** — Identify keyword and page opportunities derived from GSC data: high-impression/low-CTR queries, quick-win ranking gaps, etc.

7. **`sageo analyze`** — Compares crawl data with GSC data, finds cross-source issues that neither data source reveals alone. Free. Requires `init` + `audit run` first; GSC data is optional but strongly recommended for the best results.

8. **Read `state.json` `merged_findings`** — After `analyze`, the `merged_findings` array in `.sageo/state.json` contains prioritized action items combining crawl and GSC evidence. Work through these by priority (`HIGH` → `MEDIUM` → `LOW`).

9. **`sageo opportunities --with-serp --dry-run`** — Preview the cost of enriching GSC opportunity seeds with live SERP data before committing. Drop `--dry-run` to execute once you confirm the cost.

10. **Fix issues found in `state.json` findings** — Each finding includes `rule`, `url`, `verdict`, `why`, and `fix`. Work through findings by severity (`error` → `warning` → `info`), apply the recommended fix, then re-run `sageo audit run` to verify improvement.

---

## Data Tiers

| Tier | Cost | Requires | What you get |
|------|------|----------|-------------|
| **Tier 1** | Free | A URL | Crawl + SEO audit. Works for any public site with no credentials. |
| **Tier 2** | Free | Google OAuth | GSC search performance data: pages, keywords, trends, devices, countries, appearances, and opportunities. Requires `sageo auth login gsc` and a verified GSC property. |
| **Tier 3** | Paid | DataForSEO or SerpAPI account | Live SERP results, Labs domain/keyword intelligence, AEO AI-engine responses, GEO AI mention tracking. Always supports `--dry-run`. |

---

## Commands Reference

### Site Analysis

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `crawl run` | `sageo crawl run --url <url>` | Crawl a site and return raw page data (titles, links, status codes). | Free | `--depth` (default 2), `--max-pages` (default 50) |
| `audit run` | `sageo audit run --url <url>` | Crawl + run SEO rules, return scored findings, save to state.json if project exists. | Free | `--depth`, `--max-pages` |
| `report generate` | `sageo report generate --url <url>` | Crawl + audit + write a persistent JSON report to disk. | Free | `--depth`, `--max-pages` |
| `report list` | `sageo report list` | List all stored reports. | Free | — |

### Google Search Console

> All GSC commands require `sageo auth login gsc` and an active property set via `sageo gsc sites use`.

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `gsc sites list` | `sageo gsc sites list` | List all GSC properties accessible to the authenticated account. | Free (OAuth) | — |
| `gsc sites use` | `sageo gsc sites use <property_url>` | Set the active GSC property in config. | Free | — |
| `gsc query pages` | `sageo gsc query pages` | Return page-level performance: clicks, impressions, CTR, position. | Free (OAuth) | `--start-date`, `--end-date`, `--limit` (default 100), `--query` (filter by keyword) |
| `gsc query keywords` | `sageo gsc query keywords` | Return keyword-level performance data. | Free (OAuth) | `--start-date`, `--end-date`, `--limit` (default 100), `--page` (filter by URL) |
| `gsc query trends` | `sageo gsc query trends` | Return traffic trends aggregated by date (clicks, impressions over time). | Free (OAuth) | `--start-date`, `--end-date`, `--limit` (default 100) |
| `gsc query devices` | `sageo gsc query devices` | Return performance split by device type (mobile vs desktop vs tablet). | Free (OAuth) | `--start-date`, `--end-date`, `--limit` (default 100) |
| `gsc query countries` | `sageo gsc query countries` | Return traffic broken down by country. | Free (OAuth) | `--start-date`, `--end-date`, `--limit` (default 100) |
| `gsc query appearances` | `sageo gsc query appearances` | Return performance by rich result / search appearance type (e.g. web, image, video). | Free (OAuth) | `--start-date`, `--end-date`, `--limit` (default 100) |
| `gsc opportunities` | `sageo gsc opportunities` | Surface high-impression/low-CTR opportunity seeds from GSC data. | Free (OAuth) | `--start-date`, `--end-date`, `--limit` (default 1000) |

> **Common flag on all `gsc query` commands:** `--type` filters by search type (`web`, `image`, `video`, `news`).

### Cross-Source Analysis

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `analyze` | `sageo analyze` | Merges crawl and GSC data, produces prioritized cross-source findings saved to `state.json` `merged_findings`. | Free | — (no flags; requires `init` + `audit run` first; GSC data optional but recommended) |

### Opportunities (Combined)

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `opportunities` | `sageo opportunities` | Merge GSC opportunity seeds into a ranked list. | Free (OAuth) | `--start-date`, `--end-date`, `--limit` |
| `opportunities` | `sageo opportunities --with-serp` | Enrich GSC seeds with live SERP data for each top query. | Paid | `--with-serp`, `--serp-queries` (default 10), `--dry-run` |

### SERP Analysis

> Requires DataForSEO (`dataforseo_login` + `dataforseo_password`) or SerpAPI (`serp_api_key`). Always use `--dry-run` first.

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `serp analyze` | `sageo serp analyze --query <q>` | Fetch and analyze live SERP results for a single query. | Paid (~$0.002/query DFS, ~$0.01/query SerpAPI) | `--query` (required), `--location`, `--language`, `--num` (default 10), `--dry-run` |
| `serp compare` | `sageo serp compare --query <q1> --query <q2>` | Fetch and compare SERP results across multiple queries side by side. | Paid (per query) | `--query` (repeatable, min 2), `--location`, `--language`, `--dry-run` |

### AEO (Answer Engine Optimization)

> Requires DataForSEO credentials. Always use `--dry-run` first.

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `aeo responses` | `sageo aeo responses --prompt <text>` | Send a prompt to an AI engine and return its full response. | Paid (~$0.003/query) | `--prompt` (required), `--model` (`chatgpt`\|`claude`\|`gemini`\|`perplexity`, default `chatgpt`), `--dry-run` |
| `aeo keywords` | `sageo aeo keywords --keyword <kw>` | Return AI search volume data for a keyword across AI tools. | Paid (~$0.01/task) | `--keyword` (required), `--location`, `--language`, `--dry-run` |

### GEO (Generative Engine Optimization)

> Requires DataForSEO credentials. LLM Mentions requires a $100/month minimum commitment on the DataForSEO account. Always use `--dry-run` first.

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `geo mentions` | `sageo geo mentions --keyword <kw>` | Track how often a domain or brand is mentioned in AI-generated responses for a keyword. | Paid (~$0.01/task) | `--keyword` (required), `--domain`, `--platform` (`google`\|`bing`), `--dry-run` |
| `geo top-pages` | `sageo geo top-pages --keyword <kw>` | Show which URLs are most cited by AI engines for a keyword. | Paid (~$0.01/task) | `--keyword` (required), `--domain`, `--dry-run` |

### Labs (Competitive Intelligence)

> Requires DataForSEO credentials. Always use `--dry-run` first.

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `labs ranked-keywords` | `sageo labs ranked-keywords --target <domain>` | Get keywords a domain or URL currently ranks for in Google. | Paid (~$0.01/task) | `--target` (required), `--location` (default `United States`), `--language` (default `en`), `--limit` (default 50), `--min-volume`, `--dry-run` |
| `labs keywords` | `sageo labs keywords --target <domain>` | Get keyword ideas relevant to a domain. | Paid (~$0.01/task) | `--target` (required), `--location`, `--language`, `--limit` (default 50), `--dry-run` |
| `labs overview` | `sageo labs overview --target <domain>` | Get a domain's rank overview (organic traffic estimate, keyword count, etc). | Paid (~$0.01/task) | `--target` (required), `--location`, `--language`, `--dry-run` |
| `labs competitors` | `sageo labs competitors --target <domain>` | Identify competing domains that rank for overlapping keywords. | Paid (~$0.01/task) | `--target` (required), `--location`, `--language`, `--limit` (default 20), `--dry-run` |
| `labs keyword-ideas` | `sageo labs keyword-ideas --keyword <kw>` | Expand a seed keyword into related keyword ideas. | Paid (~$0.01/task) | `--keyword` (required), `--location`, `--language`, `--limit` (default 50), `--dry-run` |

### Config & Auth

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `login` | `sageo login` | Interactive TUI to configure GSC OAuth, DataForSEO, and SerpAPI credentials. | Free | — |
| `logout` | `sageo logout` | Clear all stored credentials and API keys. | Free | — |
| `auth login` | `sageo auth login gsc` | Start the GSC OAuth browser flow. Requires `gsc_client_id` and `gsc_client_secret` to be set. | Free | — |
| `auth logout` | `sageo auth logout gsc` | Remove stored OAuth token for a service. | Free | — |
| `auth status` | `sageo auth status` | Show authentication status (authenticated, token present/expired) for all services. | Free | — |
| `config show` | `sageo config show` | Show the full config with sensitive values redacted. | Free | — |
| `config get` | `sageo config get <key>` | Read a single config value. | Free | — |
| `config set` | `sageo config set <key> <value>` | Write a single config value. | Free | — |
| `config path` | `sageo config path` | Print the path to the active config file. | Free | — |
| `provider list` | `sageo provider list` | List available fetch providers and show which is active. | Free | — |
| `provider use` | `sageo provider use <name>` | Set the active fetch provider (e.g. `local`). | Free | — |
| `version` | `sageo version` | Print CLI version and Go runtime info. | Free | — |

### Project

| Command | Usage | What it does | Cost | Key flags |
|---------|-------|-------------|------|-----------|
| `init` | `sageo init --url <site>` | Initialize a `.sageo/state.json` project file in the current directory. | Free | `--url` (required) |
| `status` | `sageo status` | Show a summary of the current project state: site, score, pages crawled, findings count, history count, and which data sources are populated vs missing. | Free | — |

---

## Reading state.json

State is stored at `.sageo/state.json` relative to the project root. Structure:

```json
{
  "site": "https://example.com",
  "initialized": "2025-01-01T00:00:00Z",
  "last_crawl": "2025-01-02T12:00:00Z",
  "score": 74.5,
  "pages_crawled": 42,
  "findings": [
    {
      "rule": "missing-meta-description",
      "url": "https://example.com/about",
      "value": "",
      "verdict": "error",
      "why": "Pages without a meta description get auto-generated snippets that reduce CTR.",
      "fix": "Add a unique meta description (120–160 chars) to this page."
    }
  ],
  "history": [
    {
      "ts": "2025-01-02T12:00:00Z",
      "action": "audit",
      "detail": "score=74.5 issues=18 pages=42"
    }
  ]
}
```

**Field guide:**

- **`site`** — The URL this project tracks.
- **`score`** — Overall SEO health score (0–100). Higher is better.
- **`pages_crawled`** — Number of pages visited in the last crawl.
- **`findings`** — Array of all audit issues. Each entry has:
  - `rule` — Machine-readable rule ID (e.g. `missing-meta-description`, `broken-link`, `slow-page`).
  - `url` — The specific page the finding applies to.
  - `value` — The raw value that triggered the rule (empty string, a number, a URL, etc).
  - `verdict` — Severity: `error`, `warning`, or `info`.
  - `why` — Human-readable explanation of why this matters for SEO.
  - `fix` — Actionable recommendation for resolving the issue.
- **`history`** — Ordered log of actions (crawls, audits) with timestamps and summary details. Use this to avoid unnecessary re-crawls.

**Reading strategy:** Sort findings by `verdict` (`error` first), then group by `rule` to see patterns across multiple pages. The `fix` field is authoritative — apply it directly.

---

## Cross-Source Findings

Cross-source findings only exist when crawl data is compared against GSC data via `sageo analyze`. They surface issues that neither data source can reveal on its own. Results are stored in `state.json` under `merged_findings`, each with a `priority` (`HIGH`, `MEDIUM`, `LOW`).

### Rules

| Rule | What it means | What to do |
|------|--------------|------------|
| `ranking-but-not-clicking` | A page ranks well in Google (good position) but has abnormally low CTR, meaning users see it but don't click. | Rewrite the title tag and meta description to be more compelling; consider adding structured data for rich snippets. |
| `not-indexed` | A page exists on the site (found by crawl) but does not appear in any GSC query data, suggesting Google may not have indexed it. | Check for `noindex` tags, canonical issues, or crawl blocks; submit the URL via GSC's URL Inspection tool. |
| `issues-on-high-traffic-page` | A page that receives significant GSC traffic has SEO audit errors (e.g. missing meta description, broken links, slow load). | Fix the audit findings on this page first — it already has traffic, so improvements here have outsized impact. |
| `thin-content-ranking-well` | A page with very little content (low word count detected by crawl) is still ranking and getting impressions in GSC. | Expand the content with useful, relevant information — the page has ranking potential but thin content puts it at risk of losing position. |
| `schema-not-showing` | A page has structured data / schema markup (detected by crawl) but GSC shows no rich result appearances for it. | Validate the schema with Google's Rich Results Test; fix any errors so Google can display rich snippets for this page. |

---

## Do's and Don'ts

### Do
- **Always `--dry-run` before any paid command.** Confirms the estimated cost before money is spent.
- **Read `state.json` before re-crawling.** If `last_crawl` is recent and findings are already populated, skip straight to analysis.
- **Check `sources_missing` in `sageo status`.** It tells you exactly what data you still need (e.g. `gsc` not yet populated).
- **Use `sageo gsc opportunities` before `sageo opportunities --with-serp`.** GSC opportunities are free and often sufficient; SERP enrichment is an optional paid upgrade.
- **Cache is automatic for SERP/Labs calls.** Results are cached for 1 hour. If you re-run the same query within that window, it won't cost anything.
- **Prioritize `error` verdict findings.** They have the highest SEO impact and clearest fixes.
- **Always run `sageo analyze` after pulling GSC data.** That's where the real insights are — cross-source findings reveal issues invisible to either data source alone.
- **Fix `HIGH` priority merged findings first.** They're costing you traffic right now and have the clearest path to improvement.

### Don't
- **Never retry a failed paid command immediately.** If a DataForSEO or SerpAPI call fails, the charge may still have been incurred. Investigate the error first.
- **Don't run `sageo audit run` again just to check progress.** Run it only after you've actually changed something on the site.
- **Don't run `sageo init` in a directory that already has `.sageo/state.json`.** It will return an error — that's intentional to prevent overwriting state.
- **Don't use `--with-serp` without first confirming the SERP provider is configured.** Check `sageo config show` or `sageo auth status` first.
- **Don't pass secrets via `config set` in logged environments.** Use `sageo login` (interactive TUI) or environment variables (`SAGEO_DATAFORSEO_LOGIN`, etc.) instead.
- **Don't assume a non-zero exit code means no data was returned.** Always check the `error` field in the JSON envelope — partial failures may still include useful data in `data`.
