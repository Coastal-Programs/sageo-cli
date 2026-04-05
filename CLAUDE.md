# Sageo CLI — Agent Notes

## Scope (Current Phase)

Phase two is complete. The CLI has working crawl, audit, report, and provider services.

### Implemented
- BFS website crawler with depth/page limits and concurrent fetching
- SEO audit engine with rule-based checks and scoring
- JSON report generation and storage
- Provider abstraction with built-in `local` HTTP fetcher
- Full crawl → audit → report pipeline

### Do now
- Keep command architecture stable
- Maintain JSON-first output contract
- Preserve config and output consistency
- Extend audit rules as needed
- Add new providers via the registry pattern

### Do not do without explicit instructions
- Add external API provider integrations
- Change the output envelope contract
- Restructure the command hierarchy

## Conventions

- Language: Go
- CLI framework: Cobra
- Entry point: `cmd/sageo/main.go`
- Root command wiring: `internal/cli/root.go`
- Command files: `internal/cli/commands/*.go`
- Config package: `internal/common/config`
- Output package: `pkg/output`
- Provider package: `internal/provider`
- Domain packages: `internal/crawl`, `internal/audit`, `internal/report`

## Output Contract

Prefer envelope-style structured output:
- `success`
- `data`
- `error`
- `metadata`

Default command output should remain `json` for automation and agent usage.

## Configuration

Default config path:
- `~/.config/sageo/config.json`

Optional override:
- `SAGEO_CONFIG` (absolute `.json` path)

Supported env overrides:
- `SAGEO_PROVIDER`
- `SAGEO_API_KEY`
- `SAGEO_BASE_URL`
- `SAGEO_ORGANIZATION_ID`

## Validation Commands

Run after changes:

```bash
go test ./...
go vet ./...
```

For full quality gate:

```bash
make fmt
make vet
make test
make lint
```
