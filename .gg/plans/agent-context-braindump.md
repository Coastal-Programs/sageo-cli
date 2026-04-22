# Sageo Agent Context — Braindump

## The Core Question

Right now Sageo is a CLI that returns JSON. But who is holding the context between runs? Nobody. Each command is stateless. The AI agent calling it has no memory of what was already done, what was found, or what the plan is.

We need to fix that.

---

## The `.sageo/` Project File Idea

When someone runs Sageo against a site, we drop a `.sageo/` folder in their project (or working directory). This is not for humans — it's for the AI agent.

```
.sageo/
  site.json          # the site being worked on, basic metadata
  crawl.json         # last crawl result snapshot
  audit.json         # last audit result + scores
  gsc.json           # last GSC pull
  serp/              # cached SERP results per keyword
  opportunities.json # current opportunity list + status
  plan.json          # what needs to happen, prioritised
  history.jsonl      # append-only log of every action taken
  context.md         # human-readable summary for the AI to read on startup
```

The AI reads `context.md` first on every session. That tells it:
- What site we're working on
- What we already know
- What we've already done
- What the current plan is
- What's blocked / waiting

---

## The Agent Workflow (How It Should Work)

### 1. Initialise
```
sageo init --url https://example.com
```
Creates `.sageo/site.json`. Stamps the site URL, name, date started.

### 2. Crawl & Understand
```
sageo crawl run
sageo audit run
```
Writes snapshot to `.sageo/crawl.json` and `.sageo/audit.json`. The agent now has a full map of the site — pages, issues, scores.

### 3. Pull Data
```
sageo gsc query pages
sageo gsc query keywords
sageo labs ranked-keywords --target example.com
```
Writes to `.sageo/gsc.json` and SERP/Labs cache. The agent now knows what the site ranks for, what's getting impressions, what's converting.

### 4. Build a Plan
```
sageo plan generate
```
Reads all the context files, scores opportunities, produces a prioritised `plan.json`. This is the AI's to-do list.

### 5. Execute & Record
As the agent acts (fixes a title tag, updates meta, adds a canonical), it logs to `history.jsonl`:
```json
{"ts": "2026-04-06T10:00:00Z", "action": "fix_title", "page": "/about", "before": "About", "after": "About Us | Acme Co", "status": "done"}
```

### 6. Re-audit & Compare
```
sageo audit run
sageo report generate
```
Scores now vs before. Did we move the needle?

---

## The Context File (`context.md`)

Generated automatically. Re-generated after each major action. The AI reads this at the start of every session instead of re-crawling from scratch.

```markdown
# Site: example.com
Last crawl: 2026-04-05 | Pages: 142 | Audit score: 61/100

## What we know
- 23 pages missing meta descriptions
- 8 pages with duplicate H1s
- Top GSC keywords: "seo tools" (pos 14), "audit cli" (pos 7)
- Biggest opportunity: "seo tools" — high volume, position 11-20, no SERP feature captured

## What we've done
- Fixed title tags on 12 pages (2026-04-04)
- Added canonical to all paginated pages (2026-04-03)

## Current plan
1. Write meta descriptions for top 10 traffic pages
2. Fix H1 duplicates on /blog/*
3. Target "seo tools" — content gap vs current rank 3 result
```

---

## Key Questions Still Open

- Do we store `.sageo/` in the user's project repo or in `~/.config/sageo/projects/<hash>/`?
  - In-project = version controllable, visible to the team
  - Centralised = cleaner, no repo pollution

- Does `sageo init` add `.sageo/` to `.gitignore` by default? (probably yes for secrets, no for plan/history)

- How does the AI know to call `sageo` vs do something else? Is there a system prompt we ship?

- Should `history.jsonl` be the basis for a `sageo report diff` command showing before/after on any metric?

- Does the context file get sent to the AI automatically, or does the agent have to explicitly fetch it?

---

## What We're NOT Doing (Yet)

- Embedding an LLM inside the CLI itself
- Auto-executing fixes without agent oversight
- Building a dashboard or web UI
- Storing user data anywhere except local files

---

## Next Steps

1. Decide: in-project `.sageo/` vs centralised `~/.config/sageo/projects/`
2. Design the `context.md` schema more precisely
3. Build `sageo init` command
4. Build `sageo plan generate` command
5. Add `--write-context` flag to crawl/audit/gsc commands to auto-update context files
6. Build `sageo status` — shows current site state in one JSON blob the agent can read
