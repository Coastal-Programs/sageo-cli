# Wiring Trace: CLI UX (doctor, next-steps, hint errors, GSC preflight)

Date: 2026-04-23
Scope: The changes landed in commit `85ea261` — doctor command, next-steps output, error.hint field, hard GSC preflight, retrofitted error hints.

## Architecture Map

Entry Points
- `cmd/sageo/main.go` → `internal/cli/root.go` (registers commands)
- Commands touched: `init`, `auth login gsc`, `login` (TUI), `run`, `recommendations review`, `doctor`, plus 30+ other error sites

Orchestration
- `pkg/output` — envelope + hint rendering
- `internal/cli/commands/nextsteps.go` — printNextSteps helper
- `internal/cli/commands/run.go` — preflightGSCCheck

Modules
- `internal/cli/commands/doctor.go` — the 7 doctor checks
- `internal/cli/commands/auth.go` — loginGSC (shared OAuth flow)
- `internal/cli/commands/login.go` — interactive TUI wrapper

External
- `os.Stderr` (next-steps, hint text), `os.Stdout` (JSON envelope)

## Paths Traced

### Path 1: `sageo login` (TUI) → loginGSC → next-steps
- Entry: `internal/cli/commands/login.go:105 runLogin` → `runGSCLoginForm` (line ~242)
- Through: `loginGSC` in `auth.go:105` (does OAuth, prints PrintSuccess)
- Consumed: `login.go:284` after loginGSC returns, `printNextSteps(os.Stderr, ...)` fires
- Status: OK

### Path 2: `sageo auth login gsc` (direct CLI) → loginGSC
- Entry: `auth.go:46 newAuthLoginCmd` → `case "gsc": return loginGSC(...)`
- Through: `loginGSC` finishes at `auth.go:221 PrintSuccess`
- Consumed: command returns. **No printNextSteps is ever called.**
- Status: **GAP FOUND** — next-steps is wired in the TUI wrapper only, not the shared OAuth function

### Path 3: `sageo doctor` (JSON mode) → exit code
- Entry: `doctor.go:62 RunE` → `PrintSuccess(data, nil, FormatJSON)` emits `{success: true, data: {...}}`
- Through: summary check `if summary["fail"] > 0`
- Consumed: `return fmt.Errorf("doctor: %d check(s) failed", ...)` → Cobra sets exit=1
- Status: **GAP FOUND** — JSON envelope says `success: true` but process exits non-zero; shape mismatch for agent consumers

### Path 4: `sageo run` preflight → error envelope
- Entry: `run.go:120 preflightGSCCheck` returns `*preflightError` with Hint()
- Through: `PrintCodedErrorWithHint(ErrGSCNotConfigured, msg, hint, pfErr, ...)`
- Consumed: JSON envelope has `error.code=GSC_NOT_CONFIGURED`, `error.hint="Run: ..."`
- Status: OK

### Path 5: Every NO_PROJECT / INVALID_URL / AUTH_REQUIRED / CONFIG_LOAD_FAILED error site → hint
- All 33 CONFIG_LOAD_FAILED sites carry identical hint
- All 7 NO_PROJECT sites carry `sageo init --url <site>` hint
- All 2 INVALID_URL sites in init/run carry example-URL hint
- All 6 AUTH_REQUIRED sites in gsc/opportunities/auth carry `sageo auth login gsc` hint
- Status: OK

### Path 6: doctorInputs → checks
- `gatherDoctorInputs` assigns `WorkDir`, `ConfigLoadErr`, `GSCStatusErr`
- `runDoctorChecks` + its callees never read any of those three fields
- Status: **GAP FOUND** — dead state assignment; no functional failure, but the fields are misleading

## Gaps Found

| # | Where | What's Lost | Impact | Severity |
|---|-------|-------------|--------|----------|
| 1 | `auth.go:50` → `auth.go:221` | `printNextSteps` never called after direct `sageo auth login gsc` | Agents using the documented command get no guidance on what to run next (the whole point of the feature) | High |
| 2 | `doctor.go:71` → `doctor.go:83` | JSON `{success: true}` paired with exit 1 | JSON consumers that check `.success` vs exit code see a contradiction | Medium |
| 3 | `doctor.go:36,40,42` | Unused fields on `doctorInputs` | Dead code; no runtime effect | Low |

### Gap 1: Next-steps bypassed on `sageo auth login gsc`
**Trace**: `auth.go:50 case "gsc": return loginGSC(format, verbose)` → `loginGSC` prints `PrintSuccess` and returns → caller returns.
**What happens**: An agent running the documented command `sageo auth login gsc` gets the auth success envelope but no "Next steps" block. Only the `sageo login` TUI wrapper (`login.go:284`) wires next-steps. The capability works on path A (TUI) but not on path B (direct CLI).
**Fix**: Move the `printNextSteps` call into `loginGSC` itself (after OAuth succeeds, before the final `PrintSuccess`). Remove the duplicate call from `runGSCLoginForm` so both entry points emit exactly one next-steps block.

### Gap 2: doctor JSON shape inconsistent with exit code
**Trace**: `doctor.go:71 PrintSuccess(data, nil, FormatJSON)` → envelope has `success: true` → `doctor.go:83 return fmt.Errorf(...)` → Cobra exits 1.
**What happens**: A CI pipeline or agent that relies on `jq '.success'` will see `true` but the process exited with `$? = 1`. Either the exit code is a lie or the envelope is. Contract says the envelope is the contract.
**Fix**: When `summary.fail > 0` in JSON mode, emit an error envelope (`PrintCodedErrorWithHint("DOCTOR_CHECKS_FAILED", ...)`) with `data.checks[]` + `data.summary` moved into `metadata`, and then return the error. Envelope `success: false` now matches exit 1.

### Gap 3: Dead fields on doctorInputs
**Trace**: `doctor.go:33-44 doctorInputs` defines `WorkDir`, `ConfigLoadErr`, `GSCStatusErr`; assigned in `gatherDoctorInputs`; never read.
**What happens**: Nothing at runtime; adds noise to the type and tempts future authors to read values that are never populated in tests.
**Fix**: Delete the three fields. Drop the matching assignments in `gatherDoctorInputs`.

## Systemic Patterns

- **Shared function vs wrapper asymmetry.** Gap 1 is an instance of: when an enhancement (next-steps) is bolted onto the wrapper but not the shared function, every direct-entry path silently misses the feature. Putting the behaviour in the shared function once is the fix.

## Summary

- Paths traced: 6
- Gaps found: 3 (Critical: 0, High: 1, Medium: 1, Low: 1)
- Systemic patterns: 1

All three gaps are being fixed in the same session.
