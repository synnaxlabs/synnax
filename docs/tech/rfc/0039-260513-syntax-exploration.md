# Arc Function Syntax Exploration

Scratchwork comparing two competing surface designs for reactive functions in Arc, plus
a third "surface" that is actually a downstream goal: a 2D graphical representation of
the same automation. Each scenario is shown three ways so the trade-offs and the
text↔graph round-trip are both visible.

## The two text surfaces

- **Style A: Pure function-passing (Patrick).** No trigger concept anywhere. A reactive
  function is a function that _returns_ a function of the wire-bound value; captures are
  bound by partial application; what flows on a wire is a function value.

  ```go
  func my_tally(threshold f32) func(u8) bool {
      return func(sample u8) { return f32(sample) > threshold }
  }
  concrete = my_tally(threshold=5)
  sample_ch -> concrete
  // or, inline:
  sample_ch -> my_tally(threshold=5)
  ```

- **Style B: Triggers as a first-class language concept.** A function declares which
  input is the wire-bound trigger; reactivity is its own syntax, not function
  application. Two equivalent grammars (same IR, same `Trigger` metadata, pure ergonomic
  choice):

  ```go
  // Canonical: brace-block for captures, parens for the trigger.
  func my_tally{threshold f32} (sample f32) u8 { return sample > threshold }

  // Alt: leading {wire} -> clause names the trigger explicitly.
  func {sample f32} -> my_tally(threshold f32) u8 { return sample > threshold }

  // Either way:
  sample_ch -> my_tally(threshold=5)
  ```

## The graphical surface (Style C)

Long-term, Arc automations should be viewable and editable as a 2D node graph,
LabVIEW-style: functions are boxes, channels are wires, captures are typed labels on
input ports. The goal is _round-trippable_ — text ↔ graph with no loss — so test
engineers can flip between views.

