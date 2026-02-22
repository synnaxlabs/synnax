# Oracle Migration System — Phased Implementation Plan

Reference: [RFC 0027 - Oracle Migrations](./0027-oracle-migrations.md)

This document breaks the Oracle migration system into bounded, independently testable
phases. Each phase includes instructions detailed enough to hand to a Claude Code
session.

---

## Phase 1: gorp Migration Interface & Infrastructure

**Goal**: Replace the existing `gorp.Migrator` struct with a `Migration` interface and
update `EntryManager` to be the single entry point for all migrations (schema + key
re-encoding). No Oracle code generation yet — this is pure gorp infrastructure.

**Scope**: `x/go/gorp/`

### What to build

1. **`Migration` interface** (`x/go/gorp/migrate.go`):

   ```go
   type Migration interface {
       Name() string
       Run(ctx context.Context, tx kv.Tx, cfg MigrationConfig) error
   }

   type MigrationConfig struct {
       Prefix []byte
       Codec  binary.Codec
   }
   ```

2. **`TypedMigration[I, O]`** (`x/go/gorp/migrate.go`):
   - Implements `Migration` interface
   - `NewTypedMigration[I, O](name, autoFn, postFn) Migration`
   - `Run` iterates all entries under `cfg.Prefix`, decodes each as `I` (using
     `GorpUnmarshaler` if implemented, else `cfg.Codec`), calls `autoFn` then `postFn`,
     encodes as `O` (using `GorpMarshaler` if implemented, else `cfg.Codec`), writes back
   - Type aliases: `AutoMigrateFunc[I, O]`, `PostMigrateFunc[I, O]`

3. **`RawMigration`** (`x/go/gorp/migrate.go`):
   - Implements `Migration` interface
   - `NewRawMigration(name, func(ctx, Tx) error) Migration`
   - `Run` wraps `kv.Tx` into `gorp.Tx` and calls the function

4. **`KeyMigration[K, E]`** (`x/go/gorp/migrate.go`):
   - Implements `Migration` interface
   - Extracts existing `reEncodeKeys` logic
   - Name returns `"re_encode_keys"`

5. **`GorpMarshaler` / `GorpUnmarshaler` interfaces** (`x/go/gorp/marshal.go`):

   ```go
   type GorpMarshaler interface {
       GorpMarshal(ctx context.Context) ([]byte, error)
   }
   type GorpUnmarshaler interface {
       GorpUnmarshal(ctx context.Context, data []byte) error
   }
   ```

   Update `Writer.set()` to check `GorpMarshaler` before falling back to codec.
   Update `Iterator.Value()` to check `GorpUnmarshaler` before falling back to codec.

6. **Update `EntryManager[K, E]`** (`x/go/gorp/manager.go`):
   - `OpenEntryManager[K, E](ctx, db, migrations ...Migration) (*EntryManager, error)`
   - Derives prefix from `types.Name[E]()`: `"__gorp__//" + typeName`
   - Version key: `"__gorp__//" + typeName + "/version"`
   - Appends `KeyMigration[K, E]{}` as the implicit final migration
   - Runs all migrations sequentially, tracking progress with a uint16 version counter
   - Existing `migrateOldPrefixKeys` becomes a migration too, or is folded into
     `KeyMigration`
   - Zero-migration case (`OpenEntryManager(ctx, db)`) still works — only key
     re-encoding runs

7. **Backward compatibility**: The old `gorp.Migrator` struct and `MigrationSpec` can
   remain temporarily (deprecated) so ranger and other existing callers don't break.
   Mark them deprecated with a comment. They will be removed in a later phase.

### What to test (Ginkgo/Gomega in `x/go/gorp/`)

- **Migration interface**: TypedMigration decodes, transforms, encodes correctly
- **GorpMarshaler/GorpUnmarshaler**: Writer uses per-type marshaling when available,
  falls back to codec when not. Reader/Iterator same.
