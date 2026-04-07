# Sageo CLI — Agent Skill

## What Sageo Is

Sageo is a JSON-first CLI for SEO/AEO/GEO workflows. It crawls, audits, pulls GSC, enriches with PSI/SERP/Labs/Backlinks, and merges everything into prioritized findings in `.sageo/state.json`.

---

## Recommended Workflow (Run in this order)

1. `sageo init --url <site>`
2. `sageo audit run --url <site>` *(auto-runs PSI on top 5 pages unless `--skip-psi`)*
3. `sageo status`
4. `sageo auth login gsc` → `sageo gsc sites use <property>` → `sageo gsc query pages` → `sageo gsc query keywords`
5. `sageo labs ranked-keywords --target <domain>` *(adds keyword difficulty + intent)*
6. `sageo labs bulk-difficulty --from-gsc` *(difficulty for all GSC keywords in one cheap call)*
7. `sageo serp batch --keywords "top,gsc,keywords"` *(batch SERP features at ~$0.0006/query)*
8. `sageo backlinks summary --target <domain>` *(backlink profile)*
9. `sageo backlinks gap --target <domain>` *(link gap vs competitors)*
10. `sageo analyze` *(merge all sources into prioritized findings)*
11. Read `.sageo/state.json` → `merged_findings`

---

## Data Tiers

| Tier | Cost | Data source | Notes |
|---|---:|---|---|
| Tier 1 | Free | Crawl + Audit | Core technical SEO findings and score. |
| Tier 2 | Free | PSI + GSC | PSI is free (key optional for higher limits). GSC requires OAuth + property access. |
| Tier 3 | Paid | SERP + Labs | SERP features, keyword difficulty, intent, and competitive intel. Use `--dry-run`. |
| Tier 4 | Paid | Backlinks | Authority profile + gap analysis (`summary`, `list`, `referring-domains`, `competitors`, `gap`). |

---

## Commands Reference

### Site Analysis

| Command | Usage | Cost | Purpose |
|---|---|---:|---|
| `crawl run` | `sageo crawl run --url <url>` | Free | Raw crawl data. |
| `audit run` | `sageo audit run --url <url>` | Free | Crawl + audit, saves state. |
| `psi run` | `sageo psi run --url <page> --strategy mobile` | Free | Core Web Vitals + Lighthouse score. |
| `report generate` | `sageo report generate --url <url>` | Free | Persisted report artifact. |
| `report list` | `sageo report list` | Free | List saved reports. |

### Project

| Command | Usage | Cost | Purpose |
|---|---|---:|---|
| `init` | `sageo init --url <site>` | Free | Create `.sageo/state.json`. |
| `status` | `sageo status` | Free | Show available/missing data sources. |
| `analyze` | `sageo analyze` | Free | Merge sources into `merged_findings`. |

### Google Search Console

| Command | Usage |
|---|---|
| `gsc sites list` | `sageo gsc sites list` |
| `gsc sites use` | `sageo gsc sites use <property_url>` |
| `gsc query pages` | `sageo gsc query pages` |
| `gsc query keywords` | `sageo gsc query keywords` |
| `gsc query trends` | `sageo gsc query trends` |
| `gsc query devices` | `sageo gsc query devices` |
| `gsc query countries` | `sageo gsc query countries` |
| `gsc query appearances` | `sageo gsc query appearances` |
| `gsc opportunities` | `sageo gsc opportunities` |

### SERP

| Command | Usage | Cost |
|---|---|---:|
| `serp analyze` | `sageo serp analyze --query "<keyword>"` | Paid |
| `serp compare` | `sageo serp compare --query "q1" --query "q2"` | Paid |
| `serp batch` | `sageo serp batch --keywords "k1,k2,k3"` | Paid *(~70% cheaper vs individual live analyze)* |

### Labs

