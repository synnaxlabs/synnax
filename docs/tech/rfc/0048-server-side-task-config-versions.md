# 48 Server-side task configuration versions and import

- **Author**: Patrick Dotson
- **Date**: 2026-08-05
- **Related**: [RFC 0005 - MVP](0005-ontology.md),
  [RFC 0017 - General purpose device drivers](0017-drivers.md),
  [RFC 0027 - Oracle schema system](0027-oracle-schema-system.md),
  [RFC 0033 - Oracle migration system](0033-oracle-migrations.md),
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
The stored configuration moves out of the task row into that table, and an ontology
relationship links a task to its config record. The task service decomposes an incoming
config into that record on write and composes it back into `Task.Config` on retrieve, so
the task payload keeps its current shape on every wire.

Two capabilities follow. The Core owns the version chain for every config type, so
stored configs migrate server-side and clients only ever see the latest shape. The task
service implements `imex.Importer`, which unblocks server-side task import (SY-4524) and
lets export encode a typed struct instead of flattening a blob.

The task keeps its `config` field, its `type` field, and its current key. This RFC is
independent of the SY-4488 UUID re-key and composes with it in either order.

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
- **Config relationship**: the ontology relationship from a task to its config record.
  It is the only stored link between the two rows.
- **Decompose**: to split an incoming task payload into a task row and a config record.
- **Compose**: to rebuild `Task.Config` from a config record on retrieve.
- **Legacy passthrough**: a stored config whose type has no schema. It stays an opaque
  blob on the task row and keeps working.

---

## 3 Principles

1. **The Core owns persisted data and its migrations**: a config shape is persisted
   data, so its schema and version chain belong to the Core, not to each client.
2. **Write the shape one time**: one Oracle schema per task type generates the Go,
   TypeScript, Python, C++, and Protobuf code. Hand parsing is a defect.
3. **The client contract does not move**: this RFC changes storage and adds server-side
   behavior. The task payload on the wire keeps its shape, field for field.
4. **An unknown type passes through**: the task keeps its config field, so a type
   without a schema stores its blob and keeps working. Typing is incremental, and an
   integration that has not been schematized yet costs nothing.

---

## 4 Design

### 4.0 The storage split

```
┌──────────────────────────┐                   ┌──────────────────────────┐
│ task                     │    ontology       │ ni_analog_read           │
│  key      uint64         │    relationship   │  key         uuid        │
│  name     string         │ ────────────────► │  sample_rate telem.Rate  │
│  type     string         │   "configured by" │  channels    AIChannel[] │
│  internal bool           │                   │  ...                     │
│  snapshot bool           │                   │  @go version 1           │
└──────────────────────────┘                   └──────────────────────────┘
              │                                            │
              └──────── compose on retrieve ───────────────┘
                        decompose on write
                                 │
                                 ▼
                   Task.Config — unchanged on the wire
```

The task row no longer stores the config blob for a schematized type. The config record
holds it, and the ontology relationship carries the link. The task keeps `type`: it is
the dispatch key that selects the schema, the table, and the migration chain.

### 4.1 Schemas and services

**One Oracle schema file per integration**: `schemas/synnax/ni.oracle`, `opc.oracle`,
`labjack.oracle`, `modbus.oracle`, `ethercat.oracle`, `http.oracle`, plus `arc.oracle`
(amended), `pagerduty.oracle`, and `slack.oracle`. A file defines the shared parts of an
integration — channel unions, scales, endpoints — and one config type per task type. We
rejected per-type files: the 19-variant AI channel union of NI and its scale and CJC
unions are shared across its five task types.

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

A new ontology relationship type links the two rows:

```go
const RelationshipTypeConfiguredBy RelationshipType = "configured_by"
```

`From` is the task and `To` is the config record, matching the traversal that every read
performs. `RelationshipType` is a plain string in `ontology/versions/v0`, so the new
value needs no ontology migration.

Cardinality is exactly one config record per task, enforced in the task writer. The
relationship is defined and deleted in the same transaction as the task row, so no
retrieve observes a task without its config.

The relationship gives the config record an identity the task row does not: a UUID that
survives export and import, and an ontology presence that per-type endpoints and access
policies can name later. Storing the link as a field on the task instead would work for
composition but would give the config record no independent standing.

### 4.3 Composition and decomposition

The task service is the only place that splits and rejoins a payload.

**Decompose (write).** `Create` and `Update` take a task payload with an embedded
config, exactly as they do today. The writer looks up the type in its config registry.
On a hit it decodes the blob into the generated Go struct, validates it, writes the
config record, and defines the relationship. On a miss it stores the blob on the task
row as a legacy passthrough. Both paths run in the caller's transaction.

**Compose (retrieve).** The retrieve builder resolves the relationships for the whole
result set in one ontology query, reads the config records, encodes each one back into
`Task.Config`, and returns the task payload unchanged. Resolution is batched per
retrieve, not per task: the Driver's `sy_task_set` path and the Console's task list both
read many tasks at a time, and a per-task ontology lookup would put a query per task on
a hot path.

A decode failure on a known type is a validation error on the write. A decode failure on
retrieve is not possible: the record was validated before it was stored, and the
migration chain covers older stored versions.

### 4.4 Versions and migrations

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

### 4.5 Import and export

