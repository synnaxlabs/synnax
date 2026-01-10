# Arc Language Specification

Arc is a reactive automation language for hardware control and telemetry systems.
Compiles to WebAssembly for embedded execution in the Synnax platform.

## Comments

```
Comment ::= SingleLineComment | MultiLineComment
SingleLineComment ::= '//' [^\n]*
MultiLineComment ::= '/*' .*? '*/'
```

## Type System

### Primitive Types

```
Type ::= PrimitiveType | ChannelType | SeriesType

PrimitiveType ::= NumericType | 'str'

NumericType ::= IntegerType | FloatType | TemporalType

IntegerType ::= 'i8' | 'i16' | 'i32' | 'i64' | 'u8' | 'u16' | 'u32' | 'u64'

FloatType ::= 'f32' | 'f64'

TemporalType ::= 'timestamp' | 'timespan'

StringLiteral ::= '"' UTF8Character* '"'
```

**Type defaults**: Integer literals default to `i64`, float literals to `f64`.

### Channel Types

```
ChannelType ::= 'chan' (PrimitiveType | SeriesType)
```

### Series Types

```
SeriesType ::= 'series' PrimitiveType
SeriesLiteral ::= '[' ExpressionList? ']'
```

**Series** are homogeneous arrays with elementwise operations:

```arc
data := [1.0, 2.0, 3.0]
data[0] = 5.0                    // element assignment
length := len(data)              // i64
first := data[0]                 // indexing
subset := data[1:3]              // slicing [2.0, 3.0]
scaled := data * 2.0             // [2.0, 4.0, 6.0]
sum := data + [4.0, 5.0, 6.0]   // [5.0, 7.0, 9.0]
mask := data > 2.0               // [0, 0, 1] (series u8)
```

**Rules**:

- Out-of-bounds access = runtime error
- Binary ops require equal-length series
- Scalar ops require explicit type cast (e.g., `data + f32(10)`)
- Comparisons return `series u8` elementwise
- Empty `[]` requires type annotation

### Numeric Literals

```
NumericLiteral ::= IntegerLiteral | FloatLiteral
IntegerLiteral ::= Digit+                    // defaults to i64
FloatLiteral ::= Digit+ '.' Digit* | '.' Digit+  // defaults to f64
```

Examples: `42`, `3.14`, `u8(255)`, `f32(1.5)`

### Temporal Literals

```
TemporalLiteral ::= Number TemporalUnit | Number FrequencyUnit
TemporalUnit ::= 'ns' | 'us' | 'ms' | 's' | 'm' | 'h'
FrequencyUnit ::= 'hz' | 'khz' | 'mhz'
```

Examples: `100ms`, `5s`, `1m` (minute, not meter), `10hz` (= 100ms), `1khz` (= 1ms)

**Note**: `m` is always minutes. Frequency units convert to timespan by inverting
period.

### Strings

Strings are immutable UTF-8 sequences. Only `==` and `!=` operators supported. No
concatenation, indexing, or length function.

```arc
msg := "Hello"
equal := msg == "Hello"  // 1 (true)
```

### Boolean Semantics

Type `u8` serves as boolean: `0` is false, non-zero is true. Logical operators normalize
to `0` or `1` with short-circuit evaluation.

```arc
result := 2 and 3       // 1 (both truthy)
result := 5 or 0        // 1 (short-circuits)
negated := not 5        // 0
```

### Zero Values

All types have default zero values: integers/floats `0`, string `""`, timestamp/timespan
`0`, channels return zero on first read before write.

### Type Casting

Explicit casting between numeric types:

```
TypeCast ::= Type '(' Expression ')'
```

**Rules**:

- Widening (e.g., `i16(i8_val)`) is safe (sign/zero extend)
- Narrowing (e.g., `i8(i64_val)`) truncates
- Signed ↔ Unsigned saturates at bounds
- Float → Integer truncates toward zero, saturates on overflow
- Integer overflow uses two's-complement wrapping

## Channel Operations

Channel operations differ between **reactive context** (flow/stages) and **imperative
context** (function bodies).

### Reactive Context (Flow/Sequence Layer)

```

FlowStatement ::= ChannelRead '=>' Identifier // Blocking read triggering state
transition | ChannelWrite '->' Identifier // Write to channel in reactive flow |
Identifier '->' FunctionCall // Connect channel to function

```

```arc
stage monitor {
    sensor => process_reading{},     // Wait for sensor value, then trigger
    process{} -> output,             // Connect function output to channel
    interval{100ms} -> tick_handler, // Trigger on interval
}
```

### Imperative Context (Function Bodies)

