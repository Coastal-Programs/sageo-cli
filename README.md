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
    <img src="https://github.com/Coastal-Programs/sageo-cli/actions/workflows/ci.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/Coastal-Programs/sageo-cli/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
  </a>
</p>
</div>

SEO, AEO, and GEO evidence for AI agents. One binary. No subscriptions.

There are plenty of tools that crawl your site and check for SEO issues. There are also a growing number of platforms that track how your content shows up in AI-generated answers from ChatGPT, Perplexity, and Google AI Overviews. But almost all of them are either paid SaaS dashboards, browser-based apps, or Node.js tools with hundreds of dependencies. And almost none of them combine traditional SEO with the newer AEO and GEO categories in one place.

Sageo CLI is different in two ways. First, it combines site crawling, technical SEO audits, Google Search Console data, live SERP checking, and AI visibility tracking into a single command-line tool. You don't need separate products for each of those. Second, it's built specifically for AI agents to call. Every command returns structured JSON. An agent like Cursor, Claude Code, or Copilot can invoke Sageo, read the output, and decide what to do next without any human in the loop.

It's a single Go binary. Open source. Crawling and auditing your site is completely free. When you start pulling Search Console data or checking live SERPs, you're using external APIs that have usage costs (pay per call, not a monthly subscription). The difference is you only pay for what you actually use, and the tool always shows you the estimated cost before making a single call. It doesn't do keyword research in the way Ahrefs or SEMrush does (search volume estimates, difficulty scores). What it does is pull your real keyword performance from Search Console, check who actually ranks for those terms, spot where you're underperforming, and flag where your content is or isn't showing up in AI answers.

**What it does:**
- **Crawls your site** page by page with configurable depth and concurrency
- **Audits every page** for technical SEO issues and scores them by severity
- **Pulls real keyword data** from Google Search Console (clicks, impressions, CTR, position)
- **Spots opportunities** where you have impressions but low CTR, or decent ranking but room to move up
- **Checks live search results** via a SERP provider to see who ranks and what features appear
- **Tracks AI visibility** to see how your content appears in ChatGPT, Perplexity, and AI Overviews (AEO/GEO)
- **Shows costs before spending** with `--dry-run`, cost estimates, and approval gates on any paid lookup
- **Returns structured JSON** on every command so agents and scripts can parse it without guessing

## Quick Start

### Build

```bash
git clone https://github.com/Coastal-Programs/sageo-cli.git
cd sageo-cli
make build
# Binary is at build/sageo
```

Requires Go 1.26+.

### Crawl a Website

```bash
sageo crawl run --url https://example.com --depth 2 --max-pages 50
```

### Run an SEO Audit

```bash
sageo audit run --url https://example.com --depth 2 --max-pages 50
```

### Generate a Report

```bash
sageo report generate --url https://example.com --depth 2
```

## Commands

### Crawl

```bash
# BFS crawl with depth and page limits
sageo crawl run --url https://example.com --depth 3 --max-pages 100
```

### Audit

```bash
# Full SEO audit (crawl + rule checks + scoring)
sageo audit run --url https://example.com --depth 2 --max-pages 50
```

### Report

```bash
# Generate and store a JSON report
sageo report generate --url https://example.com --depth 2

# List stored reports
sageo report list
```

### Provider

```bash
# List available providers
sageo provider list

# Switch active provider
sageo provider use local
```

### Auth

```bash
# Log in to Google Search Console
sageo auth login gsc

# Check auth status
sageo auth status

# Log out
sageo auth logout gsc
```

### Google Search Console

```bash
# List accessible GSC properties
sageo gsc sites list

# Set active property
sageo gsc sites use https://example.com/

# Query page-level performance
sageo gsc query pages --start-date 2025-03-01 --end-date 2025-03-28

# Query keyword-level performance
sageo gsc query keywords --limit 50

# Find opportunity signals from GSC data
sageo gsc opportunities
```

### SERP Analysis

```bash
# Analyze SERP for a query (paid — shows cost estimate)
sageo serp analyze --query "seo tools" --dry-run

# Execute the analysis
sageo serp analyze --query "seo tools"

# Compare multiple queries
sageo serp compare --query "seo tools" --query "seo software" --dry-run
```

### Opportunities

```bash
# Find SEO opportunities from GSC data
sageo opportunities

# Enrich with live SERP data (paid — supports --dry-run)
sageo opportunities --with-serp --serp-queries 10 --dry-run

# Execute with SERP enrichment
sageo opportunities --with-serp --serp-queries 10
```

### Configuration

```bash
# Show config file path
sageo config path

# Show all config values
sageo config show

# Get / set individual values
sageo config get active_provider
sageo config set active_provider local

# Configure SERP provider
sageo config set serp_api_key YOUR_SERPAPI_KEY

# Configure GSC OAuth credentials
sageo config set gsc_client_id YOUR_CLIENT_ID
sageo config set gsc_client_secret YOUR_CLIENT_SECRET

# Set cost approval threshold (blocks paid calls above this amount)
sageo config set approval_threshold_usd 1.00
```

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output format: `json` (default), `text`, `table` |
| `--verbose` | `-v` | Enable verbose output |

## Output Contract

All commands return a structured JSON envelope:

```json
{
  "success": true,
  "data": {},
  "metadata": {}
}
```

Error responses include a machine-readable `code`:

```json
{
  "success": false,
  "error": {
    "code": "CRAWL_FAILED",
    "message": "crawl failed",
    "detail": "..."
  }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `INVALID_URL` | Missing or malformed URL input |
| `CONFIG_LOAD_FAILED` | Could not load configuration |
| `CONFIG_SAVE_FAILED` | Could not save configuration |
| `CONFIG_GET_FAILED` | Unknown config key |
| `PROVIDER_NOT_FOUND` | Unknown or invalid provider name |
| `CRAWL_FAILED` | Crawl operation failed |
| `AUDIT_FAILED` | Audit operation failed |
| `REPORT_WRITE_FAILED` | Could not write report to disk |
| `REPORT_LIST_FAILED` | Could not list stored reports |
| `FETCH_TIMEOUT` | HTTP fetch timed out |
| `CANCELLED` | Operation was cancelled |
| `AUTH_REQUIRED` | Authentication needed for this service |
| `AUTH_FAILED` | Authentication attempt failed |
| `APPROVAL_REQUIRED` | Cost exceeds approval threshold |
| `ESTIMATE_FAILED` | Could not estimate execution cost |
| `SERP_FAILED` | SERP provider request failed |
| `GSC_FAILED` | Google Search Console request failed |

## SEO Checks

The audit engine checks each crawled page for:

| Rule | Severity | Description |
|------|----------|-------------|
| `title-missing` | error | Page has no title tag |
| `title-too-long` | warning | Title exceeds 60 characters |
| `meta-description-missing` | warning | No meta description |
| `meta-description-too-long` | warning | Meta description exceeds 160 characters |
| `h1-missing` | error | No H1 heading |
| `h1-multiple` | warning | Multiple H1 headings |
| `img-alt-missing` | warning | Image without alt text |
| `canonical-missing` | info | No canonical link tag |
| `broken-page` | error/warning | HTTP 4xx/5xx status |

## Configuration

Default config path: `~/.config/sageo/config.json`

Override with env var: `SAGEO_CONFIG=/absolute/path/to/config.json`

| Env Variable | Purpose |
|---|---|
| `SAGEO_PROVIDER` | Override active provider |
| `SAGEO_API_KEY` | API key for paid providers |
| `SAGEO_BASE_URL` | Custom base URL |
| `SAGEO_ORGANIZATION_ID` | Organization identifier |
| `SAGEO_SERP_PROVIDER` | Override SERP provider (default: `serpapi`) |
| `SAGEO_SERP_API_KEY` | SerpAPI key |
| `SAGEO_APPROVAL_THRESHOLD_USD` | Cost approval threshold |
| `SAGEO_GSC_PROPERTY` | Active GSC property |
| `SAGEO_GSC_CLIENT_ID` | GSC OAuth client ID |
| `SAGEO_GSC_CLIENT_SECRET` | GSC OAuth client secret |

## Architecture

```
sageo-cli/
├── cmd/sageo/
│   └── main.go                  # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go              # Cobra root command + global flags
│   │   └── commands/
│   │       ├── crawl.go         # crawl run
│   │       ├── audit.go         # audit run
│   │       ├── report.go        # report generate, list
│   │       ├── provider.go      # provider list, use
│   │       ├── config.go        # config path, show, get, set
│   │       ├── auth.go          # auth login, status, logout
│   │       ├── gsc.go           # gsc sites, query, opportunities
│   │       ├── serp.go          # serp analyze, compare
│   │       ├── opportunities.go # opportunities (merge GSC + SERP)
│   │       └── version.go       # version
│   ├── crawl/                   # BFS crawler engine
│   ├── audit/                   # SEO rule checks + scoring
│   ├── report/                  # JSON report generation + storage
│   ├── provider/                # HTTP fetcher abstraction + registry
│   ├── auth/                    # Token store for OAuth services
│   ├── gsc/                     # Google Search Console client
│   ├── serp/                    # SERP provider abstraction
│   │   └── serpapi/             # SerpAPI adapter
│   ├── opportunities/           # Opportunity detection + merge logic
│   └── common/
│       ├── config/              # Config loading (env + JSON file)
│       ├── cost/                # Cost estimation + approval gates
│       └── cache/               # File-based response caching
├── pkg/
│   └── output/                  # JSON envelope, text, table formatting
├── go.mod
├── Makefile
└── README.md
```

**Dependencies:**

| Dependency | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/spf13/pflag` | Flag parsing (indirect, via cobra) |
| `golang.org/x/net` | HTML parsing |
| Go standard library | Everything else |

## Development

```bash
# Build
make build

# Format, vet, test, lint
make fmt
make vet
make test
make lint

# All checks at once
make precommit

# Cross-compile for release
make release

# Clean build artifacts
make clean
```

### Pre-commit Hooks

```bash
# Install pre-commit (macOS)
brew install pre-commit

# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.11.3

# Activate hooks
pre-commit install

# Run manually
pre-commit run --all-files
```

## Why this exists

Most SEO tools are built for humans clicking around dashboards. Most AEO/GEO tools are new SaaS platforms charging monthly subscriptions for AI visibility tracking. Neither category is designed for an AI agent to call programmatically, get structured data back, and act on it.

Sageo CLI is built for the way work is actually shifting. Your AI coding agent should be able to check your site's search health, see what keywords are performing, check the live SERPs, and understand your AI visibility, all through one tool that speaks JSON. No browser, no login, no dashboard.

The rule is: free and local first. Crawl and audit your site for nothing. Pull your Search Console data for nothing. Only hit paid SERP providers when you've narrowed down to the specific keywords worth checking. And even then, the tool tells you the cost upfront and waits for approval.

This isn't trying to replace Ahrefs or SEMrush. Those are massive platforms with years of backlink data, keyword databases, and features that would be pointless to replicate. Sageo is the tool that sits between your site and your AI agent, collecting the evidence that matters and giving it back in a format the agent can actually use.

## Release

```bash
make release
```

See [`ARCHITECTURE.md`](./ARCHITECTURE.md) for package structure and data flow.

---

**SEO + AEO + GEO in one CLI. Built for AI agents. Open source. Single Go binary.**
