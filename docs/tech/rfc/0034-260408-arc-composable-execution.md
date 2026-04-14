# 34 - Composable Sequential and Parallel Execution in Arc

**Feature Name**: Composable Sequential and Parallel Execution <br /> **Status**: Draft
<br /> **Start Date**: 2026-04-08 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

This RFC generalizes `sequence` and `stage` from a fixed parent-child relationship into
two composable execution modes that can be nested arbitrarily. `sequence` means "run
children in order." `stage` means "run children in parallel." This eliminates the
verbosity of expressing sequential timed operations while preserving full backwards
compatibility with existing Arc programs.

The key architectural change is separating the execution context (what's active, what
gets reset) from the composition mode (sequential vs parallel). The scheduler operates
on a unified strata tree where execution contexts are boundaries within strata, not a
separate hierarchy.

# 1 - Vocabulary

- **Step** - A child of a sequence. Steps have three kinds: stage (parallel), sequence
  (sequential), or flow (leaf). Stage and sequence steps are execution contexts that own
  their own strata. Flow steps are leaves whose nodes live in the parent sequence's
  strata.
- **Gate** - A bare expression or `wait{}` node that appears as a step in a sequence. It
  blocks progression until it evaluates to truthy, then the sequence advances.
- **Execution Context Boundary** - A synthetic scheduler-level construct that represents
  a child context (sequence or stage) within a parent's strata. It has a position in the
  parent's topological order and contains sub-strata that it manages internally.
- **Write Cascading** - The behavior where consecutive write steps in a sequence all
  execute on the same tick, advancing instantly until the sequence hits a gate, stage,
  sequence, or its end.
- **Strata Tree** - The unified hierarchy of execution strata. Each execution context
  has its own strata containing directly-owned nodes and execution context boundaries
  for child contexts.

# 2 - Motivation

## 2.0 - Sequential Operations Are Verbose

The most natural thing a hardware control user wants to express is a sequential
procedure: "open valve, wait 2 seconds, close valve, wait 1 second, open vent." In
current Arc, the only way to sequence operations over time is to split each step into
its own named stage with an explicit `=> next` transition:

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

This is 13 lines for a 5-step procedure.

## 2.1 - Boilerplate Obscures Business Logic

Analysis of every Arc program in the codebase reveals:

- ~73% of stage content is trivial channel writes
- ~60% of programs have trivial terminal stages (just cleanup)
- The pattern "set values + `wait` + `=> next`" appears in almost every program
- Real logic (interval-triggered functions, complex conditions) is diluted among
  mechanical setup/teardown operations

## 2.2 - The Root Problem

Arc's execution model is reactive: all flows within a stage execute in parallel on each
tick. There is no concept of "this line runs before that line" within a stage. A
constant like `1 -> valve_cmd` executes exactly once (it is a constant node), but
`1 -> valve_cmd` and `0 -> valve_cmd` in the same stage would both execute on the first
cycle, with the final value being nondeterministic.

Stages are the unit of sequencing. The language makes reactive dataflow easy but
imperative sequencing verbose.

## 2.3 - Existing Semantics Already Imply the Solution

The existing syntax already separates sequential from parallel:

- `sequence` bodies have no commas between children (stages). Order is positional.
- `stage` bodies have commas between children (flows). Items are listed as coexisting.

The keywords already describe composition modes. They are constrained to a fixed
parent-child relationship (`sequence` contains `stage`) when they could be composable
primitives.

# 3 - Design

## 3.0 - Core Idea

Redefine `sequence` and `stage` as two composable execution modes:

- **`sequence`**: Children execute in order. No commas between children.
- **`stage`**: Children execute in parallel (reactive). Commas between children.

These can be nested arbitrarily. The existing `sequence > stage` parent-child pattern
remains valid and is the most common form.

## 3.1 - Stageless Sequences

A sequence can contain bare operations without wrapping them in stages:

```arc
sequence main {
    1 -> valve_cmd
    wait{duration=2s}
    0 -> valve_cmd
    wait{duration=1s}
    1 -> vent_cmd
}
```

No stages, no commas, no `=> next`. Five lines for a five-step procedure.

Each line is a sequential step. Writes execute immediately and advance. Expressions and
`wait` nodes act as gates: execution blocks at that step until the expression evaluates
to truthy, then advances.

