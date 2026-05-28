# Changelog

All notable changes to bbx are documented here. Format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/); this project uses
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] — unreleased

Initial public release.

### Added

- **Pipeline management** for Atlassian Bamboo Server:
  - `plan list/get/clone/enable/disable/delete`
  - `plan branch list/get/create/delete/enable/disable`
  - `plan variable list/get/set/delete`
  - `build trigger/stop/continue/status/get/history/latest`
  - `build comment list/add/delete`
  - `build label list/add/delete`
  - `build log` (best-effort; see `docs/API_COVERAGE.md` for the 8.x session-auth caveat)
  - `queue list`
  - `deployment queue/trigger/cancel/result/preview`
- **Authentication** via Bamboo Personal Access Tokens (PAT), with config in
  `~/.config/bbx/config.yaml` and override via `BBX_TOKEN` / `BBX_CONFIG`
  environment variables. Multiple contexts supported.
- **Server version cache** — `bbx auth login` captures Bamboo's `/info` and
  stores `server-version`, `server-build`, `server-edition` on the context;
  `bbx info` refreshes on demand.
- **Dual-mode output** — `-o table|json|yaml` flag, auto-switching to JSON
  when stdout is not a TTY or any of `CLAUDECODE` / `BBX_AGENT_MODE` /
  `ANTHROPIC_API_KEY` is set.
- **Structured errors** with stable exit codes (`0` OK, `2` usage, `3` auth,
  `4` partial, `5` cancelled, `6` not-implemented). Under `-o json`, errors
  come back as `{"error":{"code","message","http_status"}}`.
- **Agent skills bundle** at `claude-plugin/skills/` — `bbx`, `bbx-setup`,
  `bbx-trigger-build`, `bbx-investigate-build`.
- **API compatibility check** — static endpoint registry under
  `internal/apispec/` plus a CI matrix that validates bbx's endpoints against
  the published Bamboo OpenAPI spec for versions 9.0.0–12.1.1.

### Verified against

- **Bamboo Server 8.2.4** (live, full read/write end-to-end).
- **Bamboo Server 9.0.0, 9.2.1, 10.0.0, 11.0.0, 12.1.1** (static API compat
  via published OpenAPI specs).
