# Oracle Migration System - Implementation War Game

This document traces every concrete change needed, every edge case, every thing that
breaks, and every downstream effect. No hand-waving.

## Current State Inventory

### gorp Layer (x/go/gorp/)

**Structs and codec flow:**

- `DB` embeds `options` which embeds `binary.Codec`. Default: `&binary.MsgPackCodec{}`.
  DB itself satisfies the `binary.Codec` interface through embedding.
- `Tx` interface extends `kv.Tx` + `Tools` (which is `binary.Codec`).
- `Table[K, E]` holds `codec binary.Codec` + `DB *gorp.DB`.
- `Table.Codec()` returns `t.codec` (public accessor exists at table.go:41).
- All query builders (Create, Retrieve, Update, Delete) store `codec binary.Codec`,
  received from `Table.NewX()` methods.
- `Reader[K, E]` and `Writer[K, E]` store `codec binary.Codec`, used for actual
  `Encode`/`Decode` calls.
- `resolveCodec(override, fallback)` returns override if non-nil, else fallback.
- Query `Exec()` methods call `resolveCodec(q.codec, tx)` where tx is the DB (fallback).

**Migration types:**

- `Migration` interface: `Name() string`, `Run(ctx, kv.Tx, MigrationConfig) error`
- `MigrationConfig`: `Prefix []byte`, `Codec binary.Codec`
- `TypedMigration[I, O]`: `name`, `auto AutoMigrateFunc[I, O]`, `post PostMigrateFunc[I, O]`
  - Both auto/post take `context.Context`
  - Run uses `cfg.Codec` for BOTH decode AND encode
- `CodecTransitionMigration[K, E]`: `name`, `codec binary.Codec`
  - Run uses `cfg.Codec` for decode, `m.codec` for encode
- `RawMigration`: `name`, `fn func(ctx, Tx) error`
  - Run calls `fn(ctx, WrapTx(kvTx, cfg.Codec))` - wraps with cfg's codec
- Deprecated `Migrator`: uint8 version, string keys. Used by cesium/pebblekv only.

**OpenTable flow (table.go:50-94):**

1. `codec := resolveCodec(cfg.Codec, cfg.DB)` -- resolves to binary if Codec set
2. `prefix := magicPrefix + types.Name[E]()` -- prefix from CURRENT type E
3. `migCfg := MigrationConfig{Prefix: prefix, Codec: codec}` -- **BUG: passes binary**
4. Runs versioned migrations with migCfg
5. `gorpTx := WrapTx(kvTx, codec)` -- wraps with resolved (binary) codec
6. Runs `migrateOldPrefixKeys` and `reEncodeKeys` with gorpTx + codec
7. Commits transaction
8. Returns `Table{codec, DB}`

### Service Layer

**Every service follows this pattern:**

```go
type ServiceConfig struct {
    DB    *gorp.DB
    Codec binary.Codec  // injected from layer.go
    // ... other deps
}

func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
    table, err := gorp.OpenTable[K, E](ctx, gorp.TableConfig[E]{
        DB:    cfg.DB,
        Codec: cfg.Codec,
        Migrations: []gorp.Migration{
            gorp.NewCodecTransition[K, E]("msgpack_to_binary", cfg.Codec),
        },
    })
    // ... rest of service setup
}
```

**layer.go wires 12 services with explicit codec imports:**
`user.UserCodec`, `ranger.RangeCodec`, `schematic.SchematicCodec`, etc.

**distribution/layer.go wires 2 more:**
`group.GroupCodec`, `ontology.RelationshipCodec`, `ontology.ResourceCodec`

**Codec is used in EXACTLY two places per service:**
1. Passed to `gorp.TableConfig.Codec`
2. Passed to `gorp.NewCodecTransition` as target codec

**ONE exception:** `ranger/migrate.go` stores `cfg.Codec` in its migration struct and
uses it directly for `m.codec.Marshal/Unmarshal` calls AND creates its own
`gorp.WrapTx(kvTx, migCfg.Codec)`.

