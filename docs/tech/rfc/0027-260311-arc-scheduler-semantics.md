# 27 - Arc Scheduler Stage Transition Semantics

**Feature Name**: Arc Scheduler Stage Transition Semantics <br /> **Status**: Draft <br />
**Start Date**: 2026-03-11 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

The Arc scheduler has three interacting problems that produce incorrect behavior when
multiple stage transitions can fire in the same execution cycle. These problems surface
clearly in the `arc_short_circuit` integration test, which verifies that when multiple
transition conditions are true simultaneously, the first-written transition takes
priority.

The three problems are:

1. **Stratification ordering determines transition priority instead of source order.**
   The short-circuit mechanism picks the transition at the shallowest stratum, but
   stratum depth is determined by chain length, not source position. Shorter chains fire
   first regardless of where they appear in the source.

2. **The convergence loop combined with append write semantics produces duplicate
   writes.** Within a single `scheduler.Next()` call, the convergence loop can execute
   multiple stages sequentially. With append semantics, writes from all intermediate
   stages accumulate in a single flush.

3. **Go and C++ interval reset behavior diverges.** C++ `Interval::reset()` sets
   `last_fired = -interval`, causing immediate firing on re-entry. Go `Interval` has no
   `Reset()` override (the base `state.Node.Reset()` is a no-op), so `lastFired` retains
   its old value. This causes different execution traces for the same program.

# 1 - Vocabulary

- **Stratum** - A layer in the topological execution order. Stratum 0 contains source
  nodes (no incoming edges). Higher strata contain nodes that depend on lower ones.
- **Strata** - The complete set of strata for a stage or global scope.
- **Convergence loop** - The `execStages()` loop that re-executes stage strata after
  transitions until no further transitions occur (stable state).
- **Short-circuit** - The `transitioned` flag mechanism that stops executing further
  nodes in a strata pass once a transition fires.
- **Append semantics** - The branch's change from `writes[key] = data` (replace) to
  `appendWriteSeries(key, data)` (accumulate), preserving writes across stage
  transitions within a single flush cycle.

# 2 - Problem Analysis

## 2.0 - Motivating Example

The `arc_short_circuit` integration test uses this program:

```arc
sequence main {
    stage on {
        "on" -> ss_stage_str,
        count{c_chan = ss_count_on},
        0 -> ss_sim_stage,
        1 -> ss_heater_cmd,
        interval{period=1s} -> (ss_temp_a > 290 and ss_temp_b > 290) => off,
        interval{period=1s} -> ss_temp_b > 300 => pause,
    }
    stage pause {
        "pause" -> ss_stage_str,
        count{c_chan = ss_count_pause},
        2 -> ss_sim_stage,
        0 -> ss_heater_cmd,
        wait{duration=1s} => on,
    }
    stage off {
        "off" -> ss_stage_str,
        3 -> ss_sim_stage,
        0 -> ss_heater_cmd,
        ss_start_cmd => on,
    }
}
```

**Phase 1**: `ss_temp_a=200, ss_temp_b=400`. Only `ss_temp_b > 300` is true, so the
program loops between `on` and `pause` every second.

**Phase 2**: `ss_temp_a=400`. Now both conditions are true. The expected behavior is that
`=> off` wins because it appears first in source order.

The test expects this write sequence:
```
ss_stage_str: ["on", "pause", "on", "pause", "on", "pause", "on", "off"]
```

## 2.1 - Problem 1: Stratification Ordering vs. Source Order

### Mechanism

The stratification algorithm (`stratifier.go:stratifySubgraph`) assigns each node a
stratum equal to `max(source_strata) + 1`. Nodes with no incoming edges start at
stratum 0. This means a node's stratum depth is determined entirely by the length of
its longest dependency chain.

In the `on` stage, the two transition chains have different depths:

```
Chain 1 (=> off):   interval -> gt_a -> gt_b -> AND -> entry_off     (4 edges)
Chain 2 (=> pause): interval -> gt_b_300 -> entry_pause              (2 edges)
```

The `entry_pause` node ends up at stratum ~2, while `entry_off` ends up at stratum ~4.

### How the Short-Circuit Interacts

The short-circuit mechanism in `execStrata` (`scheduler.go:259-278`) executes nodes
layer by layer from stratum 0 upward. When a node calls `transitionStage()`, it sets
`transitioned = true`, and `execStrata` returns immediately:

```go
func (s *Scheduler) execStrata(strata ir.Strata) {
    clear(s.changed)
    s.transitioned = false
    for i, stratum := range strata {
        for _, key := range stratum {
            // ...
            s.nodes[key].Next(s.nodeCtx)
            if inStage && s.transitioned {
                return  // First transition wins
            }
        }
    }
}
```

Since `entry_pause` is at a shallower stratum than `entry_off`, `=> pause` fires first
and short-circuits `=> off`. The transition that wins is determined by chain length, not
by source order.

### Expected vs. Actual Behavior

| Scenario | Expected | Actual |
|----------|----------|--------|
| Both conditions true | `=> off` (source order) | `=> pause` (shorter chain) |

### Consequence

Source order has no semantic meaning for transition priority. The user cannot reason
about which transition will fire first by reading the code top-to-bottom.

## 2.2 - Problem 2: Convergence Loop and Duplicate Writes

### Mechanism

The `execStages()` convergence loop (`scheduler.go:282-300`) re-executes stages after
transitions until stable:

