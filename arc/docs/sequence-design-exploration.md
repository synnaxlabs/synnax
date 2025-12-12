# Arc Sequence Design Exploration

## Current State of Arc

Arc is a reactive/dataflow language that compiles to WebAssembly. Its current strengths:

- **Reactive execution model**: Functions fire when inputs arrive
- **Implicit concurrency**: Dataflow naturally parallelizes
- **Channel-based communication**: Data flows through typed channels
- **Primary use case**: Calculated channels for telemetry processing

### What Arc Does Well

```arc
// Continuous reactive processing
pressure_sensor -> filter{} -> calibrate{} -> output

// Parallel by default - both paths run concurrently
sensor -> path_a{} -> output_a
sensor -> path_b{} -> output_b

// Routing outputs to different targets
source -> func{} -> route {
    a_output = target_a,
    b_output = target_b
}
```

### What Arc Doesn't Handle

Sequential automation workflows:
- Rocket engine test sequences
- Semiconductor fabrication processes
- Quantum dilution refrigerator cooldowns
- Any process with ordered steps, holds, and abort conditions

---

## The Problem

Real-world automation sequences require:

1. **Ordered execution**: Step A must complete before Step B starts
2. **Parallel execution within steps**: Pressurize ox AND fuel tanks simultaneously
3. **Conditional branching**: If pressure > threshold, continue; else abort
4. **Holds**: Operator can pause at certain points, with step-specific hold behavior
5. **Aborts**: Safety conditions can interrupt and redirect to abort sequences
6. **Context-dependent responses**: A "hold" during pressurization means something different than during ignition

### The Core Tension

Arc's reactive model: "When data arrives, process it"
Sequential model: "Do A, then B, then C, and handle interrupts"

These seem opposed, but state machines are actually reactive under the hood - they continuously check conditions and decide which state to be in. The question is: can sequential semantics be expressed as a subset of reactive semantics with minimal syntax additions?

---

## Design Evolution

### Rejected Approaches

#### Heavy Keyword Syntax
Adding many new keywords (`procedure`, `phase`, `step`, `await`, `parallel`, `spawn`, `checkpoint`, `abort on`, `holdable`) creates a second language within Arc and fights its reactive nature.

#### Per-Step Output Ports
Too granular. Real sequences have sequence-level aborts, not just step-level branching.

#### Implicit Token Passing
Too implicit for safety-critical workflows. "Emergent sequencing" from reactive primitives doesn't provide the clarity needed.

#### Overloaded Syntax
Using `[]` for stages and `{}` for imperative blocks while reusing `=` for different semantics in routing tables led to confusion. Same syntax, different meanings = bad.

#### Top-Level Label Syntax
Using `main:` as a bare label at the top level was inconsistent. Labels (`:`) should be used for inline labeling in chains, not standalone declarations.

#### `stage` Starting Chains
Using `stage main { } => ...` to start a chain was misleading — `stage` implies a single thing, but we're defining a sequence of stages.

---

## Core Insight: Sequences Are State Machines

The model is simple: **sequences are state machines with syntactic sugar**.

- **Stage**: A state — reactive flows run while in this state, transitions exit it
- **Sequence**: A named state machine containing stages

The sugar is:
- `next` = "the stage defined after this one" (no need to repeat names)
- Definition order = documentation of the happy path
- Stages can jump to any stage in the same sequence by name
- Stages can jump to other sequences by name

---

## Final Design: Sequences + Stages

### Core Keywords

| Keyword | Syntax | Purpose |
|---------|--------|---------|
| `sequence` | `sequence name { stages... }` | Named state machine containing stages |
| `stage` | `stage name { }` | State definition within a sequence |
| `next` | `=> next` | Transition to the next stage in definition order |
| `match` | `match { x => y }` | Transition pattern matching |
| `wait` | `wait{duration}` | One-shot timer (false until elapsed, then true) |
| `interval` | `interval{duration}` | Repeating timer (fires every N duration) |
| `log` | `log{"message"}` | Log a message (also used for lifecycle tracing) |
| `route` | `route { x = y }` | Reactive value routing (existing Arc) |

### Two Edge Types

