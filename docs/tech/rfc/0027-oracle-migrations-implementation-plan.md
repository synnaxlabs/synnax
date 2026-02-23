# Oracle Migration System — Phased Implementation Plan

Reference: [RFC 0027 - Oracle Migrations](./0027-oracle-migrations.md)

This document breaks the Oracle migration system into bounded, independently testable
phases. Each phase includes instructions detailed enough to hand to a Claude Code
session.

---

## Phase 1: gorp Infrastructure — DONE

**Status**: Complete. Merged on branch `sy-3823-gorp-tables`.

**What was built** (diverged from original plan — used `Codec[E]` + `Table[K,E]` instead
of `GorpMarshaler`/`GorpUnmarshaler` + `EntryManager`):

- **`Codec[E]` interface** (`x/go/gorp/codec.go`):
  ```go
  type Codec[E any] interface {
      Marshal(ctx context.Context, entry E) ([]byte, error)
      Unmarshal(ctx context.Context, data []byte) (E, error)
  }
  ```

- **`Table[K,E]`** (`x/go/gorp/table.go`): replaces `EntryManager`. Holds optional
  `Codec[E]`, provides codec-aware query builders (`NewCreate`, `NewRetrieve`,
  `NewUpdate`, `NewDelete`), `OpenNexter`, `Observe`, `Close`.

- **`OpenTable[K,E](ctx, TableConfig[E])`**: runs versioned `Migration` list
  sequentially with uint16 version tracking, then runs `migrateOldPrefixKeys` and
  `reEncodeKeys` automatically.

- **`Migration` interface** (`x/go/gorp/migrate.go`):
  ```go
  type Migration interface {
      Name() string
      Run(ctx context.Context, tx kv.Tx, cfg MigrationConfig) error
  }
  ```
  Implementations: `TypedMigration[I,O]` and `RawMigration`.

- **Codec threaded through** all builders, writers (`Writer.set()`), readers
  (`Reader.Get()`, `Iterator.Value()`), and observers (`Table.Observe()`).

- **Deleted** `GorpMarshaler`/`GorpUnmarshaler` interfaces (`x/go/gorp/marshal.go`)
  and standalone query builders (`NewCreate[K,E]()`, etc.).

- **All ~17 services** migrated from standalone builders to `table` methods.

- **All gorp and core tests pass**.

### Key files
- `x/go/gorp/codec.go` — `Codec[E]` interface
- `x/go/gorp/table.go` — `Table[K,E]`, `OpenTable`, `TableConfig`, `OpenNexter`
- `x/go/gorp/migrate.go` — `Migration`, `TypedMigration`, `RawMigration`, version
  tracking
- `x/go/gorp/reader.go` — codec-aware `Reader`, `Iterator`, `WrapReader`
- `x/go/gorp/writer.go` — codec-aware `Writer`
- `x/go/gorp/observe.go` — `Table.Observe()`, standalone `Observe[K,E]()`

---

## Phase 2: Oracle Codec Generation — DONE

**Status**: Complete. On branch `sy-3816-oracle-migrations`.

**What was built** (diverged from original plan — generates standalone `Codec[E]` structs
in `pb/` subpackage instead of `GorpMarshal`/`GorpUnmarshal` methods on parent type,
avoiding import cycles):

- **Oracle `go/marshal` plugin** (`oracle/plugin/go/marshal/marshal.go`):
  - Triggers on `@go marshal` annotation (not `@key`)
  - Generates `pb/codec.gen.go` with standalone codec struct
  - Codec wraps existing `ToPB`/`FromPB` translators + `proto.Marshal`/`Unmarshal`
  - Exported var: `var SchematicCodec gorp.Codec[schematic.Schematic] = schematicCodec{}`

- **`@go marshal` annotations** added to 14 `.oracle` schemas:
  arc, channel, device, group, label, lineplot, log, rack, ranger, schematic, table,
  task, user, workspace

- **14 `codec.gen.go` files generated** via `oracle sync`:
  ```
  core/pkg/service/{arc,device,lineplot,log,rack,ranger,schematic,table,task,user,workspace}/pb/codec.gen.go
  core/pkg/distribution/group/pb/codec.gen.go
  core/pkg/api/channel/pb/codec.gen.go
  x/go/label/pb/codec.gen.go
  ```

### What was NOT built (deferred)
- Oracle plugin tests (`oracle/plugin/go/marshal/marshal_test.go`) not updated to assert
  new trigger/output patterns

---

## Phase 2.5: Service Wiring — DONE

**Status**: Complete. On branch `sy-3816-oracle-migrations`.

This was originally part of Phase 4 in the old plan, but was done early since the
`Codec[E]` + `Table[K,E]` approach made it straightforward.

**What was built:**

