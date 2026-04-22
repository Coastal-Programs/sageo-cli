# Phase 1: Stabilise What We Have

## Goal
Lock down everything that's built. Fix bugs, test everything, add resilience, ship the agent skill file. No new features.

---

## Task 1: Test all untested commands

### serp analyze + compare
- Run `sageo serp analyze --query "seo tools" --dry-run` — verify cost estimate, no API call
- Run `sageo serp analyze --query "seo tools"` — verify real results return
- Run `sageo serp compare --query "seo tools" --query "seo software"` — verify multi-query works
- Check: JSON envelope correct, error handling on bad query, cost metadata present

### aeo responses + keywords
- Run `sageo aeo responses --prompt "What is Sageo CLI?" --model chatgpt --dry-run`
- Run `sageo aeo keywords --keyword "seo tools" --dry-run`
- Run one live (cheapest model) — verify real response
- Check: model flag works, cost metadata, approval gate if threshold set

### geo mentions + top-pages
- Run `sageo geo mentions --keyword "seo tools" --domain coastalprograms.com --dry-run`
- Run `sageo geo top-pages --keyword "seo tools" --dry-run`
- Run one live — verify real response
- Check: domain flag, platform flag, cost metadata

### labs remaining subcommands
- `sageo labs keywords --target coastalprograms.com --dry-run`
- `sageo labs overview --target coastalprograms.com --dry-run`
- `sageo labs competitors --target coastalprograms.com --dry-run`
- `sageo labs keyword-ideas --keyword "seo tools" --dry-run`
- Run each live once — verify data returns
- Check: limit flag, cost metadata

### provider list + use
- Run `sageo provider list` — verify "local" shows
- Run `sageo provider use local` — verify works

### approval gate
- Set `sageo config set approval_threshold_usd 0.001`
- Run any paid command (e.g. labs ranked-keywords)
- Verify: returns APPROVAL_REQUIRED, does NOT make the API call
- Reset threshold after test

### cache
- Run a paid command twice
- Second run should return cached: true in metadata
- Verify: cached response matches first response
- Run `sageo config set approval_threshold_usd 0` to reset

---

## Task 2: Fix the crawl page ordering inconsistency

### Problem
- Concurrent crawl returns pages in different order each run
- Same site, same flags, different page order in results
- Not a data bug but makes output unpredictable

### Fix
- Sort pages by URL alphabetically before returning the result
- File: internal/crawl/crawler.go, after wg.Wait()
- Add: sort.Slice(result.Pages, func(i, j int) bool { return result.Pages[i].URL < result.Pages[j].URL })

---

## Task 3: Add retry logic for free API calls

### Rules
- Free calls (GSC, crawl page fetches): retry up to 2 times with backoff
- Paid calls (DataForSEO, SerpAPI): NEVER retry. Report error, let user decide.
- Retryable status codes: 429, 500, 502, 503
- Backoff: 1s, then 3s

### Where to add
- internal/gsc/gsc.go — wrap searchAnalytics and ListSites calls
- internal/crawl/crawler.go — wrap fetcher.Fetch call
- Do NOT add to internal/dataforseo/client.go
- Do NOT add to internal/serp/serpapi/serpapi.go

---

## Task 4: Add `sageo status` command

### What it does
- Reads .sageo/state.json
- Returns the full state as JSON
- If no .sageo/ exists: return error "No project initialized — run sageo init --url"

### Output
```json
{
  "success": true,
  "data": {
    "site": "https://www.coastalprograms.com",
    "initialized": "2026-04-06T10:00:00Z",
    "last_crawl": "2026-04-06T10:05:00Z",
    "score": 75.7,
    "pages_crawled": 23,
    "findings_count": 42,
    "sources_used": ["crawl"],
    "sources_missing": ["gsc"],
    "history_count": 2
  }
}
```

### Key detail: sources_used and sources_missing
- Check what data exists in state.json
- If findings exist but no GSC data: sources_missing includes "gsc"
- This tells the agent: "you could get more data by connecting GSC"