| Syntax | Meaning |
|--------|---------|
| `->` | Reactive flow - runs continuously while in stage |
| `=>` | One-shot - runs once when condition becomes true |

**Key distinction:**
- `->` is **continuous**: every time the source fires, the flow executes
- `=>` is **one-shot**: executes once when the condition becomes true, then doesn't fire again (until stage is re-entered)

**`=>` does not inherently mean "leave the stage"** — that depends on what follows:

```arc
// One-shot command, stay in stage
wait{10ms} => 1 -> fuel_valve_cmd

// One-shot transition, leave stage
pressure > max => abort

// One-shot action with logging
wait{5s} => log{"timeout warning"}

// Transition to next in chain
ox_tank_psi > target => next
```

The "transition" behavior happens when the target is a stage name. Otherwise, `=>` just
runs the action once and stays in the current stage.

**One-shot flow semantics**: When `=>` targets a flow chain (e.g., `1 -> fuel_valve_cmd`),
the entire chain executes **once** when the condition becomes true, then stops. This is
different from a reactive flow (`->` at the top level of a stage), which runs continuously.

```arc
stage example {
    interval{100ms} -> control{},        // Continuous: runs every 100ms
    wait{10ms} => 1 -> fuel_valve_cmd    // One-shot: executes once at 10ms, then done
}
```

---

## Syntax Reference

### Sequences

A sequence is a state machine containing stages. **Definition order determines what
`next` means** — stages are ordered by their position in the file:

```arc
sequence main {
    stage precheck {
        // ...
        ok => next  // Goes to pressurization (defined next)
    }

    stage pressurization {
        // ...
        done => next,              // Goes to igniter
        hold_btn => pressurization_hold  // Jump by name
    }

    stage igniter {
        // ...
        flame => next
    }

    // Hold stages - not part of the "next" chain, but can be jumped to
    stage pressurization_hold {
        resume_btn => pressurization,  // Jump back by name
        abort_btn => abort             // Jump to different sequence
    }
}
```

### Definition Order = Happy Path

Place the nominal/happy path stages first, in order. Place exception handlers (holds,
error states) after them. The `next` keyword **always** follows definition order,
regardless of whether a stage is part of the happy path or not.

```arc
sequence main {
    // Happy path (in order)
    stage step1 { ... => next }  // next = step2
    stage step2 { ... => next }  // next = step3
    stage step3 { ... => next }  // next = step1_hold (probably wrong!)

    // Exception handlers - avoid using `next` here
    stage step1_hold { ... => step1 }  // explicit target
    stage step2_hold { ... => step2 }  // explicit target
}
```

**Warning**: Using `next` in a hold or exception stage will go to whatever stage is
defined after it, which is rarely the intended behavior. Always use explicit stage
names for transitions out of exception handlers.

**Best practice**: Place terminal stages (like `complete`) at the end of the happy path
so that `=> next` from the last non-terminal happy path stage works correctly. Then
place all exception handlers after the terminal stage — they should never use `next`.

### Transitions

- `=> next` — go to the next stage in definition order
- `=> stage_name` — jump to any stage in the same sequence
- `=> sequence_name` — jump to a different sequence (e.g., `=> abort`)

`next` on a stage with no following stage is a **compile-time error**. Terminal stages
(like `complete` or `safed`) should have no `=> next` transition — a stage with no
outgoing transitions is implicitly terminal and remains active indefinitely.

### Reactive Flows vs One-Shots

Inside a stage:

```arc
stage pressurization {
    // Reactive flows (->): run continuously while in this stage
    interval{100ms} -> ox_press_control{target=ox_target_psi},
    interval{100ms} -> fuel_press_control{target=fuel_target_psi},

    // One-shots (=>): fire once when condition becomes true
    hold_button => pressurization_hold,
    copv_temp > 350 => abort,
    wait{30s} => abort,
    ox_tank_psi > ox_target_psi => next
}
```

**Key semantics:**
- `->` flows run every time their source fires (continuous/reactive)
- `=>` one-shots run once when their condition becomes true, then don't fire again
- Multiple `=>` conditions race — first one listed wins (priority by source order)
- `=> next` continues to the next stage in the chain
- `=> stage_name` transitions to that stage
- `=> action` runs the action once but stays in the current stage

