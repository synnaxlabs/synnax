# 34 - Composable Sequential and Parallel Execution in Arc

**Feature Name**: Composable Sequential and Parallel Execution <br /> **Status**: Draft
<br /> **Start Date**: 2026-04-08 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC redesigns Arc's execution model at two levels at once.

At the surface, `sequence` and `stage` become composable execution modes that can be
nested arbitrarily. `sequence` runs children in order. `stage` runs children in
parallel. The fixed `sequence > stage` parent-child relationship goes away. Stageless
sequences, inline stages within sequences, and inline sequences within stages all become
valid.

At the intermediate representation, the four execution-shell types (`Stage`, `Sequence`,
`Step`, `Flow`) and the separate `Strata` field collapse into a single primitive,
`Scope`, parameterized by concurrency mode (`parallel` or `sequential`) and liveness
(`always` or `gated`). Synthesized `stage_entry` nodes and string-keyed boundary markers
are removed. Conditional state transitions become declarative fields on sequential
scopes rather than dataflow edges into synthetic nodes.

The dataflow layer (`Function`, `Node`, `Edge`, `Handle`, `Authorities`) is unchanged.
Every valid Arc program from before this RFC lowers to the new IR and executes with
identical observable behavior.

# 1 - Vocabulary

- **Layer 1 (dataflow)** is the graph of `Function`, `Node`, and `Edge` that expresses
  pure compute. Flat, consumed by the analyzer and the compiler. Unchanged by this RFC.
- **Layer 2 (execution shell)** is the tree rooted at `IR.root` that governs when and in
  what order subsets of the dataflow graph execute. Redesigned by this RFC.
- **Scope** is the single Layer-2 primitive. Every `sequence`, `stage`, and flow step
  lowers to a Scope.
- **Mode** is `parallel` or `sequential`. It decides whether a scope's children run
  together (respecting data dependencies) or one at a time (with transition rules).
- **Liveness** is `always` or `gated`. It decides whether a scope is continuously active
  (root-like) or activates only when its activation handle fires.
- **Phase** is one execution layer inside a parallel scope. Members in the same phase
  have no data dependency among themselves. Phase N depends only on phases 0 to N-1.
- **Member** is a child of a scope. Either a reference to a Layer-1 node (`NodeRef`) or
  a nested scope.
- **NodeRef** is a Layer-2 handle that points at a Layer-1 node by key.
- **Transition** is a declarative rule on a sequential scope: when a dataflow handle
  becomes truthy, switch the active member or exit the scope.
- **Activation** is the event of a gated scope becoming active. Activation cascades a
  reset through directly-owned node members, and recursively activates gated children
  that carry no activation handle of their own.
- **Gate** is a bare expression or `wait{}` node that appears as a sequential step. It
  advances the sequence when its output becomes truthy.
- **Write cascading** is the behavior where consecutive write steps in a sequence
  advance on the same tick. Each write is immediately truthy, and the sequential
  scheduler's convergence loop keeps advancing until it hits a step whose output is not
  truthy (a gate, a reactive stage, or a non-trivial sub-sequence).

# 2 - Motivation

## 2.0 - Sequential Operations Are Verbose

The most common request from hardware-control users is to express a sequential
procedure: "open valve, wait 2 seconds, close valve, wait 1 second, open vent." In the
pre-RFC Arc surface syntax, the only way to sequence operations over time is to split
each step into its own named stage with an explicit `=> next` transition:

```arc
sequence main {
    stage open_valve {
        1 -> valve_cmd,
        wait{duration=2s} => next,
    }
    stage close_valve {
        0 -> valve_cmd,
        wait{duration=1s} => next,
    }
    stage open_vent {
        1 -> vent_cmd,
    }
}
```

That is thirteen lines for a five-step procedure.

## 2.1 - Boilerplate Obscures Business Logic

Across every Arc program in the codebase, the pattern repeats:

- Roughly 73% of stage bodies are trivial channel writes.
- Roughly 60% of programs have terminal stages that only hold final cleanup writes.
- The "set values, wait, transition" pattern appears in almost every sequence.

Real control logic (interval-triggered control functions, conditional predicates) is
diluted among mechanical setup and teardown. The source no longer reads like a
procedure.

## 2.2 - Stages Are the Unit of Sequencing

Arc's reactive model runs every flow in a stage on every tick. Two bare writes in the
same stage body execute on the same cycle; the final value is undefined because neither
write is "after" the other. Stages are what separate "before" from "after" in the
language. The dataflow graph stays parallel; sequencing lives in stage boundaries and
one-shot `=> next` transitions.

Because the language needs stages to sequence but offers no lighter-weight construct,
users pay the stage tax on every logical step. The keywords `sequence` and `stage`
already describe two different composition modes, just wrapped in a fixed parent-child
shape. Treating them as composable modes is a small change to the surface grammar and a
large improvement in the source-to-semantics mapping.

## 2.3 - The Existing IR Resists the Change

Adding composable execution on top of the pre-RFC IR would compound problems it already
has. The IR describes execution through two parallel hierarchies:

