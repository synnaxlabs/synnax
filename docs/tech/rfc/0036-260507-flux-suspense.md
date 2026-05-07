# 36 - Flux Suspense Architecture

**Feature Name**: Suspense-First Flux Substrate <br /> **Status**: Proposed <br />
**Start Date**: 2026-05-07 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

Pluto's Flux substrate is restructured around React 19's Suspense primitive. The two
existing read APIs (`useRetrieve` async and `useSelect*` sync) collapse into a single
suspending read. Consumers receive the data directly with no `{data, status, error}`
wrapper, no null guards, and no inline loading state. A single `<Flux.Boundary>`
component wraps Suspense and an error boundary, producing a default fallback that plugs
into the existing `xstatus.Status` display pipeline. Real-time channel listeners
continue to mutate cache values in place; once a record is hydrated it is never returned
to a pending state. Mutations keep the cache-write-through optimistic pattern that
powers Drift multi-window sync. The migration is atomic: all consumer call sites move in
a single PR, with most call site changes mechanical.

# 1 - Vocabulary

- **Read** - A hook call that returns store data (`useRetrieve`, `useSelect*`,
  `useList`).
- **Hydration** - Moving a cache entry from the `pending` state (in-flight promise) to
  the `hydrated` state (resolved value).
- **One-way hydration** - Once an entry is hydrated, it never returns to `pending`.
  Updates mutate the value in place. Deletion is surfaced as a thrown status, not a
  state transition back to pending.
- **Boundary** - The `<Flux.Boundary>` component, which bundles a Suspense boundary and
  an error boundary into one consumer-facing primitive.
- **Status** - `xstatus.Status` from `x/ts/src/status`, the canonical app-wide shape for
  errors, successes, and notifications.
- **Listener** - A `Flux.ChannelListener` registered on a Synnax signal channel (e.g.,
  `sy_schematic_set`). Listeners are configured at provider mount and run for the life
  of the app.

# 2 - Motivation

## 2.0 - The Two-API Split Forces Loading-State Plumbing on Every Consumer

The current substrate exposes two read shapes. `useRetrieve({key})` is async and returns
`Result<T> = {data: T | null, status: xstatus.Status, error?: Error}`. Selectors built
with `createSelector` are synchronous reads against the same flux store, returning the
selected slice directly.

The async/sync split means every consumer of `useRetrieve` writes a null guard:

```ts
const { data } = useRetrieve({ key });
if (data == null) return null;
return <UI data={data} />;
```

And every consumer that wants to render loading or error UI inline writes:

```ts
const { data, status, error } = useRetrieve({ key });
if (status.variant === "loading") return <Spinner />;
if (error != null) return <ErrorView error={error} />;
return <UI data={data} />;
```

The plumbing repeats across hundreds of call sites. Worse, selectors that read fields
from a record (e.g., `useSelectSnapshot({key})`) have no lifecycle awareness: they
synchronously read the store, find no entry, and either return `undefined` or throw
`NotFoundError` depending on the selector's choice. The "loading" and "doesn't exist"
cases are indistinguishable at the selector layer, because the selector has no idea
whether a fetch is in flight.

## 2.1 - The Toolbar Bug from PR #2291

The selector contract drift surfaced concretely in the schematic toolbar. The toolbar
lives in Console, mounted by the layout system on the active layout key. It reads
element-level state via `useSelectElementConfig({ key, elKey })` from
`pluto/src/schematic/queries.ts`, which throws `NotFoundError` when the parent schematic
is not in the cache. The toolbar is not a child of Pluto's `<Schematic>` and never sees
the `data == null` gate that the canvas uses; nothing in the toolbar tree calls
`useRetrieve`. On first navigation into a schematic, the toolbar renders in the same
frame as the Pluto canvas. The canvas waits for retrieve and returns null. The toolbar
does not, throws on the selector, and crashes the panel render.

The selector is the visible failure, but the root cause is structural: the cache has no
way to model a key that exists on the server but has not yet been fetched into the local
store. Throwing, returning undefined, or returning a sentinel are all guesses about
which state we are in.

## 2.2 - Suspense Models the Lifecycle Once, at the Right Layer

React's Suspense primitive exists to express exactly this boundary. A read returns the
value if it is available and suspends if it is not, with a parent boundary deciding what
to render in the meantime. A consumer never has to ask "is this loaded yet" because the
consumer never executes with un-loaded data.

Adopting Suspense lets us delete loading-state plumbing from every read site, collapse
the two-API split into a single shape, and resolve the toolbar bug structurally rather
than by patching one selector at a time.

