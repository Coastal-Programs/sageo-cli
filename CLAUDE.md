# Sageo CLI — Agent Notes

## Scope

The CLI has working crawl, audit, report (JSON + HTML), provider, auth, GSC, PSI, SERP, AEO (multi-model + mentions), GEO, Labs, backlinks, merge, recommendations (merge → draft → forecast), LLM drafting (Anthropic/OpenAI), click-lift forecasting, autonomous pipeline (`sageo run`), and opportunities services.

### Implemented
- **Autonomous run** — `sageo run <url>` drives the full pipeline end-to-end in a single command: crawl → audit → GSC → PSI → Labs → SERP → backlinks → AEO fan-out → mentions scan → merge → recommendations → LLM draft → forecast. Flags: `--budget`, `--skip`, `--only`, `--max-pages`, `--prompts`, `--dry-run`, `--approve`, `--resume`. Orchestrated by `internal/pipeline`; state persisted between stages so a failure can be resumed with `--resume`.
- BFS website crawler with depth/page limits and concurrent fetching
- SEO audit engine with rule-based checks and scoring
- JSON report generation and storage
- Provider abstraction with built-in `local` HTTP fetcher
- Full crawl → audit → report pipeline
- Google Search Console integration (sites list, query pages/keywords, opportunity seeds)
- OAuth2 authentication flow for GSC with token persistence and auto-refresh
- PageSpeed Insights integration (Core Web Vitals: LCP, CLS, FCP, TBT, SI)
- PSI auto-persistence to state.json with upsert by URL+strategy
- PSI auth cascade: API key → GSC OAuth token → unauthenticated
- SERP analysis adapters (SerpAPI and DataForSEO)
- SERP feature detection (9 types: Featured Snippets, PAA, AI Overviews, Local Pack, Knowledge Graph, Top Stories, Inline Videos, Inline Shopping, Inline Images)
- SERP batch analysis via DataForSEO Standard queue ($0.0006/keyword, up to 100 keywords)
- SERP data persistence to state.json (features, related questions, top domains, our position)
- AEO and GEO command groups backed by DataForSEO
- Brand mention detection — Layer A (offline scan of stored AEO responses for brand terms) and Layer B (DataForSEO LLM Mentions API: search, top-pages, top-domains, aggregated-metrics) via `sageo aeo mentions`
- Project-level brand terms persisted in `state.BrandTerms` (set via `sageo init --brand "Name,alias"`); MentionsData upsert-by-term (including `aeo models` for the live model catalogue, cached on disk for 7 days)
- Multi-model AEO responses fan-out (`sageo aeo responses --all` / `--models` / `--tier`): parallel engine queries bounded by `--concurrency`, per-row error surfacing, summed cost gate, and brand-mention storage in state under `state.AEO.Responses` (upsert by prompt)
- Labs command group (`ranked-keywords`, `keywords`, `overview`, `competitors`, `keyword-ideas`, `bulk-difficulty`)
- Labs keyword data persistence to state.json (difficulty, volume, intent, position)
- Labs bulk keyword difficulty with `--from-gsc` flag to auto-load keywords from state
- Labs competitor data persistence to state.json
- Backlinks API integration (`summary`, `list`, `referring-domains`, `competitors`, `gap`)
- Backlinks gap auto-loads competitors from state.json when `--competitors` not provided
- Backlink data persistence to state.json (summary metrics + gap domains)
- Cost-aware execution contracts (`estimated_cost`, `requires_approval`, `cached`, `source`, `fetched_at`)
- `--dry-run` support for all paid workflows
- File-based response caching with TTL
- Approval gate blocking execution when estimated cost exceeds threshold
- Project state management (`init`, `status`, `analyze`)
- Evidence-backed recommendation ChangeTypes: `ChangeTitle`, `ChangeMeta`, `ChangeH1`, `ChangeH2`, `ChangeSchema`, `ChangeBody`, `ChangeInternalLink`, `ChangeSpeed`, `ChangeBacklink`, `ChangeIndexability`, plus AI-citation levers `ChangeTLDR`, `ChangeListFormat`, `ChangeAuthorByline`, `ChangeFreshness`, `ChangeEntityConsistency` (see `docs/research/ai-citation-signals-2026.md`)
- Cross-source merge engine with 14 rules:
  - Rules 1–5: crawl + GSC rules (ranking-but-not-clicking, not-indexed, issues-on-high-traffic-page, thin-content-ranking-well, schema-not-showing)
  - Rule 6: PSI + GSC (slow-core-web-vitals)
  - Rules 7–9: SERP-aware (ai-overview-eating-clicks, featured-snippet-opportunity, paa-content-opportunity)
  - Rules 10–11: Labs-aware (easy-win-keyword, informational-content-gap)
  - Rules 12–13: Backlinks-aware (weak-backlink-profile, broken-backlinks-found)
  - Rule 14: E-E-A-T (missing-author-signals) — emits ChangeAuthorByline + Person schema
