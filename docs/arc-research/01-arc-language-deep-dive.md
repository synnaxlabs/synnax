# Arc Programming Language - Deep Dive Research

## Overview

Arc is a **domain-specific programming language for reactive automation and control
systems** in the Synnax platform. It specializes in hardware telemetry, control
sequences, and state machine logic.

> **Beta Status**: Arc is currently in beta. A stable release is targeted for the end of
> February 2026. Language features and syntax may change before the stable release.

---

## 1. What is Arc and Why Does It Exist?

### Purpose

Arc exists to simplify complex hardware control sequences without requiring traditional
programming constructs like explicit loops or thread management. It provides:

- **Reactive execution model** with event-driven programming
- **Channel-based communication** (unbounded FIFO queues)
- **WebAssembly compilation** for sandboxed execution
- **Type-safe dimensional analysis** for physical quantities
- **Stateful computations** that persist across invocations

### Problems Arc Solves

1. **Control sequence complexity** - Traditional programming languages require manual
   thread management and event loops for hardware control. Arc eliminates this.
2. **Safety** - WASM sandboxing prevents runaway control programs from crashing systems
3. **Real-time coordination** - Glitch-free reactive execution prevents data races
4. **Hardware integration** - First-class support for channels, series data, and
   timestamps
5. **State machine patterns** - Built-in sequence/stage constructs for multi-step
   automation

### Target Users

Arc targets two audiences equally with different entry points:

**Control/Test Engineers** (limited programming experience):

- Entry point: Graph mode for visual programming
- Use case: Simple alarms and monitoring logic
- Progression: Text mode with intuitive sequence syntax

**Software Engineers** (traditional programming background):

- Entry point: Text mode with familiar imperative constructs
- Use case: Complex control sequences with full language power
- Strength: Imperative programming within function blocks

### Target Industries

1. **Aerospace** (primary) - Rocket test stands, propulsion systems, flight hardware
2. **Industrial Automation** (secondary) - Manufacturing, process control

---

## 2. Architecture: How Arc Works

### Compilation Pipeline

```
Source Code (.arc)
    ↓
Parser (ANTLR4) → Abstract Syntax Tree
    ↓
Analyzer Pass 1: Collect declarations → Symbol table
    ↓
Analyzer Pass 2: Type checking + Validation
    ↓
Analyzer Pass 3: Constraint unification (polymorphic types)
    ↓
Stratifier: Compute execution order for glitch-free reactive execution
    ↓
Graph/Text Analyzer: Build dataflow IR
    ↓
Compiler: Generate WASM + Build IR
    ↓
Output Package (IR + WASM + Memory Maps)
```

### Core Components

| Component  | Location              | Purpose                                       |
| ---------- | --------------------- | --------------------------------------------- |
| Parser     | `/arc/go/parser/`     | ANTLR4-based lexer/parser, produces AST       |
| Analyzer   | `/arc/go/analyzer/`   | 3-pass type checking and validation           |
| Stratifier | `/arc/go/stratifier/` | Computes glitch-free execution order          |
| Compiler   | `/arc/go/compiler/`   | Generates WebAssembly bytecode                |
| IR         | `/arc/go/ir/`         | Intermediate representation (nodes, edges)    |
| Symbol     | `/arc/go/symbol/`     | Hierarchical scope and symbol management      |
| Types      | `/arc/go/types/`      | Type system with units and polymorphism       |
| Runtime    | `/arc/go/runtime/`    | WASM execution environment and host functions |
| LSP        | `/arc/go/lsp/`        | Language Server Protocol for IDE support      |

### Key Files

- **Grammar**: `/arc/go/parser/ArcParser.g4` (463 lines), `/arc/go/parser/ArcLexer.g4`
- **Specification**: `/arc/docs/spec.md` (753 lines - complete language spec)
- **Tests**: `/arc/go/compiler/compiler_test.go` (2000+ lines of examples)

---

## 3. Two Representations: Text and Graph

Arc supports **two equivalent representations**:

### Text Mode

Traditional text-based programming with Arc syntax. Best for:

- Complex control sequences
- Experienced programmers
- Version control and diffs
- Full language features (sequences, stages, functions)

### Graph Mode (Visual)

Block-based visual programming interface. Best for:

