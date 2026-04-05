# 34 - Gorp In-Memory Indexes

**Feature Name**: In-Memory Secondary Indexes for Gorp Tables

# 0 - Summary

This RFC describes the implementation of in-memory secondary indexes for gorp tables.
Today, any query that does not look up entries by primary key requires a full table scan
with per-entry deserialization. By maintaining lightweight, derived index structures in
memory, we can provide O(1) exact-match lookups and O(log n) sorted/range queries on
arbitrary fields while keeping primary data in Pebble as the sole source of truth.

# 1 - Motivation

RFC 0026 identified the lack of efficient query mechanisms as the top problem with our
metadata toolchains. The core issue: gorp supports fast lookups by key but nothing else.
Any filter-based query iterates every entry in the table, deserializing each one.

Concrete examples of queries that currently trigger full table scans:

- Retrieving a channel by name
- Retrieving a user by username
- Finding all ranges with a specific label
- Listing schematics ordered by last modified time
- Counting devices of a specific make

As the number of metadata entries grows (channels alone can reach tens of thousands on
active deployments), these scans become a meaningful bottleneck, both in CPU time spent
on deserialization and in GC pressure from the throwaway allocations.

# 2 - Design

## 2.0 - Guiding Principles

1. **Indexes are derived, never source-of-truth.** An index can always be rebuilt from
   primary data without loss. No index-only fields, no denormalized aggregates.

2. **Table owns its indexes.** The `Table[K, E]` type is responsible for registering,
   maintaining, and exposing indexes. This keeps the index lifecycle co-located with the
   data lifecycle.

3. **Indexes are node-local.** Two cluster nodes may have momentarily different index
   states during Aspen replication lag. Queries always run against the local node's
   index.

4. **Start simple, optimize when measured.** Standard Go maps and slices first. Compact
   byte-array encodings, off-heap storage, and B-trees are available as future
   optimizations but are not part of the initial implementation.

## 2.1 - Index Types

Two categories of indexes serve different access patterns:

### 2.1.0 - Lookup Index

A lookup index provides O(1) exact-match access on a field value. Given a value, it
returns the set of primary keys whose entries have that value in the indexed field.

Use cases: "find channel where name = X", "find user where username = X".

For unique fields (username, email), the result set will always have one entry, but the
index does not enforce uniqueness. Uniqueness constraints belong in the service layer.

### 2.1.1 - Sorted Index

A sorted index maintains entries in order of a field value, providing O(log n) ordered
access and efficient cursor-based pagination.

Use cases: "list schematics ordered by last modified", "paginate ranges by time".

### 2.1.2 - Type-Specialized Backing Structures

Because the index structs (`Lookup[K, E, V]`, `Sorted[K, E, V]`) carry the concrete
value type `V` as a generic parameter (see 2.2), the index implementation can select a
specialized backing structure based on the type of value being indexed. This is a core
part of the design, not a future optimization.

The internal `storage` interface abstracts the backing structure:

```go
type storage[K Key, V comparable] interface {
    put(key K, value V)
    delete(key K, value V)
    get(value V) []K
}
```

`NewLookup` and `NewSorted` select the appropriate implementation at construction time
based on type reflection on `V`:

**Lookup index strategies:**

| Value type `V`                                                                | Backing structure       | Why                                                                                                                |
| ----------------------------------------------------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `bool`                                                                        | Two key lists           | Only two possible values. No map overhead, no hashing. Just two slices.                                            |
| Small integers (`uint8`, `int8`, `uint16`, `int16`)                           | Dense array `[maxV][]K` | Value space is bounded (256 or 65536 slots). Direct array indexing is faster than map hashing. No hash collisions. |
| `string`                                                                      | `map[string][]K`        | Variable-length, unbounded value space. Standard map is the right tool.                                            |
| Other `comparable` types (`int32`, `int64`, `uint32`, `uint64`, struct types) | `map[V][]K`             | Generic fallback. Go maps handle any comparable type.                                                              |

**Sorted index strategies:**

| Value type `V`                                                 | Backing structure                                          | Why                                                                                                           |
| -------------------------------------------------------------- | ---------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------- |
| Integer/timestamp types (`int64`, `uint64`, `telem.TimeStamp`) | `[]sortedEntry[K, V]` with native `<` comparison           | No comparator function call per comparison during binary search. The compiler can inline the comparison.      |
| `string`                                                       | `[]sortedEntry[K, V]` with string comparison               | Strings require lexicographic comparison. Still uses binary search, but each comparison touches string bytes. |
| Other ordered types                                            | `[]sortedEntry[K, V]` with caller-provided `Less` function | Fallback for types where the ordering isn't obvious from the type alone.                                      |

