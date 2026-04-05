---
name: fix
description: Run typechecking and linting, then spawn parallel agents to fix all issues
---

Run all linting and typechecking tools for this Go project, collect errors, group them by domain, and use the subagent tool to spawn parallel sub-agents to fix them.

## Step 1: Run Checks

Run these commands and capture their full output (including errors). Do not stop if a command fails — collect all output first.

```bash
# Format check (reports files that would change)
gofmt -l ./...

# Vet (type and correctness checks)
go vet ./...

# Lint (comprehensive static analysis — uses golangci-lint if installed, falls back to go vet)
golangci-lint run ./... 2>&1 || echo "golangci-lint not installed, skipping"

# Build check (catches compile errors)
go build ./...

# Test (catches runtime failures)
go test -race ./...
```

## Step 2: Collect and Group Errors

Parse the output from all commands above. Group errors into these domains:

- **Build errors**: Compile failures from `go build`
- **Type/vet errors**: Issues from `go vet` (e.g. incorrect format strings, unreachable code, bad struct tags)
- **Lint errors**: Issues from `golangci-lint` (e.g. unused variables, inefficient code, style violations)
- **Format errors**: Files listed by `gofmt -l` that need formatting
- **Test failures**: Failing tests from `go test`

## Step 3: Spawn Parallel Agents

For each domain that has issues, use the subagent tool to spawn a sub-agent to fix all errors in that domain. Include the full error output in each agent's task so it knows exactly what to fix.

- **Format errors**: Agent should run `gofmt -w` on the listed files.
- **Build errors**: Agent should read the failing files and fix the compile errors.
- **Type/vet errors**: Agent should read each file flagged by `go vet` and fix the reported issues.
- **Lint errors**: Agent should read each file flagged by `golangci-lint` and fix the reported issues.
- **Test failures**: Agent should read the failing test files and the source files they test, then fix the failures.

## Step 4: Verify

After all agents complete, re-run all checks to verify all issues are resolved:

```bash
gofmt -l ./...
go vet ./...
golangci-lint run ./... 2>&1 || true
go build ./...
go test -race ./...
```

If any issues remain, fix them directly.
