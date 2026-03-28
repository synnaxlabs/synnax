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

### Entry counting

`NewTypedMigration` and `NewCodecTransition` already iterate all entries. We need them
to report how many they processed. Two options:

**Option A: Return count from Run.**

Change the `Migration` interface:

```go
type Migration interface {
    Name() string
    Run(ctx context.Context, tx kv.Tx, cfg MigrationConfig) (int, error)
}
```

Pro: Clean. Con: Breaking change to every migration implementation.

**Option B: Count in MigrationConfig.**

Add a counter field to `MigrationConfig` that migrations increment:

```go
type MigrationConfig struct {
    Prefix      []byte
    DBCodec     binary.Codec
    entries     int  // unexported, incremented by built-in migrations
}

func (c *MigrationConfig) RecordEntry() { c.entries++ }
func (c *MigrationConfig) Entries() int  { return c.entries }
```

Pro: Non-breaking. Built-in migrations (`NewTypedMigration`, `NewCodecTransition`)
increment automatically. Custom migrations can optionally call `RecordEntry()`.
Con: Mutable config passed by pointer.

**Option C: Wrap with result.**

`OpenTable` wraps each migration to track count and timing externally. Since
`OpenTable` controls the iteration, it can measure duration. For entry counts, the
built-in migration types can internally track and expose their count.

```go
type MigrationResult struct {
    Entries  int
    Duration time.Duration
}
```

Add an optional interface:

```go
type EntryCounter interface {
    EntriesProcessed() int
}
```

Built-in migrations implement it. Custom migrations that don't implement it just report
0 entries (or "unknown").

Pro: Non-breaking. No interface change. Con: Optional interface pattern.

**Recommendation: Option C.** Non-breaking, clean separation, and the built-in
migrations (which are the vast majority) report counts automatically.

### Implementation plan

1. Add `alamos.Instrumentation` to `TableConfig`.
2. Add `EntriesProcessed() int` method to built-in migration types
   (`typedMigration`, `codecTransitionMigration`). They already iterate entries
   and can count trivially.
3. In `OpenTable`, after running each migration:
   - Check if migration implements `EntryCounter`
   - Log "migration complete" with count and duration
4. In `OpenTable`, before the migration loop:
   - If pending > 0, log "starting migrations"
5. After the loop:
   - Log "migrations complete" with total duration
6. On error:
   - Log "migration failed" with count and duration
7. Update service call sites to pass `Instrumentation` through `TableConfig`.
8. Add tests verifying log output.

### Changes required

| File | Change |
|------|--------|
| `x/go/gorp/table.go` | Add `Instrumentation` to `TableConfig`, add logging to `OpenTable` |
| `x/go/gorp/migrate.go` | Add `EntriesProcessed()` to built-in migration types, add `EntryCounter` interface |
| `core/pkg/service/*/service.go` | Pass `Instrumentation` to `TableConfig` at each `OpenTable` call site |
| `x/go/gorp/migrate_test.go` | Tests for entry counting and log output |