- A **composition tree**:
  `IR.root: Stage → Stages.Sequences → Sequence.Steps → Step{Flow | Stage | Sequence}`.
  Source-grammar aligned.
- A **stratification field**: `Stage.strata` and `Sequence.strata` encode the execution
  order among members of each context.

These are nominally independent concerns (structure versus ordering) but are entangled
at every integration point:

1. `Strata` entries hold magic-string markers (`entry_<seq>_<step>`, `boundary_<step>`)
   that refer back to the composition tree.
2. The scheduler probes every strata key at runtime: "is this a real node, a boundary,
   or a marker to skip?" (`arc/go/runtime/scheduler/scheduler.go:286-306` pre-RFC).
3. Three separate places format and parse the `boundary_<step>` convention (stratifier,
   Go scheduler, C++ scheduler). Drift between them produces silent activation breakage.
4. The boundary key format is not sequence-qualified. Two top-level sequences with
   same-keyed sub-sequences collide in the scheduler's global boundaries map.
5. `stage_entry` nodes are synthesized into the dataflow graph purely so conditional
   edges from transition predicates have a reactive firing point. Boundary markers serve
   a different purpose (keeping a sub-sequence executing on every parent cycle while the
   sub-sequence is active). Two mechanisms cooperate to produce state-machine behavior,
   one reactive, one passive.

From the scheduler's perspective, `Stage` and `Flow` differ only in whether they can
contain sub-sequences. Their node-state lifecycle is uniform (`Reset()` on activation,
clear `selfChanged` on deactivation). The distinction is structural, not behavioral.
`Step` exists purely as a tagged-union wrapper around the other three because Go lacks
sum types.

Building composable execution on top of this structure would mean a third mechanism, or
an extension of the magic-string convention, or another special case in the scheduler.
The simpler move is to collapse Layer 2 into one primitive and express the surface
feature through it.

# 3 - Design

## 3.0 - Two Explicit Layers

Arc programs have two explicit layers, with reference as the only coupling between them.

