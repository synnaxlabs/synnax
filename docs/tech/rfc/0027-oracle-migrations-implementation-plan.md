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

  Implementations: `TypedMigration[I,O]`, `RawMigration`, and
  `CodecTransitionMigration[K,E]`.

- **`NewCodecTransition[K,E]`** (`x/go/gorp/migrate.go`): purpose-built migration that
  re-encodes all entries from the DB's default codec (msgpack) to a target `Codec[E]`
  (binary). Originally planned as Phase 3 but built alongside the infrastructure.

- **Codec threaded through** all builders, writers (`Writer.set()`), readers
  (`Reader.Get()`, `Iterator.Value()`), and observers (`Table.Observe()`).

- **Deleted** `GorpMarshaler`/`GorpUnmarshaler` interfaces (`x/go/gorp/marshal.go`) and
  standalone query builders (`NewCreate[K,E]()`, etc.).

- **All ~17 services** migrated from standalone builders to `table` methods.

- **All gorp and core tests pass**.

### Key files

- `x/go/gorp/codec.go` — `Codec[E]` interface
- `x/go/gorp/table.go` — `Table[K,E]`, `OpenTable`, `TableConfig`, `OpenNexter`
- `x/go/gorp/migrate.go` — `Migration`, `TypedMigration`, `RawMigration`,
  `CodecTransitionMigration`, version tracking
- `x/go/gorp/reader.go` — codec-aware `Reader`, `Iterator`, `WrapReader`
- `x/go/gorp/writer.go` — codec-aware `Writer`
- `x/go/gorp/observe.go` — `Table.Observe()`, standalone `Observe[K,E]()`

---

## Phase 2: Oracle Codec Generation — DONE

**Status**: Complete on branch `sy-3816-oracle-migrations`. Codec files now live in
parent packages (moved from `pb/` in Phase 2.1).

**What was built** (diverged from original plan — generates standalone `Codec[E]`
structs instead of `GorpMarshal`/`GorpUnmarshal` methods):

- **Oracle `go/marshal` plugin** (`oracle/plugin/go/marshal/marshal.go`):
  - Triggers on `@go marshal` annotation (not `@key`)
  - Generates `codec.gen.go` in the parent package (same package as the type)
  - Codec uses direct binary encoding (BigEndian, length-prefixed fields, JSON for
    untyped fields) — no protobuf translation layer
  - Exported var: `var SchematicCodec gorp.Codec[Schematic] = schematicCodec{}`

- **`@go marshal` annotations** added to 15 `.oracle` schemas: arc, channel, device,
  group, label, lineplot, log, ontology, rack, ranger, schematic, table, task, user,
  workspace

- **15 `codec.gen.go` files generated** via `oracle sync` in parent packages

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
  construction (arc, device, label, lineplot, log, rack, ranger, schematic, table, task,
  user, workspace — imported directly from parent packages)
- **Codec wired in `core/pkg/distribution/layer.go`**: group gets `group.GroupCodec`,
  ontology gets `ontology.RelationshipCodec` and `ontology.ResourceCodec`
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

- `core/pkg/service/layer.go` — 12 codec imports + wiring (parent package imports)
- `core/pkg/distribution/layer.go` — group + ontology codec wiring (parent package
  imports)
- `core/pkg/distribution/signals/gorp.go` — `Observable` field on `GorpPublisherConfig`
- 13× `*/ontology.go` — `OnChange` and `OpenNexter` migrated
- 8× `*/service.go` — signal publisher observable wiring
- `core/pkg/distribution/group/service.go` — added `Observe()` method
- `core/pkg/service/task/service.go` — added `Observe()` method
- `core/pkg/service/driver/service.go` — uses `cfg.Task.Observe()`

---

## Phase 2.1: Move `codec.gen.go` to Parent Packages — DONE

**Status**: Complete on branch `sy-3816-oracle-migrations`.

**What was built:**

