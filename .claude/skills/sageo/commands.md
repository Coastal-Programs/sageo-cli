# Sageo Command Reference

Every command emits a JSON envelope (`success`, `data`, `error`, `metadata`). Paid commands add `estimated_cost`, `currency`, `requires_approval`, `cached`, `source`, `fetched_at`, `dry_run` to metadata.

Verify a command exists before suggesting it: `sageo <group> --help`.

## Project

```bash
sageo init --url https://example.com [--brand "Name,alias"]
sageo status
sageo analyze                       # merge all sources → findings + recommendations
sageo version
```

## Auth & config

```bash
sageo login                         # interactive: GSC OAuth, DataForSEO, API keys
sageo auth status
sageo auth login gsc
sageo auth logout gsc
sageo logout                        # wipe all credentials

sageo config show                   # sensitive fields redacted
sageo config get <key>
sageo config set <key> <value>
sageo config path
```

## Crawl, audit, report (free)

```bash
sageo crawl run --url <site> --depth 2 --max-pages 50
sageo audit run --url <site> --depth 2 --max-pages 50 [--skip-psi]
sageo report generate --url <site>
sageo report list
sageo report pdf [--output ./report.pdf] [--logo ./logo.png] [--brand-color '#1E40AF'] [--appendix]
```

`audit run` automatically runs PSI for top pages unless `--skip-psi`.

## Google Search Console (free + OAuth)

```bash
sageo gsc sites list
sageo gsc sites use https://example.com/

sageo gsc query pages       [--start-date YYYY-MM-DD] [--end-date YYYY-MM-DD] [--query <term>] [--type web|image|video|news|discover|googleNews] [--limit N] [--page <path>]
sageo gsc query keywords    [flags as above]
sageo gsc query trends      [--type web]
sageo gsc query devices     [--type web]
sageo gsc query countries   [--type web]
sageo gsc query appearances [--type web]

sageo gsc opportunities
```

## PageSpeed Insights (free, optional key)

```bash
sageo psi run --url <page> --strategy mobile|desktop
```

Uses GSC OAuth token if `psi_api_key` not set.

## SERP (paid — DataForSEO or SerpAPI)

```bash
sageo serp analyze --query "<term>" [--dry-run]
sageo serp compare --query "term1" --query "term2" [--dry-run]
sageo serp batch   --keywords "k1,k2,..." [--dry-run]   # DataForSEO Standard queue, ~$0.0006/kw, up to 100
```

Detects 9 SERP features: Featured Snippets, PAA, AI Overviews, Local Pack, Knowledge Graph, Top Stories, Inline Videos/Shopping/Images.

## Labs (paid — DataForSEO)

```bash
sageo labs ranked-keywords --target example.com [--dry-run]
sageo labs keywords        --target example.com [--limit 50]
sageo labs overview        --target example.com
sageo labs competitors     --target example.com [--limit 20]
sageo labs keyword-ideas   --keyword "<seed>" [--limit 50]
sageo labs search-intent   --keywords "k1,k2,k3"
sageo labs bulk-difficulty --from-gsc [--dry-run]       # auto-loads kws from state
sageo labs bulk-difficulty --keywords "k1,k2" [--dry-run]
```

## Backlinks (paid, $100/mo deposit — DataForSEO)

```bash
sageo backlinks summary           --target example.com [--dry-run]
sageo backlinks list              --target example.com [--limit 50] [--dofollow-only]
sageo backlinks referring-domains --target example.com
sageo backlinks competitors       --target example.com
sageo backlinks gap               --target example.com [--competitors "a.com,b.com"]   # auto-loads from state if omitted
```

## AEO — Answer Engine Optimization

```bash
sageo aeo models                                                 # catalogue (cached 7d)
sageo aeo responses   --prompt "..." --engine chatgpt|claude|gemini|perplexity [--dry-run]
sageo aeo responses   --prompt "..." --all --tier flagship       # fan out to every engine
sageo aeo responses   --prompt "..." --models gpt-5,claude-sonnet-4-6
sageo aeo keywords    --keyword "<term>" [--location "United States"]

sageo aeo mentions scan                                          # local scan of stored responses (free)
sageo aeo mentions search      --keyword "<term>"                # DataForSEO LLM Mentions
sageo aeo mentions top-pages   --keyword "<term>"
sageo aeo mentions top-domains --keyword "<term>"
```

Supported engines: `chatgpt`, `claude`, `gemini`, `perplexity`.

## GEO — Generative Engine Optimization (paid — DataForSEO)

```bash
sageo geo mentions  --keyword "<term>" --domain example.com [--dry-run]
sageo geo top-pages --keyword "<term>"
```

## Opportunities

```bash
sageo opportunities
sageo opportunities --with-serp --serp-queries 10 [--dry-run]    # paid when --with-serp
```

## Recommendations

```bash
sageo recommendations list     [--top 20] [--url <page>] [--type title|meta|h1|h2|schema|body|speed|backlink|indexability]
sageo recommendations draft    [--limit 20] [--provider anthropic|openai] [--dry-run]
sageo recommendations forecast                                   # attach monthly click-lift (AWR 2024 CTR curve)
```

## Autonomous pipeline

```bash
sageo run <url> [--budget USD] [--max-pages 100]
                [--skip backlinks,aeo] [--only crawl,audit,gsc,psi]
                [--resume] [--approve] [--dry-run]
                [--prompts /path/to/prompts.txt]
```

Stages (in order): `crawl → audit → gsc → psi → serp → labs → backlinks → aeo → merge → recommend → draft → forecast`.

State records `pipeline_cursor` after each stage; `--resume` picks up after it.
