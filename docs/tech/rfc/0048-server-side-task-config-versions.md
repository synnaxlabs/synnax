# 48 Server-side task configuration versions and import

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
  [RFC 0047 - Oracle predecessor-chain type versioning](0047-oracle-predecessor-chain-versioning.md).

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
stored configs migrate server-side and clients only ever see the latest shape. The task
service implements `imex.Importer`, which unblocks server-side task import (SY-4524) and
lets export encode a typed struct instead of flattening a blob.

Task types become a closed set. Every type has a schema, and the Core rejects a type it
does not know rather than storing an untyped blob for it.

The task keeps its current key, so this RFC is independent of the SY-4488 UUID re-key
and composes with it in either order.

---

## 1 Motivation

1. **The Core cannot migrate persisted data**: config shapes are persisted data, but
   their version chains live in clients. The Console holds a private NI chain
   (`console/src/feature/ni/task/types/v0.ts` and `v1.ts`), the Python client misreads
   old shapes, and the Driver fails to parse them.
2. **Server-side import is blocked**: RFC 0039 moved metadata import into the Core, but
   the task service implements only `imex.Exporter`, and that exporter flattens an
   opaque map into the envelope body. SY-4524 waits on a config the Core can read.
3. **A config shape has no single definition**: each one is written by hand three times,
   in Console Zod, Python Pydantic, and C++ `parser.field<T>()`. No tool reports when
   the copies disagree. NI multiplies this across roughly 50 channel variants.
4. **The Core cannot reject bad data**: a malformed config is accepted, stored, and
   fails later at the Driver, far from the client that wrote it.

---

## 2 Vocabulary

- **Config record**: the typed row that holds one task's configuration. It lives in a
  Gorp table specific to the task type and has its own UUID key.
- **Resolved field**: a field the API returns but no row stores. The service computes it
  on retrieve. `type` and `config` both become resolved fields on the task.
- **Decompose**: to split an incoming task payload into a task row and a config record.
- **Compose**: to rebuild the resolved fields from a config record on retrieve.

---

## 3 Principles

1. **The Core owns persisted data and its migrations**: a config shape is persisted
   data, so its schema and version chain belong to the Core, not to each client.
2. **Write the shape one time**: one Oracle schema per task type generates the Go,
   TypeScript, Python, C++, and Protobuf code. Hand parsing is a defect.
3. **The client contract does not move**: this RFC changes storage and adds server-side
   behavior. The task payload on the wire keeps its shape, field for field.
4. **A closed set, loudly enforced**: every task type has a schema. Task types are
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
(amended), and `pagerduty.oracle`. A file defines the shared parts of an integration —
channel unions, scales, endpoints — and one config type per task type. We rejected
per-type files: the 19-variant AI channel union of NI and its scale and CJC unions are
shared across its five task types.

The NI schema draft (32 enums, 4 unions, 57 structs) is the start point. Shared
cross-integration bases, the `sample_rate` / `stream_rate` / `data_saving` read shape
and the write shape, live in a common task schema that the per-integration files extend.
Oracle must first support extending a common shape across schema files.

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
first parent it gets, so it filters to `range`. Phase 4 audits the parent readers in
every client.

Internal tasks gain an ontology resource. The task writer creates none for them today,
so a scanner has nothing to relate and its `type` cannot resolve. Phase 4 removes that
early return: every task gets a resource and a config edge, internal or not.

The config record also gains a UUID that survives export and import, and an ontology
presence that per-type endpoints and access policies can name later.

### 4.3 Resolving `type` and `config`

The parent is an `ontology.ID` of the form `ni_analog_read:<uuid>`, so it already
carries the task type. Resolving `type` reads the parent's type part; resolving `config`
reads the record and encodes it.

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

**Compose (read)**: the reader resolves the config parent, decodes and migrates the
record, and stamps `type` and `config` back onto the payload. §4.3 names the call sites.

A decode failure on retrieve means a corrupt record, not a stale one: the record was
validated before storage, and the migration chain covers older stored versions. The
reader reports it as an internal error rather than serving a partial task.

### 4.5 Versions and migrations

Each config type carries its own `@go version` and its own `versions/vN` chain, the
standard Oracle mechanism (RFC 0033, RFC 0047). Nothing about the chain is
task-specific: a config type is an ordinary versioned Oracle type that happens to be
keyed by a task type string.

Stored records migrate on decode, so a record written under `v0` is served as the
current version without a rewrite pass. The Core therefore always serves the latest
shape, and that is the property the clients depend on.