| Command | Usage | Cost |
|---|---|---:|
| `labs ranked-keywords` | `sageo labs ranked-keywords --target <domain>` | Paid |
| `labs keywords` | `sageo labs keywords --target <domain>` | Paid |
| `labs overview` | `sageo labs overview --target <domain>` | Paid |
| `labs competitors` | `sageo labs competitors --target <domain>` | Paid |
| `labs keyword-ideas` | `sageo labs keyword-ideas --keyword "<kw>"` | Paid |
| `labs search-intent` | `sageo labs search-intent --keywords "kw1,kw2,kw3"` | Paid |
| `labs bulk-difficulty` | `sageo labs bulk-difficulty --from-gsc` | Paid *(low cost bulk difficulty)* |

### Backlinks

| Command | Usage | Cost |
|---|---|---:|
| `backlinks summary` | `sageo backlinks summary --target <domain>` | Paid |
| `backlinks list` | `sageo backlinks list --target <domain>` | Paid |
| `backlinks referring-domains` | `sageo backlinks referring-domains --target <domain>` | Paid |
| `backlinks competitors` | `sageo backlinks competitors --target <domain>` | Paid |
| `backlinks gap` | `sageo backlinks gap --target <domain>` | Paid |

---

## Cross-Source Findings (`merged_findings`)

| Rule | Sources | Meaning | Action |
|---|---|---|---|
| `ranking-but-not-clicking` | crawl + gsc | Ranking/impressions but weak clicks. | Improve title/meta + fix page issues. |
| `not-indexed` | crawl + gsc | Crawled page not showing in GSC. | Check indexing/noindex/canonicals/internal links. |
| `issues-on-high-traffic-page` | crawl + gsc | High-traffic page has technical issues. | Fix these first (highest impact). |
| `thin-content-ranking-well` | crawl + gsc | Thin page ranks now but vulnerable. | Expand content depth. |
| `schema-not-showing` | crawl + gsc | Schema present, rich-result performance weak. | Validate/fix structured data. |
| `slow-core-web-vitals` | psi + gsc | Poor PSI score on pages with impressions. | Fix LCP/CLS/perf bottlenecks. |
| `ai-overview-eating-clicks` | serp + gsc | AI Overview likely absorbing clicks. | Optimize for AI citation + direct answers. |
| `featured-snippet-opportunity` | serp + gsc | You can capture snippet position 0. | Add concise answer block + strong formatting. |
| `paa-content-opportunity` | serp + crawl | PAA questions not covered on page. | Add FAQ/H2 answers. |
| `easy-win-keyword` | labs + gsc | Low difficulty keyword with rank upside. | Improve content and internal links. |
| `informational-content-gap` | labs | Missing informational topics with opportunity. | Create new content pages. |
| `weak-backlink-profile` | backlinks + labs | Authority too weak for target keyword set. | Start focused link-building. |
| `broken-backlinks-found` | backlinks | External links point to dead URLs. | Add 301 redirects to reclaim equity. |

---

## `state.json` Structure (Agent-facing)

```json
{
  "site": "https://example.com",
  "initialized": "...",
  "last_crawl": "...",
  "score": 74.5,
  "pages_crawled": 42,
  "findings": [],
  "gsc": {
    "last_pull": "...",
    "property": "...",
    "top_pages": [],
    "top_keywords": []
  },
  "psi": {
    "last_run": "...",
    "pages": []
  },
  "serp": {
    "last_run": "...",
    "queries": []
  },
  "labs": {
    "last_run": "...",
    "target": "...",
    "keywords": [],
    "competitors": []
  },
  "backlinks": {
    "last_run": "...",
    "target": "...",
    "total_backlinks": 0,
    "total_referring_domains": 0,
    "broken_backlinks": 0,
    "gap_domains": []
  },
  "merged_findings": [],
  "last_analysis": "...",
  "history": []
}
```

---

## Do / Don’t

### Do
- Run paid commands with `--dry-run` first.
- Run `sageo labs bulk-difficulty --from-gsc` after pulling GSC keywords (one cheap bulk call for all keywords).
- Use `sageo serp batch` instead of many `serp analyze` calls (much cheaper).
- Run `sageo backlinks summary` at least once for backlink profile baseline.
- Re-run `sageo analyze` after adding new source data.

### Don’t
- Don’t re-run paid commands blindly after failures.
- Don’t re-audit repeatedly without site changes.
- Don’t run `init` in a directory that already has `.sageo/state.json`.
