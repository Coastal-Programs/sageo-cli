# Sageo Command Reference

Every command emits a JSON envelope (`success`, `data`, `error`, `metadata`). Paid commands add `estimated_cost`, `currency`, `requires_approval`, `cached`, `source`, `fetched_at`, `dry_run` to metadata.

Verify any command before suggesting it: `sageo <group> --help`.

Cost tier legend: **Free**, **Free+OAuth**, **Paid** (DataForSEO micro), **Paid LLM** (Anthropic/OpenAI), **Paid deposit** (DataForSEO Backlinks, $100/mo).

## Project

| Command | Tier | Purpose |
|---|---|---|
| `sageo init --url https://example.com [--brand "Name,alias"]` | Free | Create `.sageo/` project state for a site |
| `sageo status` | Free | Show `sources_used`, `sources_missing`, `pipeline_cursor` |
| `sageo analyze` | Free | Merge all stored sources into findings + recommendations |
| `sageo version` | Free | Print version and build metadata |

## Auth and config

| Command | Tier | Purpose |
|---|---|---|
| `sageo login` | Free | Interactive setup: GSC OAuth, DataForSEO, Anthropic, OpenAI, PSI |
| `sageo auth status` | Free | Report which credentials are present |
| `sageo auth login gsc` | Free+OAuth | GSC OAuth browser flow |
| `sageo auth logout gsc` | Free | Revoke GSC token |
| `sageo logout` | Free | Wipe all stored credentials |
| `sageo config show` | Free | Print config (sensitive fields redacted) |
| `sageo config get <key>` | Free | Read a single config value |
| `sageo config set <key> <value>` | Free | Write a single config value |
| `sageo config path` | Free | Print resolved config file path |
| `sageo provider list` / `sageo provider use <name>` | Free | Manage the HTTP fetcher provider registry |

## Crawl, audit, report

| Command | Tier | Purpose |
|---|---|---|
| `sageo crawl run --url <site> --depth 2 --max-pages 50` | Free | BFS crawl, persisted to state |
| `sageo audit run --url <site> --depth 2 --max-pages 50 [--skip-psi]` | Free | Crawl + SEO rule audit (also runs PSI for top pages unless skipped) |
| `sageo report generate --url <site>` | Free | Legacy stored report (JSON) |
| `sageo report list` | Free | List generated reports |
| `sageo report html [--output PATH] [--open] [--appendix] [--logo PATH] [--brand-color #HEX] [--title STR]` | Free | Primary output: self-contained client-ready HTML. PDF via browser Cmd+P. Default output: `.sageo/reports/sageo-report-<UTC-timestamp>.html` when a project exists, else `./sageo-report.html`. `sageo run` also mirrors the snapshot report to `.sageo/reports/latest.html` |
| `sageo report pdf ...` | Free | DEPRECATED alias for `report html`. Emits a warning. Use `report html` directly |

## Google Search Console

| Command | Tier | Purpose |
|---|---|---|
| `sageo gsc sites list` | Free+OAuth | Enumerate GSC properties on the authed account |
| `sageo gsc sites use <property>` | Free+OAuth | Pin the active property for this project |
| `sageo gsc query pages [--query T] [--type web|image|video|news|discover|googleNews] [--start-date YYYY-MM-DD] [--end-date YYYY-MM-DD] [--limit N] [--page /path]` | Free+OAuth | Top pages by clicks/impressions |
| `sageo gsc query keywords [same flags]` | Free+OAuth | Top queries |
| `sageo gsc query trends / devices / countries / appearances [--type T]` | Free+OAuth | Slice dimensions |
| `sageo gsc opportunities` | Free+OAuth | Ranking-but-not-clicking seed list |

## PageSpeed Insights

| Command | Tier | Purpose |
|---|---|---|
| `sageo psi run --url <page> --strategy mobile|desktop` | Free | Core Web Vitals (LCP, CLS, FCP, TBT, SI); uses GSC OAuth if no `SAGEO_PSI_API_KEY` |

## SERP

| Command | Tier | Purpose |
|---|---|---|
| `sageo serp analyze --query "<term>" [--dry-run]` | Paid | Single-query SERP with 9 feature types (AIO, FS, PAA, Local, KG, Top Stories, Videos, Shopping, Images) |
| `sageo serp compare --query "a" --query "b" [--dry-run]` | Paid | Side-by-side feature diff across queries |
| `sageo serp batch --keywords "k1,k2,..." [--dry-run]` | Paid | DataForSEO Standard queue, ~$0.0006/kw, up to 100 |

## Labs (DataForSEO)

| Command | Tier | Purpose |
|---|---|---|
| `sageo labs ranked-keywords --target example.com [--dry-run]` | Paid | Keywords a domain currently ranks for |
| `sageo labs keywords --target example.com [--limit 50]` | Paid | Keyword dataset for a target |
| `sageo labs overview --target example.com` | Paid | Domain-level summary |
| `sageo labs competitors --target example.com [--limit 20]` | Paid | Organic competitors (persisted to state) |
| `sageo labs keyword-ideas --keyword "<seed>" [--limit 50]` | Paid | Ideas from a seed |
| `sageo labs search-intent --keywords "k1,k2,k3"` | Paid | Intent classification |
| `sageo labs bulk-difficulty --from-gsc [--dry-run]` | Paid | KD for keywords pulled from state (GSC) |
| `sageo labs bulk-difficulty --keywords "k1,k2" [--dry-run]` | Paid | KD for an explicit list |

