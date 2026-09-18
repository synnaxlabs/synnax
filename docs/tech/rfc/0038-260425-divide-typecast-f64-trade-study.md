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

### Static and Strong Typing

A common framing of the Option B preference is that Option A "feels less type-safe"
because the output type differs from the input. That conflates a UX preference with the
two canonical type-system properties the trade actually turns on:

- **Static typing**: every value has a type known at compile time, and the compiler
  enforces type rules before the program runs.
- **Strong typing**: the language does not silently coerce values between unrelated
  types at user-visible boundaries. The programmer must explicitly convert (JavaScript's
  `"5" + 3 = "53"` is weak typing).

These are the standard programming-language type axes: see Pierce, _Types and
Programming Languages_ (MIT Press, 2002), §1.1, and Cardelli & Wegner, "On Understanding
Types, Data Abstraction, and Polymorphism" (ACM Computing Surveys, 17(4), 1985). The
strong/weak distinction converges on "no implicit coercion between unrelated types"
across [Wikipedia](https://en.wikipedia.org/wiki/Strong_and_weak_typing), Python's
[language reference](https://docs.python.org/3/reference/datamodel.html), and the
[Haskell wiki](https://wiki.haskell.org/Type), all of which describe their respective
languages as strongly typed.

Option A satisfies both.

**Static.** The `/` operator has a single, concrete, compile-time return type: `f64`.
The return type does not depend on runtime values, on the inferred type of either
operand, or on context. The analyzer fixes it during type inference.

**Strong.** Assigning the result of `/` to a non-`f64` destination is a compile error:

```arc
some_int_channel = sensor_a / sensor_b
// type mismatch: cannot write f64 to channel 'some_int_channel' (type i32)
```

The user must explicitly cast (`i32(sensor_a / sensor_b)`) or change the destination.
There is no silent bridging between unrelated types.

The internal promotion of integer operands to `f64` inside the divide operation is part
of the operator's defined signature `(T, T) -> f64`, not implicit coercion across the
type system. Every strongly typed language with numeric operators does this: Python
(`int / int -> float`), Haskell (typeclass-driven numerics), and LabVIEW (divide node
returns `DBL` from integer inputs) are all considered strongly typed.

The intuition that "outputs should match inputs" is a reasonable UX preference, but it
is not what static or strong typing means. LabVIEW is the closest precedent in the
safety-critical engineering domain Arc targets, and its divide node explicitly returns
`DBL`.

### Type System Consistency

Two type-consistency criteria are in tension, and the Trade Summary scores them
separately:

- **Type consistency** (this section's primary focus): does each operator's return type
  match what the operation semantically produces? Closed operators (`+`, `-`, `*`, `%`)
  stay in the integer domain because their result is exact. Non-closed operators (`/`,
  `**`) promote to `f64` because their result is continuous. Option A satisfies this;
  Option B does not.
- **Same-type rule uniformity**: does one surface rule (operands must match, output type
  matches input type) apply across all operators? Option B satisfies this; Option A does
  not, because `/` and `**` are deliberate exceptions.

The rest of this section explains why the trade-off resolves toward Option A.

**Principle: minimize coercion.** Coercion is the source of the most-cited class of
silent numeric bugs in industrial control languages. Arc's rule is to require coercion
to be explicit everywhere it can be, and to apply a single uniform rule where the math
forces it. Concretely, `+`, `-`, `*`, `%` are closed over the integer domain (integer
inputs produce exact integer results), so they require same-type operands and offer no
implicit conversion. `/` and `**` are not closed (`1 / 2 = 0.5`, `2**-1 = 0.5`); they
are the only binary arithmetic operators whose mathematical answer falls outside the
input domain for non-trivial inputs. Arc already has this pattern: `math.derivative`
returns `f64` regardless of input type because rate of change is inherently continuous.
`/` and `**` share that property.

**Cautionary evidence: LabVIEW.** NI's own LabVIEW documentation is the canonical
reference for what pervasive auto-coercion does in practice. The
[coercion-dots page](https://www.ni.com/docs/en-US/bundle/labview/page/coercion-dots.html)
documents `i16(-10) + u16(5)` silently producing `u16(65531)`: LabVIEW promotes the
signed value to the unsigned type and the answer is wrong by 65,536. NI's recommended
remediation is to match data types at the wire. Arc adopts the same conclusion
structurally: do not coerce on `+`, `-`, `*`, `%`. Make the user write the cast.

**Precedent for the split: Python 3 and NumPy.** Both already make exactly the split Arc
proposes:

- **Python 3**: `int(1) + int(2)` returns `int(3)`, but `int(1) / int(2)` returns
  `float(0.5)`. Same-type rule preserved for closed operators, broken deliberately for
  `/`.
- **NumPy**: `np.int32(1) + np.int32(2)` returns `np.int32(3)`, but
  `np.int32(1) / np.int32(2)` returns `np.float64(0.5)`. Same split. NumPy preserves
  `f32` when both inputs are `f32`; Arc's choice to promote `f32 / f32` to `f64` extends
  one step further as part of the single-target-type rule below.

**Single target type, owning the exception.** `/` and `**` always return `f64`,
regardless of input type. The alternative (return `f32` for `f32` inputs, `f64` for
everything else) is exactly the input-dependent output-type rule that LabVIEW's docs
warn about. One rule, uniformly applied, is easier to teach and harder to misuse. This
is the deliberate exception to the same-type rule that `+`, `-`, `*`, `%` enforce. A
user can write `i32(1) / f32(2)` but not `i32(1) + f32(2)`, and that is a real
source-level inconsistency. The exception is justified because the math is different and
the alternative (`/` and `**` also requiring explicit casts) creates more, not fewer,
opportunities for the user to silently get the wrong answer (`1 / 2` returns `0`,
`2**-1` returns `0`). The exception is communicated to the user in the editor via
operator tooltips (see [§Operator Tooltips](#operator-tooltips)).

Two integer division variants can be added later if requested: `math.floor_division()`
(round toward negative infinity, Python `//` semantics: `-7 // 2 = -4`) and
`math.quotient()` (truncate toward zero, C semantics: `-7 / 2 = -3`). These differ only
on negative inputs. YAGNI applies to both. For now, `i32(math.divide(a, b))` works for
anyone who needs integer output.

### Precision Errors

"Precision" here is proximity to the mathematical answer in the continuous domain, not
"does the operator return its own defined output." That alternative framing is
meaningless because every operator passes it by construction: an integer-division `/` is
trivially "precise" since returning `2` for `7/3` is exactly what it is defined to do.
The meaningful question is which result lands closer to the mathematical answer.

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

Option A avoids all of this. Both paths produce the same IEEE 754 result. No error
handling infrastructure needed.

The implementation cost enumerated here is scored separately in
[§Error Handling](#error-handling).

### Error Handling

A trade-study criterion separate from §Divide by Zero. §Divide by Zero asks what happens
for a specific input; this section asks what infrastructure each option requires to
surface runtime conditions through the type and ABI surface. The conditions in scope are
divide-by-zero, undefined results (`0/0`), and operations whose result cannot be
represented in the nominal output type.

**Option A: errors are values.** `+Inf`, `-Inf`, and `NaN` are valid `f64` values that
propagate through subsequent operations per IEEE 754. The expression `a / b` is
well-defined for any `a`, `b`: when `b == 0`, the result is `+Inf`, `-Inf`, or `NaN`,
all fully specified. The user does not write error-handling code at the operator
boundary; they check for non-finite values where it actually matters (typically before
writing to a control channel). No new calling-convention surface is required.

**Option B: errors must be plumbed.** Integer divide-by-zero is not a value, so it has
to be signaled out-of-band. Arc's two execution contexts (flow nodes, `func` blocks)
need a shared error-ABI surface so the same expression fails the same way in both. The
plumbing required in each context is enumerated above in
[§What Option B actually requires](#what-option-b-actually-requires): for flow nodes, a
definition of what "error" means and how it propagates; for `func` blocks, an
error-return convention for host functions to match the WASM trap behavior on native
integer divide. The cross-context divergence under the current implementation is covered
in [§Reliability and User Expectations](#reliability-and-user-expectations) under
"Cross-context consistency."

**The trade.** Option A does not require an error-handling system for these operators.
Option B does, and the cost is a calling-convention extension that touches host-function
ABIs, the WASM runtime, the flow scheduler, and every consumer of these operators in
user code. That is a substantial implementation cost the trade summary scores as a
separate criterion.

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

### Reliability and User Expectations

Reliability for a telemetry language means three things: the program does what users
mean, fails visibly when something goes wrong, and behaves the same way across every
context it can run in. Option A delivers all three; Option B fails on each. The Trade
Summary scores "Cross-context consistency" as a separate row because it also depends on
runtime ABI choices, not solely on user-facing behavior; the other two properties are
scored under "Reliability."

**Results match user expectations.** Arc's users come from Python, LabVIEW, MATLAB, and
numpy, where `/` produces a result in the continuous domain. Customer research confirms
this expectation. Under Option B, a user writing `1 / 2` and expecting `0.5` gets `0`
instead, a silent wrong answer fed straight into a control loop. That is the worst kind
of reliability failure: the program runs, produces a plausible-looking number, and there
is no signal to the operator until something downstream breaks.

This expectation already lives in the Synnax ecosystem. In Python, `int(1) / int(3)`
returns `float(0.333...)`. In numpy, `np.int64(1) / np.int64(3)` returns
`float64(0.333...)` and `np.int64(1) / np.int64(0)` returns `float64(inf)`. The Synnax
Python client's `Series.to_numpy()` returns `numpy.ndarray`, so when a user queries two
integer series from Synnax and divides them, numpy casts to `float64` and produces
`+/-Inf`/`NaN` on zero denominators. Option A makes Arc consistent with behavior that
already exists in our own client library. The Python client can already write `+Inf`,
`-Inf`, and `NaN` directly to `f64` channels:

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

For power, numpy's `np.power(int, negative_int)` raises `ValueError` rather than produce
a wrong integer answer. `np.float_power` always returns `float64`. Option A uses
`float_power` semantics for both `divide` and `pow`.

**Cross-context consistency.** The same `/` expression in a `divide{}` flow node and
inside a WASM `func` block must fail the same way. Under Option A, both produce IEEE 754
values, identical behavior. Under Option B, a flow node panics and a `func` block traps:
identical source code with two different failure modes depending on where it runs.
Closing that gap requires building error-ABI plumbing (return tuples or global error
flags from host functions) that Option A does not need. The infrastructure cost of
closing that gap is treated in [§Error Handling](#error-handling).

**Failure visibility.** Under Option A, problems surface as `NaN`/`Inf` in the UI and
logs. They are loud, traceable, and easy to point at when debugging a misbehaving
control loop. Under Option B, problems are either silent (integer truncation gives a
number that looks fine until it doesn't) or catastrophic (a scheduler panic crashes the
program with no specific signal about which expression caused it). Neither serves an
operator trying to understand what their program did.

**Output across input range.** Option A produces a meaningful, deterministic result for
every input combination, including `/0`. Option B has discontinuities: works fine for
non-zero denominators, panics on zero, and silently truncates for integer division. A
reliable operator has no cliffs.

### Safety

`+Inf`, `-Inf`, and `NaN` are safe in Arc's control model:

- **Predictable truthiness**: `isSeriesTruthy` treats `NaN` as non-truthy (it is
  genuinely undefined, since `0/0` has no magnitude). `+Inf` and `-Inf` are truthy,
  consistent with Python and JavaScript. Programs that need to guard against
  divide-by-zero triggering control should check the denominator explicitly.
- **Visible**: `∞`/`Infinity`, `-∞`/`-Infinity`, and `NaN` already render in the UI.
  They are not silent.

Under Option B, either the system outputs incorrect data (sentinel value of 0), silently
drops the output (no signal to the user), or behaves differently depending on whether
the division is in a flow node or a `func` block. All three are worse for safety than a
visible IEEE 754 value with well-defined truthiness semantics.

### Operator Tooltips

Regardless of which option is chosen, `/` and `**` need a tooltip in the editor: their
behavior is not self-evident from the operator glyph. The question is what mental model
the tooltip has to teach.

**Option A tooltip** (one sentence):

> `/` and `**` always operate in the continuous domain. Inputs are cast to `f64`; the
> result is `f64`. Divide-by-zero produces `+Inf`, `-Inf`, or `NaN` per IEEE 754.

**Option B tooltip** (multi-clause):

> `/` truncates toward zero when both inputs are integers. For continuous division, cast
> inputs explicitly: `f64(a) / f64(b)`. Divide-by-zero behavior differs between flow
> nodes (panic) and `func` blocks (trap on integer divide, IEEE 754 on float). The same
> caveats apply to `**` for negative integer exponents (see
> [§What Option B actually requires](#what-option-b-actually-requires)).

Option A's tooltip is one sentence because the rule is uniform. Option B's tooltip has
to teach truncation, the explicit-cast workaround, and a cross-context divergence on
`/0`. Easier-to-teach is the better default. This is also a clean rebuttal to "Option B
is simpler": its surface model is simpler; its teaching surface is not.

The tooltip is also the mitigation for the same-type rule break. Option A makes `/` and
`**` exceptions to the rule that `+`, `-`, `*`, `%` enforce, and the tooltip is what
closes the gap. LabVIEW's defense for its own coercion is that it is graphically
visible: a red dot appears on the wire where coercion happens. In practice the dot
reports _that_ coercion happened, not _what the resolved type is_; recovering the actual
type requires the Probe Tool on a specific wire. Arc already surfaces resolved variable
types in editor tooltips. An operator tooltip on `/` and `**` describing the `f64`
promotion combines with that existing surface to give the user full operator-level type
information at hover, with strictly less friction than a red dot plus a probe action.
The same-type rule break is real; it is also cheap to mitigate.

---

## Trade Summary

| Criterion                 | Option A | Option B |
| ------------------------- | -------- | -------- |
| Static + strong typing    | ✓        | ✓        |
| Type consistency          | ✓        |          |
| Same-type rule uniformity |          | ✓        |
| Precision                 | ✓        |          |
| Overflow consistency      | ✓        |          |
| Divide by zero            | ✓        |          |
| Error handling            | ✓        |          |
| Language identity         | ✓        |          |
| Reliability               | ✓        |          |
| Cross-context consistency | ✓        |          |
| Safety                    | ✓        |          |
| Tooltip                   | ✓        |          |

## Recommendation

Option A is the recommended option. It wins on every criterion in this study except two.
On timestamp precision, Option B preserves exact nanosecond values; under Option A, the
loss is ~2-4 nanoseconds on year-long time deltas. On same-type rule uniformity, Option
B keeps `/` and `**` under the same surface rule as `+`, `-`, `*`, `%`; under Option A,
the cost is one operator tooltip teaching a single uniform rule. Both losses are
bounded. Option B's losses (silent wrong answers from `1 / 2 = 0`, multi-clause tooltip,
cross-context divergence on `/0`) are not.

The intuition behind Option B is reasonable. "The output type should match the input
type" is a clean mental model, and integer division is familiar to anyone who has
written C. But that intuition breaks down when applied to operations that are
mathematically continuous: `1 / 2` is `0.5`, not `0`. Every language that our users
actually come from (Python, LabVIEW, MATLAB, numpy) resolved this the same way. Option B
is not more type-safe than Option A. The `/` operator under Option A is both statically
typed (single, concrete, compile-time return type of `f64`) and strongly typed (the
compiler rejects assignments to non-`f64` destinations without an explicit cast). The
argument that matching input types is "stricter" confuses a UX preference with a
type-system property. Option B also introduces more uncertainty, unreliability, and
therefore, increased risk.

If integer division is needed in the future, `math.floor_division()` (toward negative
infinity) and `math.quotient()` (truncation toward zero) can be added without changing
existing behavior.