- **Oracle `go/marshal` plugin** (`oracle/plugin/go/marshal/marshal.go`) updated:
  - Output path changed from `pb/` subdirectory to parent package (`goPath` directly)
  - Package declaration uses `naming.DerivePackageName(goPath)` instead of `"pb"`
  - Type references are unqualified (e.g., `Schematic` not `schematic.Schematic`) since
    codec is now in the same package as the type
  - External import aliases derived from `packageName` context instead of `"pb"`
  - Removed `ParentAlias`, `ParentPath`, `ParentImportPath` from template data structs

- **15 `codec.gen.go` files regenerated** via `oracle sync` in parent packages:

  ```
  core/pkg/service/{arc,device,lineplot,log,rack,ranger,schematic,table,task,user,workspace}/codec.gen.go
  core/pkg/distribution/{group,ontology}/codec.gen.go
  core/pkg/api/channel/codec.gen.go
  x/go/label/codec.gen.go
  ```

- **15 old `pb/codec.gen.go` files deleted** (pb/ directories retained for protobuf
  transport files)

- **Layer imports updated**:
  - `core/pkg/service/layer.go` — removed 12 `pb` import aliases, codec refs now use
    parent package directly (e.g., `schematic.SchematicCodec`). `xlabel` alias for
    `x/label` to avoid collision with `service/label`.
  - `core/pkg/distribution/layer.go` — removed `grouppb`/`ontologypb` imports, codec
    refs now use `group.GroupCodec`, `ontology.RelationshipCodec`,
    `ontology.ResourceCodec`

- **All tests pass**: `go build ./...`, `cd x/go && ginkgo -r` (60 suites),
  `cd core && ginkgo -r` (77 suites)

---

## Phase 3: Codec Transition Migration (msgpack → binary) — DONE

**Status**: Complete. Built as part of Phase 1 on branch `sy-3823-gorp-tables`.

Originally planned as a separate phase, `NewCodecTransition[K,E]` was implemented
directly in `x/go/gorp/migrate.go` alongside the other `Migration` implementations.

**What was built:**

