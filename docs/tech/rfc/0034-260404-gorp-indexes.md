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

5. **Read-your-own-writes inside a tx.** A `Retrieve` issued through a write tx sees
   that tx's pending creates, updates, and deletes via a per-tx delta overlay. Outside
   any tx, queries see committed-only state.

## 2.1 - Index Types

Three primitives serve different access patterns. `Lookup` and `Sorted` are the primary
types; `BytesLookup` exists to support tables whose primary key is `[]byte` and
therefore not strictly comparable.

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

Because `Lookup[K, E, V]` and `Sorted[K, E, V]` carry the concrete value type `V` as a
generic parameter, the backing structure can be specialized at construction time without
giving up type safety at the boundary. The MVP ships two specializations, hidden behind
an internal `lookupStorage[K, V]` interface:

```go
type lookupStorage[K IndexKey, V comparable] interface {
    put(key K, value V)
    remove(key K, value V)
    get(value V) []K
}
```

`newLookupStorage` selects the implementation by type-switching on the zero value of
`V`:

| Value type `V`              | Backing structure        | Why                                                                                                           |
| --------------------------- | ------------------------ | ------------------------------------------------------------------------------------------------------------- |
| `bool`                      | `boolLookupStorage[K]`   | Two slices (one per bucket). No hashing, no map overhead. The dominant case for boolean flags.                |
| Any other `comparable` type | `mapLookupStorage[K, V]` | `map[V][]K`. Default fallback for strings, structs, integer types, and anything else satisfying `comparable`. |

Sorted indexes do not type-switch. Because `V` is constrained to `cmp.Ordered`, every
`Sorted[K, E, V]` is backed by a single `sortedStorage[K, V]` that uses a sorted slice
of `sortedEntry[K, V]{Value V; Key K}` and the native `<` operator for comparison. No
caller-supplied `Less` function, no boxed comparator, and the compiler can inline the
comparison at every binary-search step. Insertion is O(log n) binary search plus O(n)
slice shift, which is acceptable for the target scale (< 100k entries per type). If
profiling reveals the shift cost is a problem, the backing can swap to a B-tree without
changing the public API.

Two specializations called out in earlier drafts have not been implemented because no
current workload requires them:

- Dense arrays for small integer values (`uint8`, `int8`, `uint16`, `int16`).
- Native-comparison sorted slices specialized per primitive width.

The `lookupStorage` interface accepts further specializations at any point.

### 2.1.3 - BytesLookup: The Byte-Keyed Variant

Some tables (notably ontology relationships) store their primary key as a `[]byte`
because the key is a composite encoded inline. Slices are not `comparable` in Go, so
they cannot be used as map keys, and the standard `Lookup` constraint
(`K Key, comparable`) excludes them. `BytesLookup[E, V]` is a parallel implementation
keyed on `[]byte`, with the same external semantics as `Lookup`:

```go
type BytesLookup[E Entry[[]byte], V comparable] struct { /* ... */ }

func NewBytesLookup[E Entry[[]byte], V comparable](
    name string,
    extract func(e *E) V,
) *BytesLookup[E, V]

func (l *BytesLookup[E, V]) Get(values ...V) [][]byte
func (l *BytesLookup[E, V]) GetTx(tx Tx, values ...V) [][]byte
func (l *BytesLookup[E, V]) Filter(values ...V) Filter[[]byte, E]
```

Internally, `BytesLookup` keys its reverse map and per-tx delta on `string([]byte)`,
which is a no-allocation conversion in modern Go. Use `Lookup` whenever the table key is
strictly comparable; reach for `BytesLookup` only when the key is genuinely `[]byte`.

### 2.1.4 - Why Not Composite Indexes?

Composite indexes (indexing on multiple fields simultaneously) add significant
complexity: you need to define field ordering, handle partial prefix queries, and the
number of possible composites grows combinatorially. For our access patterns, chaining a
sorted index with a post-filter on the remaining fields is sufficient. We can revisit
composite indexes if profiling shows a real need.

## 2.2 - Two Layers: Gorp Primitives and Oracle-Generated Wrappers

The design has two layers. Gorp provides the index primitives (backing structures,
registration, maintenance, the per-tx delta overlay). Oracle generates the ergonomic
call-site code (a per-service `indexes` struct, `MatchX` / `MatchXs` filter
constructors, sorted-index `Order` helpers, and the registration glue). You can use
gorp's primitives directly for hand-written indexes, but the intended path is oracle
generation.

### 2.2.0 - Gorp Layer: Index Primitives

Gorp exposes a sealed `Index[K, E]` interface plus the three concrete generic types that
satisfy it (`Lookup`, `Sorted`, `BytesLookup`). The interface methods are unexported, so
external packages cannot substitute their own implementations:

```go
// Index is a registered secondary index on a Table.
type Index[K Key, E Entry[K]] interface {
    Name() string
    populate() (insert func(E), finish func(), err error)
    set(key K, entry E)
    delete(key K)
    stageSet(tx Tx, key K, entry E)
    stageDelete(tx Tx, key K)
}
```

`populate` returns a pair of closures rather than running synchronously. The table walks
every entry once and fans each one into every index's `insert`, then calls `finish` per
index after the walk. `Sorted` exploits this by appending entries unordered during
`insert` and sorting once in `finish` (O(N log N) instead of O(N²)). The implementation
may hold a write lock across the whole populate phase, so `finish` is mandatory; failing
to call it leaks the lock.

`set` / `delete` are invoked by the Table observer on committed KV changes. `stageSet` /
`stageDelete` are invoked by table-bound writers on every `Set` / `Delete` call against
an open tx; they record the pending mutation in a per-tx delta that resolves at query
time. See §2.4.3.

The two index types primary callers reach for are constructed via:

```go
// IndexKey is the subset of Key that supports secondary indexes. It excludes
// unhashable Key types (notably ~[]byte).
type IndexKey interface {
    Key
    comparable
}

func NewLookup[K IndexKey, E Entry[K], V comparable](
    name string,
    extract func(e *E) V,
) *Lookup[K, E, V]

func NewSorted[K IndexKey, E Entry[K], V cmp.Ordered](
    name string,
    extract func(e *E) V,
) *Sorted[K, E, V]
```

Both types expose three query-side methods:

```go
// Get returns committed-only keys matching any of the given values.
func (l *Lookup[K, E, V]) Get(values ...V) []K

// GetTx returns the merge of committed keys and the open tx's per-tx delta.
// Used by graph-traversal helpers that probe the index outside Retrieve.
func (l *Lookup[K, E, V]) GetTx(tx Tx, values ...V) []K

// Filter returns a Filter[K, E] whose Keys are resolved at Retrieve.Exec
// time against the open tx, merging committed state with the per-tx delta.
func (l *Lookup[K, E, V]) Filter(values ...V) Filter[K, E]
```

`Sorted` adds `Ordered(dir Direction) SortedQuery[K, E, V]` and the `SortedQuery.After`
cursor for use with `Retrieve.OrderBy`. See §2.5.3.

The primitives are fully usable without oracle. Hand-written code can call
`idx.Filter(values...)` directly, or use `idx.Get(values...)` for raw key lookups
against committed state. The sealed `Index` interface and the unexported observer
contract mean only gorp can drive a backing structure, but the public constructors and
query methods are stable.

### 2.2.1 - Oracle Layer: Generated Code

Oracle reads the `.oracle` schema files, which define struct fields, their types, and
which fields are indexed via `@index lookup` or `@index sorted`. From this, oracle
generates four pieces per `@retrieve`-annotated struct in `retrieve.gen.go`:

1. A package-private `indexes` struct that bundles the per-Service index instances.
2. A `newIndexes()` constructor and an `(i indexes) all()` method for registration.
3. A `MatchX` / `MatchXs` filter function per `@filter` field. When the field also
   carries `@index`, the function routes through the corresponding index instead of
   compiling to a closure scan.
4. For sorted indexes only, an `Order` closure type, an `OrderBy` method on the
   per-service `Retrieve`, and an `OrderByX(dir, cursor...)` constructor per indexed
   field.

A schema such as:

```
Channel struct {
    name Name {
        @doc value "is the human-readable channel name."
        @filter
        @index lookup
    }
}
```

produces:

```go
// indexes bundles the per-Service secondary indexes registered on the
// Channel table.
type indexes struct {
    name *gorp.Lookup[Key, Channel, Name]
}

func newIndexes() indexes {
    return indexes{
        name: gorp.NewLookup[Key, Channel, Name](
            "name",
            func(e *Channel) Name { return e.Name },
        ),
    }
}

func (i indexes) all() []gorp.Index[Key, Channel] {
    return []gorp.Index[Key, Channel]{i.name}
}

// MatchNames is the index-routed filter constructor.
func MatchNames(vals ...Name) Filter {
    return func(r Retrieve) gorp.Filter[Key, Channel] {
        return r.indexes.name.Filter(vals...)
    }
}
```

The per-service `Filter` type is a type alias for
`gorp.BoundFilter[Retrieve, Key, Channel]`, which is itself a closure that takes the
service-defined `Retrieve` and produces a `gorp.Filter[K, E]`. This indirection lets
filter constructors read from `r.indexes`, `r.label`, `r.hostProvider`, etc., when
evaluated by `Retrieve.Where`. The `Match` / `And` / `Or` / `Not` composition helpers in
each service package are one-line wrappers around `gorp.MatchBound` / `AndBound` /
`OrBound` / `NotBound`, which means the closure plumbing is written once in the gorp
layer rather than re-emitted per service.

