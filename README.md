# bbx — Atlassian Bamboo CLI

[![CI](https://github.com/rahadiangg/bbx/actions/workflows/ci.yml/badge.svg)](https://github.com/rahadiangg/bbx/actions/workflows/ci.yml)
[![API compatibility](https://github.com/rahadiangg/bbx/actions/workflows/api-compat.yml/badge.svg)](https://github.com/rahadiangg/bbx/actions/workflows/api-compat.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`bbx` is a Go CLI for Atlassian Bamboo Server, modelled after Grafana's `gcx`.
It covers two surfaces:

- **Pipeline management** — plans, plan branches/variables, builds, queue,
  comments/labels, deployments.
- **Pipeline configuration extraction** — `bbx plan spec` (Bamboo Specs Java
  source), `bbx plan config`, project + deployment-project metadata,
  environment variables / requirements / agent assignments. Enough surface
  for an AI agent to fully understand or replicate a Bamboo pipeline in
  another CI system.

Other Bamboo API areas (server admin, users, sessions, …) are recognized as
commands but emit a "not yet implemented" notice (see
[`docs/API_COVERAGE.md`](docs/API_COVERAGE.md)).

The CLI is designed to be friendly both for humans and AI agents — output
auto-switches to JSON when stdout is not a TTY or when running inside an agent
runtime (e.g. Claude Code).

**Compatibility:** Verified live against Bamboo Server **8.2.4**;
static API-spec compatibility CI matrix covers **9.0.0, 9.2.1, 10.0.0, 11.0.0,
12.1.1** on every PR. See [`docs/API_COVERAGE.md`](docs/API_COVERAGE.md) for
the per-version notes (Bamboo 8.x has no published OpenAPI spec, so it isn't
in the CI matrix).

## Install

### One-line install (Linux / macOS)

```sh
curl -sSfL https://raw.githubusercontent.com/rahadiangg/bbx/main/install.sh | sh
```

The script detects your OS + architecture, fetches the matching archive from
the latest GitHub release, verifies its SHA-256 checksum, and installs `bbx`
into `~/.local/bin` (or `/usr/local/bin` if the former isn't writable).

Pin a specific version or change the install path via environment variables:

```sh
BBX_VERSION=v0.1.0 BBX_INSTALL_DIR=$HOME/bin \
  curl -sSfL https://raw.githubusercontent.com/rahadiangg/bbx/main/install.sh | sh
```

### Manual download

Pre-built binaries for **Linux**, **macOS**, and **Windows** on both **amd64**
and **arm64** are attached to every release at
[github.com/rahadiangg/bbx/releases](https://github.com/rahadiangg/bbx/releases).
Each release ships a `checksums.txt` for SHA-256 verification.

### From source (Go 1.26+)

```sh
git clone https://github.com/rahadiangg/bbx
cd bbx
make install   # installs `bbx` into $GOBIN (defaults to $GOPATH/bin)
```

Or build a local binary:

```sh
make build && ./bbx --help
```

## Quickstart

```sh
# 1) Configure a Bamboo context (prompts for URL + PAT)
bbx auth login

# 2) Verify
bbx auth whoami

# 3) List plans
bbx plan list

# 4) Trigger a build and watch it
bbx build trigger PROJ-PLAN
bbx build status PROJ-PLAN-42

# 5) Queue + history
bbx queue list
bbx build history PROJ-PLAN --all --limit 50

# 6) Deploy a release
bbx deployment preview --project-id 123
bbx deployment trigger --environment-id 456 --version-id 789
```

## Authentication

`bbx` uses **Bamboo Personal Access Tokens** (PAT) sent as
`Authorization: Bearer <token>`.

- `bbx auth login` — interactively prompts for a base URL and PAT, stores it
  in `~/.config/bbx/config.yaml` (0600 perms).
- `BBX_TOKEN=<pat>` — env-var override; takes precedence over the stored token.
- `bbx auth logout` — removes a context.
- `bbx config use-context <name>` — switch the active context.

The config file looks like:

```yaml
current-context: default
contexts:
  default:
    base-url: https://bamboo.example.com
    token: <PAT>
```

## Output formats

A persistent `-o / --output` flag selects the format:

| Flag        | Behavior                                                              |
|-------------|-----------------------------------------------------------------------|
| `-o table`  | Pretty table for humans (default on a TTY).                           |
| `-o json`   | Indented JSON. Default when stdout is not a TTY or `CLAUDECODE` is set.|
| `-o yaml`   | YAML output.                                                          |

Set `BBX_AGENT_MODE=1` to force JSON unconditionally.

## Exit codes

`bbx` uses structured exit codes:

| Code | Meaning                                  |
|------|------------------------------------------|
| 0    | OK                                       |
| 1    | Generic error                            |
| 2    | Usage error / bad flag                   |
| 3    | Authentication / permission failure      |
| 4    | Partial success                          |
| 5    | Cancelled                                |
| 6    | Command group not yet implemented        |

When `-o json` is active, the error payload is machine-readable:

```json
{
  "error": {
    "code": "not_found",
    "message": "Plan not found",
    "http_status": 404
  }
}
```

## Repository layout

```
cmd/         Cobra commands (one package per subtree)
internal/
  api/       Typed Bamboo REST client (hand-written, MVP endpoints only)
  config/    YAML config loader (~/.config/bbx/config.yaml)
  output/    table/json/yaml renderer + agent-mode detection
  fail/      structured errors + exit codes
  version/   build-time version variables (set via -ldflags)
docs/        COMMANDS.md, API_COVERAGE.md
```

## Agent skills

bbx ships a bundle of [Claude Code skills](https://code.claude.com/docs/en/skills)
at [`skills/`](skills/) that let AI agents drive the CLI for common workflows:

| Skill | Purpose |
|---|---|
| [`bbx`](skills/bbx/SKILL.md) | Catch-all router + read-only discovery. |
| [`bbx-setup`](skills/bbx-setup/SKILL.md) | First-time configuration, multi-context switching. |
| [`bbx-trigger-build`](skills/bbx-trigger-build/SKILL.md) | Discover a plan → confirm → trigger → poll → report. |
| [`bbx-investigate-build`](skills/bbx-investigate-build/SKILL.md) | Given a failed build, gather context and surface likely root cause. |
| [`bbx-extract-config`](skills/bbx-extract-config/SKILL.md) | Dump full plan configuration (Specs Java, project, deployments) into one JSON bundle for migration / replication. |

### Three install paths

```sh
# 1. Native Claude Code plugin (idiomatic; uses the .claude-plugin/ manifest)
/plugin marketplace add rahadiangg/bbx
/plugin install bbx@bbx-marketplace

# 2. CLI install — works for any agent runtime that scans ~/.agents/skills/
bbx agent skills install --all           # all 5 skills (offline; bundled in the binary)
bbx agent skills install bbx-setup       # selective
bbx agent skills list                    # status of each
bbx agent skills update                  # refresh after upgrading the bbx binary
bbx agent skills uninstall --all --yes

# 3. Manual (developers / read-only inspection)
git clone https://github.com/rahadiangg/bbx
cp -r bbx/skills/* ~/.agents/skills/
```

The skills follow the same conventions as Grafana's `gcx` (YAML frontmatter +
sectioned markdown). They enforce: explicit confirmation before destructive
actions, polling timeouts, exit-code-based success checks, no echoing of
secrets when summarising config.

## Scope

See [`docs/API_COVERAGE.md`](docs/API_COVERAGE.md) for the full mapping of
Bamboo API tags to MVP / future buckets.

Out of scope for the MVP:

- Server admin endpoints (permissions, users, system info, triggers, trusted
  keys, sessions, avatars)
- Build log streaming and artifact downloads (not part of the REST swagger)
- OAuth / SSO; HTTP Basic with username + password — PAT only
- Secret storage via system keychain (file-only in v1)
- Agent skill markdown bundle (deferred to a follow-up)

## License

TBD.