**Layer 1 (dataflow).** `Function`, `Node`, `Edge`, `Handle`, `Authorities`, and their
collections. A flat graph of compute. Unchanged by this RFC. Conditional edges still
gate dataflow propagation ("only push `changed` downstream when the source output is
truthy"); this is a Layer-1 concern, independent of Layer-2 state-machine semantics.

**Layer 2 (execution shell).** A tree of `Scope` values rooted at `IR.root`. Layer 2
references Layer 1 only through `NodeRef` (by node key) and `Handle` (by node key plus
parameter name).

## 3.1 - The Scope Primitive

```
Scope struct {
    key         string
    mode        ScopeMode        // parallel | sequential
    liveness    Liveness         // always | gated
    activation  Handle?          // gated only; unset for always scopes
    phases      Phase[]          // parallel only
    members     Member[]         // sequential only
    transitions Transition[]     // sequential only
}

Phase struct {
    members Member[]             // members that share this execution layer
}

Member struct {
    key     string               // position identifier within the parent scope
    nodeRef NodeRef?             // set when this member references a Layer-1 node
    scope   Scope?               // set when this member is a nested scope
}

NodeRef struct {
    key string                   // key into IR.nodes
}

Transition struct {
    on     Handle                // a Layer-1 node output
    target TransitionTarget
}

TransitionTarget struct {
    memberKey string?            // target: sibling member in this scope
    exit      bool?              // target: exit scope, yield to parent
}
```

`Member` is a tagged union of `NodeRef | Scope`. `TransitionTarget` is a tagged union of
`memberKey | exit`. Exactly one side of each union is set; the analyzer enforces the
invariant. The Oracle schema does not encode the invariant because Go has no sum types.

`ScopeMode` and `Liveness` are enums with `unspecified = 0` sentinels, matching the
existing `EdgeKind` convention at `schemas/arc/ir.oracle:29`. The full schema lives at
`schemas/arc/ir.oracle:56-167`.

## 3.2 - Valid Configurations

Three `(mode, liveness)` combinations appear in compiled programs.

| mode         | liveness | Role                                   |
| ------------ | -------- | -------------------------------------- |
| `parallel`   | `always` | The program root. Continuously active. |
| `parallel`   | `gated`  | What a `stage {}` lowers to.           |
| `sequential` | `gated`  | What a `sequence {}` lowers to.        |

The fourth combination, `sequential + always`, is theoretically expressible (a top-level
state machine that boots at member 0) but no source construct produces it. The analyzer
never emits it, and the scheduler need not handle it specially.

## 3.3 - The Root

`IR.root` is always `Scope{mode: parallel, liveness: always}`. Its phases mix `NodeRef`
members (module-scope reactive flow: channel reads, timers, bare flow statements) and
nested Scope members (top-level sequences and top-level stages). Top level stages are no
longer wrapped in an implicit single-step sequence; they sit directly as members under
the root.

Because `Member` is a tagged union of `NodeRef | Scope`, a parallel scope mixes both
kinds of child within a single phase. No special case is needed at the root.

## 3.4 - What Goes Away

The following are removed from the pre-RFC design:

- From the IR schema: `Flow`, `Stage`, `Stages`, `Step`, `Steps`, `Sequence`,
  `Sequences`, `Stratum`, and `Strata`.
- From `arc/go/stl/stage/`: the `stage_entry` STL node (`EntryNodeName`,
  `EntryActivationParam`, `EntryNodeInputs`, and its `Next` implementation).
- From the stratifier: boundary-key synthesis, entry-node placement, and the
  `stratifySequenceWithFlowSteps` path.
- From the scheduler: the `boundaries` and `transitions` maps keyed by magic strings,
  `registerBoundaries`, `registerTransitions`, and the entry-node-specific dispatch
  branch in `execStrata`.

# 4 - Language Semantics

## 4.0 - Sequence Bodies Run in Order

A `sequence` body is a list of steps executed in order. Each step produces a truthy or
falsy output. When truthy, the sequence advances; when falsy, the sequence blocks on
that step until a subsequent tick produces a truthy result.

```arc
sequence main {
    1 -> valve_cmd           // write: immediately truthy, advances
    wait{duration=2s}        // gate: blocks for 2s, then truthy
    0 -> valve_cmd           // write: advances
    pressure > 50            // gate: blocks until truthy
    1 -> vent_cmd            // write: advances
}
```

All steps follow the same rule, **evaluate, check truthiness, advance if truthy**. Each
step kind differs only in what its output looks like:

- A **write** (`value -> channel`) executes the write and produces an immediately truthy
  output. Consecutive writes advance on the same tick via the convergence loop (§6.4).
- An **expression** evaluates the expression. Truthy advances; falsy blocks until a
  later tick produces a truthy value.
- A **`wait{duration=D}`** is falsy until `D` elapses since the sequence reached the
  step, then truthy.
- A **nested `stage {}`** runs its reactive flows until one of them fires `=> next` (or
  a named jump). The step is truthy when that firing happens.
- A **nested `sequence {}`** runs from member 0. The step is truthy when the nested
  sequence exits (completes its last member or fires an explicit exit transition).

A sequence advances past its last step by exiting the scope.

## 4.1 - Stage Bodies Run in Parallel

Inside a `stage` body, execution follows the pre-RFC reactive rules:

1. All flows execute reactively. A flow fires when an upstream output marks it changed.
2. Constant writes (`1 -> channel`) execute once per activation (the constant node fires
   once).
3. Continuous flows propagate whenever the source changes.
4. Conditional edges (`=> next`, `=> name`) fire once per stage activation, when the
   source output becomes truthy.
5. Every directly-owned node is reset on stage activation.

The one observable difference from the pre-RFC model is that a stage no longer depends
on synthetic entry nodes to receive transitions. Transition handles are evaluated
directly on the parent sequential scope.

## 4.2 - Nesting Rules

**Sequence inside stage.** The sub-sequence is one of the stage's phases' members. It
runs in parallel with the stage's other members. On stage activation, the sub-sequence
resets to member 0; on stage deactivation it freezes in place (§4.4). Wiring
sub-sequence completion into other stage members is deferred work (§10.0).

**Stage inside sequence.** The inline stage becomes the current sequential step. The
sequence blocks on it until the stage fires a transition targeting the parent sequence.

**Sequence inside sequence.** An anonymous nested sequence is semantically equivalent to
inlining its steps in the outer sequence. The analyzer emits the nested scope
structurally; the scheduler treats each nested sequence as a distinct scope. Named
nested sequences are always preserved as independent scopes because they are potential
jump targets.

**Stage inside stage.** Anonymous flattens: the inner stage's items behave as members of
the outer. Named inner stages stay distinct (they are jump targets).

## 4.3 - Transitions

Transitions are declarative rules on sequential scopes. Two surface forms compile to
transitions:

- `=> next` in a flow (inside a stage inside a sequence, or inside an inline stage)
  advances the nearest enclosing sequence to the next sibling member.
- `=> name` inside a sequential-containing context jumps to the sibling member with that
  name. Names resolve by walking up the enclosing-sequence chain (§4.5).

A third surface form, `source => target_scope`, where `target_scope` names a top-level
gated scope (typically an `abort` handler), compiles to two IR entities (§5.6).

A flow step inside a sequence has its transition auto-wired by the analyzer: when the
step's final node fires truthy, advance to the next step, or exit the scope if the step
is last.

## 4.4 - Reset on Entry

One rule applies uniformly at every level: **reset on entry, not on exit.**

When a gated scope activates:

1. Each directly-owned `NodeRef` member is reset (timers restart, one-shot states clear,
   stateful nodes reinitialize).
2. Each nested gated scope member with no activation handle is recursively activated.
3. Sequential scopes additionally set `activeMember = 0`.

When a gated scope deactivates (because a transition exited it, or because its parent
sequential scope advanced past it):

1. Directly-owned `NodeRef` members have their `selfChanged` bit cleared. The node's
   `Reset()` is **not** called.
2. The `active` bit on the scope clears.
3. Nested scope members are **not** walked. Their `activeMember` state freezes.

The two asymmetries (activation cascades and resets; deactivation does not) are
preserved exactly from the pre-RFC scheduler semantics.

## 4.5 - Scope Resolution for `=> name`

`=> name` resolves by walking up the enclosing-sequence chain. At each level, the
analyzer checks the current sequential scope's direct members for one whose key matches.
If found, that is the target. If not, the search continues in the next enclosing
sequence.

Inner scopes shadow outer scopes. Within a single sequence, sibling names must be
unique; name collision at the same level is a compile-time error. Across levels,
shadowing is silent and allowed. The pre-RFC requirement of globally unique names is
relaxed.

If the target is a top-level gated scope (not a sibling in any enclosing sequence), the
analyzer falls back to cross-scope activation (§5.6).

# 5 - Lowering

This section specifies how every Arc surface construct lowers to the Scope primitive.
The authoritative implementation lives in `arc/go/text/analyze.go`.

## 5.0 - Top-Level Sequence

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
    mode:     sequential
    liveness: gated
    activation: nil  // set if and only if some `=> main` targets it
    members: [
        Member{ key: "press", scope: Scope{ parallel, gated, ... } },
        Member{ key: "done",  scope: Scope{ parallel, gated, ... } },
    ]
    transitions: [ /* auto-wired and explicit, see 5.3 and 5.4 */ ]
}
```

The Scope is added as a `Member{scope: ...}` in `IR.root`'s phases.

## 5.1 - Top-Level Stage

```arc
stage abort {
    0 -> ox_cmd,
    1 -> vent_cmd,
}
```

Lowers directly to `Scope{parallel, gated}` under the root, without being wrapped in an
implicit sequence:

```
Scope{
    key:      "abort"
    mode:     parallel
    liveness: gated
    phases:   [ Phase{members: [NodeRef{ox_cmd_write}, NodeRef{vent_cmd_write}]} ]
}
```

Activation is set when some other transition targets `abort` (§5.6). If nothing
activates it, the analyzer leaves `activation: nil` and the scope never runs. That case
is legal (top-level stages may be referenced indirectly) and is diagnosed elsewhere if
unreachable.

Anonymous top-level stages (`stage { ... }` without a name) receive a synthesized key
during analysis so the root's members remain uniquely keyed.

## 5.2 - Stageless Sequence (Flow Steps)

```arc
sequence main {
    1 -> valve_cmd
    wait{duration=2s}
    0 -> valve_cmd
}
```

Each flow step is wrapped in a parallel+gated scope containing the step's nodes in a
single phase. The parent sequential scope holds one `Member{scope: ...}` per step:

```
Scope{
    key:      "main"
    mode:     sequential
    liveness: gated
    members: [
        Member{ key: "step_0", scope: Scope{ parallel, gated, phases: [ Phase{members: [NodeRef{write_valve_1}]} ] } },
        Member{ key: "step_1", scope: Scope{ parallel, gated, phases: [ Phase{members: [NodeRef{wait_2s}]}       ] } },
        Member{ key: "step_2", scope: Scope{ parallel, gated, phases: [ Phase{members: [NodeRef{write_valve_0}]} ] } },
    ]
    transitions: [ /* auto-wired, see 5.3 */ ]
}
```

The wrapper scope is built by `analyze.flowScope` at `arc/go/text/analyze.go:1062`.
Multi-node flow chains (`a -> b -> c`) wrap all chain nodes in a single phase at
analysis time; the stratifier re-layers them by data dependencies before emitting the
final phasing (§6.1).

This wrapping adds one level of nesting per flow step compared to the pre-RFC flat
`Sequence.Strata` encoding. The runtime cost is a pointer hop per descent; the
readability gain is a uniform lowering rule.

## 5.3 - Auto-Wired Sequential Transitions

For every flow-step member in a sequential scope, the analyzer emits a `Transition` on
the parent scope that fires on the step's final node's first output:

```
Transition{
    on:     Handle{node: lastNodeOfStepN, param: firstOutputParam},
    target: TransitionTarget{memberKey: stepN+1.key},
}
```

For the last step, `target.exit = true` instead of a member key. The implementation is
`autoWireTransition` at `arc/go/text/analyze.go:1085`.

## 5.4 - Explicit Transitions (`=> next`, `=> name`)

```arc
stage press {
    1 -> press_cmd,
    pressure > 50 => next,
}
```

`=> next` inside a stage whose parent is a sequence emits a `Transition` on the
enclosing sequential scope, not on the stage itself:

```
Transition{
    on:     Handle{node: comparison_node, param: out},
    target: TransitionTarget{memberKey: next_sibling.key},
}
```

`=> named_target` produces the same shape with `memberKey: "named_target"` resolving
against the enclosing sequence's members.

## 5.5 - Inline Sub-Sequences in a Stage

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

The inline sub-sequence becomes a `Scope{sequential, gated}` member inside one of the
stage's phases. Its `activation` is nil; it inherits the stage's activation cascadingly
(§4.4). On stage re-activation the sub-sequence resets to member 0.

## 5.6 - Cross-Scope Activation (`=> abort`)

```arc
sequence main {
    stage press {
        abort_btn => abort,
    }
}

