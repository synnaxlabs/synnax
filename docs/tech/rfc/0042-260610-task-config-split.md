# 42 - Task and Task Configuration Split

**Feature Name**: Task and Task Configuration Split <br /> **Status**: Draft <br />
**Start Date**: 2026-06-10 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

The task record splits into two data structures. The task becomes a general-purpose
record (key, name, type, snapshot state) with no embedded configuration. Each task type
gets a strongly typed configuration stored by a dedicated core service (starting with
`core/pkg/service/ni`), generated from an Oracle schema and migrated server-side through
the standard `types/vN/` chain (RFC 0033, RFC 0041 §4.3). A task references its
configuration through a `parent` ontology relationship.

The wire format does not change: the API layer composes the typed configuration back
into the task payload as a resolved field (RFC 0041 §4.2), so the Console, the Python
client, and the driver are unaffected. The split is a storage, validation, and migration
change. Configuration migrations move from N independent client implementations to one
server-side migration chain that runs once per cluster at startup.

# 1 - Vocabulary

- **Task** - The general-purpose record in `core/pkg/service/task`: key (rack-encoded),
  name, type, internal and snapshot flags. After the split, no stored configuration.
- **Task Config** - A strongly typed record describing how a specific task type behaves
  (e.g. an NI analog read configuration). Owned by a config service, with its own key.
- **Config Service** - A core service owning the gorp tables, Oracle-generated types,
  and migration chain for one integration's config types (e.g. `service/ni`).
- **Config Provider** - The interface a config service implements to plug into the task
  service: validate, store, load, copy, and delete configs for a set of task types.
- **Fallback Path** - The embedded-JSON behavior that task types without a registered
  provider keep. Identical to today's behavior.
- **Resolved Field** - A field present on a type and on the wire but excluded from the
  storage codec, filled by the API layer (RFC 0041 §4.2). `Task.Config` becomes one.

# 2 - Motivation

Task configuration is an opaque JSON blob (`msgpack.EncodedJSON`) on the task record.
The core never parses it, so three problems compound as task schemas evolve:

1. **Every client migrates independently, or not at all.** When a config shape changes
   (e.g. NI analog read moved `device` from the task level onto each channel), the
   Console grows a hand-written zod union-transform that must live forever, the Python
   client gets nothing and silently misreads old configs, and the driver fails to parse
   them. The same transform is reimplemented per language or skipped.
2. **Validation happens at the wrong time.** The server accepts any blob; a malformed
   config surfaces minutes later as a driver task-configuration error rather than at
   write time.
3. **The server-side migration system cannot reach configs.** RFC 0033 built startup
   migrations for Oracle-managed types, and RFC 0041 §4.3 standardized the versioned
   type layout, but a config embedded in an envelope as untyped JSON has no schema, no
   version chain, and no migration path. RFC 0033 §7.2 explicitly anticipates moving
   client-side schema evolution to the server; tasks are the largest remaining case.

The status field already moved off the task record into the status service, and RFC 0041
§4.2 collapses `Labels`/`Parent`/`Status` into resolved fields. Configuration is the
natural next field to receive the same treatment, with the additional requirement that
it is stored, strongly typed, and migrated rather than derived.

# 3 - Principles

## 3.0 - The Wire Format Is Not the Storage Format (For Now)

Clients keep sending and receiving tasks with an embedded `config`. Where the server
stores that config, and in what representation, is invisible to them. The split must
ship without a coordinated client, Console, or driver upgrade.

Wire compatibility is a transitional constraint, not the end state. Eventually the wire
itself becomes strongly typed: clients send and receive the generated config types, and
the untyped embedded blob disappears from the protocol. Preserving the current wire
shape is what lets the migration happen gradually, one integration and one release at a
time, instead of forcing a coordinated cutover across the server, three client
libraries, the Console, and the driver.

## 3.1 - The Ontology Is a Reference Graph, Not a Data Store

The task-to-config link is an ontology relationship because relationships are the
platform's mechanism for reference queries and traversal. Config data is always fetched
through the owning service's typed retrieve, never through `ontology.Resource` data
resolution.

## 3.2 - Typed Configs Are Opt-In Per Task Type

A task type without a registered config provider behaves exactly as today. Integrations
adopt typed configs one service at a time, and third-party or experimental task types
never require core changes to exist.

## 3.3 - Copy, Then Cut

The bootstrap copies blobs into typed tables and keeps the blobs synchronized for one
release before deleting them. Every release in the rollout is independently safe to
downgrade from until the final one.

## 3.4 - The Server Owns Schema Evolution

After the split, exactly one migration implementation exists per schema change: a Go
transform in the config service's `types/vN/` chain, run at startup and reused to
normalize old-shaped wire input. Clients only ever see the current shape.

# 4 - Design

## 4.0 - The Task Record

`Task` keeps `Key`, `Name`, `Type`, `Internal`, and `Snapshot` as stored fields.
`Config` becomes a resolved field: present on the type and serialized on the wire, but
marked with the storage-exclusion marker (RFC 0041 §4.6.1) so the storage codec skips
it. For task types on the fallback path, the config continues to be stored on the task
under a legacy field until the final cutover stage removes it for migrated types.

