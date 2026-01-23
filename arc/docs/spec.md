# Arc Language Specification

Arc is a reactive automation language for hardware control and telemetry systems.
Compiles to WebAssembly for embedded execution in the Synnax platform.

## Comments

```
SingleLineComment ::= '//' [^\n]*
MultiLineComment ::= '/*' .*? '*/'
```

Comments are allowed anywhere in the source code.

## Type System

### Primitive Types

```
Type ::= PrimitiveType | ChannelType | SeriesType

PrimitiveType ::= NumericType | 'str'

NumericType ::= IntegerType | FloatType

IntegerType ::= 'i8' | 'i16' | 'i32' | 'i64' | 'u8' | 'u16' | 'u32' | 'u64'

FloatType ::= 'f32' | 'f64'
```

**Type defaults**: Integer literals default to `i64`, float literals to `f64`.

### Boolean Semantics

Type `u8` serves as boolean: `0` is false, non-zero is true. Logical operators normalize
to `0` or `1` with short-circuit evaluation.

```arc
result := 2 and 3       // 1 (both truthy)
result := 5 or 0        // 1 (short-circuits)
negated := not 5        // 0
```

### Channel Types

```
ChannelType ::= 'chan' PrimitiveType UnitSuffix?
              | 'chan' SeriesType
```

### Series Types

```
SeriesType ::= 'series' PrimitiveType UnitSuffix?
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
sum := data + [4.0, 5.0, 6.0]    // [5.0, 7.0, 9.0]
mask := data > 2.0               // [0, 0, 1] (series u8)
```

**Rules**:

- Out-of-bounds access = runtime error
- Binary ops require equal-length series
- Comparisons return `series u8` elementwise
- Empty `[]` requires type annotation

### Strings

Strings are immutable UTF-8 sequences supporting:

```arc
msg := "Hello"
greeting := msg + " World"   // concatenation
first := msg[0]              // indexing
sub := msg[1:4]              // slicing
length := len(msg)           // length
equal := msg == "Hello"      // equality (returns u8: 1 or 0)
```

**Supported operations**: `+` (concatenation), `==`, `!=`, indexing, slicing, `len()`.

### Numeric Literals

```
NumericLiteral ::= IntegerLiteral | FloatLiteral
IntegerLiteral ::= Digit+                        // defaults to i64
FloatLiteral ::= Digit+ '.' Digit* | '.' Digit+  // defaults to f64
```

Examples: `42`, `3.14`, `u8(255)`, `f32(1.5)`

### Zero Values

All types have default zero values: integers/floats `0`, string `""`, channels return
zero on first read before write.

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

## Unit System

Arc supports dimensional analysis with unit annotations on types and literals.

### Unit Suffixes on Types

Types can have unit suffixes for dimensional tracking:

```arc
velocity f64 m/s := 10.0
distance f64 m := 50.0
duration i64 ns := 1000000000
```

### Unit Literals

Numeric literals can have unit suffixes (no whitespace between number and unit):

```
TemporalUnit ::= 'ns' | 'us' | 'ms' | 's' | 'm' | 'h'
FrequencyUnit ::= 'hz' | 'khz' | 'mhz'
```

Examples: `100ms`, `5s`, `1m` (minute), `10hz` (= 100ms period), `1khz` (= 1ms period)

**Note**: `m` in temporal context is minutes. Frequency units convert to timespan by
inverting the period.

### Temporal Values

Timestamps and timespans are represented as `i64` with time units (nanoseconds):

```arc
duration := 100ms            // i64 with time units
interval := 5s               // i64 with time units
timestamp := now()           // i64 ns from now() builtin
```

### Dimensional Compatibility

- **Addition/Subtraction**: Operands must have compatible dimensions
- **Multiplication/Division**: Always valid; dimensions combine
- **Exponentiation**: If base has dimensions, exponent must be a literal integer

```arc
distance f64 m := 10.0
time f64 s := 2.0
speed := distance / time     // f64 m/s (dimensions combine)
area := distance ^ 2         // f64 m^2 (literal exponent required)
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
total $= 0           // stateful (persists across invocations)
count = count + 1    // reassignment
```