## 3.2 - Inline Sequences Within Stages

A stage can contain a `sequence` block alongside reactive flows:

```arc
stage fire {
    sequence {
        1 -> ox_mpv_cmd
        wait{duration=500ms}
        1 -> fuel_mpv_cmd
    },
    interval{period=100ms} -> control_tpc{},
    ox_pt_1 < 15 => next,
}
```

The `sequence` block progresses through its steps while the reactive flows (`interval`,
exit condition) run in parallel. The `sequence` block is one of the stage's parallel
items and requires a trailing comma like any other stage item.

## 3.3 - Inline Stages Within Sequences

A sequence can contain `stage` blocks for reactive sections:

```arc
sequence main {
    1 -> ox_mpv_cmd
    wait{duration=500ms}
    1 -> fuel_mpv_cmd
    stage {
        interval{period=100ms} -> control_tpc{},
        ox_pt_1 < 15 => next,
    }
    0 -> ox_mpv_cmd
    1 -> vent_cmd
}
```

The `stage` block runs its reactive flows until something fires `=> next`, then the
parent sequence advances to the next step.

## 3.4 - Named Blocks as Jump Targets

Named `stage` and `sequence` children within a `sequence` are jump targets for `=>`:

```arc
sequence main {
    stage pressurize {
        1 -> press_cmd,
        pressure > 50 => next,
    }
    sequence fire {
        1 -> ox_mpv_cmd
        wait{duration=500ms}
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
}
```

`=> next` advances to the next sibling in the parent sequence. `=> abort` jumps to the
named block. `=> pressurize` would restart that stage.

## 3.5 - Terminal States

A stage in a sequence without an exit condition runs forever and never advances:

```arc
sequence main {
    stage pressurize {
        1 -> press_cmd,
        pressure > 50 => next,
    }
    stage safe {
        0 -> press_cmd,
        1 -> vent_cmd,
    }
}
```

A sequence without an explicit exit condition advances automatically when its last step
completes.

## 3.6 - Comma Rules

The rule is consistent:

- **`stage` bodies**: commas between items (parallel items are listed)
- **`sequence` bodies**: no commas between items (order is positional)

This is already the case in existing programs. When a `sequence {}` block appears inside
a `stage {}`, it is one of the stage's parallel items and uses a trailing comma. When a
`stage {}` block appears inside a `sequence`, it has no trailing comma.

# 4 - Detailed Semantics

## 4.0 - Sequential Execution Model

Within a `sequence` body, every item is evaluated in order. Each item produces a truthy
or falsy output. When truthy, the sequence advances to the next item. When falsy, the
sequence blocks at that item until a subsequent tick produces a truthy result.

All items follow the same rule: **evaluate, check truthiness, advance if truthy.**

- **Write operations** (`value -> channel`): The write executes and immediately produces
  a truthy output. The sequence advances on the same tick. Consecutive writes all
  execute on the same tick because each one is immediately truthy (write cascading).
- **Expressions** (`pressure > 50`): Evaluates the expression. If truthy, advances. If
  falsy, blocks until a subsequent tick where it becomes truthy.
- **`wait{}`** (`wait{duration=2s}`): Evaluates to falsy until the duration elapses,
  then truthy. The timer starts when the sequence reaches that item.
- **Nested `stage {}` blocks**: The stage's reactive flows execute. The item is truthy
  when something inside fires `=> next`.
- **Nested `sequence {}` blocks**: The inner sequence executes from item 0. The item is
  truthy when the inner sequence completes (runs past its last item).

When the sequence advances past its last item, it is "complete."

## 4.1 - Gate Semantics

A gate is any expression that appears as a bare step in a sequence:

```arc
sequence main {
    1 -> press_cmd          // write: executes and advances
    pressure > 50           // gate: blocks until truthy
    0 -> press_cmd          // write: executes after gate passes
    wait{duration=2s}       // gate: blocks for 2 seconds
    1 -> vent_cmd           // write: executes after wait
}
```

The `wait{duration=D}` node is a gate that becomes truthy after D elapses since it was
first evaluated. Its timer starts on the tick when the sequence reaches that step.

Consecutive writes before a gate all execute on the same tick:

```arc
sequence main {
    1 -> valve_a            // tick 0
    1 -> valve_b            // tick 0 (same tick)
    0 -> press_cmd          // tick 0 (same tick)
    wait{duration=2s}       // tick 0: starts waiting. tick N: becomes truthy
    0 -> valve_a            // tick N (after wait)
    0 -> valve_b            // tick N (same tick)
}
```

## 4.2 - Parallel Execution Model (Unchanged)

Within a `stage` body, execution is unchanged from current Arc:

1. All flows execute reactively on each tick.
2. Constant writes (`1 -> channel`) execute once (constant node).
3. Continuous flows (`source -> sink`) execute whenever source changes.
4. One-shot edges (`condition => target`) fire once when truthy, per stage activation.
5. Node reset occurs on stage entry.

## 4.3 - Nesting Semantics

### `sequence` inside `stage` (parallel context)

The sequence block is an execution context boundary in the stage's strata. It progresses
through its steps on each tick while the other reactive flows also execute. When it
completes, it goes idle. The parent stage continues with its other reactive flows.
Wiring sequence completion to outgoing edges is deferred to a future RFC.

When the parent stage is re-entered, the sequence resets to step 0.

### `stage` inside `sequence` (sequential context)

The stage block becomes the current step. The sequence is blocked until the stage fires
`=> next`, which advances the parent sequence. If the stage fires `=> name`, control
jumps to that named block.

### `sequence` inside `sequence`

An anonymous inner sequence is semantically equivalent to its steps being inline in the
outer sequence. The implementation may either flatten at compile time (inline the inner
steps) or preserve the nesting in the IR and let the scheduler recurse. Named inner
sequences are always preserved as they are jump targets.

### `stage` inside `stage`

Flattens semantically. An anonymous inner stage is equivalent to its items being inline.

## 4.4 - Transition Semantics

| Context                                      | `=> next`                      | `=> name`                                            |
| -------------------------------------------- | ------------------------------ | ---------------------------------------------------- |
| Flow in a `stage` child of a `sequence`      | Advances the parent `sequence` | Jumps to named block in nearest enclosing `sequence` |
| Flow in an inline `stage {}` in a `sequence` | Advances the parent `sequence` | Jumps to named block in nearest enclosing `sequence` |
| Flow in a `stage` inside a `stage`           | Flattens; same as parent stage | N/A                                                  |

`=> next` always means "advance to the next step in the nearest enclosing sequence."

## 4.5 - Reset Behavior

One rule, applied uniformly at every level: **reset on entry.**

When a step becomes the active step (by advancement, `=> next`, or `=> name` jump), its
nodes are reset. Timers restart, one-shot states clear, stateful function nodes
reinitialize.

For sequence steps: entering a sequence sets `activeStepIdx = 0`, which enters step 0.
Step 0's nodes are reset. Other steps are reset when reached.

For stage steps: entering a stage resets all of its directly-owned nodes. Any sub-
sequence within the stage enters its step 0.

Deactivation does not call Reset(). It only clears nodes from selfChanged so they stop
executing. Reset happens on the next entry, not on exit.

## 4.6 - Scope Resolution for `=> name`

`=> name` resolves by walking up the execution context tree. At each level, it checks
siblings in the enclosing sequence for a matching name. If found, it targets that block.
If not, it checks the next enclosing sequence. This continues until a match is found or
the global scope is reached.

Inner scopes shadow outer scopes. Names must be unique within a single sequence scope
(siblings cannot share a name). Current Arc requires globally unique names; this
restriction is relaxed to per-scope uniqueness. Shadowing is allowed silently.

# 5 - Edge Cases

## 5.0 - Empty Sequence

```arc
sequence main {}
```

Completes immediately. If inside a parent sequence, advances to the next step on the
same tick.

## 5.1 - Sequence With Only Writes

```arc
sequence main {
    1 -> valve_a
    1 -> valve_b
    1 -> valve_c
}
```

All writes execute on the first tick via write cascading. Sequence completes
immediately.

## 5.2 - Multiple Exits From a Reactive Block in a Sequence

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

If `ox_pt_1 < 15` fires first, the sequence advances past the stage. If the 30s timeout
fires first, control jumps to `abort`.

## 5.3 - Jumping Backwards

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

