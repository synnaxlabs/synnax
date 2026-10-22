# 82 Arc user modules

- **Author**: Nico Alba
- **Date**: 2026-10-21
- **Related**: [RFC 0005 - MVP - Data ontology](0005-ontology.md),
  [RFC 0030 - Arc module system](0030-arc-modules.md),
  [RFC 0056 - Task autosave with deploy-on-start](0056-task-autosave-deploy-on-start.md)

## 0 Summary

An Arc is either an automation or a module. A module exports code that other Arcs import
by path, and the path is the module's position in the ontology below an "Arcs" group. An
`imports` relationship from each importer to its module lets a module edit reach its
importers. This RFC covers the implementation, not the rendering.

## 1 Motivation

Users need to write a function or sequence once and import it from many automations. Arc
resolves only standard modules today (RFC 0030 §5), so an automation engineer who writes
a shutdown sequence or a conversion function pastes it into each automation that needs
it.

## 2 Vocabulary

- **Automation**: An Arc with `type = automation`. It binds to a rack and is the parent
  of its task.
- **Module**: An Arc with `type = module`. It has no rack and no task.
- **Path**: The group names below "Arcs", then the Arc's own name, joined by `.`. A
  module's path is its import name.
- **Importer**: An Arc whose text imports a module, directly or through another module.

## 3 Design

### 3.0 Arc type

An Arc carries `type`, `automation` or `module`, chosen at creation and never changed.
Binding a rack to a module is a validation error, and so is importing an automation.

### 3.1 The Arcs group

![Arc ontology](img/0082-arc-user-modules/ontology.svg)

- **Root group**: The Arc service creates "Arcs" below the ontology root with
  `group.Service.CreateOrRetrieve`, as the channel, task, and project services do, and
  parents every new Arc under it.
- **Directories**: A group below "Arcs" is a directory. A module may sit directly under
  "Arcs" or in a group at any depth (§5.0, §5.1).
- **Names**: Group and module names are validated like channel names.
- **Resolution**: A path walks `parent_of` down from "Arcs", one segment per level, and
  matches the last segment against a module's name. An import binds the last segment:

```arc
import (
    sy_lib.m1
    sy_lib.sub_group.m3
    m4 // directly under "Arcs"
)

true => m1.f{}
true => m3.f{}
true => m4.f{}
```

### 3.2 The `imports` relationship

An import is an ontology relationship of a new type, `imports`, from the importing Arc
to the module, beside `parent_of` and `labeled_by`.

- **Derived from the text**: A save, and every resync, rewrites the Arc's outgoing
  `imports` relationships from its import statements.
- **By key, not path**: A moved or renamed module keeps its relationships, so its
  importers are still found, and the resync of §3.2.0 reports the unknown path (§5.5).
- **Unknown path, no relationship**: An import whose path resolves to no module leaves
  no relationship.
- **Cycles**: The analyzer reports the chain. The save records the flagged import
  instead of writing a relationship (§3.2.0).
- **Delete**: A module in use can be deleted. Its importers resync before the
  relationships go and fail to compile, which is the signal.

#### 3.2.0 Change propagation

An Arc's content hash (RFC 0056 §6.6) folds in its modules' hashes, transitively, so a
module edit makes every importer's task drift. The config also stores the Arc's own hash
and one per module. An as-built copy of each lives outside the hashed config, so a diff
names what changed and promotion never moves `config_hash`. A resync promotes stored to
as-built when the task status shows the Driver running the stored config. A save before
the Driver echoes a redeploy skips promotion and over-names until a later resync
matches. Three kinds of change resync Arcs.

- **Content**: A module write resyncs its importers, found through incoming `imports`
  relationships.
- **Path**: A module or group created, renamed, or moved below "Arcs" resyncs every Arc
  once, since an importer may already name the new path or lose the old one.
- **Removal**: A save that drops an import, or a delete, resyncs every recorded Arc
  (§3.2) as a module write, so it rewrites its relationships and its importers rehash.

