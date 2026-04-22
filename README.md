<div align="center">
<pre>
███████╗ █████╗  ██████╗ ███████╗ ██████╗      ██████╗██╗     ██╗
██╔════╝██╔══██╗██╔════╝ ██╔════╝██╔═══██╗    ██╔════╝██║     ██║
███████╗███████║██║  ███╗█████╗  ██║   ██║    ██║     ██║     ██║
╚════██║██╔══██║██║   ██║██╔══╝  ██║   ██║    ██║     ██║     ██║
███████║██║  ██║╚██████╔╝███████╗╚██████╔╝    ╚██████╗███████╗██║
╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝ ╚═════╝      ╚═════╝╚══════╝╚═╝
</pre>

<p align="center">
  <a href="https://go.dev/">
    <img src="https://img.shields.io/badge/go-%3E%3D1.26-00ADD8.svg" alt="Go Version">
  </a>
  <a href="https://github.com/Coastal-Programs/sageo-cli/actions">
    <img src="https://github.com/Coastal-Programs/sageo-cli/actions/workflows/test.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/Coastal-Programs/sageo-cli/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
  </a>
</p>
</div>

**Open-source SEO CLI tool** — crawl, audit, and optimize websites from the command line. Single Go binary. JSON output on every command.

Sageo replaces the need to juggle multiple SEO tools. It crawls your site, runs a technical SEO audit, pulls Google Search Console data, checks PageSpeed (Core Web Vitals), analyzes SERP features (AI Overviews, Featured Snippets, People Also Ask), gets keyword difficulty and search intent, audits your backlink profile, queries real AI engines (ChatGPT, Claude, Gemini, Perplexity) for brand visibility, generates concrete change recommendations with LLM-drafted copy, forecasts the click lift of each change, and can drive the whole pipeline end-to-end with a single `sageo run <url>` command — all from one CLI.

Crawling, auditing, and PageSpeed Insights are free. Paid features (SERP, keyword research, backlinks, AEO/GEO, LLM drafting) are cost-estimated and budget-gated — every paid command supports `--dry-run`, and `sageo run` accepts a hard `--budget` ceiling.

## What it does

- **Crawl + audit** — BFS crawler, rule-based audit, 0–100 score
- **Google Search Console** — pages, keywords, trends, devices, countries
- **PageSpeed Insights** — Core Web Vitals (LCP, CLS, FCP, TBT, SI)
- **SERP analysis** — 9 feature types including AI Overviews, Featured Snippets, PAA
- **Labs** — keyword difficulty, volume, intent, ranked keywords, competitors
- **Backlinks** — summary, referring domains, competitor gap
- **Multi-model AEO** — fan out a prompt across ChatGPT, Claude, Gemini, and Perplexity in parallel, capture responses, detect brand mentions
- **Brand mentions** — local scan of stored AEO responses (free) plus DataForSEO LLM Mentions API (search / top-pages / top-domains)
- **Recommendation engine** — merge rules emit atomic `Recommendation` objects (title, meta, H1/H2, schema, body, speed, backlink, indexability) with evidence and priority
- **LLM copy drafting** — Anthropic or OpenAI drivers fill `recommended_value` with real copy, validated against SERP length limits
- **Click-lift forecaster** — attaches estimated monthly clicks delta per recommendation using the AWR 2024 position→CTR curve
- **`sageo run` autonomous pipeline** — crawl → audit → GSC → PSI → Labs → SERP → backlinks → AEO fan-out → mentions → merge → recommend → draft → forecast, with `--budget`, `--skip`, `--only`, `--resume`, `--approve`
- **HTML report** — styled, client-ready self-contained HTML file (cover, exec summary, per-source findings, recommendation cards, sortable forecast table, optional appendix). For a PDF, open the file in your browser and press Cmd/Ctrl+P → Save as PDF.

## Install

```bash
git clone https://github.com/Coastal-Programs/sageo-cli.git
cd sageo-cli
make build
# binary: build/sageo
```

Requires Go 1.26+.

### Run without installing globally

```bash
go run ./cmd/sageo --help
go run ./cmd/sageo login
```

### Install globally (dev)

```bash
make install
export PATH="$HOME/go/bin:$PATH"

# verify
which sageo
sageo --help
```

> Add `export PATH="$HOME/go/bin:$PATH"` to `~/.zshrc` (or your shell profile) to make this permanent.

## Quick Start

The fastest path — one autonomous run plus a PDF:

```bash
sageo login
sageo run https://yoursite.com --budget 5
sageo report html
```