The `hold` stage runs reactively. If pressure drops below 40, `=> pressurize` fires and
control jumps back to `pressurize`, which resets and starts from step 0. This creates a
loop. Note that the conditional jump `pressure < 40 => pressurize` must be inside a
`stage` because flow steps in sequences cannot jump to named blocks (section 9.3).

## 5.4 - `=> next` on Last Step

Compile-time error. The last child of a sequence has no "next." If the last child is a
stage, it must either be terminal or transition to a named block.

# 6 - Grammar Changes

## 6.0 - Current Grammar

```
SequenceDeclaration ::= 'sequence' Identifier '{' StageDeclaration* '}'
StageDeclaration    ::= 'stage' Identifier '{' (StageItem (',' StageItem)* ','?)? '}'
StageItem           ::= FlowStatement | SingleInvocation
```

## 6.1 - Proposed Grammar

```
SequenceDeclaration ::= 'sequence' Identifier? '{' SequenceItem* '}'
StageDeclaration    ::= 'stage' Identifier? '{' (StageItem (',' StageItem)* ','?)? '}'

SequenceItem ::= StageDeclaration
               | SequenceDeclaration
               | FlowStatement
               | Expression

StageItem ::= FlowStatement
            | SingleInvocation
            | SequenceDeclaration ','
```

Changes:

1. `sequence` and `stage` identifiers are now optional (anonymous blocks).
2. `SequenceItem` can be a stage, nested sequence, flow, or bare expression (gate).
3. `StageItem` can now include `SequenceDeclaration` (followed by comma).
4. Items in a `sequence` body are not comma-separated.
5. Items in a `stage` body are comma-separated (unchanged).

## 6.2 - Backwards Compatibility

Every valid current Arc program remains valid. No existing syntax is changed or removed.

### Syntax Compatibility

The grammar changes are purely additive:

- `IDENTIFIER` becomes optional on `sequence` and `stage`, but all existing programs
  provide identifiers. No ambiguity since `IDENTIFIER LBRACE` is unambiguous.
- `sequenceItem` adds `flowStatement`, `singleInvocation`, and `sequenceDeclaration` as
  alternatives. Existing programs only use `stageDeclaration`, which starts with the
  `STAGE` keyword and cannot be confused with the new alternatives.
- `stageItem` adds `sequenceDeclaration`. Existing programs never have `sequence` inside
  a stage body.

### Semantic Compatibility

- `=> next` in a stage resolves to the next step in the sequence. For traditional
  programs where every step is a stage, this is identical to the current behavior.
- `=> stage_name` resolves by walking up the scope tree. For traditional programs with
  globally unique names, the first scope checked (the enclosing sequence) will always
  find the target. Same result.
- All stage entry, reset, transition, and convergence behaviors are preserved. A
  traditional sequence compiles to a `Sequence` with all `StepKindStage` steps.

# 7 - Runtime Scheduler

## 7.0 - Separation of Concerns

In the current scheduler, stages serve double duty as both execution context (what's
active, what gets reset) and composition mode (parallel reactive flows). This proposal
separates them:

- **Step**: The execution context. The scheduler operates exclusively on steps.
- **Stage**: A composition mode. At the scheduler level, a stage is just the content of
  a step that happens to have parallel reactive flows.

## 7.1 - One Strata Tree

There is a single unified strata tree. Execution contexts are boundaries within that
tree where activation rules change, not a separate hierarchy.

An execution context (sequence or stage) appears as a boundary node in its parent's
strata. It has a position in the parent's topological order. When the parent's strata
execution reaches it, the scheduler enters the child context, executes its sub-strata,
returns, and continues with the parent's remaining strata.

Example:

```arc
stage fire {
    sequence {
        1 -> ox_mpv_cmd
        wait{duration=500ms}
        1 -> fuel_mpv_cmd
    },
    interval{period=100ms} -> ctrl{},
    pressure < 15 => next,
}
```

The stage's strata:

```
stratum 0: [sequence_context, interval_node]
stratum 1: [ctrl_node]
stratum 2: [pressure_expr, entry_node]
```

`sequence_context` is an execution context boundary at stratum 0 alongside
`interval_node`. When the scheduler reaches it, it enters the sub-context, executes the
active step's sub-strata, returns, and continues. The sub-context's nodes are not in the
parent's strata directly. They are nested inside the boundary.

