# 35 - Unified Execution Shell for the Arc IR

**Feature Name**: Unified Execution Shell <br /> **Status**: Draft <br /> **Start
Date**: 2026-04-15 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

RFC 0034 (Composable Sequential and Parallel Execution) introduced a tree-shaped Arc IR
with `Stage`, `Sequence`, `Step`, and `Flow` as distinct execution-layer types, running
alongside a separate `Strata` field for execution ordering. In practice this produced
four specialized container types plus a parallel ordering field, with cross-cutting
conventions (entry nodes, boundary markers) that string- link them at runtime.

This RFC collapses Layer 2 (the execution shell) into a single primitive — `Scope` —
parameterized by concurrency mode and liveness. `Stage`, `Sequence`, `Flow`, `Step`,
`Strata`, synthesized entry nodes, and synthetic boundary markers all go away.
Conditional edges remain in Layer 1 as a dataflow primitive; they no longer drive state
transitions.

The dataflow layer (`Function`, `Node`, `Edge`) is unchanged. Arc surface syntax is
unchanged. This is strictly an internal representation and runtime change.

# 1 - Vocabulary

- **Layer 1 (Dataflow)** — the graph of `Function`, `Node`, and `Edge`. Static, flat,
  consumed by analyzer and compiler. Unchanged by this RFC.
- **Layer 2 (Execution Shell)** — the hierarchy that governs when and in what order
  subsets of the dataflow graph execute. Redesigned by this RFC.
- **Scope** — the single Layer 2 primitive. Carries mode, liveness, and mode-specific
  contents (phases or members+transitions).
- **Mode** — `PARALLEL` or `SEQUENTIAL`. Whether a Scope's members run together
  (respecting data deps) or one at a time (with transition rules).
- **Liveness** — `ALWAYS` or `GATED`. Whether the Scope is continuously active
  (root-like) or becomes active only when its activation trigger fires.
- **Phase** — one execution layer within a `PARALLEL` Scope. Members in a phase have no
  data dependency among themselves. Phase N depends only on phases 0..N-1.
- **Member** — a child of a Scope. Either a `NodeRef` (into Layer 1) or a nested
  `Scope`.
- **NodeRef** — a handle from Layer 2 pointing at a node in Layer 1 by key.
- **Transition** — a declarative rule on a `SEQUENTIAL` Scope: when some dataflow output
  becomes truthy after the active member runs, switch which member is active.
- **Activation** — the event of a `GATED` Scope becoming active. Cascades reset through
  all descendant scopes and their node members.

# 2 - Motivation

## 2.0 - Tensions in the Current IR

The current Arc IR, as shipped in SY-4043, has two parallel hierarchies describing
execution:

- A **composition tree**: `IR.root: Stage` → `Stages.Sequences` → `Sequence.Steps` →
  `Step{Flow|Stage|Sequence}`. Source-grammar-aligned.
- A **stratification field**: `Stage.strata` and `Sequence.strata` encode execution
  order among members of each context.

These are declared as separate concerns (structure vs ordering) but are entangled at
every integration point:

1. `Strata` entries hold magic-string markers (`entry_<seq>_<step>`, `boundary_<step>`)
   that reference the composition tree's structure.
2. The scheduler dispatches on those markers via runtime string probes: "is this key a
   real node? a boundary reference? skip silently?" (see
   `arc/go/runtime/scheduler/scheduler.go:286–306`).
3. Three separate places format and parse the `boundary_<step>` convention:
   `arc/go/stratifier/stratifier.go:30`, scheduler `registerBoundaries`
   (`scheduler.go:187`), C++ `register_boundaries`
   (`arc/cpp/runtime/scheduler/scheduler.h`). Drift between them is silent activation
   breakage.
4. The `boundary_<step>` key format is not sequence-qualified; two top-level sequences
   with same-keyed child sub-sequences collide in the scheduler's global `boundaries`
   map (`scheduler.go:61`). This is a latent bug.

## 2.1 - Entry Nodes Are Two Mechanisms Fighting Each Other

The analyzer (`arc/go/text/analyze.go:877`, `:919`, `:1008`) synthesizes `stage_entry`
nodes (`arc/go/stl/stage/stage.go:25`) and places them in the dataflow graph. These real
nodes exist solely to be firing points for conditional edges from transition predicates.
The scheduler's `transitionStep` (`scheduler.go:386–403`) special-cases them.

At the same time, _boundary markers_ exist for a different job: keeping a sub-sequence
executing on every parent cycle while the sub-sequence is active. Entry nodes can't do
this job because they're reactive (fire when marked changed) rather than passive (fire
on every strata walk).

