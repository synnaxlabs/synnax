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
ChannelType ::= ('chan' | '<-chan' | '->chan') (PrimitiveType | SeriesType)
```

Directional constraints (`<-chan` read-only, `->chan` write-only) can only be used in
function/stage parameters, not variable declarations.

### Series Types

```
SeriesType ::= 'series' PrimitiveType
SeriesLiteral ::= '[' ExpressionList? ']'
```

**Series are immutable** homogeneous arrays with elementwise operations:

```arc
data := [1.0, 2.0, 3.0]
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
**Rules**:
- Widening (e.g., `i16(i8_val)`) is safe (sign/zero extend)
- Narrowing (e.g., `i8(i64_val)`) truncates
- Signed ↔ Unsigned saturates at bounds
- Float → Integer truncates toward zero, saturates on overflow
- Integer overflow uses two's-complement wrapping

## Channel Operations

### Operation Grammar

```

ChannelOperation ::= ChannelWrite | ChannelRead ChannelWrite ::= Expression '->'
Identifier | Identifier '<-' Expression ChannelRead ::= Identifier ':=' '<-' Identifier
// blocking | Identifier ':=' Identifier // non-blocking

````

**Blocking read** (`<-chan`) removes oldest value from queue and waits if empty.
**Non-blocking read** (`chan`) returns newest value immediately (or zero if never
written).

```arc
value := <-sensor     // block until next value
current := sensor     // get latest value now
42 -> output          // write to channel
````

### Channel Semantics

Channels are unbounded FIFO queues. All channels internally carry series; scalar
operations are convenience wrappers:

- Scalar write `42 -> ch` creates single-element series `[42]`
- Scalar read `<-ch` returns first element of next series

**Snapshot guarantee**: All non-blocking reads within a stage activation see the same
channel state taken at activation start.

**Event ordering**: Multiple writes in same tick are deterministically ordered by (1)
timestamp, (2) topological order, (3) channel identifier.

### Channel Piping

In inter-stage layer, `sensor -> stage{}` creates reactive connection triggering stage
execution. Inside functions/stages, `sensor -> display` is a simple read-then-write.

## Variables

### Declaration and Assignment

```
LocalVariable ::= Identifier ':=' Expression
                | Identifier Type ':=' Expression

StatefulVariable ::= Identifier '$=' Expression
                   | Identifier Type '$=' Expression

Assignment ::= Identifier '=' Expression
```

**Local variables** (`:=`) reset on each function/stage invocation. **Stateful
variables** (`$=`) persist across stage activations.

```arc
count := 0           // local
total $= 0           // stateful
count = count + 1    // reassignment
```

**Rules**: Variables are function/stage scoped only. No shadowing of global names.
Declaration once per scope. Type inference from initial value.

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

## Functions

### Function Grammar

```
FunctionDeclaration ::= 'func' Identifier '(' ParameterList? ')' ReturnType? Block

Parameter ::= Identifier Type

ReturnType ::= Type                           // single return
             | '{' NamedOutput* '}'          // multi-output

NamedOutput ::= Identifier Type
```

### Examples

```arc
func add(x f64, y f64) f64 {
    return x + y
}

func process(sensor <-chan f64, alarm ->chan u8) {
    if (<-sensor) > 100 {
        1 -> alarm
    }
}

// Multi-output function
func threshold{} (value f64) {
    above f64
    below f64
} {
    if (value > 50) {
        above = value
    } else {
        below = value
    }
}
```

**Restrictions**: No recursion, no closures, no nested functions, no function variables.
Functions are strict on argument count (extra arguments = compile error).

## Stages

### Stage Grammar

```
StageDeclaration ::= 'stage' Identifier ConfigBlock? '(' ParameterList? ')' ReturnType? Block

ConfigBlock ::= '{' ConfigParameter* '}'
ConfigParameter ::= Identifier Type
```

Stages are reactive computation units with configuration and runtime parameters.

### Stage Definition

```arc
stage controller{
    setpoint f64              // config: static at instantiation
    sensor <-chan f64         // config: input channel
    actuator ->chan f64       // config: output channel
} (enable u8) {              // runtime parameter
    integral $= 0.0           // stateful variable

    if not enable {
        0 -> actuator
        return
    }

    error := setpoint - (<-sensor)
    integral = integral + error
    output := (error * 1.5) + (integral * 0.1)
    output -> actuator
}
```

