# 46 - Client Cache, Unified Reads, and Console Session State

**Feature Name**: Client-Owned Cache, Unified Reads, and Robust Console Session State
<br /> **Status**: Implemented <br /> **Start Date**: 2026-07-16 <br /> **Authors**:
Emiliano Bonilla <br />

# 0 - Summary

The console holds two kinds of state: document state, owned by the cluster and cached
client-side, and session state, owned by each console instance in Redux. The two drifted
apart routinely, and the read path connecting them had fragmented across three read
substrates and two selector systems. This RFC restructures the seam:

1. All caching and cache invalidation moves out of flux (pluto) into `client/ts` as the
   `query` package. The client owns one cache with change-stream invalidation,
   connection-epoch reconciliation, first-class deletion states, and optimistic
   write-through. Every client consumer, console or customer, inherits it.
2. Every domain client exposes one read surface of three members: `retrieve`,
   `onChange`, and `getCached`. Flux becomes a React binding over that surface and holds
   no cache.
3. Session state converges to document truth through dispatched actions. Documents
   deleted while open tombstone in place instead of tearing down views. Persisted
   session state is partitioned into global, per-cluster, and per-cluster-per-project
   scopes, swapped on context switch.

# 1 - Motivation

Deletion showed the problem most clearly. A remote delete reached a suspense view only
as a thrown 404 after cache invalidation, indistinguishable from a network failure, and
non-suspense reads never observed it at all because a listener yielding null kept the
previous value. Delivery was lossy while running: the hardened streamer silently
reopened on failure, frames lost in the gap were never replayed, and nothing reconciled
on reconnect, so a delete missed in a gap left a live-looking document forever. Session
repair was hand-wired per domain and incomplete, and persisted session state reloaded
stale keys unreconciled.

The read path was fragmented in the same way. Reads flowed through flux stores,
per-domain listener wiring repeated across ~25 queries files, and a store-subscribing
selector system with no lifecycle, where a selector on an unfetched record returned
undefined unless a parent remembered to pre-warm the cache. Dozens of event callbacks
bypassed caching entirely with direct network retrieves.

# 2 - Vocabulary

- **Document state**: the shared, broadcast state of a cluster resource. Lives in the
  client cache.
- **Session state**: per-console-instance UI state in Redux under
  `console/src/session/`: selections, panel arrangement, per-document view state.
- **Table**: the per-resource record store. The only copy of record content, holding
  rows and tombstones.
- **Space**: one question kind's answer cache, exposed as `query.Retrieves`
  (`retrieve`/`onChange`/`getCached`). Entries hold keys, never record copies.
- **Corpse**: the last cached value of a document at the moment its deletion is
  observed.
- **Tombstone**: a cache state recording that a resource was deleted, retaining its
  corpse. Distinct from absent (never fetched) and from error.
- **Connection epoch**: one contiguous interval of healthy change-stream delivery. A
  reconnect starts a new epoch; events between epochs are lost, never replayed.

# 3 - Prior Art

**Deletion UX.** VS Code keeps a remotely-deleted file's tab open with the buffer frozen
and closes tabs on local deletes; Notion renders deleted pages in place with a restore
banner; JetBrains eagerly closes on any delete and is complained about for it. We align
with VS Code and Notion: tombstone remote deletes, close local ones, no modals.
Per-scope state swap follows VS Code's per-workspace window state. Server-side trash is
deferred.

**Read caching.** react-query, Apollo, Relay, and SWR all converged on a normalized
client cache with change-driven invalidation beneath the framework binding, and
Firestore's `get`/`onSnapshot`/cache triple is the closest analog to the read surface
here. We follow that consensus: the cache stays below the framework, and the binding
subscribes rather than owns.

# 4 - Principles

1. **One cache, owned by the client.** Caching, invalidation, deletion states, and
   reconciliation live in `client/ts`, beside the connection machinery that knows when
   they are needed. Nothing above the client caches document state.
2. **Deletes are delivered or reconciled, never assumed.** Every deletion state the UI
   trusts is backed by a live change-stream event or a connection-epoch reconciliation
   pass.
3. **A remote delete of an open document mutates no session state.** The change is a
   document-state transition from present to tombstoned. Views re-render from it; focus
   and selection never shift.
