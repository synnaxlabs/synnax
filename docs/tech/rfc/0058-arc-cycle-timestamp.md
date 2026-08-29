# 58 Arc cycle timestamp

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-25
- **Related**:
  [RFC 0031 - Arc scheduler stage transition semantics](0031-arc-scheduler-semantics.md)

## 0 Summary

Arc has no shared time reference inside a scheduler pass. Seven producers in the Go
runtime and nine in C++ each own a private `telem.MonoClock` and stamp at the moment
they happen to run, and each of those clocks bumps by 1 ns to keep its own output
unique. Telemetry produced by one cycle therefore carries several close but distinct
timestamps, and no consumer can tell which samples belong together.

This RFC gives a scheduler pass one timestamp. The runtime loop reads the wall clock
once per pass through a single `telem.MonoClock` and passes the result down as a value.
Every producer stamps from it. The sixteen clocks collapse to one per runtime.

The same stamp closes a defect. Arc appends one index sample per data write
(`arc/go/stl/channels/state.go:174`), so a program writing two channels that share an
index emits two index stamps against one sample per data channel, which cesium rejects
(`sameLengthForAllSeriesError`, `cesium/writer_stream.go:850`), killing the task. Index
stamping moves from write time to flush time: one index series per group per cycle,
which is the shape the Driver already writes (`generate_index_data`,
`driver/common/sample_clock.h:219`).

## 1 Vocabulary

- **Cycle**: One `Scheduler.Next` call, including every settle pass it runs internally.
- **Cycle stamp**: The single wall-clock timestamp read at the top of a cycle.
- **Index group**: The channels in one writer that share an index channel, plus that
  index. Cesium requires every write call to carry every member of a group, with equal
  sample counts (`idxWriter.validateWrite`, `cesium/writer_stream.go:822`).
- **Provenance stamp**: An index timestamp carried in from upstream data rather than
  read from a clock.

## 2 Motivation

### 2.0 Sixteen clocks

Every Arc producer that needs a timestamp owns a clock. In Go: `constant`
(`constant.go:67`), the variable register and expression reader (`variable.go:82`,
`:115`), the WASM node (`wasm/node.go:56`), the time module (`time/time.go:125`), the
channel program state (`channels/state.go:35`), and the channel source
(`channels/channels.go:194`). C++ mirrors all seven and adds `stable`
(`stable/stable.h:61`).

Each is a `telem.MonoClock`, which exists to guarantee that consecutive reads never
return the same value, bumping by 1 ns when the platform clock is too coarse
(`x/go/telem/mono_clock.go:18`). Sixteen independent clocks each enforcing local
uniqueness produce exactly the outcome the guarantee was meant to prevent: a cycle that
writes three channels stamps them at three different instants, none of which is the time
of the cycle.

The pattern arrived by copying, not by design. No RFC introduced it, nothing documents
or enforces it, and no conformer benefits from it. It is an incident, and this RFC
replaces it everywhere.

### 2.1 Two channels on one index kill the task

`writeIndexedTimestamp` appends one index sample for every data sample written
(`arc/go/stl/channels/state.go:174`). A program that writes `valve_a` and `valve_b`,
both indexed by `cmd_time`, produces two `cmd_time` samples against one sample each.
Cesium requires equal lengths across an index group and rejects the frame. Writing only
one of the two fails the other way, on group completeness (`missingChannelError`,
`cesium/writer_stream.go:860`). There is no way to write two channels on a shared index
from Arc today.

### 2.2 Prior art

Inside Synnax, the Driver already answered both halves of this.
`SoftwareTimedSampleClock` reads `TimeStamp::now()` once at the start of each
acquisition cycle and interpolates the samples in between
(`driver/common/sample_clock.h:57`). `generate_index_data` builds one index series and
writes it to every index key in the frame (`:219`). Arc is the outlier, not the
innovator.

Outside, IEC 61131-3 runtimes sample the task time base once per scan and hold it
constant for the whole scan, and measure jitter against the scheduled cycle rather than
the wake. Ignition's tag historian stamps each tag at its own execution; grouping a scan
class into one coherent row is a standing user request rather than a feature.

