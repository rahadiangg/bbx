---
name: bbx-trigger-build
description: Use this skill to trigger a Bamboo build on a specific plan, monitor it to completion, and report the outcome. Triggers when the user says "run plan X", "trigger build", "kick off a build", "rebuild Y", "start <PROJ-PLAN>", or asks bbx to execute a pipeline. Handles plan discovery by name, disabled-plan recovery, custom variables, polling timeouts, and the Bamboo /status 404-on-finished trap.
---

# bbx — Trigger Build

End-to-end workflow: find the plan the user means, confirm, trigger, poll until finished, report.

## Core Principles

1. **Always confirm before triggering.** Quote the plan key and any variables back to the user. A build can deploy code; never trigger without an explicit OK.
2. **Use `lifeCycleState` (from `bbx build get`) — NOT `bbx build status` — to detect completion.** Bamboo's `/result/status/{key}` endpoint returns 404 once a build finishes; that's a documented Bamboo trap, not an error.
3. **Always set a polling timeout.** Default 5 minutes (30 attempts × 10s). When the cap is hit, report it explicitly — don't loop forever.
4. **If the plan is disabled, ask before enabling.** Enabling can affect other users (scheduled triggers, integrations).
5. **Echo a one-line action statement** before each `bbx` invocation that mutates state.

## Prerequisites

bbx must already have a configured context. If `bbx auth whoami` returns exit 1 or 3, hand off to **bbx-setup**.

## Step 1: Resolve the plan key

The user may give a plan key (`RES-TES`), a project name, a plan name, or a fuzzy phrase.

```bash
bbx plan list --all -o json
```

Filter client-side. If exactly one match, proceed. If multiple matches, list them and ask which one. If zero matches, ask the user to refine or paste the plan key.

```bash
bbx plan get <KEY> -o json
```

Capture: `enabled` (boolean), `name`, `projectName`.

## Step 2: Confirm with the user

```
About to trigger:
  Plan: <KEY> — <name> (<project>)
  Enabled: <true/false>
  Variables: <none / key=value, key=value>

Proceed? (yes/no)
```

If `enabled: false`, ask additionally:

```
This plan is currently disabled. Enable it before triggering? (yes/no)
```

Only proceed on explicit "yes".

## Step 3: Enable if needed

```bash
bbx plan enable <KEY>
```

Expected exit 0. On failure, stop and surface the error.

## Step 4: Trigger

If the user supplied build variables, pass each with `--var key=value`:

```bash
bbx build trigger <KEY> --var <KEY1>=<VAL1> --var <KEY2>=<VAL2> -o json > /tmp/bbx-trigger.json
```

Capture `buildResultKey` (e.g. `RES-TES-42`) from the JSON output. If exit ≠ 0, surface the error and stop.

## Step 5: Poll until finished

`bbx build get` returns the result with `lifeCycleState` ∈ `{Queued, Pending, InProgress, Finished, NotBuilt}`. Poll until `Finished`, with a hard cap of 30 attempts × 10s = 5 minutes.

```bash
KEY="<buildResultKey>"
for i in $(seq 1 30); do
  STATE=$(bbx build get "$KEY" -o json 2>/dev/null | python3 -c '
import sys, json
d = json.load(sys.stdin)
print(d.get("lifeCycleState",""))
' 2>/dev/null || echo "")
  if [ "$STATE" = "Finished" ]; then
    break
  fi
  if [ "$STATE" = "NotBuilt" ]; then
    # NotBuilt happens immediately when the plan has no runnable tasks.
    # Treat as terminal — break to surface as a failure to the user.
    break
  fi
  sleep 10
done
```

After the loop, fetch the final result:

```bash
bbx build get "$KEY" -o json
```

## Step 6: Report

Read these fields from the result JSON: `state`, `lifeCycleState`, `buildState`, `successful`, `buildDurationInSeconds`, `prettyBuildDuration`, `buildReason`, `failedTestCount`.

```
Build <KEY>:
  State: <state> (<lifeCycleState>)
  Duration: <prettyBuildDuration>
  Successful: <successful>
  Tests: <successfulTestCount> passed, <failedTestCount> failed
  Reason: <buildReason>

[ if state != "Successful":
  Want me to investigate the failure? (yes/no) -> hand off to bbx-investigate-build with $KEY
]
```

If the polling cap was hit without reaching `Finished`, say so explicitly:

```
Build <KEY> is still in state <lifeCycleState> after <N> minutes. It may be
queued behind other builds, or the agent is stuck. You can re-check later
with `bbx build get <KEY>` or stop it with `bbx build stop <KEY>`.
```

## Variable handling

bbx passes variables as `bamboo.variable.<KEY>=<VALUE>` query params to `/queue/{planKey}`. The user-visible CLI uses `--var KEY=VALUE`.

If a value contains spaces or special chars, the user is responsible for shell quoting:

```bash
bbx build trigger RES-TES --var "MESSAGE=hello world" --var "TARGET=prod"
```

## Stopping a build

If the user changes their mind mid-poll, or you hit the polling cap:

```bash
bbx build stop <buildResultKey>
```

This may exit 4xx if the build already finished — that's safe, just report it.

## Output Format

Final report (after successful trigger + poll):

```
Triggered <KEY> at <buildStartedTime>.

Result: <Successful/Failed/NotBuilt> in <prettyBuildDuration>
<failedTestCount> failed test(s) / <successfulTestCount> passed.

[ if failed: "Investigate?" prompt ]
```

## Error Handling

| Symptom | Remediation |
|---|---|
| Exit 2 + `Plan ... not found` | The plan key is wrong. Re-search and ask user. |
| Exit 3 + `Unauthorized` | PAT lacks Build permission on this plan. Tell user, suggest a permission check in the Bamboo UI. |
| Triggered but `lifeCycleState: NotBuilt` immediately | Plan has no runnable tasks or the agent is offline. Report; don't retry. |
| Polling cap hit | Report state + suggest `bbx build stop` or re-poll later. |
| `Plan is not of type ...ImmutableJob` on `bbx build stop` | The build already finished. Treat as soft success. |

## Cross-references

- Plan not found / discovery → **bbx**
- Build finished failed and user wants to know why → **bbx-investigate-build**
- Auth failure → **bbx-setup**