That single `run` drives crawl → audit → GSC → PSI → Labs → SERP → backlinks → AEO → merge → recommendations → LLM draft → forecast, capped at $5 of paid API spend.

Or run each stage manually:

```bash
# 1. Set up credentials
sageo login

# 2. Initialize a project
sageo init --url https://yoursite.com

# 3. Audit your site (includes automatic PageSpeed check)
sageo audit run --url https://yoursite.com --depth 2 --max-pages 50

# 4. Connect Google Search Console
sageo auth login gsc
sageo gsc sites use https://yoursite.com/
sageo gsc query pages
sageo gsc query keywords

# 5. Run cross-source analysis
sageo analyze

# 6. Inspect and draft recommendations
sageo recommendations list --top 20
sageo recommendations draft --limit 20
sageo recommendations forecast

# 7. Render the client-ready report
sageo report html --output ./report.html
# (open in a browser and press Cmd/Ctrl+P → Save as PDF for a PDF copy)

# 8. Check your status
sageo status
```

## Commands

### Site crawl and audit

```bash
sageo crawl run --url https://example.com --depth 2 --max-pages 50
sageo audit run --url https://example.com --depth 2 --max-pages 50
sageo audit run --url https://example.com --depth 2 --max-pages 50 --skip-psi
sageo report generate --url https://example.com
sageo report list
```

`audit run` automatically runs PageSpeed Insights for top pages unless you pass `--skip-psi`.

### Project management

```bash
sageo init --url https://example.com
sageo status
sageo analyze
```

### Google Search Console

```bash
sageo gsc sites list
sageo gsc sites use https://example.com/

sageo gsc query pages --start-date 2026-03-01 --end-date 2026-03-28 --query "brand term" --type web
sageo gsc query keywords --limit 50 --page "/pricing" --type web
sageo gsc query trends --type web
sageo gsc query devices --type web
sageo gsc query countries --type web
sageo gsc query appearances --type web

sageo gsc opportunities
```

### PageSpeed Insights

```bash
sageo psi run --url https://example.com --strategy mobile
```

- PSI uses your GSC OAuth token automatically if you've run `sageo auth login gsc`. No separate API key needed.
- Optionally set `psi_api_key` for higher rate limits (25,000/day vs OAuth limits).

### SERP

```bash
sageo serp analyze --query "seo tools" --dry-run
sageo serp analyze --query "seo tools"
sageo serp compare --query "seo tools" --query "seo software"
sageo serp batch --keywords "keyword1,keyword2,keyword3" --dry-run
```

### Labs

```bash
sageo labs ranked-keywords --target example.com --dry-run
sageo labs keywords --target example.com --limit 50
sageo labs overview --target example.com
sageo labs competitors --target example.com --limit 20
sageo labs keyword-ideas --keyword "seo tools" --limit 50
sageo labs search-intent --keywords "kw1,kw2,kw3"
sageo labs bulk-difficulty --from-gsc --dry-run
```

### Backlinks

```bash
sageo backlinks summary --target example.com --dry-run
sageo backlinks list --target example.com --limit 50 --dofollow-only
sageo backlinks referring-domains --target example.com
sageo backlinks competitors --target example.com
sageo backlinks gap --target example.com
```

### AEO (Answer Engine Optimization)

```bash
# List the live AI model catalogue (cached 7 days)
sageo aeo models

# Single engine
sageo aeo responses --prompt "What is Sageo CLI?" --engine chatgpt --dry-run
sageo aeo responses --prompt "What is Sageo CLI?" --engine claude

# Fan out to all engines in parallel
sageo aeo responses --prompt "What is Sageo CLI?" --all --tier flagship
sageo aeo responses --prompt "What is Sageo CLI?" --models gpt-5,claude-sonnet-4-6

# AI search volume for a keyword
sageo aeo keywords --keyword "seo tools" --location "United States"

# Brand mention detection
sageo aeo mentions scan                               # Layer A (local, free)
sageo aeo mentions search --keyword "seo tools"      # Layer B (DataForSEO)
sageo aeo mentions top-pages --keyword "seo tools"
sageo aeo mentions top-domains --keyword "seo tools"
```

Supported engines: `chatgpt`, `claude`, `gemini`, `perplexity`

### Recommendations

```bash
sageo recommendations list --top 20
sageo recommendations list --url https://yoursite.com/pricing --type title
sageo recommendations draft --limit 20 --provider anthropic --dry-run
sageo recommendations draft --limit 20                         # fills RecommendedValue via LLM
sageo recommendations forecast                                 # attach click-lift estimates
```

### Autonomous run

