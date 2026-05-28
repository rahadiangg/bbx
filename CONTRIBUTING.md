# Contributing to bbx

Thanks for the interest. This is a short guide; the project is small enough
that there isn't much process to wade through.

## Prerequisites

- Go **1.26** or newer (`go version`).
- `make` (any standard distribution).
- Optional: `golangci-lint` for the `make lint` target.

## Quickstart

```sh
git clone https://github.com/rahadiangg/bbx && cd bbx
make build        # produces ./bbx
make test         # go test -race ./...
make lint         # go vet + golangci-lint if installed
```

## Layout

| Path | Purpose |
|---|---|
| `cmd/` | Cobra commands, one package per subtree |
| `internal/api/` | Typed Bamboo REST client (hand-written, MVP endpoints only) |
| `internal/config/` | YAML config loader |
| `internal/output/` | table/json/yaml renderer + agent-mode detection |
| `internal/fail/` | Structured errors + exit codes |
| `internal/apispec/` | Endpoint registry + compatibility checker (used by CI) |
| `skills/` | Agent skills (markdown) — embedded into the binary via `assets.go` |
| `.claude-plugin/` | Plugin manifest for Claude Code marketplace discovery |
| `cmd/bbx/main.go` | Binary entrypoint (root package is library code for the embed) |

## Adding a new bbx command

1. Add the API method in `internal/api/<area>.go` with a typed return value.
   Match Bamboo's wire shape — when it differs across versions, prefer
   defensive decoding (try the strict shape, fall back to `map[string]any`).
2. Add a Cobra leaf in `cmd/<area>/<verb>.go` mirroring existing leaves.
3. Wire it into the parent in `cmd/<area>/<area>.go`.
4. **Add an entry to `internal/apispec/endpoints.go`** — without this, the
   compat matrix won't protect the new endpoint.
5. Add an `httptest`-backed unit test in `internal/api/<area>_test.go`.
6. Add a cmd integration test in `cmd/<area>_test.go` using the existing
   `runCmdEnv` harness.

## Adding a new skill

1. Create `skills/<name>/SKILL.md` with YAML frontmatter (`name`,
   `description`) and the body conventions used by the existing five.
2. Update `assets_test.go` to include the new name in the expected list
   — that's the safety net that the embed isn't silently dropping anything.
3. Rebuild: `make build`.
4. Verify: `./bbx agent skills install --force` and `./bbx agent skills list`.
5. Before tagging a release, bump `version` in
   [`.claude-plugin/plugin.json`](.claude-plugin/plugin.json) to match.

## Testing

- `make test` — unit + integration tests via `httptest` fakes. Hermetic.
- `BBX_BASE_URL=… BBX_TOKEN=… ./bbx auth login --name dev --base-url … --token …`
  then run any `bbx` command against a real Bamboo. The verification script
  in `docs/API_COVERAGE.md` exercises the full write surface on a disposable
  cloned plan.
- API-compat check against a specific Bamboo version:

```sh
BBX_COMPAT_SWAGGER=https://docs.atlassian.com/atlassian-bamboo/REST/12.1.1/swagger.json \
  go test ./internal/apispec/... -v
```

## Conventions

- Comments explain **why**, not what. Avoid restating the code.
- Keep tests close to the package they test. Use `t.Setenv` for env-var
  isolation; never modify global state without a `t.Cleanup` undo.
- Match the existing dual-mode-output and exit-code conventions for any new
  command (use `cmdctx.G().Emit` for stdout, `cmdctx.G().Stderr` for human
  status, return a `*fail.Error` for typed exit codes).

## Submitting a change

1. Open a PR against `main`. CI runs `go test`, `go vet`, `golangci-lint`,
   and the API-compat matrix against Bamboo 9.0–12.x.
2. Mention any new endpoint in the PR description.
3. Update `CHANGELOG.md` under `[Unreleased]`.

## Bamboo 8.x note

Atlassian does NOT publish an OpenAPI spec for Bamboo 8.x. bbx's 8.x
compatibility is verified by hand against a live server, not by CI. If you
hit a shape mismatch on a 8.x server, file an issue with the raw response
JSON and we'll add defensive decoding + a regression test.