### Stage Invocation

```arc
controller{
    setpoint: 100,
    sensor: temp_sensor,
    actuator: valve_cmd
}(1)
```

### Stage Execution Model

Stages are **purely event-driven** - they execute when receiving input values:

- **First execution**: When ALL input channels have received at least one value
- **Subsequent executions**: When ANY input channel receives new value
- No implicit intervals - use `interval{}` stdlib stage for periodic execution
- Snapshot semantics: all non-blocking reads see same activation state

**Return values**: Stages with return type create anonymous output channel. Multi-output
stages can route to different targets (see Inter-Stage Layer).

**Restrictions**: No recursion. Extra runtime arguments silently ignored (unlike
functions). Config parameters must be literals or channel identifiers (compile-time
constants only).

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

## Inter-Stage Layer

### Flow Grammar

```
FlowStatement ::= (RoutingTable | FlowNode) (ARROW (RoutingTable | FlowNode))+ ';'?

RoutingTable ::= '{' RoutingEntry (',' RoutingEntry)* '}'

RoutingEntry ::= Identifier ':' FlowNode (ARROW FlowNode)* (':' Identifier)?

FlowNode ::= ChannelIdentifier | StageInvocation | Expression
```

### Data Flow and Routing

The inter-stage layer connects stages via channels and expressions:

```arc
// Simple pipeline
sensor -> filter{threshold: 50} -> controller{} -> actuator

// Input routing: map channels to named parameters
{
    setpoint: setpoint_chan,
    measured: temp_sensor
} -> pid_controller{kp: 1.0}

// Output routing: route named outputs to different targets
alarm_detector{} -> {
    low_alarm: heater_on{},
    high_alarm: cooler_on{},
    normal: logger{}
}

// Combined routing
{
    temp_sensor_1: sensor_a,
    temp_sensor_2: sensor_b
} -> averager{} -> threshold_check{} -> {
    above: alarm{},
    below: logger{}
}
```

**Inline expressions** can act as implicit stages in flows:

```arc
temperature > 100 -> alarm{}             // comparison
(sensor1 + sensor2) / 2 -> display       // arithmetic
pressure > 100 or emergency -> shutdown{} // logical
```

Expressions can only reference channels and literals, not variables.

### Standard Library Stages

Built-in stages implemented by runtime:

- `interval{period: timespan}` - Emits at regular intervals
- `all{inputs...}` - Waits for all inputs, emits when last arrives
- `once{input}` - Emits only first value
- `any{inputs...}` - Emits immediately on any input
- `throttle{rate: frequency}` - Rate limits, drops intermediate values
- `merge{inputs...}` - Forwards any input immediately
- `tee{outputs...}` - Splits stream to multiple targets

### Cycle Detection

Stage graphs must be acyclic. Cycles detected at compile time via DFS. Self-loops and
circular dependencies are forbidden.

## Naming and Scoping

**Global namespace**: All stages, functions, and external channels must have unique
names.

**Variable scoping**: Variables are function/stage scoped. Cannot shadow global names.

**Channel declaration**: Channels are external to Arc, referenced by name.

**Stage return values**: Stages with return type create anonymous output channels.
Multi-output stages create multiple named outputs.

## Language Restrictions

These simplify implementation while maintaining expressiveness:

1. **No mixed-type arithmetic**: Explicit casts required (`f32(x) + y`)
2. **No nested function calls**: Use intermediate variables
3. **No dynamic stage creation**: All stages instantiated at compile time
4. **No assignment in expressions**: Separate statements required
5. **No partial function application**: Must provide all arguments
6. **Stage config = compile-time constants**: Only literals/channel IDs in config

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
- **Exports**: One function per stage/function
- **Memory**: Optional linear memory for multi-output stages (1 page = 64KB)

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
  "stages": [
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

Multi-output stages/functions use reserved memory regions:

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

Resolves external channels, stdlib stages, and stdlib functions during analysis.

### Runtime Responsibilities

Runtime (Synnax server) handles:

- Reactive scheduling with event queue
- Channel management (unbounded FIFO queues)
- Stage lifecycle and state persistence
- Memory management for WASM modules
- Flow graph execution in stratified order
- Snapshot semantics for non-blocking reads
