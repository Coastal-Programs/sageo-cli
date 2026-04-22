---
name: sageo
description: Run SEO, AEO, and GEO audits with the Sageo CLI (crawl, audit, GSC, PSI, SERP, Labs, backlinks, AI brand mentions, recommendations, snapshots, compare, HTML report). Use when the user asks to audit a website, analyse Core Web Vitals, track keyword rankings, check AI Overview or brand mentions across ChatGPT / Claude / Gemini / Perplexity, diff two runs, or generate a client-ready HTML report with the `sageo` command-line tool.
allowed-tools: Bash, Read, Write, Edit, Grep, Glob
---

# Sageo CLI Playbook

Sageo is a single-binary Go CLI that audits a website across SEO, AEO, and GEO. Every command emits a JSON envelope. Paid commands are cost-estimated and budget-gated. State lives at `.sageo/state.json`; per-run history lives at `.sageo/snapshots/<utc-ts>/`.

**Working directory**: commands assume the repo root of the project being audited (where `.sageo/` lives).

## Before anything else

1. Binary available: `sageo --help` (or `go run ./cmd/sageo --help` from the sageo repo).
2. Credentials: `sageo auth status`. If anything is missing, walk the user through `sageo login` (interactive). Do not paste secrets into files.
3. Project state: `sageo status`. If absent, run `sageo init --url <site>` (optionally `--brand "Name,alias"`).

## Output contract

Every command returns this envelope on stdout. Rely on it, never scrape text:

```json
{
  "success": true,
  "data": { ... },
  "error": null,
  "metadata": {
    "estimated_cost": 0.0006, "currency": "USD",
    "requires_approval": false, "cached": false,
    "source": "dataforseo", "fetched_at": "2026-04-22T...",
    "dry_run": false
  }
}
```

Parse with `jq`. Default format is `json`. Use `-o text` / `-o table` only when the user wants human output. `sageo compare` defaults to `text` (override with `--format json`). `sageo report html` writes an `.html` file; stdout stays JSON.

## The recommended flow

For almost every "audit this site" request, use this exact sequence. Do not skip steps 2 and 3 — without an active GSC property, every forecast collapses to `priority_tier: unknown` and recommendations lose their search-volume signal.

```bash
# 1. Project setup
sageo init --url https://example.com --brand "Example,example.com"

# 2. GSC auth (once per machine / per Google account)
sageo auth login gsc

# 3. Select the active GSC property for this project
sageo gsc sites list                   # shows properties you have access to
sageo gsc sites use https://example.com/   # MANDATORY before sageo run

# 4. Full autonomous pipeline
sageo run https://example.com --budget 10

# 5. Human review gate (REQUIRED before any client-facing output)
sageo recommendations review

# 6. Render the HTML deliverable
sageo report html --open
# (with no --output the file lands in .sageo/reports/sageo-report-<UTC-timestamp>.html)
```

**Do not run `sageo audit run` standalone before `sageo run`.** `sageo run` already includes the audit stage. The standalone `sageo audit run` subcommand is for isolated re-audits only.

`sageo run` stages (run in order): `crawl → audit → gsc → psi → serp → labs → backlinks → aeo → merge → recommend → draft → forecast`, ending at a `review_gate` stage that is non-interactive unless `--no-review` / `--auto-approve-all` is passed. Every run writes a snapshot under `.sageo/snapshots/<utc-ts>/` (disable with `--no-snapshot`) and mirrors the rendered report to `.sageo/reports/latest.html`.

**If you see `!! WARNING: no GSC property configured` in stderr when starting `sageo run`, stop immediately.** That warning means the flow above was not followed. Ctrl-C, run step 3, then restart step 4. Continuing past the warning produces a useless audit.