- **Version tracking**: uint16 big-endian, starts at 0, increments per migration
- **Sequential execution**: Migrations run in order, skipping already-completed ones
- **Multi-step chaining**: v1→v2→v3 runs as two full passes
- **Error handling**: Failed migration rolls back transaction, version not incremented
- **Zero-migration case**: EntryManager with no migrations still runs key re-encoding
- **RawMigration**: Gets a working gorp.Tx, can read/write entries
- **KeyMigration**: Exercises re-encode keys path
- **Mixed chain**: TypedMigration + RawMigration + KeyMigration in sequence

### Test helper (`x/go/gorp/testutil/`)

Build a `gorp/testutil` package with Gomega-compatible helpers:

```go
// testutil.TestMigration seeds old entries, runs a migration, and asserts results.
func TestMigration[I gorp.Entry[K], O gorp.Entry[K], K gorp.Key](
    oldEntries []I,
    expectedEntries []O,
    migration gorp.Migration,
) // returns Gomega-compatible assertion or uses Expect internally
```

This helper:
1. Creates an in-memory KV store (use existing test infrastructure)
2. Wraps it in gorp.DB with msgpack codec
3. Encodes and writes old entries under the canonical prefix
4. Runs the migration via a MigrationConfig
5. Reads back all entries, decodes as new type
6. Asserts results match expected entries
7. Validates version counter was incremented

### Files to modify

- `x/go/gorp/migrate.go` — rewrite (Migration interface, TypedMigration, RawMigration)
- `x/go/gorp/marshal.go` — new file (GorpMarshaler/GorpUnmarshaler interfaces)
- `x/go/gorp/manager.go` — update OpenEntryManager signature and implementation
- `x/go/gorp/writer.go` — add GorpMarshaler check in `set()`
- `x/go/gorp/reader.go` — add GorpUnmarshaler check in `Iterator.Value()`
- `x/go/gorp/migrate_test.go` — rewrite tests for new interface
- `x/go/gorp/testutil/` — new package with test helpers
- All callers of `OpenEntryManager` — update to new signature (add `...Migration`)

### Acceptance criteria

- `cd x/go && ginkgo -r ./gorp/...` passes
- All existing service code compiles (no breaking changes to callers that pass zero
  migrations)
- The old `gorp.Migrator` is marked deprecated but still compiles

---

## Phase 2: Oracle `GorpMarshal`/`GorpUnmarshal` Generation

**Goal**: Extend Oracle's `go/types` plugin to generate `GorpMarshal`/`GorpUnmarshal`
methods on all Oracle-managed gorp types. This uses the existing `oracle sync` command —
no new CLI commands needed.

**Scope**: `oracle/plugin/go/`, Oracle-managed type packages

### What to build

1. **Detect gorp entry types**: In the `go/types` plugin, identify types that have a
   `@key` annotation (these are gorp entries). Oracle already knows which types have keys.

2. **Generate `marshal.gen.go`** for each gorp entry type:

   ```go
   func (s *Schematic) GorpMarshal(ctx context.Context) ([]byte, error) {
       pb, err := SchematicToPB(ctx, *s)
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
       result, err := SchematicFromPB(ctx, pbMsg)
       if err != nil {
           return err
       }
       *s = result
       return nil
   }
   ```

   Oracle already generates `ToPB`/`FromPB` translators. The marshal methods are thin
   wrappers.

3. **Generate for all ~21 Oracle-managed gorp types**: Every type with `@key` and `@pb`
   annotations gets marshal methods.

### What to test

- `oracle sync` generates `marshal.gen.go` files for all gorp entry types
- Generated methods compile
- Round-trip test: marshal → unmarshal produces identical struct
- Fallback: types without `GorpMarshaler` still use msgpack codec (gorp tests from
  Phase 1 cover this)
- Run full `oracle check` to verify no regressions

### Acceptance criteria

- `oracle sync` generates marshal.gen.go for all gorp entry types
- `cd core && go build ./...` compiles
- Round-trip tests pass for representative types (schematic, workspace, arc, etc.)

---

## Phase 3: Oracle Migration Plugin — Codec Transition (Full Mode)

