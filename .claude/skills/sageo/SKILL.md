---
name: sageo
description: Run SEO, AEO, and GEO audits with the Sageo CLI (crawl, audit, GSC, PSI, SERP, Labs, backlinks, AI brand mentions, recommendations, PDF report). Use when the user asks to audit a website, analyse Core Web Vitals, track keyword rankings, check AI Overview / brand mentions across ChatGPT / Claude / Gemini / Perplexity, or generate an SEO recommendation report with the `sageo` command-line tool.
allowed-tools: Bash, Read, Write, Edit, Grep, Glob
---

# Sageo CLI Playbook

Sageo is a single-binary Go CLI that runs SEO / AEO / GEO analysis against a website. Every command emits a JSON envelope. Paid commands are cost-estimated and budget-gated.

**Repo working directory**: commands assume you're in the project root. State lives at `.sageo/state.json`.

## Before anything else

1. Confirm the binary is available: `sageo --help` (or `go run ./cmd/sageo --help` from the repo).
2. Confirm credentials: `sageo auth status`. If missing, walk the user through `sageo login` (interactive) — do not paste secrets into files.
3. Confirm project state: `sageo status`. If absent, run `sageo init --url <site>` (optionally `--brand "Name,alias"`).

## Output contract

Every command returns this envelope on stdout — rely on it, never scrape text:

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "metadata": {
    "estimated_cost": 0.0006,
    "currency": "USD",
    "requires_approval": false,
    "cached": false,
    "source": "dataforseo",
    "fetched_at": "2026-04-22T...",
    "dry_run": false
  }
}
```

Parse with `jq`. Default format is `json`; only pass `-o text` / `-o table` when the user wants human output.

## Cost tiers

| Tier | Cost | Commands |
|---|---|---|
| Free | $0 | `crawl`, `audit`, `psi`, `report`, `init`, `status`, `analyze`, `recommendations list/forecast` |
| Free + OAuth | $0 | all `gsc *` |
| Paid (micro) | ~$0.0006–0.02/call | `serp *`, `labs *`, `aeo keywords`, `geo *`, `opportunities --with-serp` |
| Paid (LLM) | provider-priced | `aeo responses`, `aeo mentions search/top-*`, `recommendations draft` |
| Paid (deposit) | $100/mo | `backlinks *` |

**Rules for paid commands:**

- Always run `--dry-run` first. Show the user `estimated_cost` from metadata before the real call.
- Never retry a failed paid call automatically — surface the error and ask.
- For pipeline runs, prefer `sageo run <url> --budget <USD>` over stacking individual commands.

## The one-shot: `sageo run`

For almost every "audit this site" request, prefer this single command:

```bash
sageo run https://example.com --budget 5
```

It executes: crawl → audit → GSC → PSI → SERP → Labs → backlinks → AEO → merge → recommend → draft → forecast, capped at `--budget` USD.

Useful flags:
- `--dry-run` — estimate cost across every stage, execute nothing
- `--skip backlinks,aeo` — drop expensive stages
- `--only crawl,audit,gsc,psi` — run just the free-ish core
- `--resume` — pick up from `pipeline_cursor` in state
- `--approve` — pre-approve all cost gates (no prompts mid-run)
- `--prompts <file>` — newline-delimited prompts for AEO fan-out (otherwise derived from GSC+Labs)

After a run completes, always do:
```bash
sageo recommendations list --top 20
sageo report pdf --output ./report.pdf
```

## Manual workflow (when `run` is too blunt)

```
Audit checklist:
- [ ] sageo init --url <site>
- [ ] sageo audit run --url <site> --depth 2 --max-pages 50
- [ ] sageo auth login gsc   (if GSC data wanted)
- [ ] sageo gsc sites use <property>
- [ ] sageo gsc query pages / keywords / trends
- [ ] sageo psi run --url <site> --strategy mobile
- [ ] sageo serp batch --keywords "k1,k2,..." --dry-run  → approve → rerun
- [ ] sageo labs bulk-difficulty --from-gsc --dry-run    → approve → rerun
- [ ] sageo backlinks summary --target <domain>          (if Backlinks enabled)
- [ ] sageo aeo responses --prompt "..." --all --tier flagship
- [ ] sageo aeo mentions scan
- [ ] sageo analyze
- [ ] sageo recommendations list --top 20
- [ ] sageo recommendations draft --limit 20
- [ ] sageo recommendations forecast
- [ ] sageo report pdf
```

For the full command reference with every flag, see [commands.md](commands.md).
For recipes mapped to common user intents ("improve Core Web Vitals", "find easy wins", "track AI visibility"), see [workflows.md](workflows.md).
For common failure modes (missing OAuth, approval gate, cache misses, empty GSC), see [troubleshooting.md](troubleshooting.md).

## State file — `.sageo/state.json`

Single source of truth. Every data source writes into its own key: `gsc`, `psi`, `serp`, `labs`, `backlinks`, `aeo`, `mentions`, `findings`, `merged_findings`, `recommendations`, `pipeline_cursor`, `pipeline_runs`, `history`.

Quick inspection:
```bash
jq '.data' < <(sageo status)
jq 'keys' .sageo/state.json
jq '.recommendations | length' .sageo/state.json
```

`sageo status` reports `sources_used` and `sources_missing` — always check this before running `analyze` or `recommendations`. Missing sources produce fewer merge rules.

## Defaults to remember

- Locale defaults to **Australia** (DataForSEO `location_code: 2036`, language `en`). Override per command if the user is targeting a different market.
- Config file: `~/.config/sageo/config.json` (override with `SAGEO_CONFIG`).
- Env overrides for every provider key: `SAGEO_DATAFORSEO_LOGIN`, `SAGEO_DATAFORSEO_PASSWORD`, `SAGEO_PSI_API_KEY`, `SAGEO_ANTHROPIC_API_KEY`, etc. See `CLAUDE.md` for the full list.
- Cost approval threshold: `SAGEO_APPROVAL_THRESHOLD_USD` (env) or `approval_threshold_usd` (config). Commands above threshold require `--approve` or interactive confirmation.

## Validation

After any code change in this repo, run before committing:

```bash
go vet ./...
go test ./...
```

Full gate: `make fmt && make vet && make test && make lint`.

## Don't

- Don't invent commands. If you don't see it in `commands.md` or `sageo <group> --help`, it doesn't exist.
- Don't run paid commands without first running them with `--dry-run` and surfacing the cost.
- Don't edit `.sageo/state.json` by hand — use the CLI (it owns the schema).
- Don't embed OpenAI/Anthropic keys in config files — use env vars.
- Don't add a second paid provider for a capability that already has one.