## 3 Principles

1. **A cycle has one time**: Every producer in a pass stamps from the same read.
   Uniqueness within a channel is a property of the flush, not of sixteen clocks racing.
2. **The stamp is the wall clock, sampled per pass**: Not a scheduled deadline and not a
   free-running anchor. See §6.0 and §6.1.
3. **Timing is an input, not a dependency**: The runtime loop reads the clock and passes
   the value down, exactly as it already does for `Elapsed`. The scheduler samples
   nothing.
4. **Provenance beats the clock**: A write carrying real upstream timestamps keeps them.
   The cycle stamp fills an index only when nothing else did.
5. **Cadence fidelity belongs to the loop**: SY-4693 anchors the loop period to its
   deadline (`x/cpp/loop/loop.h:65`) and SY-4694 gave Windows a high-resolution waitable
   timer. Timestamps report what happened; the loop is what makes it happen on time.

## 4 Design

### 4.0 `node.Cycle`

The three per-cycle scalars become one value, built by the caller:

```go
// Cycle carries the timing shared by every producer in one scheduler pass.
type Cycle struct {
    // Now is the wall clock read once at the top of the pass. Every producer
    // without an upstream timestamp stamps from it.
    Now telem.TimeStamp
    // Elapsed is the time since the runtime started.
    Elapsed telem.TimeSpan
    // Reason indicates what triggered this pass.
    Reason RunReason
}
```

`Context` embeds `Cycle`, so `ctx.Elapsed` and `ctx.Reason` keep working unchanged at
every existing read site and `ctx.Now` joins them. `Scheduler.Next(ctx, cycle)` replaces
`Scheduler.Next(ctx, elapsed, reason)`, and the C++ `Scheduler::next(cycle)` mirrors it.

### 4.1 One clock per runtime loop

Each loop that drives a scheduler gains one `telem.MonoClock` beside the `startTime` it
already holds, and builds the `Cycle` itself:

- `dataRuntime.next` (`core/pkg/service/arc/task/task.go:491`)
- `Runtime::run` (`arc/cpp/runtime/runtime.h:128`), which also gains the wall-clock read
  it currently lacks entirely, having only `steady_clock`
- `Calculator.Next`
  (`core/pkg/service/framer/calculation/calculator/calculator.go:238`), whose settle
  loop treats each `Scheduler.Next` as its own cycle

`MonoClock` is kept, and it is the only one left. It guarantees the stamp never
regresses, so a backwards clock step cannot walk an index backwards or land a sample
before the writer's `Start`, which cesium requires (`validateCommitRange`,
`cesium/internal/domain/writer.go:364`).

Every per-node clock field is deleted, along with the `stable.WithNow` option
(`arc/go/stl/stable/stable.go:96`) and the C++ `NowFunc` constructor parameters. A test
that wants deterministic time passes the `Cycle` it wants.

### 4.2 `Reset` takes the context

The variable register stamps its restored initial value on scope entry
(`arc/go/stl/variable/variable.go:90`), which happens inside a cycle but outside `Next`.
`Reset()` becomes `Reset(ctx Context)` in both runtimes so it reads the same stamp as
everything else. Roughly twelve Go implementations and twenty-two C++ implementations
change signature; nearly all ignore the parameter.

One call site is not inside a cycle: C++ `Scheduler::reset()` runs from `Runtime::stop`
(`arc/cpp/runtime/scheduler/scheduler.h:206`). It passes a zeroed cycle with no-op
callbacks, which is correct: no cycle has run.

### 4.3 Index stamping moves to flush

`writeIndexedTimestamp` is deleted. The data write path records only data. At flush, for
each index group that was written this cycle:

- If the group's index buffer is **empty**, synthesize one index series of `n` samples
  starting at the cycle stamp, spaced 1 ns apart, where `n` is the group's per-channel
  sample count.
- If the index buffer is **not empty**, something supplied provenance stamps. Leave it
  alone.

