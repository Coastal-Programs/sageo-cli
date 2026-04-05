# Sageo CLI — Architecture

## Overview

Sageo CLI is a Go + Cobra single-binary command-line tool for SEO crawling, auditing, and reporting. It uses a provider abstraction for HTTP fetching, a BFS crawler for page discovery, a rule-based audit engine, and JSON report storage.

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
- `local/local.go`: Built-in `net/http` fetcher with configurable timeout and User-Agent.

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
- `checker.go`: Individual check functions for title, meta description, H1, image alt, canonical, and status code.
- `engine.go`: Runs all checkers across crawl results and computes a 0–100 score.

### `internal/report`
Report generation and storage:
- `service.go`: `Service` interface (with `Generate` and `List` methods), `Request`, `Result`, `ReportMeta` types.
- `generator.go`: Writes JSON reports to `~/.config/sageo/reports/` and reads stored report metadata.

### `internal/common/config`
Config management:
- Path resolution (`SAGEO_CONFIG` override + XDG-style fallback)
- `Load` and `Save`
- Env override hooks (`SAGEO_PROVIDER`, `SAGEO_API_KEY`, `SAGEO_BASE_URL`, `SAGEO_ORGANIZATION_ID`)
- Secret redaction for safe display

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

Default file: `~/.config/sageo/config.json`
Override: `SAGEO_CONFIG` (must be absolute `.json` path)

## Build and Release

- `Makefile` for build/test/lint/release workflows
- `scripts/release.sh` cross-compiles macOS, Linux, and Windows artifacts