Key `run` flags:
- `--budget <USD>`: hard ceiling for total paid spend (0 = no cap)
- `--dry-run`: estimate per stage, call no paid APIs
- `--skip a,b` / `--only a,b`: stage filters
- `--resume`: continue after the last completed stage (`pipeline_cursor` in state)
- `--approve`: pre-approve cost gates (no mid-run prompts)
- `--no-review`: leave drafts in `pending_review` (a later `recommendations review` is still required before shipping anything)
- `--prompts <file>`: newline-delimited prompts for AEO (otherwise derived from GSC + Labs)
- `--retain N` / `--retain-within DURATION`: snapshot retention after this run

**Repeat-run flow** (tracking whether recommendations worked):

```bash
sageo run https://example.com --budget 5 --approve
sageo compare                          # latest vs previous snapshot
sageo compare --output-html ./diff-$(date +%F).html
```

`compare` infers which earlier recommendations were "addressed" via per-ChangeType detectors (cleared audit finding, PSI crossing the good-band threshold, schema appearing in the crawl, referring-domain growth). When both snapshots have paired GSC data for an addressed recommendation, an `ObservedLift` record is appended to `.sageo/calibration.json`. Subsequent forecasts use this history.

## Honest framing rules

Agents presenting Sageo output to humans MUST follow these. They are not disclaimers; they are how the tool behaves.

1. **Quote priority tiers, not specific click numbers.** The primary signal is `priority_tier`: `high`, `medium`, `low`, `unknown`. If a range is quoted, quote it as a range with the tier attached. "High tier, estimated 200 to 500 clicks/mo, calibrated against 42 prior outcomes on this site." Never "you will get 347 more clicks". On cold projects without calibration data, the tier may be derived from the rule-engine priority score (>=80 High, >=50 Medium, else Low) and is rendered with a `(provisional)` badge; treat these as directional until historical outcomes accumulate.
2. **Surface caveats every time.** The forecaster emits `caveats[]` like `low_search_volume`, `short_history`, `insufficient_data`, `uncalibrated`. Pass these through to the user.
3. **Observational data is not causal.** `compare` output is correlational. Algorithm updates, seasonality, and concurrent work are not controlled for. Say so.
4. **LLM drafts are a starting point.** `recommended_value` written by `recommendations draft` is `pending_review`. Never present it as publish-ready. Require `sageo recommendations review` before it hits a report.
5. **When calibration is thin, say so.** If `calibration_samples < MinSampleOverall` (20) or the forecast is flagged `uncalibrated: true`, explicitly state confidence is low. Do not fabricate certainty.
6. **Tier recommendations by evidence strength.** Every ChangeType maps to a section in `docs/research/ai-citation-signals-2026.md`. Some are `confirmed` (Google schema for AIO). Some are `likely` (direct-answer blocks, backlinks). Some are `unclear` (FAQPage, `llms.txt`). Never upgrade "likely" to "proven".

## State, snapshots, and calibration

- **State file** `.sageo/state.json`: single source of truth for the current run. Per-source keys: `gsc`, `psi`, `serp`, `labs`, `backlinks`, `aeo`, `mentions`, `findings`, `merged_findings`, `recommendations`, `pipeline_cursor`, `pipeline_runs`, `history`. Inspect with `sageo status` or `jq`. Never hand-edit.
- **Snapshots** `.sageo/snapshots/<utc-ts>/`: a frozen copy of state + recommendations + HTML report + metadata, written atomically by `sageo run`. Previous runs are never overwritten. Manage with `sageo snapshots list|show|path|prune`.
- **Calibration** `.sageo/calibration.json`: append-only `ObservedLift` records written by `compare` when a recommendation was addressed and both snapshots have paired GSC data. The forecaster reads this on every `recommendations forecast` call; thresholds: per-ChangeType calibration at 20 samples, overall at a conservative floor, otherwise `confidence_label = insufficient_data`.
- **Scope**: Sageo is single-site per working directory. A cross-site (global) registry is not yet shipped; do not reference `sageo sites` or a `portfolio` command.

## Cost tiers

