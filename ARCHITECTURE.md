# Sageo CLI — Architecture

## Overview

Sageo CLI is a Go + Cobra single-binary command-line tool for SEO, AEO, and GEO operations. It provides website crawling, rule-based SEO auditing, JSON report generation, Google Search Console integration, SERP analysis via SerpAPI or DataForSEO, and opportunity detection that merges signals from multiple sources.

## Data Flow

```
User Command → Provider (HTTP fetch) → Crawler (BFS) → Audit Engine → Report Generator → JSON file
```

The `crawl run` command stops after crawling. The `audit run` command runs crawl → audit. The `report generate` command runs crawl → audit → report (full pipeline).

## Command Hierarchy

- `version` — build/runtime metadata
- `config` — local config management (`show`, `get`, `set`, `path`)
- `crawl` — website crawling (`run`)
- `audit` — SEO audit (`run`)
- `report` — report generation and listing (`generate`, `list`)
- `provider` — provider management (`list`, `use`)
- `auth` — service authentication (`login`, `status`, `logout`)
- `gsc` — Google Search Console (`sites list`, `sites use`, `query pages`, `query keywords`, `opportunities`)
- `serp` — SERP analysis (`analyze`, `compare`) — paid, supports `--dry-run`
- `opportunities` — merged opportunity detection from GSC + optional SERP enrichment
- `aeo` — Answer Engine Optimization (`responses`, `keywords`) — paid, supports `--dry-run`
- `geo` — Generative Engine Optimization (`mentions`, `top-pages`) — paid, supports `--dry-run`
- `labs` — DataForSEO Labs intelligence (`ranked-keywords`, `keywords`, `overview`, `competitors`, `keyword-ideas`) — paid, supports `--dry-run`
- `login` — interactive credential setup
- `logout` — clear stored credentials

Global flags:
- `--output, -o` (`json`, `text`, `table`) default `json`
- `--verbose, -v` boolean flag

## Package Responsibilities

### `cmd/sageo/main.go`
Entrypoint that calls `internal/cli.Execute(version)`.
Version is injected via ldflags in build/release flows.

### `internal/cli`
- `root.go`: root Cobra command, global flags, command registration.
- `commands/*.go`: command constructors that wire services and output results.

### `internal/provider`
Provider abstraction layer:
- `provider.go`: `Fetcher` interface, registry, and `NewFetcher` constructor.
- `local/local.go`: built-in `net/http` fetcher with configurable timeout and User-Agent.

The registry pattern allows future providers to register via `init()`.

### `internal/crawl`
BFS website crawler:
- `types.go`: `Request`, `Result`, `PageResult`, `Link`, `Image`, `Heading`, `CrawlError` types.
- `service.go`: `Service` interface and `NewService` constructor.
- `crawler.go`: BFS implementation with depth limit, max-pages cap, same-domain scoping, and concurrency control (5 workers).
- `page.go`: HTML parsing using `golang.org/x/net/html` to extract title, meta description, canonical, headings, links, and images.

### `internal/audit`
SEO audit engine:
- `types.go`: `Severity`, `Issue`, `Request`, `Result` types.
- `service.go`: `Service` interface and `NewService` constructor.
- `checker.go`: individual check functions for title, meta description, H1, image alt, canonical, and status code.
- `engine.go`: runs all checkers across crawl results and computes a 0–100 score.

### `internal/report`
Report generation and storage:
- `service.go`: `Service` interface (with `Generate` and `List` methods), `Request`, `Result`, `ReportMeta` types.
- `generator.go`: writes JSON reports to `~/.config/sageo/reports/` and reads stored report metadata.

### `internal/auth`
OAuth token store:
- `FileTokenStore`: persists tokens to `~/.config/sageo/auth/<service>.json`
- `Save`, `Load`, `Delete`, `Status` operations
- Expiry checking based on stored `expires_at`

### `internal/gsc`
Google Search Console client:
- `Client`: authenticated API client for GSC
- `ListSites`: list accessible properties
- `QueryPages`, `QueryKeywords`: Search Analytics queries by dimension
- `QueryOpportunities`: filtered query+page pairs for opportunity detection
- `HTTPClient` interface for testability

### `internal/serp`
SERP provider abstraction:
- `Provider` interface: `Name()`, `Estimate()`, `Analyze()`
- `AnalyzeRequest`, `AnalyzeResponse`, `OrganicResult` types

### `internal/serp/serpapi`
SerpAPI adapter:
- Implements `serp.Provider`
- Cost estimation at $0.01/search
- JSON response parsing with domain extraction
- `WithBaseURL` and `WithHTTPClient` options for testing

