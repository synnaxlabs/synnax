# 37 - Action-Based Undo and Redo

**Feature Name**: Action-Based Undo and Redo <br /> **Status**: Proposed <br /> **Start
Date**: 2026-05-08 <br /> **Authors**: Emiliano Bonilla <br />

**Related:**
[PR #2290 - Schematic Action Codec and Dispatch Endpoint](https://github.com/synnaxlabs/synnax/pull/2290),
[RFC 0036 - Flux Suspense](./0036-260507-flux-suspense.md)

# 0 - Summary

Pluto's Flux substrate gains a generic undo and redo capability for any document type
that mutates through a discriminated-union action API. Each action handler — both client
and server — changes shape to return the inverse of the operation it just applied,
computed against the actual pre-state. The undo stack lives on the client, in the Flux
cache entry for the document, scoped per window. Undo dispatches the stored inverse
through the same `dispatch` pipeline as any user edit, so the server, the cluster
broadcast, and the conflict-handling code stay unchanged. Redo is symmetric.
Transactions group rapid mutations into a single undoable unit at UI affordance
boundaries.

The schematic is the first integration and ships in this work. Line plots, tables, logs,
and any future document type built on the action substrate get undo by implementing the
same reducer-with-inverse contract; no per-type stack, hook, or keyboard plumbing is
required. The server gains no new endpoints and no history storage. Durable version
history is a separate feature and is out of scope.

# 1 - Vocabulary

- **Document** - A record managed by the Flux substrate that mutates only through
  actions. A schematic is one document type; a line plot is another.
- **Document state** - The shared, broadcast state of a document. Lives in the Flux
  cache. For a schematic: nodes, edges, configs.
- **Session state** - Per-window UI state (selection, viewport, toolbar, etc.) held
  outside the action substrate. Never undoable; the undo system never sees it.
- **Action** - A single discriminated-union mutation defined per document type and
  generated from a `.oracle` schema (e.g., `SetNodePosition` for schematics).
- **Reducer** - A per-document-type function that applies an action and returns its
  inverse. One implementation in Go and one in TS per document type.
- **Inverse** - A vector of actions that, applied to the post-state, restores the
  pre-state.
- **Transaction** - A group of actions dispatched together as a single undoable unit.
  Equivalent to one `dispatch(actions[])` call.
- **Forward** - The original actions in a transaction.
- **Stack entry** - A record of `{forward, inverse, kind, ts}` on the undo or redo
  stack.
- **Undoable action** - An action that contributes to the inverse vector. Non-undoable
  actions (e.g., `SetNodeMeasured`) are applied normally but produce an empty inverse
  contribution.
- **Coalescing** - Merging two adjacent stack entries of the same kind within a
  configured time window so that one undo reverses both.
- **Stale entry** - A stack entry whose targets have been mutated by a remote session
  since the entry was pushed.

# 2 - Motivation

## 2.0 - The Action Substrate Is Already Reversible in Principle

PR #2290 introduced action-based dispatch for schematics. Subsequent work will give the
same shape to other documents (line plots, tables, logs). Each action is small,
well-typed, and replayable. This is the substrate undo systems are built on — across
document types. What is missing is the function from "action just applied" to "actions
that would reverse it" and a place to keep those inverses while a user is editing.

## 2.1 - Users Expect Undo

Schematics are the primary authoring surface in the Console; line plots and tables are
not far behind. Building a non-trivial document of any of these types involves dozens of
small placements, alignments, and parameter tweaks, many of which are exploratory.
Without undo, every mistake forces a manual revert and every experiment costs a manual
cleanup. Undo is table-stakes for any document editor.

## 2.2 - Per-Document-Type Undo Doesn't Scale

If undo ships as schematic-only code, every future action-based document type
re-implements the same machinery: the stack, the coalescing, the keyboard hook, the
staleness detection, the multi-pane sharing, the transaction API. Each instance drifts
from the others, each accumulates its own bugs, and the overall surface gets worse the
more action-based documents we have. The right level for this work is the substrate.
Per-document-type code only contributes the parts that genuinely vary by type — the
action set, the inverse logic, the kind tags. Everything else is shared.

## 2.3 - "Server-Side Undo" Conflates Two Features

The original framing (`sy-3038-add-server-side-undo-and-redo`) treats undo as a durable,
server-owned feature. This conflates two distinct user needs:

1. **In-session correction** ("take that back, I did it wrong") — sub-second feedback,
   per-user, lost on close. Every editor since the 80s works this way and users expect
   it.
2. **Time travel** ("show me what this looked like yesterday") — durable, named,
   browsable. This is version history, not undo.

A single mechanism that tries to serve both ends up bad at both: every undo becomes a
network round-trip, the server acquires UI-level concepts (transactions, coalescing,
redo trees), and users still don't get the version-history affordance they actually want
for time travel. The right architecture separates these layers and ships the in-session
feature now. Version history can be designed independently and built later on the same
action substrate.

# 3 - Principles

- **The substrate owns undo; document types own actions.** The stack, transaction API,
  coalescing, keyboard hook, staleness detection, and lifecycle live in the Flux
  substrate. Per-document-type code contributes only a reducer that returns inverses, an
  undoable predicate, and a kind classifier. Adding undo to a new document type is a
  matter of providing those pieces, not implementing a new system.
- **Undo operates on document state only.** The dispatch pipeline mutates only the
  shared document; session state (selection, viewport, toolbar, legend, control
  authority) is held outside the action substrate and never produces actions. Whether ⌘Z
  affects selection, viewport, etc. is not a design choice — those concerns are
  structurally outside the action substrate. The undo system mirrors that boundary.
- **The reducer is the single source of truth for inverses.** The same handler that
  knows how to apply an action knows what state it is overwriting. Inverse logic lives
  there and nowhere else.
- **Inverses are computed at apply time, not action-creation time.** A static inverse
  table cannot express the inverse of partial-merge actions or composite actions. The
  pre-state is required.
- **Undo is a vector of actions, not a special action type.** The server has no `Undo`
  action. Undo is "dispatch the stored inverse," which is a normal `dispatch` call.
- **The undo stack is a client concept.** It lives in the Flux cache entry, per window,
  in memory. Closing the window drops the stack. This matches every desktop editor and
  avoids the latency cost of a server round-trip per ⌘Z.
- **The server stays unchanged.** No new endpoints, no schema changes, no history
  storage. The dispatch pipeline already supports everything undo needs.
- **Multiplayer is best-effort, not transformed.** When a remote session has touched an
  entity since a stack entry was pushed, undo of that entry applies the inverse blindly
  through the reducer, which is defensive about missing or modified targets. No
  operational transformation. The substrate auto-advances past stale entries so every ⌘Z
  does something visible.
- **Transactions are UI-defined, not action-defined.** A drag is one transaction even
  though it dispatches dozens of position deltas. The dispatch list is the transaction
  boundary; UI code controls when to flush.

# 4 - Design

## 4.0 - The Generic Reducer Contract

Every action-based document type defines a reducer with the same shape:

```ts
type ActionReducer<S, A> = (state: Draft<S>, action: A) => A[];
//                                                       ^ inverse, may be empty
```

```go
type ActionReducer[S any, A any] interface {
    Apply(state *S, action A) (inverse []A, err error)
}
```

Each per-action handler computes its inverse from the pre-state it sees and then mutates
the state. The inverse is a slice because composite actions invert to multiple actions.

`ReduceAll` returns the concatenation of inverses in reverse application order — the
single inverse of the whole transaction:

```go
func ReduceAll[S any, A any](r ActionReducer[S, A], state *S, actions []A) ([]A, error) {
    inverse := make([]A, 0, len(actions))
    for _, a := range actions {
        inv, err := r.Apply(state, a)
        if err != nil { return nil, err }
        inverse = append(inv, inverse...) // prepend
    }
    return inverse, nil
}
```

The server uses the inverse only to validate that all actions in a transaction applied;
it does not store or return it. The client uses the inverse to populate its stack.

The schematic reducer is the first concrete implementation:

| Action              | Inverse (against actual pre-state)                         |
| ------------------- | ---------------------------------------------------------- |
| `SetNodePosition`   | `SetNodePosition(K, oldPosition)`                          |
| `SetNode` (replace) | `SetNode(oldNode, oldConfig)`                              |
| `SetNode` (insert)  | `RemoveNode(K)`                                            |
| `RemoveNode`        | `SetNode(oldNode, oldConfig) + AddEdge(e_i)` for each edge |
| `AddEdge`           | `RemoveEdge(K)`                                            |
| `RemoveEdge`        | `AddEdge(oldEdge) + SetConfig(K, oldEdgeConfig)?`          |
| `SetConfig`         | `SetConfig(K, oldFields)`                                  |
| `SetNodeMeasured`   | `[]` (non-undoable; renderer feedback)                     |

## 4.1 - Generic Undoability Marker

`undoable: bool` is a generic action property declarable in any `.oracle` schema's
action definition. Oracle generates a per-type `UndoableActions` set in both Go and TS.
The reducer for non-undoable actions returns an empty inverse. The substrate
short-circuits transactions whose every action is non-undoable, so they never push a
stack entry.

```
// schemas/schematic.oracle
action SetNodeMeasured {
    undoable: false
    key: string
    measured: Dimensions
}
```

## 4.2 - `Flux.createDispatch`: The Action-Based Mutation Factory

Pluto's Flux substrate exposes two parallel mutation factories:

- **`Flux.createUpdate`** (existing) — free-form update path. The user provides an
  arbitrary `update` callback that does optimistic apply, server send, and rollback on
  its own. No undo. Used for non-action-based mutations.
- **`Flux.createDispatch`** (new) — action-based mutation path. The substrate owns the
  local apply via a reducer, owns the stack, and owns the undo machinery. The user
  provides only the reducer, the send function, and a small classifier for coalesce
  kinds.

These are siblings, not wrappers. Document types pick the one that matches their
mutation style. Action-based documents (schematic, eventually line plot, table) adopt
`createDispatch` and inherit undo automatically:

```ts
const { useDispatch, useUndo, useRedo } = Flux.createDispatch<
  Schematic,
  schematic.Action,
  FluxSubStore
>({
  resourceName: "schematic",
  reduce: schematic.reduce,
  isUndoable: schematic.isUndoable,
  kindOf: schematic.kindOf,
  coalesceMs: 500,
  send: ({ client, key, sessionKey, actions }) =>
    client.schematics.dispatch(key, sessionKey, actions),
});
```

`createDispatch` returns three hooks:

- `useDispatch` — sends actions; runs the reducer, applies optimistically, pushes to the
  stack, calls `send`, rolls back and pops on server reject. Exposes the transaction API
  (`beginTransaction`).
- `useUndo` — pops the top of the undo stack, dispatches its inverse internally, pushes
  onto the redo stack. Auto-advances past stale entries (4.8).
- `useRedo` — symmetric.

Per-document-type code provides only the configuration. Adding undo to a new document
type is one call to `createDispatch` with that type's reducer and send function — the
substrate carries the rest.

## 4.3 - Stack Shape

The stacks live on the cache entry and are typed by the document's action type:

```ts
type StackEntry<A> = {
  forward: A[]; // what the user dispatched
  inverse: A[]; // what the reducer returned, in reverse application order
  kind: string; // for coalescing; internal-only in v1
  ts: number; // monotonic, for coalescing window
  staleKeys?: Set<string>; // populated lazily by listener
};

type UndoableCacheEntry<S, A> = {
  value: S;
  undo: StackEntry<A>[]; // newest at end
  redo: StackEntry<A>[]; // newest at end
  // ... existing Flux fields
};
```

Stacks are not persisted across reloads. Not synced via Drift. Not part of the broadcast
surface. They live and die with the cache entry.

## 4.4 - Dispatch and Push

The substrate-provided `useDispatch` runs the local reducer pre-pass (which already
exists for optimistic apply; this just captures the inverse it returns), pushes to the
stack, and sends to the server:

```
1. Run reducer locally with the actions, capturing newState and inverse.
2. Apply newState optimistically.
3. If at least one action is undoable and the transaction is not internally
   generated:
     stack.undo.push({forward: actions, inverse, kind: kindOf(actions), ts: now})
     stack.redo = []
     coalesce(stack.undo)
4. Send actions to the server.
5. On server reject: roll back local state, also pop the entry pushed in step 3.
```

A transaction is "internally generated" when it originates from undo or redo themselves,
or when every action in it is non-undoable.

## 4.5 - Transaction Boundaries

The stack's grain is one transaction per `dispatch(actions[])` call. This is the natural
unit:

- **Drag a node.** UI buffers position deltas during the drag, calls `dispatch` once on
  `mouseup`. One stack entry, one undoable.
- **Add a node from the palette.** UI dispatches one batch with the node and any initial
  edges. One stack entry.
- **Multi-select delete.** UI dispatches the batch of removes in one call. One stack
  entry; one undo restores all.
- **Property-panel commit.** Toolbar field dispatches one transaction per committed
  field change (on blur or enter), not per keystroke. One field-edit, one undo.

For continuous gestures that today stream actions for live preview, the substrate
exposes an explicit transaction API:

```ts
const tx = dispatch.beginTransaction({ kind: "move" });
tx.add([SetNodePosition(k, p1)]);
tx.add([SetNodePosition(k, p2)]);
// ...
tx.commit(); // sends the batch, pushes one stack entry
// or tx.abort() — rolls back local applications, no push, no send.
```

The transaction handle is window-local and does not cross into the server. Its purpose
is to coalesce the dispatch payload AND the stack entry.

## 4.6 - Undo and Redo

```
undo():
    while stack.undo not empty:
        entry = stack.undo.pop()
        if entry is stale (4.8) and would produce no visible effect:
            continue (auto-advance), up to a soft cap of 10
        dispatch(entry.inverse, internal=true)
        stack.redo.push(entry)
        return
    // stack exhausted or cap hit; affordance shows "nothing to undo"

redo():
    if stack.redo empty: return
    entry = stack.redo.pop()
    dispatch(entry.forward, internal=true)
    stack.undo.push(entry)
```

The reciprocal symmetry holds because a transaction's inverse, applied to its
post-state, restores the pre-state — and the inverse of the inverse, computed against
that restored pre-state by the reducer, is functionally the original forward (modulo
state-dependent fields, which the reducer recomputes correctly at redo time). The
substrate pushes the entry as-is on undo and relies on the reducer for correctness on
redo. If empirical drift surfaces, the alternative is to recompute and store the
reciprocal at undo time; this is a one-line change and not load-bearing.

## 4.7 - Coalescing

After a push, the substrate tests the top two entries of `stack.undo` for coalesce:

- Same `kind`.
- Same target keys (e.g., both moving node K).
- `ts` within the coalesce window (default 500ms; configurable per document type).

When all three hold, the entries merge: `forward` concatenates, `inverse` prepends, `ts`
updates to the newer. The user sees one ⌘Z reverse the entire run.

Coalescing is per-kind. The kind tag is internal — the v1 surface is keyboard-only (⌘Z,
⌘⇧Z, ⌘Y on Windows) and there is no menu item, button, or descriptive label. A future
"Edit → Undo X" affordance can read the same tag if desired.

## 4.8 - Multiplayer Behavior

Remote actions (those received via cluster broadcast with a different `sessionKey`) do
not push to the local stack. They do, however, register the keys they touched on a
short-lived "remotely-touched" set on the cache entry. When undo pops a stack entry, the
substrate checks whether any of the entry's target keys appear in this set since the
entry's `ts`; if so, the entry is flagged stale.

When undo pops a stale entry, the reducer applies the inverse and produces no visible
effect — the targets are gone or already in a state the inverse cannot restore. Rather
than burn the user's ⌘Z press on an invisible no-op, the undo handler **auto-advances**
past stale entries: it keeps popping and applying inverses until either an entry
produces a visible effect or the stack is empty. A soft cap (10 consecutive stale pops)
bounds the walk so a single ⌘Z cannot silently reach back too far in pathological cases;
on hitting the cap, the handler stops and the affordance shows "nothing to undo."

This is the same posture Figma and Google Docs take. The user is multiplayer-aware
already; predictable press-counts are not achievable in a shared document. The right
invariant is "every ⌘Z that completes does something visible, or the stack is
exhausted."

This is intentionally simpler than operational transformation. The collaboration
patterns we see today — small teams, mostly serial editing — do not justify OT's
implementation cost. The stale-key seam is where OT slots in later if the use case
materializes.

## 4.9 - Bounds and Lifecycle

- **Cap.** Each stack is capped at 200 entries. Older entries fall off the bottom.
- **Cache-lifetime.** Stacks live with the Flux cache entry, not with the on-screen
  component. Navigating away from a document to another layout in the same window does
  not clear the stack; navigating back resumes where the user left off. This follows
  from the one-way hydration rule established in RFC 0036: once a document is in cache
  it stays in cache, and the stack rides along.
- **Listeners must keep running while the document is off-screen.** Stale-detection
  depends on the cache entry receiving remote actions even when the user is on a
  different layout. The existing action listeners are registered at provider mount and
  run for the life of the app; called out so future refactors do not regress it.
- **Two panes on one document share one stack.** Multiple mounts of the same document in
  the same window resolve to the same Flux cache entry, so they share `undo` and `redo`.
  ⌘Z works the same regardless of which pane has focus. This matches user intent: one
  person, one document, one history.
- **Cache eviction takes the stack with it.** When the cache evicts an entry (LRU,
  manual purge, logout, cluster switch), the stacks are dropped along with the document.
  There is no separate stack persistence.
- **Stack memory.** A 200-entry cap, with each entry holding small forward and inverse
  vectors of typed actions, is bounded and small relative to the document itself. No
  further policy needed unless profiling shows otherwise.
- **Read-only documents.** When a document is read-only (e.g., schematic with
  `snapshot == true`), dispatch is rejected by the server. Undo and redo are no-ops in
  this state and the affordance is hidden.

## 4.10 - Keyboard Bindings

Keyboard bindings live at the layout level, not the substrate level. Each layout type
that owns an action-based document mounts a small `<UndoBindings>` component that wires
⌘Z and ⌘⇧Z (plus ⌘Y on Windows) to the active document's `useUndo` and `useRedo`. The
bindings activate when the layout has focus and yield to layout defaults when it does
not. There is no global undo registry; each layout controls its own bindings, and other
layouts can have different undo or none at all.

# 5 - Migration

## 5.0 - Substrate Work

1. **Generic action property in oracle.** Add `undoable: bool` (default `true`) as a
   top-level field on action declarations, available in any `.oracle` schema. Update Go
   and TS generators to emit a per-type `UndoableActions` set.
2. **Generic reducer signature.** Update the generators to emit the
   `ActionReducer<S, A>` shape (returning `A[]` inverse) per document type. Hand-written
   reducer handlers must conform.
3. **`Flux.createDispatch` factory** in `pluto/src/flux/`, sibling to the existing
   `Flux.createUpdate`:
   - Owns the optimistic apply path via the user-supplied reducer.
   - Stack and entry types, push/pop, coalesce, transaction handle, internal-dispatch
     flag, stale-keys tracking.
   - Returns `useDispatch`, `useUndo`, `useRedo` parametrized by document type.
   - Substrate tests against a synthetic document type so the substrate is validated
     independently of any real document.
4. **`<UndoBindings>` component** that any layout can mount to wire keyboard shortcuts
   to the active document's undo and redo.

## 5.1 - Schematic Integration

1. Mark `SetNodeMeasured` as `undoable: false` in `schemas/schematic.oracle`. Run oracle
   sync.
2. Update `core/pkg/service/schematic/actions.go` and
   `client/ts/src/schematic/actions.ts` to return inverses; one unit test per action
   asserting that applying the inverse to the post-state restores the pre-state.
3. Replace the schematic's `Flux.createUpdate` call in `pluto/src/schematic/queries.ts`
   with `Flux.createDispatch`. The existing `useDispatch` consumer surface is preserved;
   `useUndo` and `useRedo` are new.
4. Migrate drag, multi-select align, paste, and property-panel commit handlers in the
   schematic canvas and toolbar to the transaction API.
5. Mount `<UndoBindings>` in the schematic layout component.

## 5.2 - Future Document Types

Line plots, tables, and any other document type adopt undo by migrating from
`Flux.createUpdate` to `Flux.createDispatch` with their own reducer and send function.
No substrate change required, no per-type stack code.

# 6 - Alternatives Considered

## 6.0 - Server-Side Stack

Per-(user, document) action history stored on the server; ⌘Z dispatches a new `Undo`
action to the server, which pops, recomputes the inverse against current authoritative
state, and applies. **Rejected** because the only substantive win is durability across
window reloads — and every editor users have ever used drops the stack on close. The
costs are: a network round-trip on every ⌘Z, a new endpoint, a new schema concept,
storage growth, and conflation with version history. The version-history feature serves
the durability need better and ships independently.

## 6.1 - Event-Sourced Stack

Server persists every action as an append-only log; the client's stack is a query
against that log. **Rejected** because it pays the durability cost without delivering
the durability product. The log infrastructure is a separate project (audit, replay,
time-travel debugging) that, if and when it ships, undo can ride on top of for free.
Building it primarily for undo is overinvestment.

## 6.2 - Operational Transformation

Transform stored inverses against intervening remote actions before applying.
**Rejected** for now because the collaboration patterns we see today do not justify the
implementation cost. The stale-keys seam in 4.8 is where OT slots in later if the use
case materializes.

## 6.3 - Static Inverse Table

A `inverse(action) -> action` function with no state argument. **Rejected** because
partial-merge actions like `SetConfig` and composite actions like `RemoveNode` have
inverses that depend on the pre-state. Any non-trivial schema lands here eventually.

## 6.4 - Branching Redo Tree

Sublime-style; redo branches when a new action is dispatched after some undos.
**Rejected** as not worth the UX cost. Linear redo is universally understood; the
branching variant exists in a few editors and is rarely discovered. Clearing redo on new
dispatch is the right default.

## 6.5 - Per-Document-Type Implementation

Build the stack, transactions, coalesce, and keyboard hook inside each document type's
package. **Rejected** because every future action-based document type would re-implement
the same surface, drift from the others, and accumulate divergent bugs. The substrate
cost is one factory function and a small set of generic interfaces; the cost of not
doing it grows linearly with the number of document types.

# 7 - Open Questions

- **Coalesce parameters.** The 500ms window and per-kind policy are reasonable defaults.
  Ship with these and tune from telemetry on real usage.
- **`kindOf` granularity.** Document types classify a transaction by inspecting its
  actions. The granularity (one kind per action shape vs. one kind per UI gesture) is a
  per-type decision. Schematic ships with `move`, `addNode`, `removeNode`, `addEdge`,
  `removeEdge`, `configChange`, `paste` as a starting taxonomy.
- **Schematic action gaps surfaced by inverses.** Implementing inverses for the current
  schematic action set surfaced two real gaps in the schema:
  - `SetConfig` is a merge, not a replace. The inverse cannot remove keys the forward
    action introduced; phantom fields persist after undo. Closing the gap requires a
    `ReplaceConfig` action (wholesale replace, including remove when the payload is
    absent).
  - `SetNode` and `RemoveNode` operate on the nodes slice without an explicit index. The
    inverse of `RemoveNode` re-inserts at the end of the slice, so a remove + undo cycle
    restores contents but not the original index. Closing the gap requires an
    `InsertNode(node, idx)` action. Both are tracked as schema follow-ups. The v1
    reducer ships with documented imperfections in the affected paths and the
    `expectUserVisibleRoundTrip` test helper acknowledges them; nodes-and-edges identity
    and overwritten config-field values round-trip correctly.

# 8 - Implementation Plan

1. **Substrate**: oracle `undoable` field, generic reducer signature, the
   `Flux.createDispatch` factory, the `<UndoBindings>` component, substrate-level tests
   against a synthetic document type. No user-visible behavior change yet.
2. **Schematic reducer**: per-action inverse logic in Go and TS, per-action tests.
3. **Schematic integration**: replace `useDispatch`, migrate drag, multi-select, paste,
   and property-panel commit handlers to the transaction API, mount `<UndoBindings>` in
   the schematic layout.
4. **Stale detection**: substrate-level listener integration with the remotely-touched
   key set; integration tests via the Playwright suite covering concurrent-edit
   scenarios.
5. **Future**: same factory adopted by line plots, tables, and other action-based
   document types as they migrate.

Versioned snapshots ("named history") are out of scope and tracked separately.
