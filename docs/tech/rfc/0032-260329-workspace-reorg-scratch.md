# Workspace Reorganization — Projects, Panels, Components

## 1. The Problem

Synnax today has no clear strategy for organizing system configurations. At the heart of
the friction is a single structural mistake: **the mosaic does two incompatible jobs at
once.** It is both the stable home for an operations view AND the scratchpad for ad-hoc
exploration. Those are at war. The minute an operator peeks at a task config, they
vandalize their operations setup to make room — because the mosaic is the only place
anything can render.

### Symptoms

- "Workspaces" only control console layout. They don't store references to tasks,
  calculated channels, arcs, calibrations, or any other system configuration. There is
  no single artifact that represents "the configuration of this system."
- Workspaces are single-console. You cannot version, share, or export a coherent
  configuration for a 10-console launch control system.
- Ad-hoc creations (a quick line plot to investigate an anomaly) become permanent
  workspace resources named "Line Plot," "Line Plot," "Line Plot" — the "soup problem."
- Synnax permits operating with no active workspace, which makes scratch use cases feel
  easy but produces "why didn't my work save?" confusion.
- Some visualization forms auto-save; others (notably tasks) require explicit save.
- Operators report a lack of "spatial awareness" — they can't tell what configuration
  context they're in or where things belong.

Every one of these symptoms traces back to the same root cause: a single surface trying
to be both a long-lived shared operations view AND an ephemeral scratch surface. Two
distinct jobs require two distinct affordances.

---

## 2. Principles

These are the design principles this RFC commits to. Every later decision derives from
them.

1. **Two concepts: Project and Panel.** Project is the unit of identity, sharing,
   snapshots, and replication. Panel is the in-console working surface — a tab that owns
   a mosaic. The old "workspace" concept dissolves into these two.
2. **Split the mosaic's two jobs.** Mosaic is the inner layer (arrangement within a
   panel). Panel tab strip is the outer layer (which bounded context am I in). They are
   never conflated again.
3. **Global addressing, project-scoped ownership.** Every resource has a globally unique
   address. Every resource has a single owning project (or the cluster). Cross-project
   access is read-only by default; edits happen at the source.
4. **Components are the universal configuration-reuse primitive.** Any resource can be a
   Component. Instances of a Component reference it; per-property overrides allow
   controlled local divergence. Convergent across Figma, Unity, Ignition, Wonderware,
   React, Notion, Helm.
5. **Auto-save everywhere.** No save buttons. Drafts auto-save to console-local storage;
   project resources auto-save to the project on the server.
6. **You are always in a project.** No ambiguous "no project" state. The console
   auto-creates a default personal project on first launch.
7. **UI placement encodes scope.** Project on the left sidebar (scope). Panels on the
   top tab strip (current view). The position of a control teaches its semantics.
8. **Affordance weight matches gesture frequency.** Frequent gestures (switching panels)
   get lightweight affordances (tab clicks). Infrequent ones (switching projects) get
   heavy affordances (sidebar selection).
9. **Industry-standard mental models.** Tabs-for-views, sidebar-for-scope,
   components-for-reuse, drafts-and-archive lifecycle — Synnax follows the patterns
   users already know from browsers, IDEs, design tools, and industrial control systems.

---

## 3. The Architecture

### 3.1 Project

A **Project is a self-contained working context.** It contains everything required to
describe, modify, snapshot, replicate, and (where applicable) operate that context.

"Context" is intentionally broader than "system." It covers:

- Physical systems — Test Cell A, Fridge #7, Hotfire Test Stand
- Analytical contexts — Q1 Hotfire Analysis, post-campaign data review
- Reference libraries — Standards, shared components, organization templates

The criterion test: "Does this resource contribute to describing, modifying,
snapshotting, replicating, or operating this context?" If yes, it is project content.

Projects are:

- **Long-lived.** They evolve over time rather than being replaced.
- **Identity-bearing.** Each project has a name and an ontology identity; it's the unit
  of sharing, snapshots, fleet templating, and export/import.
- **Single-shape.** There are no formal project "types." Creation flows offer different
  starting points (blank, from a component, from an export), but the underlying data
  model is identical.

### 3.2 Panel

A **Panel is a tab in the project's panel tab strip.** It owns a mosaic — its own
arrangement of visualization tabs and splits.

Panels exist in two states:

- **Draft.** New panels start here. Visible only in the current console; auto-saved to
  console-local storage so they survive crashes, refreshes, and app quits.
- **Project panel (default state).** Persistent, server-side, visible to all consoles
  connected to the project.

Promotion is implicit: renaming a draft from its default "Untitled" name graduates it to
a project panel (Figma-style). Explicit "Save to project" is also available for clarity.
Demotion back to draft is rare but possible via right-click.

### 3.3 Component

A **Component is any resource designated as reusable.** Other resources can become
**Instances** of a Component — they reference it and inherit its properties.
**Per-property Overrides** allow local divergence on specific properties without
breaking the link.

This single primitive subsumes:

| Problem                           | Component-model solution                          |
| --------------------------------- | ------------------------------------------------- |
| Cross-project resource use        | Instance of a component owned by another project  |
| Fleet templating                  | Many project-instances of one project-component   |
| Schematic symbols                 | Specific instances of the general component model |
| Reusable patterns (panels, tasks) | Components at the appropriate granularity         |
| "Copy from template"              | Detach an instance into a standalone resource     |

Components work at any granularity: channels, tasks, schematic symbols, panels,
projects. They can compose (a project-component contains panel-components which contain
visualization-components). Components can live in any project or in a shared library.

Edits at the Component source propagate to all Instances except where a property is
overridden. Overrides are per-property (not whole-resource), which lets a fleet operator
override one calibration without divorcing the rest of a channel from its template.

---

## 4. What a Project Contains

### 4.1 Owned resources

A project owns the following, via `owned_by` ontology relationships:

**Hardware orchestration:**

- Tasks (DAQ tasks, arc tasks)
- Arc programs

(Device configurations are NOT a project-level concept. Devices are cluster-global
infrastructure; their identity lives with the cluster.)

**Data definitions:**

- Calculated channels — the formula is project config; the cluster stores the resulting
  data stream
- Channel aliases — see §5.2
- (Calibrations are not a separate resource; they're either task input scaling or
  calc-channel formulas.)

**Presentation:**

- Panels — with their mosaic structure
- Visualizations — plots, schematics, tables, logs. Owned by the project, referenced by
  panels.

**Organization:**

- Project-scoped groups (organize project-owned resources)
- Project-scoped views (saved queries on project data)
- Profiles (named override sets — deferred design; see §10)

**Operational artifacts produced within the project:**

- Ranges (captured by arcs / tasks)
- Status entries from arcs / tasks
- Logs from arcs / tasks
- Annotations on ranges

**External references:**

- Component instances pointing to components in other projects or the shared library

### 4.2 Project structure

The internal ontology of a project is flat: panels, visualizations, channels, tasks,
arcs are all direct children of the project node via `owned_by`. Groups are an optional
secondary organizational layer that the project also owns.

Specifically:

- **Visualizations are project-owned**, not panel-owned. Panels reference visualizations
  as component instances. A single visualization can appear in multiple panels via
  multiple instances; each instance can have per-property overrides (one panel zooms a
  plot to range X, another to range Y).
- Deleting a panel does not destroy its visualizations.

---

## 5. Cluster-Scoped Resources and Cross-Cutting Concerns

### 5.1 Truly cluster-global resources

A small set of resources lives at the cluster level, not inside any project:

- **Users, roles, policies.** Identity and permissions are cluster-wide. The existing
  Synnax permission system handles all access control — this RFC adds no new permissions
  concepts.