stage abort { ... }
```

`abort_btn => abort` names a top-level scope that is not a sibling in any enclosing
sequence. The transition does not fit into within-scope `Transition` semantics. The
analyzer decomposes it into two IR entities from the single source clause:

1. On the enclosing sequential scope:
   `Transition{on: abort_btn_handle, target: {exit: true}}`. When `abort_btn` fires, the
   current sequence exits.
2. On the target top-level scope: `activation: abort_btn_handle`. When `abort_btn`
   fires, the abort scope activates.

Both transitions reference the same `Handle`. Their order of effect is independent. The
exit path runs the pre-RFC deactivation rules (non-cascading); the activation path runs
the pre-RFC activation rules (cascading reset).

The implementation of the decomposition is `flowChainProcessor.consumeTransition` at
`arc/go/text/analyze.go:724`, with cross-scope resolution handled by `analyzeStageRef` /
`analyzeSequenceRef`.

## 5.7 - Module-Scope Reactive Flow

```arc
start_cmd => main
sensor -> filter -> clean_sensor
```

These lower as follows:

- `start_cmd => main` sets `main`'s `activation: Handle{start_cmd_node, out}`. The
  conditional edge that would, in the pre-RFC design, terminate at `main`'s entry node
  no longer exists; activation is declarative.
- `sensor -> filter -> clean_sensor` produces three `NodeRef` members in `IR.root`'s
  phases, ordered by their dataflow dependencies.

# 6 - Scheduler

The scheduler walks the Layer-2 scope tree on every cycle, executes the active members
of each reachable scope, evaluates transitions on sequential scopes, and activates gated
scopes whose activation handle fires. The authoritative implementation is in
`arc/go/runtime/scheduler/scheduler.go` and `arc/cpp/runtime/scheduler/scheduler.h`.

## 6.0 - Runtime State Tree

The scheduler mirrors the static scope tree as a runtime `scopeState` tree, built once
at construction:

```
scopeState {
    ir           *Scope
    mode         ScopeMode
    liveness     Liveness
    active       bool
    activeMember int                // sequential only; -1 when inactive
    members      []memberState      // one per Member of the static scope
    memberByKey  map[string]int
    transitionOwner []int           // which member sources each transition's handle
    transitionOnIdx []int           // dense handle ID into markedFlags
}