The result: two mechanisms, one reactive and one passive, cooperating to produce
state-machine behavior. Commit `2cda80b5` (April 2026) fixed a bug in their interaction
where entry nodes placed in the wrong stratum caused step re-activation on every tick.
The fix works but leaves the underlying architecture intact.

## 2.2 - Four Types for What Is One Abstraction

From the scheduler's perspective:

| Today's type | What it actually is                                                            |
| ------------ | ------------------------------------------------------------------------------ |
| `Stage`      | Parallel-ordered set of nodes + nested sequences, maintains state while active |
| `Flow`       | Parallel-ordered set of nodes, maintains state while active                    |
| `Sequence`   | Sequential list of steps, one active at a time, with transitions               |
| `Step`       | Tagged union wrapper for the three above                                       |

`Stage` and `Flow` differ only in whether they can contain sub-sequences. The node-state
lifecycle is uniform between them (verified in `scheduler.go:405–448`): both call
`Reset()` on activation and clear `selfChanged` on deactivation. The stage-vs-flow
distinction is a _structural_ constraint, not a _lifecycle_ one.

`Step` exists purely as a discriminated-union wrapper around the other three because Go
lacks sum types. It carries no independent semantics.

## 2.3 - What We Want From a Redesign

1. One execution primitive instead of four.
2. `Strata` collapses into the primitive's phasing, not a parallel field.
3. Entry nodes disappear — conditional state-transition logic becomes declarative.
4. Boundary markers disappear — descent into child scopes is recursion, not a
   string-convention cross-reference.
5. Dataflow layer is untouched. Arc source syntax is untouched.

# 3 - Design

## 3.0 - Layer Separation

Two explicit layers:

- **Layer 1 — Dataflow.** `Function`, `Node`, `Edge`. Flat graph of compute. Unchanged
  from today.
- **Layer 2 — Execution Shell.** A tree rooted in a single `Scope`. Replaces the current
  `Stage` / `Sequence` / `Flow` / `Step` hierarchy and the `Strata` / `Stratum` field.

The only coupling between layers is reference: Layer 2 refers to Layer 1 nodes by key
(`NodeRef`) and to Layer 1 node outputs by `Handle` (already used by edges).

## 3.1 - The Scope Primitive

```
Scope struct {
    key          string
    mode         ScopeMode        // PARALLEL | SEQUENTIAL
    liveness     Liveness         // ALWAYS | GATED
    activation   Handle?          // GATED only; absent for ALWAYS scopes
    phases       Phase[]          // PARALLEL only; empty for SEQUENTIAL
    members      Member[]         // SEQUENTIAL only; empty for PARALLEL
    transitions  Transition[]     // SEQUENTIAL only; empty for PARALLEL
}

Phase struct {
    members      Member[]         // members in this execution layer
}

Member struct {
    key          string           // position identifier within parent scope
    nodeRef      NodeRef?         // set when this member is a dataflow reference
    scope        Scope?           // set when this member is a nested scope
}
// Invariant: exactly one of Member.nodeRef or Member.scope is non-nil.
// Builder/analyzer enforces; not expressible in the Oracle schema because
// Go lacks sum types.

NodeRef struct {
    key          string           // key into IR.Nodes (Layer 1)
}

Transition struct {
    on           Handle            // a Layer 1 node output
    target       TransitionTarget
}

TransitionTarget struct {
    memberKey    string?           // target: sibling member in this scope
    exit         bool?             // target: exit scope, yield to parent
}
// Invariant: exactly one of TransitionTarget.memberKey or .exit is set.
```

`ScopeMode` and `Liveness` are enums with `unspecified = 0` sentinels in keeping with
the existing `EdgeKind` convention in `schemas/arc/ir.oracle:29`.

## 3.2 - Valid Configurations

Three `(mode, liveness)` combinations are meaningful:

| mode         | liveness | Role                                    |
| ------------ | -------- | --------------------------------------- |
| `PARALLEL`   | `ALWAYS` | Program root; always-live reactive flow |
| `PARALLEL`   | `GATED`  | What a `stage {}` compiles to           |
| `SEQUENTIAL` | `GATED`  | What a `sequence {}` compiles to        |

`SEQUENTIAL + ALWAYS` is theoretically expressible (a top-level state machine that boots
at member 0) but has no source-level construct that produces it. The analyzer never
emits this combination; the scheduler need not handle it specially.

`PARALLEL + ALWAYS` with state reset has no activation event to reset on — the
combination is degenerate. The analyzer should never emit a Parallel+Always scope with
the "reset on activation" intent; the schema does not encode a reset flag because state
lifecycle is uniform across all GATED scopes (see §5.2).

## 3.3 - Lifecycle Invariants