Ordering between a sub-sequence and sibling reactive flows is determined by the sub-
sequence's position in the parent's strata (definition order). The same positional
priority rules apply to execution context boundaries as to regular nodes.

## 7.2 - Three Kinds of Steps

There are three kinds of steps: **stage** (parallel), **sequence** (sequential), and
**flow** (leaf). The runtime mirrors the IR types:

```go
type sequenceState struct {
    ir            ir.Sequence
    activeStepIdx int
    steps         []stepState
}

type stepState struct {
    ir      ir.Step
    subSeqs []sequenceState   // runtime state for sub-sequences (stage steps only)
    subSeq  *sequenceState    // runtime state for nested sequence (sequence steps only)
}
```

The step kind is determined by which field is non-nil on `ir.Step`:

- `step.ir.Flow != nil`: flow step. No sub-state.
- `step.ir.Stage != nil`: stage step. `subSeqs` tracks inline sub-sequences within the
  stage.
- `step.ir.Sequence != nil`: sequence step. `subSeq` points to the nested sequence's
  runtime state.

A traditional `sequence main { stage a { ... } stage b { ... } }` compiles to a
`sequenceState` with two steps, each having `ir.Stage` populated.

A stageless `sequence main { 1 -> v, wait{2s}, 0 -> v }` compiles to a `sequenceState`
with three steps, each having `ir.Flow` populated. Each flow step's output is checked
for truthiness. A write produces an immediately truthy output, so the sequence advances
on the same tick. A wait produces falsy until the duration elapses. "Write cascading" is
an emergent property of consecutive flow steps that are immediately truthy.

## 7.3 - Per-Tick Step Execution

The scheduler walks the strata of the active execution context. When it encounters a
regular node key, it executes the node (if it belongs to the active flow step, for
sequences). When it encounters an execution context boundary key, it checks whether that
boundary is the active step and enters it if so.

**Stage steps**: Execute the stage's strata. All nodes execute in parallel. Execution
context boundaries for sub-sequences are entered and their active flow step nodes
execute. The stage step does not advance until something fires `=> next`.

**Sequence steps**: Walk the sequence's strata. Only execute nodes belonging to the
active flow step (filtered by `activeStepIdx`). When encountering an execution context
boundary, enter it if it's the active step. Flow step nodes use entry node + one-shot
edge machinery to advance. When a flow's output is truthy, the one-shot edge fires the
next step's entry node, advancing `activeStepIdx`. The scheduler re-evaluates
immediately (same tick), producing write cascading for immediately-truthy flows.

**Flow steps**: Not executed independently. Their nodes are in the parent sequence's
strata, filtered by `activeStepIdx`.

## 7.4 - Convergence

The convergence loop walks the execution context tree until stable:

```
Loop until stable:
    For each active sequence:
        Evaluate active step:
            Stage:    execute strata, check for transitions
            Sequence: evaluate active item, advance if truthy, recurse
    Stable when: no step advanced AND no transitions fired
```

Maximum convergence iterations = total number of steps across all sequences.

## 7.5 - Transition Table

Entry keys use the format `entry_{seqName}_{stepKey}`. With nesting, each sequence
(including nested sequences) has its own entries in the transition table. The transition
target includes enough information to identify both the sequence and the step within it.
For top-level sequences this is `(seqIdx, stepIdx)` as today. For nested sequences, the
target references the nested `sequenceState` directly (resolved during scheduler
initialization when the runtime state tree is built from the IR tree).

Named stages and named sequences get entry keys. Anonymous blocks do not.

## 7.6 - Step Reset

When a step becomes active:

1. Its directly-owned nodes are reset.
2. Its `firedOneShots` set is cleared.
3. If nested sequence: `activeStepIdx` set to 0, step 0's nodes reset. Other steps reset
   when reached.
4. If stage with sub-sequences: each sub-sequence enters step 0.

## 7.7 - Deadlines

- `wait{duration=D}` gate: reports deadline of `startTime + D`. Timer starts when the
  sequence reaches that step.
- Boolean gate (`pressure > 50`): no deadline. Relies on channel input events.
- Stage step: delegates to the stage's timer nodes.

# 8 - IR Changes

## 8.0 - Three IR Types Replace One