The key insight: these optimizations are only possible because the index knows `V` at
construction time. If we erased `V` to `string` (as the original design proposed), every
index would be a `map[string][]K` regardless of the underlying field type. Timestamps
would be hashed as strings instead of compared as integers. Booleans would go through a
hash map instead of indexing into two buckets. The generic `V` parameter keeps the door
open for the implementation to do the right thing per type.

All of this is invisible to the caller. `channel.ByName` and `channel.ByDataType` have
the same external API. The specialization is an internal implementation detail.

For the sorted index, insertions use binary search to find the correct position (O(log
n) search + O(n) shift for slice-backed storage). For the dataset sizes we're targeting
(< 100k entries per type), this is acceptable. If profiling reveals the shift cost is a
problem, we can swap the backing structure to a B-tree without changing the public API.

### 2.1.3 - Why Not Composite Indexes?

Composite indexes (indexing on multiple fields simultaneously) add significant
complexity: you need to define field ordering, handle partial prefix queries, and the
number of possible composites grows combinatorially. For our access patterns, chaining a
sorted index with a post-filter on the remaining fields is sufficient. We can revisit
composite indexes if profiling shows a real need.

## 2.2 - Two Layers: Gorp Primitives and Oracle-Generated Filters

The design has two layers. Gorp provides the index primitives (backing structures,
registration, maintenance). Oracle generates the ergonomic call-site code (filter
functions, type-safe wrappers). You can use gorp's primitives directly for hand-written
indexes, but the intended path is oracle generation.

### 2.2.0 - Gorp Layer: Index Primitives

Gorp provides generic index structs that hold backing data and satisfy a common `Index`
interface:

```go
// Index is the interface that all index types satisfy for registration on a Table.
type Index interface {
    populate(ctx context.Context, tx Tx) error
    update(key any, oldEntry any, newEntry any)
    remove(key any)
}

// Lookup is an in-memory hash index on a field of type V.
type Lookup[K Key, E Entry[K], V comparable] struct {
    name    string
    extract func(e *E) V
    data    *lookupData[K, V] // nil until registered, populated by OpenTable
}

// NewLookup creates an unpopulated Lookup index.
func NewLookup[K Key, E Entry[K], V comparable](
    name string,
    extract func(e *E) V,
) *Lookup[K, E, V]

// Get returns the primary keys matching the given values. Returns nil if
// the index has not been populated (not registered on a table).
func (l *Lookup[K, E, V]) Get(values ...V) []K

// Filter returns a Filter[K, E] that uses the index if populated, or
// falls back to field comparison if not.
func (l *Lookup[K, E, V]) Filter(values ...V) Filter[K, E]
```

Equivalent types exist for `Sorted[K, E, V]`.

The primitives are fully usable without oracle. Hand-written code can call
`idx.Filter(values...)` directly, or use `idx.Get(values...)` for raw key lookups. This
is slightly less ergonomic than the oracle-generated path but fully functional and
type-safe.

### 2.2.1 - Oracle Layer: Generated Filter Functions

Oracle has access to the `.oracle` schema files, which define struct fields, their
types, and which fields are indexed. From this, oracle generates two things per indexed
field: an index variable and a filter function.

```go
// Generated by oracle in the channel package

// The index struct. Used for registration on the table.
var nameIndex = gorp.NewLookup[uint32, Channel, string](
    "name",
    func(ch *Channel) string { return ch.Name },
)

// The filter function. Used at query call sites.
// Wraps nameIndex.Filter with the right types. Directly callable.
func ByName(values ...string) gorp.Filter[uint32, Channel] {
    return nameIndex.Filter(values...)
}

// Repeat for each indexed field.
var dataTypeIndex = gorp.NewLookup[uint32, Channel, telem.DataType](
    "data_type",
    func(ch *Channel) telem.DataType { return ch.DataType },
)

func ByDataType(values ...telem.DataType) gorp.Filter[uint32, Channel] {
    return dataTypeIndex.Filter(values...)
}

var createdAtIndex = gorp.NewSorted[uint32, Channel, telem.TimeStamp](
    "created_at",
    func(ch *Channel) telem.TimeStamp { return ch.CreatedAt },
    func(a, b telem.TimeStamp) bool { return a.Before(b) },
)

func ByCreatedAt(values ...telem.TimeStamp) gorp.Filter[uint32, Channel] {
    return createdAtIndex.Filter(values...)
}

// Index list for registration. Also generated.
var Indexes = []gorp.Index{nameIndex, dataTypeIndex, createdAtIndex}
```

