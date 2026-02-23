# 27 - Action-Based State Management for Meta-Data Structures

**Feature Name**: Action-Based State Management <br /> **Status**: Draft <br /> **Start
Date**: 2026-02-23 <br /> **Authors**: Emiliano Bonilla <br /> **Depends On**: RFC 0025
(Meta-Data Structures), RFC 0026 (Oracle Schema System)

# 0 - Summary

This RFC proposes an action-based state management architecture for Synnax's meta-data
structures (schematics, tables, line plots, etc.). Instead of transmitting entire
documents on every edit, clients dispatch granular, typed actions that flow through a
reducer pipeline shared between client and server. This design enables efficient network
transport, per-user undo/redo, and lays the architectural foundation for real-time
collaborative editing.

The design draws heavily from production collaborative editing systems (Figma, TLDraw,
Liveblocks, Linear) and adapts their patterns to Synnax's existing infrastructure:
oracle for cross-language code generation, Flux for reactive client-side state, and the
signal channel system for real-time broadcast.

# 1 - Motivation

RFC 0025 identifies several critical problems with the current meta-data toolchain. This
RFC addresses four of them directly:

- **1.1.4 - Inefficient Network Transport**: Panning a schematic sends the entire
  document (thousands of properties) to the server. An action-based system sends only
  `{type: "set_node_position", key: "abc", position: {x: 10, y: 20}}`.
- **1.1.2 - No Undo/Redo**: Action-based mutations naturally produce diffs that can be
  reversed, enabling per-user undo/redo stacks.
- **1.1.8 - Multiple Sources of Truth**: Document state moves entirely to Flux (backed
  by the server). Redux holds only client-local view state. One source of truth.
- **1.1.5 - No Collaborative Editing**: Granular actions are mergeable. Property-level
  last-writer-wins conflict resolution (the Figma model) becomes feasible.

# 2 - Prior Art

## 2.0 - Figma

Figma models documents as `Map<ObjectID, Map<Property, Value>>` — a flat two-level map.
Conflict resolution is property-level last-writer-wins, with the server defining event
ordering. Each user maintains their own undo/redo stack. Undo pushes inverse operations
computed against the current state (not the original state). Figma is explicitly **not
using CRDTs** — they use CRDT-inspired techniques with a central server authority.

Key decisions relevant to Synnax:

- Per-user undo stacks (universal across all production systems)
- Server as ordering authority (no vector clocks or Lamport timestamps needed)
- Deleted object properties stored in the undo buffer of the deleting client, not on the
  server
- LWW is sufficient for structured/graph data; CRDTs are only needed for rich text

## 2.1 - TLDraw

TLDraw uses a record-based store where every mutation produces a `RecordsDiff`:

```typescript
interface RecordsDiff<R> {
  added: Record<IdOf<R>, R>;
  updated: Record<IdOf<R>, [from: R, to: R]>;
  removed: Record<IdOf<R>, R>;
}
```

Diffs are composable (`squashRecordDiffs`), reversible (`reverseRecordsDiff`), and form
the basis of both sync and undo/redo. The history stack contains diffs interleaved with
**marks** (named stopping points). Undo replays diffs backward until hitting a mark.
`editor.run()` batches multiple operations into a single transaction/diff.

TLDraw separates state into three scopes:

| Scope    | Persisted | Synced | Undoable |
| -------- | --------- | ------ | -------- |
| Document | Yes       | Yes    | Yes      |
| Session  | Local     | No     | No       |
| Presence | No        | Yes    | No       |

## 2.2 - Liveblocks

Liveblocks uses LWW with acknowledgment-based ordering. Each mutation records a reverse
operation alongside the forward operation. Each user maintains their own undo/redo
stack. Critically, Liveblocks provides `pause()`/`resume()` for grouping interactions —
all mutations between pause and resume become a single undoable action (essential for
drag operations).

When an undo targets an object deleted by another user, the operation is silently
skipped. This is the same approach used by Figma and Google Slides.

## 2.3 - Linear

Linear uses a server-centric sync engine with MobX for client-side reactivity and
IndexedDB for local persistence. All operations execute on the server, with clients
performing optimistic local updates. Transactions track old/new values for each property
change, enabling local rollback. LWW for all property updates — CRDTs only recently
adopted for rich text (issue descriptions).

## 2.4 - Patterns Common to All Systems

1. **Per-user undo stacks**: No production system allows undoing another user's actions.
2. **Inverse operations over state snapshots**: All systems store diffs or inverse
   operations, never full state snapshots. Snapshots erase concurrent changes.
3. **Transaction/batch grouping**: All systems batch low-level mutations into single
   undoable actions (drag = one undo step, not 60 intermediate positions).
4. **LWW is sufficient for structured data**: CRDTs are reserved for rich text. For
   canvas/graph tools, property-level LWW works because concurrent edits to the same
   property on the same object are rare.
5. **Skip conflicting undos**: When undo targets a deleted/modified object, skip rather
   than resolve. Simpler, handles a rare edge case acceptably.

# 3 - Design

## 3.0 - Architecture Overview

### 3.0.0 - System Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                         CONSOLE (Tauri)                             │
│                                                                     │
│  ┌─────────────────────────┐    ┌────────────────────────────────┐  │
│  │      Redux (Session)    │    │         Flux (Document)        │  │
│  │                         │    │                                │  │
│  │  viewport, toolbar,     │    │  ScopedUnaryStore<Schematic>   │  │
│  │  edit mode, control     │    │  ┌──────────┐ ┌────────────┐  │  │
│  │                         │    │  │ reducer  │ │ channel    │  │  │
│  │  Synced across windows  │    │  │ (relapse)│ │ listener   │  │  │
│  │  via Drift IPC          │    │  └─────┬────┘ └──────┬─────┘  │  │
│  └─────────────────────────┘    │        │             │        │  │
│                                 │        │      sy_schematic_set │  │
│                                 └────────┼─────────────┼────────┘  │
│                                          │             │           │
│  ┌───────────────────────────────────────┼─────────────┼────────┐  │
│  │            Pluto Components           │             │        │  │
│  │                                       │             │        │  │
│  │  React Flow ──onNodesChange──► action │             │        │  │
│  │              ──onEdgesChange──► action │             │        │  │
│  └───────────────────────────────────────┼─────────────┼────────┘  │
└──────────────────────────────────────────┼─────────────┼────────────┘
                                           │             │
                              dispatch RPC │             │ framer stream
                                           │             │
┌──────────────────────────────────────────┼─────────────┼────────────┐
│                      SYNNAX SERVER       │             │            │
│                                          ▼             │            │
│  ┌───────────────────────────────────────────┐         │            │
│  │              API Layer                    │         │            │
│  │  RBAC ──► WithTx ──► Writer.Dispatch      │         │            │
│  └───────────────────────┬───────────────────┘         │            │
│                          │                             │            │
│  ┌───────────────────────▼───────────────────┐         │            │
│  │            Service Layer                  │         │            │
│  │                                           │         │            │
│  │  Writer                                   │         │            │
│  │  ├── gorp.ChangeErr (read-modify-write)   │         │            │
│  │  │     └── ReduceAll(state, actions)       │         │            │
│  │  └── actionObserver.Notify(ScopedAction)  │         │            │
│  │                          │                │         │            │
│  │  Service                 │                │         │            │
│  │  ├── actionObserver ◄────┘                │         │            │
│  │  └── signals (PublishFromObservable)───────┼────►    │            │
│  └───────────────────────────────────────────┘  sy_schematic_set   │
│                                                        │            │
│  ┌─────────────────────────────────────────────────────┼──────────┐ │
│  │              Storage Layer                          │          │ │
│  │  gorp/Pebble (persisted Schematic state)    framer (signal     │ │
│  │                                             channel writes)    │ │
│  └────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