The task service today implements `imex.Exporter` and nothing else, and its `Export`
flattens `map[string]any` into the envelope body with `type` and `name` stamped on top.
The flattening is deliberate — a task file is readable as a config — and this RFC keeps
that body shape. What changes is the source: `Export` encodes the typed struct.

`Import` is new. It reads `type` from the envelope body, routes to the config registry,
decodes and migrates the body into the generated struct, then writes the task row, the
config record, and the relationship in the supplied transaction. `Importer.Type()`
returns the coarse `task` ontology type, which is the asymmetric registration the imex
interface already documents: the service registers per task type and accounts under one
resource type.

Envelope versions follow the unified numbering: the version stamped on an exported task
is the config type's `@go version`, so an envelope is self-describing under the same
number the schema carries. A legacy passthrough exports its blob with the task's own
schema version, as it does now.

### 4.6 Lifecycle

Every operation stays where it is; only the transaction contents grow.

- **Create**: decompose, write the task row and the config record, define the
  relationship.
- **Update**: rewrite the config record in place. The task key and the relationship do
  not change.
- **Delete**: delete the task row, the config record, and the relationship together.
- **Copy**: copy the config record under a new UUID and relate it to the new task. The
  existing `Writer.Copy` gains one step.
- **Snapshot**: a snapshot task gets its own frozen config record. Snapshots never share
  a record with a live task.
- **Rename**: task row only. The name is not part of the config.

### 4.7 Migration of stored tasks

A one-time startup migration walks every stored task:

1. A task whose type has a schema is decomposed. The migration decodes the blob through
   the legacy chain, writes the config record, defines the relationship, and clears the
   blob from the task row.
2. A task whose type has no schema is left exactly as it is. It keeps its blob and reads
   back byte-identical, and the log names it once so the gap is visible.
3. A decode failure on a known type leaves the row untouched and logs an error. The
   migration never drops a config and never writes a partial record.

Because the passthrough path is real, the migration is not a cutover. Integrations
convert one at a time, and an unconverted one is indistinguishable from the outside.

### 4.8 Integration inventory

`ni_analog_read`, `ni_analog_write`, `ni_digital_read`, `ni_digital_write`,
`ni_counter_read`, `opc_read`, `opc_write`, `labjack_read`, `labjack_write`,
`modbus_read`, `modbus_write`, `ethercat_read`, `ethercat_write`, `http_read`,
`http_write`, `arc`, `pagerduty_alert`, and `slack_alert` (SY-3995).

The scanner types (`opc_scan`, `modbus_scan`, `ni_scanner`, …) and the rack status task
have no meaningful configuration. Each stays a legacy passthrough, which costs nothing
under §4.7 and needs no schema.

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
   schema plus inert generated code.
3. **Tables and services**: the per-integration service packages, the Gorp tables, the
   ontology registration, and the config registry the task writer will consult. Additive
   and unconsumed; Ginkgo suites exercise the packages directly.
4. **Decompose and compose**: the storage cutover. The task writer decomposes on write,
   the retrieve builder composes on read, the relationship type lands, and the §4.7
   startup migration runs. The task payload does not change, so every client keeps
   working untouched.
5. **Import and export**: the task service implements `imex.Importer`, and `Export`
   encodes the typed struct. This unblocks SY-4524.
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
3. **A `config` ontology ID field on the task — rejected** in favor of the relationship.
   The relationship gives the config record ontology standing for later per-type
   endpoints and policies; a field would only serve composition.
4. **Keying the config record by the task key — rejected**: a shared key makes the
   relationship redundant, but it denies the config a portable identity across export
   and import and couples the record to the SY-4488 re-key.
5. **Quarantining unknown types — replaced by passthrough**: the config field still
   exists, so a type without a schema keeps working at no cost. This buys incremental
   adoption, which a closed set would forbid.
6. **Rewriting stored records at migration time — rejected**: records migrate on decode,
   so the startup migration only decomposes. No rewrite pass, no version sweep.
7. **Per-type schema files — rejected** in favor of per-integration files, which keep
   the shared channel and scale unions adjacent to their users (§4.1).

---

## 7 What this RFC does not cover

- **Restructuring semantic map keys**: the EtherCAT `manual_<index>_<subindex>` and OPC
  NodeId maps still corrupt under camelCase conversion, and the `??` fallback lookups in
  the Console stay. Once the Core owns the shape, a Core-owned migration can restructure
  them into arrays of structs without client coordination. That is the natural follow-up
  and is much cheaper after this RFC than before it.
- **Typed configs on the client wire**: the Console, the Python client, and the C++
  Driver keep their hand-written config parsers. This RFC removes the version chains,
  not the parsers.
- **Drafts, deploy, and the resource lifecycle**: Resolved Decision 1.
- **Typed command args**: `Command.args` stays opaque.
- **Typed device `properties`**: the same corruption class as the map keys, in an
  adjacent effort.

---

## 8 Open questions

1. The relationship name and direction (`configured_by` from the task, or `config_of`
   from the record), and whether the ontology tree should hide config records.
2. Whether composition belongs in the retrieve builder or behind an explicit
   `.WithConfigs()` option, so that callers who only need task metadata skip the join.
3. Whether the config registry is injected into the task service at construction or
   assembled by the per-integration packages at wiring time.
4. The order of integrations across Phases 2 and 6.
