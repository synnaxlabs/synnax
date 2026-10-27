---
name: plan-to-issues
description:
  Turn the phase table at the end of an RFC into Linear issues, one per phase, linked by
  blocking relations under one project and labeled `agent-ready` so the agent dispatcher
  can start them. Use when the user invokes /plan-to-issues or asks to file, create, or
  queue the issues or agent work for an RFC's phases.
---

# Plan to issues

Each row of an RFC's phase table becomes one Linear issue in team `SY`. The agent
dispatcher (`.github/workflows/dispatch.agents.yaml`) reads these issues, so the fields
below are a contract. Load the `create-linear-issue` skill first: its rules apply to
every issue here. Use the Linear MCP tools or the Linear API.

## Steps

1. **Check the RFC is on `main`.** Agents read it from a `main` checkout. Stop if it is
   not merged.
2. **Parse the table.**
   `uv run --project .github/scripts .github/scripts/phase_table.py <rfc.md>`. On an
   error, fix the table in an RFC PR first. Never hand-copy rows.
3. **Find existing work.** Search team `SY` for the RFC path. A phase that already has
   an issue keeps it: reuse it for relations and do not file a copy. A reused issue that
   gets `agent-ready` also gets the project, the blockers, and the `## Agent` block.
   Leave `agent-ready` off a reused issue that is In Progress or In Review, since its
   body must not change. Reuse the RFC's Linear project if it has one.
4. **Show the plan and wait.** One line per phase: title, labels, estimate, blockers,
   and if it gets `agent-ready`. Name the project to use or create. Ask once; the user
   can drop `agent-ready` from a phase that a person must do.
5. **Create the project** if there is none, named after the RFC title, team `SY`.
6. **Create the issues in phase order**, so each blocker exists before the issue it
   blocks:
   - **Title**: The scope cell, unchanged.
   - **Labels**: One Type, one Package from the touches cell, zero or one Area, plus
     `agent-ready` unless the user dropped it.
   - **Estimate**: 1 or 2, from the `create-linear-issue` table. A phase that needs 4 or
     more is too big: split it in the RFC.
   - **Priority**: The RFC's umbrella issue priority, else Medium.
   - **State**: Todo. **Project**: The RFC's project. **Blocked by**: The issues of the
     phases in the depends-on cell.
   - **Description**: The template below.
7. **Verify.** Read each issue back, reused ones too. Check the labels, project, state,
   blockers, and on each `agent-ready` issue the `## Agent` block. Save again when a
   field is stale.
8. **Report** one line per phase: phase number, `SY-####`, title.

## Description template

The dispatcher parses the two items under `## Agent`. Keep their form exactly.

```markdown
## Why

Phase N of
[RFC NNNN - Title](https://github.com/synnaxlabs/synnax/blob/main/docs/tech/rfc/NNNN-slug.md).

## Scope

<The phase's prose from the RFC, in two or three sentences.> Touches `a`, `b`.

## Done when

- [ ] One PR titled `SY-####: <title>` merges into `main` alone.
- [ ] <Each check the RFC names for this phase.>

## Agent

- RFC: `docs/tech/rfc/NNNN-slug.md`
- Phase: N
```

## Rules

- One issue per row. Never merge or split rows here; change the RFC instead.
- `agent-ready` is the only signal the dispatcher acts on. Add it only with the user's
  agreement in step 4.
- A blocker that is not `agent-ready` still blocks. The dispatcher waits until it is
  Done.