```
Statement ::= ChannelRead | ChannelWrite | ...

ChannelRead ::= Identifier ':=' Identifier        // Non-blocking read
ChannelWrite ::= Identifier '=' Expression        // Channel write (assignment)
```

**Non-blocking read** (`channel`) returns newest value immediately or zero if never
written:

```arc
func process() bool {
    current := sensor           // Get latest value (non-blocking)
    return current > threshold
}
```

**Channel write** (`channel = value`) enqueues value to channel.

```arc
func initialize() u8 {
    tpc_cmd = 0                // Enqueue value to channel
    mpv_cmd = 0
    vent_cmd = 1
    return 1
}
```

### Channel Semantics

Channels are unbounded FIFO queues. All channels internally carry series; scalar
operations are convenience wrappers:

- Scalar write `ch = 42` creates single-element series `[42]` and enqueues it
- Non-blocking scalar read `ch` returns first element of newest series (or zero)

**Non-blocking snapshot guarantee**: All non-blocking reads within a function invocation
see the same channel state taken at invocation start.

**Event ordering**: Multiple writes in same tick are deterministically ordered by (1)
timestamp, (2) topological order, (3) channel identifier.

**Syntax summary**:

- **Non-blocking reads** (`channel`): Only in imperative context; return current value
- **Writes** (`channel = value`): Only in imperative context; enqueue to channel
- **Reactive flow** (`channel -> function`): Only in reactive context; connect dataflow

### Channel Piping in Reactive Context

In flow layer, `sensor -> func{}` creates reactive connection triggering function
execution when sensor value changes. Multiple functions can be chained:

```arc
sensor -> transform{} -> filter{} -> output_channel
```

## Variables

### Declaration and Assignment

```
LocalVariable ::= Identifier ':=' Expression
                | Identifier Type ':=' Expression

StatefulVariable ::= Identifier '$=' Expression
                   | Identifier Type '$=' Expression

Assignment ::= Identifier '=' Expression
```

**Local variables** (`:=`) reset on each function invocation. **Stateful variables**
(`$=`) persist across function invocations.

```arc
count := 0           // local
total $= 0           // stateful
count = count + 1    // reassignment
```

**Rules**: Variables are function scoped only. No shadowing of global names. Declaration
once per scope. Type inference from initial value.

## Operators

### Expression Grammar and Precedence

```
Expression ::= UnaryExpression | BinaryExpression | PrimaryExpression

UnaryOperator ::= '-' | 'not'
BinaryOperator ::= ArithmeticOp | ComparisonOp | LogicalOp
ArithmeticOp ::= '+' | '-' | '*' | '/' | '%' | '^'
ComparisonOp ::= '==' | '!=' | '<' | '>' | '<=' | '>='
LogicalOp ::= 'and' | 'or'
```

**Precedence** (highest to lowest):

1. `^` (right-associative)
2. `-`, `not` (unary, right-associative)
3. `*`, `/`, `%` (left-associative)
4. `+`, `-` (left-associative)
5. `<`, `>`, `<=`, `>=`, `==`, `!=`
6. `and`, `or` (short-circuit)

**Note**: `^` is exponentiation, not XOR. No bitwise operations.

Examples:

```arc
power := 2 ^ 8           // 256
neg := -2 ^ 2            // -4 (^ binds tighter than unary -)
remainder := 10 % 3      // 1
in_range := temp >= 20 and temp <= 30
```

## Built-in Functions

Arc provides the following built-in functions:

- `len(series)` - Returns the length of a series as `i64`
- `now()` - Returns the current timestamp

```arc
data := [1.0, 2.0, 3.0]
length := len(data)      // 3 (i64)
current := now()         // current timestamp
```

## Functions

### Function Grammar

```
FunctionDeclaration ::= 'func' Identifier ConfigBlock? '(' ParameterList? ')' ReturnType? Block

ConfigBlock ::= '{' ConfigParameter* '}'
ConfigParameter ::= Identifier Type

Parameter ::= Identifier Type ('=' Literal)?  // Optional default value

ReturnType ::= Type                           // single unnamed return
             | Identifier Type                // single named return
             | '(' NamedOutput (',' NamedOutput)* ')'  // multi-output

NamedOutput ::= Identifier Type
```

### Configuration Parameters

Functions can have a **config block** containing parameters that are set at
instantiation time (compile-time constants). Config parameters are enclosed in `{}`
after the function name:

```arc
func controller{
    setpoint f64              // config: static at instantiation
    sensor chan f64         // config: input channel
    actuator chan f64       // config: output channel
} (enable u8) f64 {
    // function body
}
```

**Rules:**

- Config parameters must be literals or channel identifiers (compile-time constants
  only)
