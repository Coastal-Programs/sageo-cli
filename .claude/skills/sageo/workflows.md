# Sageo Workflow Recipes

Map common user intents to the right command sequence. Prefer `sageo run` when the intent is "audit everything".

## 1. First-run client audit (end-to-end)

```bash
sageo login                                    # once per machine
sageo init --url https://client.example --brand "Client,client.example"
sageo auth login gsc && sageo gsc sites use https://client.example/
sageo run https://client.example --budget 15
sageo recommendations review                   # REQUIRED before sharing anything
sageo report html --output ./client-report.html --brand-color "#0A6AFF" --logo ./client-logo.png --open
```

Hand the `.html` file to the client. They can print to PDF in any browser (Cmd+P or Ctrl+P, then "Save as PDF").

## 2. Repeat run: did our work move the needle?

```bash
sageo run https://client.example --budget 5 --approve
sageo compare                                  # latest vs previous, text summary
sageo compare --output-html ./diff-$(date +%F).html
sageo recommendations review                   # review the new drafts
sageo report html --output ./report-$(date +%F).html
```

`compare` infers which earlier recommendations were addressed and (when paired GSC data exists) appends observed lift to `.sageo/calibration.json`. Future forecasts self-correct.

When presenting `compare` output: say it is observational, not causal. Algorithm updates, seasonality, and concurrent work are not controlled for.

## 3. Find easy-win keywords (no AEO, no backlinks)

```bash
sageo init --url https://example.com
sageo audit run --url https://example.com --depth 2 --max-pages 50
sageo auth login gsc && sageo gsc sites use https://example.com/
sageo gsc query keywords --limit 100
sageo labs bulk-difficulty --from-gsc --dry-run    # review cost
sageo labs bulk-difficulty --from-gsc
sageo analyze
sageo recommendations list --type title --top 20
sageo recommendations forecast
```

Merge rule `easy-win-keyword` fires when Labs difficulty is low and GSC position is 4 to 20.

## 4. Core Web Vitals story

```bash
sageo psi run --url https://example.com --strategy mobile
sageo psi run --url https://example.com/pricing --strategy mobile
sageo analyze
sageo recommendations list --type speed
```

Rule `slow-core-web-vitals` fires when PSI is poor AND the page has meaningful GSC clicks.

## 5. Am I losing to AI Overviews?

```bash
sageo serp batch --keywords "kw1,kw2,kw3,..." --dry-run
sageo serp batch --keywords "kw1,kw2,kw3,..."
sageo analyze
sageo recommendations list --type body
```

Look for `ai-overview-eating-clicks` and `featured-snippet-opportunity` recommendations. Cross-reference the change-type against `docs/research/ai-citation-signals-2026.md` before claiming any specific lift.

## 6. Brand visibility across AI engines

```bash
sageo init --url https://example.com --brand "Example,Example Co,example.com"
sageo aeo responses --prompt "best seo tools" --all --tier flagship
sageo aeo responses --prompt "how to audit a website" --all --tier flagship
sageo aeo mentions scan                        # free local analysis
sageo aeo mentions search --keyword "seo tools"   # paid cross-engine view (optional)
```

## 7. Backlink gap vs competitors (requires $100/mo deposit)

```bash
sageo backlinks summary --target example.com --dry-run
sageo backlinks summary --target example.com
sageo backlinks competitors --target example.com
sageo backlinks gap --target example.com       # auto-loads competitors from state
sageo analyze
sageo recommendations list --type backlink
```

## 8. Scheduled weekly refresh (CI-friendly)

```bash
sageo run https://client.example --budget 5 --approve --no-review
sageo compare --output-html ./diff-$(date +%F).html
```

Pair `--approve` with `--budget` so a runaway cost cannot happen. `--no-review` leaves drafts `pending_review`; do NOT render an HTML report for a client until a human runs `sageo recommendations review`.

## 9. Resume a run that failed halfway

```bash
sageo status                                   # check pipeline_cursor
sageo run https://example.com --resume --budget 5
```

## 10. Dry-run everything first

```bash
sageo run https://example.com --dry-run | jq '.metadata.estimated_cost, .data.per_stage_costs'
```

Surface the total to the user before running without `--dry-run`.

## 11. Debug one stage in isolation

```bash
sageo audit run --url https://example.com --depth 1 --max-pages 20
jq 'keys' .sageo/state.json
jq '.audit' .sageo/state.json
```

Useful when the pipeline's merge output seems wrong: inspect the upstream source key in state before blaming the merge engine.

## 12. Inspect snapshot history

```bash
sageo snapshots list
sageo snapshots show latest
sageo snapshots show previous
sageo snapshots path latest                    # pipe to tar/rsync if archiving
sageo snapshots prune --keep 10 --within 2160h --confirm
```

## Agent-facing framing cheat sheet

When summarising Sageo output for a user, always:

- Quote the priority tier (`high` / `medium` / `low` / `unknown`), not a single click number.
- If a range is included, phrase it as "estimated X to Y clicks/mo, calibrated against N prior outcomes" or "uncalibrated, insufficient data".
- List any `caveats[]` from the forecast (`low_search_volume`, `short_history`, `insufficient_data`).
- For `compare` output, explicitly note it is correlational.
- For LLM drafts, say they need human review before publishing.
- Link the change_type back to `docs/research/ai-citation-signals-2026.md` when the user asks why.
