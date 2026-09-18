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

A relationship between two records is stored as a row field, as an ontology edge, or as
part of a composite key, and sometimes as two of these at once. The ontology resource
table stores a second copy of every record's identity. Nothing checks that a reference
points at a row that exists. Nothing deletes a dependent when its owner is deleted. Each
service writes its own edge management.

This RFC removes the ontology. A relationship is a field on one table, declared in the
Oracle schema with `@ref`. Gorp validates that the target exists on write, keeps a
reverse index on the field, and applies a declared policy (`cascade`, `restrict`,
`detach`) when the target is deleted, inside the caller's transaction. Oracle generates
the registration, the filters, and the client types. The `/ontology/*` API, the client
relationship caches, both ontology tables, and the ontology signal channels are deleted.
The polymorphic `{type, key}` identifier survives in a `resource` package.

## 1 Motivation

RFC 0005 introduced the ontology so that authorization and navigation could relate
records without coupling their services. RFC 0026 §1.1.10 named the result: three
patterns for one job. RFC 0042 §7 deferred the fix to this RFC.

- **One relationship type carries eight meanings**: Two edge types exist, `parent`
  (`core/pkg/service/ontology/versions/v0/relationship.go:26`) and `labeled_by`
  (`core/pkg/service/label/relationship.go:18`). `parent` means group membership,
  project ownership, rack ownership, device chassis, range nesting, range snapshot, Arc
  binding, and role assignment. Readers recover the meaning by filtering on endpoint
  types, in Go (`core/pkg/service/ranger/service.go:176`,
  `core/pkg/service/schematic/writer.go:96`) and TypeScript
  (`client/ts/src/ranger/client.ts:336`, `client/ts/src/arc/client.ts:116`).
- **Identity is stored twice**: The resource table persists `Resource{ID}` only
  (`core/pkg/service/ontology/writer.go:34`); the owning service fills name and data at
  read time. The table exists so edge writes can check both ends and so a type can be
  listed. SY-4804 traced the entry-versus-resource orphan window to this second copy.
- **Integrity is hand-written and leaks**: Rack, role, policy, and range alias deletes
  leave the ontology resource behind (`core/pkg/service/rack/writer.go:127`,
  `core/pkg/service/access/rbac/role/writer.go:63`,
  `core/pkg/service/access/rbac/policy/writer.go:59`,
  `core/pkg/service/ranger/alias/writer.go:62`). Project delete orphans every schematic,
  plot, log, and table under it (`core/pkg/service/project/writer.go:94`). Range delete
  leaves aliases and key-value pairs behind. Every Arc, internal task, copied task, and
  alias has no parent. Migrations write both ontology tables through raw Gorp writers,
  skipping the cycle and existence checks
  (`core/pkg/service/panel/versions/v0/composition.go:198`).
- **The one duplicate drifts**: `device.rack` is persisted and a `rack parent_of device`
  edge is written from it (`core/pkg/service/device/writer.go:82`). The rack delete
  precondition reads the field (`core/pkg/api/rack/rack.go:199`); the tree reads the
  edge.
- **Readers assume a tree; the model is a DAG**: Fourteen readers take the first parent
  (`client/ts/src/ontology/store.ts:66`, `client/ts/src/task/client.ts:100`,
  `core/pkg/api/device/device.go:168`). The project bundle walk needs a visited set
  because a resource can have two parents (`core/pkg/service/project/bundle.go:124`).
- **The graph is slow where it matters**: Deleting a resource scans the whole
  relationship table per ID for incoming edges
  (`core/pkg/service/ontology/writer.go:194`). Children traversal prefix-scans instead
  of using the index (`core/pkg/service/ontology/retrieve.go:244`). The limit applies
  after every child is materialized (SY-4763). The TypeScript cache mirrors every edge
  and scans it on every delete (SY-4754).

## 2 Vocabulary

- **Reference**: A field holding the key of a row in another table, or the same table.
- **To-one reference**: A scalar key field. `device.rack`, `range.parent`.
- **To-many reference**: A key array field. `range.labels`, `user.roles`. A many-to-many
  relationship is a to-many reference on the side that edits it.
- **Reverse lookup**: The rows whose reference field holds a given key. A one-to-many
  relationship is the reverse lookup of a to-one reference.
