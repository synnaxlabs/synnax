# 48 Strongly typed task resources

- **Author**: Patrick Dotson
- **Date**: 2026-08-03
- **Related**: [RFC 0017 - General purpose device drivers](0017-drivers.md),
  [RFC 0027 - Oracle schema system](0027-oracle-schema-system.md),
  [RFC 0033 - Oracle migration system](0033-oracle-migrations.md),
  [RFC 0034 - Gorp in-memory indexes](0034-gorp-indexes.md),
  [RFC 0039 - Server-side metadata import/export](0039-server-side-import-export.md),
  [RFC 0041 - Action-based undo and redo](0041-action-based-undo-redo.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md),
  [RFC 0043 - Oracle support for struct unions](0043-oracle-struct-unions.md), and
  [RFC 0047 - Oracle predecessor-chain type versioning](0047-oracle-predecessor-chain-versioning.md).

## 0 Summary

Today, a task is one record with an opaque `config` field: a `map[string]any` in Go, a
`google.protobuf.Struct` on the wire. The Core does not parse it. Each integration
defines its config shape three times by hand — Console Zod, Python Pydantic, C++
`parser.field<T>()` — and the copies do not stay identical.

This RFC inverts the model. Each task type becomes a first-class resource with a strong
type. Each resource has its own Oracle schema, ontology type, Protobuf message, and
service. Examples: `ni_analog_read`, `opc_read`, `http_read`. The typed resource is the
root aggregate. It has a UUID key. It owns the name and all configuration fields. It is
the unit that users create, edit, snapshot, import, and export. The generic task becomes
a small execution record: `{key, rack, config, internal, auto_start, config_hash}`. The
task exists only while the resource is deployed to a rack. The task points to its
resource through a `config` ontology ID. A resource without a task is a draft.

Task types become a closed set. Users edit a resource through action dispatch, the same
synchronous path as schematics. The Core validates the shape of every dispatch, reports
semantic diagnostics live, and gates deploy and `start` on them. The Core owns the
config migration chain. The Driver reads each config from a per-type endpoint as a typed
Protobuf message, never as an `Any`. The Driver and the Console do not parse configs by
hand. The only parsers are the Oracle-generated Zod, Pydantic, and proto code.

---

## 1 Motivation

Five problems add up. All of them come from config opacity:

1. **There is no single source of truth.** Each config shape is written by hand three
   times — Console Zod, Python Pydantic, C++ parser — and no tool reports when the
   copies disagree. NI multiplies this by approximately 50 channel variants.
2. **The Core cannot validate or migrate.** `Task.Config` is `msgpack.EncodedJSON`; the
   Core stores raw bytes. Config migrations occur client-side or not at all: the Console
   holds a private NI version chain, the Python client reads old shapes incorrectly, and
   the Driver fails to parse them.
3. **Wire case-conversion corrupts semantic keys.** The task config has no
   `preserveCase` wrapper, so the TypeScript codec converts record keys that are data:
   OPC NodeIds, EtherCAT `manual_<index>_<subindex>` map keys. The `??` fallback lookups
   in the Console exist only to hide the corrupted persisted data.
4. **Server-side import and export are blocked.** RFC 0039 moved metadata import into
   the Core, but the task imex must flatten an opaque blob. SY-4524 (server-side task
   import) is blocked on this RFC.
5. **The open set has no remaining value.** Task types were an open set so that drivers
   could change independently (RFC 0017). Each type string is first-party, and the
   Driver and the Core ship together. Openness now only means that the Core cannot
   reject bad data and cannot read its own data.

---

## 2 Vocabulary

- **Typed task resource** (or **resource**): a first-class record with a UUID key for
  one task type (`http_read`). It holds the name and all configuration fields. A
  per-integration service owns it.
- **Task**: the slim execution record. It exists only while deployed and is always bound
  to a rack.
- **Integration**: a hardware family and its schema/service unit: `ni`, `opc`,
  `labjack`, `modbus`, `ethercat`, `http`, `arc`, `pagerduty`, `slack`.
- **Draft**: a typed resource with no task and no snapshot flag; the only editable
  undeployed state.