### Compound Assignment

```arc
count += 1           // count = count + 1
value -= 10          // value = value - 10
total *= 2           // total = total * 2
ratio /= 4           // ratio = ratio / 4
remainder %= 3       // remainder = remainder % 3
```

**Rules**: Variables are function-scoped. No shadowing of global names. Declaration once
per scope. Type inference from initial value.

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
- `len(str)` - Returns the length of a string as `i64`
- `now()` - Returns the current timestamp as `i64` with nanosecond time units

```arc
data := [1.0, 2.0, 3.0]
length := len(data)      // 3 (i64)
msg := "hello"
chars := len(msg)        // 5 (i64)
current := now()         // current timestamp (i64 ns)
```

## Functions

Arc has two execution contexts with different function syntax:

- **Reactive scope** (flow statements): Functions are instantiated as nodes using
  `func{config}` syntax
- **Imperative scope** (function bodies): Functions are called using `func(args)` syntax

### Function Declaration

```
FunctionDeclaration ::= 'func' Identifier ConfigBlock? '(' InputList? ')' OutputType? Block

ConfigBlock ::= '{' ConfigList? '}'
ConfigList ::= ConfigParameter (',' ConfigParameter)* ','?
ConfigParameter ::= Identifier Type

InputList ::= Input (',' Input)*
Input ::= Identifier Type ('=' Literal)?

OutputType ::= Type                                    // single unnamed output
             | Identifier Type                         // single named output
             | '(' NamedOutput (',' NamedOutput)* ')'  // multiple named outputs

NamedOutput ::= Identifier Type
```

### Configuration Parameters

Functions can have a **config block** containing parameters set at instantiation time
(compile-time constants). Config parameters are enclosed in `{}` after the function
name, separated by commas:

```arc
func controller{
    setpoint f64,             // config: static at instantiation
    sensor chan f64,          // config: channel reference
    actuator chan f64         // config: channel reference
} (enable u8) f64 {
    // function body
}
```

**Rules:**

- Config parameters must be literals or channel identifiers (compile-time constants)
- Config values are provided at instantiation using `=` syntax:
  `controller{setpoint=100.0}`
- **Anonymous config values** are only allowed for single-parameter functions:
  `filter{50.0}` is valid, but multi-param functions require named values

### Input Parameters

Input parameters are received at runtime when the function is triggered:

```arc
func add(x f64, y f64) f64 {
    return x + y
}
```

**Optional parameters** can have default values (must be trailing):

```arc
func clamp(value f64, min f64 = 0.0, max f64 = 1.0) f64 {
    if value < min { return min }
    if value > max { return max }
    return value
}
```

### Calling Functions (Imperative Scope)

Inside function bodies, call other functions using parentheses:

```arc
func process(x f64) f64 {
    clamped := clamp(x, 0.0, 100.0)    // function call
    return clamped * 2.0
}
```

### Instantiating Functions (Reactive Scope)

In flow statements, instantiate functions as nodes using config block syntax:

```arc
sensor -> controller{setpoint=100.0, sensor=temp, actuator=valve}
```

Input parameters are automatically wired from the incoming flow. Multiple inputs require
routing tables (see Flow Layer).

### Multi-Output Functions

Functions can have multiple named outputs:

```arc
func threshold(value f64) (above f64, below f64) {
    if value > 50.0 {
        above = value
    } else {
        below = value
    }
}
```

### Stateful Variables

Functions can declare **stateful variables** using `$=` that persist across invocations:

```arc
func counter() i64 {
    count $= 0           // persists across calls
    count = count + 1
    return count
}
```

### Channel Operations in Functions

Functions can read from and write to channels:

```arc
func controller{
    input chan f64,
    output chan f64,
    threshold f64
} () {
    value := input              // read from channel (non-blocking)
    if value > threshold {
        output = value          // write to channel
    }
}
```

Functions can also reference global channels directly by name.

