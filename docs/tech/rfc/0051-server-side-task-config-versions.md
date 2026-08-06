# 51 Server-side task configuration versions and import

- **Author**: Patrick Dotson
- **Date**: 2026-08-05
- **Related**: [RFC 0005 - MVP](0005-ontology.md),
  [RFC 0017 - General purpose device drivers](0017-drivers.md),
  [RFC 0027 - Oracle schema system](0027-oracle-schema-system.md),
  [RFC 0033 - Oracle migration system](0033-oracle-migrations.md),
  [RFC 0034 - Gorp in-memory indexes](0034-gorp-indexes.md),
  [RFC 0039 - Server-side metadata import/export](0039-server-side-import-export.md),
  [RFC 0042 - Core structure refactor](0042-core-structure-refactor.md),
  [RFC 0043 - Oracle support for struct unions](0043-oracle-struct-unions.md), and
  [RFC 0048 - Oracle predecessor-chain type versioning](0048-oracle-predecessor-chain-versioning.md).

## 0 Summary

A task holds its configuration as an opaque blob: `msgpack.EncodedJSON` in Go, a
`google.protobuf.Struct` on the wire. The Core stores the bytes and never reads them.
Every limitation follows from that. The Core cannot migrate a stored config, cannot
import one, and cannot reject a malformed one. Each client carries its own copy of every
config shape and its own version chain.

This RFC gives the Core a typed copy of every task configuration without changing what
clients send or receive. Each task type gets an Oracle schema and its own Gorp table.
The stored configuration moves out of the task row into that table, and the config
record becomes a parent of the task. Both `type` and `config` become resolved fields on
the task: the task row stores neither, and the task service composes them from the
config record on retrieve and decomposes them on write. The task payload keeps its
current shape on every wire.

Two capabilities follow. The Core owns the version chain for every config type, so
stored configs migrate server-side and clients only ever see the latest shape. Each
config type becomes an `imex.ImportExporter`, which unblocks server-side task import
(SY-4524) and lets export encode a typed struct instead of flattening a blob.

Task types become a closed set. Every type has a schema, and the Core rejects a type it
does not know rather than storing an untyped blob for it.

The task keeps its current key, so this RFC is independent of the SY-4488 UUID re-key
and composes with it in either order.

---

## 1 Motivation

1. **The Core cannot migrate persisted data**: Config shapes are persisted data, but
   their version chains live in clients. The Console holds a private NI chain
   (`console/src/feature/ni/task/types/v0.ts` and `v1.ts`), the Python client misreads
   old shapes, and the Driver fails to parse them.
2. **Server-side import is blocked**: RFC 0039 moved metadata import into the Core, but
   the task service implements only `imex.Exporter`, and that exporter flattens an
   opaque map into the envelope body. SY-4524 waits on a config the Core can read.
3. **A config shape has no single definition**: Each one is written by hand three times,
   in Console Zod, Python Pydantic, and C++ `parser.field<T>()`. No tool reports when
   the copies disagree. NI multiplies this across roughly 50 channel variants.
4. **The Core cannot reject bad data**: A malformed config is accepted, stored, and
   fails later at the Driver, far from the client that wrote it.

---

## 2 Vocabulary

- **Config record**: The typed row that holds one task's configuration. It lives in a
  Gorp table specific to the task type and has its own UUID key.
- **Resolved field**: A field the API returns but no row stores. The service computes it
  on retrieve. `type` and `config` both become resolved fields on the task.
- **Decompose**: To split an incoming task payload into a task row and a config record.
- **Compose**: To rebuild the resolved fields from a config record on retrieve.

---

## 3 Principles

1. **The Core owns persisted data and its migrations**: A config shape is persisted
   data, so its schema and version chain belong to the Core, not to each client.
2. **Write the shape one time**: One Oracle schema per task type generates the Go,
   TypeScript, Python, C++, and Protobuf code. Hand parsing is a defect.
3. **The client contract does not move**: This RFC changes storage and adds server-side
   behavior. The task payload on the wire keeps its shape, field for field.
4. **A closed set, loudly enforced**: Every task type has a schema. Task types are
   first-party dispatch keys, and the Core and the Driver ship together, so an unknown
   type is a bug and the Core rejects it.

---

## 4 Design

### 4.0 The storage split

