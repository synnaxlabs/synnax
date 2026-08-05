# 46 - Client Cache, Unified Reads, and Console Session State

**Feature Name**: Client-Owned Cache, Unified Reads, and Robust Console Session State
<br /> **Status**: Implemented <br /> **Start Date**: 2026-07-16 <br /> **Authors**:
Emiliano Bonilla <br />

# 0 - Summary

The console holds two kinds of state. Document state is owned by the cluster and cached
client-side. Session state is owned by each console instance in Redux. The two drifted
apart routinely, and the read path connecting them had fragmented across three read
substrates and two selector systems. This RFC restructures the seam:

1. Caching and invalidation move out of flux (pluto) into `client/ts` as the `query`
   package. The client owns one cache with change-stream invalidation, connection-epoch
   reconciliation, first-class deletion states, and optimistic write-through. Every
   client consumer inherits it.
2. Every domain client exposes one read surface of three members: `retrieve`,
   `onChange`, and `getCached`. Flux becomes a React binding over it and holds no cache.
3. Session state converges to document truth through dispatched actions. Documents
   deleted while open tombstone in place instead of tearing down views. Persisted
   session state splits into global, per-cluster, and per-cluster-per-project scopes,
   swapped on context switch.

# 1 - Motivation

Deletion showed the problem most clearly. A remote delete reached a suspense view only
as a thrown 404 after cache invalidation, which is indistinguishable from a network
failure. Non-suspense reads never saw it at all, because a listener yielding null kept
the previous value. Delivery was lossy while running. The hardened streamer reopened
silently on failure, frames lost in the gap were never replayed, and nothing reconciled
on reconnect, so a delete missed in a gap left a live-looking document forever. Session
repair was hand-wired per domain, and persisted session state reloaded stale keys
unreconciled.

The read path was fragmented in the same way. Reads flowed through flux stores, listener
wiring repeated across ~25 queries files, and a selector system with no lifecycle
returned undefined on an unfetched record unless a parent pre-warmed the cache. Dozens
of event callbacks skipped the cache with direct network retrieves.

# 2 - Vocabulary

- **Document state**: the shared, broadcast state of a cluster resource. Lives in the
  client cache.
- **Session state**: per-console-instance UI state in Redux under
  `console/src/session/`, covering selections, panel arrangement, and per-document view
  state.
- **Table**: the per-resource record store. The only copy of record content.
- **Space**: one question kind's answer cache. Entries hold keys, never record copies.
- **Corpse**: the last cached value of a document when its deletion is observed.
- **Tombstone**: a cache state recording that a resource was deleted, retaining its
  corpse. Distinct from absent and from error.
- **Connection epoch**: one contiguous interval of healthy change-stream delivery. A
  reconnect starts a new epoch. Events between epochs are lost, never replayed.

# 3 - Prior Art

**Deletion UX.** VS Code keeps a remotely-deleted file's tab open with the buffer frozen
and closes tabs on local deletes. Notion renders deleted pages in place with a restore
banner. JetBrains closes eagerly on any delete and is complained about for it. We follow
VS Code and Notion: tombstone remote deletes, close local ones, no modals. Per-scope
state swap follows VS Code's per-workspace window state.

**Read caching.** react-query, Apollo, Relay, and SWR all put a normalized client cache
with change-driven invalidation beneath the framework binding, and Firestore's
`get`/`onSnapshot`/cache triple is the closest analog to the read surface here. We
follow that consensus: the cache stays below the framework, and the binding subscribes.

# 4 - Principles

1. **One cache, owned by the client.** Caching, invalidation, deletion states, and
   reconciliation live in `client/ts`, beside the connection machinery that knows when
   they are needed. Nothing above the client caches document state.
2. **Deletes are delivered or reconciled, never assumed.** Every deletion state the UI
   trusts is backed by a change-stream event or an epoch reconciliation pass.
3. **A remote delete of an open document mutates no session state.** It is a document
   transition from present to tombstoned. Views re-render; focus and selection hold.
4. **One read substrate.** Every read flows through the domain client's three read
   members. Tables are value plumbing, not a read API.
5. **Answers hold keys, not copies.** An entry records which rows answer the question.
   Content comes from the table at read time, so an answer cannot disagree with it.
6. **Session state converges.** What Redux holds is what the UI renders, and the action
   log is the complete causal history of every repair.