- Priority scoring system (10–100) with automatic sorting by urgency
- `report html` command renders a self-contained, styled HTML file (cover, exec summary, per-source "what's broken", recommendation cards, sortable forecast table, optional appendix). Users print-to-PDF via the browser (Cmd/Ctrl+P). `report pdf` is preserved as a deprecated alias.
- Click-lift forecaster (`internal/forecast`) using the Advanced Web Ranking 2024 position→CTR curve, attached to recommendations via `recommendations.AttachForecasts` and exposed as `sageo recommendations forecast`
- Merge rules emit concrete `Recommendation` objects (title, meta, H1, H2, schema, body, speed, backlink, indexability changes) persisted to state via `recommendations.UpsertRecommendations`
- `sageo recommendations list` command with `--url`, `--type`, `--top`, `--format` flags
- Interactive login flow (`sageo login`) for GSC OAuth and DataForSEO credentials
- Default locale: Australia (location_code 2036, language `en`) for all DataForSEO calls
- URL normalisation utilities for cross-source data joining
- Opportunity detection merging GSC + optional SERP evidence (legacy, superseded by merge engine)
- LLM provider abstraction (`internal/llm`) with Anthropic (Messages API, `claude-sonnet-4-6`) and OpenAI (Chat Completions, `gpt-5`) drivers, used by `sageo recommendations draft` to fill `RecommendedValue` with concrete copy (titles, meta descriptions, H1/H2s, body paragraphs, JSON-LD schema) validated against SERP length limits with retry

### Do now
- Keep command architecture stable
- Maintain JSON-first output contract
- Preserve config and output consistency
- Extend audit rules as needed
- Add new providers via the registry pattern
- Add new SERP providers behind the `serp.Provider` interface
- Extend merge rules when new data sources are added

### Do not do without explicit instructions
- Add multiple paid SEO providers at once
- Embed OpenAI/Anthropic inside the CLI by default
- Change the output envelope contract incompatibly
- Restructure the command hierarchy unnecessarily

## Conventions

- Language: Go
- CLI framework: Cobra
- Entry point: `cmd/sageo/main.go`
- Root command wiring: `internal/cli/root.go`
- Command files: `internal/cli/commands/*.go`
- Config package: `internal/common/config`
- Cost package: `internal/common/cost`
- Cache package: `internal/common/cache`
- URL normalisation: `internal/common/urlnorm`
- Retry utilities: `internal/common/retry`
- Output package: `pkg/output`
- Provider package: `internal/provider`
- Auth package: `internal/auth`
- GSC package: `internal/gsc`
- PSI package: `internal/psi`
- AEO mentions (Layer A — local): `internal/aeo/mentions`
- AEO LLM Mentions API (Layer B — DataForSEO): `internal/aeo/llmmentions`
- SERP package: `internal/serp` (adapters: `internal/serp/serpapi`, `internal/serp/dataforseo`)
- DataForSEO shared client: `internal/dataforseo`
- Opportunities package: `internal/opportunities`
- LLM package: `internal/llm` (drivers: `internal/llm/anthropic`, `internal/llm/openai`; side-effect registry: `internal/llm/providers`)
- Backlinks package: `internal/backlinks`
- Merge engine: `internal/merge`
- Recommendations: `internal/recommendations` (types aliased from `internal/state` to avoid import cycle)
- State persistence: `internal/state`
- Pipeline orchestrator: `internal/pipeline`
- Domain packages: `internal/crawl`, `internal/audit`, `internal/report`, `internal/report/html`
- Forecast package: `internal/forecast` (position→CTR curve, swappable via `SetCurve`)
- Test utilities: `internal/common/testutil` (fake HTTP servers for unit tests)

## Test Safety

Strict separation between unit and integration tests:

