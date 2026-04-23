# Sageo Architecture

How the code is laid out and how a single run flows through it. Written for an engineer (human or AI) about to change something.

## High-level shape

```
                        ┌─────────────────────────────────┐
                        │   cmd/sageo/main.go (Cobra)     │
                        └───────────────┬─────────────────┘
                                        │
                        ┌───────────────▼─────────────────┐
                        │  internal/cli/commands/*.go     │
                        │  (thin handlers, flag parsing)  │
                        └───────────────┬─────────────────┘
                                        │
        ┌───────────────┬───────────────┼───────────────┬───────────────┐
        │               │               │               │               │
        ▼               ▼               ▼               ▼               ▼
  data collection   analysis      recommendations   presentation   infrastructure
  ───────────────   ────────      ───────────────   ────────────   ──────────────
   crawl            merge          recommendations    report         state
   audit                           forecast           report/html    common/config
   gsc                             compare (diff)                    common/cost
   psi                                                               common/cache
   serp                                                              common/retry
   labs                                                              common/urlnorm
   backlinks                                                         common/testutil
   aeo/mentions                                                      provider
   aeo/llmmentions                                                   auth
   opportunities                                                     llm + drivers
                                                                     dataforseo
                                                                     pipeline (orchestrator)
                                                                     version

  Data flow of `sageo run`:

  ┌─────┐   ┌───────┐   ┌─────┐   ┌─────┐   ┌──────┐   ┌────────┐   ┌─────┐
  │crawl├──►│ audit ├──►│ GSC ├──►│ PSI ├──►│ SERP ├──►│  Labs  ├──►│ AEO │
  └─────┘   └───────┘   └─────┘   └─────┘   └──────┘   └────────┘   └──┬──┘
                                                                       │
  ┌────────┐   ┌──────────┐   ┌────────┐   ┌───────┐   ┌─────────┐   ┌─▼──────┐
  │ merge  ◄───┤mentions  │◄──┤forecast│◄──┤drafter│◄──┤recommend│◄──┤backlnks│
  └───┬────┘   └──────────┘   └────────┘   └───────┘   └─────────┘   └────────┘
      │
      ▼
  ┌───────────────┐   ┌────────────────┐   ┌──────────────────────────────┐
  │ review gate   ├──►│ snapshot write ├──►│ .sageo/snapshots/<ts>/ +     │
  │ (TUI or skip) │   │ (atomic)       │   │ state.json copy              │
  └───────────────┘   └────────────────┘   └──────────────────────────────┘
```

All stages persist their results to `.sageo/state.json` as they go. `internal/pipeline` owns ordering, budget enforcement, dry-run planning, and the snapshot/calibration hooks.

## Package layout

### Data collection

| Package | Purpose |
|---|---|
| `internal/crawl` | BFS crawler with depth + page limits, concurrent fetching |
| `internal/audit` | Rule-based SEO audit over a crawl result, emits `Finding`s and a 0-100 score |
| `internal/gsc` | Google Search Console API client, OAuth token plumbing, query/pages/keywords |
| `internal/auth` | GSC OAuth flow, token persistence + refresh |
| `internal/psi` | PageSpeed Insights client (LCP, CLS, FCP, TBT, SI) |
| `internal/serp` | SERP provider interface plus SerpAPI and DataForSEO adapters (`internal/serp/serpapi`, `internal/serp/dataforseo`) |
| `internal/dataforseo` | Shared DataForSEO HTTP client used by Labs, Backlinks, AEO, GEO, SERP |
| `internal/backlinks` | DataForSEO Backlinks API (summary, list, referring-domains, competitors, gap) |
| `internal/aeo/mentions` | Layer A: offline scan of stored AEO responses for brand terms |
| `internal/aeo/llmmentions` | Layer B: DataForSEO LLM Mentions API (search, top-pages, top-domains) |
| `internal/opportunities` | GSC opportunity seeds (legacy, superseded by merge engine) |
| `internal/provider` | HTTP fetcher abstraction with built-in `local` provider |
| `internal/llm` | LLM provider interface + registry |
| `internal/llm/anthropic` | Anthropic Messages API driver (`claude-sonnet-4-6`) |
| `internal/llm/openai` | OpenAI Chat Completions driver (`gpt-5`) |
| `internal/llm/providers` | Side-effect registration of built-in drivers |

### Analysis

