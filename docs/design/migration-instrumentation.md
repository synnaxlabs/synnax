# Migration Instrumentation Design

## Status: Draft

## Problem

Migrations in `gorp.OpenTable` are operationally invisible. When a server boots and hits
pending migrations, there is zero output. The operator sees the server appear to hang.
There is no indication of what's running, how long it will take, or what went wrong if
it fails.

The only migration with any logging is `rangeGroupsMigration` in the ranger service,
which manually injects an `*alamos.Logger`. Every other migration (codec transitions,
typed migrations) is silent.

## Design

### Add `alamos.Instrumentation` to `TableConfig`

```go
type TableConfig[E any] struct {
    DB             *DB
    Codec          binary.Codec
    Migrations     []Migration
    Instrumentation alamos.Instrumentation
}
```

`alamos.Logger` is nil-safe (no-op when nil), so existing call sites that don't pass
instrumentation continue to work silently. No breaking changes.

### Log messages

All messages use structured fields via zap. Only emitted when there are pending
migrations. Normal boots (all migrations applied) produce zero output.

#### 1. "starting migrations" (Info)

Emitted once per table when pending migrations are detected.

```
INFO  starting migrations  {"table": "Schematic", "pending": 2}
```

**Why:** Tells the operator the server isn't stuck. Without this, a 30-second pause
during boot looks like a crash.

#### 2. "migration complete" (Info)

Emitted after each migration finishes successfully.

```
INFO  migration complete  {"table": "Schematic", "migration": "msgpack_to_binary", "entries": 4521, "elapsed": "1.2s"}
```

**Why:** Progress indicator. On a table with 3 chained migrations, the operator sees
progress. Entry count reveals whether the migration did real work or was a no-op.
Duration helps set expectations for future upgrades.

#### 3. "migrations complete" (Info)

Emitted once per table after all migrations finish.

```
INFO  migrations complete  {"table": "Schematic", "migrations": 2, "elapsed": "3.4s"}
```

**Why:** Bookend to message 1. Confirms the table is done and the server is moving on.

#### 4. "migration failed" (Error)

Emitted when a migration fails.

```
ERROR  migration failed  {"table": "Schematic", "migration": "v53_schema_migration", "entries_processed": 9999, "elapsed": "12.3s", "error": "..."}
```

**Why:** Entries processed before failure is critical for debugging. Failed on entry 0
means codec bug. Failed on entry 9,999,999 means corrupt entry or edge case in
transform logic.

#### 5. "already applied" (Debug)

Emitted once per table when there are applied migrations and pending migrations.
Provides context about what has already run on this database.

```
DEBUG  already applied  {"table": "Schematic", "applied": ["msgpack_to_binary"]}
```

**Why:** Helps the operator distinguish a fresh database from one that's been through
prior upgrades. Useful for debugging migration ordering issues. Debug-level because it's
context, not actionable.

#### 6. "migration progress" (Debug)

Emitted every 10,000 entries during a migration.

```
DEBUG  migration progress  {"table": "Schematic", "migration": "v53_schema_migration", "entries": 10000, "elapsed": "0.8s"}
DEBUG  migration progress  {"table": "Schematic", "migration": "v53_schema_migration", "entries": 20000, "elapsed": "1.6s"}
```

**Why:** When watching a slow migration on a large table, the operator needs to know
progress is being made and roughly estimate completion time. The interval is count-based
(not time-based) because entry processing speed varies by table. 10,000 is a reasonable
default that avoids noise on small tables while giving useful signal on large ones.

### What we don't log

- **Nothing when all migrations are applied.** Normal boot is the common case. Logging
  "0 pending" for every table on every boot is noise.
- **No per-entry logging.** Millions of log lines helps nobody.

### Logger in MigrationConfig

Built-in migrations need a logger for debug-level progress logs. Rather than using
context (which is for developer-injected service dependencies), the logger goes on
`MigrationConfig` as infrastructure:

```go
type MigrationConfig struct {
    Prefix  []byte
    DBCodec binary.Codec
    L       *alamos.Logger
}
```

`OpenTable` populates it from `TableConfig.Instrumentation.L`. Since `alamos.Logger`
is nil-safe, migrations that receive a nil logger just skip logging with no nil checks
needed.

Built-in migrations use it for progress logging:

```go
for iter.First(); iter.Valid(); iter.Next() {
    count++
    if count%10000 == 0 {
        cfg.L.Debug("migration progress",
            zap.String("migration", m.name),
            zap.Int("entries", count),
        )
    }
    // ... decode, transform, encode, set
}
```

Custom migrations can also use `cfg.L` for their own logging, replacing the need to
manually inject `*alamos.Logger` (as `rangeGroupsMigration` does today).

### Entry counting

Built-in migrations (`typedMigration`, `codecTransitionMigration`) already iterate all
entries. They track count internally during `Run` and expose it via an optional
interface:

```go
type EntryCounter interface {
    EntriesProcessed() int
}
```

`OpenTable` checks for this after each migration to include entry count in the
"migration complete" log. Custom migrations that don't implement it just log without
an entry count. Non-breaking, no changes to the `Migration` interface.

