# Sequence Stratification Design

This document describes the stratification model for Arc sequences, addressing
fundamental issues with the original design and defining the correct execution semantics.

## Problem Statement

The original stratification model treated sequences as extensions of the global dataflow
graph, creating implicit dependencies between stage entry nodes and stage-internal nodes.
This caused two fundamental problems:

### Problem 1: Stage-Local Sources

Consider this sequence:

```arc
sequence main {
    stage press {
        1 -> press_vlv_cmd,
        press_pt -> gte{100} => next
    }
}
```

The nodes `1` (constant) and `press_pt` (channel source) are **source nodes** - they have
no inputs and should fire every cycle while active.

The original model created an implicit dependency:

```
entry_main_press (stratum 0)
       |
       v  (implicit dependency)
const_1, press_pt (stratum 1)
       |
       v
write_vlv, gte (stratum 2)
```

This pushed stage sources to stratum 1+, meaning they only execute when "marked as
changed" rather than every cycle. The runtime worked around this with hacks like
`mark_stage_root_nodes_changed()`, artificially marking nodes as changed each cycle.

### Problem 2: Cyclic Stage Transitions

Consider a sequence with a loop:

```arc
sequence main {
    stage first {
        0 -> valve_cmd => second
    }
    stage second {
        1 -> valve_cmd => first
    }
}
```

The original model included transition edges in the global dependency graph:

```
entry_first -> nodes_in_first -> entry_second
                                      |
entry_first <-------------------------|
     ^                                |
     |--- nodes_in_second <-----------|
```

This creates a cycle: `entry_first -> ... -> entry_second -> ... -> entry_first`. The
stratification algorithm detects this as an error, but this is a perfectly valid state
machine - stages are mutually exclusive, so there's no actual circular dataflow.

## Design: Per-Stage Stratification

The solution separates stratification into two tiers:

1. **Global strata**: Flows outside any sequence
2. **Per-stage strata**: Each stage has its own independent stratification

### Entry Nodes Are Sinks, Not Sources

Entry nodes remain real nodes in the graph. They have meaningful stratums. However:

- Entry nodes are **sinks** (downstream targets), not sources
- Stage-internal nodes do **not** depend on their own stage's entry node
- Entry nodes receive their stratum from their **incoming edges**

### Example Stratification

For this program:

```arc
start_cmd => main

sequence main {
    stage press {
        1 -> press_vlv_cmd,
        press_pt -> gte{100} => next
    }
    stage hold {
        wait{5s} => press
    }
}
```

**Global strata:**
```
[0] start_cmd (channel source)
[1] entry_main_press (sink - receives activation)
```

**Stage `press` strata:**
```
[0] const_1, press_pt (stage-local sources)
[1] write_vlv_cmd, gte
[2] entry_main_hold (sink - receives transition)
```

**Stage `hold` strata:**
```
[0] wait (timer source)
[1] entry_main_press (sink - receives loop-back)
```

Note that `entry_main_press` appears in multiple contexts:
- In global strata at stratum 1 (downstream of `start_cmd`)
- In `hold`'s strata at stratum 1 (downstream of `wait`)

But it does **not** appear in `press`'s own strata as a dependency. Nodes in `press` are
stratum 0 within their stage.

## Execution Model

### Execution Order Per `Next()` Call

1. **Execute global strata** (in stratum order)
2. **For each active sequence** (in definition order):
   - Execute the active stage's strata (in stratum order)
   - Collect any stage transitions that fired
3. **Convergence loop**:
   - For each fired transition: deactivate old stage, activate new stage
   - Execute newly activated stage's strata
   - Repeat until no transitions fire (or iteration limit reached)

### Convergence Loop Termination

The convergence loop handles immediate transitions (conditions already true when stage
activates). To prevent infinite loops:

**Compile-time detection**: Catch obvious infinite loops like unconditional transitions:
```arc
stage first { true => second }
stage second { true => first }
```

**Runtime limit**: If transitions exceed `number_of_stages + 1` iterations in a single
`Next()` call, stop iterating and report an error. The sequence remains in its current
stage.

## Design Decisions

### Separate Nodes Per Stage

When the same channel appears in multiple stages:

```arc
stage press { press_pt -> gte{100} => next }
stage hold { press_pt -> lt{90} => press }
```

The compiler creates separate `on` nodes for each appearance:
- `on_press_pt_0` in press stage
- `on_press_pt_1` in hold stage

This fits naturally with per-stage stratification - each stage is self-contained with its
own nodes and strata.

### Multiple Sequences

When multiple sequences are active simultaneously, they execute in **definition order**.
This provides deterministic behavior when sequences interact via shared channels.

### Terminal Stages

A terminal stage (no outgoing transitions) stays active indefinitely:

```arc
stage final {
    1 -> valve_cmd  // continuous output, no transition
}
```

The sequence remains running in `final` until externally deactivated. This allows
terminal stages to maintain continuous outputs.

### Transition Priority

When multiple transition conditions become true simultaneously, the **first defined**
wins. This matches the existing spec and allows users to control priority through
ordering:

```arc
stage press {
    pressure > max => abort,      // highest priority
    hold_btn => hold,
    pressure > target => next,
    wait{30s} => abort            // lowest priority
}
```

### Oneshot Tracking

Oneshot edges fire once per stage activation. Tracking is per-sequence: when a stage
transition occurs, the sequence's fired-oneshot set is cleared. This allows oneshots to
fire again when re-entering a stage.

### Node Reset on Stage Entry

When entering a stage, all nodes owned by that stage reset:
- Timers restart from zero
- Stateful variables reinitialize
- Oneshot states clear

Global nodes (outside any sequence) never reset.

## IR Representation Changes

### Stage Structure

Each stage stores its own stratification:

```go
type Stage struct {
    Key    string   `json:"key"`
    Nodes  []string `json:"nodes"`
    Strata Strata   `json:"strata"`  // per-stage stratification
}
```

### IR Structure

The IR separates global and per-stage strata:

```go
type IR struct {
    // ... existing fields ...
    GlobalStrata Strata    `json:"global_strata"`  // flows outside sequences
    Sequences    Sequences `json:"sequences"`       // each stage has its own strata
}
```

### Stratification Algorithm Changes

The stratifier no longer adds implicit dependencies from entry nodes to stage nodes.
Instead:

1. Compute global strata for nodes not in any stage
2. For each stage, compute strata independently:
   - Stage-local sources (channels, constants) are stratum 0
   - Entry nodes of **other** stages can be targets (sinks)
   - No dependency on the stage's own entry node

## Migration Notes

### Breaking Changes

- The `IR.Strata` field is replaced by `IR.GlobalStrata` and per-stage `Stage.Strata`
- Scheduler must be updated to handle two-tier execution
- Entry nodes no longer create implicit upstream dependencies

### Compatibility

The external behavior remains the same:
- Sequences execute stages in order
- Transitions work as specified
- Oneshots fire once per stage activation

The change is to internal stratification, eliminating workarounds and enabling correct
handling of cyclic state machines.