**After table creation, codec flows ONLY through Table methods.** No leakage:
- `table.NewCreate/Retrieve/Update/Delete()` injects codec into builders
- `table.Observe()` creates codec-aware observable for signals
- `table.OpenNexter()` wraps reader with codec
- No service code accesses `table.codec` directly (it's private)
- `Table.Codec()` public method exists but is not called by any service code

---

## Change 1: Fix MigrationConfig.Codec Bug

### The Problem

`table.go:61`: `migCfg := MigrationConfig{Prefix: prefix, Codec: codec}`

`codec` is the TABLE's resolved codec (binary). But CodecTransition.Run uses
`cfg.Codec.Decode()` to read OLD data, which is msgpack. Binary decoder on msgpack data
= crash.

### The Fix

Rename `MigrationConfig.Codec` to `MigrationConfig.DBCodec`. Pass the DB's default
codec (msgpack), not the table override:

```go
type MigrationConfig struct {
    Prefix  []byte
    DBCodec binary.Codec  // Always the DB's default codec (msgpack)
}
```

```go
// table.go:61
migCfg := MigrationConfig{Prefix: prefix, DBCodec: cfg.DB}  // cfg.DB IS binary.Codec via embedding
```

### What Breaks

**TypedMigration.Run (migrate.go:161, 177):**
- Currently: `cfg.Codec.Decode(...)` and `cfg.Codec.Encode(...)`
- After rename: `cfg.DBCodec.Decode(...)` and `cfg.DBCodec.Encode(...)`
- This CHANGES BEHAVIOR: TypedMigration now decodes/encodes with msgpack by default
- This is CORRECT for pre-transition migrations but WRONG for post-transition
- Fix: TypedMigration gets inputCodec/outputCodec params (Change 2)

**CodecTransitionMigration.Run (migrate.go:214):**
- Currently: `cfg.Codec.Decode(...)` -- decodes with binary (BUG)
- After: `cfg.DBCodec.Decode(...)` -- decodes with msgpack (CORRECT)
- Encode still uses `m.codec` (unchanged, binary) -- CORRECT
- This FIXES the bug for codec transitions

**RawMigration.Run (migrate.go:249):**
- Currently: `WrapTx(kvTx, cfg.Codec)` -- wraps with binary
- After: `WrapTx(kvTx, cfg.DBCodec)` -- wraps with msgpack
- BREAKS ranger migration: ranger runs AFTER codec transition, expects binary data
- Fix: Change RawMigration API (see Change 3)

**6 test cases in migrate_test.go** that reference `cfg.Codec` by name.

### Edge Cases

- `migrateOldPrefixKeys` runs with the resolved codec (binary), not DBCodec. This is
  correct because it needs to encode entries in the table's format. BUT if we make key
  migrations explicit versioned steps, they'd use MigrationConfig.DBCodec by default.
  Need to ensure they get the right codec.

---

## Change 2: TypedMigration with inputCodec/outputCodec

### The Change

```go
type typedMigration[I, O any] struct {
    name        string
    inputCodec  binary.Codec  // nil -> use cfg.DBCodec
    outputCodec binary.Codec  // nil -> use cfg.DBCodec
    transform   func(old I) (O, error)
}

func NewTypedMigration[I, O any](
    name string,
    inputCodec binary.Codec,
    outputCodec binary.Codec,
    transform func(old I) (O, error),
) Migration
```

### Run implementation

```go
func (m *typedMigration[I, O]) Run(ctx context.Context, kvTx kv.Tx, cfg MigrationConfig) error {
    inCodec := m.inputCodec
    if inCodec == nil { inCodec = cfg.DBCodec }
    outCodec := m.outputCodec
    if outCodec == nil { outCodec = cfg.DBCodec }

    iter, err := kvTx.OpenIterator(kv.IterPrefix(cfg.Prefix))
    // ...
    for iter.First(); iter.Valid(); iter.Next() {
        var old I
        if err = inCodec.Decode(ctx, iter.Value(), &old); err != nil {
            return err
        }
        newEntry, err := m.transform(old)
        if err != nil {
            return err
        }
        data, err := outCodec.Encode(ctx, newEntry)
        if err != nil {
            return err
        }
        if err = kvTx.Set(ctx, iter.Key(), data); err != nil {
            return err
        }
    }
    return nil
}
```

### What Breaks

**6 test callers of NewTypedMigration** in migrate_test.go:
- Current: `NewTypedMigration[V1, V2]("name", autoFn, postFn)`
- New: `NewTypedMigration[V1, V2]("name", nil, nil, transformFn)`
- Must rewrite all 6 to combine auto+post into single transform

**AutoMigrateFunc and PostMigrateFunc types** become unused. Delete them.

**Context parameter removed from transform.** All 6 test transforms use `_` for context.
No behavioral change.

### Codec Transition Becomes TypedMigration

Delete `codecTransitionMigration` entirely. Replace all 15 service calls:

Before:
```go
gorp.NewCodecTransition[uuid.UUID, Schematic]("msgpack_to_binary", cfg.Codec)
```

After:
```go
gorp.NewTypedMigration[Schematic, Schematic](
    "msgpack_to_binary",
    nil,       // decode with DBCodec (msgpack)
    cfg.Codec, // encode with binary codec
    func(old Schematic) (Schematic, error) { return old, nil },
)
```

**Problem:** This uses the CURRENT `Schematic` type for both I and O. But Schematic is in
the parent package. If migrations move to a sub-package, this won't compile.

**For now (before sub-package migration):** This works because the migration is inline in
`service.go`, same package as `Schematic`.

**Later (with sub-package):** The migration uses frozen types:
```go
gorp.NewTypedMigration[v1.Schematic, v1.Schematic](
    "msgpack_to_binary",
    nil,        // DBCodec (msgpack)
    v1.Codec,   // frozen binary codec
    func(old v1.Schematic) (v1.Schematic, error) { return old, nil },
)
```

---

## Change 3: RawMigration API

### The Problem

Current RawMigration wraps kvTx with cfg.Codec:
```go
func (m *rawMigration) Run(ctx, kvTx, cfg) error {
    return m.fn(ctx, WrapTx(kvTx, cfg.Codec))
}
```

After Change 1, cfg.DBCodec is msgpack. Ranger migration runs after codec transition,
needs binary. Wrapping with msgpack breaks it.

### Option A: RawMigration passes kv.Tx directly

```go
type RawMigrationFunc func(ctx context.Context, kvTx kv.Tx, cfg MigrationConfig) error
```

Developer wraps with whatever codec they need. Ranger migration already does its own
WrapTx call, so just change it to use m.codec instead of migCfg.Codec.

**Impact on ranger/migrate.go:**
- Current: `gorpTx := gorp.WrapTx(kvTx, migCfg.Codec)`
- New: `gorpTx := gorp.WrapTx(kvTx, m.codec)` -- uses its own stored codec
- Minimal change, already stores the codec

### Option B: RawMigration receives both codecs

MigrationConfig carries both DBCodec and TableCodec. RawMigration wraps with TableCodec.

**Problem:** TableCodec is the CURRENT codec. But during migration, intermediate versions
might exist. This couples the raw migration to the current codec, not the version it
operates on.

### Recommendation: Option A

RawMigration passes raw kv.Tx + MigrationConfig. Developer wraps as needed. This is the
honest escape hatch: you get raw access, you handle codec yourself.

### What Breaks

**NewRawMigration signature changes:**
- Current: `fn func(ctx context.Context, tx gorp.Tx) error`
- New: `fn func(ctx context.Context, kvTx kv.Tx, cfg MigrationConfig) error`

**Callers:**
- ranger/migrate.go: Already implements Migration directly, NOT using NewRawMigration.
  No change needed. It receives `(ctx, kvTx kv.Tx, cfg MigrationConfig)` already.
- access/migrate.go: Needs audit. May use NewRawMigration.
- Test callers: Need signature update.

Actually wait - ranger migration already implements Migration interface directly. It does
NOT use NewRawMigration. So NewRawMigration callers are only in tests.

---

## Change 4: GorpCodec() on Entry Types

### The Change

Add optional interface detection in OpenTable:

```go
func OpenTable[K Key, E Entry[K]](ctx context.Context, cfg TableConfig[E]) (*Table[K, E], error) {
    codec := cfg.Codec
    if codec == nil {
        var zero E
        if hc, ok := any(&zero).(interface{ GorpCodec() binary.Codec }); ok {
            codec = hc.GorpCodec()
        }
    }
    codec = resolveCodec(codec, cfg.DB)
    // ... rest unchanged
}
```

### What Needs to Change in Services

**For each Oracle-managed type, add to helpers.go:**
```go
func (s Schematic) GorpCodec() binary.Codec { return SchematicCodec }
```

Or Oracle generates this in codec.gen.go alongside the codec definition.

**Then remove Codec from ServiceConfig:**
```go
type ServiceConfig struct {
    DB *gorp.DB
    // Codec binary.Codec  <-- DELETE
    // ...
}
```

**Then remove codec wiring from layer.go** (12 services) and distribution/layer.go
(2 services).

### What Breaks

**Every service's OpenTable call** currently passes `Codec: cfg.Codec`:
```go
gorp.OpenTable[K, E](ctx, gorp.TableConfig[E]{
    DB:    cfg.DB,
    Codec: cfg.Codec,  // <-- remove this
    Migrations: []gorp.Migration{
        gorp.NewCodecTransition[K, E]("msgpack_to_binary", cfg.Codec),  // <-- where does codec come from?
    },
})
```

**The migration still needs the codec.** Even though OpenTable auto-detects the codec for
runtime use, the migration needs it as a parameter to NewCodecTransition (or
NewTypedMigration with outputCodec).

Options:
a. Migration gets codec from the type: `var zero E; codec := zero.(HasCodec).GorpCodec()`
b. Migration hardcodes the codec: `SchematicCodec` is in the same package
c. Migrations move to sub-package and carry frozen codecs

For the codec transition specifically (before sub-package):
```go
table, err := gorp.OpenTable[uuid.UUID, Schematic](ctx, gorp.TableConfig[Schematic]{
    DB: cfg.DB,
    // No Codec field - auto-detected from Schematic.GorpCodec()
    Migrations: []gorp.Migration{
        gorp.NewTypedMigration[Schematic, Schematic](
            "msgpack_to_binary",
            nil,            // decode with DBCodec (msgpack)
            SchematicCodec, // encode with binary - direct reference, same package
            func(old Schematic) (Schematic, error) { return old, nil },
        ),
    },
})
```

This works because `SchematicCodec` is defined in `codec.gen.go` in the same package.
No DI needed. Direct reference.

**Ranger migration** stores cfg.Codec:
```go
type rangeGroupsMigration struct {
    codec binary.Codec
}
```

After removing Codec from ServiceConfig, where does it get the codec?
- From `ranger.RangeCodec` directly (same package)
- Change: `newRangeGroupsMigration(cfg)` -> create with `RangeCodec` constant

### Edge Case: Ontology

Ontology has TWO tables (Resource + Relationship) with TWO codecs:
```go
ontology.Config{
    RelationshipCodec: ontology.RelationshipCodec,
    ResourceCodec:     ontology.ResourceCodec,
}
```

With GorpCodec(), each type carries its own:
```go
func (Resource) GorpCodec() binary.Codec { return ResourceCodec }
func (Relationship) GorpCodec() binary.Codec { return RelationshipCodec }
```

Two OpenTable calls, each auto-detects the right codec. Clean.

### Edge Case: Distribution Layer GorpCodec Config

`distribution/layer.go:58` has:
```go
GorpCodec binary.Codec  // default: &binary.MsgPackCodec{}
```

This sets the DB-level codec. It's NOT a per-table codec. It's the default for ALL
tables in the distribution layer. This stays unchanged. It's the fallback when
GorpCodec() is not implemented.

---

## Change 5: Make Key Migrations Explicit

### The Problem

`migrateOldPrefixKeys` and `reEncodeKeys` run unconditionally after versioned migrations
(table.go:83-88). They should be explicit versioned steps.

### Implementation Challenge

These functions are generic: `migrateOldPrefixKeys[K Key, E Entry[K]](ctx, tx, codec)`.
They need the resolved table codec, not DBCodec.

If they become TypedMigration, they'd use cfg.DBCodec for decode/encode. But they need
the TABLE codec (binary) because they re-encode with the current format.

**This means they can't be simple TypedMigrations with nil codecs.** They need explicit
codec parameters.

But at migration registration time (in service.go or migrations sub-package), the table
codec is available as a constant (`SchematicCodec`).

```go
Migrations: []gorp.Migration{
    gorp.NewTypedMigration[Schematic, Schematic](
        "migrate_prefix_keys",
        SchematicCodec, // read with binary
        SchematicCodec, // write with binary
        func(old Schematic) (Schematic, error) { return old, nil },
    ),
}
```

**But this is wrong.** migrateOldPrefixKeys doesn't transform entries. It MOVES them from
old key prefix to new key prefix. The value bytes don't change. This is a KEY operation,
not a VALUE operation.

TypedMigration operates on VALUES (decode, transform, encode). Key prefix migration
operates on KEYS (delete old key, write new key with same value).

**TypedMigration is the wrong abstraction for key migrations.** Key migrations need raw
kv.Tx access to manipulate keys directly.

### Revised Plan

Keep key migrations as a SEPARATE concern. Two options:

a. Leave them as implicit steps in OpenTable (current behavior, just tracked by version)
b. Make them a third migration type: `KeyMigration` that handles key-level operations

Option (a) is simpler. The RFC mentions `KeyMigration[K, E]` as a third type. But
adding a third migration type contradicts our decision to have only TypedMigration +
RawMigration.

Option (b): implement key migrations as RawMigration implementations. The developer
(or Oracle) creates a raw migration that does the key manipulation.

**BUT:** `migrateOldPrefixKeys` needs type parameters [K, E] to compute the old prefix
(via `tx.Encode(ctx, types.Name[E]())`). A raw migration function doesn't have type
parameters. It gets raw kv.Tx.

**This is a real problem.** The key migration functions are inherently generic. They need
to know the type to compute the prefix. TypedMigration gives them the type through I/O
parameters, but they don't transform values.

### Honest Assessment

Making key migrations explicit versioned steps is HARDER than it sounds. The current
implementation works (runs every startup, cheap no-op). Forcing them into the Migration
interface creates impedance mismatch.

**Revised decision:** Keep key migrations as implicit steps in OpenTable for now. They
run after all versioned migrations, within the same transaction. They're cheap no-ops on
subsequent startups. Revisit if this becomes a problem.

---

## Change 6: Delete CodecTransitionMigration

### The Change

Delete `codecTransitionMigration[K, E]` struct and `NewCodecTransition` function from
migrate.go. Replace all 15 callers with `NewTypedMigration` identity transform.

### What Breaks

**15 service files** that call `gorp.NewCodecTransition`:
Each needs to change from:
```go
gorp.NewCodecTransition[uuid.UUID, Schematic]("msgpack_to_binary", cfg.Codec)
```
To:
```go
gorp.NewTypedMigration[Schematic, Schematic](
    "msgpack_to_binary", nil, SchematicCodec,
    func(old Schematic) (Schematic, error) { return old, nil },
)
```

**Test callers** of NewCodecTransition in migrate_test.go.

### Concern

The TypedMigration approach for codec transition is LESS efficient than
CodecTransitionMigration. TypedMigration calls the transform function for each entry
(even though it's identity). CodecTransition doesn't have this overhead.

For metadata volumes (thousands of entries), this overhead is negligible. But it's a
regression in principle.

**Alternative:** Keep NewCodecTransition as a convenience function that internally
creates a TypedMigration:

```go
func NewCodecTransition[K Key, E Entry[K]](name string, targetCodec binary.Codec) Migration {
    return NewTypedMigration[E, E](
        name, nil, targetCodec,
        func(old E) (E, error) { return old, nil },
    )
}
```

This preserves the convenient API while unifying the implementation. Best of both worlds.

---

## Implementation Order

### Phase 1: gorp Layer (x/go/gorp/)

1. Rename MigrationConfig.Codec to MigrationConfig.DBCodec
2. Fix table.go:61 to pass cfg.DB (not resolved codec)
3. Add inputCodec/outputCodec to TypedMigration
4. Change transform signature to `func(old I) (O, error)` (drop context)
5. Delete AutoMigrateFunc and PostMigrateFunc types
6. Change RawMigration to pass kv.Tx + MigrationConfig (not pre-wrapped Tx)
7. Rewrite NewCodecTransition as convenience wrapper around TypedMigration
8. Add GorpCodec() detection in OpenTable
9. Keep migrateOldPrefixKeys/reEncodeKeys as implicit steps
10. Update ALL tests in migrate_test.go

### Phase 2: Service Layer (core/pkg/)

1. Add GorpCodec() method to each Oracle-managed type (in helpers.go or codec.gen.go)
2. Remove Codec from each ServiceConfig + Override method
3. Remove codec wiring from layer.go and distribution/layer.go
4. Update OpenTable calls to remove Codec field
5. Update migration construction (NewTypedMigration instead of NewCodecTransition)
6. Update ranger/migrate.go to use RangeCodec directly instead of migCfg.Codec
7. Update ALL tests that construct ServiceConfig with Codec

### Phase 3: Migrations Sub-Package (per service)

1. Create migrations/ sub-package for each service
2. Move frozen types + codecs into sub-package
3. Create All() function returning []gorp.Migration
4. Update service.go to import migrations.All()
5. Verify no circular dependencies

---

## Risk Assessment

### Low Risk
- MigrationConfig rename (mechanical, find-and-replace)
- TypedMigration signature change (only test callers)
- GorpCodec() detection in OpenTable (additive, non-breaking)
- Removing Codec from ServiceConfig (after GorpCodec() works)

### Medium Risk
- RawMigration API change (affects ranger migration, tests)
- Deleting CodecTransitionMigration (15 call sites to update)
- Key migrations staying implicit (might need revisiting)

### High Risk
- The MigrationConfig.DBCodec fix changes behavior for ALL migrations.
  CodecTransition now correctly decodes msgpack. But any migration that relied on
  cfg.Codec being the binary codec (like ranger's RawMigration via WrapTx) breaks.
  Must audit every migration.
- Multiple services changing simultaneously. If one breaks, hard to isolate.

### Unknown Risk
- Snapshot mechanism for old schema (partially resolved - sy-3824 uses schemas/.snapshots/)

---

## CRITICAL FINDINGS FROM DEEP RESEARCH

### Finding 1: Core Doesn't Compile

`core/pkg/distribution/ontology/codec.gen.go:23` imports a non-existent package:
```go
resource "github.com/synnaxlabs/synnax/pkg/distribution/ontology/internal/resource"
```

This package does not exist. Lines 83 and 175 use `resource.Type(...)` but should use
`ResourceType(...)` (defined in the same ontology package). **The codebase is broken on
the current branch.** This must be fixed before any work can proceed.

### Finding 2: Type Assertions Break Frozen Types

Generated codecs use Go type assertions:
```go
func (schematicCodec) Encode(ctx context.Context, value any) ([]byte, error) {
    s := value.(Schematic)  // PANICS if value is v1.Schematic from different package
```

A frozen type `v1.Schematic` in `migrations/v1/` is a DIFFERENT Go type from
`schematic.Schematic`. Passing it to `SchematicCodec.Encode()` will panic.

**Consequence:** Each frozen type version MUST have its own codec that type-asserts on
the correct type. Oracle must generate a separate codec per version. You cannot share a
codec between the current type and frozen types.

**For the codec transition specifically:** This is NOT a problem because the transition
uses the CURRENT type for both input and output:
- MsgPackCodec.Decode(data, &schematic.Schematic{}) -- uses struct tags, works
- SchematicCodec.Encode(schematic.Schematic{}) -- asserts Schematic, works

Both steps use `schematic.Schematic`. No frozen type involved. The codec transition
re-encodes the current type from msgpack to binary.

**For schema change migrations (v2+):** This IS a problem. The migration decodes old
data into `v1.Schematic` (frozen type). The v1 codec must assert on `v1.Schematic`.
Oracle must generate `v1.SchematicCodec` alongside `v1.Schematic`.

### Finding 3: sy-3824 Already Has Snapshot Mechanism

The migrate plugin on sy-3824 extends `plugin.Request` with:
```go
OldResolutions *resolution.Table  // Previous schema snapshot
SnapshotVersion int               // Latest version number
```

The `oracle migrate` command on sy-3824:
1. Loads old snapshot from `schemas/.snapshots/vN/`
2. Diffs old vs new resolutions
3. Generates migration files
4. Creates new snapshot `schemas/.snapshots/v(N+1)/`

The snapshot mechanism exists. It copies `.oracle` files. It works.

### Finding 4: Binary Layout Is Deterministic by Field Order

The marshal plugin calls `resolution.UnifiedFields(typ, table)` which returns fields in
**declaration order from the .oracle file**. Field constants are numbered sequentially:
`SchematicFieldKey=0, SchematicFieldName=1, etc.`

As long as the frozen type preserves the same field declaration order (which Oracle
guarantees since it generates from the .oracle snapshot), the binary layout is identical
across packages. **The binary bytes don't depend on Go type names or package paths.**

### Finding 5: MsgPack Uses Field Names, Binary Uses Positions

- `MsgPackCodec` uses `msgpack.Marshal/Unmarshal` which reads struct tags (`msgpack:"key"`)
  Field names must match. Type name doesn't matter. Package doesn't matter.
- Generated binary codecs use positional encoding. Field ORDER must match.
  Type assertions must match (see Finding 2). Package doesn't matter for bytes.

This means:
- Codec transition (msgpack -> binary): Decode via msgpack struct tags (name-based),
  encode via binary positional. Current type works for both. No frozen type needed.
- Schema migration (v1 binary -> v2 binary): Decode with v1 frozen codec (positional,
  asserts v1.Type), encode with v2 frozen codec (positional, asserts v2.Type).
  Each version MUST have its own codec.

### Finding 6: Access Migration Is Separate

`core/cmd/start/access/migrate.go` runs during server startup via `ProvisionRootUser()`,
NOT through gorp.OpenTable. It uses its own KV flag (`sy_rbac_migration_performed`) for
tracking. This is completely separate from the gorp migration system and doesn't need to
change.

### Finding 7: sy-3824 Introduces gorp.Codec[E] Generic Interface

The sy-3824 branch adds a generic `Codec[E]` to gorp alongside `binary.Codec`:
```go
type Codec[E any] interface {
    Marshal(ctx context.Context, entry E) ([]byte, error)
    Unmarshal(ctx context.Context, data []byte) (E, error)
}
```

This was used for TypedMigration input/output codecs. **We decided to use binary.Codec
instead.** This means we're diverging from sy-3824's approach. The TypedMigration
input/output codecs will be `binary.Codec` (with `any` type assertions), not
`gorp.Codec[E]` (with generic type safety).

This works but means the type assertion happens at runtime inside the codec, not at
compile time. The frozen codec generated for v1.Schematic will do
`value.(v1.Schematic)` internally.

### Finding 8: No Frozen Codec on sy-3824's v1

On sy-3824, `migrations/v1/v1.gen.go` contains ONLY the frozen type definition. No
frozen codec. The codec transition uses `NewCodecTransition` which decodes with
`cfg.Codec` (the DB default msgpack) and encodes with the target binary codec.

For the codec transition, this is correct: msgpack decodes by field names (works with
any type that has matching struct tags), binary encodes by position (works because field
order is preserved).

For schema change migrations (v2+), frozen codecs WOULD be needed but aren't generated
yet on sy-3824. That's future work.

---

## REVISED IMPLEMENTATION ORDER

### Step 0: Fix Compilation

Fix `ontology/codec.gen.go` broken import. Replace `resource.Type()` references with
`ResourceType()`. Get core compiling.

### Step 1: gorp Layer Changes (x/go/gorp/)

1. Rename MigrationConfig.Codec -> MigrationConfig.DBCodec
2. Fix table.go:61 to pass cfg.DB as DBCodec
3. Refactor TypedMigration: single transform, inputCodec/outputCodec, no context
4. Refactor RawMigration: pass kv.Tx + MigrationConfig, not pre-wrapped Tx
5. Add GorpCodec() detection in OpenTable
6. Rewrite NewCodecTransition as convenience wrapper
7. Update ALL tests

### Step 2: Service Layer (core/pkg/)

1. Add GorpCodec() to each type's helpers.go (or generate in codec.gen.go)
2. Remove Codec from ServiceConfig structs
3. Remove codec wiring from layer.go / distribution/layer.go
4. Update OpenTable calls (remove Codec field)
5. Update migration construction (use type-local codec constants)
6. Fix ranger/migrate.go to use RangeCodec directly
7. Update tests

### Step 3: Oracle Plugin Changes

1. Modify marshal plugin to generate GorpCodec() method alongside codec
2. Verify migrate plugin generates frozen types with correct field order
3. Add frozen CODEC generation for each migration version (not just frozen types)
4. Verify snapshot mechanism works for the full dependency tree
5. Test end-to-end: change schema, run oracle migrate, verify output

### Unresolved

- How to reconcile sy-3824's work with the new design (cherry-pick vs rewrite)
- Whether to keep gorp.Codec[E] from sy-3824 or use binary.Codec only
- Exact structure of migrations sub-package (flat vs nested v1/v2 sub-packages)
- How migrateOldPrefixKeys/reEncodeKeys interact with explicit versioned migrations
