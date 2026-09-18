# 58 Gorp references

- **Author**: Patrick Dotson
- **Date**: 2026-09-18
- **Related**: [RFC 0005 - MVP - Data ontology](0005-ontology.md),
  [RFC 0026 - Meta data structures](0026-meta-data.md),
  [RFC 0034 - Gorp in-memory indexes](0034-gorp-indexes.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md),
  [RFC 0053 - Oracle explicit schema versioning](0053-oracle-explicit-schema-versioning.md),
  [RFC 0056 - Task autosave and deploy-on-start](0056-task-autosave-deploy-on-start.md)

## 0 Summary

A relationship between two records is stored as a row field, an ontology edge, or part
of a composite key, and sometimes as two of these at once. The ontology resource table
holds a second copy of every record's identity. Nothing checks that a reference points
at a row that exists, and nothing deletes a dependent when its owner is deleted.

This RFC removes the ontology. A relationship is a field on one table, declared in the
Oracle schema with `@ref`. Gorp validates the target on write, keeps a reverse index on
the field, and applies a declared policy (`cascade`, `restrict`, `detach`) when the
target is deleted, inside the caller's transaction and under one node-local write lock.
Oracle generates the rest. The `/ontology/*` API, the client relationship caches, both
ontology tables, and the ontology signals are deleted. The `{type, key}` identifier
survives as `resource.ID`.

## 1 Motivation

RFC 0005 introduced the ontology so authorization and navigation could relate records
without coupling their services. RFC 0026 §1.1.10 named the result: three patterns for
one job. RFC 0042 §7 deferred the fix to this RFC.

- **One relationship type carries eight meanings**: Two edge types exist, `parent`
  (`core/pkg/service/ontology/versions/v0/relationship.go`) and `labeled_by`
  (`core/pkg/service/label/relationship.go`). `parent` means group membership, project
  ownership, rack ownership, device chassis, range nesting, range snapshot, Arc binding,
  and role assignment. Readers recover the meaning by filtering on endpoint types
  (`core/pkg/service/ranger/service.go`, `client/ts/src/ranger/client.ts`).
- **Identity is stored twice**: The resource table persists `Resource{ID}` only
  (`core/pkg/service/ontology/writer.go`); the owning service fills name and data at
  read time. SY-4804 traced the entry-versus-resource orphan window to this copy.
- **Integrity is hand-written and leaks**: Rack, role, policy, and range alias deletes
  leave the resource behind (`writer.go` in
  `core/pkg/service/{rack,access/rbac/role,access/rbac/policy,ranger/alias}`). Project
  delete orphans every visualization under it (`core/pkg/service/project/writer.go`).
  Range delete leaves aliases and key-value pairs. Every Arc, internal task, copied
  task, and alias has no parent. Migrations write both ontology tables through raw Gorp
  writers, skipping the cycle and existence checks
  (`core/pkg/service/panel/versions/v0/composition.go`).
- **The one duplicate drifts**: `device.rack` is persisted and a `rack parent_of device`
  edge is written from it (`core/pkg/service/device/writer.go`). Rack delete reads the
  field (`core/pkg/api/rack/rack.go`); the tree reads the edge.
- **Readers assume a tree; the model is a DAG**: Fourteen readers take the first parent
  (`client/ts/src/ontology/store.ts`, `core/pkg/api/device/device.go`). The project
  bundle walk needs a visited set (`core/pkg/service/project/bundle.go`).
- **The graph is slow where it matters**: Resource delete scans the whole relationship
  table per ID for incoming edges (`core/pkg/service/ontology/writer.go`). Children
  traversal prefix-scans past the index (`core/pkg/service/ontology/retrieve.go`). The
  limit applies after every child is materialized (SY-4763). The TypeScript cache scans
  every edge on every delete (SY-4754).

## 2 Vocabulary

- **Reference**: A field holding the key of a row in another table, or the same table.
  To-one is a scalar key (`device.rack`); to-many is a key array (`range.labels`). A
  many-to-many relationship is a to-many reference on the side that edits it.
- **Reverse lookup**: The rows whose reference field holds a given key. A one-to-many
  relationship is the reverse lookup of a to-one reference.
- **Target** and **dependent**: The table a reference points at; a row holding one.
- **Policy**: What Gorp does to dependents when a target row is deleted: `cascade`,
  `restrict`, or `detach`.
- **Resource ID**: The `{type, key}` pair identifying a row in any table.

## 3 Principles

