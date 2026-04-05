# `sageo login` Command

## Context

Sageo is designed for AI agents to call, but the human still needs to set up credentials once. Currently you need to know exact config key names (`sageo config set serp_api_key ...`). The GG framework pattern is better: run `login`, it shows you what to configure, you paste keys, done.

Key insight: this is a **human-run** command, not an agent command. The agent never calls `sageo login`. So it should be interactive and user-friendly, not JSON-envelope output.

## Design

### `sageo login`

One command. Shows a numbered list of services. User picks one (or "all"). For each:

- **Google Search Console**: Needs `gsc_client_id` + `gsc_client_secret` (prompted), then runs OAuth flow
- **SerpAPI**: Needs `serp_api_key` (prompted, pasted)

After all keys are saved, shows a summary of what's configured.

### Flow

```
$ sageo login

  Sageo CLI — Login

  Services:
    1. Google Search Console (OAuth)
    2. SerpAPI (API key)

  Select a service (1-2, or 'all'): 1

  GSC Client ID: <paste>
  GSC Client Secret: <paste>
  Opening browser for authorization...
  ✓ Google Search Console authenticated

  Select a service (1-2, or 'all', 'done' to finish): done

  ✓ Setup complete
```

### `sageo logout`

Shortcut for `sageo auth logout` — clears all stored credentials and keys.

### What changes

- Add `sageo login` as a **top-level** command (not under `auth`) — it's the one humans will remember
- Add `sageo logout` as a top-level command
- Keep `sageo auth` subcommands for programmatic/agent use (status checks, etc.)
- Remove the "cost approval threshold" from the login flow — it's a config setting, not a credential. Agent docs can explain it.

### Implementation

- Interactive prompts via `bufio.Scanner` reading from stdin
- Mask secret input with `****` after entry (print redacted confirmation)  
- Save keys to config via existing `config.Set()` + `config.Save()`
- GSC OAuth reuses existing `loginGSC()` logic
- Works in terminals only (checks `os.Stdin` is a terminal)

## Steps

1. Create `internal/cli/commands/login.go` with `NewLoginCmd` implementing the interactive service selector, key prompting for SerpAPI, and GSC OAuth delegation — reading from stdin via `bufio.Scanner` and saving credentials through the existing config and auth store APIs.
2. Create `internal/cli/commands/logout.go` with `NewLogoutCmd` that clears auth tokens via `auth.FileTokenStore.Delete` and sensitive config keys (serp_api_key, gsc_client_id, gsc_client_secret) via config reset and save.
3. Wire `login` and `logout` as top-level commands in `internal/cli/root.go` and update `internal/cli/root_test.go` to include them in the expected command list.
4. Verify with `go build ./...`, `go vet ./...`, and `go test ./...`.
