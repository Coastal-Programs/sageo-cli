# Sageo Workflow Recipes

Map common user intents to the right sequence of commands. Prefer `sageo run` when the intent is "audit everything".

## 1. "Audit my site end-to-end"

```bash
sageo login                                    # once per machine
sageo init --url https://example.com --brand "Example,example.com"
sageo run https://example.com --budget 5
sageo report pdf --output ./report.pdf
```

Review: `sageo recommendations list --top 20`.

## 2. "Find easy-win keywords" (no AEO, no backlinks)

```bash
sageo init --url https://example.com
sageo audit run --url https://example.com --depth 2 --max-pages 50
sageo auth login gsc && sageo gsc sites use https://example.com/
sageo gsc query keywords --limit 100
sageo labs bulk-difficulty --from-gsc --dry-run
# review cost, then:
sageo labs bulk-difficulty --from-gsc
sageo analyze
sageo recommendations list --type title --top 20
```

The merge engine emits `easy-win-keyword` findings when Labs difficulty is low and GSC position is in 4–20.

## 3. "How does my Core Web Vitals story look?"

```bash
sageo psi run --url https://example.com --strategy mobile
sageo psi run --url https://example.com/pricing --strategy mobile
sageo analyze                                  # merges PSI with GSC high-traffic pages
sageo recommendations list --type speed
```

Rule `slow-core-web-vitals` fires when PSI is poor AND the page has meaningful GSC clicks.

## 4. "Am I losing to AI Overviews?"

```bash
sageo serp batch --keywords "kw1,kw2,kw3,..." --dry-run
sageo serp batch --keywords "kw1,kw2,kw3,..."
sageo analyze
sageo recommendations list --type body
```

Look for `ai-overview-eating-clicks` and `featured-snippet-opportunity` recommendations.

## 5. "Track brand visibility across AI engines"

```bash
sageo init --url https://example.com --brand "Example,Example Co,example.com"
sageo aeo responses --prompt "best seo tools" --all --tier flagship
sageo aeo responses --prompt "how to audit a website" --all --tier flagship
sageo aeo mentions scan                        # free local analysis of stored responses
sageo aeo mentions search --keyword "seo tools"   # paid — only if user wants cross-engine DataForSEO view
```

## 6. "Backlink gap vs competitors"

Requires DataForSEO Backlinks subscription ($100/mo deposit).

```bash
sageo backlinks summary --target example.com --dry-run
sageo backlinks summary --target example.com
sageo backlinks competitors --target example.com
sageo backlinks gap --target example.com       # auto-loads competitors from state
sageo analyze
sageo recommendations list --type backlink
```

## 7. "Client-ready PDF only"

Assuming prior `analyze` + `recommendations draft` + `forecast` have run:

```bash
sageo report pdf \
  --output ./acme-seo-report.pdf \
  --logo ./acme-logo.png \
  --brand-color '#1E40AF' \
  --appendix
```

## 8. "Resume a run that failed halfway"

```bash
sageo status                                   # check pipeline_cursor
sageo run https://example.com --resume --budget 5
```

## 9. "Dry-run everything first"

```bash
sageo run https://example.com --dry-run
```

Returns per-stage `estimated_cost` and total. Show the total to the user before executing without `--dry-run`.