4. **One read substrate.** Every read flows through the domain client's three read
   members. Tables remain value plumbing, not a read API.
5. **Answers hold keys, not copies.** A space's entry records which rows answer the
   question; content is composed from the table at read time or maintained as a derived
   table, so an answer cannot disagree with the record cache.
6. **Session state converges.** What Redux holds is what the UI renders, and the action
   log is the complete causal history of every repair.
7. **Unreachable is not deleted.** No pruning without a healthy connection to judge
   against.
8. **Projection is fenced.** Read-time resolution of stored intent is legal only where
   the intent is defined over client-owned structure, and each use carries a documented
   stored-versus-resolved contract.
9. **Session state is context-scoped by swap, not by shape.** Slices stay single-cluster
   and single-project; the persistence layer swaps whole partitions on context switch.

# 5 - Design

## 5.0 - The layer boundary

- **`client/ts`** owns tables, spaces, freshness rules, change-stream subscription,
  epochs and reconciliation, tombstones, optimism, and undo. It never touches React or
  Redux.
- **flux (pluto)** owns the React subscription bindings, `Result` mutation
  orchestration, forms, and lists. It never caches, invalidates, or runs query
  lifecycle.
- **`console/src/session/`** owns the Redux slices, the reducers holding repair policy,
  and scoped persistence with its swap. It never fetches or caches.

Nothing above `client/ts` knows a cache exists. A domain client's read behavior is fully
described by the three read members.

## 5.1 - The query package

`client/ts/src/query/` owns the mechanism:

- **`Cache`** creates tables and spaces, runs the lazy streamer, counts epochs, and
  schedules reconciliation. Constructed with an `openStreamer` (or `null` for a
  detached, local-only cache) and an `onError` sink.
- **`Table<K, V>`** is the single source of truth for one resource's records: a rows map
  plus a tombstones map. `delete` moves a row into a tombstone carrying its corpse and
  deletion time; `set` clears it. Sets of equal values are silenced by a per-table
  equality, so server echoes of local writes announce nothing, and writes return
  rollback destructors. A table may declare a `fetch` primitive, which powers keyed
  reads, listener backfill, and reconciliation, and a hydrate mode: `set` by default, or
  `if-absent` for dispatch-backed domains whose fetches must never clobber locally
  replayed edits.
- **Spaces** are created per question kind via `cache.queries()` and surfaced only as
  `query.Retrieves`. Entries hold `{ status, keys }`.
- **Channel listeners** are declared on the table as mirror specs built by
  `createSetListener`, `createDeleteListener`, and `createFetchListener` (key-announce
  channels resolved through the fetch primitive). Non-mirror reactions, such as the
  dispatch controller wire, use `cache.listen`.
- **Derived tables** (`cache.derive`) serve domains whose answers compose fields from
  sibling stores (status, labels, aliases): a composed-row table recomputed on
  source-table events, replacing rows wholesale so row identity remains the freshness
  signal. Seven domains use one; the rest compose at read time.

**Freshness: three rules, written once.** A space classifies each query instance when it
gains its first subscriber:

1. **Exact-key**: the query addresses one key. The entry seeds from the table's row or
   tombstone and follows key-scoped table events.
2. **Client-checkable**: the space declares a pure `matches(record, query)`. Table
   events admit and evict rows exactly, with unknown keys backfilled through the fetch
   primitive. No network.
3. **Server-computed**: the query sets a server field (`searchTerm`, `limit`, `offset`
   by default, declarable per space). Any relevant event schedules a debounced wholesale
   refetch, since only the server can evaluate the question.

Cross-domain dependencies are declared as watch entries: a foreign table plus an
`affects` function mapping its events to affected keys or a refetch. Answers are
maintained only while subscribed; an unsubscribed repeat retrieve refetches (in-flight
dedup remains), because nothing invalidates an unwatched entry. Keyed reads stay instant
regardless through the table fast path, which the always-on listeners and write-through
keep fresh.

## 5.2 - The domain read surface

Every domain client extends `query.Retriever`, which routes params to a single-record
space or a request space and exposes the whole read API:

```ts
retrieve(params): Promise<D> | Promise<D[]>;      // cached, deduped
onChange(params, handler): destructor.Destructor; // query-scoped subscription
getCached(params): Cached<D> | undefined;         // synchronous, no promise

type Cached<D> = D | Deleted<D>;
```