```
┌──────────────────────────┐                   ┌──────────────────────────┐
│ ni_analog_read           │    parent_of      │ task                     │
│  key         uuid        │ ────────────────► │  key      uint64         │
│  sample_rate telem.Rate  │                   │  name     string         │
│  channels    AIChannel[] │                   │  internal bool           │
│  ...                     │                   │  snapshot bool           │
│  @go version 0           │                   │                          │
└──────────────────────────┘                   └──────────────────────────┘
              │                                            │
              └──────── compose on retrieve ───────────────┘
                        decompose on write
                                 │
                                 ▼
              Task.Type + Task.Config — unchanged on the wire
```

The task row stores neither `type` nor `config`. The config record holds the
configuration, the parent relationship carries the link, and the type is a property of
that parent rather than a column the task repeats.

### 4.1 Schemas and services

**One Oracle schema file per integration**: `schemas/synnax/ni.oracle`, `opc.oracle`,
`labjack.oracle`, `modbus.oracle`, `ethercat.oracle`, `http.oracle`, `arc.oracle`
(amended), `pagerduty.oracle`, and `rack.oracle`. A file defines the shared parts of an
integration — channel unions, scales, endpoints — and one config type per task type. We
rejected per-type files: the 19-variant AI channel union of NI and its scale and CJC
unions are shared across its five task types.

The NI schema draft (32 enums, 4 unions, 57 structs) is the start point. Shared
cross-integration bases, the `sample_rate` / `stream_rate` / `data_saving` read shape
and the write shape, live in a common task schema that the per-integration files extend.
Oracle already supports this: struct and union bases resolve across schema files in the
analyzer and all four generators, covered by generator tests.

Each config type declares `@ontology type "<task type>"`, `@go version`, `@go migrate`,
and `@pb`. The type name is the task type string, so `ni_analog_read` names the schema
type, the ontology type, and the Gorp table.

**Go services are per-integration packages**: `core/pkg/service/ni`,
`core/pkg/service/opc`, and so on, adjacent to the current `arc` and `pagerduty`
packages. Each owns the Gorp table, the writer, and the retrieve builder for its config
types. The task service composes them; it never learns integration names.

### 4.2 The config relationship

The link is an ordinary parent relationship. The config record is the parent and the
task is the child:

```go
w.otgWriter.DefineRelationships(
    ctx, configID, ontology.RelationshipTypeParentOf, taskID,
)
```

No new relationship type is needed. The existing tree walk and the `parent_of` index
both apply unchanged. Cardinality is exactly one config record per task, enforced in the
task writer, and the relationship is defined and deleted in the same transaction as the
task row, so no retrieve observes a task without its config.

The edge is additive: a task keeps the group parent the writer gives it today and gains
the config record as a second one. The tree shape does not change, because the config
record has no parent of its own and no walk from the root reaches it.

A task therefore has more than one parent, and every caller that reads a task's parents
must filter by ontology type. `Task.snapshottedTo` in the TypeScript client returns the
first parent it gets, so it must filter to `range`. The ontology cache of that client
also gives a `parentID` helper that returns one parent, which makes it unsafe for a
task. Phase 4 audits the parent readers in every client.

Internal tasks gain an ontology resource. The task writer creates none for them today,
so a scanner has nothing to relate and its `type` cannot resolve. Phase 4 removes that
early return: every task gets a resource and a config edge, internal or not.

The config record also gains a UUID that survives export and import, and an ontology
presence that per-type endpoints and access policies can name later.

### 4.3 Resolving `type` and `config`

The config parent is an `ontology.ID` of the form `ni_analog_read:<uuid>`, so it already
carries the task type. A task has more than one parent (§4.2), so the reader selects the
one whose ontology type is a registered config type, never the first parent it gets.
Resolving `type` then reads that parent's type part; resolving `config` reads the record
and encodes it. Exactly one parent matches, which §4.2 enforces on write.

Three call sites must resolve, not just the retrieve builder:

- **Retrieve** composes both fields into every returned task.
- **`OnChange` and `OpenNexter`** compose before emitting, so the ontology resource and
  the search index carry the type. `SearchableFields` still reports `type`, and search
  by type keeps working because the indexed resource is built from the composed payload.
- **The `sy_task_set` metadata path** composes, so the Driver still reads a type it can
  dispatch on.

Resolution is batched per operation, not per task: a retrieve resolves the parents for
the whole result set in one ontology query. The Driver's `sy_task_set` path and the
Console's task list both read many tasks at a time, and a per-task lookup would put a
query per task on a hot path.

**Retrieving by type** queries the relationship index for parents whose ontology type is
the requested task type, rather than filtering a task column. The relationship key
encodes `From`, type, and `To`, so this is an index scan, not a table walk.

### 4.4 Composition and decomposition

The task service is the only place that splits and rejoins a payload.

