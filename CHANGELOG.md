# Changelog

All notable changes to this project will be documented in this file.

## [0.3.0] - 2026-04-05

### Added
- **Google Search Console integration**: `gsc sites list`, `gsc sites use`, `gsc query pages`, `gsc query keywords`, `gsc opportunities` commands for real search performance data.
- **OAuth2 authentication**: `auth login gsc`, `auth status`, `auth logout gsc` with local callback server and file-based token persistence.
- **SerpAPI SERP analysis**: `serp analyze` and `serp compare` commands with `--dry-run` support and cost estimation.
- **Opportunity detection**: `opportunities` command merges GSC seeds with optional SERP enrichment, classifying by type, confidence, impact, and effort.
- **Cost-aware execution**: `internal/common/cost` package with `BuildEstimate` and `EvaluateApproval` for approval gate enforcement.
- **File-based caching**: `internal/common/cache` package with TTL-based expiry, used by SERP and opportunities commands.
- **New error codes**: `AUTH_REQUIRED`, `AUTH_FAILED`, `APPROVAL_REQUIRED`, `ESTIMATE_FAILED`, `SERP_FAILED`, `GSC_FAILED`.
- **New config keys**: `serp_provider`, `serp_api_key`, `approval_threshold_usd`, `gsc_property`, `gsc_client_id`, `gsc_client_secret` with env overrides and redaction.
- Tests for cost estimation, approval gates, cache behavior, auth token store, SerpAPI adapter, and opportunity merge logic.

### Changed
- Updated `README.md` with auth, GSC, SERP, and opportunities command documentation.
- Updated `ARCHITECTURE.md` with Phase 3 package responsibilities and cost-aware execution model.
- Updated `CLAUDE.md` to reflect Phase 3 completion.

## [0.2.2] - 2026-04-05

### Added
- `PHASE3_PLAN.md` defining the next stage as agent-first SEO/AEO/GEO intelligence, with Google Search Console, one SERP provider, cost-aware execution, and dry-run planning.

### Changed
- Updated `README.md` to clearly explain what the CLI does today versus what Phase 3 will add.
- Updated `ARCHITECTURE.md` with the Phase 3 free-first / paid-second model and planned cost metadata.
- Updated `CLAUDE.md` with Phase 3 guidance for external evidence providers and cost-aware command design.

## [0.2.1] - 2026-04-05

### Changed
- Error responses now include a machine-readable `code` field (`INVALID_URL`, `CONFIG_LOAD_FAILED`, `CONFIG_SAVE_FAILED`, `CONFIG_GET_FAILED`, `PROVIDER_NOT_FOUND`, `CRAWL_FAILED`, `AUDIT_FAILED`, `REPORT_WRITE_FAILED`, `REPORT_LIST_FAILED`, `FETCH_TIMEOUT`, `CANCELLED`).
- All command error paths use `PrintCodedError` with appropriate error codes.
- Context timeout and cancellation errors are normalized in the local fetcher and crawler.
- Provider listing output is sorted deterministically.
- Removed `"scaffold": true` metadata from config and version commands.
- Updated root command description to remove scaffold language.
- Added contract tests verifying JSON envelope shape for success and error responses.

## [0.2.0] - 2026-04-05

### Added
- **Provider abstraction**: `Fetcher` interface with registry pattern in `internal/provider`. Built-in `local` provider using `net/http`.
- **Crawl service**: BFS crawler with depth limit, max-pages cap, same-domain scoping, and concurrent fetching (5 workers). HTML parsing via `golang.org/x/net/html` extracts title, meta description, canonical, headings, links, and images.
- **Audit engine**: Rule-based SEO checker covering title, meta description, H1, image alt text, canonical tag, and HTTP status codes. Produces per-page issues with severity levels and a 0–100 aggregate score.
- **Report generator**: Writes JSON audit reports to `~/.config/sageo/reports/`. `report list` reads stored report metadata.
- **Provider commands**: `provider list` shows available providers with active marker. `provider use <name>` validates and sets active provider.
- **New dependency**: `golang.org/x/net` for robust HTML tokenization.
- Tests for provider, crawl, audit, and report packages.

### Changed
- `crawl run` now performs real website crawling with `--url`, `--depth`, and `--max-pages` flags.
- `audit run` now runs crawl → audit pipeline with `--url`, `--depth`, and `--max-pages` flags.
- `report generate` now runs full crawl → audit → report pipeline.
- `report list` now reads stored reports from disk.
- `provider list` and `provider use` are now functional.
- Removed `crawl status` and `audit status` subcommands (operations are synchronous).
- Removed `newScaffoldCommand` helper (no longer needed).
- Updated documentation to reflect shipped behavior.

## [0.1.0] - 2026-04-05

### Added
- Initial `sageo-cli` scaffold using Go + Cobra single-binary architecture.
- Root command with global `--output` and `--verbose` flags.
- Command groups: `version`, `config`, `crawl`, `audit`, `report`, `provider`.
- Placeholder-only handlers for crawl/audit/report/provider commands.
- Config foundation with path resolution, load/save, env overrides, and redaction.
- Structured output package with envelope-style success/error responses.
- Domain placeholder interfaces under `internal/crawl`, `internal/audit`, and `internal/report`.
- Build/test/release tooling (`Makefile`, `scripts/release.sh`, GitHub workflows).
- Scaffold documentation (`README.md`, `ARCHITECTURE.md`, `CLAUDE.md`).
- Smoke tests for root command, output behavior, and config load/save paths.
