# 28 - Oracle Codecs and Migrations

**Feature Name**: Oracle Codecs and Migrations <br /> **Status**: Implemented <br />
**Start Date**: 2025-12-29 <br /> **Authors**: Emiliano Bonilla <br />

**Related:** [RFC 0025 - Meta Data Structures](./0026-251214-meta-data.md)

---

# 0 - Summary

In this RFC, we describe two extensions to the Oracle schema system that address the
serialization performance and schema evolution problems identified in RFC 0025. The
first is a binary codec system called ORC that replaces msgpack for metadata
serialization, eliminating reflection overhead and enabling zero-allocation encoding.
The second is a migration framework that uses Oracle's schema snapshots to detect
structural changes, generate frozen type versions, and produce migration scaffolding
that transforms existing database entries at startup.

Together, these extensions close the loop on Oracle's role in the metadata lifecycle:
the schema defines the structure, the `go/marshal` plugin generates the codec, the
`go/migrate` plugin generates the migration, and `gorp.OpenTable` executes it.

---

# 1 - Vocabulary

- **ORC** - A bespoke binary encoding format used for metadata serialization.
  Positional, untagged, big-endian. Named after Oracle. Not related to the Apache ORC
  columnar format.
- **Magic header** - The 3-byte prefix `[0x4F, 0x52, 0x43]` ("ORC" in ASCII) written at
  the start of every ORC-encoded payload. Used for format detection without trial
  decoding.
- **Fallback codec** - A codec wrapper that attempts ORC decoding first, then falls back
  to a secondary codec (typically msgpack) when the magic header is absent. Enables
  transparent migration of existing data.
- **Snapshot** - A frozen copy of all `.oracle` schema files at a specific version,
  stored in `schemas/.snapshots/vN/`. Used as a baseline for schema diff detection.
- **Frozen type** - A generated Go struct representing a previous version of a schema
  type, placed in a `migrations/vN/` sub-package. Includes its own ORC codec so it can
  decode data written by the old schema.
- **Auto-copy** - Generated code that copies fields with matching names and types
  between a frozen type and the current type, reducing boilerplate in migration
  transform functions.
- **Schema change** - A structural difference between the current schema and the latest
  snapshot, detected by deep comparison of the resolution tables.
- **Entry migration** - A `gorp.Migration` that iterates over all entries of one type,
  decodes each with the old codec, transforms it, and re-encodes with the current codec.

---

# 2 - Motivation

## 2.0 - Serialization Performance

Gorp uses `msgpack` for encoding metadata entries to Pebble. The msgpack codec relies on
Go reflection to walk struct fields at runtime, which has two costs. First, every encode
and decode allocates intermediate `reflect.Value` objects on the heap. Second, the
reflection-based field traversal is opaque to the compiler and cannot be inlined or
optimized. For hot paths like channel lookups (which decode hundreds of entries per
query), this overhead is measurable.

The serialization format itself is also suboptimal for our use case. Msgpack is
self-describing: every encoded value carries type tags and field name strings. This is
useful for schema-less data, but Gorp always knows the concrete Go type at both encode
and decode time. The type tags and field names are redundant bytes on the wire and
redundant branches in the decoder.

## 2.1 - No Migration Path for Schema Changes

When a metadata structure changes between Synnax versions (a field is added, removed, or
its type changes), existing deployments have data encoded under the old schema. Before
this work, there was no systematic way to handle this. Some services used ad-hoc
migration code, others relied on msgpack's tolerance for missing fields (which silently
zero-fills), and others simply broke on upgrade.

The problem is compounded by the fact that migrations need the old type definition to
decode existing data, but Go only has the current type definition in scope. Manually
maintaining old type copies is error-prone and scales poorly as schemas evolve across
releases.

## 2.2 - Client-Side Migrations

