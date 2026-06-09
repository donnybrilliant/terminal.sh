# CLAUDE.md — terminal.sh

Read this before editing anything in this repo. It is the contract between
the autonomous agent fleet (`donnybrilliant/terminal-sh-agents`) and the human reviewer.

## What this repo is

`terminal.sh` is a Go / Bubble Tea hacking-simulation game. Players connect
over SSH (`:2222`) or WebSocket (`:8080`) to a Bubble Tea TUI, hack
procedurally-generated servers, mine credits, and complete missions. State is
persisted via GORM (SQLite by default; Postgres in production).

## Stack

- **Go 1.25+** (pinned in `go.mod`; check `go-version-file` in CI).
- **Bubble Tea v2** + **Lipgloss** for the TUI.
- **Wish** for the SSH transport (`charmbracelet/ssh` middleware stack).
- **GORM** + **`github.com/glebarez/sqlite`** (pure-Go SQLite; `CGO_ENABLED=0` builds work).
- **Cobra-style** subcommand layout under `cmd/` (`ssh`, `web`, `all`).

## Package layout

```
cmd/             # Binary entry points. `cmd/all` is the combined SSH+Web
                 # binary used by docker-compose; `cmd/ssh` and `cmd/web`
                 # build single-purpose binaries. See @cmd/README.md.
services/        # Business logic: shop, mining, missions, chat,
                 # exploitation, achievements, etc. One file per domain.
                 # Every service constructor takes a `*database.Database`
                 # and returns a struct with explicit method receivers —
                 # no global state. See @services/README.md.
models/          # GORM models — pure schema, no behaviour. Each model gets
                 # a `BeforeCreate` hook for UUID generation. JSON columns
                 # use `gorm:"type:text;serializer:json"`.
terminal/        # Bubble Tea models, login flow, shell, renderer,
                 # input helpers. `terminal/ssh/` and `terminal/websocket/`
                 # are the two transports.
auth/            # Password hashing (bcrypt), JWT issuance and validation.
                 # **High blast radius** — every PR here must be reviewed
                 # by a human regardless of size.
filesystem/      # Virtual filesystem (`vfs.go`) and read helpers for
                 # procedurally-generated server filesystems.
config/          # Env-var loader. Single source of truth for ports, DB
                 # path, JWT secret, host-key path.
database/        # GORM init, migrations, seeders for the procgen content.
patch/           # Save-state migrations between game versions.
```

## Conventions

- **Errors**: always wrap with `fmt.Errorf("...: %w", err)`. Never swallow.
  Never `panic` on user input — return an error and let the caller decide.
- **Logging**: use `fmt.Println` only at startup banners. Inside services
  prefer the standard `log` package or pass a logger; do NOT add a new
  global logger singleton.
- **Concurrency**: SSH sessions are per-goroutine; never share `*tea.Model`
  across sessions. Anything writing the DB must hold its own `*gorm.DB`
  transaction or use `db.Transaction(...)`.
- **No new global state.** If a service needs configuration, take it as a
  constructor argument. The only sanctioned globals are in `config/` (the
  loaded `*Config`) and the SSH server reference in `terminal/ssh/server.go`
  (for graceful shutdown only).
- **GORM model changes** must include a migration entry in
  `database/database.go` and a `BeforeCreate` UUID hook if a new model.
- **Bubble Tea**: 80x24 is the canonical viewport for agents. New TUI code
  must work without resizing — agents never send `WindowSizeMsg` mid-session.

## Testing

- Run `go test ./... -race -count=1` locally before pushing. CI re-runs this
  via `.github/workflows/regression-persona.yml`.
- Test files live next to the code (`services/foo.go` ↔ `services/foo_test.go`).
- For TUI behaviour, use **teatest** (`github.com/charmbracelet/x/exp/teatest`).
  See `services/action_tracker_test.go` for the current pattern.
- Coverage is intentionally low today — when the engineer agent adds tests,
  prefer one tight reproducer over broad coverage.

## Review checklist (auto-applied by the engineer agent)

The agent reads this list before opening a PR. Reviewers should re-check.

- [ ] No new global state introduced outside `config/`.
- [ ] Every error is wrapped with `%w` or explicitly logged + returned.
- [ ] No `panic()` on any code path reachable from user input.
- [ ] No SQL string-concat — GORM query builder only.
- [ ] If the PR touches `auth/`: zero bypasses of `auth.ValidateToken` /
      `auth.HashPassword`. Reviewer: read the diff line-by-line.
- [ ] If the PR touches `terminal/`: TUI handles 80x24 without truncation.
- [ ] If the PR touches `models/`: migration added to `database/database.go`
      and a `BeforeCreate` UUID hook for new tables.
- [ ] `go test ./... -race` passes locally.

## Engineer-agent budget rules

The autonomous engineer agent (`anthropics/claude-code-action@v1`) is
invoked by `.github/workflows/ai-scaffold.yml` on Issues labelled
`ai-scaffold`. It follows the rules in that workflow's prompt:

1. **Draft PRs only.** Never auto-merge.
2. **Size cap.** If the fix would touch >3 files or >200 lines, the agent
   STOPS and opens a PR containing only a failing test that reproduces the
   bug (tagged `needs-human-design`).
3. **Reproducer.** Every Issue body has a `## Reproducer` block with
   `AGENT_SEED`, the command sequence, and a screen hash. The engineer agent
   must start by replicating that reproducer.
4. **Model.** Sonnet 4.6 by default; Opus 4.7 only when the reproducer's
   path touches `auth/`, `services/payment/` (reserved), or `terminal/`.

## CI secrets (repository settings)

| Secret | Used by |
|--------|---------|
| `ANTHROPIC_API_KEY_ENGINEER` | `ai-scaffold.yml` engineer agent |
| `ANTHROPIC_API_KEY_REGRESSION` | `regression-persona.yml` replay subagent |
| `TERMINAL_SH_AGENTS_CHECKOUT_TOKEN` | `regression-persona.yml` — PAT with read access to private `terminal-sh-agents` |

## GitHub labels

| Label | Purpose |
|-------|---------|
| `ai-scaffold` | Triggers engineer agent; marks fleet-opened Issues/PRs |
| `severity:bug` | Analyst triage |
| `severity:ux` | Analyst triage |
| `severity:balance` | Analyst triage |
| `severity:feature` | Analyst triage |
| `needs-human-design` | Engineer agent stopped at size cap |