| Package | Purpose |
|---|---|
| `internal/merge` | 14 cross-source rules that emit `MergedFinding` + atomic `Recommendation` objects |
| `internal/recommendations` | Recommendation types (aliased from `internal/state` to avoid cycles), drafter, context assembly, store helpers |
| `internal/forecast` | Position-to-CTR model (AWR 2024 baseline), calibration against `.sageo/calibration.json`, priority-tier attachment |
| `internal/compare` | Snapshot diff: typed deltas per source, per-ChangeType detectors that infer whether a recommendation was addressed, observed-lift emission |

### Presentation

| Package | Purpose |
|---|---|
| `internal/report` | JSON report generation and listing |
| `internal/report/html` | Self-contained HTML report renderer (inline CSS, optional logo, brand color, appendix) |
| `pkg/output` | JSON envelope formatter, error codes |

### Infrastructure

| Package | Purpose |
|---|---|
| `internal/state` | `.sageo/state.json` schema and per-run snapshot store (state.go, recommendations.go, snapshot.go) |
| `internal/pipeline` | Orchestrator for `sageo run`: stage ordering, budget enforcement, resume, snapshot hooks |
| `internal/common/config` | Config load/save, env-var overrides, redaction |
| `internal/common/cost` | Cost estimation helpers |
| `internal/common/cache` | On-disk response cache with TTL |
| `internal/common/urlnorm` | URL normalisation for cross-source joins |
| `internal/common/retry` | Backoff helpers |
| `internal/common/testutil` | Fake HTTP servers for unit tests (DataForSEO, Anthropic, OpenAI, PSI) |
| `internal/version` | Build metadata injected at link time |
| `internal/cli` | Cobra root wiring |
| `internal/cli/commands` | One file per top-level command |
| `cmd/sageo` | `main.go` entry point |

## Data flow: a single `sageo run`

1. **Parse.** `internal/cli/commands/run.go` parses flags, validates the URL, loads prompts, and builds a `pipeline.Config` plus an ordered `[]Stage`.
2. **Init.** If `.sageo/state.json` is absent, `state.Init` creates it with the target URL and an empty history.
3. **Plan.** `pipeline.Run` enforces `--skip` / `--only`, applies `--resume` from `state.PipelineCursor`, computes per-stage estimates, and (under `--dry-run`) prints the plan and exits.
4. **Stage loop.** For each stage:
   - Budget check: if accumulated paid spend plus this stage's estimate exceeds `--budget`, skip with `budget-exceeded`.
   - Approval gate: if the stage is paid and the estimate crosses `SAGEO_APPROVAL_THRESHOLD_USD`, require `--approve`.
   - Execute `stage.Run(ctx, state)`. The stage mutates state in place (e.g. `state.GSC = &GSCData{...}`, `state.UpsertPSI(...)`, `state.UpsertAEOResponses(...)`).
   - `state.Save(workDir)` after each successful stage. `state.PipelineCursor` advances to the stage name.
5. **Merge.** `internal/merge` applies 14 rules over the populated state, emits `MergedFinding`s, and calls `recommendations.UpsertRecommendations` with the derived `Recommendation` set.
6. **Draft.** `recommendations` drafter iterates recommendations with empty `recommended_value`, assembles context (page content + evidence), calls the configured LLM driver, validates against SERP length limits with retry, and writes drafts with `review_status = pending_review` and `original_draft` preserved.
7. **Forecast.** `forecast.AttachForecasts` computes raw click-delta estimates, reads `.sageo/calibration.json`, fits a profile if enough samples, and attaches `Forecast` with `priority_tier`, `point/low/high`, `caveats`, and `calibration_samples`.
8. **Review gate.** Unless `--no-review`, `run.go` invokes the interactive TUI (`recommendations review`) so a human approves, edits, or rejects each draft. `--auto-approve-all` bypasses (unsafe).
9. **Snapshot.** Unless `--no-snapshot`, `state.WriteSnapshot` creates `.sageo/snapshots/<ts>.tmp/`, writes `state.json`, `recommendations.json`, `report.html`, `metadata.json`, renames atomically to `.sageo/snapshots/<ts>/`, updates `index.json`, and refreshes the top-level `.sageo/state.json` copy. Old snapshots are pruned per `--retain N` and `--retain-within`.
10. **Audit log.** A `PipelineRun` entry is appended to `state.PipelineRuns` with stages run, total cost, outcome, and any error.

## State schema

Authoritative types live in `internal/state/state.go` and `internal/state/recommendations.go`. Top-level `State`:

- **`Site`, `Initialized`, `LastCrawl`, `Score`, `PagesCrawled`.** Project-level scalars from `init` and the most recent audit.
- **`Findings`, `MergedFindings`, `LastAnalysis`.** Output of `audit` and `analyze`; `MergedFindings` is `json.RawMessage` to avoid cycles.
- **`GSCData`.** Last pull timestamp, property, top pages, top keywords (each a `GSCRow`).
- **`PSIData`.** Last run timestamp, `[]PSIResult` upserted by URL + strategy.
- **`SERPData`.** Last run timestamp, `[]SERPQueryResult` with features, related questions, top domains, our position per query.
- **`LabsData`.** Target domain, `[]LabsKeyword` (keyword, volume, difficulty, CPC, intent, position), competitor domains.
- **`BacklinksData`.** Totals, broken-link count, rank, dofollow/nofollow split, spam score, top referrers, gap domains.
- **`AEOData`.** `[]AEOPromptResult` keyed by prompt; each has per-engine `AEOResponseResult`s with model, response text, fetched-at.
- **`BrandTerms`.** Set by `sageo init --brand "Name,alias"`.
- **`Mentions`.** `[]MentionsData` keyed by term: local matches from Layer A, plus domain share and top pages from Layer B.
- **`Recommendations`.** `[]Recommendation` upserted by ID. Each has a `ChangeType`, current + recommended values, evidence, priority, `ForecastedLift`, and review-gate fields (`ReviewStatus`, `ReviewedAt`, `ReviewedBy`, `OriginalDraft`).
- **`History`.** Rolling log (max 200 entries) of agent/user actions.
- **`PipelineCursor`, `PipelineRuns`.** Resume bookmark and audit log of autonomous runs.

Snapshots on disk (`.sageo/snapshots/<ts>/`) are frozen copies: `state.json`, `recommendations.json`, `report.html`, `metadata.json` (a `SnapshotMeta` with stages run, total cost, pipeline version, git commit, outcome). See `internal/state/snapshot.go` for the atomic write protocol and the `index.json` pointer. After a successful snapshot, the pipeline also mirrors the rendered report to `.sageo/reports/latest.html` so the most recent report is always at a predictable path.

Ad-hoc `sageo report html` invocations without `--output` land in `.sageo/reports/sageo-report-<UTC-timestamp>.html` when a project is detected, falling back to `./sageo-report.html` with a stderr note otherwise.

## Recommendation lifecycle

One sentence per stage.

1. **Merged finding.** `internal/merge` rules fire against populated state and emit a `MergedFinding` describing what is wrong and why.
2. **Generator.** The rule emits a `Recommendation` with `change_type`, `target_url`, `evidence`, `priority`, `effort_minutes`, and an empty `recommended_value`; `state.UpsertRecommendations` stores it keyed by ID.
3. **Drafter.** `recommendations.Draft` picks up recommendations with empty values, assembles page + evidence context, calls the LLM driver, validates output, and stores `recommended_value` with `review_status = pending_review` and `original_draft` preserved.
4. **Review gate.** `sageo recommendations review` walks the queue; the human sets `review_status` to `approved`, `edited`, or `rejected` with `reviewed_at`, `reviewed_by`, and optional `review_notes`.
5. **Forecaster.** `forecast.AttachForecasts` computes raw click delta from the position-to-CTR curve, applies the calibration profile when available, and sets `ForecastedLift` with `priority_tier`, range, caveats, and `calibration_samples`. On cold projects without calibration data, `forecast.TierWithRulePriority` falls the tier back to the rule-engine priority score (>=80 High, >=50 Medium, else Low) so the report still shows actionable tiers instead of UNKNOWN across the board; these are badged `(provisional)` in the HTML.
6. **Report.** `internal/report/html` renders approved and edited recommendations as cards, pending with a badge, rejected excluded entirely; the forecast table is sorted by priority tier.
7. **Snapshot.** The finalised `Recommendation` is frozen into `.sageo/snapshots/<ts>/recommendations.json`.
8. **Compare.** On the next run, `internal/compare` detectors check whether the recommendation was addressed (e.g. schema now visible, PSI crossed a good-band threshold, referring domains grew).
9. **Calibration.** When addressed and GSC data is paired in both snapshots, `compare` appends an `ObservedLift` record to `.sageo/calibration.json`; the forecaster reads it on the next `recommendations forecast` invocation.

## CLI UX: doctor and preflight

`sageo` uses two CLI-layer guardrails to keep agent-driven runs honest:

- **`sageo doctor`** (`internal/cli/commands/doctor.go`) runs a fixed checklist against project state, config, and stored OAuth tokens: `project_initialised`, `brand_terms`, `gsc_auth`, `gsc_property`, `psi_api_key`, `llm_provider`, `dataforseo_creds`. Each check is a small pure function receiving a `doctorInputs` bundle and returning a `doctorCheck{Name, Status, Message, Fix}`. JSON output lists `data.checks[]` plus a `data.summary{pass,warn,fail}` tally. Warnings never fail the command; any fail causes a non-zero exit.
- **Hard preflight on `sageo run`** (`preflightGSCCheck` in `run.go`) aborts with `GSC_NOT_CONFIGURED` when no active GSC property is configured, unless the user passes `--skip gsc` or `--only` excludes gsc. The returned error carries a `Hint()` with the three-line fix, which `output.PrintCodedErrorWithHint` renders in both the JSON envelope (`error.hint`) and the text output (`Hint:` line on stderr). This replaces the silent warning that produced 20 unknown-tier recommendations on the baysidebuilderswa.com.au run.

Commands that terminate setup steps (`init`, `auth login gsc`, `run`, `recommendations review`) also write a `Next steps:` block to stderr via `printNextSteps` in `internal/cli/commands/nextsteps.go`, so both humans and agents see the exact next command to run. Stderr is used unconditionally so JSON consumers parsing stdout are unaffected.

## Cost control

- **Estimate.** Each paid stage implements `EstimateUSD(*state.State) float64`. The pipeline sums estimates up front and surfaces them via `--dry-run`.
- **Approval gate.** When a stage estimate crosses `SAGEO_APPROVAL_THRESHOLD_USD`, execution blocks until the user passes `--approve` (or re-runs below the threshold). `sageo run --approve` pre-approves all stages.
- **Budget flag.** `sageo run --budget N` is a hard ceiling; stages whose estimates would push accumulated spend over `N` are skipped with a `budget-exceeded` outcome rather than partial.
- **Caching.** `internal/common/cache` stores responses on disk with per-command TTL. Cached reads set `metadata.cached = true` and `metadata.source = "cache"`.
- **Dry run.** Every paid command accepts `--dry-run`, printing estimates and returning before any paid call. `sageo run --dry-run` prints the full stage plan.
- **Test isolation.** Unit tests never make live calls (see TESTING.md). Integration tests are tagged `//go:build integration` and gated on `SAGEO_LIVE_TESTS=1`. `scripts/check-no-live-tests.sh` enforces this in `make test`.

## Extensibility

**New data source.** Add an `internal/<source>` package with a client and a state struct. Add an `Upsert<Source>` method on `state.State`. Add a command under `internal/cli/commands/<source>.go`. Register a stage in `internal/cli/commands/run.go` with an `EstimateUSD` if paid. Extend the state schema test.

**New recommendation ChangeType.** Add the constant in `internal/state/recommendations.go` with a doc comment citing the section in `docs/research/ai-citation-signals-2026.md` that supports it. Teach the merge engine to emit it (new rule in `internal/merge`). Teach the drafter prompt assembly how to describe it. Teach the HTML renderer how to display it. Teach the forecaster how to weight it. Teach the compare detectors how to decide if it was addressed.

**New LLM provider.** Add an `internal/llm/<name>` driver implementing `llm.Provider`. Register it from `internal/llm/providers/providers.go` via `llm.Register`. Add credentials to `internal/common/config` and the env-var list.

**New SERP provider.** Implement `serp.Provider` under `internal/serp/<name>`. Wire selection in `internal/cli/commands/serp.go` via the `SERPProvider` config field.

**New merge rule.** Add the rule function in `internal/merge` returning zero or more `Recommendation`s from state. Register it in the rule list. Cover with a table-driven unit test.

## What this architecture does not do

- **No server-side component.** Sageo is a CLI; there is no shared state across users. Each project has its own `.sageo/`.
- **No agent loop yet.** Recommendations are one-shot per run. There is no self-critique or multi-turn refinement; the LLM drafts once and the review gate does the quality check.
- **No post-deployment measurement without a re-run.** Sageo only learns whether a recommendation worked when the user runs `sageo run` again and `compare` sees the change.
- **No CMS auto-apply.** Recommendations describe the change; sageo never pushes code to your site. Applying copy is a human or an external integration.
- **No continuous monitoring.** There is no daemon, no watcher, no scheduled polling. Cadence is whatever the user (or their CI) invokes.
