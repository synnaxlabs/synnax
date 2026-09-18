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

A relationship between two records is stored in three different ways today: as a field
on a row, as an edge in the ontology relationship table, or as part of a composite key.
Some facts are stored two ways at once. The ontology resource table stores a second copy
of every record's identity. Nothing enforces that a reference points at a row that
exists, and nothing deletes a dependent when its owner goes away. Each service
hand-writes the edge management, and each writes it differently.

This RFC removes the ontology. A relationship is a field on exactly one table, declared
in the Oracle schema with `@ref`. Gorp learns what a reference is: it validates that the
target exists on write, keeps a reverse index on the field, and applies a declared
policy (`cascade`, `restrict`, `detach`) when the target is deleted, all inside the
caller's transaction. Oracle generates the declaration into the table, the filters, and
the client types. The `/ontology/*` API, the client relationship caches, the two
ontology tables, and the ontology signal channels are deleted. The only survivor is the
polymorphic `{type, key}` identifier, which moves to a small `resource` package.

## 1 Motivation

RFC 0005 introduced the ontology in 2022 so that authorization and navigation could
relate records without the owning services knowing about each other. RFC 0026 §1.1.10
named the result: three competing patterns for one job. RFC 0042 §7 deferred the fix to
this RFC. The evidence is now concrete.

- **One relationship type carries eight meanings**: Only two edge types exist, `parent`
  (`core/pkg/service/ontology/versions/v0/relationship.go:26`) and `labeled_by`
  (`core/pkg/service/label/relationship.go:18`). `parent` means group membership,
  project ownership, rack ownership, device chassis, range nesting, range snapshot, Arc
  binding, and role assignment. Every reader recovers the meaning by filtering on the
  endpoint types, in Go (`core/pkg/service/ranger/service.go:176`,
  `core/pkg/service/schematic/writer.go:96`) and in TypeScript
  (`client/ts/src/ranger/client.ts:336`, `client/ts/src/arc/client.ts:116`).
- **Identity is stored twice**: The resource table persists `Resource{ID}` and nothing
  else (`core/pkg/service/ontology/writer.go:34`). Name and data are filled at read time
  by the owning service. The table exists so edge writes can check that both ends exist
  and so a type can be listed. SY-4804 traced the entry-versus-resource orphan window to
  this second copy and rejected per-call-site pairing as a fix.
- **Integrity is hand-written and leaks**: Rack, role, policy, and range alias deletes
  drop the row and leave the ontology resource behind
  (`core/pkg/service/rack/writer.go:127`,
  `core/pkg/service/access/rbac/role/writer.go:63`,
  `core/pkg/service/access/rbac/policy/writer.go:59`,
  `core/pkg/service/ranger/alias/writer.go:62`). Project delete orphans every schematic,
  plot, log, and table under it (`core/pkg/service/project/writer.go:94`). Range delete
  leaves aliases and key-value pairs behind. Every Arc, internal task, copied task, and
  alias has no parent. Migrations write both ontology tables through raw Gorp writers
  and skip the cycle and existence checks
  (`core/pkg/service/panel/versions/v0/composition.go:198`).
- **The one true duplicate drifts**: `device.rack` is persisted and a
  `rack parent_of device` edge is written from it
  (`core/pkg/service/device/writer.go:82`). The rack delete precondition reads the field
  (`core/pkg/api/rack/rack.go:199`) and the tree reads the edge.
- **Readers assume a tree, the model is a DAG**: Fourteen readers take the first parent
  (`client/ts/src/ontology/store.ts:66`, `client/ts/src/task/client.ts:100`,
  `core/pkg/api/device/device.go:168`). The project bundle walk needs a visited set
  because a resource can have two parents (`core/pkg/service/project/bundle.go:124`).
- **The generic graph is slow in the ways that matter**: Deleting a resource scans the
  whole relationship table once per ID to find incoming edges
  (`core/pkg/service/ontology/writer.go:194`). Children traversal ignores the reverse
  index and prefix-scans (`core/pkg/service/ontology/retrieve.go:244`). The limit
  applies after the traversal materializes every child (SY-4763). The TypeScript cache
  mirrors every edge and scans it on every delete (SY-4754).

