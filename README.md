<div align="center">
<pre>
███████╗ █████╗  ██████╗ ███████╗ ██████╗      ██████╗██╗     ██╗
██╔════╝██╔══██╗██╔════╝ ██╔════╝██╔═══██╗    ██╔════╝██║     ██║
███████╗███████║██║  ███╗█████╗  ██║   ██║    ██║     ██║     ██║
╚════██║██╔══██║██║   ██║██╔══╝  ██║   ██║    ██║     ██║     ██║
███████║██║  ██║╚██████╔╝███████╗╚██████╔╝    ╚██████╗███████╗██║
╚══════╝╚═╝  ╚═╝ ╚═════╝ ╚══════╝ ╚═════╝      ╚═════╝╚══════╝╚═╝
</pre>

<p align="center">
  <a href="https://go.dev/">
    <img src="https://img.shields.io/badge/go-%3E%3D1.26-00ADD8.svg" alt="Go Version">
  </a>
  <a href="https://github.com/Coastal-Programs/sageo-cli/actions">
    <img src="https://github.com/Coastal-Programs/sageo-cli/actions/workflows/test.yml/badge.svg" alt="CI">
  </a>
  <a href="https://github.com/Coastal-Programs/sageo-cli/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="License">
  </a>
</p>
</div>

A command-line tool that audits a website across SEO, AEO (Answer Engine Optimization), and GEO (Generative Engine Optimization) surfaces, generates evidence-backed recommendations for what to change, drafts the new copy with an LLM, forecasts the likely impact in priority tiers, and tracks results across runs. One Go binary, JSON-first output, state persisted under `.sageo/`.

## Who it is for

- SEO consultants who want one auditable pipeline instead of ten browser tabs.
- Agency operators who need scriptable output that feeds into their own reporting, client deliverables, or agent workflows.
- Technical marketers comfortable on a command line, comfortable with OAuth, comfortable reading JSON.

It is **not** a self-serve dashboard. There is no web UI, no hosted service, no account. If you expect a button that says "run audit", use a different tool.

## What it does

- **Crawl + audit.** BFS crawl with rule-based SEO checks and a 0 to 100 score.
- **Google Search Console.** OAuth2 login, query pages/keywords, opportunity seeds, pagination.
- **PageSpeed Insights.** Core Web Vitals (LCP, CLS, FCP, TBT, SI) for mobile and desktop.
- **SERP analysis.** Nine feature types including AI Overviews, Featured Snippets, PAA, via SerpAPI or DataForSEO. Batch queueing for up to 100 keywords per call.
- **Labs keyword data.** Difficulty, volume, intent, ranked keywords, competitor domains, keyword ideas.
- **Backlinks.** Profile summary, referring domains, competitor overlap, gap analysis.
- **Multi-model AEO.** Fan prompts across ChatGPT, Claude, Gemini, Perplexity in parallel; store responses keyed by prompt.
- **Brand mention detection.** Layer A is an offline scan of stored AEO responses for your brand terms. Layer B is the DataForSEO LLM Mentions API for domain share and top-cited pages.
- **Recommendations with LLM-drafted copy.** Merge rules emit atomic `Recommendation` objects with evidence and priority. An Anthropic or OpenAI driver fills `recommended_value` with concrete copy validated against SERP length limits.
- **Priority-tiered forecasts.** Each recommendation carries a `high`, `medium`, `low`, or `unknown` tier plus an estimated click-delta range. Calibrated against historical outcomes when enough data exists; otherwise flagged uncalibrated.
- **HTML report.** Self-contained, styled, client-ready HTML file (cover, exec summary, per-source findings, recommendation cards, sortable forecast table, optional appendix). Users print to PDF via the browser.
- **Snapshot history.** Every `sageo run` is frozen under `.sageo/snapshots/<utc-timestamp>/` with state, recommendations, report, and metadata. Atomic writes; previous runs are never overwritten.
- **Cross-run comparison.** `sageo compare` diffs two snapshots and infers which earlier recommendations were addressed.
- **Forecaster calibration.** Observed lift from addressed recommendations is appended to `.sageo/calibration.json` so subsequent forecasts sharpen over time.