- Threshold-based alarms and notifications
- Simple data processing pipelines
- Non-programmers getting started
- Quick visual configuration

**Available Blocks in Graph Mode:**

| Group            | Blocks                                                                                                                     |
| ---------------- | -------------------------------------------------------------------------------------------------------------------------- |
| **Basic**        | `Constant`, `Change Status` (notifications/alarms)                                                                         |
| **Telemetry**    | `Telemetry Source` (read channel), `Telemetry Sink` (write channel)                                                        |
| **Operators**    | `Add`, `Subtract`, `Multiply`, `Divide`, `Greater Than`, `Less Than`, `Equal`, `Not Equal`, `>=`, `<=`, `And`, `Or`, `Not` |
| **Flow Control** | `Select` (route by condition), `Stable For` (debounce)                                                                     |

**Current Limitation**: graph mode does not support sequences, stages, or custom
functions. Complex multi-stage control logic requires text mode.

**Note**: Graph and text modes are separate - programs created in one mode stay in that
mode.

---

## 4. Syntax Reference

### Primitive Types

```arc
// Integer types (signed and unsigned, 8-64 bit)
i8, i16, i32, i64    // signed integers
u8, u16, u32, u64    // unsigned integers

// Floating point
f32, f64             // 32 and 64 bit floats

// String
str                  // UTF-8 string

// Compound types
chan f64             // channel of float64
series f64           // array/series of float64
chan series i32      // channel of series
```

**Type Defaults**:

- Integer literals default to `i64`
- Float literals default to `f64`
- Boolean is `u8` (0=false, non-zero=true)

### Variables

```arc
// Local variables (reset each invocation)
x := 42                  // type inferred
y f64 := 3.14           // explicit type

// Stateful variables (persist across invocations) - UNIQUE TO ARC
count $= 0              // type inferred
total f64 $= 0.0        // explicit type

// Assignment
count = count + 1
```

**Stateful variable initialization**: The `$=` operator sets the initial value on the
first execution only. On subsequent executions, the persisted value is used instead.
Crucially, you can initialize a stateful variable from an input parameter:

```arc
func delta(value f64) f64 {
    prev $= value        // initialized to first input on first call
    d := value - prev
    prev = value
    return d
}
```

On the first call, `prev` is set to `value`. On subsequent calls, `prev` retains its
value from the previous execution. This eliminates the need for "first run" flag
patterns when tracking previous values.

### Operators

```arc
// Arithmetic
result := 2 + 3 * 4      // 14
power := 2 ^ 8           // 256 (exponentiation, NOT XOR - UNIQUE)
remainder := 10 % 3      // 1

// Comparison (returns u8: 1 or 0)
in_range := temp >= 20 and temp <= 30

// Logical (short-circuit)
condition := enabled and (pressure > threshold)
negated := not flag

// Compound assignment
count += 1
value -= 10
total *= 2
```

**Operator Precedence** (highest to lowest):

1. `^` (exponentiation, right-associative)
2. `-`, `not` (unary, right-associative)
3. `*`, `/`, `%` (left-associative)
4. `+`, `-` (left-associative)
5. Comparisons: `<`, `>`, `<=`, `>=`, `==`, `!=`
6. `and`, `or` (short-circuit)

### Control Flow

```arc
// If-else-if-else
if condition {
    // statements
} else if other {
    // statements
} else {
    // statements
}

// Return (required - no implicit returns)
func example() f64 {
    if done {
        return 0.0
    }
    return compute()
}

// NOTE: No loops - use reactive patterns with stateful variables instead
```

### Functions

```arc
// Basic function
func add(x f64, y f64) f64 {
    return x + y
}

// Optional parameters (must be trailing)
func clamp(value f64, min f64 = 0.0, max f64 = 1.0) f64 {
    if value < min { return min }
    if value > max { return max }
    return value
}


// Stateful function (counter persists across calls)
func counter() i64 {
    count $= 0          // stateful variable
    count = count + 1
    return count
}
```

### Configuration Parameters - UNIQUE TO ARC

Config parameters are set at instantiation time and can be **literals or channel
references**:

```arc
func controller{
    setpoint f64,           // can be a literal OR a channel
    sensor chan f64,        // channel reference
    actuator chan f64       // channel reference
} (enable u8) f64 {
    value := sensor
    if enable {
        actuator = value - setpoint
    }
    return value
}

// Static setpoint (literal value)
sensor -> controller{setpoint=100.0, sensor=temp, actuator=valve}

// Dynamic setpoint (from another channel - e.g., operator input)
sensor -> controller{setpoint=setpoint_input, sensor=temp, actuator=valve}
```

This allows flexible control systems where setpoints can come from operator input
channels, other Arc programs, or be hardcoded values.

### Unit System - UNIQUE TO ARC

```arc
// Type annotations with units
velocity f64 m/s := 10.0
distance f64 m := 50.0

// Numeric literals with units (no whitespace allowed)
wait_time := 100ms      // i64 with time units
frequency := 10hz       // converts to period

// Temporal units: ns, us, ms, s, min, h
// Frequency units: hz, khz, mhz

// Dimensional compatibility (compiler enforces)
speed := distance / time   // f64 m/s
area := distance ^ 2       // f64 m^2
```

### Series (Arrays)

```arc
// Literals
data := [1.0, 2.0, 3.0]
empty f64 series := []

// Operations
length := len(data)              // i64
first := data[0]                 // indexing
subset := data[1:3]              // slicing

// Elementwise operations - UNIQUE TO ARC
scaled := data * 2.0             // [2.0, 4.0, 6.0]
sum := data + [4.0, 5.0, 6.0]    // [5.0, 7.0, 9.0]
mask := data > 2.0               // [0, 0, 1] (series u8)
```

### Channels

Channels connect Arc programs to Synnax telemetry data. Channel names reference channels
that exist in your Synnax cluster.

```arc
// Read from channel (non-blocking, returns zero if empty)
value := ox_pt_1

// Write to channel
ox_pt_cmd = value
```

**Channel Discovery**: The text editor provides LSP-powered autocomplete, hover
information, and suggestions for available channels in your cluster. You can also browse
channels using the Console's channel explorer before writing Arc code.

**Channel Naming**: Channel names must start with a letter or underscore, and can only
contain letters, digits, and underscores (snake_case). Examples: `ox_pt_1`, `fuel_tc_2`,
`pressure_valve_cmd`. See the Channels documentation for full naming conventions.

---

## 5. Reactive Execution Model - UNIQUE TO ARC

### Stratified Execution (Why It Matters)

Arc guarantees **deterministic, glitch-free execution** through stratified scheduling:

1. **Snapshot Consistency**: All nodes in an execution cycle see the same consistent
   snapshot of input values. If a channel updates while the graph is executing, all
   nodes see the same snapshot - no node sees a "newer" value than another.

2. **Deterministic Execution Order**: Nodes execute in a guaranteed order based on their
   dependencies (organized into "strata"). You always know node A runs before node B.

**Why this matters for aerospace/industrial control:**

```
Example: Two branches reading the same pressure sensor

              ┌─→ safety_check{}
pressure_pt ──┤
              └─→ controller{}

WITHOUT stratified execution:
- pressure_pt updates to 500 psi
- safety_check might see 500 psi
- controller might see OLD value (400 psi) if it ran first
- Inconsistent behavior, hard to debug, potential safety issue

WITH stratified execution:
- All nodes see the SAME snapshot (500 psi)
- Execution order is deterministic and repeatable
- Behavior is predictable and certifiable
```

Without this guarantee, debugging becomes difficult and safety certification nearly
impossible.

### Channels: Reactive vs Imperative Context

Channels behave differently depending on context.

#### Reactive Context (Flow Statements)

In flow statements, channels are **event-driven data sources**:

```arc
sensor -> filter{threshold=50.0} -> controller{} -> output
```

- **Event-driven**: New data arriving on `sensor` triggers the entire pipeline
- **Time-series data**: Channels carry arrays of values with timestamps
- **Watermark tracking**: Only NEW data (past a high-water mark) triggers execution
- **No polling**: The runtime automatically fires nodes when inputs have new data

#### Imperative Context (Function Bodies)

Inside functions, channel reads are **non-blocking snapshots**:

```arc
func controller{
    setpoint chan f64,    // channel passed via config
    output chan f64
} (value f64) f64 {
    target := setpoint    // Read latest value from channel (non-blocking)
    error := value - target
    if error > 10 {
        output = 1        // Write to channel (queued)
    }
    return error
}
```