```go
func (s *Scheduler) execStages() {
    for iter := 0; iter < s.maxConvergenceIterations; iter++ {
        stable := true
        for s.currSeqIdx = 0; s.currSeqIdx < len(s.sequences); s.currSeqIdx++ {
            seq := &s.sequences[s.currSeqIdx]
            if seq.activeStageIdx == -1 { continue }
            s.currStageIdx = seq.activeStageIdx
            s.execStrata(seq.stages[s.currStageIdx].strata)
            if seq.activeStageIdx != s.currStageIdx {
                stable = false
            }
        }
        if stable { return }
    }
}
```

When `transitionStage()` fires, it resets the target stage's nodes via `resetStrata`,
which calls `Reset()` on every node. Constants get `initialized = false`, allowing them
to re-fire. The convergence loop then executes the new stage's strata, where those
constants fire and produce writes.

### Interaction with Append Semantics

The branch changed write behavior from replace to append:

```go
// Before (replace): last write wins
s.channel.writes[key] = data

// After (append): all writes accumulate
s.appendWriteSeries(key, data)
```

Within a single `scheduler.Next()` call:

1. `on` stage executes. Constants fire: `"on"` -> ss_stage_str, `0` -> ss_sim_stage,
   `1` -> ss_heater_cmd. Interval fires, condition met, `=> pause` transition.
2. Convergence loop: `pause` stage now active. Constants reset and fire: `"pause"` ->
   ss_stage_str, `2` -> ss_sim_stage, `0` -> ss_heater_cmd.
3. All writes from both stages accumulate in the write buffer.
4. On flush, `ss_stage_str` contains `["on", "pause"]` instead of just one value.

### Consequence

A single flush cycle can emit writes from multiple stages, producing unexpected
duplicate values that break downstream consumers expecting clean, single-valued writes
per cycle.

## 2.3 - Problem 3: Go/C++ Interval Reset Divergence

### Mechanism

The C++ `Interval` class overrides `reset()` to make the interval fire immediately on
the next TimerTick after stage re-entry:

```cpp
// arc/cpp/runtime/time/time.h:87
void reset() override { last_fired = -1 * cfg.interval; }
```

The Go `Interval` struct has **no** `Reset()` override. The embedded
`state.Node.Reset()` is a no-op:

```go
// arc/go/runtime/state/state.go:347
func (n *Node) Reset() {}
```

| | Go `Interval` | C++ `Interval` |
|---|---|---|
| Construction | `lastFired: -period` | `last_fired: -interval` |
| `Reset()` | No-op (retains old `lastFired`) | `last_fired = -interval` |
| After stage re-entry | Fires only when wall-clock time elapses past `period` | Fires immediately on next TimerTick |

### Consequence

The same Arc program produces different execution traces on Go (rack 65537) and C++
(rack 65538). Within a convergence loop iteration where elapsed time hasn't changed:

- **C++**: Interval fires immediately after reset, potentially causing an additional
  transition and more accumulated writes.
- **Go**: Interval does not fire (elapsed - lastFired = 0 < period), so the convergence
  loop stabilizes sooner.

This is a correctness bug. Both runtimes should produce identical behavior for the same
program.

# 3 - Proposed Solutions

## 3.0 - Source-Order Transition Priority

The stratification algorithm should not determine transition priority. Instead, when
multiple transitions can fire in the same strata execution, the scheduler should respect
source order.

**Option A: Collect-and-pick.** Instead of short-circuiting on the first transition,
execute all nodes in the strata, collect all transitions that would fire, and pick the
one with the lowest source position.

**Option B: Flatten transition entries to the same stratum.** Modify the stratifier to
place all entry nodes for the same stage at the same stratum depth (the maximum of their
natural depths). This preserves topological correctness while ensuring transitions
compete at the same level, where source order within a stratum determines priority.

**Option C: Annotate edges with source position.** Add a `priority` or `source_index`
field to transition edges. The scheduler uses this to break ties when multiple
transitions fire.

## 3.1 - Write Semantics During Convergence

Define clear semantics for what writes are visible after a convergence loop.

**Option A: Only final stage writes survive.** On transition, clear the write buffer
before executing the target stage. Only the stable (final) stage's writes are flushed.

**Option B: All intermediate writes are preserved (current append behavior).** Accept
that a single flush can contain writes from multiple stages. Document this as intentional
and adjust downstream consumers.

**Option C: Writes are committed per-stage.** Each stage's writes are flushed
independently before transitioning to the next stage. This gives downstream consumers a
clean, ordered sequence but requires multiple flush calls per cycle.

## 3.2 - Interval Reset Parity

Add a `Reset()` override to Go's `Interval` that matches C++:

```go
func (i *Interval) Reset() {
    i.lastFired = -i.period
}
```

This is a straightforward bug fix independent of the other two problems.

# 4 - Impact

These problems affect any Arc program with multiple competing transitions in the same
stage. Programs with only a single transition per stage are unaffected.

The interval reset divergence affects any Arc program using `interval{}` nodes across
stage transitions, regardless of whether multiple transitions compete. The Go runtime
will exhibit different timing behavior than the C++ runtime.

# 5 - Key Files

| File | Role |
|---|---|
| `arc/go/runtime/scheduler/scheduler.go` | Go scheduler, convergence loop, short-circuit |
| `arc/cpp/runtime/scheduler/scheduler.h` | C++ scheduler (mirrors Go) |
| `arc/go/stratifier/stratifier.go` | Stratification algorithm |
| `arc/go/runtime/time/time.go` | Go interval/wait (missing Reset) |
| `arc/cpp/runtime/time/time.h` | C++ interval/wait (has reset) |
| `arc/go/runtime/state/state.go` | Go write buffer, append semantics |
| `arc/cpp/runtime/state/state.cpp` | C++ write buffer (mirrors Go) |
| `arc/go/runtime/constant/constant.go` | Go constant node (has Reset) |
| `integration/tests/arc/arc_short_circuit.py` | Integration test exercising all three problems |
