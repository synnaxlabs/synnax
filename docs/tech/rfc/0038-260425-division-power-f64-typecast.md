# 38 - Division and Power f64 Typecast

**Feature Name**: Division and Power f64 Typecast <br /> **Start Date**: 2026-04-25
<br /> **Authors**: Nico Alba <br />

**Related:** [Trade Study](0038-260425-divide-typecast-f64-trade-study.md)

# 0 - Summary

All `/` and `^` operations in Arc will return `f64` regardless of input type. Division
by zero will produce IEEE 754 values (`+Inf`, `-Inf`, `NaN`) instead of panicking and
crashing the flow scheduler. The
[trade study](0038-260425-divide-typecast-f64-trade-study.md) evaluates and justifies
this decision against the alternative of preserving the input type.

The changes span the analyzer, compiler, WASM host functions, flow node runtime, and
series operations across Go and C++. `math.divide` will become a private
(non-user-accessible) implementation detail; users will interact with it through the `/`
operator and flow node only.

# 1 - Goals

**Reliability and ease of use.** Arc is a high-level telemetry language for operators,
engineers, and IT professionals. Its users come from Python, LabVIEW, and MATLAB.
Division and power should produce mathematically correct results by default, not
silently truncate or panic. The `/` operator should behave the way it does in every
language Arc's users already know.

**Control safety.** Divide-by-zero must not crash the scheduler or silently activate
control logic. IEEE 754 special values (`+Inf`, `-Inf`, `NaN`) are deterministic,
visible in the UI, and non-truthy in Arc's control model.

**Consistency.** Both execution contexts (flow nodes and WASM `func` blocks) should
produce identical results for the same operation. Under the current behavior, a
`divide{}` flow node and division inside a `func` block have different failure modes on
divide-by-zero.

# 2 - Non-Goals

- **Integer division operators.** `//`, `math.quotient()`, and `math.floor_division()`
  are future work if requested. This RFC only changes `/` and `^`.
- **Modulo type promotion.** `%` will not promote to `f64`. It returns the input type
  (int in, int out; float in, float out). It will be expanded to accept float inputs,
  matching numpy and LabVIEW.
- **Overflow behavior.** Add, subtract, and multiply continue to return the input type.
  Overflow is a container-width problem, not a numeric-domain problem (see trade study).
- **New error handling infrastructure.** IEEE 754 handles divide-by-zero natively. No
  new error ABI, sentinel values, or skip-tick logic is needed.

# 3 - Behavior Changes

## 3.0 - Before and After

| Expression            | Before (current)          | After (proposed)                              |
| --------------------- | ------------------------- | --------------------------------------------- |
| `1 / 2`               | `i64(0)`                  | `f64(0.5)`                                    |
| `7 / 3`               | `i64(2)`                  | `f64(2.333...)`                               |
| `2 ^ -1`              | `i64(0)`                  | `f64(0.5)`                                    |
| `n / 0` (integer)     | panic (crashes scheduler) | `f64(+Inf)`                                   |
| `0 / 0`               | panic (crashes scheduler) | `f64(NaN)`                                    |
| `a / b * c` (all i64) | `i64` throughout          | `a / b` produces `f64`, `c` promoted to `f64` |

## 3.1 - What Does NOT Change

Add, subtract, multiply, modulo, bitwise operations, comparisons, and unary negation
will still return the input type. These operations always produce exact integer results
from integer inputs, so no typecast is needed. Only division and power are affected,
because they are the only arithmetic operations that can produce non-integer results
from integer inputs.

## 3.2 - Control Safety: `isSeriesTruthy`

`+Inf`, `-Inf`, and `NaN` will all be treated as **non-truthy** values. This will
prevent divide-by-zero results from triggering downstream control logic:

```arc
// sensor_b is 0, so sensor_a / sensor_b = +Inf
// +Inf is non-truthy, so the valve does NOT open
if sensor_a / sensor_b {
    open_valve()
}
```

All three IEEE 754 special values (`+Inf`, `-Inf`, `NaN`) will follow this rule. A
divide-by-zero will never silently activate a control path.

## 3.3 - Future: Integer Division

If integer division (truncating) is needed in the future, it can be added as a `//`
operator or `math.quotient()` function without changing existing `/` behavior. This
mirrors the Python 3 distinction between `/` (float) and `//` (integer).

# 4 - Changes by Layer

## 4.0 - Type Inference: Analyzer

**Files:** `arc/go/analyzer/types/infer_expression.go`

- **`InferMultiplicative`**: When any operator in the expression is `/`, the result type
  will be forced to `f64` (or `series<f64>` for series operands). Currently, only series
  division is promoted to f64.
- **`InferPower`**: Will always return `f64`. Currently it returns the base operand's
  type.

## 4.1 - Scalar Division: Compiler

**Files:** `arc/go/compiler/expression/binary.go`,
`arc/go/compiler/expression/compiler.go`

Instead of emitting native WASM division opcodes (e.g. `i64.div_s`), the compiler will
emit a call to the `math.divide` host function, which will accept operands in their
native type and return `f64`. This is consistent with how `^` already uses a host
function call.

## 4.2 - Scalar Power: Compiler

**Files:** `arc/go/compiler/expression/binary.go`

Both operands will be converted to `f64` before calling `math.pow(f64, f64) -> f64`.
Currently, `math.pow` accepts and returns the input type.

## 4.3 - Hint Stripping and Chained Expressions: Compiler

**Files:** `arc/go/compiler/expression/binary.go`, `arc/go/compiler/resolve/emit.go`

