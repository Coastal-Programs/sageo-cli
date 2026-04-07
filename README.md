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

Sageo replaces the need to juggle multiple SEO tools. It crawls your site, runs a technical SEO audit, pulls Google Search Console data, checks PageSpeed (Core Web Vitals), analyzes SERP features (AI Overviews, Featured Snippets, People Also Ask), gets keyword difficulty and search intent, audits your backlink profile, and merges everything into prioritised action items — all from one CLI.

Crawling, auditing, and PageSpeed Insights are free. Paid features (SERP, keyword research, backlinks, AEO/GEO) use DataForSEO with cost estimates and `--dry-run` on every command.

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

# 6. Check your status
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
# Query an AI model directly
sageo aeo responses --prompt "What is Sageo CLI?" --model chatgpt --dry-run
sageo aeo responses --prompt "What is Sageo CLI?" --model claude

# AI search volume for a keyword
sageo aeo keywords --keyword "seo tools" --location "United States"
```

Supported models: `chatgpt`, `claude`, `gemini`, `perplexity`

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
