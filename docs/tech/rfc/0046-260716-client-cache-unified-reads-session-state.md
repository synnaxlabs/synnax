# 46 - Client Cache, Unified Reads, and Console Session State

**Feature Name**: Client-Owned Cache, Unified Suspense-First Reads, and Robust Console
Session State <br /> **Status**: Draft <br /> **Start Date**: 2026-07-16 <br />
**Authors**: Emiliano Bonilla <br />

# 0 - Summary

The console holds two kinds of state: document state, owned by the cluster and cached
client-side, and session state, owned by each console instance in Redux. Split brain
between them is the standing failure mode, and the read path that connects them has
fragmented into six retrieve hook variants, three read substrates, and two selector
systems. This RFC restructures the whole seam:

1. All caching and cache invalidation moves out of flux (pluto) into `client/ts`. The
   client owns one cache with change-stream invalidation, connection-epoch
   reconciliation, first-class deletion states, and optimistic write-through. Every
   client consumer, console or customer, inherits the same cache.
2. Flux becomes a React binding layer over the client cache, and its read path
   collapses to one surface: `createRetrieve` returns exactly three hooks plus a
   selector factory.

   ```ts
   useRetrieve(query: Query): Data;                                // reactive, suspends
   useGet(): (query: Query, opts?: FetchOptions) => Promise<Data>; // non-reactive
   useEnsureRetrieved(query: Query): void;                         // pre-warm
   createSelector(select, equal?): [UseSelect, UseGet];            // derived reads
   ```

   The observable substrate (`useRetrieveStateful`, `useRetrieveEffect`,
   `useRetrieveObservable`, the `Result`-returning `useRetrieve`) is deleted. The
   standalone store-subscribing `Flux.createSelector` is deleted. `createForm`'s
   bespoke retrieve leg moves onto the cache. `Result<T>` survives only on mutations.
   This reverses RFC 0036's Resolved Decision 1 and completes the migration that RFC
   deferred.
3. Session state converges to document truth through dispatched actions, never through
   intentional divergence. Deleted-while-open documents tombstone in place instead of
   tearing down views. Persisted session state is partitioned into global, per-cluster,
   and per-cluster-per-project files swapped on context switch.

# 1 - Motivation

## 1.1 - Deletion is the sharpest edge of the split brain

- A remote delete reaches a suspense view only as a thrown 404 after cache
  invalidation (`pluto/src/flux/retrieve.ts:430-454`), indistinguishable from a network
  failure. The open tab degrades to a generic "Not found" fallback
  (`console/src/feature/panel/Mosaic.tsx:33-50`).
- Non-suspense `useRetrieve` never observes deletion at all: a listener yielding null
  keeps the previous value (`pluto/src/flux/retrieve.ts:245-249`).
- Delivery is lossy even while running. `HardenedStreamer.read` silently reopens the
  stream on failure (`client/ts/src/framer/streamer.ts:284-293`); frames lost in the
  gap are never replayed, and the flux client does nothing on reconnect
  (`pluto/src/flux/base/client.ts:56-62`). A delete missed in a gap leaves a
  live-looking document forever.
- Session repair is hand-wired per domain and incomplete: range and status prune on
  remote deletes, but no panel delete synchronizer exists, the five document-keyed
  slices (schematic, line, table, arc, log) prune only on console-initiated deletes,
  and `cluster.remove` leaves a dangling selection.
- Persisted session state reloads stale keys unreconciled, and the persistence engine's
  migrator hook is never wired (`console/src/session/persist/state.ts`).

## 1.2 - The read path fragmented

RFC 0036 added the suspended read path but kept the observable hooks, betting that
consumers would migrate per call site. Fourteen months later the census says the bet
failed: `useRetrieveSuspended` has exactly one production call site
(`pluto/src/log/Log.tsx:73`) while the `Result`-returning `useRetrieve` accumulated ~45
direct call sites plus renamed aliases. The two substrates diverged: options like
`beforeRetrieve` and `addStatusOnFailure` exist only on the observable side, scope
support exists only on the observable side, and every mount of an observable hook
refetches because dedup lives only in the cache the observable path never touches.

Three costs recur:

