# Plan: CLI UX self-documentation (doctor, next-steps, hard preflight, hint errors)

## Problem

The bayside audit failed silently because an agent ran `sageo run` without first running `sageo gsc sites use <property>`. The CLI gave no feedback, produced 20 unknown-tier recommendations, and the agent had no way to discover the gap short of reading the skill file.

Root cause: sageo relies on a skill file that agents may not read. Production CLIs (gh, flyctl, cloudquery, kubebuilder, homebrew, lakectl, tilt) solve this at the CLI layer with four patterns confirmed across many repos:

1. A `doctor` command that runs a health checklist.
2. "Next steps:" blocks printed after setup commands.
3. Pre-run gates that abort, not warn, when required config is missing.
4. Hint-formatted errors that include the fix command.

## Scope

- New: `sageo doctor` subcommand.
- New: Next-steps output on `init`, `run`, `auth login gsc`, `recommendations review`.
- Change: the existing `preflightGSCWarning` becomes a hard `preflightGSCCheck` that aborts.
- Change: `pkg/output` gains a `Hint` field on `ErrorPayload` and a `PrintCodedErrorWithHint` helper; retrofit the 4-5 highest-traffic error codes.
- Change: `sageo --help` and `sageo run --help` gain a `TYPICAL FLOW` block in `Long`.
- Docs: update SKILL.md, README.md, CHANGELOG.md.