7. **Unreachable is not deleted.** No pruning without a healthy connection to judge
   against.
8. **Projection is fenced.** Read-time resolution of stored intent is legal only over
   client-owned structure, and each use documents its stored-versus-resolved contract.
9. **Session state is context-scoped by swap, not by shape.** Slices stay single-cluster
   and single-project. The persistence layer swaps whole partitions on context switch.

# 5 - Design

## 5.0 - The layer boundary

- **`client/ts`** owns tables, spaces, freshness rules, change-stream subscription,
  epochs and reconciliation, tombstones, optimism, and undo. It never touches React or
  Redux.
- **flux (pluto)** owns the React subscription bindings, `Result` mutation
  orchestration, forms, and lists. It never caches, invalidates, or runs query
  lifecycle.
- **`console/src/session/`** owns the Redux slices, the reducers holding repair policy,
  and scoped persistence. It never fetches or caches.

Nothing above `client/ts` knows a cache exists. The three read members fully describe a
domain client's read behavior.

## 5.1 - The query package

`client/ts/src/query/` owns the mechanism:

- **`Cache`** creates tables and spaces, runs the lazy streamer, counts epochs, and
  schedules reconciliation. It takes an `openStreamer` (or `null` for a detached,
  local-only cache) and an `onError` sink.
- **`Table<K, V>`** holds one resource's records as a rows map plus a tombstones map.
  `delete` moves a row into a tombstone carrying its corpse and deletion time, and `set`
  clears it. A per-table equality silences sets of equal values, so server echoes of
  local writes announce nothing, and writes return rollback destructors. A table may
  declare a `fetch` primitive, which powers keyed reads, listener backfill, and
  reconciliation, and a hydrate mode: `set` by default, or `if-absent` for
  dispatch-backed domains whose fetches must not clobber locally replayed edits.
- **Spaces** are created per question kind via `cache.queries()` and surfaced only as
  `query.Retrieves`. Entries hold `{ status, keys }`.
- **Channel listeners** are declared on the table as mirror specs built by
  `createSetListener`, `createDeleteListener`, and `createFetchListener`. Non-mirror
  reactions, such as the dispatch controller wire, use `cache.listen`.
- **Derived tables** (`cache.derive`) serve the seven domains whose answers compose
  fields from sibling stores (status, labels, aliases). A composed-row table recomputes
  on source-table events and replaces rows wholesale, so row identity stays the
  freshness signal. The rest compose at read time.

**Freshness: three rules, written once.** A space classifies each query when it gains
its first subscriber:

1. **Exact-key**: the query addresses one key. The entry seeds from the table's row or
   tombstone and follows key-scoped table events.
2. **Client-checkable**: the space declares a pure `matches(record, query)`. Table
   events admit and evict rows exactly, and unknown keys backfill through the fetch
   primitive. No network.
3. **Server-computed**: the query sets a server field (`searchTerm`, `limit`, `offset`
   by default, declarable per space). Any relevant event schedules a debounced wholesale
   refetch, since only the server can answer.

Cross-domain dependencies are watch entries: a foreign table plus an `affects` function
mapping its events to affected keys or a refetch. Nothing invalidates an unwatched
entry, so answers are maintained only while subscribed and an unsubscribed repeat
retrieve refetches. Keyed reads stay instant regardless through the table fast path.

## 5.2 - The domain read surface

Every domain client extends `query.Retriever`, which routes params to a single-record
space or a request space and exposes the whole read API:

```ts
retrieve(params): Promise<D> | Promise<D[]>;      // cached, deduped
onChange(params, handler): destructor.Destructor; // query-scoped subscription
getCached(params): Cached<D> | undefined;         // synchronous, no promise

type Cached<D> = D | Deleted<D>;
```

- **`retrieve`** hashes the query, dedupes against an in-flight promise, and resolves
  instantly on a maintained hit. On a miss it fetches, populates the table, and settles
  the entry. `NotFoundError` is raised only for key-addressed reads, matching the
  server: either the fetch misses, or the key's tombstone is already cached, which makes
  mount-after-delete a "not found" without a round trip. List queries resolve empty, and
  errors never flow through `onChange`.