- **`codecTransitionMigration[K,E]`** struct (`x/go/gorp/migrate.go:187`):
  - `NewCodecTransition[K Key, E Entry[K]](name string, codec Codec[E]) Migration`
  - Iterates all entries under `cfg.Prefix`
  - Decodes each with `cfg.Codec` (DB's default msgpack codec)
  - Re-encodes with the target `Codec[E]` (binary)
  - Writes back under the same key
- **Tests** in `x/go/gorp/migrate_test.go`

---

## Phase 4: Oracle Migration Plugin — Codec Transition (Full Mode) — DONE

**Status**: Complete. On branch `sy-3824-oracle-auto-migration-plugin`.

**What was built** (diverged from original plan — uses `NewCodecTransition` directly
instead of `NewTypedMigration` with codec params, which is simpler and correct for the
codec-only MVP):

- **Oracle `go/migrate` plugin** (`oracle/plugin/go/migrate/migrate.go`):
  - Triggers on `@go migrate` annotation (requires `@key` field and `@go output` path)
  - Generates two files per type:
    1. `migrations/v1/v1.gen.go` — Legacy type snapshot with msgpack tags, `GorpKey()`,
       `SetOptions()` methods
    2. `migrate.gen.go` — Migration registration function in the parent package

- **`oracle migrate generate` CLI command** (`oracle/cmd/migrate.go`):
  - Discovers schemas from `schemas/*.oracle`
  - Runs `go/migrate` plugin (filtered from full plugin set)
  - Updates license headers, runs `gofmt`

- **Generated `migrate.gen.go` pattern** (per service):

  ```go
  package schematic

  func SchematicMigrations(codec gorp.Codec[Schematic]) []gorp.Migration {
      return []gorp.Migration{
          gorp.NewCodecTransition[Key, Schematic]("msgpack_to_binary", codec),
      }
  }
  ```

  Note: This differs from the RFC's `All()` pattern using `NewTypedMigration` with
  explicit codec params. The generated function takes the codec as a parameter and uses
  `NewCodecTransition` directly. This is the correct approach for the MVP (codec-only
  migration). The RFC's `NewTypedMigration` with `inputCodec`/`outputCodec` will be
  needed in Phase 7 when schema migrations use frozen version-specific codecs.

- **11 services + group + ontology** have generated `migrate.gen.go` + `v1/v1.gen.go`:
  arc, device, lineplot, log, rack, ranger, schematic, table, task, user, workspace,
  group (distribution), ontology (inline)

- **Tests** in `oracle/plugin/go/migrate/migrate_test.go`

### What was NOT built (deferred to Phase 4.5)

- Remaining 8 services not yet wired (see Phase 4.5)
- ~~Migration name still says `"msgpack_to_protobuf"` in some generated files~~ — FIXED

---

## Phase 4.5: Complete Service Wiring — IN PROGRESS

**Status**: Partially complete on branch `sy-3824-oracle-auto-migration-plugin`.

### What's done

13 services have full `OpenTable` wiring with `Codec` + `Migrations`:

| Service                 | OpenTable wiring                                                   |
| ----------------------- | ------------------------------------------------------------------ |
| arc                     | `ArcMigrations(cfg.Codec)`                                         |
| device                  | `DeviceMigrations(cfg.Codec)`                                      |
| lineplot                | `LinePlotMigrations(cfg.Codec)`                                    |
| log                     | `LogMigrations(cfg.Codec)`                                         |
| rack                    | `RackMigrations(cfg.Codec)`                                        |
| ranger                  | `append(RangeMigrations(cfg.Codec), newRangeGroupsMigration(cfg))` |
| schematic               | `SchematicMigrations(cfg.Codec)`                                   |
| table                   | `TableMigrations(cfg.Codec)`                                       |
| task                    | `TaskMigrations(cfg.Codec)`                                        |
| user                    | `UserMigrations(cfg.Codec)`                                        |
| workspace               | `WorkspaceMigrations(cfg.Codec)`                                   |
| group (distribution)    | `GroupMigrations(cfg.Codec)`                                       |
| ontology (distribution) | inline `NewCodecTransition` calls                                  |

### Test fixes applied

- **Rack tests** (`rack_test.go`): Added `Codec: rack.RackCodec` to 3 `OpenService`
  calls in migration tests. Added `Count()` method to `rack/retrieve.go` so tests can
  count entries through the codec-aware table (standalone `gorp.NewRetrieve` can't
  decode binary-encoded data after codec transition).

- **Ranger migration** (`ranger/migrate.go`): Added `ExcludeFieldData(true)` to ontology
  query that traverses to range children. The `rangeGroupsMigration` runs during
  `OpenTable`, before the ranger service registers with the ontology. Without
  `ExcludeFieldData(true)`, the ontology's `retrieveResource` panics because the "range"
  service isn't registered yet. The migration only needs child IDs (not full entity
  data), so `ExcludeFieldData(true)` is the correct fix.

- **Ranger service** (`ranger/service.go`): Wired `newRangeGroupsMigration(cfg)` into
  `OpenTable` migrations alongside the generated codec transition.

- **Ranger test** (`ranger/migrate_test.go`): Added `Codec: ranger.RangeCodec`.

### Completed in this session

- **Fixed migration name**: Updated `oracle/plugin/go/migrate/migrate.go` template from
  `"msgpack_to_protobuf"` to `"msgpack_to_binary"`. Updated all 13 existing
  `migrate.gen.go` files, `oracle/plugin/go/migrate/migrate_test.go`, and
  `core/pkg/distribution/ontology/ontology.go` inline calls.

- **Removed `HasPB` check** from `oracle/plugin/go/marshal/marshal.go`: The marshal
  plugin previously required `@pb` annotation to generate codecs. Removed this gate so
  types with only `@go marshal` (no `@pb`) can get binary codecs. Rebuilt oracle via
  `./oracle/install.sh`.

