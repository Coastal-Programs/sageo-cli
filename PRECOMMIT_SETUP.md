# Pre-commit + CI Guardrails for `sageo-cli`

## Goal
Prevent low-quality or broken changes from reaching a public repository by enforcing checks both locally and in GitHub.

## Recommended setup

### 1) Local developer guardrails (pre-commit)
Use the `pre-commit` framework with Go-focused checks:

- `go fmt ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`

This catches issues before code is committed or pushed.

### 2) GitHub CI as hard gate
Ensure CI runs on pull requests to `main` and includes:

- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint`

### 3) Branch protection (GitHub settings)
Enable rules for `main`:

- Require pull request before merging
- Require status checks to pass
- Restrict direct pushes to `main`

## Deliverables to add in repo

1. `.pre-commit-config.yaml` with local hooks for Go checks
2. `README.md` section: how to install and run pre-commit
3. (Optional) `make precommit` target that runs the same checks
4. Confirm CI workflow names used in branch protection

## Suggested pre-commit content (example)

```yaml
repos:
  - repo: local
    hooks:
      - id: go-fmt
        name: go fmt
        entry: go fmt ./...
        language: system
        pass_filenames: false
      - id: go-vet
        name: go vet
        entry: go vet ./...
        language: system
        pass_filenames: false
      - id: go-test
        name: go test
        entry: go test ./...
        language: system
        pass_filenames: false
  - repo: https://github.com/golangci/golangci-lint
    rev: v2.11.3
    hooks:
      - id: golangci-lint
        args: [--timeout=5m]
```

## Validation checklist

- `pre-commit run --all-files` passes locally
- `go test ./...` passes
- `go vet ./...` passes
- CI passes on PR
- Branch protection enabled with required checks