1. **Lifecycle boilerplate at every read.** Most `useRetrieve` callers ignore the
   `Result` wrapper and render `.data?` optimistically, flashing empty instead of
   showing loading state. Only ~5 sites branch on `.variant` correctly. The wrapper
   taxes every call site and pays off almost nowhere.
2. **Selectors have no lifecycle.** The ~60 store-subscribing `Flux.createSelector`
   definitions read synchronously; a selector on an unfetched record returns
   `undefined` or throws. The workaround, a parent calling `useEnsureRetrieved` so
   children can select, works only when someone remembered the pre-warm.
3. **No sanctioned non-reactive read.** 62 event callbacks and 16 service utilities
   call `client.*.retrieve` directly, bypassing the cache entirely. Each re-fetches
   data flux often already holds.

Each hole in 1.1 was patched locally as found; each variant in 1.2 was added locally as
needed. This RFC replaces patchwork with a structure in which the holes cannot exist.

# 2 - Vocabulary

- **Document state**: the shared, broadcast state of a cluster resource. Lives in the
  client cache (RFC 0040 §1 called this the flux cache; the cache relocates here).
- **Session state**: per-console-instance UI state in Redux under
  `console/src/session/`: selections, panel arrangement, per-document view state.
- **Corpse**: the last cached value of a document at the moment its deletion is
  observed.
- **Tombstone**: a first-class cache state recording that a resource was deleted,
  retaining its corpse. Distinct from absent (never fetched) and from error.
- **Connection epoch**: one contiguous interval of healthy change-stream delivery. A
  reconnect starts a new epoch; events between epochs are lost, never replayed.
- **Convergence**: the session store is repaired by dispatched actions so that the
  stored state is the state the UI renders.
- **Projection**: stored session intent resolved against document structure at read
  time by a pure function. Legal only where intent is defined over client-owned
  structure.
- **Context**: the exclusive (cluster, project) pair the console is operating in. L0
  state is context-free, L1 is per-cluster, L2 is per-cluster-per-project.
- **Reactive read**: retrieval whose result renders; the component re-renders when the
  value changes. Always `useRetrieve` or a derived selector.
- **Non-reactive read**: retrieval inside an event callback or service utility; the
  result is used once, nothing subscribes. Always `useGet`.
- **Derived selector**: a `[useSelect, useGet]` pair created from a retrieve via
  `createSelector`, reading a slice of the query's data with equality-gated
  re-renders.

# 3 - Prior Art

**Deletion UX.** VS Code keeps a remotely-deleted file's tab open with the buffer
frozen and closes tabs on local deletes; JetBrains eagerly closes on any delete and its
trackers collect complaints about it; Notion renders deleted pages in place with a
restore banner; Grafana surfaces deletion only on next navigation and retrofitted soft
delete in 2026. We align with the VS Code and Notion camp (tombstone remote, close
local, no modals) and adapt restore to autosave: the corpse in the cache plays the role
of VS Code's dirty buffer. Per-context state swap follows VS Code's per-workspace
window state. Server-side trash, the industry substrate, is deliberately deferred (§7).

**Read path.** react-query (`useSuspenseQuery` + per-query `select`), Apollo
(`useSuspenseQuery`/`useReadQuery`), Relay (suspense-first fragments), and SWR
(`suspense: true`) have converged on suspense as the default read shape with
selector-style derived subscriptions for render granularity. We align with that
consensus. We diverge in one respect: granularity is expressed only through named
derived selectors (`useSelectName`), never an inline `select` argument on the read
hook. One pattern, matching the naming convention the codebase already uses.

# 4 - Principles

1. **One cache, owned by the client.** Caching, invalidation, deletion states, and
   reconciliation live in `client/ts`, beside the connection machinery that knows when
   they are needed. Nothing above the client caches document state.
2. **Deletes are delivered or reconciled, never assumed.** Every trust the UI places
   in a deletion state is backed by either a live change-stream event or a
   connection-epoch reconciliation pass.
3. **A remote delete of an open document mutates no session state.** The change is a
   document-state transition (present to tombstoned). Views re-render from it; focus
   and selection never shift.