### 3.0.1 - State Ownership

```
┌─────────────────────────────────────────────────────────────┐
│                     Schematic State                          │
├──────────────────────────┬──────────────────────────────────┤
│    Document (Flux)       │      Session (Redux)             │
│                          │                                  │
│  nodes: Node[]           │  viewport: { x, y, zoom }       │
│  edges: Edge[]           │  mode: "select" | "pan"         │
│  props: Record<K, V>    │  editable: boolean               │
│  name: string            │  fitViewOnResize: boolean        │
│  snapshot: boolean       │  legend: { visible, position }   │
│                          │  toolbar: { activeTab }          │
│                          │  control: Control.Status         │
│                          │  authority: number               │
│  Persisted: server       │  selection: string[]             │
│  Synced: all clients     │                                  │
│  Mutations: actions      │  Persisted: local disk           │
│  Undoable: yes (future)  │  Synced: windows (Drift)         │
│                          │  Mutations: Redux dispatch       │
│                          │                                  │
│  Note: Node[] in Flux    │  Note: selection is stored       │
│  has NO selected field.  │  separately, merged with Flux    │
│  Selection is session    │  nodes at render time before     │
│  state only.             │  passing to React Flow.          │
└──────────────────────────┴──────────────────────────────────┘
```

### 3.0.2 - Before vs. After

**Before (current)**: Every edit sends the entire document.

```
User drags node ──► Redux dispatch(setNodes([...all nodes...]))
                         │
                         ├──► Redux state updated (all windows via Drift)
                         │
                         └──► setData(key, { ...entire schematic... })  ──► Server
                                        ~50 KB per edit
```

**After (this RFC)**: Only the mutation is sent.

```
User drags node ──► Flux dispatch(setNodePosition({ key: "abc", position: {x:100, y:200} }))
                         │
                         ├──► Local reducer applies instantly
                         │
                         └──► dispatch(key, [action])  ──► Server
                                  ~80 bytes per edit
```

### 3.0.3 - Multi-Client Collaboration

```
              User A (originator)              Synnax Server              User B (observer)
              ─────────────────              ──────────────              ─────────────────
                     │                              │                          │
  drag node "abc"    │                              │                          │
  to (100, 200)      │                              │                          │
                     │                              │                          │
  ┌──────────────┐   │                              │                          │
  │TS reducer    │   │                              │                          │
  │applies local │   │                              │                          │
  │(instant)     │   │                              │                          │
  └──────────────┘   │                              │                          │
                     │  POST /schematic/dispatch     │                          │
                     │  { key, session_id: "A",     │                          │
                     │    actions: [{               │                          │
                     │      type: "set_node_pos",   │                          │
                     │      set_node_pos: {         │                          │
                     │        key: "abc",           │                          │
                     │        position: {x:100,     │                          │
                     │                   y:200}     │                          │
                     │    }}]}                       │                          │
                     │ ─────────────────────────►   │                          │
                     │                              │                          │
                     │                   ┌──────────┴──────────┐               │
                     │                   │ Go reducer applies  │               │
                     │                   │ gorp persists       │               │
                     │                   │ observer.Notify()   │               │
                     │                   └──────────┬──────────┘               │
                     │                              │                          │
                     │                              │  sy_schematic_set frame  │
                     │                              │  (newline-delimited JSON)│
                     │    ◄─────────────────────────┼─────────────────────►    │
                     │                              │                          │
  ┌──────────────┐   │                              │   ┌──────────────────┐   │
  │session_id   │   │                              │   │session_id != "A"│   │
  │== "A"        │   │                              │   │TS reducer applies│   │
  │DROP (already │   │                              │   │React re-renders  │   │
  │applied)      │   │                              │   │User B sees node  │   │
  └──────────────┘   │                              │   │at (100, 200)     │   │
                     │                              │   └──────────────────┘   │
```

### 3.0.4 - Code Generation Flow (Oracle)

```
                        .oracle schema file
                    ┌─────────────────────────┐
                    │ type Schematic {         │
                    │   nodes: []Node          │
                    │   edges: []Edge          │
                    │   ...                    │
                    │ }                        │
                    │                          │
                    │ action SetNodePosition { │
                    │   key: string            │
                    │   position: XY           │
                    │ }                        │
                    │                          │
                    │ action AddNode {         │
                    │   node: Node             │
                    │   props: map<string,any> │
                    │ }                        │
                    └────────────┬─────────────┘
                                │
                    ┌───────────┼───────────┐
                    │           │           │
                    ▼           ▼           ▼
            ┌───────────┐ ┌─────────┐ ┌──────────┐
            │    Go     │ │   TS    │ │ Protobuf │
            ├───────────┤ ├─────────┤ ├──────────┤
            │           │ │         │ │          │
            │ Action    │ │ actionZ │ │ Action   │
            │ struct    │ │ (zod)   │ │ message  │
            │ (union)   │ │         │ │ (oneof)  │
            │           │ │ create- │ │          │
            │ Reduce()  │ │ Action  │ │ payload  │
            │ switch    │ │ calls   │ │ messages │
            │           │ │         │ │          │
            │ ReduceAll │ │ create- │ │          │
            │           │ │ Reducer │ │          │
            │ NewXxx()  │ │         │ │          │
            │ ctors     │ │ scoped- │ │          │
            │           │ │ ActionZ │ │          │
            └─────┬─────┘ └────┬────┘ └────┬─────┘
                  │            │            │
                  ▼            ▼            ▼
            Developer     Developer     Wire format
            writes        writes        for dispatch
            Handle()      handler:      RPC
            methods       functions
```

## 3.1 - State Separation

Following TLDraw's model, all state for a meta-data structure is classified into one of
three scopes:

| Scope    | Owner | Persisted  | Synced      | Undoable | Example                                              |
| -------- | ----- | ---------- | ----------- | -------- | ---------------------------------------------------- |
| Document | Flux  | Server     | All clients | Yes      | Nodes, edges, props, name, snapshot                  |
| Session  | Redux | Local disk | Windows\*   | No       | Viewport, selection, mode, editable, legend, toolbar |
| Presence | Flux  | No         | All clients | No       | Cursor position\*                                    |

\*Session state syncs across windows via Drift (existing behavior). Presence — real-time
awareness of other users' activity (cursor positions, active selections) — is a future
concern and not part of this RFC's implementation scope.

