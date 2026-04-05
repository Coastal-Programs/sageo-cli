# Phase 4 — DataForSEO Integration (SEO + AEO + GEO)

## Context

Phase 3 left the CLI with working crawl, audit, GSC, SERP (via SerpAPI), and opportunity detection. The problem:

- SerpAPI is expensive (~$0.01/search vs DataForSEO's $0.0006) and only covers SERP
- There's zero AEO/GEO coverage — no way to see how AI engines (ChatGPT, Gemini, Perplexity) mention or cite your brand/domain
- The CLI's tagline says "SEO, GEO, AEO" but only delivers SEO

DataForSEO solves all three in one API, one key, one billing account:
- **SEO**: SERP API replaces SerpAPI at 10–16x lower cost
- **AEO**: LLM Responses API — query ChatGPT/Claude/Gemini/Perplexity about your brand
- **GEO**: LLM Mentions API — track how often your domain/keywords appear in AI responses, with AI search volume trends

## Auth Model

DataForSEO uses HTTP Basic Auth: `login:password` (their terminology — login is your account email, password is your API password from the dashboard). No OAuth, no tokens. Just base64-encode `login:password` and put it in `Authorization: Basic <encoded>`.

## What Changes

### Config
- Add `dataforseo_login` + `dataforseo_password` fields to `Config` struct
- Add `SAGEO_DATAFORSEO_LOGIN` + `SAGEO_DATAFORSEO_PASSWORD` env overrides
- Keep `serp_api_key` + `serp_provider` for backward compat (SerpAPI still works)
- Add `dataforseo_login` and `dataforseo_password` to `logout.go` clearing

### New Package: `internal/dataforseo`
A shared DataForSEO HTTP client used by all three feature areas. Single place for Basic Auth, base URL, error handling, and request/response patterns.

### SERP: DataForSEO adapter
- New `internal/serp/dataforseo/` package implementing `serp.Provider`
- Endpoint: `POST https://api.dataforseo.com/v3/serp/google/organic/live/regular`
- Wire into `serpProvider()` in `serp.go` — when `serp_provider = "dataforseo"` (or when `dataforseo_login` is set and `serp_provider` is empty/default)
- Accurate cost: $0.0006/query standard, $0.002/query live (we'll use live for CLI responsiveness)
- Update `opportunities.go` to use DataForSEO SERP when configured

### New: AEO — `sageo aeo` command group
- `sageo aeo responses` — query ChatGPT/Claude/Gemini/Perplexity about your brand/topic, see what they say
  - Endpoint: `POST https://api.dataforseo.com/v3/ai_optimization/chat_gpt/llm_responses/live`
  - Flags: `--prompt`, `--model` (chatgpt|claude|gemini|perplexity), `--dry-run`
  - Cost: ~$0.002–$0.004/query (LLM pass-through pricing)
- `sageo aeo keywords` — AI search volume for keywords (how often used in AI tools)
  - Endpoint: `POST https://api.dataforseo.com/v3/ai_optimization/ai_keyword_data/search_volume/live`
  - Flags: `--keyword`, `--location`, `--language`
  - Cost: $0.01/task

### New: GEO — `sageo geo` command group  
- `sageo geo mentions` — how often your domain/brand is mentioned in AI responses
  - Endpoint: `POST https://api.dataforseo.com/v3/ai_optimization/llm_mentions/search/live`
  - Flags: `--domain`, `--keyword`, `--platform` (google|bing), `--dry-run`
  - Returns: mention count, AI search volume, impressions, trending over 12 months
  - Cost: per-row billing; $100/mo minimum commitment applies (surfaced clearly in output)
- `sageo geo top-pages` — which pages are cited most in AI responses for your topics
  - Endpoint: `POST https://api.dataforseo.com/v3/ai_optimization/llm_mentions/top_pages/live`
  - Flags: `--keyword`, `--domain`

### Login flow update
- `sageo login` option 2 becomes "DataForSEO (SERP + AEO/GEO)" replacing "SerpAPI (API key)"
- Prompts for DataForSEO login (email) + password
- Keep SerpAPI as option 3 for users who already have it
- Update menu label in `internal/cli/commands/login.go`

### Error codes
- Add `ErrDataForSEOFailed = "DATAFORSEO_FAILED"` and `ErrAEOFailed = "AEO_FAILED"` and `ErrGEOFailed = "GEO_FAILED"` to `pkg/output/errors.go`

### root_test.go
- Add `aeo` and `geo` to expected top-level commands

---

## File Map

| File | Action |
|------|--------|
| `internal/common/config/config.go` | Add `DataForSEOLogin`, `DataForSEOPassword` fields + env overrides + Set/Get/Redacted |
| `internal/dataforseo/client.go` | New — shared HTTP client with Basic Auth |
| `internal/serp/dataforseo/dataforseo.go` | New — `serp.Provider` implementation |
| `internal/cli/commands/serp.go` | Wire DataForSEO into `serpProvider()` |
| `internal/cli/commands/opportunities.go` | Update SERP cost basis for DataForSEO |
| `pkg/output/errors.go` | Add 3 new error codes |
| `internal/cli/commands/aeo.go` | New — `sageo aeo` command group |
| `internal/cli/commands/geo.go` | New — `sageo geo` command group |
| `internal/cli/commands/login.go` | Add DataForSEO option, relabel SerpAPI |
| `internal/cli/commands/logout.go` | Clear DataForSEO credentials on logout |
| `internal/cli/root.go` | Wire `aeo` and `geo` top-level commands |
| `internal/cli/root_test.go` | Add `aeo`, `geo` to expected commands |

---

## Steps

1. Add `DataForSEOLogin` and `DataForSEOPassword` fields to `internal/common/config/config.go` — struct fields, `Set`/`Get`/`Redacted` cases, `applyEnvOverrides` entries for `SAGEO_DATAFORSEO_LOGIN` and `SAGEO_DATAFORSEO_PASSWORD`
2. Create `internal/dataforseo/client.go` — shared client struct with Basic Auth header construction, a `Post(endpoint string, body any) ([]byte, error)` method, and a `New(login, password string) *Client` constructor
3. Create `internal/serp/dataforseo/dataforseo.go` — implement `serp.Provider` using the DataForSEO shared client, targeting `POST /v3/serp/google/organic/live/regular`, with accurate cost estimate of $0.002/query (live mode)
4. Update `internal/cli/commands/serp.go` — add `"dataforseo"` case to `serpProvider()` factory, falling back to DataForSEO when `dataforseo_login` is set and provider is unset
5. Update `internal/cli/commands/opportunities.go` — detect DataForSEO provider and use correct cost basis ($0.002/query instead of $0.01)
6. Add `ErrDataForSEOFailed`, `ErrAEOFailed`, `ErrGEOFailed` to `pkg/output/errors.go`
7. Create `internal/cli/commands/aeo.go` — `NewAEOCmd` with two subcommands: `responses` (LLM Responses live endpoint, flags: `--prompt`, `--model`, `--dry-run`) and `keywords` (AI Keyword Data search volume, flags: `--keyword`, `--location`, `--language`)
8. Create `internal/cli/commands/geo.go` — `NewGEOCmd` with two subcommands: `mentions` (LLM Mentions search live endpoint, flags: `--domain`, `--keyword`, `--platform`, `--dry-run`) and `top-pages` (LLM Mentions top pages endpoint, flags: `--keyword`, `--domain`)
9. Update `internal/cli/commands/login.go` — replace "SerpAPI" menu item with "DataForSEO (SERP + AEO/GEO)", prompting for login + password; keep SerpAPI as a separate option
10. Update `internal/cli/commands/logout.go` — clear `dataforseo_login` and `dataforseo_password` config keys on logout
11. Wire `commands.NewAEOCmd` and `commands.NewGEOCmd` into `internal/cli/root.go`
12. Update `internal/cli/root_test.go` — add `"aeo"` and `"geo"` to expected top-level commands list
13. Run `go build ./...`, `go vet ./...`, `go test ./...` and fix any issues
