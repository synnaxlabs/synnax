# 80 Arc user modules

- **Author**: Nico Alba
- **Date**: 2026-10-19

Draft. Currently basically just a conglomoration of thoughts. NOT gospel. Will change
drastically to cut info.

## 0 Summary

An Arc is either an automation or a module. An automation deploys to a rack and runs as
a task. A module exports code that other Arcs import. This RFC covers everything around
that split that is not the Arc language change itself: the Arc type, the ontology
structure that gives modules a path, the Console explorer, and the relationship that
records an import.

## 1 Vocabulary

- **Automation**: An Arc with `type = automation`. It binds to a rack and is the parent
  of its task.
- **Module**: An Arc with `type = module`. It never has a rack or a task. The Core
  compiles it after each edit and stores the result as the Arc's status.
- **Arcs group**: The group below `root` that holds every Arc, like "Channels" holds
  every channel.
- **User group**: A group below "Arcs" that a user creates. It is a directory. Nested to
  any depth.
- **Path**: The group names from "Arcs" down to an Arc, joined by `.`. The path is the
  module's import name.

## 2 Ontology

![Arc ontology map](img/0080-arc-user-modules/ontology.svg)

### 2.0 Arcs group

The Arc service creates "Arcs" below `root` with `Group.CreateOrRetrieve`, the same call
"Channels", "Statuses", and "Schematic Symbols" use. Every new Arc is parented under it.
A migration parents every existing Arc under it once, the way the "Workspaces" to
"Projects" rename re-parented projects.

Nothing else moves. An automation stays the parent of its task, and the task writer
still parents the task under "Tasks". The compile status stays keyed on the Arc ontology
ID below "Statuses".

### 2.1 User groups as namespaces

A group below "Arcs" is a directory. Its name is part of every import path below it, so
it takes the identifier rule module names already follow: letters, digits, and
underscores. Names are unique among siblings. Groups elsewhere in the ontology keep free
text, so the rule lives in the Arc service, not the group service.

Resolving a path walks the existing `parent_of` edges down from "Arcs". No new edge is
needed for resolution.

### 2.2 The `imports` relationship

An import is recorded as an ontology relationship of a new type, `imports`, from the
importing Arc to the module. It sits beside `parent_of` and `labeled_by`. The edge is an
index and nothing more: it answers "who imports this module" without reading every Arc.

The edge has one consumer, change propagation:

- **Content hash**: An Arc's semantic hash folds in the hashes of the modules it
  imports, transitively. The task config carries that hash, so a module edit changes the
  importer's hash and its task drifts, which the Console shows as "Redeploy". A save
  that leaves the module's content equal leaves the hash equal and shows nothing. A
  deleted or cyclic module contributes an empty hash, so losing a module changes the
  importer too.
- **Which changed**: The task config also records what the running instance was built
  from, so the Redeploy tooltip can name the module. §2.5.
- **Fan-out**: A module write resyncs its importers, found by walking incoming `imports`
  edges through the ontology's `byTo` index. Without the edge, the Core resyncs every
  Arc, each a parse and a hash.

Complexity per module save, with N Arcs and I importers of the module, transitively:

- **Resync all**: O(N). One table scan to list every Arc, then N resyncs, each a parse
  and a hash. Every automation pays for every module edit, imported or not.
- **Imports edge**: O(I + D). One index probe per module level D, then I resyncs. An Arc
  that does not import the module never wakes.

Benchmark, `core/pkg/service/arc/importers_bench_test.go`, Go 1.27, one machine. N
automations, a chain of D modules each importing the one before, P importers at every
module. I counts the importers of the first module, transitively. Time per module save:

| N    | D   | P   | I   | Resync all | Imports edge |
| ---- | --- | --- | --- | ---------- | ------------ |
| 10   | 1   | 1   | 1   | 1.24 ms    | 0.07 ms      |
| 10   | 1   | 5   | 5   | 1.02 ms    | 0.44 ms      |
| 10   | 4   | 1   | 7   | 2.54 ms    | 1.64 ms      |
| 100  | 1   | 1   | 1   | 11.2 ms    | 0.09 ms      |
| 100  | 1   | 5   | 5   | 14.1 ms    | 0.42 ms      |
| 100  | 4   | 1   | 7   | 15.7 ms    | 1.39 ms      |
| 100  | 4   | 5   | 23  | 11.3 ms    | 2.39 ms      |
| 1000 | 1   | 1   | 1   | 110 ms     | 0.07 ms      |
| 1000 | 1   | 5   | 5   | 107 ms     | 0.32 ms      |
| 1000 | 4   | 1   | 7   | 109 ms     | 1.06 ms      |
| 1000 | 4   | 5   | 23  | 104 ms     | 2.09 ms      |

Resync all grows with N, about 0.1 ms per Arc. The edge grows with I and D only, about
0.07 ms per importer and 0.3 ms per level, and is flat in N.