## Quickstart

Zero to a reviewed HTML report in four commands:

```bash
sageo init --url https://example.com --brand "Example,example.com"
sageo run https://example.com --budget 10
sageo recommendations review
sageo report html --output ./report.html --open
```

`sageo init` creates `.sageo/state.json` and records your brand terms. `sageo run` drives the full pipeline (crawl, audit, GSC, PSI, SERP, Labs, backlinks, AEO, mentions, merge, recommendations, draft, forecast) under a hard budget ceiling. `sageo recommendations review` walks an interactive TUI so a human approves, edits, or rejects each LLM draft before it ships. `sageo report html` renders the final artefact.

To get a PDF, open the HTML file in any modern browser and press Cmd+P (macOS) or Ctrl+P (Linux/Windows), then choose "Save as PDF".

### First-time credentials

```bash
sageo login
```

This walks through GSC OAuth, DataForSEO login/password, PSI key, Anthropic key, and OpenAI key. Credentials land in `~/.config/sageo/config.json` with mode 0600. Run `sageo auth status` to verify what is configured.

### Estimating cost before you spend

Every paid command and the top-level pipeline support `--dry-run`:

```bash
sageo run https://example.com --dry-run
sageo serp batch --keywords keywords.txt --dry-run
sageo aeo responses --prompt "best CRMs 2026" --all --dry-run
```

Dry runs print the stage plan, per-stage estimates, and total expected spend without calling any paid API.

## Install

```bash
make install
```

This runs `go install ./cmd/sageo` and places the `sageo` binary in `$(go env GOPATH)/bin`. If `sageo` is not found afterwards, ensure `~/go/bin` is on your `PATH` and reload your shell config:

```bash
export PATH="$HOME/go/bin:$PATH"
```

Verify the install:

```bash
sageo version
```

## Configuration

Config file: `~/.config/sageo/config.json` by default, overridden by `SAGEO_CONFIG` (absolute `.json` path). The complete env-var list:

| Variable | Purpose |
|---|---|
| `SAGEO_CONFIG` | Absolute path to an alternative config JSON file |
| `SAGEO_PROVIDER` | Active HTTP fetcher provider (default `local`) |
| `SAGEO_API_KEY` | API key for the active provider |
| `SAGEO_BASE_URL` | Base URL override for the active provider |
| `SAGEO_ORGANIZATION_ID` | Organisation identifier where applicable |
| `SAGEO_SERP_PROVIDER` | SERP provider (`serpapi` or `dataforseo`) |
| `SAGEO_SERP_API_KEY` | API key for SerpAPI |
| `SAGEO_DATAFORSEO_LOGIN` | DataForSEO account login (also used by Backlinks, Labs, AEO, GEO) |
| `SAGEO_DATAFORSEO_PASSWORD` | DataForSEO account password |
| `SAGEO_APPROVAL_THRESHOLD_USD` | Dollar value above which paid commands require `--approve` |
| `SAGEO_GSC_PROPERTY` | Default Google Search Console property URL |
| `SAGEO_GSC_CLIENT_ID` | Google OAuth client ID for GSC |
| `SAGEO_GSC_CLIENT_SECRET` | Google OAuth client secret for GSC |
| `SAGEO_PSI_API_KEY` | Google PageSpeed Insights API key |
| `SAGEO_LLM_PROVIDER` | LLM driver for drafting (default `anthropic`) |
| `SAGEO_ANTHROPIC_API_KEY` | Anthropic API key for drafting and AEO responses |
| `SAGEO_OPENAI_API_KEY` | OpenAI API key for drafting and AEO responses |

See `internal/common/config/config.go` for the authoritative definition.

