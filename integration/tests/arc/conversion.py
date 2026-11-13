#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc import ArcCase

# Identity casts (e.g., i32→i32) emit as no-ops in cast.go and are omitted. Among
# integer sources, i32 / u32 cover the signed-int and unsigned-int extend / convert
# / wrap opcode families at a nominal value; the i8 / i16 / i64 / u8 / u16 / u64
# nominal cases would only re-confirm "value 42 stays 42". The i64 source is kept
# strictly to exercise narrowing wrap at 2^31 (the only behavior that differs from
# i32). For float sources, {i8, i16, i32} share the same i32.trunc_* opcode and
# differ only in mask/sign-extend epilogue, so i8 stands in for the group; i64 is
# kept because it uses the distinct i64.trunc_* opcode. Same shape for unsigned.
# Negative-value variants are scoped to signed targets to avoid float→unsigned
# trunc traps.
ARC_CONVERSION_SOURCE = """
func cast_i32() {x i32} {
    out_i32_to_i8  = i8(x)
    out_i32_to_i16 = i16(x)
    out_i32_to_i64 = i64(x)
    out_i32_to_u8  = u8(x)
    out_i32_to_u16 = u16(x)
    out_i32_to_u32 = u32(x)
    out_i32_to_u64 = u64(x)
    out_i32_to_f32 = f32(x)
    out_i32_to_f64 = f64(x)
    out_i32_to_str = str(x)
}
in_i32 -> cast_i32()

func cast_u32() {x u32} {
    out_u32_to_i8  = i8(x)
    out_u32_to_i16 = i16(x)
    out_u32_to_i32 = i32(x)
    out_u32_to_i64 = i64(x)
    out_u32_to_u8  = u8(x)
    out_u32_to_u16 = u16(x)
    out_u32_to_u64 = u64(x)
    out_u32_to_f32 = f32(x)
    out_u32_to_f64 = f64(x)
    out_u32_to_str = str(x)
}
in_u32 -> cast_u32()

// i64 source: only narrowing-wrap targets at 2^31 (i32 sign-flip, u32 in-range
// bitcast, str decimal form).
func cast_i64() {x i64} {
    out_i64_to_i32 = i32(x)
    out_i64_to_u32 = u32(x)
    out_i64_to_str = str(x)
}
in_i64 -> cast_i64()

func cast_f32() {x f32} {
    out_f32_to_i8  = i8(x)
    out_f32_to_i64 = i64(x)
    out_f32_to_u8  = u8(x)
    out_f32_to_u64 = u64(x)
    out_f32_to_f64 = f64(x)
    out_f32_to_str = str(x)
}
in_f32 -> cast_f32()

// in_f32_neg avoids float→unsigned trunc traps by skipping unsigned int targets.
func cast_f32_neg() {x f32} {
    out_f32_neg_to_i8  = i8(x)
    out_f32_neg_to_i64 = i64(x)
    out_f32_neg_to_f64 = f64(x)
    out_f32_neg_to_str = str(x)
}
in_f32_neg -> cast_f32_neg()

func cast_f64() {x f64} {
    out_f64_to_i8  = i8(x)
    out_f64_to_i64 = i64(x)
    out_f64_to_u8  = u8(x)
    out_f64_to_u64 = u64(x)
    out_f64_to_f32 = f32(x)
    out_f64_to_str = str(x)
}
in_f64 -> cast_f64()

func cast_f64_neg() {x f64} {
    out_f64_neg_to_i8  = i8(x)
    out_f64_neg_to_i64 = i64(x)
    out_f64_neg_to_f32 = f32(x)
    out_f64_neg_to_str = str(x)
}
in_f64_neg -> cast_f64_neg()

// ───────── Narrow integer types (u8, u16, i8, i16) ─────────
// A narrow type holds only values in its range. Each result widens to i64 in Arc,
// because a narrow output channel cuts the value to its width when it stores it.
func wrap_cast() {x i32} {
    out_wrap_cast_u8  = i64(u8(x))
    out_wrap_cast_i8  = i64(i8(x))
    out_wrap_cast_u16 = i64(u16(x))
    out_wrap_cast_i16 = i64(i16(x))
}
in_wrap_cast -> wrap_cast()

func wrap_cast_i64() {x i64} {
    out_wrap_cast_i64_u8  = i64(u8(x))
    out_wrap_cast_i64_i8  = i64(i8(x))
    out_wrap_cast_i64_u16 = i64(u16(x))
    out_wrap_cast_i64_i16 = i64(i16(x))
}
in_wrap_cast_i64 -> wrap_cast_i64()

func wrap_cast_u16() {x u16} {
    out_wrap_cast_u16_u8  = i64(u8(x))
    out_wrap_cast_u16_i8  = i64(i8(x))
    out_wrap_cast_u16_i16 = i64(i16(x))
}
in_wrap_cast_u16 -> wrap_cast_u16()

func wrap_cast_u8() {x u8} {
    out_wrap_cast_u8_i8  = i64(i8(x))
    out_wrap_cast_u8_u16 = i64(u16(x))
    out_wrap_cast_u8_i16 = i64(i16(x))
}
in_wrap_cast_u8 -> wrap_cast_u8()

func wrap_cast_i8() {x i8} {
    out_wrap_cast_i8_u8  = i64(u8(x))
    out_wrap_cast_i8_u16 = i64(u16(x))
    out_wrap_cast_i8_i16 = i64(i16(x))
}
in_wrap_cast_i8 -> wrap_cast_i8()

func sat_cast() {x f64} {
    out_sat_cast_u8  = i64(u8(x))
    out_sat_cast_i8  = i64(i8(x))
    out_sat_cast_u16 = i64(u16(x))
    out_sat_cast_i16 = i64(i16(x))
}
in_sat_cast -> sat_cast()

func sat_cast_f32() {x f32} {
    out_sat_cast_f32_u8  = i64(u8(x))
    out_sat_cast_f32_i8  = i64(i8(x))
    out_sat_cast_f32_u16 = i64(u16(x))
    out_sat_cast_f32_i16 = i64(i16(x))
}
in_sat_cast_f32 -> sat_cast_f32()

func wrap_u8() {x u8} {
    out_wrap_u8_sub = i64(x - 1)
    out_wrap_u8_add = i64(x + 255)
    out_wrap_u8_mul = i64(x * 100)
    out_wrap_u8_neg = i64(-x)
    y u8 := x
    y += 255
    out_wrap_u8_cmp = i64(y)
}
in_wrap_u8 -> wrap_u8()

func wrap_u16() {x u16} {
    out_wrap_u16_sub = i64(x - 1)
    out_wrap_u16_add = i64(x + 65535)
    out_wrap_u16_mul = i64(x * 1000)
    out_wrap_u16_neg = i64(-x)
    y u16 := x
    y += 65535
    out_wrap_u16_cmp = i64(y)
}
in_wrap_u16 -> wrap_u16()

func wrap_i8() {x i8} {
    d i8 := -1
    out_wrap_i8_sub = i64(x - 127)
    out_wrap_i8_add = i64(x + 127)
    out_wrap_i8_mul = i64(x * 100)
    out_wrap_i8_neg = i64(-x)
    out_wrap_i8_div = i64(x / d)
    y i8 := x
    y -= 127
    out_wrap_i8_cmp = i64(y)
}
in_wrap_i8 -> wrap_i8()

func wrap_i16() {x i16} {
    d i16 := -1
    out_wrap_i16_sub = i64(x - 32767)
    out_wrap_i16_add = i64(x + 32767)
    out_wrap_i16_mul = i64(x * 1000)
    out_wrap_i16_neg = i64(-x)
    out_wrap_i16_div = i64(x / d)
    y i16 := x
    y -= 32767
    out_wrap_i16_cmp = i64(y)
}
in_wrap_i16 -> wrap_i16()

// Each result is taken from the expression and again from a stateful variable, which
// stores the value at its declared width.
func spec_cast_neg() {x i32} {
    s i8 $= 0
    s = i8(x)
    direct i64 := 0
    if i8(x) < 0 {
        direct = 1
    }
    stored i64 := 0
    if s < 0 {
        stored = 1
    }
    out_spec_cast_neg = direct
    out_spec_cast_neg_stored = stored
    out_spec_cast_neg_value = i64(i8(x))
    out_spec_cast_neg_value_stored = i64(s)
}
in_spec_cast_neg -> spec_cast_neg()

func spec_half() {x u8} {
    s u8 $= 0
    s = x + 1
    out_spec_half = i64((x + 1) / 2)
    out_spec_half_stored = i64(s / 2)
}
in_spec_half -> spec_half()

func spec_neg() {x i8} {
    s i8 $= 0
    s = -x
    direct i64 := 0
    if -x > 0 {
        direct = 1
    }
    stored i64 := 0
    if s > 0 {
        stored = 1
    }
    out_spec_neg = direct
    out_spec_neg_stored = stored
    out_spec_neg_value = i64(-x)
    out_spec_neg_value_stored = i64(s)
}
in_spec_neg -> spec_neg()

func spec_cast_wide() {x i64} {
    s i8 $= 0
    s = i8(x)
    out_spec_cast_wide = i64(i8(x))
    out_spec_cast_wide_stored = i64(s)
}
in_spec_cast_wide -> spec_cast_wide()

func spec_cast_neg_i16() {x i32} {
    s i16 $= 0
    s = i16(x)
    direct i64 := 0
    if i16(x) < 0 {
        direct = 1
    }
    stored i64 := 0
    if s < 0 {
        stored = 1
    }
    out_spec_cast_neg_i16 = direct
    out_spec_cast_neg_i16_stored = stored
    out_spec_cast_neg_i16_value = i64(i16(x))
    out_spec_cast_neg_i16_value_stored = i64(s)
}
in_spec_cast_neg_i16 -> spec_cast_neg_i16()

func spec_half_u16() {x u16} {
    s u16 $= 0
    s = x + 1
    out_spec_half_u16 = i64((x + 1) / 2)
    out_spec_half_u16_stored = i64(s / 2)
}
in_spec_half_u16 -> spec_half_u16()

func spec_neg_i16() {x i16} {
    s i16 $= 0
    s = -x
    direct i64 := 0
    if -x > 0 {
        direct = 1
    }
    stored i64 := 0
    if s > 0 {
        stored = 1
    }
    out_spec_neg_i16 = direct
    out_spec_neg_i16_stored = stored
    out_spec_neg_i16_value = i64(-x)
    out_spec_neg_i16_value_stored = i64(s)
}
in_spec_neg_i16 -> spec_neg_i16()

func spec_cast_wide_i16() {x i64} {
    s i16 $= 0
    s = i16(x)
    out_spec_cast_wide_i16 = i64(i16(x))
    out_spec_cast_wide_i16_stored = i64(s)
}
in_spec_cast_wide_i16 -> spec_cast_wide_i16()

// ───────── Flow-context casts (inline expression in flow) ─────────
// The cast.go path that compiles in-flow expressions differs from the WASM
// function-body path above: the value is read straight from channel state at
// flow-fire time rather than passed via a function parameter on the stack.
//
// String form: const cast inside concat exercises str-from-literal; channel
// cast inside concat exercises one case per from_<type> host function. The
// concat-wrapped form is a strict superset of the bare str(channel) -> chan
// form for the cast-emit path, so the bare form is omitted.
flow_trigger -> "value=" + str(42) + " items" -> flow_concat_const_int
flow_trigger -> "value=" + str(3.5) + " psi" -> flow_concat_const_float
flow_trigger -> "v=" + str(in_i32) + "!" -> flow_concat_i32
flow_trigger -> "v=" + str(in_u32) + "!" -> flow_concat_u32
flow_trigger -> "v=" + str(in_i64) + "!" -> flow_concat_i64
flow_trigger -> "v=" + str(in_u64) + "!" -> flow_concat_u64
flow_trigger -> "v=" + str(in_f32) + "!" -> flow_concat_f32
flow_trigger -> "v=" + str(in_f64) + "!" -> flow_concat_f64

// Special f64 values: division by zero yields +Inf / -Inf / NaN. Exercises
// strconv.FormatFloat's non-finite branch via from_f64.
flow_trigger -> str(f64(1)/f64(0)) -> flow_str_pos_inf
flow_trigger -> str(f64(-1)/f64(0)) -> flow_str_neg_inf
flow_trigger -> str(f64(0)/f64(0)) -> flow_str_nan

// Numeric form: one representative cast per opcode family (extend, wrap,
// convert, trunc, promote/demote) plus one cast composed inside a binary op.
flow_trigger -> i64(in_i32) -> flow_num_i32_to_i64
flow_trigger -> i32(in_i64) -> flow_num_i64_to_i32
flow_trigger -> f64(in_i32) -> flow_num_i32_to_f64
flow_trigger -> i32(in_f64) -> flow_num_f64_to_i32
flow_trigger -> f32(in_f64) -> flow_num_f64_to_f32
flow_trigger -> i32(in_f64) + 100 -> flow_num_arith
"""