**The critical migration**: document state moves out of Redux slices and into Flux. The
console's schematic Redux slice currently holds ~20 actions mixing document and session
state. After migration, the Redux slice retains only session-scoped actions (viewport,
toolbar, edit mode, control status). Document-scoped actions (add node, set position,
set props, remove edge) flow through the new action pipeline.

## 3.2 - Action Pipeline

### 3.2.0 - Data Flow

The complete lifecycle of a user edit:

```
User drags a node
  │
  ▼
Console dispatches a ScopedAction { key: schematicKey, type: "set_node_position", ... }
  │
  ├──────────────────────────────────────────────────────────────────┐
  │ OPTIMISTIC (instant)                                            │
  ▼                                                                 │
Local Flux store applies action via TS reducer                      │
  │                                                                 │
  ▼                                                                 │
React re-renders (user sees node at new position immediately)       │
  │                                                                 │
  │ PERSISTENT (async)                                              │
  ▼                                                                 │
Client sends action to server via Dispatch API endpoint             │
  │                                                                 │
  ▼                                                                 │
Server applies action via Go reducer (source of truth)              │
  │                                                                 │
  ▼                                                                 │
Server persists updated state to gorp/Pebble                        │
  │                                                                 │
  ▼                                                                 │
Server broadcasts action on signal channel (sy_schematic_set)      │
  │                                                                 │
  ▼                                                                 │
All other clients receive action via Flux channel listener ◄────────┘
  │
  ▼
Each client applies action via TS reducer → React re-renders
```

### 3.2.1 - Optimistic Application with Server Authority

The originating client applies actions optimistically for instant UI feedback. The
server is the authoritative reducer — it applies the same action independently, persists
the result, and broadcasts.

**Self-broadcast deduplication**: The server broadcasts actions to all connected
clients, including the originator. Each broadcast is tagged with the originating session
ID. When a client's Flux channel listener receives a broadcast, it checks whether the
action originated from itself — if so, it **drops the action** rather than re-applying
it. The optimistic local state is already correct.

This is necessary because many actions are not idempotent (`AddNode` applied twice would
create duplicate nodes). It also prevents visual flickering from redundant state
updates.

The one case where the originating client needs to react to the server's response is
**rejection** (e.g., access control failure, validation error). On rejection, the client
rolls back its optimistic state. This is the error path, not the normal flow.

**Concurrent multi-client edits**: The server defines the canonical action ordering.
When a client receives broadcast actions from other clients, it applies them through the
normal reducer pipeline. Because both client and server run deterministic reducers,
their states converge as broadcasts are processed.

For the initial implementation, property-level last-writer-wins handles the rare case of
concurrent edits to the same property on the same object. Full OT/CRDT conflict
resolution is not needed for structured data (per Figma, Linear, and Liveblocks
precedent).

### 3.2.2 - Multi-Window Behavior

Each window maintains its own independent Flux store and server streamer. There is no
Drift-based synchronization of document state. When Window A edits a schematic:

1. Window A applies optimistically via its local reducer
2. Window A sends action to server
3. Server broadcasts on signal channel
4. Window B's Flux streamer receives the broadcast
5. Window B applies via its local reducer

The delay between windows is the server round-trip time, which is acceptable for the
non-latency-sensitive nature of meta-data editing.

## 3.3 - Action Definitions

### 3.3.0 - Oracle Schema

Actions are defined in `.oracle` schema files alongside their target type. Oracle
generates all structural scaffolding; developers write only handler implementations.

The exact DSL syntax is an implementation detail, but the information oracle needs per
action is:

1. The **target type** the action operates on (e.g., `Schematic`)
2. The **action name** (e.g., `SetNodePosition`)
3. The **payload fields** (e.g., `key: string`, `position: XY`)

### 3.3.1 - What Oracle Generates

For each type that opts into the action system, oracle generates:

**Go (core/server side)**:

- Action payload structs (e.g., `type SetNodePosition struct { ... }`)
- A discriminated union `Action` type with a `Type` field and optional payload pointers
- A `Reduce(state S, action Action) (S, error)` dispatcher function
- Type-safe constructor functions (e.g., `NewSetNodePositionAction(...)`)
- Stub `Handle(state S) (S, error)` methods on each action struct (developer fills in
  the body)

**TypeScript (client side)**:

- Action payload types with zod schemas for validation
- `createAction()` calls for each action (part of the `relapse` library)
- A combined `createReducer()` that builds the reducer + union schema
- Stub handler functions (developer fills in the body)
- Action constructor functions for type-safe dispatch

**Protobuf (wire format)**:

- An `Action` message with a `type` field and `oneof` payload
- Individual payload messages for each action type

### 3.3.2 - What Developers Write

For each action, the developer writes handler logic in **both Go and TS**:

```go
// Go handler — developer writes the body
func (s SetNodePosition) Handle(state Schematic) (Schematic, error) {
    for i, node := range state.Nodes {
        if node.Key == s.Key {
            state.Nodes[i].Position = s.Position
            break
        }
    }
    return state, nil
}
```

```typescript
// TS handler — developer writes the body
handler: (state: Schematic, { key, position }) => {
  const node = state.nodes.find((n) => n.key === key);
  if (node != null) node.position = position;
};
```

Handler parity between Go and TS is validated through cross-language integration tests
that verify identical output for the same action applied to the same state.

## 3.4 - Server-Side Architecture

### 3.4.0 - Dispatch API Endpoint

A new `Dispatch` endpoint is added alongside existing CRUD endpoints. It accepts a
resource key and an array of actions:

```go
type DispatchRequest struct {
    Key       uuid.UUID `json:"key"`
    SessionID string    `json:"session_id"`
    Actions   []Action  `json:"actions"`
}
```

The server:

1. Retrieves the current state from gorp
2. Applies each action sequentially via the generated `Reduce` function
3. Persists the updated state
4. Broadcasts actions (tagged with `SessionID` for client-side dedup) on the signal
   channel

This endpoint will eventually replace `SetData`/`Update` for types that opt into the
action system. During the transition period, both endpoints coexist.

### 3.4.1 - Signal Channel Integration

Actions are broadcast on signal channels (`sy_schematic_set` / `sy_schematic_delete`)
using `PublishFromObservable` — the same signals infrastructure used by the ontology for
resource and relationship changes. Schematics don't currently have signal channels wired
up (the `Signals` field on `ServiceConfig` is unused), so there is no backward
compatibility concern.

The `Service` creates an internal observable at startup and wires it into
`PublishFromObservable`. When `Writer.Dispatch` persists actions, it notifies this
observable with the scoped action payload. The signal pipeline marshals the payload as
newline-delimited JSON and writes it to `sy_schematic_set`. Each action is a single JSON
object:

```json
{
  "key": "schematic-uuid",
  "session_id": "abc",
  "type": "set_node_position",
  "set_node_position": { "key": "node-uuid", "position": { "x": 100, "y": 200 } }
}
```

Flux channel listeners on the client side parse individual actions from the frame and
apply them through the local reducer, updating the Flux store and triggering React
re-renders.

## 3.5 - Client-Side Architecture

### 3.5.0 - Flux as Document Store

For each type that opts into actions, the Flux store becomes the sole owner of document
state. The store config includes:

- A `ScopedUnaryStore` keyed by resource ID (e.g., schematic UUID)
- A channel listener that receives actions from the server and applies them via the TS
  reducer
- `useRetrieve` for initial data fetch
- `useUpdate` for dispatching actions (optimistic local apply + server send)

### 3.5.1 - Redux for Session State Only

The console Redux slice for a type like schematics is reduced to session-scoped state:

```typescript
interface SchematicSessionState {
  version: string;
  viewport: Diagram.Viewport;
  mode: Viewport.Mode;
  editable: boolean;
  fitViewOnResize: boolean;
  legend: LegendState;
  control: Control.Status;
  authority: number;
  toolbar: ToolbarState;
  selection: string[];
}
```

All document-related actions (`addNode`, `setNodes`, `setEdges`, `setElementProps`,
etc.) are removed from the Redux slice. They flow through Flux instead.

**Selection merge**: React Flow expects `Node[]` with `selected: boolean` on each node.
Flux nodes do not carry selection state — it is per-window, not per-document. The
component reads nodes from Flux and selection state from Redux, merging them at render
time before passing to React Flow.

**Undo/redo**: The current `useUndoableDispatch` wraps Redux dispatches. Moving document
mutations to Flux breaks existing undo in Phase 1. This is acceptable — undo is rebuilt
properly in Phase 3 using the action-based diff mechanism.

**`useSyncComponent` removal**: The current Redux → server sync via `useSyncComponent`
is replaced entirely by the action dispatch pipeline. Document mutations flow through
Flux, which handles server communication.

**Copy/paste**: `CopyBuffer` currently holds nodes, edges, and props from Redux. After
migration, copy reads from Flux (document state) and paste dispatches actions through
the action pipeline.

### 3.5.2 - Component Integration

Pluto's Diagram component (`pluto/src/vis/diagram/`) is a general-purpose React Flow
wrapper with no knowledge of Flux, Redux, or the action system. A separate
schematic-specific integration layer bridges the gap: it receives change events from
Pluto's Diagram and routes document mutations to Flux actions while routing session
state (selections) to Redux dispatches. This layer may live in Pluto's schematic module
(`pluto/src/schematic/`) or in the console — the key constraint is that the generic
Diagram component remains isolated.

```typescript
const handleNodesChange = (nodes: Node[], changes: NodeChange[]) => {
  // Selection changes → Redux (session state)
  const selectionChanges = changes.filter((c) => c.type === "select");
  if (selectionChanges.length > 0)
    reduxDispatch(setSelection({ key, changes: selectionChanges }));

  // Document mutations → Flux action pipeline
  const docActions = changes
    .filter((c) => c.type !== "select")
    .map(nodeChangeToAction)
    .filter(Boolean);
  if (docActions.length > 0) fluxDispatch({ key: schematicKey, actions: docActions });
};
```

The console component reads document state from Flux (`useRetrieve`) and session state
from Redux (selectors), merging them before passing to Pluto's Diagram. This dual-read
pattern already exists for other resources (channels, ranges).

## 3.6 - Undo/Redo Architecture

Undo/redo is a **later phase** but the action system is designed to support it without
architectural changes.

### 3.6.0 - Diff-Based Inverse Operations

Following TLDraw and Liveblocks, the undo model is:

- When the local reducer applies an action, it produces a **diff** capturing the
  before/after state of affected fields (similar to TLDraw's `RecordsDiff` with
  `[from, to]` tuples)
- The diff is pushed onto a **per-user, client-local undo stack**
- Undo reverses the diff and dispatches the result as a normal action through the
  standard pipeline
- The server sees it as a regular action — it has no concept of undo

This works because:

- Inverse operations are just new actions that flow through normal conflict resolution
- Per-user stacks mean users can only undo their own actions
- Client-local stacks are sufficient — undo history doesn't need to survive page reload
  (cross-session recovery is better served by a trash/soft-delete mechanism)

### 3.6.1 - Transaction Grouping

Drag operations, multi-select moves, and other compound interactions must batch into
single undo steps. Following Liveblocks' `pause()`/`resume()` model:

- `beginTransaction()` starts accumulating actions
- `commitTransaction()` squashes accumulated diffs into a single undo entry
- Without explicit transactions, each action is its own undo entry

### 3.6.2 - Design Constraints for Undo Compatibility

The following constraints on the action system ensure undo can be layered on later:

1. **Reducers must be pure functions**: Given the same state and action, always produce
   the same result. No side effects in handlers.
2. **Actions must be the sole mutation pathway**: No direct state manipulation that
   bypasses the action pipeline.
3. **Handlers must not depend on external state**: An action's handler can only read the
   state it's reducing over, not global state or server state.

## 3.7 - Opt-In Per Type

The action system is opt-in. Not every meta-data structure needs it. Simple types
(users, workspaces, ranges) can continue using traditional CRUD. The developer decides
by adding action definitions to the oracle schema for a given type.

Types that benefit most from the action system are those with:

- Rich internal state (schematics, tables, line plots)
- Frequent partial updates (dragging, resizing, property editing)
- Future collaborative editing needs

# 4 - Relationship to Existing Infrastructure

## 4.0 - Oracle

Oracle generates the structural scaffolding (types, action structs, reducer dispatchers,
proto messages, zod schemas). The action system is a new oracle plugin that reads action
definitions from the schema and generates code for each target language.

Oracle's existing plugins continue to generate types, marshalers, and proto translators.
The action plugin layers on top.

## 4.1 - Oracle Migrations

Oracle migrations handle schema evolution (adding fields, changing types). The action
system handles runtime state mutations (user edits). These are orthogonal concerns:

- **Schema migration**: Schematic v1 → v2 (add a `layers` field). Runs at server
  startup.
- **Action dispatch**: User moves a node. Runs in real-time.

The `data json` → typed fields migration must complete before the action system is
implemented. Once fields are typed, actions operate on those typed fields.

## 4.2 - Flux

The Flux system already provides the `ScopedUnaryStore`, channel listeners,
`useRetrieve`, and `useUpdate` patterns needed for the action pipeline. The main new
pieces are:

- Integrating the TS reducer into the Flux store's set operations
- Adding a channel listener on `sy_schematic_set` that parses actions and applies them
  via the local reducer
- Adding `useSelect` patterns for granular subscriptions

## 4.3 - Relapse

The `relapse` library (from the reactive-schematics branch) provides the TS runtime for
`createAction()` and `createReducer()`. This library may be absorbed into oracle's
generated output or maintained as a thin runtime dependency. The Go side is fully
code-generated and doesn't need a runtime library.

## 4.4 - React Flow / Pluto Diagram

The Pluto Diagram component (`pluto/src/vis/diagram/`) is a **general-purpose** React
Flow wrapper. It has no knowledge of Flux, Redux, Synnax, or schematics. It takes
`nodes`, `edges`, and `onChange` callbacks as props and fires `NodeChange[]` /
`EdgeChange[]` events. This boundary must remain clean.

The schematic-specific integration layer sits between Pluto's Diagram and the state
management layer. It receives change events from Diagram and routes them to the correct
destination:

| Change type           | Destination                   | Reason                                |
| --------------------- | ----------------------------- | ------------------------------------- |
| `position` (dragging) | Local Flux store only         | Don't send RPCs during a drag         |
| `position` (drop)     | Flux action → server dispatch | Final position is a document mutation |
| `dimensions`          | Flux action → server dispatch | Document mutation                     |
| `add`                 | Flux action → server dispatch | Document mutation                     |
| `remove`              | Flux action → server dispatch | Document mutation                     |
| `replace`             | Flux action → server dispatch | Document mutation                     |
| `select`              | Redux dispatch                | Session state, per-window             |

This routing logic lives in the schematic-specific integration layer, not in the generic
Diagram component. Pluto's Diagram remains a generic component that fires events without
knowing where they go.

The current code sends the full node array on every change (`setNodes({ nodes })`).
After migration, the integration layer inspects individual changes, maps document
mutations to typed actions dispatched through Flux, and routes selections to Redux.

The drag case connects to open question 6.1: during a drag, the local Flux store updates
for rendering but the server dispatch is deferred until drop. This is a form of implicit
transaction grouping that also reduces network overhead.

# 5 - Implementation Plan

## Phase 0: Infrastructure

1. **Oracle action plugin**: Extend oracle to read action definitions and generate Go
   structs, TS types/zod schemas, proto messages, and reducer dispatchers for each
   target language.
2. **Dispatch API endpoint**: Add the generic `Dispatch` endpoint to the core API layer.
3. **Signal channel wiring**: Wire up signal channels for action-enabled types using
   `PublishFromObservable` with a custom observable (same pattern as ontology signals).

## Phase 1: Schematics

1. **Define schematic actions in oracle**: `SetNodePosition`, `SetNodeProps`, `AddNode`,
   `RemoveNode`, `SetEdge`, `RemoveEdge`, etc.
2. **Write Go handlers**: Implement `Handle()` methods for each action.
3. **Write TS handlers**: Implement handler functions for each action.
4. **Migrate Flux store**: Add schematic Flux store with action-aware channel listener
   and update hooks.
5. **Migrate console schematic**: Move document state out of Redux slice into Flux.
   Reduce Redux slice to session state only. This includes:
   - Separate selection state into a `string[]` of selected node/edge keys in Redux,
     merged with Flux nodes at render time before passing to React Flow
   - Remove `useSyncComponent` (replaced by the action dispatch pipeline)
   - Migrate copy/paste to read document state from Flux
   - Accept that existing `useUndoableDispatch` undo breaks (rebuilt in Phase 3)
6. **Build schematic integration layer**: Wire `onNodesChange`/`onEdgesChange` routing —
   document mutations dispatch through Flux, selection changes dispatch to Redux.
7. **Cross-language parity tests**: Integration tests verifying Go and TS reducers
   produce identical state for the same action sequence.

## Phase 2: Tables

1. **Define table actions in oracle**: `SetCellProps`, `AddRow`, `AddColumn`,
   `DeleteRow`, `DeleteColumn`, `ResizeRow`, `ResizeColumn`, etc.
2. **Write handlers in Go and TS**.
3. **Migrate Flux store and console slice** (same pattern as schematics).
4. **Validate that the oracle plugin and Flux patterns generalize** across different
   data structure shapes.

## Phase 3: Undo/Redo (Future)

1. **Diff capture in TS reducer**: Wrap reducer to capture before/after diffs.
2. **Per-user undo stack**: Client-local stack with configurable depth.
3. **Transaction grouping**: `beginTransaction()`/`commitTransaction()` for compound
   interactions.
4. **Console integration**: Wire Ctrl+Z / Ctrl+Shift+Z to undo/redo dispatch.

## Phase 4: Additional Types (Future)

Migrate line plots, logs, and other types as needed. Each follows the same pattern
established in Phases 1-2.

# 6 - Open Questions

1. **Action batching on the wire**: Should the client batch rapid-fire actions (e.g.,
   during a drag) into fewer network requests, or send each action individually?
   Batching reduces network overhead but adds latency to other clients seeing updates.
2. **Server-side action log**: Should the server persist an action log per resource for
   audit trails and time-travel debugging? Not needed for undo (client-local), but
   valuable for diagnostics. Deferred to a future RFC.
3. **Presence scope**: Real-time cursor/selection sharing for collaborative editing.
   Architecturally straightforward (ephemeral Flux state broadcast on a presence
   channel) but not in scope for initial implementation.
4. **Conflict detection**: Should the server detect and reject conflicting concurrent
   actions, or always apply LWW? For the initial implementation, LWW is sufficient.
   Conflict detection can be added later for specific action types if needed.

# 7 - Concrete Code Sketches

This section shows what the actual code looks like at each layer, grounded in the
existing codebase. These are illustrative sketches — exact signatures may change during
implementation.

## 7.0 - Go: Generated Action Infrastructure (Oracle Output)

Oracle reads action definitions from the `.oracle` schema and generates the following
for each type that opts into actions. This is analogous to the prototype on the
`sy-3304-reactive-schematics` branch, but generated by oracle instead of `relapse/gen`.

### Generated Action Struct and Reducer (`reducer.gen.go`)

```go
// Code generated by oracle; DO NOT EDIT.
package schematic

const (
    ActionSetNodePosition = "set_node_position"
    ActionSetNodeProps    = "set_node_props"
    ActionAddNode         = "add_node"
    ActionRemoveNode      = "remove_node"
    ActionSetEdge         = "set_edge"
    ActionRemoveEdge      = "remove_edge"
)

// Action is a discriminated union for all Schematic mutations.
type Action struct {
    Type string `json:"type" msgpack:"type"`

    SetNodePosition *SetNodePosition `json:"set_node_position,omitempty" msgpack:"set_node_position,omitempty"`
    SetNodeProps    *SetNodeProps    `json:"set_node_props,omitempty" msgpack:"set_node_props,omitempty"`
    AddNode         *AddNode         `json:"add_node,omitempty" msgpack:"add_node,omitempty"`
    RemoveNode      *RemoveNode      `json:"remove_node,omitempty" msgpack:"remove_node,omitempty"`
    SetEdge         *SetEdge         `json:"set_edge,omitempty" msgpack:"set_edge,omitempty"`
    RemoveEdge      *RemoveEdge      `json:"remove_edge,omitempty" msgpack:"remove_edge,omitempty"`
}

// Reduce applies the action to the given state. Returns the new state.
func (a Action) Reduce(state Schematic) (Schematic, error) {
    switch a.Type {
    case ActionSetNodePosition:
        return a.SetNodePosition.Handle(state)
    case ActionSetNodeProps:
        return a.SetNodeProps.Handle(state)
    case ActionAddNode:
        return a.AddNode.Handle(state)
    case ActionRemoveNode:
        return a.RemoveNode.Handle(state)
    case ActionSetEdge:
        return a.SetEdge.Handle(state)
    case ActionRemoveEdge:
        return a.RemoveEdge.Handle(state)
    default:
        return state, nil
    }
}

// ReduceAll applies a sequence of actions to the given state.
func ReduceAll(state Schematic, actions []Action) (Schematic, error) {
    var err error
    for _, a := range actions {
        state, err = a.Reduce(state)
        if err != nil {
            return state, err
        }
    }
    return state, nil
}

// Constructor functions
func NewSetNodePositionAction(p SetNodePosition) Action {
    return Action{Type: ActionSetNodePosition, SetNodePosition: &p}
}
// ... one constructor per action type
```