The old `Stage` type is replaced by three distinct types, each carrying only the fields
relevant to its execution model. `Step` is a tagged union that holds exactly one.

```go
type Flow struct {
    Nodes []string   // node keys belonging to this flow step
}

type Stage struct {
    Key       string       // stage name (empty if anonymous)
    Nodes     []string     // reactive flow nodes
    Strata    Strata       // multi-stratum reactive ordering
    Sequences []Sequence   // inline sub-sequences within this stage
}

type Sequence struct {
    Key    string   // sequence name (empty if anonymous)
    Steps  []Step   // ordered steps
    Strata Strata   // contains flow step nodes + execution context boundaries
}

type Step struct {
    Key      string     // name for jump targets, empty for anonymous
    Flow     *Flow      // non-nil: this step is a flow (leaf)
    Stage    *Stage     // non-nil: this step is a stage (parallel)
    Sequence *Sequence  // non-nil: this step is a sequence (sequential)
}
```

Exactly one of `Flow`, `Stage`, `Sequence` is non-nil on a `Step`. The step kind is
implicit from which field is populated.

- **Flow**: A single dataflow chain. Leaf node in the execution tree. `1 -> valve_cmd`,
  `pressure > 50`, `wait{duration=2s}`. Nodes include the flow's expression/write nodes
  and the entry node for advancing. Flow steps do not have their own strata. Their nodes
  live in the parent sequence's `Strata`.
- **Stage**: Parallel reactive flows. Contains its own nodes and strata. May also
  contain inline sub-sequences (`Sequences` field) that run alongside the reactive flows
  as execution context boundaries in the stage's strata.
- **Sequence**: Sequential execution. Contains ordered steps and a `Strata` field that
  holds all flow step nodes plus execution context boundaries for stage/sequence steps.
  The `activeStepIdx` at runtime determines which flow nodes are active. Stage and
  sequence steps appear as boundaries in the strata.

A traditional `sequence main { stage a { ... } stage b { ... } }` compiles to a
`Sequence` with two steps, each having `Stage` populated. The sequence's own `Strata`
contains only the execution context boundaries for those two stages.

A stageless `sequence main { 1 -> v, wait{2s}, 0 -> v }` compiles to a `Sequence` with
three flow steps. The sequence's `Strata` contains all the flow nodes (write nodes, wait
node, entry nodes) in topological order. The scheduler uses `activeStepIdx` to determine
which flow nodes to execute.

# 9 - Analyzer Changes

## 9.0 - Declaration Collection Becomes Recursive

The first analyzer pass (`CollectDeclarations`) currently walks top-level sequences and
registers their stage children. With nesting, this becomes recursive:

1. Walk top-level `sequenceDeclaration` nodes. Register each sequence in the symbol
   table with its own scope.
2. For each sequence, walk its `sequenceItem` children:
   - `stageDeclaration`: register stage in parent sequence's scope.
   - `sequenceDeclaration`: register nested sequence in parent sequence's scope and
     recurse.
3. For each stage, walk its `stageItem` children looking for `sequenceDeclaration`.
   Register and recurse.

The symbol scope hierarchy becomes nested (sequence > stage > sequence > ...). Each
sequence creates a scope. Name resolution via `scope.Resolve()` naturally walks up the
tree, giving us the `=> name` scoping behavior.

Name uniqueness is enforced per-scope (siblings in the same sequence cannot share a
name). Names may shadow names in enclosing scopes. The current global uniqueness
restriction is relaxed.

## 9.1 - Step Classification

The second analyzer pass walks sequence bodies and classifies each item into a step:

1. `flowStatement` or `singleInvocation`: produces `ir.Step{Flow: &ir.Flow{...}}`. The
   compiler auto-wires the flow's truthy output to an entry node for the next step via a
   one-shot edge. This is the same pattern as today's `condition => next` in a stage,
   but generated automatically.
2. `stageDeclaration`: produces `ir.Step{Stage: &ir.Stage{...}}`. Analyzed recursively
   for sub-sequences in the stage body.
3. `sequenceDeclaration`: produces `ir.Step{Sequence: &ir.Sequence{...}}`. Analyzed
   recursively.

## 9.2 - Key Generator Stack

The current `keyGenerator` has flat fields (`seqName`, `stageName`, `nextStageName`).
With nesting, it becomes a stack:

```go
type contextFrame struct {
    seqName     string
    stepKey     string
    nextStepKey string
}

type keyGenerator struct {
    occurrences map[string]int
    stack       []contextFrame
}
```

Push a frame when entering a sequence, pop when leaving. The `entry()` method and `next`
token resolution use the top frame. This generalizes cleanly to arbitrary nesting depth.

## 9.3 - Flow Steps and `=>` in Sequences

Flow steps in sequences always auto-wire to the next step. If the user writes an
explicit `=> next`, it is allowed silently (redundant but not harmful). If the user
writes `=> name` (a conditional jump to a named block), it is a compile-time error.
Conditional jumps require a `stage {}` wrapper for concurrent monitoring.

## 9.4 - Validation

- `=> next` on the last step: compile-time error (same as today, checked via
  `nextStepKey == ""` on the top stack frame).
- `=> name` in a flow step inside a sequence: compile-time error. Use a stage for
  conditional jumps.
- `=> next` explicit in a flow step: allowed silently (auto-wiring produces the same
  result).

## 9.5 - Strata Computation

Strata are computed per execution context:

1. **Global strata** (unchanged): top-level nodes + entry nodes with global inputs.
2. **Per-stage strata**: the stage's reactive flow nodes, stratified by data
   dependencies. Inline sub-sequences appear as execution context boundaries (synthetic
   keys) positioned by definition order.
3. **Per-sequence strata**: all flow step nodes across all flow steps in the sequence,
   stratified together. Stage and sequence steps appear as execution context boundaries.

Flow steps do not get their own strata. Their nodes are part of the parent sequence's
strata. The scheduler uses `activeStepIdx` to filter which flow nodes execute on a given
tick.

Execution context boundaries use synthetic keys (e.g., `boundary_{seqKey}` or
`boundary_{stageKey}`) inserted into the strata at the position determined by definition
order. The `Strata` representation stays `[][]string`. The scheduler maintains a
`map[string]*sequenceState` (or equivalent) that maps boundary keys to child context
runtime state. During strata execution, each key is checked against this map. If found,
it's a boundary and the scheduler enters the child context. If not, it's a regular node.
This avoids changing the strata type, serialization format, or C++ runtime strata
handling.

# 10 - Deferred Work

## 10.0 - Sequence Completion Wiring

When an inline sequence inside a stage completes, it goes idle. Wiring completion to
outgoing edges (e.g., `sequence { ... } => next`) is deferred. The syntax for attaching
edges to execution context boundaries needs design work.

## 10.1 - Looping

Sequences run once. To loop, use `=> name` from within a stage. A dedicated loop
construct can be added without breaking changes.

## 10.2 - Early Exit From Sequences

A bare `condition => name` as a sequential step does not make sense. The user likely
wants concurrent monitoring, which is what stages are for:

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

Gates in sequences only advance; they do not jump.

# 11 - Implementation Strategy

Ship as a single change. The unified strata tree model, execution context boundaries,
step-based sequences, and arbitrary nesting are implemented together.

Existing programs compile to the same step-based IR (every stage becomes a step of kind
`stage`), so backwards compatibility is verified by running all existing tests against
the new model.

### Implementation order:

1. **IR types**: Replace `Stage` with `Flow`/`Stage`/`Sequence`/`Step` tagged union. Add
   `Strata` to `Sequence`. Update msgpack codecs.
2. **Parser/Grammar**: Add `SequenceItem` alternatives, optional identifiers.
3. **Analyzer**: Handle mixed sequence bodies, nested scoping, step classification.
4. **Stratifier**: Compute strata tree with execution context boundaries.
5. **Scheduler**: Replace flat stage list with step-based tree walker, execution context
   boundary handling, write cascading, convergence with nested contexts.
6. **Tests**: Cover all step kinds, nesting combinations, convergence, reset, scoping.

# 12 - Examples

## 12.0 - Before and After: Valve Timing

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

## 12.1 - Before and After: Full TPC Sequence

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

The `hold` stage collapses into two bare lines. Stages with real reactive logic remain.

## 12.2 - Mixed Sequential and Reactive

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

The `tpc` stage has a sequence block (authority ramp-up) running alongside reactive
control and an exit condition. When the exit fires, the parent sequence continues with
cleanup.
