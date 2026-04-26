# 31 - Arc Scheduler Stage Transition Semantics

**Feature Name**: Arc Scheduler Stage Transition Semantics <br /> **Status**: Draft <br
/> **Start Date**: 2026-03-11 <br /> **Authors**: Emiliano Bonilla <br />

# 0 - Summary

In this RFC, we address three interacting problems in the Arc scheduler that produce
incorrect behavior when multiple stage transitions can fire in the same execution cycle.
These problems surface clearly in the `arc_short_circuit` integration test, which
verifies that when multiple transition conditions are true simultaneously, the
first-written transition takes priority.

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
- **Append semantics** - The change from `writes[key] = data` (replace) to
  `appendWriteSeries(key, data)` (accumulate), preserving writes across stage
  transitions within a single flush cycle.

# 2 - Motivation

## 2.0 - Motivating Example

The `arc_short_circuit` integration test uses the following program:

```arc
sequence main {
    stage on {
        "on" -> ss_stage_str
        count{c_chan=ss_count_on}
        0 -> ss_sim_stage
        1 -> ss_heater_cmd
        interval{period=1s} -> (ss_temp_a > 290 and ss_temp_b > 290) => off
        interval{period=1s} -> ss_temp_b > 300 => pause
    }
    stage pause {
        "pause" -> ss_stage_str
        count{c_chan=ss_count_pause}
        2 -> ss_sim_stage
        0 -> ss_heater_cmd
        wait{duration=1s} => on
    }
    stage off {
        "off" -> ss_stage_str
        3 -> ss_sim_stage
        0 -> ss_heater_cmd
        ss_start_cmd => on
    }
}
```

**Phase 1**: `ss_temp_a=200, ss_temp_b=400`. Only `ss_temp_b > 300` is true, so the
program loops between `on` and `pause` every second.

**Phase 2**: `ss_temp_a=400`. Now both conditions are true. The expected behavior is
that `=> off` wins because it appears first in source order.

The test expects this write sequence:

```
ss_stage_str: ["on", "pause", "on", "pause", "on", "pause", "on", "off"]
```

## 2.1 - Stratification Ordering vs. Source Order

The stratification algorithm (`stratifier.go:stratifySubgraph`) assigns each node a
stratum equal to `max(source_strata) + 1`. Nodes with no incoming edges start at
stratum 0. This means a node's stratum depth is determined entirely by the length of its
longest dependency chain.

In the `on` stage, the two transition chains have different depths:

```
Chain 1 (=> off):   interval -> gt_a -> gt_b -> AND -> entry_off     (4 edges)
Chain 2 (=> pause): interval -> gt_b_300 -> entry_pause              (2 edges)
```

The `entry_pause` node ends up at stratum ~2, while `entry_off` ends up at stratum ~4.

The short-circuit mechanism in `execStrata` executes nodes layer by layer from stratum 0
upward. When a node calls `transitionStage()`, it sets `transitioned = true`, and
`execStrata` returns immediately:

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
by source order. The user cannot reason about which transition will fire first by
reading the code top-to-bottom.

| Scenario             | Expected                | Actual                     |
| -------------------- | ----------------------- | -------------------------- |
| Both conditions true | `=> off` (source order) | `=> pause` (shorter chain) |

## 2.2 - Convergence Loop and Duplicate Writes

The `execStages()` convergence loop re-executes stages after transitions until stable:

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

With append semantics, all writes accumulate within a single `scheduler.Next()` call:

1. `on` stage executes. Constants fire: `"on"` -> ss_stage_str, `0` -> ss_sim_stage, `1`
   -> ss_heater_cmd. Interval fires, condition met, `=> pause` transition.
2. Convergence loop: `pause` stage now active. Constants reset and fire: `"pause"` ->
   ss_stage_str, `2` -> ss_sim_stage, `0` -> ss_heater_cmd.
3. All writes from both stages accumulate in the write buffer.
4. On flush, `ss_stage_str` contains `["on", "pause"]` instead of just one value.

A single flush cycle can emit writes from multiple stages, producing multiple values for
the same channel in one flush. This is intentional: if a channel is written on the way
out of a stage (`0 -> some_ch`) and again on entry to the next stage (`1 -> some_ch`),
both writes should be emitted in order. Silently dropping the outgoing write would hide
commands the operator explicitly requested. The responsibility for controlling write
flow belongs to the automation author, not the runtime.

## 2.3 - Go/C++ Interval Reset Divergence

The C++ `Interval` class overrides `reset()` to make the interval fire immediately on
the next TimerTick after stage re-entry:

```cpp
void reset() override { last_fired = -1 * cfg.interval; }
```

The Go `Interval` struct has no `Reset()` override. The embedded `state.Node.Reset()` is
a no-op:

```go
func (n *Node) Reset() {}
```

|                      | Go `Interval`                                         | C++ `Interval`                      |
| -------------------- | ----------------------------------------------------- | ----------------------------------- |
| Construction         | `lastFired: -period`                                  | `last_fired: -interval`             |
| `Reset()`            | No-op (retains old `lastFired`)                       | `last_fired = -interval`            |
| After stage re-entry | Fires only when wall-clock time elapses past `period` | Fires immediately on next TimerTick |