## 2 Vocabulary

- **Reference**: A field on a row that holds the key of a row in another table, or in
  the same table. The only representation of a relationship after this RFC.
- **To-one reference**: A scalar key field. `device.rack`, `range.parent`.
- **To-many reference**: A key array field. `range.labels`, `user.roles`. A many-to-many
  relationship is a to-many reference on the side that edits it.
- **Reverse lookup**: The rows whose reference field holds a given key. A one-to-many
  relationship is the reverse lookup of a to-one reference.
- **Target**: The table a reference points at.
- **Dependent**: A row that references a target row.
- **Policy**: What Gorp does to dependents when a target row is deleted. One of
  `cascade`, `restrict`, `detach`.
- **Polymorphic reference**: A reference whose target table varies per row, held as a
  `resource.ID`.
- **Resource ID**: The `{type, key}` pair that identifies any row in any table.

## 3 Principles

1. **One copy**: A relationship is stored in exactly one field of exactly one table.
   Never a field and an edge, never a field and a key encoding.
2. **The row is the unit of atomicity**: A reference lives on the row that depends on
   it, so the record and its relationship commit together on every store, including
   Aspen.
3. **Declared, then generated**: The schema declares a reference and its policy. Oracle
   generates the Gorp registration, the filters, and the client types. Services write no
   relationship code.
4. **Integrity belongs to the store**: Gorp checks that a target exists and applies the
   delete policy. It never calls service code. Side effects that are data are
   references; side effects that are not data follow the table's change stream.
5. **Ownership and placement are different fields**: `schematic.project` says who owns
   the schematic. `schematic.group` says where a toolbar shows it. Nothing about a tree
   is stored as a tree.
6. **Row-conditional rules live in service guards**: A policy is per field. A rule like
   "refuse when non-internal tasks remain" stays a `gorp.Delete.Guard`.
7. **No generic graph endpoint**: Neither the Core API nor any client exposes a
   type-agnostic children, parents, or move operation.

## 4 Design

### 4.0 Two field kinds

A reference is a scalar key or a key array. Nothing else. The reverse direction is never
declared; it is a query against the reverse index.

| Relationship shape | Storage                                        | Reverse lookup                     |
| ------------------ | ---------------------------------------------- | ---------------------------------- |
| To-one             | `rack rack.Key` on device                      | devices where `rack == k`          |
| One-to-many        | The to-one on the child, read from the target  | same                               |
| Many-to-many       | `labels label.Key[]` on the side that edits it | ranges where `labels` contains `k` |
| Polymorphic        | `owner resource.ID` on status                  | statuses where `owner == id`       |

Prior art converges on this split: the foreign key on the child row (SQL, Ent, Prisma,
Kubernetes `ownerReferences`, Grafana `folder_uid`, Home Assistant `device_id`), a
cardinality-many ref attribute for the many side (Datomic, Gel `multi link`), and a
delete policy declared where the reference is declared (SQL `ON DELETE`, Gel
`on target delete`, Datomic `:db/isComponent`). The generic edge table (Neo4j, Zanzibar,
OPC UA references) is the shape this RFC leaves. We deviate from it because the Console
never needs "everything related to X"; every reader already filters to one relation and
one endpoint type, which is a field query.

### 4.1 The schema declaration

`@ref` is a field domain. The target table is inferred from the field's type: `rack.Key`
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

- `on_delete cascade | restrict | detach`: Required. `detach` on a scalar sets the field
  to its zero value, so the field must be optional or carry a default. `detach` on an
  array removes the element.
- `types ...`: Required on a `resource.ID` field. Lists the tables the reference may
  point at.
- `unique`: At most one dependent per target. Enforced by the reverse index on write.
- `acyclic`: A self-reference may not form a cycle. Enforced by walking the chain on
  write.
- `target Name`: Only when type inference is ambiguous.

`@ref` implies `@filter` and `@index lookup`, so every reference gets a generated
`Match<Field>s` filter routed through its index (RFC 0034 §2.2.1). `@ontology type`
becomes `@resource type` and keeps generating the ID factories in every language.