- **Non-blocking**: `target := setpoint` returns immediately with the latest value
- **Snapshot**: Gets the most recent single value, not the full time-series
- **Zero on empty**: If no data has been written to the channel yet, returns 0
- **Writes are queued**: Channel writes inside functions are buffered until cycle end

#### Key Differences

| Aspect            | Reactive (`->`)      | Imperative (function body)    |
| ----------------- | -------------------- | ----------------------------- |
| **Trigger**       | New data arrival     | Function invocation           |
| **Data form**     | Time-series (arrays) | Single scalar value           |
| **Empty channel** | Skips execution      | Returns zero value            |
| **Use case**      | Main data pipelines  | Side-channel reads, setpoints |

**Best practice**: Use reactive flows (`->`) for your main data pipelines. Use
imperative channel reads inside functions for auxiliary inputs like setpoints or
configuration values that may change during execution.

### Flow Statements (Dataflow)

Arc uses **reactive dataflow** to connect channels and functions:

```arc
// Simple pipeline with continuous flow (->)
sensor -> filter{threshold=50.0} -> controller{} -> actuator

// Expressions in flows (implicit function)
temperature > 100 -> alarm{}
(sensor1 + sensor2) / 2.0 -> display
```

### Two Flow Operators

| Operator | Name       | Behavior                                                         |
| -------- | ---------- | ---------------------------------------------------------------- |
| `->`     | Continuous | Reactive flow executes every cycle                               |
| `=>`     | One-Shot   | Fires once when condition becomes true, resets on stage re-entry |

#### Continuous Flow (`->`) - Real-World Examples

Use `->` when you need to **repeatedly execute** a function on incoming data:

```arc
// Feed every pressure reading through a doubler for display scaling
ox_pt_1 -> doubler{} -> ox_pt_doubled

// Bang-bang control at 50ms intervals
interval{period=50ms} -> bang_bang{}

// Continuous sensor averaging
{ ox_pt_1: a, ox_pt_2: b } -> averager{} -> ox_pt_avg
```

#### One-Shot Transition (`=>`) - Real-World Examples

Use `=>` for **conditional transitions** that should fire once:

```arc
// Abort sequence when oxidizer pressure exceeds limit (fires once)
ox_pt_1 > 30 => abort

// Log a message once when pressure drops below threshold
ox_pt_1 < 20 => log_message{}

// Transition to next stage when target reached
pressure > 500 => next
```

The key distinction: `->` keeps flowing data continuously, while `=>` triggers once and
stops (until the stage is re-entered).

---

## 6. Sequences (State Machines) - UNIQUE TO ARC

### Concurrency Within Stages

In traditional programming, code executes line-by-line, top to bottom. In Arc stages,
**all flows run concurrently**. Every line in a stage is active at the same time.

```arc
stage pressurize {
    // ALL of these run simultaneously, not sequentially:
    sensor -> pressure_control{target=500},    // control loop running
    ox_pt_1 > 600 => abort,                    // checking abort condition 1
    fuel_pt_1 > 400 => abort,                  // checking abort condition 2
    tank_temp > 300 => abort,                  // checking abort condition 3
    leak_detected => abort,                    // checking abort condition 4
    abort_btn => abort,                        // listening for operator abort
    hold_btn => hold,                          // listening for operator hold
    pressure > 500 => next                     // checking for success condition
}
```

**What this means:**

- You don't write loops to "keep checking" conditions - they're always being checked
- You don't need threads or async code - concurrency is automatic
- All abort conditions are monitored simultaneously while control logic runs

**Line Order Matters for Transition Conflicts:** When multiple one-shot transitions
(`=>`) are true simultaneously, **the one listed first wins**. This means you should put
safety-critical conditions first:

```arc
stage pressurize {
    // SAFETY CONDITIONS FIRST - these take priority for transitions
    ox_pt_1 > 600 => abort,         // 1st priority
    fuel_pt_1 > 400 => abort,       // 2nd priority
    abort_btn => abort,             // 3rd priority

    // SUCCESS CONDITIONS AFTER SAFETY
    pressure > 500 => next,         // only triggers if no abort conditions are true

    // CONTINUOUS FLOWS - all execute, order doesn't determine priority
    sensor -> pressure_control{target=500}
}
```

