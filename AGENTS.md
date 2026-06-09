# AGENTS.md — terminal.sh (fleet integration)

The autonomous agent fleet lives in the sibling repo **`terminal-sh-agents`**
([`donnybrilliant/terminal-sh-agents`](https://github.com/donnybrilliant/terminal-sh-agents)).

## Running agents

```bash
cd ../terminal-sh-agents
uv run python -m terminal_sh_agents.run --provider claude --persona newbie --cycles 1
```

Providers: `claude`, `codex`, `cursor`. All share stdio MCP tools and one observation pipeline.

Concurrent runs:

```bash
uv run python -m terminal_sh_agents.run \
  --provider claude,codex --persona newbie,speedrunner \
  --concurrency 2 --cycles 1 --start-game
```

## Issue → draft PR loop

1. Fleet Analyst files Issues with label `ai-scaffold` + `severity:*`.
2. `.github/workflows/ai-scaffold.yml` opens a **draft** PR (never auto-merge).
3. `.github/workflows/regression-persona.yml` replays the Issue reproducer on the PR branch.

## CI secrets (this repo)

| Secret | Purpose |
|--------|---------|
| `ANTHROPIC_API_KEY_ENGINEER` | Engineer agent (`ai-scaffold.yml`) |
| `ANTHROPIC_API_KEY_REGRESSION` | Regression replay subagent |
| `TERMINAL_SH_AGENTS_CHECKOUT_TOKEN` | PAT to check out private `terminal-sh-agents` in CI |

## Cursor automation

`.cursor/automations/agent-fleet.yaml` runs a smoke session every 6 hours.