- **Unit tests** (default `go test ./...` / `make test`): zero network, zero cost. Use `httptest.NewServer` via `internal/common/testutil` factories, or a mock `HTTPClient`. No build tag.
- **Integration tests** (`make test-integration`): may hit paid APIs. MUST start with `//go:build integration` and guard every test function with `if os.Getenv("SAGEO_LIVE_TESTS") != "1" { t.Skip(...) }`. Named `*_integration_test.go`.
- `scripts/check-no-live-tests.sh` (wired via `make check-tests`, invoked by `make test`) enforces the rule.
- See `TESTING.md` for the full convention and copy-paste templates.

## Output Contract

Prefer envelope-style structured output:
- `success`
- `data`
- `error`
- `metadata`

Default command output should remain `json` for automation and agent usage.

Paid commands include additional metadata keys:
- `estimated_cost`, `currency`, `requires_approval`, `cached`, `source`, `fetched_at`, `dry_run`

Command-specific payload shapes:
- **Multi-model AEO** (`aeo responses --all` / `--models`): `data.results[]` — one row per engine/model with `{ engine, model, response, brand_mentions, error, cost }`; `metadata.estimated_cost` is the summed cost across all rows.
- **Recommendations list / draft** (`recommendations list`, `recommendations draft`): `data.recommendations[]` where each item matches `state.Recommendation` (`id`, `target_url`, `target_query`, `change_type`, `current_value`, `recommended_value`, `rationale`, `evidence[]`, `priority`, `effort_minutes`, `forecasted_lift`, `merged_finding_id`, `created_at`).
- **Forecast** (`recommendations forecast`): `data.forecasts[]` where each item is `{ recommendation_id, target_url, change_type, forecast: { estimated_monthly_clicks_delta, confidence_low, confidence_high, method } }`.
- **HTML report** (`report html`): non-envelope side effect — writes a self-contained HTML file to `--output`; stdout emits `{ success, data: { path, size_bytes } }`. Users produce a PDF via browser print-to-PDF (Cmd/Ctrl+P). `report pdf` remains as a deprecated alias routed to the HTML renderer.

## Configuration

Default config path:
- `~/.config/sageo/config.json`

Optional override:
- `SAGEO_CONFIG` (absolute `.json` path)

Supported env overrides:
- `SAGEO_PROVIDER`
- `SAGEO_API_KEY`
- `SAGEO_BASE_URL`
- `SAGEO_ORGANIZATION_ID`
- `SAGEO_SERP_PROVIDER`
- `SAGEO_SERP_API_KEY`
- `SAGEO_DATAFORSEO_LOGIN`
- `SAGEO_DATAFORSEO_PASSWORD`
- `SAGEO_APPROVAL_THRESHOLD_USD`
- `SAGEO_GSC_PROPERTY`
- `SAGEO_GSC_CLIENT_ID`
- `SAGEO_GSC_CLIENT_SECRET`
- `SAGEO_PSI_API_KEY`
- `SAGEO_LLM_PROVIDER` (default `anthropic`)
- `SAGEO_ANTHROPIC_API_KEY`
- `SAGEO_OPENAI_API_KEY`

> Backlinks API uses the existing DataForSEO credentials (`SAGEO_DATAFORSEO_LOGIN` / `SAGEO_DATAFORSEO_PASSWORD`) — no new env vars needed.

## Validation Commands

Run after changes (safe — no network, no cost):

```bash
go test ./...
go vet ./...
```

For full quality gate (still safe — unit tests only):

```bash
make fmt
make vet
make test
make lint
```

Opt-in live API coverage (costs money — requires credentials):

```bash
make test-integration   # SAGEO_LIVE_TESTS=1 go test -tags integration ./...
make test-all           # unit + integration
```

## Local CLI Install/Update

Use a single global command (`sageo`) by installing from source:

```bash
make install
```

If `sageo` is not found after install, ensure `~/go/bin` is on PATH and reload shell config.

## Lightweight Release Policy

Default release flow should stay lightweight (avoid costly multi-platform packaging unless explicitly requested):

1. Run fast checks:
   - `go vet ./...`
   - `go test -race ./...`
2. Commit and push to `main`
3. Create/push a semver tag (patch bump)
4. Create a GitHub Release from the tag **without** attached binary assets

Only run `make release` when explicitly asked to produce packaged binaries.

## Commit Message Quality Policy

Every commit should include a clear summary of what changed in the project:

- Subject line: one line, starts with `Add`, `Update`, `Fix`, `Remove`, or `Refactor`
- Body: short bullet points grouped under relevant sections (omit empty sections):
  - `Added:`
  - `Updated:`
  - `Fixed:`
  - `Docs:`
- Bullets must describe concrete changes (file/function/behavior), not generic wording.