Visualization configurations (schematics, line plots, tables) are stored on the server
but migrated in the Console. This means the Console must bundle migration logic for
every schema version it might encounter, and older SDK versions cannot read data written
by newer ones. Moving migrations to the server, where they run once at startup,
eliminates this class of incompatibility.

---

# 3 - ORC Binary Codec

## 3.0 - Wire Format

ORC is a positional, untagged binary format. Fields are written in declaration order
with no field names or type tags. Multi-byte integers and floats use big-endian byte
order.

| Go Type         | Encoding                                          |
| --------------- | ------------------------------------------------- |
| `uint8`         | 1 byte                                            |
| `uint16`        | 2 bytes big-endian                                |
| `uint32`        | 4 bytes big-endian                                |
| `uint64`        | 8 bytes big-endian                                |
| `int8/16/32/64` | Same width as unsigned, cast on read              |
| `float32`       | `math.Float32bits` to 4 bytes big-endian          |
| `float64`       | `math.Float64bits` to 8 bytes big-endian          |
| `bool`          | 1 byte: `0x01` = true, `0x00` = false             |
| `string`        | 4-byte big-endian length prefix + raw UTF-8 bytes |
| `[]byte`        | 4-byte big-endian length prefix + raw bytes       |
| `uuid.UUID`     | 16 bytes raw (via `Write(key[:])`)                |
| Nested struct   | Recursive: fields written inline, no framing      |

Every encoded payload is prefixed with the 3-byte magic header `[0x4F, 0x52, 0x43]`.
This prefix does not conflict with msgpack (leading bytes `0x80`-`0xdf`, `0xc0`-`0xd3`)
or JSON (leading bytes `0x22`-`0x7b`), enabling reliable format detection.

## 3.1 - Codec Interfaces

The ORC codec is built on two interfaces that types implement directly
(`x/go/encoding/orc/codec.go`):

```go
type SelfEncoder interface {
    EncodeOrc(w *Writer) error
}

type SelfDecoder interface {
    DecodeOrc(r *Reader) error
}
```

Types that implement both satisfy `SelfCodec`. The generated code calls `Writer` and
`Reader` methods directly, with no reflection, no allocations beyond the pooled
writer/reader buffers, and no branching on field types.

## 3.2 - Reader Safety Guards

The `Reader` includes guards against corrupt or adversarial payloads:

- **String length validation** - `String()` checks the 4-byte length prefix against
  `MaxStringLen` (default 128 MB) before allocating.
- **Collection length validation** - `CollectionLen()` checks against `MaxCollectionLen`
  (default 10,000,000) before allowing slice or map allocations.
- **Recursion depth tracking** - `PushDepth(limit)` / `PopDepth()` prevent stack
  overflow from deeply nested or circular type graphs.

## 3.3 - Singleton and Pooling

`orc.Codec` is a package-level singleton implementing `encoding.Codec`. It uses
`sync.Pool` for both `Writer` and `Reader` to avoid allocation on every encode/decode
cycle. The flow is:

1. **Encode**: Pull `*Writer` from pool, reset, write magic header, call
   `value.EncodeOrc(w)`, copy bytes out, return writer to pool.
2. **Decode**: Check magic header on first 3 bytes, pull `*Reader` from pool,
   `ResetBytes(data[3:])`, call `value.DecodeOrc(r)`, return reader to pool.

## 3.4 - Fallback Codec

`orc.NewCodec(fallback encoding.Codec)` returns a codec that bridges old and new
encodings:

- **On encode**: If the value implements `SelfEncoder`, encode as ORC. Otherwise,
  delegate to the fallback codec.
- **On decode**: If the data starts with the ORC magic header, decode as ORC. Otherwise,
  delegate to the fallback codec.

This enables transparent, zero-downtime migration of existing data. When a Gorp DB is
opened with `orc.NewCodec(msgpack.Codec)`, all existing msgpack-encoded entries decode
correctly. As entries are re-written (through normal operations or explicit migrations),
they are re-encoded as ORC. Over time, all data migrates to the new format without a
bulk rewrite.

