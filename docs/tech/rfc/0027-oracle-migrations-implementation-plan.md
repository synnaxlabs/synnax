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

- **All tests pass**: view (13/13), role (29/29), ranger (20/20), RBAC (15/15), policy
  (30/30).

### What's remaining

5 services still need `@go marshal` + `@go migrate`, `oracle sync`,
`oracle migrate generate`, and `OpenTable` + `layer.go` wiring:

| Service                    | File                   | Schema status                          |
| -------------------------- | ---------------------- | -------------------------------------- |
| `access/rbac/policy`       | `policy/service.go:63` | Needs `@go marshal` + `@go migrate`    |
| `auth` (SecureCredentials) | `auth/kv.go:32`        | Needs `@go marshal` + `@go migrate`    |
| `schematic/symbol`         | `symbol/service.go:83` | Needs `@go marshal` + `@go migrate`    |
| `ranger/alias`             | `alias/service.go:84`  | Has `@go marshal`, needs `@go migrate` |
| `ranger/kv` (Pair)         | `kv/service.go:64`     | Has `@go marshal`, needs `@go migrate` |

3 services have `codec.gen.go` generated but still need `@go migrate`,
`oracle migrate generate`, and `OpenTable` + `layer.go` wiring:

| Service                | File                    | Schema status                          |
| ---------------------- | ----------------------- | -------------------------------------- |
| `distribution/channel` | `channel/service.go:97` | Has `@go marshal`, needs `@go migrate` |
| `access/rbac/role`     | `role/service.go:75`    | Has both, needs `OpenTable` wiring     |
| `view`                 | `view/service.go:86`    | Has both, needs `OpenTable` wiring     |

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
  - Exit 1 + actionable error: "schema changed but no migration generated. Run 'oracle
    migrate generate' and commit the result." with per-file diff listing

- **`printSnapshotCreated` helper** added to `oracle/cmd/style.go`

- **Fixed pre-existing marshal test failures**
  (`oracle/plugin/go/marshal/marshal_test.go`):
  - `"package pb"` → `"package test"` (codec now generates in parent package per Phase
    2.1)
  - `"marshaltest"` → `"marshalType"` (recursive helpers are in-package, no
    cross-package prefix)

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

## Phase 7: Schema Diff Engine & Skeleton/Propagation Mode Generation — DONE

**Status**: Complete on branch `sy-3824-oracle-auto-migration-plugin`.

**What was built:**

1. **`inputCodec`/`outputCodec` added to `TypedMigration`** (`x/go/gorp/migrate.go`):
   - `NewTypedMigration[I,O](name, inputCodec Codec[I], outputCodec Codec[O], auto, post)`
   - When `inputCodec` is nil, decodes using `cfg.Codec` (DB's msgpack)
   - When `outputCodec` is nil, encodes using `cfg.Codec`
   - All existing callers updated to pass `nil, nil` for backward compatibility
   - 3 new test cases: inputCodec only, outputCodec only, both codecs

2. **Schema diff engine** (`oracle/diff/`):
   - `DiffStructs(old, new, oldTable, newTable)` — compares `UnifiedFields` by name
   - `DiffTables(old, new)` — diffs all struct types across two tables
   - `FormatTypeRef(ref)` — canonical string comparison for TypeRef
   - 16 Ginkgo tests covering all change types

3. **Dependency graph** (`oracle/deps/`):
   - `Build(table)` — forward/reverse edges from type references
   - `AffectedEntries(changedTypes)` — BFS through reverse edges, filtered to
     `HasKeyDomain` gorp entries
   - 8 Ginkgo tests: direct, transitive, shared, independent, deep nesting, cycles

4. **Generation modes** (`oracle/plugin/go/migrate/migrate.go`):
   - `generateWithDiff` determines skeleton vs propagation per type
   - **Skeleton mode**: `auto.gen.go` (copy unchanged fields, TODO for changes),
     `migrate.go` (post-migrate template), `vN.gen.go` (frozen snapshot)
   - **Propagation mode**: `auto.gen.go` copies all parent fields (nested types get
     zero-initialized via msgpack deserialization)
   - `migrate.gen.go` emits `NewTypedMigration` entries before `NewCodecTransition`
   - Uses `nil, nil` for inputCodec/outputCodec (data is still msgpack before codec
     transition)

5. **`oracle migrate generate` updated** (`oracle/cmd/migrate.go`):
   - Loads latest snapshot `.oracle` files → parses into old resolution table
   - Passes both tables to plugin via `Request{OldResolutions, SnapshotVersion}`
   - Plugin runs diff + deps internally

- **All tests pass**: gorp (121/121), diff (16/16), deps (8/8), migrate plugin (19/19),
  full oracle suite (37 suites, 0 failures)

---

## Phase 8: Test Infrastructure & Migration Test Helpers — DONE

**Status**: Complete on branch `sy-3824-oracle-auto-migration-plugin`.

**What was built** (scope changed to `x/go/gorp/testutil/` for idiomatic Go naming):

- **`gorp.EncodeKey[K, E](key)`** — exported helper in `gorp/entries.go` that produces
  the full storage key (prefix + encoded key) for seeding data under a specific type's
  key space.

- **`migratetest.SeedAndMigrate[K, Old, New]`** — primary test helper:
  - Creates in-memory KV store via `memkv.New()`
  - Seeds old entries under New's key prefix (encoded with seedCodec or default msgpack)
  - Runs migrations through `gorp.OpenTable`
  - Returns `Result[K, New]` for querying migrated data

- **`migratetest.Result[K, E]`** — wraps migration outcome:
  - `Entries(ctx)` — retrieves all entries
  - `Entry(ctx, key)` — retrieves single entry by key
  - `EntryCount(ctx)` — returns count of entries
  - `Version(ctx)` — reads migration version counter
  - `Close()` — releases resources

- **`migratetest.RunAutoPost[I, O]`** — tests auto+post as pure functions (no KV store):
  - Runs auto-migrate, then post-migrate on a single input
  - Returns the transformed output
  - Either function may be nil

- **10 Ginkgo tests** covering:
  - Typed migration with field transformation
  - Codec transition migration
  - Custom seed codec with matching inputCodec
  - Empty seed
  - Migration chain (typed + codec transition)
  - Version counter tracking
  - Entries retrieval
  - RunAutoPost: auto only, auto+post, post only

- **All tests pass**: migratetest (10/10), gorp (121/121), oracle (37 suites)

### Key files

- `x/go/gorp/entries.go` — added `EncodeKey[K, E]` export
- `x/go/gorp/testutil/migratetest.go` — `SeedAndMigrate`, `Result`, `RunAutoPost`
- `x/go/gorp/testutil/migratetest_test.go` — 10 tests
- `x/go/gorp/testutil/migratetest_suite_test.go` — test suite bootstrap

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
Phase 7: Schema Diff Engine & Skeleton/Propagation  DONE (sy-3824)
    ↓
Phase 8: Test Infrastructure ........................ DONE (sy-3824)
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