### `internal/dataforseo`
Shared DataForSEO HTTP client:
- Basic Auth credential handling
- Configurable base URL and HTTP client
- Reused by AEO, GEO, Labs, and DataForSEO-backed SERP commands

### `internal/serp/dataforseo`
DataForSEO SERP adapter:
- Implements `serp.Provider`
- Uses DataForSEO organic search endpoint
- Provider-selected in CLI via `serp_provider = "dataforseo"`

### `internal/cli/commands/labs.go`
DataForSEO Labs command implementations:
- `ranked-keywords` — keywords a domain/URL ranks for, with position, volume, and difficulty
- `keywords` — keyword ideas relevant to a domain
- `overview` — domain ranking distribution and estimated traffic
- `competitors` — competing domains by ranking overlap
- `keyword-ideas` — keyword ideas from a seed keyword
- All subcommands: credential check, $0.01/task cost estimate, approval gate, `--dry-run`

### `internal/opportunities`
Opportunity detection and merge logic:
- `Merge`: combines GSC seeds with optional SERP evidence
- Classifies opportunities by type (`page`, `keyword`, `answer`)
- Scores by confidence, impact estimate, and effort estimate
- SERP enrichment adds position validation and answer box detection

### `internal/common/config`
Config management:
- Path resolution (`SAGEO_CONFIG` override + XDG-style fallback)
- `Load` and `Save`
- Env override hooks for all keys including GSC, SERP, DataForSEO, and approval settings
- Secret redaction for safe display (API keys, client secrets, passwords)

### `internal/common/cost`
Cost estimation and approval gates:
- `BuildEstimate`: computes cost from unit pricing
- `EvaluateApproval`: blocks execution when estimated cost exceeds threshold
- Used by paid commands (`serp`, `opportunities` when enriched, `aeo`, `geo`, `labs`)

### `internal/common/cache`
File-based response caching:
- `FileStore`: persists cached responses to `~/.config/sageo/cache/<provider>/<hash>.json`
- TTL-based expiry
- `Metadata` type for output envelope integration

### `pkg/output`
Shared envelope and renderer:
- JSON-first for machine consumption
- Optional text/table rendering modes
- Success and error helpers for consistent command responses
- Machine-readable error codes (`errors.go`) for programmatic error classification
- `PrintCodedError` attaches an error code to the envelope; `PrintErrorResponse` delegates with an empty code

## Config Model

Keys:
- `active_provider` — which provider to use for HTTP fetching (default: `local`)
- `api_key` (redacted on read/show)
- `base_url`
- `organization_id`
- `serp_provider` — SERP data provider (default: `serpapi`)
- `serp_api_key` (redacted on read/show)
- `dataforseo_login` (redacted on read/show)
- `dataforseo_password` (redacted on read/show)
- `approval_threshold_usd` — cost gate threshold; 0 means no gate
- `gsc_property` — active GSC property URL
- `gsc_client_id` (redacted on read/show)
- `gsc_client_secret` (redacted on read/show)

Default file: `~/.config/sageo/config.json`
Override: `SAGEO_CONFIG` (must be absolute `.json` path)

## Cost-Aware Execution Model

### Free-first, paid-second
Recommended execution order:
1. Local crawl/audit (free)
2. Google Search Console data (free, requires OAuth)
3. Paid lookups only for narrowed, high-value checks (SERP/AEO/GEO/Labs)

### Cost metadata
Paid commands expose machine-readable metadata:
- `estimated_cost` — computed cost estimate before execution
- `currency` — always `USD`
- `requires_approval` — true when estimate exceeds `approval_threshold_usd`
- `cached` — whether the response came from cache
- `source` — which provider produced the data
- `fetched_at` — RFC3339 timestamp of data retrieval
- `dry_run` — true when `--dry-run` flag was used

### Approval gate
When `approval_threshold_usd` is set (> 0), any paid command whose estimated cost exceeds the threshold returns `APPROVAL_REQUIRED` without executing.

### Caching
Paid responses are cached to `~/.config/sageo/cache/` with configurable TTL. Cache hits are reflected in output metadata and avoid repeat charges.

### Why this matters
`sageo-cli` is designed for AI agents. The CLI collects, normalizes, and prices evidence. The external AI agent decides what to do next.

## Build and Release

- `Makefile` for build/test/lint/release workflows
- `scripts/release.sh` cross-compiles macOS, Linux, and Windows artifacts