## Backlinks (DataForSEO, $100/mo deposit)

| Command | Tier | Purpose |
|---|---|---|
| `sageo backlinks summary --target example.com [--dry-run]` | Paid deposit | Profile metrics |
| `sageo backlinks list --target example.com [--limit 50] [--dofollow-only]` | Paid deposit | Raw backlink rows |
| `sageo backlinks referring-domains --target example.com` | Paid deposit | Referring domains |
| `sageo backlinks competitors --target example.com` | Paid deposit | Backlink competitors |
| `sageo backlinks gap --target example.com [--competitors "a.com,b.com"]` | Paid deposit | Gap analysis; auto-loads competitors from state if omitted |

## AEO

| Command | Tier | Purpose |
|---|---|---|
| `sageo aeo models` | Free | DataForSEO AI model catalogue (cached 7d) |
| `sageo aeo responses --prompt "..." --engine chatgpt|claude|gemini|perplexity [--dry-run]` | Paid LLM | Query one engine |
| `sageo aeo responses --prompt "..." --all --tier flagship` | Paid LLM | Fan out to every engine |
| `sageo aeo responses --prompt "..." --models gpt-5,claude-sonnet-4-6` | Paid LLM | Specific models |
| `sageo aeo keywords --keyword "<term>" [--location "United States"]` | Paid | AI search volume |
| `sageo aeo mentions scan` | Free | Local scan of stored responses for brand terms (Layer A) |
| `sageo aeo mentions search --keyword "<term>"` | Paid | DataForSEO LLM Mentions search (Layer B) |
| `sageo aeo mentions top-pages --keyword "<term>"` | Paid | Pages AI engines cite for the keyword |
| `sageo aeo mentions top-domains --keyword "<term>"` | Paid | Domains AI engines cite for the keyword |

## GEO

| Command | Tier | Purpose |
|---|---|---|
| `sageo geo mentions --keyword "<term>" --domain example.com [--dry-run]` | Paid | Generative-engine domain mentions |
| `sageo geo top-pages --keyword "<term>"` | Paid | Top pages cited by generative engines |

## Opportunities (legacy, superseded by merge)

| Command | Tier | Purpose |
|---|---|---|
| `sageo opportunities` | Free+OAuth | GSC-based opportunity detection |
| `sageo opportunities --with-serp --serp-queries 10 [--dry-run]` | Paid | Adds SERP evidence |

## Recommendations

| Command | Tier | Purpose |
|---|---|---|
| `sageo recommendations list [--top 20] [--url PAGE] [--type title|meta|h1|h2|schema|body|speed|backlink|indexability]` | Free | Read stored recommendations sorted by priority |
| `sageo recommendations draft [--limit 20] [--provider anthropic|openai] [--type T] [--url U] [--dry-run]` | Paid LLM | Fill empty `recommended_value` with LLM copy. Output starts `pending_review` |
| `sageo recommendations review [--type T] [--url U] [--reviewer NAME] [--auto-approve-under-priority N] [--format interactive|json]` | Free | MANDATORY human gate before reports. Approve / edit / reject each draft |
| `sageo recommendations forecast` | Free | Attach priority tier and click-delta range; reads `.sageo/calibration.json` |

## Autonomous pipeline

| Command | Tier | Purpose |
|---|---|---|
| `sageo run <url> [--budget USD] [--max-pages N] [--skip a,b] [--only a,b] [--resume] [--approve] [--dry-run] [--prompts FILE] [--no-review] [--no-snapshot] [--retain N] [--retain-within 2160h]` | Varies | Full pipeline: `crawl → audit → gsc → psi → serp → labs → backlinks → aeo → merge → recommend → draft → forecast`. Writes a snapshot |

`--auto-approve-all` exists but is labelled UNSAFE; never use for client-facing output.

## Snapshots and compare

| Command | Tier | Purpose |
|---|---|---|
| `sageo snapshots list` | Free | Snapshots newest-first with stage, cost, outcome |
| `sageo snapshots show <ref>` | Free | Metadata for one snapshot (`latest`, `previous`, or timestamp prefix) |
| `sageo snapshots path <ref>` | Free | Absolute path (pipe-friendly) |
| `sageo snapshots prune [--keep 20] [--within 2160h] [--confirm]` | Free | Retention prune; dry-run unless `--confirm` |
| `sageo compare [--from REF] [--to REF] [--format text|json] [--output-html PATH]` | Free | Diff two snapshots; infers addressed recommendations; appends observed lift to `.sageo/calibration.json` |

Snapshots live at `.sageo/snapshots/<utc-ts>/` and are written atomically. Previous runs are never overwritten.

## Note on multi-site

There is currently no `sageo sites` command group and no cross-site registry. Sageo is single-site per working directory. A global registry is discussed in `BRAINDUMP.md` as future work; do not reference it as if it exists.