## 4.1 - Config Services

Each integration gets a config service following the standard service anatomy (RFC 0041
§4.3.0): one gorp table per config type (e.g. `AnalogReadConfig`, `DigitalWriteConfig`),
a writer, a retrieve builder, ontology registration, and a `types/vN/` version chain.
Types are generated from the integration's Oracle schema (`schemas/current/ni.oracle`)
by adding a core Go output and migration generation to the schema that already generates
the TS, Python, and C++ client types. One schema is the source of truth for the server,
all clients, and (eventually) the driver.

Config records have their own keys, independent of the task key. Independent identity
keeps config lifecycle decoupled from task lifecycle: snapshots, history, and templated
configs become representable later without schema surgery.

## 4.2 - The Provider Registry

The task service exposes a registry mapping task-type prefixes to config providers:

```go
type ConfigProvider interface {
    // TypePrefix returns the task-type prefix this provider handles (e.g. "ni_").
    TypePrefix() string
    // Create validates cfg, stores a typed record in the provider's tables, and
    // returns the ontology ID of the created config resource.
    Create(ctx context.Context, tx gorp.Tx, task Key, taskType string,
        cfg msgpack.EncodedJSON) (ontology.ID, error)
    // Load returns the wire representation of the config for the given task.
    Load(ctx context.Context, tx gorp.Tx, task Key) (msgpack.EncodedJSON, error)
    // Copy duplicates the config of one task for another (task copy, snapshots).
    Copy(ctx context.Context, tx gorp.Tx, from, to Key) (ontology.ID, error)
    // Delete removes the config record and its ontology resource.
    Delete(ctx context.Context, tx gorp.Tx, task Key) error
}
```

The task writer dispatches on `task.Type`. With a matching provider, create routes the
config to the provider and defines the relationship; without one, the embedded-blob
fallback path runs. Snapshot semantics (config preserved when updating a snapshot task)
and task copy route through `Copy`. All provider methods receive the writer's `gorp.Tx`,
so a task and its config are created, copied, or deleted in one transaction, following
the device writer's pattern.

## 4.3 - Linkage via Parent Relationship

The task is the ontology parent of its config: `task:<key> -> parent -> <config-id>`. No
new relationship type is introduced. Consumers resolve a task's config by traversing
children and filtering by resource type, the same pattern ranger uses; the config
resource type also identifies which service and table own the record. `DeleteResource`
cascades the relationship on either end.

Relationship management here is hand-written in the writers, consistent with RFC 0041
§4.2.3. When the follow-up relationship-management RFC (RFC 0041 §7) lands schema-level
relationship declarations, the task-to-config link is a candidate for generation.

## 4.4 - Wire Composition

On retrieve, the API layer fills `Task.Config` from the provider (or from the legacy
embedded field on the fallback path) inside the same transaction. Unlike
`Labels`/`Parent`/`Status`, the config is always included; no `Include` flag gates it,
because every existing consumer requires it. On create, the API layer strips the config
from the payload and routes it through the writer's provider dispatch.

The driver is unaffected in every phase: `sy_task_set` carries only task keys, the
driver re-fetches the full task, and the fetched payload is byte-compatible.

## 4.5 - Migration

**Storage.** Config tables migrate at `OpenTable` exactly as every other versioned
resource (RFC 0033 §4.2.3, RFC 0041 §4.4.1): edit the schema, bump `@version`, fill the
generated `migrate.go` skeleton, golden-test the transform. A config shape change is one
Go transform instead of one transform per client language.

**Wire input.** Startup migration normalizes stored data but not API input: an outdated
Console can still send an old-shaped config to `task.Create`. Provider create handles
old shapes with the same `types/vN/` chain used by storage migration: attempt a decode
at the current version, and on failure decode at older versions and walk the per-version
`Migrate` functions forward (the per-payload runner from RFC 0041 §4.4.1). The NI
chain's `v0 -> v1` transform (task-level `device` spread onto each channel) is the
hand-written Go port of the Console's existing transform, written once and used by both
the bootstrap and the write path.

## 4.6 - Cutover

The rollout spans three releases, each independently safe:

- **Stage 0 (inert).** The provider registry, relationship wiring, and API composition
  ship with no providers registered. No behavior change.
- **Stage 1 (copy + dual-write).** The NI service ships with a bootstrap migration in
  its startup chain. For each task with an `ni_` type prefix: decode the embedded blob
  through the version chain, write the typed record, define the parent relationship, and
  leave the blob in place. Config writes dual-write the typed record (source of truth)
  and the blob, so downgrading the server binary remains safe. A blob that decodes at no
  version leaves its task on the fallback path with a warning status; a malformed config
  must never block startup. The pass runs in one transaction and is recorded as applied.
  The task service opens before the NI service, so task records are already at the
  current envelope version when the bootstrap reads them.
- **Stage 2 (cut).** One release later: stop dual-writing, ship a migration clearing the
  leftover blobs from migrated task types, and delete the Console's client-side config
  transforms.

## 4.7 - Workflows

### 4.7.0 - Task Creation

