---
name: create-linear-issue
description:
  Synnax Linear issue conventions. Use when creating, splitting, retitling, relabeling,
  or rewriting a Linear issue.
---

# Linear issues

Team `Synnax` (key `SY`).

## Before you create

Search open and closed issues first. Update or link an existing one instead of filing a
copy. Put narrow work under a general issue as a sub-issue.

## Title

- Imperative, bugs included: "Fix `X` doing Y when Z", never "`X` does Y".
- Sentence case, no period, at most 60 characters, no package or area prefix.
- Backticks for identifiers, paths, and branches (`main`).
- No abbreviations: TypeScript, not TS.

## Labels

One Type, one Package, zero or one Area.

- **Type:** Bug, Feature, Refactor (no behavior change), Performance, Test (tests,
  flakes, compliance audits), Tooling (CI, builds, linters), Documentation, Release.
- **Package:** Core, Console, Pluto, Lyra, Driver, Client, Arc, Oracle, Cesium, Aspen,
  Freighter, Drift, Alamos, Media, X, Studio, Site, Integration (`integration/`), Go,
  Python, TypeScript, C++ (a whole language), Cross-cutting (several packages).
- **Area:** Channels, Calculated channels, Ranges, Labels, Import and export, Analysis,
  Telemetry, Control, Line plots, Schematics, Tables, Logs, Projects, Panels (tabs,
  layout, windows, deep links), Views, Arc language, Arc editor, Arc automations,
  Devices, Racks, Tasks, NI, LabJack, OPC UA, Modbus, EtherCAT, HTTP, CAN, Serial, MQTT,
  ARINC 429, MIL-STD-1553, PagerDuty, Slack, Sift, LabVIEW, Epsilon3, Users, Access
  control, Statuses, Nodes and clusters. Use the integration (NI), not Tasks, for an
  integration-specific issue.

## Priority: when we act

| Priority | Meaning                                                               |
| -------- | --------------------------------------------------------------------- |
| Urgent   | Within hours: a blocked customer or release, broken production or CI. |
| High     | This cycle or next.                                                   |
| Medium   | This quarter. The floor for customer requests.                        |
| Low      | Someday. Never cancel for age.                                        |

For a customer request, link the request (for example the Plain thread). Never write a
customer's name in the repo.

## Estimate: human attention, not coding time

| Estimate | Shape                                                   |
| -------- | ------------------------------------------------------- |
| 1        | Under an hour. One small PR, no decision.               |
| 2        | A few hours. One PR with a new spec or repro.           |
| 4        | About a day. One design decision or several packages.   |
| 8        | Several days. Several PRs; give it sub-issues.          |
| 16       | Unknown. Needs a design and a split before work starts. |

## Description

Bugs use `## Problem`, `## Reproduce` (with version and platform), `## Done when`. Other
Types use `## Why`, `## Scope`, `## Done when`. "Done when" is a checklist and is
required; for a bug it includes a spec that fails before the fix.

Keep the description very short, with no fluff. Write in ASD-STE100. Write "TODO: owner
to fill" instead of guessing. Title an unconfirmed, possibly fixed bug "Verify …".

## Rewriting an issue

- Keep all content; move it under the headings.
- Copy every image, video, and embed verbatim, stripping only `?signature=`. Convert raw
  `<img>` tags to `![alt](url)`; Linear breaks them.
- In Progress or In Review: change fields only, never the body.

## After saving

- `save_issue` can report success without applying. Check the returned `updatedAt` and
  title; save again if either is stale.
- Marking an issue Done adds it to the current cycle. Clear it with `cycle: null` when
  the work was not done this cycle.