### 4.2 Gorp

Gorp gains one concept, the reference, and three behaviors around it. Every piece is
node-local and lives in the caller's transaction.

**Registry**: `gorp.DB` holds a schema registry. `OpenTable` registers the table's name,
key parser, and references; `Table.Close` unregisters. This is the first cross-table
state in Gorp. It is an explicitly constructed instance on the DB, populated at wiring
time, matching how `TableConfig.Indexes` attach today (`x/go/gorp/table.go:141`).

**Reference declaration**: `TableConfig.References` carries one `gorp.Reference` per
declared field. A reference owns a `LookupIndex` on the field. For arrays, the index
fans out one entry per element; this is the multi-valued extractor RFC 0034 did not
ship. The per-tx delta overlay (`x/go/gorp/delta.go`) already gives read-your-own-writes
for index probes, which cascade and existence checks need. Oracle generates the
declaration into the `newIndexes` block that `retrieve.gen.go` already emits.

**Existence on write**: `Create.Exec` and `Update.Exec` resolve each reference field
through the registry and probe the target table inside the tx. A Pebble indexed batch
reads its own writes (`x/go/kv/pebblekv/pebblekv.go:123`), and the Aspen tx reads
through its underlying batch (`aspen/internal/kv/tx.go:34`), so a rack and its device
created in one tx validate when the rack is written first. Checks are immediate, never
deferred. `unique` and `acyclic` run here too.

**Policy on delete**: `Delete.Exec` runs guards, then asks the registry for every
reference that targets this table and probes each reverse index with the doomed keys.
`restrict` fails before any mutation. `cascade` deletes the dependents through the same
path, with a visited set for self-references. `detach` updates the dependent rows. The
whole graph commits in the caller's tx. Gorp never calls a service.

**Polymorphic targets**: The registry is keyed by table name, which is the
`@resource type` string. A `resource.ID` reference probes the table named by `id.Type`
and parses `id.Key` with that table's key parser. Oracle already generates that parser
as `KeyFromOntologyID`; it becomes `KeyFromResourceID`.

**Index readiness**: Indexes populate asynchronously at open. Existence and delete paths
wait on `Table.WaitForIndexes`. A sticky populate failure falls back to a scan, as
`Retrieve` does today (`x/go/gorp/retrieve.go:246`).

### 4.3 The `resource` package

`core/pkg/service/resource` holds `resource.ID`, `resource.Type`, and the generated type
enum. Search, access enforcement, import and export addressing, panel tabs, policy
objects, and status owners keep using it unchanged in shape. Nothing else from
`core/pkg/service/ontology` survives.

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
| alias `range---channel` key                                | key components declared as references              | cascade both       |
| kv `range<--->key` key                                     | `range` key component declared as reference        | cascade            |
| `Tasks`, `Statuses`, `Views` groups                        | deleted; these toolbars are lists                  |                    |
| `Metrics` group                                            | an ordinary channel group the metrics service owns |                    |

Composite keys stay. Alias and key-value pairs are natural keys and the pair is the
uniqueness constraint; a reference declared on a key component gives them cascade
without a UUID migration.

### 4.5 Groups

A group is a placement container. The question is what contains a group: a toolbar root,
a project, another group, or, in future, a rack. Adding a field per container type does
not scale. Instead:

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
`schematic_symbol`). It is constant down a nesting chain, so a service guard checks that
a child group's scope equals its parent's. A toolbar's root is "groups of my scope with
no parent" plus "resources of my type with no group". A project's tree is "groups whose
parent is the project" plus "visualizations whose project is P and group is nil".
Grouping devices under a rack is `group.parent = rack:1`, and the device service guard
checks that `device.rack` equals the group's rack ancestor. No builtin root group
exists; the seven created today by `Group.CreateOrRetrieve`
(`core/pkg/service/channel/service.go:161` and siblings) are deleted with the migration
in §4.11.

Group delete is `restrict` on members, which keeps today's "cannot delete a group with
children" (`core/pkg/service/group/service.go:206`). Ungroup in the Console already
moves members first.

### 4.6 Statuses