### Developer-Written Action Handlers (`actions.go`)

Developers hand-write each action's payload struct and `Handle` method. Oracle generates
the struct skeleton; the developer fills in the body.

```go
package schematic

import "github.com/synnaxlabs/x/spatial"

type SetNodePosition struct {
    Key      string     `json:"key" msgpack:"key"`
    Position spatial.XY `json:"position" msgpack:"position"`
}

func (s SetNodePosition) Handle(state Schematic) (Schematic, error) {
    for i, node := range state.Nodes {
        if node.Key == s.Key {
            state.Nodes[i].Position = s.Position
            break
        }
    }
    return state, nil
}

type AddNode struct {
    Node  Node           `json:"node" msgpack:"node"`
    Props map[string]any `json:"props,omitempty" msgpack:"props,omitempty"`
}

func (a AddNode) Handle(state Schematic) (Schematic, error) {
    state.Nodes = append(state.Nodes, a.Node)
    if a.Props != nil {
        if state.Props == nil {
            state.Props = make(map[string]map[string]any)
        }
        state.Props[a.Node.Key] = a.Props
    }
    return state, nil
}

// ... remaining handlers follow the same pattern
```

## 7.1 - Go: Service and API Layer Changes

### Service Writer: New `Dispatch` Method (`writer.go`)

The existing `Writer` gains a `Dispatch` method alongside the existing CRUD methods. The
Writer holds a reference to the action observable (passed via `NewWriter` from the
Service) and is responsible for both persisting the reduced state and notifying the
signal pipeline.

```go
type Writer struct {
    tx             gorp.Tx
    otgWriter      ontology.Writer
    otg            *ontology.Ontology
    table          *gorp.Table[uuid.UUID, Schematic]
    actionObserver observe.Observer[ScopedAction]
}

func (s *Service) NewWriter(tx gorp.Tx) Writer {
    tx = gorp.OverrideTx(s.DB, tx)
    return Writer{
        tx:             tx,
        otgWriter:      s.Ontology.NewWriter(tx),
        otg:            s.Ontology,
        table:          s.table,
        actionObserver: s.actionObserver,
    }
}

func (w Writer) Dispatch(
    ctx context.Context,
    key uuid.UUID,
    sessionID string,
    actions []Action,
) error {
    if err := w.table.NewUpdate().WhereKeys(key).
        ChangeErr(func(_ gorp.Context, s Schematic) (Schematic, error) {
            if s.Snapshot {
                return s, errors.Wrapf(validate.ErrValidation, "cannot dispatch on snapshot")
            }
            return ReduceAll(s, actions)
        }).Exec(ctx, w.tx); err != nil {
        return err
    }
    w.actionObserver.Notify(ctx, ScopedAction{
        Key:       key,
        SessionID: sessionID,
        Actions:   actions,
    })
    return nil
}
```

`gorp.Table.NewUpdate().ChangeErr()` handles read-modify-write atomically within the
transaction. After the persist succeeds, the Writer notifies the action observable,
which triggers the signal pipeline to broadcast the action on `sy_schematic_set`.

### API Layer: `Dispatch` Endpoint (`api/schematic.go`)

```go
type DispatchRequest struct {
    Key       uuid.UUID          `json:"key" msgpack:"key"`
    SessionID string             `json:"session_id" msgpack:"session_id"`
    Actions   []schematic.Action `json:"actions" msgpack:"actions"`
}

func (s *Service) Dispatch(
    ctx context.Context,
    req DispatchRequest,
) (res types.Nil, err error) {
    if err = s.access.Enforce(ctx, access.Request{
        Subject: auth.GetSubject(ctx),
        Action:  access.ActionUpdate,
        Objects: []ontology.ID{schematic.OntologyID(req.Key)},
    }); err != nil {
        return res, err
    }
    return res, s.db.WithTx(ctx, func(tx gorp.Tx) error {
        return s.internal.NewWriter(tx).Dispatch(ctx, req.Key, req.SessionID, req.Actions)
    })
}
```

The Dispatch endpoint follows the exact same pattern as `SetData` and `Rename` — RBAC
enforcement, then transactional writer call.

### Signal Channel Wiring

Schematics don't currently have signal channels. The service wires them up using
`PublishFromObservable` with a custom observable — the same pattern the ontology uses
for resource and relationship changes.

The `Service` holds an observable that `Writer.Dispatch` notifies after persisting.
`ScopedAction` wraps the action payload with the schematic key and originating session
ID for broadcast deduplication:

```go
type ScopedAction struct {
    Key       uuid.UUID `json:"key" msgpack:"key"`
    SessionID string    `json:"session_id" msgpack:"session_id"`
    Actions   []Action  `json:"actions" msgpack:"actions"`
}

type Service struct {
    ServiceConfig
    Symbol         *symbol.Service
    table          *gorp.Table[uuid.UUID, Schematic]
    actionObserver observe.Observer[ScopedAction]
    signals        io.Closer
}
```

At startup, the service wires the observable into the signal pipeline using
`PublishFromObservable` — the same pattern the ontology uses for resource and
relationship changes:

```go
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
    // ... existing setup ...

    s.actionObserver = observe.New[ScopedAction]()

    translated := observe.Translator[ScopedAction, []change.Change[[]byte, struct{}]]{
        Observable: s.actionObserver,
        Translate: func(ctx context.Context, sa ScopedAction) ([]change.Change[[]byte, struct{}], bool) {
            b, err := json.Marshal(sa)
            if err != nil {
                return nil, false
            }
            return []change.Change[[]byte, struct{}]{
                {Variant: change.VariantSet, Key: append(b, '\n')},
            }, true
        },
    }

    s.signals, err = cfg.Signals.PublishFromObservable(ctx, signals.ObservablePublisherConfig{
        Name:          "schematic",
        Observable:    translated,
        SetChannel:    channel.Channel{Name: "sy_schematic_set", DataType: telem.JSONT, Internal: true},
        DeleteChannel: channel.Channel{Name: "sy_schematic_delete", DataType: telem.UUIDT, Internal: true},
    })

    return s, nil
}
```

`Writer.Dispatch` (shown above) calls `w.actionObserver.Notify()` after persisting,
which triggers this signal pipeline to broadcast the scoped action on `sy_schematic_set`
via the framer.

## 7.2 - TypeScript: Relapse Library

The relapse library provides the TS runtime for typed actions with Zod validation and
Immer-based immutable reduction. This is essentially the same as the prototype, with
minor refinements.

### Core relapse API (`x/ts/src/relapse/relapse.ts`)