**Note on continuous flows (`->`):** All continuous flows execute, but if multiple flows
write to the same channel, **last write wins** based on line order:

```arc
stage press {
    0 -> press_vlv_cmd,    // writes 0
    1 -> press_vlv_cmd,    // writes 1 (overwrites the 0)
}
// Result: press_vlv_cmd receives 1
```

**The mental model:** Think of a stage as a set of rules that are ALL active at once.
Arc handles the concurrency, but when conflicts arise, earlier lines take priority.
**Always put safety-critical abort conditions at the top of your stages.**

```arc
sequence main {
    stage pressurize {
        // Reactive flows: run continuously while stage is active
        1 -> pressure_valve,

        // One-shot transitions: fire once when condition is true
        pressure > 500.0 => next,
        abort_btn => abort
    }

    stage ignite {
        igniter_cmd = 1,
        flame_detected => next
    }

    stage shutdown {
        0 -> pressure_valve,
        pressure < 50.0 => complete
    }

    stage complete {
        // terminal stage
    }
}

sequence abort {
    stage safed {
        all_valves_cmd = 0
    }
}

// Entry points - triggered by channels
start_cmd => main           // channel triggers sequence
emergency_stop => abort     // multiple entries allowed
```

### Triggering Sequences

Entry points like `start_cmd` are **u8 virtual channels** that you create in Synnax. To
start a sequence:

1. Create a u8 virtual channel (e.g., `start_cmd`)
2. Wire that channel to a button on a Console schematic
3. When the user clicks the button, it writes `1` to the channel
4. Arc sees the truthy value and triggers the sequence

```
Console Schematic          Synnax Channel          Arc Program
┌─────────────┐            ┌─────────────┐         ┌─────────────┐
│ [Start Btn] │ ──click──→ │ start_cmd=1 │ ──────→ │ => main     │
└─────────────┘            └─────────────┘         └─────────────┘
```

Triggers can also come from other sources: another Arc program, external systems writing
to the channel, or sensor values exceeding thresholds.

### Transition Targets

- `=> next` - Go to next stage in definition order
- `=> stage_name` - Jump to any stage in same sequence
- `=> sequence_name` - Jump to different sequence

### Stage Semantics

- Only one stage active per sequence at a time
- On stage entry: one-shot states reset, stateful nodes reset
- Reactive flows start fresh
- Stages are stateless between entries

---

## 7. Standard Library

Arc includes a standard library of nodes and functions. The current library focuses on
core timing, statistics, and I/O operations. A full math library (including `sum`,
`sqrt`, `abs`, trigonometric functions, etc.) is planned for future releases.

### Core Built-In Functions

```arc
len(series)     // i64 - length of series
len(str)        // i64 - length of string (bytes)
now()           // i64 ns - current timestamp
```

### Timing Nodes and Real-Time Scheduling

Timing is fundamental to control systems. Arc provides two timing nodes for creating
periodic and delayed behaviors.

#### Interval Node - Periodic Control Loops

```arc
// Create a 50ms control loop
interval{period=50ms} -> read_sensor -> compute_control -> write_actuator
```

**How intervals work:**

1. The interval node tracks elapsed time since program start
2. When `elapsed - lastFired >= period`, it outputs `1` and triggers downstream nodes
3. The first tick fires immediately (at t=0)
4. Downstream nodes execute each time the interval fires

**Creating a control loop:**

```arc
// Bang-bang pressure control at 20Hz (50ms)
interval{period=50ms} -> pressure_controller{target=500, hysteresis=10}

// Data logging at 10Hz
interval{period=100ms} -> log_all_sensors{}
```

**Timing precision:**

- Precision depends on the runtime's scheduler tick frequency
- The C++ driver with RT_EVENT mode can achieve sub-millisecond precision
- Intervals are **elapsed-time based**, making execution deterministic and repeatable

**Overrun handling:**

- If processing takes longer than the interval period, the next interval fires on the
  next eligible tick
- Missed intervals are skipped (not queued up)

#### Wait Node - One-Shot Delays

```arc
sequence test {
    stage hold {
        // Wait 5 seconds before proceeding
        wait{duration=5s} => next
    }
    stage proceed {
        // Continue here after delay
    }
}
```

**How wait works:**

