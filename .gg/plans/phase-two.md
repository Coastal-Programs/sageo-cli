# Phase Two: Real Crawl, Audit, and Report Services

## Analysis

### Current State
- Scaffold CLI with Cobra commands that return "not_implemented" envelopes
- Domain packages (`internal/crawl`, `internal/audit`, `internal/report`) have interfaces and types but no implementations
- Config package fully functional with provider/key/url/org fields and env overrides
- Output package has JSON envelope contract (`success`/`data`/`error`/`metadata`)
- Only dependency beyond stdlib is `github.com/spf13/cobra`

### Design Decisions

**New dependency**: `golang.org/x/net/html` for robust HTML tokenization (quasi-stdlib, widely used for Go crawlers). All other logic uses stdlib.

**Provider abstraction**: A `provider.Fetcher` interface abstracts HTTP fetching so the crawl service doesn't depend on `net/http` directly. A "local" provider (built-in `net/http` client) is the default. The provider registry reads `active_provider` from config. This is extensible for future remote/API providers.

**Crawl service**: BFS crawler with depth limit, max-pages cap, same-domain scoping, concurrency control via a semaphore, and visited-URL deduplication. Each crawled page produces a `PageResult` with status code, title, meta description, headings, links, and images extracted via HTML parsing.

**Audit service**: Takes crawl results and runs a fixed set of SEO checks against each page:
- Title: missing, empty, too long (>60 chars)
- Meta description: missing, empty, too long (>160 chars)
- H1: missing, multiple H1s
- Images without alt text
- Broken internal links (4xx/5xx status codes)
- Missing canonical tag

Each issue has a severity (error/warning/info) and produces a per-page + aggregate score.

**Report service**: Generates a summary report from audit results. Supports JSON output (default) and stores reports to a configurable directory (`~/.config/sageo/reports/`). The `list` subcommand reads stored reports.

**Provider command**: `provider list` shows available providers, `provider use <name>` sets `active_provider` in config.

### Files Changed/Created

#### New files
- `internal/provider/provider.go` — Fetcher interface + registry
- `internal/provider/local/local.go` — net/http-based fetcher
- `internal/provider/local/local_test.go` — tests with httptest
- `internal/crawl/crawler.go` — BFS crawler implementation
- `internal/crawl/crawler_test.go` — crawler tests with httptest server
- `internal/crawl/page.go` — HTML parsing / page data extraction
- `internal/crawl/page_test.go` — parser unit tests
- `internal/audit/checker.go` — SEO check functions
- `internal/audit/engine.go` — audit engine that runs checks on crawl results
- `internal/audit/engine_test.go` — audit engine tests
- `internal/report/generator.go` — report generation + file storage
- `internal/report/generator_test.go` — report tests

#### Modified files
- `internal/crawl/types.go` — expand with PageResult, Link, Image, Heading types
- `internal/crawl/service.go` — update Service interface, add constructor
- `internal/audit/types.go` — expand with Issue, CheckResult, Severity types
- `internal/audit/service.go` — update Service interface, add constructor
- `internal/report/service.go` — expand types, update Service interface, add constructor
- `internal/cli/commands/crawl.go` — wire real crawl service
- `internal/cli/commands/audit.go` — wire real audit service
- `internal/cli/commands/report.go` — wire real report service
- `internal/cli/commands/provider.go` — wire provider list/use
- `internal/cli/commands/config.go` — remove `newScaffoldCommand` helper (no longer needed)
- `internal/cli/root.go` — no structural changes needed (commands already registered)
- `go.mod` / `go.sum` — add `golang.org/x/net`
- `README.md` — update to reflect shipped behavior
- `ARCHITECTURE.md` — update package descriptions and remove "scaffold" language
- `CHANGELOG.md` — add 0.2.0 entry
- `CLAUDE.md` — update scope section

### Risks
- `golang.org/x/net/html` parser may need careful handling of malformed HTML — mitigate with recover/timeout
- Crawl depth + max pages needs sensible defaults to avoid runaway crawls (default: depth=2, maxPages=50)
- File I/O for report storage needs proper error handling for permissions
- Tests using httptest servers must be careful about port conflicts in parallel runs (httptest handles this)

## Steps

