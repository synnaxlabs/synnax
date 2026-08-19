# 54 Flux read unification

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-05

## 0 Summary

RFC 0047 moved caching into `client/ts` and left the read-surface cutover as its
remaining program. This RFC completes that cutover and amends it in three places:

1. **Two read idioms, chosen by where the read happens**: Render-path reads suspend: one
   primary `use(query): Data` plus named selectors minted from the same definition.
   Event and callback reads use the domain client directly. The `Result`-shaped primary
   and its stateful, effect, and observable variants are deleted; a non-suspending
   `useResult` remains for decoration reads (§5.1). The planned non-reactive `useGet`
   never ships: the client cache is the non-reactive read API.
2. **A list answer is one ordered set of keys per base query**: Offset and limit stop
   being cache identity and become fetch hints that fill a window of the set. Freshness
   classification then applies unchanged: enumerable lists patch membership locally on
   change events, search lists stay refetch-maintained with cancellation.
3. **Domain query definitions become worker-safe and are bound twice**: `Flux` (React)
   mints suspense hooks and selectors from them. A lowercase `flux` aether binding mints
   reactive reads for worker components, killing the hand-rolled lifecycle code that
   currently leaves worker visuals blind to deletes and renames.

## 1 Motivation

Both read idioms have coexisted since the SY-4493 chain landed, and the split is
lopsided in the wrong direction. The `Result`-shaped `useRetrieve` has ~45 call sites
while the suspended read it was meant to yield to has 3. The suspense machinery that
actually carries the Console is `useEnsureRetrieved` (13 sites, six identical
`Suspended.tsx` wrappers) feeding 60 `createSelector` definitions, each of which
hand-writes a `subscribe` closure restating what the domain's `createRetrieve` config
already knows. A selector read cold returns nothing usable, so cache warming is an
undocumented correctness requirement enforced by convention.

Lists never joined the 0047 freshness model. `List.usePager` defaults `searchTerm` to
`""`, which classifies every paged list as server-computed, so each loaded page is its
own cache entry with a whole-table subscription and a debounced wholesale refetch. One
mutation anywhere in a table refetches every loaded page. Offset is cache identity, so
an insert or delete ahead of the window shifts rows across page boundaries: duplicates
are deduped away, skipped rows are lost until the next replace. The requery path has no
cancellation, so a stale response can overwrite a newer one, and a virtualizer
fetch-more can swallow a pending search inside the shared debounce. These defects are
the visible flashing in remote selection dialogs.

On the aether side, the worker constructs its own full client, but nothing on the worker
uses `onChange` or `getCached`. Every read is retrieve-once, guarded by per-source
`valid` flags and generation counters that re-implement query lifecycle by hand. Channel
metadata is fetched on both threads through both caches for the same component, and a
rename or delete observed by the worker cache never reaches a rendered value, line, or
log.

## 2 Vocabulary

- **Render-path read**: A read whose result is returned from a component's render.
- **Event read**: A read performed inside a callback, effect, or utility, off the render
  path.
- **Definition**: A domain's `{ retrieve, subscribe, getCached }` closures over the
  domain client. React-free by construction.
- **Selector**: A named, projected render-path read minted from a definition.
- **Decoration read**: A render-path read that decorates another domain's record (a
  channel name on a graph node, a tick type on an axis) where suspension is illegal and
  no boundary can enumerate the keys. Serves the cache, fetches in the background, and
  reads absence as a loading or error `Result`.
- **Base query**: A list query stripped of `offset` and `limit`.
- **Window**: The slice of an ordered set a consumer currently renders. Filled by
  fetches; never part of cache identity.
- **Enumerable list**: A base query whose membership and order the client can compute
  (`matches` plus a replicable sort). Rule 2.
- **Search list**: A base query whose order is server-ranked (`searchTerm` set). Rule 3.

## 3 Prior art

**Reads.** RFC 0036 introduced the suspended path; RFC 0047 rebased both idioms on the
client cache and scheduled this cutover. React's own direction (`use()`,
render-as-you-fetch) treats suspension as the default read posture with boundaries
owning degradation, which is the posture adopted here.

**Lists.** react-query's infinite queries refetch every loaded page on invalidation,
which is our status quo and its best-known weakness. Relay connections are cursor-based
and robust to drift but built for append-mostly feeds, a poor fit for mutable
name-sorted sets. Firestore query listeners maintain membership server-side and push
deltas; that is the correct end state and requires a server program this RFC does not
take on. We adopt Firestore's shape with the maintenance client-side, where the change
stream already delivers every event, and keep rule 3's refetch as the seam a server-push
strategy can later replace.

