# 100 Gorp references

- **Author**: Patrick Dotson, Emiliano Bonilla
- **Date**: 2026-11-04
- **Related**: [RFC 0005 - MVP - Data ontology](0005-ontology.md),
  [RFC 0026 - Meta data structures](0026-meta-data.md),
  [RFC 0033 - Oracle migration system](0033-oracle-migrations.md),
  [RFC 0034 - Gorp in-memory indexes](0034-gorp-indexes.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md),
  [RFC 0099 - Toolbar tree search](0099-toolbar-tree-search.md)

## 0 Summary

The ontology stores relationships in two Gorp tables, resources and relationships, with
two edge types, `parent` and `labeled_by`. About 38 hand-written edge writes in 22
services keep them. Nothing checks that an edge points at a record that exists, and each
service has its own delete rule.

This RFC moves each relationship into a typed reference field on the record that holds
it. The Oracle schema declares the field with `@ref` and a delete policy (`cascade`,
`restrict`, or `detach`). Gorp checks the target on write, keeps a reverse index, and
applies the policy on delete, in the caller's transaction. The ontology package, both of
its tables, and the `/ontology/*` routes are deleted at the end. The `{type, key}`
identifier stays as `resource.ID`.

## 1 Motivation

RFC 0026 §1.1.10 named the problem, and RFC 0042 §7 deferred its fix to this RFC.

- **Two edge types carry eight meanings**: `parent`
  (`core/pkg/service/ontology/versions/v0/relationship.go`) is group membership, project
  ownership, rack ownership, device chassis, range nesting, range snapshot, Arc binding,
  and role assignment. Readers filter on endpoint types to get the meaning back.
- **Copies drift**: Only two edges mirror a stored field, `device.rack` and the Arc task
  `arc_key`. Rack delete reads the field (`core/pkg/api/rack/rack.go`) and the tree
  reads the edge. `Range.parent`, `Range.labels`, `Status.labels`, and `Panel.parent`
  are filled from edges at read time.
- **Integrity is hand-written and leaks**: Range delete cascades, group delete refuses,
  and project delete orphans its visualizations (`core/pkg/service/project/writer.go`).
  Role delete (`core/pkg/service/access/rbac/role/writer.go`) leaves the role's edges.
  The policy walk reads only edges (`core/pkg/service/access/rbac/policy/retriever.go`),
  so the users of a deleted role keep its access.
- **Identity is stored twice, and the graph is slow**: The resource table holds a second
  copy of each ID, the cause of the orphan window in SY-4804. Resource delete scans the
  full relationship table per ID, and the TypeScript cache scans each edge on each
  delete (SY-4754).

## 2 Vocabulary

- **Reference**: A field that holds the key of another record, as a scalar or an array.
- **Target** and **dependent**: The record a reference points at, and one that holds it.
- **Closure**: The full set of records that one delete removes: the records the query
  matches, and each `cascade` dependent of a record in the set.

## 3 Principles

1. **One copy**: A relationship is one field on one record. Never a field and an edge.
2. **Declared, then generated**: The schema declares the reference and its policy.
   Oracle generates the rest. Services write no relationship code.
3. **Integrity belongs to the store**: Gorp checks targets and applies policies. It
   never calls service code.
4. **Ownership and placement are different fields**: `schematic.project` is the owner.
   `schematic.group` is where a toolbar shows it.
5. **Record-specific rules are guards**: A policy applies to a full field. A rule such
   as "refuse a group that has members" stays a `gorp.Delete.Guard`.

## 4 Design

### 4.0 The schema declaration

`@ref` is a field domain. Oracle gets the target from the field type: `rack.Key` points
at the struct whose `@key` has that type. A `resource.ID` field lists its target tables.
A field with only a policy takes the short form `@ref on_delete restrict`.

```
Status struct {
    owner resource.ID? {
        @ref {
            types     task rack device channel
            on_delete cascade
            unique
        }
    }
}
```

- `on_delete cascade | restrict | detach`: Required. `cascade` deletes the dependent.
  `restrict` refuses the delete while the dependent exists. `detach` keeps the dependent
  and removes the target: it clears a scalar, which must be optional or have a default,
  and removes an array element.
- `types`: Required on a `resource.ID` field, and allowed only there.
- `unique`: At most one dependent per target. A nil value holds no target.
- `acyclic`: A chain of references in one table must not come back to its start.

For each `@ref`, Oracle generates a `gorp.Reference` on the table and a `Match<Field>s`
filter that reads its reverse index. `@ontology type` becomes `@resource type`, and it
names the table in the Gorp registry.