- **`retrieve`** hashes the query, dedupes against an in-flight promise, resolves
  instantly on a maintained hit, and on a miss fetches, populates the table, and settles
  the entry. `NotFoundError` is raised only for key-addressed reads, matching the
  server: either the fetch misses or the key's tombstone is already cached, so
  mount-after-delete is "not found" without a round trip. List queries resolve empty.
  Errors never flow through `onChange`.
- **`onChange`** fires with the new cached answer whenever it changes: a matching record
  set or deleted, or the entry repaired by epoch reconciliation. Query-scoped is the
  primitive because "which changes affect this query" is domain knowledge.
- **`getCached`** is the pull side of the same contract: it returns exactly what
  `onChange` last pushed or would push. `undefined` means nothing cached or invalidated.

Deletion is a value, not a variant envelope: `query.Deleted<D>` is a class holding the
`corpse` and `deletedAt`, guarded by `Deleted.matches(x)`. The live case is the stored
row itself, so referential stability is free (`useSyncExternalStore` sees the same
object until the answer really changes), and the deleted case is interned once per
tombstone so it is equally stable. A corpse never appears as bare data.

The single space is derived by default from the table and its fetch primitive:
`retrieve({ key })` resolves one record and deletion flips the answer to deleted. A bare
key is shorthand for `{ key }`. Domains whose single query is richer declare an explicit
single schema; params addressing a single key that fail it throw `ValidationError`
rather than falling through to a full-table request. Extra question kinds are public
named spaces typed `query.Retrieves` (ontology `children`/`parents`, ranger
`children`/`parent`/`kv`); overriding the base read surface is banned. All 21 domain
clients sit on this shape.

## 5.3 - Streams, epochs, and reconciliation

The cache subscribes to the `sy_<resource>_set` and `sy_<resource>_delete` signal
channels through `framer.HardenedStreamer`, which gained reopen and drop hooks and a
retry-forever default. Delete channels are processed before set channels within a frame,
preserving delete-then-recreate ordering. The streamer opens lazily on first demand, so
constructing a client never opens a stream nothing reads from, and demand is fired
unawaited from reads so a streamer failure never blocks them.

The streamer stops recovering silently: the epoch is 0 before streaming, 1 once live,
and increments on every reconnect. On each bump the cache runs a diff reconciliation
over every table with a fetch primitive: bulk existence-check of cached keys,
tombstoning entries that vanished during the gap (corpse already in hand) and refreshing
entries that changed. Maintained spaces refetch on the same bump. Reads during a gap
serve the cache; the epoch pass repairs on reconnect. Cluster replacement resets the
cache and returns the epoch to 0.

## 5.4 - Writes, optimism, and undo

Writes are plain domain methods taking `WriteOptions`. Mutations write the cache
optimistically and roll back on failure via the table's rollback destructors; a
successful send re-applies the write, so a stale set-channel echo cannot resurrect an
optimistically deleted or renamed record. Every local write mirrors into the tables
after the server acks, so a composed read issued right after a write sees it without
waiting on the change stream.

Undo moved into the client as `client/ts/src/actions/`: a `Controller` owns per-domain
undo/redo stacks and dispatch coalescing, and the six dispatch-backed document domains
(schematic, panel, table, lineplot, log, arc) expose `undo(key)`/`redo(key)` methods.
Flux's `createDispatch` is a thin binding that subscribes to the controller's undo-state
changes; the stacks live below React.

Caching is the transparent default. `cache: false` at construction yields a detached
cache: retrieves go straight to the network and no stream ever opens. The decision is a
property of the application, made once at the construction boundary.

## 5.5 - The flux binding

Flux holds no cache. `createRetrieve` config is
`{ name, retrieve, subscribe?, getCached?, deriveCached? }`, where `subscribe` and
`getCached` delegate to the domain client's read members, and it returns both read
idioms rebased on the one substrate:

- **`useRetrieveSuspended`** and **`useEnsureRetrieved`** serve a cached answer
  synchronously (no suspense flash on a warm mount), throw `DeletedError` on a
  tombstoned one, and otherwise suspend on the fetch. A cached not-found on a suspending
  read waits briefly for the create broadcast before settling, so create-then-open flows
  never flash a failure.
