# Workspace Reorganization - UX Research & Design

## 1. The Problem

One of the key remaining pieces of friction in Synnax is the lack of a clear strategy
for organizing system configurations.

### What Exists Today

1. Channels: The core of Synnax, used to stream and store samples from a logical
   telemetry source.
2. Workspaces: A collection of visualizations and a layout in the console.
3. Racks: A remote Synnax data acquisition process that manages a set of Devices and
   Tasks.
4. Tasks: A set of instructions for acquiring data, outputting commands, and running
   sequences (such as an arc task).
5. Devices: Physical hardware devices that can be connected to Synnax.
6. Users: Synnax users with roles and policies.
7. Roles: User roles with permissions.
8. Policies: User policies that define access control.
9. Arcs: Compiled automation sequences.
10. Line Plots: Visualizations of channel data over time.
11. Tables: Tabular representations of channel data.
12. Schematics: Diagrams of hardware and software systems.
13. Schematic Symbols: Customer, user uploaded Icons and shapes used in schematics.
14. Logs: Visualization Records of events and errors.
15. Ranges: Time ranges that organize historical data.
16. Clusters: A cluster of Synnax cores acting as a single logical data space.
17. Nodes: A single Synnax core deployed as part of a cluster.
18. Views: A predefined query for a set of data structures (currently used for ranges
    and statuses)
19. Statuses: Data structures representing general purpose system statuses.
20. Labels: Used to categorize data structures.
21. Groups: Collections of data structures.

### Specific Pain Points

- Workspaces only control layout of a single console, and don't store references to
  configurations for tasks, calculated channels, arcs, etc.
- Workspaces only control layout for a single console, what happens if I want to version
  for an entire launch control system with 10 consoles?
- Right now workspaces have this weird liminal auto-save feature that makes it when I
  create a visualization just for looking at a channel temporarily it creates a new line
  plot with the same name as all of the other line plots. I kind of end up with this
  line plot soup.
- There is no way to version control configurations, review, and release changes through
  a tool like Git.
- The fact that you aren't required to be in a workspace makes Synnax good for doing
  scratch things, but it also makes users wonder "why wasn't my layout or schematic
  saved".
- If you go to the workspaces toolbar while not in a workspace, you open the
  visualization inside that workspace it will load the visualization and let you edit
  it, but it won't actually save the visualization. Makes our users wonder: why did I
  lose all of my changes?
- Some visualizations (plots, logs, schematics, workspaces) have auto save on their
  forms while other ones require a manual save.
- Users complain a lack of 'spatial awareness' i.e. what configuration context they are
  in, how to find things, etc.

---

## 2. Reference Research

### What the Best Tools Teach Us

**Figma** - Auto-save everything, always. No save button. The soup problem is solved not
by saving less, but by giving things a clear home. Drafts (personal scratch space) vs.
Projects (shared, organized). Files have stable identity independent of where they live.
You can reorganize without breaking anything. Named versions (Cmd+S) as bookmarks in
auto-save history.

**Vercel** - Environments are different configurations of the same thing. Immutable
snapshots make rollback trivial. Preview before promote. Git-centric workflow that
developers already know.

**VS Code** - A "workspace" is a collection of things + settings that scope your
context. Three-level settings (global, workspace, folder) with "more specific wins."
Progressive disclosure: simple by default, powerful when needed.

**NASA/Mission Control (Open MCT)** - Organization by operational role/position, not by
person or hardware topology. Phase-based configurations (the same system looks different
during setup vs. operations vs. analysis). Shared baseline + personal customization.
Telemetry dictionary fully decoupled from display configuration.

**TIA Portal / Rockwell** - Template-based visualization bound to data types, not
individual channels. A "Motor Faceplate" binds to a Motor data structure. Add a motor,
drop a faceplate, done. Position/Area-of-Responsibility-based multi-operator assignment.

**Linear** - Team-first, projects cross-cut teams. Saved views as flexible lenses on the
same data rather than rigid hierarchy. Opinionated defaults prevent chaos.

**Terraform** - "Plan" before "apply" for scary changes. Workspaces as named instances
of the same configuration with different state.

---

## 3. Design Questions

These are the questions we need to answer. Each one has an initial take followed by
space for discussion. The goal is to fill these in together until we have a coherent UX
story.