### 4.1 Gorp

**Registry** (`gorp/registry.go`, specified in `gorp/reference_test.go`): Each `gorp.DB`
holds its open tables by `TableConfig.Name`, which defaults to the entry type name. Two
open tables cannot share a name. `TableConfig.References` holds one `gorp.NewReference`
per field, with `Name`, `Tables`, `Targets`, `OnDelete`, `Detach`, `Unique`, and
`Acyclic`. A `gorp.Target` is a table name and a text key
(`gorp/internal/key/format.go`), so one delete walk crosses string, integer, and UUID
keys.

**Write checks** (`gorp/table_references.go`): `Create` and `Update` check each
reference immediately, in the caller's transaction. The target table must be in
`Tables`, the target must exist, and `unique` and `acyclic` must hold. A target written
earlier in the same transaction passes. Only queries that a `Table` builds are checked.
Package-level queries such as `gorp.NewCreate` skip the checks, as they skip indexes.

**Reverse index**: Each reference keeps an in-memory index from target to dependents,
with one entry per array element (`gorp/internal/index/multi.go`). A probe sees the
writes staged in its own transaction. The index fills in the background at open, and a
probe waits for it. A failed fill fails each probe, with no fallback scan.

**Delete**: A `Delete` that a `Table` builds runs in three steps (`gorp/delete.go`):

1. Guards run on the entries the query matches, never on entries a cascade adds.
2. The plan reads only. It adds each `cascade` dependent to the closure and repeats
   until nothing new comes in. `restrict` fails only for a dependent outside the
   closure. `detach` applies only to dependents the delete keeps.
3. The apply step detaches first, then deletes each entry in the closure.

A closure larger than `gorp.WithMaxDeleteSize` (default `1 << 20`, `gorp/options.go`)
fails before any change. No service code runs inside a cascade (Ent issue #2031 shows
hooks and store cascades that disagree). A side effect that is not data, such as freeing
Cesium storage, follows the table's change stream.

**Errors**: A failed write check is a `validate.PathError` on the field, such as
`rack: rack 1 does not exist`. A refused `restrict` is `validate.ErrValidation` with
`cannot delete X because Y references it through F`.

**Isolation**: Integrity holds within one node, and concurrent transactions are not
isolated today. A device create and a delete of its rack in two transactions can both
commit. §7.0 keeps the lock design open.

### 4.2 Every relationship, mapped

| Field                          | On delete        | Today                     |
| ------------------------------ | ---------------- | ------------------------- |
| `device.rack rack.Key`         | restrict         | field and `parent` edge   |
| `device.parent device.Key?`    | cascade, acyclic | `parent` edge             |
| `task.rack rack.Key`           | cascade          | field                     |
| Arc task `arc_key`             | cascade          | field and `parent` edge   |
| `range.parent ranger.Key?`     | cascade, acyclic | `parent` edge             |
| `<labeled>.labels label.Key[]` | detach           | `labeled_by` edge         |
| `status.owner resource.ID?`    | cascade, unique  | text in the status key    |
| `task.range ranger.Key?`       | cascade          | snapshot `parent` edge    |
| `alias.range`, `pair.range`    | cascade          | text in the composite key |
| `<vis>.range ranger.Key?`      | cascade          | snapshot `parent` edge    |
| `<vis>.project project.Key?`   | cascade          | `parent` edge             |
| `<record>.group group.Key?`    | detach           | `parent` edge             |
| `group.parent resource.ID?`    | cascade, acyclic | `parent` edge             |
| `<subject>.roles role.Key[]`   | detach           | `parent` edge             |
| `policy.role role.Key`         | cascade          | `parent` edge             |

`<labeled>` is a range or a status, and `<subject>` is a user or a rack. `<vis>` is each
visualization: schematic, line plot, log, table, and panel. A visualization holds
exactly one of `project` and `range`, and one shared validator checks it. Alias and
key-value pairs keep their composite keys, which Oracle derives from the `@key` fields.

### 4.3 Groups

A group is a folder. Each groupable record has a `group group.Key?` field with `detach`,
so a record is in one folder at most. A group has `parent resource.ID?` with
`types group project rack`, `cascade`, and `acyclic`, and a `scope resource.Type` that
names its tree (`channel`, `device`, `schematic_symbol`). The group service checks that
`scope` stays the same down a chain. A toolbar root shows the groups of its scope with
no parent, and the records of its type with no group. The built-in root groups that
`Group.CreateOrRetrieve` makes in ten services are deleted.