**Decompose (write)**: `Create` and `Update` take a task payload with an embedded config
and a `type`, exactly as they do today. The writer looks up the type in its config
registry, decodes the blob into the generated Go struct, validates it, writes the config
record, and defines the relationship, all in the caller's transaction. An unknown type
is a validation error on the write, and so is a config that does not decode.

**Compose (read)**: The reader resolves the config parent, decodes the record, and
stamps `type` and `config` back onto the payload. §4.3 names the call sites.

A decode failure on retrieve means a corrupt record, not a stale one: the record was
validated before storage, and the startup rewrite keeps stored versions current. The
reader reports it as an internal error rather than serving a partial task.

### 4.5 Versions and migrations

Each config type carries its own `@go version` and its own `versions/vN` chain, the
standard Oracle mechanism (RFC 0033, RFC 0048). Nothing about the chain is
task-specific: a config type is an ordinary versioned Oracle type that happens to be
keyed by a task type string.

Stored records migrate at startup, the standard Gorp mechanism: a version bump ships a
bootup pass that rewrites its table to the current shape. Decode therefore always reads
the current version, no record carries a version stamp, and the Core always serves the
latest shape, which is the property the clients depend on.

Clients delete their own chains. The Console loses its NI `v0`/`v1` types, the Python
client loses its old-shape readers, and the Driver loses the tolerance it carries for
configs it cannot parse. None of this is a wire change: the field is the same field,
carrying a shape the client already understands.

### 4.6 Import and export

The task service today implements `imex.Exporter` and nothing else, and its `Export`
flattens `map[string]any` into the envelope body with `type` and `name` stamped on top.
The flattening is deliberate — a task file is readable as a config — and this RFC keeps
that body shape. What changes is the source.

The two halves land in different services, because the registry routes them by different
keys. `Export` routes by the ontology type of the ID it is given, and `Import` routes by
the type string the envelope carries.

- **The task service stays an `imex.Exporter`**: It keeps the `task` export
  registration, resolves the task's type, asks that type's service for the encoded
  config record, and stamps `type` and `name` on top. It gains no import half.
- **Each config type gets an `imex.ImportExporter`**: The per-integration service
  registers one for every type it owns, so an `ni_analog_read` envelope routes straight
  to it. It decodes and migrates the body, then writes the config record, the task row,
  and the relationship in the supplied transaction. Its export half serves the config
  record on its own UUID.

An envelope naming an unknown type fails the import with a clear error, because no
importer is registered under that string.

The registration keys then converge. `imex` keys exporters by ontology resource type but
importers by a plain string, and the two differ only because a task type is not an
ontology type today. This RFC makes every task type an ontology type, so importers key
on `ontology.ResourceType` as well, and the asymmetric-registration indirection retires
with it: `Importer.Type` and the `ImporterType` lookup behind it exist to map a
fine-grained string onto a coarse type, and the enforcer can read the registration key
instead.

Envelope versions follow the unified numbering, one chain per config type. The coarse
`task` ontology type carries no version of its own. The service that encodes the config
stamps its `@go version` on the envelope, so `ni_analog_read` and `opc_read` files
version independently and the single `task.Version` constant stamped on every task file
today is deleted. The task row contributes only `type` and `name` to the body, and the
envelope already carries both as its routing key and its identity, so nothing outside
the config needs a version.

### 4.7 Lifecycle

Every operation stays where it is; only the transaction contents grow.

- **Create**: Decompose, write the task row and the config record, define the
  relationship.
- **Update**: Rewrite the config record in place. The task key and the relationship do
  not change.
- **Delete**: Delete the task row, the config record, and the relationship together.
  `Writer.Delete` today deletes only the task's own ontology resource, which drops the
  task's relationships but reaches no parent, so this is new work. The rule is positive:
  delete the one parent whose ontology type is a registered config type, and leave every
  other parent alone — the group the task hangs under, the range a snapshot is attached
  to, and any parent a later feature adds. Deleting the ontology resource does not
  remove the record from its per-type Gorp table, so the config service deletes that row
  in the same transaction.
- **Copy**: Copy the config record under a new UUID and relate it to the new task.
  `Writer.Copy` today defines a resource for the new key and no relationship at all, so
  it gains both the record copy and the edge. Without the edge the copied task resolves
  no `type`, which fails the retrieve rather than degrading it.
- **Snapshot**: A snapshot task gets its own frozen config record. Snapshots never share
  a record with a live task.
- **Rename**: Task row only. The name is not part of the config.

### 4.8 Migration of stored tasks