**Goal**: Build the Oracle migration code generator for the MVP: codec transition
(msgpack → protobuf) for all Oracle-managed gorp types. This is "full mode" — both
auto-migrate and post-migrate are fully generated, developer touches nothing.

**Scope**: `oracle/plugin/migrate/`, `oracle/cmd/`

### What to build

1. **New Oracle plugin**: `oracle/plugin/migrate/` — a new plugin that generates
   migration files. This plugin is invoked by a new `oracle migrate generate` CLI
   command, NOT by `oracle sync`.

2. **`oracle migrate generate` CLI command** (`oracle/cmd/migrate.go`):
   - New cobra subcommand under `oracle migrate`
   - For the MVP (codec transition), the behavior is simple:
     - Enumerate all Oracle-managed gorp types (types with `@key` + `@pb`)
     - For each type, generate `migrations/v1/` sub-package with:
       - `auto.gen.go`: Legacy type snapshot (identical fields, msgpack tags) +
         `AutoMigrateV1ToV2` function (identity transform copying all fields)
       - `migrate.go`: `PostMigrateV1ToV2` function (empty body, codec transition only)
     - Generate `migrations/v2/` sub-package with:
       - `auto.gen.go`: Current type snapshot with `GorpMarshal`/`GorpUnmarshal`
         (no auto-migrate yet — v2 is the current version)
       - `pb/`: Snapshotted protobuf definitions
     - Generate `migrations/migrate.gen.go`: `All()` function returning the ordered
       migration list with a single `gorp.NewTypedMigration` call

3. **Legacy type snapshot generation**: The `v1/auto.gen.go` legacy type must be a
   faithful snapshot of the current struct — same fields, same types. For pre-transition
   types, use `msgpack` struct tags (no `GorpUnmarshaler`). The migration runner will
   decode these using the generic msgpack codec.

4. **Migration file placement**: Files go at `<@go output>/migrations/vN/`. Oracle
   already knows the `@go output` path for each type.

5. **`GorpKey()` and `SetOptions()` on legacy types**: The legacy type in `v1/` needs
   these methods so it satisfies the iterator's expectations. Generate them.

### What to test

- `oracle migrate generate` produces correct file structure for each type
- Generated `v1/auto.gen.go` has correct legacy type snapshot
- Generated `AutoMigrateV1ToV2` copies all fields correctly
- Generated `PostMigrateV1ToV2` compiles (empty body)
- Generated `migrate.gen.go` has correct `All()` function
- Generated code compiles: `go build ./...`
- **Integration test**: Seed entries encoded with msgpack, run the generated migration,
  verify entries are now protobuf-encoded and decode correctly
- Use the `gorp/testutil` helpers from Phase 1 to test each generated migration

### Oracle-managed gorp types (~21 types)

All of these get codec transition migrations:

| Type | Package | Key Type |
|------|---------|----------|
| Workspace | `core/pkg/service/workspace` | `uuid.UUID` |
| User | `core/pkg/service/user` | `uuid.UUID` |
| Task | `core/pkg/service/task` | `Key` (uint64) |
| Schematic | `core/pkg/service/schematic` | `uuid.UUID` |
| Symbol | `core/pkg/service/schematic/symbol` | `uuid.UUID` |
| Device | `core/pkg/service/device` | `string` |
| View | `core/pkg/service/view` | `uuid.UUID` |
| Arc | `core/pkg/service/arc` | `uuid.UUID` |
| Table | `core/pkg/service/table` | `uuid.UUID` |
| LinePlot | `core/pkg/service/lineplot` | `uuid.UUID` |
| Rack | `core/pkg/service/rack` | `Key` (uint32) |
| Log | `core/pkg/service/log` | `uuid.UUID` |
| Range | `core/pkg/service/ranger` | `uuid.UUID` |
| Alias | `core/pkg/service/ranger/alias` | `string` |
| Pair | `core/pkg/service/ranger/kv` | `string` |
| Group | `core/pkg/distribution/group` | `uuid.UUID` |
| Relationship | `core/pkg/distribution/ontology` | `[]byte` |
| Label | `x/go/label` | `uuid.UUID` |
| Role | `core/pkg/service/access/rbac/role` | `uuid.UUID` |
| Policy | `core/pkg/service/access/rbac/policy` | `uuid.UUID` |
| SecureCredentials | `core/pkg/service/auth` | `string` |