Membership is `detach`, not `restrict`, because a project delete cascades to its groups
and a grouped channel must not block it. A guard in the group service
(`core/pkg/service/group/service.go`) refuses a direct delete of a group that is not
empty. Guards run only on the entries the query names, so the guard does not block a
cascade. To group devices under a rack, set `group.parent` to the rack. The device
service checks that `device.rack` matches.

### 4.4 Statuses

Status keys hold the owner as text today (`rack.StatusKey`), and the Driver computes
them at many sites (`driver/common/status.h`). The key becomes a UUID, and `owner`
(§4.0) holds the owner. The status set endpoint accepts an owner and upserts through the
unique index, so the Driver sends `{owner: task_id, ...}` and keeps its idempotent write
with no round trip. `rack.StatusKey`, `calculation.StatusKey`, the C++ `status_key`
functions, and the owner keys in `details` are deleted.

### 4.5 Services and access

Each `ontology.Writer` call and `ontology.Service` implementation is deleted. Range
delete drops its hand-written walk (`core/pkg/service/ranger/writer.go`), and project
delete cascades to its visualizations. Rack delete keeps its refusals: `device.rack` is
`restrict`, and a guard refuses while tasks that are not internal remain. `task.rack` is
`cascade`, so internal tasks go with the rack.

The policy walk becomes the subject's `roles`, then `policy.retrieve({role})`. A role
delete removes its policies and detaches its subjects, which closes the leak in §1.

Channel delete frees Cesium storage from a subscriber on the channel table's delete
stream, not in `core/pkg/service/channel/writer.go`, so a cascade takes the same path.

### 4.6 API and clients

The `/ontology/*` routes and their access checks are deleted. Group create, rename, and
delete move to `/group/*`, and a move into a group updates the record's `group` field.

`/batch` takes an ordered list of typed requests, each one an existing endpoint and its
payload. It runs them in one `gorp.WithTx` and stops at the first error. It accepts only
requests whose work is all inside that Gorp transaction, so it refuses channel create,
which also writes Cesium. A drag of a mixed selection stays atomic through `/batch`.

The TypeScript client deletes `client/ts/src/ontology`, Python deletes
`synnax.ontology`, and the ID moves to `resource`. The Console tree
(`console/src/platform/tree`) stays generic over a root `resource.ID`, but each item
type gives a `children` resolver: a rack lists the devices whose `rack` is that rack. An
open node is a list query that stays live through the `sy_<table>_set` and
`sy_<table>_delete` channels. RFC 0099 §2.0 ancestors become a walk up `group.parent`.

### 4.7 Migration

One startup migration per table derives its reference fields from the old edges through
`gorp.WrapReader`, as `core/pkg/service/project/versions/v1/migrate.go` does:

- `device.parent`, `range.parent`, the snapshot `range` fields, `<vis>.project`, and the
  role fields come from `parent` edges. Labels come from `labeled_by` edges.
- Each `group` field comes from a group edge, unless the group is a built-in root.
  `group.parent` comes from the edge into the group, and a built-in root maps to nil.
  `group.scope` comes from the built-in root at the top of the chain.
- The status key becomes a UUIDv5 of the old key, and `owner` is parsed from the old
  key. Each node derives the same key, as RFC 0033 §4.2.5 requires.

After each service opens, a step behind a marker key drops both ontology tables and the
built-in groups. A visualization with no project and no range is deleted; no path
reaches it today. The migration is one-way: a Core that ran it cannot go back to a
release that reads the ontology.

## 5 Implementation phases

Phase 1 is a mechanical rename, and Phase 2 adds the engine with no Core table on it.
The ontology serves each domain that has not moved. Phase 8 adds references inside a
record (`lineplot.channels`, task config keys, panel tabs, the PagerDuty `status` key)
and `channel.index`. Gorp needs no change for them, because `Targets` is a function.

