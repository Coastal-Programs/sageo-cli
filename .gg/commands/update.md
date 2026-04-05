---
name: update
description: Update dependencies, fix deprecations and warnings
---

## Step 1: Check for Updates

List all dependencies and check which ones have newer versions available:

```bash
go list -m -u all
```

Review the output — modules with `[v...]` annotations have updates available.

## Step 2: Update Dependencies

Update all direct and indirect dependencies to their latest compatible versions:

```bash
go get -u ./...
go mod tidy
go mod verify
```

Then check for known vulnerabilities:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
govulncheck ./...
```

If vulnerabilities are found, update the affected modules to patched versions.

## Step 3: Check for Deprecations & Warnings

Run a clean build and read ALL output carefully:

```bash
go build ./...
go vet ./...
```

Look for:
- Deprecation warnings from the compiler or vet
- Deprecated API usage flagged by `staticcheck` or `golangci-lint`
- Breaking changes in updated dependencies
- Any build errors from incompatible updates

Also run the linter to catch deprecated API usage:

```bash
make lint
```

## Step 4: Fix Issues

For each warning, deprecation, or error:
1. Research the recommended replacement or fix (check the dependency's changelog/release notes)
2. Update code to use the new API
3. Re-run `go build ./...` and `go vet ./...`
4. Verify no warnings remain

## Step 5: Run Quality Checks

Run the full quality gate:

```bash
make fmt
make vet
make test
make lint
```

Fix all errors before completing.

## Step 6: Verify Clean Build

Clear the module cache and do a fresh resolution to confirm everything is clean:

```bash
go clean -cache
go mod tidy
go mod verify
go build ./...
go test ./...
```

Verify ZERO warnings or errors in the output.