1. **One copy**: A relationship is one field on one table. Never a field and an edge,
   never a field and a key encoding.
2. **The row is the unit of atomicity**: A reference lives on the dependent row, so the
   record and its relationship commit together on every store, including Aspen.
3. **Declared, then generated**: The schema declares the reference and its policy;
   Oracle generates the rest. Services write no relationship code.
4. **Integrity belongs to the store**: Gorp checks existence and applies the policy. It
   never calls service code. A side effect that is data is a reference; one that is not
   follows the table's change stream.
5. **Ownership and placement are different fields**: `schematic.project` is the owner;
   `schematic.group` is where a toolbar shows it. No tree is stored as a tree.
6. **Row-conditional rules live in service guards**: A policy is per field. "Refuse when
   non-internal tasks remain" stays a `gorp.Delete.Guard`.
7. **No generic graph endpoint**: No Core or client API exposes a type-agnostic
   children, parents, or move operation.

## 4 Design

### 4.0 Two field kinds

A reference is a scalar key or a key array; the reverse direction is never declared.

| Relationship shape | Storage                                        | Reverse lookup                     |
| ------------------ | ---------------------------------------------- | ---------------------------------- |
| To-one             | `rack rack.Key` on device                      | devices where `rack == k`          |
| One-to-many        | The to-one on the child                        | same                               |
| Many-to-many       | `labels label.Key[]` on the side that edits it | ranges where `labels` contains `k` |
| Polymorphic        | `owner resource.ID` on status                  | statuses where `owner == id`       |

Prior art agrees: a foreign key on the child row (SQL, Ent, Prisma, Kubernetes
`ownerReferences`, Grafana `folder_uid`), a cardinality-many ref for the many side
(Datomic, Gel `multi link`), and a delete policy declared with the reference (SQL
`ON DELETE`, Gel `on target delete`, Datomic `:db/isComponent`). The generic edge table
(Neo4j, Zanzibar, OPC UA) is the shape this RFC leaves: every reader filters to one
relation and one endpoint type, which is a field query.

### 4.1 The schema declaration

`@ref` is a field domain. The target is inferred from the field type: `rack.Key`
resolves to the struct in `rack.oracle` whose `@key` field has that type, and
`target Name` overrides an ambiguous match. A `resource.ID` field lists its targets:

```
Device struct {
    rack   rack.Key {
        @ref on_delete restrict
    }
    parent Key? {
        @ref { on_delete cascade
               acyclic }
    }
}

Status struct {
    labels label.Key[] = [] {
        @ref on_delete detach
    }
    owner  resource.ID? {
        @ref { types task rack device channel
               on_delete cascade
               unique }
    }
}
```