- **Deploy / undeploy**: to deploy is to make a task on a rack for a resource; to
  undeploy is to delete that task. Undeploy never deletes the resource.
- **Action dispatch**: the shared update path behind schematic editing: a batch of
  schema-declared actions reduced atomically and broadcast over signals.
- **`config` reference**: the one stored link between the two records: an ontology ID
  (`<type>:<uuid>`) on the task that points to its parent resource. It is never null.
- **`empty` resource**: the minimal resource type: a name and nothing more. The parent
  of the scanners and the rack status task.

---

## 3 Principles

1. **The Core owns persisted data and its migrations.** Config shapes are persisted
   data; their schema and version chain belong to the Core, not to each client.
2. **Write the shape one time.** One Oracle schema generates the Go, TypeScript, Python,
   C++, and Protobuf code. Hand parsing is a defect.
3. **The specific depends on the generic.** Integration services compose the task
   service. The task service never learns integration names: the `config` reference is
   opaque data that it stores and indexes, and that only per-type consumers dereference.
4. **Closed set, loud failure.** Task types are internal dispatch keys; the Core rejects
   an unknown type. The one soft edge is the migration of pre-cutover data (§4.6).
5. **The resource is what users own.** The configuration is the durable artifact: users
   draft, snapshot, import, and export it. Execution is temporary and rack-bound:
   portable UUID identity on the resource, rack binding on the task.

---

## 4 Design

### 4.0 The split

```
┌─────────────────────────────┐          ┌──────────────────────────────┐
│ http_read (resource)        │          │ task (execution record)      │
│  key        uuid            │  config  │  key          uuid           │
│  name       string          │ ◄─────── │  rack         rack.Key       │
│  snapshot   bool            │          │  config       ontology.ID    │
│  endpoints  Endpoint[]      │          │  internal     bool           │
│  rate       telem.Rate      │          │  auto_start   bool           │
│  ...                        │          │  config_hash  uint64         │
│  task       uuid (resolved) │          │  (status via status service) │
└─────────────────────────────┘          └──────────────────────────────┘
```

The task stores a `config` field: the ontology ID of its parent resource
(`http_read:<uuid>`). This heterogeneous reference is the only stored link between the
two records, and it is never null. The resource's `task` field is not stored. The server
resolves it at retrieve time from a Gorp index on `config` (RFC 0034). The ontology
keeps a `parent_of` edge from the resource to the task for the Console tree. The server
stamps the edge in the same transaction that makes the task.

Invariant: **each task has exactly one parent resource.** The non-null `config` field
enforces it. The scanner tasks and the rack-status task of the Driver obey this rule:
the Driver makes an `empty` resource for them at startup. The task row does not need a
`type` field. The type part of the `config` ontology ID carries it.

### 4.1 The slim task

The task keeps only what execution needs:

| Field         | Why it stays                                                     |
| ------------- | ---------------------------------------------------------------- |
| `key`         | UUID identity; the target of `sy_task_cmd` and status            |
| `rack`        | the deployment target; a mutable field, not key bits             |
| `config`      | the ontology ID of the parent resource; never null; carries type |
| `internal`    | hides Driver-created tasks from users                            |
| `auto_start`  | execution behavior, lifted out of each per-type config           |
| `config_hash` | drift detection for deploy-on-start (§4.3)                       |

These fields leave the task: `name` (the resource owns it), `type` (the type part of the
`config` reference owns it), the embedded config object (the resource _is_ the config),
and `snapshot` (snapshots do not have task rows, §4.3). Status stays in the status
service, keyed by the ontology ID of the task. A draft has no status because nothing
executes.

The generic `/task/create` endpoint is not exposed. Users create resources through the
per-type endpoints; the Driver mints its internal tasks through the deploy path against
`empty` resources.

### 4.2 Schemas and services

**One Oracle schema file per integration**: `schemas/synnax/ni.oracle`, `opc.oracle`,
`labjack.oracle`, `modbus.oracle`, `ethercat.oracle`, `http.oracle`, plus `arc.oracle`
(amended), `pagerduty.oracle`, `slack.oracle` (the Slack integration, SY-3995, lands
directly in this shape), and the `empty` resource. A file defines the shared parts of an
integration (channel unions, scales, endpoints) and its task-type resources. We rejected
per-type files: the 19-variant AI channel union of NI and its scale/CJC unions are
shared across its five task types.

