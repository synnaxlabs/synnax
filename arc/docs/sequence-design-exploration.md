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

## Core Insight: Sequences and Stages

The model has two concepts:

- **Stage**: A reactive context — flows run until a transition fires. The atomic unit.
- **Sequence**: A named container for one or more chains of stages. Defines entry points and paths.

A sequence is not a runtime concept separate from stages — it's syntactic structure that makes the declaration honest about what you're defining. Under the hood, it's all stages and transitions.

---

## Final Design: Sequences + Stages

### Core Keywords

| Keyword | Syntax | Purpose |
|---------|--------|---------|
| `sequence` | `sequence name { chains }` | Named container for chains of stages |
| `stage` | `stage name { }` | Standalone stage definition |
| `route` | `route { x = y }` | Reactive value routing (existing Arc) |
| `match` | `match { x => y }` | Transition pattern matching |

### Two Edge Types

| Syntax | Meaning |
|--------|---------|
| `->` | Reactive flow - runs continuously while in stage, doesn't leave |
| `=>` | Transition - fires once, leaves the stage |

### The `:` Rule

**`:` labels inline stages within chains** — it's like a Go label, marking a point you can jump to.

- Inside `sequence { }`: use `name: stage { }` for inline stage definitions
- Outside sequences: use `stage name { }` for standalone definitions
- `:` is a label, not a declaration

---

## Syntax Reference

### Sequences

A sequence is a named container for one or more chains:

```arc
sequence main {
    start_cmd => precheck: stage { ... } => pressurization: stage { ... } => igniter: stage { ... }
}
```

### Multiple Chains in a Sequence

Sequences can have multiple chains, separated by `,`. These are racing entry points — whichever event fires first determines the path:

```arc
sequence main {
    // Primary path
    start_cmd => precheck: stage { ... } => pressurization: stage { ... },

    // Skip precheck (for testing)
    skip_to_press_cmd => pressurization,

    // Direct abort from idle
    emergency_cmd => abort
}
```

Stages can be shared across chains by referencing them by name.

### Standalone Stage Definitions

Stages outside sequences use definition style:

```arc
stage precheck_hold {
    resume_btn => precheck
    abort_btn => abort
}

stage pressurization_hold {
    interval{100ms} -> ox_press_maintain{}
    resume_btn => pressurization
    abort_btn => abort
}
```

### Reactive Flows vs Transitions

Inside a stage:

```arc
pressurization: stage {
    // Reactive flows (->): run continuously while in this stage
    interval{100ms} -> ox_press_control{target=ox_target_psi}
    interval{100ms} -> fuel_press_control{target=fuel_target_psi}

    // Transitions (=>): fire once to leave this stage
    hold_button => pressurization_hold
    copv_temp > 350 => abort
    wait{30s} => abort
    ox_tank_psi > ox_target_psi && fuel_tank_psi > fuel_target_psi => next
}
```

**Key semantics:**
- `->` flows run every time their source fires (reactive)
- `=>` transitions race against each other — first one to fire wins
- `=> next` continues to the next stage in the chain

### Imperative Blocks with Match

When you need to run imperative logic and route based on the result:

```arc
precheck: stage {
    {
        if !verify_connections() { return connection_fail }
        if !verify_sensors() { return sensor_fail }
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
- Returns a tag/value that gets routed by `match`
- `match { tag => target }` maps return values to transition targets
- Uses `=>` for consistency with transitions

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
// Main sequence with primary path
sequence main {
    start_cmd => precheck: stage {
        {
            if !verify_connections() { return connection_fail }
            if !verify_sensors() { return sensor_fail }
            return ok
        } => match {
            ok => next,
            connection_fail => abort,
            sensor_fail => abort
        },
        abort_btn => abort,
        hold_btn => precheck_hold
    } => pressurization: stage {
        interval{100ms} -> ox_press_control{target=ox_target_psi}
        interval{100ms} -> fuel_press_control{target=fuel_target_psi}
        hold_button => pressurization_hold
        copv_temp > 350 => abort
        ox_tank_psi > ox_max_psi => abort
        fuel_tank_psi > fuel_max_psi => abort
        wait{30s} => abort
        ox_tank_psi > ox_target_psi && fuel_tank_psi > fuel_target_psi => next
    } => igniter: stage {
        1 -> igniter_cmd
        wait{2s} => abort
        flame_detected => next
        chamber_pressure > abort_pressure => abort
    } => main_engine_start: stage {
        1 -> ox_valve_cmd
        wait{10ms} -> 1 -> fuel_valve_cmd
        interval{10ms} -> ox_valve_ramp{rate=10}
        interval{10ms} -> fuel_valve_ramp{rate=10}
        chamber_pressure > abort_pressure => abort
        wait{5s} => abort
        ox_valve_state == 1 && fuel_valve_state == 1 && chamber_pressure > min_chamber => next
    } => steady_state: stage {
        interval{10ms} -> ox_valve_control{target=100}
        interval{10ms} -> fuel_valve_control{target=100}
        chamber_pressure > abort_pressure => abort
        chamber_pressure < min_chamber => abort
        ox_inlet_temp > max_ox_temp => abort
        wait{burn_duration} => next
    } => shutdown: stage {
        0 -> ox_valve_cmd, fuel_valve_cmd, igniter_cmd
        wait{10s} => abort
        ox_valve_state == 0 && fuel_valve_state == 0 && chamber_pressure < 50 => next
    } => safing: stage {
        1 -> ox_vent_cmd, fuel_vent_cmd
        ox_tank_psi < 20 && fuel_tank_psi < 20 => next
    } => complete: stage {
        log{"Test complete - nominal"}
    }
}

// Hold stages (standalone definitions)
stage precheck_hold {
    resume_btn => precheck
    abort_btn => abort
}

stage pressurization_hold {
    interval{100ms} -> ox_press_maintain{}
    interval{100ms} -> fuel_press_maintain{}
    resume_btn => pressurization
    abort_btn => abort
    ox_tank_psi < ox_min_hold_psi => abort
    fuel_tank_psi < fuel_min_hold_psi => abort
}

// Abort sequence
sequence abort {
    stage {
        0 -> ox_valve_cmd, fuel_valve_cmd, igniter_cmd
        wait{2s} => next
        ox_valve_cmd == 0 && fuel_valve_cmd == 0 && igniter_cmd == 0 => next
    } => abort_safing: stage {
        1 -> ox_vent_cmd, fuel_vent_cmd, emergency_vent_cmd
        interval{100ms} -> abort_safing_log{}
        ox_tank_psi < 50 && fuel_tank_psi < 50 => next
        wait{60s} => next
    } => abort_complete: stage {
        log{"ABORT COMPLETE - SYSTEM SAFED"}
    }
}
```

---

## Design Decisions Summary

### Why explicit `sequence` keyword?
- Makes declarations honest — you're defining a sequence, not a single stage
- A sequence can contain multiple chains (multiple entry points)
- Clearly distinguishes composition (sequence) from atomic unit (stage)
- Under the hood, it's still all stages — `sequence` is syntactic clarity

### Why allow multiple chains in a sequence?
- Real systems have multiple entry points (normal start, test skip, emergency)
- Chains race — whichever event fires first determines the path
- Stages can be shared across chains by name
- Mirrors how stages work internally (multiple racing transitions)

### Why `stage name { }` for standalone stages?
- Clear definition syntax for stages not in a sequence
- Hold stages, utility stages live outside sequences
- Can be referenced by any sequence or stage

### Why `:` for inline labels?
- `:` is a label (like Go's goto labels)
- Labels mark points you can jump to
- `name: stage { }` labels an inline stage in a chain
- Distinct from `:=` which would be variable declaration

### Why `match` with `=>`?
- Consistent with transition semantics everywhere
- `ok => next` in match means the same as `condition => target` in a stage
- No semantic overloading

### Why `->` vs `=>`?
- `->` is reactive: runs continuously, stays in stage
- `=>` is transition: fires once, leaves stage
- Clear visual distinction, consistent semantics

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
- Transition conditions race — first match wins

### Type System
- `match` keys must match possible return values from imperative block
- Transition targets must be valid stage names (in scope)
- `->` sources and sinks must type-check per existing Arc rules
- `next` only valid in stages that are part of a chain
- Stage names must be unique within their scope

---

## Open Questions

1. **Parallel stages**: How to express "run these stages in parallel, continue when both complete"?
2. **Stage parameters**: Can stages accept parameters for reuse?
3. **Nested sequences**: Can sequences contain other sequences?
4. **Resume semantics**: When returning from hold, does the stage restart or continue?
5. **Scope rules**: Can sequences reference stages from other sequences?
6. **Anonymous stages**: First stage in abort sequence has no label — is this okay?

---

## References

- IEC 61131-3 Sequential Function Charts (SFC) - Industrial standard for sequential control
- State machine patterns - Underlying execution model
- Arc language spec - Base reactive model