memberState {
    key     string
    nodeKey string                  // set for NodeRef members
    nodeIdx int                     // dense index into the flag slices
    scope   *scopeState             // set for nested scope members
}
```

Node identity is cached in two dense per-node slices indexed by a scheduler-assigned
integer: `changedFlags` (pending upstream change) and `selfChangedFlags` (the node
requested re-execution on the next cycle). A third dense slice, `markedFlags`, holds one
flag per `(node, param)` pair that sources a sequential-scope transition, pre-resolved
at construction so evaluation is a single array load.

Edge targets, activation targets, and transition handles are all pre-resolved onto
per-output tables on the source node at construction time. The hot path performs no hash
lookups.

## 6.1 - Cycle Structure

`Next()` executes one cycle by walking from the root:

```
walk(s):
    if not s.active: return
    if s.mode == parallel:
        for phase in s.ir.phases:
            for member in phase.members:
                execute(member)
    else (sequential):
        loop up to len(s.members)+1 times:
            if s.activeMember < 0: break
            execute(s.members[s.activeMember])
            if not evaluate_transitions(s): break
    for gated child c in s's members with activation handle:
        if markChanged seen on activation output this cycle: activate(c)

execute(member):
    if member.scope: walk(member.scope)
    else: run the node, honouring changed / selfChanged gates
```

The stratifier emits parallel-scope members in phase-flattened order so that
`walkParallel` iterates `ss.members` directly, using each member's pre-resolved
`nodeIdx` without any per-access map lookup.

## 6.2 - Activation and Reset

Activation of a gated scope:

1. Set `active = true`.
2. If `sequential`: set `activeMember = 0`, recurse into member 0 (reset its node or
   activate its nested scope).
3. If `parallel`: for each direct member, reset its NodeRef node and clear its
   `selfChanged`. For each nested scope member that is `gated` with no activation handle
   of its own, recursively activate it (cascade). Nested gated scopes that carry an
   activation handle are left inactive; they wait for their handle to fire.

Deactivation of a scope:

1. Clear `selfChanged` for each direct NodeRef member. Do **not** call `Reset()`.
2. If sequential: set `activeMember = -1`.
3. Set `active = false`. Do **not** recurse into nested scope members; their state
   freezes.

This preserves the pre-RFC asymmetry exactly (activation cascades and resets;
deactivation does neither), and moves it from scattered site-specific code in the
pre-RFC scheduler into one recursive function.

## 6.3 - Transition Evaluation

`evaluateTransitions` walks a sequential scope's transitions in source order, firing the
first one whose handle was freshly marked truthy during the current cycle.

Two gates apply before firing:

- **Ownership.** If `transitionOwner[i] >= 0 && transitionOwner[i] != activeMember`, the
  transition is skipped. Its source node is owned by a sibling member that is not
  currently active, so its output is stale. A transition whose source sits outside the
  scope (for example, a module-scope channel read driving a cross-scope activation) has
  owner `-1` and fires regardless of which member is active.
- **Fresh mark.** The transition fires only when `markedFlags[transitionOnIdx[i]]` is
  non-zero. A stale-truthy source that did not re-mark this cycle does not fire. This
  mirrors the conditional-edge firing semantic of the pre-RFC scheduler and prevents
  one-shot nodes (wait, interval, latched comparisons) from driving spurious repeated
  transitions on later cycles.

Firing consumes the mark (clears `markedFlags[i]`), deactivates the current active
member, then either deactivates the whole scope (for `exit: true`) or activates the
target sibling. First match wins; later transitions are not evaluated on that cycle.

## 6.4 - Convergence

A cascade of same-cycle transitions (for example, a chain of writes that each advance
the sequence on the same tick) is handled by the per-scope bounded loop in
`walkSequential`:

```
budget = len(members) + 1
loop budget times:
    if activeMember < 0: break
    execute active member
    if not evaluate_transitions: break