Oracle generates all of this from the `.oracle` file. The struct field names, types, and
index annotations are all available at generation time. No runtime reflection, no
generic type gymnastics.

### 2.2.2 - Registration Site

```go
table, err := gorp.OpenTable[uint32, channel.Channel](ctx, gorp.TableConfig[channel.Channel]{
    DB:      db,
    Indexes: channel.Indexes,
})
```

Oracle generates the `Indexes` slice. Registration is a single line.

### 2.2.3 - Query Site

```go
// Single value lookup
table.NewRetrieve().
    Where(channel.ByName("sensor_1")).
    Entry(&ch).
    Exec(ctx, tx)

// Multi-value lookup (OR semantics: match any of these names)
table.NewRetrieve().
    Where(channel.ByName("sensor_1", "sensor_2", "sensor_3")).
    Entries(&results).
    Exec(ctx, tx)

// Compose with additional non-indexed filters
table.NewRetrieve().
    Where(channel.ByDataType(telem.Float32)).
    Where(channel.IsVirtual()).
    Entries(&results).
    Exec(ctx, tx)
```

The call site reads as natural language: `Where(channel.ByName("sensor_1"))`. The value
type is compile-time checked. `channel.ByName(42)` does not compile. The caller doesn't
know or care whether an index exists behind the function.

### 2.2.4 - Self-Contained Filters: The Index Holds Its Own Backing Data

The index struct holds a pointer to its backing data. `Retrieve` stays completely
unaware of indexes.

Here's the lifecycle:

1. **Definition time.** `NewLookup` creates the index struct as a package-level var. Its
   backing data pointer is nil. The index is defined but unpopulated.

2. **Registration time.** `OpenTable` scans all entries and populates the backing
   structure. It writes the populated data into the index struct. This is a one-time
   write at startup.

3. **Query time.** When `channel.ByName("sensor_1")` is called, it calls
   `nameIndex.Filter("sensor_1")`, which checks the backing data pointer. If populated,
   it looks up keys in the map directly. If nil, it falls back to a field comparison
   closure.

From `Retrieve`'s perspective, the filter is just a `Filter[K, E]` with an `Eval`
closure. `Retrieve` doesn't know or care whether that closure does a map lookup or a
field comparison.

This means:

- **`Retrieve` doesn't change.** No new execution paths, no index awareness, no
  special-casing.
- **Adding or removing an index changes performance but never correctness.** The filter
  always produces the same results.
- **Testing is trivial.** Unit tests that don't care about performance don't need to
  register indexes. The generated filter functions work without registration, just
  slower.

### 2.2.5 - Why This Design?

1. **Oracle does the hard work.** The Go type system makes it awkward to have a single
   value that is both callable and interface-satisfying with access to internal state.
   Instead of fighting the type system with generic tricks, oracle generates exactly the
   right concrete code for each indexed field at compile time.

2. **Compile-time type safety.** The value type is checked at every call site. No string
   names, no `any` casts.

3. **Zero new API concepts for callers.** Call sites use `Where()` with a function call.
   The function happens to be index-backed. Callers don't need to know.

4. **Graceful degradation.** Generated filter functions work without index registration.
   They fall back to field comparison. Performance changes, correctness doesn't.

5. **`Retrieve` is untouched.** The query executor doesn't know about indexes. The
   filter is self-contained.

6. **Clean separation of concerns.** Gorp provides primitives. Oracle generates
   ergonomics. Hand-written code can use the primitives directly when oracle isn't
   available.

### 2.2.6 - Why Register at Open Time?

Registering indexes at open time (rather than lazily on first query) gives us:

1. **Deterministic startup cost.** The full scan to populate indexes happens once,
   predictably, during server boot.
2. **No cold-query penalty.** Every query after startup hits a warm index.
3. **Simpler concurrency model.** Indexes exist for the full lifetime of the table. No
   races around lazy initialization.