The NI schema draft (32 enums, 4 unions, 57 structs) is the start point. We adapt it so
that the task configs become root resources, not embedded config shapes. Shared
cross-integration bases (the `sample_rate` / `stream_rate` / `data_saving` read shapes
and the write shapes) live in a common task schema. The per-integration files extend it.
Oracle must first support the extension of a common shape across schema files. That is
groundwork for this RFC.

Each resource has the standard tags: `@ontology type "<task type>"`, `@retrieve`,
`@search`, `@create`, `@go migrate`, `@pb`. Each resource also declares an action set
for its edits — per-field and per-channel actions in the schematic and log shape. The
outputs go to `core/pkg/service/<integration>`, `client/ts`, `client/py`, and
`client/cpp`.

**Go services are per-integration packages**: `core/pkg/service/ni`,
`core/pkg/service/opc`, and so on, with the standard service anatomy, adjacent to the
current `arc` and `pagerduty` packages. Each package owns the Gorp tables, the writers,
the retrieve builders, the ontology registration, and the imex of its resources.

**Structural rules for the new schemas:**

- Maps with semantic keys become arrays of structs. The key becomes a field. The
  EtherCAT `auto_<pdo>` / `manual_<index>_<subindex>` channel maps are the canonical
  case. This change removes the caseconv corruption class at the root. We then delete
  the `??` fallback lookups.
- Validation splits into two tiers (§4.3): the shape tier is the decode into the
  generated types; the semantic tier covers required fields, bounds, and discriminants
  (generated from the schema) plus rules outside Oracle's reach (stream rate ≤ sample
  rate, port uniqueness), written in the Go service. Clients can restate the semantic
  rules as UI-side refinements.

### 4.3 Lifecycle

**Create.** `POST /<type>/create` makes a draft: a resource with no task. The resource
does not have and does not accept a rack.

**Edit.** Every edit is an action dispatch: the per-type writer reduces the batch
atomically in one transaction and broadcasts it over signals, the machinery behind
schematics. Editing is synchronous across clients, and the Console gets autosave and
undo/redo from the standard action path. The shape tier rejects a dispatch that does not
decode into the generated types, so the stored config is always well-typed. The semantic
tier reports diagnostics and never rejects: an incomplete draft commits, and the
diagnostics gate deploy and `start` only. A dispatch to a deployed resource behaves the
same; it also updates `config_hash` on the task row (found through the `config` index)
and fires the `sy_task_set` metadata refresh. Active tasks do not restart; the Console
shows the drift until the next `start`.

**Deploy.** Deploy is an execution transition, not an edit, so it stays outside the
action set. The per-type deploy operation receives the resource key and a rack key. In
one transaction, it checks the semantic diagnostics and the `integrations` list of the
rack, mints the task with its `config` reference, and stamps the ontology edge. It has
no validation logic of its own. A rack without `ni` is a hard error for an
`ni_analog_read` deploy. A deploy to a different rack writes the new `rack` value on the
task; the old Driver tears down, as the task draft/deploy design (SY-4488) specifies for
a rack move.

**Start / stop, deploy-on-start.** This RFC builds on the draft/deploy design and keeps
its behavior: `sy_task_set` is metadata-only, and the `start` command absorbs
configuration. `start` refuses a resource with outstanding semantic diagnostics. The
Core computes `config_hash` (xxhash64 over canonical bytes) when it writes the resource,
and stores the hash on the task row. On `start`, the Driver compares the hash with its
active instance and fetches the typed resource only on a mismatch; drift is the
difference between the stored and active hashes. Two points change relative to that
design: the input of the hash is the typed resource, and drafts are resources without
tasks, not tasks without racks (Resolved Decision 6).

**Undeploy / delete.** To delete the task is to undeploy: the execution record and its
status go away, the resource stays as a draft, and its resolved `task` field becomes
empty. To delete the resource is to also delete its task, found through the `config`
index (the Arc `deleteChildTasks` pattern). The delete action on a task entry in the
Console tree deletes the resource, because the resource is what the user sees.

