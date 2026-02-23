# Bug: Asymmetric Authority Reclaim on Stage Re-entry (C++ Runtime)

## Summary

When an ARC bang-bang controller transitions through `start → stop → yield → start`
(re-entry), the C++ runtime fails to reclaim authority on one of two channels. The first
channel in the stage's `set_authority` list does not re-fire on re-entry, while the
second channel correctly reclaims. The Go runtime handles this correctly.

## Customer Report

Reported by Yale (Jason, John) during a call on 2026-02-22. They observed:

- A bang-bang controller with two valves (`press_vlv_cmd`, `vent_vlv_cmd`) using
  separate `high_bang` and `low_bang` function blocks
- On stop → yield → start re-entry (when `bb_start_cmd` stays active), **Valve 1**
  (press/high_bang) would "instantaneously release control and then be reclaimed," but
  **Valve 2** (vent/low_bang) correctly released
- Swapping both valves to use the same function block made the bug disappear
- The bug was consistent and deterministic

## Reproduction

**Test file**: `integration/tests/arc/arc_bang_bang_authority.py`

**Key conditions to reproduce**:

1. Two separate function blocks (`high_bang`, `low_bang`) driving two different channels
2. Per-channel `set_authority` in both `start` (value=220) and `yield` (value=0) stages
3. Trigger stop **without clearing the start signal** — this causes yield to immediately
   re-enter start via `bb_start_cmd => start`
4. Run on the C++ runtime (rack_key 65539)

**What happens**:

- `press_vlv_cmd`: ARC fails to reclaim authority 220 on re-entry. External writer at
  authority 50 can take control.
- `vent_vlv_cmd`: ARC correctly reclaims authority 220. External writer at 50 is
  rejected.

**Test results** (4/4 runs consistent):

| Runtime          | rack_key | press_vlv_state | vent_vlv_state | Result |
| ---------------- | -------- | --------------- | -------------- | ------ |
| Go (server)      | 65538    | 1.0 (ARC)       | 1.0 (ARC)      | PASS   |
| C++ (driver)     | 65539    | 0.0 (external)  | 1.0 (ARC)      | FAIL   |

## ARC Source (Minimal Reproducer)

```arc
authority (press_vlv_cmd 210 vent_vlv_cmd 210)

func high_bang{
    sensor chan f32,
    set_point f32,
    lower_deadband f32,
    upper_deadband f32,
    abort_threshold f32
}() u8 {
    state $= 0
    if sensor > (set_point + upper_deadband) {
        state = 0
    } else if sensor < (set_point - lower_deadband) {
        state = 1
    }
    if sensor > abort_threshold {
        state = 0
    }
    return state
}

func low_bang{
    sensor chan f32,
    set_point f32,
    lower_deadband f32,
    upper_deadband f32,
    abort_threshold f32
}() u8 {
    state $= 0
    if sensor < (set_point - lower_deadband) {
        state = 1
    } else if sensor > (set_point + upper_deadband) {
        state = 0
    }
    if sensor > abort_threshold {
        state = 0
    }
    return state
}

sequence bang_bang_controller {
    stage start {
        set_authority{value=220, channel=press_vlv_cmd},
        set_authority{value=220, channel=vent_vlv_cmd},
        interval{period=200ms} -> high_bang{...} -> press_vlv_cmd,
        interval{period=200ms} -> low_bang{...} -> vent_vlv_cmd,
        bb_stop_cmd => stop
    }
    stage stop {
        0 -> press_vlv_cmd,
        0 -> vent_vlv_cmd,
        wait{duration=250ms} => yield
    }
    stage yield {
        set_authority{value=0, channel=press_vlv_cmd},
        set_authority{value=0, channel=vent_vlv_cmd},
        bb_start_cmd => start
    }
}

bb_start_cmd => bang_bang_controller
```

## Root Cause Analysis

The bug is in the C++ ARC runtime's handling of `set_authority` one-shot nodes during
stage re-entry within the scheduler's convergence loop.

`set_authority` nodes are **one-shot** — they fire once per stage activation via an
`initialized` flag, and `Reset()` clears the flag on stage entry. During the
yield → start convergence cascade:

1. Yield stage fires: `set_authority{value=0}` on both channels
2. `bb_start_cmd` trigger fires: transition to start
3. Start stage should fire: `set_authority{value=220}` on both channels

The Go runtime correctly resets and re-fires both `set_authority` nodes in step 3. The
C++ runtime only re-fires the second one (`vent_vlv_cmd`), leaving the first
(`press_vlv_cmd`) at authority 0.

### Likely location

The C++ ARC runtime's stage transition / one-shot reset logic. Key areas to investigate:

- `driver/arc/` — C++ runtime task and scheduler implementation
- One-shot node `Reset()` / `initialized` flag handling during convergence loop
  transitions
- Whether the first `set_authority` node's reset is being skipped or its authority
  change is being overwritten

## What Does NOT Trigger the Bug

- Using the same function block for both channels (both `high_bang` or both `low_bang`)
- Clearing `bb_start_cmd` before triggering stop (no yield → start re-entry)
- Running on the Go runtime
- Simple threshold functions without deadband parameters (bug is not parameter-related)

## Related Changes

- `err_on_unauthorized=false` fix (commit `09cd625cf8`) — this is a separate fix that
  prevents the acquisition pipeline from erroring on unauthorized writes. It does NOT
  fix this asymmetric reclaim bug. The bug reproduces with `err_on_unauthorized=false`.

## Files

- **Integration test**: `integration/tests/arc/arc_bang_bang_authority.py`
- **Test registry**: `integration/tests/arc_tests.json` (authority sequence)
- **C++ runtime**: `driver/arc/task.h`
- **Go runtime (working)**: `core/pkg/service/arc/runtime/`
- **Go unit tests**: `core/pkg/service/arc/runtime/task_test.go` (lines 981+)
- **Scheduler unit tests**: `arc/go/runtime/scheduler/scheduler_test.go`