For sorted indexes, oracle additionally generates:

```go
type Order func(r Retrieve) gorp.OrderQuery[Key, Channel]

func (r Retrieve) OrderBy(o Order) Retrieve {
    r.gorp = r.gorp.OrderBy(o(r))
    return r
}

func OrderByCreatedAt(dir gorp.Direction, cursor ...telem.TimeStamp) Order {
    return func(r Retrieve) gorp.OrderQuery[Key, Channel] {
        q := r.indexes.createdAt.Ordered(dir)
        if len(cursor) > 0 {
            q = q.After(cursor[0])
        }
        return q
    }
}
```

The `Order` closure mirrors `Filter`. It captures the typed cursor at construction time
and resolves the typed `SortedQuery` against `r.indexes` when `OrderBy` invokes it. The
cursor never crosses the boundary as `any`, so there is no boxing on the pagination
path.

### 2.2.2 - Registration Site

Indexes live on the per-service `Service` struct. The `OpenService` pattern is:

```go
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
    cfg, err := config.New(DefaultServiceConfig, cfgs...)
    if err != nil {
        return nil, err
    }
    s := &Service{
        cfg:     cfg,
        indexes: newIndexes(),
    }
    s.table, err = gorp.OpenTable(ctx, gorp.TableConfig[Key, Channel]{
        DB:              cfg.DB,
        Migrations:      []migrate.Migration{ /* ... */ },
        Indexes:         s.indexes.all(),
        Instrumentation: cfg.Instrumentation,
    })
    if err != nil {
        return nil, err
    }
    return s, nil
}
```

`newIndexes` allocates one `*gorp.Lookup` (or `*gorp.Sorted`) per `@index` field, and
`(i indexes) all()` returns the heterogeneous `[]gorp.Index[K, E]` that `OpenTable`
consumes. Each Service holds its own index instances; reopening the service rebuilds
them. There is no package-level shared index state.

The per-service `Retrieve` carries a copy of the `indexes` struct so generated filter
constructors can resolve fields off `r.indexes.<field>` at evaluation time without
reaching for global state.

### 2.2.3 - Query Site

```go
// Single-value lookup.
err := s.NewRetrieve().
    Where(channel.MatchNames("sensor_1")).
    Entry(&ch).
    Exec(ctx, tx)

// Multi-value lookup (OR semantics: match any of these names).
err := s.NewRetrieve().
    Where(channel.MatchNames("sensor_1", "sensor_2", "sensor_3")).
    Entries(&results).
    Exec(ctx, tx)

// Compose with additional filters. The And is built by gorp.AndBound under
// the hood.
err := s.NewRetrieve().
    Where(channel.And(
        channel.MatchDataType(telem.Float32),
        channel.MatchVirtual(true),
    )).
    Entries(&results).
    Exec(ctx, tx)
```

The call site is type-checked at compile time. `channel.MatchNames(42)` does not
compile. Whether the underlying field is index-backed or scan-backed is invisible at the
call site; `MatchNames` happens to be index-backed because `name` carries
`@index lookup` in the schema, while a `@filter`-only field generates the same shape but
routes through a `gorp.Match` closure scan.

### 2.2.4 - Self-Contained Filters: Deferred Resolution Against the Open Tx

Indexed filters must resolve against the open tx, because the answer depends on both
committed state and the per-tx delta (§2.4.3). Rather than hand the tx down into every
filter constructor, the index's `Filter` method returns a `gorp.Filter[K, E]` whose
`resolve` field is a closure that takes the tx and returns the merged candidate keys at
query time:

```go
func (l *Lookup[K, E, V]) Filter(values ...V) Filter[K, E] {
    captured := append([]V(nil), values...)
    return Filter[K, E]{
        resolve: func(_ context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
            return l.resolveTx(tx, captured), indexedKeyMembership[K], nil
        },
    }
}
```

`Retrieve.Exec` invokes the resolver before dispatching, populates `Filter.Keys` and
`Filter.membership` from the result, and then routes through the existing `execKeys`
fast path (§2.5.0). Composition (`And` / `Or` / `Not`) propagates the resolver: if any
child filter has one, the composed filter carries a deferred resolver that fires each
child's resolver and recombines via `intersectKeys` / `unionKeys`. `Not` drops `Keys`
and the resolver because inverting a key set requires the universe of all keys.

`Retrieve` itself was not left completely unchanged. It gained a small set of additions
to support the new shape:

- A `resolveFilter` step at the top of `Exec` / `Exists` / `Count`.
- An `OrderBy` method and an internal `OrderQuery[K, E]` field for sorted-index
  iteration.
- `Validator[K, E]` batch checks via `Retrieve.Validate`.
- A `RawFilter` predicate that runs before decoding.
- A single-filter fast path in `Where` that bypasses the `And` combinator's closure
  allocation.

Adding or removing an `@index` annotation changes performance but never correctness. The
resolver always returns the same effective key set; the slow path (full table scan with
field comparison) is what callers fall back to when no index is registered.

### 2.2.5 - Why This Design?

1. **Oracle does the hard work.** The Go type system makes it awkward to have a single
   value that is both callable and interface-satisfying with access to internal state.
   Instead of fighting the type system, oracle generates exactly the right concrete code
   for each indexed field at compile time.

2. **Compile-time type safety.** The value type is checked at every call site. No string
   names, no `any` casts, no runtime reflection.

3. **No package-level state.** Index instances live on the `Service` struct, not on
   package-level vars. Two services in the same process (e.g. tests, embedded
   deployments) can each open their own indexes without global coordination.

4. **Graceful degradation.** A `MatchX` constructor whose underlying field is not
   indexed compiles to a `gorp.Match` closure scan. Performance changes, correctness
   does not.

5. **`Retrieve` does exactly one new thing per query.** A single `resolveFilter` call at
   the top of `Exec` is the only addition on the hot path for non-indexed queries, and
   it short-circuits on `filter.resolve == nil`.

6. **Clean separation of concerns.** Gorp provides primitives, including the per-tx
   overlay that handles read-your-own-writes. Oracle generates the ergonomic call-site
   shape. Hand-written code can use the primitives directly when oracle isn't available.

### 2.2.6 - Why Register at Open Time?

Registering indexes at open time (rather than lazily on first query) gives us:

1. **Deterministic startup cost.** The full scan to populate indexes happens once,
   predictably, during service open.
2. **No cold-query penalty.** Every query after open hits a warm index.
3. **Simpler concurrency model.** Indexes exist for the full lifetime of the service. No
   races around lazy initialization.

## 2.3 - Index Population

When `OpenTable` is called, after migrations complete:

1. Each registered index returns an `(insert func(E), finish func(), err error)` triple
   from its `populate` method. The implementation typically takes a write lock in
   `populate` and releases it in `finish`, so reads cannot observe a partially loaded
   index.
2. The orchestrator opens a `Nexter` over the table's prefix and walks every entry once.
3. Each decoded entry is fanned through every index's `insert` closure.
4. After the walk, every index's `finish` closure is called (always, even on error, so
   the populate-phase write locks are released cleanly).

This is a single sequential scan over the table. For a table with N entries and M
indexes, the cost is O(N) decode plus O(N) extract calls per index plus the insertion
cost for each backing structure.

`Sorted` exploits the `(insert, finish)` shape to bulk-load. `insert` appends entries
unordered to the slice, and `finish` sorts the whole slice once via `slices.SortFunc`.
This makes sorted-index population O(N log N) instead of the O(N²) cost of inserting one
entry at a time with binary search plus shift.

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
Maintenance happens at two points:

1. **Inside the writing transaction**, via `Writer.set` / `Writer.delete` calling
   `idx.stageSet` / `idx.stageDelete` for every registered index. These mutate a per-tx
   delta, not the committed index.
2. **At commit time**, via either the per-tx delta's flush hook (for table-bound
   writers) or the table's KV observer (for any other write source on the same DB). Both
   paths converge on the same committed-index update; the flush is a performance
   optimization that avoids re-decoding the entry.

### 2.4.0 - Two Hook Points: Writer Staging and the KV Observer

**Writer staging.** Table-bound queries (`Table.NewCreate` / `Table.NewUpdate` /
`Table.NewDelete`) build a `Writer` via `wrapTableWriter` that carries the table's index
list. Each `Set` / `Delete` call iterates the indexes and invokes
`idx.stageSet(tx, key, entry)` / `idx.stageDelete(tx, key)`. Staging records the pending
mutation in the per-tx delta keyed off `tx.txIdentity()`, so a `Retrieve` issued through
the same tx will see its own writes.

When the writing tx commits, the delta's cleanup hook fires `flush` on each index, which
promotes the staged values into committed storage. The flush path uses the
already-decoded indexed values directly, so there is no second decode of the entry. On
rollback (`Close` without prior `Commit`), the delta is discarded without ever touching
committed state.

**KV observer.** Each DB has a single observer attached to its underlying KV prefix when
the first table opens an index (`attachIndexObserver`). The observer deserializes
committed changes and fans every set/delete into every index's `set` / `delete` method.
This path is what keeps indexes consistent for write sources that don't go through a
table-bound writer (raw `WrapWriter`, direct KV writes, replication from peers via
Aspen).