1. On first execution, records the start time
2. Fires once when `elapsed - startTime >= duration`
3. Never fires again until reset
4. **Resets when the stage is re-entered** (allows re-use in loops)

**Use cases:**

- Timed holds in test sequences ("pressurize for 30 seconds")
- Delays before stage transitions
- Timeout conditions

| Node       | Config               | Output | Behavior                   |
| ---------- | -------------------- | ------ | -------------------------- |
| `interval` | `period: TimeSpan`   | u8     | Fires repeatedly at period |
| `wait`     | `duration: TimeSpan` | u8     | Fires once after duration  |

#### Timing and Real-Time Performance

Arc's timing model is **elapsed-time based**, not wall-clock based. This means:

- **Deterministic**: The same program produces the same results given the same inputs
- **Testable**: You can simulate time progression in tests
- **Portable**: Works the same on Go runtime (server) and C++ runtime (driver)

For real-time hardware control, deploy to the **C++ driver runtime** which provides:

- Sub-millisecond timing precision with RT_EVENT mode
- SCHED_FIFO thread priority for consistent scheduling
- Support for control loops up to 1kHz

### Statistical Functions

```arc
// avg - running average with optional reset conditions
sensor -> avg{duration=10s} -> avg_display     // reset every 10s
sensor -> avg{count=100} -> avg_display        // reset every 100 samples

// min/max - running min/max with same reset options
sensor -> min{duration=1min} -> min_display
sensor -> max{count=50} -> max_display
```

| Function | Config                | Purpose         |
| -------- | --------------------- | --------------- |
| `avg`    | `duration?`, `count?` | Running average |
| `min`    | `duration?`, `count?` | Running minimum |
| `max`    | `duration?`, `count?` | Running maximum |

All statistical functions support a `reset` input signal (u8) for manual reset.

### Telemetry I/O Nodes

```arc
// on - read from a Synnax channel (source node)
on{channel=ox_pt_1} -> processor{}

// write - write to a Synnax channel (sink node)
processor{} -> write{channel=ox_pt_cmd}
```

| Node    | Config         | Purpose                  |
| ------- | -------------- | ------------------------ |
| `on`    | `channel: u32` | Read from Synnax channel |
| `write` | `channel: u32` | Write to Synnax channel  |

### Signal Processing Nodes

```arc
// select - split signal based on condition
condition -> select{} -> { true: handler_a{}, false: handler_b{} }

// stable_for - only output when value stable for duration (debounce)
noisy_sensor -> stable_for{duration=100ms} -> stable_output
```

| Node         | Config               | Purpose                         |
| ------------ | -------------------- | ------------------------------- |
| `select`     | -                    | Route by condition (true/false) |
| `stable_for` | `duration: TimeSpan` | Debounce/filter noisy signals   |

### Status/Notification Node

Arc can update statuses that trigger notifications in the Console.

**Workflow:**

1. Create a status in Synnax Console (appears in status toolbar)
2. Reference that status in Arc using `set_status`
3. When Arc updates the status, it sends a notification and updates the toolbar

```arc
// Update status when threshold exceeded
ox_pt_1 > 500 -> set_status{statusKey="overpressure", variant="error", message="Pressure exceeded limit"}
```

| Config      | Options                             | Purpose                   |
| ----------- | ----------------------------------- | ------------------------- |
| `statusKey` | String (status name)                | Which status to update    |
| `variant`   | `"success"`, `"warning"`, `"error"` | Status severity/color     |
| `message`   | String                              | Notification message text |

This is the primary use case for graph mode - building visual alarm logic that triggers
notifications when sensor values exceed thresholds.

### Constant Node

```arc
// constant - output a static value
constant{value=100.0} -> setpoint
```

### Math Operations

The `^` operator provides power/exponentiation for all numeric types:

```arc
base := 2
result := base ^ 8    // 256
```

### Series Operations (Element-wise)

All arithmetic and comparison operators work element-wise on series:

```arc
data := [1.0, 2.0, 3.0]

// Arithmetic (series op scalar)
scaled := data * 2.0          // [2.0, 4.0, 6.0]
offset := data + 10.0         // [11.0, 12.0, 13.0]

// Arithmetic (series op series) - must be equal length
sum := data + [4.0, 5.0, 6.0] // [5.0, 7.0, 9.0]

// Comparisons return series u8
mask := data > 2.0            // [0, 0, 1]
```