Most users set the two DataForSEO credentials and one of the two LLM keys, then use `sageo auth login gsc` for OAuth. The PSI key is optional: without it, PSI falls back to the GSC OAuth token or unauthenticated quota.

### Per-project state

Each project has its own `.sageo/` directory in the working directory:

```
.sageo/
├── state.json              (latest snapshot copy; always up to date)
├── snapshots/
│   ├── index.json          (ordered list of all runs)
│   └── 2026-04-22T10-15-33Z/
│       ├── state.json      (frozen state at run completion)
│       ├── recommendations.json
│       ├── report.html
│       └── metadata.json
└── calibration.json        (observed-lift data points for the forecaster)
```

Do not hand-edit these files. The CLI owns the schema.

## Data tiers

| Tier | Commands | Cost |
|---|---|---|
| Free | `init`, `status`, `analyze`, `crawl run`, `audit run`, `psi run`, `report *`, `recommendations list`, `recommendations forecast`, `snapshots *`, `compare` | $0 |
| Free + OAuth | `gsc *`, `auth *` | $0 (Google account required) |
| Paid micro (~$0.0006 to $0.02/call) | `serp *`, `labs *`, `aeo keywords`, `geo *`, `opportunities --with-serp` | DataForSEO or SerpAPI credit |
| Paid LLM | `aeo responses`, `aeo mentions` (API variants), `recommendations draft` | Anthropic or OpenAI tokens, plus DataForSEO for mentions |
| Paid deposit | `backlinks *` | DataForSEO Backlinks ($100/mo minimum deposit) |

Every paid command supports `--dry-run`. `sageo run` accepts a hard `--budget` ceiling in USD; stages whose estimates would push accumulated spend over the ceiling are skipped with a budget-exceeded outcome.

Approval gate: when a paid command's estimate exceeds `SAGEO_APPROVAL_THRESHOLD_USD`, execution blocks until you pass `--approve` or re-run below the threshold. Default threshold is 0 (no gate).

## Output format

Every command emits a JSON envelope by default:

```json
{
  "success": true,
  "data": { },
  "error": null,
  "metadata": { }
}
```

Paid commands add `estimated_cost`, `currency`, `requires_approval`, `cached`, `source`, `fetched_at`, and `dry_run` to `metadata`. HTML reports are the only non-envelope output: they write a file to disk and emit `{ path, size_bytes }` in `data`. See CLAUDE.md for the complete contract including `recommendations forecast` and multi-model AEO shapes.

Override with `--output text|table` for human reading, or pipe JSON into `jq`:

```bash
sageo recommendations list --top 20 | jq '.data.recommendations[] | {id, change_type, priority}'
```

## Run history and learning

Every `sageo run` writes a timestamped directory under `.sageo/snapshots/` containing the frozen state, recommendations, HTML report, and invocation metadata. The top-level `.sageo/state.json` is always a copy of the latest snapshot.

Manage history:

```bash
sageo snapshots list                 # newest first
sageo snapshots show latest          # metadata for the most recent run
sageo snapshots path previous        # pipe-friendly absolute path
sageo snapshots prune --keep 20      # retain the 20 most recent
```

Compare two runs:

```bash
sageo compare                        # latest vs previous
sageo compare --from 2026-04-01 --to latest
sageo compare --output-html ./diff.html
```

`compare` emits typed deltas for GSC, PSI, SERP, AEO, backlinks, audit findings, and recommendation outcomes. It uses per-ChangeType detectors to infer whether each earlier recommendation was addressed (cleared audit finding, PSI crossing a good-band threshold, referring-domain growth, schema appearing in crawl). Every addressed recommendation with paired GSC data in both snapshots produces an `ObservedLift` appended (never overwritten) to `.sageo/calibration.json`.