Status keys embed the owner (`rack.StatusKey` at `core/pkg/service/rack/rack.go:22`),
the Driver recomputes them from the task or device at more than twenty sites
(`driver/common/status.h:72`), and `details` carries a second copy of the rack and task
keys. The key becomes a UUID. `owner resource.ID` is a unique cascade reference. The
status set endpoint accepts an owner and upserts through the unique index, so the Driver
sends `{owner: task_id, ...}` and keeps its idempotent write with no round trip. The
denormalized keys leave `details`. Calculation statuses take the channel as owner.

### 4.7 Services

Every `ontology.Writer` call in a service is deleted: 22 services register with the
ontology today and every one defines resources and edges by hand (§1). What replaces
them is generated. `Match<Field>s` filters route through the reference index, so
`device.retrieve({rack})`, `range.retrieve({parent})`, `schematic.retrieve({project})`,
`user.retrieve({role})`, and `group.retrieve({parent, scope})` cost one index probe.
Range delete drops its BFS (`core/pkg/service/ranger/writer.go:174`); cascade walks the
`parent` reference. Project delete gains the cascade it never had.

Guards keep the row-conditional rules. Rack delete refuses while non-internal tasks
remain and cascades the internal ones; the policy on `task.rack` is `cascade`, and the
refusal is a `gorp.Delete.Guard` in the rack service, as `embeddedGuard` is today
(`core/pkg/api/rack/rack.go:171`).

Side effects that are not data follow the change stream. Channel delete calls Cesium
storage delete after the row delete and outside the tx today
(`core/pkg/service/channel/writer.go:643`). It moves to a subscriber on the channel
table's delete stream, paired with the startup reclaim sweep RFC 0042 §7 already
specifies. A cascaded channel delete is then cleaned up by the same path as a direct
one.

### 4.8 API and clients

The `/ontology/retrieve`, `add-children`, `remove-children`, and `move-children`
endpoints (`core/pkg/api/ontology/ontology.go`) and the `/ontology/*-group` endpoints
are deleted. Group create, rename, and delete become `/group/*`. Moving a resource into
a group is an update of its `group` field through its own service; the Console's
drag-and-drop calls one typed update per resource type instead of `moveChildren`.

The TypeScript client deletes `client/ts/src/ontology` except the ID payload, which
moves to `resource`. The relationship cache, the two `RelationshipIndexes`, and the
`sy_ontology_*` listeners go with it. Each domain client already keeps a per-table query
cache with lookup indexes (`client/ts/src/query/indexes.ts`); the reference fields are
ordinary indexed fields there, so `ranges.retrieve({parent})` and its watcher match on
`range.parent` with no relationship state.

The Console tree component stays generic over a root `resource.ID`, but its children
come from the item types, not from a graph query. Each `Tree.createItem` gains a
`children` resolver: the rack item lists devices whose `rack` is the rack, the group
item lists child groups and grouped resources of its scope, the role item lists users
whose `roles` contain the role, the range item lists child ranges. The toolbars keep
their `Tree.Tree root={...}` shape (`console/src/feature/channel/Toolbar.tsx:63`) with a
synthetic scope root instead of a builtin group looked up by name
(`pluto/src/device/queries.ts:125`).

Python deletes `synnax.ontology`; `Range.children` becomes a `parent` filter. C++
renames `ontology::ID` to `resource::ID`, and the Driver's status writes send an owner
(§4.6).

### 4.9 Search, access, import and export

Search registers services by `resource.Type` and indexes flat documents
(`core/pkg/service/search/search.go:36`); it never read an edge and changes only its
import. Access enforcement matches `resource.ID` values exactly or by type wildcard
(`core/pkg/service/access/rbac/service.go:269`). The only traversal, subject to role to
policy (`core/pkg/service/access/rbac/policy/retriever.go:30`), becomes two index
probes: `user.roles`, then `policy.retrieve({role})`. The export request stays a
`resource.ID`. The project bundle walk (`core/pkg/service/project/bundle.go:157`)
becomes typed queries: groups by parent, visualizations by project and group. The
visited set goes away because a visualization has one group.

### 4.10 Signals

