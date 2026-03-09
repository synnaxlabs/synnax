# RFC: Arc Runtime Timer Resolution

## Problem

The Arc runtime loop ticks at the `base_interval`, which is the GCD of all timer
periods in the program. Time-based nodes (Wait, Interval) only advance their timers on
`TimerTick` cycles. This creates a fundamental timing resolution problem for Wait nodes.

### Example

```
load_current > 50 => wait{duration=2s} => next
```

- `base_interval = GCD(2s) = 2s`. Loop ticks at t=0, 2, 4, 6...
- `load_current > 50` fires at t=1s via a channel input. Wait records `start_time = 1s`
- Wait needs `elapsed - start_time >= duration - tolerance` (i.e., `>= ~2s`)
- t=2s (TimerTick): `2 - 1 = 1s < 2s`. Not fired.
- t=4s (TimerTick): `4 - 1 = 3s >= 2s`. Fired. **Actual wait was 3s, not 2s.**

Worst-case overshoot is nearly one full `base_interval`.

### Why Interval nodes are unaffected

Interval nodes are periodic and self-correcting. The GCD-based tick rate guarantees
ticks land on exact multiples of every interval period. Intervals don't start at
arbitrary times, so the phase alignment problem doesn't exist.

### Why Wait nodes are affected

Wait is a one-shot timer that starts at an arbitrary point in time (when a condition
fires, when a stage is entered, etc.). Its start time has no relationship to the loop's
tick phase. The timer can only be checked on TimerTick cycles, so the granularity of
those ticks directly bounds the timing accuracy.

### When it matters and when it doesn't

| Scenario | base_interval | Max overshoot | Problem? |
|---|---|---|---|
| `wait{2s}` alone | 2s | ~2s | Severe |
| `wait{3s}` + `wait{5s}` | 1s | ~1s | Noticeable |
| `interval{100ms}` + `wait{5s}` | 100ms | ~100ms | Acceptable |
| `interval{100ms}` + `wait{2s}` | 100ms | ~100ms | Acceptable |
| `wait{100ms}` + `wait{200ms}` | 100ms | ~100ms | Acceptable |
| Only intervals, no waits | GCD of periods | N/A | No |

Programs with fast intervals naturally get fine ticks. Programs with only waits (or
waits whose GCD is large) suffer.

### Go vs C++ tolerance difference

The Go runtime masks this problem with a generous tolerance of `base_interval / 2`. For
a 2s wait, tolerance = 1s, so the wait fires when `elapsed - start >= 2s - 1s = 1s`.
This means a "2s wait" can fire after just 1s, which is a 50% timing error.

The C++ runtime has a tighter tolerance (capped at 5ms for EVENT_DRIVEN mode), which
correctly demands ~2s of elapsed time but then can't achieve it with 2s tick resolution.

Neither behavior is correct. The Go tolerance is too loose. The C++ tick rate is too
coarse.

## Current architecture

### How base_interval is computed

`time::Factory` computes `base_interval` as the GCD of all Wait durations and Interval
periods:

- C++: `arc/cpp/runtime/time/time.h` (Factory::update_base_interval)
- Go: `arc/go/runtime/time/time.go` (Factory.updateBaseInterval)

### How base_interval becomes the loop tick rate

- C++: `loop::Config::apply_defaults(base_interval)` in `arc/cpp/runtime/loop/loop.h`
  sets `cfg.interval = base_interval` when no explicit interval is configured
- Go: The ticker runtime in `core/pkg/service/arc/runtime/task.go` uses
  `timeFactory.BaseInterval` as its ticker interval

### How timer ticks flow through the system

1. The loop wakes on timer or pipe input
2. The first scheduler cycle per wake gets `RunReason::TimerTick`; subsequent queued
   frames get `RunReason::ChannelInput`
3. Wait/Interval nodes only advance their timers on `TimerTick`
4. Between timer ticks, channel inputs keep the scheduler running (via `selfChanged`)
   but Wait returns early because `reason != TimerTick`

### Relevant files

| File | Role |
|---|---|
| `arc/cpp/runtime/time/time.h` | C++ Wait, Interval, Factory |
| `arc/go/runtime/time/time.go` | Go Wait, Interval, Factory |
| `arc/cpp/runtime/loop/loop.h` | Loop config, apply_defaults, execution modes |
| `arc/cpp/runtime/loop/loop_darwin.cpp` | macOS loop (kqueue) |
| `arc/cpp/runtime/loop/loop_linux.cpp` | Linux loop (epoll/timerfd) |
| `arc/cpp/runtime/loop/loop_windows.cpp` | Windows loop |
| `arc/cpp/runtime/loop/loop_polling.cpp` | Fallback polling loop |
| `arc/cpp/runtime/runtime.h` | C++ runtime main loop |
| `arc/cpp/runtime/scheduler/scheduler.h` | C++ scheduler |
| `arc/go/runtime/scheduler/scheduler.go` | Go scheduler |
| `core/pkg/service/arc/runtime/task.go` | Go runtime task (ticker interval) |

## How other systems solve this

**libuv / Node.js**: Deadline-driven. No fixed tick. Computes
`min(next_timer_deadline - now)` as the OS sleep duration. Timers fire precisely when
their deadline arrives.

