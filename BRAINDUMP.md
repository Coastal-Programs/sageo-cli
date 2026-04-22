# Sageo, Braindump

Open ideas, decisions, and notes that are still relevant. Everything that's been implemented, shipped, or superseded has been removed. This is a working doc, not a changelog.

---

## User types

Three kinds of people who use this tool:

1. **Site owner with full access.** Has GSC, has the codebase, can make changes. Uses the tool to find and fix their own SEO issues.
2. **SEO consultant or agency.** Has GSC access granted by client, may not have the codebase. Uses Sageo to audit and produce a report for the client.
3. **Cold auditor.** No GSC access. Just has a URL. Wants to crawl, audit, and show the site owner what's wrong to win the work.

### What this means for the tool

- Crawl and audit must work standalone with just a URL (user type 3).
- GSC commands are a bonus layer: more data, better opportunities, but not required.
- Paid commands (SERP, Labs, AEO, GEO, Backlinks) are the deepest layer: competitive intel and keyword data.
- Tool gracefully handles missing access. State should reflect what data sources were used so the agent knows what it has and what it's missing.

---

## Cost positioning

Full Sageo pipeline on one website is roughly $0.07 to $0.15 in paid API calls.

**Free tier ($0):**
- Crawl and audit (our crawler)
- PSI (Google API, optional key for higher limits)
- GSC (Google API, OAuth)

**Paid tier (DataForSEO):**
- SERP batch (top 20 GSC keywords): 20 x $0.0006 = $0.012
- Labs ranked-keywords: ~$0.01
- Labs bulk-difficulty (all GSC keywords): ~$0.001
- Labs competitors: ~$0.01
- Labs search-intent: ~$0.001
- Backlinks summary: $0.02
- Backlinks gap: ~$0.02
- AEO multi-model responses (4 engines x 1 prompt): ~$0.012

**Estimated total per full analysis: $0.07 to $0.15.**

Compare against:
- Ahrefs Lite: $129/month
- Semrush Pro: $139/month
- Screaming Frog: $259/year
- Typical SEO agency audit: $500 to $2,000 one-off

This is the selling point. One full audit for under 20 cents.

---

## Retry policy (principle, not negotiable)

**Retry free calls with backoff. Never retry paid API calls.**

- GSC rate-limited: retry with backoff. Page timeout on crawl: retry once.
- DataForSEO, SerpAPI, any paid call: no retry. The user already paid for the attempt. Report the error, let them decide.
- We do not silently spend users' money.

Verify this is actually enforced in `internal/common/retry` and every paid-endpoint caller. If not enforced, it's a bug.

---

## Open question: MCP server

Previous decision: no. CLI is the point. MCP means tool calls, means context overhead, means slower.

Worth revisiting. MCP adoption has grown significantly, every major AI coding tool supports it, and a properly-scoped MCP server would let agents call sageo functions with typed inputs and outputs instead of parsing JSON from stdout.

**Revisit when:** there is a clear user asking for it, or we see patterns where agents consistently mis-parse our CLI JSON output.

**If we build it:** scope tightly. Expose read-only query functions (`list_recommendations`, `get_state`, `compare_snapshots`) as tool calls. Do NOT expose paid commands without explicit cost-gated confirmation at the tool-call level.

---

## Possible unimplemented diagnostic: schema vs actual rich result

We extract JSON-LD schema from the crawl (what the page HAS). GSC's `searchAppearance` dimension tells us what Google is actually SHOWING. These two data sources should be cross-referenced in the merge engine.

Findings this unlocks:
- "Page has FAQPage schema but Google isn't showing FAQ rich results" (schema invalid, or Google not picking it up).
- "Page has no schema but content could qualify for [X]."

**Action:** check `internal/merge/merge.go`. If no rule exists that joins crawl schema data with GSC search appearance data, add one. Depends on `gsc query appearances` command which already exists.

---

## Future: Bing Webmaster Tools

Bing is 3 to 8% of search share depending on market. Traffic-wise marginal.

But: Bing powers Microsoft Copilot, DuckDuckGo, and Yahoo Search. For AEO, Bing data could matter more than the traffic share suggests.

API exists: https://www.bing.com/webmasters/apidocs

**Decision: note it, don't build it yet.** Extra OAuth flow, extra complexity. Revisit when AEO becomes the primary focus or a user asks for it. If we add it, same tiered approach as GSC: optional layer, graceful when missing.

---

## Future: Google Business Profile

Reviews, photos, posts, Q&A. Only relevant for local businesses. Out of scope currently, sageo is focused on search and content, not local listings.

Revisit if we build a LocalBusiness-focused flow.

---

## Known tension: multi-site vs single-site

The tool is currently single-site per working directory. The optional global registry (in progress) adds cross-site calibration.

Open question: should power users want a multi-site dashboard-like command? E.g. `sageo portfolio` listing all registered sites with last-run stats, open recommendations, forecasted totals.

**Probably yes eventually.** Not a priority while the per-site flow is still being refined.