`sy_ontology_relationship_set`, `sy_ontology_relationship_delete`,
`sy_ontology_resource_set`, and `sy_ontology_resource_delete` are deleted. Every
Gorp-backed table already publishes its own set and delete channels through
`signals.PublishFromGorp`, and a set signal carries the row, so a reference change
arrives as an entity change. A cascade emits one delete signal per dependent row, which
is what the client caches need and what they never got from the ontology.

### 4.11 Migration

One startup migration per table derives its reference fields from the old edges. A
migration reads the relationship table through `gorp.WrapReader[string, Relationship]`,
which the project v1 migration already does
(`core/pkg/service/project/versions/v1/migrate.go:170`), so the relationship type is
frozen in `ontology/versions/v0` and its chain ends with a tombstone per RFC 0053.

Derivations, all from `parent` edges unless noted:

- `device.parent` from device-to-device edges. `channel.group`, `device.group`,
  `schematic.group`, and the other `group` fields from group-to-X edges whose group is
  not a builtin root.
- `range.parent`, `task.range`, `schematic.range` from range edges. `range.labels` and
  `status.labels` from `labeled_by` edges.
- `X.project` from project-to-X edges, or through the group chain to the project.
- `group.parent` from the edge into the group: a builtin root maps to nil, a project or
  group maps to its ID. `group.scope` from the builtin root at the top of the chain.
- `user.roles` and `policy.role` from role edges.
- `status.key` becomes a UUID and `status.owner` is parsed from the old key. The
  PagerDuty alert config that holds status keys is rewritten by the same migration.

A final step in the service layer, after every service has opened, deletes both ontology
table prefixes and the builtin groups. It runs once behind a marker key. Rows that
resolve to nothing, such as a visualization with neither project nor range, are deleted;
they are unreachable today. Migrations run per node with no coordination, as RFC 0033
§4.2.5 requires, and every derivation is deterministic.

### 4.12 Kill list

- `core/pkg/service/ontology` except the frozen `versions/v0` types, and
  `core/pkg/service/ontology/signals`.
- `core/pkg/api/ontology`, the `/ontology/*` routes in `core/pkg/api/layer.go`, and the
  ontology access checks.
- Every `ontology.Writer` call, every `OntologyID` helper that only feeds an edge write,
  `ontology.Service` implementation, and `RegisterService` call in
  `core/pkg/service/**`.
- `ranger.Writer.Delete`'s BFS, `schematic.findParentProject`,
  `ranger.RetrieveParentKey`, `symbol.rescueStrays`,
  `metrics.maybeDefineGroupRelationship`, and every `Group.CreateOrRetrieve` root group.
- `Status.Details.{Task,Rack,Device}`, `rack.StatusKey`, `calculation.StatusKey`, and
  the C++ `status_key` functions.
- `client/ts/src/ontology` except the ID payload, `Cache.parentID`, every `parents[0]`
  reader, `client/py/synnax/ontology`, `client/cpp/ontology`.
- `console/src/platform/tree` children query, `moveChildren`, and the by-name builtin
  group lookups in `pluto/src/{user,project,device}/queries.ts`.

## 5 Implementation phases

- **Phase 1: Gorp references and Oracle `@ref`.** The registry, `gorp.Reference`, the
  multi-valued index, existence, unique, acyclic, and the three policies, with tests in
  `x/go/gorp`. The `resource` package and the `@ontology type` to `@resource type`
  rename. Oracle emits reference registration and index-routed filters. Nothing in the
  Core declares a reference yet. Earned by risk isolation: the store change lands and is
  reviewed apart from any behavior change.
- **Phase 2: Domain cutovers.** One pull request per domain, each adding its fields and
  migration, deleting its ontology writes and readers, and updating its TypeScript,
  Python, and Console code in the same change. The ontology keeps serving the domains
  not yet cut over, so every boundary is green. Order: labels and statuses (arrays, UUID
  status keys, the Driver owner write); ranges (parent, snapshots, alias and key-value
  references); groups, channels, and the Console tree resolvers; racks, devices, tasks,
  and Arc; projects, visualizations, panels, and bundle export; users, roles, and
  policies.