- **`useRetrieve`** and its stateful/effect/observable variants keep their
  `Result`-shaped API: they retrieve, then subscribe, converting cache pushes into
  results. A subscribed read is served from cache; an unsubscribed one refetches.
- **`createList`** stays imperative but sources purely from the client: pages subscribe
  per query, `retrieveByKey`/`subscribeByKey` give item-level liveness, and cached
  answers seed the list synchronously.
- **`createForm`**'s retrieve leg reads through the cached domain client and
  re-subscribes through `onChange`; the save leg keeps `Result`, because save state is
  genuinely inline mutation state.
- **`createSelector`** returns a `[useSelect, useGet]` pair over a caller-supplied
  subscription with an opt-in equality, for derived reads with render granularity.

Two typed errors carry degradation to boundaries: `DisconnectedError` (client-level)
when a client-requiring read has no client or the unreachable short circuit fires (RFC
0048), and flux's `DeletedError`, thrown when the cached answer is a tombstone and
carrying the corpse. Boundaries that know about deletion render the tombstone state
(§5.6); plain boundaries render a generic fallback. Signatures never widen to
`Data | undefined`.

The planned second half of the read program, promoting suspended semantics to the
primary name, deleting the `Result`-returning read path, migrating the standalone
selector definitions, and adding a sanctioned non-reactive `useGet`, has not landed; see
§6.

## 5.6 - Deleted-document UX

**Local delete closes.** A delete initiated in this console is deliberate: the shared
delete hook closes affected tabs after the server confirms, so a failed delete orphans
nothing, and remote consoles see only the tombstone.

**Remote delete tombstones in place.** When the cache tombstones an open document, the
view stays. A resource tab renders the deleted state: the corpse's name, "was deleted",
a Close affordance, and Restore for restorable types. No modal, no focus shift, no
session mutation (Principle 3). Tab names render the corpse name through the same guard,
and a `DeletedError` bubbling out of a view tab that reads someone else's resource falls
back generically: only a resource tab's own deletion gets the tombstone treatment.

**Restore recreates from the corpse.** A registry maps restorable ontology types
(schematic, lineplot, log, table, arc) to restorers that write the corpse back through
the domain client. Corpses keep their original keys, so `create` re-registers the
document under the same key and every reference to it works again. Types outside the
registry offer Close only. There is no server-side trash; restore is a client-side
resurrection available to whoever had the document open.

**No corpse, no tombstone.** A persisted view reference whose target was deleted while
the console was closed has nothing to freeze; the first reconcile pass closes it.

## 5.7 - Session convergence

Dead references are removed by ordinary dispatched actions, from two triggers through
one repair path: live delete events while running, and the reconciliation sweep after
each epoch. A synchronizer is `{ name, use }`, where `name` is human-readable and `use`
is a hook returning `{ reconcile, listen }`: `reconcile` is an idempotent boundary
repair, `listen` mounts steady-state subscriptions. Most domains build theirs from a
factory taking `{ name, onDelete, retrieveExisting, selectKeys, remove }`, keeping
repair policy in reducers: pure, unit-testable, visible in the action log.

Per-domain `SYNCHRONIZERS` consts merge into one registry mounted by
`Session.Settled.Provider` through `useSynchronizers`, above every boundary the
workspace can crash, so repair outlives what it repairs. Ten domains register: the
document slices (arc, lineplot, log, schematic, table), panel (key pruning, selection
reconcile, tab-selection reconcile, window titles), project and cluster (selection
repair and cluster-key adoption), range (persisted-only pruning plus remote-edit
mirroring), and status (favorite pruning plus notification piping). The host reads the
connection epoch from the client's connection status: reconciles run only at epoch >= 1
and re-run on every bump, so nothing ever prunes while disconnected, and a generation
guard keeps a stale pass from reporting verification.

The settled gate consumes the result: `Session.Settled.use()` is true when first contact
is made (epoch >= 1), the answering cluster matches the session's selection, no
persistence swap is in flight, and a reconcile pass has completed. RFC 0048's console
regimes render off it.