### Error context on failure

When a migration fails, the error should identify which entry caused the problem.
Today the error is just:

```
migration (v53_schema_migration) failed: invalid msgpack data
```

With proper error wrapping:

```
migration (v53_schema_migration) failed: entry 550e8400-e29b-41d4-a716-446655440000: invalid msgpack data
```

#### Tighten `NewTypedMigration` type constraints

Change from `[I, O any]` to `[IK, OK Key, I Entry[IK], O Entry[OK]]`:

```go
func NewTypedMigration[IK Key, OK Key, I Entry[IK], O Entry[OK]](
    name string,
    inputCodec, outputCodec binary.Codec,
    transform TransformFunc[I, O],
) Migration
```

This gives typed key access on both input and output entries, enabling precise error
wrapping at each failure point:

- **Decode failure:** Raw bytes (`%q` on `iter.Key()`). Can't do better since the entry
  didn't decode.
- **Transform failure:** Input key via `old.GorpKey()`. Clean, human-readable.
- **Encode failure:** Output key via `newEntry.GorpKey()`. Clean, human-readable.

#### Call site impact

- 1 generated call site (`core/pkg/service/arc/migrate.gen.go`)
- 2 oracle code generation templates (`oracle/plugin/go/migrate/migrate.go`)
- 9 test call sites (`x/go/gorp/migrate_test.go`)

All manageable. Oracle templates need the old/new key type added to their generation
context.

### Implementation plan

#### Phase 1: Instrumentation infrastructure

1. Add `alamos.Instrumentation` to `TableConfig`.
2. Add `L *alamos.Logger` to `MigrationConfig`.
3. `OpenTable` populates `MigrationConfig.L` from `TableConfig.Instrumentation.L`.

#### Phase 2: Entry counting and progress logging

4. Add `EntryCounter` optional interface to `migrate.go`.
5. Add entry count tracking + `EntriesProcessed()` to `typedMigration` and
   `codecTransitionMigration`.
6. Add debug-level progress logging (every 10,000 entries) inside the iteration
   loops of built-in migrations, using `cfg.L`.

#### Phase 3: Error context

7. Change `NewTypedMigration` signature from `[I, O any]` to
   `[IK, OK Key, I Entry[IK], O Entry[OK]]`.
8. Add error wrapping with typed keys in `typedMigration.Run`:
   - Decode failure: raw key bytes (`%q`)
   - Transform failure: `old.GorpKey()`
   - Encode failure: `newEntry.GorpKey()`
9. Update oracle code generation templates for new type parameters.
10. Update all test call sites.

#### Phase 4: OpenTable logging

11. In `OpenTable`, before the migration loop:
    - If pending > 0, log "starting migrations" (Info)
    - Log "already applied" with applied migration names (Debug)
12. After each migration:
    - Check `EntryCounter` interface, log "migration complete" with count and
      duration (Info)
13. After the loop:
    - Log "migrations complete" with total count and duration (Info)
14. On error:
    - Log "migration failed" with entries processed, duration, error (Error)

#### Phase 5: Service wiring

15. Update all `OpenTable` call sites in `core/pkg/service/*/service.go` to pass
    `Instrumentation` through `TableConfig`.

#### Phase 6: Tests

16. Tests for entry counting (`EntriesProcessed()`).
17. Tests for error wrapping with typed keys.
18. Tests for log output (info and debug messages).

### Error handling policy

Migration failure is fatal to server startup. This is the correct behavior and should
not change. Rationale:

- **Codec transition failure:** Data is unreadable in the new format. Table can't
  function.
- **Schema migration failure:** Service expects the new struct shape. Reads would decode
  garbage.
- **Cross-service migration failure:** New code expects the reorganized data.
  Unpredictable behavior if half-migrated.

The transaction atomicity guarantees the database is never left in a corrupt state. On
failure, everything rolls back. The operator fixes the root cause and reboots. The
migration re-runs from scratch.

The logging and error context improvements make the failure actionable: the operator
sees which migration failed, which entry caused it, how far through the table it got,
and how long it ran before failing.

### Out of scope (decided not to solve now)

- **Memory bounding / batched commits:** Tables are small enough today that single-
  transaction migration is fine. Would require checkpointing and break atomicity.
  Revisit if migrations start hitting multi-GB batch sizes.
- **Retry / resumability:** Tightly coupled to batching. Fix the root cause instead.
  Error context improvements make diagnosis easy.
- **Tracing:** Logging provides sufficient visibility. Not adding OpenTelemetry spans.

### Changes required

| File | Change |
|------|--------|
| `x/go/gorp/table.go` | Add `Instrumentation` to `TableConfig`, add logging to `OpenTable` |
| `x/go/gorp/migrate.go` | Change `NewTypedMigration` signature, add `EntryCounter`, add `L` to `MigrationConfig`, add error wrapping, add progress logging |
| `oracle/plugin/go/migrate/migrate.go` | Update code generation templates for new type parameters |
| `core/pkg/service/*/service.go` | Pass `Instrumentation` to `TableConfig` at each `OpenTable` call site |
| `x/go/gorp/migrate_test.go` | Tests for counting, error context, log output |