INPUT_CHANNELS: list[tuple[str, sy.DataType]] = [
    ("in_i32", sy.DataType.INT32),
    ("in_u32", sy.DataType.UINT32),
    ("in_i64", sy.DataType.INT64),
    ("in_u64", sy.DataType.UINT64),
    ("in_f32", sy.DataType.FLOAT32),
    ("in_f32_neg", sy.DataType.FLOAT32),
    ("in_f64", sy.DataType.FLOAT64),
    ("in_f64_neg", sy.DataType.FLOAT64),
    ("flow_trigger", sy.DataType.INT64),
    ("in_wrap_cast", sy.DataType.INT32),
    ("in_wrap_cast_i64", sy.DataType.INT64),
    ("in_wrap_cast_u16", sy.DataType.UINT16),
    ("in_wrap_cast_u8", sy.DataType.UINT8),
    ("in_wrap_cast_i8", sy.DataType.INT8),
    ("in_sat_cast", sy.DataType.FLOAT64),
    ("in_sat_cast_f32", sy.DataType.FLOAT32),
    ("in_wrap_u8", sy.DataType.UINT8),
    ("in_wrap_u16", sy.DataType.UINT16),
    ("in_wrap_i8", sy.DataType.INT8),
    ("in_wrap_i16", sy.DataType.INT16),
    ("in_spec_cast_neg", sy.DataType.INT32),
    ("in_spec_half", sy.DataType.UINT8),
    ("in_spec_neg", sy.DataType.INT8),
    ("in_spec_cast_wide", sy.DataType.INT64),
    ("in_spec_cast_neg_i16", sy.DataType.INT32),
    ("in_spec_half_u16", sy.DataType.UINT16),
    ("in_spec_neg_i16", sy.DataType.INT16),
    ("in_spec_cast_wide_i16", sy.DataType.INT64),
]