## 2.3 - Index Population

When `OpenTable` is called, after migrations complete:

1. Open a Pebble iterator over the table's key prefix.
2. For each entry, decode it and pass it to every registered index's `Extract` function.
3. Insert the extracted value and primary key into the appropriate index structure.

This is a single sequential scan. For a table with N entries and M indexes, the cost is
O(N \* M) extract calls plus the insertion cost for each index structure.

### 2.3.0 - Startup Performance Expectations

For our target scale (< 100k entries per type, typically < 10k), population should
complete in low milliseconds. If a specific table proves slow, we can:

- Populate indexes asynchronously and serve queries from Pebble until the index is ready
  (adds complexity, defer until needed).
- Use a snapshot-based warm start where index state is periodically serialized to disk
  (adds durability concerns, defer until needed).

Neither optimization is planned for the initial implementation.

## 2.4 - Index Maintenance

Indexes must stay consistent with primary data through creates, updates, and deletes.

### 2.4.0 - Hook Point: The Observe Pipeline

`Table[K, E]` already exposes `Observe()`, which translates raw KV changes into typed
change sequences. This is the natural hook point for index maintenance.

**One observer per table, not per index.** Each table opens exactly one observer on its
underlying KV prefix. When a change arrives, the observer fans it out to all registered
indexes in a single callback. If a table has 3 indexes, one change event triggers 3
extract calls and 3 backing structure updates, but only 1 observer invocation.

This is another reason eager registration is the right call. Because all indexes are
registered before `OpenTable` returns, the observer closure captures a fixed set of
indexes. No locking on the index list during change processing, no dynamic registration
after the fact, no tearing down and rebuilding observers.

When the observer processes a KV change:

- **Set (create or update):** Extract the new index value from the decoded entry. If
  updating, remove the old index entry first (using the reverse map, see below), then
  insert the new entry. The observer calls each index's extract function and update
  method in sequence.