```typescript
import { produce } from "immer";
import z from "zod";

export interface CreateActionParams<
  State,
  Type extends string,
  Payload extends z.ZodType,
> {
  type: Type;
  payload: Payload;
  handler: (state: State, payload: z.infer<Payload>) => void;
}

export interface ActionDef<State, Type extends string, Payload extends z.ZodType> {
  type: Type;
  payload: Payload;
  handler: (state: State, payload: z.infer<Payload>) => void;
  // Also callable as a constructor: actionDef({ key: "abc", ... })
  (payload: z.infer<Payload>): { type: Type; payload: z.infer<Payload> };
}

export const createAction = <State, Type extends string, Payload extends z.ZodType>(
  params: CreateActionParams<State, Type, Payload>,
): ActionDef<State, Type, Payload> => {
  const action = (payload: z.infer<Payload>) => ({
    type: params.type,
    payload,
  });
  action.type = params.type;
  action.payload = params.payload;
  action.handler = params.handler;
  return action as ActionDef<State, Type, Payload>;
};

export interface ReducerSystem<State> {
  reducer: (state: State, action: { type: string; payload: unknown }) => State;
  actionZ: z.ZodType;
}

export const createReducer = <State>(
  actions: ActionDef<State, string, z.ZodType>[],
): ReducerSystem<State> => {
  const handlers = new Map<
    string,
    { schema: z.ZodType; handler: (state: State, payload: unknown) => void }
  >();

  for (const action of actions)
    handlers.set(action.type, { schema: action.payload, handler: action.handler });

  const reducer = (state: State, action: { type: string; payload: unknown }): State => {
    const h = handlers.get(action.type);
    if (h == null) return state;
    const parsed = h.schema.parse(action.payload);
    return produce(state, (draft) => h.handler(draft as State, parsed));
  };

  const actionZ = z.union(
    actions.map((a) => z.object({ type: z.literal(a.type), payload: a.payload })) as [
      z.ZodType,
      z.ZodType,
      ...z.ZodType[],
    ],
  );

  return { reducer, actionZ };
};
```

### Oracle-Generated Schematic Actions (`client/ts/src/schematic/actions.gen.ts`)

Oracle generates the action definitions with Zod schemas. Developers fill in handlers.

```typescript
// Code generated by oracle; DO NOT EDIT action types.
// Handler implementations are in actions.ts.

import { relapse, xy } from "@synnaxlabs/x";
import z from "zod";
import { type Schematic, nodeZ, edgeZ } from "./payload";

export const setNodePosition = relapse.createAction<Schematic>()({
  type: "set_node_position" as const,
  payload: z.object({ key: z.string(), position: xy.xy }),
  handler: (state, { key, position }) => {
    const node = state.nodes.find((n) => n.key === key);
    if (node != null) node.position = position;
  },
});

export const setNodeProps = relapse.createAction<Schematic>()({
  type: "set_node_props" as const,
  payload: z.object({ key: z.string(), props: z.record(z.unknown()) }),
  handler: (state, { key, props }) => {
    state.props[key] = props;
  },
});

export const addNode = relapse.createAction<Schematic>()({
  type: "add_node" as const,
  payload: z.object({
    node: nodeZ,
    props: z.record(z.unknown()).optional(),
  }),
  handler: (state, { node, props }) => {
    state.nodes.push(node);
    if (props != null) state.props[node.key] = props;
  },
});

export const removeNode = relapse.createAction<Schematic>()({
  type: "remove_node" as const,
  payload: z.object({ key: z.string() }),
  handler: (state, { key }) => {
    const i = state.nodes.findIndex((n) => n.key === key);
    if (i !== -1) state.nodes.splice(i, 1);
  },
});

export const setEdge = relapse.createAction<Schematic>()({
  type: "set_edge" as const,
  payload: edgeZ,
  handler: (state, edge) => {
    const i = state.edges.findIndex((e) => e.key === edge.key);
    if (i !== -1) state.edges[i] = edge;
    else state.edges.push(edge);
  },
});

export const removeEdge = relapse.createAction<Schematic>()({
  type: "remove_edge" as const,
  payload: z.object({ key: z.string() }),
  handler: (state, { key }) => {
    const i = state.edges.findIndex((e) => e.key === key);
    if (i !== -1) state.edges.splice(i, 1);
  },
});

// Generated reducer + union schema
export const { reducer, actionZ } = relapse.createReducer<Schematic>([
  setNodePosition,
  setNodeProps,
  addNode,
  removeNode,
  setEdge,
  removeEdge,
]);

export type Action = z.infer<typeof actionZ>;

// Scoped action = action + schematic key + session ID
export const scopedActionZ = actionZ.and(
  z.object({
    key: z.string(),
    sessionId: z.string().optional(),
  }),
);
export type ScopedAction = z.infer<typeof scopedActionZ>;
```

## 7.3 - TypeScript: Flux Integration

This is where the action system meets Pluto's existing Flux infrastructure. The key
changes are: (1) the Flux store config gets an action-aware channel listener, (2) a new
`useDispatch` hook handles optimistic apply + server send, and (3) self-broadcast dedup
prevents double-application.

### Modified Flux Store Config (`pluto/src/schematic/queries.ts`)

```typescript
import { schematic } from "@synnaxlabs/client";
import { Flux } from "@/flux";
import { reducer, scopedActionZ, type ScopedAction } from "./actions";

export const FLUX_STORE_CONFIG: Flux.UnaryStoreConfig<
  FluxSubStore,
  schematic.Key,
  schematic.Schematic
> = {
  listeners: [
    {
      channel: "sy_schematic_set",
      schema: scopedActionZ,
      onChange: ({ changed: action, store, client }) => {
        // Self-broadcast dedup: skip actions from our own client instance
        if (action.sessionId === client.key) return;

        const current = store.schematics.get(action.key);
        if (current == null) return;
        const next = reducer(current, action);
        store.schematics.set(action.key, next);
      },
    },
  ],
};
```

The channel listener receives frames from `sy_schematic_set`. The Flux streamer splits
frames into newline-delimited JSON objects and parses each one through the
`scopedActionZ` schema (this is the standard Flux `ChannelListener` behavior —
`onChange` is called once per parsed item, not once per frame). The listener checks
`sessionId` to skip self-originated broadcasts, then runs the local reducer for all
other clients' actions.

### Dispatch Hook (`pluto/src/schematic/queries.ts`)

```typescript
export interface DispatchParams {
  key: schematic.Key;
  actions: Action | Action[];
}

export const { useUpdate: useDispatch } = Flux.createUpdate<
  DispatchParams,
  FluxSubStore
>({
  name: "schematic",
  verbs: { present: "update", past: "updated", participle: "updating" },
  update: async ({ client, data, store, rollbacks }) => {
    const { key, actions } = data;
    const actionArray = Array.isArray(actions) ? actions : [actions];

    // 1. Optimistic local apply via reducer
    const current = store.schematics.get(key);
    if (current != null) {
      let next = current;
      for (const action of actionArray) next = reducer(next, action);
      rollbacks.push(store.schematics.set(key, next));
    }

    // 2. Send to server (client attaches its own key as sessionId internally)
    await client.schematics.dispatch(key, actionArray);

    return data;
  },
});
```

This uses the existing `Flux.createUpdate` pattern. The `beforeUpdate` parameter could
also be used for the optimistic apply, but doing it inside `update` with rollbacks is
simpler and already well-established.