4. **Two read patterns, total.** Reactive reads suspend; non-reactive reads are async
   getters. Nothing else. A third pattern is a design defect.
5. **One read substrate.** Every read, whether hook, selector, or form load, flows
   through the client cache. Per-record caches remain canonical value plumbing; they
   are not a read API.
6. **Lifecycle lives in boundaries.** Loading renders at `<Suspense>` fallbacks;
   errors, disconnection, and deletion render at error boundaries. Call sites never
   branch on load state. `Result<T>` is a mutation shape.
7. **Session state converges.** What Redux holds is what the UI renders; the action
   log is the complete causal history of every repair. Divergence is a bug with a
   visible signature, not a design feature.
8. **Unreachable is not deleted.** No pruning without a healthy connection to judge
   against.
9. **Projection is fenced.** Read-time resolution of stored intent is legal only where
   the intent is defined over client-owned structure, and each use carries a
   documented stored-versus-resolved contract.
10. **Session state is context-scoped by swap, not by shape.** Slices stay
    single-cluster and single-project; the persistence layer swaps whole files on
    context switch.

# 5 - Design

## 5.0 - The layer boundary

| Layer | Owns | Never does |
| --- | --- | --- |
| `client/ts` | Resource caches, change-stream subscription, epochs and diff reconciliation, tombstones, optimistic write-through and rollback | React, Redux |
| flux (pluto) | The read surface (`useRetrieve`, `useGet`, derived selectors), suspense bindings, `Result` mutation orchestration, forms | Caching, invalidation |
| `console/src/session/` | Redux slices, reducers owning repair policy, persistence and swap | Fetching, caching |

## 5.1 - The client cache

The per-resource unary stores, the streamer adapter, and the query cache that today
live under `pluto/src/flux/base/` move into `client/ts` as a single cache subsystem.
Framework-free; observed through per-key and store-wide `onSet` and `onDelete`
subscriptions, the same seam flux listeners use today.

**Composition.** A framework-free core engine owns the mechanism: keyed stores, the
scope registry, the streamer loop, epochs and reconciliation, and query lifecycle. It
has zero domain knowledge. The `Synnax` constructor injects the engine (alongside
dependent clients) into each domain client, which binds its own store configuration
and exposes a typed sub-store plus read-through retrieves. There is no `client.store`
namespace and no privileged flux backdoor: flux consumes the same per-domain surface
any client user gets.

**Entry states.** A cache entry is absent, present, or tombstoned. Tombstones live in
a map parallel to the entries map: `delete` moves the value there with a deletion
timestamp, `set` clears it, and `get` semantics are unchanged. `status(key)` and
`getTombstone(key)` expose the distinction, so consumers can tell "never fetched",
"exists", and "was deleted" apart without inference from errors.

**One store kind.** `UndoableUnaryStore` does not move; it dissolves. Undo/redo stacks
and dispatch bookkeeping become a dispatch subsystem operating on plain domain stores,
landing beside the action codecs in `client/ts/src/actions/`. Stacks are session-local
and dropped when a tombstone replaces their entry. The query cache shrinks to lifecycle
only: promise dedup and loading/error transitions, driven by store events; epoch
reconciliation touches stores only. `channel.CacheRetriever` is absorbed.

**Change streams.** The cache subscribes to the `sy_<resource>_set` and
`sy_<resource>_delete` signal channels through `HardenedStreamer`. Delete channels are
processed before set channels within a frame, preserving the delete-then-recreate
ordering guarantee (`pluto/src/flux/base/streamer.ts:26-32` moves with the subsystem).
Four domains (schematic, lineplot, log, table) have no delete signal channel today;
adding `sy_<x>_delete` in the Go core is the program's opening move. The streamer
opens lazily on first cache demand, encoded in the streamer layer itself, so
constructing a client never opens a stream nothing reads from.

**Connection epochs.** `HardenedStreamer` stops recovering silently: each reconnect
surfaces as a new epoch. On every epoch the cache runs a diff reconciliation: bulk
existence-check of all cached keys against the cluster, tombstoning entries that
vanished during the gap (corpse already in hand) and refreshing entries that changed.
Diff, not nuke-and-refetch: full invalidation would re-suspend every open view on any
network blip and cannot distinguish deleted from evicted. Reads during a gap serve the
cache; the epoch pass repairs on reconnect.

