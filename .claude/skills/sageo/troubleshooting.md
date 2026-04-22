# Sageo Troubleshooting

Common failure modes and their fixes. All errors come through the JSON envelope at `.error.code`.

## `AUTH_REQUIRED` / `AUTH_FAILED`

Missing or expired credentials.

- GSC: `sageo auth login gsc`, then `sageo gsc sites use <property>`
- DataForSEO: `sageo login` (interactive) or set `SAGEO_DATAFORSEO_LOGIN` + `SAGEO_DATAFORSEO_PASSWORD`
- PSI: falls back to GSC OAuth; for higher limits set `SAGEO_PSI_API_KEY`
- LLM (recommendations draft, aeo responses): `SAGEO_ANTHROPIC_API_KEY` or `SAGEO_OPENAI_API_KEY`

Verify with `sageo auth status`.

## `APPROVAL_REQUIRED`

The command's estimated cost exceeds `SAGEO_APPROVAL_THRESHOLD_USD` (or `approval_threshold_usd` in config).

- Re-run with `--dry-run` and show the user the estimate.
- Raise the threshold: `sageo config set approval_threshold_usd 5.00`
- Pass `--approve` on `sageo run` to pre-approve the whole pipeline.

## `ESTIMATE_FAILED`

Cost estimator could not compute a price (usually missing provider config).

- Run `sageo config show`; confirm provider, login, and base URL are set.
- Ensure `serp_provider` is `dataforseo` or `serpapi`.

## Empty GSC results

GSC only returns data for properties the authenticated account has access to, with a ~2 to 3 day data lag.

- Confirm the property: `sageo gsc sites list`, then `sageo gsc sites use <url>`
- Dates default to the last 28 days; widen with `--start-date` / `--end-date`.

## `recommendations list` returns nothing

Recommendations are populated by `analyze` (and `sageo run`).

- Run `sageo status`; check `sources_used` vs `sources_missing`.
- Fewer sources = fewer merge rules fire = fewer recommendations. Add PSI, GSC, Labs to unlock more rules.
- After ingesting new data, re-run `sageo analyze`.

## `recommendations draft` skips rows

`draft` only fills rows where `recommended_value` is empty AND the `change_type` is draftable (title, meta_description, h1, h2_add, schema_add, body_expand, internal_link_add, tldr_add, list_format, author_byline, freshness_refresh, entity_consistency).

- `--limit N` bounds cost. Run `--dry-run` first.
- Speed / backlink / indexability recommendations have no copy to draft.

## `recommendations review` shows stale drafts

Drafts stay `pending_review` until a human decides. Bulk-approve low-priority only with `--auto-approve-under-priority N`. Never use `sageo run --auto-approve-all` for client-facing output: it bypasses the gate entirely.

## Forecast is flagged `uncalibrated` or `insufficient_data`

Calibration needs outcome history. Thresholds:

- Per-change-type calibration kicks in at 20 paired observations on this site.
- Below the overall floor, forecasts are flagged `uncalibrated: true` and `confidence_label: insufficient_data`.

Fix: run more cycles (`sageo run`, make changes, `sageo run` again, `sageo compare`). Every `compare` where a recommendation was addressed AND both snapshots have paired GSC data appends an `ObservedLift` to `.sageo/calibration.json`. Until then, present tier + caveats and do not quote specific click numbers.

## `compare` reports no addressed recommendations

Detection is per-ChangeType. An "addressed" flag requires observable proof in the later snapshot: the audit finding cleared, PSI crossed the good-band threshold, schema appears in the crawl, referring-domain count grew, etc.

- Verify the later run actually re-ran the source: `jq '.data.sources_used' <(sageo status)`.
- For speed/schema/indexability changes, the site must have been re-crawled after the fix shipped.

## `sageo run --resume` restarts from scratch

`--resume` picks up after `state.pipeline_cursor`. If the cursor is empty (fresh state or corruption), it runs everything.

- Check: `jq '.pipeline_cursor' .sageo/state.json`
- Force-skip completed stages with `--skip`.

## Stale cached responses

Paid responses are cached on disk with a TTL. To bypass:

- Delete the relevant cache directory (under `~/.cache/sageo/` or the project cache path).
- Or wait out the TTL. Check `metadata.cached` and `metadata.fetched_at` to confirm hits.

## Missing / broken snapshot

Snapshots are written atomically during `sageo run`. If a run crashed mid-write you may see a partial directory.

- List: `sageo snapshots list`
- Inspect: `sageo snapshots show <ref>`; if metadata is unparseable, the snapshot is broken. Prune with `sageo snapshots prune --confirm` after confirming.
- Recovery: re-run `sageo run`; previous snapshots are never overwritten.

## `report pdf` prints a deprecation warning

Expected. `report pdf` is an alias for `report html`; switch to `sageo report html --output ./report.html --open` and use the browser's Cmd+P / Ctrl+P "Save as PDF".

## `go test` fails with "live tests"

Integration tests that hit paid APIs are gated behind `SAGEO_LIVE_TESTS=1` and do not run by default.

- Unit tests: `go test ./...`
- Live tests (opt in): `SAGEO_LIVE_TESTS=1 go test ./...`

See `TESTING.md` for the full matrix.

## JSON envelope parsing

Always parse with `jq`, never grep:

```bash
sageo status | jq '.data.sources_missing'
sageo recommendations list --top 5 | jq '.data[] | {type, url, priority}'
sageo run https://example.com --dry-run | jq '.metadata.estimated_cost'
sageo compare --format json | jq '.data.addressed[]'
```

If a command errors, `jq '.error'` returns `{code, message, details}`.