- **Schematic symbols.** A shared library of reusable graphical primitives.
- **Labels.** Cluster-global. Labels play a central role in cross-project analysis;
  consistent labeling across projects is what makes "find all ranges labeled
  hotfire-success" work.
- **Devices.** The physical device identity (LabJack #42, NI cDAQ-9189-1A2B3C4) lives at
  the cluster level. Project-level configuration of how a project uses a device lives in
  the project's tasks.

### 5.2 Channels: cluster identity, project naming

Channels are split into two flavors:

- **Raw channels** are **cluster-owned**. Their identity (`channel.Key`) is
  cluster-wide; their data stream is stored once in Cesium. Drivers and device tasks
  create them. They are not duplicated per project.
- **Calculated channels** are **project-owned**. The formula is part of project
  configuration; the resulting data stream is still cluster-addressable.

Channel **aliases** provide project-scoped naming on top of cluster channel identity.
This generalizes Synnax's existing range-alias system to multiple levels:

- **Range alias** (existing) — test-specific override.
- **Profile alias** — for Gen A / Gen B configurations within a project.
- **Project alias** — the project's stable vocabulary.
- **Cluster default** — the raw channel's registered name.

Name resolution walks up the context hierarchy from most-specific to least. The existing
range-alias semantics are preserved; project and profile aliases are added as new
layers.

Full multi-level alias design is **deferred** until other architectural pieces
stabilize.

### 5.3 Operational artifacts: scope inherits from producer

Ranges, status entries, and logs are produced by something. That producer's scope
determines the artifact's scope:

- Cluster-infrastructure producers (drivers, nodes, connections) create
  **cluster-scoped** artifacts.
- Project-runtime producers (arcs, tasks running within a project) create
  **project-scoped** artifacts.

Both are globally queryable for cross-cutting analysis (fleet status, log aggregation
across projects).

### 5.4 Groups and Views: scope inherited from contents

- **Groups** can be either project-owned (organize project resources) or cluster-shared
  (organize cluster-global resources like schematic symbols). Scope is inherited from
  what the group contains.
- **Views** (saved queries) can be either project-owned or cluster-shared depending on
  the view's intended scope.

---

## 6. Cross-Project Sharing

Two patterns. Inheritance, originally proposed as a third, dissolves into the Component
model.

### 6.1 Reference (live read-only link)

Project B includes a resource from Project A by global ID. The link is read-only and
live — source edits propagate to all viewers; viewers cannot edit without forking.

Use cases: analyst includes operational ranges in an analysis project; a panel in one
project displays a visualization owned by another project.

### 6.2 Copy (independent fork)

Project B duplicates a Project A resource. The result is a new resource owned by Project
B; no link to the source. No propagation in either direction.

Use cases: "Start mine from this template panel."

### 6.3 Component instance (inheritance with overrides)

This is the Component model from §3.3. Project B is an instance of Project A (designated
as a Component). Project B inherits all of Project A's resources; per- property
overrides on the instance allow controlled local divergence.

Use cases: Rigetti's 20 fridges from one reference; Honeywell test cells of similar
shape; reusable panel layouts shared across projects.

The propagation rules are clear:

- **Reference** — source edits propagate; viewers can't edit.
- **Copy** — no propagation, ever.
- **Component instance** — source edits propagate to instances except where overridden.

---

## 7. Cross-Cluster Export / Import

A project export contains:

- All resources owned by the project (full content)
- All external references (paths only)

On import to another cluster, references must be rebound:

- Raw hardware channels → match to new cluster's devices
- Cross-project references → match to new cluster's projects
- Schematic symbols, labels, users → match by name or prompt

**Proposed approach: semantic-name matching with hybrid rebind.** Auto-match unambiguous
logical paths; prompt the user for ambiguous cases. Standard in industry (TIA Portal,
Ignition, Wonderware all do this).

The semantic-name approach requires resources to have stable logical paths within their
project. Raw hardware-channel references in particular are the hardest case — you cannot
move a physical sensor between clusters; you have to rebind to whatever local hardware
the target cluster has.

---

## 8. Panel and Visualization UX

### 8.1 Panel-centric mental model

The user thinks "I'm adding a plot to my panel." Visualizations _feel_ panel-local even
though they're project-owned underneath. The project-level reality is exposed only when
actually useful (reuse, multi-panel display, the sidebar drawer). This matches Figma's
component pattern: most users create components without realizing they've created
components; the abstraction surfaces only when reuse happens.

### 8.2 Tab strip is the central panel navigation

All published panels in the project are visible in the tab strip when the project is
active. No separate browser, drawer, or picker for panels.

- **Drafts** are also in the tab strip, with a visual distinction (italic name, dot
  indicator, or lighter weight — exact treatment TBD).
- **Tab groups** (Chrome-style) provide organization for projects with many panels.
- **Horizontal scroll / overflow** handles long strips.
- **Grid view (Figma-style)** is reserved as a future addition for very complex
  deployments. Out of current scope.

The tab strip IS the central navigation. No "where do I find a panel?" problem because
they are all visible.

### 8.3 Close semantics — non-destructive

Close × is non-destructive across panels and visualizations. Explicit destruction is a
separate, deliberate action.

- **Draft panel:** × visible on tab. Click × → draft is marked closed (dims visually).
  Remains in the tab strip; click to re-activate. Auto-archives after N days closed.
- **Project panel:** no × on tab. Right-click → **Delete** (destructive, with
  confirmation) or → **Move to drafts** (rare).
- **Visualization instance:** close-tab removes the instance from the panel. The
  visualization moves to the Drafts section of the Visualizations drawer if no other
  instances exist. Auto-archives after N days unused. Explicit destruction via
  right-click → Delete.

**No save prompt on close.** Content auto-saves during the session; close × is safe
(non-destructive). Undo-toast covers accidental closes for immediate recovery.

### 8.4 Visualizations sidebar drawer

A left-rail drawer for the project's visualizations, opened on demand (matching the
existing Synnax drawer pattern for Channels, Tasks, etc.). Two sections:

- **Unlabeled top section** — visualizations currently in use (referenced by at least
  one panel). The everyday section.
- **Drafts** — visualizations not currently in any panel but recent. Labeled section.

**Lifecycle:**

- New visualization → in-use
- Close last instance → moves to Drafts
- After N days in Drafts → auto-deleted
- Restoration = drag/click into a panel (moves out of Drafts)

No always-visible Archive surface; drafts are auto-deleted after the window. The exact N
(30 days follows industry precedent — macOS Trash, Figma deleted files, Gmail) is a TBD
calibration.

### 8.5 Vocabulary

- **States:** Draft (labeled, transient) and (default — no name needed, just "a panel in
  the project").
- **Actions:** Save to project (Draft → Project), Move to drafts (Project → Draft,
  rare), Delete (any → gone, explicit).
- Promotion is likely **implicit via naming**: a draft starts "Untitled"; renaming
  graduates it (Figma-style). Explicit "Save to project" available.

### 8.6 Auto-save behavior

- **Drafts** auto-save to console-local storage. Preserved across window close, app
  quit, refresh.
- **Project resources** auto-save to the project (server-side). Available to all
  consoles on the project.
- **Tasks today require explicit save** — this is an inconsistency with the auto-save
  principle. Long-term: migrate tasks to auto-save + explicit "Apply" action for
  hardware activation. Short-term: accept the inconsistency.

---

## 9. Console UI Surface

```
┌──────────────────────────────────────────────────────────┐
│ [Project ▾] │   Panel tab strip                  │ Sys │
├────┬────────┴────────────────────────────────────┴──────┤
│    │                                                    │
│  R │       Active panel's mosaic                        │
│  a │       (tabs, splits, visualizations)               │
│  i │                                                    │
│  l │                                                    │
│    ├────────────────────────────────────────────────────┤
│    │  Viz toolbar (bottom drawer) — context-sensitive   │
└────┴────────────────────────────────────────────────────┘
```

**Top bar (left to right):**

- Window controls
- **Project selector** — opens project switcher; defines current scope
- **Panel tab strip** — all panels in the active project; current view
- Command palette
- System badges (cluster, user, version)

**Left rail drawers** (each on-demand, single-click preview, double-click open):

- Channels, Ranges, Devices, Tasks, Arcs, Statuses (existing)
- **Visualizations** (new) — see §8.4
- Users, Schematic Symbols (cluster-global drawers)

**Bottom drawer:**

- Visualization toolbar — context-sensitive to the active visualization. Unchanged from
  today.

**Center:**

- Active panel's mosaic (tabs, splits, visualizations within the panel).

---

## 10. Open Areas and Deferred Design

### 10.1 Fleet management (deferred — needs trade studies)

Three critical pressures confirmed by users:

- **Cross-project propagation semantics** — when a template changes, how do updates flow
  to instances? Live? Opt-in per instance? With diff review?
- **Override visibility** — each instance must clearly show inherited vs. locally
  overridden vs. locally added (Unity prefab variants are the model).
- **Fleet-wide views** — a surface that summarizes N projects at once.

The major unresolved question: is the fleet-wide surface a _cross-project surface_ (new
top-level concept outside any project) or a _fleet project_ (aggregator project that
references its instances)? Both have real trade-offs and warrant a dedicated design
exploration.

### 10.2 Profiles (lower priority — not blocking)

Profiles are a project-level "mode" that activates coordinated variant selections across
multiple components (Figma modes / Helm environments style). Real use case: Honeywell
test cell switching between Gen A and Gen B generator configurations while preserving
project identity.

Lower-priority in the design chain; will revisit once components ship and we know
whether instances alone are sufficient.

### 10.3 Snapshots and version control (wider redesign)

The current per-resource immutable-copy snapshot mechanism is too granular for the new
model. We need at minimum:

- Project-level snapshots
- Component-version pinning per instance
- Defined behavior for "snapshot a project that instantiates an evolving component"

This is its own design problem — not solved within this RFC. The current per- resource
snapshot mechanism stays in place; the wider version-control rework follows.

### 10.4 Channel aliases (multi-level, partial)

The four-layer alias resolution (range → profile → project → cluster default) is the
working direction. Full design — write semantics, conflict resolution, UI for managing
aliases at each level — is deferred until the component model stabilizes.

### 10.5 Tab groups, visual treatment, auto-archive window

Smaller calibrations not yet decided:

- Tab groups (Chrome-style) for organization in projects with many panels — concept
  noted, not detailed.
- Visual treatment of draft tabs (dot, italic, lighter weight) — implementation detail.
- Exact N for draft auto-archive (likely 30 days; industry default).

### 10.6 Multi-window model

The current branch implements per-window panel state. With panels now project-scoped
(server-side), the relationship between window-local panel state and project-level
panels needs explicit design:

- Each window shows its own subset of the project's panels?
- Or each window shows the same set (the project's published panels)?
- Detached single-panel windows for multi-monitor setups?

---

## 11. Migration

Out of scope for this RFC's architecture work. Will require:

- Migrating existing Synnax workspaces → projects + panels
- Converting workspace-scoped visualizations → project-owned + initial panel references
- Console-app state migration (the v10 work on the `sy-4193-panels` branch is a
  precursor)
- Forward compatibility for in-flight customer projects

A dedicated migration plan follows the architecture lock-in.

---

## 12. Risks and Mitigations

**Tab proliferation.** Panels accumulate over a project's lifetime. _Mitigation:_ tab
groups (planned); horizontal scroll/overflow; future grid view; periodic prompts to
delete or archive panels that haven't been touched in long periods.

**Component-model complexity.** A genuinely new concept for users to learn.
_Mitigation:_ the panel-centric UX hides the abstraction until it's needed; users
discover components only when they create reusable patterns. Same approach as Figma's
components model, which has been adopted successfully by non-technical users.

**Cross-project reference drift.** A project that references resources from another
project breaks if the source project changes incompatibly. _Mitigation:_ per-property
overrides allow defensive pinning; future version-control work will introduce
component-version pinning.

**Multi-window coordination.** Detaching panels into separate OS windows is useful for
multi-monitor setups but creates window-local state to coordinate with project- level
state. _Mitigation:_ explicit design needed (see §10.6); the existing Drift
infrastructure handles this kind of state synchronization.

**Channel alias resolution complexity.** Multi-level resolution (range → profile →
project → cluster default) is more complex than today's range-only model. _Mitigation:_
lazy — design the resolution algorithm carefully; provide a "resolve" diagnostic in the
UI showing which level a name resolved at; preserve existing range-alias behavior
unchanged.

**Tasks-vs-everything-else save inconsistency.** Tasks require explicit save today while
panels/vizes auto-save. _Mitigation:_ long-term migration of tasks to auto-save +
explicit Apply; short-term acceptance of inconsistency.

---

## 13. Status & Open Questions

### Locked

- Two-level model: Project + Panel.
- Naming: Project (top-level), Panel (inner-tab).
- Component + Instance + Override is the universal configuration-reuse primitive.
- Per-property overrides on instances.
- Project ownership principle: global addressing, project-scoped ownership.
- Panel-centric mental model.
- Tab strip is central panel navigation; future grid view for complex deployments.
- Visualizations are project-owned; panels reference via instances.
- Visualizations sidebar drawer with In Use + Drafts sections.
- Close × is non-destructive across panels and visualizations.
- Drafts auto-save to console-local; project resources auto-save to server-side.
- Draft / unlabeled-default vocabulary.
- Channels: raw cluster-owned, calculated project-owned; aliases multi-level.
- Operational artifacts inherit scope from producer.
- Labels cluster-global; groups scope-inherited from contents.
- Out of scope: permissions, fleet management (deferred), profiles (deferred).

### Open

- Fleet management surface (cross-project vs. fleet project) — needs trade studies.
- Snapshot / version-control rework — wider design problem.
- Multi-window coordination semantics.
- Channel alias write semantics and conflict resolution.
- Tab groups detail (creation, management, visual treatment).
- Exact N for draft auto-archive (likely 30 days).
- Visual treatment of draft tabs.
- Cross-panel visualization reuse mechanics (drag tab vs. right-click menu).
- Panel creation flow specifics.
- Migration plan from existing workspaces.
- Task auto-save migration.

---

## Appendix A: Industry Precedent

The model in this RFC draws on convergent designs across very different tools:

- **Figma** — components + variants + per-property overrides; drafts persistence; team
  libraries.
- **Unity** — prefab + prefab variants with per-property override.
- **Ignition (Inductive Automation)** — project inheritance with resource-level
  overrides; "local wins" conflict resolution.
- **TIA Portal** — project-scoped configurations with hardware rebind on import.
- **Wonderware ArchestrA** — galaxy templates + instances with parameter bindings.
- **Open MCT (NASA)** — operational role-based layouts; phase-based configurations.
- **Grafana** — folders + dashboards (two-level); template variables for variation.
- **VS Code** — file tabs + project sidebar; settings hierarchy (global → workspace →
  folder).
- **Notion** — synced blocks for cross-page reuse; drafts and archive lifecycle.
- **Helm / Terraform** — values overlay over chart templates.

These independently converged on the same primitives — Component + Instance + Override,
two-level navigation (scope + view), auto-save with explicit publishing — because
they're the minimum sufficient model for configuration at scale. Synnax inherits the
convergence.