**Optimistic write-through.** Mutations write the cache first and roll back on failure,
relocating the rollback machinery from flux updates. A local delete tombstones
immediately in the writer's scope; writer-scope suppression (the scoped-store seam) is
preserved so a client can distinguish its own deletes from remote ones. This scope
discriminator drives local-close versus remote-tombstone UX in 5.7. Scope-qualified
cache entries carry over from the store design: two scopes never collide on an entry,
and unscoped reads keep today's behavior.

**API surface.** Caching is the transparent default: `client.<resource>` retrieves
populate and serve the cache with live invalidation. Opt-out at construction
(`cache: false` in the client config) restores direct network semantics for one-shot
scripts that want no background stream. The cache-or-not decision is a property of the
application, made once at the construction boundary.

## 5.2 - The read surface

`createRetrieve` config is unchanged: `{ name, retrieve, mountListeners?,
allowDisconnected? }`. Its return becomes:

```ts
export interface CreateRetrieveReturn<Query extends base.Query, Data extends state.State> {
  useRetrieve(query: Query): Data;
  useGet(): (query: Query, opts?: base.FetchOptions) => Promise<Data>;
  useEnsureRetrieved(query: Query): void;
  createSelector: <S>(
    select: (data: Data) => S,
    equal?: (a: S, b: S) => boolean,
  ) => [UseSelect<Query, S>, UseGetSelected<Query, S>];
}
```

All four bind to the client cache via `useSyncExternalStore`; flux holds no state of
its own.

- **`useRetrieve`** is today's `useRetrieveSuspended` promoted to the primary name:
  returns the value or suspends on the in-flight promise; errors throw to the boundary;
  concurrent reads of one query share a fetch; listener pushes re-render without
  re-suspending. With one variant left, the `Suspended` suffix carries no information.
- **`useGet`** returns a stable async getter: cache-first, falls back to the query's
  `retrieve` fn, writes the result through the cache so subsequent reactive reads hit
  it. This is the sanctioned non-reactive pattern for callbacks. Naming mirrors the
  existing selector pair convention (reactive `useSelect*` / non-reactive `useGet*`).
- **`useEnsureRetrieved`** keeps its RFC 0036 contract: suspend until cached, no
  subscription. Still the narrow tool for parents that branch on data presence or
  collapse N child fetches.
- **`createSelector`** replaces the standalone store-subscribing factory. `useSelect`
  subscribes to the cache entry, suspends when unfetched (fetching via the retrieve),
  and re-renders only when `select`'s output changes under `equal`. `useGet` (the
  selected form) is a synchronous cache read returning `S | undefined`; the
  load-bearing consumers are redux slice code and navigation (`Panel.useGetTab` at
  `console/src/session/lineplot/selectors.ts:121`, `useGetTabLeaf` at
  `console/src/session/panel/slice.ts:153`).

Deleted: `useRetrieveStateful` (4 call sites), `useRetrieveEffect` (2),
`useRetrieveObservable` (7), the `Result`-returning `useRetrieve` (~45 + aliases), and
`useObservableBase`/`useStateful`/`useDirect`/`useEffect` internals
(`pluto/src/flux/retrieve.ts:199-394`). Options that die with the substrate:
`beforeRetrieve` (the cache subsumes short-circuiting), `addStatusOnFailure` (errors go
to boundaries, not the status aggregator). The non-suspense stale-forever hole (§1.1)
closes structurally: deletion is an observable cache state, not a null the binding
refuses to store.

## 5.3 - Selector migration

All ~60 standalone `Flux.createSelector` definitions are server-backed document-field
selectors (lineplot 20, panel 13, schematic 7, arc 7, log 7, table 5, ranger 1,
ethercat 1) subscribing per-key to a store a `createRetrieve` + streamer populates.
Each migrates mechanically to the domain retrieve's `createSelector`: exported names
and `select` bodies survive; the `subscribe` implementations, per-key `store.onSet`
plumbing, are deleted because the retrieve's `mountListeners` already forwards cache
events.