- **Added `@go marshal` to 3 schemas**: `channel.oracle`, `alias.oracle`, `kv.oracle`.
  Generated `codec.gen.go` for each (Channel, Alias, Pair).

- **Added `@go marshal` + `@go migrate` + `@go output` to 2 schemas**: `role.oracle`,
  `view.oracle`. Generated `codec.gen.go`, `types.gen.go`, `migrate.gen.go`, and
  `migrations/v1/v1.gen.go` for each (Role, View).

- **Resolved type conflicts**: Removed hand-written `Role` struct from `role/role.go`
  and `View` struct from `view/view.go` in favor of oracle-generated `types.gen.go`.
  Kept `GorpKey()`, `SetOptions()`, `OntologyID()` methods.

- **All tests pass**: view (13/13), role (29/29), ranger (20/20), RBAC (15/15),
  policy (30/30).

### What's remaining

5 services still need `@go marshal` + `@go migrate`, `oracle sync`, `oracle migrate
generate`, and `OpenTable` + `layer.go` wiring:

| Service                    | File                    | Schema status                                |
| -------------------------- | ----------------------- | -------------------------------------------- |
| `access/rbac/policy`       | `policy/service.go:63`  | Needs `@go marshal` + `@go migrate`          |
| `auth` (SecureCredentials) | `auth/kv.go:32`         | Needs `@go marshal` + `@go migrate`          |
| `schematic/symbol`         | `symbol/service.go:83`  | Needs `@go marshal` + `@go migrate`          |
| `ranger/alias`             | `alias/service.go:84`   | Has `@go marshal`, needs `@go migrate`       |
| `ranger/kv` (Pair)         | `kv/service.go:64`      | Has `@go marshal`, needs `@go migrate`       |

3 services have `codec.gen.go` generated but still need `@go migrate`, `oracle migrate
generate`, and `OpenTable` + `layer.go` wiring:

| Service                    | File                      | Schema status                          |
| -------------------------- | ------------------------- | -------------------------------------- |
| `distribution/channel`     | `channel/service.go:97`   | Has `@go marshal`, needs `@go migrate` |
| `access/rbac/role`         | `role/service.go:75`      | Has both, needs `OpenTable` wiring     |
| `view`                     | `view/service.go:86`      | Has both, needs `OpenTable` wiring     |

### What to build

1. **Add `@go migrate` annotations** to Channel, Alias, Pair schemas (already have
   `@go marshal`).

2. **Add `@go marshal` + `@go migrate` annotations** to Policy, Symbol,
   SecureCredentials schemas.

3. **Run `oracle sync`** + **`oracle migrate generate`** for all remaining types.

4. **Wire `Codec` + `Migrations` into `OpenTable`** calls for all 8 services.

5. **Wire codecs in `layer.go`** — add codec imports and pass to ServiceConfig.

### Acceptance criteria

- All services with gorp tables have Codec + Migrations wired
- `cd core && ginkgo -r` passes
- `go build ./...` compiles cleanly

---

## Phase 5: Delete Deprecated `gorp.Migrator` — DONE

**Status**: Complete on branch `sy-3824-oracle-auto-migration-plugin`.

**What was built:**

- **Deleted deprecated types** from `x/go/gorp/migrate.go`:
  - `Migrator` struct
  - `MigrationSpec` struct
  - `Migrator.Run()` method
  - `ErrMigrationCountExceeded` var
  - Removed unused `query` import

- **Deleted deprecated tests** from `x/go/gorp/migrate_test.go`:
  - Entire `GorpRunner` describe block (basic execution, error handling, force flag,
    version tracking, transaction behavior)

- **Verified no remaining references** to old migrator pattern in production code.

- **All tests pass**: gorp (118/118), core builds clean.

---

## Phase 6: Schema Snapshots & `oracle migrate check` (CI Enforcement) — DONE