### Q1: What is a workspace?

**Answer**: A workspace is a long-lived, named, saved, shared collection of
configuration and visualizations. Most commonly it represents a physical system or
facility ("Test Cell A", "Fridge #7"), but it can also be an analytical context
("Q1 Hotfire Comparison") with no hardware config at all.

A workspace evolves over time rather than being replaced. When Honeywell switches Test
Cell A from testing Turbine X to Turbine Y, they don't create a new workspace. They
modify the existing one: swap some channels, update schematics, reconfigure tasks, maybe
bring in a Dewesoft vibe cart. The previous configuration is preserved as a named
snapshot in the workspace's history.

**Mental model**: A workspace is like a Git repository. It has a continuous history with
named snapshots (tagged commits). Campaigns/test series are chapters in the workspace's
timeline, not separate workspaces.

**Examples**:
- Orbex: "Denmark Hotfire Stand" is one workspace. "Scotland Structures Facility" is
  another. Vehicle versions are snapshots within the hotfire workspace's history.
- Rigetti: Each fridge is its own workspace (Fridge #1, Fridge #2, ... Fridge #20),
  stamped from a common template workspace.
- Honeywell: Each test cell is a workspace. Switching test articles = modifying the
  workspace config (maybe 15% changes) and snapshotting the previous state.
- Data review: "Q1 Hotfire Analysis" is a workspace containing comparison plots and
  tables that reference ranges from across the cluster. No hardware config.

**Key properties**:
1. Long-lived, usually represents a physical system but can also be an analytical context
2. Contains everything relevant to its purpose: hardware config, visualizations, operator
   views, arc programs (operational) or just visualizations and data queries (analytical)
3. Has version history with named snapshots (permanent, immutable records)
4. Can be created from another workspace: either as an independent copy (like GitHub
   template repos) or as a linked instance with live inheritance (like GitHub forks).
   These are creation modes, not different types of workspace.
5. Changes accumulate over time rather than forking into separate workspaces
6. Profiles within a workspace support switching between configurations (e.g., different
   generator models on the same test cell) without linear snapshot rollback

**The system has two top-level concepts, not more**:
- **Workspace**: the shared system/analysis configuration (lives on server)
- **Console**: your personal window into a workspace (layout, position, scratch)

Fleet management (templates, inheritance) is a relationship between workspaces, not a
third concept. Like GitHub forks: a fork is just a repo with a link to an upstream repo.



---

### Q2: What is scratch?

**Answer (in progress)**: Scratch is the console's personal space. It lives in the
console, not in any workspace. Scratch tabs auto-save locally (survive restart) but
never touch the workspace or the server. The workspace resource tree stays clean because
scratch never enters it.

Scratch is Figma's "Drafts" equivalent: a personal area for exploration that doesn't
pollute shared spaces.

**Key design direction**: The default action for quick data exploration should create
scratch, not workspace resources. Where you initiate the action determines what you get:

- From channel list, command palette, keyboard shortcut → scratch (exploring)
- From within a workspace resource tree → workspace resource (building)

A scratch plot is lightweight: instant, no name required, doesn't appear in any resource
tree. Visual treatment communicates "this is temporary."

**Promote flow**: When scratch turns out to be useful, user can promote it to a
workspace via right-click → "Save to workspace" or drag into workspace tree.

**Open questions (to pick up next session)**:
- Is there a case where defaulting to scratch is annoying and the user genuinely wants
  a permanent resource from the start?
- Should scratch tabs have a lifespan? (Auto-clean after N days? Or persist until
  manually closed?)
- What's the visual treatment? Subtle badge? Different background? No title?
- Can you have scratch tabs open alongside workspace tabs in the same console?
  (Probably yes.)

---

### Q3: How do we solve the visualization soup problem?

**Answer**: The soup problem comes from both creation paths (right-click "plot this
channel" AND "new line plot" from menu) creating permanent named workspace resources
when the user's intent is usually temporary.

The fix: both paths default to scratch tabs (see Q2). Workspace resources are only
created when explicitly initiated from within a workspace context. This means:
- "Plot this channel" → scratch tab (exploring)
- "New Line Plot" from menu/palette → scratch tab (exploring)
- Right-click in workspace tree → "Add Line Plot" → workspace resource (building)
- Promote from scratch → workspace resource (keeping)

The soup disappears because throwaway visualizations never enter the workspace.

**Open questions (to pick up next session)**:
- Should workspace visualizations require naming on creation? (Probably yes, to
  encourage intentionality.)
- What about the common flow where you open 5 scratch plots, then want to keep 2 of
  them? Batch promote?

---

### Q4: Should we auto-save?

**Answer**: Yes. Always. Everywhere. No save button.

- Scratch: auto-saves locally (console-level persistence, survives restart)
- Workspace: auto-saves to server (shared, persistent, available to all consoles)

Named snapshots (Figma's Cmd+S model) provide the safety net for workspace content. If
you break something, restore a named checkpoint.

**Open questions (to pick up next session)**:
- Visible save indicator? (Figma: cloud icon showing sync status)
- Conflict handling when two people edit the same workspace simultaneously?
- Undo history per-visualization beyond auto-save?



---

### Q5: How do multiple consoles work within a workspace?

**Answer**: A workspace defines one or more "views" (name TBD, could also be "station"
or "position"). Each view is a named console layout: a configuration of tabs, windows,
and visualizations. This is what the current "workspace" concept actually is today. It
gets demoted from top-level concept to a thing inside a workspace.

Example: "Hotfire Test Stand" workspace has:
- "Propulsion Operator" view: P&ID schematic, propulsion plot, command table
- "Electrical Operator" view: Electrical schematic, power monitoring, logs
- "Test Director" view: Overview dashboard, all-channel table, sequence status

**Daily experience**: The console remembers your last workspace + view. You open Synnax,
you're right back where you were. No daily selection prompt. Switching views is possible
but rare. Operators typically sit at the same position every day.

**First-time setup**: Pick a workspace, pick a view. After that, it just opens.

**Open questions**:
- Can two people be in the same view simultaneously? (Not constrained for now.)
- Naming: "view" is clean but overloaded. "Station" maps to hardware. "Position" maps
  to mission control. Need to pick one.



---

### Q6: What does a workspace own vs. reference?

**Answer (partial)**:

A workspace **owns** (lives and dies with the workspace):
- Visualizations (line plots, schematics, tables, logs)
- Console views / positions
- Arc programs (automation sequences)
- Profiles (e.g., Gen A config, Gen B config)

A workspace **references** (exists independently at cluster level):
- Channels (global, shared across workspaces)
- Devices (physical hardware, global)
- Ranges (global, tagged/labeled by system. Any workspace can reference any range.
  This is critical for data review: an analysis workspace pulls ranges from across
  the cluster without needing cross-workspace references.)

**The global-but-scoped model**: All resources live in the global namespace. A workspace
is an ontology node that has relationships to its resources. Being "in a workspace"
means the UI filters the global graph by those relationships. No special "owned vs
referenced" distinction in storage.

**Unsolved tension: workspace-scoped configuration.** Some resource state needs to vary
per-workspace or per-profile. Example: a derived channel's calibration curve might be
different in Profile "Gen A" vs "Gen B" on the same workspace, or different across two
workspaces using the same physical channel. The resource identity and stored data are
global, but the configuration is contextual.

**Unsolved tension: snapshot depth.** A snapshot needs to capture actual resource state,
not just a list of relationship pointers (otherwise restoring a snapshot gets you old
relationships pointing at resources that have since changed). This means either
deep-copying resource state into the snapshot, or versioning every resource individually
and recording which version the snapshot references. TBD.

**Open questions**:
- What is the exact mechanism for workspace-scoped resource configuration?
- Should tasks be workspace-scoped or global? (A task config is tied to a specific
  setup, but the same task might appear in multiple workspaces.)
- What about schematic symbols? Per-workspace or shared library?
- How do calculated/derived channels work across profiles?



---

### Q7: How does version history work?

**Answer**: Continuous auto-save with named snapshots. Snapshots serve double duty: both
as rollback points AND as permanent configuration records tied to test history.

- Every change auto-saves (no user action needed).
- Users can explicitly create a named snapshot at any time ("Pre-hotfire config",
  "Turbine X - Final", "Added vibe cart for Campaign 3").
- Snapshots are immutable and permanent. They capture the complete workspace state:
  all visualizations, task configs, views, arc programs.
- Restoring a snapshot creates a new state (non-destructive).
- Snapshots can be linked to ranges (test data), creating a permanent record:
  "This is the exact configuration that produced Hotfire #47."

This is critical for traceability. When an engineer asks "what was the system configured
as when we ran that test last March?", the answer is a snapshot.

For teams that want Git-based workflows, the export format should be human-readable
and diffable.

**Open questions**:
- What granularity? Whole workspace snapshots, or per-visualization history?
- How long do auto-save checkpoints last vs. named snapshots? (Figma: auto-checkpoints
  expire after 30 days, named snapshots live forever.)
- Is comparing two snapshots important? (Probably yes for config review.)



---

### Q8: What does spatial awareness look like?

**Initial take**: The user should always be able to answer three questions by glancing
at the UI:

1. **Where am I?** - Which workspace, which position/layout.
2. **What's here?** - What resources belong to this workspace.
3. **What's the state?** - Is everything saved? Am I in scratch? Are others editing?

Concrete ideas:
- Persistent breadcrumb: `Hotfire Test Stand > Operator 1 > Propulsion P&ID`
- Workspace color/theme (subtle tint so different workspaces feel visually distinct)
- Clear "scratch mode" indicator (like Figma's drafts having a different background)
- Resource tree scoped to current workspace by default, with option to browse all

**Open questions**:
- How prominent should the workspace identity be? Always visible? Minimal until needed?
- Should switching workspaces feel like switching apps, or switching tabs?
- How do we handle the resource tree? Workspace-scoped vs. global?

**Discussion**:



---

### Q9: How does a user set up a new system from zero?

**Initial take** (UX walkthrough):

1. Alice connects her console to a Synnax cluster. She's in scratch mode.
2. She configures her hardware: plugs in devices, creates tasks. These exist at the
   cluster level, not in any workspace yet.
3. She creates a workspace: "Engine Test Stand A". Synnax creates it with a default
   empty layout.
4. She builds her first visualization (a P&ID schematic) inside the workspace. It
   auto-saves as she works.
5. She adds a second layout position ("Test Director") and builds a different set of
   views for it.
6. She creates a named snapshot: "Initial Setup v1".
7. Bob opens a second console, connects to the same cluster, opens the "Engine Test
   Stand A" workspace, and picks the "Test Director" position.
8. Both operators now see their respective layouts, sharing the same underlying data.

**Open questions**:
- Step 2 feels heavy. Should workspace creation happen earlier? Should hardware setup
  happen inside a workspace?
- Is there a "getting started" wizard, or is it all progressive discovery?

**Discussion**:



---

### Q10: How does daily operation work?

**Initial take** (UX walkthrough):

1. Operator opens console. Sees list of available workspaces (or their recent ones).
2. Selects "Engine Test Stand A" and their position ("Operator 1").
3. Their layout loads instantly with all visualizations, live data streaming.
4. During the test, they might open a quick scratch plot to investigate an anomaly.
   This doesn't pollute the workspace.
5. If they find something worth keeping, they promote the scratch plot to the workspace.
6. After the test, the test director creates a named snapshot: "Hotfire #47 - nominal".
7. The historical data is there. The configuration that produced it is snapshotted.

**Discussion**:



---

### Q11: What about operational phases?

Some systems look different during different phases: setup, pre-test, active test,
post-test, maintenance. Mission control systems handle this with phase-based display
sets.

**Open questions**:
- Do we need phases? Or are they just different workspaces?
- If phases, are they predefined or user-configurable?
- Does changing phase change which visualizations are visible, which channels are
  active, which limits apply?

**Discussion**:



---

### Q12: Factory floor / replicated setups?

**Answer**: Workspaces can be created from other workspaces, either as independent copies
or as linked instances with live inheritance. This is a creation mode, not a separate
concept.

For Rigetti's fleet of 20 mostly-identical fridges: one workspace is the reference
config. The other 20 are linked instances that inherit from it. Changes to the reference
propagate automatically. Per-fridge overrides (extra sensor, different calibration) are
marked as local and protected from propagation. The research points to Unity's prefab
variant model as the gold standard here: per-property override granularity with clear
visual indicators (inherited vs. overridden vs. locally added).

For the rare fridge that needs to fully diverge: "detach" severs the link, making it
an independent workspace. This is one-way, like Figma's detach or GitHub's relationship
between a fork and its upstream.

For very little divergence (which is Rigetti's actual case): changes propagate
automatically with no per-instance review step. Local overrides are explicit and sticky.
Everything else flows from the reference.

For Honeywell's production cell testing different generator models: profiles within
a single workspace, not separate linked workspaces. The cell is the workspace. Gen A
and Gen B configs are profiles. Switching profiles swaps the model-specific overrides
while keeping the cell infrastructure stable.

**Key research finding**: Ignition (Inductive Automation) has the most relevant prior
art. Their project inheritance model uses resource-level inheritance with clear visual
indicators for inherited vs. overridden vs. locally added resources. Live propagation,
"local wins" conflict resolution, granular revert-to-inherited per resource.



---

## 4. Target Scenarios

Once we agree on answers to the above, we write concrete end-to-end scenarios:

- [ ] New test stand setup (from zero to first test)
- [ ] Daily test operations (3 operators, routine test)
- [ ] Configuration change mid-campaign (add a sensor, update a schematic)
- [ ] Handoff between teams (day shift to night shift)
- [ ] "I just want to look at some data" (scratch use case)
- [ ] Factory floor with 20 stations doing the same thing
- [ ] Exporting a workspace to version control
- [ ] Rolling back after a bad configuration change

---

## 5. Principles (Draft)

These are the design principles we're converging on. We'll refine as we answer the
questions above.

1. **Two concepts, not more.** Workspace (shared system/analysis config) and Console
   (personal window into a workspace). Fleet management, templates, and profiles are
   features of workspaces, not separate concepts.
2. **Everything auto-saves. Always.** No save button, no "did I lose my work?"
3. **Scratch is first-class and default.** Quick exploration creates scratch (console-
   local). Workspace resources require intentional creation. This kills the soup.
4. **A workspace is a named collection of related configuration.** Usually a physical
   system, but can also be an analytical context. Not just a console layout.
5. **All resources are global, scoped by relationships.** The ontology graph defines
   what belongs to a workspace. "Being in a workspace" = filtering the global graph.
6. **You always know where you are.** Spatial awareness is a core UX requirement.
7. **Named snapshots for version history.** Immutable, permanent records of workspace
   state. Linked to ranges for test traceability.
8. **Opinionated defaults, flexible overrides.** The happy path should require zero
   configuration. Power users can customize.
9. **Progressive disclosure for fleet features.** Workspace inheritance, profiles, and
   template propagation are invisible until you need them.

---

## 6. Status & Open Questions (End of Session 2025-03-29)

### What's decided:
- Two top-level concepts: Workspace + Console
- Workspace = long-lived config container (operational or analytical)
- Console = personal window with layout, position, scratch
- Workspace history with named immutable snapshots tied to test ranges
- Fleet/template = workspace-to-workspace relationships (linked copy with inheritance),
  not a third concept. Research supports Ignition/Unity prefab model for inheritance UX.
- Profiles within workspace for temporal config switching (Gen A / Gen B)
- Ranges are global (any workspace can reference any range)
- Auto-save everywhere, scratch defaults for quick exploration
- Console remembers last workspace + view, sticky positions

### What's unsolved:
- **Workspace-scoped resource configuration**: How do derived channels, calibrations,
  and calculations vary per-workspace or per-profile when resources are global?
- **Snapshot depth**: What exactly does a snapshot capture? Deep copy of state vs.
  versioned resource references?
- **Tasks owned or referenced?** Strong case for workspace-scoped but needs more thought
- **Data review UX**: Analysis workspaces work conceptually but the actual UX for
  comparing 47 tests hasn't been designed
- **Scratch details**: Visual treatment, lifespan, promote flow, edge cases
- **Spatial awareness**: Breadcrumb design, resource tree filtering, workspace identity
- **View/position naming**: "view" is overloaded, "station" and "position" are options
- **Operational phases**: Separate concept or just profiles?
- **Conflict resolution**: Two people editing same workspace simultaneously