**Tokio (Rust)**: Hierarchical timer wheel with 1ms base resolution. The runtime sleeps
until the next timer expiry. No fixed tick.

**PLC runtimes**: Fixed scan cycle (1-100ms). Timer resolution equals scan time. Closest
model to Arc's current approach.

**Linux kernel**: Originally hierarchical timing wheel with fixed tick (HZ). Modern
kernels use `hrtimers` backed by hardware interrupts scheduled for the exact next expiry,
plus tickless mode when no timers are pending.

The consensus across modern runtimes is **deadline-driven scheduling**: sleep until the
next thing needs to happen, not on a fixed cadence.

## Proposed solution: deadline-driven timer scheduling

Instead of periodic ticks at `base_interval`, the loop should sleep until the earliest
active timer deadline. This eliminates the tick-phase alignment problem entirely.

### Design

1. **Node interface**: Add `next_deadline() -> optional<TimeSpan>` returning the
   absolute elapsed time at which this node needs its next TimerTick. Returns nullopt
   when the node has no pending deadline (fired, inactive, or not a time node).

2. **Wait.next_deadline()**: Returns `start_time + duration` when active and unfired.
   Returns nullopt when fired or not yet started.

3. **Interval.next_deadline()**: Returns `last_fired + period`.

4. **Scheduler.next_deadline()**: Iterates all nodes in active strata and returns the
   minimum deadline across all of them. Returns nullopt when no timers are active.

5. **Loop.wait(max_timeout)**: The `wait()` method gains an optional max timeout
   parameter. Internally uses `min(periodic_timer_remaining, max_timeout)` in the OS
   wait call. The periodic timer can be kept for backward compatibility or removed
   entirely.

6. **Runtime loop**: After each `scheduler.next()` call, computes
   `next_deadline - elapsed` and passes it to `loop.wait()` as the max timeout.

### Why this works with existing OS primitives

The loop implementations already block on OS calls that accept timeout parameters:

- macOS: `kevent(kq, ..., &timeout)` accepts a `timespec` timeout
- Linux: `epoll_wait(epfd, ..., timeout_ms)` accepts a millisecond timeout
- Windows: `WaitForMultipleObjects(..., timeout_ms)` accepts a millisecond timeout

No new OS primitives are needed. The change is: instead of passing a fixed timeout
derived from `base_interval`, pass `min(existing_timeout, next_deadline - now)`.

### What changes per file

| File | Change |
|---|---|
| `arc/cpp/runtime/node/node.h` | Add `virtual optional<TimeSpan> next_deadline()` with default nullopt |
| `arc/cpp/runtime/time/time.h` | Implement `next_deadline()` for Wait and Interval |
| `arc/cpp/runtime/scheduler/scheduler.h` | Add `next_deadline()` method iterating active nodes |
| `arc/cpp/runtime/loop/loop.h` | Add max_timeout parameter to `wait()` |
| `arc/cpp/runtime/loop/loop_darwin.cpp` | Use min(timer, max_timeout) in kevent |
| `arc/cpp/runtime/loop/loop_linux.cpp` | Use min(timer, max_timeout) in epoll_wait |
| `arc/cpp/runtime/loop/loop_windows.cpp` | Use min(timer, max_timeout) in WaitForMultipleObjects |
| `arc/cpp/runtime/loop/loop_polling.cpp` | Use min(timer, max_timeout) in sleep |
| `arc/cpp/runtime/runtime.h` | Compute deadline, pass to wait() |
| Go equivalents | Mirror the same pattern |

Estimated scope: ~100-150 lines of logic across both languages.

### Edge cases

- **No active timers**: `next_deadline()` returns nullopt. Loop falls back to existing
  behavior (periodic timer or EVENT_DRIVEN timeout).
- **Stage transitions**: Deadlines change when stages activate/deactivate. The runtime
  recalculates after each `scheduler.next()` call, which includes stage transitions.
- **Multiple waits**: `scheduler.next_deadline()` returns the minimum across all active
  waits. When the earliest fires, the next cycle recalculates with the remaining waits.
- **Wait + Interval**: Both report deadlines. The minimum is used. Interval's periodic
  nature is preserved because it keeps reporting `last_fired + period`.
- **base_interval**: Can be removed from the timing path entirely, or kept as a fallback
  when no nodes report deadlines.

### Tolerance

With deadline-driven scheduling, tolerance becomes what it should be: compensation for
OS scheduling jitter, not compensation for coarse tick alignment.

- C++: `calculate_tolerance` can use small values (5ms for EVENT_DRIVEN, 100us for
  RT_EVENT) because the tick granularity is no longer a limiting factor
- Go: `CalculateTolerance` should be tightened from `baseInterval / 2` to a small fixed
  value (e.g., 5ms), since the generous tolerance was masking this bug

### What about the selfChanged mechanism?

The `selfChanged` mechanism (added in this branch) remains necessary. It ensures that
Wait/Interval nodes in higher strata (behind one-shot edges) continue to be executed by
the scheduler on subsequent cycles. Without it, the scheduler wouldn't even call
`next()` on the Wait node after the one-shot fires.

The deadline-driven approach complements `selfChanged`: `selfChanged` ensures the node
is *executed*, and the deadline ensures the loop *wakes up* at the right time to execute
it.