- **`Codec` field** added to each service's `ServiceConfig` + `Override` method
- **Codecs wired in `core/pkg/service/layer.go`**: 12 services get their codec at
  construction (arcpb, devicepb, labelpb, lineplotpb, logpb, rackpb, rangerpb,
  schematicpb, tablepb, taskpb, userpb, workspacepb)
- **Codec wired in `core/pkg/distribution/layer.go`**: group gets `grouppb.GroupCodec`
- **All 13 `Observe` callers migrated**: ontology.go `OnChange` methods use
  `s.table.Observe()` instead of `gorp.Observe[K,E](s.DB)`
- **All 13 `OpenNexter` callers migrated**: ontology.go `OpenNexter` methods use
  `s.table.OpenNexter(ctx)` instead of `gorp.WrapReader[K,E](db).OpenNexter(ctx)`
- **Signal publishers migrated**: 8 service signal callers + group signals + driver
  service use codec-aware observables via `Observable` field on `GorpPublisherConfig`
  (`core/pkg/distribution/signals/gorp.go`)
- **Auth KV fix**: `layer.go` uses `auth.OpenKV(ctx, db)` instead of direct struct
  construction

- **All core tests pass** (60+ suites, 0 failures)

### Key files modified
- `core/pkg/service/layer.go` — 12 codec imports + wiring
- `core/pkg/distribution/layer.go` — group codec wiring
- `core/pkg/distribution/signals/gorp.go` — `Observable` field on `GorpPublisherConfig`
- 13× `*/ontology.go` — `OnChange` and `OpenNexter` migrated
- 8× `*/service.go` — signal publisher observable wiring
- `core/pkg/distribution/group/service.go` — added `Observe()` method
- `core/pkg/service/task/service.go` — added `Observe()` method
- `core/pkg/service/driver/service.go` — uses `cfg.Task.Observe()`

---

## Phase 3: Codec Transition Migration (msgpack → protobuf) — NOT STARTED

**Goal**: Build and wire the actual data migration that converts existing msgpack-encoded
entries to protobuf at server startup. **This is the critical missing piece** — without
it, a server with pre-existing data will crash because the protobuf codec can't decode
msgpack bytes.

**Scope**: `x/go/gorp/`, service `OpenTable` call sites

### What to build