Drawn directly from `arc/go/runtime/scheduler/scheduler.go:405–440` and preserved in the
new design:

1. **Activation is cascading.** Activating a `GATED` Scope calls `Reset()` on every
   direct member node, clears `selfChanged` for those nodes, and recursively activates
   any GATED nested Scopes (each at its initial state — member 0 for SEQUENTIAL, all
   members present for PARALLEL).
2. **Deactivation is non-cascading.** Deactivating a Scope clears `selfChanged` for its
   member nodes and marks itself inactive. It does not recurse into nested Scopes; their
   `activeMember` state is frozen. On the next activation of the parent, descendants are
   activated fresh, which overwrites the stale state.
3. **Parallel phases run every cycle while active.** All phases execute in order; phase
   N's members run after phase N-1's completes within the cycle.
4. **Sequential transitions evaluate after the active member runs.** If multiple
   transitions become truthy in the same cycle, first in source order wins (preserves
   current behavior documented in `scheduler_test.go:988`).
5. **ALWAYS scopes have no activation event.** Their members are reset once at program
   start (at scheduler construction) and then run every cycle.

## 3.4 - The Root

`IR.root: Scope` replaces today's `IR.root: Stage`. The root is
`Scope{mode: PARALLEL, liveness: ALWAYS}`. Its `phases` contain a mix of `NodeRef`
members (module-scope reactive flow: channel reads, timers, bare flow statements) and
nested `Scope` members (top-level sequences and stages).

Because `Member` is a tagged union of `NodeRef | Scope`, a parallel scope can contain
both kinds of child in any phase. No special-casing of the root is needed.

## 3.5 - What Goes Away

From `schemas/arc/ir.oracle`, the following types are removed:

- `Flow`, `Stage`, `Stages`, `Step`, `Steps`, `Sequence`, `Sequences`
- `Stratum`, `Strata`

From `arc/go/stl/stage/stage.go`, the entire `stage_entry` STL node definition is
removed (`EntryNodeName`, `EntryActivationParam`, `EntryNodeInputs`, the node's `Next`
implementation).

From `arc/go/stratifier/stratifier.go`, boundary and entry-node placement logic is
removed. The stratifier reduces to "given a Parallel Scope's members and the dataflow
edges among them, compute Phase[] by dep-ordering." No synthetic marker injection.

From `arc/go/runtime/scheduler/scheduler.go` and the C++ equivalent, the `boundaries`
map, the `transitions` map, and entry-node-specific dispatch go away. The
`registerBoundaries` and `registerTransitions` setup passes are replaced by a single
recursive Scope-tree walk that builds `scopeState` entries.

## 3.6 - What Stays

Layer 1 is unchanged: `Function`, `Node`, `Edge`, `Handle`, `Authorities`, and their
collections keep their current shape and semantics.

`EdgeKind` (`continuous` | `conditional`) remains. Conditional edges continue to gate
dataflow propagation — "only push the `changed` signal downstream when the source output
is truthy." This is a Layer 1 concern, independent of Layer 2's state-machine semantics.

# 4 - Lowering Arc Source to the New IR

This section specifies how every Arc surface construct lowers to the Layer 2 primitives.
Source syntax does not change; only the IR produced by `arc/go/text/analyze.go` does.

## 4.0 - Top-Level Sequence

```arc
sequence main {
    stage press { ... }
    stage done { ... }
}
```

Lowers to:

```
Scope{
    key:      "main"
    mode:     SEQUENTIAL
    liveness: GATED
    activation: nil      // set only if an outer `=> main` exists
    members: [
        Member{ key: "press", scope: Scope{...} },
        Member{ key: "done",  scope: Scope{...} },
    ]
    transitions: [/* auto-wired from the last node of each step, see 4.3 */]
}
```

The Scope is added as a `NodeRef`-or-`Scope` member of `IR.root`'s phases.

## 4.1 - Top-Level Stage Declaration

```arc
stage abort {
    0 -> ox_cmd,
    1 -> vent_cmd,
}
```

Today (`arc/go/text/analyze.go:964–987`) wraps this in a single-step sequence. Under
this RFC, it is no longer wrapped — the top-level stage is directly a
`Scope{PARALLEL, GATED}` member of `IR.root`:

```
Scope{ key: "abort", mode: PARALLEL, liveness: GATED, phases: [...] }
```

Activation is set when a transition elsewhere targets `abort` (see §4.6).

## 4.2 - Stageless Sequences and Flow-Step Sequences

```arc
sequence main {
    1 -> valve_cmd
    wait{duration=2s}
    0 -> valve_cmd
}
```

Today produces `Sequence.Steps[]` with `Flow` variants. Under this RFC:

```
Scope{
    key:         "main"
    mode:        SEQUENTIAL
    liveness:    GATED
    members: [
        Member{ key: "step_0", scope: Scope{
            mode: PARALLEL, liveness: GATED,
            phases: [ Phase{ members: [ NodeRef{write_valve_1} ] } ]
        }},
        Member{ key: "step_1", scope: Scope{
            mode: PARALLEL, liveness: GATED,
            phases: [ Phase{ members: [ NodeRef{wait_2s} ] } ]
        }},
        Member{ key: "step_2", scope: Scope{
            mode: PARALLEL, liveness: GATED,
            phases: [ Phase{ members: [ NodeRef{write_valve_0} ] } ]
        }},
    ]
    transitions: [ /* auto-wired from last node of each step */ ]
}
```

Each flow step becomes a nested `PARALLEL + GATED` Scope. Flows that today contain a
single node are simply Scopes with one `NodeRef` member in one phase. Multi-node flow
chains (`a -> b -> c`) become Scopes whose phasing reflects the data dependencies.

Note that this _increases_ nesting depth compared to the current IR, which flattens
flow-step nodes into the parent sequence's `Strata`. The trade-off is explicit: uniform
lowering at the cost of one extra Scope per flow step. Runtime cost is trivial (one
pointer hop per descent); IR memory cost is bounded by program size.

## 4.3 - Auto-Wired Sequence Transitions

Today, the analyzer auto-wires conditional edges from the last node of each step to the
next step's entry node (`arc/go/text/analyze.go:887–899`, `:929–936`).

Under this RFC, auto-wiring emits a `Transition` on the parent SEQUENTIAL Scope instead:

```
Transition{
    on:     Handle{ node: lastNodeOfStepN, param: firstOutputParam },
    target: TransitionTarget{ memberKey: step_{N+1}.key }
}
```

For the final step in a sequence, the auto-wired transition has `target.exit = true`
instead of a member key.

## 4.4 - Explicit Transitions (`=> next`, `=> named`)

```arc
stage press {
    1 -> press_cmd,
    pressure > 50 => next,
}
```

Today the `=> next` produces a conditional edge from the comparison node's output to the
next step's entry node.

Under this RFC, `=> next` emits a `Transition` on the enclosing SEQUENTIAL Scope (the
`main` sequence in this example) rather than on the stage itself:

```
Transition{
    on:     Handle{ node: comparison_node, param: out },
    target: TransitionTarget{ memberKey: next_sibling.key }
}
```

`=> named_target` produces the same shape with `memberKey: "named_target"` targeting a
named sibling in the parent sequence.

## 4.5 - Inline Sub-Sequences Within Stages

```arc
stage fire {
    sequence {
        1 -> ox_cmd
        wait{duration=500ms}
        1 -> fuel_cmd
    },
    interval{period=100ms} -> control,
    ox_pt < 15 => next,
}
```

The inline sub-sequence becomes a `Scope{SEQUENTIAL, GATED}` member of the stage's
PARALLEL scope. No boundary marker is needed: the sub-sequence is simply a child Scope,
and the scheduler's recursive walk descends into it when the stage is active.

The sub-sequence's `activation` is nil; it is implicitly activated when its parent stage
activates (cascading activation, §3.3). On stage re-activation, the sub-sequence resets
to member 0.

## 4.6 - Cross-Scope Activation (`=> abort`)

```arc
sequence main {
    stage press {
        abort_btn => abort,
    }
    ...
}
stage abort { ... }
```

`abort_btn => abort` references a named top-level scope. This does not fit within-scope
`Transition` semantics (the transition would target something not a member of the
current SEQUENTIAL scope).

The decomposition: the compiler emits **two** IR entities from one source statement:

1. On the `main` sequence (or whichever SEQUENTIAL parent contains the `=>` clause): a
   `Transition{ on: abort_btn, target: { exit: true } }` — exit whatever sequence we're
   currently in.
2. On the `abort` Scope: `activation: Handle{ abort_btn, out }` — the abort scope
   activates when `abort_btn` fires truthy.

Both reference the same `Handle`. On the tick when `abort_btn` is truthy: `main` exits,
`abort` activates, their orderings are independent (neither depends on the other).

This keeps `TransitionTarget` narrow (just `memberKey | exit`). Cross-scope travel is
decomposed into exit + activate, which are already primitives.

## 4.7 - Module-Scope Reactive Flow

```arc
start_cmd => main
sensor -> filter -> clean_sensor
```

These lower to members of `IR.root`'s phases:

- `start_cmd => main` sets `main`'s `activation: Handle{start_cmd, out}`. The
  conditional edge from `start_cmd` that today terminates at `main`'s entry node no
  longer exists; activation is declarative.
