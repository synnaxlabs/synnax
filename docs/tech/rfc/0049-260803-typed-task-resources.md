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

# 0 - Summary

A task today is one record with an opaque `config` field: `map[string]any` in Go,
`json::object_t` in C++, `dict[str, Any]` in Python, `record.Unknown` in TS, and a
`google.protobuf.Struct` on the wire. The Core never parses it. Every integration
defines its config shape three separate times by hand — Console Zod, Python
pydantic, C++ `parser.field<T>()` — and the shapes drift.

This RFC inverts the model. Every task type becomes a first-class, strongly typed
resource with its own Oracle schema, ontology type, protobuf message, and service:
`ni_analog_read`, `opc_read`, `http_read`, and so on. The typed resource is the
root aggregate: it is uuid-keyed, owns the name and all configuration fields, and
is the unit users create, edit, snapshot, import, and export. The generic task
shrinks to a slim execution record — `{key, rack, internal, auto_start,
config_hash}` — that exists only while the resource is deployed to a rack. The
typed resource references its task; a resource without a task is a draft.

Task types become a closed set. The Core validates every configuration at the
create/update chokepoint using Oracle-generated validation, owns the config
migration chain, and serves the driver a strongly typed payload (a protobuf
`oneof`, never an `Any`). The driver and Console stop hand-parsing configs
entirely: the only parsing anywhere is Oracle-generated Zod, pydantic, and proto
code.

---

# 1 - Motivation

Five compounding problems, all rooted in config opacity:

1. **No single source of truth.** OPC UA's read config is defined in
   `console/src/feature/opc/task/types.ts`, `client/py/synnax/opcua/types.py`, and
   `driver/opc/read_task.h` — three hand-written copies that must agree on every
   field name, default, and bound, with no tooling to say when they don't. NI
   multiplies this by ~50 channel variants.
2. **The Core cannot validate or migrate.** `Task.Config` is
   `msgpack.EncodedJSON` (`core/pkg/service/task/versions/v1/types.gen.go`), stored
   as raw JSON bytes and passed through untouched. Config schema evolution happens
   client-side (the Console carries a private version chain in
   `console/src/feature/ni/task/types/v0.ts`) or not at all: the Python client
   silently misreads old shapes, and the driver fails to parse them.
3. **Wire case-conversion corrupts semantic keys.** Task config is not
   `preserveCase`-wrapped, so the TS codec (`x/ts/src/binary/codec.ts`) camelCases
   record keys that are values, not identifiers — OPC NodeIds, EtherCAT
   `manual_<index>_<subindex>` map keys. The `??` case-fallback lookups in
   `console/src/feature/opc/task/Read.tsx` and
   `console/src/feature/ethercat/task/types.ts` exist solely to paper over
   already-corrupted persisted data.
4. **Server-side import/export is blocked.** RFC 0039 moved metadata import into
   the Core, but `core/pkg/service/task/imex.go` must flatten an opaque blob and
   says so: the flattening is temporary "once task configs are strongly typed".
   SY-4524 (server-side task import) is blocked on this RFC.
5. **The open set has no remaining value.** Task types were an open set so that
   drivers could evolve independently (RFC 0017). In practice every type string is
   first-party, the driver and Core ship together, and openness now only means the
   Core can neither reject garbage nor understand its own data.

---

# 2 - Vocabulary

- **Typed task resource** (or just **resource**) — a first-class, uuid-keyed record
  for one task type (`http_read`, `ni_analog_read`), holding the name and all
  configuration fields inline. Owned by a per-integration service.
- **Task** — the slim execution record. Exists only while deployed; always bound to
  a rack.
- **Integration** — a hardware family and its schema/service unit: `ni`, `opc`,
  `labjack`, `modbus`, `ethercat`, `http`, `arc`, `pagerduty`, `slack`.
- **Draft** — a typed resource with no task. The only undeployed state.
- **Deploy / undeploy** — minting a task on a rack for a resource / deleting that
  task. Undeploy never deletes the resource.
- **Resolved task payload** — the wire shape the driver consumes: the slim task
  joined with its resource's typed config, composed at the API layer (RFC 0041's
  resolved-field pattern).
- **`empty` resource** — the minimal resource type (a name and nothing else)
  backing internal tasks with no configuration: scanners and the rack status task.

---

# 3 - Principles

1. **Core owns persisted data and its migrations.** Config shapes are persisted
   data; their schema and version chain belong to the Core, not to each client.