- `on_delete cascade | restrict | detach`: Required. What happens to dependents when the
  target is deleted: `cascade` deletes them (a project's schematics), `restrict` fails
  the delete while any exist (a rack with devices), and `detach` keeps them and clears
  the reference (a deleted label leaves every `range.labels`). `detach` zeroes a scalar,
  so the field must be optional or have a default, and removes the element from an
  array.
- `types ...`: Required on a `resource.ID` field. The tables the reference may point at.
- `unique`: At most one dependent per target, enforced by the reverse index on write. A
  nil value never counts. The scalar already gives a status one owner; `unique` gives an
  owner one status, which is what lets the Driver upsert by owner.
- `acyclic`: A self-reference may not form a cycle, checked by walking the chain.

`@ref` implies `@filter` and `@index lookup`, so every reference gets a generated
`Match<Field>s` filter routed through its index (RFC 0034 §2.2.1). `@ontology type`
becomes `@resource type` and still generates the ID factories in every language.

### 4.2 Gorp

Every piece below is node-local and runs in the caller's transaction.

**Registry**: `gorp.DB` holds a schema registry. `OpenTable` registers the table's name,
key parser, and references; `Table.Close` unregisters. This is Gorp's first cross-table
state, attached as `TableConfig.Indexes` are today (`x/go/gorp/table.go`).
`TableConfig.References` carries one `gorp.Reference` per field, each owning a
`LookupIndex` on the field; for arrays it fans out one entry per element, the
multi-valued extractor RFC 0034 did not ship. The per-tx delta overlay
(`x/go/gorp/delta.go`) gives index probes read-your-own-writes. Indexes populate
asynchronously at open, as today. An existence check is a primary-key get and needs no
index. A policy probe waits on `Table.WaitForIndexes`, and a populate failure is an
error on the probe, never a fallback scan: a table that could not be scanned at open
cannot be scanned at delete, and a scan per delete is the slow path §1 removes.

**Locking**: Gorp has no isolation between transactions. One tx can read that a rack
exists and create a device under it while another reads that the rack has no devices and
deletes it; both commit. `gorp.DB` holds one write lock. A `Create` or `Update` on a
table with references takes it before the first check, a `Delete` on a table that is a
reference target takes it before the first probe, and the tx holds it until commit or
close. `unique` runs under the same lock, so the Driver's upsert by owner is safe
against the Core's own status writes. The lock is node-local.

**Existence on write**: `Create` and `Update` resolve each reference through the
registry and probe the target in the tx. A Pebble indexed batch reads its own writes
(`x/go/kv/pebblekv/pebblekv.go`) and the Aspen tx reads through its batch
(`aspen/internal/kv/tx.go`), so a rack and its device created in one tx validate if the
rack is written first. Checks are immediate, never deferred; `unique` and `acyclic` run
with them.

**Policy on delete**: `Delete.Exec` runs guards, then computes the closure: it probes
every reverse index that targets the doomed keys, adds `cascade` dependents to the
closure, and repeats on them until nothing is added, with a visited set for
self-references. `restrict` is then checked against dependents outside the closure, the
SQL `NO ACTION` rule, so a project delete succeeds when its groups hold only schematics
the same delete removes. `detach` updates the dependents that remain. Nothing mutates
until every check passes. The whole graph commits in the caller's tx, whatever its size.

**Aspen**: A tx reads its local store (`aspen/internal/kv/tx.go`); a write from another
node arrives by gossip and reaches the indexes through the observer that feeds them
today. Every check is node-local. A target created on another node fails the existence
check until gossip lands, a dependent created there is invisible to a probe for the same
window, and the write lock does not reach across nodes. Conflicting writes on two nodes
in that window leave a dangling reference. The populate scan at open logs each one it
finds.

**Polymorphic targets**: The registry is keyed by table name, the `@resource type`
string. A `resource.ID` reference probes the table named by `id.Type` and parses
`id.Key` with that table's key parser, the generated `KeyFromOntologyID`, renamed
`KeyFromResourceID`. `core/pkg/service/resource` holds `resource.ID`, `resource.Type`,
and the type enum; search, access enforcement, export addressing, panel tabs, policy
objects, and status owners use them unchanged. Nothing else from the ontology survives.

### 4.3 Every relationship, mapped

| Today                                                      | Field                                                | Policy             |
| ---------------------------------------------------------- | ---------------------------------------------------- | ------------------ |
| `rack parent_of device` + `device.rack`                    | `device.rack rack.Key` (exists)                      | restrict           |
| `device parent_of device` (chassis)                        | `device.parent device.Key?`                          | cascade, acyclic   |
| `task.rack`                                                | exists                                               | cascade (see §4.6) |
| `arc parent_of task` + `config.arc_key`                    | `arc/task.Config.arc_key` (exists)                   | cascade            |
| task config store keyed by task                            | `config.task task.Key` (the key)                     | cascade            |
| `range parent_of range`                                    | `range.parent ranger.Key?`                           | cascade, acyclic   |
| `range`, `status labeled_by label`                         | `range.labels`, `status.labels label.Key[]`          | detach             |
| status key `type:key`                                      | `status.owner resource.ID` (§4.5)                    | cascade, unique    |
| `range parent_of task`, schematic (snapshot)               | `task.range`, `schematic.range ranger.Key?`          | cascade            |
| `project parent_of` schematic, lineplot, log, table, panel | `X.project project.Key?`                             | cascade            |
| `group parent_of` X                                        | `X.group group.Key?`                                 | restrict           |
| `group parent_of group`, project, rack                     | `group.parent resource.ID?` (§4.4)                   | cascade, acyclic   |
| `role parent_of user`                                      | `user.roles role.Key[]`                              | detach             |
| `role parent_of policy`                                    | `policy.role role.Key`                               | cascade            |
| alias `range---channel`, kv `range<--->key` keys           | `alias.range`, `pair.range` (composite `@key`)       | cascade            |
| `Tasks`, `Statuses`, `Views`, `Metrics` groups             | deleted; `Metrics` becomes an ordinary channel group |                    |

Composite keys stay: alias and key-value pairs are natural keys whose pair is the
uniqueness constraint. Both pair fields carry `@key`, Oracle derives the string
`GorpKey` from them, and the hand-written builders and parsers
(`core/pkg/service/ranger/alias/alias.go`) are deleted. The reference index reads the
row fields, so nothing parses a key.

### 4.4 Groups

A group is a placement container. Its container is a toolbar root, a project, another
group, or in future a rack. A field per container type does not scale, so:

```
Group struct {
    key    Key
    name   string
    scope  resource.Type
    parent resource.ID? {
        @ref { types group project rack
               on_delete cascade
               acyclic }
    }
}
```

`scope` names the tree the group belongs to (`channel`, `device`, `project`,
`schematic_symbol`) and is constant down a nesting chain, checked by the group service
on write. A tree root is "groups of my scope whose parent is nil or the project" plus
"resources of my type with no group". Grouping devices under a rack sets
`group.parent = rack:1`; the device service checks on write that `device.rack` matches
the group's rack ancestor. No builtin root group exists; the seven created by
`Group.CreateOrRetrieve` (`core/pkg/service/channel/service.go` and siblings) are
deleted by the migration in §4.9. Group delete is `restrict` on members, keeping today's
"cannot delete a group with children" (`core/pkg/service/group/service.go`); the Console
ungroups first.

### 4.5 Statuses

Status keys embed the owner (`rack.StatusKey`, `core/pkg/service/rack/rack.go`), the
Driver recomputes them at more than twenty sites (`driver/common/status.h`), and
`details` carries a second copy of the rack and task keys. The key becomes a UUID and
`owner resource.ID?` is a unique cascade reference. The status set endpoint accepts an
owner and upserts through the unique index, so the Driver sends `{owner: task_id, ...}`
and keeps its idempotent write with no round trip. A status with no owner, such as one
from the Console's "Create status" command or the client `set` call, keeps `owner` nil
and is never cascaded. `Status.Details.{Task,Rack,Device}`, `rack.StatusKey`,
`calculation.StatusKey`, and the C++ `status_key` functions are deleted. Calculation
statuses take the channel as owner.

### 4.6 Services

Every `ontology.Writer` call, `ontology.Service` implementation, and `RegisterService`
call is deleted; 22 services register today and each defines resources and edges by
hand. Generated `Match<Field>s` filters route through the reference index, so
`device.retrieve({rack})`, `range.retrieve({parent})`, `schematic.retrieve({project})`,
`user.retrieve({role})`, and `group.retrieve({parent, scope})` cost one index probe.
Range delete drops its BFS (`core/pkg/service/ranger/writer.go`); cascade walks
`parent`. Project delete cascades. `schematic.findParentProject`,
`ranger.RetrieveParentKey`, `symbol.rescueStrays`, and
`metrics.maybeDefineGroupRelationship` go with the edges they read.

Rack delete keeps today's refusals (`core/pkg/api/rack/rack.go`): `device.rack` is
`restrict`, and the refusal while non-internal tasks remain is a `gorp.Delete.Guard` in
the rack service like `embeddedGuard`. `task.rack` is `cascade`, so internal tasks go
with the rack. A schematic, line plot, log, table, or panel holds exactly one of
`project` and `range`. The schema cannot say so, and each visualization service
validates it on create and update.

Side effects that are not data follow the change stream. Channel delete calls Cesium
storage delete after the row delete, outside the tx
(`core/pkg/service/channel/writer.go`). It moves to a subscriber on the channel table's
delete stream, paired with the startup reclaim sweep from RFC 0042 §7, so a cascaded
channel delete takes the same path as a direct one.

### 4.7 API and clients

`/ontology/retrieve`, `add-children`, `remove-children`, `move-children`
(`core/pkg/api/ontology/ontology.go`), the `/ontology/*-group` endpoints, and the
ontology access checks are deleted. Group create, rename, and delete become `/group/*`.
Moving a resource into a group is an update of its `group` field through its own
service; Console drag-and-drop issues one typed update per resource type instead of
`moveChildren`.

The TypeScript client deletes `client/ts/src/ontology` except the ID payload, which
moves to `resource`, along with the relationship cache, `Cache.parentID`, every
`parents[0]` reader, the two `RelationshipIndexes`, and the `sy_ontology_*` listeners.
Each domain client keeps a per-table query cache with lookup indexes
(`client/ts/src/query/indexes.ts`), where reference fields are ordinary indexed fields.

The Console tree stays generic over a root `resource.ID`, but children come from the
item types. Each `Tree.createItem` gains a `children` resolver: the rack item lists
devices whose `rack` is the rack, the group item lists child groups and grouped
resources of its scope. Toolbars keep `Tree.Tree root={...}`
(`console/src/feature/channel/Toolbar.tsx`) with a synthetic scope root instead of a
builtin group found by name (`pluto/src/device/queries.ts`).

An expanded node stays live through the table streams. The TypeScript client keeps one
streamer per table on its `sy_<table>_set` and `sy_<table>_delete` channels
(`client/ts/src/query/cache.ts`) and applies each change to that table's cache. An
expanded rack node is a device list query with `rack` as its parameter; a device set
whose `rack` changed leaves one list and enters another through the lookup index on the
field, which is the move the ontology signals carried. Subscriptions scale with tables,
not with expanded nodes. Python deletes `synnax.ontology`; `Range.children` becomes a
`parent` filter. C++ renames `ontology::ID` to `resource::ID`; the Driver's status
writes send an owner.

### 4.8 Search, access, export, and signals

Search registers services by `resource.Type` and indexes flat documents
(`core/pkg/service/search/search.go`), unchanged but for its import. Access enforcement
matches `resource.ID` exactly or by type wildcard
(`core/pkg/service/access/rbac/service.go`). Its one traversal, subject to role to
policy (`core/pkg/service/access/rbac/policy/retriever.go`), becomes `user.roles` then
`policy.retrieve({role})`. The export request stays a `resource.ID`. The project bundle
walk (`core/pkg/service/project/bundle.go`) becomes typed queries by parent, project,
and group; its visited set goes because a visualization has one group.

The four `sy_ontology_*` channels are deleted. Every Gorp-backed table publishes set and
delete channels through `signals.PublishFromGorp`; a set signal carries the row, so a
reference change is an entity change, and a cascade emits one delete per dependent.

### 4.9 Migration

One startup migration per table derives its reference fields from the old edges. It
reads the relationship table through `gorp.WrapReader[string, Relationship]`, as the
project v1 migration does (`core/pkg/service/project/versions/v1/migrate.go`), so the
relationship type stays frozen in `ontology/versions/v0` and its chain ends with a
tombstone per RFC 0053. Derivations, from `parent` edges unless noted:

- `device.parent` from device-to-device edges. Every `group` field from group-to-X edges
  whose group is not a builtin root.
- `range.parent`, `task.range`, `schematic.range` from range edges; `range.labels` and
  `status.labels` from `labeled_by` edges.
- `X.project` from project-to-X edges, or through the group chain to the project.
- `group.parent` from the edge into the group: a builtin root maps to nil, a project or
  group to its ID. `group.scope` from the builtin root at the top of the chain.
- `user.roles` and `policy.role` from role edges.
- `status.key` becomes UUIDv5 of the old key over a fixed namespace (`uuid.NewSHA1`), so
  every node derives the same key, and `status.owner` is parsed from the old key. A key
  that parses to no owner leaves `owner` nil. The PagerDuty alert config's `status`
  field is rewritten by the same function as a plain value migration; it becomes a
  declared reference in Phase 4.

After every service has opened, a service-layer step behind a marker key drops both
ontology tables and the builtin groups. Rows that resolve to nothing (a visualization
with neither project nor range) are deleted; they are unreachable today. Migrations run
per node with no coordination (RFC 0033 §4.2.5); every derivation is deterministic.

## 5 Implementation phases

- **Phase 1: Gorp references and Oracle `@ref`.** The registry, `gorp.Reference`, the
  write lock, the multi-valued index, existence, unique, acyclic, the three policies,
  and composite `@key`, tested in `x/go/gorp`. The `resource` package and the
  `@resource type` rename. Oracle emits reference registration and index-routed filters.
  No Core table declares a reference yet, so the store change is reviewed alone.
- **Phase 2: Domain cutovers.** One pull request per domain: add its fields and
  migration, delete its ontology writes and readers, update its TypeScript, Python, and
  Console code. The ontology keeps serving the domains not yet cut over, so every
  boundary is green. Order: labels and statuses (arrays, UUID status keys, the Driver
  owner write); ranges (parent, snapshots, alias and key-value references); groups,
  channels, and the Console tree resolvers; racks, devices, tasks, and Arc; projects,
  visualizations, panels, and bundle export; users, roles, and policies.
- **Phase 3: Delete the ontology.** The package, the API, the signals, the client
  modules, the generic children query, and the table-drop step. A pure deletion.
- **Phase 4: Nested references.** `@ref` on a field inside a record or array
  (`lineplot.channels`, `log.channels`, task config `device` and `channel` keys, `panel`
  tab resources, `policy.objects`, the PagerDuty `status` key). Oracle emits a path
  extractor, as FoundationDB's nested index key expressions do; `detach` removes the
  element or nulls the field.
- **Phase 5: Channel references.** `index channel.Key` and a stored, compiler-derived
  `requires channel.Key[]`, both `restrict`. The leaseholder stays in the key; a node is
  not a table. Phases 4 and 5 are named now so the declaration language is designed for
  them, and sequenced last because nothing earlier depends on them.

## 6 What this RFC does not cover

- Cross-node integrity on Aspen. A Gorp tx still splits into one batch per leaseholder
  (`aspen/internal/kv/tx.go`), and every check reads the local replica (§4.2). A
  reference is atomic with its row on every store; a cascade that spans leaseholders
  keeps today's guarantee.
- Storage-first channel create atomicity (RFC 0042 §7).
- Search index structure and the Console search palette, which change only an import.
- Console session state, which never held relationships.

## 7 Resolved decisions

1. **Fields, not edges, and never both**: Keeping the edge table as the single copy,
   with fields resolved from it, was rejected. It reverses RFC 0056 §6.5, which made
   `task.rack` a field so a rack change moves no edges; it puts the record and its
   relationship in different Aspen batches; and no reader needs an untyped traversal.
   Cost: a fact many types share, such as group membership, is a field on each of them.
2. **Delete both ontology tables**: SY-4804 targeted the resource table alone; deleting
   only it keeps the dual-write.
3. **Cascade never calls service code**: A per-table delete hook, so a cascaded channel
   delete could free Cesium storage synchronously, was rejected. Ent (issue #2031) shows
   hooks and store-level cascade disagreeing on which runs.
4. **Groups take a polymorphic parent and a scope**: A field per container type was
   rejected because each new container adds a field; per-domain group tables were
   rejected as duplication. Cost: scope consistency is a service guard, not a type.
5. **No stored tree flag**: A `tree` marker on references was rejected. Placement is the
   `group` field; ownership is the owning reference; the toolbar decides what it shows.
6. **UUID status keys**: Owner-derived string keys were rejected. The key encoding is a
   hidden reference, and the unique index gives the Driver the same idempotent upsert.
7. **Composite keys stay for alias and key-value**: A UUID migration was rejected. The
   pair is the identity; Oracle derives the key from the `@key` fields, and the
   reference on those fields gives cascade.
8. **Immediate existence checks**: Deferred constraints (SQL `INITIALLY DEFERRED`) were
   rejected. Parent-first ordering is a small rule; deferral adds a commit-time phase.
9. **Restrict on group members**: Cascade was rejected. Deleting a folder must not
   delete channels, and the Console ungroups first.
10. **No generic endpoint of any kind**: A typed `/relationship` façade was rejected; it
    reintroduces the untyped graph.
11. **Orphans are deleted in migration**: Preserving unreachable rows was rejected. They
    are invisible today, and a required reference cannot hold nothing.
12. **One write lock per DB**: A lock per target table was rejected. A cascade acquires
    locks lazily across tables, so two transactions acquiring in opposite orders
    deadlock. Metadata write rates make one lock cheap.
13. **Restrict is checked after the closure**: Immediate `restrict` (SQL `RESTRICT`) was
    rejected because a project delete would fail on its own groups.
14. **Deterministic status keys**: A random UUID was rejected. RFC 0033 §4.2.5 migrates
    each replica independently, so two nodes would derive two keys for one row.
15. **`status.owner` is optional**: A required owner was rejected. The Console and the
    clients create statuses with no owner today.
16. **Unbounded fan-out**: A batch limit on cascade and detach was rejected. A delete of
    any size is one transaction.
17. **No fallback scan on a policy probe**: A failed index populate makes the probe
    fail. A scan per delete is the cost §1 removes.

## 8 Open questions

- Whether `group.scope` can be dropped once every group has a parent or a member.
- The `@ref` block grammar: newline-separated like `@ts { }`, or a delimiter.
- Device chassis policy: `cascade` (discovered subdevices) or `detach` (hand-created).
- The reclaim sweep cadence for Cesium storage after a cascaded channel delete.
- Repair of the dangling references the populate scan logs on Aspen.