A one-time startup migration walks every stored task. It decodes the blob through the
legacy chain, writes the config record, defines the relationship, and clears `type` and
`config` from the task row.

A user task the migration cannot convert — an unknown type string, or a config that
fails to decode — is quarantined: the task row and its raw blob are preserved untouched,
the task is not served or configured, and the log names it once. The migration never
drops a user config and never writes a partial record. Quarantine is a loud failure that
an operator resolves, not a mode the system runs in.

An internal task is deleted instead. Its configuration holds no operator intent and the
Driver recreates it on boot, so quarantining one would keep a dead row that nothing can
repair. This is the path the stale `heartbeat` and `Rack Status` rows of §4.10 take.

### 4.9 Access control

`ontology.ResourceType` is a closed generated enum, so every config type adds a member
to it. Phase 3 extends the enum with the §4.10 inventory and regenerates the four
clients.

Each new member is a nameable RBAC object, and a policy that grants `task` today grants
nothing on `ni_analog_read`. The task endpoints keep checking `task` alone: a config
record reached through its task inherits the decision made for that task.

Import is the one place the finer type reaches the enforcer. The import API checks
create on the resource type behind the envelope's type, so a per-type importer (§4.6) is
enforced under its own name. Three built-in policies name `task` today, and each gains
the §4.10 inventory beside it: the Owner object list, `Engineer Edit Access`, and
`Host Edit Access`. Without that, `task` create no longer permits a task import.

Three hand-written lists that must grow with every new task type invite drift, so the
config types ship as one exported slice that each policy splices in. A new task type
then joins all three roles as a consequence of its schema.

### 4.10 Integration inventory

The closed set is twenty-four types. This list is the whole of it, because a set with a
fallback is not closed:

- **NI**: `ni_analog_read`, `ni_analog_write`, `ni_digital_read`, `ni_digital_write`,
  `ni_counter_read`, `ni_scanner`.
- **OPC UA**: `opc_read`, `opc_write`, `opc_scan`.
- **LabJack**: `labjack_read`, `labjack_write`, `labjack_scan`.
- **Modbus**: `modbus_read`, `modbus_write`, `modbus_scan`.
- **EtherCAT**: `ethercat_read`, `ethercat_write`, `ethercat_scan`.
- **HTTP**: `http_read`, `http_write`, `http_scan`.
- **Arc**: `arc`.
- **PagerDuty**: `pagerduty_alert`.
- **Rack**: `rack_status`.

The six scanners and `rack_status` have no meaningful configuration, but the closed set
admits no exceptions: each gets its own schema type over a shared empty base, so its
type resolves like any other. The schemas are one line each.

The rack status task needs a rename before Phase 4. The Driver configures it under the
literal type string `Rack Status`, and it still deletes a legacy `heartbeat` task on
boot. A type string names a schema type, an ontology type, and a Gorp table, so
`Rack Status` cannot stand. Phase 2 renames it to `rack_status`, which the Driver
already uses as the integration name. Both old names then leave through the legacy
deletion path the Driver carries today.

Per-type `StatusData` — the `errors[]` of NI, the read status of EtherCAT — gets a type
in each integration schema and threads through the status details generic. The Console
types these by hand today.

---

## 5 Implementation phases

Each phase is a PR, or a short series where noted. At each boundary the tree builds, the
tests pass, and the product can ship. No phase changes the task payload.

- **Phase 1: Oracle groundwork — complete.** Cross-file extension of a common shape
  already works, covered by analyzer and generator tests. The Gorp table and ontology
  registration stay hand-written per service, on the pattern of `core/pkg/service/view`;
  generating them is deferred until the pattern settles across a few integrations.
- **Phase 2: Schema authorship.** The per-integration `.oracle` files and their
  generated artifacts, not yet wired. One PR per integration or small group, each a
  reviewable schema plus inert generated code. Every task type in §4.10 is covered
  before Phase 4 lands, because the closed set has no fallback.
- **Phase 3: Tables and services.** The per-integration service packages, the Gorp
  tables, the `ontology.ResourceType` members (§4.9), the ontology registration, and the
  config registry the task writer will consult. Additive and unconsumed; Ginkgo suites
  exercise the packages directly.
- **Phase 4: Decompose and compose.** The storage cutover. The task writer decomposes on
  write; retrieve, `OnChange`, and `sy_task_set` compose on read; the config record
  becomes a second parent of the task; internal tasks gain an ontology resource; and the
  §4.8 startup migration runs. `type` and `config` become resolved. The task payload
  keeps its shape, so no client needs a new field, but the parent readers of §4.2 gain
  their type filter in the same phase.
