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
  and removes the target: it clears an optional scalar, writes zero to a scalar with a
  zero default, and removes an array element. A required field in an array element
  detaches by removing the element. Oracle refuses `detach` on any other field.
- `types`: Required on a `resource.ID` field, and allowed only there.
- `unique`: At most one dependent per target. A field that holds no target is exempt.
- `acyclic`: A chain of references in one table must not come back to its start.

A nil value, an empty array, and a zero scalar with a zero default (`= 0`, `= ""`) hold
no target. A new reference uses `X?`. The zero rule exists for the stored task configs
that use zero for "none" (§6).

`@ontology type` becomes `@resource type`. It names the table in the Gorp registry, and
Oracle emits it as `const tableName`. For each `@ref` of a `@retrieve` type, Oracle
generates:

- A `gorp.KeyReference`, or a `gorp.TargetReference` for a `resource.ID` field, in the
  `indexes` struct. `indexes.references()` returns them for `TableConfig.References`.
- `Match<Field>s(vals...)`, which matches the entries that hold any of `vals`, for a
  scalar and for an array (`MatchLabels` for a `labels` array).
- `MatchHas<Field>(has)`, for a field that can hold no target.
- An unexported `set<Field>(table, v, keys...)` update, for a `detach` scalar. The
  service writer calls it, so each groupable type writes no move code.

A generic type uses `@retrieve indexes`, which generates the references and no filters.
Oracle also walks the arrays, optional structs, and unions of a `@retrieve` type, and
puts each nested `@ref` in the same `indexes` struct, with no filter. A type that a
generic table stores (a task config) uses `@retrieve references`, which generates only
an unexported `<type>References()` function. The same zero and detach rules hold at
every depth. Line plot channels and panel tabs keep hand-written references, because
their detach rebuilds the record. `channel.index` does too, because an index channel
holds no target.

### 4.1 Gorp

**Registry** (`gorp/registry.go`, specified in `gorp/reference_test.go`): Each `gorp.DB`
holds its open tables that have a `TableConfig.Name` or references, by name. The name
defaults to the entry type name, and two of these tables cannot share one. A table with
no name and no references stays out of the registry, and nothing can reference it.
`TableConfig.References` holds one reference per field, with `Name`, its target tables,
`Targets`, `OnDelete`, `Detach`, `Unique`, and `Acyclic`. A `gorp.NewKeyReference`
points at one table and holds typed keys. A `gorp.NewTargetReference` points at any of
several tables and holds `gorp.Target` values: a table name and a key in the text form
of `gorp.FormatKey` (`gorp/internal/key/format.go`). A write fails when that key is not
in the text form. The delete walk uses the text form, so one walk crosses string,
integer, and UUID keys.

**Write checks** (`gorp/table_references.go`): `Create` and `Update` check each
reference immediately, in the caller's transaction. The target table must be in
`Tables`, the target must exist, and `unique` and `acyclic` must hold. A target written
earlier in the same transaction passes. A write does not check again a target that the
stored entry already holds, so a rename reads no targets. Only queries that a `Table`
builds are checked. Package-level queries such as `gorp.NewCreate` skip the checks, as
they skip indexes. `Table.NewUncheckedWriter` keeps the indexes current and skips the
checks and the delete plan. It is for migrations that write before their target tables
open.

**Reverse index**: Each reference keeps an in-memory index from target to dependents,
with one entry per array element, and a set of the entries that hold no target
(`gorp/internal/index/multi.go`). A probe (`Get`, `Filter`, `FilterHolding`, and
`Table.Dependents`) sees the writes staged in its own transaction. The index fills in
the background at open, and a probe waits for it or for its context. After a failed
fill, `Filter` and `FilterHolding` fall back to a scan, and the other probes fail with
`gorp.ErrIndexInvalid`.

**Delete**: A `Delete` that a `Table` builds runs in three steps (`plan` and `apply` in
`gorp/registry.go`):

1. Guards run on the entries the query matches, never on entries a cascade adds.
2. The plan reads only. It adds each `cascade` dependent to the closure and repeats
   until nothing new comes in. `restrict` fails only for a dependent outside the
   closure. `detach` applies only to dependents the delete keeps.
3. The apply step detaches first, then deletes each entry in the closure. After the
   commit, `Table.ObserveDetaches` notifies each entry that the delete detached.