### Acceptance criteria

- `oracle migrate generate` produces migration files for all ~21 types
- `go build ./...` compiles with generated files
- Codec transition integration tests pass for representative types
- Each migration is independently testable via `gorp/testutil`

---

## Phase 4: Service Wiring & End-to-End Codec Transition

**Goal**: Wire up the generated migrations into each service's `OpenEntryManager` call.
Run the full codec transition end-to-end.

**Scope**: `core/pkg/service/*/`, `core/pkg/distribution/*/`

### What to build

1. **Update each service's `OpenService`** to pass migrations to `OpenEntryManager`:

   ```go
   entryManager, err := gorp.OpenEntryManager[uuid.UUID, Schematic](
       ctx, cfg.DB, migrations.All()...,
   )
   ```

   Do this for all ~21 services that have Oracle-managed gorp types.

2. **Remove old `gorp.Migrator` usage** from any service that previously used it
   (currently only ranger uses it). For ranger, keep its existing `migrateRangeGroups`
   as a `RawMigration` passed before the codec transition in the migration chain.
   Actually — defer ranger port to Phase 7. For now, ranger keeps its old migrator
   AND the new entryManager pattern.

3. **Server startup order**: Verify that services start in dependency order. Schematic
   before Workspace (if Workspace depends on Schematic), etc. The existing startup
   sequence in `core/` should already handle this.

### What to test

- Server starts successfully with existing msgpack-encoded data
- After startup, all entries are protobuf-encoded
- Server restarts (second time) — migrations are no-op (version counter already at
  latest)
- Read/write operations work correctly after codec transition
- Existing integration tests pass

### Acceptance criteria

- Server boots and runs codec transition for all Oracle-managed types
- All existing tests pass
- `go test ./...` passes across the core module

---

## Phase 5: Schema Snapshots & `oracle migrate check` (CI Enforcement)

**Goal**: Implement schema snapshot storage and the `oracle migrate check` CLI command
for CI enforcement. This ensures no `.oracle` schema change ships without a
corresponding migration.

**Scope**: `oracle/`, `schemas/.snapshots/`

### What to build

1. **Snapshot storage**: When `oracle migrate generate` runs, copy current `.oracle`
   source files into `schemas/.snapshots/v<N>/`. The snapshot is the authoritative
   record of schema state at each migration version.

2. **Update `oracle migrate generate`** to:
   - Create snapshot after generating migration files
   - Store version metadata alongside snapshots

3. **`oracle migrate check` CLI command** (`oracle/cmd/migrate.go`):
   - Diff current `.oracle` files against latest snapshot
   - If schemas differ and no new migration files exist, exit with error code 1
   - If schemas match (or new migrations account for changes), exit with code 0
   - Print actionable error message: "schema changed but no migration generated.
     Run 'oracle migrate generate' and commit the result."

4. **Schema diffing**: Compare two sets of `.oracle` files. This is a file-level diff
   (not field-level yet — that comes in Phase 6). If any `.oracle` file content differs
   from the snapshot, a migration is required.

### What to test

- `oracle migrate generate` creates snapshots in correct location
- `oracle migrate check` passes when schemas match snapshot
- `oracle migrate check` fails when schemas differ without migration
- `oracle migrate check` passes when schemas differ WITH migration
- Snapshot files are human-readable copies of `.oracle` source

### Acceptance criteria

- `oracle migrate check` can be added to CI pipeline
- Snapshot directory structure is correct
- Diffing logic correctly detects changes

---

## Phase 6: Schema Diff Engine & Skeleton/Propagation Mode Generation

**Goal**: Build the schema diff engine that compares two `resolution.Table`s
field-by-field and generates skeleton mode (direct schema changes) and propagation mode
(nested dependency changes) migrations. This is the most complex phase and requires
thorough unit testing.

