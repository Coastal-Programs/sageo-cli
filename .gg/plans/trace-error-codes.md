# Wiring Trace: Machine-Readable Error Codes

Date: 2026-04-05
Scope: Full project

## Architecture Map

```
[Entry Points]          → [Orchestration]     → [Modules]              → [External]
internal/cli/commands/    pkg/output/           internal/crawl/           net/http
  crawl.go                  output.go           internal/audit/
  audit.go                  errors.go           internal/report/
  report.go               internal/cli/         internal/provider/local/
  provider.go               root.go             internal/common/config/
  config.go
  version.go
```

## Paths Traced

### Path 1: crawl run → PrintCodedError → JSON envelope
- Entry: `crawl.go:38,43,48,64` — 4 error paths
- Through: `output.go:46` — `PrintCodedError`
- Destination: stdout/stderr as JSON with `code` field
- Status: **OK** — all paths use coded errors, context errors normalized at :58-63

### Path 2: audit run → PrintCodedError → JSON envelope
- Entry: `audit.go:39,44,49,65,73` — 5 error paths
- Through: `output.go:46` — `PrintCodedError`
- Destination: stdout/stderr as JSON with `code` field
- Status: **OK** — all paths use coded errors, context errors normalized at :60-64

### Path 3: report generate → PrintCodedError → JSON envelope
- Entry: `report.go:43,48,53,70,79,88` — 6 error paths
- Through: `output.go:46` — `PrintCodedError`
- Destination: stdout/stderr as JSON with `code` field
- Status: **OK** — all paths use coded errors, context errors normalized at :65-69

### Path 4: report list → PrintCodedError → JSON envelope
- Entry: `report.go:112` — 1 error path
- Through: `output.go:46` — `PrintCodedError`
- Destination: stdout/stderr as JSON with `code` field
- Status: **OK**

### Path 5: provider list → PrintCodedError → JSON envelope
- Entry: `provider.go:32` — 1 error path
- Through: `output.go:46` — `PrintCodedError`
- Destination: stdout/stderr as JSON with `code` field
- Status: **OK**

### Path 6: provider use → PrintCodedError → JSON envelope
- Entry: `provider.go:67,72,77` — 3 error paths
- Through: `output.go:46` — `PrintCodedError`
- Destination: stdout/stderr as JSON with `code` field
- Status: **OK** — save error now correctly uses `ErrConfigSaveFailed`

### Path 7: config show/get/set → PrintCodedError → JSON envelope
- Entry: `config.go:36,54,59,80,84,88` — 6 error paths
- Through: `output.go:46` — `PrintCodedError`
- Destination: stdout/stderr as JSON with `code` field
- Status: **OK** (FIXED during trace — was bare `return err`)

### Path 8: config path → no error paths
- Status: **OK** — only returns success

### Path 9: version → no error paths
- Status: **OK** — only returns success

### Path 10: root Execute fallback
- Entry: `root.go:20` — `PrintError(err.Error(), nil)`
- Destination: stderr as simple JSON `{"error":..., "detail":...}`
- Status: **OK** — intentional fallback for Cobra-level errors (unknown command, etc.)

### Path 11: local fetcher context normalization
- Entry: `local/local.go:72-77` — checks `context.DeadlineExceeded` and `context.Canceled`
- Through: crawl error accumulation → command handler context check
- Status: **OK**

### Path 12: scaffold metadata removal
- Checked: all `PrintSuccess` calls in config.go, version.go
- Status: **OK** — no `"scaffold": true` in any output metadata

## Gaps Found (during trace, all fixed)

| # | Where | What Was Lost | Impact | Severity | Status |
|---|-------|-------------|--------|----------|--------|
| 1 | config.go:36,54,59,80,84,88 | 6 bare `return err` bypassed error envelope | Config errors returned as unstructured Cobra errors, not JSON envelope | Critical | **FIXED** |
| 2 | provider.go:77 | Save error used `ErrConfigLoadFailed` | Wrong error code for save failures | Medium | **FIXED** |
| 3 | errors.go | Missing `CONFIG_SAVE_FAILED`, `CONFIG_GET_FAILED` | No code for config set/get failures | High | **FIXED** |

## Systemic Patterns

- None remaining. All command error paths now flow through `PrintCodedError` with appropriate codes.

## Summary
- Paths traced: 12
- Gaps found: 3 (Critical: 1, High: 1, Medium: 1) — **all fixed during trace**
- Systemic patterns: 0
- All error code constants are used at least once
- No `PrintErrorResponse` calls remain in commands
- No bare `return err` calls remain in commands
- Tests pass, vet clean