# 3 - Principles

- **One read API.** Reads either return the value or suspend. There is no escape hatch
  hook that returns `T | undefined`. Cases that need to render before data is loaded use
  the boundary's `loading` fallback.
- **Hydration is one-way.** Once a record is in the cache, it stays in the cache.
  Real-time updates mutate the value field of the entry; they never move the entry back
  to `pending`. Re-suspending mid-session is forbidden.
- **The substrate does not invent error types.** Reads can throw any `Error` (typically
  the typed errors from `@synnaxlabs/client`) or a `Status` directly. The boundary
  normalizes both to `xstatus.Status` using the existing `errorToStatus` machinery and
  renders via the existing `Status.Summary` component.
- **Mutations write through the cache.** The optimistic write + rollback pattern used
  today by `createUpdate` continues unchanged. Mutations only mutate the `value` field
  of a hydrated entry. Drift multi-window sync depends on the shared cache surface and
  is unaffected.
- **One boundary primitive, consumer-placed.** The substrate exports `<Flux.Boundary>`.
  Where to place it is a UX decision the consumer makes per-panel, not a topology the
  substrate prescribes.
- **Atomic migration.** All consumer call sites move in one PR. There is no coexistence
  period with both APIs alive.

# 4 - Design

## 4.0 - Cache State Machine

A cache entry is in exactly one of two states.

| State      | Holds        | Read behavior             |
| ---------- | ------------ | ------------------------- |
| `pending`  | `Promise<T>` | `use(promise)` -> suspend |
| `hydrated` | `T`          | return `T`                |

There is no `deleted` state and no `error` state. A delete event causes the next read to
throw a `DeletedStatus` (or equivalent), which the error boundary catches. A failed
initial fetch causes the pending entry's promise to reject, which Suspense unwraps into
a thrown error caught by the boundary.

There is no `stale` state. Real-time channel listeners are the freshness mechanism; the
cache trusts that listeners will deliver updates. Stale-while-revalidate adds no value
when the live tail is reliable.

## 4.1 - Read API

The public hook names are preserved. Their return shapes change.

```ts
// Single-record read
const data = useRetrieve({ key });
//    ^ T. Suspends if entry is pending. Throws if the entry's promise rejected
//      or if a real-time delete event flagged the key.

// Selector read
const snapshot = useSelectSnapshot({ key });
//    ^ boolean. Same suspension semantics as useRetrieve.
//      Selectors no longer throw NotFoundError.
```

`useRetrieve` is implemented in terms of `use()` (React 19). On entry, the hook looks up
the cache by key. If the entry is `hydrated`, it returns the value. If the entry is
`pending`, it calls `use(promise)`, which suspends. If the entry is absent, the hook
constructs the promise (via the configured `retrieve` implementation), writes a
`pending` entry, and calls `use(promise)`. Concurrent reads of the same key share the
in-flight promise via the cache lookup.

Selectors are implemented in terms of `useRetrieve` plus a synchronous slice function.
The selector first ensures the parent record is hydrated (suspending if not), then
derives and returns the slice. The selector subscribes to store mutations using
`useSyncExternalStoreWithSelector` for the steady-state case; once hydrated, the
selector never re-suspends.

## 4.2 - List API

`useList` suspends on the first page only. Subsequent pages are async-but-non-
suspending.

```ts
const channels = useList({ workspace });
//    ^ ListHandle<Channel>:
//      { items: Channel[]; hasMore: boolean; loadMore: () => Promise<void>;
//        isLoadingMore: boolean }
```

`loadMore()` returns a promise that resolves when the next page lands. Consumers render
an inline "loading more..." indicator using `isLoadingMore` rather than suspending the
whole list.

A list read populates the per-key cache as a side effect. A `useChannels()` call that
fetches a page of 20 channels writes 20 entries into the channel cache. A subsequent
`useChannel({key})` for an item already in the list hits the cache and returns
immediately without suspending. List handles and per-key entries share one underlying
store; real-time creates and deletes fan out to both layers.

## 4.3 - Real-Time Updates

The streamer remains always-on at provider mount, with all configured channel listeners
subscribed for the life of the app. The current `streamer.ts` wiring (open
`HardenedStreamer` on the union of listener channels, parse frames with the listener's
Zod schema, dispatch to `onChange`) is unchanged.

Listener `onChange` handlers mutate the `value` field of hydrated entries:

```ts
onChange: ({ changed, store }) => {
  const current = store.schematics.get(changed.key);
  if (current == null) return; // entry not loaded; ignore
  const next = applyChange(current, changed);
  store.schematics.set(changed.key, next); // value swap, status stays hydrated
};
```