## 3.5 - Generated Codec Methods

The Oracle `go/marshal` plugin generates `EncodeOrc` and `DecodeOrc` methods for every
struct annotated with `@go marshal`. The generated code is placed in `codec.gen.go`
alongside the type definition.

For example, given the ranger schema:

```oracle
Range struct {
    key        Key
    name       string
    time_range telem.TimeRange
    color      color.Color?

    @go marshal
}
```

Oracle generates (`core/pkg/service/ranger/codec.gen.go`):

```go
func (rv Range) EncodeOrc(w *orc.Writer) error {
    w.Write(rv.Key[:])
    w.String(rv.Name)
    if err := rv.TimeRange.EncodeOrc(w); err != nil {
        return err
    }
    if err := rv.Color.EncodeOrc(w); err != nil {
        return err
    }
    return nil
}

func (rv *Range) DecodeOrc(r *orc.Reader) error {
    var err error
    if _, err := r.Read(rv.Key[:]); err != nil {
        return err
    }
    if rv.Name, err = r.String(); err != nil {
        return err
    }
    if err = rv.TimeRange.DecodeOrc(r); err != nil {
        return err
    }
    if err = rv.Color.DecodeOrc(r); err != nil {
        return err
    }
    return nil
}
```

Fields are written and read in declaration order. Nested types that implement
`SelfCodec` are encoded inline via recursive calls. The generated code has no
reflection, no allocations, and no error paths beyond I/O errors.

---

# 4 - Migration Framework

## 4.0 - Overview

The migration framework has two halves: a code generation pipeline in Oracle that
produces migration scaffolding at development time, and a runtime in Gorp that executes
migrations at startup. The pipeline is:

```
oracle snapshot  →  schemas/.snapshots/vN/  (freeze current schema)
    ... schema changes ...
oracle migrate   →  detect diffs  →  generate frozen types + codecs + transforms
    ... developer edits transform if needed ...
gorp.OpenTable   →  run pending migrations at startup
```

## 4.1 - Schema Snapshots

`oracle snapshot` freezes the current schema files into `schemas/.snapshots/vN/`, where
`N` is derived from the Synnax Core version (`major * 1000 + minor`, read from
`core/pkg/version/VERSION`). For example, version `0.53.4` produces snapshot `v53`.

The snapshot is a complete copy of all `.oracle` files at that point in time. It serves
as the baseline for the next round of schema change detection.

## 4.2 - Schema Diff Detection

When `oracle migrate` runs, it loads both the current schemas and the latest snapshot,
analyzes both into resolution tables, and performs a deep structural comparison
(`oracle/plugin/go/migrate/schema.go`).

The comparison walks the type graph from each annotated entry, checking:

- **Struct fields**: count, order, names, optionality, and type references.
- **Enum values**: count, names, and assigned values.
- **Type aliases and distinct types**: target type identity.
- **Nested types**: recursive structural comparison with cycle detection.

The result is a `map[string]TypeDiff` keyed by qualified type name, where each diff
classifies the change:

- `TypeUnchanged` - No structural change.
- `TypeChanged` - The type's own fields changed (added, removed, type changed,
  optionality changed).
- `TypeDescendantChanged` - A nested type changed, but the type's own fields are the
  same.

Each changed field is classified as `FieldKindAdded`, `FieldKindRemoved`,
`FieldKindTypeChanged`, or `FieldKindOptionalityChanged`.

## 4.3 - Frozen Type Generation

For each type with a detected schema change, Oracle generates a `migrations/vN/`
sub-package containing:

- **`types.gen.go`** - The old struct definition, exactly as it was in the snapshot.
  Includes `GorpKey()` and `SetOptions()` methods so Gorp can iterate over these
  entries.
- **`codec.gen.go`** - ORC `EncodeOrc`/`DecodeOrc` methods for the old struct, so
  existing data encoded under the old schema can be decoded.