**Scope**: `oracle/plugin/migrate/`

### What to build

1. **Schema diff engine** (`oracle/plugin/migrate/diff/`):
   - Compare two `resolution.Table`s (old snapshot vs. current)
   - Classify changes per type:
     - **Unchanged**: All fields match by name and type
     - **Field added**: New field name not in old type
     - **Field removed**: Old field name not in new type
     - **Field type changed**: Same field name, different type
   - No rename detection (shows as remove + add)
   - Output: `[]TypeDiff` with per-field change classification

2. **Dependency graph** (`oracle/plugin/migrate/deps/`):
   - Build a directed graph of type dependencies from `resolution.Table`
   - Track which gorp entry types transitively contain which nested types
   - When a nested type changes, identify all affected gorp entries
   - Use Oracle's existing `TopologicalSort()` for ordering

3. **Generation modes**:

   **Skeleton mode** (direct schema changes):
   - `auto.gen.go`: Legacy type snapshot + auto-migrate that copies all unchanged
     fields. Changed/new/removed fields get comments.
   - `migrate.go`: Post-migrate template with TODOs for each changed field. Oracle
     pre-populates the TODO with the change type (added, removed, type changed).

   **Propagation mode** (nested dependency changes):
   - `auto.gen.go`: Parent legacy type snapshot + auto-migrate that walks nested
     collections and calls leaf auto-migrate + post-migrate for each element.
   - `migrate.go`: Empty post-migrate template (parent-level logic usually not needed).

4. **Update `oracle migrate generate`** to:
   - Parse current `.oracle` files AND latest snapshot `.oracle` files
   - Run diff engine to classify changes
   - Determine generation mode per type (full/skeleton/propagation)
   - Generate appropriate files
   - Update `migrate.gen.go` with new registrations
   - Create new snapshot

### What to test — THIS IS CRITICAL

The diff engine and dependency tracking need comprehensive unit tests:

**Diff engine tests**:
- Field added: detects new field, classifies correctly
- Field removed: detects missing field
- Field type changed: same name, different type
- Multiple changes in one type
- No changes: types match
- Nested type reference: detects that parent needs migration when child changes
- Array of nested type: detects change in array element type
- Optional field added/removed
- Map field changes