| Tier | Cost | Commands |
|---|---|---|
| Free | $0 | `crawl`, `audit`, `psi`, `report html`, `init`, `status`, `analyze`, `compare`, `snapshots *`, `recommendations list/forecast`, `aeo mentions scan` |
| Free + OAuth | $0 | all `gsc *` |
| Paid (micro, ~$0.0006 to $0.02 per call) | paid | `serp *`, `labs *`, `aeo keywords`, `geo *`, `aeo mentions search/top-*`, `opportunities --with-serp` |
| Paid (LLM, provider-priced) | paid | `aeo responses`, `recommendations draft` |
| Paid deposit ($100/mo) | paid | `backlinks *` |

**Rules for paid commands:**

- Always run `--dry-run` first. Show `metadata.estimated_cost` to the user before the real call.
- Never retry a failed paid call automatically. Surface the error and ask.
- Prefer `sageo run <url> --budget N` over stacking individual paid commands.

## Review gate (non-negotiable for client work)

LLM drafts never auto-apply to reports. The review workflow:

```bash
sageo recommendations review                       # interactive TUI
sageo recommendations review --format json         # for agent pipelines
sageo recommendations review --type title --url https://example.com/pricing
sageo recommendations review --auto-approve-under-priority 30  # bulk-approve low-priority only
```

Only `approved` or `edited` recommendations render in `sageo report html`. `pending_review` shows a badge; `rejected` is excluded.

`sageo run --auto-approve-all` exists but is labelled UNSAFE. Do not use it for any output the user will share.

## HTML report (primary output)

```bash
sageo report html --output ./report.html --open
sageo report html --logo ./logo.png --brand-color "#0A6AFF" --appendix --title "Acme Audit"
```

Self-contained `.html` with inlined CSS and minimal JS, works offline. For a PDF, open in any modern browser and press Cmd+P (macOS) or Ctrl+P (Linux/Windows), then "Save as PDF". `sageo report pdf` exists as a deprecated alias that prints a warning and runs `html`; prefer `report html`.

## Full reference

- Command table with every flag: [commands.md](commands.md)
- Recipes mapped to common intents (Core Web Vitals, easy wins, AI Overviews, backlink gap, repeat-run deltas): [workflows.md](workflows.md)
- Failure modes (AUTH, APPROVAL, empty GSC, thin calibration, stale cache, missing snapshots): [troubleshooting.md](troubleshooting.md)
- Evidence base for every ChangeType: `docs/research/ai-citation-signals-2026.md` in the repo
- Architecture, testing, configuration: `README.md`, `CLAUDE.md`, `ARCHITECTURE.md`, `TESTING.md`

## Defaults to remember

- Locale: Australia (DataForSEO `location_code: 2036`, language `en`). Override per command for other markets.
- Config file: `~/.config/sageo/config.json` (override with `SAGEO_CONFIG`).
- Secrets via env: `SAGEO_DATAFORSEO_LOGIN`, `SAGEO_DATAFORSEO_PASSWORD`, `SAGEO_PSI_API_KEY`, `SAGEO_ANTHROPIC_API_KEY`, `SAGEO_OPENAI_API_KEY`, `SAGEO_SERP_API_KEY`, `SAGEO_GSC_CLIENT_ID/SECRET`. See `CLAUDE.md` for the complete list.
- Cost approval threshold: `SAGEO_APPROVAL_THRESHOLD_USD` env or `approval_threshold_usd` config. Above threshold requires `--approve` or an interactive confirmation.

## Validation

After any code change in this repo, run before committing:

```bash
go vet ./...
go test ./...
```

Full gate: `make fmt && make vet && make test && make lint`.

## Don't

- Don't invent commands. If `sageo <group> --help` does not list it, it does not exist.
- Don't run paid commands without `--dry-run` first and surfacing the cost.
- Don't edit `.sageo/state.json`, `.sageo/calibration.json`, or any snapshot by hand. The CLI owns the schema.
- Don't present a specific monthly click number as a prediction. Quote the tier and the range, with caveats.
- Don't ship LLM-drafted copy to a client report without `recommendations review`.
- Don't embed Anthropic/OpenAI keys in config files. Env vars only.