---

## 8. WebAssembly Compilation and Runtime Environments

Arc compiles to WebAssembly (WASM) and can execute in **two runtime environments**
depending on deployment target.

### Deployment Model

When users deploy an Arc program, they select a **rack** (a logical group of hardware
managed by a driver instance). This determines where the code runs:

```
Console: User creates Arc, selects rack
              ↓
Synnax Server: Routes task to appropriate rack
              ↓
┌─────────────────────────────────────────────────────┐
│  Option A: Go Runtime (Server)                      │
│  - Runs in Synnax server process                    │
│  - General-purpose automation                       │
│  - No real-time guarantees                          │
└─────────────────────────────────────────────────────┘
              OR
┌─────────────────────────────────────────────────────┐
│  Option B: C++ Runtime (Driver)                     │
│  - Runs in driver process on hardware machine       │
│  - Real-time execution with pthread scheduling      │
│  - Sub-millisecond latency possible                 │
└─────────────────────────────────────────────────────┘
```

### Go Runtime (Server-Side)

**Use case**: Service-level automation, non-critical control, metadata management

- Executes using **wazero** (Go WASM runtime)
- Runs in Synnax server goroutines
- Shares server resources (no isolation)
- GC pauses possible (not suitable for hard real-time)

### C++ Runtime (Driver-Side)

**Use case**: Real-time hardware control loops with timing guarantees

The C++ runtime provides **configurable execution modes** supporting control loops up to
**1kHz**:

| Mode           | Latency | CPU Usage | Use Case                    |
| -------------- | ------- | --------- | --------------------------- |
| `RT_EVENT`     | <1ms    | Low       | Real-time with SCHED_FIFO   |
| `HIGH_RATE`    | Sub-ms  | High      | Software timing, <1ms       |
| `HYBRID`       | 1-5ms   | Medium    | Balanced spin + block       |
| `EVENT_DRIVEN` | >5ms    | Lowest    | Slow intervals or triggered |

**Real-time features** (RT_EVENT mode on Linux):

- `SCHED_FIFO` thread priority (default: 47/99)
- Optional CPU affinity pinning
- Memory locking (`mlock()`) to prevent page faults
- Dedicated high-priority thread per task

### Why Two Runtimes?

| Aspect        | Go Runtime           | C++ Runtime            |
| ------------- | -------------------- | ---------------------- |
| **Location**  | Synnax server        | Driver process         |
| **Real-Time** | No (Go scheduler)    | Yes (SCHED_FIFO)       |
| **Latency**   | Variable (GC pauses) | Deterministic (<1ms)   |
| **Isolation** | Shared with server   | Separate process       |
| **Use Case**  | Meta-level logic     | Hardware control loops |

Both runtimes use the **same reactive execution model** - identical semantics regardless
of deployment target.

### WASM Module Structure

```
WASM Module Contains:
├── Type Signatures (i32, i64, f32, f64)
├── Imports (Host Functions)
│   ├── channel_read_f64(channel_id: i32) -> f64
│   ├── channel_write_i32(channel_id: i32, value: i32) -> void
│   ├── series_len(series_ptr: i32) -> i64
│   ├── state_load_f64(state_id: i32, index: i32) -> f64
│   └── now() -> i64
├── Memory (Linear memory for function execution)
├── Functions (One per Arc function)
└── Exports (Function names)
```

### Type Mapping

| Arc Type            | WASM Type |
| ------------------- | --------- |
| i8, i16, i32, u8-32 | i32       |
| i64, u64            | i64       |
| f32                 | f32       |
| f64                 | f64       |

---

## 9. Console Integration

### Entry Points for Users

1. **Navigation Drawer**: Click "+" button in Arc toolbar (keyboard: "A")
2. **Hardware Rack Context Menu**: Right-click rack → "Create Arc automation"
3. **Command Palette**: Search "Create an Arc automation"
4. **Arc Explorer**: Click "Create an Arc" in empty state

### Editor Modes

**Graph Editor** (Visual):

- Drag-and-drop stages from toolbar
- Connect stages with edges
- Property panel for configuration
- Undo/redo (30-level history)
- Copy/paste, multi-select, alignment tools

**Text Editor** (built into Console):

