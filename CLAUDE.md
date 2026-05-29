# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Always update tests

Every code change must come with tests: create one for new behaviour, update existing tests when you change a contract, fix red tests rather than skip them. A PR without test changes for code changes is incomplete. `make test` must stay green.

## Build, test, lint

Go 1.26+. All targets in `Makefile`:

```sh
make build         # ./bbx with ldflags-injected version/commit/date
make test          # go test -race ./...
make lint          # go vet + golangci-lint (if installed)
make tidy          # go mod tidy

# Single test
go test ./internal/api/... -run TestClonePlan -v
go test ./cmd/... -run TestPlanBranchEnableDisable -v

# API-compat check (hermetic by default; overrides per matrix cell in CI)
BBX_COMPAT_SWAGGER=https://docs.atlassian.com/atlassian-bamboo/REST/12.1.1/swagger.json \
  go test ./internal/apispec/... -run TestAPICompat -v

# Real Bamboo (don't touch ~/.config/bbx)
BBX_CONFIG=/tmp/scratch.yaml BBX_TOKEN=<pat> ./bbx auth login --name X --base-url ...
```

End-to-end live-server sequence: `docs/API_COVERAGE.md`.

## Architecture (three layers)

1. **`cmd/`** — Cobra leaves, one package per noun. Each leaf reads args, calls the client via `cmdctx.G().Client(ctx)`, emits via `cmdctx.G().Emit(v)`.
2. **`cmd/cmdctx/`** — package-level `Globals` **singleton** set by `root.go` `PersistentPreRunE`. Cmd tests reset it with `cmdctx.Set(cmdctx.Globals{})` between invocations (the `runCmd` helper in `cmd/testhelpers_test.go` does this).
3. **`internal/api/`** — typed Bamboo client. `Client.Do(...)` auto-prefixes `/rest`. **`doDownload`** is the only method that bypasses it (used by `GetBuildLog` for `/download/*`). **`doRawJSON`** is used when the response shape varies across Bamboo versions (`ClonePlan` falls back from strict `Plan` decode to `map[string]any`).

Supporting:

- **`internal/config/`** — YAML at `~/.config/bbx/config.yaml`. `Context.Active()` honours `BBX_TOKEN` env override. `server-version` captured at `auth login` from `/api/latest/info` for version-gated code without an extra round-trip.
- **`internal/output/`** — `Format` enum + `IsAgentMode()` (checks `CLAUDECODE`, `BBX_AGENT_MODE`, `ANTHROPIC_API_KEY`, then TTY). Types can implement `RenderTable(w io.Writer)`; otherwise table mode falls back to YAML.
- **`internal/fail/`** — `Error{Code, Message, HTTPStatus, Exit}`. Exit codes: `0 OK, 1 Generic, 2 Usage, 3 Auth, 4 Partial, 5 Cancelled, 6 NotImpl`. API error parser (`internal/api/errors.go`) maps 401/403→Auth, 400/404/409→Usage, others→Generic.
- **`internal/apispec/`** — endpoint registry (`endpoints.go`) + compat checker.

`cmd/root.go` `Execute()` → `output.PrintError` → `exitCodeFor(err)`. **Any non-`*fail.Error` from Cobra (bad arg count, unknown flag) maps to `ExitUsage`** — not Generic. Test helpers mirror this.

`internal/api/pagination.go`: generic `Iterate[T](ctx, opts, limit, fetch)` loops `start-index` until short/empty page. `PageOpts` carries `start-index`, `max-results`, `expand`, free-form `Extra url.Values`.

## Adding a new bbx command — five places

Skip any of these and CI silently fails to protect the new endpoint.

1. `internal/api/<area>.go` — new `*Client` method. Match wire shape; prefer defensive decoding for variant fields.
2. `cmd/<area>/<verb>.go` — Cobra leaf. `cmdctx.G().Emit(result)` for stdout, `cmdctx.G().Stderr(...)` for human status.
3. `cmd/<area>/<area>.go` — wire `c.AddCommand(newVerbCmd())` in `New()`.
4. `internal/apispec/endpoints.go` — registry entry (method + swagger-style path with `{placeholder}` names).
5. Tests — `httptest` unit test in `internal/api/<area>_test.go` (use `newFakeBamboo` from `testserver_test.go`) + cmd integration via `runCmdEnv` in `cmd/<area>_test.go`.

## Bamboo wire-shape gotchas (regression-tested; don't undo)