- `sensor -> filter -> clean_sensor` lowers to three `NodeRef` members in `IR.root`'s
  phases, ordered by the dataflow dependencies among them (standard stratification).

## 4.8 - Channel Writes, Reads, and `set_authority`

These are Layer 1 nodes. Their representation is unchanged. Their membership in Scopes
follows the same source-structure rules as any other node.

## 4.9 - Expressions, Routing Tables, Wait Nodes

Expressions compile to nodes (unchanged). Routing tables compile to `{...}` syntax nodes
(unchanged). Wait nodes are regular nodes whose outputs drive either auto-wired
transitions (stepless sequences) or explicit transitions (`=>`). Their `MarkSelfChanged`
behavior is scheduler-internal and unchanged.

# 5 - Scheduler Semantics

## 5.0 - Runtime State

Replace the current per-sequence state (`sequenceState`, `stepState`, `boundaries`,
`transitions` maps — `scheduler.go:42–65`) with a single recursive structure:

```
scopeState {
    ir            *Scope
    active        bool
    activeMember  int             // SEQUENTIAL only; -1 if inactive
    children      []scopeState    // one per nested Scope member
}
```

The scheduler maintains a tree of `scopeState` mirroring the `Scope` tree, built once at
construction. Node registry (`nodes map[key]Node`) is unchanged.

## 5.1 - Cycle Structure

```
Next():
    clear changed, transitioned
    walk(rootScope)

walk(s):
    if s.mode == PARALLEL:
        for each phase in s.ir.phases:
            for each member m in phase:
                execute(m, s)
        // PARALLEL scopes have no transitions to evaluate
    else:  // SEQUENTIAL
        if s.activeMember < 0: return
        m := s.ir.members[s.activeMember]
        execute(m, s)
        // evaluate transitions in source order
        for each t in s.ir.transitions:
            if isTruthy(t.on):
                deactivate(s.activeMember)
                if t.target.exit:
                    s.active = false
                    s.activeMember = -1
                else:
                    activate(s, memberIndex(t.target.memberKey))
                break  // first-match wins
    // check gated-child activations
    for each child scope c in s's members:
        if c is GATED and not c.active and isTruthy(c.activation):
            activate(c)

execute(m, parent):
    if m is NodeRef:
        run m.key's node
    else:  // nested Scope
        walk(m.scope)
```

This replaces `execStrata`, `execSequences`, `execSequenceStep`, `transitionStep`, and
the boundaries dispatch branch (`scheduler.go:256–448`). The new walk is ~60 lines; the
replaced code is ~190 lines.

## 5.2 - Activation and Reset

```
activate(scope):
    scope.active = true
    if scope.mode == SEQUENTIAL:
        scope.activeMember = 0
    for each member m in scope's members:
        if m is NodeRef: m.node.Reset(); clear m from selfChanged
        if m is nested GATED Scope: activate(recursively)
        // nested ALWAYS scopes do not occur inside GATED scopes
        //   by analyzer invariant (ALWAYS is only valid for root)

deactivate(scope):
    if scope.mode == SEQUENTIAL:
        scope.activeMember = -1
    for each member m in scope's members:
        if m is NodeRef: clear m from selfChanged
        // do NOT recurse into child scopes
        // do NOT call Reset()
    scope.active = false
```

This preserves the two key asymmetries from today: activation cascades and resets;
deactivation does neither (see §3.3). The behavior documented in `scheduler.go:405–440`
is preserved byte-for-byte in semantic terms.

## 5.3 - Convergence

Today's `execSequences` (`scheduler.go:339–357`) runs a convergence loop with a bound of
`countSteps(seq)` iterations to absorb cascading transitions in a single tick.

The new walk replaces this. A transition fires inside the recursive walk, and the walk's
structure — execute active member, evaluate transitions, take at most one — produces one
transition per SEQUENTIAL scope per cycle. If a new member's execution triggers further
transitions, they fire on the next cycle.

This changes an observable behavior: today, write-cascading within a single sequence (a
chain of writes that advance instantly through steps) all happens on one tick via the
convergence loop. Under the new model, each write advances by one step per cycle.

**This change requires discussion.** Options:

- Preserve convergence semantics by wrapping the SEQUENTIAL walk in a bounded loop (as
  today). This is a straightforward addition to the algorithm in §5.1.
- Accept the one-step-per-cycle model. This simplifies the scheduler significantly and
  removes `maxConvergenceIter` bookkeeping, but it is a behavioral change visible to Arc
  programs that rely on same-cycle cascading.

The RFC recommends **preserving convergence semantics** to avoid a surface behavior
change. The bound is `depth of SEQUENTIAL nesting × members per level`, summed
recursively, mirroring `countSteps` today.

