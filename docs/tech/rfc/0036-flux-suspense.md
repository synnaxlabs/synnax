# 36 Flux suspense architecture

- **Author**: Emiliano Bonilla
- **Date**: 2026-05-07

## 0 Summary

This RFC describes the Suspense-shaped read path added to Pluto's Flux substrate.
Consumers can now call `useRetrieveSuspended(query)` and receive the resolved value
directly: the hook either returns `T` or suspends on the in-flight promise. Concurrent
reads of the same query share a single fetch through a per-client `QueryCache` keyed by
a deterministic hash of the query input. Channel-listener updates push new values into
the cache without re-suspending mounted consumers. The substrate ships alongside the
existing observable hooks (`useRetrieve`, `useRetrieveStateful`,
`useRetrieveObservable`, selectors) rather than replacing them; consumers opt into the
suspended API per call site.

## 1 Vocabulary

- **Suspended read**: A hook that returns `T` synchronously or throws a promise that
  React unwraps via `<Suspense>`. Implemented with React 19's `use()`.
- **QueryCache**: The per-client `Map<string, Result<T>>` that holds in-flight and
  resolved queries. One cache per `flux.Client`, addressable via `useQueryCache()`.
- **Query hash**: A deterministic string built from the `{name, query}` pair by
  recursively sorting object keys. Two calls with structurally equal queries collide on
  the same cache entry.
- **`mountListeners`**: The per-retrieve callback that wires channel-listener
  subscriptions. Suspended reads pass the cache plus a setter-style `onChange` so
  listeners can update the cached value without re-suspending.
- **`SuspenseBoundary`**: The consumer-facing wrapper that bundles React's `<Suspense>`
  with `Errors.Boundary`. Lives at `pluto/src/errors/SuspenseBoundary.tsx`.

## 2 Motivation

The existing read path returns a `Result<T> = {variant, status, data, ...}` wrapper that
consumers must inspect at every call site:

```tsx
const { data, variant } = Ranger.useRetrieve({ key });
if (variant === "loading") return <Spinner />;
if (variant === "error") return <ErrorView status={status} />;
if (data == null) return null;
return <RangeDetails range={data} />;
```

Three problems recur. First, every consumer writes the same null guard, often without
the loading branch (rendering nothing instead of a spinner). Second, selectors
(`createSelector` against the same store) read synchronously and have no lifecycle
awareness: a selector on a record that has not yet been fetched returns `undefined` or
throws, with no path to the "in flight" state. Third, concurrent reads of the same query
each issue their own fetch, because dedup is owned per-hook rather than at the
substrate.

React 19's `use()` plus `<Suspense>` solves all three structurally. A consumer either
gets the value or suspends; lifecycle plumbing moves out of the call site and into a
boundary. Adding the substrate is additive. Observable hooks stay for cases that
genuinely need inline loading/error rendering (mutation pending states, indicators in
toolbars), and consumers migrate per call site.

## 3 Design

### 3.0 Cache state machine

A `QueryCache` entry is a `Result<T>` in one of three variants:

| Variant   | Holds                         | Read behavior                  |
| --------- | ----------------------------- | ------------------------------ |
| `loading` | `Promise<T>` (via `.promise`) | `use(promise)` → suspend       |
| `success` | `T` (via `.data`)             | Return `data`                  |
| `error`   | `Status` (via `.status`)      | `throw status.toError(status)` |

A read against an absent hash kicks off the configured `retrieve` function, wraps the
returned promise in `pendingResult(name, promise)`, and writes it to the cache via
`cache.set(hash, ...)`. `set` inspects the result: when it is `loading` with a promise,
the cache wires `.then` / `.catch` handlers that auto-transition the entry to `success`
or `error` when the promise settles.

The settle-time replacement is **identity-gated** by `replaceIfStill`. If a channel
listener pushes a new value into the cache between the read and the settle, the
listener's `success` result replaces the original `loading` entry. When the original
promise eventually resolves, `replaceIfStill` sees that the current entry is no longer
the `loading` result it captured and refuses to overwrite. Late resolutions cannot
clobber fresher listener-driven state.

`cache.set` accepts an optional value-equality function. On a `success` → `success`
transition where the new and previous data are equal, the entry is left in place and
subscribers are not notified. Derived selectors built on the suspended retrieve use this
to skip re-renders when the slice they care about did not change despite the parent
record having changed.

### 3.1 Suspended read API

`createRetrieve` returns two new hooks alongside the existing four:

```ts
// Returns T, or suspends/throws via React 19's use().
const data = Ranger.useRetrieveSuspended({ key });

// Gates rendering on data being present without subscribing the caller to
// subsequent cache updates. Use in parents whose children read fresh values
// via their own selectors.
Ranger.useEnsureRetrieved({ key });
```

`useRetrieveSuspended` is the primary entry point. On mount it:

1. Hashes the `{name, query}` pair via `hashQuery`.
2. Subscribes to the cache hash via `useSyncExternalStore`, so listener-driven `set`
   calls re-render the consumer without re-suspending.
3. Looks up the entry. On `success` returns `data`. On `error` throws
   `status.toError(status)`. On `loading` calls `use(promise)`.
4. On a cache miss, invokes `retrieve(...)`, writes a `pendingResult` to the cache, and
   calls `use(promise)`. Concurrent consumers of the same hash on the same render pass
   land on step 3 and share the promise.

`useEnsureRetrieved` follows the same flow but does **not** call `cache.subscribe`. It
suspends or throws while the cache entry settles, then returns `void`. Parents that want
to gate their subtree on data presence without re-rendering when the data changes use
it; their children re-read via their own selectors.

Both hooks throw a synchronous `Error` if the Synnax client is null and
`allowDisconnected` is not set, matching the contract of the existing `useRetrieve`.

### 3.2 Listener integration

`RetrieveParams` gained an optional `cache?: QueryCache` field. `retrieve` and
`mountListeners` implementations receive it and can read or update entries directly. The
suspended hook passes a setter-style `onChange` to `mountListeners`:

```ts
const onChange = (value: state.SetArg<Data | undefined>) => {
  const current = cache.get<Data>(hash);
  const prev = current?.variant === "success" ? current.data : undefined;
  const next = state.executeSetter(value, prev);
  if (next == null) return;
  cache.set(hash, successResult(name, next));
};
```

A listener firing for a hash whose entry is in `success` state replaces the value in
place. Subscribers re-render via the `useSyncExternalStore` path. A listener firing for
a hash that is still `loading` triggers the identity-gated replacement described in
§3.0: the listener's `success` result wins, and the eventual promise resolution is
discarded.

The cache and the store remain separate concerns. Per-record stores
(`flux.UnaryStore<K, V>`) hold canonical values keyed by record ID; the query cache
holds query lifecycles keyed by query shape. A single-record retrieve with shape
`{key: X}` and a list retrieve with shape `{workspace, filter}` both live in the cache,
addressed by their respective hashes, while the underlying records also live in the
per-record store.

### 3.3 Errors and the boundary

The substrate's contract with a consumer's error boundary is: a thrown value is always a
`Error` (with the original `xstatus.Status` on `cause`). `x/ts/src/status/status.ts`
gains `status.toError(status)`, which builds an `Error` whose `message` is the wrapped
status message, copies `name` and `stack` from the inner error preserved on
`details.error`, and stashes the full status on `cause`. The suspended hooks call it
when an entry is in the `error` variant; custom fallbacks recover the rich shape via
`(error.cause as Status)`.

`Errors.SuspenseBoundary` bundles `Errors.Boundary` (the existing error boundary) with
React's `<Suspense>`:

```tsx
<Errors.SuspenseBoundary loading={<Skeleton />}>
  <Display />
</Errors.SuspenseBoundary>
```

Both `loading` and `FallbackComponent` are optional. `loading` defaults to nothing;
`FallbackComponent` defaults to the diagnostic `Errors.Fallback` page that
`Errors.Boundary` ships with today. Placement is a consumer-level UX decision: one
boundary per panel, per page, per card, or whatever the surface requires.

### 3.4 Hash determinism

`hashQuery` recursively walks the query input and emits a canonical string:

```ts
hashQuery({ a: 1, b: 2 }) === hashQuery({ b: 2, a: 1 }); // true
hashQuery([1, { c: 3 }]) === '[1,{"c":3}]';
hashQuery({ key: "x" }) === '{"key":"x"}';
```

Object keys are sorted at every level. Arrays preserve order. Primitives are
`JSON.stringify`d. The hash is structural: two calls with deep-equal queries collide
regardless of object identity, which is what makes dedup work across renders and across
components.

Functions, `Map`, `Set`, `Date`, and other non-plain types are not supported as query
inputs. Callers pass primitives, plain objects, and arrays. This matches what
`useMemoDeepEqual` already enforces upstream of the cache lookup.

## 4 What this RFC does not cover

- **Atomic migration of existing call sites.** The substrate is additive. Consumers
  migrate per call site as the suspense-shaped read becomes a clear improvement;
  observable hooks remain available for mutation pending states and inline loading
  indicators.
