# Testing

How to run tests in this repo and how to add new ones.

## Three modes

| Mode | Command | Network | Cost | Gate |
|---|---|---|---|---|
| Unit | `make test` (or `go test ./...`) | None | $0 | Default; always safe |
| Integration | `make test-integration` | Live paid APIs | $1 to $3 per run | Requires `SAGEO_LIVE_TESTS=1`; prompts for 5s before running |
| All | `make test-all` | Unit then live | $1 to $3 | Unit first, then integration |

`make test` runs `make check-tests` first: `scripts/check-no-live-tests.sh` scans every `*_test.go` under `internal/` and fails if any non-integration test references `http.DefaultClient`, `http.Get(`, `http.Post(`, or `http.Head(`. This is the guardrail: unit tests never touch the real internet.

## Writing new tests

**Rule.** Every new test file must satisfy one of:

1. **Fully mocked.** Use `httptest.NewServer` via `internal/common/testutil`, or inject a mock `HTTPClient`. No outbound calls.
2. **Integration tagged and gated.** First line `//go:build integration`, file named `*_integration_test.go`, every test function guarded by `if os.Getenv("SAGEO_LIVE_TESTS") != "1" { t.Skip(...) }`.

`scripts/check-no-live-tests.sh` enforces this in CI and in `make test`. If you add a test that legitimately needs live coverage, put it behind the integration tag.

## Test utilities

`internal/common/testutil/httpfake.go` gives you local fake servers wired into each external client. All factories register automatic teardown via `t.Cleanup`.

Available factories:

- `NewFakeDataForSEO(t, handler) (*dataforseo.Client, *httptest.Server)`
- `NewFakeAnthropic(t, handler) (*anthropic.Client, *httptest.Server)`
- `NewFakeOpenAI(t, handler) (*openai.Client, *httptest.Server)`
- `NewFakePSI(t, handler) (*psi.Client, *httptest.Server)` (rewrites outbound URL scheme + host, since PSI's base URL is a package constant)

Minimal example:

```go
package mypkg_test

import (
    "net/http"
    "testing"

    "github.com/jakeschepis/sageo-cli/internal/common/testutil"
)

func TestDataForSEOBilling(t *testing.T) {
    client, _ := testutil.NewFakeDataForSEO(t, func(w http.ResponseWriter, r *http.Request) {
        _, _ = w.Write([]byte(`{"status_code":20000,"cost":0.0006,"tasks":[]}`))
    })

    got, err := client.DoSomething(/* ... */)
    if err != nil {
        t.Fatalf("call: %v", err)
    }
    if got.Cost != 0.0006 {
        t.Errorf("cost = %v, want 0.0006", got.Cost)
    }
}
```

For clients without a dedicated factory, use `httptest.NewServer` directly and inject the resulting URL via the client's option (e.g. `WithBaseURL`), or inject a mock `HTTPClient` interface (see `internal/dataforseo/client_test.go` for the pattern).

## Integration test template

Copy-paste starter:

```go
//go:build integration

package mypkg_test

import (
    "context"
    "os"
    "testing"

    "github.com/jakeschepis/sageo-cli/internal/common/config"
    "github.com/jakeschepis/sageo-cli/internal/dataforseo"
)

func TestDataForSEOLive(t *testing.T) {
    if os.Getenv("SAGEO_LIVE_TESTS") != "1" {
        t.Skip("set SAGEO_LIVE_TESTS=1 to run paid integration tests")
    }

    cfg, err := config.Load()
    if err != nil {
        t.Fatalf("config load: %v", err)
    }
    if cfg.DataForSEOLogin == "" || cfg.DataForSEOPassword == "" {
        t.Skip("DataForSEO credentials not set")
    }

    client := dataforseo.New(cfg.DataForSEOLogin, cfg.DataForSEOPassword)
    _, err = client.Ping(context.Background())
    if err != nil {
        t.Fatalf("live call failed: %v", err)
    }
}
```

Name the file `*_integration_test.go`. Keep the build tag on line 1 with a blank line after it.

## Running tests locally

Day-to-day:

```bash
make test                 # unit only; expected duration <30s on a modern laptop
go test ./internal/merge  # single package
go test -run Forecast ./internal/forecast
```

Before a release:

```bash
make precommit            # fmt + vet + test + lint
make test-integration     # live paid APIs; expect $1 to $3
```

Coverage:

```bash
make coverage             # runs `make test` then opens HTML coverage in browser
```

## CI

CI runs unit tests only: `make test` plus `make vet`. No paid API calls. The `check-tests` guard fails the build if any unit test references live HTTP. Integration tests are run manually by a maintainer before tagging a release, and are not part of the automated pipeline.

Keep unit tests fast. A full `go test -race ./...` run should stay under 60 seconds locally.