`useSelectName(query)` suspending when unfetched removes most `useEnsureRetrieved`
choreography: children no longer depend on a parent having pre-warmed the cache.

The console session selectors (`console/src/session/*/selectors.ts`) are react-redux
over the Drift session store, a different system holding client-local UI state. They
are governed by §5.8, not by this migration. `useCanReverse` inside `createDispatch`
(`pluto/src/flux/dispatch.ts:98`) is internal machinery over the undo stack, not a
public selector; it keeps its subscription.

## 5.4 - Non-reactive reads: callbacks and utilities

The census found zero direct `client.*.retrieve` calls in React render paths; the 78
production bypass sites are event callbacks (62: tree `onSelect`, palette search, deep
links, snapshot clicks, task `onConfigure`) and callback-invoked plain utilities (16:
export extractors, import, `loadSchematic`). All migrate to `useGet`:

- Hooks and components capture the getter: `const getRange = Ranger.useGet()` and call
  `await getRange({ key })` inside the callback. Cache hits skip the network; misses
  fetch through the query's `retrieve`.
- Plain utilities take the getter as an argument instead, threaded from the calling
  hook. Utilities too far from React to receive one remain on `client`; with the cache
  living in the client, even these now hit the cache. Each such site still justifies
  itself (see Open Questions).

## 5.5 - Forms

`createForm`'s retrieve leg (`pluto/src/flux/form.ts:187-214`), its own
`retrieveAsync` + `useAsyncEffect` + listener mounting, moves onto the cache:

- `useForm` suspends until the record arrives, then seeds `Form.use` with mapped
  values. The config's `retrieve` returns values rather than imperatively calling
  `reset` into the form.
- Create-mode (`key == null`) short-circuits to `initialValues` without suspending.
- The save leg keeps `Result`: it is a mutation, same contract as `createUpdate`. The
  conflation of load state and save state in one `Result` ends; `useForm` returns save
  state only.
- `mountListeners` (server echo → form `set`/`reset`) is unchanged.

## 5.6 - Disconnection and deletion at boundaries

`allowDisconnected` marks queries whose `retrieve` never touches the cluster client;
those work unchanged. For client-requiring queries with no client connected,
`useRetrieve` throws a typed `DisconnectedError` (built with `errors.createTyped`,
following `x/ts/src/errors`) to the boundary. The console fallback matches the type and
renders a standard "no core connected" state. Signatures do not change: no
`Data | undefined`.

Deletion follows the same shape: a read of a tombstoned entry throws a typed
`DeletedError` carrying the resource identity and access to the corpse. Boundaries that
know about deletion (the mosaic tab boundary) render the tombstone state (§5.7); plain
boundaries render a generic deleted fallback. `useGet` rejects with the same typed
error so callbacks can branch on it.

Reconnection: error boundaries do not auto-reset, so the console boundary resets keyed
on client identity; the flux `Provider` already swap-rebuilds when the Synnax client
changes (`pluto/src/flux/Provider.tsx:83-88`). Epoch reconciliation (§5.1) handles
silent stream recoveries beneath that.

No new boundary infrastructure is required. `Errors.SuspenseBoundary`
(`pluto/src/errors/SuspenseBoundary.tsx`) and the console's panelKey-annotated wrapper
already sit at the consumer seams: modal bodies, mosaic panel content and tab names,
toolbars. Migration adds boundaries only where a migrated read has no ancestor
boundary; placement remains a per-surface UX decision per RFC 0036.

## 5.7 - Deleted-document UX

**Local delete closes.** A delete initiated in this console is deliberate; the existing
`beforeUpdate` session dispatches that close affected views remain the mechanism.

**Remote delete tombstones in place.** When the cache tombstones an open document, the
view stays. The tab renders a deleted state: frozen corpse content, a banner naming
what happened, a Close affordance, and Restore. No modal, no focus shift, no session
mutation (Principle 3). An operator mid-run never has a surface yanked from under them.

**Restore recreates from the corpse.** The console still holds the document's last
state; Restore writes it back to the cluster as a new document. Key and name-collision
semantics are open parameters. There is no server-side trash; restore is a client-side
resurrection, available to whoever had the document open.

