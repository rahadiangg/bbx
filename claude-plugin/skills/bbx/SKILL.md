---
name: bbx
description: Use bbx CLI to manage Atlassian Bamboo Server resources — plans (pipelines), builds, queue, deployments, plan variables, plan branches, build comments and labels. Trigger when the user wants to inspect, list, find, or modify any Bamboo resource. Maps user intent to specific bbx commands and dispatches to specialised skills (bbx-setup, bbx-trigger-build, bbx-investigate-build) when appropriate.
---

# bbx — Atlassian Bamboo CLI orchestrator

You drive `bbx`, a Go CLI that wraps Bamboo Server's REST API for both human operators and AI agents. This skill is the catch-all router: it covers discovery, lookup, and small ad-hoc commands; for setup, build-triggering, and failure investigation, hand off to the specialised skills.

## Core Principles

1. **Trust exit codes, not stdout.** bbx exits 0 OK, 2 usage, 3 auth, 4 partial, 5 cancelled, 6 not-implemented-on-this-bamboo. A non-zero exit with `-o json` always produces `{"error":{"code","message","http_status"}}`.
2. **Use `-o json` for any output you'll parse.** bbx auto-switches to JSON when stdout is piped, but be explicit in scripts and pipelines.
3. **Never write secrets to chat output.** When showing config, do NOT pass `--show-secrets`.
4. **Confirm before destructive actions.** `plan delete`, `plan disable` on a shared plan, `build stop`, or `build trigger` on a non-test plan all require an explicit user OK in the chat. Quote the target plan key in the confirmation.
5. **Prefix each action with one short "I'm about to…" line** so the user can interrupt before you change state.

## Prerequisites

If `./bbx config view -o json | python3 -c 'import sys,json; print(len(json.load(sys.stdin)["contexts"]))'` returns 0, or the user has never run `bbx auth login`, hand off to the **bbx-setup** skill first.

## When to delegate to a specialised skill

| User intent | Skill |
|---|---|
| "set up bbx", "first time", "configure", "log in", "add another Bamboo" | **bbx-setup** |
| "run plan X", "trigger build", "rebuild Y", "kick off Z and watch it" | **bbx-trigger-build** |
| "why did X fail?", "investigate this build", "what's wrong with PROJ-PLAN-N" | **bbx-investigate-build** |
| "export plan config", "extract pipeline", "migrate this to <other CI>", "give me everything about plan X" | **bbx-extract-config** |
| "list plans", "find plan X", "show queue", "what's running", any read-only inspection | stay here |
| "modify variable / label / comment / branch", non-trigger writes | stay here |

## Common discovery flows

### Find a plan by name (fuzzy)

```bash
# List all plans and filter client-side. Use --all to paginate, --limit if many.
./bbx plan list --all -o json | python3 -c "
import sys, json
needle = '<user search string>'.lower()
for p in json.load(sys.stdin):
    blob = (p.get('name','') + ' ' + p.get('key','')).lower()
    if needle in blob:
        print(p['key'], '-', p.get('name',''))
"
```

Show the user the matches and ask which to act on if more than one.

### Show the live queue

```bash
./bbx queue list -o json
```

Empty queue returns `[]` — that's normal, not an error.

### Inspect a plan

```bash
./bbx plan get <PROJ-PLAN> -o json
./bbx plan variable list <PROJ-PLAN> -o json
./bbx plan branch list <PROJ-PLAN> -o json
./bbx build history <PROJ-PLAN> --max-results 10 -o json
```

### Extract full plan configuration (for replication / migration)

For complete config dump — stages, jobs, tasks, all variables, branches,
linked deployments — defer to the **bbx-extract-config** skill. Quick
single-command summary:

```bash
./bbx plan spec <PROJ-PLAN>                     # Bamboo Specs Java source
./bbx plan config <PROJ-PLAN> -o json           # same data as nested JSON
./bbx plan artifact list <PROJ-PLAN> -o json
./bbx plan vcs-branches <PROJ-PLAN> --all -o json
./bbx project get <PROJ> -o json
./bbx project variable list <PROJ> -o json
./bbx project repository list <PROJ> -o json
./bbx deployment project list --for-plan <PROJ-PLAN> -o json
```

### Inspect the active context (no secrets!)

```bash
./bbx config view -o json
./bbx info -o json   # live server version
```

### Resource modification (non-trigger)

```bash
./bbx plan variable set    <PROJ-PLAN> <NAME> <value>
./bbx plan variable delete <PROJ-PLAN> <NAME>
./bbx plan branch  enable  <PROJ-PLAN> <branch-name>
./bbx plan branch  disable <PROJ-PLAN> <branch-name>
./bbx plan branch  create  <PROJ-PLAN> <branch-name> --vcs-branch <vcs-branch>
./bbx plan branch  delete  <PROJ-PLAN> <branch-name>
./bbx build label   add    <BUILD-KEY> <label>
./bbx build label   delete <BUILD-KEY> <label>
./bbx build comment add    <BUILD-KEY> "<text>"
./bbx build comment delete <BUILD-KEY> <id>
```

Always echo "I'm about to <verb> <target>." before executing.

## What bbx CANNOT do on this Bamboo

- **Create a plan from scratch** — Bamboo Server's REST has no `POST /plan`. Use `bbx plan clone <src> <dst>` (clones an existing plan), or have the user create it via the Bamboo UI / Bamboo Specs.
- **Fetch build logs reliably on Bamboo 8.2.4** — the `/download/*` paths require session-cookie auth. `bbx build log` returns `session_auth_required` (exit 3) on servers that gate logs that way. On newer Bamboo versions PAT may work.
- **Push Bamboo Specs back to the server** — `bbx plan spec` / `bbx deployment project spec` only *read* the Specs Java source. Publishing Specs is its own (separate-auth) subsystem.
- **Server admin** (`permissions`, `users`, `system`, `triggers`, `trusted-keys`, `session`, `avatars`) — these return exit 6 (`not_implemented`). Note: trigger *configuration* per plan is embedded in the Specs Java that `bbx plan spec` returns.
- **Stream live build logs** — only finished/in-progress full-text fetch, no tail.

## Output Format (when reporting back to the user)

Keep it short. For lookups:

```
Found <n> plan(s) matching "<query>":
- <KEY-1> — <name> (<enabled/disabled>)
- <KEY-2> — <name> (<enabled/disabled>)

Which would you like to act on?
```

For single-resource gets:

```
<KEY> — <name>
Project: <project-name> (<project-key>)
Enabled: <true/false>
Last build: <build-key> (<state>, <pretty-time>)   # if available
```

Never dump raw JSON unless the user asks. Surface the 3-5 most relevant fields.

## Error Handling

| Symptom | Remediation |
|---|---|
| Exit 1 + `no contexts configured` | Hand off to **bbx-setup**. |
| Exit 3 + `Unauthorized` | Token expired/wrong. `bbx info` to confirm reachability, then `bbx auth login` to refresh. |
| Exit 2 + `Plan ... not found` | Suggest `bbx plan list -o json` to search, or fix the key. |
| Exit 6 + `not_implemented` | Tell user this command is admin-only / out-of-scope; do not retry. |
| Network/DNS error | Confirm `bbx config view` shows the expected base-url; ask the user. |

## Cross-references

- First-time configuration → **bbx-setup**
- "trigger build X" → **bbx-trigger-build**
- "why did X fail" → **bbx-investigate-build**