A delete that would delete or rewrite more entries than `gorp.WithMaxDeleteSize`
(default `1 << 20`, `gorp/options.go`) fails before any change. Detached dependents
count. The plan also stops when the dependents that restrict the delete pass the cap, so
a refused delete reads a bounded set. No service code runs inside a cascade (Ent issue
#2031 shows hooks and store cascades that disagree). So a record whose delete has a side
effect outside Gorp, such as a channel and its Cesium storage, is never a `cascade`
target.

**Errors**: A failed write check is a `validate.PathError` on the field, such as
`rack: rack 1 does not exist`. A refused `restrict` is a `gorp.RestrictError`, a
validation error that holds the target, the holder, and the reference. The API rewrites
it with the names of the records: `cannot delete channel X because task Y uses it`. The
task service maps a config record to its task.

**Isolation** (`gorp/gorp.go`): Each `gorp.DB` holds one write lock. A transaction takes
it when it opens and releases it at commit or close, before observers run, because an
observer can open a transaction. A write through the DB used as a `Tx` runs in a
transaction of its own. So the read in each check and the write it guards never
interleave with another transaction: a device create and a delete of its rack cannot
both commit. A goroutine that opens a second transaction while its first is open
deadlocks. `OpenTx` logs a warning when it waits long for the lock. Integrity holds
within one node (§8).

### 4.2 Every relationship, mapped

| Field                          | On delete        | Today                     |
| ------------------------------ | ---------------- | ------------------------- |
| `device.rack rack.Key`         | restrict         | field and `parent` edge   |
| `device.parent device.Key?`    | cascade, acyclic | `parent` edge             |
| `task.rack rack.Key`           | cascade          | field                     |
| Arc task `arc_key`             | restrict         | field and `parent` edge   |
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
| task config channel and device | restrict         | nothing                   |
| `channel.index channel.Key`    | restrict         | Cesium check              |
| line plot and log channels     | detach           | nothing                   |
| `panel` tabs                   | detach           | `parent` edge             |
| PagerDuty `status` key         | detach           | nothing                   |

`<labeled>` is a range or a status, and `<subject>` is a user or a rack. `<vis>` is each
visualization: schematic, line plot, log, table, and panel. A visualization holds
exactly one of `project` and `range`, and one shared validator checks it. Alias and
key-value pairs keep their composite keys, which Oracle derives from the `@key` fields.
A task config refuses the delete of a channel or device it reads or writes, so a running
task never loses one. The `arc_key` reference sits on the config record, where a cascade
would delete only the config, so the Arc writer deletes the tasks of an Arc first. A
detach from a line plot also clears the lines of that channel.

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

Channel delete frees Cesium storage in the same call
(`core/pkg/service/channel/writer.go`), so a delete while a writer is open fails and
changes nothing. No reference cascades to a channel, and `/batch` refuses channel
delete.

### 4.6 API and clients

The `/ontology/*` routes and their access checks are deleted. Group create, rename, and
delete move to `/group/*`, and a move into a group updates the record's `group` field.

Search and lookup across types move to a service in `core/pkg/service/resource`. Each
domain gives its type a text search, a list, and a name lookup by ID, wired once in
`core/pkg/service/layer.go`. The service holds no relationships. The Console palette
searches every type in one request, and an empty term lists each type's records. A
lookup leaves out an ID that names no record. A screen that shows an ID of any type gets
its name from it, and the TypeScript client keeps an open search live through each
type's change channels. Rejected: each client sends one request per type, which costs
about twenty requests per palette keystroke and copies the fan-out into each client.

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
  `group.scope` comes from the built-in root at the top of the chain. A membership in a
  group whose scope does not hold the record's type is dropped and logged.
- The status key becomes a UUIDv5 of the old key, and `owner` is parsed from the old
  key. Each node derives the same key, as RFC 0033 §4.2.5 requires.

These migrations read edges through the frozen types in
`core/pkg/service/ontology/versions/v0`. A record can already hold a key whose target is
gone, and its next write would fail the check. `gorp.DanglingMigration` runs
`Table.DropDangling` on each entry of a table: it removes each missing target through
the `Detach` of its reference, and skips a reference with no `Detach`. An import runs
`Table.DropDangling` on the record before it writes it. After each service opens, one
last migration drops both ontology tables and the built-in groups. A visualization with
no project and no range is deleted; no path reaches it today. The migration is one-way:
a Core that ran it cannot go back to a release that reads the ontology.

## 5 Implementation phases

Phase 1 is a mechanical rename, and Phase 2 adds the engine with no Core table on it.
The ontology serves each domain that has not moved, and Phase 10 deletes it. Phase 9
adds references inside a record (line plot and log channels, task config keys, panel
tabs, the PagerDuty `status` key) and `channel.index`. Each phase stacks on the one
before it.

| Phase | Scope                                                     | Touches                                                                                                                                                                                                  | Depends on |
| ----- | --------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Rename ontology IDs to resource IDs                       | `schemas`, `oracle`, `core/pkg/service`, `client/ts`, `client/py`, `client/cpp`                                                                                                                          | -          |
| 2     | Add typed references to Gorp and Oracle                   | `gorp`, `oracle`, `x/go/kv`, `core/pkg/service`, `core/pkg/api`, `core/pkg/distribution`, `docs/tech/rfc`                                                                                                | 1          |
| 3     | Move labels, statuses, and ranges to references           | `schemas`, `core/pkg/service/label`, `core/pkg/service/status`, `core/pkg/service/ranger`, `driver`, `client/ts`, `client/py`, `client/cpp`, `console`                                                   | 2          |
| 4     | Move groups, channels, and the Console tree to references | `schemas`, `core/pkg/service/group`, `core/pkg/service/channel`, `core/pkg/api`, `core/pkg/transport`, `client/ts`, `pluto`, `console`                                                                   | 3          |
| 5     | Move racks, devices, tasks, and Arc to references         | `schemas`, `core/pkg/service/rack`, `core/pkg/service/legacy/device`, `core/pkg/service/task`, `core/pkg/service/arc`, `client/ts`, `client/py`, `console`                                               | 4          |
| 6     | Move projects and visualizations to references            | `schemas`, `core/pkg/service/project`, `core/pkg/service/schematic`, `core/pkg/service/lineplot`, `core/pkg/service/log`, `core/pkg/service/table`, `core/pkg/service/panel`, `client/ts`, `console`     | 5          |
| 7     | Move roles and policies to references                     | `schemas`, `core/pkg/service/access`, `core/pkg/service/user`, `core/pkg/service/rack`, `core/pkg/api`, `core/pkg/transport`, `client/ts`, `client/py`, `pluto`, `console`                               | 6          |
| 8     | Add the resource search and name service                  | `core/pkg/service/resource`, `core/pkg/service`, `core/pkg/api`, `core/pkg/transport`, `client/ts`, `client/py`, `pluto`, `console`                                                                      | 7          |
| 9     | Add nested and channel references                         | `schemas`, `gorp`, `oracle`, `core/pkg/service/channel`, `core/pkg/service/lineplot`, `core/pkg/service/log`, `core/pkg/service/task`, `core/pkg/service/panel`, `core/pkg/service/pagerduty`, `console` | 8          |
| 10    | Delete the ontology                                       | `core/pkg/service/ontology`, `core/pkg/service`, `core/pkg/api`, `core/pkg/transport`, `client/ts`, `client/py`, `client/cpp`, `pluto`, `console`, `integration`                                         | 9          |

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
10. **Visualizations with no owner are deleted**: A recovery project was rejected.
    Shipped Cores leave the snapshots of a deleted range with no owner, and the upgrade
    deletes them with a warning for each, as a range delete now does.
11. **One write lock per DB, taken at open**: A lock taken at the first write misses a
    read that comes before that write, and each check reads first. Per-table locks
    deadlock when two transactions take them in opposite orders. Revisit the lock when
    measured lock waits on a busy Core are long.
12. **Zero means no target**: A scalar `@ref` with a zero default holds no target while
    it is zero. Stored task configs use channel `0` and device `""` for "none" in about
    40 fields, so requiring `X?` would migrate each config and the Driver.
13. **Typed target keys**: A reference to one table holds the key type of that table.
    Text keys measured about 138 B per entry against 25 B for a typed lookup index, at
    1M entries. Only the delete walk and `resource.ID` references use the text form.
14. **A detach or a cascade reaches clients like a normal write**: After commit, each
    record that a cascade or a detach changed is published on its table's change
    channel. A table that streams actions publishes a set action from its table observer
    for each record it did not write itself, never from code in the cascade. Clients
    keep no copy of an `on_delete` rule.

## 7 Open questions

1. The `@ref` block grammar: newline-separated like `@ts { }`, or a delimiter.
2. The default closure cap, from a measured worst case. A range can hold one alias per
   channel.
3. Whether a reference can skip its reverse index. Phase 9 puts one over the channel
   table, up to about a million entries, in place of the `local_index` lookup index.
   With no index, a delete cannot find dependents.
4. The device chassis policy: `cascade` for discovered modules, or `detach` for
   hand-made chassis.
5. Whether `group.scope` can go once each group has a parent or a member.
6. Whether a task config keeps `restrict` on its channels and devices, or moves to
   `detach` with a binding the task form shows as broken.

## 8 What this RFC does not cover

- Integrity across nodes. Each check is node-local, and a Gorp transaction splits into
  one batch per leaseholder (`aspen/internal/kv/tx.go`).
- Validation of stored data. Gorp checks a reference on write only, and each reference
  here comes from a migration that derives it.
- Atomic channel create across Cesium and Gorp (RFC 0042 §7).
- Template bindings. A later RFC builds them on these references.
- Channel keys held in schematic symbol props, table cells, Arc programs,
  `alias.channel`, calculated channel expressions, and device property maps. Each stays
  unchecked.