- Syntax highlighting
- Auto-complete for channels and keywords
- Inline error diagnostics
- Hover information

### Deployment Process

1. Edit Arc in graph or text mode
2. Select target hardware rack from dropdown
3. Click "Configure" to save to Synnax server
4. Click "Play" to start execution
5. Monitor status (Running/Stopped/Error)

### File Format

- Extension: `.arc`
- Export: JSON containing full state (graph, text, mode, viewport)

---

## 10. Example Programs

### Basic Calculation

```arc
func calc(val f32) f32 {
    return val * 2
}

ox_pt_1 -> calc{} -> ox_pt_doubled
```

### Stateful Counter

```arc
func counter() i64 {
    count $= 0
    count = count + 1
    return count
}

trigger -> counter{} -> output
```

### Rocket Engine Test Sequence

```arc
sequence main {
    stage pressurize {
        1 -> pressure_valve,
        pressure > 500.0 => next,
        abort_btn => abort
    }

    stage ignite {
        igniter_cmd = 1,
        flame_detected => next
    }

    stage shutdown {
        0 -> pressure_valve,
        pressure < 50.0 => complete
    }

    stage complete {
    }
}

sequence abort {
    stage safed {
        all_valves_cmd = 0
    }
}

start_cmd => main
emergency_stop => abort
```

---

## 11. Unique Features Summary

| Feature                  | Description                                | Other Languages |
| ------------------------ | ------------------------------------------ | --------------- |
| `^` Exponentiation       | `2 ^ 8 = 256` (not XOR)                    | Python `**`     |
| `$=` Stateful Variables  | Persists across invocations                | None            |
| `=>` One-Shot Transition | Fires once, resets on stage re-entry       | None            |
| `->` Continuous Flow     | Reactive dataflow operator                 | None            |
| Series Elementwise Ops   | `[1,2,3] * 2 = [2,4,6]`                    | NumPy           |
| Unit Types               | `f64 m/s` - dimensional analysis built-in  | F#, Frink       |
| Config Parameters `{}`   | Compile-time constants                     | None            |
| Reactive Iteration       | Use stateful variables + reactive triggers | Event-driven    |

---

## 12. Current Constraints

1. **No mixed-type arithmetic** - explicit casts required
2. **No dynamic instantiation** - config blocks evaluated at compile time
3. **No assignment in expressions** - separate statements required
4. **No partial function application** - all args required
5. **No closures** - functions cannot capture enclosing scope
6. **No nested functions** - flat function definitions
7. **No implicit returns** - explicit return statements required

---

## 13. Error Handling

### Compile-Time Errors

The compiler catches these before deployment:

- Type errors and mismatches
- Undefined identifiers
- Duplicate declarations
- Missing returns
- Flow graph cycles (except valid state transitions)
- Dimensional incompatibilities

### Runtime Errors

These errors stop the Arc task and report status (they do NOT crash the rack or driver):

- Division/modulo by zero
- Out-of-bounds array/string access
- Type mismatches on channels

**Error Isolation**: A runtime error in one Arc task only stops that task. Other tasks
and the hardware driver continue running normally. The error status is reported to the
Console where users can see what went wrong.

### Debugging Arc Programs

- **Console status**: Check task status for error messages
- **LSP diagnostics**: Text editor shows compile-time errors inline
- **Status nodes**: Use `set_status` to output debug information during execution
- **Channel inspection**: Monitor channel values in Console to trace data flow

---

## 14. Key Source Locations

| Purpose          | Path                                |
| ---------------- | ----------------------------------- |
| Language Spec    | `/arc/docs/spec.md`                 |
| Grammar          | `/arc/go/parser/ArcParser.g4`       |
| Lexer            | `/arc/go/parser/ArcLexer.g4`        |
| Analyzer         | `/arc/go/analyzer/analyzer.go`      |
| Compiler         | `/arc/go/compiler/compiler.go`      |
| IR               | `/arc/go/ir/ir.go`                  |
| Runtime          | `/arc/go/runtime/`                  |
| LSP              | `/arc/go/lsp/`                      |
| Tests            | `/arc/go/compiler/compiler_test.go` |
| Console Editor   | `/console/src/arc/`                 |
| Syntax Highlight | `/arc/ts/src/arc.tmLanguage.json`   |
