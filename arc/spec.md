# Arc Language Specification

## Comments

```
Comment ::= SingleLineComment | MultiLineComment

SingleLineComment ::= '//' [^\n]*

MultiLineComment ::= '/*' .*? '*/'
```

```arc
// This is a single-line comment

/* This is a
   multi-line comment */
```

## Type System

### Primitive Types

```
Type ::= PrimitiveType | ChannelType | SeriesType

PrimitiveType ::= NumericType | 'string'

// String literals
StringLiteral ::= '"' UTF8Character* '"'

NumericType ::= IntegerType | FloatType | TemporalType

IntegerType ::= 'i8' | 'i16' | 'i32' | 'i64'
              | 'u8' | 'u16' | 'u32' | 'u64'

FloatType ::= 'f32' | 'f64'

TemporalType ::= 'timestamp' | 'timespan'
```

### Channel Types

```
ChannelType ::= ('chan' | '<-chan' | '->chan') (PrimitiveType | SeriesType)
```

Channels are typed communication primitives:

- `chan` - Bidirectional channel
- `<-chan` - Read-only channel (only in function/stage parameters)
- `->chan` - Write-only channel (only in function/stage parameters)

Note: Directional constraints (`<-chan`, `->chan`) can only be specified in function or
stage parameters, not in variable declarations or the inter-stage layer.

### Series Types

```
SeriesType ::= 'series' PrimitiveType

SeriesLiteral ::= '[' ExpressionList? ']'

ExpressionList ::= Expression (',' Expression)*
```

Series are homogeneous arrays of primitive types with full array operations.

#### Series Operations

```arc
// Literals
data := [1.0, 2.0, 3.0, 4.0]  // series f64 inferred
empty series i32 := []         // Empty series

// Length
length := len(data)            // Built-in function returns i64

// Indexing (0-based)
first := data[0]               // 1.0
last := data[len(data)-1]     // 4.0

// Slicing [start:end] (end exclusive)
subset := data[1:3]            // [2.0, 3.0]
rest := data[2:]               // [3.0, 4.0]
prefix := data[:2]             // [1.0, 2.0]

// Elementwise arithmetic with scalars (explicit cast required for type mismatch)
data_f32 series f32 := [1.0, 2.0, 3.0]
scaled := data_f32 * 2.0       // [2.0, 4.0, 6.0] (f32 scalar)
shifted := data_f32 + f32(10)  // [11.0, 12.0, 13.0] (i32 must be cast)

// Elementwise arithmetic between series (same length, same type)
series1 := [1.0, 2.0, 3.0]
series2 := [4.0, 5.0, 6.0]
sum := series1 + series2       // [5.0, 7.0, 9.0]
product := series1 * series2   // [4.0, 10.0, 18.0]

// All comparisons produce series u8 (elementwise)
mask := data_f32 > 2.5         // [0, 0, 1]
equal := series1 == series2    // [0, 0, 0] (elementwise comparison)
not_equal := series1 != series2 // [1, 1, 1]
```

#### Series Rules

- **Immutable**: Series cannot be modified after creation
- **Bounds checking**: Out-of-bounds access is a runtime error
- **Type consistency**: All elements must be the same type
- **Length matching**: Binary operations require equal-length series (runtime error if
  lengths differ)
- **Zero value**: Empty series `[]` for any series type
- **Scalar operations**: Mixed scalar/series operations require explicit type casting
  (no implicit promotion)
  - In mixed scalar/series expressions, the scalar's type MUST be explicitly cast to the
    series element type.
- **Series equality**: `==` and `!=` perform elementwise comparison, returning
  `series u8`, requiring equal lengths
- **Empty series typing**: Bare `[]` without type context is a compile error - must use
  explicit type annotation or infer from context
- **Length function**: `len(series)` returns `i64` representing the number of elements

### Type Annotations

```
TypedIdentifier ::= Identifier Type

// Examples:
voltage f32
data series f32
// Note: Channel directions only in function/stage parameters
```

### Numeric Literals

```
NumericLiteral ::= IntegerLiteral | FloatLiteral

IntegerLiteral ::= Digit+

FloatLiteral ::= Digit+ '.' Digit* | '.' Digit+

Digit ::= '0'..'9'

// Examples:
42                 // i64 (default integer type)
u8(42)             // u8 via explicit cast
1000000            // i64
255                // i64
u32(65535)         // u32 via explicit cast
10                 // i64
u8(172)            // u8 via explicit cast
3.14               // f64 (default float type)
f32(3.141592)      // f32 via explicit cast
100.0              // f64
0.001              // f64
f32(1000.0)        // f32 via explicit cast
```

**Default Types**:

- Integer literals default to `i64`
- Float literals default to `f64`
- Use type casting (e.g., `i32(42)`) to specify a different type

### Temporal Literals

```
TemporalLiteral ::= Number TemporalUnit | Number FrequencyUnit

TemporalUnit ::= 'ns' | 'us' | 'ms' | 's' | 'm' | 'h'

FrequencyUnit ::= 'hz' | 'khz' | 'mhz'

// Examples:
100ms       // 100 milliseconds
5s          // 5 seconds
1m          // 1 minute (NOT meters)
1h          // 1 hour
10hz        // 100ms (1/10 second)
1khz        // 1ms (1/1000 second)
50hz        // 20ms (1/50 second)
```

**Unit Disambiguation**:

- `m` always means minutes in temporal contexts (never meters or milliseconds)
- `ms` always means milliseconds
- Arc has no spatial units - all units are temporal or frequency

Frequency literals are automatically converted to timespan by inverting the period.
**Case**: Frequency units are case-insensitive (`hz`, `kHz`, `MHz` all valid). The
canonical form is lowercase in this spec.

### Type Notes

- **u8 as boolean**: u8 doubles as boolean type (0=false, non-zero=true)
- **Temporal types**: `timestamp` and `timespan` are i64 nanoseconds since Unix epoch
- **Channels**: Can only carry primitives or series of primitives

### String Semantics

Strings in Arc are immutable UTF-8 encoded sequences:

- **Encoding**: UTF-8
- **Immutability**: All strings are immutable (cannot be modified after creation)
- **Comparisons**: Only `==` and `!=` are supported (no lexicographic ordering)
- **No concatenation**: String manipulation is not supported
- **Literals only**: Strings can only be created from literals
- **Operations**: Strings have no operations beyond equality - no length function, no
  indexing, no slicing

```arc
msg := "Hello"           // String literal
equal := msg == "Hello"  // Returns 1 (true)
different := msg != "Hi" // Returns 1 (true)

// Not supported:
// msg < "World"         // Error: no lexicographic comparison
// msg + " World"        // Error: no string concatenation
```

### Value Semantics

Variables in Arc can be reassigned, but compound values cannot be mutated in-place:

- **Variables are reassignable**: Variables can be assigned new values
- **No in-place mutation**: Cannot modify parts of strings or series
- **Series reassignment**: Can create new series with modified values

```arc
// Variables can be reassigned using =
x := 5          // Declare x
x = 10          // OK: reassignment

// Strings cannot be mutated
msg := "Hello"
// msg[0] = 'h'  // Error: cannot mutate string

// Series element assignment creates new series
data := [1, 2, 3]
data = [5, data[1], data[2]]  // Create new series with modified first element

// Or should we support this syntax?
// data[0] = 5   // Would create new series [5, 2, 3] and reassign to data
```

**Note**: Series element assignment syntax is under consideration.

### Boolean Semantics

The u8 type serves as Arc's boolean type with the following behavior:

- **Truth values**: 0 is false, any non-zero value is true
- **Logical operators return u8**: `&&` and `||` return 0 or 1 (normalized)
- **Short-circuit evaluation**: `&&` and `||` use short-circuit evaluation
- **Normalization**: Logical operations normalize to 0 or 1

```arc
// Logical operations return normalized u8 (0 or 1)
result := 2 && 3          // Returns 1 (both truthy)
result := 5 || 0          // Returns 1 (5 is truthy)
result := 0 && anything   // Returns 0 (short-circuits, doesn't evaluate 'anything')

// Boolean context in conditionals
value u8 := 42
if value {                // Any non-zero is true
    // This executes
}

// Negation
negated := !5             // Returns 0 (5 is truthy, negated to false)
negated := !0             // Returns 1 (0 is false, negated to true)
```

### Zero Values

All types have a default zero value used for initialization:

```arc
// Numeric zero values
i8, i16, i32, i64: 0
u8, u16, u32, u64: 0
f32, f64: 0.0

// Other zero values
string: ""
timestamp: 0 (Unix epoch)
timespan: 0 (no duration)

// Channel reads before first write return zero value
value := sensor_chan  // Returns 0.0 if sensor_chan is f64
```

### Type Casting

Explicit type casting between numeric types:

```
TypeCast ::= Type '(' Expression ')'
```

#### Casting Rules

**Integer Widening** (always safe):

```arc
// Zero-extend unsigned, sign-extend signed
u16_val := u16(u8_val)    // Zero extends
i16_val := i16(i8_val)    // Sign extends
i64_val := i64(i32_val)   // Sign extends
```

**Integer Narrowing** (truncates):

```arc
// Keeps low bits only
u8_val := u8(u16_val)     // Keeps low 8 bits
i8_val := i8(i64_val)     // Keeps low 8 bits
```

**Signed ↔ Unsigned** (saturates):

```arc
// Same width: saturate at bounds
u32_val := u32(i32_val)   // Negative → 0, positive unchanged
i32_val := i32(u32_val)   // > i32::MAX → i32::MAX

// Different widths: convert then widen/narrow
u8_val := u8(i32(-5))     // Saturates to 0
i8_val := i8(u32(200))    // Saturates to 127 (i8::MAX)
```

**Float ↔ Float**:

```arc
f64_val := f64(f32_val)   // Exact promotion
f32_val := f32(f64_val)   // Rounds to nearest even (IEEE default)
```

**Float → Integer**:

```arc
// Truncates toward zero, saturates on overflow
i32_val := i32(3.7_f64)   // 3
i32_val := i32(-3.7_f64)  // -3
u32_val := u32(-1.0_f64)  // 0 (saturates)
```

**Integer → Float**:

```arc
// Exact for small integers, rounds for large
f32_val := f32(i32_val)   // Exact within ±16M range
f64_val := f64(i64_val)   // Exact within ±2^53 range
```

**Integer Overflow Behavior**: Arithmetic operations use two's-complement wrapping on
overflow (like C/Rust in release mode). For example, `u8(255) + u8(1)` equals `u8(0)`.

## Channel Operations

### Operation Grammar

```
ChannelOperation ::= ChannelWrite | ChannelRead | ChannelPipe

ChannelWrite ::= Expression '->' Identifier
               | Identifier '<-' Expression

ChannelRead ::= BlockingRead | NonBlockingRead

BlockingRead ::= Identifier ':=' '<-' Identifier

NonBlockingRead ::= Identifier ':=' Identifier

ChannelPipe ::= Identifier '->' Identifier
              | Identifier '<-' Identifier
```

### Write Operations

Write values to channels using arrow notation:

```arc
42 -> output_chan        // Write 42 to output_chan
output_chan <- 42        // Equivalent syntax
sensor_value -> log      // Write sensor value to log channel

// Writing series to channels
data series f32 := [1.0, 2.0, 3.0]
data -> history_chan    // Write entire series to channel
```

### Read Operations

Two types of read operations are supported:

#### Blocking Read

Waits for the next value to be written to the channel:

```arc
value := <-input_chan    // Block until next value arrives
```

#### Non-Blocking Read

Gets the current/latest value immediately:

```arc
current := sensor_chan   // Get current value without blocking
```

### Channel Piping

Channel-to-channel operations copy values:

```arc
sensor -> display        // Read current value from sensor, write to display
display <- sensor        // Equivalent syntax

// In functions/tasks, this is equivalent to:
value := sensor          // Non-blocking read
value -> display         // Write value
```

**IMPORTANT**: In the inter-stage layer, `sensor -> stage{}` creates a reactive
connection that triggers stage execution. Inside functions/tasks, `sensor -> display` is
a simple read-then-write operation with no triggering.

### Channel Semantics

Channels are unbounded FIFO queues with the following behavior:

- **Queue model**: Each channel maintains an unbounded queue of values
- **Write operations**: Append values to the end of the queue
- **Blocking read (`<-chan`)**: Removes and returns the oldest value from queue
- **Non-blocking read (`chan`)**: Returns the newest value without removing from queue
- **Ordering guarantee**: Values are delivered in the order they were written
- **Piping**: Tasks execute when new values arrive (not on connection)

#### Series-Based Channel Implementation

**Implementation Note**: Runtimes are recommended to implement all channels as series
channels internally. A channel typed as `chan f32` actually carries series of f32
values. Scalar channel operations are conveniences that interact with the first element:

- **Scalar write**: `42.0 -> ch` creates a single-element series `[42.0]`
- **Scalar read**: `<-ch` returns the first element of the next series
- **Series read**: `<-ch[]` returns the entire next series (future syntax)

This unified model simplifies runtime implementation and naturally supports streaming
telemetry where sensors often produce batches of samples.

```arc
// Blocking read - waits for value, removes from queue
value := <-sensor

// Non-blocking read - returns newest value, queue unchanged
current := sensor

// Returns zero if queue is empty (never written)
temp := uninitialized_chan  // Returns 0.0 for f64 channel - can mask wiring bugs!
```

**⚠️ Potential Gotcha**: Non-blocking reads return the zero value if a channel has never
been written to. This can mask wiring bugs where channels aren't properly connected.
Consider using explicit checks or the future `??` operator when implemented:

```arc
// Current behavior (can mask bugs)
value := sensor_chan  // Returns 0.0 if never written

// Future consideration - explicit default handling
// value := sensor_chan ?? -1.0  // Would make empty channels explicit
```

**Memory Note**: Queue growth is managed by the runtime. Maximum queue sizes are a topic
for future discussion.

### Event Ordering

The runtime MUST produce a deterministic total order of stage activations. When multiple
channel writes occur within the same scheduler tick, their resulting activations are
ordered deterministically by (1) enqueue timestamp, then (2) a stable topological order
of edges, then (3) a stable tie-break on channel identifier. Implementations may choose
any fixed scheme, but it MUST be deterministic across runs given the same inputs.

**Capacity Note**: While channels are specified as logically unbounded, implementations
MAY impose a high-water mark for resource safety. If imposed, the drop policy MUST be
deterministic (e.g., `drop_oldest` or `drop_newest`) and documented. The semantic
guarantees above (ordering, snapshot) still apply to all retained items.

### Channel Rules

- Write operations respect channel direction (`->chan` can only be written to)
- Read operations respect channel direction (`<-chan` can only be read from)
- Piping triggers stage execution on new values
- Type compatibility is enforced at compile time
- **Per-Activation Snapshot**: All non-blocking reads within a single stage activation
  see the same channel state snapshot taken at activation start