**Literal sources**: When a literal value is the source of a `->` flow (e.g., `1 -> cmd`),
it executes **once on stage entry**, not continuously. Use `interval{}` for continuous
execution.

```arc
stage igniter {
    1 -> igniter_cmd,                    // Once: sends 1 on stage entry
    interval{100ms} -> control{},        // Continuous: runs every 100ms
}
```

**Transition priority:** When multiple transition conditions become true simultaneously,
the first-listed transition wins. This means **safety-critical transitions (like abort
conditions) should be listed before nominal transitions**.

```arc
stage pressurization {
    // 1. Safety conditions first (highest priority)
    copv_temp > 350 => abort,
    ox_tank_psi > ox_max_psi => abort,

    // 2. Operator controls
    hold_button => pressurization_hold,

    // 3. Nominal completion
    ox_tank_psi > ox_target_psi => next,

    // 4. Timeouts (lowest priority)
    wait{30s} => abort
}
```

The analyzer will emit warnings when it detects potentially dangerous ordering, such as
abort transitions appearing after non-abort transitions.

### Timer Built-ins: `wait{}` and `interval{}`

**`wait{}`** is a one-shot timer — false until the duration elapses, then true (once).

**`interval{}`** is a repeating timer — fires repeatedly every N duration.

**Idiomatic usage:**

```arc
interval{100ms} -> ox_press_control{}  // Continuous: every 100ms, run control
wait{2s} => abort                       // One-shot: after 2s, abort
wait{10ms} => 1 -> fuel_valve           // One-shot: after 10ms, send command
```

Other combinations (`interval{} =>`, `wait{} ->`) work but are redundant — `wait` only
fires once regardless of operator, and `interval` with `=>` acts like `wait`.

**Semantics:**
- Timer starts when stage is entered
- Multiple `wait{}` conditions can coexist — all start at stage entry and race
- `wait{}` accepts literal durations (`2s`, `100ms`) or config parameters
- When stage is re-entered (including from hold), timers restart from zero

**Example with multiple timers:**

```arc
stage igniter {
    1 -> igniter_cmd,

    // Both timers start at stage entry
    wait{500ms} => log{"igniter active"},  // Log at 500ms, stay in stage
    wait{2s} => abort,                      // Abort at 2s if no flame
    flame_detected => next                  // Success exits before timeout
}
```

### Common Safety Conditions

Rather than duplicating abort conditions across every stage, define reusable functions:

```arc
fn check_common_aborts{} -> bool {
    return copv_temp > 350 or pressure > max_pressure  // Use 'and', 'or', 'not' keywords
}

fn check_thermal_limits{} -> bool {
    return ox_inlet_temp > max_ox_temp or fuel_inlet_temp > max_fuel_temp
}
```

Then invoke them in each stage:

```arc
stage pressurization {
    // Common safety (reusable)
    check_common_aborts{} => abort,

    // Stage-specific safety
    ox_tank_psi > ox_max_psi => abort,

    // Nominal completion
    ox_tank_psi > target => next
}

stage igniter {
    check_common_aborts{} => abort,
    check_thermal_limits{} => abort,
    chamber_pressure > abort_pressure => abort,
    flame_detected => next
}
```

This keeps safety checks explicit at point of use (no hidden globals) while avoiding
duplication (logic defined once).

### Lifecycle Tracing with `log{}`

Arc does not provide built-in lifecycle hooks (on_enter, on_exit). Use `log{}` for
tracing stage transitions:

```arc
stage pressurization {
    log{"Entered pressurization stage"},   // Logs on entry (literal -> log)

    interval{100ms} -> ox_press_control{},

    ox_tank_psi > target => next
}

stage complete {
    log{"Sequence complete - nominal"}     // Terminal stage log
}
```

For more detailed tracing, use one-shot logs at specific points:

```arc
stage igniter {
    log{"Igniter stage entered"},
    wait{500ms} => log{"Igniter active for 500ms"},
    flame_detected => next
}
```

### Imperative Blocks with Match

When you need to run imperative logic and route based on the result:

```arc
stage precheck {
    {
        if not verify_connections() { return connection_fail }
        if not verify_sensors() { return sensor_fail }
        return ok
    } => match {
        ok => next,
        connection_fail => abort,
        sensor_fail => abort
    },
    abort_btn => abort,
    hold_btn => precheck_hold
}
```

**Key semantics:**
- `{ }` without a keyword is an imperative block (inline function body)
- Runs once on stage entry, must complete before ANY transitions are evaluated
- Returns a tag/value that gets routed by `match`
- `match { tag => target }` maps return values to transition targets
- Uses `=>` for consistency with transitions

**Execution order on stage entry:**
1. Imperative block executes (if present) — blocks until complete
2. Reactive flows start
3. Transition conditions begin evaluation

**Warning — Safety Critical**: Imperative blocks **block the entire stage**, including
ALL transition checks (even abort conditions). No transitions can fire until the block
completes. There is no built-in timeout or async execution — if a block hangs, the
sequence hangs and cannot be aborted.

```arc
stage precheck {
    {
        slow_operation()  // If this hangs, abort_btn cannot fire!
    } => match { ... },
    abort_btn => abort    // Cannot fire until block completes
}
```

**Developer responsibilities:**
- Keep imperative blocks fast (< 10ms recommended)
- Build timeouts into any function that might block (network, I/O, hardware)
- For long operations, restructure as reactive flows with `wait{}` or `interval{}`
- Never perform unbounded loops or blocking waits in imperative blocks

The runtime does not protect against slow imperative blocks. This is a deliberate design
choice to keep the execution model simple and predictable.

### Reactive Routing (Existing Arc)

For routing values in the reactive model:

```arc
source -> func{} -> route {
    a_output = target_a,
    b_output = target_b
}
```

**Key semantics:**
- Routes values from named outputs to targets
- Uses `=` because values flow (not just signals)
- This is the existing Arc model, unchanged

---

## Complete Example: Hotfire Test Sequence

```arc
// Entry point - top-level routing
start_cmd => main

// Main sequence - happy path stages first, then holds
sequence main {
    stage precheck {
        {
            if not verify_connections() { return connection_fail }
            if not verify_sensors() { return sensor_fail }
            return ok
        } => match {
            ok => next,
            connection_fail => abort,
            sensor_fail => abort
        },
        abort_btn => abort,
        hold_btn => precheck_hold
    }

    stage pressurization {
        // Reactive flows
        interval{100ms} -> ox_press_control{target=ox_target_psi},
        interval{100ms} -> fuel_press_control{target=fuel_target_psi},

        // Safety aborts (highest priority)
        copv_temp > 350 => abort,
        ox_tank_psi > ox_max_psi => abort,
        fuel_tank_psi > fuel_max_psi => abort,

        // Operator controls
        hold_button => pressurization_hold,

        // Nominal completion
        ox_tank_psi > ox_target_psi and fuel_tank_psi > fuel_target_psi => next,

        // Timeout (lowest priority)
        wait{30s} => abort
    }

    stage igniter {
        // Reactive flows
        1 -> igniter_cmd,

        // Safety aborts
        chamber_pressure > abort_pressure => abort,

        // Nominal completion
        flame_detected => next,

        // Timeout
        wait{2s} => abort
    }

    stage main_engine_start {
        // Immediate and delayed commands
        1 -> ox_valve_cmd,
        wait{10ms} => 1 -> fuel_valve_cmd,

        // Reactive flows
        interval{10ms} -> ox_valve_ramp{rate=10},
        interval{10ms} -> fuel_valve_ramp{rate=10},

        // Safety aborts
        chamber_pressure > abort_pressure => abort,

        // Nominal completion
        ox_valve_state == 1 and fuel_valve_state == 1 and chamber_pressure > min_chamber => next,

        // Timeout
        wait{5s} => abort
    }

    stage steady_state {
        // Reactive flows
        interval{10ms} -> ox_valve_control{target=100},
        interval{10ms} -> fuel_valve_control{target=100},

        // Safety aborts
        chamber_pressure > abort_pressure => abort,
        chamber_pressure < min_chamber => abort,
        ox_inlet_temp > max_ox_temp => abort,

        // Nominal completion
        wait{burn_duration} => next
    }

    stage shutdown {
        // Reactive flows
        0 -> ox_valve_cmd,
        0 -> fuel_valve_cmd,
        0 -> igniter_cmd,

        // Nominal completion
        ox_valve_state == 0 and fuel_valve_state == 0 and chamber_pressure < 50 => next,

        // Timeout
        wait{10s} => abort
    }

    stage safing {
        1 -> ox_vent_cmd,
        1 -> fuel_vent_cmd,
        ox_tank_psi < 20 and fuel_tank_psi < 20 => next
    }

    stage complete {
        log{"Test complete - nominal"}
    }

    // ----- Hold stages (exception handlers, not part of next chain) -----

    stage precheck_hold {
        resume_btn => precheck,
        abort_btn => abort
    }

    stage pressurization_hold {
        interval{100ms} -> ox_press_maintain{},
        interval{100ms} -> fuel_press_maintain{},

        // Safety aborts
        ox_tank_psi < ox_min_hold_psi => abort,
        fuel_tank_psi < fuel_min_hold_psi => abort,

        // Resume
        resume_btn => pressurization,
        abort_btn => abort
    }
}

// Abort sequence
sequence abort {
    stage close_valves {
        0 -> ox_valve_cmd,
        0 -> fuel_valve_cmd,
        0 -> igniter_cmd,

        // Check actual valve STATE, not the command we just sent
        ox_valve_state == 0 and fuel_valve_state == 0 and igniter_state == 0 => next,
        wait{2s} => next
    }

    stage vent {
        1 -> ox_vent_cmd,
        1 -> fuel_vent_cmd,
        1 -> emergency_vent_cmd,
        interval{100ms} -> abort_safing_log{},

        ox_tank_psi < 50 and fuel_tank_psi < 50 => next,
        wait{60s} => next
    }

    stage safed {
        log{"ABORT COMPLETE - SYSTEM SAFED"}
    }
}
```