The two paths are mutually consistent. When a table-bound writer commits, the flush
fires before or after the observer for the same change set, but they idempotently
produce the same committed-index state. The `set` path checks the reverse map and no-ops
if the value hasn't changed, so a double-apply (flush then observer, or vice versa) is
safe. To avoid actual double work in the common case, the implementation plumbs
`IgnoreHostLeaseholder` through `aspen.NewObservable`: leaseholder changes that already
flushed via the per-tx delta are filtered out of the observer stream.

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
update/delete. The reverse map costs roughly 40 bytes per entry (for a uint32 key plus a
20-byte string value plus map overhead) but makes both operations O(1).

For `BytesLookup`, the reverse map is keyed on `string` rather than the slice itself,
since `[]byte` is not strictly comparable.

### 2.4.2 - Consistency Model

Within a write tx, indexed reads see the per-tx delta on top of committed state, so a
query inside the same tx that just created a row finds it. See §2.4.3.

Outside any tx (queries against the DB directly, or through a different tx), indexes
reflect committed state plus any post-commit flushes that have run. Flushes fire from
the tx's cleanup hook immediately after the underlying KV commit returns, so the window
between data being durable in Pebble and the index reflecting it is bounded by the
cleanup-hook execution time (microseconds). For our metadata write rates (tens of writes
per second), this is indistinguishable from synchronous.

The KV observer is the second consistency path, used for writes that do not flow through
a table-bound writer. For those writes, the index lag is bounded by the observer's
notification latency, which is the same one-second-or-less window that applies to any
other observer-driven subsystem.

Across cluster nodes, the index reflects whatever Aspen has replicated so far. This is
the same consistency model as the primary data itself, so indexes do not introduce new
inconsistency windows.

### 2.4.3 - Transaction-Local Overlay (Read-Your-Own-Writes)

Earlier drafts of this RFC deferred transaction-local overlays to a future revision. The
implementation ships with the overlay because the calling code needs
read-your-own-writes: services routinely do something like "create channel, then look it
up by name within the same tx" during validation, and without the overlay the lookup
misses the just-staged write.

The overlay is implemented in three pieces:

**`txState` and the `Tx` identity.** The `gorp.Tx` interface carries an unexported
`txIdentity() *txState` method. The `*tx` implementation returns a stable per-tx
pointer; the `*DB` implementation returns nil. Subsystems that want per-tx state
register cleanup hooks on the state via `state.onCleanup`, which run in registration
order from the tx's `Commit` (with `committed=true`) or `Close` (with `committed=false`)
after the underlying KV operation completes. The `Tx` interface is sealed: the
unexported method prevents external implementations, so every code path that hands a tx
to gorp goes through one of these two implementations.

**`deltaOverlay[SK, V]` per index.** Each `Lookup` / `Sorted` / `BytesLookup` embeds a
`deltaOverlay[SK, V]` that owns a `map[*txState]*delta[SK, V]` guarded by a dedicated
mutex. `stageSet` / `stageDelete` call `overlay.loadOrCreate(state)` to get the open
tx's delta and record the pending mutation. `loadOrCreate` registers a cleanup hook the
first time it sees a given `txState`. The hook removes the delta from the map and, on
commit, calls the index's `flush` to promote staged values into committed storage.

**`resolveTx` at query time.** `Lookup.Filter` returns a `gorp.Filter[K, E]` with a
`resolve` closure. `Retrieve.Exec` invokes it with the open tx, and the closure calls
`overlay.resolve(tx, committedKeys, values)`. The merge walks the delta and applies its
three rules:

1. A staged delete removes the key from the committed result.
2. A staged set whose value matches one of the queried values adds the key.
3. A staged set whose value does not match removes the key (overrides any committed
   match for that key).

Because `state` and `forward` in the delta are authoritative within the tx, the merge is
O(|delta| + |committed|) and does not require scanning the committed structure.

**Sorted-index ordered iteration is not covered by the overlay.** The `OrderBy` /
`walkOrder` path reads the committed sorted slice directly and ignores the per-tx delta.
This is a known v1 limitation. Equality `Filter` on a `Sorted` index does merge the
overlay; only ordered iteration does not. If a use case requires ordered-with-overlay
iteration, the merge needs to be extended to produce a sorted view, tracked as a v2
follow-up.

## 2.5 - Query Integration

Because `Lookup.Filter`, `Sorted.Filter`, and `BytesLookup.Filter` produce standard
`Filter[K, E]` values with deferred resolvers, index-backed queries flow through the
existing `Where` API. `OrderBy` is the only new method on `Retrieve` that callers reach
for directly.