The same Arc program produces different execution traces on Go and C++. Within a
convergence loop iteration where elapsed time hasn't changed, C++ fires the interval
immediately after reset, while Go does not fire because
`elapsed - lastFired == 0 < period`. This is a correctness bug. Both runtimes must
produce identical behavior for the same program.

# 3 - Design

## 3.0 - Source-Order Transition Priority

The stratification algorithm should not determine transition priority. When multiple
transitions can fire in the same strata execution, the scheduler should respect source
order. We considered three approaches:

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

We chose **Option B**. Flattening entry nodes to the same stratum is the least invasive
change. It preserves the existing short-circuit mechanism and requires no changes to the
scheduler itself. The stratifier already has access to entry node metadata, so the
modification is localized to `stratifier.go`.

## 3.1 - Write Semantics During Convergence

We need clear semantics for what writes are visible after a convergence loop. We
considered three approaches:

**Option A: Only final stage writes survive.** On transition, clear the write buffer
before executing the target stage. Only the stable (final) stage's writes are flushed.

**Option B: All intermediate writes are preserved (current append behavior).** Accept
that a single flush can contain writes from multiple stages. Document this as
intentional and adjust downstream consumers.

**Option C: Writes are committed per-stage.** Each stage's writes are flushed
independently before transitioning to the next stage. This gives downstream consumers a
clean, ordered sequence but requires multiple flush calls per cycle.

We chose **Option B**. Preserving all intermediate writes is the correct semantic for
control systems. When a program transitions through multiple stages in a single cycle,
downstream consumers need the complete record of what happened. Discarding intermediate
writes (Option A) would hide state changes that may be safety-relevant. Per-stage
flushing (Option C) adds complexity to the flush pipeline without a clear benefit.

## 3.2 - Interval Reset Parity

This is a straightforward bug fix independent of the other two problems. We add a
`Reset()` override to Go's `Interval` that matches the existing C++ behavior:

```go
func (i *Interval) Reset() {
    i.Node.Reset()
    i.lastFired = -i.period
}
```

After this change, both runtimes fire the interval immediately on the first tick after
stage re-entry.

# 4 - Implementation

## 4.0 - Stratifier Changes

The stratifier (`arc/go/stratifier/stratifier.go`) is modified to detect entry nodes
within a stage's subgraph and promote them to the same stratum. After the initial
topological stratification, we find the maximum stratum among all entry nodes for the
current stage and move any shallower entry nodes up to that depth. This ensures the
short-circuit mechanism compares transitions at the same level, where source order
within the stratum determines priority.

## 4.1 - State Write Buffer

The state module (`arc/go/runtime/state/state.go`, `arc/cpp/runtime/state/state.cpp`)
uses append semantics for channel writes. When a channel is written to multiple times
within a single `scheduler.Next()` call (e.g., across stage transitions in the
convergence loop), all values accumulate in order. The flush operation returns the
complete series for each channel.

## 4.2 - Interval Reset

Both Go (`arc/go/runtime/time/time.go`) and C++ (`arc/cpp/runtime/time/time.h`) now
implement `Reset()` with identical behavior: setting `lastFired` (or `last_fired`) to
the negative of the period. This guarantees the interval fires on the first tick after
stage re-entry in both runtimes.

## 4.3 - Scheduler Changes

The scheduler (`arc/go/runtime/scheduler/scheduler.go`,
`arc/cpp/runtime/scheduler/scheduler.h`) handles string accumulation correctly during
convergence. Previously, string writes could accumulate across convergence iterations in
unexpected ways. The fix ensures string series are handled consistently with other data
types.

# 5 - Testing Strategy

## 5.0 - Unit Tests

- **Scheduler tests** (`arc/go/runtime/scheduler/scheduler_test.go`,
  `arc/cpp/runtime/scheduler/scheduler_test.cpp`): Verify convergence loop behavior,
  transition priority, and write accumulation across stage transitions.
- **State tests** (`arc/go/runtime/state/state_test.go`,
  `arc/cpp/runtime/state/state_test.cpp`): Verify append write semantics, flush
  behavior, and series accumulation.
- **Stratifier tests** (`arc/go/stratifier/stratifier_test.go`): Verify entry node
  promotion to the same stratum depth.
- **Series tests** (`x/cpp/telem/series_test.cpp`): Verify series append operations used
  by the state module.

## 5.1 - Integration Tests

The `arc_short_circuit` integration test (`integration/tests/arc/arc_short_circuit.py`)
exercises all three problems together. It runs the motivating example program and
verifies:

- The program loops between `on` and `pause` when only one condition is true (Phase 1).
- The `=> off` transition wins over `=> pause` when both conditions become true (Phase
  2), respecting source order.
- The complete write sequence for `ss_stage_str` matches the expected output.
- Write counts for `ss_count_on` and `ss_count_pause` are consistent across both
  runtimes.

# 6 - Impact

These problems affect any Arc program with multiple competing transitions in the same
stage. Programs with only a single transition per stage are unaffected.

The interval reset divergence affects any Arc program using `interval{}` nodes across
stage transitions, regardless of whether multiple transitions compete.
