#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

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
func cast_i32(x i32) {
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
in_i32 -> cast_i32{}

func cast_u32(x u32) {
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
in_u32 -> cast_u32{}

// i64 source: only narrowing-wrap targets at 2^31 (i32 sign-flip, u32 in-range
// bitcast, str decimal form).
func cast_i64(x i64) {
    out_i64_to_i32 = i32(x)
    out_i64_to_u32 = u32(x)
    out_i64_to_str = str(x)
}
in_i64 -> cast_i64{}

func cast_f32(x f32) {
    out_f32_to_i8  = i8(x)
    out_f32_to_i64 = i64(x)
    out_f32_to_u8  = u8(x)
    out_f32_to_u64 = u64(x)
    out_f32_to_f64 = f64(x)
    out_f32_to_str = str(x)
}
in_f32 -> cast_f32{}

// in_f32_neg avoids float→unsigned trunc traps by skipping unsigned int targets.
func cast_f32_neg(x f32) {
    out_f32_neg_to_i8  = i8(x)
    out_f32_neg_to_i64 = i64(x)
    out_f32_neg_to_f64 = f64(x)
    out_f32_neg_to_str = str(x)
}
in_f32_neg -> cast_f32_neg{}

func cast_f64(x f64) {
    out_f64_to_i8  = i8(x)
    out_f64_to_i64 = i64(x)
    out_f64_to_u8  = u8(x)
    out_f64_to_u64 = u64(x)
    out_f64_to_f32 = f32(x)
    out_f64_to_str = str(x)
}
in_f64 -> cast_f64{}

func cast_f64_neg(x f64) {
    out_f64_neg_to_i8  = i8(x)
    out_f64_neg_to_i64 = i64(x)
    out_f64_neg_to_f32 = f32(x)
    out_f64_neg_to_str = str(x)
}
in_f64_neg -> cast_f64_neg{}

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
    """

    arc_source = ARC_CONVERSION_SOURCE
    arc_name_prefix = "ArcConversion"
    start_cmd_channel = "start_conversion_cmd"
    subscribe_channels = ALL_CHANNELS

    def setup(self) -> None:
        self.set_manual_timeout(300)
        for name, dtype in INPUT_CHANNELS:
            create_virtual_channel(self.client, name, dtype)
        for name, dtype in OUTPUT_CHANNELS:
            create_virtual_channel(self.client, name, dtype)
        super().setup()

    def verify_sequence_execution(self) -> None:
        self._test_32()
        self._test_i64_overflow()
        self._test_f32()
        self._test_f64()
        self._test_flow_context()

    def _test_32(self) -> None:
        self.log("=== i32 = 42, u32 = 42 ===")
        self.writer.write({"in_i32": 42, "in_u32": 42})
        self.wait_for_eq("out_i32_to_i8", 42, is_virtual=True)
        self.wait_for_eq("out_i32_to_i16", 42, is_virtual=True)
        self.wait_for_eq("out_i32_to_i64", 42, is_virtual=True)
        self.wait_for_eq("out_i32_to_u8", 42, is_virtual=True)
        self.wait_for_eq("out_i32_to_u16", 42, is_virtual=True)
        self.wait_for_eq("out_i32_to_u32", 42, is_virtual=True)
        self.wait_for_eq("out_i32_to_u64", 42, is_virtual=True)
        self.wait_for_near("out_i32_to_f32", 42.0, tolerance=1e-5, is_virtual=True)
        self.wait_for_near("out_i32_to_f64", 42.0, tolerance=1e-9, is_virtual=True)
        self.wait_for_eq("out_i32_to_str", "42", is_virtual=True)
        self.wait_for_eq("out_u32_to_i8", 42, is_virtual=True)
        self.wait_for_eq("out_u32_to_i16", 42, is_virtual=True)
        self.wait_for_eq("out_u32_to_i32", 42, is_virtual=True)
        self.wait_for_eq("out_u32_to_i64", 42, is_virtual=True)
        self.wait_for_eq("out_u32_to_u8", 42, is_virtual=True)
        self.wait_for_eq("out_u32_to_u16", 42, is_virtual=True)
        self.wait_for_eq("out_u32_to_u64", 42, is_virtual=True)
        self.wait_for_near("out_u32_to_f32", 42.0, tolerance=1e-5, is_virtual=True)
        self.wait_for_near("out_u32_to_f64", 42.0, tolerance=1e-9, is_virtual=True)
        self.wait_for_eq("out_u32_to_str", "42", is_virtual=True)

    def _test_i64_overflow(self) -> None:
        # i32(2^31) sign-flips to -2^31, u32(2^31) stays in range, str is decimal.
        self.log("=== i64 = 2^31 (narrowing wrap) ===")
        self.writer.write("in_i64", 2**31)
        self.wait_for_eq("out_i64_to_i32", -(2**31), is_virtual=True)
        self.wait_for_eq("out_i64_to_u32", 2**31, is_virtual=True)
        self.wait_for_eq("out_i64_to_str", "2147483648", is_virtual=True)

    def _test_f32(self) -> None:
        self.log("=== f32 = 3.5, f32_neg = -7.0 ===")
        self.writer.write({"in_f32": 3.5, "in_f32_neg": -7.0})
        self.wait_for_eq("out_f32_to_i8", 3, is_virtual=True)
        self.wait_for_eq("out_f32_to_i64", 3, is_virtual=True)
        self.wait_for_eq("out_f32_to_u8", 3, is_virtual=True)
        self.wait_for_eq("out_f32_to_u64", 3, is_virtual=True)
        self.wait_for_near("out_f32_to_f64", 3.5, tolerance=1e-5, is_virtual=True)
        self.wait_for_eq("out_f32_to_str", "3.5", is_virtual=True)
        self.wait_for_eq("out_f32_neg_to_i8", -7, is_virtual=True)
        self.wait_for_eq("out_f32_neg_to_i64", -7, is_virtual=True)
        self.wait_for_near("out_f32_neg_to_f64", -7.0, tolerance=1e-5, is_virtual=True)
        self.wait_for_eq("out_f32_neg_to_str", "-7", is_virtual=True)

    def _test_f64(self) -> None:
        self.log("=== f64 = 3.5, f64_neg = -7.0 ===")
        self.writer.write({"in_f64": 3.5, "in_f64_neg": -7.0})
        self.wait_for_eq("out_f64_to_i8", 3, is_virtual=True)
        self.wait_for_eq("out_f64_to_i64", 3, is_virtual=True)
        self.wait_for_eq("out_f64_to_u8", 3, is_virtual=True)
        self.wait_for_eq("out_f64_to_u64", 3, is_virtual=True)
        self.wait_for_near("out_f64_to_f32", 3.5, tolerance=1e-5, is_virtual=True)
        self.wait_for_eq("out_f64_to_str", "3.5", is_virtual=True)
        self.wait_for_eq("out_f64_neg_to_i8", -7, is_virtual=True)
        self.wait_for_eq("out_f64_neg_to_i64", -7, is_virtual=True)
        self.wait_for_near("out_f64_neg_to_f32", -7.0, tolerance=1e-5, is_virtual=True)
        self.wait_for_eq("out_f64_neg_to_str", "-7", is_virtual=True)

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
        self.wait_for_eq("flow_concat_const_int", "value=42 items", is_virtual=True)
        self.wait_for_eq("flow_concat_const_float", "value=3.5 psi", is_virtual=True)

        # str(): channel cast inside concat
        self.wait_for_eq("flow_concat_i32", "v=42!", is_virtual=True)
        self.wait_for_eq("flow_concat_u32", "v=42!", is_virtual=True)
        self.wait_for_eq("flow_concat_i64", "v=2147483648!", is_virtual=True)
        self.wait_for_eq("flow_concat_u64", "v=42!", is_virtual=True)
        self.wait_for_eq("flow_concat_f32", "v=3.5!", is_virtual=True)
        self.wait_for_eq("flow_concat_f64", "v=3.5!", is_virtual=True)

        # str(): special float values via division by zero. Go's
        # strconv.FormatFloat emits "+Inf" / "-Inf" / "NaN".
        self.wait_for_eq("flow_str_pos_inf", "+Inf", is_virtual=True)
        self.wait_for_eq("flow_str_neg_inf", "-Inf", is_virtual=True)
        self.wait_for_eq("flow_str_nan", "NaN", is_virtual=True)

        # Numeric: one representative per opcode family + one composed in a
        # binary op. i32(in_i64) at 2^31 sign-flips to -2^31.
        self.wait_for_eq("flow_num_i32_to_i64", 42, is_virtual=True)
        self.wait_for_eq("flow_num_i64_to_i32", -(2**31), is_virtual=True)
        self.wait_for_near("flow_num_i32_to_f64", 42.0, tolerance=1e-9, is_virtual=True)
        self.wait_for_eq("flow_num_f64_to_i32", 3, is_virtual=True)
        self.wait_for_near("flow_num_f64_to_f32", 3.5, tolerance=1e-5, is_virtual=True)
        self.wait_for_eq("flow_num_arith", 103, is_virtual=True)
