# 0049 - Strongly Typed Task Resources

**Feature Name:** Per-Integration Task Resources with a Slim Execution Task

**Status:** Draft

**Related:** [RFC 0017](./0017-240104-drivers.md),
[RFC 0027](./0027-260223-oracle-schema-system.md),
[RFC 0033](./0033-260320-oracle-migrations.md),
[RFC 0039](./0039-260409-server-side-import-export.md),
[RFC 0041](./0041-260527-core-structure-refactor.md),
[RFC 0042](./0042-260331-oracle-struct-unions.md),
[RFC 0046](./0046-260720-oracle-predecessor-chain-versioning.md), the task
draft/deploy RFC (PR #2595), and the superseded task/config split RFC (PR #2471).

---

## 0 - Summary

Today, a task is one record with an opaque `config` field. The field is a
`map[string]any` in Go, a `json::object_t` in C++, a `dict[str, Any]` in Python,
and a `record.Unknown` in TS. On the wire, it is a `google.protobuf.Struct`. The
Core does not parse it. Each integration defines its config shape three times by
hand: in Console Zod, in Python pydantic, and in C++ `parser.field<T>()` calls.
The three copies do not stay identical.

This RFC inverts the model. Each task type becomes a first-class resource with a
strong type. Each resource has its own Oracle schema, ontology type, protobuf
message, and service. Examples: `ni_analog_read`, `opc_read`, `http_read`. The
typed resource is the root aggregate. It has a uuid key. It owns the name and all
configuration fields. It is the unit that users create, edit, snapshot, import,
and export. The generic task becomes a small execution record: `{key, rack,
internal, auto_start, config_hash}`. The task exists only while the resource is
deployed to a rack. The typed resource points to its task. A resource without a
task is a draft.

Task types become a closed set. The Core validates each configuration at the
create and update entry points with Oracle-generated validation. The Core owns
the config migration chain. The Core serves the driver a payload with a strong
type: a protobuf `oneof`, never an `Any`. The driver and the Console do not parse
configs by hand. The only parsers are the Oracle-generated Zod, pydantic, and
proto code.

---

## 1 - Motivation

Five problems add up. All of them come from config opacity:

1. **There is no single source of truth.** The OPC UA read config is defined in
   `console/src/feature/opc/task/types.ts`, in `client/py/synnax/opcua/types.py`,
   and in `driver/opc/read_task.h`. These three copies are written by hand. They
   must agree on each field name, default, and bound. No tool reports when they do
   not agree. NI multiplies this problem by approximately 50 channel variants.
2. **The Core cannot validate or migrate.** `Task.Config` is
   `msgpack.EncodedJSON` (`core/pkg/service/task/versions/v1/types.gen.go`). The
   Core stores it as raw JSON bytes and does not touch it. Config schema changes
   occur on the client side, or not at all. The Console holds a private version
   chain in `console/src/feature/ni/task/types/v0.ts`. The Python client reads old
   shapes incorrectly and does not report it. The driver fails to parse them.
3. **Wire case-conversion corrupts semantic keys.** The task config does not have
   a `preserveCase` wrapper. Thus the TS codec (`x/ts/src/binary/codec.ts`)
   converts record keys that are data, not identifiers. Examples: OPC NodeIds and
   EtherCAT `manual_<index>_<subindex>` map keys. The `??` fallback lookups in
   `console/src/feature/opc/task/Read.tsx` and
   `console/src/feature/ethercat/task/types.ts` exist only to hide this corrupted
   persisted data.
4. **Server-side import and export are blocked.** RFC 0039 moved metadata import
   into the Core. But `core/pkg/service/task/imex.go` must flatten an opaque blob.
   The code says that this flattening is temporary until task configs get strong
   types. SY-4524 (server-side task import) is blocked on this RFC.
5. **The open set has no remaining value.** Task types were an open set so that
   drivers could change independently (RFC 0017). In practice, each type string is
   first-party, and the driver and the Core ship together. Openness now only means
   that the Core cannot reject bad data and cannot read its own data.

---

## 2 - Vocabulary

- **Typed task resource** (or **resource**) — a first-class record with a uuid key
  for one task type (`http_read`, `ni_analog_read`). It holds the name and all
  configuration fields inline. A per-integration service owns it.
- **Task** — the slim execution record. It exists only while deployed. It is
  always bound to a rack.
- **Integration** — a hardware family and its schema/service unit: `ni`, `opc`,
  `labjack`, `modbus`, `ethercat`, `http`, `arc`, `pagerduty`, `slack`.
- **Draft** — a typed resource with no task. This is the only undeployed state.
- **Deploy / undeploy** — to deploy is to make a task on a rack for a resource. To
  undeploy is to delete that task. Undeploy never deletes the resource.
- **Resolved task payload** — the wire shape that the driver consumes: the slim
  task joined with the typed config of its resource. The API layer composes it
  (the resolved-field pattern of RFC 0041).
- **`empty` resource** — the minimal resource type. It has a name and nothing
  more. It is the parent of internal tasks with no configuration: the scanners and
  the rack status task.

---

## 3 - Principles

1. **The Core owns persisted data and its migrations.** Config shapes are
   persisted data. Their schema and version chain belong to the Core, not to each
   client.
2. **Write the shape one time.** One Oracle schema generates the Go, TS, Python,
   C++, and protobuf code. The clients and the driver consume generated code. Hand
   parsing is a defect.
3. **The specific depends on the generic.** Integration services compose the task
   service. The task service never learns integration names. Type dispatch occurs
   above it, in a registry that is wired explicitly at the API composition layer.
4. **Closed set, loud failure.** Task types are internal dispatch keys. An unknown
   type is a composition bug. The Core rejects it. The Core never stores it and
   never drops it without a report. The one soft edge is the migration of data
   that exists before the cutover (§4.6).
5. **The resource is what users own.** The configuration is the durable artifact.
   Users draft it, snapshot it, import it, and export it. Execution is temporary
   and rack-bound. The data model shows this: portable uuid identity on the
   resource, rack binding on the task.

---

## 4 - Design

### 4.0 - The split

```
┌────────────────────────────┐        ┌──────────────────────────────┐
│ http_read (resource)       │        │ task (execution record)      │
│  key        uuid           │  0..1  │  key          uuid           │
│  name       string         │ ─────► │  rack         rack.Key       │
│  snapshot   bool           │  task  │  internal     bool           │
│  endpoints  Endpoint[]     │        │  auto_start   bool           │
│  rate       telem.Rate     │        │  config_hash  uint64         │
│  ...                       │        │  (status via status service) │
└────────────────────────────┘        └──────────────────────────────┘
```

The resource holds an optional `task` field (the uuid of the task). The ontology
mirrors this with a `parent_of` edge from the resource to the task. The server
stamps the edge in the same transaction that makes the task. PR #2496 established
this pattern for Arc. This RFC extends the pattern to each integration.

Invariant: **each task has exactly one parent resource.** The scanner tasks and
the rack-status task of the driver obey this rule: the driver makes an `empty`
resource for them at startup. Because the edge always exists, the task row does
not need a `type` field. The ontology type of the resource is the type of the
task. On the wire, the `oneof` case in the resolved payload carries the type.

### 4.1 - The slim task

The task keeps only what execution needs:

| Field         | Why it stays                                                      |
| ------------- | ----------------------------------------------------------------- |
| `key`         | uuid identity (PR #2603); the target of `sy_task_cmd` and status  |
| `rack`        | the deployment target; a mutable field, not key bits (PR #2603)   |
| `internal`    | hides driver-created tasks from users                             |
| `auto_start`  | execution behavior, lifted out of each per-type config            |
| `config_hash` | drift detection for deploy-on-start (§4.3)                        |

These fields leave the task: `name` (the resource owns it), `type` (the ontology
edge and the wire `oneof` own it), `config` (the resource *is* the config), and
`snapshot` (snapshots do not have task rows, §4.3). Status stays in the status
service, keyed by the ontology ID of the task. A draft has no status because
nothing executes.

The generic `/task/create` endpoint is not exposed. Nothing needs it. Users
create resources through the per-type endpoints. The driver mints its internal
tasks through the deploy path against `empty` resources.

### 4.2 - Schemas and services

**One Oracle schema file per integration**: `schemas/synnax/ni.oracle`,
`opc.oracle`, `labjack.oracle`, `modbus.oracle`, `ethercat.oracle`,
`http.oracle`, plus `arc.oracle` (amended), `pagerduty.oracle`, `slack.oracle`
(the Slack RFC lands directly in this shape), and the `empty` resource. A file
defines the shared parts of an integration (channel unions, scales, endpoints)
and its task-type resources. We rejected per-type files. The 19-variant AI
channel union of NI, and its scale/CJC unions, are shared across its five task
types. A split would cause heavy cross-file imports and give no isolation.

The NI schema from PR #2433 (32 enums, 4 unions, 57 structs) is the start point.
We adapt it so that the task configs become root resources, not embedded config
shapes. Shared cross-integration bases (the `sample_rate` / `stream_rate` /
`data_saving` read shapes and the write shapes) live in a common task schema.
The per-integration files extend it. Oracle must first support the extension of a
common shape across schema files. That is groundwork for this RFC.

Each resource has the standard tags: `@ontology type "<task type>"`, `@retrieve`,
`@search`, `@create`, `@go migrate`, `@pb`. The outputs go to
`core/pkg/service/<integration>`, `client/ts`, `client/py`, and `client/cpp`.

**Go services are per-integration packages**: `core/pkg/service/ni`,
`core/pkg/service/opc`, and so on, with the standard service anatomy, adjacent to
the current `arc` and `pagerduty` packages. Each package owns the gorp tables,
the writers, the retrieve builders, the ontology registration, and the imex of
its resources.

**Structural rules for the new schemas:**

- Maps with semantic keys become arrays of structs. The key becomes a field. The
  EtherCAT `auto_<pdo>` / `manual_<index>_<subindex>` channel maps are the
  canonical case. This change removes the caseconv corruption class at the root.
  We then delete the `??` fallback lookups.
- Validation that Oracle can express (bounds, discriminants, required fields)
  lives in the schema. Rules outside its reach (stream rate ≤ sample rate, port
  uniqueness) live in the validation of the Go service, at the same entry point.
  Clients can state them again as UI-side refinements.

### 4.3 - Lifecycle

**Create.** `POST /<type>/create` makes a draft: a resource with no task. The
resource does not have and does not accept a rack.

**Deploy.** A deploy operation on the per-type endpoint receives the resource key
and a rack key. In one transaction, the service validates the config, examines
the `integrations` list of the rack, mints the task, writes the `task` field of
the resource, and stamps the ontology edge. A rack without `ni` causes a hard
error for an `ni_analog_read` deploy, because that deploy can never succeed. A
deploy to a different rack writes the new value into the `rack` field of the
task. The old driver then tears down, as the draft/deploy RFC specifies for a
rack move.

**Start / stop, deploy-on-start.** This RFC builds on the task draft/deploy RFC
(PR #2595) and its SY-4488 implementation stack, and keeps its behavior:
`sy_task_set` is metadata-only, and the `start` command absorbs configuration.
The Core computes `config_hash` (xxhash64 over canonical bytes) when it writes
the resource, and stores the hash on the task row. On `start`, the driver
compares the hash with its active instance. Only on a mismatch does it fetch the
resolved payload again. Drivers report the active hash in the status details.
Drift is the difference between the two hashes. Two points change relative to
that RFC: the input of the hash is the typed resource, and drafts are resources
without tasks, not tasks without racks (§6.6).

**Edit while deployed.** A write to the resource updates `config_hash` on its
task row in the same transaction. This fires the current gorp-observe /
`sy_task_set` path as a metadata refresh. Active tasks do not restart. The
Console shows the drift until the user deploys again with `start`.

**Undeploy / delete.** To delete the task is to undeploy. The execution record
and its status go away. The resource stays as a draft with `task` cleared. To
delete the resource is to also delete its task (the Arc `deleteChildTasks`
pattern). The delete action on a task entry in the Console tree deletes the
resource, because the resource is the object that the user sees.

**Snapshot.** A snapshot is a copy of the resource: a new uuid, `snapshot: true`.
The copy is immutable through the same guard that protects snapshots today.
Ranges refer to it exactly as they refer to task snapshots now. A snapshot does
not get a task row. A frozen config has no execution, so no part of it belongs
in the task table.

**Copy.** A copy of a resource is a per-type create from the fields of a current
resource. The `Copy` method of the task writer goes away with the config that it
existed to clone.

### 4.4 - Wire format

Configs have strong types on each wire. `google.protobuf.Struct` and `Any` are
gone.

- **Per-type protobuf messages**, generated by the Oracle `@pb` output for each
  integration. The resolved task payload for gRPC (`TaskRetrieve` and the fetch
  of the driver) is the slim task plus a `oneof` across all resource config
  messages. The `oneof` case is the task type on the wire. The driver switches on
  it and gives the typed message to the factory. The `ErrTaskNotHandled` scan
  goes away, because dispatch is exact.
- **C++** consumes the generated proto types directly. We delete the
  `x::json::Parser` config layer in `driver/*/` for each integration when it cuts
  over.
- **TS and Python** use the HTTP JSON/msgpack transports with generated Zod
  schemas and pydantic models against the per-type endpoints. The schema controls
  the wire field names, so the TS case conversion is exact. The maps with
  semantic keys became arrays (§4.2), so the task schemas do not need
  `preserveCase` exits.

### 4.5 - The resolved join

The task service stays unaware of integrations. At the API composition layer, a
registry maps the ontology type to the resource service. The registry is wired
explicitly. `TaskRetrieve` uses it to attach the typed config of each task to
the response. This is the resolved-field pattern from RFC 0041. PR #2496 proved
it for `Arc.Task`. Here the direction inverts: the task resolves its parent
resource through the ontology edge. This is the one place where generic code
dispatches to specific code. It is composed at the wire-up site in `layer.go`,
never through self-registration.

The flow of the driver keeps its shape: one round trip returns all data that
configuration needs. The factories of the in-process Go driver (`arc`,
`pagerduty`, `slack`) receive the typed struct. They do not unmarshal
`EncodedJSON`.

### 4.6 - Migration

A one-time startup migration converts each stored task:

1. For each task row, the migration dispatches on the legacy `type` string to the
   applicable integration migrator. The migrator decodes the config through the
   version chain of the integration. This makes the shapes that the Console
   migrated on the client side into current shapes. It also repairs the semantic
   keys that camelCase conversion corrupted, when it restructures the
   EtherCAT/OPC channel maps into arrays. The migrator then creates the typed
   resource, links it, and makes the task row slim.
2. Snapshot tasks convert to snapshot resources. The migration deletes their task
   rows and points the range references to the new resources.
3. Internal scanner and status tasks convert to `empty` resources. The
   create-if-not-exists path in the driver startup moves to the deploy flow.
4. **Unknown type strings cause a warning and quarantine.** The migration keeps
   the row under a quarantine key with an error status. The driver does not
   configure it. The log reports it clearly. The migration does not drop it, and
   also does not keep it active without a report.

The migration composes with the uuid re-key migration from PR #2603, which must
land first. The migration of this RFC assumes uuid task keys and rack-as-field.

### 4.7 - Integration inventory

| Resource types                              | Notes                             |
| ------------------------------------------- | --------------------------------- |
| `ni_analog_read/_write`,                    | schema from PR #2433              |
| `ni_digital_read/_write`, `ni_counter_read` |                                   |
| `opc_read`, `opc_write`                     |                                   |
| `labjack_read`, `labjack_write`             |                                   |
| `modbus_read`, `modbus_write`               |                                   |
| `ethercat_read`, `ethercat_write`           | channel maps → arrays             |
| `http_read`, `http_write`                   |                                   |
| `arc`                                       | retrofit: the standard edge       |
|                                             | replaces `config{arc_key}`;       |
|                                             | PR #2496 already moved creation   |
|                                             | server-side                       |
| `pagerduty_alert`, `slack_alert`            | Go-side factories; Slack lands in |
|                                             | this shape                        |
| `empty`                                     | scanners (`*_scan`, `ni_scanner`) |
|                                             | and rack status; name only        |

The scanner task types (`opc_scan`, `modbus_scan`, …) have no real configuration
today. They become internal tasks with `empty` parents. They do not get their own
resource types. The `"Rack Status"` type string (with the space) and the other
legacy strings go away with the `type` field itself.

Per-type `StatusData` (the `errors[]` of NI, the read status of EtherCAT) gets a
type in the schema of each integration, threaded through the status details
generic. The Console already types these by hand. The schemas make it official.

---

## 5 - Implementation Phases

Prerequisite (external to this RFC): the SY-4488 stack — uuid task keys with rack
as a field (#2603) and deploy-on-start (#2604). The Console autosave PR (#2605)
must target the typed resource in Phase 6. It must not land against
`task.config`.

Each numbered phase is a PR, or a short series where noted. At each boundary the
tree builds, the tests pass, and the product can ship.

1. **Oracle groundwork.** Cross-file extension of a common shape for the shared
   task config bases. Per-type endpoint generation for resources that hold a
   `task` reference. `oneof` emission for the resolved payload. This is pure
   generator work with generator tests. No schema consumes it yet.
2. **Schema authorship.** The per-integration `.oracle` files and their generated
   Go/TS/Py/C++/pb artifacts, not yet wired (the NI file adapted from #2433). One
   PR per integration or per small group. Each PR is a reviewable schema plus
   inert generated code.
3. **Core services, additive.** The per-integration service packages, the `empty`
   resource, the per-type endpoints, the deploy verb, and the API-layer registry
   with the resolved join. All of it exists adjacent to the untouched task shape.
   Production does not consume it yet. Ginkgo suites exercise it directly. The
   tree stays green because the change is additive.
4. **Core cutover.** The atomic PR. It makes the task row slim (it drops `type`,
   `name`, `config`, and `snapshot`, and adds `auto_start`). It runs the §4.6
   startup migration. It routes the Go driver factories (arc, pagerduty) through
   typed payloads. It removes `/task/create` and the config/copy paths of the
   task writer. It points the task imex at the resources, which unblocks SY-4524.
   The PR is large by necessity: a split of the migration and the shape change
   would make an intermediate state that is wired only in part. Phases 1–3 limit
   this PR to the cutover itself.
5. **Driver cutover.** One PR per integration wave. Each wave consumes the
   generated C++ proto configs, deletes the `parser.field` config structs, and
   switches dispatch to the `oneof`. The scanner startup moves to
   `empty`-resource deploys in the first wave.
6. **Console migration.** Its own phase, one PR per integration. The forms bind
   to typed resources through generated Zod (the UI-only refinements stay). The
   tabs key on resource uuids. Autosave (the flow of #2605) targets the resource.
   The drift and deploy UX follows the draft/deploy RFC. We delete the `??`
   fallback lookups and the client-side NI version chain.
7. **Python rewiring.** The generated pydantic resource models replace the
   hand-written config models. The wrappers in the style of `StarterStopperMixin`
   stay as sugar over create, deploy, and start.
8. **Arc + Slack alignment.** Complete the Arc retrofit on top of #2496 (drop
   `config{arc_key}`, use the standard edge). Land `slack_alert` (SY-3995)
   directly in the new shape.

**Compatibility:** the wire for task payloads breaks at Phase 4. The Core, the
driver, and the Console ship the cutover together (lockstep releases, no window
of coexistence). Persisted data migrates forward automatically. There is no
downgrade path across Phase 4. This is the standard position for storage
migrations in this codebase.

---

## 6 - Resolved Decisions

1. **A config union inside `Task` — rejected.** This was the alternative to the
   full RFC: keep one task table and type `config` as an Oracle discriminated
   union. It gives typed configs with much less machinery, and that trade is
   real. We rejected it because it types the blob but does not repair the model.
   It gives no per-type ontology presence or permissions. It gives no drafts. It
   gives no portable uuid identity for imex (task keys carry rack history). And
   snapshot and copy stay attached to execution.
2. **The task/config split RFC (PR #2471) and its Phase 1 (#2472) —
   superseded.** That draft split the same data but kept the task as the root,
   kept the wire format (config as a composed resolved field), kept an
   embedded-blob fallback for types without a registration, and used a
   three-release dual-write rollout. Its caution was its purpose — and is the
   reason we supersede it. To keep the wire is to keep the open set and the
   opaque payload that this RFC exists to remove. The registry idea survives at
   the API composition layer (§4.5). The writer dispatch, the dual write, and the
   fallback machinery do not.
3. **The task points to its config (option 2) — rejected.** This puts a per-type
   reference on the generic row. The task service must then resolve specifics,
   which is the wrong dependency direction. The identity of the portable artifact
   stays rack-bound. Drafts get no home. The typed resource that points to its
   task follows the Arc precedent and keeps the specific→generic arrow.
4. **`google.protobuf.Any` / `Struct` for configs — rejected.** Typed `oneof`
   only. `Any` returns dispatch to strings and defeats the purpose. `Struct` is
   the current state that we remove.
5. **Per-type schema files — rejected** in favor of per-integration files. These
   keep the shared channel and scale unions adjacent to their users (§4.2).
6. **Draft tasks without racks (the draft/deploy RFC model) — replaced.** Two
   draft representations (a task without a rack, a resource without a task)
   cannot exist together. A task now always has a rack. The resource without a
   task is the only draft. The deploy-on-start, hash, and metadata-only-set
   behavior of the draft/deploy RFC stays.
7. **The node as a first-class deployment target — deferred.** Each node has
   exactly one embedded rack. A deploy to a node is thus sugar: the deploy
   endpoint resolves the node to its embedded rack. A polymorphic location type
   waits for the rack and device uuid re-key follow-up.
8. **A `task.name` mirror — rejected.** The resource owns the name. Generic task
   lists read through the resolved join. A synchronized mirror is a second source
   of truth, and no consumer needs it more than the join.
9. **Snapshot task rows — removed.** A snapshot is a frozen resource copy.
   Execution state has no place in it.
10. **The open task-type set — closed.** A third-party task type would need a
    schema, a service, and a driver factory in any case. An exit that skips
    validation makes the problems of today occur again, and for exactly the tasks
    that are most likely to be malformed. Test suites use a registered test-only
    type behind the service config seam, not a wire bypass.
11. **Multi-deploy (one resource, many tasks) — future work.** Validation holds
    the `task` field to 0..1. Fan-out changes the channel-write behavior and
    needs its own design.

---

## 7 - What This RFC Does Not Cover

- **Typed command args** (`start`, `stop`, `tare`, `test_connection`, `browse`):
  `Command.args` stays opaque. A follow-up can type per-command args with the
  same union machinery.
- **The uuid re-key of statuses, racks, and devices**, and the move of device
  management to the server side. Status keys stay task-ontology-ID strings here.
- **Typed device `properties`.** The channel maps with NodeId keys inside device
  properties are the same corruption class as §4.2. They must get the same
  array-of-structs treatment in an adjacent effort.
- **Protobuf as the TS and Python wire.** Those clients stay on HTTP
  JSON/msgpack. The snake↔camel removal effort tracks the larger proto
  migration.
- **Multi-deploy fan-out** (Resolved Decision 11).

---

## 8 - Open Questions

1. The final name of the `empty` resource (`empty`, `plain`, or `internal`).
2. The endpoint shape of the deploy verb: `POST /<type>/deploy`, or a `rack`
   parameter on create and update. This is a parameter-level choice. The
   transaction behavior in §4.3 holds in both shapes.
3. The quarantine surface for unknown-type rows at migration (§4.6): the status
   variant, and whether the Console lists them.
4. The wave order of the driver and Console cutovers across integrations in
   Phases 5–6.