### 2.5.0 - Execution Path

`Retrieve.Exec` follows this dispatch order:

1. **Resolve the filter** (`r.resolveFilter`). If `filter.resolve` is non-nil (i.e. the
   filter is index-backed or composes one that is), invoke it to produce the merged
   `[]K` and the lazy-membership build function. Populate `filter.Keys` and
   `filter.membership`. No-op for bare `Match` / `MatchRaw` filters.
2. **OrderBy** (`r.execOrdered`). If the query has an `OrderQuery`, walk it, `GetMany`
   the resulting keys, run any post-filters, and return.
3. **execKeys (fast path)** (`r.execKeys`). If `WhereKeys` was called or the resolved
   filter has `Keys != nil`, fetch only those keys via `reader.GetMany` and run the
   remaining `Eval` / `Raw` predicates as post-checks.
4. **execFilter (full scan)** (`r.execFilter`). Otherwise, iterate every entry in the
   table, applying `Raw` (pre-decode) and `Eval` (post-decode) predicates.

`effectiveKeys` is the helper that combines `WhereKeys`-supplied keys with an indexed
filter's `Keys`. When both are present, it walks the `WhereKeys` slice and probes the
filter's typed O(1) membership predicate, which avoids `any` boxing on the per-key
comparison.

Filter membership is materialized lazily. `Filter.membership` carries a
`*lazyMembership[K]` that holds the keys slice plus a build function; the underlying map
is allocated on first probe via `sync.Once`. This matters heavily for large indexed key
sets that compose with smaller filters: when `intersectKeys` walks the larger side
directly (no membership probe on that side), the larger filter's membership map never
allocates. On a 12500-key filter intersected with a 1-key filter, this saves roughly 150
KB per query.

### 2.5.1 - Composing Indexed and Non-Indexed Filters

Index narrowing composes naturally with post-filtering:

```go
s.NewRetrieve().
    Where(channel.And(
        channel.MatchDataTypes(telem.Float32),
        channel.MatchVirtual(true),
    )).
    Entries(&results).
    Exec(ctx, tx)
```