Out of scope for this PR:
- A full `PersistentPreRunE` auth gate on the root command (gh's model). Too aggressive: many sageo subcommands deliberately work without auth. Revisit if warranted.
- Retrofitting every error call site with hints. Do the most-hit ones (~5); the rest becomes a follow-up.
- Interactive wizard on `sageo init`. Agent-driven runs need non-interactive mode by default.

## Evidence base (from `mcp__grep__searchGitHub`)

- `doctor` command: confirmed as the industry-standard name across `larksuite/cli`, `tilt-dev/tilt`, `treeverse/lakeFS`, `alda-lang/alda`, `go-vikunja/vikunja`, `entireio/cli` and originated with Homebrew + Flutter. Short descriptions cluster on "Run diagnostic checks" / "health check: config, auth, and connectivity".
- "Next steps:" block: confirmed in `cloudquery/cloudquery`, `kubernetes-sigs/kubebuilder`, `sphireinc/Foundry`, `dymensionxyz/roller`, `browseros-ai/BrowserOS`, `netbirdio/netbird`, `micro/go-micro`, `claraverse-space/ClaraVerse`. Format: blank line, `Next steps:`, numbered list of exact commands.
- Hard preflight: `cli/cli` root.go uses `PersistentPreRunE` + `IsAuthCheckEnabled` annotations to abort unauthed commands with a pointer at `gh auth login`. Per-command variant is what we'll adopt for `sageo run`.
- Hint-formatted errors: `vespa-engine/vespa` (`Error:` + cyan `Hint:`), `buildsafedev/bsf`, `cockroachdb/cockroach` (`cmd.Println("Hint:", hint)`), `minio/minio` (`HINT:` in rendered text).

## Architecture

### A. `sageo doctor`

New file `internal/cli/commands/doctor.go`. Top-level subcommand wired in `internal/cli/root.go`.

Types:

```go
type checkStatus string
const (
    checkPass checkStatus = "pass"
    checkWarn checkStatus = "warn"
    checkFail checkStatus = "fail"
)

type doctorCheck struct {
    Name    string      `json:"name"`
    Status  checkStatus `json:"status"`
    Message string      `json:"message"`
    Fix     string      `json:"fix,omitempty"`
}
```

Checks, run in order, each a small function that takes the loaded state + config + auth store and returns a `doctorCheck`:

1. `project_initialised` — `state.Exists(".")`. Fail if absent. Fix: `sageo init --url <site>`.
2. `brand_terms` — warn if empty. Fix: `sageo init --url <site> --brand "Name,alias"` or `sageo config set ...`.
3. `gsc_auth` — `auth.FileTokenStore.Status("gsc").Authenticated`. Fail if absent or expired-without-refresh-token. Fix: `sageo auth login gsc`.
4. `gsc_property` — `cfg.GSCProperty != ""`. Fail if gsc_auth passed but property empty. Fix: `sageo gsc sites list` then `sageo gsc sites use <property>`.
5. `psi_api_key` — warn if both `cfg.PSIAPIKey` and `SAGEO_PSI_API_KEY` env empty and no GSC token to fall back on. Fix: `sageo config set psi_api_key <key>`.
6. `llm_provider` — warn if selected provider's API key empty. Covers drafter only. Fix: `sageo config set anthropic_api_key <key>` or the OpenAI variant.
7. `dataforseo_creds` — warn if login or password empty. Fix: `sageo config set dataforseo_login <email>` and `dataforseo_password`.

Output:

- JSON envelope default: `data.checks[]`, `data.summary{pass,warn,fail}`.
- Text/table: coloured pass/warn/fail rows with the fix line indented below.
- Exit code: 0 if zero fails, 1 if any fail. Warnings never fail.

Files:
- `internal/cli/commands/doctor.go` (~220 LoC)
- `internal/cli/commands/doctor_test.go` (~200 LoC, table-driven per check)

### B. Next-steps output helper

New helper `internal/cli/commands/nextsteps.go`:

```go
// printNextSteps writes a "Next steps:" block to the given writer.
// Always writes to stderr so stdout stays parseable JSON.
func printNextSteps(w io.Writer, steps []string) {
    if len(steps) == 0 { return }
    fmt.Fprintln(w)
    fmt.Fprintln(w, "Next steps:")
    for i, s := range steps {
        fmt.Fprintf(w, "  %d. %s\n", i+1, s)
    }
}
```

Call sites:

- `init.go` RunE, on success: pass `cmd.ErrOrStderr()`, steps:
  1. `sageo auth login gsc`
  2. `sageo gsc sites use https://<your-site>/`
  3. `sageo run <your-site> --budget 10`
- `login.go` GSC success path: steps:
  1. `sageo gsc sites list`
  2. `sageo gsc sites use <property>`
  3. `sageo run <url> --budget 10`
- `run.go` RunE on success (non-dry-run, Outcome != "failed"): steps:
  1. `sageo recommendations review`
  2. `sageo report html --open`
- `recommendations_review.go` on completion: step:
  1. `sageo report html --open`

Rule: always stderr. Always unconditional of `--output` flag (cloudquery / kubebuilder pattern). Humans and agents both see it; agents that parse JSON from stdout are unaffected.

Files touched: `init.go`, `login.go`, `run.go`, `recommendations_review.go`. New: `nextsteps.go`, `nextsteps_test.go`.

### C. Hard preflight gate on `sageo run`

Replace `preflightGSCWarning` + its `preflightGSCWarningWithProperty` with:

```go
func preflightGSCCheck(skip, only map[string]bool, gscProperty string) error
```

- Returns `nil` when gsc is `--skip`ped, not in `--only`, or `gscProperty != ""`.
- Returns a typed `preflightError` otherwise. The error's message names the missing config, and its `Hint()` returns the three-line fix.

In `run.go` RunE, call it early, before stage execution:

```go
if err := preflightGSCCheck(toSet(skipFlag), toSet(onlyFlag), cfg.GSCProperty); err != nil {
    return output.PrintCodedErrorWithHint(
        "GSC_NOT_CONFIGURED",
        "GSC stage would run without an active property",
        err.Hint(),
        err, nil, output.Format(*format),
    )
}
```

Exit non-zero. User opts out via `--skip gsc`. The `preflightGSCWarning` function is deleted; the existing tests move to asserting the new abort behaviour. Warning-only behaviour stays for the config-load failure path (can't load config for some reason; don't block on that).

New error code: `GSC_NOT_CONFIGURED` in `pkg/output/errors.go`.

Files: `run.go`, `run_preflight_test.go`, `pkg/output/errors.go`.

### D. Hint field on error envelope

`pkg/output/output.go`:

```go
type ErrorPayload struct {
    Code    string `json:"code,omitempty"`
    Message string `json:"message"`
    Detail  string `json:"detail,omitempty"`
    Hint    string `json:"hint,omitempty"`
}

func PrintCodedErrorWithHint(code, message, hint string, err error, metadata map[string]any, format Format) error {
    // ... same as PrintCodedError plus Error.Hint = hint
    // In text/non-json render path, print "Error: ..." then "Hint: ..." on stderr.
}
```

`PrintCodedError` remains a thin wrapper that calls `PrintCodedErrorWithHint` with empty hint.

Retrofit the 5 highest-traffic error paths to pass a hint:

- `NO_PROJECT` (status.go, report.go, others): hint `sageo init --url <site>`.
- `GSC_NOT_CONFIGURED` (new, run.go): 3-line fix.
- `INVALID_URL` (init.go, run.go): hint `Use a full URL, for example https://example.com/`.
- `AUTH_REQUIRED` (auth.go, gsc.go callers): hint `sageo auth login <service>`.
- `CONFIG_LOAD_FAILED` (many): hint `sageo config list` to inspect, or re-init.

Leave the rest for follow-up.

Files: `pkg/output/output.go`, `pkg/output/errors.go`, `pkg/output/output_test.go`, and the 5 call sites above.

### E. `TYPICAL FLOW` block in `--help`

Extend `Long` on:

- Root (`internal/cli/root.go`):
  ```
  sageo is a command-line tool for SEO, GEO, and AEO operations.

  TYPICAL FLOW:
    1. sageo init --url https://example.com --brand "Example,example.com"
    2. sageo auth login gsc
    3. sageo gsc sites use https://example.com/   (MANDATORY before sageo run)
    4. sageo run https://example.com --budget 10
    5. sageo recommendations review
    6. sageo report html --open

  Run 'sageo doctor' to check your project is ready.
  ```

- `NewRunCmd` (`internal/cli/commands/run.go`): same flow, plus a "BEFORE YOU RUN" bullet listing the three prerequisites and mentioning `--skip gsc` as the explicit opt-out.

No code-logic change; just longer `Long` strings. Cobra renders them verbatim under `--help`.

Files: `internal/cli/root.go`, `internal/cli/commands/run.go`.

## Test strategy

1. `doctor_test.go` (new). Table-driven: set up a temp project dir, inject config + auth state, call the command, assert JSON `data.checks[]` contents and exit code. Cover: fresh project (many fails), fully configured (all pass), partial (mix).
2. `nextsteps_test.go` (new). Unit-test `printNextSteps` formatting (blank line, numbered list, skip on empty).
3. `run_preflight_test.go` (update). Existing tests become `preflightGSCCheck` with expected error / nil returns.
4. `init_test.go` / `run_test.go`: add assertions that stderr contains the `Next steps:` heading and the three expected commands on success.
5. `pkg/output/output_test.go`: extend to cover `PrintCodedErrorWithHint` JSON output and text rendering (`Hint: ...` line on stderr).

Quality gates: `go vet ./...`, `go test -race ./...`, `golangci-lint run`.

## Risks

- **Hard preflight may annoy users who explicitly don't want GSC.** Mitigation: `--skip gsc` opt-out is documented in the error hint. Test covers that path.
- **Next-steps on stderr may surprise JSON-only consumers.** Mitigation: stderr stays stderr; stdout JSON envelope is unchanged; documented in CHANGELOG.
- **Doctor check list will drift from reality.** Mitigation: each check is a small named function with a table-driven test; adding a new check is a 3-line addition. Doctor also lists itself in the root help as the escape hatch when the flow is unclear, so users self-service before opening issues.

## Steps

1. Add `Hint` field to `output.ErrorPayload`, add `PrintCodedErrorWithHint` helper, update `PrintCodedError` to be a thin wrapper, extend text-format path to print `Hint:` on stderr. Update `pkg/output/output_test.go`.
2. Add `GSC_NOT_CONFIGURED` error code in `pkg/output/errors.go`.
3. Create `internal/cli/commands/nextsteps.go` with `printNextSteps` helper. Add `nextsteps_test.go`.
4. Replace `preflightGSCWarning` in `internal/cli/commands/run.go` with a hard `preflightGSCCheck` that returns a typed error; update `run.go` RunE to call `PrintCodedErrorWithHint` with `GSC_NOT_CONFIGURED` and the 3-line fix hint. Update `run_preflight_test.go` to assert the abort behaviour.
5. Create `internal/cli/commands/doctor.go` implementing the 7-check doctor command. Wire it in `internal/cli/root.go`.
6. Create `internal/cli/commands/doctor_test.go` with table-driven coverage per check and an end-to-end "fresh project" / "fully configured" pair.
7. Add `printNextSteps` calls to `init.go` (init), `login.go` (gsc login success), `run.go` (on success), `recommendations_review.go` (on completion).
8. Extend `Long` on root command in `internal/cli/root.go` with TYPICAL FLOW block and a pointer at `sageo doctor`.
9. Extend `Long` on `NewRunCmd` in `internal/cli/commands/run.go` with BEFORE YOU RUN block and the `--skip gsc` opt-out note.
10. Retrofit hints on the 5 chosen error paths: `NO_PROJECT`, `INVALID_URL`, `AUTH_REQUIRED`, `CONFIG_LOAD_FAILED`, `GSC_NOT_CONFIGURED` (already done in step 4). Grep for each error code and pass a hint in every call site.
11. Update `.claude/skills/sageo/SKILL.md` recommended-flow block: add a "Run `sageo doctor` if any step is unclear" line and note the hard preflight on run.
12. Update `.claude/skills/sageo/commands.md`: add the doctor row; update the run row to note the hard preflight.
13. Update `README.md`: add doctor to the command list; add a sentence to the client-audit example about running doctor first.
14. Update `ARCHITECTURE.md`: brief paragraph on the doctor checks and the preflight gate in the recommendation-lifecycle or a new "CLI UX" section.
15. Update `CHANGELOG.md` Unreleased section with Added (doctor, next-steps, hint errors) and Changed (hard preflight, extended help) entries.
16. Run `go vet ./...`, `go test -race ./...`, `golangci-lint run`. Fix any issues.
17. Commit with grouped subject + body, push, tag next patch version, publish GitHub release with concise notes.