- **Target**: The table a reference points at.
- **Dependent**: A row that references a target row.
- **Policy**: What Gorp does to dependents when a target row is deleted: `cascade`,
  `restrict`, or `detach`.
- **Polymorphic reference**: A reference whose target table varies per row, held as a
  `resource.ID`.
- **Resource ID**: The `{type, key}` pair that identifies a row in any table.

## 3 Principles

1. **One copy**: A relationship is one field on one table. Never a field and an edge,
   never a field and a key encoding.
2. **The row is the unit of atomicity**: A reference lives on the dependent row, so the
   record and its relationship commit together on every store, including Aspen.
3. **Declared, then generated**: The schema declares the reference and its policy.
   Oracle generates the registration, the filters, and the client types. Services write
   no relationship code.
4. **Integrity belongs to the store**: Gorp checks existence and applies the policy. It
   never calls service code. A side effect that is data is a reference; one that is not
   follows the table's change stream.
5. **Ownership and placement are different fields**: `schematic.project` is the owner.
   `schematic.group` is where a toolbar shows it. No tree is stored as a tree.
6. **Row-conditional rules live in service guards**: A policy is per field. "Refuse when
   non-internal tasks remain" stays a `gorp.Delete.Guard`.
7. **No generic graph endpoint**: No Core or client API exposes a type-agnostic
   children, parents, or move operation.

## 4 Design

### 4.0 Two field kinds

A reference is a scalar key or a key array. The reverse direction is never declared; it
is a query against the reverse index.

| Relationship shape | Storage                                        | Reverse lookup                     |
| ------------------ | ---------------------------------------------- | ---------------------------------- |
| To-one             | `rack rack.Key` on device                      | devices where `rack == k`          |
| One-to-many        | The to-one on the child                        | same                               |
| Many-to-many       | `labels label.Key[]` on the side that edits it | ranges where `labels` contains `k` |
| Polymorphic        | `owner resource.ID` on status                  | statuses where `owner == id`       |

Prior art agrees: the foreign key on the child row (SQL, Ent, Prisma, Kubernetes
`ownerReferences`, Grafana `folder_uid`, Home Assistant `device_id`), a cardinality-many
ref for the many side (Datomic, Gel `multi link`), and a delete policy declared with the
reference (SQL `ON DELETE`, Gel `on target delete`, Datomic `:db/isComponent`). The
generic edge table (Neo4j, Zanzibar, OPC UA references) is the shape this RFC leaves.
The Console never asks "everything related to X"; every reader filters to one relation
and one endpoint type, which is a field query.

### 4.1 The schema declaration

`@ref` is a field domain. The target is inferred from the field type: `rack.Key`
resolves to the struct in `rack.oracle` whose `@key` field has that type. A
`resource.ID` field lists its allowed targets. Proposed syntax:

```
Device struct {
    rack   rack.Key {
        @ref on_delete restrict
    }
    parent Key? {
        @ref { on_delete cascade
               acyclic }
    }
    group  group.Key? {
        @ref on_delete restrict
    }
}

Range struct {
    parent Key? {
        @ref { on_delete cascade
               acyclic }
    }
    labels label.Key[] = [] {
        @ref on_delete detach
    }
}

Status struct {
    owner resource.ID {
        @ref { types task rack device channel
               on_delete cascade
               unique }
    }
}
```

Expressions:

- `on_delete cascade | restrict | detach`: Required. `detach` on a scalar sets the zero
  value, so the field must be optional or carry a default. `detach` on an array removes
  the element.
- `types ...`: Required on a `resource.ID` field. The tables the reference may point at.
- `unique`: At most one dependent per target, enforced by the reverse index on write.
- `acyclic`: A self-reference may not form a cycle, enforced by walking the chain on
  write.
- `target Name`: Only when type inference is ambiguous.

`@ref` implies `@filter` and `@index lookup`, so every reference gets a generated
`Match<Field>s` filter routed through its index (RFC 0034 §2.2.1). `@ontology type`
becomes `@resource type` and still generates the ID factories in every language.

### 4.2 Gorp

Every piece below is node-local and runs in the caller's transaction.