`ProgramState.Flush` gains the stamp: `Flush(fr, now)`. The 1 ns spacing matches what
the Core already does for auto-index ("subsequent samples in the same write are spaced 1
nanosecond apart", `docs/site/src/pages/reference/client/advanced/auto-index.mdx`) and
what the channel source already synthesizes (`telem.Arrange(clock.Now(), n, 1ns)`,
`arc/go/stl/channels/channels.go:293`).

RFC 0031 §3.1 preserves every intermediate write in a cycle, so `n` can exceed one when
stage transitions cascade. A cycle emitting `n` samples spans `n` nanoseconds: one
cycle, one timestamp base. Cross-channel coherence holds because every member of the
group starts at the same stamp.

Flush reports the highest stamp it emitted and the loop's clock resumes above it.
Without this, a fast loop on a coarse platform clock could return `T+1` on the next
cycle after emitting `T` through `T+9`, and the index would walk backwards.

### 4.4 Provenance passes through

Graph programs write through the channel sink, which passes its input's time series into
the index (`arc/go/stl/channels/channels.go:280`). This is unchanged and load bearing: a
calculated channel inherits its source samples' timestamps, so cycle-stamping it would
misdate anything not arriving live. Text programs write through the WASM host functions,
supply no time, and get cycle stamps.

Where a single group mixes the two, the sample counts do not line up and the write
errors. See §6.2.

### 4.5 `now()`

The `now()` builtin compiles to the `time.now` host binding
(`arc/go/stl/time/time.go:139`), which now returns the cycle stamp. Two calls in one
cycle return the same value, which is what the builtin should always have meant. The
runtime pushes the cycle into the time module before each pass; the module is
constructed only in the Arc task runtime and the Driver, not in calculations.

The language reference (`arc/docs/spec.md:285`) is updated to state that `now()` is
constant within a cycle.

## 5 Implementation phases

### 5.0 Phase 1: The cycle stamp

`node.Cycle`, `Context` embedding, `Scheduler.Next(ctx, cycle)`, `Reset(ctx)`, one
`MonoClock` per runtime loop, `stable.WithNow` and the C++ `NowFunc` parameters removed,
`now()` reading the cycle stamp, spec updated. Every node-side clock is deleted:
`constant`, both variable nodes, the WASM node, the time module, `stable`, and the
channel source. Go and C++ together, since RFC 0031 §3.2 makes runtime parity an
invariant.

The one clock this phase leaves alone is `channels.ProgramState`'s, because Phase 2
deletes it outright rather than converting it. Pushing the cycle stamp into the write
path first would build machinery that Phase 2 immediately removes.

This phase is green on its own and changes no persisted shape: producers that stamped at
slightly different instants now stamp at one.

### 5.1 Phase 2: Index coalescing

`writeIndexedTimestamp` and the last `MonoClock` in `channels.ProgramState` deleted,
`Flush(fr, now)` synthesizing one index series per group, the flush reporting its
highest stamp, and the loops advancing their clocks past it. Both runtimes.

The split earns its boundary on risk isolation. Phase 1 changes which instant a stamp
reads; Phase 2 changes how many samples reach the index. Landing them together would
leave a bisect pointing at one diff for two unrelated failure modes.

## 6 Resolved decisions

**6.0 The stamp is the wall clock, not the scheduled deadline.** Stamping the deadline
gives an even cadence, and that is its problem: when the loop runs late the record still
claims it ran on time, which is the wrong failure mode for a post-incident trace. The
deadline is also the minimum across every timer node (`scheduler.go:149`), so in a
program with a 100 ms and a 30 ms interval it is not the cadence of either one, and it
is undefined on channel-input passes, requiring a second regime inside one program. The
Driver's free-running clock is only sound because a clamped PID steers it back to the
system clock (`HardwareTimedSampleClock`, `driver/common/sample_clock.h:143`); Arc has
no hardware clock to free-run on. The trade is real: consecutive cycles are unevenly
spaced, and on a stepped clock two adjacent cycles can land 1 ns apart. That is a
faithful record of a jittery machine.

**6.1 Not a start anchor plus monotonic elapsed.** Reading the wall clock once at start
and adding monotonic elapsed never regresses and never jumps, but it never gets
corrected either, so it drifts at the crystal's rate. Every other writer in the system
stamps with the corrected wall clock, including cesium's own auto-index
(`writer_stream.go:340`). An Arc task running for a day would put its command channel
seconds away from the sensor channel it was reacting to, silently. Cross-source
alignment is what the index is for. The trade is real: the wall clock can step, which is
why the single read goes through `MonoClock`.

**6.2 Partial and unequal index groups keep erroring.** A cycle that writes only some
members of a group, or writes one member more often than another, still fails cesium's
completeness and equal-length checks. Filling the gaps by repeating each channel's last
value was considered and rejected for this pass: it writes values the program did not
produce, and on a command channel it re-issues a command. Holding partial writes until
the group completes stalls indefinitely when a channel is written conditionally and
delivers stale values under a later cycle's stamp. Rejecting such programs at compile
time needs proof that every group member is written on every path, which conditional
writes make impossible, and conditional writes are the normal way to write Arc.

The consequence is stated plainly: today the failure is deterministic, because two
channels on one index always kill the task. After this change it is conditional.
Programs that write their group unconditionally are fixed. Programs with conditional
writes fail at the first cycle that writes only one member, which is later and harder to
reproduce than failing immediately. §7.0 carries the open question.

**6.3 The runtime loop owns the clock, not the scheduler.** Putting the clock in the
scheduler would serve all three callers from one implementation, but it would give the
scheduler a hidden dependency and force tests to inject through it. `Elapsed` already
establishes that per-cycle timing is the caller's input. The trade is real: three loops
each call `clock.Now()` instead of one place doing it.

**6.4 `driver::common::SampleClock` is not reused.** It is the closest analog and it
does not fit. `driver/common` depends on the C++ client and the Driver's task and
pipeline, and the Driver depends on Arc, so reuse would invert the dependency. Its
`wait()` also rate-limits at a fixed rate, which Arc cannot use: Arc wakes on channel
input as well as timers, and recomputes its deadline every cycle from the program's
nodes. Removing `wait()` from `SoftwareTimedSampleClock` leaves `TimeStamp::now()`. The
primitives worth sharing already live in `x/`: `telem::MonoClock`, `telem::NowFunc`, and
`x::loop`.

**6.5 `Reset` gains the context rather than one node keeping a clock pointer.**
Injecting a shared cycle pointer into the variable node through `node.Config` avoids
about thirty-four mechanical signature changes, but leaves two ways to read one value
and keeps a clock field on the one node type, which is the thing this RFC ends.

## 7 Open questions

**7.0 Partial index groups.** §6.2 defers the policy. The choice is between repeating
each channel's last value to complete the row, rejecting such programs when the task
starts (`deps.Writes` is known at open, so a shared index among written channels is
exactly detectable), or leaving the runtime error. It needs its own issue.

**7.1 Shared index generation with the Driver.** The Driver's `generate_index_data` and
Arc's new synthesis do the same job in C++ with different inputs: the Driver
interpolates between a start and an end over a known sample count, Arc has one stamp and
a per-channel count. They are candidates to merge if the shapes converge. Go needs its
own regardless.

## 8 What this RFC does not cover

**8.0 Calculation groups sharing an index.** Two calculators in one calculation group
writing channels that share an index are broken today, and this RFC does not fix them.
Each calculator owns its own `ProgramState`, so each appends its own index series into
the shared frame and cesium rejects the second (`oneSeriesPerChannelError`,
`cesium/writer_stream.go:846`). No single flush sees the group, so §4.3 cannot reach it.
It needs its own issue.

**8.1 Loop cadence.** SY-4693 and SY-4694 landed the deadline anchor and the Windows
high-resolution timer. This RFC changes what a timestamp says, never when a cycle runs.

**8.2 Index channels the program writes explicitly.** A program that writes a timestamp
channel by name supplies its own stamps and is unaffected by §4.3, which only fills an
empty index buffer.
