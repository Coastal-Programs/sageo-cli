# Sageo CLI — Agent Notes

## Scope

The CLI has working crawl, audit, report, provider, auth, GSC, SERP, AEO, GEO, and opportunities services.

### Implemented
- BFS website crawler with depth/page limits and concurrent fetching
- SEO audit engine with rule-based checks and scoring
- JSON report generation and storage
- Provider abstraction with built-in `local` HTTP fetcher
- Full crawl → audit → report pipeline
- Google Search Console integration (sites list, query pages/keywords, opportunity seeds)
- OAuth2 authentication flow for GSC with token persistence
- SERP analysis adapters (SerpAPI and DataForSEO)
- AEO and GEO command groups backed by DataForSEO
- Labs command group is planned next for DataForSEO Labs endpoints (`ranked-keywords`, `keywords`, `overview`, `competitors`, `keyword-ideas`)
- Cost-aware execution contracts (`estimated_cost`, `requires_approval`, `cached`, `source`, `fetched_at`)
- `--dry-run` support for paid workflows
- File-based response caching with TTL
- Approval gate blocking execution when estimated cost exceeds threshold
- Opportunity detection merging GSC + optional SERP evidence

### Do now
- Keep command architecture stable
- Maintain JSON-first output contract
- Preserve config and output consistency
- Extend audit rules as needed
- Add new providers via the registry pattern
- Add new SERP providers behind the `serp.Provider` interface

### Do not do without explicit instructions
- Add multiple paid SEO providers at once
- Add backlink/domain analytics suites prematurely
- Embed OpenAI/Anthropic inside the CLI by default
- Change the output envelope contract incompatibly
- Restructure the command hierarchy unnecessarily

## Conventions

- Language: Go
- CLI framework: Cobra
- Entry point: `cmd/sageo/main.go`
- Root command wiring: `internal/cli/root.go`
- Command files: `internal/cli/commands/*.go` (planned addition: `labs.go`)
- Config package: `internal/common/config`
- Cost package: `internal/common/cost`
- Cache package: `internal/common/cache`
- Output package: `pkg/output`
- Provider package: `internal/provider`
- Auth package: `internal/auth`
- GSC package: `internal/gsc`
- SERP package: `internal/serp` (adapters: `internal/serp/serpapi`, `internal/serp/dataforseo`)
- DataForSEO shared client: `internal/dataforseo`
- Opportunities package: `internal/opportunities`
- Domain packages: `internal/crawl`, `internal/audit`, `internal/report`

## Output Contract

Prefer envelope-style structured output:
- `success`
- `data`
- `error`
- `metadata`

Default command output should remain `json` for automation and agent usage.

Paid commands include additional metadata keys:
- `estimated_cost`, `currency`, `requires_approval`, `cached`, `source`, `fetched_at`, `dry_run`

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
- `SAGEO_SERP_PROVIDER`
- `SAGEO_SERP_API_KEY`
- `SAGEO_DATAFORSEO_LOGIN`
- `SAGEO_DATAFORSEO_PASSWORD`
- `SAGEO_APPROVAL_THRESHOLD_USD`
- `SAGEO_GSC_PROPERTY`
- `SAGEO_GSC_CLIENT_ID`
- `SAGEO_GSC_CLIENT_SECRET`

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