- **`onChange`** fires with the new cached answer whenever it changes: a matching record
  set or deleted, or the entry repaired by reconciliation. Query-scoped is the primitive
  because which changes affect a query is domain knowledge.
- **`getCached`** is the pull side of the same contract. It returns exactly what
  `onChange` last pushed or would push. `undefined` means nothing cached or invalidated.

Deletion is a value, not a variant envelope. `query.Deleted<D>` is a class holding the
`corpse` and `deletedAt`, guarded by `Deleted.matches(x)`. The live case is the stored
row itself and the deleted case is interned once per tombstone, so both are
referentially stable. A corpse never appears as bare data.

The single space is derived by default from the table and its fetch primitive, so
`retrieve({ key })` resolves one record and deletion flips the answer to deleted. A bare
key is shorthand for `{ key }`. Domains whose single query is richer declare an explicit
single schema, and params addressing a single key that fail it throw `ValidationError`
rather than falling through to a full-table request. Extra question kinds are public
named spaces (ontology `children`/`parents`, ranger `children`/`parent`/`kv`), but
overriding the base read surface is banned. All 21 domain clients sit on this shape.

## 5.3 - Streams, epochs, and reconciliation

The cache subscribes to the `sy_<resource>_set` and `sy_<resource>_delete` signal
channels through `framer.HardenedStreamer`, which gained reopen and drop hooks and a
retry-forever default. Delete channels are processed before set channels within a frame,
preserving delete-then-recreate ordering. The streamer opens lazily on first demand, so
constructing a client opens nothing, and demand is fired unawaited so a streamer failure
never blocks a read.

The streamer stops recovering silently. The epoch is 0 before streaming, 1 once live,
and increments on every reconnect. On each bump the cache diffs every table with a fetch
primitive: it bulk-checks cached keys, tombstones entries that vanished during the gap
(the corpse is already in hand), and refreshes entries that changed. Maintained spaces
refetch on the same bump. Reads during a gap serve the cache. Cluster replacement resets
the cache and returns the epoch to 0.

## 5.4 - Writes, optimism, and undo

Writes are plain domain methods taking `WriteOptions`. Mutations write the cache
optimistically and roll back on failure through the table's rollback destructors. A
successful send re-applies the write, so a stale set-channel echo cannot resurrect an
optimistically deleted or renamed record, and a read issued right after a write sees it
without waiting on the change stream.

Undo moved into the client as `client/ts/src/actions/`. A `Controller` owns per-domain
undo/redo stacks and dispatch coalescing, and the six dispatch-backed document domains
(schematic, panel, table, lineplot, log, arc) expose `undo(key)`/`redo(key)`. Flux's
`createDispatch` only subscribes to the controller's undo-state changes.

Caching is the transparent default, decided once at the construction boundary.
`cache: false` yields a detached cache: retrieves go straight to the network and no
stream opens.

## 5.5 - The flux binding

Flux holds no cache. `createRetrieve` takes
`{ name, retrieve, subscribe?, getCached?, deriveCached? }`, where `subscribe` and
`getCached` delegate to the domain client, and returns both read idioms on the one
substrate:

- **`useRetrieveSuspended`** and **`useEnsureRetrieved`** serve a cached answer
  synchronously, throw `DeletedError` on a tombstoned one, and otherwise suspend on the
  fetch. A cached not-found waits briefly for the create broadcast before settling, so
  create-then-open never flashes a failure.
- **`useRetrieve`** and its stateful/effect/observable variants keep their
  `Result`-shaped API. They retrieve, then subscribe, turning cache pushes into results.
- **`createList`** stays imperative but sources purely from the client. Pages subscribe
  per query, `retrieveByKey`/`subscribeByKey` give item-level liveness, and cached
  answers seed the list synchronously.
- **`createForm`**'s retrieve leg reads through the domain client and re-subscribes
  through `onChange`. Its save leg keeps `Result`, which is genuine inline mutation
  state.
- **`createSelector`** returns a `[useSelect, useGet]` pair over a caller-supplied
  subscription, for derived reads with render granularity.

Two typed errors carry degradation to boundaries. `DisconnectedError` fires when a
client-requiring read has no client or the unreachable short circuit trips (RFC 0048).
Flux's `DeletedError` carries the corpse when the cached answer is a tombstone.
Deletion-aware boundaries render the tombstone state (§5.6); plain ones render a generic
fallback. Signatures never widen to `Data | undefined`.