Other uses, each one traversal instead of a parse of every Arc (IMPORTANT!):

- **Export bundle**: Exporting an automation walks its outgoing edges and bundles its
  modules with their group paths. Import restores both and rewrites the edges on create.
  Today an export is one Arc and its imports break on the other side.
- **Used by**: A module's context menu lists its importers through `Importers`, and a
  delete dialog says how many automations import it. A list, not a guard.
- **Move or rename**: Edges are keyed by ID, so they survive a move while the paths in
  the importers' text do not. The Core can rewrite the `import` lines of the importers
  it finds, or the Console can list what breaks.
- **Explorer**: An automation node lists the modules it imports through a forward
  traverser; a module node lists its importers.
- **Access**: A policy on an automation can imply read access on its modules by walking
  the edges, the same way `parent_of` is walked today.

Edge rules:

- **Written on save**: `Create` and `Dispatch` rewrite the Arc's outgoing `imports`
  edges from its import statements in the same transaction. The text stays the source of
  truth; the edge is derived.
- **Keyed by ID**: An edge points at the module Arc, not its path. Moving or renaming a
  module keeps its edges, so its importers still resync, and their next compile reports
  the broken path.
- **Unknown paths leave no edge**: A path that resolves to no module, STL modules
  included, is skipped. A new module therefore resyncs every Arc once, the one place the
  full scan remains, since an importer may already name it.
- **Cycles are the compiler's**: `DefineRelationships` refuses an edge that closes a
  cycle. The edge is skipped and the analyzer reports the cycle with its chain.
- **Delete**: The ontology drops a resource's edges with it, so a module delete collects
  its importers first and resyncs them after.
- **No delete guard**: A module in use can be deleted. Its importers drift and fail to
  compile on redeploy, which is the signal.

### 2.3 Import syntax

An import names a module, never a function. `import ranges.create` stays an error, and
the caller writes `ranges.create(...)` after `import ranges`. Only a module can be
imported. An automation cannot.

Moving or renaming a module does not rewrite its importers, because no infrastructure
rewrites Arc text on a rename. A channel rename does not either. Importers report an
unknown module on the next analysis.

How languages do it:

| Lang      | Syntax                      | Path is               | Binds                  | Alias              | Nesting                                     |
| --------- | --------------------------- | --------------------- | ---------------------- | ------------------ | ------------------------------------------- |
| Go        | `import "a/b/c"`            | string, dir path      | pkg name (usually `c`) | `import x "a/b/c"` | by directory                                |
| Python    | `import a.b.c`              | dotted, dir or file   | `a` (top)              | `as x`             | by directory, `from a.b import c` binds `c` |
| Java      | `import a.b.C`              | dotted, dir           | `C` (last)             | none               | by directory                                |
| Rust      | `use a::b::c`               | `::` path, mod tree   | `c` (last)             | `as x`             | `mod` declarations, any depth               |
| TS/JS     | `import { x } from "./a/b"` | string, file path     | named members          | `as x`             | by file                                     |
| C         | `#include "a/b.h"`          | string, file          | nothing, textual paste | none               | by file                                     |
| C++20     | `import a.b;`               | dotted name, not path | nothing, names global  | none               | `.` is cosmetic, flat                       |
| Elixir    | `alias A.B.C`               | dotted name           | `C` (last)             | `as:`              | by name only, no files                      |
| Arc today | `import a.b`                | dotted                | last segment           | `as x`             | parses, lookup flat                         |

`import "sylib/alert"` is the Go and C spelling. It makes two rules: a bare identifier
for the STL and a string for a user module. `/` is also not an Arc token. Rejected.

#### Decision: `import sylib.alert`.

- Zero grammar change. Zero parser change.
- Path = group names under "Arcs", `.`-joined. Last segment = module name = default
  alias. Matches Java, Rust, and Go binding, Python spelling.
- One rule for the STL and user modules: STL = depth 1, user = depth 1 or more under the
  Arcs group.
- Code change = `lookupModule` only: walk segments, fall back to a resolver for user
  modules. Same shape as the channel `globalResolver`.
- Group name identifier validation is already done, so every segment is a legal
  identifier. That is why the ontology step came first.

Must guard against:

- **STL conflicts**: A group or module directly under "Arcs" named like an STL module,
  such as `time`. Options:
  - Warn and use the Synnax module.
  - Error on the implicit shadowing.
  - Refuse a name under "Arcs" that an existing module already uses.
- **Alias shadowing**: Two user modules at different paths with the same name, or an
  import list like the one below. Both bind `time`. The duplicate import diagnostic
  already catches a repeated alias at the root, so the second needs `as`.

  ```arc
  import (
      time
      my_modules.time
  )
  ```

### 2.4 Exports

How languages do it:

| Lang          | Mechanism        | Default     | Marker                         | Granularity                |
| ------------- | ---------------- | ----------- | ------------------------------ | -------------------------- |
| Go            | capitalization   | private     | `Name` vs `name`               | per identifier             |
| Python        | convention       | public      | `_name`, optional `__all__`    | per identifier, unenforced |
| Rust          | keyword          | private     | `pub`, `pub(crate)`            | per item                   |
| TS/JS         | keyword          | private     | `export`, or `export { a, b }` | per declaration            |
| C             | header + linkage | public      | `static` hides                 | per symbol                 |
| C++20 modules | keyword          | private     | `export`, `export { ... }`     | per declaration or block   |
| Java          | keyword          | package     | `public` / `private`           | per member                 |
| Elixir        | two keywords     | public      | `def` vs `defp`                | per function               |
| Zig           | keyword          | private     | `pub`                          | per declaration            |
| Haskell       | header list      | public      | `module M (a, b) where`        | per module list            |
| Lua           | return table     | private     | `return { a = a }`             | per module list            |
| Swift         | keyword          | module-wide | `public` / `private`           | per declaration            |

#### Decisions:

- **Default private**: A top-level `func`, `stage`, `sequence`, or variable declaration
  is visible to importers only when it starts with `export`, the keyword that pairs with
  `import` on the other side.
- **Modules only**: `export` in an automation is an error.
- **Same line**: `export` on a different line from its declaration is an error.
- **No authority block**: A top-level authority block in a module is an error.
- **No flow**: A top-level flow in a module is an error.
- **No re-export**: `Main` must `import status` itself to call `status.set`, even when
  the module it imports does too.
- **No automation exports**: Automation arcs throw an error if the `export` key word is
  used.

### 2.5 Deploy drift in the task config

"Redeploy" says the running instance is stale. The tooltip should say why: the Arc's
own text, one of its modules, or both. That needs what the instance was built from,
stored somewhere the Console reads.

The task config is that place. It is the deploy unit: the Driver echoes its hash in
the task status, and the Console compares the two to show drift. The alternatives do
not fit. An Arc status is the Arc's own state and would put a row per automation in the
status explorer. An ontology relationship has no fields.

`arc_task` v5 adds four fields to `Config`:

- **`modules`**: The hash of each import, in import order, as `ModuleHash` pairs of
  `path` and `hash`. A module's hash folds in its own imports. A list, not a map: the
  client camel-cases map keys, and a path may carry underscores.
- **`content_hash`**: The Arc's own hash, imports left out.
- **`deployed_modules`** and **`deployed_content_hash`**: The same two as they stood
  when the running instance was built.

The Core keeps them current in `UpdateTask` and `syncTask`. A sync that changes the
Arc's hash first reads the task status. When the Driver reports the stored config's
hash, the instance runs that config, so the stored `modules` and `content_hash` become
the deployed copies. Otherwise the deployed copies stay, so edits before a redeploy
accumulate. Then the new values are stamped and the task rewritten.

The Console diffs each pair and the Redeploy tooltip lists one line per finding, "Arc
changed since deploy" and "Modules changed since deploy: `sylib.wave`", then "Redeploy
to apply". Neither line means the drift came from the rack or the performance level.

Two Oracle changes carry this:

- **`oracle migrate arc_task`**: v4 shipped in 0.59.0, a minor release, so it cannot
  change in place and v5 is a new version file. Its migration starts the lists and
  content hashes empty; the next sync of each task fills them.
- **Enum defaults through a version alias**: v5 keeps `Performance = v4.Performance`
  and `performance Performance = auto`. The analyzer resolved the field's type to the
  alias, not the enum, and rejected the default. `TypeRef.ResolveUnderlying` now
  follows alias targets, and the enum and union variant resolvers use it. Every
  generator renders defaults through those resolvers, so Go and TypeScript both pick
  up the fix.

### 2.6 Open

- Which STL conflict option in §2.3 to take.

## 3 Console

### 3.0 Explorer

The Arc explorer becomes an ontology tree rooted at "Arcs", built on `platform/tree`
like the channel tree, with the group context menu item. Ranges and statuses keep their
flat lists.

### 3.1 Toolbar

The "Arc automations" toolbar lists automations only. Modules are reached through the
explorer.

### 3.2 Views

Open. A saved view over Arcs would filter by type and mode. Whether the view framework
sits above a tree body is undecided.

## 4 Phases

- **Phase 1: Arc type.** Done. `type` is `automation` or `module`, chosen at creation,
  never changed. Module names are identifiers.
- **Phase 2: Compile status.** Done. The Core compiles a module after each edit and
  stores the result as the Arc's status. The editor shows it in place of a task status.
- **Phase 3: Arcs group.** The root group, the migration, and the explorer tree.
- **Phase 4: Imports.** Done. The `imports` relationship and change propagation in §2.2.
