---
name: bbx-investigate-build
description: Use this skill to investigate why a Bamboo build failed, hung, or behaved unexpectedly. Gathers the result detail, comments, labels, and (where available) plain-text logs from every job, then surfaces likely root causes. Trigger when the user says "why did <PROJ-PLAN-N> fail", "investigate this build", "what went wrong with the last build", "look at build X", or pastes a build result key with a question.
---

# bbx — Investigate Build

Given a build result key (`PROJ-PLAN-N`), assemble enough context to tell the user *why* it failed and what to look at next. Be concise — these are experienced operators, not first-day users.

## Core Principles

1. **Don't restate what's obvious from the build name** ("the build failed"). Tell the user *what* failed (test, compile, deploy step, agent unavailable, timeout).
2. **Read-only.** This skill does NOT modify state. No labels, no comments, no stop/continue.
3. **Be honest about logs.** Bamboo 8.2.4 gates `/download/*` behind session-cookie auth. If `bbx build log` returns `session_auth_required`, say so and point the user at the Bamboo UI for full logs.
4. **Stop early for non-actionable cases.** If the build was successful, or is still running, don't run the full investigation — surface that and ask.
5. **Surface the result key in every message** so the user can scroll back.

## Prerequisites

bbx must have a configured context (see **bbx-setup** if not). The user must provide a build result key in the form `PROJECT-PLAN-N` (e.g. `RES-TES-42`).

## Step 1: Get the build result

```bash
bbx build get <KEY> -o json
```

Read these fields: `state`, `lifeCycleState`, `buildState`, `successful`, `buildStartedTime`, `buildCompletedTime`, `buildDurationInSeconds`, `prettyBuildDuration`, `buildReason`, `failedTestCount`, `successfulTestCount`.

## Step 2: Early-exit checks

Branch on `lifeCycleState` + `successful`:

- `lifeCycleState == "InProgress"` or `"Queued"` or `"Pending"` →
  > "Build `<KEY>` is still <state>. I can come back when it finishes. Want me to wait? (poll until done with bbx-trigger-build's logic)" — stop here unless the user says wait.

- `successful == true` →
  > "Build `<KEY>` succeeded in <prettyBuildDuration>. Nothing to investigate. Want a summary of what it did anyway?" — stop unless user says yes.

- `lifeCycleState == "NotBuilt"` →
  > "Build `<KEY>` was created but never ran (status: NotBuilt). This usually means the plan has no runnable tasks, or an agent compatible with the plan's requirements was offline. Check the plan configuration and agent status."

- `buildState == "Failed"` or `successful == false` → continue to Step 3.

## Step 3: Fetch comments + labels

These often contain human notes from previous failures of the same plan.

```bash
bbx build comment list <KEY> -o json
bbx build label   list <KEY> -o json
```

Look at comments — quote any that say "known issue", "flaky", "agent X", etc. Labels like `flaky`, `known-bug`, `infra` are also informative.

## Step 4: Attempt log fetch (best-effort)

```bash
bbx build log <KEY> -o json
```

Three outcomes:

| Outcome | What to do |
|---|---|
| Exit 0, JSON array of `{jobKey, log}` | Skim each `log` for the LAST 50 lines or any `ERROR`/`FAILED`/exception/stack-trace marker. Quote the most relevant 5-15 lines. |
| Exit 3, `code: "session_auth_required"` | Bamboo's `/download/*` gate. Tell the user logs can't be fetched via PAT on this Bamboo, and point them at the Bamboo UI: `<base-url>/browse/<KEY>/log`. |
| Exit 2 + `not found` | The build never produced logs (NotBuilt). Already covered in Step 2. |

When you do have logs, classify the failure into one of these buckets if you can:

- **Test failure** — `failedTestCount > 0` from Step 1 corroborates. Quote failing test names from the log.
- **Compile / build tool error** — look for `error:`, `compiler error`, `cannot find symbol`, `npm ERR!`, `make: ***`.
- **Deploy / IO error** — `Permission denied`, `Connection refused`, `No such file`, `404`/`500` to a deploy target.
- **Agent issue** — `agent unresponsive`, `agent disconnected`, `timed out waiting for an agent`.
- **Infrastructure** — `OutOfMemoryError`, `out of disk`, `no space left`, `oom-killer`.
- **Other** — quote and admit you don't recognise the pattern.

## Step 5: Check the plan's recent history for a pattern

```bash
bbx build history <PROJ-PLAN> --max-results 5 -o json
```

(Extract `<PROJ-PLAN>` from the key by trimming the trailing `-N`.) If the last 3+ builds were all failed, this is a persistent issue, not a one-off. Say so.

## Step 6: Report

Use this exact structure:

```
Build <KEY> — <Failed/NotBuilt/...>

Plan: <plan key> — <plan name>
Reason: <buildReason>
Duration: <prettyBuildDuration>
Tests: <successfulTestCount> passed, <failedTestCount> failed

Likely cause: <one short line>

Evidence:
- <quoted log line 1>
- <quoted log line 2>
- <quoted log line 3>

History: <m of last 5 builds failed | this is a one-off>

Comments on this build: <count, or "(none)">
[ quote any non-trivial comments ]

Next actions:
- <action 1, e.g. "Re-run after fixing X">
- <action 2, e.g. "Open Bamboo UI for full log: <url>/browse/<KEY>/log">
```

Keep total output under ~30 lines unless the user explicitly asks for more.

## Output Format (when logs aren't fetchable)

```
Build <KEY> — Failed

Plan: <plan key>
Duration: <prettyBuildDuration>
Tests: <pass>/<fail>
Reason: <buildReason>

Could not fetch logs (Bamboo's session-auth gate). View in the UI:
  <base-url>/browse/<KEY>/log

History: <m of last 5 builds failed | one-off>

Want me to:
- Add a comment to this build documenting the investigation?
- Trigger a retry?
```

## Error Handling

| Symptom | Remediation |
|---|---|
| Exit 2 + `Build ... not found` | Wrong key. Ask user; suggest `bbx build history <plan-key>` to discover recent build keys. |
| Exit 3 on `bbx build comment list` (rare) | Bamboo permissions. Skip and continue without comments. |
| `bbx build log` returns `session_auth_required` | Expected on 8.2.4. Document in the output and point at the UI URL. |

## Cross-references

- "Run it again" → **bbx-trigger-build**
- Auth / config issue → **bbx-setup**
- General lookups → **bbx**