**Registry**: `gorp.DB` holds a schema registry. `OpenTable` registers the table's name,
key parser, and references; `Table.Close` unregisters. This is the first cross-table
state in Gorp: an explicit instance on the DB, populated at wiring time, as
`TableConfig.Indexes` attach today (`x/go/gorp/table.go:141`).

**Reference declaration**: `TableConfig.References` carries one `gorp.Reference` per
field. A reference owns a `LookupIndex` on the field. For arrays the index fans out one
entry per element, the multi-valued extractor RFC 0034 did not ship. The per-tx delta
overlay (`x/go/gorp/delta.go`) gives index probes read-your-own-writes, which cascade
and existence checks need. Oracle emits the declaration into the `newIndexes` block that
`retrieve.gen.go` already carries.

**Existence on write**: `Create.Exec` and `Update.Exec` resolve each reference field
through the registry and probe the target table inside the tx. A Pebble indexed batch
reads its own writes (`x/go/kv/pebblekv/pebblekv.go:123`) and the Aspen tx reads through
its batch (`aspen/internal/kv/tx.go:34`), so a rack and its device created in one tx
validate when the rack is written first. Checks are immediate, never deferred. `unique`
and `acyclic` run here.

**Policy on delete**: `Delete.Exec` runs guards, then asks the registry for every
reference targeting this table and probes each reverse index with the doomed keys.
`restrict` fails before any mutation. `cascade` deletes dependents through the same
path, with a visited set for self-references. `detach` updates the dependent rows. The
whole graph commits in the caller's tx. Gorp never calls a service.

**Polymorphic targets**: The registry is keyed by table name, the `@resource type`
string. A `resource.ID` reference probes the table named by `id.Type` and parses
`id.Key` with that table's key parser, today's generated `KeyFromOntologyID`, renamed
`KeyFromResourceID`.

**Index readiness**: Indexes populate asynchronously at open. Existence and delete paths
wait on `Table.WaitForIndexes`. A sticky populate failure falls back to a scan, as
`Retrieve` does today (`x/go/gorp/retrieve.go:246`).

### 4.3 The `resource` package

`core/pkg/service/resource` holds `resource.ID`, `resource.Type`, and the generated type
enum. Search, access enforcement, export addressing, panel tabs, policy objects, and
status owners use it unchanged in shape. Nothing else from `core/pkg/service/ontology`
survives.

### 4.4 Every relationship, mapped

| Today                                                      | Field                                              | Policy             |
| ---------------------------------------------------------- | -------------------------------------------------- | ------------------ |
| `rack parent_of device` + `device.rack`                    | `device.rack rack.Key` (exists)                    | restrict           |
| `device parent_of device` (chassis)                        | `device.parent device.Key?`                        | cascade, acyclic   |
| `task.rack`                                                | exists                                             | cascade (see §4.7) |
| `arc parent_of task` + `config.arc_key`                    | `arc/task.Config.arc_key` (exists)                 | cascade            |
| task config store keyed by task                            | `config.task task.Key` (the key)                   | cascade            |
| `range parent_of range`                                    | `range.parent ranger.Key?`                         | cascade, acyclic   |
| `range labeled_by label`                                   | `range.labels label.Key[]`                         | detach             |
| `status labeled_by label`                                  | `status.labels label.Key[]`                        | detach             |
| status key `type:key`                                      | `status.owner resource.ID` (§4.6)                  | cascade, unique    |
| `range parent_of task` (snapshot)                          | `task.range ranger.Key?`                           | cascade            |
| `range parent_of schematic` (snapshot)                     | `schematic.range ranger.Key?`                      | cascade            |
| `project parent_of` schematic, lineplot, log, table, panel | `X.project project.Key?`                           | cascade            |
| `group parent_of` X                                        | `X.group group.Key?`                               | restrict           |
| `group parent_of group`, project, rack                     | `group.parent resource.ID?` (§4.5)                 | cascade, acyclic   |
| `role parent_of user`                                      | `user.roles role.Key[]`                            | detach             |
| `role parent_of policy`                                    | `policy.role role.Key`                             | cascade            |
| alias `range---channel` key                                | references on both key components                  | cascade both       |
| kv `range<--->key` key                                     | reference on the `range` key component             | cascade            |
| `Tasks`, `Statuses`, `Views` groups                        | deleted; these toolbars are lists                  |                    |
| `Metrics` group                                            | an ordinary channel group the metrics service owns |                    |