The working hypothesis: **Style B maps onto this view trivially**, because the trigger
annotation tells the renderer which input port is the "wire-attachment" side of the box.
Style A makes this implicit (it's "whatever the returned function's first parameter
is"), which forces the graph renderer to introspect the function's return type to figure
out what's a port vs. what's a captured value. If that hypothesis holds, Style B wins as
a tiebreaker even if A and B score equally on textual ergonomics.

Style C is not in the elimination round. It is **evidence** that helps decide between A
and B: how naturally does each text surface convert to a graph the end-user can read?

## Meta-question: lowering (only if both A and B survive)

Separate from "which text surface wins" is a compiler question: _what does the chosen
surface lower to internally?_ If B wins and we still want to expose function values as a
power-user escape hatch, B's surface can lower to A's IR; the audience writes in the
trigger frame, sophisticated users drop to A. Deployment decision, not a fourth column.
Moot if one of A or B is eliminated.

---

## Scenario 1: Threshold tally

> Function takes a captured threshold and a wire-fed sample. Emits 1 when the sample
> exceeds the threshold, 0 otherwise. The smallest possible reactive-function example.
> Originally the motivating sketch for this exploration.

### 1.A Function-passing

```go
func my_tally(threshold f32) func(u8) u8 {
    return func(sample u8) u8 {
        if f32(sample) > threshold { return 1 }
        return 0
    }
}

sample_ch -> my_tally(threshold=5) -> tally_out
```

### 1.B Triggers

```go
// Canonical: brace-block for the capture, parens for the trigger.
func my_tally{threshold f32} (sample u8) u8 {
    if f32(sample) > threshold { return 1 }
    return 0
}

// Alt grammar: leading {wire} -> clause names the trigger explicitly.
func {sample u8} -> my_tally(threshold f32) u8 {
    if f32(sample) > threshold { return 1 }
    return 0
}

// Either way, the call site is identical:
sample_ch -> my_tally(threshold=5) -> tally_out
```

---

## Scenario 2: Average 3 sensor readings, update on any new reading

> Three independent sensor channels feed a single fusion node. Any new sample on any of
> the three triggers a re-computation of the rolling average. The classic "multi-trigger
> fan-in" shape; the analyzer has to handle a node with more than one wire-bound input.

### 2.A Function-passing

```go
func my_avg(s1 f32, s2 f32, s3 f32) f32 {
    return (s1 + s2 + s3) / 3
}

// `react_on_any` invokes the function with latest values whenever any source
// fires. Not in today's stdlib; other fire-modes (`all`, mixed) need more.
react_on_any(my_avg, [sample_a_ch, sample_b_ch, sample_c_ch]) -> avg_ch

// Fire-mode is a boolean tree passed as data to a generic combinator.
// Leaves are channels, positionally mapped to the function's params.
// Structural: analyzer can validate it, renderer can draw it.
react_when(my_avg, OR(s_ch1, s_ch2, s_ch3)) -> avg_ch              // OR
react_when(my_avg, AND(s_ch1, s_ch2, s_ch3)) -> avg_ch             // AND
react_when(my_avg, OR(AND(s_ch1, s_ch2), s_ch3)) -> avg_ch         // (s1 AND s2) OR s3

// Same tree, but with the channel-to-param binding made explicit on the
// call-site arrow (boolean references param names, not channels).
{s1: s_ch1, s2: s_ch2, s3: s_ch3} -> react_when(my_avg, OR(AND(s1, s2), s3)) -> avg_ch
```

### 2.B Triggers

```go
// As triggers: fire-mode declared in the leading brace (`|` any, `&` all,
// mixes allowed). Locked at decl; call site just attaches channels.
func {s1 f32 | s2 f32 | s3 f32} -> my_avg() f32 { return (s1 + s2 + s3) / 3 }
{s1: s_ch1, s2: s_ch2, s3: s_ch3} -> my_avg() -> avg_ch

// As inputs (TriggerOnly): no trigger machinery; trigger sources merged at
// the call site. Flexible but redundant if values overlap with sources.
func my_avg_2(s1 f32, s2 f32, s3 f32) f32 { return (s1 + s2 + s3) / 3 }
{s_ch1 | s_ch2 | s_ch3 | any_thing} -> my_avg_2(s_ch1, s_ch2, s_ch3) -> avg_ch

// Explicit TriggerOnly
func {} -> my_avg_3(s1 f32, s2 f32, s3 f32) f32 { return (s1 + s2 + s3) / 3 }
{s_ch1 | s_ch2 | s_ch3 | any_thing} -> my_avg_3(s_ch1, s_ch2, s_ch3) -> avg_ch
```

**Takeaway**: B locks fire-mode into the multi-trigger function's signature, so the
graphical box owns its terminal labels and the call site is plain binding. A keeps the
function generic and wraps it in a `react_when` boolean at each call site; text can
approach B's compactness, but A always pays an extra wrapper box graphically.
TriggerOnly functions (implicit, or explicit with `func {} ->`) defer fire-mode to the
call site under B, letting multiple channels merge into the single trigger.

---

## Scenario 3: Three parallel ops, advance on completion (range creation)

> Three independent operations run in parallel. When all three have completed (each
> emits a "done" signal), the flow advances to the next phase, which creates a new
> range. Tests the "barrier" or "fan-in synchronizer" pattern as well as side-effecting
> calls like `range.create`.

- OP 1: Lox Press
- OP 2: Fuel Press
- OP 3: Battery Charging
- When all done, begin OP: Terminal_Count

```go
/// Shared code

sequence lox_press {
   stage press {
      0 -> lox_vent
      1 -> lox_press_valve
      lox_pt > l_threshold => next
      time.interval{1s} -> check_lox_abort() => abort
   }
   stage done {
      status.set("Lox Press", "Done", "success")
      1 -> lox_done
      0 -> lox_vent
      0 -> lox_press_valve
   }
   stage abort {
      status.set("Lox Press", "Abort", "error")
      1 -> lox_vent
      0 -> lox_press_valve
   }

}

sequence fuel_press {
   stage press {
      0 -> fuel_vent
      1 -> fuel_press_valve
      fuel_pt > f_threshold => next
      time.interval{1s} -> check_fuel_abort() => abort
   }
   stage done {
      status.set("Fuel Press", "Done", "success")
      1 -> fuel_done
      0 -> fuel_vent
      0 -> fuel_press_valve
   }
   stage abort {
      status.set("Fuel Press", "Abort", "error")
      1 -> fuel_vent
      0 -> fuel_press_valve
   }

}

sequence battery_charge {
   stage press {
      0 -> b_discharge
      1 -> b_charge
      b_volt > v_threshold => next
      time.interval{1s} -> check_batt_abort() => abort
   }
   stage done {
      status.set("Battery Charge", "Done", "success")
      1 -> batt_done
      0 -> b_discharge
      0 -> b_charge
   }
   stage abort {
      status.set("Battery Charge", "Abort", "error")
      1 -> b_discharge
      0 -> b_charge
   }

}
```

### 3.A Function-passing

```go
// `evaluate_conditions` follows A's function-passing pattern: outer returns
// the inner closure. Inner's `u8` input is the wire-bound trigger value from
// `react_when` (unused in the body here, but the signature has to take it).
func evaluate_conditions() func(u8) u8 {
   return func(trigger u8) u8 {
      if some_conditions() {
         return 1
      } else {
         return 0
      }
   }
}

start_all_command => lox_press
start_all_command => fuel_press
start_all_command => battery_charge

// `react_when` is not a built-in — A must define it. Sketched here for 3.A
// (handler is `func(u8) u8`); the fully generic version (any handler
// signature, BoolExpr leaves carrying bindings) needs variadic generics
// Arc does not have.
func react_when<T>(handler func(u8) T, mode BoolExpr) reactive<T> {
   // subscribe to channels in `mode`; on each fire, if `mode` evaluates true,
   // emit handler(trigger_value)
   ...
}

// Barrier: AND of three done signals lives in `react_when`'s boolean
// argument. `evaluate_conditions()` returns the inner closure that runs when
// all three fire; the closure's return value gates the transition.
react_when(evaluate_conditions(), AND(lox_done, fuel_done, batt_done)) => start_terminal_count
```

### 3.B Triggers

```go
func {} -> evaluate_conditions() u8 {
   if some_conditions() {
      return 1
   }
   else {
      return 0
   }
}

// Option 1: Current way to kick off all
start_all_command => lox_press
start_all_command => fuel_press
start_all_command => battery_charge
// Option 2: Hypothetical combined way
start_all_command => {lox_press, fuel_press, battery_charge}

// Wait for signal to start terminal count
{lox_done & fuel_done & batt_done} -> evaluate_conditions{} => start_terminal_count


```

---

## Scenario 4: Trigger on `pressure > threshold` OR 2 hours elapsed

> Whichever of the two conditions fires first wins; the second is canceled or ignored.
> Mixes a sample-driven predicate with a wall-clock timeout. Tests the
> "OR-of-asynchronous-events" pattern and forces the two syntactic styles to reckon with
> `time.wait` as an event source vs. a capture.

### 4.A Function-passing

```go
// Pressure predicate: outer captures threshold, inner takes the sample.
func is_above(threshold f32) func(f32) u8 {
   return func(p f32) u8 {
      return p > threshold
   }
}

// Stateful timer predicate. `$=` state cell lives in the closure.
func has_elapsed(duration i64) func(u8) u8 {
   start i64 $= 0
   return func(_ u8) u8 {
      if (start == 0) {
         start = time.now()
         return 0
      }
      return (time.now() - start) >= duration
   }
}

// Pass-through handler for `react_when`'s required slot.
func always_fire() func(u8) u8 {
   return func(_ u8) u8 { return 1 }
}

// Polled: tick at 1Hz, evaluate the stateful timer.
time.interval{1s} -> has_elapsed(duration=2h) -> timer_done_ch
react_when(always_fire(), OR(pressure_ch > p_threshold, timer_done_ch)) => next_phase

// Stdlib one-shot: `time.wait(2h)` is the event source. No polling, no state cell.
react_when(always_fire(), OR(pressure_ch > p_threshold, time.wait(2h))) => next_phase
```

### 4.B Triggers

```go
// Stateful polled timer. First call latches `start`; subsequent calls
// compare elapsed time against the window.
func get_time_passed() u8 {
    start i64 $= 0
    if (start == 0) {
        start = time.now()
        return 0
    }
    passed := time.now() - start
    return passed < 5400s
}

time.interval{1s} -> {pressure_ch > p_threshold | get_time_passed(100s)} => next_phase
```

---

## Scenario 5: Stateful accumulator

> Counter that increments per wire-delivered sample and emits the running total.
> Exercises per-instance state: where does the cell live, and what's the allocation
> model? Natural under B's stateful primitives, awkward under A.

### 5.A Function-passing

```go
// State cell `total` lives in the outer's closure. Each `tally()` invocation
// allocates a fresh cell; two wires sharing one closure share one cell.
func tally() func(i64) i64 {
   total i64 $= 0
   return func(sample i64) i64 {
      total = total + sample
      return total
   }
}

sample_ch -> tally() -> total_ch
```

### 5.B Triggers

```go
// State cell `total` is per-instance. Each call site is an independent
// instance with its own cell.
func {sample i64} -> tally() i64 {
   total i64 $= 0
   total = total + sample
   return total
}

sample_ch -> tally() -> total_ch
```

---

## Scenario 6: Stdlib `ExecBoth` (`status.set`)

> Per the parent RFC, `status.set` must round-trip between a wire sink
> (`-> status.set{...}`) and an in-expression call (`status.set("...", "...", "...")`).
> The motivating use case for the unification work; tests whether each style preserves
> the dual-call shape.

### 6.A Function-passing

```go
// In-expression call: same as B.
status.set("aborted", "aborted", "error")

// Wire-sink form: A has no native "trigger calls function" syntax — wrap
// in a closure that takes the wire value (unused) and side-effects.
func sink_status(name str, message str, variant str) func(u8) u8 {
   return func(_ u8) u8 {
      status.set(name, message, variant)
      return 0
   }
}

error_ch -> sink_status(name="aborted", message="aborted", variant="error")
```

### 6.B Triggers

```go
// Same function, two call forms — unified params per the parent RFC.

// Wire-sink (brace form):
error_ch -> status.set{name="aborted", message="aborted", variant="error"}

// In-expression call (parens form):
status.set("aborted", "aborted", "error")
```

**Takeaway**: B's unified-params model makes `status.set` symmetric — same symbol, two
call forms, no glue. A's wire-sink form needs an explicit closure adapter because
function-passing has no native "wire value into side-effect" pattern; the in-expression
form is unaffected. The motivating RFC use case round-trips trivially under B and
asymmetrically under A.

---

## Scenario 7: Debounce / throttle

> Forward a sample only if at least N seconds have passed since the last forward.
> Combines state (last-emit timestamp), a clock source, and a gating decision.

### 7.A Function-passing

```go
// Stateful gating closure. Sample-and-hold: only updates `last_val` when
// the interval has elapsed, otherwise re-emits the last accepted sample.
func debounce(interval i64) func(f32) f32 {
   last_emit i64 $= 0
   last_val f32 $= 0
   return func(sample f32) f32 {
      if (time.now() - last_emit) >= interval {
         last_emit = time.now()
         last_val = sample
      }
      return last_val
   }
}

sample_ch -> debounce(1s) -> sample_out_ch
```

### 7.B Triggers

```go
// Same gating logic; sample-and-hold so the return type is plain `f32`.
func {sample f32} -> debounce(interval i64) f32 {
   last_emit i64 $= 0
   last_val f32 $= 0
   if (time.now() - last_emit) >= interval {
      last_emit = time.now()
      last_val = sample
   }
   return last_val
}

sample_ch -> debounce(interval=1s) -> sample_out_ch
```

---

## Scenario 8: Sequencing / step machine

> Linear pipeline of side-effecting steps:
> `range.start -> wait -> set_valve -> range.end`. The shape that motivated the RFC's
> stage/sequence discussion. Sequences are shared infrastructure; the only A-vs-B
> difference is the surface form of reactive bits inside stages (e.g. `time.wait`).

### 8.A Function-passing

```go
// `time.wait(30s)` is a function call returning a reactive value firing once.
sequence test_run {
   stage init {
      range.start("test_run_42")
      => wait
   }
   stage wait {
      time.wait(30s) => actuate
   }
   stage actuate {
      1 -> valve_open
      time.wait(5s) => close
   }
   stage close {
      0 -> valve_open
      range.end("test_run_42")
   }
}
```

### 8.B Triggers

```go
// `time.wait{30s}` is a brace-capture construct; output is the firing channel.
sequence test_run {
   stage init {
      range.start("test_run_42")
      => wait
   }
   stage wait {
      time.wait{30s} => actuate
   }
   stage actuate {
      1 -> valve_open
      time.wait{5s} => close
   }
   stage close {
      0 -> valve_open
      range.end("test_run_42")
   }
}
```

---

## Discussion

Open questions to resolve as we fill in the scenarios:

1. **Does Style A require first-class function types and closures in Arc?** Neither
   exists today. What does `func(u8) bool` look like in the surface grammar, and how
   does the IR represent a closure that captures a stack variable? Knock-on effects on
   the codec and the WASM ABI.
2. **Partial application semantics under A.** Is `my_tally(threshold=5)` a value of
   function type, or a node with one input pre-bound? The first adds a value kind to the
   type system; the second is a syntactic rearrangement of what B already does.
3. **Where does state live under A?** If a wire just carries a function value, the
   runtime needs somewhere to put the per-instance state cell that today's `stateful.*`
   symbols rely on. Closures over a state variable work in theory but require allocation
   semantics Arc does not currently have.
4. **What does the RFC lose if triggers are dropped entirely?** Today `Trigger` metadata
   drives analyzer behavior: which input gets the wire-bound value, which params are
   bind-time vs call-time, which calls are legal as flow sinks. Under A all of that has
   to be recovered from the function's type signature alone.
5. **Does Style B map to the 2D graph more cleanly than Style A?** Working hypothesis:
   yes, because triggers tell the renderer where the wire attaches without introspecting
   return types. If the C sections bear that out, it's a tiebreaker for B even if A and
   B look comparable in text.
6. **If both A and B survive: do we lower one to the other?** Only worth asking if the
   scenarios don't eliminate one outright. If B's surface lowers to A's IR, we keep the
   trigger-frame syntax for the audience while gaining the power-user escape hatch. If
   the lowering adds compiler complexity without buying anything, drop it.

---

## Important consideration: can function-passing be drawn at all?

Technically yes, but the representation hides what makes function-passing
function-passing.

**Single-trigger case.** A call like `sample_ch -> my_tally(threshold=5)` has two boxes
structurally: the higher-order `my_tally` (input `threshold`, output a function value),
and the returned closure (input `sample`, output bool). To draw this faithfully you'd
need an edge between the two boxes carrying _a function_, not data. Dataflow graphs
(LabVIEW, Simulink, Node-RED) don't really do that — edges carry values that flow, not
behavior that gets installed. In practice you'd collapse the two boxes into one
"configured tally" node with both `threshold` and `sample` as input ports. Which is
fine, except that collapsed view is literally Style B's view of the same computation.
The graph throws away the partial-application structure and renders the trigger-shape.

**Multi-trigger case.** This is where it gets worse. Take a 3-channel rolling average
where any channel update should re-fire:

- Under B, `avg` is a box with three input ports `a`, `b`, `c`; `Trigger` metadata says
  "any of these can be wire-bound." The renderer just draws three input ports and lets
  the user attach wires to whichever ones are active.
- Under A, `avg` is `func(f32, f32, f32) f32`. To get "fire on any new sample from any
  port" you need a separate combinator, call it `react_on_any(avg, [a, b, c])`, that
  subscribes to three channels and invokes `avg` whenever any fires. The graph either
  renders that combinator as an extra box (clutter that doesn't exist in the code's
  mental model) or inlines it away (lose round-trip fidelity).

The trigger metadata in B is exactly the annotation the graph renderer needs. Under A,
you have to reconstruct that annotation from external context (the combinator, or
convention about argument position) every time you want to draw the box.

**So:**

- A _can_ be drawn, but only by rendering it as B would be rendered.
- The text↔graph round-trip is lossy under A: drawing the graph involves picking a
  "canonical" combinator pattern, and going back from graph to text involves choosing
  which closure-shape to emit. Either direction adds editorial decisions the user didn't
  write.
- B's round-trip is mechanical: each box is one function-call site; each port is one
  parameter; the trigger annotation says which port is wire-bound.

This is the multi-trigger future feature the parent RFC already opens the door for.
Under B it's a `Trigger` metadata extension. Under A it's a new combinator stdlib, and
the graph view becomes a question of which combinator to draw.

In practice, authoring will likely happen in text; the graph view is more plausible as
an operational read-out used during an op or a test, with drill-down to text for any
edit.

Drill-down fidelity is asymmetric. Under A, a "box" the user sees in the graph was
synthesized from a higher-order function plus a closure, so clicking in shows a
different structural shape than the box implied. Under B, the box maps 1:1 to the
function declaration. For a high-level monitoring view that's friction; if the graph
ever becomes an authoring surface, it's a blocker.

All of this assumes a hypothetical, well-developed graph UI that doesn't exist today.
