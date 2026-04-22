# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- **Evidence-backed AI-citation ChangeTypes** in `internal/state/recommendations.go`: `ChangeTLDR`, `ChangeListFormat`, `ChangeAuthorByline`, `ChangeFreshness`, `ChangeEntityConsistency`. Each type carries a doc comment citing the supporting section of `docs/research/ai-citation-signals-2026.md`.
- **New merge rule `missing-author-signals`** — fires on any URL whose crawl findings include `missing-author-byline`; emits `ChangeAuthorByline` + a Person-schema recommendation. Priority scored 45-70 depending on GSC impressions.
- **Drafter prompts for the new ChangeTypes** (`internal/recommendations/drafter.go`) with research-grounded constraints: `ChangeTLDR` enforces a 40-70 word budget and target query in the first sentence; `ChangeListFormat` enforces 3-7 dash-prefixed items; `ChangeAuthorByline`, `ChangeFreshness`, and `ChangeEntityConsistency` produce short plain-text outputs validated by new `listValidator` / `lineCountValidator` helpers.
- **Forecaster handling** for the new AI-citation ChangeTypes — modelled as CTR-only uplifts (target position = current) in `internal/forecast/forecast.go`.

### Changed
- **Report output swapped from PDF to self-contained HTML** — new `sageo report html` command (`internal/report/html`) replaces the previous `gofpdf`-based PDF generator. HTML is easier to build, iterate on, and style; supports interactivity (collapsible evidence sections, sortable forecast table); and is responsive across devices. Users produce a PDF for free via browser print-to-PDF (Cmd+P → Save as PDF). Output is a single `.html` file with inlined CSS, minimal vanilla JS, and no external resource references — works offline. Includes a dedicated `@media print` stylesheet that expands all `<details>`, forces clean A4 page breaks, and preserves readability in monochrome. `internal/report/pdf` and the `codeberg.org/go-pdf/fpdf` dependency have been removed. `sageo report pdf` is kept as a deprecated alias that prints a warning and routes to the HTML renderer so existing scripts keep working.
- **Merge rule → ChangeType mappings rewritten against research evidence** (`internal/merge/recommendations.go`):
  - `ai-overview-eating-clicks` now emits `ChangeTLDR` + `ChangeListFormat` + `ChangeSchema(FAQPage)` + `ChangeAuthorByline` + H2s per PAA (was: `ChangeBody` + `ChangeSchema` + H2s). Rationale cites Growth Memo's 44.2% first-30% finding and the signals matrix "likely" rows for list/table formatting and author bylines.
  - `featured-snippet-opportunity` now emits `ChangeTLDR` (40-60 word definition block) rather than a generic `ChangeBody` rewrite — same passage serves both Featured Snippets and AI Overview / ChatGPT citation.
  - `schema-not-showing` rationale updated to scope to Tier-1 types (Organization, Article, BreadcrumbList, Product, LocalBusiness); FAQPage reserved for the AI-Overview rule where Google's pipeline specifically indexes Q&A markup; HowTo dropped (removed from Google rich results Aug 2023).
- Each rule generator carries a comment explaining *why* it emits its specific ChangeTypes, with pointers back to `docs/research/ai-citation-signals-2026.md`.

## [0.6.0] - 2026-04-22

### Added
- **Autonomous pipeline** — new `sageo run <url>` command (`internal/pipeline`, `internal/cli/commands/run.go`) that drives crawl → audit → GSC → PSI → SERP → Labs → backlinks → AEO → merge → recommendations → draft → forecast in a single invocation. Flags: `--budget`, `--skip`, `--only`, `--max-pages`, `--prompts`, `--dry-run`, `--approve`, `--resume`. State persisted between stages so `--resume` picks up at the last successful stage.
- **Multi-model AEO fan-out** — `sageo aeo responses` now supports `--all`, `--models`, `--tier`, `--engine`, `--model-name`, and `--concurrency` for parallel queries across ChatGPT, Claude, Gemini, Perplexity. Output shape: `data.results[]` with summed `estimated_cost`. Responses (and per-response brand mentions) upserted to `state.AEO.Responses` by prompt.
- **AI model catalogue** — new `sageo aeo models` command backed by `internal/aeo/catalogue.go`. Loads the live DataForSEO AI optimization model list with 7-day disk cache.
- **Brand mention detection** — new `sageo aeo mentions` command group with `scan` (Layer A: offline scan of stored AEO responses, free), `search`, `top-pages`, `top-domains` (Layer B: DataForSEO LLM Mentions API). Packages: `internal/aeo/mentions`, `internal/aeo/llmmentions`.
- **Brand terms in state** — `sageo init --brand "Name,alias"` persists project-level brand terms to `state.BrandTerms`; consumed by the mentions scanner.
- **Recommendation engine** — merge rules now emit concrete `state.Recommendation` objects (one per `ChangeType`: `title`, `meta_description`, `h1`, `h2_add`, `schema_add`, `body_expand`, `internal_link_add`, `speed_fix`, `backlink_outreach`, `indexability_fix`). Stable hashed IDs via `recommendations.ID`, upserted via `recommendations.UpsertRecommendations`. Package: `internal/recommendations`. Type aliases avoid cycles with `internal/state`.
- **`sageo recommendations list`** — list stored recommendations sorted by priority, with `--url`, `--type`, `--top`, `--format` filters.
- **LLM copy drafter** — new `sageo recommendations draft` command fills empty `RecommendedValue` fields with LLM-drafted copy (titles, meta descriptions, H1s, H2s, body paragraphs, JSON-LD schema), validated against SERP length limits with retry. Flags: `--provider anthropic|openai`, `--url`, `--type`, `--limit`, `--dry-run`. Packages: `internal/llm`, `internal/llm/anthropic` (Anthropic Messages API, `claude-sonnet-4-6`), `internal/llm/openai` (OpenAI Chat Completions, `gpt-5`), `internal/llm/providers` (side-effect registry).
- **Click-lift forecaster** — new `sageo recommendations forecast` command and `internal/forecast` package. Uses the Advanced Web Ranking 2024 position→CTR curve (swappable via `forecast.SetCurve`) to attach `state.Forecast { estimated_monthly_clicks_delta, confidence_low, confidence_high, method }` to every stored recommendation. Exposed programmatically via `recommendations.AttachForecasts`.
- **PDF report** — new `sageo report pdf` command (`internal/report/pdf`) renders a styled client-ready PDF: cover, executive summary, per-source "what's broken" sections, recommendation cards with evidence, forecast table, and optional raw-data appendix. Flags: `--output`, `--logo`, `--brand-color`, `--appendix`.
- **Test isolation guardrails** — new `internal/common/testutil` package with `NewFakeDataForSEO`, `NewFakeAnthropic`, `NewFakeOpenAI`, `NewFakePSI` helpers. New `scripts/check-no-live-tests.sh` blocks unit tests from touching the network, wired into `make test` via `make check-tests`.
- **Opt-in integration tests** — new `make test-integration` / `make test-all` targets. Integration tests require both the `integration` build tag and `SAGEO_LIVE_TESTS=1`. Coverage for DataForSEO, PSI, Anthropic, and OpenAI live smoke tests.
- **New env vars**: `SAGEO_LLM_PROVIDER` (default `anthropic`), `SAGEO_ANTHROPIC_API_KEY`, `SAGEO_OPENAI_API_KEY`, `SAGEO_LIVE_TESTS`.