- `planResultKey.entityKey` is an **object** `{"key":"..."}`, not a string (`EntityKey` in `build.go`).
- `queuedDeployments` is an envelope `{size, queuedDeployment:[...]}`, not a bare array (`rawDeploymentQueueEnvelope`).
- Error JSON's `status-code` is an **int** (decoding as string silently falls back to raw body).
- Some errors use `errors[]` instead of `message` (e.g. `POST /queue/deployment` validation) — parser joins them.
- Empty lists pre-allocate `[]` (not `null`): `ListQueue`, `ListDeploymentQueue`, `ListBuildComments`, `ListPlanVariables`.
- `POST /plan/{key}/variables` takes **query params** (`name=&value=`), NOT a JSON body. `PUT` of the same takes a JSON body. Asymmetric.
- `GET /result/{key}/comment` omits content + dates without `expand=comments.comment`.
- `POST /result/{key}/comment` returns **204 No Content** — `AddBuildComment` returns just `error`.
- `GET /result/status/{key}` returns **404 for finished builds**. Poll via `GetBuild` + `lifeCycleState`, not `/status`.
- `POST/DELETE /queue/{key}` on a finished build returns 4xx — safe to ignore in cleanup.
- `DELETE /plan/{key}/branch/{name}` returns **405** on 8.2.4. Branches are themselves plans; `DeletePlanBranch` does `GetPlanBranch` → `DeletePlan(branch.Key)`. Same for branch `enable`/`disable`.
- No `POST /plan` exists — `bbx plan clone` (`PUT /clone/{src}:{dst}`) is the only REST creation path.
- `/download/*` requires **session-cookie auth**, not PAT. `GetBuildLog` detects the HTML-login-redirect and returns typed `session_auth_required` (exit 3).
- Bamboo **8.x has no published OpenAPI spec**; CI compat matrix starts at 9.0.0. 8.x compatibility is hand-verified.

## Agent skills

`skills/*/SKILL.md` — markdown contracts for AI agents (YAML frontmatter `name` + `description`, sectioned body). They are NOT code; they route user intent to `bbx` commands. When changing a command surface (rename, new/removed flags or verbs), grep skills for the literal command string and update.

The 5 skills (`bbx`, `bbx-setup`, `bbx-trigger-build`, `bbx-investigate-build`, `bbx-extract-config`) enforce: explicit `-o json` for parsing, confirmation before destructive actions, polling timeouts, trust exit codes over stdout, never echo secrets.

`skills/*/SKILL.md` is the **single source of truth**. Four consumers read it, all from that one directory:

1. **bbx binary** — `//go:embed skills` in `assets.go` (root `package bbx`) bakes the bundle in. `bbx agent skills install [--all|<names...>] [--target <agent>...|-a] [--scope global|project] [--dir <path>] [--force] [--dry-run]` extracts it. Agent→dir mapping lives in `cmd/agent/skills/targets.go` (registry: `agents` default → `~/.agents/skills`, `claude-code` → `~/.claude/skills`, `codex` → `~/.codex/skills`, `cursor`, `opencode`, `cline`, `github-copilot`). `--dir` overrides the registry; default with no flags stays `~/.agents/skills` (backward compatible). Same per-agent paths as the `vercel-labs/skills` CLI.
2. **Claude Code plugin** — added via `/plugin marketplace add rahadiangg/bbx` → `/plugin install bbx@bbx`. Requires **both** `.claude-plugin/marketplace.json` (catalog with `plugins[]`, `source: "."`) **and** `.claude-plugin/plugin.json` (the plugin manifest). Skills auto-discovered from `skills/`.
3. **`npx skills` / skills.sh** — works out of the box (`npx skills add rahadiangg/bbx`) because vercel-labs/skills scans the `skills/<name>/SKILL.md` flat layout. Discovery is automatic via install telemetry; no registry submission.
4. **npm wrapper** — `npm/` ships `@rahadiangg/bbx-skills` (a dependency-free Node port of the installer, `npm/install.mjs`, mirroring the same target mapping). `npm/skills/` is a **synced copy** of `skills/`, kept in lockstep by `scripts/sync-npm-skills.sh` (`make sync-npm`); CI runs `make check-npm-sync` to fail on drift. Never edit `npm/skills/` by hand.

When changing a command surface (rename, new/removed flags or verbs), grep skills for the literal command string and update. When changing `targets.go`'s registry, mirror it in `npm/install.mjs` (both have tests asserting the mapping).

## CI

- `.github/workflows/ci.yml` — `go test -race` + `go vet` + `golangci-lint` + `make build` on PR + push to `main`. The `npm-skills` job runs `make check-npm-sync` (fails on `skills/` ↔ `npm/skills/` drift) + the wrapper's `node --test`.
- `.github/workflows/api-compat.yml` — `TestAPICompat` matrix over Bamboo 9.0.0/9.2.1/10.0.0/11.0.0/12.1.1 with `BBX_COMPAT_SWAGGER` per cell. `fail-fast: false`. Weekly cron catches upstream spec changes.
- Live-Bamboo CI is intentionally not wired (needs Atlassian license + agent provisioning).

## Release

Ship a release: `git tag v<x.y.z> && git push --tags`. That fires `.github/workflows/release.yml`, which runs `goreleaser` (config in `.goreleaser.yaml`) to build all 6 OS/arch combos, generate `checksums.txt`, derive the changelog from conventional-commit messages, and publish a GitHub release. `install.sh` at the repo root consumes that release for `curl|sh` installation.

The same workflow has a `npm` job (after goreleaser) that syncs `npm/skills/`, sets the package version from the tag (strips the leading `v`), runs the wrapper tests, and `npm publish`es `@rahadiangg/bbx-skills`. The publish step is **gated on the `NPM_TOKEN` repo secret** — without it, the job runs but skips publishing. Bump `.claude-plugin/plugin.json` `version` on each release too (it gates Claude Code plugin updates).

Smoke-test locally before tagging: `goreleaser check` then `goreleaser build --snapshot --clean --single-target` (~3s).
