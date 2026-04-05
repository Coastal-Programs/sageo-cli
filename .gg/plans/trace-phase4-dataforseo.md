# Wiring Trace: DataForSEO Integration (Phase 4)

Date: 2026-04-05
Scope: Full project — all files added or modified in the Phase 4 implementation

---

## Architecture Map

```
[Entry Points]                   [Orchestration]                [Modules]                      [External]
root.go                          config/config.go               dataforseo/client.go           api.dataforseo.com
commands/aeo.go                  commands/serp.go:serpProvider  serp/dataforseo/dataforseo.go
commands/geo.go                  commands/opportunities.go:     serp/serpapi/serpapi.go
commands/serp.go                   fetchSERPForSeeds
commands/opportunities.go
commands/login.go
commands/logout.go
```

**Config flow**: `login.go` writes credentials → `config.go` persists → `config.Load()` returns cfg → commands read `cfg.DataForSEOLogin / cfg.DataForSEOPassword / cfg.SERPProvider` → client/adapter instantiated → DataForSEO API called.

---

## Paths Traced

### Path 1: `sageo login` → DataForSEO credentials → `sageo serp analyze`
- Entry: `commands/login.go:loginDataForSEOInteractive` — saves `dataforseo_login` + `dataforseo_password`, leaves `serp_provider` unchanged
- Through: `config/config.go:Save` — persists the two new fields
- Through: `commands/serp.go:serpProvider` — reads `cfg.SERPProvider` to select provider
- Destination: `serp/dataforseo/dataforseo.go:Analyze` (intended)
- **Status: GAP FOUND** — `serp_provider` is never updated to `"dataforseo"` by login, so `serpProvider()` never reaches the DataForSEO branch

### Path 2: `serpProvider()` auto-fallback for DataForSEO credentials
- Entry: `commands/serp.go:247` — `case "serpapi", "": if cfg.SERPProvider == "" && cfg.DataForSEOLogin != ""`
- Through: `config/config.go:NewDefault` — sets `SERPProvider: "serpapi"` (line 88)
- Destination: the condition `cfg.SERPProvider == ""` (line 240)
- **Status: GAP FOUND** — `SERPProvider` is `"serpapi"` by default, never `""`. The auto-fallback branch is unreachable in practice. Same logic duplicated at `opportunities.go:167-168`.

### Path 3: `sageo aeo responses` → DataForSEO LLM endpoint
- Entry: `commands/aeo.go:43` — validates `--prompt`, checks credentials
- Through: `dataforseo/client.go:Post` — builds Basic Auth header, sends POST
- Through: `aeoEndpointForModel(model)` — maps model name to endpoint path
- Destination: DataForSEO API endpoint
- **Status: GAP FOUND (minor)** — invalid `--model` values silently route to chatgpt with no error or warning

### Path 4: `sageo aeo keywords` → DataForSEO AI keyword endpoint
- Entry: `commands/aeo.go:159` — validates `--keyword`, checks credentials, evaluates approval
- Through: `dataforseo/client.go:Post`
- Destination: `/v3/ai_optimization/ai_keyword_data/search_volume/live`
- **Status: GAP FOUND** — no `--dry-run` flag; cost estimate runs and approval gate is checked, but there is no way for the user to preview cost without executing

### Path 5: `sageo geo mentions` → DataForSEO LLM Mentions endpoint
- Entry: `commands/geo.go:42` — validates `--keyword`, checks credentials, estimates cost, evaluates approval, supports `--dry-run`
- Through: `dataforseo/client.go:Post`
- Destination: `/v3/ai_optimization/llm_mentions/search/live`
- **Status: OK**

### Path 6: `sageo geo top-pages` → DataForSEO LLM Mentions Top Pages endpoint
- Entry: `commands/geo.go:160` — validates `--keyword`, checks credentials
- Through: `dataforseo/client.go:Post`
- Destination: `/v3/ai_optimization/llm_mentions/top_pages/live`
- **Status: GAP FOUND** — no cost estimation, no approval gate, no `--dry-run`. Goes directly from credential check to live API call.

### Path 7: `sageo opportunities --with-serp` → DataForSEO SERP (via fetchSERPForSeeds)
- Entry: `commands/opportunities.go:167` — `usingDataForSEO := cfg.SERPProvider == "dataforseo" || (cfg.SERPProvider == "" && ...)`
- Through: same dead-code condition as Path 2
- Destination: `serpdforseo.New(...)` (intended)
- **Status: GAP FOUND** — same unreachable branch; DataForSEO is silently bypassed unless user explicitly sets `serp_provider = "dataforseo"`

### Path 8: `sageo logout` → credential clearing
- Entry: `commands/logout.go:runLogout`
- Through: `cfg.Set("dataforseo_login", "")`, `cfg.Set("dataforseo_password", "")`
- Destination: `config/config.go:Save`
- **Status: OK**

### Path 9: `config.Redacted()` → `sageo config show`
- Entry: `config/config.go:Redacted` — includes `dataforseo_login` (clear) and `dataforseo_password` (redacted)
- **Status: OK** — both fields present and correctly handled

### Path 10: `SAGEO_DATAFORSEO_LOGIN` / `SAGEO_DATAFORSEO_PASSWORD` env overrides
- Entry: `config/config.go:applyEnvOverrides` — lines 217–221
- Destination: `cfg.DataForSEOLogin` / `cfg.DataForSEOPassword`
- **Status: OK** — wired correctly

---

## Gaps Found