```

The budget allows a chain of `N` members to fire `N` consecutive transitions within one
cycle, matching the pre-RFC `execStages` convergence-loop semantics. Integration tests
that rely on write cascading (`integration/tests/arc/stageless_workflow.py` and others)
continue to observe one-tick completion of simple sequences.

Convergence is now per sequential scope, not global. Two independent top-level sequences
each get their own budget. This is a strict refinement of the pre-RFC global-budget
design and fixes the latent same-key collision between top-level sequences' boundaries.

## 6.5 - Mark Propagation and Conditional Edges

`markChanged(outputIdx)` is called by a node's `Next()` to announce that one of its
outputs produced a new value. The scheduler uses the pre-resolved output table:

1. Query truthiness on that output (`IsOutputTruthy(param)`).
2. If truthy and the output sources a sequential transition, set the appropriate
   `markedFlags` bit.
3. For each outgoing edge:
   - Continuous edge: always set the target's `changedFlags` bit.
   - Conditional edge: set the target's bit only if the output is truthy.
4. For each gated scope whose activation handle is this output: if it is not yet active,
   call `activateScope` on it.

No hash lookups happen on the hot path; every table is pre-resolved by key at scheduler
construction.

## 6.6 - Self-Changed

`MarkSelfChanged` is called by a node that wants to re-execute on the next cycle
regardless of upstream changes (for example, a stateful timer that has not yet reached
its deadline). The scheduler preserves the pre-RFC semantics:

- A node that called `MarkSelfChanged` during its `Next()` is re-added to the `changed`
  set on the next cycle.
- When a scope containing that node deactivates, the member's `selfChanged` bit is
  cleared.
- Activation does not explicitly add to `selfChanged`; it only clears.

The logic lives in the same place it did pre-RFC; only the clearing sites move (they are
now in `activateScope`, `activateSequentialMember`, `deactivateMember`, and
`deactivateScope`).

## 6.7 - Authority Flush Ordering

Authority mutations from `set_authority` nodes versus channel writes within a cycle are
ordered by phase dependencies: the stratifier places `set_authority` nodes in earlier
phases than the channel writes that depend on them. This is unchanged from the pre-RFC
design and falls out of the standard dataflow-dependency phasing. See
`docs/tech/rfc/0031-260311-arc-scheduler-semantics.md` for the underlying invariants.

# 7 - Grammar

## 7.0 - Grammar Changes

```
sequenceDeclaration ::= 'sequence' IDENTIFIER? '{' sequenceItem* '}'
sequenceItem        ::= stageDeclaration
                      | sequenceDeclaration
                      | flowStatement
                      | singleInvocation

stageDeclaration    ::= 'stage' IDENTIFIER? stageBody
stageBody           ::= '{' (stageItem (',' stageItem)* ','?)? '}'
stageItem           ::= flowStatement
                      | singleInvocation
                      | sequenceDeclaration
