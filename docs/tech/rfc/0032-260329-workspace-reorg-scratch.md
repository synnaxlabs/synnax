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

## 2. Current Console UI Map

### Top Bar

Left to right:

- **Window controls** (macOS traffic lights)
- **Logo**
- **Workspace Selector** - Dropdown showing current workspace name or "No workspace."
  Click opens a dialog with search, a list of workspaces, a "Clear" button (go to
  no-workspace mode), and a "New" button.
- **Command Palette** (center) - Ctrl+P for search, Ctrl+Shift+P for commands. Entry
  point for creating visualizations, opening docs, etc.
- **Version badge, User badge, Cluster connection badge, Docs button** (right side)

### Left Sidebar Rail

A vertical strip of icons. Single-press previews a drawer, double-press opens it fully.
Each icon opens a drawer panel to the right of the rail:

| Key            | Trigger | Drawer Content                                                                                                                    |
| -------------- | ------- | --------------------------------------------------------------------------------------------------------------------------------- |
| **Channels**   | C       | Ontology tree of all channels in groups. Create channels and calculated channels. Drag channels to visualizations.                |
| **Ranges**     | R       | Favorited time ranges. Drop target for ranges. Shows stage icon, time duration, labels.                                           |
| **Workspaces** | W       | Ontology tree of workspace groups. Create workspaces. Browsing only, switching via top-bar selector.                              |
| **Devices**    | D       | Ontology tree of hardware devices (LabJack, NI, Modbus, OPC, HTTP, EtherCAT).                                                     |
| **Tasks**      | T       | Flat list of all DAQ tasks with status, start/stop, type labels. Edit config, enable/disable data saving, rename, delete, export. |
| **Users**      | U       | Ontology tree of users. Create new users (admin only).                                                                            |
| **Arc**        | A       | List of Arc programs with status, start/stop. Create, edit, rename, delete.                                                       |
| **Status**     | S       | Status entries with severity, messages, timestamps. Filter by labels.                                                             |

### Bottom Drawer Rail

| Key               | Trigger | Drawer Content                                                                                                                                                                                                                                                    |
| ----------------- | ------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Visualization** | V       | Context-sensitive toolbar for the active visualization. Different controls by type: Line Plot (data, lines, axes, properties, rules), Table (cell properties), Schematic (tools), Log (tools), Arc Editor (tools). "No visualization selected" if nothing active. |

### Main Content Area (Mosaic)

The central tiled workspace. A tree of resizable split panes, each containing tabs.

- **Tabs** represent open visualizations (line plots, schematics, tables, logs, arc
  editors)
- **"+" button** or **Ctrl+T** opens a visualization type selector (Line Plot,
  Schematic, Log, Table)
- **Empty state** shows Synnax logo watermark and "New Component" hint
- Supports drag-and-drop: drag channels onto plots, drag ontology items into mosaic,
  rearrange tabs between panes

### Visualization Creation Paths (Current)

All of these create permanent workspace resources:

1. **Command palette**: "Create a line plot" etc. Creates with default name "Line Plot"
2. **Ctrl+T / "+" button**: Opens type selector grid, click a type to create
3. **Right-click channel -> plot**: Creates plot with that channel
4. **Double-click in workspace tree**: Loads existing visualization from server
5. **Drag from tree to mosaic**: Same as double-click but specifies drop location
6. **Right-click workspace in tree -> Create visualization**: Creates in that workspace
   (the only path that's workspace-context-aware)

### The Soup Problem in This Map

Paths 1-3 all create resources named "Line Plot", "Line Plot", "Line Plot" that
immediately become workspace resources (if a workspace is active) or local-only state
(if no workspace). There's no distinction between "I'm exploring" and "I'm building."
The soup comes from paths 1-3 treating every creation as permanent.

---

## 3. Reference Research (External)

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

## 4. Design Questions

These are the questions we need to answer. Each one has an initial take followed by
space for discussion. The goal is to fill these in together until we have a coherent UX
story.

### Q1: What is a project? What is a workspace?

**Naming decision**: We use two terms:

- **Project** = the big container (new concept). Represents a physical system, facility,
  or analytical context.
- **Workspace** = a console layout within a project (existing concept, mostly
  unchanged). What users currently call "workspaces" stays as "workspaces." They just
  get a parent.

**Project**: A project is a long-lived, named, saved, shared collection of configuration
and visualizations. Most commonly it represents a physical system or facility ("Test
Cell A", "Fridge #7"), but it can also be an analytical context ("Q1 Hotfire
Comparison") with no hardware config at all.

A project evolves over time rather than being replaced. When Honeywell switches Test
Cell A from testing Turbine X to Turbine Y, they don't create a new project. They modify
the existing one: swap some channels, update schematics, reconfigure tasks, maybe bring
in a Dewesoft vibe cart. The previous configuration is preserved as a named snapshot in
the project's history.

**Workspace**: A workspace is a named console layout within a project. It defines what
tabs, windows, and visualizations an operator sees. A project can have multiple
workspaces for different operator roles.

Example: Project "Hotfire Test Stand" has:

- Workspace "Propulsion Operator": P&ID schematic, propulsion plot, command table
- Workspace "Electrical Operator": Electrical schematic, power monitoring, logs
- Workspace "Test Director": Overview dashboard, all-channel table, sequence status

**Mental model**: A project is like a Git repository. A workspace is like a saved window
layout in your IDE for that repo.

**Examples**:

- Orbex: "Denmark Hotfire Stand" is one project. "Scotland Structures Facility" is
  another. Vehicle versions are snapshots within the project's history.
- Rigetti: Each fridge is its own project (Fridge #1, Fridge #2, ... Fridge #20),
  stamped from a common template project.
- Honeywell: Each test cell is a project. Switching test articles = modifying the
  project config (maybe 15% changes) and snapshotting the previous state.
- Data review: "Q1 Hotfire Analysis" is a project containing comparison plots and tables
  that reference ranges from across the cluster. No hardware config.

**Key properties of a project**:

1. Long-lived, usually represents a physical system but can also be an analytical
   context
2. Contains everything relevant to its purpose: hardware config, visualizations,
   workspaces, arc programs (operational) or just visualizations and data queries
   (analytical)
3. Has version history with named snapshots (permanent, immutable records)
4. Can be created from another project: either as an independent copy (like GitHub
   template repos) or as a linked instance with live inheritance (like GitHub forks).
   These are creation modes, not different types of project.
5. Changes accumulate over time rather than forking into separate projects
6. Profiles within a project support switching between configurations (e.g., different
   generator models on the same test cell) without linear snapshot rollback

**The system has two top-level concepts, not more**:

- **Project**: the shared system/analysis configuration (lives on server)
- **Workspace**: a named console layout within a project (what users already know)

(The console application itself remembers your last project + workspace. Scratch lives
in the console, outside any project.)

Fleet management (templates, inheritance) is a relationship between projects, not a
third concept. Like GitHub forks: a fork is just a repo with a link to an upstream repo.

---

### Q2: Is there a scratch concept?

**Answer**: No. There is no scratch, no orphan, no special mode.

The user is **always** in a project + workspace. A project always has at least one
workspace. On first launch, Synnax auto-creates a default project ("My Project") with a
default workspace ("Default"). The user can rename these, add more, but they can never
be in an ambiguous "no project" state.

Every visualization is always a child of the current workspace. "Plot this channel" from
anywhere creates a visualization in the current workspace. If it turns out to be
throwaway, the user just closes and deletes the tab. No orphan lifecycle, no dual
persistence, no promote flow.

**Defaults:**

- First launch: auto-create "My Project" with workspace "Default"
- Creating a new project: comes with one workspace "Default"
- Can't delete the last workspace in a project

**The soup problem is solved by:**

1. Always having a home for everything (no lost work)
2. Users can close/delete throwaway visualizations from their workspace
3. The default personal project absorbs casual exploration without polluting shared
   operational projects

---

### Q3: How do we solve the visualization soup problem?

**Answer**: The user is always in a project + workspace. Every visualization is always a
child of the current workspace. There are no orphans, no special scratch mode.

The soup problem was caused by visualizations being created with no clear home. Now
every creation path puts the visualization in the current workspace.

The remaining soup risk is accumulation within a workspace: user creates 50 quick plots
over a week and never cleans up. This is a UX polish problem (better delete/cleanup
flows, maybe a "recently created" filter) rather than an architectural problem.

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

### Q5: How do multiple consoles work within a project?

**Answer**: A project defines one or more workspaces. Each workspace is a named console
layout: a configuration of tabs, windows, and visualizations. This IS the current
"workspace" concept, unchanged. It just lives inside a project now.

Example: Project "Hotfire Test Stand" has:

- Workspace "Propulsion Operator": P&ID schematic, propulsion plot, command table
- Workspace "Electrical Operator": Electrical schematic, power monitoring, logs
- Workspace "Test Director": Overview dashboard, all-channel table, sequence status

**Daily experience**: The console remembers your last project + workspace. You open
Synnax, you're right back where you were. No daily selection prompt. Switching
workspaces is possible but rare. Operators typically sit at the same position every day.

**First-time setup**: Pick a project, pick a workspace. After that, it just opens.

**Open questions**:

- Can two people be in the same workspace simultaneously? (Not constrained for now.)

---

### Q6: What does a project own vs. reference? What about workspaces?

**Definitions**:

- **Project** = the configuration of a hardware system. Tasks, channels, devices, arcs,
  calibrations. Shared, objective, represents the physical reality.
- **Workspace** = a user's preferred view into that system. A saved layout that
  references project-level resources. Fundamentally about presentation, not config.

**Project contains** (scoped via ontology relationships):

- Tasks (DAQ configurations, project-scoped)
- Arc programs (automation sequences)
- Profiles (e.g., Gen A config, Gen B config)
- Workspaces (console layouts)
- Visualizations (can be children of the project or of specific workspaces)

**Global resources** (exist independently at cluster level, referenced by projects):

- Channels (shared across projects)
- Devices (physical hardware)
- Ranges (any project can reference any range, critical for data review)
- Schematic symbols (shared library)
- Users / roles

**Workspace stores**: A saved layout (tab arrangement, split positions, window config)
that references visualizations. The workspace is lightweight, not a heavy container.

**Visualization ownership**: A visualization is a child of a specific workspace in the
ontology (its "home" in the tree). But other workspaces within the same project can
reference and display it in their layout. The P&ID schematic that took 40 hours to build
isn't locked inside one workspace. It's reusable across workspaces within the project.

Two kinds of workspace-to-visualization relationships:

- **Owns** (parent-of): created here, shows in this workspace's subtree
- **References** (displayed-in): borrowed from another workspace or project level,
  appears in layout but lives elsewhere

**The global-but-scoped model**: All resources live in the global namespace. A project
is an ontology node with relationships to its resources. Being "in a project" means the
UI filters the global graph by those relationships.

**Unsolved tension: project-scoped configuration.** Some resource state needs to vary
per-project or per-profile. Example: a derived channel's calibration curve might be
different in Profile "Gen A" vs "Gen B" on the same project, or different across two
projects using the same physical channel. The resource identity and stored data are
global, but the configuration is contextual.

**Unsolved tension: snapshot depth.** A snapshot needs to capture actual resource state,
not just a list of relationship pointers (otherwise restoring a snapshot gets you old
relationships pointing at resources that have since changed). This means either
deep-copying resource state into the snapshot, or versioning every resource individually
and recording which version the snapshot references. TBD.

**Open questions**:

- What is the exact mechanism for project-scoped resource configuration?
- How do calculated/derived channels work across profiles?

---

### Q7: How does version history work?

**Answer**: Continuous auto-save with named snapshots. Snapshots serve double duty: both
as rollback points AND as permanent configuration records tied to test history.

- Every change auto-saves (no user action needed).
- Users can explicitly create a named snapshot at any time ("Pre-hotfire config",
  "Turbine X - Final", "Added vibe cart for Campaign 3").
- Snapshots are immutable and permanent. They capture the complete workspace state: all
  visualizations, task configs, views, arc programs.
- Restoring a snapshot creates a new state (non-destructive).
- Snapshots can be linked to ranges (test data), creating a permanent record: "This is
  the exact configuration that produced Hotfire #47."

This is critical for traceability. When an engineer asks "what was the system configured
as when we ran that test last March?", the answer is a snapshot.

For teams that want Git-based workflows, the export format should be human-readable and
diffable.

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

**Answer** (UX walkthrough, updated with project/workspace naming):

1. Operator opens console. It remembers last project + workspace. Loads instantly.
2. They're in Project "Engine Test Stand A", Workspace "Propulsion Operator".
3. Live data streaming, all visualizations active.
4. During the test, they might open a quick scratch plot to investigate an anomaly. This
   doesn't pollute the project.
5. Test completes. Arc program captures a range. The workspace's time context shifts
   from "live" to the test range for quick review. Same layout, same plots, now showing
   historical data from the test.
6. Operator does quick review: checks key parameters, marks pass/fail, adds notes. These
   annotations attach to the range.
7. Operator or Arc shifts time context back to "live" for the next test.
8. At end of campaign, test director creates a named snapshot of the project config.
9. For deep analysis across multiple tests, someone creates a separate analysis project
   referencing the ranges from this campaign.

---

### Q11: What about operational phases and data review modes?

**Answer**: There is no separate "data review mode" or "operational phase" concept. The
workspace has a **time context** that can be set to "live" or to a specific range.

**Three usage patterns, one workspace:**

1. **Live operations**: Time context = live. Data streaming in real-time. Task controls
   active. Visualizations show current state.
2. **Quick review**: Time context = a range. Same workspace, same layout. All
   visualizations shift to showing historical data for that time window. UI adapts
   subtly (time scrubber appears, task controls dim). Used between tests on a production
   cell.
3. **Deep analysis**: Separate analysis project. Purpose-built comparison visualizations
   referencing ranges from across the cluster. Different session, different intent.

**Guided workflows**: The production cell workflow (run test, capture range, review,
pass/fail, next test) is orchestrated by Arc programs, not by workspace/project
features. Arc controls the time context transitions. The project/workspace just needs to
support smooth transitions between live and historical time contexts.

**Phases (pre-test, active, post-test) are not a workspace concept.** They're either
profiles within a project (if the visualizations change per phase) or Arc workflow
states (if it's about what the operator should be doing). This is Arc's domain.

---

### Q12: Factory floor / replicated setups?

**Answer**: Projects can be created from other projects, either as independent copies or
as linked instances with live inheritance. This is a creation mode, not a separate
concept.

For Rigetti's fleet of 20 mostly-identical fridges: one project is the reference config.
The other 20 are linked instances that inherit from it. Changes to the reference
propagate automatically. Per-fridge overrides (extra sensor, different calibration) are
marked as local and protected from propagation. The research points to Unity's prefab
variant model as the gold standard here: per-property override granularity with clear
visual indicators (inherited vs. overridden vs. locally added).

For the rare fridge that needs to fully diverge: "detach" severs the link, making it an
independent project. This is one-way, like Figma's detach or GitHub's relationship
between a fork and its upstream.

For very little divergence (which is Rigetti's actual case): changes propagate
automatically with no per-instance review step. Local overrides are explicit and sticky.
Everything else flows from the reference.

For Honeywell's production cell testing different generator models: profiles within a
single project, not separate linked projects. The cell is the project. Gen A and Gen B
configs are profiles. Switching profiles swaps the model-specific overrides while
keeping the cell infrastructure stable.

**Key research finding**: Ignition (Inductive Automation) has the most relevant prior
art. Their project inheritance model uses resource-level inheritance with clear visual
indicators for inherited vs. overridden vs. locally added resources. Live propagation,
"local wins" conflict resolution, granular revert-to-inherited per resource.

---

### Q13: How does data review / multi-test comparison work?

**Answer**: No separate "data review" concept. Two patterns:

**Quick review (same project, same workspace):** The workspace has a time context.
During live operations, time context = live. When a test completes (either manually or
via Arc), time context shifts to the captured range. The same plots and schematics now
show historical data. The operator reviews, annotates, marks pass/fail. Then shifts back
to live for the next test. No mode switch, no project switch. Just a time context
change.

**Deep analysis (separate analysis project):** An engineer creates a project like "Q1
Hotfire Analysis." It contains comparison visualizations that reference ranges from
across the cluster. Streamlined creation flow: select multiple ranges, right-click,
"Compare in new analysis." Synnax creates a project pre-populated with those ranges and
sensible default comparison views.

The analysis project is a regular project. No special type. It just happens to have no
tasks or hardware config.

**Open questions**:

- What does the streamlined "compare these ranges" creation flow actually look like?
- What are the default comparison views? (Overlaid plots for shared channels? Summary
  table of key metrics?)
- How does range selection work across projects? (Global range browser? Search?)

---

## 5. Target Scenarios

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

## 6. Principles (Draft)

These are the design principles we're converging on. We'll refine as we answer the
questions above.

1. **Two concepts: Project + Workspace.** Project = hardware system config. Workspace =
   user's preferred view into a project (existing concept, kept as-is). Fleet
   management, templates, and profiles are features of projects, not separate concepts.
2. **Everything auto-saves. Always.** No save button, no "did I lose my work?"
3. **Always in a project + workspace.** No ambiguous "no project" state. Default
   personal project created on first launch. Every visualization has a parent.
4. **A project is the configuration of a hardware system.** Tasks, arcs, device refs,
   calibrations. Can also be an analytical context (no hardware config).
5. **All resources are global, scoped by relationships.** The ontology graph defines
   what belongs to a project. "Being in a project" = filtering the global graph.
6. **You always know where you are.** Spatial awareness is a core UX requirement.
7. **Named snapshots for version history.** Immutable, permanent records of project
   state. Linked to ranges for test traceability.
8. **Opinionated defaults, flexible overrides.** The happy path should require zero
   configuration. Power users can customize.
9. **Progressive disclosure for fleet features.** Project inheritance, profiles, and
   template propagation are invisible until you need them.

---

## 7. Status & Open Questions

### What's decided:

- Two top-level concepts: Project + Workspace
- Project = long-lived config container (operational or analytical)
- Workspace = named console layout within a project (existing concept unchanged)
- Console app remembers last project + workspace, scratch lives outside any project
- Project history with named immutable snapshots tied to test ranges
- Fleet/template = project-to-project relationships (linked copy with inheritance), not
  a third concept. Research supports Ignition/Unity prefab model for inheritance UX.
- Profiles within project for temporal config switching (Gen A / Gen B)
- Tasks are project-scoped (not global)
- Schematic symbols are a shared library (global)
- Ranges are global (any project can reference any range)
- Auto-save everywhere, scratch defaults for quick exploration

### What's decided (session 2, 2025-03-30):

- Naming: **Project** (hardware system config) + **Workspace** (user's view, existing
  concept)
- Tasks are project-scoped, not global
- Schematic symbols are a shared library (global)
- No separate "data review mode." Workspace has a time context (live or range). Quick
  review = same workspace, time context shifts to a range. Deep analysis = separate
  analysis project.
- Operational phases are not a workspace concept. They're Arc workflow states or project
  profiles.
- Guided production workflows (test, review, pass/fail, next) are Arc's job.
- User is always in a project + workspace. No ambiguous empty state. Default personal
  project + workspace created on first launch.
- No orphan/scratch concept. Every visualization is a child of the current workspace.
  Soup solved by always having a home + delete on close for throwaway tabs.

### What's unsolved:

- **Project-scoped resource configuration**: How do derived channels, calibrations, and
  calculations vary per-project or per-profile when resources are global?
- **Snapshot depth**: What exactly does a snapshot capture? Deep copy of state vs.
  versioned resource references?
- **Data review creation flow**: The "compare these 47 ranges" streamlined entry point
- **Spatial awareness**: Breadcrumb design, resource tree filtering, project identity
- **Conflict resolution**: Two people editing same project simultaneously
