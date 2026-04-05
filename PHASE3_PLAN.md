# Phase 3 Plan — Agent-first SEO / AEO / GEO Intelligence

Date: 2026-04-05

## Goal
Turn `sageo-cli` from a technical SEO crawler/auditor into an **agent-facing intelligence CLI** that combines:
- local crawl + audit data
- Google Search Console performance data
- one paid SERP data provider
- cost-aware execution rules
- clear machine-readable outputs that let an external AI agent decide what to do next

This phase is **not** a dashboard phase.
This phase is **not** an embedded-LLM phase.
The CLI should remain the structured data/action layer for AI agents.

## Real-world pattern signals used in this update

### Cost estimation / approval gates
Observed pattern:
- tools often expose fields like `estimated_cost`, `require_approval`, and budget checks before expensive operations

Examples found:
- `Kocoro-lab/Shannon` → `BudgetCheckResult` with:
  - `can_proceed`
  - `require_approval`
  - `reason`
  - `warnings`
  - `estimated_cost`
- other public codebases also expose `estimated_cost` directly in JSON payloads

### Dry-run command design
Observed pattern:
- CLI tools commonly expose `--dry-run` to preview expensive/destructive work before executing

Examples found:
- `supabase/cli`
- `argoproj/argo-workflows`
- `raystack/guardian`
- `pachyderm/pachyderm`

### Existing CLI ergonomics already aligned
- `SilenceErrors: true`
- `SilenceUsage: true`
- persistent `--output`

## Product simplification (important)

Do **not** integrate every SEO API category.
Start with the smallest useful external stack.

## Phase 3 integration stack

### Required integrations
1. **Google Search Console**
   - primary source for actual search performance
   - clicks, impressions, CTR, average position, queries, pages

2. **One SERP provider only**
   Choose exactly one for phase 3:
   - `DataForSEO` **or** `SerpAPI`

   Purpose:
   - live SERP snapshots
   - ranking validation
   - competitor page discovery
   - SERP feature visibility

### Do not add yet
- backlink APIs
- domain analytics APIs
- merchant APIs
- app data APIs
- broad “all-in-one” SEO intelligence vendors
- embedded LLM providers inside the CLI by default

## Why this is the right architecture

Because `sageo-cli` is being built for **AI agents**, not for a human sitting in a terminal all day.

That means:
- the CLI should **collect and normalize evidence**
- the external AI agent should **reason and decide**

So the CLI should return:
- crawl data
- audit findings
- GSC performance data
- SERP snapshots
- opportunity candidates
- cost estimates
- freshness / source metadata

The AI agent consuming the CLI can then decide:
- what to run next
- whether cost is worth it
- what recommendations to present to the user

## Core design principle for paid APIs

### Free-first, paid-second
Every workflow should prefer:
1. local crawl/audit data
2. Search Console data
3. paid SERP lookups only when necessary

### Never do broad paid fan-out by default
Do **not** design commands that casually make 100+ paid API calls.

Instead:
- get top candidates from GSC first
- narrow to top 5 / 10 / 20 opportunities
- then call SERP provider only for those

## Required cost-aware behavior

Every paid command should support:
- `--dry-run` → show what would be called, not actually call it
- `estimated_cost` in JSON output
- `requires_approval` boolean in JSON output when above threshold
- `source` metadata per result (`gsc`, `serp_provider`, `local_crawl`)
- `cached` boolean
- `fetched_at` timestamp

Recommended metadata shape:

```json
{
  "estimated_cost": 0.42,
  "currency": "USD",
  "requires_approval": false,
  "cached": true,
  "source": "serpapi",
  "fetched_at": "2026-04-05T12:00:00Z"
}
```

## Phase 3 command families

Add these command groups conceptually:

### `auth`
Purpose: external provider auth/config bootstrap

Suggested commands:
- `sageo auth login gsc`
- `sageo auth status`
- `sageo auth logout gsc`

### `gsc`
Purpose: Search Console data retrieval

Suggested commands:
- `sageo gsc sites list`
- `sageo gsc sites use <property>`
- `sageo gsc query pages`
- `sageo gsc query keywords`
- `sageo gsc opportunities`

### `serp`
Purpose: live search result lookup using exactly one configured provider

Suggested commands:
- `sageo serp analyze --query "..." --dry-run`
- `sageo serp compare --query "..." --competitor example.com`

### `opportunities`
Purpose: combine local crawl/audit + GSC + optional SERP evidence into machine-readable opportunities

Suggested commands:
- `sageo opportunities pages`
- `sageo opportunities keywords`
- `sageo opportunities answers`

## What Phase 3 should actually deliver

### 1. Search Console integration
The agent should be able to ask:
- what queries are already getting impressions?
- which pages have low CTR but decent ranking?
- which queries/pages are underperforming?

### 2. SERP validation with one provider
The agent should be able to ask:
- what currently ranks for this query?
- which competitor pages dominate?
- does this query show strong SERP features that affect optimization strategy?

### 3. Opportunity outputs
The CLI should produce machine-readable opportunity objects, for example:
- page opportunity
- keyword opportunity
- answer opportunity

Each should include:
- target page/query
- evidence
- confidence
- impact estimate
- effort estimate (even if heuristic)
- source metadata
- estimated external API cost for the run

## What Phase 3 should not do

- no dashboard
- no visual analytics app
- no auto-writing content with embedded LLMs by default
- no giant multi-provider SEO suite
- no backlink intelligence platform yet
- no “query everything” workflows

## What changes because this is AI-agent-first

The documentation and command design should now assume:
- commands are invoked by agents programmatically
- outputs must be stable and deterministic
- costs must be predictable before execution
- every expensive workflow should be previewable first

## Documentation updates required in this phase

Update:
- `README.md`
- `ARCHITECTURE.md`
- `CLAUDE.md`
- `CHANGELOG.md`

And explicitly document:
- free vs paid commands
- cost-estimation behavior
- caching behavior
- approval threshold behavior
- GSC-first / SERP-second strategy

## Recommended implementation order

1. Add phase-3 documentation and command roadmap
2. Add provider/auth configuration model for GSC + one SERP provider
3. Implement cost-estimate and dry-run contract types before live calls
4. Implement GSC auth + property selection
5. Implement GSC query/opportunity commands
6. Implement one SERP provider adapter only
7. Implement agent-facing opportunity commands that merge sources
8. Add caching and freshness metadata
9. Add tests for cost preview, no-call dry-run, and deterministic JSON outputs

## Simple rule for decision-making

If a feature does not help an AI agent:
- discover evidence,
- estimate cost,
- and decide the next best action,

it probably does **not** belong in phase 3.