**Status**: Complete on branch `sy-3824-oracle-auto-migration-plugin`.

**What was built:**

- **`oracle/snapshot/` package** (`oracle/snapshot/snapshot.go`):
  - `LatestVersion(snapshotsDir)` — finds highest `v<N>` snapshot version (0 if none
    exist or directory missing)
  - `Create(schemasDir, snapshotsDir, version)` — copies all `.oracle` files into
    `schemas/.snapshots/v<N>/` (flat copy, byte-for-byte)
  - `Check(schemasDir, snapshotsDir)` — compares current `.oracle` files against latest
    snapshot; returns nil if identical, descriptive error listing added/removed/modified
    files if different

- **Updated `oracle migrate generate`** (`oracle/cmd/migrate.go`):
  - After syncing migration files, calls `snapshot.LatestVersion()` then
    `snapshot.Create()` at version+1
  - Prints styled confirmation: "snapshot v<N> created"

- **New `oracle migrate check` CLI command** (`oracle/cmd/migrate.go`):
  - Subcommand of `migrateCmd` (invoked as `oracle migrate check`)
  - Calls `snapshot.Check(schemasDir, snapshotsDir)`
  - Exit 0 + success message when schemas match latest snapshot
  - Exit 1 + actionable error: "schema changed but no migration generated. Run
    'oracle migrate generate' and commit the result." with per-file diff listing

- **`printSnapshotCreated` helper** added to `oracle/cmd/style.go`

- **Fixed pre-existing marshal test failures** (`oracle/plugin/go/marshal/marshal_test.go`):
  - `"package pb"` → `"package test"` (codec now generates in parent package per
    Phase 2.1)
  - `"marshaltest"` → `"marshalType"` (recursive helpers are in-package, no cross-package
    prefix)

- **19 Ginkgo tests** in `oracle/snapshot/snapshot_test.go`:
  - `LatestVersion`: nonexistent dir, empty dir, single version, multiple versions,
    non-version entries ignored
  - `Create`: file copying, directory auto-creation, byte preservation, `.oracle`-only
    filtering, empty schemas
  - `Check`: matching schemas, no snapshots error, modified content detection, added
    file detection, removed file detection, latest-version comparison
  - Integration: create→check pass, modify→check fail, re-snapshot→check pass

- **All tests pass**: snapshot (19/19), marshal (8/8), full oracle suite (35 suites)

### Key files

- `oracle/snapshot/snapshot.go` — core snapshot logic
- `oracle/snapshot/snapshot_test.go` — 19 Ginkgo tests
- `oracle/snapshot/snapshot_suite_test.go` — test suite bootstrap
- `oracle/cmd/migrate.go` — `migrateCheckCmd` + snapshot creation in generate
- `oracle/cmd/style.go` — `printSnapshotCreated` output helper

### Design decisions

- **Snapshot directory**: `schemas/.snapshots/v<N>/` at repo root
- **Version = sequential integer**: determined by counting existing snapshot dirs
- **File-level diffing only**: byte-for-byte `.oracle` comparison (Phase 7 adds
  field-level)
- **Always snapshot on generate**: every `oracle migrate generate` creates a new version
- **Version encoded in directory name**: no separate metadata file needed

---

## Phase 7: Schema Diff Engine & Skeleton/Propagation Mode Generation — NOT STARTED

**Goal**: Build the schema diff engine that compares two `resolution.Table`s
field-by-field and generates skeleton mode (direct schema changes) and propagation mode
(nested dependency changes) migrations. This is the most complex phase and requires
thorough unit testing.

**Scope**: `oracle/plugin/migrate/`, `x/go/gorp/migrate.go`

### What to build