1. Add `golang.org/x/net` dependency to go.mod via `go get golang.org/x/net`
2. Create `internal/provider/provider.go` with `Fetcher` interface (method `Fetch(ctx, url) → (statusCode int, body []byte, headers http.Header, err error)`), `Registry` map, and `NewFetcher(providerName string) (Fetcher, error)` constructor
3. Create `internal/provider/local/local.go` implementing the `Fetcher` interface using `net/http.Client` with configurable timeout, User-Agent header, and redirect policy
4. Create `internal/provider/local/local_test.go` testing the local fetcher against an `httptest.Server`
5. Rewrite `internal/crawl/types.go` expanding `Request` (add MaxPages, UserAgent fields) and `Result` (add Pages []PageResult, Errors []CrawlError), and adding `PageResult` (URL, StatusCode, Title, MetaDescription, Canonical, Headings []Heading, Links []Link, Images []Image), `Link`, `Image`, `Heading`, `CrawlError` structs
6. Create `internal/crawl/page.go` with an `extractPageData(url string, statusCode int, body []byte) PageResult` function that parses HTML using `golang.org/x/net/html` tokenizer to extract title, meta description, canonical, headings, links, and images
7. Create `internal/crawl/page_test.go` with unit tests for HTML extraction covering normal pages, missing elements, and malformed HTML
8. Rewrite `internal/crawl/service.go` to keep the `Service` interface (updated `Run` signature returning expanded `Result`) and add a `NewService(fetcher provider.Fetcher) Service` constructor
9. Create `internal/crawl/crawler.go` implementing the `Service` interface with BFS crawl logic: URL queue, visited set, same-domain check, depth tracking, concurrency semaphore (5 workers), and max-pages cap
10. Create `internal/crawl/crawler_test.go` with integration tests using `httptest.Server` serving a small site tree, testing depth limits, max pages, same-domain filtering, and error handling
11. Rewrite `internal/audit/types.go` adding `Severity` type (Error/Warning/Info constants), `Issue` struct (Rule, Severity, URL, Message, Detail), expanding `Result` (add Issues []Issue, PageCount int, IssueCount map[Severity]int), and updating `Request` to accept crawl results directly
12. Create `internal/audit/checker.go` with individual check functions: `checkTitle`, `checkMetaDescription`, `checkH1`, `checkImageAlt`, `checkCanonical`, `checkStatusCode` — each takes a `crawl.PageResult` and returns `[]Issue`
13. Create `internal/audit/engine.go` implementing the `Service` interface: iterates crawl page results, runs all checkers, aggregates issues, computes a 0–100 score (100 minus weighted deductions per issue severity)
14. Create `internal/audit/engine_test.go` testing the audit engine with synthetic `crawl.PageResult` data covering each checker rule
15. Rewrite `internal/report/service.go` updating `Request` (accept audit `Result` + output dir), `Result` (add FilePath, Summary map), and the `Service` interface (add `List` method), plus a `NewService() Service` constructor
16. Create `internal/report/generator.go` implementing report `Service`: `Generate` writes a JSON report file to the reports directory (`~/.config/sageo/reports/<timestamp>.json`), `List` reads and returns stored report metadata
17. Create `internal/report/generator_test.go` testing report generation and listing with a temp directory
18. Rewrite `internal/cli/commands/crawl.go` replacing scaffold stubs: `crawl run --url <url> [--depth N] [--max-pages N]` loads config, creates provider fetcher, runs crawl service, outputs result envelope; `crawl status` removed (synchronous operation)
19. Rewrite `internal/cli/commands/audit.go` replacing scaffold stubs: `audit run --url <url> [--depth N] [--max-pages N]` runs crawl then audit, outputs result envelope; `audit status` removed
20. Rewrite `internal/cli/commands/report.go` replacing scaffold stubs: `report generate --url <url> [--depth N]` runs crawl→audit→report pipeline, outputs envelope; `report list` reads stored reports
21. Rewrite `internal/cli/commands/provider.go`: `provider list` returns available providers with active marker from config; `provider use <name>` validates and sets `active_provider` in config
22. Remove the `newScaffoldCommand` helper from `internal/cli/commands/config.go` (no longer used by any command)
23. Update `internal/cli/root_test.go` to verify subcommand structure still matches (crawl/audit/report may have different subcommands now)
24. Run `go test ./...` and `go vet ./...` to verify all tests pass and code is clean
25. Update `README.md` to document shipped crawl/audit/report/provider commands with usage examples
26. Update `ARCHITECTURE.md` to describe real service implementations, provider abstraction, and data flow
27. Update `CHANGELOG.md` with a `[0.2.0]` entry listing all phase-two additions
28. Update `CLAUDE.md` scope section to reflect phase-two completion
29. Run `make build` to verify the binary compiles, then run final `go test -race ./...` to confirm everything passes
