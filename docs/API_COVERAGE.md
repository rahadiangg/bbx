# Bamboo Server API coverage in `bbx`

Source of truth: <https://dac-static.atlassian.com/server/bamboo/swagger.v3.json>

The Bamboo Server REST API surface is large (~183 endpoints across 11 tags).
This document tracks which tags `bbx` covers today and which are deferred.

## Status legend

| Status         | Meaning                                                                 |
|----------------|-------------------------------------------------------------------------|
| ✅ MVP          | Implemented and exposed via a leaf command                              |
| 🚧 MVP-adjacent | Partially used by other commands but not exposed as a standalone tree   |
| 🛑 Future       | Not implemented; a placeholder command emits `ExitNotImpl` (6)          |

## Coverage matrix

| Bamboo tag             | Endpoints | Status         | Notes                                                                                       |
|------------------------|----------:|----------------|---------------------------------------------------------------------------------------------|
| Build                  |        29 | ✅ MVP          | Plans, builds, queue, plan branches, plan variables, build comments, build labels           |
| deployment             |         7 | ✅ MVP          | Deployment queue, trigger, cancel, result, version preview                                  |
| Core                   |        29 | 🚧 MVP-adjacent | `currentUser` used for `auth whoami`; environment & agent endpoints are not yet exposed     |
| Build number           |         2 | 🛑 Future       | `getNextBuildNumber`, `bumpBuildNumber`                                                     |
| Permissions            |        77 | 🛑 Future       | Plan/project/deployment permissions admin                                                   |
| User management        |        24 | 🛑 Future       | User/group CRUD                                                                             |
| Triggers               |         1 | 🛑 Future       | Plan trigger admin                                                                          |
| System information     |         1 | 🛑 Future       | Server info                                                                                 |
| Trusted keys management |        3 | 🛑 Future       | SSH/SSL trusted keys                                                                       |
| Session                |         1 | 🛑 Future       | Session info                                                                                |
| avatar                 |         3 | 🛑 Future       | Avatar upload/get/delete                                                                    |

## MVP command surface