### Files
- New: internal/cli/commands/status.go
- Edit: internal/cli/root.go (register command)
- Edit: internal/state/state.go (add helper to compute sources)

---

## Task 5: Wire crawl command to save to state.json

### Current state
- Only audit run writes to state.json
- Crawl run prints JSON and forgets

### What to do
- After crawl completes, if .sageo/ exists, save crawl summary to state.json
- Don't save the full page data (too big) — save: pages_crawled, urls found, crawl errors, timestamp
- Add history entry: "crawl: 23 pages, 0 errors"

### Files
- Edit: internal/cli/commands/crawl.go
- Edit: internal/state/state.go (add crawl summary fields if needed)

---

## Task 6: Write the agent skill file

### What it is
- One markdown file that teaches an AI agent everything about Sageo
- Shipped in `.claude/skills/sageo/` (Agent Skills convention) with a flat-file mirror at `.gg/skills/sageo.md`
- Agent reads it at session start, knows every command and the workflow

### Contents
- What Sageo is (one paragraph)
- The recommended workflow: init → crawl → audit → gsc → opportunities
- Every command with: usage, flags, what it returns, cost (free/paid)
- The tiered model: URL-only → +GSC → +paid APIs
- How to read state.json
- Do's and don'ts:
  - Always use --dry-run before paid commands
  - Never retry paid calls
  - Always check sources_missing in status
  - Round numbers for human output

### Files
- New: `.claude/skills/sageo/SKILL.md` (plus `commands.md`, `workflows.md`, `troubleshooting.md`)
- New: `.gg/skills/sageo.md`

---

## Task 7: Write unit tests for the state package

### What to test
- Init creates .sageo/state.json correctly
- Init fails if state.json already exists
- Load reads state.json correctly
- Save writes and is re-loadable
- UpdateAudit replaces findings and updates timestamp
- AddHistory appends without losing existing entries
- Exists returns true/false correctly
- Path returns correct path

### Files
- New: internal/state/state_test.go

---

## Task 8: Add contract tests for JSON envelope

### What to test
- Every command output has "success" key (true or false)
- Success responses have "data" key
- Error responses have "error" key with "message"
- Paid commands have metadata with: estimated_cost, currency, source
- No command ever prints raw text outside the envelope

### How
- Run each command in a test, capture stdout, parse as JSON, validate structure
- Can be a single test file that iterates through command configs

### Files
- New: internal/cli/commands/envelope_test.go

---

## Done criteria
- All commands tested live and documented in testing log
- Bugs from this session fixed (crawl ordering, retry logic)
- sageo status works
- Crawl saves to state.json
- Agent skill file written
- State package has unit tests
- JSON envelope contract tested

---

## Phase 1 Final Verification Results (2026-04-06)

### Static checks — ALL PASS
- `go test ./...` — **21 packages tested, all pass** (4 skipped: no test files)
- `go vet ./...` — **clean, no issues**
- `go build ./...` — **compiles clean**

### End-to-end manual test — ALL PASS
- `sageo init --url https://www.coastalprograms.com` — ✅ creates .sageo/state.json
- `sageo status` (after init) — ✅ shows pages_crawled=0, score=0, findings_count=0
- `sageo crawl run --url ... --depth 1 --max-pages 3` — ✅ returns 3 pages with full data
- `sageo status` (after crawl) — ✅ pages_crawled=3, history_count=1
- `sageo audit run --url ... --depth 1 --max-pages 3` — ✅ score=73.3, 7 warnings found
- `sageo status` (after audit) — ✅ score=73.3, findings_count=7, history_count=2

### Data integrity checks — ALL PASS
- state.json findings have `why` and `fix` fields — ✅
- History has entries for both crawl and audit — ✅
- Score rounded to 1 decimal (73.3) — ✅
- No double error printing on `audit run` without --url — ✅ (single JSON error on stdout, empty stderr)

### Issues found during verification
- **None.** All tasks completed cleanly; no fixes needed.