- **Phase 5: Import and export.** Each config type registers an `imex.ImportExporter`,
  `imex` keys importers by ontology resource type, the task service delegates its export
  body, and the built-in role policies gain the config types (§4.6, §4.9). This unblocks
  SY-4524 and lands with that work, after Phase 4. Export never regresses in the
  interim: from the cutover on, the task exporter reads the composed payload, so the
  flattened body it writes today is unchanged.
- **Phase 6: Client compat deletion.** One PR per client, no wire change. The Console
  loses its NI version chain, the Python client loses its old-shape readers, and the
  Driver loses its legacy parse tolerance.

**Compatibility**: No phase breaks the wire. Phase 4 migrates persisted data with no
downgrade path, the standard position for storage migrations in this codebase. A client
that never updates keeps working through every phase; the benefit of Phase 6 is deleted
code, not new behavior.

---

## 6 Resolved decisions

1. **Typed task resources as root aggregates — deferred**: An earlier draft made each
   task type a first-class resource with a UUID, drafts, a deploy verb, and action
   dispatch. It repairs the ownership model, but it rewrites every client. The config
   record here is the storage half of that design and does not foreclose it.
2. **A typed config union on the wire — rejected**: It solves the hand-written parser
   problem, and that trade is real. It also breaks the Console, the Python client, and
   the Driver at once, which is the cost this RFC exists to avoid.
3. **`type` stored on the task row — rejected**: The parent relationship already names
   the type, and a stored column would be a second source of truth that can drift from
   the record it describes.
4. **A dedicated relationship type — rejected**: In favor of `parent_of`. The config
   record is the task's parent in every sense the ontology models, and reusing the
   existing type inherits the tree walk and the index. The cost is that a task now has
   two parents, which §4.2 handles.
5. **A `config` ontology ID field on the task — rejected**: In favor of the
   relationship, which gives the config record ontology standing for later per-type
   endpoints and policies. A field would only serve composition.
6. **Keying the config record by the task key — rejected**: A shared key makes the
   relationship redundant, but it denies the config a portable identity across export
   and import and couples the record to the SY-4488 re-key.
7. **A legacy passthrough for types without schemas — rejected**: Every task type is
   first-party and ships with the Core, so an untyped escape hatch would preserve the
   problem this RFC removes. Migration quarantines what it cannot convert (§4.8).
8. **Migrating stored records on decode — rejected**: Gorp migrations are startup
   rewrite passes, and the stored codec carries no per-record version stamp, so
   decode-time migration would need new `x/go/gorp` and codec machinery. Config versions
   bump through bootup rewrites like every other Gorp table (§4.5).
9. **Per-type schema files — rejected**: In favor of per-integration files, which keep
   the shared channel and scale unions adjacent to their users (§4.1).
10. **A task-service importer — rejected**: Import routes on the envelope's type string,
    so each config type registers its own `imex.ImportExporter` and the task service
    keeps only its `task` exporter (§4.6). A central handler would relearn every config
    shape.
11. **One envelope version for every task — rejected**: The `task` ontology type is a
    routing key, not a schema, so it carries no version. Each config type stamps its own
    (§4.6), which replaces the single `task.Version` constant used today.
12. **The config registry is an injected input**: The wiring site builds it from the
    per-integration services and passes it on the task service `Config`, which
    `Validate` requires. A package-level registry that the integration packages
    self-populate through the import graph is the alternative, and it is a mutable
    global that hides the dependency and orders itself by import.

---

## 7 What this RFC does not cover

- **Typed configs on the client wire**: The Console, the Python client, and the C++
  Driver keep their hand-written config parsers. This RFC removes the version chains,
  not the parsers.
- **Device properties, in any form**: `device.properties` stays an opaque blob, with no
  typing and no migration. The EtherCAT `manual_<index>_<subindex>` keys and the OPC
  NodeIds that camelCase conversion corrupts live there, so the `getChannelByMapKey`
  fallback in the Console stays. Repairing that class is its own effort, and the pattern
  here transfers to it directly.
- **Drafts, deploy, and the resource lifecycle**: §6, decision 1.
- **Typed command args**: `Command.args` stays opaque.

---

## 8 Open questions

1. Whether composition belongs in the retrieve builder or behind an explicit
   `.WithConfigs()` option, so callers who only need a task name skip the join. Making
   `type` resolved raises the stakes: a caller that filters by type always needs it.
2. The surface for a quarantined row (§4.8): whether it carries an error status, and
   whether the Console lists it.
3. The order of integrations across Phases 2 and 6.