If a listener fires for a key that is not in the cache, it is ignored. The next read of
that key will trigger a fresh fetch; the listener's payload is not used to hydrate,
because the listener payload is typically a delta rather than the full record.

A delete event is the one case that transitions an entry out of the hydrated state

- and only out of it. The listener marks the entry as deleted (or removes it and records
  a tombstone), which causes the next read to throw a deletion status. The error
  boundary catches it and renders the deleted-state UI.

## 4.4 - Mutations

`createUpdate` is unchanged in shape. Mutations write optimistically into the store,
perform the server roundtrip, and roll back on failure using the existing destructor
stack.

The single new constraint: a mutation must never move a cache entry from `hydrated` to
`pending`. Optimistic updates and rollbacks operate on the `value` field only.
Subscribers re-render via the existing `onSet` notification path. Mutations never
re-suspend.

`useOptimistic` from React 19 is not used. Its scope is component-local React state;
Synnax's optimistic values must be visible across windows via Drift, which requires the
optimistic value to live in the shared cache.

## 4.5 - Errors and the Boundary

The substrate's contract with the boundary is: any thrown value other than a promise
lands at the boundary as an `xstatus.Status`. The boundary normalizes the thrown value:

```ts
const status = isStatus(thrown) ? thrown : errorToStatus(thrown as Error);
```

`errorToStatus` (from `x/ts/src/status/status.ts`) already handles errors that implement
`toStatus()` and falls back to a generic `error` variant for those that do not. No new
error types are introduced.

The boundary is exposed as:

```ts
<Flux.Boundary
  loading={<Skeleton />}
  error={(status) => <CustomErrorView status={status} />}
>
  <Toolbar />
  <Canvas />
</Flux.Boundary>
```

Both `loading` and `error` are optional. `loading` defaults to a sensible empty
fallback. `error` defaults to `<Status.Summary status={status} />`, which already
renders deletion, permission denial, network failure, and generic errors with the
correct variant styling.

Internally, `<Flux.Boundary>` is
`<ErrorBoundary><Suspense>...</Suspense></ ErrorBoundary>` with the error boundary
normalizing thrown values to a Status before invoking the consumer's `error` prop.

## 4.6 - Boundary Placement

Placement is a UX decision left to the consumer. The substrate ships only the
`<Flux.Boundary>` component; it does not impose a topology.

For the schematic surface, the expected placement is one boundary per panel (toolbar,
canvas, properties drawer) so each panel resolves independently. The shell layout paints
instantly and pieces fill in as their data arrives. Where panels read truly independent
data, this is a strict UX improvement over the current behavior of "everything blocks
until the canvas is ready."

For surfaces with a single dominant resource (e.g., a settings dialog that reads one
record), a single outer boundary is appropriate. For lists of cards where each card
reads independent data, a boundary per card is appropriate.

# 5 - Migration

The migration is atomic. All ~300 consumer call sites of `useRetrieve` and `useSelect*`
move in a single PR alongside the substrate change. The old API is deleted in the same
commit.

## 5.0 - Mechanical Pass

Most call sites are mechanical conversions. A jscodeshift transform handles the common
shapes:

```ts
// Before
const { data } = useRetrieve({ key });
if (data == null) return null;

// After
const data = useRetrieve({ key });
```

```ts
// Before
const { data, status } = useRetrieve({ key });
if (status.variant === "loading") return <Spinner />;
if (data == null) return null;

// After (loading hoisted to parent <Flux.Boundary loading={<Spinner/>}>)
const data = useRetrieve({ key });
```

The codemod handles destructure stripping and null-guard deletion. Loading and error UI
hoisting is detected by the codemod and flagged for human review.

## 5.1 - Architectural Pass

Call sites that consumed `status` or `error` to render UI inline need judgment. For most
of them, the right move is to delete the inline UI and add a `<Flux.Boundary>` at the
appropriate panel boundary with the loading or error UI moved into the boundary's
fallback prop.

A small number of sites use the lifecycle for genuinely inline UI - a "Saving..."
indicator on a button, a refresh spinner that should not unmount the surrounding panel.
These cases survive on the mutation side (`useUpdate` still exposes `Result`) and do not
need a Suspense rewrite. Reads do not have this case in the existing codebase.

## 5.2 - The Result Pattern

`Result<T>` is retired for reads. Mutations still use it, because mutation consumers
legitimately need to render save state inline (a button's "Saving..." label, a form's
success toast). The `Result` type stays in the substrate; only its read-side use is
removed.