**The projection exception.** Panel tab selection stays a projection: the slice stores a
recency-ordered intent list resolved against the live tree at read time, and a reconcile
action converges it (one tab per leaf, most recent first) when the tree changes. The
justification is structural: the tree changes for reasons session never initiates, and
converging stored selection into Redux would mirror client-owned structure. The recency
list also gives most-recently-used focus repair on tab close for free. Tab selection is
the pattern's only member; new uses require the same justification in review.

## 5.8 - Scoped persistence

Persisted session state is partitioned by scope and swapped on context switch. The live
store shape never changes; the persistence layer keys partitions by scope:

| Scope     | Slices                                                                       |
| --------- | ---------------------------------------------------------------------------- |
| `global`  | cluster, color, docs, theme                                                  |
| `cluster` | project                                                                      |
| `project` | arc, drift, haul, lineplot, log, nav, panel, range, schematic, status, table |

Global exists outside any cluster: preferences plus the cluster registry and selection
needed to pick a context. The cluster scope remembers which project was last active in
that cluster. The project scope is the workspace: window arrangement, panel and tab
selection, per-document view state, and favorited statuses. Each partition names the
next one down, which is why logout clears the cluster last: by then the project
selection it flushes is already cleared.

- **Partitions are structured key prefixes in the one existing KV store** (`global.<v>`,
  `cluster.<key>.<v>`, `project.<cluster>.<project>.<v>`), each with its own version
  pointer and a four-slot history ring backing revert.
- **Switch = flush and swap, reload-free.** The persist middleware detects the
  context-key change, flushes the outgoing partitions, loads the target's (zero state on
  first visit), and dispatches a hydrate action that a root reducer wrapper applies by
  replacing the swapped slices wholesale. A generation guard makes concurrent swaps
  safe, and a `swapping` flag folds into the settled gate so the workspace never renders
  mid-swap. When the cluster changes, the target project is re-derived from the loaded
  cluster partition. `window.location.reload` survives only in the revert and clear
  escape hatches.
- **Drift merges rather than replaces.** `Drift.restoreWindows` keeps the running
  process's own bookkeeping (main window, config, unreserved pre-renders) and adopts the
  stored project windows with their runtime counters zeroed; the drift sync middleware
  then reopens them from the state diff, and window ordinals are preserved.
- **Reconciliation simplifies.** The live store only ever contains keys from the
  connected context, so the epoch sweep judges everything it sees; no reconciler is
  cluster-aware.
- **Migrations are per-slice.** Loaded partitions pass each slice through its declared
  migrator before hydration; a failed migration falls back to that slice's initial state
  instead of aborting the swap. Only the main window persists; secondary windows run a
  pass-through middleware.

# 6 - Implementation

Landed as the SY-4493 stacked PR chain, ordered so the substrate landed first and each
boundary stayed green:

1. **The query package**, additive, with nothing consuming it: tables, spaces, rules,
   streamer plumbing, and the hardened-streamer hooks.
2. **The 21-domain rebind**: every domain client onto `query.Retriever`, declarations
   replacing hand-written listener closures, derived tables for the composing domains.
3. **The connection lifecycle** (RFC 0048), whose epochs the reconciliation rides.
4. **The flux cutover**: flux stores, streamer adapter, and query lifecycle deleted;
   both read idioms rebased on the domain read surface; typed deletion and disconnection
   errors.
5. **Session robustness**: synchronizers and the settled gate, tombstone and restore UX,
   and scoped persistence with the reload-free swap.

Testing follows two tiers, both existing patterns: pure specs driving the cache with
scripted stream openers (tombstone transitions, epoch bumps, reconciliation, optimistic
rollback), and live-core specs through `createTestClient` exercising real signal
channels end to end. Console synchronizer and persistence specs run against stores built
from the production reducer.

**Remaining: the read-surface cutover.** The suspense-first promotion (suspended
semantics under the primary name, the `Result`-returning read path deleted, a
non-reactive `useGet`, standalone selector definitions migrated onto derived selectors)
is designed here but not landed; both idioms currently coexist on the one substrate. It
is the widest-touch migration and lands as its own program once the session program has
soaked.

# 7 - What This RFC Does Not Cover

- **Server-side soft delete and trash.** Restore-from-corpse is designed so a core trash
  can later back it without rework.
