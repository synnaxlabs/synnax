# 27 - Oracle Migration System

- **Feature Name** - Oracle Migration System
- **Status** - Draft
- **Start Date** - 2026-02-22
- **Authors** - Emiliano Bonilla

# 0 - Summary

Design for a standardized migration system integrated with Oracle's schema-first code
generation. When Oracle schemas change, migrations are generated (automatically or via
prompt) in the Go layer, allowing developers to define custom migration logic. Each
schema version gets its own `migrations/vN/` sub-package containing a structural copy of
the type at that version. The migration chain maps entirely within the `migrations/`
subtree (`v1.TypeV1 → v2.TypeV2 → ...`), never importing the parent service package.
The parent package's current type is generated separately by Oracle and is structurally
identical to the highest-versioned snapshot.

# 1 - Vocabulary

- **Oracle** — Schema-first code generation tool that compiles `.oracle` files into Go,
  TypeScript, Python, C++, and Protobuf types.
- **gorp** — Type-safe ORM wrapping Pebble KV store with `Reader[K, E]` and
  `Writer[K, E]` generics. Stores entries under a canonical KV prefix derived from the
  Go type name (e.g., `__gorp__//Schematic`).
- **EntryManager** — `gorp.EntryManager[K, E]` manages key re-encoding when the key
  encoding scheme changes. Does not run schema migrations.
- **Migration** — A versioned transform that converts entries from one schema version to
  the next during server startup.
- **Auto-migrate** — Oracle-generated function (`migrations/vN/auto.gen.go`) that copies
  all 1:1 fields from the old type to the new type and calls nested auto-migrations.
  Never edited by the developer.
- **Post-migrate** — Oracle-generated template (`migrations/vN/migrate.go`) called after
  auto-migrate on each entry. The developer adds custom logic here: setting default
  values for new fields, computing derived values, transforming data.
- **KV prefix** — The key prefix under which gorp stores all entries of a type (e.g.,
  `__gorp__//Schematic`). Derived from the Go type name via `types.Name[E]()`.
- **Codec** — Serialization format used to encode/decode entries in the KV store.
  Currently msgpack; transitioning to protobuf.
- **Legacy type** — A snapshot of a type's struct definition at a previous schema
  version, stored in `migrations/vN/auto.gen.go`.
- **Snapshot** — A copy of `.oracle` source files at the time of a migration generation,
  stored in `schemas/.snapshots/<version>/` for CI diffing.

# 2 - Motivation

Several Oracle-managed types — schematics, workspace layouts, device configurations —
are stored as loosely typed blobs in the KV store. As the platform matures, these types
need stronger typing, and the storage encoding needs to transition from msgpack to
protobuf. Both changes require a systematic way to evolve stored data without data loss
or downtime.

The goals of this migration system are:

- Standardize migration tooling across all Oracle-managed types
- Strongly type server-side data structures (schematics, workspace layouts, etc.)
- Support automatic migration generation when Oracle schemas change
- Allow custom migration logic defined by the developer
- Enable eventual transition from msgpack to protobuf for storage encoding

The first migration to run through the system is the codec switch from msgpack to
protobuf for gorp-stored entries. This is a high-value change that exercises the core
migration pipeline (iterate all entries, decode with old codec, re-encode with new
codec) without requiring schema changes to any individual type. It validates the
infrastructure before more complex per-type schema migrations are needed.

MVP requirements for this use case:

- Migration execution on server startup (per-type, eager)
- Per-type version tracking (KV key, uint16)
- Oracle-generated migration code (decode msgpack → encode protobuf)
- Independent per-node execution
- CI check that migrations are generated when needed

# 3 - Design

## 1 - Migration Model

### 1 - Per-Type Versioning

Each Oracle type (e.g., `Schematic`, `Workspace`) has its own independent version
counter. Types can evolve at different rates. When type A depends on type B (e.g., a
struct field references another Oracle type), the migration system must track these
dependencies — a migration on type B may require a corresponding migration on type A.

### 2 - Two-Function Pattern

Every migration has two functions in two files — an auto-migrate (generated, never
touched) and a post-migrate (template, developer edits).

**Auto-migrate** (`migrations/vN/auto.gen.go`) — Oracle-generated, regeneratable, **never
edited by the developer**. Handles all mechanical work: copying 1:1 fields from old to
new type, calling nested auto-migrations, walking arrays of nested types. This file can
be regenerated at any time without losing work.

**Post-migrate** (`migrations/vN/migrate.go`) — Oracle-generated **once** as a template,
then owned by the developer. Called after auto-migrate on each entry. The developer adds
custom logic here: setting default values for new fields, computing derived values,
transforming data. Oracle pre-populates this with TODOs for fields it can't infer.

The runtime calls them in sequence for each entry:

```
old entry → AutoMigrate (generated) → PostMigrate (developer) → new entry
```

The primary API is a typed per-entry transform:

```go
func(ctx context.Context, old OldSchematic) (NewSchematic, error)
```

The framework handles iteration over all entries of that type, decoding with the old
type, calling the transform, and writing back the new type. For complex cases
(cross-type migrations, ontology restructuring), a raw transaction fallback is
available:

```go
func(ctx context.Context, tx gorp.Tx) error
```

The typed approach is preferred and should cover the vast majority of cases. The raw
transaction mode exists as an escape hatch, not the default path.

#### How It Works for Direct Schema Changes

Adding a `label` field to `graph.Node`:

**Auto-migrate** (`schemas/arc/graph/migrations/v1/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v1

import v2 "schemas/arc/graph/migrations/v2"

type NodeV1 struct {
    Key      string     `json:"key" msgpack:"key"`
    Type     string     `json:"type" msgpack:"type"`
    Config   any        `json:"config" msgpack:"config"`
    Position spatial.XY `json:"position" msgpack:"position"`
}

func AutoMigrateV1ToV2(ctx context.Context, old NodeV1) (v2.NodeV2, error) {
    return v2.NodeV2{
        Key:      old.Key,
        Type:     old.Type,
        Config:   old.Config,
        Position: old.Position,
        // Label: zero value (new field, set in PostMigrate)
    }, nil
}
```

**Post-migrate template** (`schemas/arc/graph/migrations/v1/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v1

import v2 "schemas/arc/graph/migrations/v2"

func PostMigrateV1ToV2(ctx context.Context, node *v2.NodeV2, old NodeV1) error {
    node.Label = "" // TODO: set default value for new field
    return nil
}
```

The developer's only job is to decide what `Label` should default to.

### 3 - File Ownership Rules

| File                     | Generated by  | Regeneratable | Developer edits |
| ------------------------ | ------------- | ------------- | --------------- |
| `vN/auto.gen.go`         | Oracle        | Yes (always)  | **Never**       |
| `vN/migrate.go`          | Oracle (once) | No            | **Yes**         |

This separation means Oracle can improve its auto-migration logic and regenerate
`auto.gen.go` files without clobbering developer work in `migrate.go`.

### 4 - Three Generation Modes

Oracle has three generation modes depending on the nature of the change:

- **Full mode**: Mechanical migrations (codec transitions). Both auto-migrate and
  post-migrate are generated. Developer touches neither file.
- **Skeleton mode**: Direct schema changes (field add/remove/rename/type change).
  Auto-migrate copies 1:1 fields. Post-migrate template has TODOs for changed fields.
  Developer edits the post-migrate.
- **Propagation mode**: A nested dependency changed. Parent auto-migrate walks the tree
  and calls nested auto-migrate + post-migrate. Parent post-migrate template is empty.
  Developer only edits the leaf post-migrate.

## 2 - Schema Change Detection

### 1 - Manual Generation with CI Enforcement

The developer manually runs `oracle migrate generate` to create migration files when
they change a schema. Oracle stores a snapshot of the schema state at the time of the
last migration generation. CI runs `oracle migrate check` which validates that the
current `.oracle` files match the latest snapshot — if the schema has changed but no
migration has been generated, CI fails. This avoids over-eager migration generation
while ensuring no schema change ships without a corresponding migration.

**CI workflow**:

1. CI runs `oracle migrate check` as part of the build pipeline.
2. The command diffs current `.oracle` files against `schemas/.snapshots/<latest>/`.
3. If schemas differ from the snapshot and no new migration files exist, CI fails with a
   message like:
   `schema changed but no migration generated. Run 'oracle migrate generate' and commit the result.`
4. If schemas match the snapshot, or new migration files exist that account for the
   changes, CI passes.

All schema changes require a migration entry. Any modification to an Oracle schema —
adding a field, removing a field, renaming, changing a field's type, changing
required/optional status — requires a corresponding migration. Explicit is better than
implicit. Even additive changes that are technically safe with msgpack deserialization
get a migration entry. This ensures a complete audit trail and prevents surprises when
the storage codec changes (e.g., msgpack → protobuf where additive changes may not be
free).

### 2 - Schema Snapshots

When `oracle migrate generate` runs, Oracle copies the current `.oracle` source files
into a versioned snapshot directory (e.g., `schemas/.snapshots/v3/schematic.oracle`). CI
diffs current `.oracle` files against the latest snapshot — if they differ and no new
migration has been generated, CI fails. Legacy types are regenerated from the snapshot
by re-running Oracle's parser on the old `.oracle` source.

This approach is human-readable, git-diffable, and uses Oracle's own source format as
the authoritative record. Developers can inspect snapshots directly to understand what
changed between migration versions. The `resolution.Table` (Oracle's internal parsed
representation) is already serializable, so re-parsing snapshots to generate legacy
types is straightforward.

### 3 - CLI Interface

#### `oracle migrate generate`

Generates migration files for all schema changes since the last snapshot.

```bash
oracle migrate generate
```

**Behavior**:

1. Diffs current `.oracle` files against the latest snapshot in
   `schemas/.snapshots/<version>/`.
2. For each changed type, determines the generation mode:
   - **Full mode**: Codec transitions (e.g., first migration for msgpack→protobuf). Both
     `_auto.gen.go` and `.go` are fully generated. Developer touches neither.
   - **Skeleton mode**: Direct schema changes (field add/remove/rename/type change).
     `_auto.gen.go` copies unchanged fields. `.go` template has TODOs for changed
     fields.
   - **Propagation mode**: A nested dependency changed. Parent `_auto.gen.go` walks the
     tree and calls nested migrations. Parent `.go` template is empty.
3. Generates files into the appropriate `migrations/` directories.
4. Regenerates `migrate.gen.go` for each affected service (appends new registration).
5. Creates a new snapshot in `schemas/.snapshots/<version+1>/`.
6. Prints a summary of generated files and which ones need developer attention.

**Output example**:

```
Generated migrations for 3 types:

  graph.Node v1→v2 (skeleton mode)
    ✏️  schemas/arc/graph/migrations/v1/migrate.go    ← EDIT THIS
    🔒 schemas/arc/graph/migrations/v1/auto.gen.go
    🔒 schemas/arc/graph/migrations/v2/auto.gen.go    (new current snapshot)

  arc.Arc v5→v6 (propagation mode)
    🔒 core/pkg/service/arc/migrations/v5/auto.gen.go
    🔒 core/pkg/service/arc/migrations/v5/migrate.go
    🔒 core/pkg/service/arc/migrations/v6/auto.gen.go  (new current snapshot)

  Updated: core/pkg/service/arc/migrations/migrate.gen.go
  Snapshot: schemas/.snapshots/v6/

Files marked ✏️ need developer attention.
Files marked 🔒 are auto-generated — do not edit.
```

#### `oracle migrate check`

Validates that all schema changes have corresponding migrations. Used in CI.

```bash
oracle migrate check
```

**Behavior**:

1. Diffs current `.oracle` files against the latest snapshot.
2. If schemas differ and no new migration files exist, exits with error code 1.
3. If schemas match the snapshot (or new migrations account for all changes), exits with
   code 0.

**CI integration** (GitHub Actions example):

```yaml
- name: Check Oracle migrations
  run: oracle migrate check
```

#### `oracle migrate regenerate` (deferred — not in MVP)

Regenerates all `_auto.gen.go` files and `migrate.gen.go` without creating new
migrations. Useful when Oracle's auto-migrate logic improves or when resolving merge
conflicts in generated files. Deferred to a later phase — not needed for the initial
implementation.

```bash
oracle migrate regenerate
```

**Behavior**:

1. Re-reads all snapshots and regenerates every `vN/auto.gen.go` file from scratch.
2. Regenerates all `migrate.gen.go` registration files.
3. Does not touch `vN/migrate.go` template files (developer-owned).
4. Does not create new snapshots or bump versions.

## 3 - Migration Execution

### 1 - Eager Startup Execution

All entries of a type are migrated during server startup before the service accepts
requests. This is consistent with how `EntryManager` already works and guarantees that
all data is in the current format at runtime — no need to support reading both old and
new formats simultaneously. The tradeoff is potentially slower startup with large
datasets, but metadata entries (schematics, workspaces, devices) are expected to number
in the thousands, not millions, so this is acceptable.

### 2 - Sequential Chaining

Migrations run in sequential order (v1→v2→v3). Each migration only handles a single
version step. When a customer skips versions (e.g., v0.48 → v0.51), all intermediate
migrations execute in order. This matches the existing `gorp.Migrator` pattern.

No rollback support. Migrations are tested before shipping to customers — development
data is disposable, so broken migrations during development are fixed by wiping local
data and re-running. If a bad migration ships to customers, a corrective migration is
released in a subsequent version.

When multiple migrations are pending (e.g., v1→v2→v3), the current design runs each
migration as a full pass over all entries:

- **v1→v2**: Iterate all entries, decode as V1, transform to V2, encode, write.
- **v2→v3**: Iterate all entries again, decode as V2, transform to V3, encode, write.

This is simple but involves multiple full passes. An optimization would be to chain
transforms in memory (decode once, apply all transforms, encode once, write once).
However, this complicates the runner because:

1. Intermediate types exist only at compile time — the runner can't dynamically chain
   `func(V1) V2` and `func(V2) V3` without generics gymnastics.
2. Raw migrations can't be chained — they need the KV state from the previous step.

Multiple full passes is the design for now. Simple, correct, and fast enough for
metadata volumes (thousands of entries, not millions). Optimize later if startup time
becomes a concern.

### 3 - Multi-Node Coordination

Each node in a Synnax cluster runs migrations on its own local KV store during startup.
There is no leader election or distributed locking for migrations. This works because
Aspen gossip replicates metadata — each node holds a local copy that it can migrate
independently. The implication is that migrations must be deterministic and idempotent:
given the same input data, every node must produce the same output.

### 4 - Integration with Existing Infrastructure

Oracle is a code generation tool — it generates migration files but does not own
runtime migration execution. The migration runtime lives in gorp.

`Migration` is an interface in gorp. Concrete implementations include:

- **`TypedMigration[I, O]`**: Per-entry transform migration. Iterates all entries under
  the canonical KV prefix, decodes each as type `I`, calls auto-migrate + post-migrate
  to produce type `O`, encodes, and writes back. Oracle generates the auto-migrate and
  post-migrate functions that plug into this.
- **`RawMigration`**: Escape hatch for cross-type migrations. Receives a `gorp.Tx` and
  the developer writes arbitrary logic. Used for complex cases like ranger group
  restructuring or RBAC migrations.
- **`KeyMigration[K, E]`**: Re-encodes keys when the key encoding scheme changes.
  This is the existing `reEncodeKeys` logic, now expressed as a `Migration`.

`gorp.EntryManager[K, E]` is the single entry point for all migrations. It accepts an
ordered list of `Migration`s and runs them sequentially by version counter during
`OpenEntryManager`. Key re-encoding is implicitly appended as the final migration —
services don't need to think about it.

**Startup sequence** (each service):

```
1. gorp.OpenEntryManager[K, E](ctx, db, migrations.All()...)
   → Reads __gorp__/<type>/version from KV store
   → Runs pending schema migrations in order (v1→v2→v3...)
   → Implicitly runs key re-encoding as the final step
   → Updates version counter
   → (future) builds/updates indexes

2. Service accepts requests
```

Each service's `migrations/migrate.gen.go` exports an `All()` function returning the
ordered `[]gorp.Migration` list. Oracle generates this file and appends new registrations
on each `oracle migrate generate` run. The developer never edits this file.

The version counter is stored at `__gorp__//<Type>/version` as a uint16 (big-endian,
max 65,535 migrations). This is a new key format — existing `gorp.Migrator` keys for
non-Oracle types are untouched.

`GorpMarshal`/`GorpUnmarshal` methods are generated by `oracle sync` (the `go/types`
plugin), not by the migration plugin. They are a normal type generation concern.

```
Oracle (build time)              gorp.EntryManager (runtime)
  │                                │
  ├─ oracle sync                   ├─ reads __gorp__/<type>/version
  │   ├─ generates types.gen.go    ├─ runs pending migrations in order:
  │   ├─ generates GorpMarshal/    │   ├─ TypedMigration: decode → transform → encode
  │   │  GorpUnmarshal             │   ├─ RawMigration: arbitrary gorp.Tx logic
  │   └─ generates pb/             │   └─ KeyMigration: re-encode keys
  │                                ├─ increments version counter
  ├─ oracle migrate generate       └─ (future) manages indexes
  │   ├─ generates vN/ snapshots
  │   ├─ generates auto-migrate
  │   ├─ generates post-migrate
  │   └─ generates migrate.gen.go
  │
  └─ oracle migrate check
      └─ CI enforcement
```

## 4 - Code Generation

### 1 - Legacy Type Snapshots

When the developer runs `oracle migrate generate`, Oracle snapshots the current type
definition into the `migrations/` directory as a Go struct before applying the schema
change. The developer then writes the transform function that maps the old type to the
new type. This eliminates boilerplate and ensures the legacy type exactly matches what
was previously stored. Oracle maintains an internal record of the schema state at each
migration point to enable accurate snapshot generation.

### 2 - Migration Stub Generation

Every migration version produces a `vN/` Go sub-package with two files:

- **`vN/auto.gen.go`**: Auto-migrate function + legacy type snapshot. Oracle-generated,
  regeneratable, never edited. Contains the structural copy of the type at version N and
  a transform that maps `vN.TypeVN → v(N+1).TypeV(N+1)`. Post-codec-transition versions
  also include `GorpMarshal`/`GorpUnmarshal` methods using snapshotted protobuf
  definitions.
- **`vN/migrate.go`**: Post-migrate template. Oracle-generated once, then owned by the
  developer. Contains a post-migrate function called after auto-migrate. Oracle
  pre-populates TODOs for new/changed/removed fields.

This replaces the old "empty skeleton" approach. The auto-migrate handles all mechanical
work; the developer only writes logic for fields that require human judgment.

### 3 - Migration Registration

Legacy types and migration functions live co-located with the service that owns the
type. Each schema version gets its own `vN/` Go sub-package:

```
core/pkg/service/schematic/
├── types.gen.go           # Current Schematic type (Oracle-generated)
├── marshal.gen.go         # GorpMarshal/GorpUnmarshal for current type
├── service.go             # Service logic
├── pb/                    # Current protobuf definitions
└── migrations/
    ├── migrate.gen.go     # GENERATED: Migrator() function (never edit)
    ├── v1/                # Pre-transition snapshot (msgpack era)
    │   ├── auto.gen.go    # SchematicV1 + AutoMigrateV1ToV2 → v2.SchematicV2
    │   └── migrate.go     # PostMigrateV1ToV2 (developer edits)
    ├── v2/                # Post-transition snapshot
    │   ├── auto.gen.go    # SchematicV2 + GorpMarshal/Unmarshal + AutoMigrateV2ToV3
    │   ├── migrate.go     # PostMigrateV2ToV3 (developer edits)
    │   └── pb/            # Snapshotted protobuf for V2
    └── v3/                # Current snapshot (no AutoMigrate yet)
        ├── auto.gen.go    # SchematicV3 + GorpMarshal/Unmarshal
        └── pb/            # Snapshotted protobuf for V3
```

**Key structural rules:**

- Each `vN/` is a separate Go package. Migration functions map `vN.TypeVN →
  v(N+1).TypeV(N+1)` — they never import the parent service package. This eliminates
  circular dependency risk.
- The highest-numbered `vN/` is the "current" snapshot. It has no `AutoMigrate` or
  `PostMigrate` yet — Oracle adds those when the next migration is created, then creates
  `v(N+1)/` as the new current.
- Pre-codec-transition versions (e.g., `v1/`) have no `pb/` directory or
  `GorpUnmarshaler` — they decode using the DB's generic msgpack codec.
- Post-codec-transition versions (e.g., `v2/`, `v3/`) include snapshotted `.proto` files
  in a `pb/` sub-package and implement `GorpMarshal`/`GorpUnmarshal` using those
  snapshotted protos. The `.proto` files are regenerated from snapshotted `.oracle`
  source files — Oracle can reproduce them deterministically.
- The parent package's current type (in `types.gen.go`) is structurally identical to the
  highest `vN/` snapshot. Oracle ensures this by generating both from the same `.oracle`
  source.

**`migrate.gen.go` is Oracle-generated and regenerated on every
`oracle migrate generate` run.** It exports an `All()` function that returns an ordered
`[]gorp.Migration` list. Oracle owns the registration of all migrations. When a new
migration is generated, Oracle appends the new `gorp.NewTypedMigration` call to
`migrate.gen.go`. The developer never edits this file — Oracle maintains it as the
single source of truth for migration ordering.

### 4 - Protobuf Marshaling Interfaces

A generic `ProtobufCodec` implementing `binary.Codec` with `Encode(ctx, any)` is
problematic — protobuf marshaling is type-specific (each proto message has its own
generated `Marshal`/`Unmarshal`). A generic codec would need type assertions or
reflection, which is fragile.

Gorp entries optionally implement marshaling interfaces. Oracle generates these methods
using the protobuf translators it already produces. Gorp checks for the interface; if
not implemented, it falls back to the generic DB codec (msgpack for legacy types).

```go
// x/go/gorp/marshal.go

// GorpMarshaler is an optional interface that entries can implement to
// control their own serialization. When implemented, gorp uses this
// instead of the generic DB codec.
type GorpMarshaler interface {
    GorpMarshal(ctx context.Context) ([]byte, error)
}

// GorpUnmarshaler is the decoding counterpart.
type GorpUnmarshaler interface {
    GorpUnmarshal(ctx context.Context, data []byte) error
}
```

**Writer changes** (`writer.go`):

```go
func (w *Writer[K, E]) set(ctx context.Context, entry E) error {
    var data []byte
    var err error
    if m, ok := any(&entry).(GorpMarshaler); ok {
        data, err = m.GorpMarshal(ctx)
    } else {
        data, err = w.Encode(ctx, entry)
    }
    if err != nil {
        return err
    }
    return w.BaseWriter.Set(ctx, w.keyCodec.encode(entry.GorpKey()), data,
        entry.SetOptions()...)
}
```

**Note**: The type assertion uses `any(&entry)` (pointer), not `any(entry)` (value).
`GorpMarshal` has a pointer receiver (`*Schematic`), and Go's type assertion on an
interface only matches if the concrete type in the interface matches exactly. Passing a
value would fail to find the pointer-receiver method.

**Reader/Iterator changes** (`reader.go`):

```go
func (k *Iterator[E]) Value(ctx context.Context) (entry *E) {
    k.value = new(E)
    if u, ok := any(k.value).(GorpUnmarshaler); ok {
        if err := u.GorpUnmarshal(ctx, k.Iterator.Value()); err != nil {
            k.err = err
            return nil
        }
    } else {
        if err := k.decoder.Decode(ctx, k.Iterator.Value(), k.value); err != nil {
            k.err = err
            return nil
        }
    }
    return k.value
}
```

**Oracle-generated methods** (in `types.gen.go`):

```go
func (s *Schematic) GorpMarshal(ctx context.Context) ([]byte, error) {
    pb, err := pb.SchematicToPB(ctx, *s)
    if err != nil {
        return nil, err
    }
    return proto.Marshal(pb)
}

func (s *Schematic) GorpUnmarshal(ctx context.Context, data []byte) error {
    pbMsg := &pb.Schematic{}
    if err := proto.Unmarshal(data, pbMsg); err != nil {
        return err
    }
    result, err := pb.SchematicFromPB(ctx, pbMsg)
    if err != nil {
        return err
    }
    *s = result
    return nil
}
```

**This naturally solves the old/new codec problem for migrations:**

- **Pre-transition legacy types** (e.g., `v1/`): Don't implement `GorpUnmarshaler`. The
  migration runner decodes them with the generic DB codec (msgpack). No `pb/`
  sub-package needed.
- **Post-transition legacy types** (e.g., `v2/`): Implement
  `GorpMarshal`/`GorpUnmarshal` using snapshotted protobuf definitions in their own
  `pb/` sub-package. Each version decodes with its own proto definition — field numbers
  are irrelevant across versions because data is never read with a different version's
  proto.
- **Current types** (in `types.gen.go`): Implement `GorpMarshaler`/`GorpUnmarshaler`.
  Written back using protobuf.
- **No explicit codec declaration per-migration needed.** The type itself determines its
  encoding format.

## 5 - Protobuf Transition

### 1 - Codec Transition Scope

The goal is a full codec replacement — all Oracle-managed types stored via gorp switch
from msgpack to protobuf encoding. The migration system handles the format transition
(decode old entries with msgpack, re-encode with protobuf). If a full cutover proves
impractical, a per-type gradual migration is acceptable, but the strong preference is a
clean, complete switch. Oracle already generates `.proto` files and Go translators, so
the protobuf infrastructure exists.

Since Oracle owns the schema and generates both msgpack-compatible Go structs and
protobuf definitions + translators, the codec transition should be Oracle-generated
rather than manually written per-type. The exact mechanism (gorp-layer detection vs.
explicit migration entry) is an implementation detail to be resolved during development.
The key principle is that Oracle should be able to generate all the code needed for the
codec switch — the developer shouldn't have to write repetitive msgpack→protobuf
boilerplate for each type.

### 2 - Protobuf Field Number Stability

Protobuf field number stability across schema versions does not matter because each
version snapshot includes its own protobuf definitions. Version N's data is always
decoded with version N's proto, never with version N+1's proto. The migration chain
decodes with the old version's `GorpUnmarshal` (using the snapshotted proto in
`vN/pb/`), transforms to the new version's type, then encodes with the new version's
`GorpMarshal` (using the snapshotted proto in `v(N+1)/pb/`). Oracle can freely assign
and reassign field numbers on each generation without stability constraints.

### 3 - Protobuf Struct Precision for JSON Fields

JSON fields (`Schematic.data`, `Workspace.layout`) originate from TypeScript/JavaScript
where all numbers are float64. The `google.protobuf.Struct` type's float64-only numeric
representation matches the source data — no precision loss in practice. When these
fields are eventually promoted to full Oracle types, they'll get proper protobuf
messages with correct numeric types.

### 4 - Codec Transition Startup Sequence

After the protobuf switch, the DB's generic codec remains msgpack (unchanged). It serves
as the fallback for types that don't implement the marshaling interfaces. The startup
sequence:

```
1. gorp.Wrap(kvStore, gorp.WithCodec(&MsgPackCodec{}))
   ↳ Generic codec stays msgpack — used as fallback for pre-transition legacy types

2. migrations.Migrator().Run(ctx, db)
   ↳ Migration v1→v2: reads raw bytes from KV
   ↳ Decodes using MsgPackCodec into v1.SchematicV1 (no GorpUnmarshaler)
   ↳ Calls transform: v1.SchematicV1 → v2.SchematicV2
   ↳ Encodes using v2.SchematicV2.GorpMarshal() → protobuf bytes
   ↳ Writes back to KV

3. gorp.OpenEntryManager[K, Schematic]()
   ↳ reEncodeKeys reads entries — Schematic implements GorpUnmarshaler
   ↳ All data is already protobuf, this is effectively a no-op

4. Service accepts requests
   ↳ All reads use GorpUnmarshal (protobuf)
   ↳ All writes use GorpMarshal (protobuf)
   ↳ Generic MsgPackCodec never touched at runtime
```

**After migration completes, DB state is clean**: every entry is protobuf, no ambiguity,
no fallback codec at runtime. The generic msgpack codec exists only for the migration
runner to decode legacy types.

## 6 - Nested and Shared Type Migrations

### 1 - Dependency Tracking

Oracle's transitive dependency tracking detects when a type references another Oracle
type. When type B changes and type A contains a field of type B, Oracle flags that a
migration is needed for type A as well. The CI check fails if type B changes but type A
doesn't have a corresponding migration.

### 2 - Nested Type Propagation

When a nested type changes (e.g., `graph.Node` gains a `label` field), Oracle's
dependency graph detects that a gorp entry (e.g., `Arc`) contains `graph.Node`. Oracle:

1. Bumps the gorp entry's version.
2. Generates the auto-migrate for the parent that walks the tree and calls Node's
   auto-migrate and post-migrate on each node.
3. Generates an empty post-migrate template for the parent (in case the developer needs
   custom parent-level logic too).

**Parent auto-migrate** (`core/pkg/service/arc/migrations/v5/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v5

import (
    v6 "core/pkg/service/arc/migrations/v6"
    graphv1 "schemas/arc/graph/migrations/v1"
    graphv2 "schemas/arc/graph/migrations/v2"
)

func AutoMigrateV5ToV6(ctx context.Context, old ArcV5) (v6.ArcV6, error) {
    nodes := make([]graphv2.NodeV2, len(old.Graph.Nodes))
    for i, n := range old.Graph.Nodes {
        migrated, err := graphv1.AutoMigrateV1ToV2(ctx, n)
        if err != nil { return v6.ArcV6{}, err }
        if err := graphv1.PostMigrateV1ToV2(ctx, &migrated, n); err != nil {
            return v6.ArcV6{}, err
        }
        nodes[i] = migrated
    }
    return v6.ArcV6{
        Key:    old.Key,
        Name:   old.Name,
        Mode:   old.Mode,
        Graph:  v6.GraphV6{
            Viewport:  old.Graph.Viewport,
            Functions: old.Graph.Functions,
            Edges:     old.Graph.Edges,
            Nodes:     nodes,
        },
        Text:   old.Text,
        Module: old.Module,
        Status: old.Status,
    }, nil
}
```

**Parent post-migrate template** (`core/pkg/service/arc/migrations/v5/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v5

import v6 "core/pkg/service/arc/migrations/v6"

func PostMigrateV5ToV6(ctx context.Context, a *v6.ArcV6, old ArcV5) error {
    // No additional Arc-level changes needed for this migration.
    // Edit this function to add custom logic if required.
    return nil
}
```

For a pure nested propagation, the developer doesn't touch the parent post-migrate at
all.

### 3 - Shared Nested Types

When a nested type is referenced by multiple gorp entries (e.g., `ir.Edge` used by both
`Arc` and `Schematic`), Oracle generates parent auto-migrations for **every** entry that
contains the changed type. All generated auto-migrations call the same leaf
auto-migrate + post-migrate functions. The leaf transform logic is written once.

### 4 - Execution Order

Nested type migrations don't execute independently — they're called _within_ the parent
entry's auto-migrate. Only gorp entries (types with KV prefixes) have their own
migration runner. Nested types are migrated as part of their parent's
read-transform-write cycle.

### 5 - Cross-Type Migration Ordering

Packages don't have circular dependencies. Services start in dependency order — if
Workspace depends on Schematic, Schematic's service (and its migrations) runs first. By
the time Workspace's migrations execute, everything Workspace depends on is already
migrated. No additional ordering mechanism needed.

## 7 - Strongly Typing JSON Fields

### 1 - Incremental Adoption

The end goal is to define the complete nested structure of complex fields (schematic
data, workspace layout, etc.) as Oracle types — symbols, connections, positions, the
entire tree. Oracle generates Go/TS/Python types for maximum type safety.

The adoption path is incremental: start by defining the top-level structure as an Oracle
type while nested fields remain `json`/`any`. Then use the migration system itself to
progressively promote nested fields to strongly typed Oracle types. Each promotion step
is a schema change + migration, eating the elephant one bite at a time.

### 2 - Unknown JSON Keys

When promoting a `json` field to a struct, unknown keys are silently dropped. The
migration only copies keys that map to struct fields. Rationale:

1. **The server is adopting the client's type definition.** The client already ignores
   unknown keys when deserializing. The server should behave the same way.
2. **Unknown keys are noise, not data.** They come from deprecated fields, experimental
   client features, or typos. Preserving them adds complexity with no value.
3. **No catch-all field.** A `map[string]any` overflow field defeats the purpose of
   strong-typing and creates a permanent escape hatch that never gets cleaned up.
4. **If specific data needs preserving**, the developer can explicitly copy it in the
   post-migrate function before the migration drops it.

### 3 - Client-Server Source of Truth

Oracle-generated types will coexist with manually defined TypeScript types during the
transition. Over time, manual client-side definitions are migrated to Oracle schemas.
Some client-only types (UI state, local caching) may always remain manual. The migration
system supports this by allowing the server to handle schema evolution independently of
the client's adoption timeline.

## 8 - Testing

### 1 - Test Helpers

The migration framework provides test utilities that reduce boilerplate for migration
authors. The primary helper is `oracle.TestMigration` which handles setup, execution,
and assertion in a single call.

**Concrete example** (`core/pkg/service/schematic/migrations/v2/v2_test.go`):

```go
package v2_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"

    "github.com/synnaxlabs/oracle"
    v2 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v2"
    v3 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v3"
)

var _ = Describe("SchematicV2ToV3", func() {
    It("Should set description to nil for existing entries", func() {
        oracle.TestMigration(
            // Old entries to seed into the KV store.
            []v2.SchematicV2{
                {Key: uuid.New(), Name: "My Schematic", Data: someJSON},
            },
            // Expected new entries after migration.
            []v3.SchematicV3{
                {Key: /* same */, Name: "My Schematic", Data: someJSON, Description: nil},
            },
            // The migration to run.
            oracle.RegisterTyped(
                "add_description",
                v2.AutoMigrateV2ToV3,
                v2.PostMigrateV2ToV3,
            ),
        )
    })
})
```

**What `TestMigration` does internally**:

1. Creates an in-memory KV store
2. Encodes and writes old entries under the canonical prefix
3. Runs the migration function
4. Reads back all entries, decodes as the new type
5. Asserts results match expected entries (using `gomega.Equal`)
6. Validates the version counter was incremented

**Testing nested migrations**:

```go
var _ = Describe("NodeV1ToV2 (nested)", func() {
    It("Should set label to empty string for existing nodes", func() {
        // For nested types, test the leaf migration directly.
        old := v1.NodeV1{Key: "node-1", Type: "opc", Position: spatial.XY{X: 10, Y: 20}}
        migrated, err := v1.AutoMigrateV1ToV2(ctx, old)
        Expect(err).ToNot(HaveOccurred())
        Expect(v1.PostMigrateV1ToV2(ctx, &migrated, old)).To(Succeed())
        Expect(migrated.Label).To(Equal(""))
    })
})
```

Nested type migrations are tested as pure functions (no KV store needed) since they
execute within the parent's migration runner.

## 9 - Scope and Boundaries

### 1 - Non-Oracle Types

Non-Oracle gorp-stored types (Cesium internals, Aspen KV metadata, ontology resources)
retain their existing migration mechanisms. This RFC covers only Oracle-managed types.

### 2 - Version Counter Storage

Each Oracle-managed type gets a dedicated KV key (e.g., `__oracle__/schematic/version`)
that stores its current migration version. All entries of that type are assumed to be at
the same version after startup migration completes. This is consistent with the eager
migration model — once startup finishes, every entry has been migrated. No version field
is added to individual entries.

The version counter is uint16 (max 65,535 migrations per type). This is an upgrade from
the current uint8 (255) to future-proof against the cumulative effect of requiring
migrations for all schema changes. 65,535 is more than sufficient while remaining cheap
to store (2 bytes in the KV store).

### 3 - Relationship to RFC 0025

The Oracle migration system builds the server-side infrastructure for schema evolution.
RFC 0025 (moving client-side migrations to the server) will layer on top once server
types are strongly typed via Oracle. The sequence is:

1. **This RFC (0027)**: Oracle migration system + msgpack→protobuf codec transition
2. **Strongly type JSON fields**: Incrementally promote `json` fields to Oracle types
3. **RFC 0025**: Server owns all schema evolution; client sends/receives Oracle-typed
   data instead of managing its own migrations

# 4 - Migration Scenarios

This section walks through every known migration pattern — real and anticipated — with
concrete code showing what the migration file, legacy type, and transform function would
look like under the proposed system. The purpose is to stress-test the API design before
building it.

## Scenario 1: Codec Transition (msgpack → protobuf)

**Context**: The MVP. Every Oracle-managed type switches storage encoding. No schema
change — same fields, same types, different wire format.

**Auto-migrate** (`core/pkg/service/schematic/migrations/v1/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v1

import (
    "context"

    "github.com/google/uuid"
    "github.com/synnaxlabs/x/go/binary"
    v2 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v2"
)

// SchematicV1 is the legacy type snapshot. Same fields, msgpack encoding.
// No GorpUnmarshaler — decoded using the generic DB codec (msgpack).
type SchematicV1 struct {
    Key      uuid.UUID                 `json:"key" msgpack:"key"`
    Name     string                    `json:"name" msgpack:"name"`
    Data     binary.MsgpackEncodedJSON `json:"data" msgpack:"data"`
    Snapshot bool                      `json:"snapshot" msgpack:"snapshot"`
}

func (s SchematicV1) GorpKey() uuid.UUID { return s.Key }
func (s SchematicV1) SetOptions() []any  { return nil }

func AutoMigrateV1ToV2(
    ctx context.Context,
    old SchematicV1,
) (v2.SchematicV2, error) {
    return v2.SchematicV2{
        Key:      old.Key,
        Name:     old.Name,
        Data:     old.Data,
        Snapshot: old.Snapshot,
    }, nil
}
```

**Post-migrate template** (`core/pkg/service/schematic/migrations/v1/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v1

import (
    "context"

    v2 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v2"
)

func PostMigrateV1ToV2(
    ctx context.Context,
    s *v2.SchematicV2,
    old SchematicV1,
) error {
    // Codec transition only — no field changes. Nothing to do.
    return nil
}
```

**Observations**:

- The developer writes nothing. Both files are Oracle-generated. The auto-migrate copies
  all fields 1:1. The post-migrate is empty because there are no schema changes.
- `v1/` is a pre-transition snapshot: no `pb/` directory, no `GorpUnmarshaler`. The
  runner decodes using the generic DB codec (msgpack).
- `v2/` is a post-transition snapshot: has `pb/` with snapshotted proto, implements
  `GorpMarshal`/`GorpUnmarshal`. The runner encodes the output using `GorpMarshal` →
  protobuf bytes.
- Every Oracle-managed type (~15+ types) gets the same pair of `vN/` packages.
  Repetitive but explicit, auditable, and consistent with "all changes require
  migration."

---

## Scenario 2: Add a Field

**Context**: Add a `description` field to `Workspace`. Additive change — old entries
don't have it, new entries do.

**Schema change** (`schemas/workspace.oracle`):

```diff
 Workspace struct {
     key    Key    { @key }
     name   string { @validate required }
     author uuid?
     layout json   { @ts preserve_case }
+    description string?
 }
```

**Auto-migrate** (`core/pkg/service/workspace/migrations/v2/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v2

import v3 "github.com/synnaxlabs/synnax/core/pkg/service/workspace/migrations/v3"

// WorkspaceV2 is the schema at version 2 (before description was added).
type WorkspaceV2 struct {
    Key    uuid.UUID                 `json:"key" msgpack:"key"`
    Name   string                    `json:"name" msgpack:"name"`
    Author uuid.UUID                 `json:"author" msgpack:"author"`
    Layout binary.MsgpackEncodedJSON `json:"layout" msgpack:"layout"`
}

func AutoMigrateV2ToV3(
    ctx context.Context,
    old WorkspaceV2,
) (v3.WorkspaceV3, error) {
    return v3.WorkspaceV3{
        Key:    old.Key,
        Name:   old.Name,
        Author: old.Author,
        Layout: old.Layout,
        // Description: zero value (new field, set in PostMigrate)
    }, nil
}
```

**Post-migrate template** (`core/pkg/service/workspace/migrations/v2/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v2

import v3 "github.com/synnaxlabs/synnax/core/pkg/service/workspace/migrations/v3"

func PostMigrateV2ToV3(
    ctx context.Context,
    w *v3.WorkspaceV3,
    old WorkspaceV2,
) error {
    // TODO: set default value for new field 'Description'
    w.Description = nil
    return nil
}
```

**Observations**:

- The auto-migrate copies all 1:1 fields. The post-migrate template has a TODO for the
  new `Description` field. The developer decides the default value (here, `nil`).
- The two-file split means Oracle can regenerate `v2/auto.gen.go` freely (e.g., if the
  auto-migrate logic improves) without clobbering the developer's `v2/migrate.go`.
- With msgpack, this migration is technically unnecessary — missing fields decode as
  zero values. But all changes require a migration, so it exists for the audit trail and
  for protobuf compatibility.

---

## Scenario 3: Remove a Field

**Context**: Remove the `author` field from `Workspace` (hypothetical).

**Schema change**:

```diff
 Workspace struct {
     key    Key    { @key }
     name   string { @validate required }
-    author uuid?
     layout json   { @ts preserve_case }
 }
```

**Auto-migrate** (`core/pkg/service/workspace/migrations/v3/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v3

import v4 "github.com/synnaxlabs/synnax/core/pkg/service/workspace/migrations/v4"

type WorkspaceV3 struct {
    Key    uuid.UUID                 `json:"key" msgpack:"key"`
    Name   string                    `json:"name" msgpack:"name"`
    Author uuid.UUID                 `json:"author" msgpack:"author"`
    Layout binary.MsgpackEncodedJSON `json:"layout" msgpack:"layout"`
}

func AutoMigrateV3ToV4(
    ctx context.Context,
    old WorkspaceV3,
) (v4.WorkspaceV4, error) {
    return v4.WorkspaceV4{
        Key:    old.Key,
        Name:   old.Name,
        Layout: old.Layout,
        // Author: deliberately dropped (removed field)
    }, nil
}
```

**Post-migrate template** (`core/pkg/service/workspace/migrations/v3/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v3

import v4 "github.com/synnaxlabs/synnax/core/pkg/service/workspace/migrations/v4"

func PostMigrateV3ToV4(
    ctx context.Context,
    w *v4.WorkspaceV4,
    old WorkspaceV3,
) error {
    // Field 'Author' was removed. No action needed unless you want to
    // preserve the data elsewhere.
    return nil
}
```

**Observations**:

- The auto-migrate omits the removed field — clean and explicit. The post-migrate
  template is generated with a comment about the removed field.
- The developer only touches the post-migrate if they need to preserve the removed data
  elsewhere (e.g., copy Author to an audit log).
- With protobuf, this also reclaims storage (the field is no longer serialized).

---

## Scenario 4: Rename a Field

**Context**: Rename `data` to `spec` in `Schematic` (hypothetical).

**Schema change**:

```diff
 Schematic struct {
     key      Key    { @key }
     name     string
-    data     json   { @ts preserve_case }
+    spec     json   { @ts preserve_case }
     snapshot bool
 }
```

**Auto-migrate** (`core/pkg/service/schematic/migrations/v3/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v3

import v4 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v4"

type SchematicV3 struct {
    Key      uuid.UUID                 `json:"key" msgpack:"key"`
    Name     string                    `json:"name" msgpack:"name"`
    Data     binary.MsgpackEncodedJSON `json:"data" msgpack:"data"`
    Snapshot bool                      `json:"snapshot" msgpack:"snapshot"`
}

func AutoMigrateV3ToV4(
    ctx context.Context,
    old SchematicV3,
) (v4.SchematicV4, error) {
    return v4.SchematicV4{
        Key:      old.Key,
        Name:     old.Name,
        // Spec: cannot infer mapping (new field, set in PostMigrate)
        // Data was removed — see PostMigrate
        Snapshot: old.Snapshot,
    }, nil
}
```

**Post-migrate template** (`core/pkg/service/schematic/migrations/v3/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v3

import v4 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v4"

func PostMigrateV3ToV4(
    ctx context.Context,
    s *v4.SchematicV4,
    old SchematicV3,
) error {
    // TODO: field 'Data' was removed and 'Spec' was added.
    // If this is a rename, map the old field to the new one:
    s.Spec = old.Data
    return nil
}
```

**Observations**:

- The auto-migrate copies all unchanged fields (Key, Name, Snapshot) but can't infer the
  rename. Oracle generates the auto-migrate with the unchanged fields and leaves the
  renamed/new/removed fields to the post-migrate.
- This is where the post-migrate shines: the developer writes one line mapping old field
  to new field name. Oracle can't know that `data` became `spec` without developer
  input, so the post-migrate template has TODOs for both the removed and added fields.

---

## Scenario 5: Strongly Type a JSON Field (Incremental)

**Context**: Promote `Schematic.data` from `json` to a top-level struct, but keep nested
fields as `json` initially.

**Schema change**:

```diff
+SchematicData struct {
+    symbols json { @ts preserve_case }
+    connections json { @ts preserve_case }
+    viewport json { @ts preserve_case }
+}

 Schematic struct {
     key      Key    { @key }
     name     string
-    data     json   { @ts preserve_case }
+    data     SchematicData
     snapshot bool
 }
```

**Auto-migrate** (`core/pkg/service/schematic/migrations/v4/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v4

import v5 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v5"

type SchematicV4 struct {
    Key      uuid.UUID                 `json:"key" msgpack:"key"`
    Name     string                    `json:"name" msgpack:"name"`
    Data     binary.MsgpackEncodedJSON `json:"data" msgpack:"data"`
    Snapshot bool                      `json:"snapshot" msgpack:"snapshot"`
}

func AutoMigrateV4ToV5(
    ctx context.Context,
    old SchematicV4,
) (v5.SchematicV5, error) {
    return v5.SchematicV5{
        Key:      old.Key,
        Name:     old.Name,
        // Data: type changed from json to SchematicData (set in PostMigrate)
        Snapshot: old.Snapshot,
    }, nil
}
```

**Post-migrate template** (`core/pkg/service/schematic/migrations/v4/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v4

import v5 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v5"

func PostMigrateV4ToV5(
    ctx context.Context,
    s *v5.SchematicV5,
    old SchematicV4,
) error {
    // TODO: field 'Data' changed type from json to SchematicData.
    // Parse the unstructured JSON into the new structured type.
    rawData := old.Data.Map()
    s.Data = v5.SchematicDataV5{
        Symbols:     rawData["symbols"],
        Connections: rawData["connections"],
        Viewport:    rawData["viewport"],
    }
    return nil
}
```

**Observations**:

- The auto-migrate copies all unchanged fields (Key, Name, Snapshot). The type-changed
  field (`Data`) is left to the post-migrate because Oracle can't infer how to parse
  unstructured JSON into a struct.
- The post-migrate template has a TODO and a suggested pattern. The developer extracts
  known keys from the JSON blob and maps them to struct fields.
- Unknown keys in the JSON blob are silently dropped. See the resolved open question
  below.

---

## Scenario 6: Change a Field's Type

**Context**: Change `Workspace.author` from `uuid?` to a new `Author` struct
(hypothetical).

**Schema change**:

```diff
+Author struct {
+    key  uuid
+    name string
+}

 Workspace struct {
     key    Key    { @key }
     name   string { @validate required }
-    author uuid?
+    author Author?
     layout json   { @ts preserve_case }
 }
```

**Auto-migrate** (`core/pkg/service/workspace/migrations/v5/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v5

import v6 "github.com/synnaxlabs/synnax/core/pkg/service/workspace/migrations/v6"

type WorkspaceV5 struct {
    Key    uuid.UUID                 `json:"key" msgpack:"key"`
    Name   string                    `json:"name" msgpack:"name"`
    Author uuid.UUID                 `json:"author" msgpack:"author"`
    Layout binary.MsgpackEncodedJSON `json:"layout" msgpack:"layout"`
}

func AutoMigrateV5ToV6(
    ctx context.Context,
    old WorkspaceV5,
) (v6.WorkspaceV6, error) {
    return v6.WorkspaceV6{
        Key:    old.Key,
        Name:   old.Name,
        // Author: type changed from uuid? to Author? (set in PostMigrate)
        Layout: old.Layout,
    }, nil
}
```

**Post-migrate template** (`core/pkg/service/workspace/migrations/v5/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v5

import v6 "github.com/synnaxlabs/synnax/core/pkg/service/workspace/migrations/v6"

func PostMigrateV5ToV6(
    ctx context.Context,
    w *v6.WorkspaceV6,
    old WorkspaceV5,
) error {
    // TODO: field 'Author' changed type from uuid? to Author?.
    if old.Author != uuid.Nil {
        w.Author = &v6.AuthorV6{
            Key:  old.Author,
            Name: "", // TODO: resolve user name (requires DB access)
        }
    }
    return nil
}
```

**Observations**:

- The auto-migrate copies unchanged fields. The type-changed field (`Author`) is left to
  the post-migrate.
- **This stretches the typed per-entry API.** The post-migrate needs to look up a `User`
  by UUID to populate `Author.Name`, but the signature `func(ctx, *new, old) error`
  doesn't provide database access.
- **Options**:
  1. Accept incomplete data (set `Name: ""`) and backfill later.
  2. Pass additional context (like a user lookup function) via the `context.Context`.
  3. Use the raw transaction escape hatch for this migration instead.
- This is a real case where the escape hatch may be needed, or the typed API needs to be
  enriched with injectable dependencies via context.

---

## Scenario 7: Port Existing Ranger Migration (Cross-Type Structural)

**Context**: The existing `migrateRangeGroups` migration restructures ontology
relationships — it reads groups, queries children, creates new parent ranges, and
rewires ontology edges. This is a cross-type, cross-service migration.

**Under the new system** (`core/pkg/service/ranger/migrations/v1.go`):

```go
func MigrateV1RangeGroups(ctx context.Context, tx gorp.Tx) error {
    // This is the raw transaction escape hatch — the typed per-entry API
    // cannot express this migration because it:
    // 1. Reads from multiple types (Group, Range, ontology.Resource)
    // 2. Creates new entries (Range)
    // 3. Modifies ontology relationships
    // 4. Deletes groups

    // ... existing migrateRangeGroups logic, unchanged ...
}
```

**Observations**:

- This migration fundamentally cannot use the typed per-entry API. It operates across
  types, creates new entries, and modifies external systems (ontology).
- The raw transaction escape hatch is essential for this case.
- **Question**: How does this migration get access to the ontology writer and group
  service? The current code accesses them via `s.cfg.Ontology` and `s.cfg.Group` on the
  service struct. Under the new system, do migrations receive a dependency injection
  context?

---

## Scenario 8: Port Existing RBAC Migration (Cross-Type + External State)

**Context**: The RBAC migration reads legacy policies, determines user roles based on
policy analysis, assigns roles, and deletes legacy policies. It accesses the user
service, role service, and ontology.

**Under the new system**:

```go
func MigrateV1RBAC(ctx context.Context, tx gorp.Tx) error {
    // Raw transaction escape hatch.
    // Needs access to: User service, Role service, Ontology, provisioned role keys.
    // This migration can't be typed per-entry because:
    // 1. It reads LegacyPolicy entries
    // 2. It reads User entries
    // 3. It writes Role assignments
    // 4. It deletes LegacyPolicy entries
    // 5. It depends on external state (provisioned role keys)

    // ... existing MigratePermissions logic ...
}
```

**Observations**:

- Same as Scenario 7: raw transaction escape hatch is required.
- **Additional concern**: This migration depends on external state (provisioned role
  keys from `ProvisionResult`). How are external dependencies injected into migration
  functions? The migration function signature needs access to more than just `gorp.Tx`.

---

## Scenario 9: Transitive Dependency — Type B Changes, Type A References B

**Context**: `Device` has a `rack` field of type `rack.Key`. If `rack.Key` changes from
`uint32` to `uint64`, Device also needs a migration even though its schema didn't change
directly.

**Rack schema change**:

```diff
-Key = uint32
+Key = uint64
```

**Device migration needed** (`core/pkg/service/device/migrations/v2/auto.gen.go`):

```go
package v2

import v3 "github.com/synnaxlabs/synnax/core/pkg/service/device/migrations/v3"

type DeviceV2 struct {
    Key        string                    `json:"key" msgpack:"key"`
    Rack       uint32                    `json:"rack" msgpack:"rack"` // old type
    Location   string                    `json:"location" msgpack:"location"`
    // ... other fields
}

func AutoMigrateV2ToV3(
    ctx context.Context,
    old DeviceV2,
) (v3.DeviceV3, error) {
    return v3.DeviceV3{
        Key:      old.Key,
        Rack:     uint64(old.Rack), // widen uint32 → uint64
        Location: old.Location,
        // ...
    }, nil
}
```

**Observations**:

- Oracle's transitive dependency tracking must detect that `Device` references
  `rack.Key` and flag that a Device migration is needed when `rack.Key` changes.
- The CI check should fail if `rack.Key` changes but Device doesn't have a corresponding
  migration.
- The typed per-entry API handles the actual transform fine.

---

## Scenario 10: Nested Type Change (Add Field to graph.Node)

**Context**: Add a `label` field to `graph.Node`. Node is not a gorp entry — it's
serialized inside `Arc` (and potentially other types). The nesting chain is:

```
Arc (gorp entry, stored in KV)
  └─ graph.Graph
       └─ graph.Nodes ([]graph.Node)
            └─ graph.Node   ← changed here
```

**Leaf auto-migrate** (`schemas/arc/graph/migrations/v1/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v1

import v2 "schemas/arc/graph/migrations/v2"

type NodeV1 struct {
    Key      string     `json:"key" msgpack:"key"`
    Type     string     `json:"type" msgpack:"type"`
    Config   any        `json:"config" msgpack:"config"`
    Position spatial.XY `json:"position" msgpack:"position"`
}

func AutoMigrateV1ToV2(ctx context.Context, old NodeV1) (v2.NodeV2, error) {
    return v2.NodeV2{
        Key:      old.Key,
        Type:     old.Type,
        Config:   old.Config,
        Position: old.Position,
        // Label: zero value (new field, set in PostMigrate)
    }, nil
}
```

**Leaf post-migrate template** (`schemas/arc/graph/migrations/v1/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v1

import v2 "schemas/arc/graph/migrations/v2"

func PostMigrateV1ToV2(ctx context.Context, node *v2.NodeV2, old NodeV1) error {
    node.Label = "" // TODO: set default value for new field
    return nil
}
```

**Parent auto-migrate** (`core/pkg/service/arc/migrations/v5/auto.gen.go`):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v5

import (
    v6 "core/pkg/service/arc/migrations/v6"
    graphv1 "schemas/arc/graph/migrations/v1"
    graphv2 "schemas/arc/graph/migrations/v2"
)

type ArcV5 struct {
    Key    uuid.UUID      `json:"key" msgpack:"key"`
    Name   string         `json:"name" msgpack:"name"`
    Mode   string         `json:"mode" msgpack:"mode"`
    Graph  GraphV5        `json:"graph" msgpack:"graph"`
    Text   string         `json:"text" msgpack:"text"`
    Module string         `json:"module" msgpack:"module"`
    Status string         `json:"status" msgpack:"status"`
}

type GraphV5 struct {
    Viewport  any                  `json:"viewport" msgpack:"viewport"`
    Functions any                  `json:"functions" msgpack:"functions"`
    Edges     any                  `json:"edges" msgpack:"edges"`
    Nodes     []graphv1.NodeV1     `json:"nodes" msgpack:"nodes"`
}

func AutoMigrateV5ToV6(ctx context.Context, old ArcV5) (v6.ArcV6, error) {
    nodes := make([]graphv2.NodeV2, len(old.Graph.Nodes))
    for i, n := range old.Graph.Nodes {
        migrated, err := graphv1.AutoMigrateV1ToV2(ctx, n)
        if err != nil { return v6.ArcV6{}, err }
        if err := graphv1.PostMigrateV1ToV2(ctx, &migrated, n); err != nil {
            return v6.ArcV6{}, err
        }
        nodes[i] = migrated
    }
    return v6.ArcV6{
        Key:    old.Key,
        Name:   old.Name,
        Mode:   old.Mode,
        Graph:  v6.GraphV6{
            Viewport:  old.Graph.Viewport,
            Functions: old.Graph.Functions,
            Edges:     old.Graph.Edges,
            Nodes:     nodes,
        },
        Text:   old.Text,
        Module: old.Module,
        Status: old.Status,
    }, nil
}
```

**Parent post-migrate template** (`core/pkg/service/arc/migrations/v5/migrate.go`):

```go
// Generated by Oracle as a template — edit this file.
package v5

import v6 "core/pkg/service/arc/migrations/v6"

func PostMigrateV5ToV6(ctx context.Context, a *v6.ArcV6, old ArcV5) error {
    // No additional Arc-level changes needed. Edit if custom logic required.
    return nil
}
```

**Observations**:

- The developer touches exactly one file: `v1/migrate.go` (the leaf post-migrate).
  Everything else is generated and never edited.
- Six files total across 4 `vN/` packages: 2 auto-generates + 2 post-migrate templates +
  2 current snapshot packages (leaf and parent). Developer fills in the leaf TODO.
- Oracle's dependency graph detects that `Arc` contains `graph.Node`, bumps `Arc`'s
  version, and generates the propagation chain.
- The parent auto-migrate calls both `graphv1.AutoMigrateV1ToV2` and
  `graphv1.PostMigrateV1ToV2` — the developer's custom logic runs as part of the
  parent's migration, not as a separate pass.

---

## Scenario 11: Shared Nested Type Change (ir.Edge Used by Multiple Entries)

**Context**: `ir.Edge` is defined in `schemas/arc/ir.oracle` and is embedded in both
`Arc` (via `graph.Graph.edges`) and a hypothetical `Schematic` (via
`SchematicData.edges`). Adding a `weight` field to `ir.Edge` triggers migrations on both
parent types.

```
Arc (gorp entry)                    Schematic (gorp entry)
  └─ graph.Graph                      └─ SchematicData
       └─ ir.Edges ([]ir.Edge)             └─ ir.Edges ([]ir.Edge)
            └─ ir.Edge ← changed                └─ ir.Edge ← same change
```

**Leaf migration** (`schemas/arc/ir/migrations/v1/`):

```go
// v1/auto.gen.go — GENERATED, DO NOT EDIT
package v1

import v2 "schemas/arc/ir/migrations/v2"

type EdgeV1 struct {
    Source ir.Handle `json:"source" msgpack:"source"`
    Target ir.Handle `json:"target" msgpack:"target"`
    Kind   *string   `json:"kind" msgpack:"kind"`
}

func AutoMigrateV1ToV2(ctx context.Context, old EdgeV1) (v2.EdgeV2, error) {
    return v2.EdgeV2{
        Source: old.Source,
        Target: old.Target,
        Kind:   old.Kind,
    }, nil
}

// v1/migrate.go — template, developer edits
func PostMigrateV1ToV2(ctx context.Context, edge *v2.EdgeV2, old EdgeV1) error {
    edge.Weight = 1.0 // TODO: set default value for new field
    return nil
}
```

**Oracle auto-generates BOTH parent migrations** — each parent's `auto.gen.go` calls
`irv1.AutoMigrateV1ToV2` + `irv1.PostMigrateV1ToV2` for each edge. Both parent
post-migrate templates are empty (no parent-level changes needed).

Files generated:

```
schemas/arc/ir/migrations/
  ├── v1/
  │   ├── auto.gen.go         # leaf auto-migrate (generated)
  │   └── migrate.go          # leaf post-migrate (developer edits)
  └── v2/
      └── auto.gen.go         # current snapshot (generated)

core/pkg/service/arc/migrations/
  ├── v6/
  │   ├── auto.gen.go         # Arc auto-migrate (generated, calls edge migration)
  │   └── migrate.go          # Arc post-migrate (empty template)
  └── v7/
      └── auto.gen.go         # current snapshot (generated)

core/pkg/service/schematic/migrations/
  ├── v4/
  │   ├── auto.gen.go         # Schematic auto-migrate (generated, calls edge migration)
  │   └── migrate.go          # Schematic post-migrate (empty template)
  └── v5/
      └── auto.gen.go         # current snapshot (generated)
```

**Observations**:

- Developer writes in one file (`v1/migrate.go`). Oracle generates multiple other files.
- Both parent auto-migrations import and call the same leaf functions — transform logic
  is written once, applied everywhere.
- CI catches missing parent migrations: if `ir.Edge` changes but `Schematic`'s migration
  hasn't been regenerated, CI fails.

---

## Scenario 12: Deep Nesting (Change to types.Param Inside Function Inside Arc)

**Context**: Change `types.Param.value` from `any?` to a new `ParamValue` union type.
The nesting chain is 5 levels deep:

```
Arc (gorp entry)
  └─ graph.Graph
       └─ ir.Functions ([]ir.Function)
            └─ ir.Function.config (types.Params)
                 └─ types.Param   ← changed here
```

**What Oracle generates** (6 files):

1. `schemas/arc/types/migrations/v1/auto.gen.go` — Leaf auto-migrate (generated)
2. `schemas/arc/types/migrations/v1/migrate.go` — Leaf post-migrate (**developer edits**)
3. `schemas/arc/ir/migrations/v1/auto.gen.go` — Intermediate auto-migrate (generated,
   calls param leaf migration for each param in config/inputs/outputs)
4. `schemas/arc/ir/migrations/v1/migrate.go` — Intermediate post-migrate (empty
   template)
5. `core/pkg/service/arc/migrations/vN/auto.gen.go` — Top-level auto-migrate (generated,
   calls function migration for each function)
6. `core/pkg/service/arc/migrations/vN/migrate.go` — Top-level post-migrate (empty
   template)

The propagation is **transitive**: Param change → Function migration → Arc migration.
Each intermediate level gets its own auto-migrate + post-migrate pair so that if
`Function` appears in other gorp entries, those entries also get auto-generated parent
migrations.

**Observations**:

- Developer edits exactly one file (`v1/migrate.go`) regardless of nesting depth.
- Oracle generates multiple files automatically. The intermediate and top-level post-migrate
  templates are empty — the developer only touches them if they need custom logic at
  those levels.
- Same two-function pattern applied recursively through the tree.

# 5 - Scenario Analysis

| Scenario               | Typed       | Raw Tx    | Mode        | Dev Edits                      |
| ---------------------- | ----------- | --------- | ----------- | ------------------------------ |
| 1. Codec transition    | Yes         | No        | Full        | Nothing (both files generated) |
| 2. Add field           | Yes         | No        | Skeleton    | Post-migrate only              |
| 3. Remove field        | Yes         | No        | Skeleton    | Post-migrate only              |
| 4. Rename field        | Yes         | No        | Skeleton    | Post-migrate only              |
| 5. Strongly type JSON  | Yes         | No        | Skeleton    | Post-migrate only              |
| 6. Change field type   | **Partial** | **Maybe** | Skeleton    | Post-migrate only              |
| 7. Ranger groups       | No          | **Yes**   | Raw         | Full raw migration             |
| 8. RBAC permissions    | No          | **Yes**   | Raw         | Full raw migration             |
| 9. Transitive dep      | Yes         | No        | Propagation | Leaf post-migrate only         |
| 10. Nested type change | Yes         | No        | Propagation | Leaf post-migrate only         |
| 11. Shared nested type | Yes         | No        | Propagation | Leaf post-migrate only         |
| 12. Deep nesting       | Yes         | No        | Propagation | Leaf post-migrate only         |

## Key Findings

**The typed per-entry API covers 9 of 12 scenarios cleanly.** Scenarios 7 and 8 require
the raw transaction escape hatch. Scenario 6 is borderline — it works if you accept
incomplete data or enrich the API.

**Every migration version is a `vN/` sub-package with two files**:

- **`vN/auto.gen.go`**: Auto-migrate + type snapshot. Generated, never touched. Copies
  1:1 fields, calls nested migrations. Oracle can regenerate at any time.
- **`vN/migrate.go`**: Post-migrate template. Generated once, developer owns. Called
  after auto-migrate. Developer fills in TODOs for new/changed/removed fields.

**Oracle has three generation modes**:

- **Full mode**: Mechanical migrations (codec transitions). Both auto-migrate and
  post-migrate are generated. Developer touches neither file.
- **Skeleton mode**: Direct schema changes (field add/remove/rename/type change).
  Auto-migrate copies 1:1 fields. Post-migrate template has TODOs for changed fields.
  Developer edits the post-migrate.
- **Propagation mode**: A nested dependency changed. Parent auto-migrate walks the tree
  and calls nested auto-migrate + post-migrate. Parent post-migrate template is empty.
  Developer only edits the leaf post-migrate.

**Nested type changes propagate automatically.** When a nested type changes, Oracle
bumps versions and generates auto-migrate + post-migrate pairs for every gorp entry that
transitively contains it. The developer edits exactly one post-migrate file — on the
type that actually changed. This scales to arbitrary nesting depth (Scenario 12) and
shared types (Scenario 11).

**Cross-type migrations get enriched dependency injection.** Scenarios 7 and 8 need
access to services (ontology, group, user, role). The raw transaction escape hatch will
be enriched to inject service dependencies beyond `gorp.Tx`.

**Unknown JSON keys during strong-typing (Scenario 5): drop silently.** When promoting
`json` to a struct, unknown keys are dropped. The migration only copies fields that map
to the new struct. If specific data needs preserving, the developer handles it in the
post-migrate.

# 6 - Mechanical Analysis

This section traces through the real gorp code to understand what the migration runner
must do at the KV level. The scenarios above showed the _what_ (transform functions);
this section shows the _how_ (the framework machinery that calls them).

## 1 - The Key Prefix Problem

Gorp's `Reader[K, E]` and `Writer[K, E]` derive their KV prefix from the Go type name
via `types.Name[E]()`. For example:

- `Reader[uuid.UUID, Schematic]` → prefix `__gorp__//Schematic`
- `Reader[uuid.UUID, SchematicV1]` → prefix `__gorp__//SchematicV1`

**Problem**: Data stored as `Schematic` lives under prefix `__gorp__//Schematic`. If the
migration runner tries to read it using `Reader[uuid.UUID, SchematicV1]`, it gets prefix
`__gorp__//SchematicV1` — no entries found.

**Implication**: The migration runner **cannot use gorp's typed Reader/Writer** with
legacy types directly. It must operate at a lower level.

## 2 - How EntryManager Already Solves This

`manager.go` already has a migration that reads and rewrites entries. Here's what
`reEncodeKeys` does:

```go
func reEncodeKeys[K Key, E Entry[K]](ctx context.Context, tx Tx) error {
    iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{})  // uses current type's prefix
    // ...
    writer := WrapWriter[K, E](tx)
    for iter.First(); iter.Valid(); iter.Next() {
        writer.BaseWriter.Delete(ctx, iter.Key())     // delete old KV entry
        writer.Set(ctx, *iter.Value(ctx))              // write new KV entry
    }
}
```

This works because it reads and writes the **same type** — it's re-encoding keys, not
changing the type shape. The codec (msgpack) decodes into `E` and re-encodes `E`.

## 3 - What the Migration Runner Must Do

For a schema migration (e.g., `SchematicV1` → `Schematic`), the runner needs to:

1. **Iterate raw KV entries** under the canonical prefix (`__gorp__//Schematic`)
2. **Decode raw bytes** using the old codec into the old type (`SchematicV1`)
3. **Call the transform** function: `old SchematicV1 → new Schematic`
4. **Encode the new type** using the current codec
5. **Write back** under the same KV key

Step 2 is the critical one. The runner can't use `WrapReader[K, SchematicV1]` (wrong
prefix). Instead, it must:

```go
// Open a raw KV iterator with the canonical prefix
prefix := []byte("__gorp__//Schematic")
rawIter, _ := tx.OpenIterator(kv.IterPrefix(prefix))

// For each entry:
for rawIter.First(); rawIter.Valid(); rawIter.Next() {
    rawKey := rawIter.Key()
    rawValue := rawIter.Value()

    // Decode raw bytes into old type using the old codec
    var old SchematicV1
    oldCodec.Decode(ctx, rawValue, &old)

    // Call auto-migrate (generated, copies 1:1 fields)
    new, _ := autoMigrateFn(ctx, old)

    // Call post-migrate (developer-written, handles new/changed fields)
    postMigrateFn(ctx, &new, old)

    // Encode new type using the current codec
    newValue, _ := newCodec.Encode(ctx, new)

    // Delete old entry and write new one (key stays the same)
    tx.Delete(ctx, rawKey)
    tx.Set(ctx, rawKey, newValue)
}
```

## 4 - Multi-Step Chaining

When a customer jumps from v1 to v3, the runner executes each migration as a separate
full pass over all entries (consistent with Section 3.3.2):

```
Pass 1: Read all entries → decode as v1.V1 → transform to v2.V2 → encode → write
Pass 2: Read all entries → decode as v2.V2 → transform to v3.V3 → encode → write
```

Each pass reads, transforms, and writes back all entries. This is simple but involves
multiple full passes. An in-memory chaining optimization (decode once, apply all
transforms, encode once) is not feasible because intermediate types exist only at compile
time — the runner can't dynamically chain `func(V1) V2` and `func(V2) V3` without
generics gymnastics, and raw migrations need the KV state from the previous step.
Multiple full passes is correct and fast enough for metadata volumes.

## 5 - Codec Transition Specifics

For the msgpack → protobuf MVP, there is no generic "protobuf codec." Each type handles
its own protobuf serialization via `GorpMarshaler`/`GorpUnmarshaler` (Section 3.4.4).
The DB's generic codec stays msgpack — it's the fallback for pre-transition legacy types
that don't implement the marshaling interfaces.

The transform function is identity (same fields), so the chain is:

```
Raw msgpack bytes → tx.Decode into v1.SchematicV1 (no GorpUnmarshaler, uses msgpack)
    → identity transform → v2.SchematicV2
    → v2.SchematicV2.GorpMarshal() → protobuf bytes → write
```

The canonical type prefix (`__gorp__//Schematic`) doesn't change — only the value
encoding does.

## 6 - The Canonical Prefix Registry

The migration runner needs to know the canonical KV prefix for each type — the prefix
under which data is actually stored. This can't be derived from the legacy type name
(e.g., `v1.SchematicV1` would give `__gorp__//SchematicV1`, not `__gorp__//Schematic`).

The solution is **`EntryManager[K, E]`'s generic type parameter**. Since `EntryManager`
is generic on the current type `E`, it derives the canonical prefix via
`types.Name[E]()` — the same mechanism gorp already uses for `Reader` and `Writer`.
The `migrations/migrate.gen.go` file exports an `All()` function returning
`[]gorp.Migration` — it only imports `vN/` sub-packages, never the parent service
package, avoiding circular dependencies. The prefix and version key are derived by
`EntryManager`, not by the migration files.

# 7 - Go API Surface

Based on the mechanical analysis, here are the concrete interfaces and types. All
runtime migration types live in `gorp`, not in an `oracle` package. Oracle is a
build-time code generation tool; gorp owns runtime execution.

## 1 - Core Types

```go
package gorp

import "context"

// Migration is the interface for a single versioned migration step. All
// migrations — typed, raw, and key re-encoding — implement this interface.
// EntryManager runs them sequentially by version counter.
type Migration interface {
    // Name returns a human-readable identifier for this migration (e.g.,
    // "msgpack_to_protobuf", "add_description_field").
    Name() string

    // Run executes the migration within the given transaction. The
    // MigrationConfig provides the canonical KV prefix and fallback codec.
    Run(ctx context.Context, tx kv.Tx, cfg MigrationConfig) error
}

// MigrationConfig is passed to each Migration.Run call by EntryManager.
type MigrationConfig struct {
    // Prefix is the canonical KV prefix for this type (e.g.,
    // "__gorp__//Schematic"). Derived from types.Name[E]() by EntryManager.
    Prefix []byte
    // Codec is the DB's generic codec (msgpack). Used as fallback for types
    // that don't implement GorpMarshaler/GorpUnmarshaler.
    Codec binary.Codec
}

// AutoMigrateFunc is the generated per-entry transform. It copies all 1:1
// fields from the old type to the new type and calls nested auto-migrations.
// Generated by Oracle, never edited by the developer.
//
// I is the old (input) type, O is the new (output) type.
type AutoMigrateFunc[I, O any] func(ctx context.Context, old I) (O, error)

// PostMigrateFunc is the developer-written hook called after AutoMigrateFunc
// on each entry. Receives a pointer to the new entry (already populated by
// AutoMigrateFunc) and the old entry for reference. The developer sets default
// values for new fields, transforms data, etc.
type PostMigrateFunc[I, O any] func(ctx context.Context, new *O, old I) error
```

## 2 - Migration Implementations

### TypedMigration

```go
// TypedMigration is a per-entry transform migration. It iterates all entries
// under the canonical KV prefix, decodes each as type I, calls auto-migrate
// + post-migrate to produce type O, encodes, and writes back.
type TypedMigration[I, O any] struct {
    name        string
    autoMigrate AutoMigrateFunc[I, O]
    postMigrate PostMigrateFunc[I, O]
}

func NewTypedMigration[I, O any](
    name string,
    autoMigrate AutoMigrateFunc[I, O],
    postMigrate PostMigrateFunc[I, O],
) Migration {
    return &TypedMigration[I, O]{
        name:        name,
        autoMigrate: autoMigrate,
        postMigrate: postMigrate,
    }
}

func (m *TypedMigration[I, O]) Name() string { return m.name }

func (m *TypedMigration[I, O]) Run(
    ctx context.Context,
    tx kv.Tx,
    cfg MigrationConfig,
) error {
    iter, err := tx.OpenIterator(kv.IterPrefix(cfg.Prefix))
    if err != nil {
        return err
    }
    defer func() { err = errors.Combine(err, iter.Close()) }()

    for iter.First(); iter.Valid(); iter.Next() {
        // Decode: use GorpUnmarshaler if I implements it,
        // otherwise fall back to the generic codec (msgpack).
        var old I
        if u, ok := any(&old).(GorpUnmarshaler); ok {
            if err := u.GorpUnmarshal(ctx, iter.Value()); err != nil {
                return errors.Wrapf(err, "migration %s: decode failed", m.name)
            }
        } else {
            if err := cfg.Codec.Decode(ctx, iter.Value(), &old); err != nil {
                return errors.Wrapf(err, "migration %s: decode failed", m.name)
            }
        }

        // Auto-migrate: copy 1:1 fields, call nested migrations.
        migrated, err := m.autoMigrate(ctx, old)
        if err != nil {
            return errors.Wrapf(err, "migration %s: auto-migrate failed", m.name)
        }

        // Post-migrate: developer-written logic for new/changed fields.
        if err := m.postMigrate(ctx, &migrated, old); err != nil {
            return errors.Wrapf(err, "migration %s: post-migrate failed", m.name)
        }

        // Encode: use GorpMarshaler if O implements it,
        // otherwise fall back to the generic codec.
        var encoded []byte
        if mar, ok := any(&migrated).(GorpMarshaler); ok {
            encoded, err = mar.GorpMarshal(ctx)
        } else {
            encoded, err = cfg.Codec.Encode(ctx, migrated)
        }
        if err != nil {
            return errors.Wrapf(err, "migration %s: encode failed", m.name)
        }

        if err := tx.Set(ctx, iter.Key(), encoded); err != nil {
            return err
        }
    }
    return iter.Error()
}
```

### RawMigration

```go
// RawMigration is the escape hatch for cross-type or complex migrations that
// can't be expressed as a per-entry transform. Used for migrations like ranger
// group restructuring or RBAC permission migrations that operate across
// multiple types.
type RawMigration struct {
    name    string
    migrate func(ctx context.Context, tx Tx) error
}

func NewRawMigration(
    name string,
    fn func(ctx context.Context, tx Tx) error,
) Migration {
    return &RawMigration{name: name, migrate: fn}
}

func (m *RawMigration) Name() string { return m.name }

func (m *RawMigration) Run(
    ctx context.Context,
    tx kv.Tx,
    cfg MigrationConfig,
) error {
    return m.migrate(ctx, WrapTx(tx, cfg.Codec))
}
```

### KeyMigration

```go
// KeyMigration re-encodes all keys for a type when the key encoding scheme
// changes. This is the existing reEncodeKeys logic expressed as a Migration.
// EntryManager implicitly appends this as the final migration — services
// don't need to register it.
type KeyMigration[K Key, E Entry[K]] struct{}

func (m *KeyMigration[K, E]) Name() string { return "re_encode_keys" }

func (m *KeyMigration[K, E]) Run(
    ctx context.Context,
    tx kv.Tx,
    cfg MigrationConfig,
) error {
    gorpTx := WrapTx(tx, cfg.Codec)
    return reEncodeKeys[K, E](ctx, gorpTx)
}
```

## 3 - EntryManager as Migration Runner

```go
// EntryManager manages the full migration lifecycle for a gorp-stored type.
// It accepts an ordered list of Migration implementations and runs them
// sequentially, tracking progress via a version counter in the KV store.
// Key re-encoding is implicitly appended as the final migration.
type EntryManager[K Key, E Entry[K]] struct{}

// OpenEntryManager creates an EntryManager, runs all pending migrations, and
// returns. The version counter is stored at "__gorp__//<Type>/version" as a
// uint16 (big-endian). Key re-encoding is always run as the final step.
func OpenEntryManager[K Key, E Entry[K]](
    ctx context.Context,
    db *DB,
    migrations ...Migration,
) (*EntryManager[K, E], error) {
    // Derive canonical prefix and version key from the current type E.
    prefix := []byte(magicPrefix + types.Name[E]())
    versionKey := []byte(magicPrefix + types.Name[E]() + "/version")

    // Append key re-encoding as the implicit final migration.
    allMigrations := append(migrations, &KeyMigration[K, E]{})

    if err := runMigrations(ctx, db, prefix, versionKey, allMigrations); err != nil {
        return nil, err
    }
    return &EntryManager[K, E]{}, nil
}

func runMigrations(
    ctx context.Context,
    db *DB,
    prefix, versionKey []byte,
    migrations []Migration,
) error {
    return db.WithTx(ctx, func(tx Tx) error {
        currentVersion, err := readVersion(ctx, tx, versionKey)
        if err != nil {
            return err
        }

        if int(currentVersion) >= len(migrations) {
            return nil // already up to date
        }

        cfg := MigrationConfig{
            Prefix: prefix,
            Codec:  db.Codec,
        }

        for i := currentVersion; i < uint16(len(migrations)); i++ {
            migration := migrations[i]
            if err := migration.Run(ctx, tx.KVTx(), cfg); err != nil {
                return errors.Wrapf(err, "migration %d (%s) failed",
                    i+1, migration.Name())
            }
            if err := writeVersion(ctx, tx, versionKey, i+1); err != nil {
                return err
            }
        }
        return nil
    })
}

func readVersion(ctx context.Context, tx Tx, key []byte) (uint16, error) {
    b, closer, err := tx.Get(ctx, key)
    if err := errors.Skip(err, query.ErrNotFound); err != nil {
        return 0, err
    }
    if closer != nil {
        defer closer.Close()
    }
    if len(b) < 2 {
        return 0, nil
    }
    return binary.BigEndian.Uint16(b), nil
}

func writeVersion(ctx context.Context, tx Tx, key []byte, v uint16) error {
    b := make([]byte, 2)
    binary.BigEndian.PutUint16(b, v)
    return tx.Set(ctx, key, b)
}
```

## 4 - Service Wiring

```go
// GENERATED BY ORACLE — DO NOT EDIT
// core/pkg/service/schematic/migrations/migrate.gen.go
package migrations

import (
    "github.com/synnaxlabs/x/gorp"
    v1 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v1"
    v2 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v2"
)

// All returns the ordered migrations for the Schematic type. Oracle generates
// this function and appends new registrations on each
// `oracle migrate generate` run. The developer never edits this file.
//
// Note: this file only imports vN/ sub-packages — it never imports the parent
// schematic package, so there is no circular dependency.
func All() []gorp.Migration {
    return []gorp.Migration{
        gorp.NewTypedMigration(
            "msgpack_to_protobuf",
            v1.AutoMigrateV1ToV2,
            v1.PostMigrateV1ToV2,
        ),
        gorp.NewTypedMigration(
            "add_description",
            v2.AutoMigrateV2ToV3,
            v2.PostMigrateV2ToV3,
        ),
    }
}
```

```go
// core/pkg/service/schematic/service.go
func OpenService(ctx context.Context, cfg ServiceConfig) (*Service, error) {
    // OpenEntryManager runs all pending migrations (schema migrations +
    // key re-encoding) and returns the ready EntryManager.
    entryManager, err := gorp.OpenEntryManager[uuid.UUID, Schematic](
        ctx, cfg.DB, migrations.All()...,
    )
    if err != nil {
        return nil, err
    }

    s := &Service{ServiceConfig: cfg, entryManager: entryManager}
    // ... rest of service init ...
    return s, nil
}
```

## 5 - Migration File Layout

Migration files always live next to their source Go type — at the `@go output` path
with `/migrations/vN/` appended. Each `vN/` is a separate Go package.

```
core/pkg/service/schematic/
├── types.gen.go              # Current Schematic type (Oracle-generated)
├── marshal.gen.go            # GorpMarshal/GorpUnmarshal (Oracle sync)
├── service.go                # Service init, calls OpenEntryManager
├── pb/                       # Current protobuf definitions
└── migrations/
    ├── migrate.gen.go        # GENERATED: All() — ordered migration list
    ├── v1/                   # Pre-transition snapshot (msgpack era)
    │   ├── auto.gen.go       # SchematicV1 + AutoMigrateV1ToV2
    │   ├── migrate.go        # PostMigrateV1ToV2 (developer edits)
    │   └── migrate_test.go   # Tests for this migration
    ├── v2/                   # Post-transition snapshot
    │   ├── auto.gen.go       # SchematicV2 + GorpMarshal/Unmarshal + AutoMigrateV2ToV3
    │   ├── migrate.go        # PostMigrateV2ToV3 (developer edits)
    │   └── pb/               # Snapshotted protobuf for V2
    └── v3/                   # Current snapshot (no AutoMigrate yet)
        ├── auto.gen.go       # SchematicV3 + GorpMarshal/Unmarshal
        └── pb/               # Snapshotted protobuf for V3
```

**v1/auto.gen.go** (Oracle-generated, never touched):

```go
// GENERATED BY ORACLE — DO NOT EDIT
package v1

import (
    "context"

    "github.com/google/uuid"
    "github.com/synnaxlabs/x/binary"
    v2 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v2"
)

type SchematicV1 struct {
    Key      uuid.UUID                 `json:"key" msgpack:"key"`
    Name     string                    `json:"name" msgpack:"name"`
    Data     binary.MsgpackEncodedJSON `json:"data" msgpack:"data"`
    Snapshot bool                      `json:"snapshot" msgpack:"snapshot"`
}

func AutoMigrateV1ToV2(
    ctx context.Context,
    old SchematicV1,
) (v2.SchematicV2, error) {
    return v2.SchematicV2{
        Key:      old.Key,
        Name:     old.Name,
        Data:     old.Data,
        Snapshot: old.Snapshot,
    }, nil
}
```

**v1/migrate.go** (Oracle-generated template, developer edits):

```go
// Generated by Oracle as a template — edit this file.
package v1

import (
    "context"

    v2 "github.com/synnaxlabs/synnax/core/pkg/service/schematic/migrations/v2"
)

func PostMigrateV1ToV2(
    ctx context.Context,
    s *v2.SchematicV2,
    old SchematicV1,
) error {
    // Codec transition only — no field changes. Nothing to do.
    return nil
}
```

## 6 - Multi-Step Chaining Design

When multiple migrations are pending (e.g., v1→v2→v3), the current design runs each
migration as a full pass over all entries. This means:

- **v1→v2**: Iterate all entries, decode as V1, transform to V2, encode, write.
- **v2→v3**: Iterate all entries again, decode as V2, transform to V3, encode, write.

This is simple but involves multiple full passes. An optimization would be to chain
transforms in memory (decode once, apply all transforms, encode once, write once).
However, this complicates the runner because:

1. Intermediate types exist only at compile time — the runner can't dynamically chain
   `func(V1) V2` and `func(V2) V3` without generics gymnastics.
2. Raw migrations can't be chained — they need the KV state from the previous step.

Multiple full passes is the design for now. Simple, correct, and fast enough for
metadata volumes (thousands of entries, not millions). Optimize later if startup time
becomes a concern.

# 8 - Architecture Overview

```
Developer Workflow (direct schema change):
  1. Modify .oracle schema file (e.g., add field to Schematic)
  2. Run `oracle migrate generate`
     → Oracle generates vN_auto.gen.go (legacy type + auto-migrate, skeleton mode)
     → Oracle generates vN.go (post-migrate template with TODOs)
     → Oracle regenerates migrate.gen.go (appends new registration)
     → Oracle creates new schema snapshot for CI
  3. Developer edits vN.go — fills in TODOs for new/changed/removed fields
  4. Run tests using built-in migration test helpers
  5. Commit — CI runs `oracle migrate check` to validate

Developer Workflow (nested type change):
  1. Modify .oracle schema file (e.g., add field to graph.Node)
  2. Run `oracle migrate generate`
     → Oracle generates leaf auto-migrate + post-migrate template (skeleton mode)
     → Oracle detects all parent gorp entries that contain graph.Node
     → Oracle bumps parent versions, generates parent auto-migrate + post-migrate
       (propagation mode, calls leaf migration for each nested instance)
     → Oracle regenerates all affected migrate.gen.go files
  3. Developer edits ONLY the leaf post-migrate (e.g., node_v1.go)
  4. Parent post-migrate templates are empty — edit only if custom logic needed
  5. Commit — CI runs `oracle migrate check` to validate all affected types

Server Startup (per service):
  1. gorp.OpenEntryManager[K, E](ctx, db, migrations.All()...)
     a. Derives prefix from types.Name[E]() → "__gorp__//<Type>"
     b. Reads __gorp__//<Type>/version from KV store
     c. For each pending migration (currentVersion+1 → latest):
        i.   Call Migration.Run(ctx, tx, cfg)
             - TypedMigration: iterate entries, decode, transform, encode, write
             - RawMigration: arbitrary gorp.Tx logic
        ii.  Increment version counter
     d. Implicitly runs KeyMigration as final step (re-encode keys)
  2. Service begins accepting requests

Directory Layout (nested migrations — files always at @go output path):
  <graph @go output>/
  └── migrations/
      └── v1/
          ├── auto.gen.go      # 🔒 GENERATED: NodeV1 type + AutoMigrate
          └── migrate.go       # ✏️  Template: PostMigrateV1ToV2 (developer edits)

  <ir @go output>/
  └── migrations/
      └── v1/
          ├── auto.gen.go      # 🔒 GENERATED: EdgeV1 type + AutoMigrate
          └── migrate.go       # ✏️  Template: PostMigrateV1ToV2 (developer edits)

  core/pkg/service/arc/
  ├── types.gen.go             # Current Arc type (Oracle-generated)
  ├── service.go               # Service logic
  └── migrations/
      ├── migrate.gen.go       # 🔒 GENERATED: All() registration
      ├── v1/                  # 🔒 GENERATED: Codec transition (full mode)
      │   ├── auto.gen.go
      │   └── migrate.go
      ├── v5/                  # 🔒 GENERATED: Node propagation
      │   ├── auto.gen.go      # Calls node leaf migration
      │   └── migrate.go       # Empty template (propagation mode)
      └── v6/                  # 🔒 GENERATED: Edge propagation
          ├── auto.gen.go      # Calls edge leaf migration
          └── migrate.go       # Empty template (propagation mode)

  core/pkg/service/schematic/
  ├── types.gen.go             # Current Schematic type (Oracle-generated)
  └── migrations/
      ├── migrate.gen.go       # 🔒 GENERATED: All() registration
      ├── v1/                  # 🔒 GENERATED: Codec transition (full mode)
      │   ├── auto.gen.go
      │   └── migrate.go
      └── v4/                  # 🔒 GENERATED: Edge propagation
          ├── auto.gen.go      # Calls edge leaf migration
          └── migrate.go       # Empty template (propagation mode)
```

# 9 - Open Questions

## 1 - Codec Transition Approach

Oracle-generated per-type migration files. 15 individual migration files is fine as long
as Oracle generates them fully — the critical principle is minimizing user-written code.
For purely mechanical migrations (codec transitions, trivial field copies), Oracle
generates the complete transform function, not just a skeleton. Codec transitions are
deterministic and can be fully auto-generated.

## 2 - Dependency Injection for Raw Transaction Migrations

Raw transaction migrations access service dependencies via either:

1. **Closures** that capture dependencies from the service config at registration time.
2. **Methods on the service struct** (like the current ranger migration pattern), where
   the service registers `s.migrateRangeGroups` directly and accesses `s.cfg.*`.

Both are simple and explicit with no framework magic. The choice between them is a
per-migration decision — closures work well for Oracle-generated registration in
`migrations/migrate.go`, while service methods work well for migrations that are tightly
coupled to the service's internal logic. No generic dependency container needed.

## 3 - Nested Migration Execution Ordering

When multiple nested types change simultaneously (e.g., both `Node` and `Edge` change in
the same release), one `oracle migrate generate` invocation produces **one** version
bump per affected gorp entry. The generated parent migration calls all leaf migrations
in a single transform function. No composition complexity — Oracle captures everything
that changed since the last snapshot in a single migration step.

If nested types change in separate releases (separate `oracle migrate generate` runs),
they become separate sequential migrations as expected.

## 4 - Snapshot Wire-Format Safety for Unchanged Nested Types

Parent snapshots use current types for unchanged nested types (e.g., `arc.Text` in
`ArcV5`). This is always safe because:

1. **If the nested type changed**, it wouldn't be "unchanged" — Oracle would version it
   in the snapshot.
2. **If it didn't change**, the current type's definition exactly matches the stored
   data's encoding, so it decodes correctly.
3. **Ordering within the same service**: If `Arc` has both a `Text` migration (v4) and a
   `Node` migration (v5) pending, v4 runs first. By the time v5's snapshot references
   `arc.Text`, v4 has already migrated `Text` to its current form. Oracle controls
   version ordering and guarantees this sequencing.

# 10 - Roadmap

1. **Phase 1 (This RFC)**: Oracle migration framework + msgpack→protobuf codec switch
2. **Phase 2**: Incrementally strongly type JSON fields (schematic data, workspace
   layout) using the migration system
3. **Phase 3**: RFC 0025 — server owns all schema evolution, replacing client-side
   migrations