The forecaster reads `.sageo/calibration.json` on every `recommendations forecast` call. When enough data points exist, it fits a calibration profile and scales raw model output toward reality; when not, it emits raw estimates flagged `uncalibrated: true`. The longer you use sageo on a site, the sharper its forecasts become for that site.

## Honest framing

Sageo is deliberate about what it does and does not claim:

- **Forecasts are ranges and priority tiers, not specific click numbers.** The primary signal is the tier (`high`, `medium`, `low`, `unknown`). The click range is supporting context, subject to the calibration caveat. Treat a tier of `high` with a 200 to 500 clicks/mo range as "this is probably worth doing", not "you will get 347 more clicks".
- **Observational data is not causal.** `compare` surfaces correlated changes between two snapshots. Algorithm updates, seasonality, and concurrent work on the site are not controlled for. A ranking jump after a recommendation shipped does not prove the recommendation caused it.
- **LLM-drafted copy requires human review.** Every drafted `recommended_value` starts as `pending_review`. The review gate (`sageo recommendations review`) is the contract before anything ships to a client-facing report. Rejected drafts are excluded entirely; pending drafts render with a "Pending review" badge.
- **Recommendations are evidence-backed but not proven on any specific site.** Each ChangeType cites the research section in `docs/research/ai-citation-signals-2026.md` that supports it. No guarantee of outcome on your URL.
- **Short windows and low volumes are called out in caveats.** The forecaster attaches caveats such as "low search volume" or "short history" to its output; the HTML report surfaces them in the recommendation card.

These are not disclaimers tucked at the bottom of a marketing page. They are how the tool behaves. Lean in.

## How recommendations work

A `Recommendation` is the atomic unit of "what to change on the site". Each one is scoped to a single URL, optionally a single query, and carries:

- A `change_type` drawn from a fixed set: `title`, `meta_description`, `h1`, `h2_add`, `schema_add`, `body_expand`, `internal_link_add`, `speed_fix`, `backlink_outreach`, `indexability_fix`, `tldr_add`, `list_format`, `author_byline`, `freshness_refresh`, `entity_consistency`.
- `current_value` and `recommended_value` (the latter drafted by the LLM when eligible).
- `rationale` and an `evidence[]` array naming the source (gsc, psi, serp, labs, backlinks, aeo, crawl, audit), metric, and value that triggered it.
- A `priority` score from 1 to 100 and an optional `effort_minutes`.
- A `forecasted_lift` with raw and calibrated click-delta bounds, a priority tier, a confidence label, and caveats.
- A `review_status`: `pending_review`, `approved`, `edited`, or `rejected`.

Each ChangeType is backed by a section in `docs/research/ai-citation-signals-2026.md`. `ChangeTLDR`, for example, is grounded in the Growth Memo finding that ~44% of ChatGPT citations come from the first 30% of an article. `ChangeAuthorByline` is grounded in Google E-E-A-T and Perplexity Person schema signals. `ChangeSchema` is scoped to Tier-1 types (Organization, Article, BreadcrumbList, Person) where the evidence is strongest.

The merge engine applies 14 cross-source rules to produce the initial recommendation set. The LLM drafter (`recommendations draft`) fills empty `recommended_value` fields. The review gate (`recommendations review`) requires a human decision on every draft. The forecaster (`recommendations forecast`) attaches a tier and range. Only approved or edited recommendations appear in the final HTML report.

## Command reference