Clients delete their own chains. The Console loses its NI `v0`/`v1` types, the Python
client loses its old-shape readers, and the Driver loses the tolerance it carries for
configs it cannot parse. None of this is a wire change: the field is the same field,
carrying a shape the client already understands.

### 4.6 Import and export

The task service today implements `imex.Exporter` and nothing else, and its `Export`
flattens `map[string]any` into the envelope body with `type` and `name` stamped on top.
The flattening is deliberate — a task file is readable as a config — and this RFC keeps
that body shape. What changes is the source.

The task service dispatches on type; it never encodes a config itself. `Export` resolves
the task's type, routes to that type's service — `http_read`, `ni_analog_read`, and so
on — asks it for the encoded config record, then stamps `type` and `name` on top.
`Import` reverses this. It reads `type` from the envelope body and routes to the same
service, which decodes, migrates, and writes the config record, while the task service
writes the task row and the relationship in the supplied transaction. The task service
owns the envelope shape and the per-integration services own the config shape, the same
split as §4.1.

`Importer.Type()` returns the coarse `task` ontology type, which is the asymmetric
registration the imex interface already documents: the service registers per task type
and accounts under one resource type. An envelope naming an unknown type fails the
import with a clear error.

Envelope versions follow the unified numbering: the version stamped on an exported task
is the config type's `@go version`, so an envelope is self-describing under the same
number the schema carries.

### 4.7 Lifecycle

Every operation stays where it is; only the transaction contents grow.

- **Create**: decompose, write the task row and the config record, define the
  relationship.
- **Update**: rewrite the config record in place. The task key and the relationship do
  not change.
- **Delete**: delete the task row, the config record, and the relationship together. The
  writer deletes every parent of the task that is not a group, so a config record never
  outlives the task it configures.
- **Copy**: copy the config record under a new UUID and relate it to the new task. The
  existing `Writer.Copy` gains one step.
- **Snapshot**: a snapshot task gets its own frozen config record. Snapshots never share
  a record with a live task.
- **Rename**: task row only. The name is not part of the config.

### 4.8 Migration of stored tasks

A one-time startup migration walks every stored task. It decodes the blob through the
legacy chain, writes the config record, defines the relationship, and clears `type` and
`config` from the task row.

A row the migration cannot convert — an unknown type string, or a config that fails to
decode — is quarantined: the task row and its raw blob are preserved untouched, the task
is not served or configured, and the log names it once. The migration never drops a
config and never writes a partial record. Quarantine is a loud failure that an operator
resolves, not a mode the system runs in.

### 4.9 Access control

`ontology.ResourceType` is a closed generated enum, so every config type adds a member
to it. Phase 3 extends the enum with the §4.10 inventory and regenerates the four
clients.

Each new member is a nameable RBAC object, and a policy that grants `task` today grants
nothing on `ni_analog_read`. The task service therefore keeps checking access against
the `task` type alone: a config record inherits the decision made for its task. The
built-in role policies enumerate their objects by type, and they gain no entries here.
Naming a config type as a policy object waits for the per-type endpoints of Resolved
Decision 1.

### 4.10 Integration inventory

`ni_analog_read`, `ni_analog_write`, `ni_digital_read`, `ni_digital_write`,
`ni_counter_read`, `opc_read`, `opc_write`, `labjack_read`, `labjack_write`,
`modbus_read`, `modbus_write`, `ethercat_read`, `ethercat_write`, `http_read`,
`http_write`, `arc`, and `pagerduty_alert`.

The scanner types (`opc_scan`, `modbus_scan`, `ni_scanner`, …) and the rack status task
have no meaningful configuration, but the closed set admits no exceptions: each gets its
own schema type over a shared empty base, so its type resolves like any other. The
schemas are one line each.

Per-type `StatusData` — the `errors[]` of NI, the read status of EtherCAT — gets a type
in each integration schema and threads through the status details generic. The Console
types these by hand today.

---

## 5 Implementation phases

Each numbered phase is a PR, or a short series where noted. At each boundary the tree
builds, the tests pass, and the product can ship. No phase changes the task payload.

1. **Oracle groundwork**: cross-file extension of a common shape for the shared config
   bases, and Gorp table plus ontology registration output for a config type. Pure
   generator work with generator tests. No schema consumes it yet.
2. **Schema authorship**: the per-integration `.oracle` files and their generated
   artifacts, not yet wired. One PR per integration or small group, each a reviewable
   schema plus inert generated code. Every task type in §4.10 is covered before Phase 4
   lands, because the closed set has no fallback.