**Restrictions**: No closures, no nested functions. Recursion is allowed.

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
    alarm = 1
} else if pressure > 80 {
    warning = 1
} else {
    alarm = 0
}
```

### Return Statements

```arc
func example() f64 {
    if done {
        return 0.0       // early return with value
    }
    return compute()     // explicit return required
}

func sideEffect() {
    if skip {
        return           // early exit (no value for void functions)
    }
    // more work
}
```

Explicit `return` statements are required for functions with return types. No implicit
returns.

## Flow Layer

The flow layer connects functions via channels in the reactive scope.

### Flow Statement Grammar

```
FlowStatement ::= (RoutingTable | FlowNode) (FlowOperator (RoutingTable | FlowNode))+

FlowOperator ::= '->'       // continuous flow
               | '=>'       // one-shot flow

RoutingTable ::= '{' RoutingEntry (',' RoutingEntry)* '}'

RoutingEntry ::= Identifier ':' FlowNode ('->' FlowNode)* (':' Identifier)?

FlowNode ::= Identifier           // channel, stage, or sequence name
           | FunctionInvocation   // func{config}
           | Expression           // inline computation
           | 'next'               // next stage (sequences only)
```

### Simple Pipelines

```arc
sensor -> filter{threshold=50.0} -> controller{} -> actuator
```

### Output Routing Tables

Route named outputs to different targets:

```arc
sensor -> demux{} -> {
    high: alarm{},
    low: logger{}
}
```

### Input Routing Tables

Map multiple sources to named input parameters:

```arc
{ sensor1: a, sensor2: b } -> combiner{}
```

### Combined Routing

```arc
{ temp1: a, temp2: b } -> averager{} -> threshold{} -> {
    above: alarm{},
    below: logger{}
}
```

### Expressions in Flows

Inline expressions act as implicit functions. Expressions can reference global-scope
identifiers (channels), literals, and function calls—but not function-local variables.

```arc
temperature > 100 -> alarm{}              // comparison
(sensor1 + sensor2) / 2.0 -> display      // arithmetic
pressure > 100 or emergency -> shutdown{} // logical
```

### Cycle Detection

Flow graphs must be acyclic within each scope:

- **Top-level flows** (`->`): Must be acyclic
- **Flows within a stage** (`->`): Must be acyclic
- **Stage transitions** (`=>`): Can be cyclic (valid state machines)

## Sequences

Sequences extend Arc's reactive model to support sequential automation workflows like
test sequences, state machines, and ordered procedures.

### Core Concepts

**Sequence**: A state machine containing ordered stages. Only one stage is active at a
time per sequence.

**Stage**: A state within a sequence. When active, its reactive flows execute; when
inactive, they don't.

**Two edge types**:

- `->` (Continuous): Reactive flow that runs while the stage is active
- `=>` (OneShot): Fires once when condition becomes true, then doesn't fire again until
  stage is re-entered

### Sequence Syntax

```
SequenceDeclaration ::= 'sequence' Identifier '{' StageDeclaration+ '}'

StageDeclaration ::= 'stage' Identifier '{' StageItem* '}'