- Config values are provided at instantiation using `=` syntax:
  `controller{setpoint=100}`

### Optional Parameters

Parameters can be made optional by providing a default value using the `=` operator.
Optional parameters must appear after all required parameters (trailing-only
constraint).

**Rules:**

- Default values must be literals (numeric, float, or temporal constants)
- Optional parameters must follow all required parameters
- When a parameter with a default is omitted at the call site, the default value is used

**Example:**

```arc
func add(x f64, y f64 = 0) f64 {
    return x + y
}

func configure(threshold f64, min f64 = 0.0, max f64 = 100.0) {
    // min and max are optional with defaults
}
```

### Examples

```arc
func add(x f64, y f64) f64 {
    return x + y
}

func process(sensor chan f64, alarm chan u8) {
    if (sensor) > 100 {
        1 -> alarm
    }
}

// Function with config block
func controller{
    setpoint f64
    sensor chan f64
    actuator chan f64
} (enable u8) {
    integral $= 0.0           // stateful variable persists across invocations

    if not enable {
        0 -> actuator
        return
    }

    error := setpoint - sensor
    integral = integral + error
    output := (error * 1.5) + (integral * 0.1)
    output -> actuator
}

// Multi-output function
func threshold(value f64) (above f64, below f64) {
    if value > 50 {
        above = value
    } else {
        below = value
    }
}

// Function with optional parameters
func clamp(value f64, min f64 = 0.0, max f64 = 1.0) f64 {
    if value < min {
        return min
    }
    if value > max {
        return max
    }
    return value
}
```

**Restrictions**: No recursion, no closures, no nested functions, no function variables.
Functions are strict on argument count, but optional parameters may be omitted.

### Stateful Variables

Functions can declare **stateful variables** using `$=` that persist across invocations:

```arc
func counter() i64 {
    count $= 0           // persists across calls
    count = count + 1
    return count
}
```

Stateful variables are initialized on first invocation and retain their values between
subsequent invocations. This enables reactive patterns without explicit loops.

### Reactive Execution Model

Functions with config blocks are **purely event-driven** - they execute when receiving
input values on their input channels:

- **First execution**: When ALL input channels have received at least one value
- **Subsequent executions**: When ANY input channel receives new value
- No implicit intervals - use `interval{period=100ms}` for periodic execution
- Snapshot semantics: all non-blocking reads within an invocation see same state

**Return values**: Functions with return type create anonymous output channels.
Multi-output functions can route outputs to different targets (see Flow Layer).

## Control Flow

```
IfStatement ::= 'if' Expression Block ElseIfClause* ElseClause?
ElseIfClause ::= 'else' 'if' Expression Block
ElseClause ::= 'else' Block
```

Only conditional statements supported. No loops (reactive model handles iteration via
events + stateful variables).

```arc
if pressure > 100 {
    true -> alarm
} else if pressure > 80 {
    true -> warning
} else {
    false -> alarm
}
```

## Flow Layer

### Flow Grammar

```
FlowStatement ::= (RoutingTable | FlowNode) (ARROW (RoutingTable | FlowNode))+

RoutingTable ::= '{' RoutingEntry (',' RoutingEntry)* '}'

RoutingEntry ::= Identifier '=' FlowNode (ARROW FlowNode)*

FlowNode ::= ChannelIdentifier | FunctionInvocation | Expression
```

### Data Flow and Routing

The flow layer connects functions via channels and expressions:

```arc
// Simple pipeline
sensor -> filter{threshold=50} -> controller{} -> actuator

// Input routing: map channels to named parameters
{
    setpoint=setpoint_chan,
    measured=temp_sensor
} -> pid_controller{kp=1.0}

// Output routing: route named outputs to different targets
alarm_detector{} -> {
    low_alarm=heater_on{},
    high_alarm=cooler_on{},
    normal=logger{}
}

// Combined routing
{
    temp_sensor_1=sensor_a,
    temp_sensor_2=sensor_b
} -> averager{} -> threshold_check{} -> {
    above=alarm{},
    below=logger{}
}
```

**Inline expressions** can act as implicit functions in flows:

```arc
temperature > 100 -> alarm{}             // comparison
(sensor1 + sensor2) / 2 -> display       // arithmetic
pressure > 100 or emergency -> shutdown{} // logical
```

Expressions can only reference channels and literals, not variables.

### Runtime-Provided Functions

Built-in functions implemented by the Synnax runtime:

- `interval{period=100ms}` - Emits at regular intervals
- `all{inputs...}` - Waits for all inputs, emits when last arrives
- `once{input}` - Emits only first value
- `any{inputs...}` - Emits immediately on any input
- `throttle{rate=10hz}` - Rate limits, drops intermediate values
- `merge{inputs...}` - Forwards any input immediately
- `tee{outputs...}` - Splits stream to multiple targets

### Cycle Detection

Flow graphs must be acyclic. Cycles detected at compile time via DFS. Self-loops and
circular dependencies are forbidden.

## Sequences

Sequences extend Arc's reactive model to support sequential automation workflows like
test sequences, state machines, and ordered procedures.

### Core Concepts

**Sequence**: A state machine containing ordered stages. Only one stage is active at a
time.

**Stage**: A state within a sequence. When active, its reactive flows execute; when
inactive, they don't.

**Two edge types**:

- `->` (Continuous): Reactive flow that runs while the stage is active
- `=>` (EdgeKindOneShot): Fires once when condition becomes true, then doesn't fire again until
  stage is re-entered

### Sequence Syntax

```
SequenceDeclaration ::= 'sequence' Identifier '{' StageDeclaration+ '}'

StageDeclaration ::= 'stage' Identifier '{' StageItem* '}'

StageItem ::= ReactiveFlow | EdgeKindOneShotTransition

ReactiveFlow ::= Expression '->' FlowTarget

EdgeKindOneShotTransition ::= Expression '=>' TransitionTarget
```

### Example

```arc
sequence main {
    stage pressurize {
        // Reactive flows: run continuously while stage is active
        interval{100ms} -> pressure_control{target=500},

        // One-shot transitions: fire once when condition is true
        pressure > 500 => next,
        wait{30s} => abort,
        abort_btn => abort
    }

    stage ignite {
        1 -> igniter_cmd,
        flame_detected => next,
        wait{2s} => abort
    }

    stage complete {
        log{"Sequence complete"}
    }
}

sequence abort {
    stage safed {
        0 -> all_valves_cmd,
        log{"System safed"}
    }
}
```

### Definition Order and `next`

The `next` keyword resolves to the next stage in definition order:

```arc
stage step1 { ... => next }  // next = step2
stage step2 { ... => next }  // next = step3
stage step3 { ... }          // terminal (no outgoing transitions)
```

**Best practice**: Place happy-path stages first in order, exception handlers (holds) at
the end. Avoid `next` in exception handlers—use explicit stage names.

Using `next` on a stage with no following stage is a compile-time error.

### Transition Targets

- `=> next` — Go to the next stage in definition order
- `=> stage_name` — Jump to any stage in the same sequence
- `=> sequence_name` — Jump to a different sequence (starts at its first stage)

### Timer Built-ins

**`wait{duration}`**: One-shot timer. Returns false until duration elapses, then true
once.

**`interval{period}`**: Repeating timer. Fires every period.

```arc
interval{100ms} -> control{}       // Continuous: runs every 100ms
wait{5s} => abort                   // One-shot: triggers after 5s
wait{10ms} => 1 -> valve_cmd        // One-shot action after 10ms
```

Timers reset when their stage is entered (including re-entry from holds).

### Reactive vs One-Shot Semantics

**Reactive flows (`->`)**: Execute every time the source produces a value while the
stage is active.

**One-shot transitions (`=>`)**: Execute once when the condition becomes true, then
stop. The "one-shot" state resets when the stage is re-entered.

```arc
stage example {
    interval{100ms} -> control{},   // Runs every 100ms
    wait{10ms} => 1 -> valve_cmd    // Runs once at 10ms, then done
}
```

### Transition Priority

When multiple transition conditions become true simultaneously, the first listed wins.
List safety-critical conditions first:

```arc
stage pressurize {
    // 1. Safety (highest priority)
    pressure > max => abort,

    // 2. Operator controls
    hold_btn => hold,

    // 3. Nominal completion
    pressure > target => next,

    // 4. Timeout (lowest priority)
    wait{30s} => abort
}
```

### Stage Entry Semantics

When entering a stage:

1. All `wait{}` timers reset to zero
2. All reactive flows start fresh
3. One-shot transition states reset (can fire again)

Stages are stateless between entries—no implicit memory of previous time in the stage.

### Cross-Sequence Transitions

When transitioning to another sequence (e.g., `=> abort`):

1. Source sequence terminates entirely (flows stop, timers cancelled)
2. Target sequence starts at its first defined stage
3. This is one-way—no built-in "return" mechanism

### Imperative Blocks with Match