A create request is unchanged on the wire: `{name, type, config}`. The API layer strips
the config; the task writer allocates the rack-encoded key, writes the envelope, sets
status, and dispatches on the type prefix. A matching provider normalizes the config
through the `vN` chain, validates it against the generated schema, and writes the typed
record; the writer defines the parent relationship. Without a provider, the blob is
stored on the task as today. The whole flow is one transaction, so a malformed config
rejects the create with a structured error the Console can surface in the form, instead
of being accepted and failing minutes later as a driver task status. Downstream is
untouched: `sy_task_set` carries keys, the driver re-fetches, and retrieve composes a
byte-compatible payload.

### 4.7.1 - Import and Export

The Console's current file format (`{...config, type}`) keeps working: export reads the
composed payload, and an imported file is ordinary wire input, normalized on write like
any other create. Old exported files therefore continue to import correctly even after
Stage 2 deletes the Console's client-side transforms.

Long term, server-side import/export (RFC 0039) registers each task subtype as its own
importer and one task exporter covering all subtypes. The split supplies the two pieces
tasks are missing for that design: a schema to validate `env.Data` against and a
meaningful per-resource `version` to dispatch on. The flat envelope
`{version, type, name, ...fields}` maps directly onto the config record plus the task
name, and import walks the same `vN` chain as storage migration and normalize-on-write.
RFC 0039's goal of importing a task config while the target driver is offline becomes
real, because the server validates instead of the driver. One interaction is flagged for
that RFC: imex import takes no parent ontology ID, but task keys encode their rack, so a
task importer must be handed one; resolved alongside RFC 0039's container-association
question.

# 5 - Resolved Design Decisions

| #   | Decision                                     | Rationale                                                            |
| --- | -------------------------------------------- | -------------------------------------------------------------------- |
| 1   | Config composed into the wire payload        | No coordinated client/driver upgrade; split is invisible at the API. |
| 2   | Standard `parent` relationship, no new type  | Traversal, indexes, and cascade already built around it.             |
| 3   | Config records have independent keys         | Decouples config lifecycle; enables history/templates later.         |
| 4   | One gorp table per config type               | Scoped migrations; `task.Type` already selects the table.            |
| 5   | Fallback path for unregistered task types    | Preserves driver pluggability; typed configs adopt incrementally.    |
| 6   | Copy-then-cut with one dual-write release    | No rollback exists in the migration system; downgrade stays safe.    |
| 7   | Wire normalization reuses the `vN` chain     | One transform per schema change, shared by storage and write paths.  |
| 8   | Config is always composed, no `Include` flag | Every existing consumer requires it; gating adds only failure modes. |

# 6 - Implementation Phases

Sequenced so inert, dependency-unblocking work lands first. Phases 1 and 2 build off
`rc` immediately; phases 3 onward require the struct-union and NI-schema work (PRs
#2416, #2433) to merge first, since NI config types are discriminated unions and the
schema must not be forked.

- **Phase 1 - Provider registry and composition (§4.2, §4.4).** The `ConfigProvider`
  interface, writer dispatch with fallback, relationship wiring, and API-layer
  composition. Tested with a fake provider; golden tests assert payloads are
  byte-identical with and without a provider. Inert in production.
- **Phase 2 - Wire-input decode helper (§4.5).** The per-payload decode-and-walk runner
  over a `types/vN/` chain, shared with the imex per-payload path where possible.
- **Phase 3 - Union support in migration codegen.** The migrate plugin's auto-copy
  generator handles struct, alias, and distinct forms but not unions
  (`oracle/plugin/go/migrate/auto_copy.go`). NI configs are built from unions, so
  auto-migrate generation for them is blocked until a `UnionForm` case exists.
- **Phase 4 - `service/ni` (§4.1).** Core Go output and migration generation on the NI
  schema, the service, and provider registration. No bootstrap: new tasks get typed
  configs, existing tasks keep working through the fallback. A mixed-state cluster is a
  valid state by construction.
- **Phase 5 - Bootstrap and dual-write (§4.6 Stage 1).** The copy migration, the
  hand-written `v0 -> v1` NI transform with golden tests against captured fixtures, and
  normalize-on-write wiring.
- **Phase 6 - Cut (§4.6 Stage 2).** One release after Phase 5: stop dual-writing, clear
  blobs, delete Console-side transforms.

# 7 - Open Questions

- **Config versions on the wire.** Normalize-on-write currently shape-detects by
  attempting decodes newest-to-oldest. Stamping an explicit config version into task
  payloads (mirroring imex's `{version, type}` peek) would make detection exact, but
  requires clients to echo the version back. Deferred until shape detection proves
  insufficient.
- **Access control granularity.** Config resources enter the ontology, so RBAC rules
  could target them directly. Whether config access should ever diverge from access to
  the owning task is undecided; until then, the API enforces access on the task only.
- **Cross-field validation placement.** Oracle validates shape, not relationships
  between fields (port uniqueness, scale monotonicity, stream-rate bounds). These checks
  stay hand-written in the Console for form UX; whether to duplicate them at the server
  write seam (RFC 0041 §4.5) is left to the NI service implementation.