@dataclass
class NarrowCase:
    """A value to write to an input channel and the i64 outputs to expect."""

    input: str
    value: int | float
    expected: dict[str, int]


NARROW_CASES: list[NarrowCase] = [
    # An integer cast wraps to the width of the target.
    NarrowCase(
        input="in_wrap_cast",
        value=-1,
        expected={
            "out_wrap_cast_u8": 255,
            "out_wrap_cast_i8": -1,
            "out_wrap_cast_u16": 65535,
            "out_wrap_cast_i16": -1,
        },
    ),
    NarrowCase(
        input="in_wrap_cast",
        value=40000,
        expected={
            "out_wrap_cast_u8": 64,
            "out_wrap_cast_i8": 64,
            "out_wrap_cast_u16": 40000,
            "out_wrap_cast_i16": -25536,
        },
    ),
    NarrowCase(
        input="in_wrap_cast",
        value=42,
        expected={
            "out_wrap_cast_u8": 42,
            "out_wrap_cast_i8": 42,
            "out_wrap_cast_u16": 42,
            "out_wrap_cast_i16": 42,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_i64",
        value=300,
        expected={
            "out_wrap_cast_i64_u8": 44,
            "out_wrap_cast_i64_i8": 44,
            "out_wrap_cast_i64_u16": 300,
            "out_wrap_cast_i64_i16": 300,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_i64",
        value=-40000,
        expected={
            "out_wrap_cast_i64_u8": 192,
            "out_wrap_cast_i64_i8": -64,
            "out_wrap_cast_i64_u16": 25536,
            "out_wrap_cast_i64_i16": 25536,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_i64",
        value=4294967303,
        expected={
            "out_wrap_cast_i64_u8": 7,
            "out_wrap_cast_i64_i8": 7,
            "out_wrap_cast_i64_u16": 7,
            "out_wrap_cast_i64_i16": 7,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_u16",
        value=300,
        expected={
            "out_wrap_cast_u16_u8": 44,
            "out_wrap_cast_u16_i8": 44,
            "out_wrap_cast_u16_i16": 300,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_u16",
        value=65535,
        expected={
            "out_wrap_cast_u16_u8": 255,
            "out_wrap_cast_u16_i8": -1,
            "out_wrap_cast_u16_i16": -1,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_u8",
        value=200,
        expected={
            "out_wrap_cast_u8_i8": -56,
            "out_wrap_cast_u8_u16": 200,
            "out_wrap_cast_u8_i16": 200,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_u8",
        value=100,
        expected={
            "out_wrap_cast_u8_i8": 100,
            "out_wrap_cast_u8_u16": 100,
            "out_wrap_cast_u8_i16": 100,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_i8",
        value=-1,
        expected={
            "out_wrap_cast_i8_u8": 255,
            "out_wrap_cast_i8_u16": 65535,
            "out_wrap_cast_i8_i16": -1,
        },
    ),
    NarrowCase(
        input="in_wrap_cast_i8",
        value=5,
        expected={
            "out_wrap_cast_i8_u8": 5,
            "out_wrap_cast_i8_u16": 5,
            "out_wrap_cast_i8_i16": 5,
        },
    ),
    # A float cast saturates at the bounds of the target.
    NarrowCase(
        input="in_sat_cast",
        value=70000.0,
        expected={
            "out_sat_cast_u8": 255,
            "out_sat_cast_i8": 127,
            "out_sat_cast_u16": 65535,
            "out_sat_cast_i16": 32767,
        },
    ),
    NarrowCase(
        input="in_sat_cast",
        value=-70000.0,
        expected={
            "out_sat_cast_u8": 0,
            "out_sat_cast_i8": -128,
            "out_sat_cast_u16": 0,
            "out_sat_cast_i16": -32768,
        },
    ),
    NarrowCase(
        input="in_sat_cast",
        value=42.9,
        expected={
            "out_sat_cast_u8": 42,
            "out_sat_cast_i8": 42,
            "out_sat_cast_u16": 42,
            "out_sat_cast_i16": 42,
        },
    ),
    NarrowCase(
        input="in_sat_cast_f32",
        value=70000.0,
        expected={
            "out_sat_cast_f32_u8": 255,
            "out_sat_cast_f32_i8": 127,
            "out_sat_cast_f32_u16": 65535,
            "out_sat_cast_f32_i16": 32767,
        },
    ),
    NarrowCase(
        input="in_sat_cast_f32",
        value=-70000.0,
        expected={
            "out_sat_cast_f32_u8": 0,
            "out_sat_cast_f32_i8": -128,
            "out_sat_cast_f32_u16": 0,
            "out_sat_cast_f32_i16": -32768,
        },
    ),
    NarrowCase(
        input="in_sat_cast_f32",
        value=-42.5,
        expected={
            "out_sat_cast_f32_u8": 0,
            "out_sat_cast_f32_i8": -42,
            "out_sat_cast_f32_u16": 0,
            "out_sat_cast_f32_i16": -42,
        },
    ),
    # Arithmetic wraps to the width of the type.
    NarrowCase(
        input="in_wrap_u8",
        value=0,
        expected={
            "out_wrap_u8_sub": 255,
            "out_wrap_u8_add": 255,
            "out_wrap_u8_mul": 0,
            "out_wrap_u8_neg": 0,
            "out_wrap_u8_cmp": 255,
        },
    ),
    NarrowCase(
        input="in_wrap_u8",
        value=17,
        expected={
            "out_wrap_u8_sub": 16,
            "out_wrap_u8_add": 16,
            "out_wrap_u8_mul": 164,
            "out_wrap_u8_neg": 239,
            "out_wrap_u8_cmp": 16,
        },
    ),
    NarrowCase(
        input="in_wrap_u16",
        value=0,
        expected={
            "out_wrap_u16_sub": 65535,
            "out_wrap_u16_add": 65535,
            "out_wrap_u16_mul": 0,
            "out_wrap_u16_neg": 0,
            "out_wrap_u16_cmp": 65535,
        },
    ),
    NarrowCase(
        input="in_wrap_u16",
        value=100,
        expected={
            "out_wrap_u16_sub": 99,
            "out_wrap_u16_add": 99,
            "out_wrap_u16_mul": 34464,
            "out_wrap_u16_neg": 65436,
            "out_wrap_u16_cmp": 99,
        },
    ),
    NarrowCase(
        input="in_wrap_i8",
        value=1,
        expected={
            "out_wrap_i8_sub": -126,
            "out_wrap_i8_add": -128,
            "out_wrap_i8_mul": 100,
            "out_wrap_i8_neg": -1,
            "out_wrap_i8_div": -1,
            "out_wrap_i8_cmp": -126,
        },
    ),
    NarrowCase(
        input="in_wrap_i8",
        value=-128,
        expected={
            "out_wrap_i8_sub": 1,
            "out_wrap_i8_add": -1,
            "out_wrap_i8_mul": 0,
            "out_wrap_i8_neg": -128,
            "out_wrap_i8_div": -128,
            "out_wrap_i8_cmp": 1,
        },
    ),
    NarrowCase(
        input="in_wrap_i8",
        value=2,
        expected={
            "out_wrap_i8_sub": -125,
            "out_wrap_i8_add": -127,
            "out_wrap_i8_mul": -56,
            "out_wrap_i8_neg": -2,
            "out_wrap_i8_div": -2,
            "out_wrap_i8_cmp": -125,
        },
    ),
    NarrowCase(
        input="in_wrap_i16",
        value=1,
        expected={
            "out_wrap_i16_sub": -32766,
            "out_wrap_i16_add": -32768,
            "out_wrap_i16_mul": 1000,
            "out_wrap_i16_neg": -1,
            "out_wrap_i16_div": -1,
            "out_wrap_i16_cmp": -32766,
        },
    ),
    NarrowCase(
        input="in_wrap_i16",
        value=-32768,
        expected={
            "out_wrap_i16_sub": 1,
            "out_wrap_i16_add": -1,
            "out_wrap_i16_mul": 0,
            "out_wrap_i16_neg": -32768,
            "out_wrap_i16_div": -32768,
            "out_wrap_i16_cmp": 1,
        },
    ),
    NarrowCase(
        input="in_wrap_i16",
        value=40,
        expected={
            "out_wrap_i16_sub": -32727,
            "out_wrap_i16_add": -32729,
            "out_wrap_i16_mul": -25536,
            "out_wrap_i16_neg": -40,
            "out_wrap_i16_div": -40,
            "out_wrap_i16_cmp": -32727,
        },
    ),
]

# A narrow value from an expression and from a stateful variable. Each output has a
# copy with a "_stored" suffix that must match it.
STORED_CASES: list[NarrowCase] = [
    NarrowCase(
        input="in_spec_cast_neg",
        value=200,
        expected={"out_spec_cast_neg": 1, "out_spec_cast_neg_value": -56},
    ),
    NarrowCase(
        input="in_spec_cast_neg",
        value=100,
        expected={"out_spec_cast_neg": 0, "out_spec_cast_neg_value": 100},
    ),
    NarrowCase(input="in_spec_half", value=9, expected={"out_spec_half": 5}),
    NarrowCase(input="in_spec_half", value=255, expected={"out_spec_half": 0}),
    NarrowCase(
        input="in_spec_neg",
        value=-5,
        expected={"out_spec_neg": 1, "out_spec_neg_value": 5},
    ),
    NarrowCase(
        input="in_spec_neg",
        value=-128,
        expected={"out_spec_neg": 0, "out_spec_neg_value": -128},
    ),
    NarrowCase(input="in_spec_cast_wide", value=5, expected={"out_spec_cast_wide": 5}),
    NarrowCase(
        input="in_spec_cast_wide", value=511, expected={"out_spec_cast_wide": -1}
    ),
    NarrowCase(
        input="in_spec_cast_neg_i16",
        value=40000,
        expected={"out_spec_cast_neg_i16": 1, "out_spec_cast_neg_i16_value": -25536},
    ),
    NarrowCase(
        input="in_spec_cast_neg_i16",
        value=100,
        expected={"out_spec_cast_neg_i16": 0, "out_spec_cast_neg_i16_value": 100},
    ),
    NarrowCase(input="in_spec_half_u16", value=9, expected={"out_spec_half_u16": 5}),
    NarrowCase(
        input="in_spec_half_u16", value=65535, expected={"out_spec_half_u16": 0}
    ),
    NarrowCase(
        input="in_spec_neg_i16",
        value=-5,
        expected={"out_spec_neg_i16": 1, "out_spec_neg_i16_value": 5},
    ),
    NarrowCase(
        input="in_spec_neg_i16",
        value=-32768,
        expected={"out_spec_neg_i16": 0, "out_spec_neg_i16_value": -32768},
    ),
    NarrowCase(
        input="in_spec_cast_wide_i16", value=5, expected={"out_spec_cast_wide_i16": 5}
    ),
    NarrowCase(
        input="in_spec_cast_wide_i16",
        value=131071,
        expected={"out_spec_cast_wide_i16": -1},
    ),
]

NARROW_CASES += [
    NarrowCase(
        input=case.input,
        value=case.value,
        expected=case.expected
        | {f"{name}_stored": v for name, v in case.expected.items()},
    )
    for case in STORED_CASES
]

OUTPUT_CHANNELS: list[tuple[str, sy.DataType]] = [
    # i32 source
    ("out_i32_to_i8", sy.DataType.INT8),
    ("out_i32_to_i16", sy.DataType.INT16),
    ("out_i32_to_i64", sy.DataType.INT64),
    ("out_i32_to_u8", sy.DataType.UINT8),
    ("out_i32_to_u16", sy.DataType.UINT16),
    ("out_i32_to_u32", sy.DataType.UINT32),
    ("out_i32_to_u64", sy.DataType.UINT64),
    ("out_i32_to_f32", sy.DataType.FLOAT32),
    ("out_i32_to_f64", sy.DataType.FLOAT64),
    ("out_i32_to_str", sy.DataType.STRING),
    # u32 source
    ("out_u32_to_i8", sy.DataType.INT8),
    ("out_u32_to_i16", sy.DataType.INT16),
    ("out_u32_to_i32", sy.DataType.INT32),
    ("out_u32_to_i64", sy.DataType.INT64),
    ("out_u32_to_u8", sy.DataType.UINT8),
    ("out_u32_to_u16", sy.DataType.UINT16),
    ("out_u32_to_u64", sy.DataType.UINT64),
    ("out_u32_to_f32", sy.DataType.FLOAT32),
    ("out_u32_to_f64", sy.DataType.FLOAT64),
    ("out_u32_to_str", sy.DataType.STRING),
    # i64 source (overflow only)
    ("out_i64_to_i32", sy.DataType.INT32),
    ("out_i64_to_u32", sy.DataType.UINT32),
    ("out_i64_to_str", sy.DataType.STRING),
    # f32 source
    ("out_f32_to_i8", sy.DataType.INT8),
    ("out_f32_to_i64", sy.DataType.INT64),
    ("out_f32_to_u8", sy.DataType.UINT8),
    ("out_f32_to_u64", sy.DataType.UINT64),
    ("out_f32_to_f64", sy.DataType.FLOAT64),
    ("out_f32_to_str", sy.DataType.STRING),
    # f32 neg source
    ("out_f32_neg_to_i8", sy.DataType.INT8),
    ("out_f32_neg_to_i64", sy.DataType.INT64),
    ("out_f32_neg_to_f64", sy.DataType.FLOAT64),
    ("out_f32_neg_to_str", sy.DataType.STRING),
    # f64 source
    ("out_f64_to_i8", sy.DataType.INT8),
    ("out_f64_to_i64", sy.DataType.INT64),
    ("out_f64_to_u8", sy.DataType.UINT8),
    ("out_f64_to_u64", sy.DataType.UINT64),
    ("out_f64_to_f32", sy.DataType.FLOAT32),
    ("out_f64_to_str", sy.DataType.STRING),
    # f64 neg source
    ("out_f64_neg_to_i8", sy.DataType.INT8),
    ("out_f64_neg_to_i64", sy.DataType.INT64),
    ("out_f64_neg_to_f32", sy.DataType.FLOAT32),
    ("out_f64_neg_to_str", sy.DataType.STRING),
    # Flow-context str(): const cast inside concat
    ("flow_concat_const_int", sy.DataType.STRING),
    ("flow_concat_const_float", sy.DataType.STRING),
    # Flow-context str(): channel cast inside concat
    ("flow_concat_i32", sy.DataType.STRING),
    ("flow_concat_u32", sy.DataType.STRING),
    ("flow_concat_i64", sy.DataType.STRING),
    ("flow_concat_u64", sy.DataType.STRING),
    ("flow_concat_f32", sy.DataType.STRING),
    ("flow_concat_f64", sy.DataType.STRING),
    # Flow-context str(): special float values
    ("flow_str_pos_inf", sy.DataType.STRING),
    ("flow_str_neg_inf", sy.DataType.STRING),
    ("flow_str_nan", sy.DataType.STRING),
    # Flow-context numeric casts
    ("flow_num_i32_to_i64", sy.DataType.INT64),
    ("flow_num_i64_to_i32", sy.DataType.INT32),
    ("flow_num_i32_to_f64", sy.DataType.FLOAT64),
    ("flow_num_f64_to_i32", sy.DataType.INT32),
    ("flow_num_f64_to_f32", sy.DataType.FLOAT32),
    ("flow_num_arith", sy.DataType.INT32),
    # Narrow integer types, widened to i64 in Arc
    *(
        (name, sy.DataType.INT64)
        for name in dict.fromkeys(
            name for case in NARROW_CASES for name in case.expected
        )
    ),
]

ALL_CHANNELS = [name for name, _ in INPUT_CHANNELS] + [
    name for name, _ in OUTPUT_CHANNELS
]


class Conversion(ArcCase):
    """End-to-end coverage of numeric typecasts plus numeric→str.

    Two compilation paths are covered.

    - WASM-body casts: each source type runs a
    single Arc function that casts its input to every target via direct channel
    writes. Identity casts are omitted (they emit as no-ops). i32 / u32 act as
    smoke tests for the signed and unsigned int opcode families at a nominal
    value; i64 covers narrowing wrap at 2^31. For float sources, only one
    representative target per trunc-opcode group is asserted ({i8, i64} signed;
    {u8, u64} unsigned), with negative-value variants scoped to signed targets
    only to avoid float→unsigned trunc traps.

    - Flow-context casts: cast applied as an inline expression in flow position
    rather than inside a function body. String form covers const cast and
    channel cast both wrapped in concat (the wrapped form is a strict superset
    of the bare form's cast-emit path). Numeric form covers one representative
    pair per opcode family (extend, wrap, convert, trunc, promote/demote) plus
    one cast composed inside a binary op.

    - Narrow integer types: an integer cast into u8, u16, i8, or i16 wraps, a float
    cast saturates, and arithmetic on these types wraps.
    """

    arc_source = ARC_CONVERSION_SOURCE
    arc_name_prefix = "ArcConversion"
    start_cmd_channel = "start_conversion_cmd"
    subscribe_channels = ALL_CHANNELS

    def setup(self) -> None:
        self.set_manual_timeout(300)
        for name, dtype in INPUT_CHANNELS:
            create_virtual_channel(self.client, name, data_type=dtype)
        for name, dtype in OUTPUT_CHANNELS:
            create_virtual_channel(self.client, name, data_type=dtype)
        super().setup()

    def verify_sequence_execution(self) -> None:
        self._test_32()
        self._test_i64_overflow()
        self._test_f32()
        self._test_f64()
        self._test_flow_context()
        self._test_narrow()

    def _test_32(self) -> None:
        self.log("=== i32 = 42, u32 = 42 ===")
        self.writer.write({"in_i32": 42, "in_u32": 42})
        self.wait_for_eq("out_i32_to_i8", 42)
        self.wait_for_eq("out_i32_to_i16", 42)
        self.wait_for_eq("out_i32_to_i64", 42)
        self.wait_for_eq("out_i32_to_u8", 42)
        self.wait_for_eq("out_i32_to_u16", 42)
        self.wait_for_eq("out_i32_to_u32", 42)
        self.wait_for_eq("out_i32_to_u64", 42)
        self.wait_for_near("out_i32_to_f32", 42.0, tolerance=1e-5)
        self.wait_for_near("out_i32_to_f64", 42.0, tolerance=1e-9)
        self.wait_for_eq("out_i32_to_str", "42")
        self.wait_for_eq("out_u32_to_i8", 42)
        self.wait_for_eq("out_u32_to_i16", 42)
        self.wait_for_eq("out_u32_to_i32", 42)
        self.wait_for_eq("out_u32_to_i64", 42)
        self.wait_for_eq("out_u32_to_u8", 42)
        self.wait_for_eq("out_u32_to_u16", 42)
        self.wait_for_eq("out_u32_to_u64", 42)
        self.wait_for_near("out_u32_to_f32", 42.0, tolerance=1e-5)
        self.wait_for_near("out_u32_to_f64", 42.0, tolerance=1e-9)
        self.wait_for_eq("out_u32_to_str", "42")

    def _test_i64_overflow(self) -> None:
        # i32(2^31) sign-flips to -2^31, u32(2^31) stays in range, str is decimal.
        self.log("=== i64 = 2^31 (narrowing wrap) ===")
        self.writer.write("in_i64", 2**31)
        self.wait_for_eq("out_i64_to_i32", -(2**31))
        self.wait_for_eq("out_i64_to_u32", 2**31)
        self.wait_for_eq("out_i64_to_str", "2147483648")

    def _test_f32(self) -> None:
        self.log("=== f32 = 3.5, f32_neg = -7.0 ===")
        self.writer.write({"in_f32": 3.5, "in_f32_neg": -7.0})
        self.wait_for_eq("out_f32_to_i8", 3)
        self.wait_for_eq("out_f32_to_i64", 3)
        self.wait_for_eq("out_f32_to_u8", 3)
        self.wait_for_eq("out_f32_to_u64", 3)
        self.wait_for_near("out_f32_to_f64", 3.5, tolerance=1e-5)
        self.wait_for_eq("out_f32_to_str", "3.5")
        self.wait_for_eq("out_f32_neg_to_i8", -7)
        self.wait_for_eq("out_f32_neg_to_i64", -7)
        self.wait_for_near("out_f32_neg_to_f64", -7.0, tolerance=1e-5)
        self.wait_for_eq("out_f32_neg_to_str", "-7")

    def _test_f64(self) -> None:
        self.log("=== f64 = 3.5, f64_neg = -7.0 ===")
        self.writer.write({"in_f64": 3.5, "in_f64_neg": -7.0})
        self.wait_for_eq("out_f64_to_i8", 3)
        self.wait_for_eq("out_f64_to_i64", 3)
        self.wait_for_eq("out_f64_to_u8", 3)
        self.wait_for_eq("out_f64_to_u64", 3)
        self.wait_for_near("out_f64_to_f32", 3.5, tolerance=1e-5)
        self.wait_for_eq("out_f64_to_str", "3.5")
        self.wait_for_eq("out_f64_neg_to_i8", -7)
        self.wait_for_eq("out_f64_neg_to_i64", -7)
        self.wait_for_near("out_f64_neg_to_f32", -7.0, tolerance=1e-5)
        self.wait_for_eq("out_f64_neg_to_str", "-7")

    def _test_narrow(self) -> None:
        self.log("=== Narrow integer casts and arithmetic ===")
        for case in NARROW_CASES:
            self.log(f"Writing {case.value} to {case.input}")
            self.writer.write(case.input, case.value)
            for out_ch, result in case.expected.items():
                self.wait_for_eq(out_ch, result)

    def _test_flow_context(self) -> None:
        # Pre-write the source channels so they hold values when flow_trigger
        # fires the inline-expression flows.
        self.log("=== flow-context casts ===")
        self.writer.write(
            {
                "in_i32": 42,
                "in_u32": 42,
                "in_i64": 2**31,
                "in_u64": 42,
                "in_f32": 3.5,
                "in_f64": 3.5,
                "flow_trigger": 1,
            }
        )

        # str(): const cast inside concat
        self.wait_for_eq("flow_concat_const_int", "value=42 items")
        self.wait_for_eq("flow_concat_const_float", "value=3.5 psi")

        # str(): channel cast inside concat
        self.wait_for_eq("flow_concat_i32", "v=42!")
        self.wait_for_eq("flow_concat_u32", "v=42!")
        self.wait_for_eq("flow_concat_i64", "v=2147483648!")
        self.wait_for_eq("flow_concat_u64", "v=42!")
        self.wait_for_eq("flow_concat_f32", "v=3.5!")
        self.wait_for_eq("flow_concat_f64", "v=3.5!")

        # str(): special float values via division by zero. Go's
        # strconv.FormatFloat emits "+Inf" / "-Inf" / "NaN".
        self.wait_for_eq("flow_str_pos_inf", "+Inf")
        self.wait_for_eq("flow_str_neg_inf", "-Inf")
        self.wait_for_eq("flow_str_nan", "NaN")

        # Numeric: one representative per opcode family + one composed in a
        # binary op. i32(in_i64) at 2^31 sign-flips to -2^31.
        self.wait_for_eq("flow_num_i32_to_i64", 42)
        self.wait_for_eq("flow_num_i64_to_i32", -(2**31))
        self.wait_for_near("flow_num_i32_to_f64", 42.0, tolerance=1e-9)
        self.wait_for_eq("flow_num_f64_to_i32", 3)
        self.wait_for_near("flow_num_f64_to_f32", 3.5, tolerance=1e-5)
        self.wait_for_eq("flow_num_arith", 103)