Composite keys stay. Alias and key-value pairs are natural keys whose pair is the
uniqueness constraint; a reference on a key component gives them cascade without a UUID
migration.

### 4.5 Groups

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
`schematic_symbol`) and is constant down a nesting chain; a service guard checks a child
group's scope against its parent's. A toolbar root is "groups of my scope with no
parent" plus "resources of my type with no group". A project tree is "groups whose
parent is the project" plus "visualizations whose project is P and group is nil".
Grouping devices under a rack sets `group.parent = rack:1`; the device guard checks that
`device.rack` matches the group's rack ancestor. No builtin root group exists. The seven
created by `Group.CreateOrRetrieve` (`core/pkg/service/channel/service.go:161` and
siblings) are deleted by the migration in §4.11.

Group delete is `restrict` on members, which keeps today's "cannot delete a group with
children" (`core/pkg/service/group/service.go:206`). The Console ungroups before it
deletes.

### 4.6 Statuses

Status keys embed the owner (`rack.StatusKey`, `core/pkg/service/rack/rack.go:22`). The
Driver recomputes them at more than twenty sites (`driver/common/status.h:72`), and
`details` carries a second copy of the rack and task keys. The key becomes a UUID.
`owner resource.ID` is a unique cascade reference. The status set endpoint accepts an
owner and upserts through the unique index, so the Driver sends `{owner: task_id, ...}`
and keeps its idempotent write with no round trip. The copies leave `details`.
Calculation statuses take the channel as owner.

### 4.7 Services

Every `ontology.Writer` call is deleted; 22 services register with the ontology and each
defines resources and edges by hand. Generated `Match<Field>s` filters route through the
reference index, so `device.retrieve({rack})`, `range.retrieve({parent})`,
`schematic.retrieve({project})`, `user.retrieve({role})`, and
`group.retrieve({parent, scope})` cost one index probe. Range delete drops its BFS
(`core/pkg/service/ranger/writer.go:174`); cascade walks `parent`. Project delete
cascades.

Guards keep row-conditional rules. Rack delete refuses while non-internal tasks remain
and cascades the internal ones: `task.rack` is `cascade`, and the refusal is a
`gorp.Delete.Guard` in the rack service, as `embeddedGuard` is today
(`core/pkg/api/rack/rack.go:171`).

Side effects that are not data follow the change stream. Channel delete calls Cesium
storage delete after the row delete, outside the tx
(`core/pkg/service/channel/writer.go:643`). It moves to a subscriber on the channel
table's delete stream, paired with the startup reclaim sweep from RFC 0042 §7. A
cascaded channel delete then takes the same path as a direct one.

### 4.8 API and clients

`/ontology/retrieve`, `add-children`, `remove-children`, `move-children`
(`core/pkg/api/ontology/ontology.go`), and the `/ontology/*-group` endpoints are
deleted. Group create, rename, and delete become `/group/*`. Moving a resource into a
group is an update of its `group` field through its own service; Console drag-and-drop
issues one typed update per resource type instead of `moveChildren`.

The TypeScript client deletes `client/ts/src/ontology` except the ID payload, which
moves to `resource`. The relationship cache, the two `RelationshipIndexes`, and the
`sy_ontology_*` listeners go with it. Each domain client keeps a per-table query cache
with lookup indexes (`client/ts/src/query/indexes.ts`); reference fields are ordinary
indexed fields there, so `ranges.retrieve({parent})` and its watcher match on
`range.parent` with no relationship state.

The Console tree stays generic over a root `resource.ID`, but children come from the
item types. Each `Tree.createItem` gains a `children` resolver: the rack item lists
devices whose `rack` is the rack, the group item lists child groups and grouped
resources of its scope, the role item lists users whose `roles` contain the role, the
range item lists child ranges. Toolbars keep `Tree.Tree root={...}`
(`console/src/feature/channel/Toolbar.tsx:63`) with a synthetic scope root instead of a
builtin group found by name (`pluto/src/device/queries.ts:125`).

Python deletes `synnax.ontology`; `Range.children` becomes a `parent` filter. C++
renames `ontology::ID` to `resource::ID`; the Driver's status writes send an owner
(§4.6).

### 4.9 Search, access, import and export

