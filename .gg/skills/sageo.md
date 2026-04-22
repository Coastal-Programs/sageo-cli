---
name: sageo
description: Run SEO, AEO, and GEO audits with the Sageo CLI (crawl, audit, GSC, PSI, SERP, Labs, backlinks, AI brand mentions, recommendations, snapshots, compare, HTML report). Use when the user asks to audit a website, analyse Core Web Vitals, track keyword rankings, check AI Overviews / brand mentions across ChatGPT / Claude / Gemini / Perplexity, diff two runs, or generate a client-ready report with the `sageo` command-line tool.
---

Sageo is a single-binary Go CLI. Every command emits a JSON envelope (`success`, `data`, `error`, `metadata`). State lives at `.sageo/state.json`. Per-run history lives at `.sageo/snapshots/<utc-ts>/`. Paid commands include `estimated_cost`, `currency`, `cached`, `source`, `fetched_at`, `dry_run` in metadata and are budget-gated.

## Recommended flow (use this first)

```bash
sageo login                                    # once per machine
sageo init --url https://example.com --brand "Example,example.com"
sageo run https://example.com --budget 10      # full pipeline, writes a snapshot
sageo recommendations review                   # REQUIRED gate before any client output
sageo report html --output ./report.html --open
```

HTML is the primary output. For PDF: open the `.html` in a browser and press Cmd+P / Ctrl+P, "Save as PDF". `sageo report pdf` still works but is a deprecated alias.

## Repeat-run flow

```bash
sageo run https://example.com --budget 5 --approve
sageo compare                                  # latest vs previous snapshot
sageo compare --output-html ./diff.html
```

`compare` infers which earlier recommendations were addressed and appends observed lift to `.sageo/calibration.json`. Subsequent forecasts sharpen.

## Honest framing rules (non-negotiable)

1. Quote **priority tiers** (`high` / `medium` / `low` / `unknown`), not specific click numbers. If a range is included, phrase it as a range with the tier and calibration sample count.
2. Always surface caveats (`low_search_volume`, `short_history`, `insufficient_data`, `uncalibrated`).
3. `compare` output is observational, not causal. Say so.
4. LLM drafts start as `pending_review`. Never present as publish-ready without `sageo recommendations review`.
5. When `calibration_samples < 20` or `uncalibrated: true`, say confidence is low.
6. Each ChangeType maps to a section in `docs/research/ai-citation-signals-2026.md`. Do not upgrade "likely" evidence to "proven".

## Cost tiers

- **Free**: `crawl`, `audit`, `psi`, `report html`, `init`, `status`, `analyze`, `compare`, `snapshots *`, `recommendations list/forecast/review`, `aeo mentions scan`
- **Free + OAuth**: all `gsc *`
- **Paid micro (~$0.0006 to $0.02/call)**: `serp *`, `labs *`, `aeo keywords`, `aeo mentions search/top-*`, `geo *`, `opportunities --with-serp`
- **Paid LLM**: `aeo responses`, `recommendations draft`
- **Paid deposit ($100/mo)**: `backlinks *`

Always `--dry-run` paid commands first and surface `metadata.estimated_cost`. Never retry failed paid calls automatically.

## Command groups (verify with `sageo <group> --help`)

`aeo {models, responses, keywords, mentions {scan, search, top-pages, top-domains}}` · `analyze` · `audit run` · `auth {login, logout, status}` · `backlinks {summary, list, referring-domains, competitors, gap}` · `compare` · `config {get, set, show, path}` · `crawl run` · `geo {mentions, top-pages}` · `gsc {sites, query {pages, keywords, trends, devices, countries, appearances}, opportunities}` · `init` · `labs {ranked-keywords, keywords, overview, competitors, keyword-ideas, search-intent, bulk-difficulty}` · `login` · `logout` · `opportunities` · `provider {list, use}` · `psi run` · `recommendations {list, draft, review, forecast}` · `report {generate, list, html, pdf}` · `run <url>` · `serp {analyze, compare, batch}` · `snapshots {list, show, path, prune}` · `status` · `version`

Sageo is single-site per working directory. There is no `sites` group and no global registry shipped yet.

## Don't

- Don't invent commands. If `sageo <group> --help` does not list it, it does not exist.
- Don't run paid commands without `--dry-run` first.
- Don't hand-edit `.sageo/state.json`, `.sageo/calibration.json`, or snapshots.
- Don't quote specific click numbers. Tier + range + caveats, always.
- Don't ship LLM drafts to a client report without `recommendations review`.
- Don't use `sageo run --auto-approve-all` for anything a user will share.

## Validation after code changes

```bash
go vet ./...
go test ./...
```

Full gate: `make fmt && make vet && make test && make lint`.

## Full reference

See `.claude/skills/sageo/SKILL.md` and sibling `commands.md`, `workflows.md`, `troubleshooting.md` for the complete playbook.