| Command                                          | Endpoint                                                    |
|--------------------------------------------------|-------------------------------------------------------------|
| `bbx auth whoami`                                | `GET /rest/api/latest/currentUser`                          |
| `bbx plan list`                                  | `GET /rest/api/latest/plan`                                 |
| `bbx plan get <key>`                             | `GET /rest/api/latest/plan/{key}`                           |
| `bbx plan enable <key>`                          | `POST /rest/api/latest/plan/{key}/enable`                   |
| `bbx plan disable <key>`                         | `DELETE /rest/api/latest/plan/{key}/enable`                 |
| `bbx plan delete <key> --yes`                    | `DELETE /rest/api/latest/plan/{key}`                        |
| `bbx plan branch list <plan>`                    | `GET /rest/api/latest/plan/{key}/branch`                    |
| `bbx plan branch get <plan> <branch>`            | `GET /rest/api/latest/plan/{key}/branch/{branch}`           |
| `bbx plan branch create <plan> <branch>`         | `PUT /rest/api/latest/plan/{key}/branch/{branch}`           |
| `bbx plan variable list <plan>`                  | `GET /rest/api/latest/plan/{key}/variables`                 |
| `bbx plan variable get <plan> <name>`            | `GET /rest/api/latest/plan/{key}/variables/{name}`          |
| `bbx plan variable set <plan> <name> <value>`    | `PUT/POST /rest/api/latest/plan/{key}/variables/{name}`     |
| `bbx plan variable delete <plan> <name>`         | `DELETE /rest/api/latest/plan/{key}/variables/{name}`       |
| `bbx queue list`                                 | `GET /rest/api/latest/queue`                                |
| `bbx plan clone <src> <dst>`                     | `PUT /rest/api/latest/clone/<src>:<dst>`                    |
| `bbx plan branch enable  <plan> <branch>`        | `GET /branch/{name}` + `POST /plan/{branchKey}/enable`      |
| `bbx plan branch disable <plan> <branch>`        | `GET /branch/{name}` + `DELETE /plan/{branchKey}/enable`    |
| `bbx plan branch delete  <plan> <branch>`        | `GET /branch/{name}` + `DELETE /plan/{branchKey}`           |
| `bbx build log <build-key> [--job <jobKey>]`     | `GET /rest/api/latest/result/{key}?expand=stages...` + `GET /download/{jobKey}/build_logs/{jobKey}-N.log` (non-REST) |
| `bbx info`                                       | `GET /rest/api/latest/info`                                 |
| `bbx plan spec <key>`                            | `GET /rest/api/latest/plan/{key}/specs`                     |
| `bbx plan config <key>`                          | `GET /rest/api/latest/plan/{key}?expand=stages…`           |
| `bbx plan artifact list <key>`                   | `GET /rest/api/latest/plan/{key}/artifact`                  |
| `bbx plan vcs-branches <key>`                    | `GET /rest/api/latest/plan/{key}/vcsBranches`               |
| `bbx project get <key>`                          | `GET /rest/api/latest/project/{key}`                        |
| `bbx project spec <key>`                         | `GET /rest/api/latest/project/{key}/specs`                  |
| `bbx project variable list <key>`                | `GET /rest/api/latest/project/{key}/variables`              |
| `bbx project variable get <key> <name>`          | `GET /rest/api/latest/project/{key}/variable/{name}`        |
| `bbx project repository list <key>`              | `GET /rest/api/latest/project/{key}/repository`             |
| `bbx deployment project list [--for-plan <key>]` | `GET /rest/api/latest/deploy/project/all` or `/forPlan`     |
| `bbx deployment project get <id>`                | `GET /rest/api/latest/deploy/project/{id}`                  |
| `bbx deployment project spec <id>`               | `GET /rest/api/latest/deploy/project/{id}/specs`            |
| `bbx deployment project repository list <id>`    | `GET /rest/api/latest/deploy/project/{id}/repository`       |
| `bbx deployment version list <id>`               | `GET /rest/api/latest/deploy/project/{id}/versions`         |
| `bbx deployment environment get <envId>`         | `GET /rest/api/latest/deploy/environment/{envId}`           |
| `bbx deployment environment variable list <envId>` | `GET /rest/api/latest/deploy/environment/{envId}/variables` |
| `bbx deployment environment requirement list <envId>` | `GET /rest/api/latest/deploy/environment/{envId}/requirement` |
| `bbx deployment environment agent list <envId>`  | `GET /rest/api/latest/deploy/environment/{envId}/agent-assignment` |
| `bbx build trigger <plan>`                       | `POST /rest/api/latest/queue/{plan}`                        |
| `bbx build stop <build-key>`                     | `DELETE /rest/api/latest/queue/{key}`                       |
| `bbx build continue <build-key>`                 | `PUT /rest/api/latest/queue/{key}`                          |
| `bbx build status <build-key>`                   | `GET /rest/api/latest/result/status/{key}`                  |
| `bbx build get <build-key>`                      | `GET /rest/api/latest/result/{key}`                         |
| `bbx build history <plan>`                       | `GET /rest/api/latest/result/{plan}`                        |
| `bbx build latest`                               | `GET /rest/api/latest/result`                               |
| `bbx build comment list <build>`                 | `GET /rest/api/latest/result/{key}/comment`                 |
| `bbx build comment add <build> <content>`        | `POST /rest/api/latest/result/{key}/comment`                |
| `bbx build comment delete <build> <id>`          | `DELETE /rest/api/latest/result/{key}/comment/{id}`         |
| `bbx build label list <build>`                   | `GET /rest/api/latest/result/{key}/label`                   |
| `bbx build label add <build> <label>`            | `POST /rest/api/latest/result/{key}/label`                  |
| `bbx build label delete <build> <label>`         | `DELETE /rest/api/latest/result/{key}/label/{label}`        |
| `bbx deployment queue`                           | `GET /rest/api/latest/queue/deployment`                     |
| `bbx deployment trigger --env --version`         | `POST /rest/api/latest/queue/deployment`                    |
| `bbx deployment cancel <id>`                     | `DELETE /rest/api/latest/queue/deployment/{id}`             |
| `bbx deployment result <id>`                     | `GET /rest/api/latest/deploy/result/{id}`                   |
| `bbx deployment preview`                         | `GET /rest/api/latest/deploy/preview/version`               |

## Known limitations on Bamboo Server 8.2.4

- **Build log download (`bbx build log`)** — Bamboo serves `/download/*` paths
  via a different servlet that requires **session-cookie auth**, NOT Personal
  Access Tokens. PAT requests are silently redirected to an HTML login page.
  bbx detects this case and returns a typed `session_auth_required` error
  (exit code 3). The command still works on Bamboo versions / instances that
  honour PAT on `/download/`. Workarounds: view logs in the Bamboo UI at
  `<base-url>/browse/<BUILD-KEY>/log`, or add HTTP Basic + cookie-capture
  support to bbx (out of scope today).

- **Plan creation from scratch** — no REST endpoint exists in 8.2.4 (verified
  via `OPTIONS /rest/api/latest/plan` → `Allow: OPTIONS, HEAD, GET`).
  Workarounds: `bbx plan clone <src> <dst>`, or Bamboo Specs (separate
  subsystem, not exposed by bbx).

- **Plan branch DELETE on `/plan/{key}/branch/{name}`** — returns 405 on
  8.2.4. bbx works around this with a two-step lookup: get the branch's own
  plan-key, then call DELETE on that plan-key.

## Conventions

- All endpoints target the `/rest/api/latest/...` namespace. Bamboo's
  versioned aliases (e.g. `/rest/api/1.0/...`) are not used.
- Pagination: `start-index` and `max-results` query params. List commands
  expose `--all` (paginate to exhaustion) and `--limit` (cap items) flags.
- Auth: `Authorization: Bearer <PAT>` for all requests.

## Future work (rough priority)

1. **Build number** (`buildNumber/{plan}`, `bumpBuildNumber`) — lightweight,
   useful for release engineering scripts.
2. **Triggers** — manage plan-level triggers from CI.
3. **System information** — `bbx system info` for diagnostics.
4. **Permissions** — large surface (77 endpoints), needed only for admin
   workflows.
5. **User management** — admin-only.
6. **Avatars / Trusted keys / Session** — niche.