For complex entry logic, use an imperative block with match routing:

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
    }
}
```

**Warning**: Imperative blocks block the entire stage, including abort checks. Keep them
fast (< 10ms).

### Top-Level Entry Points

Entry points connect external events to sequences:

```arc
start_cmd => main           // Channel triggers sequence
emergency_stop => abort     // Multiple entries allowed
```

The sequence starts when the source produces any value. The value is discarded; only the
event matters.

## Naming and Scoping

**Global namespace**: All functions and external channels must have unique names.

**Variable scoping**: Variables are function scoped. Cannot shadow global names.

**Channel declaration**: Channels are external to Arc, referenced by name.

**Function return values**: Functions with return type create anonymous output channels.
Multi-output functions create multiple named outputs.

## Language Restrictions

These simplify implementation while maintaining expressiveness:

1. **No mixed-type arithmetic**: Explicit casts required (`f32(x) + y`)
2. **No dynamic function instantiation**: All functions with config blocks are
   instantiated at compile time
3. **No assignment in expressions**: Separate statements required
4. **No partial function application**: Must provide all arguments
5. **Config = compile-time constants**: Only literals/channel IDs in config blocks

## Error Handling

### Compile-Time Errors

- **Type errors**: Mismatches, invalid casts, bare `[]` without type
- **Name resolution**: Undefined identifiers, duplicates, shadowing
- **Structural**: Missing returns, unreachable code, wrong argument count
- **Flow graph**: Cycles, unconnected inputs, type mismatches in edges

### Runtime Errors

- **Arithmetic**: Division/modulo by zero
- **Array**: Out-of-bounds access, length mismatch in operations
- **Channel**: Type mismatches (if not statically verified)

## Compilation Target

Arc compiles exclusively to WebAssembly (WASM).

### WASM Module Structure

Generated module contains:

- **Imports**: Host functions for channel ops, series ops, state persistence, builtins
- **Exports**: One exported WASM function per Arc function
- **Memory**: Optional linear memory for multi-output functions (1 page = 64KB)

Example imports:

```wasm
"env"."channel_read_f64": [i32] -> [f64]
"env"."channel_write_i32": [i32, i32] -> []
"env"."series_len": [i32] -> [i64]
"env"."state_load_f64": [i32, i32] -> [f64]
"env"."now": [] -> [i64]
```

**Type mapping**: `i8`-`i32`, `u8`-`u32` → `i32`; `i64`, `u64` → `i64`; `f32` → `f32`;
`f64` → `f64`. WASM uses i32 for all sub-32-bit integers.

### Compilation Output

Compiler produces JSON with base64-encoded WASM module plus metadata:

```json
{
  "version": "1.0.0",
  "module": "<base64-wasm-bytes>",
  "functions": [
    {
      "key": "controller",
      "config": [{ "name": "setpoint", "type": "f64" }],
      "params": [{ "name": "enable", "type": "u8" }],
      "outputs": { "output": "f64" }
    }
  ],
  "nodes": [
    {
      "key": "controller_0",
      "type": "controller",
      "config": { "setpoint": 100 }
    }
  ],
  "edges": [
    {
      "source": { "node": "sensor_0", "param": "output" },
      "target": { "node": "controller_0", "param": "sensor" }
    }
  ]
}
```

### Multi-Output Memory Layout

Multi-output functions use reserved memory regions:

```
[base_addr + 0]: dirty_flags (i64 bitmap)
[base_addr + 8]: output0 value
[base_addr + 8 + sizeof(output0)]: output1 value
...
```

Each multi-output callable gets its own memory region starting at 0x1000.

### Stratified Execution

Runtime computes execution strata for deterministic reactive scheduling. Nodes are
assigned stratum levels via iterative deepening:

- Initialize all nodes to stratum 0
- If node A depends on B, then `stratum(A) = max(stratum(A), stratum(B) + 1)`
- Execute nodes in stratum order (0, 1, 2, ...) each tick

### Compilation Pipeline

```
Parser → AST
  ↓
Analyzer Pass 1: Collect declarations → Symbol table
  ↓
Analyzer Pass 2: Type check + Validation
  ↓
Stratifier: Compute execution levels
  ↓
Compiler Pass 1: Build graph (nodes, edges, specs)
  ↓
Compiler Pass 2: Generate WASM + Metadata
```

### Global Resolver Interface

Analyzer requires a resolver for external symbols:

```go
type SymbolResolver interface {
    Resolve(ctx context.Context, name string) (Symbol, error)
}
```

Resolves external channels and runtime-provided functions during analysis.

### Runtime Responsibilities

Runtime (Synnax server) handles:

- Reactive scheduling with event queue
- Channel management (unbounded FIFO queues)
- Function lifecycle and state persistence
- Memory management for WASM modules
- Flow graph execution in stratified order
- Snapshot semantics for non-blocking reads