```bash
sageo run https://yoursite.com --budget 5
sageo run https://yoursite.com --skip backlinks,aeo
sageo run https://yoursite.com --only crawl,audit,gsc,psi
sageo run https://yoursite.com --resume             # pick up from last successful stage
sageo run https://yoursite.com --dry-run            # estimate only, no paid calls
```

### HTML report

The primary report output is a single self-contained HTML file (inlined CSS,
minimal vanilla JS, no external resources — works offline). For a PDF, open
the file in any modern browser and press **Cmd+P** (macOS) or **Ctrl+P**
(Linux/Windows) and choose *Save as PDF*.

```bash
sageo report html
sageo report html --output ./client-report.html --logo ./logo.png --brand-color '#1E40AF'
sageo report html --appendix
sageo report html --open            # open in default browser after generation
```

> `sageo report pdf` is kept as a deprecated alias that prints a warning and
> produces the same HTML output.

### GEO (Generative Engine Optimization)

```bash
# How often a domain appears in AI responses for a keyword
sageo geo mentions --keyword "seo tools" --domain example.com --dry-run
sageo geo mentions --keyword "seo tools" --domain example.com

# Which pages are most cited by AI engines
sageo geo top-pages --keyword "seo tools"
```

### Opportunities

```bash
sageo opportunities
sageo opportunities --with-serp --serp-queries 10 --dry-run
sageo opportunities --with-serp --serp-queries 10
```

### Auth / Config

```bash
sageo auth login gsc
sageo auth status
sageo auth logout gsc

sageo config show
sageo config get serp_provider
sageo config set serp_provider dataforseo
sageo config set psi_api_key YOUR_KEY
sageo config path
```

## Environment variables

| Variable | Purpose |
|---|---|
| `SAGEO_CONFIG` | Override config file path |
| `SAGEO_PROVIDER` | Override active provider |
| `SAGEO_API_KEY` | Provider API key override |
| `SAGEO_BASE_URL` | Provider base URL override |
| `SAGEO_ORGANIZATION_ID` | Provider organization/account override |
| `SAGEO_SERP_PROVIDER` | `dataforseo` or `serpapi` |
| `SAGEO_SERP_API_KEY` | SerpAPI key |
| `SAGEO_DATAFORSEO_LOGIN` | DataForSEO login (email) |
| `SAGEO_DATAFORSEO_PASSWORD` | DataForSEO API password |
| `SAGEO_APPROVAL_THRESHOLD_USD` | Cost approval threshold |
| `SAGEO_GSC_PROPERTY` | Active GSC property |
| `SAGEO_GSC_CLIENT_ID` | GSC OAuth client ID |
| `SAGEO_GSC_CLIENT_SECRET` | GSC OAuth client secret |
| `SAGEO_PSI_API_KEY` | Google PageSpeed Insights API key (optional, for higher rate limits) |
| `SAGEO_LLM_PROVIDER` | LLM provider for `recommendations draft` — `anthropic` (default) or `openai` |
| `SAGEO_ANTHROPIC_API_KEY` | Anthropic API key (used by `recommendations draft`) |
| `SAGEO_OPENAI_API_KEY` | OpenAI API key (used by `recommendations draft`) |
| `SAGEO_LIVE_TESTS` | Set to `1` to opt in to integration tests that hit paid APIs |

## Data Tiers

| Tier | Cost | What you get |
|------|------|-------------|
| Free | $0 | Crawl, audit, PageSpeed Insights, project state |
| Free + OAuth | $0 | Google Search Console data |
| Paid | ~$0.001-0.02/call | SERP features, Labs keywords, AEO/GEO |
| Paid + commitment | $100/mo deposit | Backlinks API |

## Development

```bash
make fmt && make vet && make test && make lint
make precommit
```

See [`ARCHITECTURE.md`](./ARCHITECTURE.md) for package structure and data flow.

---

## Why Sageo?

- **One tool, not five** — crawl + audit + GSC + SERP + backlinks in a single binary
- **Free where it counts** — crawling, auditing, PageSpeed, and GSC cost nothing
- **Cost-aware** — every paid call shows a cost estimate before executing, with `--dry-run` support
- **Agent-native** — JSON output on every command, designed for AI agents and automation
- **Cross-source analysis** — finds issues that only appear when you compare crawl data against GSC, SERP features, keyword difficulty, and backlinks together
- **70% cheaper SERP** — batch mode uses DataForSEO Standard Queue at $0.0006/query vs $0.002 live

---

**SEO + AEO + GEO + Backlinks. Open source. Single Go binary.**
