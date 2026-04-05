# sageo-cli

Sageo CLI is a Go + Cobra command-line tool for SEO crawling, auditing, and reporting.

## Features

- **Crawl** — BFS website crawler with depth/page limits and concurrent fetching
- **Audit** — SEO checks for titles, meta descriptions, headings, images, canonicals, and status codes
- **Report** — Generate and store JSON audit reports locally
- **Provider** — Pluggable HTTP fetcher abstraction (ships with `local` provider using `net/http`)
- **Config** — Persistent local configuration with env overrides

## Install / Build

```bash
go mod tidy
make build
```

Binary output: `./build/sageo`

## Usage

### Crawl a website

```bash
sageo crawl run --url https://example.com --depth 2 --max-pages 50
```

### Run an SEO audit

```bash
sageo audit run --url https://example.com --depth 2 --max-pages 50
```

### Generate a report

```bash
sageo report generate --url https://example.com --depth 2
```

### List stored reports

```bash
sageo report list
```

### Manage providers

```bash
sageo provider list
sageo provider use local
```

### Configuration

```bash
sageo config path
sageo config show
sageo config get active_provider
sageo config set active_provider local
```

## Global Flags

- `--output, -o` — output format: `json` (default), `text`, `table`
- `--verbose, -v` — enable verbose output

## Config

Default config path: `~/.config/sageo/config.json`

Override with env var: `SAGEO_CONFIG=/absolute/path/to/config.json`

Env overrides:
- `SAGEO_PROVIDER`
- `SAGEO_API_KEY`
- `SAGEO_BASE_URL`
- `SAGEO_ORGANIZATION_ID`

## Output Contract

All commands return a structured JSON envelope:

```json
{
  "success": true,
  "data": {},
  "metadata": {}
}
```

Error responses include a machine-readable `code` field:

```json
{
  "success": false,
  "error": {
    "code": "CRAWL_FAILED",
    "message": "crawl failed",
    "detail": "..."
  }
}
```

### Error Codes

| Code | Description |
|------|-------------|
| `INVALID_URL` | Missing or malformed URL input |
| `CONFIG_LOAD_FAILED` | Could not load configuration |
| `CONFIG_SAVE_FAILED` | Could not save configuration |
| `CONFIG_GET_FAILED` | Unknown config key |
| `PROVIDER_NOT_FOUND` | Unknown or invalid provider name |
| `CRAWL_FAILED` | Crawl operation failed |
| `AUDIT_FAILED` | Audit operation failed |
| `REPORT_WRITE_FAILED` | Could not write report to disk |
| `REPORT_LIST_FAILED` | Could not list stored reports |
| `FETCH_TIMEOUT` | HTTP fetch timed out |
| `CANCELLED` | Operation was cancelled |

## SEO Checks

The audit engine checks each crawled page for:

| Rule | Severity | Description |
|------|----------|-------------|
| `title-missing` | error | Page has no title tag |
| `title-too-long` | warning | Title exceeds 60 characters |
| `meta-description-missing` | warning | No meta description |
| `meta-description-too-long` | warning | Meta description exceeds 160 characters |
| `h1-missing` | error | No H1 heading |
| `h1-multiple` | warning | Multiple H1 headings |
| `img-alt-missing` | warning | Image without alt text |
| `canonical-missing` | info | No canonical link tag |
| `broken-page` | error/warning | HTTP 4xx/5xx status |

## Development

```bash
make fmt
make vet
make test
make lint
```

### Pre-commit Hooks

This project uses [pre-commit](https://pre-commit.com/) to run `go fmt`, `go vet`, `go test`, and `golangci-lint` before every commit.

**Install:**

```bash
# Install pre-commit (macOS)
brew install pre-commit

# Or via pip
pip install pre-commit

# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.11.3

# Activate the hooks in this repo
pre-commit install
```

**Run manually:**

```bash
# Run all hooks against all files
pre-commit run --all-files

# Or use the Makefile shortcut (same checks, no pre-commit required)
make precommit
```

## Release

```bash
make release
```

See [`ARCHITECTURE.md`](./ARCHITECTURE.md) for package structure and data flow.