The second half of the read program has not landed; see §6.

## 5.6 - Deleted-document UX

**Local delete closes.** A delete initiated in this console is deliberate. The shared
delete hook closes affected tabs after the server confirms, so a failed delete orphans
nothing, and remote consoles see only the tombstone.

**Remote delete tombstones in place.** When the cache tombstones an open document, the
view stays. A resource tab renders the deleted state: the corpse's name, "was deleted",
a Close affordance, and Restore for restorable types. No modal, no focus shift, no
session mutation (Principle 3). Tab names render the corpse name through the same guard.
A `DeletedError` from a view tab reading someone else's resource falls back generically,
so only a resource tab's own deletion gets the tombstone treatment.

**Restore recreates from the corpse.** A registry maps restorable ontology types
(schematic, lineplot, log, table, arc) to restorers that write the corpse back through
the domain client. Corpses keep their original keys, so every reference to the document
works again. Types outside the registry offer Close only.

**No corpse, no tombstone.** A persisted view reference whose target was deleted while
the console was closed has nothing to freeze. The first reconcile pass closes it.

## 5.7 - Session convergence

Dead references are removed by ordinary dispatched actions, from two triggers through
one repair path: live delete events while running, and the reconciliation sweep after
each epoch. A synchronizer is `{ name, use }`, where `use` is a hook returning
`{ reconcile, listen }`. `reconcile` is an idempotent boundary repair; `listen` mounts
steady-state subscriptions. Most domains build theirs from a factory taking
`{ name, onDelete, retrieveExisting, selectKeys, remove }`, which keeps repair policy in
reducers: pure, unit-testable, and visible in the action log.

Per-domain `SYNCHRONIZERS` consts merge into one registry mounted by
`Session.Settled.Provider`, above every boundary the workspace can crash, so repair
outlives what it repairs. Ten domains register: arc, lineplot, log, schematic, table,
panel, project, cluster, range, and status. Reconciles run only at epoch >= 1 and re-run
on every bump, so nothing prunes while disconnected, and a generation guard keeps a
stale pass from reporting verification.

`Session.Settled.use()` is true when first contact is made (epoch >= 1), the answering
cluster matches the session's selection, no persistence swap is in flight, and a
reconcile pass has completed. RFC 0048's console regimes render off it.

**The projection exception.** Panel tab selection stays a projection. The slice stores a
recency-ordered intent list resolved against the live tree at read time, and a reconcile
action converges it (one tab per leaf, most recent first) when the tree changes. The
tree changes for reasons session never initiates, so converging stored selection into
Redux would mirror client-owned structure. Tab selection is the pattern's only member;
new uses require the same justification in review.

## 5.8 - Scoped persistence

Persisted session state is partitioned by scope and swapped on context switch. The live
store shape never changes; only the persistence layer knows about scopes:

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

- **Partitions are structured key prefixes in the one existing KV store** (`global`,
  `cluster.<key>`, `project.<cluster>.<project>`), each with a four-slot history ring
  backing revert and a `.slot` pointer naming the live slot. The pointer is unrelated to
  the schema version each slice carries in its own value.
- **Switch = flush and swap, reload-free.** The persist middleware detects the
  context-key change, flushes the outgoing partitions, loads the target's (zero state on
  first visit), and dispatches a hydrate action that replaces the swapped slices
  wholesale. A generation guard makes concurrent swaps safe, and a `swapping` flag folds
  into the settled gate so the workspace never renders mid-swap.
  `window.location.reload` survives only in the revert and clear escape hatches.
- **Drift merges rather than replaces.** `Drift.restoreWindows` keeps the running
  process's own bookkeeping (main window, config, unreserved pre-renders) and adopts the
  stored project windows with their runtime counters zeroed. The sync middleware then
  reopens them from the state diff, preserving window ordinals.
- **Reconciliation simplifies.** The live store only ever contains keys from the
  connected context, so the epoch sweep judges everything it sees. No reconciler is
  cluster-aware.
- **Migrations are per-slice.** Loaded partitions pass each slice through its declared
  migrator before hydration, and a failed migration falls back to that slice's initial
  state instead of aborting the swap. Only the main window persists.

# 6 - Implementation

Landed as the SY-4493 stacked PR chain, ordered so the substrate landed first and each
boundary stayed green:

1. **The query package**, additive, with nothing consuming it.
2. **The 21-domain rebind**: every domain client onto `query.Retriever`, declarations
   replacing hand-written listener closures, derived tables for the composing domains.
3. **The connection lifecycle** (RFC 0048), whose epochs the reconciliation rides.
4. **The flux cutover**: flux stores, streamer adapter, and query lifecycle deleted;
   both read idioms rebased on the domain read surface; typed deletion and disconnection
   errors.
5. **Session robustness**: synchronizers and the settled gate, tombstone and restore UX,
   and scoped persistence with the reload-free swap.

Testing follows two existing tiers: pure specs driving the cache with scripted stream
openers, and live-core specs through `createTestClient` exercising real signal channels
end to end. Console synchronizer and persistence specs run against stores built from the
production reducer.

**Remaining: the read-surface cutover.** Suspended semantics move under the primary
name, the `Result`-returning read path goes away, a non-reactive `useGet` arrives, and
the standalone selector definitions migrate onto derived selectors. Both idioms coexist
on the one substrate until then. It is the widest-touch migration and lands as its own
program once the session program has soaked.

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
3. **Reactive derivation as the session default, rejected.** It means two truths, subtle
   read-path resolution, and no convergence of persisted garbage. Convergence with epoch
   reconciliation keeps one truth and a causal action log.
4. **Nuke-and-refetch epoch handling, rejected.** It re-suspends every open view on any
   blip and cannot distinguish deleted from evicted, which tombstone integrity requires.
5. **Cluster-scoping the session state shape, rejected.** Namespacing slices by cluster
   or project contaminates every reducer and selector for a concern the persistence
   layer owns entirely.
6. **Opt-in caching, rejected.** Cache is the default with construction-time opt-out.
   The cost is that existing scripts gain a background stream unless they opt out, and
   cached reads during a gap can be briefly stale.
7. **Moving the cache without moving the responsibility, rejected mid-review.** The
   first cut relocated stores into `client/ts` but left the query lifecycle in flux,
   leaking store getters and cache internals. The domain client owns the lifecycle end
   to end.
8. **Domain-global `onChange`, rejected as the primitive.** Which changes affect a query
   is domain knowledge; a global feed pushes that judgment onto every subscriber.
9. **A tagged `Cached` envelope, built then replaced.** Wrapping every answer in
   `{ variant, data }` allocated a fresh wrapper per read, breaking referential
   stability under `useSyncExternalStore`. The bare union `D | Deleted<D>` keeps the tag
   on the rare case, and future restore metadata rides the `Deleted` class.
10. **Record copies in answers, rejected; materialized compose, the amendment.** For
    domains with real composition, per-read recompose churned referential identity, so
    those maintain a derived table. Content exists twice, but the copy is single-writer
    and rebuildable.
11. **Serving settled entries without subscribers, rejected.** Nothing invalidates an
    unwatched entry, so unsubscribed caching is permanent staleness.
12. **Set-hydration for dispatch domains, rejected.** Dispatch mutates documents
    server-side, so a fetch that plainly `set` its result clobbered documents holding
    locally replayed edits. Those tables hydrate `if-absent`.
13. **Lists stay imperative.** Suspense on requery would blow away rendered rows under a
    boundary fallback on every keystroke.
14. **Disconnection and deletion throw typed, no `Data | undefined`.** Threading
    `undefined` through every signature re-imports the null-checking the design removes.
15. **Pure-data reconciler declarations, amended.** A declarative keys-to-resource map
    was proposed; per-domain synchronizer hooks on a shared factory won, so a forgotten
    wire stays a structural failure.
16. **A pluto-visible dispatch seam, rejected.** Undo stacks and coalescing live in the
    client's actions controller; flux only subscribes.

# 9 - Open Questions

1. **Cache memory bounds.** Entries are keys-only and cheap, but tables retain rows and
   tombstones for the session, and eviction interacts with corpse retention.
2. **The read-surface cutover** (§6): scheduling, and whether the non-reactive `useGet`
   shape survives contact with the call sites it is meant to absorb.
3. **Refetch-on-relevant-change freshness for server-computed shapes.** The debounced
   wholesale refetch accepts drift the server could resolve more precisely. The
   three-member API is the seam a better strategy swaps in behind.
