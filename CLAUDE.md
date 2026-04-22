# Sageo CLI: Agent Notes

Notes for an AI coding agent working in this repository. Keep it accurate; keep it tight.

## Scope

Working features, grouped by area:

- **Data collection.** Crawl, audit, GSC (OAuth2), PSI, SERP (SerpAPI + DataForSEO), Labs, backlinks, multi-model AEO fan-out, brand mentions (local scan + DataForSEO LLM Mentions API).
- **Analysis.** Cross-source merge engine with 14 rules producing evidence-backed `Recommendation` objects. Priority scoring 10 to 100.
- **Recommendations pipeline.** LLM drafter (Anthropic, OpenAI) fills `recommended_value`. Review gate (`recommendations review`) requires human approve/edit/reject before anything ships. Forecaster (`recommendations forecast`) attaches a priority tier and calibrated click-delta range.
- **Orchestration.** `sageo run <url>` drives crawl, audit, GSC, PSI, Labs, SERP, backlinks, AEO, mentions, merge, recommend, draft, forecast, review-gate under a single `--budget` ceiling with `--skip`, `--only`, `--resume`, `--approve`, `--dry-run`.
- **Presentation.** Self-contained HTML report (`report html`). `report pdf` is a deprecated alias routing to HTML. Users print to PDF via the browser (Cmd/Ctrl+P).
- **History + learning.** Per-run snapshots under `.sageo/snapshots/<ts>/`. `snapshots list|show|path|prune`. `compare` diffs two snapshots and writes observed-lift records to `.sageo/calibration.json`, which the forecaster consumes.
- **Infrastructure.** JSON envelope output, cost estimation, approval gate, on-disk cache with TTL, `--dry-run` on every paid command, per-project state at `.sageo/state.json` with `BrandTerms` set by `sageo init --brand`.

### Do now

- Keep the command hierarchy stable.
- Maintain the JSON envelope output contract.
- Extend audit rules, merge rules, and ChangeTypes as needed (each ChangeType must cite a section in `docs/research/ai-citation-signals-2026.md`).
- Add new providers via the existing registry patterns (`internal/provider`, `internal/llm`, `internal/serp`).
- Never let LLM drafts bypass the review gate into a client-facing artefact. The review gate is the contract.
- Every paid command supports `--dry-run` and includes the paid-metadata keys.

### Do not do without explicit instructions

- Add multiple paid SEO providers at once.
- Change the output envelope contract incompatibly.
- Restructure the command hierarchy.
- Introduce live network calls into unit tests. Use `internal/common/testutil` fakes or `//go:build integration`.
- Hand-edit `.sageo/state.json` or `.sageo/snapshots/`. The CLI owns the schema.

## Conventions

- **Language.** Go.
- **CLI framework.** Cobra.
- **Entry point.** `cmd/sageo/main.go`.
- **Root wiring.** `internal/cli/root.go`.
- **Command files.** `internal/cli/commands/*.go`, one file per top-level command (plus split files for large groups).
- **Domain packages.** `internal/crawl`, `internal/audit`, `internal/gsc`, `internal/psi`, `internal/serp` (adapters `internal/serp/serpapi`, `internal/serp/dataforseo`), `internal/dataforseo` (shared client), `internal/backlinks`, `internal/aeo/mentions` (local Layer A), `internal/aeo/llmmentions` (Layer B DataForSEO), `internal/opportunities`, `internal/merge`, `internal/recommendations`, `internal/forecast`, `internal/compare`, `internal/pipeline`.
- **State.** `internal/state` (state.go, recommendations.go, snapshot.go). `Recommendation` is defined here so it can be embedded without import cycles; `internal/recommendations` re-exports via type aliases.
- **LLM.** `internal/llm` (interface, registry), drivers `internal/llm/anthropic`, `internal/llm/openai`, side-effect registry `internal/llm/providers`.
- **Provider abstraction.** `internal/provider`, built-in fetcher `internal/provider/local`.
- **Auth.** `internal/auth` (GSC OAuth token persistence + refresh).
- **Infrastructure.** `internal/common/config`, `internal/common/cost`, `internal/common/cache`, `internal/common/urlnorm`, `internal/common/retry`, `internal/common/testutil`.
- **Presentation.** `internal/report` (JSON reports), `internal/report/html` (self-contained HTML).
- **Output.** `pkg/output` (envelope, error codes).
- **Version.** `internal/version`.

## Output contract

Default output is JSON. Envelope shape:

```
{ "success": bool, "data": ..., "error": {...} | null, "metadata": {...} }
```

Paid commands add to `metadata`:
- `estimated_cost` (USD float)
- `currency` (usually `"USD"`)
- `requires_approval` (bool)
- `cached` (bool)
- `source` (`"live"` | `"cache"` | `"dry-run"`)
- `fetched_at` (RFC3339)
- `dry_run` (bool)

Command-specific shapes:

- **Multi-model AEO** (`aeo responses --all|--models|--tier`): `data.results[]` with `{ engine, model, response, brand_mentions, error, cost }`. `metadata.estimated_cost` is the summed cost.
- **Recommendations list / draft**: `data.recommendations[]` where each item matches `state.Recommendation` (`id`, `target_url`, `target_query`, `change_type`, `current_value`, `recommended_value`, `rationale`, `evidence[]`, `priority`, `effort_minutes`, `forecasted_lift`, `merged_finding_id`, `created_at`, `review_status`, `reviewed_at`, `reviewed_by`, `review_notes`, `original_draft`).
- **Forecast** (`recommendations forecast`): `data.top[]` with `{ id, target_url, change_type, priority_tier (high|medium|low|unknown), point_estimate, range_low, range_high, raw_estimate, calibrated (bool), confidence_label, caveats[], calibration_samples }`. Aggregates: `tier_counts`, `estimated_range_low`, `estimated_range_high`. The embedded `Forecast` struct on each recommendation has `raw_estimate`, `raw_confidence_low`, `raw_confidence_high`, optional `calibrated_*`, `priority_tier`, `confidence_label`, `caveats[]`, `method`, `calibration_samples`. Read `priority_tier` as the primary signal.
- **Analyze** (`analyze`): `metadata.review_status_counts: { pending, approved, edited, rejected }`.
- **Compare** (`compare`): structured deltas (`GSCDelta`, `PSIDelta`, `SERPDelta`, `AEODelta`, `BacklinksDelta`, `AuditDelta`, `RecommendationsDelta`) plus `metadata.causation_caveats[]`. `--output-html` also writes a self-contained HTML diff.
- **HTML report** (`report html`): non-envelope side effect. Writes a file; stdout emits `{ success, data: { path, size_bytes } }`.

## Configuration

Default config path: `~/.config/sageo/config.json`. Override with `SAGEO_CONFIG` (absolute `.json` path).

Supported env overrides (see `internal/common/config/config.go`):

- `SAGEO_PROVIDER`, `SAGEO_API_KEY`, `SAGEO_BASE_URL`, `SAGEO_ORGANIZATION_ID`
- `SAGEO_SERP_PROVIDER`, `SAGEO_SERP_API_KEY`
- `SAGEO_DATAFORSEO_LOGIN`, `SAGEO_DATAFORSEO_PASSWORD` (also used by Backlinks, Labs, AEO, GEO)
- `SAGEO_APPROVAL_THRESHOLD_USD`
- `SAGEO_GSC_PROPERTY`, `SAGEO_GSC_CLIENT_ID`, `SAGEO_GSC_CLIENT_SECRET`
- `SAGEO_PSI_API_KEY`
- `SAGEO_LLM_PROVIDER` (default `anthropic`), `SAGEO_ANTHROPIC_API_KEY`, `SAGEO_OPENAI_API_KEY`

Default DataForSEO locale is Australia (location_code 2036, language `en`).

## Validation commands

Safe (no network, no cost):

```bash
go vet ./...
go test ./...
make fmt
make vet
make test
make lint
```

Opt-in live API coverage (costs money, requires credentials):

```bash
make test-integration   # SAGEO_LIVE_TESTS=1 go test -tags integration ./...
make test-all           # unit then integration
```

`make test` runs `scripts/check-no-live-tests.sh` first via `make check-tests`; it refuses if any non-integration test references `http.DefaultClient`, `http.Get(`, etc. See TESTING.md.

## Local CLI install

```bash
make install
```

Installs via `go install`. Ensure `$(go env GOPATH)/bin` (typically `~/go/bin`) is on `PATH`.

## Release policy

Default flow is lightweight; avoid multi-platform packaging unless explicitly requested.

1. `go vet ./...`
2. `go test -race ./...`
3. Commit and push to `main`.
4. Create and push a semver tag (patch bump by default).
5. Create a GitHub Release from the tag **without** attached binary assets.

Run `make release` only when asked to produce packaged binaries.

## Commit message policy

Every commit must summarise what changed.

- Subject line: one line, starts with `Add`, `Update`, `Fix`, `Remove`, or `Refactor`.
- Body: short bullets grouped under `Added:`, `Updated:`, `Fixed:`, `Docs:`. Omit empty sections.
- Bullets name concrete files, functions, or behaviours. No generic wording.

## Skills

Sageo ships agent skills in two locations:

- `.claude/skills/` for Claude-family agents (e.g. `.claude/skills/sageo/`).
- `.gg/skills/` for the GG tooling (e.g. `.gg/skills/sageo.md`).

Keep both in sync when the command surface changes.

## Honest framing rules

These rules bind any output the agent produces on behalf of sageo, in reports, recommendations, commit messages, or user-facing text.

1. **Never quote a specific click number.** Always present forecasts as a priority tier (`high`, `medium`, `low`, `unknown`) plus a range. The tier is the primary signal; the range is supporting context.
2. **Surface caveats.** When volume is low, history is short, or an algorithm update is known-recent, say so in the same breath as the forecast. The `Forecast.Caveats` field exists for this reason: propagate it.
3. **Do not claim causation from observational data.** `compare` output is correlational. Ranking changes between two snapshots do not prove any recommendation caused them.
4. **LLM drafts are pending by default.** Any `recommended_value` drafted by the LLM carries `review_status = pending_review`. Do not treat pending drafts as approved copy. Do not include rejected drafts in reports.
5. **Cite the research section that supports a recommendation.** When presenting a ChangeType to a user, name the section in `docs/research/ai-citation-signals-2026.md` that underwrites it (e.g. "ChangeTLDR: Growth Memo direct-answer study, B.1.2"). No anonymous claims.