1. **Add `inputCodec`/`outputCodec` to `TypedMigration`** (`x/go/gorp/migrate.go`):

   The current `NewTypedMigration[I,O]` takes only `name`, `auto`, and `post`. For
   post-transition schema migrations (v2→v3, v3→v4, etc.), each step needs to decode
   with the previous version's frozen binary codec and encode with the new version's
   frozen binary codec. Add optional codec parameters:

   ```go
   func NewTypedMigration[I, O any](
       name string,
       inputCodec Codec[I],   // nil → use MigrationConfig.Codec (msgpack)
       outputCodec Codec[O],  // nil → use MigrationConfig.Codec (msgpack)
       auto AutoMigrateFunc[I, O],
       post PostMigrateFunc[I, O],
   ) Migration
   ```

   When `inputCodec` is nil, decode using `cfg.Codec` (DB's msgpack). When `outputCodec`
   is nil, encode using `cfg.Codec`. This preserves backward compatibility — existing
   `TypedMigration` callers in tests just pass nil for both.

2. **Schema diff engine** (`oracle/plugin/migrate/diff/`):
   - Compare two `resolution.Table`s (old snapshot vs. current)
   - Classify changes per type:
     - **Unchanged**: All fields match by name and type
     - **Field added**: New field name not in old type
     - **Field removed**: Old field name not in new type
     - **Field type changed**: Same field name, different type
   - No rename detection (shows as remove + add)
   - Output: `[]TypeDiff` with per-field change classification

3. **Dependency graph** (`oracle/plugin/migrate/deps/`):
   - Build a directed graph of type dependencies from `resolution.Table`
   - Track which gorp entry types transitively contain which nested types
   - When a nested type changes, identify all affected gorp entries
   - Use Oracle's existing `TopologicalSort()` for ordering

4. **Generation modes**:

   **Skeleton mode** (direct schema changes):
   - `auto.gen.go`: Legacy type snapshot + auto-migrate that copies all unchanged
     fields. Changed/new/removed fields get comments.
   - `migrate.go`: Post-migrate template with TODOs for each changed field. Oracle
     pre-populates the TODO with the change type (added, removed, type changed).

   **Propagation mode** (nested dependency changes):
   - `auto.gen.go`: Parent legacy type snapshot + auto-migrate that walks nested
     collections and calls leaf auto-migrate + post-migrate for each element.
   - `migrate.go`: Empty post-migrate template (parent-level logic usually not needed).

5. **Update `oracle migrate generate`** to:
   - Parse current `.oracle` files AND latest snapshot `.oracle` files
   - Run diff engine to classify changes
   - Determine generation mode per type (full/skeleton/propagation)
   - Generate appropriate files
   - Update `migrate.gen.go` with new registrations (now using `NewTypedMigration` with
     frozen codecs for post-transition migrations)
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

## Phase 8: Test Infrastructure & Migration Test Helpers — NOT STARTED

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
- Helpers work with TypedMigration, RawMigration, CodecTransitionMigration
- Helpers correctly detect migration failures
- Helpers work with different key types (uuid, string, uint32, etc.)

### Acceptance criteria

- Test helpers are ergonomic and reduce migration test boilerplate
- Documented with examples
- Used by Phase 7 tests

---

## Dependency Graph

```
Phase 1: gorp Infrastructure ...................... DONE (sy-3823)
    ↓
Phase 2: Oracle Codec Generation .................. DONE (sy-3816)
    ↓
Phase 2.5: Service Wiring ......................... DONE (sy-3816)
    ↓
Phase 2.1: Move codec.gen.go to Parent Packages ... DONE (sy-3816)
    ↓
Phase 3: Codec Transition Migration ............... DONE (absorbed into Phase 1)
    ↓
Phase 4: Oracle Migration Plugin (Full Mode) ...... DONE (sy-3824)
    ↓
Phase 4.5: Complete Service Wiring ................ IN PROGRESS (sy-3824)
    ↓
Phase 5: Delete Deprecated gorp.Migrator .......... DONE (sy-3824)
    ↓
Phase 6: Schema Snapshots & CI Check .............. DONE (sy-3824)
    ↓
Phase 7: Schema Diff Engine & Skeleton/Propagation  NOT STARTED             ← NEXT
    ↓                                               (includes TypedMigration codec params)

Phase 8: Test Infrastructure (can start anytime after Phase 1)
```

## Key Design Decisions

1. **`Codec[E]` interface over `GorpMarshaler`/`GorpUnmarshaler`**: Codec injected into
   `Table` via `TableConfig`. Currently generated in `pb/` subdirectory (see decision
   #7).
2. **`Table[K,E]` over `EntryManager[K,E]`**: Single struct owns codec + DB + query
   builders. Services hold `table` field, call `table.NewCreate()`, etc.
3. **`Migration` is an interface** with three implementations: `TypedMigration[I,O]`,
   `RawMigration`, and `CodecTransitionMigration[K,E]`.
4. **`OpenTable` runs everything**: versioned migrations → old prefix migration → key
   re-encoding. Version counter at `__gorp_migration__//{TypeName}` (uint16).
5. **Oracle = build-time code gen only**. Runtime migration execution lives in gorp.
6. **`@go marshal` annotation** triggers codec generation (not `@key`). `@go migrate`
   annotation triggers migration file generation.
7. **Codec structs in parent package (Phase 2.1 complete)** — `codec.gen.go` lives in
   the parent package alongside `types.gen.go`, where it semantically belongs. `pb/` is
   reserved for protobuf transport (`ToPB`/`FromPB`, `.pb.go`).
8. **Ranger migration already ported** — `rangeGroupsMigration` implements
   `gorp.Migration` directly and is passed to `OpenTable` alongside the generated codec
   transition. Old `gorp.Migrator` still exists but is deprecated (removed in Phase 5).
9. **Direct binary encoding over protobuf** — Oracle-generated codecs use positional
   binary encoding (BigEndian, length-prefixed) instead of protobuf. Eliminates the
   `ToPB`/`FromPB` translation layer for storage. `pb/` package retains protobuf for
   transport (gRPC/HTTP). Each migration version will freeze a `codec.gen.go` snapshot.
10. **`TypedMigration` codec params deferred to Phase 7** — The current
    `NewTypedMigration` takes only `name`, `auto`, and `post`. Explicit
    `inputCodec`/`outputCodec` parameters (nil → fallback to msgpack) will be added when
    schema migrations need version-specific frozen codecs. For the MVP codec transition,
    `NewCodecTransition` handles everything.
11. **Generated migration function pattern** —
    `{Type}Migrations(codec Codec[T]) []Migration` rather than `All() []Migration`. The
    codec is passed as a parameter to avoid circular imports and to allow
    `NewCodecTransition` to use it directly.
12. **Migrations run before service registration** — `OpenTable` runs migrations during
    table construction, which happens BEFORE the service registers with the ontology.
    Any migration that uses the ontology must call `ExcludeFieldData(true)` on queries
    that traverse to the service's own type, since `retrieveResource` panics when the
    service isn't registered. This was encountered with `rangeGroupsMigration` — the fix
    was adding `ExcludeFieldData(true)` to the child-range traversal query. The
    migration only needs the IDs (loaded from the KV store separately), not the full
    entity data.
13. **Hand-written migrations appended to generated list** — Services with custom
    migrations (e.g., ranger's `rangeGroupsMigration`) append them after the generated
    codec transition:
    `append(RangeMigrations(cfg.Codec), newRangeGroupsMigration(cfg))`. Migration
    ordering matters: codec transition runs first (converting msgpack → binary), then
    custom migrations operate on binary-encoded data.
14. **Tests must provide Codec** — Any test that calls `OpenService` with a codec-aware
    service must pass the `Codec` field. Without it, `cfg.Codec` is nil, and
    `NewCodecTransition(codec)` panics when the migration tries to marshal. Tests that
    count or retrieve entries after codec transition must use the table's codec-aware
    methods (e.g., `svc.NewRetrieve().Count()`) rather than standalone
    `gorp.NewRetrieve[K,E]().Count()` which uses msgpack and can't decode binary data.