1. **`CodecTransitionMigration[K,E]`** (`x/go/gorp/migrate.go`):

   A new `Migration` implementation purpose-built for codec transitions:

   ```go
   func NewCodecTransition[K Key, E Entry[K]](name string, codec Codec[E]) Migration
   ```

   `Run()` behavior:
   - Iterate all entries under `cfg.Prefix`
   - Decode each with `cfg.Codec` (the DB's default msgpack codec)
   - Re-encode with `codec` (the target protobuf codec)
   - Write back under the same key

   This is simpler than `TypedMigration` because the type doesn't change — only the
   encoding format changes.

2. **Wire into each service's `OpenTable`** — pass codec transition as a migration:

   ```go
   table, err := gorp.OpenTable[uuid.UUID, Schematic](ctx, gorp.TableConfig[Schematic]{
       DB:    cfg.DB,
       Codec: cfg.Codec,
       Migrations: []gorp.Migration{
           gorp.NewCodecTransition[uuid.UUID, Schematic]("msgpack_to_protobuf", cfg.Codec),
       },
   })
   ```

   Do this for all 13 codec-ed services + group.

3. **Handle the channel type**: channel is in the distribution layer
   (`core/pkg/distribution/channel/`). Check if it has a table with codec and needs the
   migration too.

### What to test

- Seed msgpack-encoded entries, run `CodecTransitionMigration`, verify protobuf decoding
- Second run is a no-op (version counter already incremented)
- Empty DB — migration runs without error
- Round-trip: write with protobuf codec → read back → data matches
- Full server boot with pre-existing msgpack data → all services start correctly

### Acceptance criteria

- `cd x/go/gorp && ginkgo` passes with new codec transition tests
- `cd core && ginkgo -r` passes (all core tests)
- Server boots cleanly against a DB with pre-existing msgpack-encoded data

---

## Phase 4: Oracle Migration Plugin — Codec Transition (Full Mode) — NOT STARTED

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
     - Enumerate all Oracle-managed gorp types (types with `@go marshal` + `@pb`)
     - For each type, generate `migrations/v1/` sub-package with:
       - `auto.gen.go`: Legacy type snapshot (identical fields, msgpack tags) +
         `AutoMigrateV1ToV2` function (identity transform copying all fields)
       - `migrate.go`: `PostMigrateV1ToV2` function (empty body, codec transition only)
     - Generate `migrations/v2/` sub-package with:
       - `auto.gen.go`: Current type snapshot with protobuf codec
         (no auto-migrate yet — v2 is the current version)
       - `pb/`: Snapshotted protobuf definitions
     - Generate `migrations/migrate.gen.go`: `All()` function returning the ordered
       migration list with a single `gorp.NewTypedMigration` call

3. **Legacy type snapshot generation**: The `v1/auto.gen.go` legacy type must be a
   faithful snapshot of the current struct — same fields, same types. For pre-transition
   types, use `msgpack` struct tags (no codec). The migration runner will decode these
   using the generic msgpack codec.

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
- Use the `gorp/testutil` helpers from Phase 8 to test each generated migration

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

## Phase 5: Service Wiring & End-to-End Codec Transition — NOT STARTED

**Goal**: Wire up the generated migrations into each service's `OpenTable` call. Run the
full codec transition end-to-end.

**Scope**: `core/pkg/service/*/`, `core/pkg/distribution/*/`

### What to build

1. **Update each service's `OpenService`** to pass migrations to `OpenTable`:

   ```go
   table, err := gorp.OpenTable[uuid.UUID, Schematic](ctx, gorp.TableConfig[Schematic]{
       DB:         cfg.DB,
       Codec:      cfg.Codec,
       Migrations: migrations.All(),
   })
   ```

   Do this for all ~21 services that have Oracle-managed gorp types.

2. **Remove old `gorp.Migrator` usage** from any service that previously used it
   (currently only ranger uses it). For ranger, keep its existing `migrateRangeGroups`
   as a `RawMigration` passed before the codec transition in the migration chain.
   Actually — defer ranger port to Phase 7. For now, ranger keeps its old migrator
   AND the new `OpenTable` pattern.

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
- `cd core && ginkgo -r` passes

---

## Phase 6: Schema Snapshots & `oracle migrate check` (CI Enforcement) — NOT STARTED

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
   (not field-level yet — that comes in Phase 7). If any `.oracle` file content differs
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

## Phase 7: Schema Diff Engine & Skeleton/Propagation Mode Generation — NOT STARTED

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

## Phase 8: Port Existing Migrations to New System — NOT STARTED

**Goal**: Port existing hand-written migrations (ranger, rack, task, device status
migrations) to the new `Migration` interface. Remove the deprecated `gorp.Migrator`.

**Scope**: `core/pkg/service/ranger/`, `core/pkg/service/rack/`,
`core/pkg/service/task/`, `core/pkg/service/device/`, `x/go/gorp/`

### What to build

1. **Port ranger's `migrateRangeGroups`**:
   - Wrap as `gorp.NewRawMigration("range_groups", s.migrateRangeGroups)`
   - Pass to `OpenTable` alongside Oracle-generated migrations
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

## Phase 9: Test Infrastructure & Migration Test Helpers — NOT STARTED

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
- Used by Phase 4 and Phase 7 tests

---

## Dependency Graph

```
Phase 1: gorp Infrastructure ...................... DONE
    ↓
Phase 2: Oracle Codec Generation .................. DONE
    ↓
Phase 2.5: Service Wiring ......................... DONE
    ↓
Phase 3: Codec Transition Migration ............... NEXT  ← critical path
    ↓
Phase 4: Oracle Migration Plugin (Full Mode) ...... NOT STARTED
    ↓
Phase 5: Service Wiring & E2E Codec Transition .... NOT STARTED
    ↓
Phase 6: Schema Snapshots & CI Check .............. NOT STARTED
    ↓
Phase 7: Schema Diff Engine & Skeleton/Propagation  NOT STARTED
    ↓
Phase 8: Port Existing Migrations ................. NOT STARTED

Phase 9: Test Infrastructure (can start anytime after Phase 1)
```

## Key Design Decisions

1. **`Codec[E]` interface over `GorpMarshaler`/`GorpUnmarshaler`**: Avoids import cycles
   by keeping codec in `pb/` subpackage, injected via `TableConfig`. Cleaner separation.
2. **`Table[K,E]` over `EntryManager[K,E]`**: Single struct owns codec + DB + query
   builders. Services hold `table` field, call `table.NewCreate()`, etc.
3. **`Migration` is an interface** with `TypedMigration[I,O]`, `RawMigration`, and
   (soon) `CodecTransitionMigration[K,E]` implementations.
4. **`OpenTable` runs everything**: versioned migrations → old prefix migration → key
   re-encoding. Version counter at `__gorp_migration__//{TypeName}` (uint16).
5. **Oracle = build-time code gen only**. Runtime migration execution lives in gorp.
6. **`@go marshal` annotation** triggers codec generation (not `@key`).
7. **Standalone codec structs** in `pb/` subpackage, exported as
   `var XxxCodec gorp.Codec[parent.Xxx]`.
8. **Ranger port deferred to Phase 8** — validates `RawMigration` path but doesn't
   block codec transition.