# 6 - Alternatives Considered

## 6.0 - Two Read APIs (Suspending + Snapshot)

Considered exposing both a suspending read and a non-suspending "snapshot" read that
returns `T | undefined`. The snapshot variant would cover cases like hover previews
(peek without fetching) and inline status indicators (need the lifecycle flag without
unmounting).

Rejected. The peek-without-fetching case is rare and can be served by the boundary's
`loading` fallback rendering placeholder UI. The inline-lifecycle case does not exist
for reads in the current codebase. Two APIs would force consumers to pick correctly at
every call site, and the wrong pick reintroduces the two-shapes problem we are
eliminating.

## 6.1 - Substrate-Imposed Boundary Topology

Considered shipping a per-resource wrapper component (e.g.,
`<Schematic.Boundary resourceKey={key}>`) that hardcodes Suspense placement around each
resource. Consumers would not pick where boundaries live; the substrate would.

Rejected. Where to place a boundary is a UX call (per-panel for partial loading,
per-page for atomic loading) that varies across surfaces. The substrate ships the
building block; placement is consumer judgment.

## 6.2 - useOptimistic for Mutations

Considered migrating optimistic updates to React 19's `useOptimistic` hook.

Rejected. `useOptimistic`'s scope is component-local React state. Synnax's optimistic
values must propagate across Drift windows, which requires writing to the shared flux
cache. The current write-through pattern is already correct for this case; switching to
`useOptimistic` would break multi-window sync.

## 6.3 - New Error Class Hierarchy

Considered introducing substrate-specific error classes (`DeletedError`,
`PermissionError`, `NetworkError`, `ValidationError`) that consumers pattern-match on in
the boundary fallback.

Rejected. The existing `xstatus.Status` system already discriminates errors by
`variant`, exposes `toStatus()` for typed errors, and ships display components
(`Status.Summary`, `Status.Indicator`) that consumers already know. Reinventing the
taxonomy adds types without adding capability.

## 6.4 - Selector Throw to Undefined

Considered the smaller fix: change `useSelectElementConfig` and similar selectors to
return `undefined` for missing parent records instead of throwing `NotFoundError`.
Consumers would handle the undefined case.

Rejected as a long-term solution, though it remains the right minimal patch for PR #2291
if the Suspense migration is delayed. The fundamental problem - that the selector cannot
distinguish "loading" from "not found" - is unsolved by silencing the throw. It only
laundered into nullable returns scattered through every consumer.

## 6.5 - Substrate-Only Migration with Coexistence

Considered shipping the new substrate alongside the old one, migrating one resource per
PR over a longer rollout.

Rejected. The two patterns interact poorly: the cache invariants (one-way hydration, no
re-suspend) only hold if every consumer participates. A half- migrated resource leaves
the toolbar bug class alive in unmigrated code and forces every reviewer to remember
which API a given file uses. Atomic flip is the cleaner outcome.

# 7 - Open Questions

## 7.0 - Cache Eviction

Out of scope. The current substrate keeps every entry forever and Suspense is neutral on
that. If long-session memory growth becomes a problem, an LRU cap can be added without
touching the read API.

## 7.1 - Dev-Time Boundary Detection

Consumers that read without a `<Flux.Boundary>` ancestor produce confusing behavior: the
thrown promise propagates to whatever Suspense ancestor exists (or unhandled). A
dev-mode warning that detects unwrapped reads is desirable but not blocking.

## 7.2 - Codemod Coverage

The mechanical pass covers most call sites but not all. A budget of ~10% of sites
requiring human conversion is realistic. The architectural pass (loading/error hoisting)
is fully manual.

# 8 - Implementation Plan

The work breaks into substrate, codemod, and consumer migration. All three land in one
PR.

1. Implement the new `createRetrieve`, `createSelector`, and `createList` primitives.
   Cache state machine, promise dedup, suspension via `use()`.
2. Implement `<Flux.Boundary>`. Suspense + ErrorBoundary composition with status
   normalization.
3. Update channel listener handlers to enforce one-way hydration. Audit every listener
   for the "fire on missing entry" behavior; either ignore or trigger delete-state
   transition.
4. Update `createUpdate` to enforce the no-re-suspend invariant. Audit rollback paths.
5. Write the jscodeshift transform.
6. Run the codemod across all consumers. Resolve the architectural pass call sites by
   hand. Add `<Flux.Boundary>` placements panel by panel.
7. Delete `Result<T>` from read paths. Keep on mutation paths.
8. Verify the schematic toolbar bug from PR #2291 is structurally resolved.