## Variables

### Variable Declaration and Assignment

```
VariableDeclaration ::= LocalVariable | StatefulVariable

LocalVariable ::= Identifier ':=' Expression
                | Identifier Type ':=' Expression

StatefulVariable ::= Identifier '$=' Expression
                   | Identifier Type '$=' Expression

Assignment ::= Identifier '=' Expression

Identifier ::= Letter (Letter | Digit | '_')*
```

### Local Variables

Local variables are scoped to the current function or stage execution and reset on each
invocation. They are declared with `:=`:

```arc
count := 0                    // Type inferred local declaration
voltage f32 := 3.3           // Explicit type local declaration
sensor_val := <-input        // Read from channel into new local

// Reassignment uses =
count = count + 1            // Reassign existing local variable
voltage = 5.0                // Reassign with new value
```

### Stateful Variables

Stateful variables persist across reactive stage executions. They are declared with
`$=`:

```arc
total $= 0                   // Stateful variable declaration
last_state u8 $= 0          // State tracking with explicit type
previous_time timestamp $= now()  // Time tracking

// Reassignment uses =
total = total + value        // Update existing stateful variable
last_state = new_state      // Reassign state
```

### Assignment vs Declaration

- **`:=`** - Declares a new local variable (compile error if variable already exists)
- **`$=`** - Declares a new stateful variable (compile error if variable already exists)
- **`=`** - Assigns to an existing variable (compile error if variable doesn't exist)

```arc
// Examples
x := 5          // Declare local x
x = 10          // OK: reassign x
// x := 15      // Error: x already declared

count $= 0      // Declare stateful count
count = count + 1  // OK: update count
// count $= 5   // Error: count already declared

// y = 10       // Error: y not declared
```

### Variable Rules

- **Scope**: Variables are function/stage scoped only (no globals)
- **Type inference**: Types can be inferred from initial value
- **No shadowing**: Variable names must be unique within scope
- **Initialization required**: All variables must be initialized at declaration
- **No channel variables**: Channels cannot be declared as variables (only as
  parameters)
- **Declaration once**: Each variable can only be declared once in its scope
- **Assignment requires prior declaration**: Cannot assign to undeclared variables

## Operators

### Expression Grammar

```
Expression ::= UnaryExpression | BinaryExpression | PrimaryExpression

UnaryExpression ::= UnaryOperator Expression

BinaryExpression ::= Expression BinaryOperator Expression

UnaryOperator ::= '-' | '!'

BinaryOperator ::= ArithmeticOp | ComparisonOp | LogicalOp

ArithmeticOp ::= '+' | '-' | '*' | '/' | '%' | '^'

ComparisonOp ::= '==' | '!=' | '<' | '>' | '<=' | '>='

LogicalOp ::= '&&' | '||'
```

### Operator Precedence and Associativity

Operators are evaluated in the following precedence order (highest to lowest):

1. **Exponential**: `^` (right-associative)
2. **Unary**: `-`, `!` (right-associative)
3. **Multiplicative**: `*`, `/`, `%` (left-associative)
4. **Additive**: `+`, `-` (left-associative)
5. **Comparison**: `<`, `>`, `<=`, `>=`, `==`, `!=` (left-associative)
6. **Logical**: `&&`, `||` (left-associative, short-circuit evaluation)

### Associativity Examples

```arc
// Exponentiation is right-associative (matches mathematical notation)
2 ^ 3 ^ 2  // equals 2 ^ (3 ^ 2) = 2 ^ 9 = 512

// Exponentiation binds tighter than unary minus
-2 ^ 2     // equals -(2 ^ 2) = -4
(-2) ^ 2   // equals 4

// Other operators are left-associative
10 - 5 - 2  // equals (10 - 5) - 2 = 3
20 / 4 / 2  // equals (20 / 4) / 2 = 2.5
```

### Operator Examples

```arc
// Arithmetic
result := 2 + 3 * 4      // 14
power := 2 ^ 8           // 256
remainder := 10 % 3      // 1
neg := -2 ^ 2            // -4 (because ^ > unary -)

// Modulo works on all numeric types with consistent truncated division
float_mod := 5.5 % 2.0   // 1.5
negative_mod := -7 % 3   // -1

// Modulo operation rules:
// - Result sign always matches dividend (first operand)
// - Uses truncated division toward zero (C-style)
// - Integer: a % b = a - (a/b)*b where a/b truncates toward zero
// - Float: Uses IEEE fmod behavior (also truncated toward zero)

// Comparison
is_high := pressure > 100
in_range := temp >= 20 && temp <= 30

// Logical
alarm := pressure > 100 || temp > 80
safe := !alarm && system_ready
```

### Type Compatibility

- Arithmetic operators work on numeric types
- Comparison operators work on compatible types
- Logical operators work on boolean expressions (u8 as bool)
- Type mismatches are compile-time errors

### No Bitwise Operations

Arc does not support bitwise operations. The `^` operator is used for exponentiation,
not XOR. Bitwise operations (`&`, `|`, `^`, `~`, `<<`, `>>`) are not available. For
hardware register manipulation, use dedicated driver tasks or external functions.

### Evaluation Order

- Binary operators evaluate their left operand before the right operand.
- Function call arguments are evaluated left-to-right.
- Short-circuit operators `&&` and `||` only evaluate the right operand if needed.

## Functions

### Function Grammar

```
FunctionDeclaration ::= 'func' Identifier '(' ParameterList? ')' ReturnType? Block

ParameterList ::= Parameter (',' Parameter)*

Parameter ::= Identifier Type

ReturnType ::= Type

Block ::= '{' Statement* '}'

Statement ::= VariableDeclaration
            | ChannelOperation
            | Assignment
            | IfStatement
            | ReturnStatement
            | FunctionCall

ReturnStatement ::= 'return' Expression?

FunctionCall ::= Identifier '(' ArgumentList? ')'

ArgumentList ::= Expression (',' Expression)*
```

### Function Examples

```arc
// Function with return type
func calculate_pid(error f64, kp f64, ki f64) f64 {
    return (error * kp) + (error * ki)
}

// Function with channel parameters
func log_alarm(sensor <-chan f64, alarm ->chan string) {
    if (<-sensor) > 100 {
        "HIGH TEMP" -> alarm
    }
}

// Void function
func process_data(input f64, output ->chan f64) {
    result := input * 2.5
    result -> output
}

// Function argument strictness
func add(x f64, y f64) f64 {
    return x + y
}
// result := add(1, 2)       // OK: returns 3
// result := add(1, 2, 3, 4) // Error: extra arguments not allowed
// result := add(1)          // Error: missing required parameter
```

### Function Restrictions

- **No recursion**: Functions cannot call themselves directly or indirectly
- **No closures**: Functions cannot capture variables from outer scopes
- **No nested functions**: Functions cannot be defined inside other functions
- **Variable scope**: Variables declared in functions are local to that function
- **No function variables**: Functions are not first-class values

### Function Rules

- Parameter types must be explicitly declared
- Return type is optional (void if omitted)
- Functions execute synchronously in the calling stage's context
- Functions can accept channels with directional constraints
- All code paths must return a value if return type is specified
- No default parameter values allowed
- Must provide at least all declared parameters when calling
- **Extra arguments cause compile error** (functions are strict)

## Tasks

### Stage Grammar

```
StageDeclaration ::= 'stage' Identifier ConfigBlock? ParameterBlock ReturnType? Block

ConfigBlock ::= '{' ConfigParameter* '}'

ConfigParameter ::= Identifier Type

ParameterBlock ::= '(' ParameterList? ')'

ReturnType ::= Type

// Stage invocation (used in inter-stage layer)
StageInvocation ::= Identifier ConfigValues? Arguments?

ConfigValues ::= '{' ConfigAssignment* '}'

ConfigAssignment ::= Identifier ':' Expression

Arguments ::= '(' ArgumentList? ')'
```

### Stage Definition

Tasks are isolated units of reactive computation:

```arc
// Stage with config and runtime parameters
stage controller{
    setpoint f64         // Config: static at instantiation
    sensor <-chan f64    // Config: input channel
    actuator ->chan f64  // Config: output channel
    interval timespan    // Config: execution rate
} (enable u8) {         // Runtime parameter
    // Stateful variable persists across executions
    integral $= 0.0

    if !enable {
        0 -> actuator
        return
    }

    error := setpoint - (<-sensor)
    integral $= integral + error

    output := (error * 1.5) + (integral * 0.1)
    output -> actuator
}

// Stage with return value (creates anonymous output channel)
stage doubler{
    input <-chan f64
} () f64 {              // Return type specified
    return (<-input) * 2
}

// Simple stage without return
stage alarm{
    threshold f64
    input <-chan f64
    alert ->chan string
} () {
    if (<-input) > threshold {
        "ALARM" -> alert
    }
}
```

### Stage Invocation

Tasks are invoked with configuration values and arguments:

```arc
// Invoke with config and runtime argument
controller{
    setpoint: 100,
    sensor: temp_sensor,
    actuator: valve_cmd,
    interval: 100ms
}(1)

// Invoke without runtime arguments
alarm{
    threshold: 85.0,
    input: pressure_chan,
    alert: notify_chan
}()
```

### Stage Execution Model

Tasks are purely event-driven - they execute when they receive input values:

- **Event-driven only**: Tasks execute when upstream sources send values
- **No implicit intervals**: Config params like `interval timespan` are just data, not
  triggers
- **Interval execution via stdlib**: Use the `interval{}` stage for periodic execution
- **Multi-input triggering**: Tasks with multiple input channels trigger:
  - **First execution**: When ALL input channels have received at least one value
  - **Subsequent executions**: When ANY input channel receives a new value (uses stale
    values for other channels)

```arc
// Tasks execute when they receive input
sensor -> controller{}  // controller runs when sensor sends value

// For interval-based execution, use the interval stage
interval{period: 100ms} -> sampler{}  // sampler runs every 100ms

// Combining events and intervals
all{sensor, interval{period: 1s}} -> logger{}  // Log when both arrive

// Multi-input stage triggering example (snapshot-safe)
stage combiner{
    ch1 <-chan f64
    ch2 <-chan f64
} () {
    // Non-blocking snapshot reads: both values come from the activation snapshot.
    v1 := ch1
    v2 := ch2
    (v1 + v2) -> output_chan
}

// Execution sequence:
// 1. ch1 receives 10.0 → stage does NOT execute (ch2 never written)
// 2. ch2 receives 5.0  → stage executes with v1=10.0, v2=5.0
// 3. ch1 receives 20.0 → stage executes with v1=20.0, v2=5.0 (stale)
// 4. ch2 receives 7.0  → stage executes with v1=20.0, v2=7.0 (stale)
```

### Stage Rules

- **Isolation**: Each stage instance has isolated state, no shared memory
- **Channel communication**: Tasks communicate only through channels
- **Stateful variables**: Use `$=` for state that persists across executions
- **Config immutability**: Configuration parameters cannot change after instantiation
- **No stage recursion**: Tasks cannot invoke themselves
- **Extra runtime arguments ignored**: Tasks safely ignore extra runtime arguments
  (unlike functions)

### Stage Return Value Plumbing

When a stage has a return type, it creates an anonymous output channel that other tasks
can consume:

```arc
// Stage with return value
stage multiplier{
    factor f64
    input <-chan f64
} () f64 {
    return (<-input) * factor
}

// The return value creates an anonymous channel
// This channel can be used directly in inter-stage flows:
sensor -> multiplier{factor: 2.0} -> display{}

// Equivalent to:
// 1. multiplier creates anonymous channel when instantiated
// 2. sensor values flow into multiplier's input
// 3. multiplier returns values to its anonymous output channel
// 4. display receives values from multiplier's output channel

// Multiple return channels can be chained
sensor -> doubler{} -> tripler{} -> display{}

// Return channels can fork to multiple consumers using tee
source -> processor{} -> tee{logger{}, monitor{}, storage{}}
```

**Return Value Semantics**:

- A stage with return type `T` creates an anonymous `->chan T`
- The return statement sends a value to this channel
- The anonymous channel is automatically connected in inter-stage flows
- Tasks without return types have no output channel
- **Anonymous channels are normal channels**: They follow the same unbounded FIFO queue
  semantics, snapshot behavior, and channel rules as explicitly declared channels

## Control Flow

### Control Flow Grammar

```
ControlFlow ::= IfStatement

IfStatement ::= 'if' Expression Block ElseIfClause* ElseClause?

ElseIfClause ::= 'else' 'if' Expression Block

ElseClause ::= 'else' Block

Block ::= '{' Statement* '}'
```

### Conditional Statements

Arc supports only conditional statements for decision-making:

```arc
// Simple if statement
if pressure > 100 {
    true -> alarm
}

// If-else chain
if pressure > 100 {
    true -> alarm
} else if pressure > 80 {
    true -> warning
} else {
    false -> alarm
    false -> warning
}

// Compound conditions
if sensor_enabled && value > threshold {
    value -> output
}
```

### Control Flow Rules

- **No loops**: Reactive execution model handles repetition via events/intervals
- **Block syntax required**: All conditions must use braces `{}`
- **Boolean context**: Conditions evaluate to u8 (0=false, non-zero=true)
- **Function/stage scope only**: Control flow only allowed inside functions and tasks
- **No switch/case**: Use if/else chains for multiple conditions

### Design Rationale

The absence of explicit loops is intentional - Arc's reactive model means:

- Tasks re-execute based on events or intervals
- State persistence via `$=` variables enables iteration across executions
- This design ensures predictable real-time behavior

## Inter-Stage Layer

### Inter-Stage Flow Grammar

```
InterTaskFlow ::= FlowStatement*

FlowStatement ::= FlowSource '->' FlowTarget

FlowSource ::= ChannelIdentifier | StageInvocation | Expression
FlowTarget ::= ChannelIdentifier | StageInvocation

StageInvocation ::= Identifier ConfigValues? Arguments?
```

### Data Flow

The inter-stage layer connects tasks and channels to create reactive automation
pipelines:

```arc
// Simple pipeline
sensor -> filter{threshold: 50} -> controller{} -> actuator

// Channel to stage to channel
temp_sensor -> controller{setpoint: 100} -> valve_cmd

// Stage chaining
startup{} -> pressurize{} -> ignition{} -> shutdown{}
```

### Inline Expression Tasks

Expressions can act as implicit tasks in the inter-stage layer, but can only reference
channels (not variables):

```arc
// Channel pass-through - trigger stage on any channel change
ox_pt_1 -> logger{}                   // Log every ox_pt_1 value
temperature -> controller{}           // Run controller on each temperature update

// Comparison expressions emit u8 (boolean)
ox_pt_1 > 100 -> alarm{}              // ox_pt_1 is a channel
temp_sensor < 20 -> heater_on{}       // temp_sensor is a channel

// Arithmetic expressions emit computed values
(ox_pt_1 + ox_pt_2) / 2 -> average_display    // Both are channels
sensor * 1.8 + 32 -> fahrenheit_display       // sensor is a channel

// Logical expressions for complex conditions
ox_pt_1 > 100 && fuel_pt_1 > 50 -> ready_signal
pressure_chan > 100 || emergency_stop -> shutdown{}

// Chaining expressions with tasks
temperature_chan > 100 -> once{} -> high_temp_alarm{}
(sensor1 + sensor2) / 2 > 50 -> logger{}

// Note: Only literals and channels allowed, no variables
pressure_sensor > 100 -> alarm{}     // Valid: channel and literal
// pressure_sensor > threshold -> alarm{}  // Invalid: threshold is a variable
```

These inline expressions:

- Include simple channel references that pass through values
- Can only reference channels and literal values
- Execute reactively when referenced channels receive new values
- Emit their result to the downstream target
- Eliminate need for trivial filter/transform tasks

### Standard Library Orchestration Tasks

Common flow patterns are implemented as standard library tasks with specific behavioral
contracts:

```arc
// all: Wait for all inputs before proceeding
all{ox_pressure, fuel_pressure, control_power} -> ignition{}

// once: Emit only the first value
once{start_signal} -> sequence{duration: 60s}

// any: Proceed when any input arrives
any{abort_button, timeout_signal, complete} -> shutdown{}

// throttle: Rate limiting
sensor -> throttle{rate: 10hz} -> logger{}

// merge: Combine multiple inputs
merge{sensor1, sensor2, sensor3} -> aggregator{}

// tee: Fork to multiple outputs
sensor -> tee{logger{}, display{}, storage{}}
```

#### Standard Library Stage Contracts

**all{}**: Waits for all inputs to receive at least one value since instantiation or
last emit. If an input fires multiple times before others, only the latest value is
kept. Emits when the last required input arrives, then resets and waits for all inputs
again.

**once{}**: Emits only the first value received from its input, then ignores all
subsequent values. Useful for single-shot triggers or preventing repeated execution.

**any{}**: Emits immediately whenever any of its inputs receives a value. Does not wait
for other inputs and does not reset - continues forwarding values from any input as they
arrive.

**throttle{rate: R}**: Rate-limits its input by ensuring output values are emitted at
most every 1/R period. Intermediate values that arrive too quickly are dropped. Always
emits the most recent value when the rate period expires.

**merge{}**: Forwards any value from any of its inputs immediately to the output.
Similar to `any{}` but semantically indicates combining multiple similar streams rather
than event selection.

**tee{}**: Takes one input and forwards it to multiple output tasks simultaneously. All
configured output tasks receive the same input value when it arrives.

### Complex Flow Examples

```arc
// Multi-stage process with synchronization
start_button -> once{start_button} -> startup{}
startup{} -> all{ox_ready, fuel_ready} -> ignition{duration: 10s}
ignition{} -> steady_state{duration: 30s} -> shutdown{}

// Parallel monitoring
pressure_sensor -> monitor{
    threshold: 100,
    output: pressure_alarm
}
temp_sensor -> monitor{
    threshold: 85,
    output: temp_alarm
}
any{pressure_alarm, temp_alarm} -> emergency_stop{}
```

### Inter-Stage Rules

- **Isolation**: Tasks cannot share memory or state
- **Channel typing**: Connections must respect channel types
- **Unidirectional flow**: Data flows in direction of arrows
- **Reactive execution**: Tasks execute when inputs arrive
- **No cycles**: Stage graphs must be acyclic (no feedback loops allowed)

### Cycle Detection

Cycles in the stage graph are detected at compile time through static analysis:

- The compiler builds a directed graph of all stage connections
- Depth-first search detects any cycles
- All cycles are forbidden (no feedback loops)
- Compile error reports the tasks involved in the cycle

```arc
// Error: cycle detected
task_a{} -> task_b{} -> task_c{} -> task_a{}  // Compile error

// Error: self-loop
processor{} -> processor{}  // Compile error

// OK: parallel paths that reconverge using tee
sensor -> tee{path_a{}, path_b{}} -> merger{}  // No cycle
```

## Naming and Scoping

### Global Namespace

All items at the global scope must have unique names:

- Tasks
- Functions
- External channels (defined outside Arc)

```arc
// Error: duplicate names at global scope
stage pump{} { }
func pump() { }     // Error: name already used by stage
// pump_chan        // Error if external channel has same name
```

### Variable Scoping

Variables within functions/tasks cannot shadow global names:

```arc
stage controller{} { }

func process() {
    // controller := 5  // Error: shadows global stage name
    value := 5          // OK: unique name
}
```

### Channel Declaration

Channels are defined externally to Arc and referenced by name:

```arc
// In inter-stage layer
temperature -> controller{}  // temperature is external channel
pressure -> logger{}         // pressure is external channel

// In stage/function
stage monitor{
    sensor <-chan f64       // Parameter receives external channel
} () {
    value := <-sensor       // Read from channel parameter
}
```

### Stage Return Values

Tasks with return types create anonymous output channels:

```arc
stage doubler{
    input <-chan f64
} () f64 {
    return (<-input) * 2    // Returns to anonymous channel
}

// In inter-stage layer
sensor -> doubler{input: sensor} -> display
// Return value flows as first argument to display
```

## Language Restrictions

These restrictions simplify implementation while maintaining expressiveness:

### No Mixed Type Arithmetic

Type conversions must be explicit:

```arc
// Error: type mismatch
result := sensor_f32 + counter_u32

// Correct: explicit cast required
result := sensor_f32 + f32(counter_u32)
```

### No Nested Function Calls

Function calls cannot be nested in expressions:

```arc
// Disallowed
result := calculate(process(sensor), transform(data))

// Required: use intermediate variables
processed := process(sensor)
transformed := transform(data)
result := calculate(processed, transformed)
```

### No Dynamic Stage Creation

Tasks can only be instantiated at compile time:

```arc
// All stage invocations must be statically defined
controller{setpoint: 100}  // OK: compile-time instantiation

// Cannot create tasks based on runtime conditions
if condition {
    // new_task{}  // Error: dynamic stage creation
}
```

### No Assignment in Expressions

Assignments cannot appear within expressions:

```arc
// Disallowed
if (value := <-sensor) > 100 {
    // ...
}

// Required: separate assignment
value := <-sensor
if value > 100 {
    // ...
}
```

### No Partial Function Application

Functions must be called with all arguments:

```arc
func add(x f64, y f64) f64 {
    return x + y
}

result := add(1.0, 2.0)  // OK: all arguments provided
// partial := add(1.0)   // Error: partial application not allowed
```

### Stage Config Must Be Compile-Time Constants

Stage configuration values must be literals or channel identifiers:

````arc
// Valid: literals and channel names
controller{
    setpoint: 100,        // OK: literal
    sensor: temp_chan,    // OK: channel identifier
    interval: 100ms       // OK: temporal literal
}

// Invalid: variables or expressions
threshold := 100
controller{
    // setpoint: threshold,      // Error: variable not allowed
    // sensor: getChannel(),     // Error: function call not allowed
    // interval: delay * 2       // Error: expression not allowed
}

## Error Handling

### Compile-Time Errors

The following are detected at compile time:

#### Type Errors
- Type mismatch in operations or assignments
- Incompatible channel types in connections
- Missing type conversions
- Invalid type casts
- Bare empty series literal without type context (`[]` without annotation)

#### Name Resolution Errors
- Undefined identifiers (variables, functions, tasks, channels)
- Duplicate names in same scope
- Shadowing of global names
- Use of reserved keywords

#### Structural Errors
- Missing return statements in non-void functions
- Unreachable code after return
- Missing required parameters
- Extra arguments to functions (not tasks)
- Invalid channel directions

#### Flow Graph Errors
- Cyclic dependencies in stage graph
- Unconnected required stage inputs
- Type mismatches in stage connections

#### Syntax Errors
- Malformed expressions
- Missing braces or parentheses
- Invalid operators
- Invalid literals

### Runtime Errors

The following cause runtime errors:

#### Arithmetic Errors
- Division by zero
- Modulo by zero

#### Array Errors
- Out-of-bounds array access
- Length mismatch in series operations
- Empty series operations requiring elements

#### Channel Errors
- Type mismatch at runtime (if types can't be statically verified)

### Error Examples

```arc
// Compile-time type error
value f32 := 10 + "hello"  // Error: cannot add f32 and string

// Compile-time name error
result := undefined_var + 5  // Error: undefined_var not found

// Compile-time argument error
func strict(x f64) f64 { return x * 2 }
y := strict(1, 2)  // Error: too many arguments (functions are strict)

// Runtime arithmetic error
result := 10 / 0  // Runtime error: division by zero

// Runtime array error
data := [1, 2, 3]
value := data[10]  // Runtime error: index out of bounds
````

## Compilation Target

Arc compiles exclusively to WebAssembly (WASM). The compiler generates a WASM module
along with metadata describing the reactive stage graph and channel connections.

### WASM Module Structure

The generated WASM module contains:

```
Module {
  // Imported host functions (provided by runtime)
  // Note: WASM uses i32 for all <32-bit integers
  imports: [
    // Channel operations (per-type)
    "env"."channel_read_i8": [i32] -> [i32]     // Returns i8 in i32
    "env"."channel_read_i16": [i32] -> [i32]    // Returns i16 in i32
    "env"."channel_read_i32": [i32] -> [i32]
    "env"."channel_read_i64": [i32] -> [i64]
    "env"."channel_read_u8": [i32] -> [i32]     // Returns u8 in i32
    "env"."channel_read_u16": [i32] -> [i32]    // Returns u16 in i32
    "env"."channel_read_u32": [i32] -> [i32]
    "env"."channel_read_u64": [i32] -> [i64]
    "env"."channel_read_f32": [i32] -> [f32]
    "env"."channel_read_f64": [i32] -> [f64]
    "env"."channel_read_string": [i32] -> [i32] // Returns handle

    "env"."channel_write_i8": [i32, i32] -> []  // i8 passed as i32
    "env"."channel_write_i16": [i32, i32] -> [] // i16 passed as i32
    "env"."channel_write_i32": [i32, i32] -> []
    "env"."channel_write_i64": [i32, i64] -> []
    "env"."channel_write_u8": [i32, i32] -> []  // u8 passed as i32
    "env"."channel_write_u16": [i32, i32] -> [] // u16 passed as i32
    "env"."channel_write_u32": [i32, i32] -> []
    "env"."channel_write_u64": [i32, i64] -> []
    "env"."channel_write_f32": [i32, f32] -> []
    "env"."channel_write_f64": [i32, f64] -> []
    "env"."channel_write_string": [i32, i32] -> [] // Write handle

    // ... blocking reads follow same pattern

    // Series operations (type-agnostic)
    "env"."series_len": [i32] -> [i64]
    "env"."series_slice": [i32, i32, i32] -> [i32]

    // Series operations (per-type)
    "env"."series_create_empty_i32": [i32] -> [i32]
    "env"."series_create_empty_i64": [i32] -> [i32]
    "env"."series_create_empty_f32": [i32] -> [i32]
    "env"."series_create_empty_f64": [i32] -> [i32]
    // ... etc for all types

    "env"."series_index_i32": [i32, i32] -> [i32]
    "env"."series_index_i64": [i32, i32] -> [i64]
    "env"."series_index_f32": [i32, i32] -> [f32]
    "env"."series_index_f64": [i32, i32] -> [f64]
    // ... etc for all types

    // State persistence (per-type)
    "env"."state_load_i32": [i32, i32] -> [i32]
    "env"."state_load_i64": [i32, i32] -> [i64]
    "env"."state_load_f32": [i32, i32] -> [f32]
    "env"."state_load_f64": [i32, i32] -> [f64]
    "env"."state_load_string": [i32, i32] -> [i32]
    // ... etc

    // String operations
    "env"."string_from_literal": [i32, i32] -> [i32]

    // Built-in functions
    "env"."now": [] -> [i64]

    // Error handling
    "env"."panic": [i32, i32] -> []  // ptr, len to error message
  ]

  // Exported functions (one per stage/function)
  exports: [
    "func_add": function
    "task_controller": function
    "task_alarm": function
  ]

  // Linear memory for data
  memory: (initial: 1 page)
}
```

### Compilation Output Format

The compiler produces a JSON structure containing the WASM module and metadata:

```json
{
  "version": "1.0.0",
  "module": "<base64-encoded-wasm-bytes>",
  "tasks": [
    {
      "name": "controller",
      "key": "task_controller",
      "config": [
        { "name": "setpoint", "type": "f64" },
        { "name": "sensor", "type": "chan_in" },
        { "name": "actuator", "type": "chan_out" }
      ],
      "args": [{ "name": "enable", "type": "u8" }],
      "stateful_vars": [{ "name": "integral", "type": "f64", "key": 0 }],
      "return": null
    },
    {
      "name": null,
      "key": "task_expr_0",
      "config": [{ "name": "ox_pt_1", "type": "chan_in" }],
      "args": [],
      "stateful_vars": [],
      "return": "u8"
    }
  ],
  "functions": [
    {
      "name": "add",
      "key": "func_add",
      "params": [
        { "name": "x", "type": "f64" },
        { "name": "y", "type": "f64" }
      ],
      "return": "f64"
    }
  ],
  "nodes": [
    {
      "key": "controller_0",
      "type": "task_controller",
      "config": {
        "setpoint": 100,
        "sensor": "temp_sensor",
        "actuator": "valve_cmd"
      }
    },
    {
      "key": "expr_0",
      "type": "task_expr_0",
      "config": {
        "ox_pt_1": "ox_pt_1"
      }
    },
    {
      "key": "temp_sensor",
      "type": "channel_f64",
      "config": {
        "channel": "temp_sensor"
      }
    },
    {
      "key": "valve_cmd",
      "type": "channel_f64",
      "config": {
        "channel": "valve_cmd"
      }
    },
    {
      "key": "ox_pt_1",
      "type": "channel_f64",
      "config": {
        "channel": "ox_pt_1"
      }
    }
  ],
  "edges": [
    {
      "from": { "key": "temp_sensor", "param": "output" },
      "to": { "key": "controller_0", "param": "sensor" }
    },
    {
      "from": { "key": "controller_0", "param": "actuator" },
      "to": { "key": "valve_cmd", "param": "input" }
    },
    {
      "from": { "key": "expr_0", "param": "output" },
      "to": { "key": "alarm", "param": "input" }
    }
  ]
}
```

### Host Function Interface

The runtime provides these functions to the WASM module:

#### Channel Operations

```go
// Per-type channel read operations (non-blocking)
func channelReadI8(channelID uint32) int8
func channelReadI16(channelID uint32) int16
func channelReadI32(channelID uint32) int32
func channelReadI64(channelID uint32) int64
func channelReadU8(channelID uint32) uint8
func channelReadU16(channelID uint32) uint16
func channelReadU32(channelID uint32) uint32
func channelReadU64(channelID uint32) uint64
func channelReadF32(channelID uint32) float32
func channelReadF64(channelID uint32) float64
func channelReadString(channelID uint32) int32  // Returns string handle

// Per-type channel write operations
func channelWriteI8(channelID uint32, value int8)
func channelWriteI16(channelID uint32, value int16)
func channelWriteI32(channelID uint32, value int32)
func channelWriteI64(channelID uint32, value int64)
func channelWriteU8(channelID uint32, value uint8)
func channelWriteU16(channelID uint32, value uint16)
func channelWriteU32(channelID uint32, value uint32)
func channelWriteU64(channelID uint32, value uint64)
func channelWriteF32(channelID uint32, value float32)
func channelWriteF64(channelID uint32, value float64)
func channelWriteString(channelID uint32, handle int32)  // Write string by handle

// Per-type blocking read operations
func channelBlockingReadI8(channelID uint32) int8
func channelBlockingReadI16(channelID uint32) int16
func channelBlockingReadI32(channelID uint32) int32
func channelBlockingReadI64(channelID uint32) int64
func channelBlockingReadU8(channelID uint32) uint8
func channelBlockingReadU16(channelID uint32) uint16
func channelBlockingReadU32(channelID uint32) uint32
func channelBlockingReadU64(channelID uint32) uint64
func channelBlockingReadF32(channelID uint32) float32
func channelBlockingReadF64(channelID uint32) float64
func channelBlockingReadString(channelID uint32) int32  // Returns string handle
```

#### Series Operations

```go
// Type-agnostic operations
func seriesLen(seriesID int32) int64
func seriesSlice(seriesID int32, start int32, end int32) int32

// Per-type series creation
func seriesCreateEmptyI8(length int32) int32
func seriesCreateEmptyI32(length int32) int32
func seriesCreateEmptyI64(length int32) int32
func seriesCreateEmptyU8(length int32) int32
func seriesCreateEmptyU32(length int32) int32
func seriesCreateEmptyU64(length int32) int32
func seriesCreateEmptyF32(length int32) int32
func seriesCreateEmptyF64(length int32) int32

// Per-type element operations
func seriesSetElementI8(seriesID int32, index int32, value int8)
func seriesSetElementI32(seriesID int32, index int32, value int32)
func seriesSetElementI64(seriesID int32, index int32, value int64)
func seriesSetElementU8(seriesID int32, index int32, value uint8)
func seriesSetElementU32(seriesID int32, index int32, value uint32)
func seriesSetElementU64(seriesID int32, index int32, value uint64)
func seriesSetElementF32(seriesID int32, index int32, value float32)
func seriesSetElementF64(seriesID int32, index int32, value float64)

func seriesIndexI8(seriesID int32, index int32) int8
func seriesIndexI32(seriesID int32, index int32) int32
func seriesIndexI64(seriesID int32, index int32) int64
func seriesIndexU8(seriesID int32, index int32) uint8
func seriesIndexU32(seriesID int32, index int32) uint32
func seriesIndexU64(seriesID int32, index int32) uint64
func seriesIndexF32(seriesID int32, index int32) float32
func seriesIndexF64(seriesID int32, index int32) float64

// Per-type arithmetic operations (examples for F64, replicate for other types)
func seriesElementMulF64(seriesID int32, scalar float64) int32
func seriesElementAddF64(seriesID int32, scalar float64) int32
func seriesSeriesAddF64(series1 int32, series2 int32) int32
func seriesSeriesMulF64(series1 int32, series2 int32) int32

func seriesElementMulI32(seriesID int32, scalar int32) int32
func seriesElementAddI32(seriesID int32, scalar int32) int32
func seriesSeriesAddI32(series1 int32, series2 int32) int32
func seriesSeriesMulI32(series1 int32, series2 int32) int32
// ... etc for all numeric types
```

#### State Persistence

```go
// Per-type state load operations
func stateLoadI8(taskID int32, varID int32) int8
func stateLoadI16(taskID int32, varID int32) int16
func stateLoadI32(taskID int32, varID int32) int32
func stateLoadI64(taskID int32, varID int32) int64
func stateLoadU8(taskID int32, varID int32) uint8
func stateLoadU16(taskID int32, varID int32) uint16
func stateLoadU32(taskID int32, varID int32) uint32
func stateLoadU64(taskID int32, varID int32) uint64
func stateLoadF32(taskID int32, varID int32) float32
func stateLoadF64(taskID int32, varID int32) float64
func stateLoadString(taskID int32, varID int32) int32  // Returns string handle

// Per-type state store operations
func stateStoreI8(taskID int32, varID int32, value int8)
func stateStoreI16(taskID int32, varID int32, value int16)
func stateStoreI32(taskID int32, varID int32, value int32)
func stateStoreI64(taskID int32, varID int32, value int64)
func stateStoreU8(taskID int32, varID int32, value uint8)
func stateStoreU16(taskID int32, varID int32, value uint16)
func stateStoreU32(taskID int32, varID int32, value uint32)
func stateStoreU64(taskID int32, varID int32, value uint64)
func stateStoreF32(taskID int32, varID int32, value float32)
func stateStoreF64(taskID int32, varID int32, value float64)
func stateStoreString(taskID int32, varID int32, handle int32)
```

#### String Operations

```go
// Create string from literal in WASM memory
// ptr: Pointer to UTF-8 bytes in WASM data section
// len: Length in bytes
// Returns: String handle
func stringFromLiteral(ptr int32, length int32) int32

// Get string length (for future string operations)
func stringLen(handle int32) int32
```

#### Built-in Functions

```go
// Get current timestamp (nanoseconds since Unix epoch)
func now() int64
```

#### Error Handling

```go
// Trigger runtime panic with error message
// ptr: Pointer to UTF-8 error message in WASM memory
// len: Length of message in bytes
func panic(ptr int32, length int32)
```

### Memory Layout

WASM uses **no dynamic memory allocation**. All memory is either:

- **Static**: String literals in the data section
- **Stack**: Local variables and computation temporaries

```
[0x0000-0x0FFF]  Reserved (4KB)
[0x1000-0x4FFF]  String constants (16KB) - compile-time allocated
```

No heap, no malloc, no dynamic allocation. The WASM module has:

- **No linear memory exports** - runtime cannot write to WASM memory
- **Fixed memory size** - 1 page (64KB) maximum
- **Stack-only computation** - all temporaries on the operand stack

#### String Storage

String literals are stored at compile-time in the data section:

```
[offset][length: u32][utf8_bytes...]
```

String operations return handles (i32) to host-managed strings.

#### Series Storage

**Series are entirely managed by the host runtime**. WASM code never directly accesses
series memory. All series operations (creation, indexing, arithmetic) are performed via
host function calls. The WASM module only handles series as opaque handle values (i32).

### Function Calling Convention

#### Stage Functions

Tasks are compiled to WASM functions with this signature:

```wasm
;; task_name(config_channels..., runtime_params...)
(func $task_controller
  (param $sensor_chan i32)      ;; Channel IDs passed as i32
  (param $actuator_chan i32)
  (param $setpoint f64)          ;; Config values by type
  (param $enable i32)            ;; Runtime params
)
```

#### Regular Functions

Functions maintain their declared signature:

```wasm
(func $func_add
  (param $x f64)
  (param $y f64)
  (result f64)
)
```

### Error Handling Protocol

Errors are handled through two mechanisms:

1. **WASM Traps**: For unrecoverable errors
   - Division by zero
   - Out-of-bounds array access
   - Type mismatches (should be caught at compile time)

2. **Host Function Errors**: For runtime failures
   - Channel not found: Returns NaN
   - Series operation failures: Returns -1 handle
   - State persistence failures: Logged but execution continues

### Runtime Responsibilities

The runtime (execution VM) handles:

1. **Reactive Scheduling**
   - Maintains event queue for channel writes
   - Determines stage execution order
   - Implements snapshot semantics for channel reads

2. **Channel Management**
   - Implements unbounded FIFO queues
   - Routes data between tasks
   - Manages channel ID allocation

3. **Stage Lifecycle**
   - Instantiates tasks from metadata
   - Manages stage-specific state persistence
   - Handles stage activation triggers

4. **Memory Management**
   - Manages WASM linear memory growth
   - Garbage collects series handles
   - Maintains string constant table

5. **Flow Graph Execution**
   - Parses and validates flow graph from metadata
   - Creates channel connections
   - Manages data flow between tasks

### Analyzer & Compiler Interface

#### Global Resolver

The analyzer requires a resolver to validate external globals (channels, stdlib tasks,
etc):

```go
type GlobalResolver interface {
    // ResolveChannel returns the ID and type for a channel name
    // Returns error if channel doesn't exist
    ResolveChannel(name string) (uint32, string, error)

    // ResolveStdTask checks if a stage is a standard library stage
    // Returns stage signature if it exists, error otherwise
    ResolveStdTask(name string) (*StdTaskSignature, error)

    // ResolveStdFunction checks if a function is a standard library function
    // Returns function signature if it exists, error otherwise
    ResolveStdFunction(name string) (*StdFunctionSignature, error)
}

type StdTaskSignature struct {
    Name       string
    ConfigParams []ParamSpec
    RuntimeParams []ParamSpec
    Returns    string  // Return type or empty
}

type StdFunctionSignature struct {
    Name    string
    Params  []ParamSpec
    Returns string
}

type ParamSpec struct {
    Name string
    Type string
}

// StubResolver for testing or pure syntax checking
type StubResolver struct{}

func (s *StubResolver) ResolveChannel(name string) (uint32, string, error) {
    // Accept any channel name, assume f64
    return hash(name), "f64", nil
}

func (s *StubResolver) ResolveStdTask(name string) (*StdTaskSignature, error) {
    // Recognize common stdlib tasks
    switch name {
    case "interval":
        return &StdTaskSignature{
            Name: "interval",
            ConfigParams: []ParamSpec{{Name: "period", Type: "timespan"}},
        }, nil
    case "all", "any", "once", "merge":
        return &StdTaskSignature{
            Name: name,
            ConfigParams: []ParamSpec{{Name: "inputs", Type: "variadic"}},
        }, nil
    default:
        return nil, fmt.Errorf("unknown stdlib stage: %s", name)
    }
}

func (s *StubResolver) ResolveStdFunction(name string) (*StdFunctionSignature, error) {
    // Built-in functions like len(), now()
    switch name {
    case "len":
        return &StdFunctionSignature{
            Name: "len",
            Params: []ParamSpec{{Name: "series", Type: "series"}},
            Returns: "i64",
        }, nil
    case "now":
        return &StdFunctionSignature{
            Name: "now",
            Params: []ParamSpec{},
            Returns: "timestamp",
        }, nil
    default:
        return nil, fmt.Errorf("unknown stdlib function: %s", name)
    }
}
```

The analyzer uses this interface to:

- Validate external channel references exist and get their types
- Verify stdlib stage invocations are valid
- Check stdlib function calls have correct arguments
- Provide autocomplete suggestions in LSP (all available globals)

The compiler uses the validated information to:

- Generate channel nodes in the graph with correct IDs
- Mark stdlib tasks in nodes (not generating WASM for them)
- Emit correct host function calls for stdlib functions

#### Standard Library Tasks

These tasks are recognized by the compiler but implemented by the runtime:

- `interval{period: duration}` - Emits at regular intervals
- `all{inputs...}` - Waits for all inputs
- `once{input}` - Emits only first value
- `any{inputs...}` - Emits on any input
- `throttle{rate: frequency}` - Rate limits
- `merge{inputs...}` - Combines streams
- `tee{outputs...}` - Splits stream

The compiler:

- Parses their configuration
- Adds them as nodes with type = stage name
- Does NOT generate WASM functions for them
- Runtime provides the implementation

### Compilation Pipeline

The Arc compilation pipeline consists of three distinct phases:

#### Phase 1: Parser (1 pass)

**Input:** Source text **Output:** Abstract Syntax Tree (AST) **Responsibilities:**

- Lexical analysis (tokenization)
- Syntactic analysis (grammar validation)
- Build parse tree structure
- Report syntax errors

The parser does NOT:

- Check types
- Validate identifiers exist
- Understand semantics

#### Phase 2: Analyzer (2 passes)

**Input:** AST + GlobalResolver **Output:** Validated AST + Symbol Table + Diagnostics

**Pass 1: Symbol Collection**

- Build symbol table for all declarations:
  - User-defined tasks and functions
  - Variables (local and stateful)
  - Parameters
- Collect all identifier references:
  - Channel names
  - Stage invocations
  - Function calls

**Pass 2: Validation**

- Type check all expressions
- Verify identifier resolution:
  - Variables are defined before use
  - Functions/tasks exist (user-defined or stdlib)
  - Channels exist (via GlobalResolver)
- Validate flow statements:
  - Stage invocations have valid config
  - Channel connections type-match
- Check for cycles in stage graph
- Ensure all code paths return (for non-void functions)

The analyzer does NOT:

- Generate any code
- Build runtime data structures
- Assign instance IDs

#### Phase 3: Compiler (2 passes)

**Input:** Validated AST + Symbol Table + GlobalResolver **Output:** WASM module +
Metadata (JSON)

**Pass 1: Collection & Graph Building**

- Extract specifications:
  - Stage specs → tasks array
  - Function specs → functions array
- Process flow statements:
  - Create node instances (with unique keys)
  - Build edges from connections
  - Generate anonymous tasks for inline expressions
- Resolve channel IDs via GlobalResolver

**Pass 2: Code Generation**

- Generate WASM functions:
  - One per user-defined stage/function
  - One per inline expression
  - Map Arc types to WASM types
- Emit host function calls for:
  - Channel operations
  - Series operations
  - State persistence
  - Built-in functions
- Build string table for literals
- Assemble final WASM module

The compiler does NOT:

- Validate types (trusts analyzer)
- Check identifier existence
- Report semantic errors

### Division of Responsibilities

| Component           | Parser | Analyzer | Compiler |
| ------------------- | ------ | -------- | -------- |
| Syntax validation   | ✓      |          |          |
| Type checking       |        | ✓        |          |
| Channel validation  |        | ✓        |          |
| Symbol resolution   |        | ✓        |          |
| Cycle detection     |        | ✓        |          |
| Flow graph building |        |          | ✓        |
| Instance creation   |        |          | ✓        |
| WASM generation     |        |          | ✓        |
| Metadata output     |        |          | ✓        |

### Error Reporting

- **Parser errors**: Syntax errors only (missing braces, invalid tokens)
- **Analyzer errors**: Semantic errors (type mismatches, undefined variables, missing
  channels)
- **Compiler errors**: Should be rare (out of memory, WASM limits exceeded)