- **Delete:** Remove the index entry for the deleted key. The observe pipeline gives us
  the key but not the entry (it's already gone from Pebble). The reverse map provides
  the old indexed value so we can remove it from the forward map.

### 2.4.1 - Reverse Map for Efficient Updates and Deletes

Each index maintains a reverse map (`map[K]V`) that maps every primary key to its
current indexed value. This serves two purposes:

1. **Updates:** When a channel's name changes from "sensor_1" to "sensor_2", we need to
   remove the old `"sensor_1" -> key` mapping before inserting the new one. The reverse
   map gives us the old value in O(1).

2. **Deletes:** The observe pipeline provides the deleted key but not the deleted entry.
   The reverse map tells us what indexed value to remove from the forward map.

The alternative is scanning the forward map's value slices looking for the key. For a
lookup index on channel names with 50k channels, that's 50k comparisons per
update/delete. The reverse map costs roughly 40 bytes per entry (for a uint32 key + a
20-byte string value + map overhead) but makes both operations O(1).

### 2.4.2 - Consistency Model

Index updates are applied via the observer, which fires immediately after the KV
transaction commits (pebblekv applies the batch to disk, then calls `NotifyGenerator`).
This means there is a tiny window (microseconds) between when the data is committed to
Pebble and when the index reflects it. In practice:

- Within a single node, the index is consistent with committed data within microseconds
  of each commit. For our metadata write rates (tens of writes per second), this is
  indistinguishable from synchronous.
- Across nodes, the index reflects whatever Aspen has replicated so far. This is the
  same consistency model as the primary data itself, so indexes don't introduce new
  inconsistency windows.

### 2.4.3 - Transaction Isolation

A question arises: should uncommitted writes within a transaction be visible to
index-backed queries within that same transaction?

For the initial implementation: **no.** Index queries always reflect committed state.
This matches the current behavior where `Retrieve` reads from the underlying KV store,
which (in Pebble) provides snapshot isolation at the transaction level. Adding
transaction-local index overlays is complex and not needed for our current access
patterns, where writes and reads in the same transaction are rare.

If this becomes a requirement, we can add a transaction-local overlay that is merged
with the committed index at query time.

## 2.5 - Query Integration

Because `Lookup.Filter` and `Sorted.Filter` produce standard `Filter[K, E]` values,
index-backed queries flow through the existing `Where` API. No new query methods are
needed for basic lookup operations.

### 2.5.0 - Execution Path

`Retrieve` does not know about indexes. The intelligence lives in the `Filter`'s `Eval`
closure, which the index's `.Filter(values...)` method produces:

1. If the index's backing data is populated, the `Eval` closure looks up candidate keys
   directly from the in-memory map, fetches those entries from Pebble by key, and
   returns the matches. This converts the query into the existing fast `execKeys` path
   internally.
2. If the backing data is nil (index not registered), the `Eval` closure falls back to a
   field comparison against each decoded entry, the same as any non-indexed filter.

From `Retrieve`'s perspective, it's just calling `Eval` on a `Filter`. The filter
decides whether to use the fast path or the slow path.

### 2.5.1 - Composing Indexed and Non-Indexed Filters

Index narrowing composes naturally with post-filtering:

```go
table.NewRetrieve().
    Where(channel.ByDataType(telem.Float32)).
    Where(channel.IsVirtual()).
    Entries(&results).
    Exec(ctx, tx)
```

Execution: the `ByDataType` filter's closure hits the index and produces candidate keys.
Those entries are fetched from Pebble and then filtered through `IsVirtual`. This is a
significant improvement over scanning the entire table even when the post-filter is
needed.

### 2.5.2 - Multiple Indexed Filters

When multiple indexed filters appear in the same query, the first index-backed filter
encountered produces candidate keys. The remaining filters (indexed or not) are applied
as post-filters on those candidates. We can add index intersection as an optimization
later if profiling shows it matters.

### 2.5.3 - Sorted Index Queries

Sorted indexes need query surface beyond what `Where` provides. For ordering and
cursor-based pagination, we add methods to `Retrieve`:

```go
table.NewRetrieve().
    OrderBy(channel.ByCreatedAt, gorp.Desc).
    Limit(20).
    After(lastSeenCursor).
    Entries(&results).
    Exec(ctx, tx)
```

`OrderBy` accepts an oracle-generated filter function that is backed by a `Sorted`
index. `After` takes a cursor value (the sort value of the last entry from the previous
page) and uses binary search to start iteration from that point, giving O(log n + limit)
pagination instead of O(offset + limit).

Note: `OrderBy` is a separate method from `Where` because it expresses a different
intent (ordering, not filtering). A sorted index filter can appear in either position:
in `Where` for exact match filtering, in `OrderBy` for result ordering.

## 2.6 - Concurrency

Index structures must be safe for concurrent reads and writes.

### 2.6.0 - Read-Write Lock

The simplest approach: a `sync.RWMutex` per index. Reads acquire a read lock, writes
acquire a write lock. Given that our write rate for metadata is low (tens of writes per
second at most) and reads are fast (map lookup or binary search), contention should be
negligible.

### 2.6.1 - Lock Granularity

Starting with one lock per index (not per table or per DB). This allows concurrent reads
on different indexes of the same table, and concurrent reads on the same index with
writes on a different one.

If profiling shows the per-index lock overhead is a problem, the first optimization is
collapsing to per-table locks. Since the single observer per table already serializes
writes to all that table's indexes, per-table write locking loses nothing. This is a
one-line internal change with no API impact.

## 2.7 - Memory Overhead

For a lookup index on a table with N entries:

- Forward map: N entries, each a string key + slice of primary keys. For channels with
  ~20 byte names and uint32 keys: ~(20 + 8 + 24) = ~52 bytes per entry.
- Reverse map: N entries, each a uint32 key + string value: ~(4 + 20 + 16) = ~40 bytes
  per entry.
- Total: ~92 bytes per entry per index.

At 100k channels with 2 lookup indexes: ~18 MB. Acceptable.

For sorted indexes, similar math applies with the addition of the sorted slice overhead
(24 bytes for slice header, 8 bytes per entry for the value pointer).

## 2.8 - Rebuild and Recovery

Since indexes are purely derived, recovery from any corruption is simple: drop the index
and rebuild from primary data. This can be triggered:

1. Automatically on startup (the current plan, since startup always rebuilds).
2. Manually via an admin API if needed in the future.

There is no WAL, no crash recovery log, and no durability concern for index data.

# 3 - What This RFC Does Not Cover

- **Detailed oracle generation logic.** Phase 3 covers wiring oracle to generate index
  variables and filter functions, but the specifics of `.oracle` schema annotation
  syntax and the generator implementation are not specified here.

- **Composite indexes.** As discussed in 2.1.2, deferred until measured need.

- **Async index population.** Deferred until startup time becomes a measured problem.

- **Transaction-local index overlays.** Deferred until needed (see 2.4.2).

- **Full-text search indexes.** A different problem with different solutions (inverted
  indexes, trigram indexes). Not needed for our current access patterns.

- **Cross-node index coordination.** Indexes are node-local. If we ever need globally
  consistent secondary indexes, that's a distributed systems problem that belongs in
  Aspen, not gorp.

- **Index intersection for multi-index queries.** Deferred per 2.5.2; use first index
  and post-filter for now.

# 4 - Implementation Order

## Phase 1: Gorp Index Primitives

1. Define the `Index` interface and the `Lookup[K, E, V]` and `Sorted[K, E, V]` structs
   with their internal backing structures (forward map + reverse map for lookup, sorted
   slice for sorted).
2. Implement type-specialized backing structure selection in `NewLookup` and
   `NewSorted`.
3. Implement `Lookup.Filter` and `Sorted.Filter` methods: when backing data is
   populated, short-circuit to key-based fetch; when nil, fall back to field comparison.
4. Add the `Indexes []Index` field to `TableConfig`.
5. Implement index population during `OpenTable` (post-migration scan).
6. Wire up index maintenance through the single-observer-per-table observe pipeline.
7. Add `sync.RWMutex` per index for concurrent access.
8. Add `DPanic` guards for unregistered usage and double registration.

## Phase 2: Query Integration

1. Implement `OrderBy` and `After` on `Retrieve` for sorted index pagination.
2. Ensure composition with existing `Where`, `WhereKeys`, `Limit`, `Offset`.
3. Verify that index-backed filters degrade to full-scan correctly when unregistered.

## Phase 3: Oracle Generation

1. Add index annotation support to `.oracle` schema files.
2. Generate index variables, filter functions, and `Indexes` slices per entry type.
3. Wire generated `Indexes` into `OpenTable` calls for channel, user, and other
   high-traffic tables.
4. Migrate existing hand-written filter functions to use the generated versions.

# 5 - Resolved Decisions

1. **Two-layer design: gorp primitives + oracle generation.** Gorp provides generic
   index structs (`Lookup[K, E, V]`, `Sorted[K, E, V]`) with backing data and an `Index`
   interface. Oracle generates the ergonomic layer: callable filter functions (`ByName`,
   `ByDataType`), index variables, and the `Indexes` registration slice. This avoids
   fighting Go's type system to make a single value both callable and
   interface-satisfying. Oracle resolves everything at compile time.

2. **The index struct holds its own backing data.** `Retrieve` does not need to discover
   indexes. `OpenTable` populates the backing structure and writes it into the index
   struct. At query time, the generated filter function calls `idx.Filter(values...)`,
   which checks the backing data pointer internally. `Retrieve` is completely unchanged.

3. **Type-specialized backing structures are part of the core design.** The generic `V`
   parameter allows `NewLookup` and `NewSorted` to select optimized implementations
   based on the value type (dense arrays for small integers, two-bucket lists for
   booleans, native comparison for numeric/timestamp sorted indexes).

4. **Unregistered index usage fires `zap.DPanic`.** If an index's `.Filter()` method is
   called but the index was never registered on a table, it falls back to a full scan
   (correctness is preserved) and fires a `DPanic`. This panics in development (catching
   the mistake immediately) and logs at error level in production (not crashing a
   running deployment over a performance issue). The `DPanic` fires once per index via
   `sync.Once`, not once per query.

5. **Double registration fires `zap.DPanic`.** If `OpenTable` encounters an index whose
   backing data pointer is already non-nil (meaning it was already registered on another
   table, or registered twice on the same table), it fires a `DPanic`. Same rationale:
   panic in development, log in production. This catches both cross-table double
   registration and duplicate entries in the `Indexes` slice.

6. **Sorted index range queries are deferred.** The MVP supports exact match via `Where`
   and ordering/cursor pagination via `OrderBy`/`After`. Range queries (`.Between`,
   `.After`, `.Before` methods on the `Sorted` struct) are left as post-filters for now.
   The sorted index backing structure supports range operations naturally, so adding
   them later does not require architectural changes. We want real usage patterns before
   committing to the API shape.