**No corpse, no tombstone.** A persisted view reference whose target was deleted while
the console was closed has nothing to freeze and nothing to restore. Epoch
reconciliation closes it. A notification listing what was closed is polish, not a
requirement.

## 5.8 - Session convergence

Dead references are removed by ordinary dispatched actions, from two triggers through
one repair path: live `onDelete` events while running, and the epoch reconciliation
sweep after each connect. Repair policy lives in reducers (the panel `remove` guard,
`console/src/session/panel/slice.ts:92-95`, is the template): pure, unit-testable,
visible in the action log.

**The synchronizer registry.** Each domain exports a synchronizer hook that mounts its
own listening and reconciliation however it needs. The hooks compose in a module-level
const registry consumed by one mount site, replacing the scattered
`useListenForChanges` mounts in `Primary.tsx`. The registry contract: every hook covers
both triggers. A slice holding cluster references without a registry entry is a
review-visible structural omission, not a silent runtime hole.

**The projection exception.** Panel tab selection stays a projection: the slice stores
a recency-ordered intent list, and `useSelectSelection`
(`pluto/src/panel/queries.ts:295-336`) resolves it against the live tree. The
justification is structural, not delete-driven: the tree changes for reasons session
never initiates (drags between leaves, splits, panels loading after session hydrates),
and converging stored selection to it would mirror client-owned structure into Redux,
manufacturing the split brain this RFC removes. The recency list also carries MRU
information the resolved value cannot, giving most-recently-used focus repair on tab
close for free. Tab selection is the pattern's only member; new uses require the same
justification in review.

## 5.9 - Context-scoped persistence

Persisted session state is partitioned by scope and swapped on context switch. The live
store shape never changes; the persistence layer keys partitions by scope:

| Scope | Slices |
| --- | --- |
| L0 (global) | `theme`, `docs`, `color`, `cluster` |
| L1 (per-cluster) | `project`, `status` |
| L2 (per-cluster-per-project) | `panel`, `nav`, `drift`, `range`, `schematic`, `line`, `table`, `arc`, `log`, `haul` |

- L0 exists outside any cluster: preferences plus the cluster registry and selection
  needed to pick a context. L1 remembers, per cluster, which project was active and
  which statuses are favorited. L2 is the workspace: window arrangement, panel and tab
  selection, per-document view state. `range` is L2 because an active range is tied to
  the test being run in a project. `haul` is transient and persist-excluded regardless
  of scope.
- **Partitions are structured key prefixes in the one existing KV store** (`l0.<v>`,
  `l1.<clusterKey>.<v>`, `l2.<clusterKey>.<projectKey>.<v>`), each with its own version
  pointer and history ring. Per-context files were rejected: a dynamic set of store
  files adds orphan bookkeeping for no isolation the key scheme doesn't already give.
- **Switch = flush and swap, reload-free.** Entering a different cluster or project
  flushes the current partitions and loads the target's, or zero state on first visit.
  The persist middleware detects the context-key change and dispatches a hydrate action
  that a root reducer wrapper applies by replacing the swapped slices wholesale; no
  `window.location.reload` (which stays only as the `revertState`-style escape hatch).
  Returning to a context restores its whole multi-window workspace, upgrading today's
  `Panel.reset()` on project switch from destroy to preserve-and-restore.
- **Drift carve-out.** Drift merges rather than replaces: the swap keeps live
  main-window bookkeeping (label, config, main-window record) and adopts the stored
  project windows, which the drift sync middleware then reopens from the state diff.
  The reserved-window filter in `resetInitialState` is the precedent.
- **Reconciliation simplifies.** The live store only ever contains keys from the
  connected context, so the epoch sweep judges everything it sees. No cluster-awareness
  in any reconciler. Project deletion drops the project's partition instead of pruning
  surgically.
