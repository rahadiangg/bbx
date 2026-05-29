---
name: bbx-extract-config
description: Use this skill to extract the complete configuration of a Bamboo plan — its Bamboo Specs Java source, structured JSON view, artifact definitions, plan variables, parent project (metadata + variables + repositories), and every linked deployment project (config, Specs, environments, environment variables, requirements, agent assignments, versions). Trigger when the user says "extract plan config", "give me everything about plan X", "I want to migrate this plan to <other CI>", "dump pipeline X", "what does plan Y actually do", or asks for a comprehensive plan export to feed into another system.
---

# bbx — Extract Plan Configuration

Walk a user through a complete config dump for one Bamboo plan + everything around it (parent project, linked deployments, all environment settings). The output is a single JSON document the user (or a downstream tool / AI agent) can use to understand or replicate the pipeline in any other CI system.

## Core Principles

1. **Read-only.** This skill never modifies Bamboo. No labels, no comments, no triggers, no deletes.
2. **Be explicit about secrets.** Bamboo masks sensitive values as `********` in variable listings. Surface this to the user — they'll need to re-enter actual secrets at the target system.
3. **Don't dump raw JSON to chat unless asked.** Summarise: how many stages/jobs/tasks, how many variables, how many linked deployments. Write the full bundle to a file (default `/tmp/bbx-extract-<plan-key>.json`) and tell the user the path.
4. **Be honest about what's reachable.** Trigger schedules and notifications live inside the Specs Java — bbx returns the source; parsing it for those fields is the agent's job. Build logs on Bamboo 8.x are gated behind session auth (separate concern; mention if user wants them).

## Prerequisites

bbx must have a configured context with a token that has *Read* permission on the target plan + project + deployment projects. If not configured: hand off to **bbx-setup** first.

The user must supply a **plan key** in the form `PROJECT-PLAN` (e.g. `ANVIS-ASD`).

## Step 1: Confirm the plan exists + capture project key

```bash
bbx plan get <PLAN_KEY> -o json
```

Extract `projectKey` from the response. Stop with a clear error if the plan doesn't exist (exit 2 + `Plan ... not found`) — ask for the right key.

## Step 2: Fetch plan-side config

Run these in sequence (none take more than a second each):

```bash
bbx plan spec    <PLAN_KEY>                 # Bamboo Specs Java source — THE primary artifact
bbx plan config  <PLAN_KEY>      -o json    # same info, structured
bbx plan artifact list <PLAN_KEY> -o json
bbx plan variable list <PLAN_KEY> -o json   # ******** for sensitive vars
bbx plan branch  list <PLAN_KEY>  -o json   # Bamboo plan branches
bbx plan vcs-branches  <PLAN_KEY> --all -o json   # raw branches in the repo
```

If any returns exit 3 (`Unauthorized`), the user's token lacks the right permission on that plan — stop and tell them.

## Step 3: Fetch project context

```bash
bbx project get        <PROJECT_KEY>        -o json
bbx project variable   list <PROJECT_KEY>   -o json
bbx project repository list <PROJECT_KEY>   -o json
bbx project spec       <PROJECT_KEY>        -o json   # bulk Specs export (all plans in project)
```

The bulk `project spec` is large (hundreds of KB for big projects). If you only care about the one plan, the per-plan `bbx plan spec` from Step 2 is enough.

## Step 4: Discover + walk linked deployment projects

```bash
bbx deployment project list --for-plan <PLAN_KEY> -o json
```

For each `id` returned (often zero or one), fetch the full deployment config:

```bash
bbx deployment project get   <ID> -o json    # includes embedded environments
bbx deployment project spec  <ID> -o json    # Bamboo Specs Java for the deploy side
bbx deployment project repository list <ID> -o json
bbx deployment version list  <ID> --max-results 5 -o json   # latest 5 versions
```

For each environment ID inside the `environments` array of the deployment project:

```bash
bbx deployment environment get             <ENV_ID> -o json
bbx deployment environment variable    list <ENV_ID> -o json
bbx deployment environment requirement list <ENV_ID> -o json
bbx deployment environment agent       list <ENV_ID> -o json
```

If `deployment project list --for-plan` returns `[]`, the plan has no deployment side — skip Step 4 entirely and tell the user.

## Step 5: Bundle + summarise

Assemble all the captured JSON into a single document with this shape:

```json
{
  "plan": { ... },
  "plan_spec_java": "...",
  "plan_config": { ... },
  "plan_artifacts": [...],
  "plan_variables": [...],
  "plan_branches": [...],
  "plan_vcs_branches": [...],
  "project": { ... },
  "project_variables": [...],
  "project_repositories": [...],
  "deployments": [
    {
      "project": { ... },
      "spec_java": "...",
      "repositories": [...],
      "versions": [...],
      "environments": [
        {
          "env": { ... },
          "variables": [...],
          "requirements": [...],
          "agents": [...]
        }
      ]
    }
  ]
}
```

Write to `/tmp/bbx-extract-<PLAN_KEY>.json` and tell the user the path + size.

## Output Format

```
Extracted configuration for <PLAN_KEY> (<plan-name>):

  Source plan: <PROJECT_KEY> / <plan-name>
  Stages × jobs × tasks: read from plan_spec_java (count if you can)
  Plan variables:    <n> (incl. <m> masked sensitive)
  Plan branches:     <n>     VCS branches: <n>
  Artifacts:         <n>
  Project variables: <n>     Project repositories: <n>
  Deployment projects: <n>
    └─ <n> environments total

Sensitive values are masked as "********". You'll need to re-enter the
real secrets at your target system.

Saved full bundle to: /tmp/bbx-extract-<PLAN_KEY>.json (<size> bytes)

Next steps:
  • Inspect the Bamboo Specs Java to understand task definitions:
      cat /tmp/bbx-extract-<PLAN_KEY>.json | jq -r .plan_spec_java
  • Hand the bundle to a translation step for another CI system.
```

## Error Handling

| Symptom | Remediation |
|---|---|
| `Plan ... not found` (exit 2) | Wrong key. Run `bbx plan list -o json` and ask the user. |
| `Unauthorized` (exit 3) on a sub-call | Token lacks permission on that resource. Surface which one; user may need a wider PAT. |
| `bbx deployment project list --for-plan` returns `[]` | Normal — the plan has no linked deployments. Skip Step 4 and continue. |
| Any `********` value | Always note in the summary. Don't quote masked values back as if they were real. |
| Plan `spec` 404 | Theoretical on weird Bamboo versions; tell the user and proceed without the Java source (the structured `plan config` JSON still works). |

## Cross-references

- Configuration is dumped — translation to a different CI system is OUTSIDE bbx's responsibility. Hand the bundle to the user's preferred AI agent + target-CI prompt.
- For triggering or investigating a build → **bbx-trigger-build** / **bbx-investigate-build**.
- For general lookups / non-extraction tasks → **bbx**.
