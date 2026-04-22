# Sageo CLI — Testing

Sageo enforces a strict separation between **unit tests** (zero network, zero
cost, always run) and **integration tests** (may hit paid live APIs, never run
by default).

## Three Test Modes

| Mode | Command | Network? | Cost? |
|------|---------|----------|-------|
| Unit | `make test` or `go test ./...` | No | No |
| Integration | `make test-integration` | Yes | Yes (~$1–3) |
| Both | `make test-all` | Yes | Yes |

- `make test` is safe on any machine, credentials present or not.
- `make test-integration` prints a cost warning, sleeps 5 seconds, then runs
  `SAGEO_LIVE_TESTS=1 go test -tags integration ./...`.
- CI runs `make test` only. Integration tests are developer-invoked.

## The Rules

Every `*_test.go` file falls into one of two camps.

### Unit test (default)

- No build tag.
- Must not make real HTTP requests.
- Use `httptest.NewServer` (preferred) or a mock `HTTPClient` to stub APIs.
- See `internal/common/testutil/httpfake.go` for factories:
  `NewFakeDataForSEO`, `NewFakeAnthropic`, `NewFakeOpenAI`, `NewFakePSI`.

### Integration test

- Two build-tag lines at the very top of the file:
  ```go
  //go:build integration
  // +build integration
  ```
- Every test function begins with an env-var skip guard.
- Lives alongside the package it exercises, named `*_integration_test.go`.

## Integration Test Template

Copy-paste:

```go
//go:build integration
// +build integration

package mypkg

import (
	"os"
	"testing"

	"github.com/jakeschepis/sageo-cli/internal/common/config"
)

func TestSomething_Live(t *testing.T) {
	if os.Getenv("SAGEO_LIVE_TESTS") != "1" {
		t.Skip("set SAGEO_LIVE_TESTS=1 to run integration tests (real API calls, costs money)")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.DataForSEOLogin == "" {
		t.Skip("credentials not configured; skipping live test")
	}

	// ... exercise the real API ...
}
```

## Unit Test Template (httptest)

```go
import "github.com/jakeschepis/sageo-cli/internal/common/testutil"

func TestFoo(t *testing.T) {
	client, _ := testutil.NewFakeDataForSEO(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status_code":20000,"tasks":[{"status_code":20000}]}`))
	})
	if err := client.VerifyCredentials(); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
}
```

## Enforcement

`make check-tests` (invoked automatically by `make test`) runs
`scripts/check-no-live-tests.sh`, which fails if any non-integration test
file references `http.DefaultClient`, `http.Get(`, `http.Post(`, or
`http.Head(`.

## Where Live Smoke Tests Live

| Package | File |
|---------|------|
| DataForSEO | `internal/dataforseo/client_integration_test.go` |
| PageSpeed Insights | `internal/psi/psi_integration_test.go` |
| Anthropic | `internal/llm/anthropic/anthropic_integration_test.go` |
| OpenAI | `internal/llm/openai/openai_integration_test.go` |

Each is a minimal single smoke test per paid endpoint — enough to confirm
credentials and contract, nothing more.

## End-to-End Pipeline Testing

For full pipeline runs against a real site (crawl → audit → PSI → GSC → SERP
→ Labs → analyze), use the manual workflow:

1. `sageo init --url <site>`
2. `sageo audit run --url <site>`
3. `sageo psi run --url <homepage> --strategy mobile`
4. `sageo gsc query pages`
5. `sageo gsc query keywords`
6. `sageo serp analyze` (on top GSC keywords)
7. `sageo labs ranked-keywords --target <domain>`
8. `sageo analyze` (merge)
9. Inspect `.sageo/state.json` → `merged_findings`.