## 5.4 - Self-Changed Nodes and Waits

`selfChanged` (`scheduler.go:246–248`) remains a scheduler-level set keyed by node key.
The invariants from today are preserved:

- A node that calls `MarkSelfChanged()` during its `Next()` is re-added to the `changed`
  set on the next cycle so it re-executes.
- When a scope containing that node deactivates, its members are removed from
  `selfChanged`.
- Activation does not explicitly add to `selfChanged`; it only clears.

This logic is unchanged by the redesign. It lives where it does today.

## 5.5 - Authority Flush Ordering

Authority mutations vs. channel writes within a cycle are ordered by phase dependencies
— `set_authority` nodes are placed in earlier phases than channel writes that depend on
them, exactly as today. This is a function of the stratifier's dependency-ordering logic
and is unaffected by the Scope redesign. See
`docs/tech/rfc/0031-260311-arc-scheduler-semantics.md` for the underlying invariants.

# 6 - Migration Plan

This is a large, coordinated change across ~9500 lines of code (counted: scheduler 469
Go + 402 C++, stratifier 502, analyzer 1088, plus 2620 + 1618 + 2856 lines of tests). It
is not piecewise-mergeable — the schema change invalidates every consumer
simultaneously.

## 6.0 - Branching Strategy

A long-lived feature branch (`sy-XXXX-unified-execution-shell`) off `rc`. All work lands
on the branch in many small commits; merge to `rc` only after the full suite is green.

No incremental-migration strategy (e.g., parallel-types, feature flags) is proposed. The
cost of maintaining two IR shapes during migration exceeds the cost of doing the full
migration in one branch.

## 6.1 - Phase 1: Schema and Regeneration

**Scope:** `schemas/arc/ir.oracle`.

**Changes:**

- Remove types `Flow`, `Stage`, `Stages`, `Step`, `Steps`, `Sequence`, `Sequences`,
  `Stratum`, `Strata`.
- Add enums `ScopeMode`, `Liveness`.
- Add structs `Scope`, `Phase`, `Member`, `NodeRef`, `Transition`, `TransitionTarget`.
- Change `IR.root` from `Stage` to `Scope`.

**Action:** Author edits the schema. User runs `oracle sync` per project convention
(oracle is not run by the assistant — see user memory). Regeneration produces new Go,
C++, and TypeScript bindings.

**Expected state after Phase 1:** every Go package that references the removed types
fails to compile. This is deliberate; the migration continues in Phase 2.

**Protoc:** the `.pb.go` files (generated by `protoc` from `.proto`) must be regenerated
separately from `oracle sync`. Per prior behavior observed in this branch (see SY-4043
notes), bazel rebuilds will force proto regen; a clean `bazel build //arc/...` after
oracle sync is necessary.

## 6.2 - Phase 2: Analyzer Rewrite

**Scope:** `arc/go/text/analyze.go` (1088 lines, most of which stays but the
Step/Stage/Sequence construction paths rewrite).

**Changes:**

- `analyzeSequence` (starts ~line 794) emits a `Scope{SEQUENTIAL, GATED}` instead of an
  `ir.Sequence`.
- `analyzeStage` (referenced from ~line 845) emits a `Scope{PARALLEL, GATED}`.
- `analyzeFlow` (~line 597) produces nodes as today, but the calling code wraps them in
  a `Scope{PARALLEL, GATED}` with a single `Phase` containing `NodeRef`s, instead of
  emitting a `Step{Flow: &Flow{Nodes: ...}}`.
- Entry node synthesis (lines 877, 919, 1008) is deleted entirely. The
  `stage.EntryNodeName` constant and the `stage_entry` STL type become unused and are
  removed (Phase 7).
- Auto-wiring of conditional edges between step entries (lines 887–899, 929–936) is
  replaced by emission of `Transition` values on the parent SEQUENTIAL Scope.
- Cross-scope `=> target` (`abort_btn => abort`) is decomposed into: (a) a `Transition`
  with `target.exit = true` on the current sequence, and (b) an
  `activation: Handle{...}` on the target Scope. The compiler must track scope
  identifiers across the module to wire both sides.
- Module-scope flow (`a -> b -> c`, channel reads, timers, `=> main`) populates
  `IR.root`'s phases directly.

**Tests:** `arc/go/text/text_test.go` is the main test surface; expect ~5–15 assertion
sites to change shape, not behavior.

## 6.3 - Phase 3: Stratifier Simplification

**Scope:** `arc/go/stratifier/stratifier.go` (502 lines → estimated ~200 after
simplification).

**Changes:**

