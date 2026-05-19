# Workspace Reorg — Working Question Map

Companion to `0032-260329-workspace-reorg-scratch.md`. Tracks the open architectural
questions, current status, and decisions log. Updated as the design firms up.

**Status legend:** ❓ untouched · 🔧 partial · ✅ decided

---

## 0. Use cases (the grounding)

Every other question is answered relative to who's doing what. Without this anchor we
keep arguing in the abstract. The RFC has fragments in Q9/Q10/Q12 but no systematic
catalog.

### Archetypes ✅

- **Operator** — runs tests, watches live data, executes commands. Daily,
  time-pressured.
- **Test engineer** — designs the test, configures hardware, builds visualizations that
  operators use. Weekly-to-daily.
- **Test director** — oversees a campaign, sets pass/fail, decides go/no-go. Per-test.
- **Fleet manager** — administers a fleet of similar projects (Rigetti's 20 fridges).
  Less common but high-stakes.
- **Analyst** — reviews historical data across tests/campaigns. Post-hoc.
- **Ops director / customer-facing rep** — demos, reviews, executive view. Occasional.

**Important role-mapping constraint:** Operator and test engineer **collapse to one
person in R&D** (SpaceX, Orbex early-stage), but are **distinct roles with different
technical capabilities in production** (Honeywell). The system must support both: a
merged role with full configuration access, AND a split role where operators have
restricted access to project config that test engineers control.

### Dominant workflows per archetype

**Operator** (🔧)

- Open console, resume last view.
- Watch live data (mostly passive monitoring).
- React to anomalies — peek at underlying channel / task / calibration (read-only).
- Run workflows / execute commanded actions.
- Quick post-test review on a captured range.
- Annotate ranges with notes / pass-fail.
- Hand off cleanly to next shift.

**Key constraints:**

- Operators rarely _edit panels_ (the project-scoped containers).
- Operators **frequently create ad-hoc visualizations** — random plots, schematics,
  tables for inspection. Slightly less than test engineers, but still often.
- Existing Synnax permission system (with built-in operator role) handles access control
  — out of scope for this RFC.

**Implications for design:**

- Ad-hoc visualization creation must be fast and non-destructive — this is a daily
  operator gesture, not an edge case.
- Default UI for operators de-emphasizes project-config-editing affordances but keeps
  visualization-creation prominent.
- Ephemeral panels are _primarily_ for ad-hoc visualization spins, both for operators
  and test engineers.

**Test engineer** (🔧)

- Configure hardware (tasks, devices, calibrations, channels).
- **Build visualizations** — a meaningful chunk of their time.
- Build / curate panels (group visualizations into role-based views).
- Author / iterate on Arc programs.
- Test and validate the system end-to-end.
- Snapshot before campaigns.
- Diagnose post-test issues.

**Biggest pains today:**

1. **No single source for end-to-end system configuration.** No clean way to import,
   export, or switch between system configurations. Today's "workspace" only covers
   layout, not tasks / calibrations / arcs / etc.
2. **The soup problem.** Ad-hoc review and analysis creates a mess of throwaway
   plots/schematics/tables that pollute the workspace and can't be cleaned up easily.

**Implications for design:**

- The Project concept must be the single source for system config (tasks, calibrations,
  arcs, channels references, schematics, panels) — and must support import/export and
  switching cleanly. This directly motivates the Project-as-identity-unit decision.
- The ephemeral panel tier solves the soup problem **for both test engineers AND
  operators**, since both create ad-hoc visualizations frequently.

**Fleet manager** (🔧)

- Maintain a reference/template project that propagates to instances.
- Push config updates across the fleet (new sensor type, updated calibration).
- Review fleet-wide health.
- Manage per-instance overrides — sanction them, prevent unwanted drift.
- Handle "this instance has diverged enough, detach it" decisions.

**Critical design pressures (all confirmed important by user):**

- **Cross-project propagation semantics.** Update model: live propagation vs.
  opt-in-per-instance vs. confirmation/diff review.
- **Override visibility.** Each instance must clearly show inherited vs. locally
  overridden vs. locally added. Unity prefab variants are the model.
- **Fleet-wide views.** A surface that summarizes N projects at once — fundamentally
  different from being-in-a-project.

**Major unresolved question:** is fleet-wide surface a _cross-project surface_ (new
top-level concept outside any project) or a _fleet project_ (aggregator project that
references its instances)? This is genuinely out of scope for quick decision — **needs
its own design exploration with trade studies between models.**

**Test director, analyst, ops director:** ❓ TBD

### Stress cases

❓ — TBD

**Status:** 🔧 — archetypes locked; workflows and stress cases TBD

---

## 1. Vocabulary

Blocking. We have wobbled between "workspace" and "project." Cannot proceed without a
stake in the ground.

- ✅ Top-level concept name: **Project**
- ✅ Inner-tab concept name: **Panel**
- Names for ephemeral vs persistent variants — **deferred** until mechanic is decided
  (Q3). Current lean: **Draft / Published** (Figma-style).

**Status:** 🔧 — top-level and inner locked; variant names deferred

---

## 2. Top-level concept definition (Project)

### Principle ✅

A **Project is a self-contained working context.** It contains everything required to
describe, modify, snapshot, replicate, and (where applicable) operate that context.

"Context" is intentionally broader than "system" — it covers analytical projects (Q1
Hotfire Analysis), shared libraries, research efforts, in addition to physical systems
(Test Cell A, Fridge #7). The "operate" verb is conditional; analytical projects don't
get operated, their other four properties still hold.

The criterion test: "Does this resource contribute to describing, modifying,
snapshotting, replicating, or operating this context?" If yes, it's project-owned.

### Project contents (proposed)

**Hardware orchestration:**

- Tasks (DAQ tasks, arc tasks)
- Arc programs

(Device configurations are _not_ a project-level concept. Devices are cluster-global;
their identity and config live with the device, not duplicated per project.)

**Data definitions:**

- **Raw channels** — cluster-owned. Cluster-wide `channel.Key` identity, created by
  drivers / device tasks. Projects don't own raw channels; projects use them via
  aliases.
- **Calculated channels** — project-owned (formula is project config; the cluster stores
  the resulting data stream by `channel.Key`).
- (Calibrations are _not_ a separate resource — they're either part of task config
  (input scaling) or implemented as calc channels.)
- **Channel aliases** — multi-level resolution, generalizing the existing range alias
  system:
  - Range alias (existing — test-specific override)
  - Profile alias (likely needed for Gen A / Gen B configurations)
  - Project alias (new — the project's stable vocabulary)
  - Cluster default (raw channel's registered name)
  - Resolution walks up the context hierarchy from most-specific to least.
  - **Multi-level model is OK as a starting point; full design deferred** until other
    architectural pieces resolve.

**Presentation:**

- Panels (with their mosaic structure)
- Visualizations (plots, schematics, tables, logs)

**Organization:**

- **Groups** — _can be either_ project-owned or cluster-shared depending on what they
  contain. (Example: schematic symbols are organized into a cluster-shared group;
  project-local groups organize project-owned resources.) Scope inherited from contents.
- **Labels** — **cluster-global**. Labels play a central role in cross-project analysis;
  consistent labeling across projects is what makes "find all ranges labeled
  hotfire-success" work. Moving from "project organization" to "cluster-shared library"
  alongside schematic symbols, users, roles.
- **Profiles** — _don't exist today_. A profile would be a project-level "mode" that
  activates a coordinated set of variant selections across multiple components (Figma
  modes / Helm environments style). Real use case: Honeywell test cell switching between
  Gen A and Gen B generator configurations while preserving project identity.
  **Lower-priority in the design chain** — not blocking current architecture work. Will
  be revisited once components ship and we know whether instances alone are sufficient
  or profiles-as-project-modes are needed.
- **Views** — existing Synnax concept (saved queries on ranges/statuses). Can be
  **project-owned OR cluster-shared** depending on the view's intended scope; the
  ontology already supports both via owning relationships.

**Operational artifacts** — scope inherits from producer:

- Driver / node / connection status, driver logs → **cluster-scoped** (producer is
  cluster infrastructure)
- Arc / task emitted status, arc logs, arc-captured ranges → **project-scoped**
  (producer is a project's runtime)
- Annotations (parented to ranges → follow range's scope)

**General rule:** an operational artifact inherits the scope of its producer.
Cluster-infrastructure producers create cluster-scoped artifacts; project-runtime
producers create project-scoped artifacts. Both are globally queryable for cross-cutting
analysis (fleet status, log aggregation).

**External references:**

- Component instances pointing to components in other projects or shared libraries

### Structural relationship ✅

- **Project owns visualizations**, not panels (decided 2026-05-19).
- Project also owns panels, channels, tasks, arcs as direct children.
- **Panels reference visualizations as component instances** — they don't own them.
- A visualization can appear in multiple panels via instances; each instance can have
  per-property overrides (panel A zooms plot to range X, panel B to range Y).
- Deleting a panel doesn't destroy its visualizations.

**Q5 closed:** can a viz be in multiple panels? **Yes via instances.**

Rationale: real customers want the same plot in multiple panels (overview/detail, shared
key telemetry across operator + test director views); per-instance overrides unlock
useful patterns (live in one panel, range-specific in another); component model handles
this natively; deleting a panel without losing visualization work is the right safety
default.

### No formal project "types"

Operational vs. analytical vs. empty were considered. Rejected. All projects have the
same shape; _creation flows_ offer different starting points (instantiate from a project
component, blank empty, etc.). Same model as Figma "Create from template."

Why: typing locks decisions upfront. An analytical project that later needs to acquire
data shouldn't have to convert types.

### Create / delete / duplicate semantics

- **New empty** — blank project with name + one default panel.
- **New from component** — instantiate a project-marked-as-component. Owned by you, with
  live link to source; per-resource overrides allowed.
- **Duplicate** — deep copy, no link. Independent thereafter.
- **Delete** — destroy owned resources. Behavior for dangling references from other
  projects: TBD (probably auto-convert to copies, or warn).

### Identity

Project has a name, description, owner. Snapshots / versioning interact with the wider
version control rework (Q7).

**Status:** 🔧 (structure proposed; remaining: dangling-reference behavior on delete,
external library scope, naming/path conventions)

---

## 3. Panel definition

- What does a panel contain (visualizations? mosaic? settings? time context?)
- Lifetime: ephemeral vs persistent
- Mechanic for the ephemeral → persistent transition
- Cardinality: how many panels per top-level concept?
- Cross-concept sharing / templating (move a panel design between top-level concepts?)
- Default panels on new top-level-concept creation

### Working direction: panels are just one resource type under the component model

If we adopt the Component + Instance + Override primitive (see Q4), panel reuse and
cross-project sharing fall out automatically. A panel can be a Component; another
project can instantiate it; overrides allow per-instance variation.

**Status:** 🔧 (scoping decided; mechanics open; component model under consideration)

---

## 4. Resource scoping

- Which resources are project-scoped? (Tasks? Arcs? Visualizations? Calibrations?)
- Which are truly cluster-global? (Users? Roles? Schematic symbols?)
- How do project-scoped resources get referenced from other projects?
- Profiles — what's the scope?
- (Permissions: out of scope — existing Synnax permission system handles this.)

### Working model (in progress)

**Principle: Global addressing, project-scoped ownership.**

Every resource has an _owning project_ (modeled as a relationship in the ontology).
Anyone can read any resource via its global ID. Only the owning project can edit.
"Global" is about addressing, not ownership.

**Resources we believe are project-owned (not cluster-global):**

- Calc channels (the formula is project config)
- Tasks, arcs, panels, profiles
- Visualizations (plots, schematics, tables, logs)
- Raw channels created by a project's device tasks — owned by that project, even though
  the physical sensor is shared

**Resources that are genuinely cluster-global:**

- Users, roles, policies
- Schematic symbols (shared library)
- Ranges? — TBD; probably project-owned (the test that captured them) but freely
  referenceable.

### Cross-project sharing patterns — collapsing into the component model

**Proposed unifying primitive: Component + Instance + Override.**

Convergent across non-trivial systems (Figma, Unity prefabs, Ignition templates, React,
Notion synced blocks, Wonderware, Helm) — independent designers all arrived at the same
triplet because:

- Reuse without inheritance produces drift.
- Inheritance without override produces brittleness.
- This triplet is the minimum sufficient model.

**The mechanism:**

- Any resource can be designated as a Component (status flag — same data otherwise).
- An Instance is a resource that points to its Component.
- Overrides are per-property local changes on an Instance.
- Edits to the Component propagate to all Instances except where overridden.

**What collapses into this primitive:**

| Old concept              | New mechanism                                         |
| ------------------------ | ----------------------------------------------------- |
| Cross-project Reference  | Instance of a component owned by another project      |
| Fleet inheritance        | Many project-instances of one project-component       |
| Schematic symbols        | Already shaped like components; generalizes naturally |
| Profiles (Gen A / Gen B) | Named override sets applied conditionally             |
| Snapshots                | Immutable component versions                          |
| "Copy from template"     | Detach an instance (or re-create without the link)    |

Six concepts become one.

**Components at multiple axes:**

1. **Granularity** — channel, task, schematic symbol, panel, project. All use the same
   mechanism; only resource type differs.
2. **Composition** — components contain other components; transitive inheritance.
3. **Library/scope** — project-local components, shared-library components, both with
   same mechanism.

**Hard problems (real, need design work — not punted):**

- ✅ **Override granularity:** **per-property** (decided 2026-05-19). Required for fleet
  cases like overriding a single calibration value without divorcing the rest of the
  channel from its template. Commits us to a structured diff model per instance.
- **Versioning.** Live propagation vs. pinned versions. Lean: live default with optional
  pinning. _Snapshot interaction is its own wider design problem — see below._
- **Composition semantics.** Overrides on nested instances; don't bubble up.
- ✅ **Naming: "Component"** (decided 2026-05-19). React-component collision is
  internal-only and acceptable; users come from Figma/Ignition/Unity backgrounds where
  "component" is native vocabulary.

### Cross-cluster export / import

Project export = owned resources (full content) + external references (paths only). On
import to a new cluster, references need to be rebound:

- Raw hardware channels → match to new cluster's devices
- Cross-project references → match to new cluster's projects

**Proposed approach:** semantic-name matching with hybrid rebind — auto-match
unambiguous logical paths (`engine.pressure` → equivalent on target), prompt user for
ambiguous cases. Standard in industry (TIA Portal, Ignition, Wonderware).

**Status:** 🔧 (ownership principle locked; component direction open; rebind approach
sketched)

---

## 5. Visualization placement & lifecycle

Resolved by the project structure (Q2) + component model (Q4):

- Visualizations are **project-owned** (children of the project, not the panel).
- Panels **reference visualizations as component instances** — read-only by default,
  with per-property override capability (zoom range, axis config, etc.).
- A visualization **can appear in multiple panels** via multiple instances.
- **Moving a visualization across panels** = delete one instance, create another (the
  visualization itself isn't moved; only the references change).
- Deleting a panel **does not destroy** its visualizations — they remain in the project.

**Status:** ✅ resolved by Q2 + Q4.

---

## 6. UI surface

- Console UI zones at the highest level (rails, top bar, mosaic, drawers)
- Where workspace + panel selectors live and why
- Toolbar inventory: what we need, what must change
- Multi-window / multi-monitor model
- Drag-and-drop semantics (channels into viz, tabs across panels, panels across windows)

### Panel + visualization management — working notes

**Mental model:** panel-centric. The user thinks "I'm adding a plot to my panel." Vizes
feel panel-local even though they're project-owned underneath. The project-level reality
is exposed only when actually useful (reuse, multi-panel display). Matches Figma's
component pattern.

**Close-tab semantics:** non-destructive. Closing a tab removes the _instance_ from the
panel; the visualization is preserved.

**Visualizations sidebar drawer:** exists. Rail-icon, opens on demand (matches existing
Synnax drawer pattern). Two sections:

- _Unlabeled top section_ — vizes currently in use (referenced by at least one panel).
- _Drafts_ — vizes not currently in any panel but recently closed.

**Lifecycle:** new viz → in-use; close last instance → moves to Drafts; after N days in
Drafts → auto-deleted (no separate archive layer). Restoration = add back to a panel.

**Status:** 🔧 — mental model, close semantics, sidebar shape, viz lifecycle locked.
Still open: creation flows (drag/+/palette/right-click), cross-panel reuse, exact N for
draft auto-delete, panel-draft promotion mechanic.

### Panel navigation and discovery ✅

**Decision: tab strip is the central navigation. All panels live there.**

- All **published panels** are visible in the tab strip whenever the project is active.
  No separate browser, drawer, or picker.
- **Drafts** are also in the tab strip, with a visual distinction (italic name, dot
  indicator, lighter weight — exact treatment TBD).
- **Tab groups** (Chrome-style) for organization in projects with many panels — test
  engineer creates groups; operators see the structure.
- **Horizontal scroll / overflow** for projects that grow large.
- **Grid view (Figma-style)** as a future addition for very complex deployments (50+
  panels). Not in current scope; will revisit if pain emerges.

### Close semantics — non-destructive ✅

Unified across panels and vizes: close × is non-destructive everywhere. Explicit
destruction is a separate, deliberate action.

- **Draft panel:** × visible on tab. Click × → marks the draft as _closed_ (dims
  visually, signals "I'm done for now"). Draft remains in the tab strip; user can
  re-activate by clicking it again. Auto-archive after N days closed.
- **Project panel:** no × on tab. To remove from project: right-click → **Delete**
  (destructive, with confirmation). To return to draft state: right-click → **Move to
  drafts** (rare).
- **Viz instance** (already locked): close-tab is non-destructive. Viz moves to Drafts
  in the Visualizations drawer if no other instances exist; auto-archive after N days.
  Explicit destruction via right-click → Delete.

**No save prompt on close.** Close × is safe (non-destructive) and content auto-saves
during the session — there's nothing to save in the file-editor sense. If a user
accidentally closes something they wanted to keep, undo-toast covers immediate recovery.

**Why Position 2 over Position 1:** consistency between panels and vizes; matches
familiar mental models (Figma drafts, Notion drafts, email drafts, etc.); auto-archive
handles accumulation. Position 1 (destroy on close) was rejected because it was
inconsistent with the viz model and created a footgun for accumulated work.

### Vocabulary

- States: **Draft** (labeled, transient, console-local) / unlabeled default state ("a
  panel in the project")
- Actions: **Save to project** (Draft → Project), **Move to drafts** (Project → Draft,
  rare), **Delete** (any → gone, explicit)
- Promotion likely implicit via renaming a draft (Figma-style: "Untitled" stays draft
  until you give it a name).

### Save behavior ✅

- Drafts auto-save to console-local storage. Preserved across window close, app quit,
  refresh.
- Project panels auto-save to server-side project storage. Preserved everywhere, visible
  to all operators on the project.
- **Tasks today require explicit save** — this is an inconsistency with the auto-save
  principle. Long-term: migrate tasks to auto-save + explicit "Apply" for hardware
  activation. Short-term: accept the inconsistency.

---

## 7. Data model & sync

- Server-side schema for the top-level concept, panels, visualizations
- Auto-save semantics — what writes when, debouncing
- Snapshot granularity and lifecycle
- Conflict resolution for concurrent edits
- What persists local-to-console vs server-side

### Current snapshot mechanism (for reference)

Synnax today has a per-resource immutable-copy snapshot mechanism. Resources like
schematics carry a `Snapshot bool` flag; if set, the resource is immutable and only
rename actions are accepted. There is no project-level / cross-resource snapshot today.

### Snapshot + Component interaction (flagged as a wider problem)

The component model intersects with version control non-trivially. A project might
contain instances of components that live elsewhere; what does "snapshot this project"
mean when an instance's component evolves later?

This is its own design problem and **needs to be reworked as part of a wider version
control redesign — not solved within this RFC**. The current per-resource snapshot
mechanism is too granular for projects + components; we need at minimum:

- Project-level snapshots
- Component-version pinning per instance
- Defined behavior for "snapshot a project that instantiates an evolving component"

**Status:** ❓ — version control is a deferred wider design problem.

---

## 8. Migration

- Existing workspaces / visualizations on upgrade
- Console-app state migration (the v10 work on this branch is a hint)
- Backwards compatibility window

**Status:** ❓

---

## Decisions log

Each decision: what was decided, when, the rationale, and a link back to the question.

1. **Two-level model, not three.** Collapsed old {Project, Workspace, Mosaic} into
   {Top-level concept, Panel-with-mosaic}. Rationale: affordance weight should match
   gesture frequency; industrial tooling is overwhelmingly two-level (Ignition,
   Wonderware, Grafana, Open MCT); reduces cognitive load. (Q1, Q2, Q3)
2. **Panels are scoped to the top-level concept (server-side, shared).** A panel lives
   with its workspace/project on the server and is visible to all consoles connected to
   it; consoles choose which panels to display locally. Rationale: shared work surfaces,
   role-based pinning, snapshot inclusion. (Q3)
3. **Ephemeral tier is structurally necessary.** Some form of "this panel doesn't
   pollute shared state" must exist to solve the soup problem. The exact mechanic
   (manual pin, naming-as-promotion, implicit signals) is still open. (Q3)
4. **Top-level concept is "Project."** Rationale: "Project" is the only candidate that
   handles both operational containers (Test Cell A, Fridge #7) and analytical
   containers (Q1 Hotfire Analysis) without forcing. Industrial precedent (Ignition, TIA
   Portal, Wonderware). "Workspace" was rejected because reusing the same word for a
   different concept causes lexical confusion; "System" was rejected because Synnax
   itself is a control system; "Site"/"Installation"/"Plant" all break for analytical
   projects. (Q1)
5. **Inner-tab concept is "Panel."** Rationale: matches the tab-strip affordance
   directly; no collisions in existing Synnax vocabulary (sidebars are "drawers");
   "View" was rejected because it collides with Synnax's existing "Views" concept (saved
   queries on ranges/statuses). Reusing the now-freed word "Workspace" was considered
   for lower migration cost, but creates subtle confusion where users conflate the
   project-scope and view-scope containers. (Q1)
6. **Ownership principle: global addressing, project-scoped ownership.** Resources are
   reachable via global IDs from any project, but each resource has a single owning
   project that controls edits. "Global" is about _addressing_, not _ownership_.
   Replaces the RFC's prior "everything is global" framing, which broke for calc
   channels and other configuration-bearing resources. (Q4)
7. **Component + Instance + Override is the proposed universal configuration-reuse
   primitive.** Subsumes cross-project Reference, fleet inheritance, schematic symbols,
   profiles, snapshots, and "copy from template" into one mechanism. Convergent across
   Figma, Unity, Ignition, React, Notion, Wonderware, Helm. Rationale: reuse without
   inheritance produces drift; inheritance without override produces brittleness; this
   triplet is the minimum sufficient model. User accepts complexity and scope expansion
   in service of a unified long-term architecture. (Q3, Q4)

---

## Sequencing recommendation

1. **#1 Vocabulary** (blocking, fast)
2. **#0 Use cases** (grounds the rest)
3. **#2, #3, #5** (the core concept work)
4. **#4** falls out of #2
5. **#6, #7, #8** once the model is locked