## 4 Principles

1. **Two idioms and one sanctioned middle**: A read either suspends on the render path
   or calls the client in an event. The decoration read is the single exception (RD11),
   and no other shape survives.
2. **Suspension is a boundary concern**: Only boundary-level components suspend.
   Fine-grained render loops (React Flow nodes, list rows) read warm caches and never
   suspend. Where no boundary can enumerate the keys a loop will reference, the
   decoration read (`useResult`) fetches from the leaf without suspending.
3. **A mounted selector always returns `Selected`**: Deletion throws typed. A cold miss
   is a composition bug and throws loud. A transient invalidation holds the last live
   value until the maintained repair lands.
4. **A list answer is an ordered set; windows are fetch hints**: Cache identity never
   encodes pagination.
5. **Freshness classification is universal**: The three rules of RFC 0047 §5.1 govern
   lists exactly as they govern every other read.
6. **Definitions are written once and bound twice**: React and aether consume the same
   per-domain definition through thin, lifecycle-native bindings.
7. **The client surface is the only cache API** (RFC 0047 P1, P4, reaffirmed): Flux on
   either thread holds no cache and no lifecycle.

## 5 Design

### 5.0 The read model

Every current read maps onto one idiom:

| Today                               | Becomes                                     |
| ----------------------------------- | ------------------------------------------- |
| `useRetrieve` (`Result`, ~45 sites) | suspended `use`, a selector, or `useResult` |
| `useRetrieveStateful` (4)           | event read; query held as caller state      |
| `useRetrieveObservable` (8)         | event read (`client.<domain>.retrieve`)     |
| `useRetrieveEffect` (3)             | event read inside the effect                |
| `useRetrieveSuspended` (3)          | renamed to `use`                            |
| `createSelector` closures (60)      | selectors minted from the definition        |
| selector `useGet` element (~6)      | event read (`client.<domain>.getCached`)    |
| direct client calls (78 callbacks)  | unchanged; already the event idiom          |

`Result` survives on the write path, where pending state is genuinely inline
(`createUpdate`, `createDispatch`, the form save leg), and on the decoration read, where
a leaf renders its own loading and error. The form retrieve leg suspends (carried from
RFC 0047). Lists stay imperative (RFC 0047 RD13, reaffirmed).

### 5.1 The suspended primary

`createRetrieve` takes a definition and returns the read surface:

- **`use(query): Data`**: Serves a cached answer synchronously, otherwise suspends on
  the fetch; subscribes and re-renders on change; throws `DeletedError` on a tombstone
  and `DisconnectedError` when a client-requiring read has none. Today's `useSuspended`
  semantics under the primary name. Named for the call site: `Channel.use({ key })`.
- **`useEnsure(query): void`**: Warms the cache without subscribing. It is a correctness
  requirement for selector-reading subtrees (Principle 2) and a batching tool that
  collapses N child suspensions into one fetch.
- **`useResult(query): Result<Data>`**: The decoration read. Serves the cached answer,
  subscribes for changes, and starts a deduped background fetch on a cold miss, sharing
  the suspended path's in-flight and settled bookkeeping. Never suspends and never
  throws. A miss reads as loading, a deleted record and a failed fetch as an error, and
  a null client or query as disabled, so a fine-grained loop renders its own fallback
  and repairs live when the fetch or a create broadcast lands. A null query is legal,
  which is how a leaf renders before its target is chosen. The fetch starts after the
  render commits, so a render React discards cannot fetch.
- **`useInvalidate`** and **`useTombstone`** survive unchanged: the first discards a
  settled error so a boundary retry can refetch, the second reads deletion as a value
  for tombstone UX.
- The definition's `subscribe` config key is renamed **`onChange`**, matching the client
  member it delegates to.

The create-broadcast wait (a cached not-found briefly awaiting its create event) and the
typed-error contract carry over from RFC 0047 §5.5 unchanged.

Three rules the implementation forced:

- **The decoration read carries a `Result`, not `Data | undefined`**: A bare `undefined`
  collapses four states a leaf must tell apart: not fetched yet, deleted, fetch failed,
  and no client. Every other flux hook already returned a `Result`, so the decoration
  read adopting one closes the gap instead of widening it. The data inside is still
  `Data | undefined`, so this is a widening of the read surface, not an escape from one:
  the earlier rejection of `undefined`-returning selectors does not survive it.