### Changed
- `sageo aeo responses` moved from `--model` (single, required) to an engine/model selector with multiple modes (`--engine`, `--all`, `--models`, `--tier`); single-engine usage remains backward-compatible when `--engine` is supplied explicitly.
- Data flow in `ARCHITECTURE.md` extended to include AEO, recommendations, draft, forecast, and PDF stages.
- `TESTING.md` and `CLAUDE.md` updated to document the unit/integration test split and new validation targets.

### Fixed
- Unit tests no longer hit paid APIs accidentally — guarded by `scripts/check-no-live-tests.sh` and factored through `internal/common/testutil` fakes.

## [0.5.0] - 2026-04-05

### Added
- **Labs command group** (`labs`) for DataForSEO Labs intelligence:
  - `labs ranked-keywords` — keywords a domain/URL ranks for
  - `labs keywords` — keyword ideas relevant to a domain
  - `labs overview` — domain ranking distribution and estimated traffic
  - `labs competitors` — competing domains by ranking overlap
  - `labs keyword-ideas` — keyword ideas from a seed keyword

### Improved
- **Interactive login** (`sageo login`) rewritten with Charm Huh forms — selector flow, masked secret inputs, Esc-to-back navigation, and setup summary on exit

## [0.1.0] - 2026-04-05

### Added
- **Website crawler**: BFS crawler with depth limit, max-pages cap, same-domain scoping, and concurrent fetching. HTML parsing extracts title, meta description, canonical, headings, links, and images.
- **SEO audit engine**: Rule-based checker covering title, meta description, H1, image alt text, canonical tag, and HTTP status codes. Produces per-page issues with severity levels and a 0–100 aggregate score.
- **Report generator**: JSON audit reports stored to `~/.config/sageo/reports/` with metadata listing.
- **Provider abstraction**: `Fetcher` interface with registry pattern. Built-in `local` provider using `net/http` with configurable timeout and User-Agent.
- **Google Search Console integration**: `gsc sites list`, `gsc sites use`, `gsc query pages`, `gsc query keywords`, `gsc opportunities` commands for real search performance data.
- **OAuth2 authentication**: `auth login gsc`, `auth status`, `auth logout gsc` with local callback server and file-based token persistence.
- **SerpAPI SERP analysis**: `serp analyze` and `serp compare` commands with `--dry-run` support and cost estimation.
- **DataForSEO SERP adapter**: Implements `serp.Provider` against DataForSEO's organic search endpoint. Selected via `serp_provider = "dataforseo"` config key.
- **AEO command group** (Answer Engine Optimization):
  - `aeo responses` — query AI models (ChatGPT, Claude, Gemini, Perplexity) and view the response. Supports `--model`, `--dry-run`, cost estimation, and approval gate.
  - `aeo keywords` — retrieve AI search volume data for a keyword from DataForSEO.
- **GEO command group** (Generative Engine Optimization):
  - `geo mentions` — track how often a domain or brand appears in AI-generated responses for a keyword. Supports `--domain`, `--platform`, `--dry-run`, cost estimation, and approval gate.
  - `geo top-pages` — show which pages are most cited by AI engines for a keyword.
- **Opportunity detection**: `opportunities` command merges GSC seeds with optional SERP enrichment, classifying by type, confidence, impact, and effort.
- **Interactive login**: `sageo login` guides setup for Google Search Console (OAuth), DataForSEO (login + password), and SerpAPI (API key) in a single flow. `sageo logout` clears all stored credentials.
- **Cost-aware execution**: Estimation, approval gates, `--dry-run` support, and file-based response caching with TTL for all paid API commands.
- **Structured output**: JSON envelope with `success`, `data`, `error`, `metadata` fields. Machine-readable error codes for programmatic classification.
- **Config management**: JSON config at `~/.config/sageo/config.json` with environment variable overrides and secret redaction.
- **Build tooling**: Makefile for build/test/lint/release workflows. Cross-compilation script for macOS, Linux, and Windows.