```

Changes from the pre-RFC grammar:

1. Identifiers on `sequence` and `stage` are optional.
2. `sequenceItem` now allows `flowStatement`, `singleInvocation`, and nested
   `sequenceDeclaration` in addition to `stageDeclaration`.
3. `stageItem` now allows `sequenceDeclaration`.
4. Items in a `sequence` body are not comma-separated (order is positional).
5. Items in a `stage` body are comma-separated (unchanged).

## 7.1 - Backwards Compatibility

Every pre-RFC Arc program parses and compiles unchanged:

- `IDENTIFIER LBRACE` is unambiguous, so making the identifier optional introduces no
  parse conflict.
- Traditional programs use only `stageDeclaration` as the `sequenceItem`, which starts
  with the `STAGE` keyword and cannot be confused with the new alternatives.
- Pre-RFC programs never have `sequence` inside a stage body, so adding
  `sequenceDeclaration` to `stageItem` is purely additive.
- `=> next` in a stage resolves to the next step in the enclosing sequence. For
  traditional programs where every step is a stage, this is identical to the pre-RFC
  behavior.
- `=> name` walks up the scope chain. For traditional programs with globally unique
  stage names, the first chain level always resolves.
- All stage activation, reset, and transition behaviors compile to the same runtime
  semantics. A traditional `sequence main { stage a { ... } stage b { ... } }` is a
  `Scope{sequential, gated}` with two gated-parallel members, each preserving its own
  scope-level activation semantics.

# 8 - Edge Cases

## 8.0 - Empty Sequence

```arc
sequence main {}
```

Compiles to `Scope{sequential, gated, members: []}`. On activation, `activeMember` is
set but there are no members; the sequence exits on the same cycle. If it is a member of
a parent sequence, the parent advances on the same cycle.

## 8.1 - Sequence of Only Writes

```arc
sequence main {
    1 -> valve_a
    1 -> valve_b
    1 -> valve_c
}
```

Every step is a flow whose last node is a constant write (immediately truthy). The
convergence loop (§6.4) advances through all three on the first cycle. The sequence
exits after member 2's transition fires `exit: true`.

## 8.2 - Multiple Exits from a Reactive Step

```arc
sequence main {
    1 -> ox_mpv_cmd
    stage {
        interval{period=100ms} -> control_tpc{},
        ox_pt_1 < 15 => next,
        wait{duration=30s} => abort,
    }
    0 -> ox_mpv_cmd
}

