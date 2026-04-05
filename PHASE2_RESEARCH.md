# Phase 2 Research Notes (Real-world Pattern Pass)

Date: 2026-04-05

This note captures practical patterns observed from public Go repositories and applies them to `sageo-cli` phase-two implementation.

## 1) CLI command ergonomics (validated)

Observed pattern:
- Cobra CLIs commonly use `SilenceErrors: true` and `SilenceUsage: true`
- Global output flags are often persistent (`--output`, sometimes `--verbose`)

`Source examples`:
- `aquasecurity/trivy` (`SilenceUsage` + `SilenceErrors`)
- `runpod/runpodctl` (`rootCmd.PersistentFlags().StringVarP(... "output", "o", "json" ...)`)

Status in `sageo-cli`:
- Already aligned in `internal/cli/root.go`.

## 2) Job status lifecycle naming (recommended)

Observed pattern (common across Go services/CLIs):
- `pending`, `running`, `completed`/`succeeded`, `failed`, optional `cancelled`

`Source examples`:
- `mudler/LocalAI`
- `netbirdio/netbird`

Recommendation for `sageo-cli`:
- Standardize lifecycle enums for crawl/audit/report job flows:
  - `pending`
  - `running`
  - `completed`
  - `failed`
  - `cancelled` (if cancellation supported)

## 3) Structured machine errors (recommended)

Observed pattern:
- Stable machine-readable errors usually include at least:
  - `code`
  - `message`
- Some APIs also include metadata/timestamps/path

`Source examples`:
- `moby/moby` (`ErrorResponse{Message}`)
- `daytonaio/daytona` (`statusCode`, `message`, `code`, ...)

Recommendation for `sageo-cli` output envelope:
- Keep existing envelope, but add optional `code` in `error` payload for stable automation.
- Example codes:
  - `INVALID_TARGET_URL`
  - `FETCH_TIMEOUT`
  - `PROVIDER_NOT_FOUND`
  - `REPORT_WRITE_FAILED`

## 4) Timeout and cancellation handling (recommended)

Observed pattern:
- Explicit handling of context deadline in network paths:
  - `errors.Is(err, context.DeadlineExceeded)`

`Source examples`:
- `influxdata/telegraf`
- `elastic/beats`

Recommendation for `sageo-cli`:
- Normalize timeout/cancel errors to explicit error codes/messages.
- Ensure crawl/audit/report services consistently propagate context cancellation.

## 5) Deterministic output ordering (recommended)

Observed pattern:
- Sort map-derived lists before returning (`sort.Strings(names)`) to avoid flaky output/tests.

`Source examples`:
- `hashicorp/nomad`
- `ollama/ollama`

Recommendation for `sageo-cli`:
- Sort provider names from registry before returning from `Available()`.
- Keep deterministic ordering in report listing and any map-derived JSON data.

## 6) Current repo gap check (important)

Current state found:
- `internal/crawl`, `internal/audit`, `internal/report` contain real service logic.
- CLI command files under `internal/cli/commands/` still show scaffold labels/placeholders for:
  - `crawl`
  - `audit`
  - `report`

Recommendation:
- Wire real command handlers to the implemented services before expanding feature breadth.
- Update command help text to remove scaffold wording once wired.

## 7) Practical next-pass checklist

1. Wire `crawl run` command to `internal/crawl.NewService(...)`
2. Wire `audit run` command to `internal/audit.NewService()` using crawl output
3. Wire `report generate` / `report list` to `internal/report.NewService()`
4. Add error `code` field to envelope for machine-stable failures
5. Add deterministic sorting to provider registry output
6. Add contract tests for command JSON envelopes (success + failure)
7. Update `README.md` and `ARCHITECTURE.md` to reflect non-scaffold behavior

---

This file is intended as a short guardrail artifact to reduce phase-two drift and keep implementation aligned with established Go CLI patterns.
