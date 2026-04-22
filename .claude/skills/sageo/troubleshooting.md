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
- Or pass `--approve` on `sageo run` to pre-approve the whole pipeline.

## `ESTIMATE_FAILED`

Cost estimator couldn't compute a price (usually missing provider config).

- Run `sageo config show` — confirm provider, login, and base URL are set.
- Ensure `serp_provider` is `dataforseo` or `serpapi`.

## Empty GSC results

GSC only returns data for properties the authenticated account has access to, and only from ~2–3 days ago.

- Confirm the property: `sageo gsc sites list` → `sageo gsc sites use <url>`
- Dates default to the last 28 days; widen with `--start-date` / `--end-date`.

## `recommendations list` returns nothing

Recommendations are populated by `analyze` (and `sageo run`).

- Run `sageo status` — check `sources_used` vs `sources_missing`.
- Fewer sources = fewer merge rules fire = fewer recommendations. Add PSI, GSC, Labs to unlock more rules.
- After ingesting new data, re-run `sageo analyze`.

## `recommendations draft` skips rows

`draft` only fills rows where `recommended_value` is empty and the `type` is draftable (title, meta, h1, h2, body, schema).

- Use `--limit N` to bound cost. Run `--dry-run` first.
- Speed / backlink / indexability recommendations have no copy to draft.

## `sageo run --resume` restarts from scratch

`--resume` picks up after `state.pipeline_cursor`. If the cursor is empty (e.g. fresh state or corrupted), it runs everything.

- Check with `jq '.pipeline_cursor' .sageo/state.json`.
- Force-skip completed stages with `--skip`.

## Stale cached responses

Paid responses are cached on disk with a TTL. To bypass:

- Delete the relevant cache directory (under `~/.cache/sageo/` or the project cache path).
- Or rerun after TTL expiry. Check `metadata.cached` and `metadata.fetched_at` to confirm cache hits.

## `go test` fails with "live tests"

Integration tests that hit paid APIs are gated behind `SAGEO_LIVE_TESTS=1`. They should not run by default.

- Unit tests: `go test ./...`
- Live tests (opt in): `SAGEO_LIVE_TESTS=1 go test ./...`

See `TESTING.md` for the full matrix.

## JSON envelope parsing

Always parse with `jq`, never grep:

```bash
sageo status | jq '.data.sources_missing'
sageo recommendations list --top 5 | jq '.data[] | {type, url, priority}'
sageo run https://example.com --dry-run | jq '.metadata.estimated_cost'
```

If a command errors, `jq '.error'` will have `{code, message, details}`.