---

## Design Decisions Summary

### Why explicit `sequence` keyword?
- A sequence is a state machine — the keyword makes this explicit
- Sequences own their stages — stages belong to exactly one sequence
- Provides a scope for stage names and `next` resolution

### Why definition order = happy path?
- No special chain syntax needed — file order *is* the specification
- `next` is just syntactic sugar for "the stage defined after this one"
- Exception handlers (holds) go at the bottom, out of the `next` chain
- Self-documenting: read top-to-bottom to understand normal flow

### Why stages inside sequences only?
- Eliminates ambiguity about which sequence a stage belongs to
- `=> stage_name` always refers to a stage in the same sequence
- `=> sequence_name` explicitly crosses to a different sequence
- Simpler scoping rules, easier to implement

### Cross-Sequence Transition Rules
When a transition targets another sequence (e.g., `=> abort`):

1. **Scope**: Only sequences in the same file can be targeted. Cross-file sequence
   references are not supported.
2. **Entry point**: Execution always starts at the first defined stage in the target
   sequence. There is no syntax for jumping to a specific stage in another sequence.
3. **Source termination**: The source sequence terminates entirely — all reactive flows
   stop, timers are cancelled, and state is discarded. Only the target sequence runs.

```arc
// In main sequence:
copv_temp > 350 => abort  // Terminates main, starts abort at close_valves

// abort sequence always starts here:
sequence abort {
    stage close_valves { ... }  // First stage = entry point
    stage vent { ... }
    stage safed { ... }
}
```

This is a one-way transition — there is no built-in mechanism to "return" to the source
sequence. If resumption is needed, model it explicitly with a new transition back.

### Why `->` vs `=>`?
- `->` is continuous: runs every time source fires, stays in stage
- `=>` is one-shot: runs once when condition becomes true
- `=>` to a stage name transitions; `=>` to an action stays in stage
- Clear visual distinction, consistent semantics

### Why top-level entry routing?
- `start_cmd => main` at top level routes channel events to sequences
- Sequences don't declare their own entry — that's external routing
- Keeps sequences focused on internal structure

### Top-Level Entry Point Rules

Entry points connect external events to sequences:

```arc
start_cmd => main           // Channel triggers sequence
emergency_stop => abort     // Multiple entry points allowed
sensor > threshold => alert // Expressions work too
```