#### 3.2.1 Cost

The alternative is to resync every Arc on every module write: O(N) against O(I + D),
with N Arcs stored, I Arcs that depend on the saved module, and D module levels. A
prototype benchmark stores N automations and a chain of D modules with P automations
importing each, so I = (D − 1) + P × D:

![Time per module save](img/0082-arc-user-modules/importers.svg)

The scan costs about 0.1 ms per Arc stored, 1 s at N = 10000, whatever I is. Ontology
imports cost about 0.06 ms per dependent Arc, 50 ms at I = 807, whatever N is.

### 3.3 Name conflicts

- **Same name, different directories**: Allowed. An Arc that imports both binds the name
  twice, which the duplicate-import diagnostic rejects: all but one need `as`.
- **Same name, one directory**: Rejected on create, rename, and move, whatever the
  types. Existing duplicates stay.
- **A standard module's name at depth 1**: `time` directly under "Arcs" has the same
  path as the standard `time`. Open in §7.0.

### 3.4 Exports and compilation

A top-level `func`, `stage`, `sequence`, or variable in a module is visible to importers
only when its declaration starts with `export`, the keyword that pairs with `import`.
`export` in an automation, a top-level flow or authority block in a module, and an
import that names a function are errors. There is no re-export. An importer's program
contains the exported code of its modules, so the Driver receives one program per task,
as today. Each importer gets its own copy of an exported variable, which the importer
cannot write to. A module has no task, so the Core compiles it on each save and stores
the result as the Arc's status.

### 3.5 Migrations

- **Arc**: Existing Arcs re-parent under "Arcs". Type defaults to `automation`.
- **Task config**: A new version adds the stored hashes of §3.2.0, empty until the next
  sync. The as-built record starts empty.

## 4 Implementation phases

Each phase is one pull request into `main`.

- **Phase 1: Arc type and the Arcs group.** New and existing Arcs are parented under it.
- **Phase 2: Module resolution and `export`.** Path walk, analyzer resolution, and cycle
  diagnostics.
- **Phase 3: The `imports` relationship.** Rewrite on save, importer lookup, hash
  folding, and the resyncs of §3.2.0, with the benchmark of §3.2.1.
- **Phase 4: Compile modules into the importer.**
- **Phase 5: The task config hashes of §3.2.0.** A new config version.

No stable release stores a module or an `imports` relationship. Shipped task configs
take the migration of §3.5.

## 5 Resolved decisions

**5.0 The path is ontology position, not a field.** A `path` field would need its own
uniqueness index and could disagree with the tree.

**5.1 A module may sit at depth 1.** Forcing a group would reserve depth 1 for the
standard library, at one group per user. The trade leaves §7.0 open.

**5.2 `export` marks visibility, not capitalization.** Capitalization makes visibility a
property of the name, so a rename changes what importers see.

**5.3 Relationships over a full scan.** O(I + D) against O(N) per module write (§3.2.1),
for one ontology write per import per save.

**5.4 Drift names the module that changed.** An as-built hash per module lives beside
the task config (§3.2.0). RFC 0056 §4.4 rejected a deployed config snapshot, which
duplicates Driver-held state. A hash is one string, kept outside `config_hash`. No
Driver change.

**5.5 The relationship is derived from the text, not authoritative.** Rewriting
importers on a rename needs a rename engine in the Core and a merge into open editors
([SY-5173](https://linear.app/synnax/issue/SY-5173)).

## 6 What this RFC does not cover

Parameterized sequences and stages, `=> name` inside imported code, modules in graph
mode, Arc programs following renames, bundle import and export of modules (RFC 0052),
access control on modules, and the Console beyond Phase 1's explorer work.

## 7 Open questions

**7.0 How is a user name that matches a standard module handled (§3.3)?**

- [ ] Refuse the name on create, rename, or move.
- [ ] The standard module wins and the importing Arc reports the shadowed one.
- [ ] Error only when the path is imported.
