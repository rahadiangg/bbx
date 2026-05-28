---
name: bbx-setup
description: Use this skill to set up bbx for first-time use against an Atlassian Bamboo Server — configure authentication with a Personal Access Token, capture the server version, switch between multiple Bamboo contexts, and verify connectivity. Trigger when the user says "set up bbx", "configure bbx", "first time using bbx", "log in to Bamboo", "add another Bamboo server", or when any other skill fails with "no contexts configured" / exit code 1.
---

# bbx Setup

Configure `bbx` to talk to an Atlassian Bamboo Server. Covers first-time setup, adding additional contexts (servers), switching between them, and recovering from auth failures.

## Core Principles

1. **Never echo the user's PAT to the chat.** When you pass `--token <value>`, the value is in the command line — that's fine, but in *summaries to the user* always redact (`***`).
2. **Be explicit about which Bamboo you're configuring** — quote the base-url back to the user before running login.
3. **Login is idempotent.** Re-running `bbx auth login` with the same `--name` overwrites that context safely.
4. **One PAT, one context.** Don't share a PAT across multiple `--name` contexts.

## Prerequisites

- bbx binary on PATH (or invoked as `./bbx`). Check with `./bbx version`.
- A Bamboo Personal Access Token (PAT). User generates it via Bamboo UI → avatar menu → Profile → Personal access tokens → Create token.
- The Bamboo base URL (e.g. `https://bamboo.example.com`, `http://build.dev.weefer.co.id:8085`).

## Step 1: Check existing configuration

```bash
./bbx config view -o json
```

If exit 0 and the JSON shows one or more `contexts`, surface them to the user:

```
You already have <n> bbx context(s) configured:
- <name> — <base-url>  (Bamboo <server-version>)   [current]
- <name> — <base-url>  (Bamboo <server-version>)

Do you want to (a) use the current one, (b) switch to another, or (c) add a new Bamboo server?
```

Branch:
- (a) → run `./bbx auth whoami` to confirm and stop.
- (b) → see Step 4 (switch context).
- (c) → continue to Step 2.

If exit 1 with `no contexts configured`, continue to Step 2.

## Step 2: Gather inputs

Ask the user for:
1. **Context name** (short label like `prod`, `dev`, `weefer` — defaults to `default` if they have no preference).
2. **Base URL** (with `http://` or `https://`, no trailing slash — bbx normalises but it's nice to be clean).
3. **Personal Access Token** — *do not display the token back*. Ask the user to paste it into the chat, then redact it from any echo you produce.

If the Bamboo uses a self-signed certificate, ask whether to set `--insecure`. Default off.

## Step 3: Run login

Echo the action with a redacted token:

> "I'm about to: `./bbx auth login --name <ctx> --base-url <url> --token <REDACTED>`"

Then execute:

```bash
./bbx auth login \
  --name "<ctx>" \
  --base-url "<url>" \
  --token "<pat>" \
  ${insecure:+--insecure}
```

Expected stderr: `Saved context "<ctx>" (current: <ctx>, server: Bamboo <version>)`

If the server-version line is missing (Bamboo wasn't reachable for `/info`), `bbx` still saved the credentials and emits a `warning:` line — that's OK. Surface this to the user and continue to Step 5.

## Step 4: Switch context (if user picked option b in Step 1)

```bash
./bbx config use-context <name>
```

Then jump to Step 5.

## Step 5: Verify with whoami + info

```bash
./bbx auth whoami -o json
./bbx info -o json
```

Both should exit 0. Show the user a short summary:

```
✓ Connected to Bamboo <version> (build <buildNumber>) at <base-url>
✓ Authenticated as <user.fullName> (<user.name>)
```

If `whoami` exits 3 → the PAT was rejected. Tell the user to regenerate the PAT in Bamboo and re-run `bbx auth login`.

## Step 6: Tell the user what's next

```
You're set up. Common things you can do now:
- List plans:      bbx plan list
- Trigger a build: ask me to "run plan <key>"
- Investigate a failure: paste the build result key (e.g. PROJ-PLAN-42)
```

## Switching contexts (without re-login)

If the user has multiple contexts already and wants to point at a different one:

```bash
./bbx config contexts -o json
./bbx config use-context <name>
./bbx info             # confirm we're pointing at the right server
```

## Per-invocation override (without changing the active context)

For one-off commands against a non-active context:

```bash
./bbx --context <name> plan list
```

## Removing a context

Confirm with the user first (quote the name and base-url). Then:

```bash
./bbx auth logout --name <name>
```

## Output Format (final message to the user)

After a successful setup:

```
Configured bbx context "<ctx>"
  Bamboo: <version> at <base-url>
  Authenticated as: <user>
  Current context: <ctx>

What would you like to do first?
```

## Error Handling

| Symptom | Remediation |
|---|---|
| Exit 2 + `--base-url is required in non-interactive mode` | The user didn't paste a URL; ask again. |
| Exit 3 + `Client must be authenticated` from `whoami` | PAT rejected. Have the user regenerate it and re-run login. |
| `warning: could not fetch server info` on stderr after login | Network blip or PAT lacks /info access. Login still saved — run `bbx info` later to refresh the cached version. |
| `connection refused` / DNS error | Wrong base URL or Bamboo down. Confirm URL with the user. |
| User pastes a token that's clearly mangled (too short, has spaces) | Ask them to regenerate. |

## Cross-references

- After setup → handoff to **bbx** for general use or **bbx-trigger-build** if the user wants to run something.
- "Why did this build fail" → **bbx-investigate-build**.