- **Phase 3: Delete the ontology.** The package, the API, the signals, the client
  modules, the generic children query, and the final table-drop step. Earned by
  reviewability: a pure deletion with no behavior to review.
- **Phase 4: Nested references.** `@ref` on a field inside a record or array
  (`lineplot.channels`, `log.channels`, task config `device` and `channel` keys, `panel`
  tab resources, `policy.objects`). Oracle emits a path extractor, and `detach` removes
  the element or nulls the field. FoundationDB's nested index key expressions are the
  precedent.
- **Phase 5: Channel references.** `index channel.Key` as a `restrict` reference and a
  stored, compiler-derived `requires channel.Key[]` as a `restrict` reference. The
  leaseholder stays in the key because a node is not a table.

Phases 4 and 5 are named so the declaration language is designed for them now, and are
sequenced last because nothing in Phases 1 to 3 depends on them.

## 6 What this RFC does not cover

- Cross-leaseholder atomicity on Aspen. A Gorp tx still splits into one batch per
  leaseholder (`aspen/internal/kv/tx.go:29`). A reference on a row is atomic with its
  row on every store; a cascade that spans leaseholders keeps today's guarantee.
- Storage-first channel create atomicity (RFC 0042 §7).
- Search index structure and the Console search palette, which change only their import.
- Console session state, which never held relationships.

## 7 Resolved decisions

1. **Fields, not edges, and never both**: An edge table was kept and considered as the
   single copy with fields resolved from it. Rejected: it reverses RFC 0056 §6.5, which
   just made `task.rack` a field so a rack change moves no edges; it puts the record and
   its relationship in different Aspen batches; and the Console never needs an untyped
   traversal. The trade is real: a fact that many types share, such as group membership,
   is now a field on every type that has it.
2. **Delete both ontology tables**: The resource table was the fix's target from
   SY-4804. Deleting only it and keeping edges leaves the dual-write. Both go.
3. **Cascade never calls service code**: A per-table delete hook was considered so a
   cascaded channel delete could free Cesium storage synchronously. Rejected: Ent's
   experience (issue #2031) is that hooks and store-level cascade disagree on which
   runs. Data side effects are references; the rest follows the change stream.
4. **Groups take a polymorphic parent and a scope**: A field per container type
   (`project`, `rack`) was rejected because each new container adds a field. Per-domain
   group tables were rejected as duplication. The trade is real: scope consistency is a
   service guard, not a type.
5. **No stored tree flag**: A `tree` marker on references was rejected. Placement is the
   `group` field; ownership is the owning reference; the toolbar decides what it
   renders.
6. **UUID status keys**: Owner-derived string keys were considered because the Driver
   computes them. Rejected: the key encoding is a hidden reference. The unique index
   gives the Driver the same idempotent upsert by owner.
7. **Composite keys stay for alias and key-value**: A UUID migration was considered.
   Rejected: the pair is the identity, and a reference on a key component gives cascade.
8. **Immediate existence checks**: Deferred constraints (SQL `INITIALLY DEFERRED`) were
   considered for child-before-parent writes in one tx. Rejected: parent-first ordering
   is a small rule and deferral adds a commit-time phase to Gorp.
9. **Restrict on group members**: Cascade was considered. Rejected: deleting a folder
   must not delete channels, and the Console already ungroups first.
10. **No generic endpoint of any kind**: A typed `/relationship` façade was considered.
    Rejected: a façade reintroduces the untyped graph, and relationships are managed
    only through per-service endpoints.
11. **Orphans are deleted in migration**: Preserving unreachable rows was considered.
    Rejected: they are invisible today and a required reference cannot hold nothing.

## 8 Open questions

- Whether `group.scope` can be dropped once every group has a non-nil `parent` or a
  member, which would leave only root groups needing it.
- The exact `@ref` block grammar: whether expressions separate by newline, as `@ts { }`
  does today, or by a delimiter.
- Whether Phase 2's device chassis policy is `cascade` or `detach`; discovered
  subdevices argue for cascade, hand-created ones for detach.
- The reclaim sweep cadence for Cesium storage after a cascaded channel delete.