- Remove `BoundaryKey` function and all uses.
- Remove `entryKey` function and all uses.
- Remove `flattenEntryNodes` (specific to entry-node ordering).
- Remove `stratifySequenceWithFlowSteps` (no longer needed; flow steps are Scopes).
- The core recursion becomes: for each `Scope` with `mode = PARALLEL`, compute `Phase[]`
  from the dataflow edges among its members, restricted to edges sourced from and
  targeting keys in this scope's membership. For `SEQUENTIAL` scopes, no phasing to
  compute; recurse into child scopes.
- Members that are nested `Scope` values are treated as atomic for phase-ordering
  purposes (their internal phasing is their own concern).

**Tests:** `arc/go/stratifier/stratifier_test.go` (1618 lines). Most tests will rewrite
to construct `Scope` instances instead of `Sequence` + `Step` + `Stage`. Roughly ~40
test sites.

## 6.4 - Phase 4: Go Scheduler Rewrite

**Scope:** `arc/go/runtime/scheduler/scheduler.go` (469 lines → estimated ~250).

**Changes:**

- Replace `sequenceState`, `stepState`, `boundaries`, `transitions` maps with a
  recursive `scopeState` tree.
- Rewrite `Next()` per §5.1.
- Rewrite `activateStep` / `deactivateStep` per §5.2.
- Remove `registerBoundaries`, `registerTransitions`.
- Preserve convergence loop semantics per §5.3.
- Preserve `selfChanged` semantics unchanged.

**Tests:** `arc/go/runtime/scheduler/scheduler_test.go` (2620 lines). All 75+ tests will
have their IR construction rewritten to the `Scope` shape. Test _assertions_ about
execution counts, transition behavior, and self-changed behavior should remain
identical; only the construction boilerplate changes.

This is the most mechanical and highest-volume change in the migration.

## 6.5 - Phase 5: C++ Scheduler Rewrite

**Scope:** `arc/cpp/runtime/scheduler/scheduler.h` (402 lines → est. ~230).

Mirror the Go changes. C++ uses `x::mem::indirect<T>` for recursive types (already in
use for the current `Step`); this pattern extends naturally to nested `Scope` references
within `Member`.

**Tests:** `arc/cpp/runtime/scheduler/scheduler_test.cpp` (2856 lines). Same shape of
migration as Go tests; expected ~80+ test sites.

## 6.6 - Phase 6: Test Utilities

**Scope:** `arc/go/ir/testutil/builder.go`, `arc/cpp/ir/testutil/testutil.h`.

The existing `IRBuilder.Sequence(key, []StageSpec)` helper becomes
`IRBuilder.Sequence(key, []MemberSpec)` where `MemberSpec` describes a `Member` shape.
Add helpers for common cases:

```go
func Parallel(nodeKeys ...string) Member   // single-phase parallel scope of nodes
func Sequential(steps ...Member) Member     // sequential scope with members
```

Keep the existing chainable-builder pattern documented in
`arc/go/ir/testutil/builder.go:27–36`.

## 6.7 - Phase 7: STL Cleanup

**Scope:** `arc/go/stl/stage/stage.go`, related C++ files.

- Delete `EntryNodeName`, `EntryActivationParam`, `EntryNodeInputs`, `SymbolResolver`
  entry, and the `stage_entry` node's `Next` implementation.
- Delete all references in analyzer and tests.

## 6.8 - Phase 8: Documentation

- Update `docs/tech/rfc/0034-260408-arc-composable-execution.md` to note that Layer 2's
  internal representation changed per this RFC. Source semantics are unchanged; the RFC
  stays relevant for the language-level design.
- Update `docs/tech/rfc/0031-260311-arc-scheduler-semantics.md` if its examples
  reference the old `Stage` / `Sequence` / `Flow` IR types.
- Update `docs/claude/architecture.md` and `docs/claude/toolchains/go.md` if they
  describe the old IR shape.

## 6.9 - Test Strategy

Run after each phase:

- Phase 1: oracle sync completes; `.oracle` validates.
- Phase 2: `arc/go/text/` test suite green.
- Phase 3: `arc/go/stratifier/` test suite green.
- Phase 4: `arc/go/runtime/scheduler/` test suite green.
- Phase 5: `bazel test //arc/cpp/...` green.
- Phase 6: `arc/go/ir/testutil/` and all consumers of it compile and test green.
- Phase 7: cleanup; re-run all suites.
- Full: `ginkgo -r` from `arc/go/`, `bazel test //arc/...` clean before merging to `rc`.

Integration tests (`/integration/tests/arc/*.py`) exercise the full Arc compile-and-run
path. These become the final verification: the composable- execution integration tests
from SY-4043 should pass with the new IR without modification to their Arc source files,
validating that source-level semantics are preserved.

# 7 - Alternatives Considered