stage abort {
    0 -> ox_mpv_cmd,
    1 -> vent_cmd,
}
```

If `ox_pt_1 < 15` fires first, the enclosing sequence advances past the stage. If the
30s timeout fires first, `=> abort` decomposes (§5.6) into an exit on the current
sequence and an activation on the top-level `abort` scope.

## 8.3 - Jumping Backwards

```arc
sequence main {
    sequence pressurize {
        1 -> press_cmd
        pressure > 50
    }
    stage hold {
        0 -> press_cmd,
        wait{duration=2s} => next,
        pressure < 40 => pressurize,
    }
}
```

Inside `hold`, `pressure < 40 => pressurize` compiles to a transition on the enclosing
`main` scope with `memberKey: "pressurize"`. Firing activates the `pressurize` member
(which resets and starts from its first step). The cycle re-enters `pressurize` on the
same tick via the convergence loop.

`=> pressurize` must be inside a stage. Flow steps in sequences cannot jump to named
targets (§8.5).

## 8.4 - `=> next` on the Last Step

Compile-time error. The last member of a sequence has no "next" sibling; either the
member should be terminal (a stage that does not fire `=> next`) or it should use
`=> name` to jump explicitly.

The check happens when auto-wiring transitions in `autoWireTransition` and when explicit
`=> next` intents resolve via `shellBuilder.top().nextMember()`.

## 8.5 - `=> name` in a Flow Step Inside a Sequence

Compile-time error. Flow steps in sequences already carry an auto-wired transition to
the next step. Adding a conditional jump would make advancement ambiguous. To express a
conditional jump, wrap the flow in an explicit `stage {}`:

```arc
sequence main {
    stage {
        1 -> press_cmd,
        pressure < 40 => abort,
    }
    0 -> press_cmd
}
```

Explicit `=> next` inside a flow step is allowed silently (the auto-wiring produces the
same result).

# 9 - Examples

## 9.0 - Before and After: Valve Timing

Before:

```arc
sequence main {
    stage open_ox {
        1 -> ox_mpv_cmd,
        wait{duration=500ms} => next,
    }
    stage open_fuel {
        1 -> fuel_mpv_cmd,
    }
}
```

After:

```arc
sequence main {
    1 -> ox_mpv_cmd
    wait{duration=500ms}
    1 -> fuel_mpv_cmd
}
```

## 9.1 - Full TPC Cold-Flow Sequence

Before:

```arc
sequence main {
    stage initialize {
        0 -> ox_press_cmd,
        0 -> ox_mpv_cmd,
        0 -> gas_booster_iso_cmd,
        1 -> ox_vent_cmd,
        ox_vent_state == 1 => next,
    }
    stage press {
        1 -> press_iso_cmd,
        1 -> ox_press_cmd,
        1 -> gas_booster_iso_cmd,
        ox_pt_1 > 50 => next,
    }
    stage press_high {
        0 -> ox_press_cmd,
        0 -> press_iso_cmd,
        1 -> gas_booster_iso_cmd,
        press_pt_1 > 150 => next,
    }
    stage hold {
        0 -> gas_booster_iso_cmd,
        wait{duration=2s} => next,
    }
    stage tpc {
        1 -> ox_mpv_cmd,
        1 -> press_iso_cmd,
        interval{period=100ms} -> control_tpc{},
        ox_pt_1 < 15 => next,
    }
    stage safe {
        0 -> ox_press_cmd,
        0 -> ox_mpv_cmd,
        0 -> gas_booster_iso_cmd,
        1 -> ox_vent_cmd,
    }
}
```

After:

```arc
sequence main {
    stage initialize {
        0 -> ox_press_cmd,
        0 -> ox_mpv_cmd,
        0 -> gas_booster_iso_cmd,
        1 -> ox_vent_cmd,
        ox_vent_state == 1 => next,
    }
    stage press {
        1 -> press_iso_cmd,
        1 -> ox_press_cmd,
        1 -> gas_booster_iso_cmd,
        ox_pt_1 > 50 => next,
    }
    stage press_high {
        0 -> ox_press_cmd,
        0 -> press_iso_cmd,
        1 -> gas_booster_iso_cmd,
        press_pt_1 > 150 => next,
    }
    0 -> gas_booster_iso_cmd
    wait{duration=2s}
    stage tpc {
        1 -> ox_mpv_cmd,
        1 -> press_iso_cmd,
        interval{period=100ms} -> control_tpc{},
        ox_pt_1 < 15 => next,
    }
    stage safe {
        0 -> ox_press_cmd,
        0 -> ox_mpv_cmd,
        0 -> gas_booster_iso_cmd,
        1 -> ox_vent_cmd,
    }
}
```

The `hold` stage collapses into two bare lines. Stages with real reactive logic remain
intact.

## 9.2 - Mixed Sequential and Reactive

```arc
sequence main {
    1 -> ox_mpv_cmd
    wait{duration=500ms}
    1 -> fuel_mpv_cmd
    stage tpc {
        sequence {
            set_authority{value=220}
            wait{duration=100ms}
            set_authority{value=200}
        },
        interval{period=100ms} -> control_tpc{},
        ox_pt_1 < 15 => next,
    }
    0 -> ox_mpv_cmd
    0 -> fuel_mpv_cmd
    1 -> vent_cmd
}
```

The `tpc` stage carries an inline sub-sequence (authority ramp-down) running alongside
reactive control and an exit predicate. When the exit fires, the parent sequence
continues with cleanup.

# 10 - Deferred Work

## 10.0 - Sub-Sequence Completion Wiring

When an inline sub-sequence in a stage completes, it becomes inactive and stays inactive
until the parent stage re-activates. Wiring completion as a signal into other stage
members (`sequence { ... } => next`) is not supported. The surface and IR changes needed
to express "done" as a Handle on a sequential scope are future work.

## 10.1 - Explicit Loop Constructs

A sequence runs once per activation. To loop, use `=> name` from within a stage, or bind
the sequence's exit to a re-activation handle. A dedicated `loop { ... }` construct can
be added later without breaking changes.

## 10.2 - Early Exit from Sequences

A bare `condition => name` as a sequential step is not allowed (§8.5). The user almost
always wants concurrent monitoring, which is what stages are for:

```arc
stage main {
    sequence {
        1 -> valve_cmd
        wait{duration=2s}
        0 -> valve_cmd
    },
    pressure > 200 => abort,
}
```

Sequential gates only advance; they do not jump.

# 11 - Risks and Notes

## 11.0 - Wire-Format Break

The IR schema is a hard break from the pre-RFC shape. Programs compiled against the
pre-RFC IR cannot be loaded by the new runtime. No Arc programs were deployed against
the pre-RFC IR; the change is internal and accepted.

## 11.1 - TypeScript Client Regeneration

`client/ts/src/arc/ir/types.gen.ts` regenerates from the schema. Any TypeScript code
that discriminated on `Step.Flow`, `Step.Stage`, or `Step.Sequence` migrates to
discriminating on `Member.nodeRef` versus `Member.scope` and on `Scope.mode`. Impact is
small: the client is primarily a consumer, not an IR manipulator.

## 11.2 - Error Messages

Pre-RFC error paths referenced `Stage.Key`, `Step.Key`, and similar fields. These
identifiers remain accessible as `Scope.Key` and `Member.Key`, but the surface-level
terminology ("stage", "sequence", "step") no longer maps one-to-one onto IR types. Error
messages should be phrased from source-level context (the AST node where the diagnostic
fires) rather than from IR-level type names, so that a user reading a diagnostic does
not need to know whether a given construct is a `parallel` or `sequential` Scope.

# 12 - Appendix: Source References

Anchors for reviewers:

- IR schema: `schemas/arc/ir.oracle`
- Generated Go IR: `arc/go/ir/types.gen.go`, `arc/go/ir/ir.go`
- Generated C++ IR: `arc/cpp/ir/types.gen.h`
- Analyzer: `arc/go/text/analyze.go` (`shellBuilder`, `analyzeSequence`, `analyzeStage`,
  `flowScope`, `autoWireTransition`)
- Stratifier: `arc/go/stratifier/stratifier.go`
- Go scheduler: `arc/go/runtime/scheduler/scheduler.go`
- C++ scheduler: `arc/cpp/runtime/scheduler/scheduler.h`
- Prior scheduler RFC: `docs/tech/rfc/0031-260311-arc-scheduler-semantics.md`
- Integration tests that exercise the surface change:
  `integration/tests/arc/stageless_workflow.py`,
  `integration/tests/arc/inline_sequence_in_stage.py`,
  `integration/tests/arc/inline_stage_in_sequence.py`,
  `integration/tests/arc/backward_jump.py`, `integration/tests/arc/edge_cases.py`