- **Hint stripping**: When the first operator in a multiplicative expression is `/`, the
  compiler will strip the caller's type hint so operands compile in their natural type.
  The f64 conversion will be handled by the host function, not by hint propagation.
- **Chained expressions**: After a division, subsequent operations in the same
  expression (e.g. `a / b * c`) will convert the right-hand operand to `f64` to match
  the intermediate result.

## 4.4 - f64 Conversion Helper: WASM Writer

**Files:** `arc/go/compiler/wasm/writer.go`

`WriteConvertToF64(fromType)` will be added to the WASM writer to emit the appropriate
conversion opcode (`f64.convert_i32_s`, `f64.promote_f32`, etc.) based on the source
type. No-op for `f64`.

## 4.5 - Host Functions: WASM Runtime

**Files:** `arc/go/stl/math/math.go`, `arc/cpp/stl/math/math.h`

- **`math.divide`** host functions (`divide_i32`, `divide_i64`, `divide_f32`,
  `divide_f64`, etc.): Will change from `(T, T) -> T` to `(T, T) -> f64`. Integer
  operands will be cast to `float64` before division, producing IEEE 754 results
  (`+Inf`, `-Inf`, `NaN`) on division by zero instead of panicking.
- **`math.pow`** host functions: Will change from `(T, T) -> T` to `(T, T) -> f64`.
  Integer power will use `math.Pow(float64(base), float64(exp))` instead of the custom
  `IntPow` utility.
- **`math.divide` will be private**: The `math.divide` symbol will not be
  user-accessible. It will serve as an internal implementation backing the `/` operator
  and the `divide{}` flow node.

## 4.6 - Flow Node Runtime

**Files:** `arc/go/stl/math/math.go`, `arc/go/runtime/node/state.go`

- **`divide{}` flow node**: Will get a dedicated code path (separate from add/subtract/
  multiply) that outputs `f64` series regardless of input type.
- **`isSeriesTruthy`**: See [3.2 - Control Safety](#32---control-safety-isseriestruthy).

## 4.7 - Series Operations

**Files:** `x/go/telem/op/gen.go`, `x/go/telem/op/op_generated.go`

- **`DivideXxx` operations** (e.g. `DivideI64`, `DivideU32`): Will change to always
  output a `Float64T` series. Operands will be cast to `float64` before division.
- Integer divide-by-zero guard will be removed: `float64` division handles it natively.

## 4.8 - LSP Hover Documentation

**Files:** `arc/go/lsp/hover.go`

- `math.divide` hover doc will be updated to note f64 output and IEEE 754 divide-by-zero
  behavior.
- `math.pow` hover doc will be updated to note f64 output.

# 5 - Migration

Existing automation that writes the output of a division or power operation to a
non-`f64` channel will fail to compile after this change. The compiler will report a
clear type mismatch error:

```
type mismatch: cannot write f64 to channel 'some_int_channel' (type i32)
```

The fix is straightforward: cast the result before writing:

```arc
// Before (will fail to compile):
some_int_channel = sensor_a / sensor_b

// After (option 1: cast to match existing channel):
some_int_channel = i32(sensor_a / sensor_b)

// After (option 2: use an f64 channel instead):
some_float_channel = sensor_a / sensor_b
```

The same applies to compound assignment (`/=`) on integer channels, which would need to
be rewritten as an explicit assignment with a cast.

In practice, integer-to-integer division that silently truncates to `0` (e.g. `1 / 2`)
is rarely the intended behavior, so most affected programs are likely already producing
incorrect results.

# 6 - Change Footprint

~30 files, ~1400 insertions, ~1100 deletions.

| Area             | Files | Insertions | Deletions |
| ---------------- | ----- | ---------- | --------- |
| Go (production)  | ~10   | ~1000      | ~800      |
| Go (tests)       | ~15   | ~350       | ~260      |
| C++ (production) | ~2    | ~35        | ~20       |
| C++ (tests)      | ~2    | ~25        | ~10       |
| Python (tests)   | ~1    | ~40        | ~1        |

The Go production numbers are inflated by `op_generated.go` (~700 lines), which is
auto-generated from `gen.go`. The actual hand-written production changes are ~300 lines
across ~9 files. The bulk of the test changes are updating expected return types and
opcodes, not new test logic.

# 7 - Test Coverage

All existing test suites will need to be updated to expect `f64` output from division
and power:

- **Compiler unit tests**: ~60 entries across expression, statement, and literal tests
- **Analyzer type inference tests**: Division and power return type expectations
- **WASM integration tests**: End-to-end execution through wazero
- **Series operation tests**: `telem/op` generated operation tests
- **C++ runtime tests**: `math.h` host function tests
- **Integration tests**: `stl_math.py` end-to-end arc program tests

New test scenarios required by this change:

- **Division by zero**: `n/0` produces `+Inf`, `-n/0` produces `-Inf`, `0/0` produces
  `NaN` (instead of panicking)
- **`isSeriesTruthy` on special values**: `+Inf`, `-Inf`, and `NaN` all return
  non-truthy
- **Integer division output type**: `i32 / i32` returns `f64`, `i64 / i64` returns `f64`
- **Series integer division output type**: `series<i64> / scalar` returns `series<f64>`
- **Chained expressions**: `a / b * c` produces `f64` throughout when inputs are integer
- **f32 precision gain**: `f32 / f32` computed in `f64` produces a more precise result
- **Negative exponents**: `2 ^ -1` returns `f64(0.5)`, not `i64(0)`
- **Type mismatch error**: Assigning `f64` division result to an `i32` channel produces
  a compile-time error