**Semantics:**
- The left side can be any valid flow source (channel, expression, function call)
- The sequence starts when **any value** is written/produced by the source
- Multiple entry points are allowed — each can trigger a different sequence
- If a sequence is already running, the entry point behavior depends on the runtime
  (typically: ignore, queue, or error)

**Type requirements:** None — any type triggers the sequence. The value itself is
discarded; only the event matters.

### Why first-listed-wins for transition priority?
- Simple, predictable rule
- Users control priority by ordering their transitions
- Safety-critical conditions listed first take precedence

---

## Graphical Representation

The model maps naturally to visual editing:

- **Sequences** = Containers/groups
- **Stages** = Boxes/nodes within sequences
- **`=>`** = Arrows between stages
- **`->`** flows = Internal reactive wiring (collapsible detail)
- **Multiple chains** = Multiple entry arrows into a sequence
- **Imperative blocks** = "Check" or "Script" icon within a stage
- **`match`** = Decision diamond or routing table

The main sequence forms the happy path. Abort sequence branches off. Hold stages are standalone nodes that stages can transition to and from.

---

## Implementation Considerations

### Compilation
- Sequences compile to named groups of states
- Stages compile to state machine states
- `=>` transitions compile to state transitions with guards
- `->` flows compile to reactive subscriptions active only in that state
- Imperative blocks compile to functions called on state entry
- `next` resolves to concrete stage names at compile time
- Multiple chains in a sequence compile to racing entry conditions

### Runtime
- Stage scheduler manages current state
- System starts in designated entry sequence (e.g., `main`)
- First chain's triggering event activates that path
- Reactive flows for current stage are active
- Transition conditions race — first listed wins

### Tick Execution Model

Each scheduler tick follows this precise order:

1. **Snapshot**: Capture current state of all channels at tick start
2. **Execute reactive flows**: Run all `->` flows for current stage using snapshot values
3. **Evaluate transitions**: Check all `=>` conditions using snapshot values
4. **Fire transition**: If any transition condition is true, fire the first one (by source
   order) — current flow execution completes before transition occurs
5. **Enter new stage**: If transition fired, stop old flows and start new stage's flows

**Key guarantees:**

- **Snapshot isolation**: All reads within a tick see the same state (captured at tick
  start). Writes from reactive flows are not visible until the next tick.
- **Flow completion**: A reactive flow that is mid-execution when a transition becomes
  true will complete before the transition fires. Flows are never interrupted.
- **Source order priority**: When multiple transitions become true in the same tick, the
  first one listed in source code wins. No timestamp or priority annotation — ordering
  in the source file is the priority mechanism.
- **Atomic transition**: Transitions are instantaneous. There is no interleaving between
  the old stage's flows and the new stage's flows.

### Stage Transition Semantics

When a transition fires, the following sequence occurs:

1. **Current flow completes**: If a reactive flow is mid-execution when the transition
   condition becomes true, that execution completes before the transition occurs
2. **Exit current stage**: All reactive flows (`->`) for the current stage stop
3. **Transition fires**: Instantaneous, no user code runs during transition
4. **Enter new stage**: All reactive flows for the new stage begin

There is no overlap between stages — flows from the old stage stop before flows from
the new stage start. This provides clean handoff semantics.

**Implication**: Reactive flows should be quick and non-blocking. Long-running operations
should be broken into smaller incremental steps triggered by `interval{}`.

### Stage Restart Semantics

**Stages are stateless between entries.** Every time you enter a stage — whether for
the first time, returning from a hold, or via any other transition — the stage starts
fresh:

- All `wait{}` timers reset to zero
- All reactive flows start from their initial state
- No implicit "memory" of previous time in the stage

This keeps the mental model simple: entering a stage is always a clean start.

**Example**: If you're 25 seconds into a 30-second timeout, transition to a hold stage,
then return — the timeout restarts from 0, giving you a fresh 30 seconds.

### Stateful Variables in Sequences

Stateful variables (`$=`) can only be declared inside **functions**, not directly in
stages. To track state across stage entries (retry counts, elapsed time, etc.), use a
function with stateful variables:

```arc
fn retry_tracker{} -> u8 {
    count $= 0
    count = count + 1
    return count
}

stage retry_stage {
    retry_tracker{} > 3 => abort,  // Abort after 3 retries
    some_condition => next,
    wait{5s} => retry_stage        // Retry (count increments each entry)
}
```

**Cross-sequence behavior**: When transitioning to a different sequence (e.g., `=> abort`),
all state from the source sequence is discarded. Stateful variables in functions called
by the source sequence are reset.

### Type System
- `match` keys must match possible return values from imperative block
- Transition targets must be valid stage names (in scope)
- `->` sources and sinks must type-check per existing Arc rules
- `next` only valid in stages that have a following stage in definition order
- Stage names must be unique within their containing sequence
- **Name collision rule**: A stage cannot have the same name as any sequence in the file.
  This prevents ambiguity in `=> target` transitions (compile-time error if collision).

### Symbol Table Extensions

The symbol table (`arc/go/symbol/symbol.go`) needs two new symbol kinds:

```go
const (
    // ... existing kinds ...

    // KindSequence represents a sequence (state machine) declaration.
    KindSequence Kind = iota
    // KindStage represents a stage within a sequence.
    KindStage
)
```

**Scoping rules:**
- Sequences are declared at file scope (same level as functions)
- Stages are declared within sequence scope
- Stage names must be unique within their containing sequence
- `=> stage_name` resolves to a stage in the current sequence
- `=> sequence_name` resolves to a sequence in the same file

**Symbol resolution order for transitions:**
1. Check if target is `next` (resolve to next stage in definition order)
2. Check if target is a stage name in current sequence
3. Check if target is a sequence name in current file
4. Error: unknown transition target

### IR Representation

The IR (`arc/go/ir/ir.go`) needs new node types for sequences and stages:

```go
// Sequence represents a state machine containing stages.
type Sequence struct {
    Key    string   // Unique identifier
    Stages []Stage  // Ordered by definition (for `next` resolution)
    Entry  string   // Key of first stage (entry point)
}

// Stage represents a state within a sequence.
type Stage struct {
    Key         string        // Unique identifier
    Flows       []Edge        // Reactive flows (->)
    Transitions []Transition  // One-shot transitions (=>)
    Imperative  *Function     // Optional imperative block
    IsTerminal  bool          // True if no outgoing transitions
}

// Transition represents a one-shot state transition.
type Transition struct {
    Condition Expression  // Guard condition (nil = always true)
    Target    string      // Stage key or sequence key
    Priority  int         // Source order (lower = higher priority)
}
```

**Compilation mapping:**
- `sequence main { ... }` → `Sequence{Key: "main", ...}`
- `stage precheck { ... }` → `Stage{Key: "precheck", ...}`
- `interval{100ms} -> control{}` → `Edge` in `Stage.Flows`
- `pressure > max => abort` → `Transition` in `Stage.Transitions`
- `{ if ... } => match { ... }` → `Stage.Imperative` + `Transitions`

---

## Open Questions

1. ~~**Parallel stages**: How to express "run these stages in parallel, continue when both complete"?~~
   **Deferred**: Parallel stage execution is out of scope for the initial design. For now,
   parallel operations should be combined into a single stage with multiple reactive flows:
   ```arc
   stage pressurization {
       // Both run concurrently within the same stage
       interval{100ms} -> ox_press_control{target=ox_target_psi},
       interval{100ms} -> fuel_press_control{target=fuel_target_psi},
       // Completion requires both conditions
       ox_tank_psi > ox_target_psi and fuel_tank_psi > fuel_target_psi => next
   }
   ```
   Future iterations may add explicit parallel stage syntax if needed.
2. ~~**Sequence completion**: What happens when `next` is used on the last stage? End of sequence? Error?~~
   **Resolved**: `next` on a stage with no following stage is a compile-time error. Terminal
   stages have no outgoing transitions and remain active indefinitely.
3. **Entry point syntax**: Is `start_cmd => main` the right syntax for top-level routing?
4. **Stage parameters**: Can stages accept config parameters for reuse?

---

## References

- IEC 61131-3 Sequential Function Charts (SFC) - Industrial standard for sequential control
- State machine patterns - Underlying execution model
- Arc language spec - Base reactive model