- **Settled answers**: A definition whose `retrieve` result never lands in the cache
  (the current user, a group's ID) is kept in the per-client settled map and served from
  there. `useResult` re-renders itself on settle, since no subscription announces an
  answer the cache never saw. Such an answer is served once and never updated.
- **Answers built per call need `equal`**: A `getCached` that allocates a fresh array or
  object every call breaks `useSyncExternalStore`; the definition supplies `equal` so
  the previous answer is held when the next compares equal. The same pressure applies to
  the client: a filter query it cannot prove complete must return undefined rather than
  a partial answer, or the read never fetches the rest. Channels and devices therefore
  drop their record-store approximations entirely. Requests that name their own keys
  keep a warm path without guessing, since the record table fetches only the keys it is
  missing.

**Testing.** A hook that suspends on its first render never commits under RTL's
`renderHook`, leaving `result.current` null forever. Pluto's testutil gains
`renderHookSuspended`, which renders inside an awaited `act` and returns the same result
object.

### 5.2 Selectors

The `createRetrieve` return gains a selector factory. A selector shares the definition's
wiring and projects the answer:

```ts
export const useName = retrieve.createSelector((panel) => panel.name);
```

- Same subscription and cache access as `use`; re-renders only when the selected slice
  changes, gated by an optional equality function. The transform memoization and
  `Scope.bindSelector` sugar survive as they are.
- **Selectors never suspend**: Suspension inside React Flow rendering causes
  concurrent-rendering errors, so selectors are warm-cache reads and the parent boundary
  owns the fetch (Principle 2).
- The contract (Principle 3): cold miss throws typed (`NotFoundError` class), a
  tombstone throws `DeletedError`, and a transient invalidation while subscribed holds
  the last live value until the repair lands. The first two codify what the hand-written
  `require*` closures already do; the hold replaces a latent crash-to-boundary on cache
  gaps. The trade is real: for the duration of one refetch a selector can render a
  superseded value, the same drift the rule-3 debounced refetch already accepts,
  resolved by the same push.

A definition mints selectors in both read shapes. `createSelector` follows Principle 3
and throws on a cold miss, for subtrees a boundary has already warmed.
`createResultSelector` projects a `Result<Selected>` for decoration reads, which is what
the channel alias, name, status, and data type selectors use.

The 60 standalone `createSelector` definitions migrate onto minted selectors and
`Flux.createSelector`'s hand-supplied `subscribe`/`select` form is deleted along with
the `useGet` pair element.

### 5.3 Event reads

The domain client is the whole event-read API. `client.<domain>.retrieve` is cached,
deduped, and instant on a maintained hit; `client.<domain>.getCached` is the synchronous
pull with a keyed table fast path that needs no prior fetch. The 78 existing callback
sites already read this way; the ~21 hook-based event reads (stateful, observable,
effect, selector `useGet`) migrate to it. No `useGet` hook ships anywhere: a wrapper
adding nothing over `Synnax.use()` plus a client call is a pass-through with no boundary
to enforce.

### 5.4 Lists

**The ordered set.** The query package gains list spaces keyed by base query. An entry
holds an ordered key array; content stays in tables (RFC 0047 P5). `offset` and `limit`
are stripped from the hash and become fetch hints: a window fill retrieves a slice,
writes records through the table, and merges keys into the set at their sorted
positions.

**Freshness.** Classification is unchanged from RFC 0047 §5.1:

- **Enumerable lists** declare `matches` and a sort comparator. Set and delete events
  insert, move, or remove keys exactly. No refetch, no page entries, no drift: page
  boundaries no longer exist in the answer, only in fetching.
- **Search lists** keep rule 3: relevance ranking is not client-replicable. The refetch
  is hardened: one request covering the loaded window rather than one per page, and
  monotonic request generations with abort so a stale response can never overwrite a
  newer answer.

**Window fills.** Offset fills cannot be the end state for enumerable lists. The API
exposes no ordering contract: with no `OrderBy` set, Gorp walks keyspace order
(`x/go/gorp/retrieve.go`), which is UUID byte order for ranges, and no service retrieve
sets one. An offset into an order the client does not display fetches the wrong slice,
and an unanchored offset silently skips records that shift under concurrent mutation.
The designed fill contract is a **cursor fill**: order by a declared sorted index, after
a cursor key, limit N. Gorp already implements this (`SortedIndex.Ordered(dir)` with
`.After(cursor)`, RFC 0034); the Core program exposing it through the service layer,
API, and schemas is deferred (§7). Until it lands, enumerable lists hydrate wholesale:
the base query is fetched once and windowed client-side, which is correct for the small
domains dialogs actually enumerate (panels, ranges, devices, racks). Domains too large
to hydrate wholesale (channels) stay on hardened rule 3 until cursor fills exist. The
fill primitive is the seam: cursor fills swap in behind it with no consumer change.

`usePager` stops defaulting `searchTerm` to `""`; absence of a search term is what
admits a list to rule 2. The flux `useList` hook keeps its imperative shape
(`data: K[]`, `getItem`, `subscribe`, `retrieve`) and sources from the list space; its
per-page bookkeeping, post-await `clearPages`, and null-row placeholders are deleted.
`retrieveByKey`/`subscribeByKey` item liveness rides the table directly, closing the gap
where only ontology declared item subscriptions.

**Dialog repairs.** With the set model underneath, the remaining flash causes are
UI-layer and land with it: rendered rows persist through a search flight (the set is
replaced only when the new answer settles), `emptyContent` identity stops churning with
status keys, search and fetch-more stop sharing one debounce, the virtualizer's
measurement gap is bridged, and a reopened dialog reconciles its input with the query it
left behind.

### 5.5 The aether binding

**The definition split.** Each domain's definition moves to a worker-safe module
(lowercase namespace, no React imports), following the existing casing law that
lowercase means worker-legal. The React `queries.ts` files mint hooks from the
definition; the worker imports the definition itself.

**The binding.** A lowercase `flux` module gives aether components the same
boilerplate-kill React gets: subscribe on mount, unsubscribe on teardown, dedupe by
deep-equal query, initial value from `getCached` with a fetch on miss, and a change push
the component answers with `requestRender`. No suspense and no `Result`: aether's
lifecycle is `afterUpdate`/`afterDelete`, and its degradation paths are its own. The
three consumers with hand-rolled lifecycles migrate onto it: the control controller's
bare channel retrieves, the telem remote sources' `valid`/generation bookkeeping, and
the lineplot range provider's bespoke retrieve-plus-subscribe. Worker visuals become
live to renames and deletes for the first time.

**Topology.** Two threads keep two clients and two caches. `getCached` must stay
synchronous, so any shared-cache design forces async messaging or replication; two lazy
caches fed by the same change stream are that replication, already built. The telem
frame cache is untouched: it caches series buffers, not documents, and stays beside the
query cache by design.

### 5.6 The kill list

- `useRetrieve` (`Result` form), `useRetrieveStateful`, `useRetrieveEffect`,
  `useRetrieveObservable`, `useObservableRetrieve`, and the observable read plumbing in
  `retrieve.ts` (`useObservableBase`, result mapping on the old read path). The
  decoration read keeps a `Result`; what dies is the observable machinery under it.
- `useRetrieveSuspended` as a name (semantics move under `use`).
- `Flux.createSelector`'s standalone form, its `useGet` element, and all 60 hand-written
  subscribe closures.
- Per-page list entries, `Page` bookkeeping, post-await `clearPages`, null-row
  placeholders, the `searchTerm: ""` pager default, the shared search/fetch-more
  debounce.
- Aether hand-rolled lifecycles: controller bare retrieves, per-source `valid` flags and
  `readGeneration` counters, the lineplot range provider's private range map and
  staleness heuristic.
- The dead export tail: unused `result.ts` constructors and variant types, `FluxError`'s
  unused surface, `useListItem`'s standalone export, and the parameter/return types
  nothing imports.

## 6 Implementation phases

- **Phase 1: The definition split and the aether binding.** Mechanical relocation of
  definitions to worker-safe modules with React hooks re-minted on top, then the
  lowercase binding and the three worker migrations. Additive and green throughout;
  isolates the only wire-adjacent risk (worker bundle composition) from behavior
  changes.
- **Phase 2: The React read cutover.** Suspended semantics under the primary name,
  variants and standalone selectors deleted, selector factory in, `useGet` dead, and the
  ~120 call sites migrated. Atomic, zero coexistence: no site is left on a legacy idiom
  and no compatibility alias ships. This program ships the two phases above: the
  query-model cutover is the whole scope. The list set model (§5.4) is locked design but
  lands as its own follow-on program: list spaces in the query package, the flux list
  rebind, and the pager and dialog repairs. It follows the cutover so the new read
  substrate soaks first, and the ordered-retrieval Core program (§5.4) is sequenced
  after that, swapping in behind the fill primitive.

No wire formats, persisted schemas, or server behavior change; every phase is
client-and-pluto internal, so no migration or compatibility window is needed.

## 7 What this RFC does not cover

- **Server-side query subscriptions**: Rule 3's refetch remains the seam a server-push
  membership strategy swaps in behind.
- **The ordered-retrieval Core program**: Exposing `OrderBy` and after-cursor pagination
  through the service layer, API, schemas, and clients, backed by Gorp's sorted indexes
  (RFC 0034). Deferred; the fill primitive is designed for it to slot into.
- **Cache parity in the Python, Go, or C++ clients**:
- **Telemetry reads**: The telem client, its frame cache, and its streamer are out of
  scope.
- **Multiplayer presence and conflict resolution** (RFC 0041 remains the seam).

## 8 Resolved decisions

1. **Suspending selectors, rejected**: Suspension inside React Flow rendering causes
   concurrent-rendering errors. The cost is that `useEnsure` stays a correctness
   requirement for selector subtrees rather than an optimization; boundaries own every
   fetch.
2. **The non-reactive `useGet`, planned in RFC 0047 and removed**: It was conceived when
   the cache lived in flux. With the client owning the cache, a hook wrapping
   `Synnax.use()` plus one client call enforces no boundary; the sites it was meant to
   absorb already call the client.
3. **Throwing on transient invalidation, rejected**: The gap is cache bookkeeping, not
   document truth; deletion is a pushed state so a hold can never mask it, and RFC 0047
   §5.3 already serves the cache during gaps. The trade is one refetch window of
   possible staleness.
4. **Raw client reads on aether, rejected**: It pushed subscription lifecycle onto every
   worker component, which is exactly the boilerplate the `valid`-flag code proves
   nobody writes correctly by hand.
5. **A shared cross-thread cache, rejected**: `getCached` is synchronous; sharing forces
   async messaging or replication, and two lazy caches over one change stream are the
   cheapest correct replication.
6. **Two parallel list models, rejected as framing**: One model (ordered set plus
   windows) under the existing three-rule classification; "enumerable versus search" is
   rule 2 versus rule 3, not new machinery.
7. **Per-page cache identity, rejected**: Offset-keyed entries are why drift, refetch
   storms, and the filter-versus-offset hole exist. Pagination is a fetch concern.
8. **Suspense for lists, rejection reaffirmed** (RFC 0047 RD13): A requery under a
   boundary would blow away rendered rows on every keystroke.
9. **Inline select args on `use`, rejection carried** from the SY-4494 interview: Render
   granularity lives in named selectors only.
10. **Offset fills for enumerable lists, rejected as the end state**: The Core has no
    API-level ordering contract, and an unanchored offset skips shifted records
    silently. Cursor fills over server-declared sorted indexes are the contract; the
    Core exposure is deferred, so wholesale hydration (small domains) and hardened rule
    3 (channels) bridge until it lands.
11. **Boundary key enumeration for decoration reads, rejected**: The selector-plus-
    `useEnsure` shape requires the plot or graph boundary to enumerate every channel key
    its loop references, and the ensure re-suspends the whole subtree each time the set
    grows (picking a channel on a node blanks the editor). Hand-rolled per-site
    fetch-and-subscribe effects were rejected as the stateful variant reborn.
    `useResult` is the sanctioned middle Principle 1 names: minted once from the
    definition, stateless over the client cache, non-suspending, `Result`-reading.

## 9 Open questions

1. **Cancellation mechanics** for hardened rule-3 refetch: An `AbortSignal` threaded
   through the fetch primitive versus space-level generation counters.
2. **The window-fill contract**: How `hasMore` and totals are expressed once fills are
   cursor-based, and what a fill returns when the set already covers it.
3. **Sorted-index coverage**: Which domains declare which server-side sorted indexes
   when the ordered-retrieval program lands, and how a client comparator is checked
   against the index it mirrors.
4. **Cache memory bounds** (RFC 0047 OQ1, carried): Ordered sets add key arrays to the
   tables and tombstones already retained for the session.
5. **Stale-window presentation**: Whether rows held during a search flight render
   plainly or visibly dimmed.