The package boundary provides namespacing. The frozen `Range` type lives in
`ranger/migrations/v53`, so it coexists with the current `Range` in `ranger/` without
naming conflicts.

When types reference other types across packages, the entire reachable type graph is
frozen. Output paths in the old resolution table are rewritten from `core/pkg/service/X`
to `core/pkg/service/X/migrations/vN` so that cross-package imports resolve correctly
within the frozen sub-package.

## 4.4 - Migration Transform Generation

Oracle generates three files for the migration transform:

**`migrate.gen.go`** (generated, not editable) wires up the migration chain:

```go
func RangeMigrations() []gorp.Migration {
    return []gorp.Migration{
        gorp.NewEntryMigration[Key, Key, v53.Range, Range](
            "v53_schema_migration",
            MigrateRange,
        ),
    }
}
```

When multiple schema changes accumulate across versions, the chain links through
intermediate frozen types with dependency declarations so they execute in order.

**`migrate.go`** (generated once as a template, developer-editable) contains the
transform function:

```go
func MigrateRange(
    ctx context.Context,
    old v53.Range,
) (Range, error) {
    return AutoMigrateRange(ctx, old)
}
```

The developer can customize this function to handle fields that require non-trivial
transformation. For simple changes (field additions with zero defaults, unchanged fields
that just need copying), the auto-generated code handles everything.

**`migrate_auto.gen.go`** (generated, not editable) contains the auto-copy logic that
transfers all fields with matching names and types between the old and new structs.
Comments flag new or changed fields that the developer may need to handle manually.

## 4.5 - Version Chaining

When a schema changes across multiple releases (v53 → v54 → v55), Oracle chains the
migrations. An entry encoded at v53 is decoded with the v53 frozen codec, transformed to
v54, then v54 to v55, then v55 to current. Each link declares a dependency on the
previous link via `gorp.WithDependencies`, and the topological sort in `gorp.OpenTable`
ensures correct execution order.

When a new version is created, Oracle retargets the previous version's
developer-editable `migrate.go` into the new version's sub-package, rewriting its
package declaration. The top-level `migrate.go` is regenerated to point at the latest
frozen version.

---

# 5 - Gorp Migration Runtime

## 5.0 - Migration Interface

The runtime is built on a minimal interface (`x/go/gorp/migrate.go`):

```go
type Migration interface {
    Name() string
    Run(ctx context.Context, tx Tx, ins alamos.Instrumentation) error
}
```

Two optional interfaces extend it:

- `EntryCounter` - Reports how many entries were processed, used for progress logging.
- `DependencyDeclarer` - Declares dependencies on other migrations by name, enabling
  topological ordering.

## 5.1 - Migration Constructors

`NewEntryMigration[IK, OK Key, I Entry[IK], O Entry[OK]]` creates a migration that
iterates over all entries of type `I`, decodes each using the DB's codec, applies a
`TransformFunc[I, O]`, and writes the result as type `O`. Progress is logged at
logarithmically spaced intervals (1, 10, 100, 1000, then every 1000).

`NewMigration` creates an arbitrary migration that receives a `gorp.Tx` for direct
read/write access. Used for migrations that don't fit the entry-transform pattern.

`WithDependencies` wraps any migration to declare run-after dependencies by name.

## 5.2 - Dependency Injection

Migrations sometimes need access to services or other resources not available through
the `gorp.Tx` alone. Two context helpers provide this:

```go
func WithMigrationDep[T any](ctx context.Context, dep T) context.Context
func MigrationDep[T any](ctx context.Context) (T, error)
```

Services inject dependencies into the context before calling `OpenTable`. Migration
transform functions retrieve them via `MigrationDep`. Dependencies are keyed by Go type,
so each concrete type can be injected once.

## 5.3 - OpenTable Execution

`gorp.OpenTable[K, E]` is the entry point for both table creation and migration
execution. On every startup:

1. Computes the table name from `types.Name[E]()`.
2. Reads the set of already-applied migration names from a version key in the KV store.
3. Prepends a built-in `normalizeKeysMigration` that re-keys all entries to the current
   prefix and key encoding format. This runs before any user migrations.
4. Wraps all user-provided migrations with an implicit dependency on `normalize_keys`.
5. Calls `topoSort` to filter out applied migrations and produce a valid execution
   order.
6. Executes each pending migration within the open transaction, logging progress and
   elapsed time.
7. After each successful migration, persists the migration name to the applied set.
8. Commits the transaction.

The `normalizeKeysMigration` deserves special mention. Gorp's key encoding has changed
over time (prefix format, key serialization). This built-in migration detects whether
entries are stored under an old prefix and, if so, re-keys them without re-encoding the
values. It runs exactly once and is tracked alongside user migrations.

## 5.4 - Service Wiring

Each service calls `gorp.OpenTable` during initialization, passing the Oracle-generated
migration list plus any hand-written migrations:

```go
table, err := gorp.OpenTable[uuid.UUID, Range](ctx, gorp.TableConfig[Range]{
    DB:              cfg.DB,
    Migrations:      append(RangeMigrations(), &groupsMigration{cfg: cfg}),
    Instrumentation: cfg.Instrumentation,
})
```

`RangeMigrations()` is generated by Oracle and returns the schema-change migration
chain. Service-specific migrations (like `groupsMigration` above) are appended after.

---

# 6 - Oracle CLI Commands

## 6.0 - `oracle snapshot`

Freezes the current schemas into `schemas/.snapshots/vN/`. The version is derived from
`core/pkg/version/VERSION` using `major * 1000 + minor`. Run this before making schema
changes to establish a baseline.

## 6.1 - `oracle migrate`

Loads the current schemas and the latest snapshot, diffs them, and generates frozen
types, codecs, and migration scaffolding for any changed types. After generation, it
automatically runs `oracle sync` to regenerate types and codecs for the current schema.

The command prints actionable hints:

- `✏️ {path} ← edit this` for new developer-editable transform files.
- `🔌 Wire {Pkg}Migrations() into your gorp.OpenTable call` for services that need
  migration wiring for the first time.

## 6.2 - `oracle migrate create <name>`

Creates a hand-written migration scaffold in `migrations/vN/{name}.go` for cases that
don't fit the schema-diff pattern (data backfills, index rebuilds, etc.). Automatically
sets up dependency chaining against the latest existing migration.

---

# 7 - Data Flow

A complete example of the lifecycle for a schema change:

1. **Snapshot**: `oracle snapshot` freezes `schemas/` into `schemas/.snapshots/v53/`.
2. **Schema change**: Developer adds a `color` field to `Range` in
   `schemas/ranger.oracle`.
3. **Migrate**: `oracle migrate` detects the diff, generates `ranger/migrations/v53/`
   with the old `Range` type and ORC codec, generates `migrate.gen.go` with the
   migration chain, and generates `migrate.go` with the transform template.
4. **Customize**: Developer edits `migrate.go` if the new field needs a non-zero
   default. For zero-valued defaults, the auto-copy handles everything.
5. **Startup**: `gorp.OpenTable` reads the applied migration set, finds
   `v53_schema_migration` is pending, and executes it. The migration iterates all Range
   entries, decodes each with the v53 frozen ORC codec, transforms via `MigrateRange`,
   and re-encodes with the current ORC codec.
6. **Steady state**: All entries are now encoded under the current schema. The fallback
   codec path is no longer hit for Range entries.

---

# 8 - References

- [RFC 0025 - Meta Data Structures](./0026-251214-meta-data.md)
- [Oracle Schema System](../../oracle/) (implementation)
- [ORC Codec](../../x/go/encoding/orc/) (implementation)
- [Gorp Migration Runtime](../../x/go/gorp/migrate.go) (implementation)
