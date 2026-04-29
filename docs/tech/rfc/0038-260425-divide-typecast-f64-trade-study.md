# Division and Power Output Type: Trade Study

## Overview

This trade study evaluates two options for the output type of `divide` and `pow`
operations in Arc, and the behavior of division by zero.

## Options

- **Option A**: `divide` and `pow` always return `f64`. Division by zero produces IEEE
  754 values (`+Inf`, `-Inf`, `NaN`). This is what LabVIEW, numpy, and scipy do.
- **Option B**: `divide` and `pow` return the input type (no typecast to `f64`).
  Division by zero must be handled separately, either by erroring, trapping, or
  producing a sentinel value. This is what C does.

---

## Trade Criteria

### Type Strictness and Strong Typing

These terms overlap in casual use but mean different things:

- **Strictly typed**: Every value has a definite type known at compile time, and the
  compiler enforces type rules at every boundary.
- **Strongly typed**: No implicit coercion between unrelated types. The programmer must
  explicitly convert (e.g., JavaScript's `"5" + 3 = "53"` is weak typing).

Option A satisfies both. The `/` operator has a **concrete, compile-time return type:
`f64`**. Always. There is no ambiguity, no runtime coercion, no type that depends on
context. If you try to assign the result to an `i32` channel, the compiler rejects it
with a clear error. That is the opposite of "fuzzy."

The concern behind Option B is often: "the output type doesn't match the input type, and
that feels surprising." That is a fair UX reaction, but it is a preference, not a
type-system property. LabVIEW is considered strongly typed: every wire has an explicit
type, and the divide node explicitly returns `DBL` (double).

**"Type-strict" means the compiler enforces types at every boundary. It does not mean
outputs must match inputs.**

### Type System Consistency

**What "consistency" means here.** Option A's consistency is that each operator returns
a type matching **what the operation actually represents**, not that every operator
returns its input type. `+`, `-`, `*`, `%` stay in the integer domain because they
produce exact integer results from integer inputs. `/` and `**` do not, so they return
`f64`. The rule is in the math (like derivative), not in matching the input datatype,
which ignores what the operators actually do.

**"Divide and pow behave differently from add/subtract/multiply."**

LabVIEW, numpy, and scipy all have separate divide (float) and quotient (integer)
operations. This is standard. Both `divide` and `pow` can produce non-integer results
from integer inputs (`1/2 = 0.5`, `2**-1 = 0.5`), unlike add, subtract, and multiply,
which always produce exact integers from integer inputs.

Arc already has precedent: `math.derivative` returns `f64` regardless of input type
because rate of change is inherently continuous. Division and power have the same
property.

**"A user might expect `i32 / i32 → i32`."**

Arc's users come from Python, LabVIEW, and MATLAB. In all of these environments, `/`
produces a result in the continuous domain. Customer research confirms this expectation.
A C-style integer truncation would surprise the majority of Arc's actual users.

Two integer division variants can be added later if requested: `math.floor_division()`
(round toward negative infinity, Python `//` semantics: `-7 // 2 = -4`) and
`math.quotient()` (truncate toward zero, C semantics: `-7 / 2 = -3`). These differ only
on negative inputs. YAGNI applies to both. For now, `i32(math.divide(a, b))` works for
anyone who needs integer output.

**Existing behavior in the Synnax ecosystem:**

In Python, `int(1) / int(3)` returns `float(0.333...)`. In numpy,
`np.int64(1) / np.int64(3)` returns `float64(0.333...)` and `np.int64(1) / np.int64(0)`
returns `float64(inf)`. The Synnax Python client's `Series.to_numpy()` returns
`numpy.ndarray`, so when a user queries two integer series from Synnax and divides them,
numpy casts to `float64` and produces `+/-Inf`/`NaN` on zero denominators. Option A
makes Arc consistent with behavior that already exists in our own client library. The
Python client can already write `+Inf`, `-Inf`, and `NaN` directly to `f64` channels:

```python
import synnax as sy
import numpy as np

client = sy.Synnax()

idx = client.channels.retrieve("inf_nan_time")
ch = client.channels.retrieve("inf_nan_f64")

VALUES = [np.inf, -np.inf, np.nan]

now = sy.TimeStamp.now()
timestamps = [int(now + i * sy.TimeSpan.SECOND) for i in range(len(VALUES))]

with client.open_writer(start=now, channels=[idx.key, ch.key]) as w:
    w.write({idx.key: timestamps, ch.key: VALUES})
```

For power, numpy's `np.power(int, negative_int)` raises `ValueError` entirely rather
than produce a wrong integer answer. `np.float_power` always returns `float64`. Option A
uses `float_power` semantics for both `divide` and `pow`.

### Precision Errors

| Scenario       | Option A (`f64` output)  | Option B (integer output)   |
| -------------- | ------------------------ | --------------------------- |
| `1 / 2`        | `0.5`, correct           | `0`, 100% error             |
| `7 / 3`        | `2.333...`, correct      | `2`, 14% error (truncation) |
| `(2^53+1) / 1` | loses 1 bit of precision | exact                       |

Option B has increased error due to not having the ability to represent precision for a
solution in the continuous domain. Option A only loses precision for integers exceeding
2^53 (~9 quadrillion), which is outside the range of any real sensor.

For `f32` inputs, promoting to `f64` is a free precision gain: `f32` has ~7 decimal
digits of precision, while `f64` has ~15. `f32(1.0) / f32(3.0)` computed in `f64`
produces a more accurate result than computing in `f32`. Option B would return the less
precise `f32` answer.

#### Timestamps

Synnax timestamps are int64 nanoseconds since epoch (~1.7 x 10^18), which exceeds f64's
2^53 exact integer range. So `timestamp / n` would lose nanoseconds.

However, dividing a raw timestamp (`now / 3`) is not a meaningful operation. What is
meaningful is dividing time _deltas_ (`(now - yesterday) / 3`). A one-day delta is ~8.6
x 10^13 nanoseconds (well within 2^53). Even a one-year delta (~3.15 x 10^16) only loses
~2-4 nanoseconds of precision in the f64 representation, which is negligible for any
real sensor application.

More importantly, under integer division, `timedelta / n` where `timedelta < n` returns
**zero**. That is not a precision loss; that is a total loss. The f64 path gives a
meaningful sub-nanosecond answer; the integer path gives nothing.

### Overflow and Consistency with Other Operations

**"If division auto-promotes to f64, shouldn't multiplication also auto-promote to
prevent overflow? Otherwise the type system is inconsistent."**

No. Division and overflow are fundamentally different problems:

- Division and power produce results that **leave the integers entirely**. `1 / 2 = 0.5`
  is not representable as any integer type. The correct result lives in a different
  numeric domain.
- Overflow produces a result that **is still an integer**, just one that doesn't fit in
  the container. `u8(255) * u8(255) = 65025` is a perfectly valid integer; the problem
  is the container width, not the numeric domain.
- For division, there is a single well-defined promotion target: `f64`. For overflow,
  there is no single right answer: promote to `u16`? `u32`? `u64`? It depends on the
  values at runtime, which would require arbitrary-precision integers or
  runtime-dependent types, both of which actually _would_ make the type system fuzzy.

If multiplication promotes to prevent overflow, so must addition and subtraction (they
also overflow). Then every arithmetic operation returns a wider type, and having
different integer widths becomes pointless.

Every mainstream language treats these differently. Python 3: `int / int = float`, but
`int * int = int`. Rust, Go, C: multiplication wraps, division truncates. Nobody
auto-promotes multiplication.

Arc integer arithmetic wraps on overflow (two's complement, WASM semantics), the same as
C and Go.

**Overflow is a width problem. Division is a domain problem. Different problems,
different solutions.** Overflow behavior is out of scope for this RFC and is an
independent design decision that does not affect the case for Option A.

### Divide by Zero

| Scenario             | Option A (`f64`)                                                                           | Option B (integer)                               |
| -------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------ |
| `n/0`, `-n/0`, `0/0` | `+Inf`, `-Inf`, `NaN`. Deterministic per IEEE 754.                                         | Panic (current), or requires new error handling. |
| Downstream impact    | `NaN` is non-truthy (won't trigger control). `+Inf`/`-Inf` are truthy, matching Python/JS. | No output, possible cascade failure.             |
| Debugging            | `+Inf`, `-Inf`, `NaN` visible in the Log view, traceable.                                  | Depends on resolution.                           |
| Recovery             | System keeps running.                                                                      | Depends on resolution.                           |

#### What Option B actually requires

The current behavior before this change is a panic that crashes the flow scheduler. That
is a bug, not a design choice. If we pursued Option B properly, we would need to resolve
it. The options are different depending on context:

**Flow nodes** (e.g. `divide{}` in flow statements):

1. **Don't output** (skip the tick). Downstream nodes never fire. The user gets no
   signal that something went wrong. Silent failure.
2. **Report an error**. `node.Context` has a `report_error` callback, but it is for
   logging, not stopping execution. We would need to define what "error" means: stop the
   whole program? Just this node? Just this tick?
3. **Output a sentinel value** (e.g. 0). This is lying about the result, which is the
   original problem.

**WASM functions** (division inside `func` blocks):

WASM already traps on integer divide-by-zero (controlled termination). But division
backed by host functions returns values, not errors. We would need an error ABI (return
a tuple, or set a global error flag) to surface the failure.

**The inconsistency problem**: under Option B, a `divide{}` flow node behaves
differently from division in a `func` block. The user writes the same operation in two
contexts and gets two different failure modes.

Option A avoids all of this. Both paths produce the same IEEE 754 result. No error
handling infrastructure needed. No inconsistency between flow and WASM.

### Language Identity

Arc's actual users convert Python scripts and LabVIEW diagrams. The C user is
hypothetical.

| Context                            | Option A                                               | Option B  |
| ---------------------------------- | ------------------------------------------------------ | --------- |
| Arc users converting Python        | Matches Python `/` behavior                            |           |
| LabVIEW precedent (decades, FPGAs) | Exact same behavior (divide returns DBL)               |           |
| C behavior expectation             | Addressed by future `//` operator if needed            | Matches C |
| FPGA deployment concern            | LabVIEW proves float divide works predictably on FPGAs |           |

Designing for a hypothetical C user at the expense of current Python and LabVIEW users
violates YAGNI.

### Reliability

What does "reliable" mean for a telemetry language?

The operations that preserve input type (add, subtract, multiply, min, max) do so
because integer inputs always produce exact integer results. Division and power do not
have that property: `1 / 2` is `0.5`, `2**-1` is `0.5`. Returning `f64` is the
mathematically correct choice, consistent with how `math.derivative` already works.

The counterargument is that low-level hardware engineers expect C-style integer
division. Every customer we have spoken to disagrees. They come from Python, LabVIEW, or
MATLAB, where division returns floats.

### Safety

`+Inf`, `-Inf`, and `NaN` are safe in Arc's control model:

- **Predictable truthiness**: `isSeriesTruthy` treats `NaN` as non-truthy (it is
  genuinely undefined, since `0/0` has no magnitude). `+Inf` and `-Inf` are truthy,
  consistent with Python and JavaScript. Programs that need to guard against
  divide-by-zero triggering control should check the denominator explicitly.
- **Visible**: `∞`/`Infinity`, `-∞`/`-Infinity`, and `NaN` already render in the UI.
  They are not silent.
- **Consistent**: Both flow nodes and WASM functions produce the same values.

Under Option B, either the system outputs incorrect data (sentinel value of 0), silently
drops the output (no signal to the user), or behaves differently depending on whether
the division is in a flow node or a `func` block. All three are worse for safety than a
visible IEEE 754 value with well-defined truthiness semantics.

---

## Trade Summary

| Criterion            | Option A | Option B |
| -------------------- | -------- | -------- |
| Type strictness      | ✓        | ✓        |
| Type consistency     | ✓        |          |
| Precision            | ✓        |          |
| Overflow consistency | ✓        |          |
| Divide by zero       | ✓        |          |
| Language identity    | ✓        |          |
| Reliability          | ✓        |          |
| Safety               | ✓        |          |

## Recommendation

Option A is the recommended option. It wins on every criterion evaluated in this study
except the timestamp precision sub-topic, where Option B preserves exact nanosecond
values. Under Option A, the precision loss is ~2-4 nanoseconds on year-long time deltas.

The intuition behind Option B is reasonable. "The output type should match the input
type" is a clean mental model, and integer division is familiar to anyone who has
written C. But that intuition breaks down when applied to operations that are
mathematically continuous: `1 / 2` is `0.5`, not `0`. Every language that our users
actually come from (Python, LabVIEW, MATLAB, numpy) resolved this the same way. Option B
is not more type-safe or strict than Option A. The `/` operator under Option A has a
single, concrete, compile-time return type (`f64`). That is the definition of
type-strict. The argument that matching input types is "stricter" confuses a UX
preference with a type-system property. Option B also introduces more uncertainty,
unreliability, and therefore, increased risk.

If integer division is needed in the future, `math.floor_division()` (toward negative
infinity) and `math.quotient()` (truncation toward zero) can be added without changing
existing behavior.