At compose time, `And` inspects each child's `Keys`. If at least one is bounded, the
result `Keys` is the intersection across bounded children (children with `Keys == nil`
are unbounded and don't restrict). The intersection is computed by sorting children by
`Keys` length and walking the LARGEST one, probing the smaller siblings' lazy
memberships per candidate. This looks counterintuitive but is the memory-optimal choice:
walking a filter touches its `Keys` slice directly without ever building its membership
map. With lazy membership, the walked side's hash map is never allocated.

When any child carries a deferred `resolve` (i.e. is index-backed), the whole `And`
defers. The composed filter carries a resolver that fires every child's resolver at
query time and recombines via `intersectKeys`. The eager compose-time path is skipped to
avoid snapshotting a stale `Keys` set.

### 2.5.2 - Multiple Indexed Filters

`And` and `Or` both compose multiple indexed filters in one query. `Or` unions child key
sets when every child has `Keys` (a single unbounded child collapses the union back to
unbounded). `Not` drops `Keys` and the resolver because inverting a key set requires the
universe of all keys; the inverted filter falls through to a full scan.

The composition optimizations described in §2.5.0 (lazy membership, walking the larger
side) apply to both `And` and `Or`. For `Or`, walking is in ascending-Keys-length order
so the largest contributor's membership is never built; only smaller contributors'
memberships are probed during dedup.

### 2.5.3 - Sorted Index Queries

For ordered iteration and cursor-based pagination, oracle generates per-field `OrderByX`
constructors and a per-service `OrderBy` method on `Retrieve`:

```go
err := s.NewRetrieve().
    OrderBy(schematic.OrderByLastModified(gorp.Desc)).
    Limit(20).
    Entries(&results).
    Exec(ctx, tx)

// Resume after the previously visited cursor.
err := s.NewRetrieve().
    OrderBy(schematic.OrderByLastModified(gorp.Desc, lastSeen)).
    Limit(20).
    Entries(&results).
    Exec(ctx, tx)
```

`OrderByLastModified` is a typed closure over the cursor: passing the wrong type fails
at compile time. The closure resolves the typed `SortedQuery[K, E, V]` against
`r.indexes.lastModified` when `OrderBy` invokes it, which keeps the cursor on the typed
path with no `any` boxing.

`OrderBy` is a separate concept from `Where`. A sorted index can appear in either: in
`Where` for exact-match equality (which goes through the same overlay merge as a
`Lookup`), or in `OrderBy` for cursor iteration (which reads the committed sorted slice
directly and does not consult the overlay).

## 2.6 - Concurrency

Each index struct (`Lookup`, `Sorted`, `BytesLookup`) carries:

- A `sync.RWMutex` guarding the committed forward storage and reverse map. Reads acquire
  a read lock; `populate`, `set`, `delete`, and `flush` acquire a write lock.
- A separate `sync.Mutex` (`deltaMu`) inside the embedded `deltaOverlay`, guarding the
  `txDeltas` map. Only `loadOrCreate` and the cleanup hook take this lock; staging
  within an open delta is serialized by the writer's single-goroutine usage contract.

Splitting the locks means populate / commit-time flush (write lock on the main RWMutex)
does not block staging on other open transactions (`deltaMu`), and neither blocks
committed-state reads (read lock on the main RWMutex).

Lock granularity is per-index. This allows concurrent reads on different indexes of the
same table, and concurrent reads on the same index with writes on a different one. If
profiling shows the per-index lock overhead is a problem, the first optimization is
collapsing to per-table locks. Since the single observer per table already serializes
writes to all that table's indexes, per-table write locking loses nothing. This is a
one-line internal change with no API impact.

## 2.7 - Memory Overhead

For a lookup index on a table with N entries:

- Forward map: N entries, each a value key plus a slice of primary keys. For channels
  with ~20 byte names and uint32 keys: ~(20 + 8 + 24) = ~52 bytes per entry.
- Reverse map: N entries, each a uint32 key plus a string value: ~(4 + 20 + 16) = ~40
  bytes per entry.
- Total: ~92 bytes per entry per index.

At 100k channels with 2 lookup indexes: ~18 MB. Acceptable.

For sorted indexes, similar math applies with the addition of the sorted slice overhead
(24 bytes for the slice header, plus the per-entry `sortedEntry` cost of
`sizeof(V) + sizeof(K)`).

The per-tx delta overlay adds transient cost proportional to staged write volume. A tx
that creates 1000 channels with two indexes registered carries roughly 2000 delta
entries of ~80 bytes each (state map plus forward map slot), or roughly 160 KB for the
duration of the tx. The delta is freed in the cleanup hook after commit or rollback.

The lazy membership pattern means the membership hash maps for index-backed filters are
not allocated unless they are actually probed. For the dominant single-filter query
shape (`Where(MatchNames("foo"))`), the membership map is never built; `execKeys`
fetches `Filter.Keys` directly without going through `containsKey`.

## 2.8 - Rebuild and Recovery

Since indexes are purely derived, recovery from any corruption is simple: drop the index
and rebuild from primary data. This is triggered automatically on service open, since
open always rebuilds. There is no WAL, no crash recovery log, and no durability concern
for index data.

# 3 - What This RFC Does Not Cover

- **Detailed oracle generation logic.** The `@index lookup` / `@index sorted`
  annotations and the generated shape are described above; the specifics of the
  resolver, the import manager, and the template engine live in
  `/oracle/plugin/go/query/`.

- **Composite indexes.** Deferred until measured need. The current API does not preclude
  them: a `Lookup[K, E, V]` where `V` is a struct value already works for fixed
  composites, but oracle does not yet generate the extract function.

- **Async index population.** Deferred until startup time becomes a measured problem.

- **Sorted-index ordered iteration with the per-tx delta overlay.** The equality
  `Filter` path on `Sorted` merges the overlay; ordered cursor iteration via `OrderBy`
  reads only committed state. Extending the merge to produce a sorted view is a v2
  follow-up.

- **Full-text search indexes.** A different problem with different solutions (inverted
  indexes, trigram indexes). Not needed for our current access patterns.

- **Cross-node index coordination.** Indexes are node-local. If we ever need globally
  consistent secondary indexes, that's a distributed systems problem that belongs in
  Aspen, not gorp.

- **Index intersection at registration time.** `intersectKeys` runs per query and walks
  the larger side. A pre-computed multi-index intersection (e.g. Bloom filters per index
  pair) is a future optimization if profiling shows the per-query cost matters.

# 4 - Implementation Status

The MVP has shipped on `sy-4056-gorp-indexes`. The implementation lives in `x/go/gorp/`
(`index.go`, `index_storage.go`, `bytes_lookup.go`, `delta.go`, `filter.go`,
`order_by.go`, plus updates to `retrieve.go`, `table.go`, `writer.go`, `gorp.go`,
`observe.go`, `options.go`) and `oracle/plugin/go/query/`. Coverage:

- **Phase 1 (gorp primitives):** `Lookup`, `Sorted`, `BytesLookup` with their backing
  structures, RWMutex, populate/set/delete, reverse map, per-tx delta overlay with
  commit-time flush. Bool specialization for `Lookup`; small-int dense-array
  specialization is not implemented.
- **Phase 2 (query integration):** `Filter.resolve` deferred resolution, `Retrieve`
  dispatch through `resolveFilter`, lazy membership, `OrderBy` / `OrderQuery` /
  `SortedQuery.After`, composition via `And` / `Or` / `Not` with `intersectKeys` /
  `unionKeys`, single-filter fast path on `Retrieve.Where`.
- **Phase 3 (oracle generation):** `@index lookup` and `@index sorted` schema
  annotations, per-Service `indexes` struct, `MatchX` / `MatchXs` filter constructors
  (index-routed when the field also has `@index`), per-service `Filter` alias over
  `gorp.BoundFilter`, sorted-index `Order` closures with `OrderByX` constructors.
  Migrations of `channel`, `user`, `arc`, `device`, `label`, `rack`, `ranger`, `task`,
  and others to the generated form.

Beyond the original RFC scope, the implementation also includes:

- A standalone `BytesLookup` primitive for `[]byte`-keyed tables (ontology
  relationships).
- Per-tx delta overlay with read-your-own-writes semantics across `Lookup`, `Sorted`,
  and `BytesLookup`.
- `Validator[K, E]` batch checks on `Retrieve`.
- `RawFilter` predicate for pre-decode filtering.
- `aspen.NewObservable` with `IgnoreHostLeaseholder` so the index observer does not
  double-apply leaseholder mutations that already flushed via the per-tx delta.

# 5 - Resolved Decisions

1. **Two-layer design: gorp primitives + oracle generation.** Gorp provides sealed
   generic structs (`Lookup[K, E, V]`, `Sorted[K, E, V]`, `BytesLookup[E, V]`) plus the
   `Index[K, E]` interface. Oracle generates the per-Service `indexes` struct, the
   `MatchX` / `MatchXs` filter constructors, and the sorted-index `OrderByX` closures.

2. **Indexes live on the Service, not as package-level vars.** Each `OpenService`
   constructs its own `indexes` value via `newIndexes`. The Service owns the lifetime;
   `Retrieve` carries a copy of the bundle so filter constructors can resolve fields off
   `r.indexes.<field>` without global state.

3. **Type-specialized backing structures, with one specialization shipped.**
   `bool`-valued `Lookup` uses two slice buckets. Every other comparable type falls back
   to `map[V][]K`. `Sorted` uses a single `cmp.Ordered`-constrained slice with native
   `<` comparison, no caller-supplied comparator. Further specializations (dense arrays
   for small ints, etc.) can be added behind the existing `lookupStorage` interface
   without API impact.

4. **Filter resolution is deferred to the open tx.** `Lookup.Filter` returns a
   `Filter[K, E]` with a `resolve` closure that runs at `Retrieve.Exec` time, merges
   committed index state with the per-tx delta, and populates `Filter.Keys` plus a lazy
   membership wrapper. Composition (`And` / `Or`) propagates resolvers; `Not` drops
   them.

5. **Per-tx delta overlay for read-your-own-writes.** Earlier drafts deferred
   transaction-local overlays. The implementation ships with them because service code
   needs to look up entries it just wrote in the same tx. `Tx.txIdentity()` returns a
   stable `*txState`; each index tracks its per-tx mutations in a `deltaOverlay` keyed
   off that pointer; commit fires a cleanup hook that flushes the delta into committed
   storage (avoiding a second decode of the entry); rollback discards the delta.
   Sorted-index ordered iteration is the one path that bypasses the overlay (v1
   limitation, tracked).

6. **Lazy membership for filter `Keys`.** A `Filter[K, E]`'s O(1) membership predicate
   is only materialized when something probes it. Walking a key slice directly (the
   common path in `intersectKeys` / `unionKeys`) avoids the build entirely, which is the
   dominant memory saving on the composition path.

7. **`Retrieve` gained a small surface, not a redesign.** The hot-path addition for
   non-indexed queries is one `resolveFilter` call that short-circuits on
   `filter.resolve == nil`. New methods (`OrderBy`, `Validate`, `WhereRaw`) are additive
   and opt-in.

8. **Sorted-index range queries are deferred.** The MVP supports exact match via
   `Where(MatchX(...))` and ordering / cursor pagination via
   `OrderBy(OrderByX(dir, cursor...))`. Range queries (`.Between`, `.Before`, `.After`
   as standalone predicates on the `Sorted` struct) are left as post-filters for now.
   The sorted backing supports range operations naturally; the public API can be added
   without architectural changes.

9. **`Tx` is sealed.** The `txIdentity` method is unexported, which prevents external
   packages from substituting their own `Tx` implementation. This is deliberate:
   subsystems like the index delta overlay scope state off the identity handle, and an
   external implementation that didn't return a stable `*txState` would silently break
   the read-your-own-writes contract.
