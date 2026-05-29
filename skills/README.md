# bbx agent skills

These are the canonical [Agent Skills](https://agentskills.io/specification) for
[bbx](https://github.com/rahadiangg/bbx) — markdown contracts (`SKILL.md` with YAML
frontmatter) that route an AI agent's intent to the right `bbx` commands for managing
Atlassian Bamboo. `SKILL.md` is an open standard, so these work with any agent that
understands it: Claude Code, OpenAI Codex CLI, Cursor, OpenCode, Cline, GitHub Copilot,
and more.

This directory is the **single source of truth**. The bbx binary embeds it
(`//go:embed skills`), the npm wrapper bundles it, and the `npx skills` / Claude Code
plugin tooling reads it directly.

## Skills

| Skill | What it does |
|-------|--------------|
| `bbx` | Top-level router — maps intent to bbx commands and dispatches to the specialised skills below. |
| `bbx-setup` | First-time setup: authenticate with a PAT, capture server version, switch Bamboo contexts. |
| `bbx-trigger-build` | Trigger a plan build, poll to completion, report the outcome. |
| `bbx-investigate-build` | Diagnose why a build failed/hung — result detail, comments, labels, logs. |
| `bbx-extract-config` | Export a plan's full configuration (Specs, variables, artifacts, deployments). |

## Install

### Claude Code (native plugin)

```text
/plugin marketplace add rahadiangg/bbx
/plugin install bbx@bbx
```

### Any agent via the `skills` CLI (no bbx binary needed)

```sh
npx skills add rahadiangg/bbx                       # all skills, auto-detect agents
npx skills add rahadiangg/bbx --skill bbx-setup     # just one
npx skills add rahadiangg/bbx -a claude-code -a codex   # target specific agents
```

Discover more at [skills.sh](https://skills.sh).

### Any agent via the bbx binary

```sh
bbx agent skills install --all                  # generic ~/.agents/skills
bbx agent skills install --all --target codex   # ~/.codex/skills
bbx agent skills install --all --target claude-code --target cursor
bbx agent skills install --all --dir ~/.someagent/skills   # niche/unknown agent
```

### npm wrapper

```sh
npx @rahadiangg/bbx-skills --all --target cursor
```

### Manual

```sh
git clone https://github.com/rahadiangg/bbx
cp -r bbx/skills/* ~/.agents/skills/   # or your agent's skills dir
```