- **Migrations wire in.** Loaded partitions pass through slice migrators before
  hydration, activating the existing unwired `migrator` hook. The current single
  persisted blob migrates once into the partitioned layout (its contents become L0
  plus the current cluster and project's L1/L2 partitions).

# 6 - Implementation Phases

Ordered so the cache substrate lands first, session robustness lands on it, and the
read-surface cutover, the widest-touch migration, comes last.

**Phase 1 — client cache subsystem.** Opens with the Go core change adding the four
missing delete signal channels (`sy_schematic_delete`, `sy_lineplot_delete`,
`sy_log_delete`, `sy_table_delete`) so every cached domain has a uniform delete
signal. Then: caches, change-stream wiring, epochs with diff reconciliation,
tombstones, optimistic write-through, scope-qualified entries, construction opt-out.
Additive in `client/ts` with a live-core test suite; nothing above changes. Buys risk
isolation and a green boundary.

**Phase 2 — flux rebind.** Flux binds its existing hook surface to the client cache;
the flux store, streamer adapter, and query cache in pluto are deleted (zero
coexistence). `DisconnectedError`/`DeletedError` and boundary fallbacks land here.
Behavior-preserving for consumers; buys a reviewable unit along the flux/client seam.

**Phase 3 — deletion UX and session convergence.** Tombstone view state with Restore,
local-close wiring review, synchronizer registry with both triggers, reducer repair
completion (panel, cluster, doc slices), L0/L1/L2 partitioned persistence with the
reload-free hydrate swap and the one-time layout migration, migrator wiring, startup
close-on-reconcile. This completes the SY-4493 scope.

**Phase 4 — read cutover (atomic).** Migrate all reactive read sites: ~45 `useRetrieve`
+ aliases onto suspended semantics, 13 stateful/effect/observable sites onto
`useRetrieve`/`useGet`, ~60 selector definitions onto derived selectors, access
`useGranted` internals (136 boolean wrapper call sites keep their signatures). Delete
the observable substrate, the standalone `Flux.createSelector`, the `Result`-returning
read path, and dead exports. Zero coexistence after this lands.

**Phase 5 — forms and callbacks.** `createForm` retrieve leg onto the cache (16
definitions); 78 callback/utility direct-client sites onto `useGet`. Separated from
phase 4 because it changes behavior (fetch-on-submit becomes cache-first) rather than
read mechanics, so a bisection points at the right culprit.

Compatibility: no wire-format changes; the four delete signal channels are additive.
The only persisted-state change is the one-way, idempotent session-layout migration in
phase 3. Console and pluto specs migrate with
their subjects; live-core query specs exercise the same production paths through the
new substrate.

# 7 - What This RFC Does Not Cover

- **Lists.** `createList`/`useList` (19 definitions, 28 call sites) keep their
  imperative, `Result`-shaped API. Lists' query lifecycle is event-driven (search
  debounce, append-mode pagination `pluto/src/flux/list.ts:345-349`), suspending an
  already-rendered list on re-query is a UX regression, and inline pending state is one
  of the legitimate `Result` uses. List modernization is a separate later project;
  lists do rebind to the client cache in phase 2.
- **Server-side soft delete and trash.** Restore-from-corpse is designed so a core
  trash can later back it without rework.
- **Cache parity in the Python, Go, or C++ clients.**
- **Mutations.** `createUpdate` and `createDispatch` keep their surfaces; only their
  cache writes relocate.
- **Aether-worker reads.** The two worker-side direct retrieves
  (`pluto/src/lineplot/range/aether/provider.ts:113`,
  `pluto/src/telem/client/client.ts:137`) predate this design and live outside the
  React read path.
- **Multiplayer presence, conflict resolution, operational transformation** (RFC 0040
  §3.8 remains the seam).
- **Notification polish** for reconciliation prunes and startup closes.

# 8 - Resolved Decisions

1. **Eager teardown on remote delete, rejected.** Closing views on remote deletes is
   the most user-complained-about behavior in surveyed tools and unacceptable
   mid-operation. The trade is real: tombstones keep dead surfaces on screen until
   dismissed.
2. **Server-side trash, deferred; local trash, rejected.** Restore-from-corpse covers
   the accidental-remote-delete case without a core lifecycle program. A client-side
   trash for unopened documents would put persisted document state on the wrong side of
   the core-owns-persistence rule and help only the machine that happened to cache it.