Search registers services by `resource.Type` and indexes flat documents
(`core/pkg/service/search/search.go:36`); it never read an edge and changes only its
import. Access enforcement matches `resource.ID` exactly or by type wildcard
(`core/pkg/service/access/rbac/service.go:269`). Its one traversal, subject to role to
policy (`core/pkg/service/access/rbac/policy/retriever.go:30`), becomes two index
probes: `user.roles`, then `policy.retrieve({role})`. The export request stays a
`resource.ID`. The project bundle walk (`core/pkg/service/project/bundle.go:157`)
becomes typed queries: groups by parent, visualizations by project and group. The
visited set goes away because a visualization has one group.

### 4.10 Signals

`sy_ontology_relationship_set`, `sy_ontology_relationship_delete`,
`sy_ontology_resource_set`, and `sy_ontology_resource_delete` are deleted. Every
Gorp-backed table publishes its own set and delete channels through
`signals.PublishFromGorp`, and a set signal carries the row, so a reference change
arrives as an entity change. A cascade emits one delete signal per dependent row.

### 4.11 Migration

One startup migration per table derives its reference fields from the old edges. It
reads the relationship table through `gorp.WrapReader[string, Relationship]`, as the
project v1 migration does (`core/pkg/service/project/versions/v1/migrate.go:170`), so
the relationship type stays frozen in `ontology/versions/v0` and its chain ends with a
tombstone per RFC 0053.

Derivations, from `parent` edges unless noted:

- `device.parent` from device-to-device edges. Every `group` field from group-to-X edges
  whose group is not a builtin root.
- `range.parent`, `task.range`, `schematic.range` from range edges. `range.labels` and
  `status.labels` from `labeled_by` edges.
- `X.project` from project-to-X edges, or through the group chain to the project.
- `group.parent` from the edge into the group: a builtin root maps to nil, a project or
  group to its ID. `group.scope` from the builtin root at the top of the chain.
- `user.roles` and `policy.role` from role edges.
- `status.key` becomes a UUID and `status.owner` is parsed from the old key. The
  PagerDuty alert config that holds status keys is rewritten in the same migration.

A final step in the service layer, after every service has opened, deletes both ontology
table prefixes and the builtin groups, once, behind a marker key. Rows that resolve to
nothing, such as a visualization with neither project nor range, are deleted; they are
unreachable today. Migrations run per node with no coordination (RFC 0033 §4.2.5), and
every derivation is deterministic.

### 4.12 Kill list

- `core/pkg/service/ontology` except the frozen `versions/v0` types;
  `core/pkg/service/ontology/signals`.
- `core/pkg/api/ontology`, the `/ontology/*` routes in `core/pkg/api/layer.go`, and the
  ontology access checks.
- Every `ontology.Writer` call, every `OntologyID` helper that only feeds an edge write,
  every `ontology.Service` implementation and `RegisterService` call in
  `core/pkg/service/**`.
- `ranger.Writer.Delete`'s BFS, `schematic.findParentProject`,
  `ranger.RetrieveParentKey`, `symbol.rescueStrays`,
  `metrics.maybeDefineGroupRelationship`, every `Group.CreateOrRetrieve` root group.
- `Status.Details.{Task,Rack,Device}`, `rack.StatusKey`, `calculation.StatusKey`, the
  C++ `status_key` functions.
- `client/ts/src/ontology` except the ID payload, `Cache.parentID`, every `parents[0]`
  reader, `client/py/synnax/ontology`, `client/cpp/ontology`.
- The `console/src/platform/tree` children query, `moveChildren`, and the by-name
  builtin group lookups in `pluto/src/{user,project,device}/queries.ts`.

## 5 Implementation phases

- **Phase 1: Gorp references and Oracle `@ref`.** The registry, `gorp.Reference`, the
  multi-valued index, existence, unique, acyclic, and the three policies, tested in
  `x/go/gorp`. The `resource` package and the `@ontology type` to `@resource type`
  rename. Oracle emits reference registration and index-routed filters. Nothing in the
  Core declares a reference yet. Earned by risk isolation: the store change is reviewed
  apart from any behavior change.
