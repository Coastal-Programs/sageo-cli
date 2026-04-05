# Phase 2 Research Recommendations — Implementation Plan

## Analysis

### Issues Found

**Scaffold remnants:**
- `internal/cli/root.go` lines 29-32: Short/Long descriptions say "scaffold"
- `internal/cli/commands/config.go`: All 4 config subcommands emit `"scaffold": true` in metadata
- `internal/cli/commands/version.go` line 22: Emits `"scaffold": true` in metadata

**Missing error codes (§3):**
- `ErrorPayload` has no `Code` field for machine-readable error classification
- All `PrintErrorResponse` calls pass no error code
- Need new `PrintCodedError` or add `code` param to existing helper

**No timeout/cancel normalization (§4):**
- `internal/provider/local/local.go` wraps errors generically — no context.DeadlineExceeded / context.Canceled detection
- `internal/crawl/crawler.go` doesn't check ctx cancellation in the main loop

**Non-deterministic provider output (§5):**
- `provider.Available()` iterates a map — output order is random
- Already noted in research doc, sort needed

**Report listing already sorted** (by CreatedAt desc) — OK.

**Missing contract tests (§6):**
- `pkg/output/output_test.go` tests exist but don't verify the `code` field or full envelope shape
- No command-level integration tests that verify JSON success/error envelopes

### Approach

- Add `Code` field to `ErrorPayload`
- Add `PrintCodedError` helper (accepts code string) — keep backward compat by making existing `PrintErrorResponse` pass empty code
- Create error code constants in `pkg/output/errors.go`
- Normalize context errors in local fetcher and crawler
- Sort `provider.Available()` output
- Strip all `"scaffold": true` metadata from commands
- Update root.go Short/Long text
- Add contract tests for JSON envelope shape (success + error with code)
- Update docs

## Steps

1. Add `Code` field to `ErrorPayload` in `pkg/output/output.go` and update `PrintErrorResponse` to accept an optional code string parameter
2. Create error code constants (`INVALID_URL`, `CONFIG_LOAD_FAILED`, `PROVIDER_NOT_FOUND`, `CRAWL_FAILED`, `AUDIT_FAILED`, `REPORT_WRITE_FAILED`, `FETCH_TIMEOUT`, `CANCELLED`, `REPORT_LIST_FAILED`) in `pkg/output/errors.go`
3. Normalize timeout and cancel errors in `internal/provider/local/local.go` Fetch method by detecting `context.DeadlineExceeded` and `context.Canceled` and returning structured error messages
4. Add context cancellation check in `internal/crawl/crawler.go` main loop and fetch goroutines so crawl aborts promptly on cancel/timeout
5. Sort the slice returned by `provider.Available()` in `internal/provider/provider.go` for deterministic output
6. Update all `PrintErrorResponse` calls in `internal/cli/commands/crawl.go`, `audit.go`, `report.go`, `provider.go` to pass appropriate error codes
7. Remove `"scaffold": true` from metadata in `internal/cli/commands/config.go` (all 4 subcommands) and `internal/cli/commands/version.go`
8. Update root command Short/Long text in `internal/cli/root.go` to remove scaffold language
9. Add contract tests in `pkg/output/output_test.go` verifying full JSON envelope shape for success (has `success`, `data`, `metadata` keys) and error (has `success`, `error.message`, `error.code` keys) responses
10. Update `README.md` to document error codes in the output contract section
11. Update `ARCHITECTURE.md` to document error codes in the output contract and remove any scaffold language
12. Update `CHANGELOG.md` to add a `[0.2.1]` entry listing the polish changes
13. Run `go test ./...` and `go vet ./...` to verify everything passes
