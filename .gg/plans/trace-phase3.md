# Wiring Trace: Phase 3 (Auth, GSC, SERP, Opportunities, Cost, Cache)

Date: 2026-04-05
Scope: Full project

## Architecture Map

```
[Entry Points]          → [Orchestration]           → [Modules]              → [External]
internal/cli/root.go      internal/cli/commands/       internal/auth/            Google OAuth2
  auth command               auth.go                  internal/gsc/             GSC API
  gsc command                gsc.go                   internal/serp/serpapi/     SerpAPI
  serp command               serp.go                  internal/opportunities/
  opportunities command      opportunities.go         internal/common/cost/
                                                      internal/common/cache/
                                                      internal/common/config/
                                                      pkg/output/
```

## Paths Traced

### Path 1: Auth login → token store → GSC consumption
- Entry: `root.go:52` → `auth.go:21` (NewAuthCmd)
- Through: `auth.go:101` (loginGSC) → `auth.go:202` (exchangeGSCCode) → `auth/auth.go:54` (Save)
- Consumed: `gsc.go:245` (gscClient) → `gsc/gsc.go:27` (NewClient)
- Status: **FIXED** — was missing expiry check before consumption

### Path 2: Auth status/logout
- Entry: `root.go:52` → `auth.go:54,78`
- Through: `auth/auth.go:107` (Status), `auth/auth.go:94` (Delete)
- Status: OK

### Path 3: Config → GSC commands
- Entry: `config.go` (Set/Get/Redacted) → `gsc.go:80,268,273`
- Config keys: `gsc_property`, `gsc_client_id`, `gsc_client_secret`
- Env overrides: `config.go:208-218`
- Status: OK

### Path 4: Config → SERP commands
- Entry: `config.go` → `serp.go:230-240` (serpProvider)
- Config keys: `serp_provider`, `serp_api_key`, `approval_threshold_usd`
- Env overrides: `config.go:199-209`
- Status: OK

### Path 5: Cost estimation → approval gate → SERP execute
- Entry: `serp.go:65` (Estimate) → `serp.go:70` (EvaluateApproval) → `serp.go:82` (dry-run) / `serp.go:90` (approval block) / `serp.go:109` (execute)
- Status: OK

### Path 6: SERP cache flow
- Entry: `serp.go:97` (cache check) → `cache/cache.go:53` (Get) → `serp.go:109` (execute on miss) → `serp.go:116` (cache set)
- Status: OK

### Path 7: GSC SiteURL → API URL construction
- Entry: `gsc.go:129` (QueryRequest with SiteURL) → `gsc/gsc.go:166` (searchAnalytics URL)
- Status: **FIXED** — was not URL-encoding SiteURL

### Path 8: Auth URL construction
- Entry: `auth.go:126-129` (authURL sprintf)
- Status: **FIXED** — was using fmt.Sprintf instead of url.Values

### Path 9: Opportunities merge flow
- Entry: `opportunities.go:104-105` (GSC seeds) → `opportunities.go:115-123` (SERP enrichment) → `opportunities/opportunities.go:36` (Merge)
- Status: **FIXED** — EstimatedCost field was never populated

### Path 10: Root command registration + tests
- Entry: `root.go:46-55` (AddCommand)
- Tests: `root_test.go:8-19` (top-level), `root_test.go:38-47` (subcommands)
- Status: OK

### Path 11: Error codes → command usage
- Entry: `errors.go:16-22` (new codes)
- Consumed by: `auth.go`, `gsc.go`, `serp.go`, `opportunities.go`
- Status: OK

## Gaps Found

| # | Where | What Was Lost | Impact | Severity | Status |
|---|-------|-------------|--------|----------|--------|
| 1 | `gsc.go:165` | SiteURL not URL-encoded in API path | GSC API calls return 404 for all properties | Critical | **Fixed** |
| 2 | `gsc.go:248`, `opportunities.go:41` | Expired token used without checking expiry | Silent 401 errors from Google API | High | **Fixed** |
| 3 | `auth.go:127` | Auth URL params not properly encoded | Malformed OAuth URL in edge cases | Medium | **Fixed** |
| 4 | `opportunities.go:26` | `EstimatedCost` field always 0 | Dead field in output contract | Medium | **Fixed** |

## Systemic Patterns

- **No token refresh**: When GSC access tokens expire (typically 1 hour), the user must re-run `auth login gsc`. There's no automatic refresh using the stored refresh token. This is acceptable for Phase 3 but will need addressing when GSC usage increases.
- **`serp compare` skips caching**: Unlike `serp analyze`, the compare command doesn't check or populate cache. Minor inconsistency — acceptable since compare is multi-query and less commonly cached.

## Summary
- Paths traced: 11
- Gaps found: 4 (Critical: 1, High: 1, Medium: 2) — **all fixed**
- Systemic patterns: 2 (noted, not blocking)