| Phase | Scope                                                     | Touches                                                                                                                                                                                              | Depends on |
| ----- | --------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Rename ontology IDs to resource IDs                       | `schemas`, `oracle`, `core/pkg/service`, `client/ts`, `client/py`, `client/cpp`                                                                                                                      | -          |
| 2     | Add typed references to Gorp and Oracle                   | `gorp`, `oracle`, `docs/tech/rfc`                                                                                                                                                                    | 1          |
| 3     | Move labels, statuses, and ranges to references           | `schemas`, `core/pkg/service/label`, `core/pkg/service/status`, `core/pkg/service/ranger`, `driver`, `client/ts`, `client/py`, `client/cpp`, `console`                                               | 2          |
| 4     | Move groups, channels, and the Console tree to references | `schemas`, `core/pkg/service/group`, `core/pkg/service/channel`, `core/pkg/api`, `core/pkg/transport`, `client/ts`, `pluto`, `console`                                                               | 2          |
| 5     | Move racks, devices, tasks, and Arc to references         | `schemas`, `core/pkg/service/rack`, `core/pkg/service/legacy/device`, `core/pkg/service/task`, `core/pkg/service/arc`, `client/ts`, `client/py`, `console`                                           | 2          |
| 6     | Move projects and visualizations to references            | `schemas`, `core/pkg/service/project`, `core/pkg/service/schematic`, `core/pkg/service/lineplot`, `core/pkg/service/log`, `core/pkg/service/table`, `core/pkg/service/panel`, `client/ts`, `console` | 4          |
| 7     | Move access to references and delete the ontology         | `schemas`, `core/pkg/service/access`, `core/pkg/service/user`, `core/pkg/service/rack`, `core/pkg/service/ontology`, `core/pkg/api`, `core/pkg/transport`, `client/ts`, `client/py`, `console`       | 3, 4, 5, 6 |
| 8     | Add nested and channel references                         | `schemas`, `core/pkg/distribution/channel`, `core/pkg/service/lineplot`, `core/pkg/service/log`, `core/pkg/service/task`, `core/pkg/service/panel`, `core/pkg/service/pagerduty`                     | 2          |

## 6 Resolved decisions

1. **One field per link, with no edge table**: An edge table as the one copy puts the
   record and its relationship in different Aspen batches, and no reader needs an
   untyped walk. Cost: group membership is a field on each groupable type.
2. **No generic tree layer**: A thin tree service over the fields brings back the
   untyped graph. Each toolbar builds its tree from typed queries (§4.6).
3. **Immediate checks**: Deferred constraints (SQL `INITIALLY DEFERRED`) add a commit
   phase. Cost: write the target first, and two tables cannot require each other.
4. **Closure first, then change**: A per-step `restrict` (SQL `RESTRICT`) fails a delete
   on a dependent that the delete also removes.
5. **One transaction with a cap**: A cascade split across transactions leaves dangling
   references when it stops halfway. The cap turns an out-of-memory crash into an error.
6. **Only queries a `Table` builds are checked**: Package-level queries have no table.
   Cost: a service that writes around its table skips the checks.
7. **`resource.ID` with required `types` for polymorphic references**: A field per
   target type adds a field for each new container. Only `status.owner` and
   `group.parent` use one.
8. **Groups detach their members**: `cascade` deletes channels with a folder. `restrict`
   lets placement block ownership (§4.3). Cost: `scope` is a service check.
9. **UUID status keys, derived in migration**: An owner-derived key is a hidden
   reference. A random UUID gives each node a different key. `owner` is optional because
   the Console and the clients create statuses with no owner.
10. **Visualizations with no owner are deleted**: A recovery project was rejected. No
    path reaches them today.

## 7 Open questions

### 7.0 Locking

This RFC adds no lock. Per-table locks taken in sorted order prevent a deadlock only
inside one operation, and one transaction can run two operations that lock tables in
different orders. Transaction A deletes a project: it locks `project`, then the cascade
locks `schematic`. Transaction B updates a schematic, which locks `schematic`, then
creates a project, which needs `project`. Each waits on the other.

The other option is one write lock for the DB, taken at the first write and held to
commit. It cannot deadlock. Aspen already commits one transaction at a time in a serial
persist stage with `pebble.Sync` (`aspen/internal/kv/persist.go`), so the lock adds only
the time from the first write to commit. In that time the rack status write, one per
rack per second, waits on any open delete.

### 7.1 Other questions

1. The `@ref` block grammar: newline-separated like `@ts { }`, or a delimiter.
2. The default closure cap, from a measured worst case. A range can hold one alias per
   channel.
3. Whether a reference can skip its reverse index. Phase 8 puts one over the channel
   table, up to about a million entries, but with no index a delete cannot find
   dependents.
4. The device chassis policy: `cascade` for discovered modules, or `detach` for
   hand-made chassis.
5. Whether `group.scope` can go once each group has a parent or a member.

## 8 What this RFC does not cover

- Integrity across nodes. Each check is node-local, and a Gorp transaction splits into
  one batch per leaseholder (`aspen/internal/kv/tx.go`).
- Validation of stored data. Gorp checks a reference on write only, and each reference
  here comes from a migration that derives it.
- Atomic channel create across Cesium and Gorp (RFC 0042 §7).
- Template bindings. A later RFC builds them on these references.