- **List API.** `createList` was not modified. A list-shaped suspended hook is a
  follow-up; the current `QueryCache` accommodates arbitrary query shapes, so it can be
  added without changing the cache contract.
- **Mutations.** `createUpdate` is unchanged. Mutations continue to write through the
  per-record store with the existing optimistic-update + rollback pattern. The suspended
  read path observes whatever the store currently holds via its `mountListeners` wiring.
- **Cache eviction.** Entries live for the life of the client. No LRU, no TTL. If
  long-session memory growth becomes a problem, an eviction policy can be added behind
  the existing `cache.invalidate(hash)` method without touching the read API.
- **Dev-mode boundary detection.** A consumer that calls `useRetrieveSuspended` without
  a `<Suspense>` ancestor produces React's default suspended-without-boundary warning. A
  Synnax-specific dev warning that points at the `SuspenseBoundary` primitive is a
  follow-up.

## 5 Implementation status

The substrate has shipped on `sy-4158-refactor-flux-to-support-react-suspense`. The
implementation lives in:

- `pluto/src/flux/base/queryCache.ts` holds the `QueryCache` class and `hashQuery`.
- `pluto/src/flux/retrieve.ts` appends `useRetrieveSuspended` and `useEnsureRetrieved`
  to `createRetrieve`'s return, and adds an optional `cache` field to `RetrieveParams`.
- `pluto/src/flux/result.ts` extends `LoadingResult` with optional `promise` and `name`
  fields and adds the `pendingResult(name, promise)` constructor.
- `pluto/src/flux/Provider.tsx` and `base/client.ts` give `Client` a `queryCache` field
  and expose it via `useQueryCache()`.
- `pluto/src/errors/SuspenseBoundary.tsx` is the bundled Suspense + error boundary
  component.
- `x/ts/src/status/status.ts` adds the `status.toError` bridge.

Coverage on the substrate:

- `pluto/src/flux/retrieve.spec.tsx`: suspension, dedup of concurrent reads, error
  routing to the fallback.
- `pluto/src/errors/SuspenseBoundary.spec.tsx`: loading, error fallback, custom fallback
  component.
- `pluto/src/flux/select.spec.tsx`: derived selectors over the cache (12 cases).

## 6 Resolved decisions

1. **Substrate is additive.** The original RFC proposed an atomic migration of all read
   sites onto a single suspended API. The shipped implementation keeps the observable
   hooks (`useRetrieve`, `useRetrieveStateful`, `useRetrieveObservable`) and adds the
   suspended pair (`useRetrieveSuspended`, `useEnsureRetrieved`) on the same
   `createRetrieve` factory. Per-call-site migration is cheaper than a flag-day move,
   and the observable hooks have legitimate uses (mutation pending states, indicators)
   that suspension does not serve.

2. **Three cache states, not two.** The original RFC proposed a two-state machine
   (`pending` and `hydrated`) with errors surfacing only via thrown deletion statuses.
   The shipped cache holds the full `Result<T>` discriminated union (`loading`,
   `success`, `error`), which lets a failed initial fetch persist as an `error` entry
   until something invalidates it. This matches how `Result` already works elsewhere in
   Flux and avoids forcing every suspension miss to re-fetch.

3. **Identity-gated settle.** When `cache.set` wires up the promise's `.then`, it
   captures the loading-result object identity. At settle time, `replaceIfStill` checks
   that the current entry is still that exact object before overwriting. This resolves
   the race between listener pushes (which can replace a `loading` entry with a
   `success` mid-flight) and the original promise eventually resolving. Listener-driven
   freshness always wins.

4. **Optional value equality for derived selectors.** `cache.set` accepts an
   `equal(a, b)` callback. Selectors built on `useRetrieveSuspended` plus a slice
   function pass this to skip re-renders when the slice did not change despite the
   parent record having changed. The cache itself does not opine on equality; it just
   honors the hook's choice.

5. **`Errors.SuspenseBoundary` lives in `errors/`, not `flux/`.** The boundary is not
   Flux-specific. It bundles React's `<Suspense>` with the existing `Errors.Boundary`,
   both of which already live in `pluto/src/errors/`. Placing the wrapper there keeps
   the Flux module focused on read substrate and lets non-Flux suspended consumers reuse
   the boundary.

6. **`hashQuery` sorts keys recursively, supports plain shapes only.** The hash is
   structural over plain objects, arrays, and primitives. `Map`, `Set`, `Date`, and
   class instances are not supported. This matches the assumption already enforced by
   `useMemoDeepEqual` at the hook boundary, so callers that pass valid input to the
   existing hooks pass valid input here too.
