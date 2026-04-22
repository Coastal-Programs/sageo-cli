---
name: sageo
description: Run SEO, AEO, and GEO audits with the Sageo CLI (crawl, audit, GSC, PSI, SERP, Labs, backlinks, AI brand mentions, recommendations, PDF report). Use when the user asks to audit a website, analyse Core Web Vitals, track keyword rankings, check AI Overviews / brand mentions across ChatGPT / Claude / Gemini / Perplexity, or generate an SEO recommendation report with the `sageo` command-line tool.
---

Sageo is a single-binary Go CLI. Every command emits a JSON envelope (`success`, `data`, `error`, `metadata`). State lives at `.sageo/state.json`. Paid commands include `estimated_cost`, `currency`, `cached`, `source`, `fetched_at`, `dry_run` in metadata and are budget-gated.

## The one-shot (prefer this)

```bash
sageo login                                    # once per machine
sageo init --url https://example.com --brand "Example,example.com"
sageo run https://example.com --budget 5       # crawl→audit→gsc→psi→serp→labs→backlinks→aeo→merge→recommend→draft→forecast
sageo recommendations list --top 20
sageo report pdf --output ./report.pdf
```

Flags on `sageo run`: `--budget`, `--skip a,b`, `--only a,b`, `--resume`, `--approve`, `--dry-run`, `--prompts <file>`, `--max-pages`.

## Cost tiers

- **Free**: `crawl`, `audit`, `psi`, `report`, `init`, `status`, `analyze`, `recommendations list/forecast`
- **Free + OAuth**: all `gsc *`
- **Paid micro (~$0.0006–0.02/call)**: `serp *`, `labs *`, `aeo keywords`, `geo *`, `opportunities --with-serp`
- **Paid LLM**: `aeo responses`, `aeo mentions search/top-*`, `recommendations draft`
- **Paid deposit ($100/mo)**: `backlinks *`

## Rules

- Always `--dry-run` paid commands first and surface `metadata.estimated_cost` to the user.
- Never retry failed paid calls automatically.
- Always check `sageo status` for `sources_missing` before `analyze`.
- Never hand-edit `.sageo/state.json` — the CLI owns the schema.
- Default locale is Australia (DataForSEO location_code 2036, `en`).
- Config at `~/.config/sageo/config.json` (override via `SAGEO_CONFIG`). Secrets live in env: `SAGEO_DATAFORSEO_LOGIN/PASSWORD`, `SAGEO_PSI_API_KEY`, `SAGEO_ANTHROPIC_API_KEY`, `SAGEO_OPENAI_API_KEY`.

## Command groups (verify with `sageo <group> --help`)

`aeo {models,responses,keywords,mentions {scan,search,top-pages,top-domains}}` ·
`analyze` · `audit run` · `auth {login,logout,status}` · `backlinks {summary,list,referring-domains,competitors,gap}` · `config {get,set,show,path}` · `crawl run` · `geo {mentions,top-pages}` · `gsc {sites,query {pages,keywords,trends,devices,countries,appearances},opportunities}` · `init` · `labs {ranked-keywords,keywords,overview,competitors,keyword-ideas,search-intent,bulk-difficulty}` · `login` · `logout` · `opportunities` · `psi run` · `recommendations {list,draft,forecast}` · `report {generate,list,pdf}` · `run <url>` · `serp {analyze,compare,batch}` · `status` · `version`

## Validation after code changes

```bash
go vet ./...
go test ./...
```

Full gate: `make fmt && make vet && make test && make lint`.

## Full reference

See `.claude/skills/sageo/SKILL.md` and sibling `commands.md`, `workflows.md`, `troubleshooting.md` for the complete playbook.