2. **Write the shape once.** One Oracle schema generates Go, TS, Python, C++, and
   protobuf. Clients and the driver consume generated code; hand parsing is a
   defect.
3. **Specific depends on generic.** Integration services compose the task service.
   The task service never learns integration names; type dispatch happens above it,
   in an explicitly wired registry at the API composition layer.
4. **Closed set, loud failure.** Task types are internal dispatch keys. An unknown
   type is a composition bug and is rejected — never stored, never silently
   dropped. The one soft edge is migration of pre-existing data (§4.6).
5. **The resource is what users own.** Configuration is the durable artifact —
   drafted, snapshotted, imported, exported. Execution is transient and rack-bound.
   The data model mirrors this: portable uuid identity on the resource, rack
   binding on the task.

---

# 4 - Design

## 4.0 - The split

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

The resource carries an optional `task` field (the task's uuid). The ontology
mirrors it with a `parent_of` edge resource → task, stamped server-side in the
same transaction that mints the task — the pattern PR #2496 established for Arc,
generalized to every integration.

Invariant: **every task has exactly one parent resource.** The driver's scanner
and rack-status tasks satisfy it by creating an `empty` resource at startup.
Because the edge always exists, the task row needs no `type` field: the resource's
ontology type is the task's type, and the `oneof` case in the resolved payload
carries it on the wire.

## 4.1 - The slim task

The task keeps exactly what execution needs:

| Field         | Why it stays                                                      |
| ------------- | ----------------------------------------------------------------- |
| `key`         | uuid identity (PR #2603), targeted by `sy_task_cmd` and statuses  |
| `rack`        | deployment target; mutable field, not key-encoded (PR #2603)      |
| `internal`    | hides driver-created tasks from users                             |
| `auto_start`  | execution semantics, lifted out of every per-type config          |
| `config_hash` | drift detection for deploy-on-start (§4.3)                        |

Dropped from the task: `name` (owned by the resource), `type` (owned by the
ontology edge and the wire `oneof`), `config` (the resource *is* the config), and
`snapshot` (snapshots no longer have task rows at all, §4.3). Status stays keyed
by the task's ontology ID in the status service; a draft has no status because
there is nothing executing.

The generic `/task/create` endpoint is not exposed. Nothing needs it: users create
resources through per-type endpoints, and the driver's internal tasks are minted
through the deploy path against `empty` resources.

## 4.2 - Schemas and services

**One Oracle schema file per integration**: `schemas/synnax/ni.oracle`,
`opc.oracle`, `labjack.oracle`, `modbus.oracle`, `ethercat.oracle`,
`http.oracle`, plus `arc.oracle` (amended), `pagerduty.oracle`, `slack.oracle`
(RFC 0045-slack lands directly in this shape), and the `empty` resource. A file
defines that integration's shared pieces (channel unions, scales, endpoints) and
its several task-type resources. Per-type files were rejected: NI's 19-variant AI
channel union and scale/CJC unions are shared across its five task types, and
splitting them forces heavy cross-file imports for no isolation gain.

The NI schema authored in PR #2433 (32 enums, 4 unions, 57 structs) is the
starting point, adapted so task configs become root resources rather than
embedded config shapes. Shared cross-integration bases (`sample_rate` /
`stream_rate` / `data_saving` read shapes, write shapes) live in a common task
schema that per-integration files extend — this requires Oracle support for
extending a common shape across schema files, which is groundwork for this RFC.

Each resource carries the standard tags: `@ontology type "<task type>"`,
`@retrieve`, `@search`, `@create`, `@go migrate`, `@pb`, with outputs to
`core/pkg/service/<integration>`, `client/ts`, `client/py`, and `client/cpp`.

**Go services are per-integration packages**: `core/pkg/service/ni`,
`core/pkg/service/opc`, etc., following the standard service anatomy, alongside
the existing `arc` and `pagerduty`. Each owns its resources' gorp tables, writers,
retrieve builders, ontology registration, and imex.

**Structural rules for the new schemas:**

- Semantic-keyed maps become arrays of structs with the key as a field. EtherCAT's
  `auto_<pdo>` / `manual_<index>_<subindex>` channel maps are the canonical case;
  restructuring kills the caseconv corruption class at the root, and the `??`
  fallback lookups are deleted.
- Cross-field validation that is expressible in Oracle (bounds, discriminants,
  required-ness) lives in the schema. Rules beyond its reach (stream rate ≤ sample
  rate, port uniqueness) live in the Go service's validation at the same
  chokepoint, and clients may re-state them as UI-side refinements.

## 4.3 - Lifecycle

**Create.** `POST /<type>/create` creates a draft: a resource with no task. No
rack is required or accepted on the resource itself.

**Deploy.** A deploy operation on the per-type endpoint takes the resource key and
a rack key. In one transaction the service validates the config, checks the rack's
`integrations` list (a rack without `ni` hard-errors an `ni_analog_read` deploy —
it can never succeed), mints the task, writes the resource's `task` field, and
stamps the ontology edge. Re-deploying to a different rack rewrites the task's
`rack` field; the old driver tears down per the draft/deploy RFC's rack-move flow.

**Start / stop, deploy-on-start.** This RFC builds on the task draft/deploy RFC
(PR #2595) and its SY-4488 implementation stack, and keeps its semantics:
`sy_task_set` is metadata-only; the `start` command absorbs configuration. The
Core computes `config_hash` (xxhash64 over canonical bytes) whenever the resource
is written, and stores it on the task row. On `start`, the driver compares the
hash against its running instance, re-fetching the resolved payload only on
mismatch. Drivers report the running hash in status details; drift is derived by
comparison. What changes relative to that RFC: the hash's input is the typed
resource, and drafts are taskless resources rather than rack-less tasks (§6.6).

**Edit while deployed.** Writing the resource updates `config_hash` on its task
row inside the same transaction, which fires the existing gorp-observe /
`sy_task_set` path as a metadata refresh. Running tasks are not restarted; the
Console surfaces drift until the user redeploys via `start`.

**Undeploy / delete.** Deleting the task is undeploying: the execution record and
its status disappear, the resource remains as a draft with `task` cleared.
Deleting the resource cascades to its task (the Arc `deleteChildTasks` pattern).
The Console tree's delete on a task entry deletes the resource — the user-facing
object.

**Snapshot.** Snapshotting copies the resource: a new uuid, `snapshot: true`,
immutable via the same guard that protects snapshots today, referenced from ranges
exactly as task snapshots are now. No task row is created for a snapshot — a
frozen config has no execution, so nothing about it belongs in the task table.

**Copy.** Copying a resource is a per-type create from an existing resource's
fields. The task writer's `Copy` disappears with the config it existed to clone.

## 4.4 - Wire format

Configs are strongly typed on every wire. `google.protobuf.Struct` and `Any` are
gone.

- **Per-type protobuf messages**, generated by Oracle's `@pb` output per
  integration. The resolved task payload used by gRPC (`TaskRetrieve`, the
  driver's fetch) is the slim task plus a `oneof` over every resource config
  message. The `oneof` case is the task type on the wire; the driver switches on
  it and hands the typed message to the factory. `ErrTaskNotHandled` scanning
  disappears — dispatch is exact.
- **C++** consumes the generated proto types directly. The `x::json::Parser`
  config-parsing layer in `driver/*/` is deleted per integration as it cuts over.
- **TS and Python** use the HTTP JSON/msgpack transports with Oracle-generated
  Zod schemas and pydantic models against the per-type endpoints. Wire field
  names are schema-driven, so TS case conversion is exact; with semantic-keyed
  maps restructured into arrays (§4.2), no `preserveCase` escape hatches are
  needed in task schemas.

## 4.5 - The resolved join

The task service stays ignorant of integrations. At the API composition layer, an
explicitly wired registry maps ontology type → resource service, and
`TaskRetrieve` uses it to attach each task's typed config to the response — the
resolved-field pattern from RFC 0041, already proven for `Arc.Task` in PR #2496
(inverted here: task resolves its parent resource via the ontology edge). This is
the one place generic-over-specific dispatch exists, and it is composed at the
wiring site in `layer.go`, never via self-registration.

The driver's flow is unchanged in shape: one round trip returns everything needed
to configure. The in-process Go driver's factories (`arc`, `pagerduty`, `slack`)
receive the typed struct instead of unmarshalling `EncodedJSON`.

## 4.6 - Migration

A one-time startup migration converts every stored task:

1. For each task row, dispatch on the legacy `type` string to the matching
   integration migrator. It decodes the config through the integration's version
   chain (normalizing shapes the Console migrated client-side until now, and
   repairing camelCase-corrupted semantic keys as EtherCAT/OPC channel maps are
   restructured into arrays), creates the typed resource, links it, and slims the
   task row.
2. Snapshot tasks convert to snapshot resources; their task rows are deleted and
   range references re-pointed.
3. Internal scanner/status tasks convert to `empty` resources; the driver's
   startup create-if-not-exists path moves to the deploy flow.
4. **Unknown type strings warn and quarantine.** The row is preserved under a
   quarantine key with an error status, excluded from driver configuration, and
   logged loudly. It is not dropped — but it is also not silently kept alive.

The migration composes with the uuid re-keying migration from PR #2603, which must
land first; this RFC's migration assumes uuid task keys and rack-as-field.

## 4.7 - Integration inventory

| Resource types                              | Notes                             |
| ------------------------------------------- | --------------------------------- |
| `ni_analog_read/_write`,                    | schema from PR #2433              |
| `ni_digital_read/_write`, `ni_counter_read` |                                   |
| `opc_read`, `opc_write`                     |                                   |
| `labjack_read`, `labjack_write`             |                                   |
| `modbus_read`, `modbus_write`               |                                   |
| `ethercat_read`, `ethercat_write`           | channel maps → arrays             |
| `http_read`, `http_write`                   |                                   |
| `arc`                                       | retrofit: `config{arc_key}`       |
|                                             | replaced by the standard edge;    |
|                                             | PR #2496 already moved creation   |
|                                             | server-side                       |
| `pagerduty_alert`, `slack_alert`            | Go-side factories; Slack lands in |
|                                             | this shape                        |
| `empty`                                     | scanners (`*_scan`, `ni_scanner`) |
|                                             | and rack status; name only        |

Scanner task types (`opc_scan`, `modbus_scan`, …) carry no real configuration
today; they become `empty`-backed internal tasks rather than getting dedicated
resource types. The `"Rack Status"` type string (with the space) and other legacy
strings disappear with the `type` field itself.

Per-type `StatusData` (NI's `errors[]`, EtherCAT's read status) is typed in each
integration's schema and threaded through the status details generic — the
Console already hand-types these; the schemas make it official.

---

# 5 - Implementation Phases

Prerequisite (external to this RFC): the SY-4488 stack — uuid task keys with rack
as a field (#2603) and deploy-on-start (#2604). The Console autosave PR (#2605)
re-targets the typed resource in Phase 6 rather than landing against `task.config`.

Each numbered phase is a PR (or a short series where noted); every boundary
leaves the tree green and shippable.

1. **Oracle groundwork.** Cross-file common-shape extension for the shared task
   config bases; per-type endpoint generation for resources carrying a `task`
   reference; `oneof` emission for the resolved payload. Pure generator work,
   covered by generator tests, no schema consumers yet.
2. **Schema authoring.** The per-integration `.oracle` files and their generated
   Go/TS/Py/C++/pb artifacts, unwired (the NI file adapted from #2433; one PR per
   integration or small groups — each is reviewable schema + inert generated
   code).
3. **Core services, additive.** Per-integration service packages, the `empty`
   resource, per-type endpoints, the deploy verb, and the API-layer registry +
   resolved join — all live alongside the untouched task shape. Nothing consumes
   them in production yet; Ginkgo suites exercise them directly. Green because
   additive.
4. **Core cutover.** The atomic PR: slim the task row (drop `type`/`name`/
   `config`/`snapshot`, add `auto_start`), run the §4.6 startup migration, route
   the Go driver factories (arc, pagerduty) through typed payloads, remove
   `/task/create` and the task writer's config/copy paths, and re-point task imex
   (unblocking SY-4524). Big by necessity — the migration and the shape change
   cannot be split without a half-wired intermediate state — but Phases 1–3 keep
   it to the cutover itself.
5. **Driver cutover.** One PR per integration wave: consume the generated C++
   proto configs, delete the `parser.field` config structs, switch dispatch to the
   `oneof`. Scanner startup moves to `empty`-resource deploys in the first wave.
6. **Console migration.** Its own phase, one PR per integration: forms bind to
   typed resources through generated Zod (UI-only refinements retained), tabs key
   on resource uuids, autosave (#2605's flow) targets the resource, drift and
   deploy UX per the draft/deploy RFC. The `??` case-fallback lookups and the
   client-side NI version chain are deleted.
7. **Python rewire.** Generated pydantic resource models replace the hand-written
   config models; the `StarterStopperMixin`-style wrappers survive as sugar over
   create/deploy/start.
8. **Arc + Slack alignment.** Complete the Arc retrofit on top of #2496 (drop
   `config{arc_key}`, ride the standard edge) and land `slack_alert` (SY-3995)
   directly in the new shape.

**Compatibility:** the wire breaks for task payloads at Phase 4; Core, driver, and
Console ship the cutover together (lockstep releases, no coexistence window).
Persisted data migrates forward automatically; there is no downgrade path across
Phase 4, which is the standard posture for storage migrations in this codebase.

---

# 6 - Resolved Decisions

1. **Config union inside `Task` — rejected.** The alternative to this whole RFC:
   keep one task table and type `config` as an Oracle discriminated union. It
   yields typed configs with far less machinery, and that trade is real. Rejected
   because it types the blob without fixing the model: no per-type ontology
   presence or permissions, no drafts, no portable uuid identity for imex (task
   keys are rack-history-bound), and snapshot/copy stay welded to execution.
2. **The task/config split RFC (PR #2471) and its Phase 1 (#2472) —
   superseded.** That draft split the same data but kept the task as root, the
   wire format unchanged (config as a composed resolved field), an embedded-blob
   fallback for unregistered types, and a three-release dual-write rollout. Its
   caution was the point — and is why it's superseded: preserving the wire freezes
   the open set and the opaque payload this RFC exists to remove. The
   provider-registry idea survives at the API composition layer (§4.5); the
   writer-dispatch, dual-write, and fallback machinery does not.
3. **Task points at its config (option 2) — rejected.** Puts a per-type reference
   on the generic row, forcing the task service to resolve specifics (wrong
   dependency direction), keeps identity rack-bound for the portable artifact, and
   leaves drafts homeless. The typed resource pointing at its task matches the Arc
   precedent and keeps the specific→generic arrow.
4. **`google.protobuf.Any` / `Struct` for configs — rejected.** Typed `oneof`
   only. `Any` re-introduces stringly-typed dispatch and defeats the point;
   `Struct` is the status quo being removed.
5. **Per-type schema files — rejected** in favor of per-integration files, which
   keep shared channel/scale unions next to their users (§4.2).
6. **Rack-less draft tasks (draft/deploy RFC model) — replaced.** Two draft
   representations (rack-less task, taskless resource) cannot coexist. A task now
   always has a rack; the taskless resource is the only draft. The draft/deploy
   RFC's deploy-on-start, hash, and metadata-only-set semantics are kept.
7. **Node as a first-class deployment target — deferred.** Every node has exactly
   one embedded rack, so deploy-to-node is sugar resolving node → embedded rack at
   the deploy endpoint. A polymorphic location type waits for the rack/device uuid
   re-keying follow-up.
8. **`task.name` mirror — rejected.** The resource owns the name; generic task
   listings read through the resolved join. A synced mirror is a second source of
   truth with no consumer that can't use the join.
9. **Snapshot task rows — removed.** A snapshot is a frozen resource copy;
   execution state has no business in it.
10. **Open task-type set — closed.** Third-party task types would need a schema,
    service, and driver factory anyway; an escape hatch that skips validation
    reproduces today's problems for exactly the tasks most likely to be malformed.
    Test suites use a registered test-only type behind the service config seam,
    not a wire bypass.
11. **Multi-deploy (one resource, many tasks) — future work.** The `task` field is
    0..1 by validation; fan-out changes channel-write semantics and deserves its
    own design.

---

# 7 - What This RFC Does Not Cover

- **Typed command args** (`start`/`stop`/`tare`/`test_connection`/`browse`):
  `Command.args` stays opaque; a follow-up can type per-command args with the same
  union machinery.
- **UUID re-keying of statuses, racks, and devices**, and moving device management
  server-side. Status keys remain task-ontology-ID strings here.
- **Device `properties` typing.** The NodeId-keyed channel maps inside device
  properties are the same corruption class as §4.2 and should get the same
  array-of-structs treatment in a sibling effort.
- **Protobuf as the TS/Python wire.** Those clients stay on HTTP JSON/msgpack;
  the broader proto migration is tracked by the snake↔camel removal effort.
- **Multi-deploy fan-out** (Resolved Decision 11).

---

# 8 - Open Questions

1. The `empty` resource's final name (`empty` vs `plain` vs `internal`).
2. The deploy verb's endpoint shape: `POST /<type>/deploy` vs a `rack` parameter
   on create/update. Parameter-level choice; the transaction semantics in §4.3
   hold either way.
3. Quarantine surfacing for unknown-type rows at migration (§4.6): status variant
   and whether the Console lists them.
4. Driver/Console cutover wave order across integrations in Phases 5–6.