**Snapshot.** A snapshot is a copy of the resource: a new UUID, `snapshot: true`, no
task row. The dispatch path rejects every action except `Rename` on a snapshot, the same
guard the schematic writer applies. Ranges refer to it exactly as they refer to task
snapshots now.

**Copy.** A copy of a resource is a per-type create from the fields of a current
resource. The `Copy` method of the task writer goes away with the config that it existed
to clone.

### 4.4 Wire format

Configs have strong types on each wire. `google.protobuf.Struct` and `Any` are gone.

- **Per-type Protobuf messages**, generated by the Oracle `@pb` output for each
  integration. The Driver switches on the type part of the task's `config` ontology ID
  and calls the generated client for that type; each retrieve returns the typed message.
  The `ErrTaskNotHandled` scan goes away, because dispatch is exact.
- **C++** consumes the generated proto types directly. We delete the `x::json::Parser`
  config layer in `driver/*/` for each integration when it cuts over.
- **TypeScript and Python** use the HTTP JSON/MessagePack transports with generated Zod
  schemas and Pydantic models against the per-type endpoints. The schema controls the
  wire field names, so the TypeScript case conversion is exact. The maps with semantic
  keys became arrays (§4.2), so the task schemas do not need `preserveCase` exits.

### 4.5 The stored link and the resolved `task` field

The task service stays unaware of integrations. The `config` field is opaque data to it:
an ontology ID that the service stores, indexes, and never dereferences. No registry and
no composed payload exist at the API layer. A consumer that holds a task goes to the
per-type endpoint that the `config` type names. A consumer that holds a resource reads
its `task` field, which the resource service resolves from the Gorp index on `config`.

The flow of the Driver: `sy_task_set` and the task row give it the slim task. On `start`
with a hash mismatch, the Driver fetches the typed resource from the per-type endpoint.
One fetch returns one typed message. The factories of the in-process Go driver (`arc`,
`pagerduty`, `slack`) receive the typed struct in the same way. They do not unmarshal
`EncodedJSON`.

### 4.6 Migration

A one-time startup migration converts each stored task:

1. For each task row, the migration dispatches on the legacy `type` string to the
   integration migrator. The migrator decodes the config through the version chain,
   repairs the semantic keys that camelCase conversion corrupted (restructuring the
   EtherCAT/OPC channel maps into arrays), creates the typed resource, writes the
   `config` reference on the task row, and makes the row slim.
2. Snapshot tasks convert to snapshot resources. The migration deletes their task rows
   and points the range references to the new resources.
3. Internal scanner and status tasks convert to `empty` resources. The
   create-if-not-exists path in the Driver startup moves to the deploy flow.
4. **Unknown type strings cause a warning and quarantine.** The row keeps an error
   status under a quarantine key, the Driver does not configure it, and the log reports
   it. The migration never drops a row silently.

The migration composes with the UUID re-key migration from the SY-4488 stack, which must
land first. The migration of this RFC assumes UUID task keys and rack-as-field.

### 4.7 Integration inventory

- `ni_analog_read`, `ni_analog_write`, `ni_digital_read`, `ni_digital_write`,
  `ni_counter_read`: schema from the NI draft (§4.2).
- `opc_read`, `opc_write`.
- `labjack_read`, `labjack_write`.
- `modbus_read`, `modbus_write`.
- `ethercat_read`, `ethercat_write`: channel maps become arrays.
- `http_read`, `http_write`.
- `arc`: retrofit — the standard edge replaces `config{arc_key}`. Arc creation is
  already server-side.
- `pagerduty_alert`, `slack_alert`: Go-side factories. Slack lands in this shape.
- `empty`: scanners (`*_scan`, `ni_scanner`) and rack status. Name only.

The scanner task types (`opc_scan`, `modbus_scan`, …) have no real configuration; they
become internal tasks with `empty` parents, not resource types of their own. The
`"Rack Status"` type string (with the space) and the other legacy strings go away with
the `type` field itself.

Per-type `StatusData` (the `errors[]` of NI, the read status of EtherCAT) gets a type in
the schema of each integration, threaded through the status details generic. The Console
already types these by hand. The schemas make it official.

