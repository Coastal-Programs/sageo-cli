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

SEO, AEO, and GEO intelligence for AI agents. Single Go binary. JSON output on every command.

Sageo gives AI agents structured access to search and AI visibility data: crawling, auditing, Google Search Console, live SERP results, and AI engine responses, all through one CLI that returns machine-readable JSON. No dashboard. No browser. No monthly subscription.

Crawling and auditing are free. Paid operations (SERP, AEO, GEO) always show a cost estimate before executing and support `--dry-run`.

## Install

```bash
git clone https://github.com/Coastal-Programs/sageo-cli.git
cd sageo-cli
make build
# binary: build/sageo
```

Requires Go 1.26+.

## Setup

```bash
# Interactive credential setup (GSC OAuth, DataForSEO, SerpAPI)
sageo login

# Or set individual keys
sageo config set dataforseo_login your@email.com
sageo config set dataforseo_password YOUR_API_PASSWORD
sageo config set serp_provider dataforseo

# Set a cost approval threshold (paid calls above this require explicit approval)
sageo config set approval_threshold_usd 1.00
```

> DataForSEO API password is separate from your account password. Find it at `app.dataforseo.com/api-access`.

## Commands

### Site crawl and audit

```bash
sageo crawl run --url https://example.com --depth 2 --max-pages 50
sageo audit run --url https://example.com --depth 2 --max-pages 50
sageo report generate --url https://example.com
sageo report list
```

### Google Search Console

```bash
sageo gsc sites list
sageo gsc sites use https://example.com/
sageo gsc query pages --start-date 2025-03-01 --end-date 2025-03-28
sageo gsc query keywords --limit 50
sageo gsc opportunities
```

### SERP

```bash
sageo serp analyze --query "seo tools" --dry-run
sageo serp analyze --query "seo tools"
sageo serp compare --query "seo tools" --query "seo software"
```

### Opportunities

```bash
sageo opportunities
sageo opportunities --with-serp --serp-queries 10 --dry-run
sageo opportunities --with-serp --serp-queries 10
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

### Auth

```bash
sageo auth login gsc
sageo auth status
sageo auth logout gsc
```

### Config

```bash
sageo config show
sageo config get serp_provider
sageo config set serp_provider dataforseo
sageo config path
```

## Output

Every command returns a JSON envelope:

```json
{
  "success": true,
  "data": {},
  "metadata": {}
}
```

Errors include a machine-readable code:

```json
{
  "success": false,
  "error": {
    "code": "SERP_FAILED",
    "message": "SERP analysis failed",
    "detail": "..."
  }
}
```

Paid commands include cost metadata:

```json
{
  "metadata": {
    "estimated_cost": 0.002,
    "currency": "USD",
    "requires_approval": false,
    "cached": false,
    "source": "dataforseo",
    "fetched_at": "2026-04-05T10:00:00Z"
  }
}
```

## Environment variables

| Variable | Purpose |
|---|---|
| `SAGEO_CONFIG` | Override config file path |
| `SAGEO_PROVIDER` | Override active provider |
| `SAGEO_SERP_PROVIDER` | `dataforseo` or `serpapi` |
| `SAGEO_DATAFORSEO_LOGIN` | DataForSEO login (email) |
| `SAGEO_DATAFORSEO_PASSWORD` | DataForSEO API password |
| `SAGEO_SERP_API_KEY` | SerpAPI key |
| `SAGEO_APPROVAL_THRESHOLD_USD` | Cost approval threshold |
| `SAGEO_GSC_PROPERTY` | Active GSC property |
| `SAGEO_GSC_CLIENT_ID` | GSC OAuth client ID |
| `SAGEO_GSC_CLIENT_SECRET` | GSC OAuth client secret |

## Development

```bash
make fmt && make vet && make test && make lint
make precommit
```

See [`ARCHITECTURE.md`](./ARCHITECTURE.md) for package structure and data flow.

---

**SEO + AEO + GEO. Built for AI agents. Open source. Single Go binary.**