3. **Reactive derivation as the session default, rejected.** Self-healing reads are
   real, but the costs decided it: two truths (devtools shows state the UI does not
   render), subtle read-path resolution logic, no convergence of persisted garbage, and
   a GC mechanism needed anyway. Convergence with epoch reconciliation keeps one truth
   and a causal action log. Projection survives only as the fenced exception in §5.8.
4. **Nuke-and-refetch epoch handling, rejected.** Re-suspends every open view on any
   blip and cannot distinguish deleted from evicted, which tombstone integrity
   requires.
5. **Cluster-scoping the session state shape, rejected.** Namespacing slices by cluster
   or project contaminates every reducer and selector for a concern the persistence
   layer can own entirely via file swap.
6. **Pure-data reconciler declarations, amended.** A declarative keys-to-resource map
   was proposed; per-domain synchronizer hooks in a const registry won for flexibility
   (each domain mounts what it needs) while keeping the forgot-a-wire failure
   structural.
7. **Opt-in caching, rejected.** Cache is the transparent default with
   construction-time opt-out. The trade is real: existing scripts gain a background
   stream unless they opt out, and cached reads during a gap can be briefly stale until
   the epoch pass repairs.
8. **Suspended semantics take the primary name; the `Result` read path dies.** Keeping
   both (RFC 0036's additive bet) produced one suspended call site in fourteen months
   while the `Result` path grew. The trade is real: ~5 call sites that branched on
   `.variant` for legitimate inline states move that rendering to boundaries, and the
   migration touches every read site. Accepted: coexistence demonstrably prevents
   convergence.
9. **No inline `select` argument on `useRetrieve`.** react-query offers one; we do not.
   Two granularity idioms (inline + named selectors) is exactly the two-pattern problem
   this RFC exists to kill. Named derived selectors match the codebase's existing
   `useSelectName` convention and keep selector definitions co-located in query files.
10. **Selectors read the cache, not per-record stores.** Composed query results (a
    range sugared with labels and parent) exist only as query results, never as single
    store records, so store-keyed selectors cannot express them. Per-record caches
    remain canonical plumbing; the query cache is the read surface.
11. **Lists stay imperative.** Suspense-on-requery would blow away rendered rows under
    a boundary fallback on every keystroke; React's remedy (transitions over suspense
    caches) is heavy machinery for zero user-visible gain. The codebase ends with two
    read idioms, suspended records and imperative collections, and the split is judged
    real, not incidental.
12. **Disconnection and deletion throw typed, no `Data | undefined`.** Threading
    `undefined` through every signature re-imports the null-checking the design kills;
    suspending forever looks hung. A typed throw to an aware boundary is honest and
    centralized. The trade: disconnected panels show a fallback rather than stale-empty
    content.
13. **Forms migrate their retrieve leg only.** Save state is genuinely inline mutation
    state; forcing it through boundaries would be shape-for-shape's-sake.
14. **One RFC, not two.** This document absorbs the separately-drafted unified-retrieve
    RFC (briefly numbered 0046 in its own worktree): the read surface and the cache it
    reads are one design, and sequencing them independently invited the substrate being
    built twice.

# 9 - Open Questions

1. Restore key semantics: recreate under the original key or mint a new one, and
   name-collision handling.
2. Tombstone visual design (banner layout, iconography, corpse dimming).
3. Bulk existence-check API shape per resource type for epoch reconciliation.
4. Cache memory bounds: entry eviction interacts with corpse retention (RFC 0036 left
   eviction open; tombstones sharpen it).
5. Error-reactive side effects: `console/src/feature/status/Toolbar.tsx:74` dispatches
   a remove-favorite when its status 404s. Candidate shapes: boundary `onError`
   callback, or branching on `DeletedError`. Decide during phase 4 on the concrete
   site.
6. Getter plumbing depth for plain utilities: how far a `useGet`-obtained getter
   threads into non-React code before the indirection costs more than the `client`
   bypass it removes. Decide per-site in phase 5.
7. Selector `equal` default: deep-equal (matching `useMemoDeepEqual` conventions) vs
   `Object.is`. Benchmark on the lineplot selectors, the widest fan-out.