## 7.0 - Status Quo

Keep the current IR; fix `#2` (typed stratum entries) and `#8` (boundary key collision)
as targeted changes.

**Trade-off:** cheaper now, but the underlying tension between composition and
stratification persists. Future features (history states, richer transitions,
multi-level composability) become harder to add cleanly.

Rejected because the review concluded the tension is architectural, not local.

## 7.1 - Gate Primitive Only (Narrower Reframing)

Introduce a `Gate` primitive that unifies boundary markers and entry nodes, while
keeping `Stage` / `Sequence` / `Flow` / `Step` as distinct types.

**Trade-off:** addresses the boundary/entry mechanism overlap but leaves the four-type
specialization of Layer 2 intact. Half the architectural cleanup.

Rejected in favor of the full unification; the incremental complexity of maintaining
four container types is the larger long-term cost.

## 7.2 - Typed Stratum Entries

Keep the hierarchy but make `Stratum` entries typed (`NodeRef | Boundary | EntryRef`)
instead of bare strings.

**Trade-off:** addresses the magic-string problem but does not touch entry nodes or the
four-type hierarchy. Cosmetic.

Rejected; the underlying architecture remains unchanged.

## 7.3 - Collapse to Strata Everywhere

Remove the composition tree entirely; express everything as strata with state-machine
annotations. The IR becomes a flat list of executable units with cross-references.

**Trade-off:** loses source-grammar alignment entirely. Hard to map errors back to
source. Rejected.

# 8 - Open Questions

## 8.0 - Convergence Loop Preservation

§5.3 flags a behavioral choice. The RFC recommends preserving same-cycle cascading
transitions (via a bounded loop), but this requires confirmation against the integration
test suite — specifically, which Arc programs rely on multiple transitions firing within
one tick.

**Action:** before Phase 4, audit `arc/go/runtime/scheduler/scheduler_test.go` for tests
that assert same-cycle cascading. The `testCascadingTransitionsComplete` tests (grep:
`testCascadingTransitions`) are the relevant cases. If any program behavior depends on
within-tick cascading, the convergence loop must be preserved.

## 8.1 - `Member.key` and `Scope.key` Relationship

Both `Member.key` and a nested `Scope.key` can carry the same string (the source-level
stage name, e.g., `"press"`). Today, entry-node keys use the step key; boundaries use
the step key. In the new model, transitions target `Member.key`. The nested `Scope.key`
is its own identifier used for debugging and potentially for cross-scope activation
resolution.

**Recommendation:** keep both fields distinct but allow the analyzer to set them
identically when the source construct names them together. The IR does not enforce they
match.

## 8.2 - `TransitionTarget` Tagged Union

`TransitionTarget { memberKey: string?, exit: bool? }` has the same "exactly one of"
invariant as `Step` today, and the same Go-limitations rationale for not enforcing it at
the schema level. Builder/analyzer must ensure correctness; runtime panics on invalid
configurations.

## 8.3 - Debug and Error Messages

Current error paths reference `Stage.Key`, `Step.Key`, etc. These identifiers remain
accessible as `Scope.Key` and `Member.Key`, but error messages referencing "stages" or
"sequences" become ambiguous (both are Scopes now). Error messages should be updated to
use source-level terminology (taken from AST context) rather than IR-level type names.

## 8.4 - TypeScript Client Impact

`client/ts/src/arc/ir/types.gen.ts` regenerates from the schema. Any TypeScript code
that discriminates on `Step.Flow` / `Step.Stage` / `Step.Sequence` will need migration.
Likely impact: small (the client is primarily a consumer, not an IR manipulator).

## 8.5 - Backward Compatibility

None. This is a hard wire-format break, as SY-4043 already was. No Arc programs in the
wild are deployed against the current IR; the change is internal.

# 9 - Appendix: Current Code References

Anchors cited above, for the reviewer's convenience:

- Current IR schema: `schemas/arc/ir.oracle`
- Current Go IR types: `arc/go/ir/types.gen.go`, `arc/go/ir/sequence.go`
- Current C++ IR types: `arc/cpp/ir/types.gen.h`
- Current analyzer: `arc/go/text/analyze.go` (especially 794–999)
- Current stratifier: `arc/go/stratifier/stratifier.go`
- Current Go scheduler: `arc/go/runtime/scheduler/scheduler.go`
- Current C++ scheduler: `arc/cpp/runtime/scheduler/scheduler.h`
- Stage entry STL: `arc/go/stl/stage/stage.go`
- Prior scheduler RFC: `docs/tech/rfc/0031-260311-arc-scheduler-semantics.md`
- Prior composable-execution RFC:
  `docs/tech/rfc/0034-260408-arc-composable-execution.md`
