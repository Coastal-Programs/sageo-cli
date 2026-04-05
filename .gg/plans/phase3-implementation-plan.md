# Phase 3 Implementation Plan (Agent-first SEO/AEO/GEO Layer)

## Context Snapshot

Current codebase is Phase 2-complete and stable:
- Root command wiring in `internal/cli/root.go:46-52` currently registers `version`, `config`, `crawl`, `audit`, `report`, `provider` only.
- Config model in `internal/common/config/config.go:13-18` supports only core provider/fetch settings.
- Output envelope contract in `pkg/output/output.go:21-27` already supports `metadata`, which is the right place for cost/freshness/caching metadata.
- Provider registry pattern in `internal/provider/provider.go:31-59` is minimal and currently oriented to HTTP page fetching (`Fetcher`), with only `local` registered (`internal/provider/local/local.go:94-97`).
- Tests already enforce command hierarchy (`internal/cli/root_test.go:5-55`) and config behavior (`internal/common/config/config_test.go:8-53`).

Important mismatch found:
- `README.md:28-40` claims GSC + live SERP + AI visibility tracking, but code does not yet implement these command families. Phase 3 should close this gap without breaking current command architecture.

## Phase 3 Scope (from project guidance)

Must add:
- Google Search Console integration
- Exactly one SERP provider integration
- Cost-aware execution contract (`estimated_cost`, `requires_approval`, `cached`, `source`, `fetched_at`)
- `--dry-run` for paid workflows

Must avoid for this phase:
- Multiple paid providers at once
- Embedded LLM execution in CLI by default
- Breaking output envelope
- Unnecessary command hierarchy rewrites

## Provider Choice Recommendation

Recommend **SerpAPI** (single provider for phase 3) because:
- Simpler request model and faster integration path for first paid SERP adapter.
- Easier dry-run + cost estimation behavior with deterministic unit tests.
- Keeps DataForSEO as a future adapter behind the same interfaces.

## Proposed Architecture Additions

- Keep existing crawl/audit/report flow untouched.
- Add new domain services instead of overloading existing provider fetcher abstraction.
- Preserve JSON envelope shape; add new metadata fields through existing `metadata` map.
- Introduce an explicit cost contract package used by GSC/SERP/opportunity commands.

### New package boundaries

- `internal/gsc`
  - GSC client + service layer (sites list, query pages/keywords, opportunities seed data)
- `internal/serp`
  - SERP service abstraction + `serpapi` adapter
- `internal/opportunities`
  - Merge logic: crawl/audit + GSC + optional SERP evidence
- `internal/common/cost`
  - cost estimation and approval gate types/logic
- `internal/common/cache`
  - simple file-based cache for paid responses and metadata
- `internal/auth` (or `internal/gsc/auth`)
  - token store + auth state helpers for CLI OAuth lifecycle

## Config and Environment Model Changes

Extend `internal/common/config/config.go` safely (non-breaking defaults):
- Add fields for GSC and SERP runtime selection and approval guardrails, e.g.:
  - `serp_provider` (default `serpapi`)
  - `serp_api_key`
  - `approval_threshold_usd`
  - `gsc_property`
  - `gsc_client_id`, `gsc_client_secret` (optional depending on auth strategy)
- Add env overrides for new keys (pattern already in `applyEnvOverrides`, lines `143-156`).
- Update `Set/Get/Redacted` switch handling (`101-141`) with secret redaction for any sensitive values.

## CLI Command Expansion Plan

Add new command files under `internal/cli/commands/` and wire in `internal/cli/root.go`:
- `auth.go`
- `gsc.go`
- `serp.go`
- `opportunities.go`

Root registration update point:
- `internal/cli/root.go:46-52`

Tests to extend:
- `internal/cli/root_test.go:8-15` expected top-level commands
- `internal/cli/root_test.go:34-40` expected subcommand maps

## Output Contract Extensions (No Breaking Changes)

Continue using `output.PrintSuccess(..., metadata, ...)` and add standard metadata keys:
- `estimated_cost`
- `currency`
- `requires_approval`
- `cached`
- `source`
- `fetched_at`
- `dry_run`

For errors, add new machine codes in `pkg/output/errors.go` for auth/provider/cost gate failures, e.g.:
- `AUTH_REQUIRED`
- `AUTH_FAILED`
- `APPROVAL_REQUIRED`
- `ESTIMATE_FAILED`
- `SERP_FAILED`
- `GSC_FAILED`