| # | Where | What's Lost | Impact | Severity |
|---|-------|-------------|--------|----------|
| 1 | `login.go:loginDataForSEOInteractive` → `serp.go:serpProvider:238` | `serp_provider` never set to `"dataforseo"` after login | After `sageo login`, SERP and opportunities commands still try to use SerpAPI and fail with "serp_api_key not configured" | **High** |
| 2 | `serp.go:240` + `opportunities.go:168` | `cfg.SERPProvider == ""` is unreachable; `NewDefault()` sets `"serpapi"` | Auto-fallback to DataForSEO is dead code. Two files share the same broken condition. | **High** |
| 3 | `geo.go:newGEOTopPagesCmd` (line 152–230) | No cost estimation, no approval gate, no `--dry-run` | `sageo geo top-pages` bills the user immediately with no preview or threshold enforcement | **High** |
| 4 | `aeo.go:newAEOKeywordsCmd` (line 152–255) | No `--dry-run` flag | User cannot preview cost before executing; inconsistent with every other paid command | **Medium** |
| 5 | `aeo.go:aeoEndpointForModel` (line 258–269) | Invalid `--model` silently routes to chatgpt | User typos or unsupported values produce wrong output with no error | **Medium** |

---

### Gap 1: Login doesn't set `serp_provider = "dataforseo"`
**Trace**: `login.go:loginDataForSEOInteractive` saves credentials → `serp.go:serpProvider:238` reads `cfg.SERPProvider` → is `"serpapi"` (default) → falls into serpapi branch → returns "serp_api_key not configured"
**What happens**: User runs `sageo login`, enters DataForSEO credentials successfully, then runs `sageo serp analyze --query "..."` and gets an error asking for a SerpAPI key. Feature is completely non-functional via login flow.
**Fix**: Add `cfg.Set("serp_provider", "dataforseo")` in `loginDataForSEOInteractive` after saving credentials, then save config.

---

### Gap 2: Auto-fallback condition is unreachable (`cfg.SERPProvider == ""`)
**Trace**: `config.go:NewDefault:88` sets `SERPProvider: "serpapi"` → `serp.go:240` checks `cfg.SERPProvider == ""` → condition is always false for all real users
**What happens**: Even if a user manually sets DataForSEO credentials via `sageo config set`, unless they also run `sageo config set serp_provider dataforseo`, SERP commands and opportunities will fail. The auto-detect logic is silent and never fires. Same bug in two places: `serp.go:240` and `opportunities.go:167–168`.
**Fix**: Change the condition from `cfg.SERPProvider == ""` to `cfg.SERPProvider != "dataforseo"` (i.e. the outer case branch already handles it), OR restructure `serpProvider()` to check DataForSEO credentials first regardless of `SERPProvider` value when it's the default `"serpapi"`.

---

### Gap 3: `geo top-pages` has no cost gate
**Trace**: `geo.go:160` checks credentials → `geo.go:176` calls `dataforseo.New(...)` → `geo.go:185` calls `client.Post(...)` → live API call, billing occurs
**What happens**: The approval threshold is ignored. A user with `approval_threshold_usd = 0` who expects to be gated on all paid calls will be billed without prompt. The `--dry-run` flag that the rest of the system provides is absent. Inconsistent with `geo mentions` in the same file.
**Fix**: Add cost estimation + `cost.EvaluateApproval` + `--dry-run` handling to `newGEOTopPagesCmd`, mirroring the pattern in `newGEOMentionsCmd`.

---

### Gap 4: `aeo keywords` missing `--dry-run`
**Trace**: `aeo.go:185` evaluates approval and gates on threshold — but the user has no way to trigger a dry-run preview. The flag simply doesn't exist.
**What happens**: Users cannot safely check what a keyword query will cost before it runs. This breaks the cost-transparency contract the CLI provides everywhere else (serp analyze, serp compare, opportunities, aeo responses, geo mentions).
**Fix**: Add `var dryRun bool`, register `--dry-run` flag, and add a dry-run early-return block after approval evaluation in `newAEOKeywordsCmd`.

---

### Gap 5: Invalid `--model` silently falls through to chatgpt
**Trace**: `aeo.go:96` calls `aeoEndpointForModel(model)` → `aeo.go:258` switch has no `default` error case, falls through to chatgpt
**What happens**: `sageo aeo responses --model gpt4` or any typo silently queries chatgpt, charges for it, and returns chatgpt results labeled incorrectly. The response `model` field from the API will reflect the real model, but the command metadata `"model": model` in the envelope will show the user's incorrect input.
**Fix**: Add an explicit `default: return "", fmt.Errorf("unsupported model %q; valid values: chatgpt, claude, gemini, perplexity", model)` and return that error before calling `client.Post` in `newAEOResponsesCmd`.

---

## Systemic Patterns

1. **Default `serp_provider` value poisons the auto-fallback** — `NewDefault()` sets `"serpapi"`, so any conditional on `SERPProvider == ""` is dead. This pattern appears in two files (`serp.go`, `opportunities.go`). Any future feature that tries to detect "provider not explicitly set" will hit the same trap.

2. **New paid commands inconsistently implement the cost gate** — `geo mentions` does it right; `geo top-pages` skips it entirely; `aeo keywords` is halfway (gate but no dry-run). The pattern exists but isn't being enforced as a checklist when adding new paid subcommands.

---

## Summary
- Paths traced: 10
- Gaps found: 5 (High: 3, Medium: 2)
- Systemic patterns: 2