- **Phase 2: Domain cutovers.** One pull request per domain: add its fields and
  migration, delete its ontology writes and readers, update its TypeScript, Python, and
  Console code. The ontology keeps serving the domains not yet cut over, so every
  boundary is green. Order: labels and statuses (arrays, UUID status keys, the Driver
  owner write); ranges (parent, snapshots, alias and key-value references); groups,
  channels, and the Console tree resolvers; racks, devices, tasks, and Arc; projects,
  visualizations, panels, and bundle export; users, roles, and policies.
- **Phase 3: Delete the ontology.** The package, the API, the signals, the client
  modules, the generic children query, and the table-drop step. Earned by reviewability:
  a pure deletion.
- **Phase 4: Nested references.** `@ref` on a field inside a record or array
  (`lineplot.channels`, `log.channels`, task config `device` and `channel` keys, `panel`
  tab resources, `policy.objects`). Oracle emits a path extractor; `detach` removes the
  element or nulls the field. FoundationDB's nested index key expressions are the
  precedent.
- **Phase 5: Channel references.** `index channel.Key` as a `restrict` reference and a
  stored, compiler-derived `requires channel.Key[]` as a `restrict` reference. The
  leaseholder stays in the key; a node is not a table.

Phases 4 and 5 are named now so the declaration language is designed for them, and
sequenced last because nothing earlier depends on them.

## 6 What this RFC does not cover

- Cross-leaseholder atomicity on Aspen. A Gorp tx still splits into one batch per
  leaseholder (`aspen/internal/kv/tx.go:29`). A reference is atomic with its row on
  every store; a cascade that spans leaseholders keeps today's guarantee.
- Storage-first channel create atomicity (RFC 0042 §7).
- Search index structure and the Console search palette, which change only an import.
- Console session state, which never held relationships.

## 7 Resolved decisions

1. **Fields, not edges, and never both**: Keeping the edge table as the single copy,
   with fields resolved from it, was rejected. It reverses RFC 0056 §6.5, which made
   `task.rack` a field so a rack change moves no edges; it puts the record and its
   relationship in different Aspen batches; and no reader needs an untyped traversal.
   Cost: a fact many types share, such as group membership, is a field on each of them.
2. **Delete both ontology tables**: The resource table alone was SY-4804's target.
   Deleting only it keeps the dual-write. Both go.
3. **Cascade never calls service code**: A per-table delete hook, so a cascaded channel
   delete could free Cesium storage synchronously, was rejected. Ent (issue #2031) shows
   hooks and store-level cascade disagreeing on which runs. Data side effects are
   references; the rest follows the change stream.
4. **Groups take a polymorphic parent and a scope**: A field per container type
   (`project`, `rack`) was rejected because each new container adds a field. Per-domain
   group tables were rejected as duplication. Cost: scope consistency is a service
   guard, not a type.
5. **No stored tree flag**: A `tree` marker on references was rejected. Placement is the
   `group` field; ownership is the owning reference; the toolbar decides what it shows.
6. **UUID status keys**: Owner-derived string keys were rejected. The key encoding is a
   hidden reference, and the unique index gives the Driver the same idempotent upsert by
   owner.
7. **Composite keys stay for alias and key-value**: A UUID migration was rejected. The
   pair is the identity, and a reference on a key component gives cascade.
8. **Immediate existence checks**: Deferred constraints (SQL `INITIALLY DEFERRED`) for
   child-before-parent writes were rejected. Parent-first ordering is a small rule;
   deferral adds a commit-time phase to Gorp.
9. **Restrict on group members**: Cascade was rejected. Deleting a folder must not
   delete channels, and the Console ungroups first.
10. **No generic endpoint of any kind**: A typed `/relationship` façade was rejected. It
    reintroduces the untyped graph. Relationships are managed only through per-service
    endpoints.
11. **Orphans are deleted in migration**: Preserving unreachable rows was rejected. They
    are invisible today, and a required reference cannot hold nothing.

## 8 Open questions

- Whether `group.scope` can be dropped once every group has a parent or a member,
  leaving only root groups that need it.
- The `@ref` block grammar: newline-separated expressions, as `@ts { }` uses today, or a
  delimiter.
- The device chassis policy: `cascade` or `detach`. Discovered subdevices argue for
  cascade, hand-created ones for detach.
- The reclaim sweep cadence for Cesium storage after a cascaded channel delete.
