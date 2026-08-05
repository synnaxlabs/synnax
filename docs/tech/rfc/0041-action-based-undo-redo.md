# 41 Action-based undo and redo

- **Author**: Emiliano Bonilla
- **Date**: 2026-05-08
- **Related**:
  [PR #2290 - Schematic Action Codec and Dispatch Endpoint](https://github.com/synnaxlabs/synnax/pull/2290),
  [RFC 0036 - Flux suspense architecture](0036-flux-suspense.md)

## 0 Summary

Pluto's Flux substrate gains an undo and redo capability for any document type that
mutates through a discriminated-union action API. Each TS action handler returns the
inverse of the operation it just applied, computed against the actual pre-state. The
undo stack lives on the client, in the Flux cache entry for the document, per window.
Undo dispatches the stored inverse through the same dispatch pipeline as any user edit,
so the server, the cluster broadcast, and the conflict-handling code stay unchanged.
Redo dispatches the original forward against the post-undo state and relies on the
reducer to recompute equivalent post-state. Adjacent same-kind, same-target dispatches
inside a configurable window coalesce into a single undoable so streamed gestures like
dragging a node collapse to one ⌘Z.

The schematic is the first integration and ships in this work. Line plots, tables, and
any future document type built on the action substrate get undo by adopting the same
`createUndoableStore` + `createDispatch` factories with their own reducer, channel, and
send function. The server gains no new endpoints and no history storage. Durable version
history is a separate feature and is out of scope.

## 1 Vocabulary

- **Document**: A record managed by the Flux substrate that mutates only through
  actions. A schematic is one document type; a line plot will be another.
- **Document state**: The shared, broadcast state of a document. Lives in the Flux
  cache.
- **Action**: A single discriminated-union mutation defined per document type and
  generated from a `.oracle` schema (e.g., `SetNodePosition` for schematics).
- **Reducer**: A per-document-type function that applies a vector of actions and returns
  `{ next, inverse, targets }`. TS-only; the server reducer is unchanged.
- **Inverse**: A vector of actions that, applied to the post-state, restores the
  pre-state.
- **Targets**: The set of entity keys an action vector touched. Drives coalesce identity
  and stale detection.
- **Transaction**: A staged group of dispatches committed atomically as one undoable.
  Equivalent to one dispatch of the accumulated actions.
- **Forward**: The original action vector in a stack entry.
- **Stack entry**: `{forward, inverse, kind, ts, targets}` on the undo or redo stack.
- **Kind**: A tag the document classifies a transaction with; used for coalesce.
- **Coalescing**: Merging two adjacent stack entries with the same kind and targets
  within a configured time window so that one undo reverses both.
- **Stale entry**: A stack entry whose targets have been touched by a remote session
  since the entry was pushed.

## 2 Motivation

### 2.0 The action substrate is already reversible in principle

PR #2290 introduced action-based dispatch for schematics. Subsequent work will give the
same shape to other documents. Each action is small, well-typed, and replayable. What is
missing is the function from "action just applied" to "actions that would reverse it"
and a place to keep those inverses while a user is editing.

### 2.1 Users expect undo

Schematics are the primary authoring surface in the Console; line plots and tables are
not far behind. Building a non-trivial document of any of these types involves dozens of
small placements, alignments, and parameter tweaks, many of which are exploratory.
Without undo, every mistake forces a manual revert and every experiment costs a manual
cleanup. Undo is table-stakes for any document editor.

### 2.2 Per-document-type undo doesn't scale

If undo ships as schematic-only code, every future action-based document type
re-implements the same machinery: the stack, the coalescing, the staleness detection,
the multi-pane sharing, the transaction API. Each instance drifts from the others. The
right level for this work is the substrate. Per-document-type code only contributes the
parts that genuinely vary by type — the action set, the inverse logic, the kind
classifier.

### 2.3 "server-side undo" conflates two features

The original framing (`sy-3038-add-server-side-undo-and-redo`) treated undo as a
durable, server-owned feature. This conflates two distinct user needs:

1. **In-session correction** ("take that back") — sub-second feedback, per-user, lost on
   close. Every desktop editor since the 80s works this way and users expect it.
2. **Time travel** ("show me what this looked like yesterday") — durable, named,
   browsable. This is version history, not undo.

A single mechanism that tries to serve both ends up bad at both: every undo becomes a
network round-trip, the server acquires UI-level concepts (transactions, coalescing,
redo trees), and users still don't get the version-history affordance they actually want
for time travel. The shipped architecture separates these layers and delivers the
in-session feature now. Version history can be designed independently and built later on
the same action substrate.

## 3 Design

### 3.0 The reducer contract

Each action-based document type defines a reducer with the shape:

```ts
type DispatchReducer<State, Action> = (
  state: State,
  actions: Action[],
) => { next: State; inverse: Action[]; targets: readonly string[] };
```

Per-action handlers are generated from the `.oracle` schema and take a `Draft<State>`
plus the action's payload, returning `{ inverse, targets }`. The generated
`createReduceAll` wraps the handler map in Immer's `produce`, drives the handlers in
order, concatenates `targets`, and returns the per-handler inverses in reverse
application order — the single inverse of the whole vector.

```ts
const handlers: Handlers = {
  setNodePosition: (state, payload) => {
    const node = state.nodes.find((n) => n.key === payload.key);
    if (node == null) return { inverse: [], targets: [] };
    const oldPosition = { x: node.position.x, y: node.position.y };
    node.position = payload.position;
    return {
      inverse: [setNodePosition({ key: payload.key, position: oldPosition })],
      targets: [payload.key],
    };
  },
  // ...
};

export const reduceAll = createReduceAll(handlers);
```

The schematic reducer is the first concrete implementation. Its inverses, against the
actual pre-state:

| Action            | Inverse                                          |
| ----------------- | ------------------------------------------------ |
| `Rename`          | `Rename(oldName)`                                |
| `SetNodePosition` | `SetNodePosition(K, oldPosition)`                |
| `SetNodeMeasured` | `[]` (renderer feedback, non-undoable)           |
| `SetNode`         | `SetNode(oldNode, oldConfig)` or `RemoveNode(K)` |
| `RemoveNode`      | `SetNode(oldNode, oldConfig)`                    |
| `AddEdge`         | `RemoveEdge(K)`                                  |
| `RemoveEdge`      | `AddEdge(oldEdge)`                               |
| `SetConfig`       | `SetConfig(K, oldFields)` (partial; see §3.7)    |

Inverses are TS-only. The Go server's per-action `Handle` is unchanged: it still returns
`(State, error)`, and the writer's dispatch path applies the vector inside a single
transaction. The server uses the action stream as its source of truth and broadcasts the
original forward over the action signal channel; the client computes inverses locally
for stack bookkeeping.

### 3.1 Undoability

Per-document-type code passes an `isUndoable(action) => bool` callback into
`createUndoableStore`. The substrate filters non-undoable actions out of the recorded
`forward` before pushing a stack entry; if every action in a transaction is
non-undoable, nothing is pushed. For the schematic, `set_node_measured` is the only
non-undoable action — it carries layout-derived dimensions from the renderer, not user
intent. Its reducer also returns empty targets so a remote `set_node_measured` cannot
invalidate undoables targeting the same node.

A schema-level `undoable: bool` annotation on `.oracle` action declarations was
considered and dropped. A per-store callback captures the same intent in one line and
avoids a generator change; promoting it to a generated `UndoableActions` set is a
follow-up if a second document type accumulates more than a handful of exclusions.

### 3.2 `Flux.createUndoableStore` and `Flux.createDispatch`

Pluto's Flux substrate exposes two parallel mutation factories:

- **`Flux.createUpdate`** (existing): Free-form path. The user provides an arbitrary
  `update` callback that does optimistic apply, server send, and rollback on its own. No
  undo. Used for non-action-based mutations like create, rename, delete, copy.
- **`Flux.createDispatch`** (new): Action-based path. The substrate owns the local apply
  via the reducer, owns the stack, and owns the undo machinery. The user provides only
  the reducer, the channel and schema for remote broadcasts, and a send function.

The action-based path is split across two factories so doc-type code can build the store
independently of the dispatch hooks:

```ts
export const FLUX_STORE_CONFIG = Flux.createUndoableStore<
  schematic.Key,
  schematic.Schematic,
  schematic.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  reduce: schematic.reduceAll,
  preprocess: augmentWithEdgeSegments,
  channel: schematic.SET_CHANNEL_NAME,
  schema: schematic.scopedActionZ,
  isUndoable: schematic.isUndoable,
  kindOf: kindOfTransaction,
});

export const { useDispatch, useUndo, useRedo } = Flux.createDispatch<
  schematic.Key,
  schematic.Schematic,
  schematic.Action,
  typeof FLUX_STORE_KEY,
  FluxSubStore
>({
  storeKey: FLUX_STORE_KEY,
  send: ({ client, key, actions, sessionKey }) =>
    client.schematics.dispatch(key, sessionKey, actions),
});
```

`createUndoableStore` returns a `UnaryStoreConfig` that registers the remote action
listener and constructs the `UndoableUnaryStore` under the given store key.
`createDispatch` returns three hooks:

- `useDispatch` — runs the reducer, applies optimistically, pushes to the stack, calls
  `send`, rolls back local state and pops on server reject. Also exposes the transaction
  API.
- `useUndo` / `useRedo` — drive `prepareUndo` / `prepareRedo` on the store, dispatch the
  chosen action vector through the same apply path, and skip preprocess (since the
  stored entry already holds processed actions). Each returns
  `{ undo|redo, canUndo|canRedo }` so consumers can both fire the action and react to
  availability.

### 3.3 Stack shape

The store keeps per-document undo state in its own scoped unary store, separate from the
doc state:

```ts
interface StackEntry<A> {
  forward: A[];
  inverse: A[];
  kind: string;
  ts: TimeStamp;
  targets: readonly string[];
}

interface UndoState<A> {
  undo: StackEntry<A>[];
  redo: StackEntry<A>[];
  remoteTouched: Record<string, TimeStamp>;
}
```

Stacks are not persisted across reloads, not synced via Drift, and not part of the
broadcast surface. They live and die with the cache entry.

### 3.4 Dispatch, push, rollback

The dispatch path in `apply` (shared by `useDispatch`, `useUndo`, `useRedo`):

1. Replay the reducer locally against the cached doc. Capture
   `{ processed, inverse, targets, rollback }`. If the doc isn't cached, abort.
2. Run the per-op stack mutation. For a fresh dispatch this calls
   `recordEntry(forward = processed, inverse, targets)`, which filters non-undoable
   forward actions, derives the kind via `kindOf`, coalesces against the top entry,
   clears redo, and returns a rollback. For undo or redo it moves the prepared entry
   from one stack to the other.
3. `await send(processed)`.
4. On failure, run the stack rollback (pop or un-coalesce the pushed entry; reverse the
   undo↔redo transition) and the doc rollback (restore the prior state), and surface the
   error to the status aggregator.

Undo and redo also pass `skipPreprocess = true` so the stored processed entry is not
re-augmented at replay time.

### 3.5 Transactions

The stack's natural grain is one transaction per `dispatch(actions[])` call. A single
dispatch with a vector of actions is one undoable. For consumers that want to stage
actions across multiple `add` calls and commit them atomically, the substrate exposes
`beginTransaction`:

```ts
const tx = beginTransaction({ key, kind: "move" });
tx.add([SetNodePosition(k, p1)]);
tx.add([SetNodePosition(k, p2)]);
await tx.commit(); // sends the batch, pushes one stack entry
// or tx.abort() — rolls back local applications, no push, no send.
```

The transaction snapshot is the pre-transaction state, so the inverse a commit pushes
restores to where the user started rather than the last-add post-state. The handle is
window-local and never crosses into the server. It coalesces both the dispatch payload
and the stack entry.

The schematic does not currently use `beginTransaction` directly — its consumers (drag,
paste, property-panel commit, palette-add) each call `dispatch` once with the full
action vector for the gesture, and rely on coalescing (§3.6) to merge any streamed
dispatches inside the gesture into one undoable. The transaction API is in place for
future consumers whose gestures cannot be naturally framed as a single dispatch.

### 3.6 Coalescing

After a push, the substrate tests the top two entries of the undo stack for coalesce:

- Same `kind`.
- Same `targets` (compared as unordered primitive arrays).
- `ts` within the coalesce window (default 500 ms; configurable per document type).

When all three hold, the entries merge: `forward` concatenates, `inverse` prepends, `ts`
updates to the newer. The user sees one ⌘Z reverse the entire run.

The kind tag is computed by the per-document-type `kindOf(actions)` callback. The
schematic classifier returns `"move"` for any transaction whose actions are only
`set_node_position` plus `set_config` (the latter is synthesized by the edge-segment
preprocessor for any edge whose endpoints just moved), the raw action `type` for
single-action transactions, and `"transaction"` otherwise. Move was the only gesture
that streams enough frames to need explicit collapse; everything else either fits in one
dispatch already or doesn't repeat fast enough to clear the window.

### 3.7 Surfaced schema gaps

Implementing inverses against the current schematic action set surfaced two limitations
in the schema. Both ride along on top of the v1 implementation as documented
imperfections; both are tracked as schema follow-ups.

- **`SetConfig` is a merge, not a replace.** The inverse can restore values for keys
  that existed before the action, but it cannot remove keys the forward action newly
  introduced — those keys persist as phantom fields on undo. Closing the gap requires a
  `ReplaceConfig` action (wholesale replace, including remove when the payload is
  absent).
- **`SetNode` and `RemoveNode` lack an explicit slice index.** The inverse of
  `RemoveNode` reinserts at the end of the slice, so a remove + undo cycle restores
  contents but not the original position. Closing the gap requires an
  `InsertNode(node, idx)` action.

The schematic reducer test suite uses an `expectUserVisibleRoundTrip` helper that
acknowledges both gaps: nodes-and-edges identity and previously-existing config fields
round-trip correctly; index-position and newly-introduced config keys do not.

### 3.8 Multiplayer behavior

Remote actions (broadcast frames whose `sessionKey` differs from the local client's)
apply through the reducer just like local dispatches, but bypass the stack. Their
targets are written into a `remoteTouched: Record<string, TimeStamp>` map on the
document's undo state, keyed by entity ID; later remote actions over the same key
overwrite the timestamp.

When `useUndo` or `useRedo` prepares a reversal, the substrate walks the chosen stack
from the top and skips any entry whose targets contain a key whose `remoteTouched`
timestamp is newer than the entry's `ts`. Skipping a stale entry advances to the next. A
soft cap of 10 consecutive stale entries bounds the walk; if the entire walked tail is
stale, the substrate drops it from the stack and the reversal returns null (the
affordance reports nothing to undo).

This is intentionally simpler than operational transformation. The collaboration
patterns we see today — small teams, mostly serial editing — do not justify OT's
implementation cost. The remote-touched map is where OT slots in later if the use case
materializes. The invariant we hold is: every ⌘Z that completes does something visible,
or the stack is exhausted.

### 3.9 Bounds and lifecycle

- **Cap.** Each stack is capped at 200 entries. Older entries fall off the bottom.
- **Cache-lifetime.** Stacks live with the Flux cache entry, not with the on-screen
  component. Navigating away from a document to another layout in the same window does
  not clear the stack; navigating back resumes where the user left off. This follows
  from the one-way hydration rule in RFC 0036.
- **Two panes on one document share one stack.** Multiple mounts of the same document in
  the same window resolve to the same Flux cache entry, so they share `undo` and `redo`.
  ⌘Z works the same regardless of which pane has focus.
- **Cascade delete.** Deleting a document key drops both its doc state and its undo
  state in one rollback-aware operation.
- **Listeners must keep running while the document is off-screen.** Stale detection
  depends on the cache entry receiving remote actions even when the user is on a
  different layout. The existing action listeners are registered at provider mount and
  run for the life of the app; called out so future refactors do not regress it.

### 3.10 Keyboard bindings

Keyboard bindings live in the consumer, not the substrate. The schematic wires ⌘Z and
⌘⇧Z to its `useUndo` / `useRedo` via the existing `Diagram.useTriggers` hook in
`Schematic.tsx`:

```ts
BaseDiagram.useTriggers({
  onUndo: undo,
  onRedo: redo,
  enabled: enableTriggers,
  // ...
});
```

The bindings activate when the layout has focus and yield to layout defaults when it
does not. Other layouts can have different undo or none at all. A reusable
`<UndoBindings>` component would be a one-file follow-up if a second consumer needs the
same wiring; until then, the principle "bindings at the layout, not global" is upheld
without an extra abstraction.

## 4 What this RFC does not cover

- **Server-side stack or history.** The server has no `Undo` action, no per-user history
  table, no new endpoints. The dispatch endpoint is the same one introduced by PR #2290.
- **Durable version history.** Out of scope. A separate feature on the same action
  substrate; see §2.3.
- **Server-side inverse computation.** The Go reducer is unchanged. The server applies
  actions in order inside a single transaction and broadcasts the original forward; the
  client carries inverses for stack bookkeeping.
- **Schema-level `undoable: bool` annotation.** The v1 surface uses a per-store
  callback. Promoting it to a generated `UndoableActions` set is a follow-up if a second
  document type wants the same exclusion behavior.
- **Operational transformation.** Stale handling is detection-and-skip, not transform.
  The remote-touched map is where OT slots in if the use case materializes.
- **Snapshot affordance.** Read-only schematics (`snapshot == true`) reject dispatch at
  the server. The substrate does not currently hide the undo/redo affordance for
  read-only docs; the user can press ⌘Z and the dispatch rolls back. A consumer-side
  gate is a follow-up.
- **Branching redo.** Linear redo only; new dispatches clear the redo stack.

## 5 Implementation status

The substrate ships on `sy-3038-add-server-side-undo-and-redo`. Files of interest:

- `pluto/src/flux/base/undoable.ts` — `UndoableStore` class, `createUndoableStore`
  factory, stack/coalesce/remote-touched bookkeeping, transaction commit.
- `pluto/src/flux/dispatch.ts` — `createDispatch` factory exposing `useDispatch` /
  `useUndo` / `useRedo` and the shared `apply` path the three hooks all run through.
- `pluto/src/schematic/queries.ts` — the schematic's `createUndoableStore` +
  `createDispatch` configuration, the `augmentWithEdgeSegments` preprocessor, and the
  `kindOfTransaction` classifier.
- `pluto/src/schematic/Schematic.tsx` — wires ⌘Z / ⌘⇧Z to `useUndo` / `useRedo` via
  `Diagram.useTriggers`.
- `client/ts/src/schematic/actions.ts` — per-action handlers returning
  `{ inverse, targets }`, `isUndoable` predicate.
- `oracle/plugin/ts/actions/actions.go` — emits the `Handlers` / `HandlerResult` types
  and the `createReduceAll` factory consumed by the schematic reducer.

Coverage on the substrate:

- `pluto/src/flux/base/undoable.spec.ts` — 44 cases covering replay, recordEntry,
  coalescing, prepareUndo / prepareRedo (including stale skip and full-stale drop),
  beginTransaction (commit, abort, doc-not-cached, send-failure rollback), applyRemote,
  markRemoteTouched, cascade delete, hasUndo / hasRedo, onUndoStateChange.
- `pluto/src/flux/dispatch.spec.tsx` — dispatch / undo / redo / coalescing /
  transactions / cascade / empty stacks / multi-key isolation through the hook surface.
- `client/ts/src/schematic/actions.spec.ts` — per-action behavior and per-action
  `reduceAll(state, [a]); reduceAll(next, inverse)` round-trip assertions, with the
  documented `setConfig` and `removeNode` gaps gated through
  `expectUserVisibleRoundTrip`.

## 6 Resolved decisions

1. **Inverses are TS-only.** The Go reducer was not changed. The server uses the action
   stream as its source of truth and broadcasts the original forward; inverses are a
   client-side bookkeeping concern. Cross-language symmetry would have required
   reworking every per-action `Handle`, and the only purpose it would serve — validating
   that all actions in a transaction applied — is already handled by the existing
   transactional dispatch path.

2. **`undoable` is a callback, not a schema annotation.** Adding `undoable: bool` to the
   Oracle schema language would have touched the generator, the Go emitter, the TS
   emitter, and every existing action declaration. A per-store `isUndoable` callback
   expresses the same intent in one line and confines the change to the substrate.

3. **Store and dispatch are separate factories.** `createUndoableStore` builds the
   channel-listener-wired store; `createDispatch` builds the hooks. Splitting them lets
   the store live in a doc-type's Flux config alongside other stores (relationships,
   resources) without forcing the dispatch hooks to be constructed in the same call
   site, and lets tests substitute the dispatch surface while keeping the store under
   test.

4. **Stale handling skips ahead instead of applying-and-detecting.** An earlier draft
   described "apply the inverse blindly, observe no visible effect, then auto-advance."
   The shipped implementation walks the stack in `prepareUndo` / `prepareRedo` and skips
   entries whose targets were remote-touched after the entry's `ts`. This avoids
   dispatching no-ops through the reducer, avoids any chance of a stale inverse
   producing a visible-but-wrong effect on a partially-changed target, and makes the
   auto-advance cap (10) cheap to enforce.

5. **`beginTransaction` is exposed but not yet used by the schematic.** Every gesture in
   the schematic surface today (drag, paste, property-panel commit, palette-add)
   naturally batches into a single `dispatch` call, and the move-kind coalesce window
   collapses streamed drag frames into one undoable. The transaction API is in place for
   future consumers whose gestures cannot be naturally framed as a single dispatch.

6. **Keyboard bindings live in the consumer.** The schematic wires ⌘Z / ⌘⇧Z through its
   existing `Diagram.useTriggers` hook rather than through a substrate-provided
   `<UndoBindings>` component. The principle "bindings at the layout, not global" is
   upheld without an extra abstraction; a reusable component is a follow-up if a second
   consumer needs the same wiring.

7. **Two documented schema gaps ride along.** `SetConfig` is merge-only and cannot
   inverse-remove newly-introduced keys; `RemoveNode` cannot restore the original slice
   index. Closing them requires new actions (`ReplaceConfig`, `InsertNode`). The reducer
   ships with imperfect inverses in the affected paths; the test suite gates round-trip
   assertions on what currently round-trips (nodes-and-edges identity,
   previously-existing config fields).