---

## 5 Implementation phases

Prerequisite (external to this RFC): the SY-4488 stack — UUID task keys with rack as a
field, and deploy-on-start. The Console autosave work must target the typed resource in
Phase 9. It must not land against `task.config`.

Each numbered phase is a PR, or a short series where noted. At each boundary the tree
builds, the tests pass, and the product can ship. The order front-loads storage: the
typed tables and the migration land under an unchanged task envelope, so the Core owns
config shape and migrations well before any client changes.

1. **Oracle groundwork.** Cross-file extension of a common shape for the shared task
   config bases. Per-type endpoint generation, including the action dispatch endpoints.
   Support for an `ontology.ID` field with a Gorp index, for a resolved `task` field on
   resources, and for a resolved `config` field that composes a task's payload from the
   resource it references. This is pure generator work with generator tests. No schema
   consumes it yet.
2. **Schema authorship.** The per-integration `.oracle` files and their generated
   Go/TypeScript/Python/C++/Protobuf artifacts, not yet wired (the NI file adapted from
   the draft). One PR per integration or per small group. Each PR is a reviewable schema
   plus inert generated code.
3. **Gorp tables for every resource.** The per-integration service packages, the `empty`
   resource, the Gorp tables, the ontology registration, and the per-type retrieve
   endpoints. No production path writes to a table yet; Ginkgo suites exercise them
   directly. The tree stays green because the change is additive.
4. **The resolved `config` field.** The storage cutover. The task row stores the
   `config` ontology ID and drops the embedded config object, `name`, and `snapshot`.
   The task envelope does not change: the writer decomposes an incoming config into its
   typed resource, and retrieve recomposes the payload from that resource as a resolved
   field. The §4.6 startup migration and the live-write decomposer share one code path.
   From here the Core owns the config version chain and always serves the migrated
   shape. The integrations whose schemas restructured (the EtherCAT and OPC map keys,
   §4.2) change payload shape here, so the Console and Driver readers for those types
   move in the same release and the `??` fallback lookups go away with them. Resolved
   Decision 12 rejects a composed payload as an end state; here it is a compatibility
   shim that Phase 7 deletes.
5. **Server-side import and export.** The task imex reads and writes typed resources
   instead of flattening an opaque blob. This unblocks SY-4524.
6. **Client compat deletion.** One PR per client, no wire change. The Console loses its
   import ingesters and its private NI version chain, the Driver loses its legacy-shape
   parse paths, and the Python client loses its old-shape readers. Each client handles
   exactly one shape from here, because the Core guarantees it.
7. **API cutover.** The task payload slims to §4.1 and gains `auto_start`; the resolved
   compat fields go away. The per-type create and action dispatch endpoints with
   two-tier validation become the write path, which brings drafts and the deploy verb.
   It removes `/task/create` and the config/copy paths of the task writer, and routes
   the Go driver factories (arc, pagerduty) through typed payloads.
8. **Driver cutover.** One PR per integration wave. Each wave consumes the generated C++
   proto configs, deletes the `parser.field` config structs, and switches dispatch to
   the `config` ontology ID with a per-type retrieve for each fetch. The scanner startup
   moves to `empty`-resource deploys in the first wave.
9. **Console migration.** One PR per integration. The forms edit typed resources through
   action dispatch with generated Zod (the UI-only refinements stay), which carries
   autosave and undo/redo. The tabs key on resource UUIDs. The drift and deploy UX
   follows the draft/deploy design.
10. **Python rewiring.** The generated Pydantic resource models replace the hand-written
    config models. The wrappers in the style of `StarterStopperMixin` stay as sugar over
    create, deploy, and start.
11. **Arc + Slack alignment.** Complete the Arc retrofit (drop `config{arc_key}`, use
    the standard edge). Land `slack_alert` (SY-3995) directly in the new shape.

**Compatibility:** Phase 4 migrates persisted data and keeps the task envelope, so
clients keep working across it; only the restructured integrations shift payload shape,
and their readers move in that release. The envelope itself breaks at Phase 7, and the
Core, the Driver, and the Console ship that one together (lockstep releases, no window
of coexistence). There is no downgrade path across Phase 4 or Phase 7. This is the
standard position for storage migrations in this codebase.