**Dependency graph tests**:
- Direct dependency: Type A has field of Type B
- Transitive dependency: A → B → C, change C triggers A and B
- Shared dependency: A and B both reference C, change C triggers both
- No dependency: independent types don't affect each other
- Circular reference handling (shouldn't happen with Oracle schemas but be safe)
- Deep nesting (5+ levels)

**Generation mode tests**:
- Skeleton mode generates correct auto-migrate (copies unchanged fields)
- Skeleton mode post-migrate template has correct TODOs
- Propagation mode auto-migrate walks nested collections correctly
- Propagation mode calls both auto-migrate and post-migrate on nested elements
- Multiple nested types changed simultaneously → single parent migration
- Generated code compiles

**Integration tests**:
- Add a field to a type → `oracle migrate generate` → correct files generated
- Remove a field → correct files
- Change nested type → propagation to all parents
- Generated migration actually works (seed old data, run migration, verify)

### Acceptance criteria

- Diff engine has >90% branch coverage
- Dependency graph handles all nesting patterns from RFC scenarios 9-12
- Generated skeleton and propagation migrations compile and pass tests
- `oracle migrate generate` correctly handles schema changes

---

## Phase 7: Port Existing Migrations to New System

**Goal**: Port existing hand-written migrations (ranger, rack, task, device status
migrations) to the new `Migration` interface. Remove the deprecated `gorp.Migrator`.

**Scope**: `core/pkg/service/ranger/`, `core/pkg/service/rack/`,
`core/pkg/service/task/`, `core/pkg/service/device/`, `x/go/gorp/`

### What to build

1. **Port ranger's `migrateRangeGroups`**:
   - Wrap as `gorp.NewRawMigration("range_groups", s.migrateRangeGroups)`
   - Pass to `OpenEntryManager` alongside Oracle-generated migrations
   - The raw migration must come BEFORE the codec transition migration in the chain
     (ranger's migration reads msgpack-encoded entries)
   - Handle the dependency injection: ranger's migration accesses `s.cfg.Ontology`,
     `s.cfg.Group`, etc. Use a closure that captures the service config.

2. **Port status migrations** (rack, task, device):
   - These currently use existence-check patterns (not version-tracked)
   - Convert to `RawMigration` implementations
   - Ensure idempotency (they should be no-ops on second run)

3. **Remove deprecated `gorp.Migrator`**:
   - Delete old `Migrator` struct and `MigrationSpec`
   - Remove `Run` method
   - Clean up any remaining references

### What to test

- Ranger migration works correctly via `RawMigration` path
- Status migrations are idempotent
- No regression in ranger, rack, task, device behavior
- Old `gorp.Migrator` code is fully removed

### Acceptance criteria

- All existing migrations ported to new system
- `gorp.Migrator` deleted
- All tests pass

---

## Phase 8: Test Infrastructure & Migration Test Helpers

**Goal**: Build comprehensive migration test helpers that make writing migration tests
trivial. This phase can run in parallel with earlier phases after Phase 1 establishes
the interface.

**Scope**: `x/go/gorp/testutil/`

### What to build

1. **`testutil.TestMigration`** — the primary test helper:
   - Creates in-memory KV store
   - Seeds old entries encoded with appropriate codec
   - Runs migration
   - Asserts results match expected entries
   - Validates version counter
   - Returns Gomega-compatible errors

2. **`testutil.TestNestedMigration`** — for nested type migrations:
   - Tests auto-migrate + post-migrate as pure functions (no KV store)
   - Validates field copying and transform logic

3. **`testutil.TestMigrationChain`** — for multi-step migrations:
   - Seeds entries at v1
   - Runs full chain v1→v2→v3
   - Validates final state

4. **Gomega custom matchers** if needed:
   - `HaveMigratedTo(expectedEntries)` matcher for readable test assertions

### What to test

- Test helpers themselves are tested (meta-tests)
- Helpers work with TypedMigration, RawMigration
- Helpers correctly detect migration failures
- Helpers work with different key types (uuid, string, uint32, etc.)

### Acceptance criteria

- Test helpers are ergonomic and reduce migration test boilerplate
- Documented with examples
- Used by Phase 3 and Phase 6 tests

---

## Dependency Graph

```
Phase 1: gorp Migration Interface
    ↓
Phase 2: Oracle GorpMarshal/GorpUnmarshal Generation
    ↓
Phase 3: Oracle Migration Plugin (Codec Transition)
    ↓
Phase 4: Service Wiring & E2E Codec Transition
    ↓
Phase 5: Schema Snapshots & CI Check
    ↓
Phase 6: Schema Diff Engine & Skeleton/Propagation Mode
    ↓
Phase 7: Port Existing Migrations

Phase 8: Test Infrastructure (can start after Phase 1, used by Phase 3+)
```

## Key Design Decisions (from RFC discussion)

1. **Oracle = build-time code gen only**. Runtime migration execution lives in gorp.
2. **`Migration` is an interface** with `TypedMigration[I,O]`, `RawMigration`,
   `KeyMigration[K,E]` implementations.
3. **`EntryManager[K,E]` runs everything**. Single entry point. Key re-encoding is
   implicit. Version counter at `__gorp__//<Type>/version` (uint16).
4. **`GorpMarshal`/`GorpUnmarshal`** generated by `oracle sync` (go/types plugin),
   not the migration plugin.
5. **`vN/` sub-packages** per migration version. Migrations map
   `vN.TypeVN → v(N+1).TypeV(N+1)`. Never import parent service package.
6. **Migration files at `@go output` path** — never in `schemas/`.
7. **`oracle migrate regenerate` deferred** — not needed for MVP.
8. **Ranger port deferred to Phase 7** — validates `RawMigration` path but doesn't
   block codec transition.