- **Cache parity in the Python, Go, or C++ clients.**
- **Aether-worker reads**, which live outside the React read path.
- **Multiplayer presence, conflict resolution, operational transformation** (RFC 0040
  remains the seam).

# 8 - Resolved Decisions

1. **Eager teardown on remote delete, rejected.** Closing views on remote deletes is the
   most-complained-about behavior in surveyed tools and unacceptable mid-operation. The
   cost is that tombstones keep dead surfaces on screen until dismissed.
2. **Server-side trash, deferred; local trash, rejected.** Restore-from-corpse covers
   accidental remote deletes without a core lifecycle program. A client-side trash for
   unopened documents would put persisted document state on the wrong side of the
   core-owns-persistence rule.
3. **Reactive derivation as the session default, rejected.** Self-healing reads are
   real, but they mean two truths, subtle read-path resolution, and no convergence of
   persisted garbage. Convergence with epoch reconciliation keeps one truth and a causal
   action log. Projection survives only as the fenced exception in §5.7.
4. **Nuke-and-refetch epoch handling, rejected.** It re-suspends every open view on any
   blip and cannot distinguish deleted from evicted, which tombstone integrity requires.
5. **Cluster-scoping the session state shape, rejected.** Namespacing slices by cluster
   or project contaminates every reducer and selector for a concern the persistence
   layer owns entirely via partition swap.
6. **Opt-in caching, rejected.** Cache is the default with construction-time opt-out.
   The cost is that existing scripts gain a background stream unless they opt out, and
   cached reads during a gap can be briefly stale until the epoch pass repairs.
7. **Moving the cache without moving the responsibility, rejected mid-review.** The
   first cut relocated stores into `client/ts` but left the query lifecycle in flux,
   leaking store getters and cache internals so flux could keep operating them. The
   domain client owns the lifecycle end to end and exposes the three read members only.
8. **Domain-global `onChange`, rejected as the primitive.** Which changes affect a query
   is domain knowledge; a global feed pushes that judgment onto every subscriber.
9. **A tagged `Cached` envelope, built then replaced.** The first contract wrapped every
   answer in `{ variant, data }`, so even held-state answers allocated a fresh wrapper
   per read, breaking referential stability under `useSyncExternalStore`. The landed
   encoding is the bare union `D | Deleted<D>`: the tag rides the rare case, and pull
   and push still share one contract. Future restore metadata rides the `Deleted` class
   without taxing the live path.
10. **Record copies in answers, rejected; materialized compose, the amendment.** Answers
    hold keys and compose content at read time. For domains with real composition,
    per-read recompose churned referential identity, so those maintain a derived table:
    content exists twice, but the copy is single-writer, rebuildable, and maintained by
    the same event stream as the primaries.
11. **Serving settled entries without subscribers, rejected.** Nothing invalidates an
    unwatched entry, so unsubscribed caching is permanent staleness. Unsubscribed repeat
    retrieves refetch; keyed reads stay instant through the table fast path.
12. **Set-hydration for dispatch domains, rejected.** Dispatch mutates documents
    server-side, so a fetch that plainly `set` its result clobbered documents holding
    locally replayed edits. Those tables hydrate `if-absent`.
13. **Lists stay imperative.** Suspense on requery would blow away rendered rows under a
    boundary fallback on every keystroke.
14. **Disconnection and deletion throw typed, no `Data | undefined`.** Threading
    `undefined` through every signature re-imports the null-checking the design removes.
15. **Pure-data reconciler declarations, amended.** A declarative keys-to-resource map
    was proposed; per-domain synchronizer hooks on a shared factory won, so each domain
    mounts what it needs while a forgotten wire stays a structural failure.
16. **A pluto-visible dispatch seam, rejected.** Undo stacks and coalescing live in the
    client's actions controller; flux only subscribes.

# 9 - Open Questions

1. **Cache memory bounds.** Entries are keys-only and cheap, but tables retain rows and
   tombstones for the session; eviction interacts with corpse retention.
2. **The read-surface cutover** (§6): scheduling, and whether the non-reactive `useGet`
   shape survives contact with the call sites it is meant to absorb.
3. **Refetch-on-relevant-change freshness for server-computed shapes.** Today's
   debounced wholesale refetch accepts drift the server could resolve more precisely;
   the three-member API is the seam a better strategy swaps in behind.