```
sageo init              Initialise a .sageo project for a site
sageo status            Show current project state and which sources are populated
sageo login             Interactive credential setup for GSC, DataForSEO, Anthropic, OpenAI
sageo logout            Clear all stored credentials
sageo auth              login | logout | status for individual services (GSC OAuth)
sageo config            get | set | show | path
sageo provider          list | use
sageo crawl run         BFS crawl of a site
sageo audit run         Crawl plus rule-based SEO audit
sageo gsc               sites | query (pages, keywords) | opportunities
sageo psi run           PageSpeed Insights for a URL
sageo serp              analyze | compare | batch
sageo labs              ranked-keywords | keywords | overview | competitors | keyword-ideas | bulk-difficulty
sageo backlinks         summary | list | referring-domains | competitors | gap
sageo aeo               models | responses | keywords | mentions
sageo geo               mentions | top-pages
sageo opportunities     GSC plus optional SERP opportunity detection (legacy, superseded by merge)
sageo analyze           Run cross-source analysis and merge findings
sageo recommendations   list | draft | review | forecast
sageo run <url>         End-to-end autonomous pipeline
sageo snapshots         list | show | path | prune
sageo compare           Diff two snapshots
sageo report            generate | list | html | pdf (pdf is a deprecated alias for html)
sageo version           Print version and build metadata
```

Run `sageo <command> --help` for flags.

## Common workflows

Single-run client audit:

```bash
sageo init --url https://client.example --brand "Client,client.example"
sageo auth login gsc
sageo run https://client.example --budget 15
sageo recommendations review
sageo report html --output ./client-report.html --brand-color "#0A6AFF" --logo ./client-logo.png --open
```

Scheduled weekly refresh (CI-friendly, no prompts):

```bash
sageo run https://client.example --budget 5 --approve --no-review
sageo compare --output-html ./diff-$(date +%F).html
```

Pair `--approve` with `--budget` so a runaway cost cannot happen. Pair `--no-review` with a follow-up interactive `sageo recommendations review` before shipping anything client-facing.

Cost-conscious iteration on a single stage:

```bash
sageo run https://client.example --only crawl,audit,gsc,psi    # free stages only
sageo serp batch --keywords ./top-20.txt --dry-run              # see SERP estimate
sageo serp batch --keywords ./top-20.txt --approve              # then run it
```

## Documentation

- [ARCHITECTURE.md](ARCHITECTURE.md): package layout, data flow, recommendation lifecycle, extensibility recipes.
- [CLAUDE.md](CLAUDE.md): conventions for AI agents and contributors working in this repo.
- [TESTING.md](TESTING.md): unit versus integration test rules, test utilities, CI contract.
- [docs/research/ai-citation-signals-2026.md](docs/research/ai-citation-signals-2026.md): cross-engine synthesis backing the recommendation ChangeTypes.

## Contributing

Issues and pull requests welcome. Read CLAUDE.md for conventions (commit message format, language choices, output contract, test safety rules). Run `make precommit` before pushing.

When adding a new data source, a new recommendation ChangeType, a new merge rule, or a new LLM provider, follow the extensibility recipes in ARCHITECTURE.md. Keep to the JSON envelope contract. Do not introduce live network calls into unit tests: `scripts/check-no-live-tests.sh` enforces this and is wired into `make test`.

## Troubleshooting

**`state.json already exists` on `sageo init`.** You have an existing project in this directory. Run `sageo status` to inspect it, or remove `.sageo/` if you want to start over.

**`no project initialized` on `sageo run`.** Run `sageo init --url <site>` first. `sageo run` auto-initialises if the URL matches, but only on the very first invocation.

**`ANTHROPIC_API_KEY not set` on `recommendations draft`.** Either set `SAGEO_ANTHROPIC_API_KEY`, or switch provider with `SAGEO_LLM_PROVIDER=openai` plus `SAGEO_OPENAI_API_KEY`. `sageo login` walks both.

**GSC returns empty results for a property you own.** Confirm the OAuth login used the correct Google account: `sageo auth status`. Re-login with `sageo auth login gsc` if needed. Property URLs must match exactly (trailing slash matters for URL-prefix properties).

**Paid commands keep returning cached data.** The on-disk cache has a TTL per command. Pass `--no-cache` (where supported) or delete the cache directory under `.sageo/cache/` to force a refresh.

**`sageo compare` says "no previous snapshot".** You only have one run so far. Run `sageo run` again to produce a second snapshot.

## License

MIT.