The flow is:

1. Get current state from the Flux store
2. Run each action through the TS reducer (optimistic)
3. Push a rollback destructor (reverts to old state on server error)
4. Send actions to server via `client.schematics.dispatch()`
5. If the server rejects, rollbacks fire automatically

### Component Integration

The schematic integration layer routes Pluto Diagram events to the right destination:

```typescript
import { useDispatch } from "./queries";
import { setNodePosition, removeNode, removeEdge, setEdge } from "./actions";

const SchematicInner: FC<SchematicProps> = ({ schematicKey }) => {
  const schematic = useRetrieve({ key: schematicKey });
  const { update: dispatch } = useDispatch();

  const handleNodesChange = useCallback(
    (changes: NodeChange[]) => {
      const actions = changes
        .map((change) => {
          switch (change.type) {
            case "position":
              if (change.position == null) return null;
              return setNodePosition({ key: change.id, position: change.position });
            case "remove":
              return removeNode({ key: change.id });
            default:
              return null;
          }
        })
        .filter((a) => a != null);

      if (actions.length > 0) dispatch({ key: schematicKey, actions });
    },
    [schematicKey, dispatch],
  );

  const handleEdgesChange = useCallback(
    (changes: EdgeChange[]) => {
      const actions = changes
        .map((change) => {
          switch (change.type) {
            case "remove":
              return removeEdge({ key: change.id });
            default:
              return null;
          }
        })
        .filter((a) => a != null);

      if (actions.length > 0) dispatch({ key: schematicKey, actions });
    },
    [schematicKey, dispatch],
  );

  return (
    <Diagram
      nodes={schematic.data?.nodes ?? []}
      edges={schematic.data?.edges ?? []}
      onNodesChange={handleNodesChange}
      onEdgesChange={handleEdgesChange}
    />
  );
};
```

### Client Library Addition (`client/ts/src/schematic/client.ts`)

The client library adds a `dispatch` method following the existing `sendRequired`
pattern used by all other client methods (`rename`, `setData`, `delete`, etc.):

```typescript
const dispatchReqZ = z.object({
  key: keyZ,
  sessionId: z.string(),
  actions: actionZ.array(),
});
const emptyResZ = z.object({});

export class Client {
  // ... existing methods (create, retrieve, rename, setData, delete, copy)

  async dispatch(key: Key, actions: Action[]): Promise<void> {
    await sendRequired(
      this.client,
      "/schematic/dispatch",
      { key, sessionId: this.client.key, actions },
      dispatchReqZ,
      emptyResZ,
    );
  }
}
```

## 7.4 - End-to-End Worked Example

A concrete scenario: User A drags node "valve-3" to position (100, 200) on a schematic.
User B has the same schematic open in a different browser. Here is every payload at
every boundary.

### Step 1: React Flow Event (User A's browser)

React Flow fires an `onNodesChange` callback:

```typescript
// React Flow produces this change event
[{ type: "position", id: "valve-3", position: { x: 100, y: 200 } }];
```

### Step 2: Action Construction (User A's browser)

`handleNodesChange` maps the React Flow event to a typed action:

```typescript
// setNodePosition constructor produces this action
{
  type: "set_node_position",
  payload: { key: "valve-3", position: { x: 100, y: 200 } }
}
```

### Step 3: Optimistic Local Apply (User A's browser, instant)

The Flux `useDispatch` hook runs the TS reducer against the local store:

```typescript
// Before: store.schematics.get("d4a2e1f0-...")
{
  nodes: [
    { key: "valve-3", position: { x: 50, y: 80 }, ... },
    { key: "pump-1",  position: { x: 300, y: 100 }, ... }
  ],
  edges: [...],
  props: { "valve-3": { label: "Main Valve" }, ... }
}

// reducer applies setNodePosition
// After: store.schematics.set("d4a2e1f0-...", newState)
{
  nodes: [
    { key: "valve-3", position: { x: 100, y: 200 }, ... },  // ← changed
    { key: "pump-1",  position: { x: 300, y: 100 }, ... }
  ],
  edges: [...],
  props: { "valve-3": { label: "Main Valve" }, ... }
}

// Rollback destructor saved in case server rejects
```

React re-renders immediately. User A sees valve-3 at (100, 200).

### Step 4: Server RPC (User A → Synnax Server)

```
POST /api/schematic/dispatch
Content-Type: application/msgpack
```

```json
{
  "key": "d4a2e1f0-...",
  "session_id": "a7b3c9d1-...",
  "actions": [
    {
      "type": "set_node_position",
      "set_node_position": {
        "key": "valve-3",
        "position": { "x": 100, "y": 200 }
      }
    }
  ]
}
```

### Step 5: Server Processing

```
API Layer
  │
  ├── RBAC: access.Enforce(ctx, ActionUpdate, SchematicOntologyID("d4a2e1f0-..."))
  │
  └── WithTx:
        │
        Writer.Dispatch(ctx, "d4a2e1f0-...", "a7b3c9d1-...", actions)
          │
          ├── gorp.ChangeErr: reads Schematic from Pebble
          │     │
          │     └── ReduceAll(state, actions)
          │           Go SetNodePosition.Handle() finds node "valve-3",
          │           sets position to {100, 200}, returns new state
          │
          ├── gorp persists updated Schematic to Pebble
          │
          └── actionObserver.Notify(ctx, ScopedAction{
                Key:       "d4a2e1f0-...",
                SessionID: "a7b3c9d1-...",
                Actions:   [{Type: "set_node_position", ...}],
              })
```

### Step 6: Signal Broadcast

The `observe.Translator` marshals the `ScopedAction` as JSON, appends `\n`, and emits it
as a `change.VariantSet`. `PublishFromObservable` writes it to `sy_schematic_set` via
the framer.

Bytes written to `sy_schematic_set`:

```
{"key":"d4a2e1f0-...","session_id":"a7b3c9d1-...","actions":[{"type":"set_node_position","set_node_position":{"key":"valve-3","position":{"x":100,"y":200}}}]}\n
```

### Step 7: Client Reception

Both User A and User B have Flux channel listeners subscribed to `sy_schematic_set`. The
Flux streamer splits the frame on `\n`, parses each line through `scopedActionZ`.

**User A's listener**:

```typescript
// parsed ScopedAction
{ key: "d4a2e1f0-...", sessionId: "a7b3c9d1-...", actions: [...] }

// SESSION_ID === "a7b3c9d1-..." → match → DROP
// (already applied optimistically in Step 3)
```

**User B's listener**:

```typescript
// parsed ScopedAction
{ key: "d4a2e1f0-...", sessionId: "a7b3c9d1-...", actions: [...] }

// SESSION_ID === "f2e8b4a0-..." → no match → APPLY
// reducer(currentState, action) → store.schematics.set(key, newState)
// React re-renders, User B sees valve-3 at (100, 200)
```

### Step 8 (Error Path): Server Rejection

If the server rejects (e.g., user lacks write access), the RPC returns an error.
`Flux.createUpdate` fires the rollback destructor saved in Step 3, reverting User A's
local state to the pre-drag position. User A sees valve-3 snap back to (50, 80).
