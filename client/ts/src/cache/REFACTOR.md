# Cache Refactors

Working list. Delete when the work lands.

Each entry leads with the contract in one sentence. Anything in the API that doesn't
fall out of that sentence is invention and goes. An entry without a contract is not
ready to implement, no matter how obvious the rename looks.

## 1. Error handling — LOCKED

**Contract:** the error handler is where the cache puts an error that has no caller to
throw to.

One optional param on the client, defaulting to console:

```ts
onInternalError?: (error: Error) => void;
```

`Error`, not `unknown`: `unknown` is what `catch` hands you, not what a contract hands a
caller, and making every implementer normalize is N copies of `errors.fromUnknown`
(`x/ts/src/errors/errors.ts:187`). Not a status: variant, key, and timestamp are a
display shape, and how an error is presented is the caller's call.

Callers catch locally, wrap with their own context via `cause`, normalize, and report.
The sites that need it are the ones with no stack to throw up: `Table` listener fan-out,
streamer frame handling, and background reconcile.

Lives in `SynnaxParams`; functions in a zod schema are fine, see
`pluto/src/vis/button/aether/button.ts` for the zod 4 form.

Injection sites, all of which have the sink in hand at construction:

- `pluto/src/synnax/Provider.tsx:152`
- `pluto/src/synnax/aether/provider.ts:50` — props are serialized and can't carry a
  function, so it spreads them and adds its own after
- `pluto/src/flux/Provider.tsx` — detached engine
- `console/src/testutil/Synnax.ts:164`

Deletes:

- `ErrorHandler`, `AsyncErrorHandler` (`cache/types.ts`)
- `consoleErrorHandler`, `consoleAsyncErrorHandler` (`cache/engine.ts`)
- `setErrorHandlers`, `errorSink`, `asyncErrorSink`, both delegating closures, and the
  `errorHandler` getter (`cache/engine.ts`)

**Out of scope:** pluto's `status.ErrorHandler` and `AsyncErrorHandler`
(`pluto/src/status/aether/errorHandler.ts`) have ~90 call sites across pluto and console
for general UI error handling. The seam means none of them move. Pluto adapts at the
boundary:

```ts
onInternalError: (error) => addStatus(status.fromException(error));
```

## 2. `Engine` → `Cache` — contract not derived

`cache.Cache` is legal under the CLAUDE.md namespace rule (a package's core item may
share the package's exact name, like `channel.Channel`). "Engine" is a noise word.

The rename forces the `Handle` question: it's a second facade over the same object
(`enabled`, `epoch`, `onEpoch`, `close`, and a throwing `engine` getter), so after the
rename `cache.Cache` and `cache.Handle` both mean "the cache" and `client.cache` returns
the wrong one. Handle exists only because cache-disabled is `null` and something has to
own the flag and the throw. A no-op Cache deletes Handle, the `enabled` flag, and the
`store_?` / `engine_?` / `queries_?` optionality in every domain client.

Needs: one sentence on what a Cache is for.

## 3. `ScopedUnaryStore` → `Table` — contract not derived

Seven types currently carry "Store": `Store`, `UnaryStore`, `ScopedUnaryStore`,
`Stores`, `InternalStore`, `StoreConfig`, `UnaryStoreConfig`. "Unary" distinguishes it
from nothing; there is no other arity. "Scoped" is an implementation detail in a type
name. Target is `Table`, `Table.Config`, `Tables`: keyed rows, tombstones as deleted
rows.

Separately, `set` is overloaded on arity across three runtime-discriminated call shapes
(`Array.isArray`, `"key" in key`), with `value` meaning either a setter or a variant
depending on the branch, and four `as SetExtra` casts covering the gap. The `UnaryStore`
conditional type exists only to shift the variant argument's position and duplicates
every method to do it. Splitting into `set` / `setMany` with the variant in an options
object collapses both.

Needs: one sentence on what a Table is for.

## 4. What scoping means — contract not derived

As built, scope does exactly one thing: suppress notifications to listeners whose scope
equals the writer's. It is not isolation, namespacing, or multi-tenancy, and the name
promises all three.

Twenty-one domain clients define `MOUNT_SCOPE = "<domain>.mounts"`. Every one exists to
be a string that isn't `""`, so their mount listeners are _not_ suppressed. Nobody in
the client uses scope to suppress anything; they are all opting out of it via a
constant. The comment in `label/client.ts` says so outright: the streamer writes in the
default scope, "which would silence default-scope subscriptions entirely."

The only genuine consumer is `createScopedStore` (`pluto/src/flux/base/store.ts:96`),
per dispatch controller, for undo.

Blocked on §6. If queries become derived views owned by the Table, the Table applies a
change and notifies views in one pass, there is no echo, and this whole mechanism
disappears rather than getting renamed.

## 5. `state` → `x/state` — contract not derived

Two copies today: `client/ts/src/cache/state.ts` (32 lines) and
`pluto/src/state/state.ts` (138). Pluto's is a superset, but half of it is React
(`usePassthrough`, `usePurePassthrough`, `usePersisted`, `Use` / `UseReturn`), and x has
no React dependency and shouldn't grow one.

Split: `x/state` takes the pure core (`State`, `SetArg`, `SetFunc`, `isSetter`,
`executeSetter`, `skipNull`, `skipUndefined`, `Initial`, `executeInitialSetter`);
`pluto/state` keeps the hooks.

Open: 42 files import `@/state` in pluto and console, and most want both halves. The
no-compat-aliases rule says `pluto/state` shouldn't re-export x's half, but then those
files carry two imports and a name collision on `state`. Needs a call.

## 6. Queries as derived views — open, and it gates §4

Today `Queries` owns query answers, `ScopedUnaryStore` owns canonical records, and each
domain client hand-writes the reconciliation between them as mount closures (four in
`label`, five in `ranger`, across 21 clients). Every seam between those three owners is
one of the symptoms above.

The proposal is that a query becomes a first-class derived view over the Table with a
declared match predicate, so mounting is registering a predicate rather than writing
listener closures per query shape.

The open doubt: server-computed queries. `requestFilter` in `label/client.ts` already
concedes it is "permissive for server-computed shapes (search), which accept every
change and drift toward the server's answer." If search, limit, and offset can't be
predicates, the derived-view model buys less than it looks and the two-layer split may
be load-bearing. Answerable by reading `requestFilter` across the 21 clients.

## Not doing

`Verbs` in `cache/types.ts` stays.