3. **Tables and services**: the per-integration service packages, the Gorp tables, the
   `ontology.ResourceType` members (§4.9), the ontology registration, and the config
   registry the task writer will consult. Additive and unconsumed; Ginkgo suites
   exercise the packages directly.
4. **Decompose and compose**: the storage cutover. The task writer decomposes on write;
   retrieve, `OnChange`, and `sy_task_set` compose on read; the config record becomes a
   second parent of the task; internal tasks gain an ontology resource; and the §4.8
   startup migration runs. `type` and `config` become resolved. The task payload keeps
   its shape, so no client needs a new field, but the parent readers of §4.2 gain their
   type filter in the same phase.
5. **Import and export**: the task service implements `imex.Importer`, and both verbs
   dispatch to the per-type service that owns the config (§4.6). This unblocks SY-4524.
6. **Client compat deletion**: one PR per client, no wire change. The Console loses its
   NI version chain, the Python client loses its old-shape readers, and the Driver loses
   its legacy parse tolerance.

**Compatibility**: no phase breaks the wire. Phase 4 migrates persisted data with no
downgrade path, the standard position for storage migrations in this codebase. A client
that never updates keeps working through every phase; the benefit of Phase 6 is deleted
code, not new behavior.

---

## 6 Resolved decisions

1. **Typed task resources as root aggregates — deferred**: an earlier draft made each
   task type a first-class resource with a UUID, drafts, a deploy verb, and action
   dispatch. It repairs the ownership model, but it rewrites every client. The config
   record here is the storage half of that design and does not foreclose it.
2. **A typed config union on the wire — rejected**: it solves the hand-written parser
   problem, and that trade is real. It also breaks the Console, the Python client, and
   the Driver at once, which is the cost this RFC exists to avoid.
3. **`type` stored on the task row — rejected**: the parent relationship already names
   the type, and a stored column would be a second source of truth that can drift from
   the record it describes.
4. **A dedicated relationship type — rejected**: in favor of `parent_of`. The config
   record is the task's parent in every sense the ontology models, and reusing the
   existing type inherits the tree walk and the index. The cost is that a task now has
   two parents, which §4.2 handles.
5. **A `config` ontology ID field on the task — rejected**: in favor of the
   relationship, which gives the config record ontology standing for later per-type
   endpoints and policies. A field would only serve composition.
6. **Keying the config record by the task key — rejected**: a shared key makes the
   relationship redundant, but it denies the config a portable identity across export
   and import and couples the record to the SY-4488 re-key.
7. **A legacy passthrough for types without schemas — rejected**: every task type is
   first-party and ships with the Core, so an untyped escape hatch would preserve the
   problem this RFC removes. Migration quarantines what it cannot convert (§4.8).
8. **Rewriting stored records at migration time — rejected**: records migrate on decode,
   so the startup migration only decomposes. No rewrite pass, no version sweep.
9. **Per-type schema files — rejected**: in favor of per-integration files, which keep
   the shared channel and scale unions adjacent to their users (§4.1).
10. **The task service encoding configs for export — rejected**: each per-integration
    service encodes and decodes its own configs, and the task service only dispatches on
    type (§4.6). A central encoder would relearn every config shape.

---

## 7 What this RFC does not cover

- **Typed configs on the client wire**: the Console, the Python client, and the C++
  Driver keep their hand-written config parsers. This RFC removes the version chains,
  not the parsers.
- **Device properties, in any form**: `device.properties` stays an opaque blob, with no
  typing and no migration. The EtherCAT `manual_<index>_<subindex>` keys and the OPC
  NodeIds that camelCase conversion corrupts live there, so the `getChannelByMapKey`
  fallback in the Console stays. Repairing that class is its own effort, and the pattern
  here transfers to it directly.
- **Drafts, deploy, and the resource lifecycle**: Resolved Decision 1.
- **Typed command args**: `Command.args` stays opaque.

---

## 8 Open questions

1. Whether composition belongs in the retrieve builder or behind an explicit
   `.WithConfigs()` option, so callers who only need a task name skip the join. Making
   `type` resolved raises the stakes: a caller that filters by type always needs it.
2. Whether the config registry is injected into the task service at construction or
   assembled by the per-integration packages at wiring time.
3. Which version an exported task envelope carries (§4.6). The body merges task row
   fields with config fields, the envelope holds one number, and the task type has a
   `@go version` of its own.
4. The surface for a quarantined row (§4.8): whether it carries an error status, and
   whether the Console lists it.
5. The order of integrations across Phases 2 and 6.