## External Integration Notes

### Google Search Console
- Implement via official Google API client for Go and OAuth2 flow suitable for CLI.
- Support property formats:
  - URL-prefix properties (`https://example.com/`)
  - Domain properties (`sc-domain:example.com`)
- First commands should focus on:
  - list accessible sites
  - query pages
  - query keywords
  - opportunities seed extraction (impressions + CTR + position filters)

### SerpAPI
- Add one adapter only in phase 3.
- Commands to support deterministic paid behavior:
  - `serp analyze --query ... [--dry-run]`
  - optional compare mode after analyze is stable
- All paid calls pass through a cost gate before execution.

## Caching and Freshness Model

Introduce file-based cache under Sageo config dir (e.g. `~/.config/sageo/cache/`):
- Keyed by provider + normalized request hash.
- Cached item includes:
  - payload
  - `source`
  - `fetched_at`
  - TTL metadata
- Command metadata should surface `cached` and `fetched_at` consistently.

## Opportunities Layer

`internal/opportunities` should produce machine-readable objects combining:
- crawl/audit quality signals
- GSC page/query performance
- optional SERP ranking validation for narrowed candidates

Recommended first object fields:
- `type` (`page`, `keyword`, `answer`)
- `target`
- `evidence`
- `confidence`
- `impact_estimate`
- `effort_estimate`
- `sources`
- `estimated_cost`

## Risks and Mitigations

- OAuth complexity in headless/dev environments
  - Mitigate with `auth status` + clear token path + explicit failure codes.
- README/product claims currently ahead of implementation
  - Mitigate by shipping docs updates with each command family increment.
- Cost-estimation drift vs actual provider billing
  - Mitigate by documenting estimate assumptions and returning estimate basis in metadata.
- Test flakiness with external APIs
  - Mitigate by strict interface mocking and fixture-based tests; no live API calls in unit tests.

## Verification Criteria

- Existing commands (`crawl`, `audit`, `report`, `provider`, `config`) remain backward compatible.
- New commands return envelope-style JSON with stable metadata keys.
- `--dry-run` on paid workflows performs no network calls to paid provider.
- Approval threshold blocks execution when estimated cost exceeds config limit and sets `requires_approval=true`.
- Cache hit/miss behavior is deterministic and reflected in output metadata.
- Command tree tests and config tests are updated and passing.
- Full quality gate per CLAUDE guidance:
  - `go test ./...`
  - `go vet ./...`
  - plus optional full make gate when needed.

## Steps
1. Add phase-3 core domain contracts by creating `internal/common/cost` and `internal/common/cache` packages with typed cost, approval, and cache metadata models used by future commands.
2. Extend `internal/common/config/config.go` and `internal/common/config/config_test.go` to support GSC/SERP settings, approval thresholds, env overrides, and redaction-safe read/write behavior.
3. Add new machine-readable error codes in `pkg/output/errors.go` for auth, cost-gate, GSC, and SERP failure classes used by phase-3 commands.
4. Implement GSC auth/token state support in a new `internal/auth` (or `internal/gsc/auth`) package and add CLI command group `internal/cli/commands/auth.go` with `login gsc`, `status`, and `logout gsc` behavior.
5. Implement GSC service integration in `internal/gsc` and add `internal/cli/commands/gsc.go` with `sites list`, `sites use`, `query pages`, `query keywords`, and `opportunities` subcommands returning envelope JSON.
6. Implement SERP abstraction in `internal/serp` with exactly one adapter (`serpapi`) and add `internal/cli/commands/serp.go` with `analyze` (and optional `compare`) including `--dry-run`, estimate metadata, and approval gate enforcement.
7. Implement `internal/opportunities` merge logic and `internal/cli/commands/opportunities.go` to combine crawl/audit + GSC + optional SERP evidence into deterministic opportunity objects.
8. Wire new top-level commands in `internal/cli/root.go` and expand `internal/cli/root_test.go` to assert registration and subcommand structure for `auth`, `gsc`, `serp`, and `opportunities`.
9. Add focused unit tests for cost estimation, approval blocking, dry-run no-call guarantees, cache metadata behavior, and deterministic output envelopes across new command handlers.
10. Update `README.md`, `ARCHITECTURE.md`, `CLAUDE.md`, and `CHANGELOG.md` to accurately document shipped phase-3 capabilities, free-vs-paid command boundaries, and cost/caching/approval behavior.