---

## 6 Resolved decisions

1. **A config union inside `Task` — rejected.** It types the blob with much less
   machinery, and that trade is real. But it does not repair the model: no per-type
   ontology presence or permissions, no drafts, no portable UUID identity for ImEx, and
   snapshot and copy stay attached to execution.
2. **The earlier task/config split draft and its Phase 1 implementation — superseded.**
   It kept the task as the root, the open wire format, an embedded-blob fallback, and a
   three-release dual-write rollout. The stored `config` reference and the per-type
   endpoints replace all of that machinery (§4.5).
3. **The task as the root aggregate — rejected.** The portable artifact's identity stays
   rack-bound, drafts get no home, and snapshot and copy stay attached to execution. The
   resource as root follows the Arc precedent.
4. **`google.protobuf.Any` / `Struct` for configs — rejected.** Typed per-type messages
   only. `Any` returns dispatch to strings; `Struct` is the current state we remove.
5. **Per-type schema files — rejected** in favor of per-integration files, which keep
   the shared channel and scale unions adjacent to their users (§4.2).
6. **Draft tasks without racks (the original draft/deploy model) — replaced.** Two draft
   representations cannot exist together: a task always has a rack, and the resource
   without a task is the only draft. The deploy-on-start behavior stays.
7. **The node as a first-class deployment target — deferred.** Each node has exactly one
   embedded rack, so deploy-to-node is sugar; a polymorphic location type waits for the
   rack and device UUID re-key.
8. **A `task.name` mirror — rejected.** The resource owns the name; a synchronized
   mirror is a second source of truth that no consumer needs.
9. **Snapshot task rows — removed.** A snapshot is a frozen resource copy. Execution
   state has no place in it.
10. **The open task-type set — closed.** A third-party type needs a schema, a service,
    and a driver factory in any case; a validation bypass recreates today's problems.
    Tests register a test-only type behind the service config seam, not a wire bypass.
11. **Multi-deploy (one resource, many tasks) — future work.** Validation permits at
    most one task per resource. Fan-out changes the channel-write behavior and needs its
    own design.
12. **A resolved `config` payload on the task (registry + `oneof` envelope) —
    replaced.** The stored ontology ID gives the same answer as plain data, deletes the
    registry and envelope, and keeps the task service generic; deploy-on-start makes the
    config fetch lazy. Mutual stored references were also rejected: two stored links can
    drift apart.
13. **Config validation inside the deploy operation — replaced** by the two dispatch
    tiers (§4.3). A hard semantic gate on updates was also rejected: it would reject
    every incomplete intermediate state and make drafts uneditable.
14. **Deploy inside the action union — rejected.** Reducers are pure state-to-state
    maps, and the execution lifecycle (`start`, `stop`, undeploy) already lives outside
    the action stream. Undo history never contains a deployment.

---

## 7 What this RFC does not cover

- **Typed command args** (`start`, `stop`, `tare`, `test_connection`, `browse`):
  `Command.args` stays opaque. A follow-up can type them with the same union machinery.
- **The UUID re-key of statuses, racks, and devices**, and server-side device
  management. Status keys stay task-ontology-ID strings here.
- **Typed device `properties`.** The NodeId-keyed channel maps in device properties are
  the same corruption class as §4.2 and need the same array-of-structs treatment in an
  adjacent effort.
- **Protobuf as the TypeScript and Python wire.** Those clients stay on HTTP
  JSON/MessagePack. The snake↔camel removal effort tracks the larger proto migration.
- **Multi-deploy fan-out** (Resolved Decision 11).

---

## 8 Open questions

1. The final name of the `empty` resource (`empty`, `plain`, or `internal`).
2. The reporting surface for semantic diagnostics: the dispatch response, a status on
   the resource, or client-side recompute from the generated validation code.
3. The quarantine surface for unknown-type rows at migration (§4.6): the status variant,
   and whether the Console lists them.
4. The wave order of the Driver and Console cutovers across integrations in Phases 8–9.