StageItem ::= FlowStatement
```

### Example

```arc
sequence main {
    stage pressurize {
        // Reactive flows: run continuously while stage is active
        sensor -> pressure_control{target=500.0},

        // One-shot transitions: fire once when condition is true
        pressure > 500.0 => next,
        abort_btn => abort
    }

    stage ignite {
        igniter_cmd = 1,
        flame_detected => next
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
```

### Definition Order and `next`

The `next` keyword resolves to the next stage in definition order:

```arc
stage step1 { 1 => next }  // next = step2
stage step2 { 1 => next }  // next = step3
stage step3 { }            // terminal (no outgoing transitions)
```

**Rules**:

- `next` is only valid within a stage (not in top-level flow statements)
- Using `next` on the last stage in a sequence is a compile-time error

### Transition Targets

- `=> next` — Go to the next stage in definition order
- `=> stage_name` — Jump to any stage in the same sequence
- `=> sequence_name` — Jump to a different sequence (starts at its first stage)

### Reactive vs One-Shot Semantics

**Reactive flows (`->`)**: Execute every time the source produces a value while the
stage is active.

**One-shot transitions (`=>`)**: Execute once when the condition becomes true, then
stop. The "one-shot" state resets when the stage is re-entered.

### Stage Entry Semantics

When entering a stage:

1. All one-shot transition states reset (can fire again)
2. All stateful nodes in the stage are reset
3. Reactive flows start fresh

Stages are stateless between entries—no implicit memory of previous time in the stage.

### Cross-Sequence Transitions

When transitioning to another sequence (e.g., `=> abort`):

1. Source sequence's active stage is deactivated
2. Target sequence starts at its first defined stage
3. This is one-way—no built-in "return" mechanism

### Top-Level Entry Points

Entry points connect external events to sequences:

```arc
start_cmd => main           // channel triggers sequence
emergency_stop => abort     // multiple entries allowed
```

The sequence starts when the source produces a truthy value.

## Naming and Scoping

**Global namespace**: All functions, sequences, and external channels must have unique
names.

**Variable scoping**: Variables are function-scoped. Cannot shadow global names.

**Channel declaration**: Channels are external to Arc, referenced by name.

## Language Restrictions

These simplify implementation while maintaining expressiveness:

1. **No mixed-type arithmetic**: Explicit casts required (`f32(x) + y`)
2. **No dynamic function instantiation**: All functions with config blocks are
   instantiated at compile time
3. **No assignment in expressions**: Separate statements required
4. **No partial function application**: Must provide all required arguments
5. **Config = compile-time constants**: Only literals/channel IDs in config blocks
6. **No closures**: Functions cannot capture variables from enclosing scope
7. **No nested functions**: Functions cannot be defined inside other functions
8. **No loops**: Use reactive patterns with stateful variables instead

## Error Handling

### Compile-Time Errors

- **Type errors**: Mismatches, invalid casts, bare `[]` without type
- **Name resolution**: Undefined identifiers, duplicates, shadowing
- **Structural**: Missing returns, unreachable code, wrong argument count
- **Flow graph**: Cycles in non-transition flows, unconnected inputs, type mismatches
- **Dimensional**: Incompatible units in operations, non-literal exponent with
  dimensioned base

### Runtime Errors

- **Arithmetic**: Division/modulo by zero
- **Array/String**: Out-of-bounds access, length mismatch in series operations
- **Channel**: Type mismatches (if not statically verified)

## Compilation Target

Arc compiles exclusively to WebAssembly (WASM).

### Compilation Output

The compiler produces a package containing:

1. **IR (Intermediate Representation)**:
   - Functions: declared function signatures and metadata
   - Nodes: instantiated function instances with config values
   - Edges: dataflow connections between nodes
   - Strata: topological execution ordering
   - Sequences: state machine definitions with stages

2. **WASM Module**: Compiled bytecode for function bodies

3. **Output Maps**: Memory layout information for multi-output functions

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

**Type mapping**: `i8`-`i32`, `u8`-`u32` → WASM `i32`; `i64`, `u64` → WASM `i64`; `f32`
→ WASM `f32`; `f64` → WASM `f64`.

### Stratified Execution

The runtime executes nodes in stratified order for deterministic reactive scheduling:

1. **Global strata**: Nodes not in any sequence execute first
2. **Stage strata**: Active stage nodes execute in topological order
3. **Convergence**: Stage transitions trigger re-evaluation until stable

### Compilation Pipeline

```
Source Code
    ↓
Parser (ANTLR4) → AST
    ↓
Analyzer Pass 1: Collect declarations → Symbol table
    ↓
Analyzer Pass 2: Type checking + Validation
    ↓
Analyzer Pass 3: Constraint unification (if type variables)
    ↓
Stratifier: Compute execution order
    ↓
Compiler: Generate WASM + Build IR
    ↓
Output Package (IR + WASM + Maps)
```

### Runtime Responsibilities

The Synnax runtime handles:

- Reactive scheduling with event queue
- Channel management (unbounded FIFO queues)
- Function lifecycle and state persistence
- Memory management for WASM modules
- Flow graph execution in stratified order
- Snapshot semantics for non-blocking reads
