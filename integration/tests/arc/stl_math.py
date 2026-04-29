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
from framework.utils import create_indexed_pair, create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_STL_MATH_SOURCE = """
// ─────────────────────── pow operator (const, const) ────────────────
func pow_cc_f64() f64 {
    return 3.0 ^ 2.0
}
func pow_cc_int() f64 {
    return 5 ^ 2
}
pow_cc_trigger -> pow_cc_f64{} -> pow_cc_f64_out
pow_cc_trigger -> pow_cc_int{} -> pow_cc_int_out
// ─────────────────────── pow operator (chan, const) ─────────────────
func pow_xc_f64(base f64) f64 {
    return base ^ 2.0
}
func pow_xc_i64(base i64) i64 {
    return base ^ 2
}
pow_xc_f64_in -> pow_xc_f64{} -> pow_xc_f64_out
pow_xc_i64_in -> pow_xc_i64{} -> pow_xc_i64_out
// ─────────────────────── pow operator (const, chan) ─────────────────
func pow_cx_f64(exp f64) f64 {
    return 2.0 ^ exp
}
func pow_cx_i64(exp i64) i64 {
    return 2 ^ exp
}
pow_cx_f64_in -> pow_cx_f64{} -> pow_cx_f64_out
pow_cx_i64_in -> pow_cx_i64{} -> pow_cx_i64_out
// ─────────────────────── pow operator (chan, chan) ──────────────────
func pow_xx_f64(base f64) {
    pow_xx_f64_out = base ^ pow_xx_f64_exp
}
func pow_xx_i64(base i64) {
    pow_xx_i64_out = base ^ pow_xx_i64_exp
}
pow_xx_f64_in -> pow_xx_f64{}
pow_xx_i64_in -> pow_xx_i64{}

// ─────────────────────── arithmetic operators (WASM) ────────────────

func do_add(a f64) { op_add_out = a + op_add_b }
op_add_a -> do_add{}

func do_sub(a f64) { op_sub_out = a - op_sub_b }
op_sub_a -> do_sub{}

func do_mul(a f64) { op_mul_out = a * op_mul_b }
op_mul_a -> do_mul{}

func do_div(a f64) { op_div_out = a / op_div_b }
op_div_a -> do_div{}

func do_mod(a i64) { op_mod_out = a % op_mod_b }
op_mod_a -> do_mod{}

func do_neg(a f64) { op_neg_out = -a }
op_neg_a -> do_neg{}

// ─────────────────────── math.avg / math.min / math.max ─────────────

stat_in -> math.avg{} -> stat_avg_out
stat_in -> math.min{} -> stat_min_out
stat_in -> math.max{} -> stat_max_out

// ─────────────────────── math.avg/min/max (count window) ────────────

stat_count_in -> math.avg{count=5} -> stat_avg_count_out
stat_count_in -> math.min{count=5} -> stat_min_count_out
stat_count_in -> math.max{count=5} -> stat_max_count_out

// ─────────────────────── math.avg/min/max (duration window) ─────────

stat_dur_in -> math.avg{duration=500ms} -> stat_avg_dur_out
stat_dur_in -> math.min{duration=500ms} -> stat_min_dur_out
stat_dur_in -> math.max{duration=500ms} -> stat_max_dur_out

// ─────────────────────── math.avg/min/max (negative values) ─────────

stat_neg_in -> math.avg{} -> stat_neg_avg_out
stat_neg_in -> math.min{} -> stat_neg_min_out
stat_neg_in -> math.max{} -> stat_neg_max_out

// ─────────────────────── math.avg/min/max (edge cases) ──────────────

stat_edge_in -> math.avg{} -> stat_edge_avg_out
stat_edge_in -> math.min{} -> stat_edge_min_out
stat_edge_in -> math.max{} -> stat_edge_max_out

// ─────────────────────── math.derivative ────────────────────────────

stat_deriv_in -> math.derivative{} -> stat_deriv_out

"""


@dataclass
class PowCase:
    label: str
    in_ch: str | None
    in_dtype: sy.DataType | None
    out_ch: str
    write_val: int | float | None
    expected: float


@dataclass
class ChanChanCase:
    label: str
    base_ch: str
    base_dtype: sy.DataType
    exp_ch: str
    exp_dtype: sy.DataType
    out_ch: str
    base_val: int | float
    exp_val: int | float
    expected: float


# pow(const, const) — triggered by pow_cc_trigger, value unused
CC_CASES = [
    PowCase("cc_f64", None, None, "pow_cc_f64_out", None, 9.0),
    PowCase("cc_int", None, None, "pow_cc_int_out", None, 25.0),
]

# pow(chan, const) — base from dataflow
XC_CASES = [
    PowCase("xc_f64", "pow_xc_f64_in", sy.DataType.FLOAT64, "pow_xc_f64_out", 3.0, 9.0),
    PowCase("xc_i64", "pow_xc_i64_in", sy.DataType.INT64, "pow_xc_i64_out", 7, 49.0),
]

# pow(const, chan) — exp from dataflow
CX_CASES = [
    PowCase("cx_f64", "pow_cx_f64_in", sy.DataType.FLOAT64, "pow_cx_f64_out", 3.0, 8.0),
    PowCase("cx_i64", "pow_cx_i64_in", sy.DataType.INT64, "pow_cx_i64_out", 3, 8.0),
]

# pow(chan, chan) — base from dataflow, exp from body channel ref
XX_CASES = [
    ChanChanCase(
        "xx_f64",
        "pow_xx_f64_in",
        sy.DataType.FLOAT64,
        "pow_xx_f64_exp",
        sy.DataType.FLOAT64,
        "pow_xx_f64_out",
        base_val=2.0,
        exp_val=3.0,
        expected=8.0,
    ),
    ChanChanCase(
        "xx_i64",
        "pow_xx_i64_in",
        sy.DataType.INT64,
        "pow_xx_i64_exp",
        sy.DataType.INT64,
        "pow_xx_i64_out",
        base_val=2,
        exp_val=10,
        expected=1024.0,
    ),
]

FLOW_CASES: list[PowCase] = list(XC_CASES) + list(CX_CASES)


@dataclass
class BinaryOpCase:
    label: str
    a_ch: str
    b_ch: str
    out_ch: str
    a_val: float
    b_val: float
    expected: float


OP_BINARY_CASES = [
    BinaryOpCase("add", "op_add_a", "op_add_b", "op_add_out", 10.0, 3.0, 13.0),
    BinaryOpCase("sub", "op_sub_a", "op_sub_b", "op_sub_out", 10.0, 3.0, 7.0),
    BinaryOpCase("mul", "op_mul_a", "op_mul_b", "op_mul_out", 4.0, 5.0, 20.0),
    BinaryOpCase("div", "op_div_a", "op_div_b", "op_div_out", 20.0, 4.0, 5.0),
    BinaryOpCase("mod", "op_mod_a", "op_mod_b", "op_mod_out", 10, 3, 1),
]

STAT_VIRTUAL_INPUTS = [
    "stat_in",
    "stat_count_in",
    "stat_dur_in",
    "stat_neg_in",
    "stat_edge_in",
    "stat_deriv_in",
]

STAT_INDEXED_OUTPUTS = [
    "stat_avg_out",
    "stat_min_out",
    "stat_max_out",
    "stat_avg_count_out",
    "stat_min_count_out",
    "stat_max_count_out",
    "stat_avg_dur_out",
    "stat_min_dur_out",
    "stat_max_dur_out",
    "stat_neg_avg_out",
    "stat_neg_min_out",
    "stat_neg_max_out",
    "stat_edge_avg_out",
    "stat_edge_min_out",
    "stat_edge_max_out",
    "stat_deriv_out",
]


class StlMath(ArcConsoleCase):
    """Test math module: pow, avg, min, max, derivative.

    pow — parameterized across input types and data types.
    avg/min/max/derivative — behavioral tests for windowing and edge cases.
    """

    arc_source = ARC_STL_MATH_SOURCE
    arc_name_prefix = "ArcStlMath"
    start_cmd_channel = "start_stl_math_cmd"
    subscribe_channels: list[str] = []

    def setup(self) -> None:
        self._setup_pow_channels()
        self._setup_op_channels()
        self._setup_stat_channels()
        super().setup()

    def _setup_pow_channels(self) -> None:
        create_virtual_channel(self.client, "pow_cc_trigger", sy.DataType.FLOAT64)

        for c in CC_CASES:
            create_indexed_pair(self.client, c.out_ch, sy.DataType.FLOAT64)

        for fc in FLOW_CASES:
            assert fc.in_ch is not None and fc.in_dtype is not None
            create_virtual_channel(self.client, fc.in_ch, fc.in_dtype)
            out_dtype = (
                sy.DataType.INT64
                if fc.label in ("xc_i64", "cx_i64")
                else sy.DataType.FLOAT64
            )
            create_indexed_pair(self.client, fc.out_ch, out_dtype)

        for xxc in XX_CASES:
            create_virtual_channel(self.client, xxc.base_ch, xxc.base_dtype)
            create_virtual_channel(self.client, xxc.exp_ch, xxc.exp_dtype)
            out_dtype = (
                sy.DataType.INT64 if xxc.label == "xx_i64" else sy.DataType.FLOAT64
            )
            create_virtual_channel(self.client, xxc.out_ch, out_dtype)

        all_ch: list[str] = ["pow_cc_trigger"]
        for cc in CC_CASES:
            all_ch.append(cc.out_ch)
        for fc in FLOW_CASES:
            assert fc.in_ch is not None
            all_ch.extend([fc.in_ch, fc.out_ch])
        for xxc in XX_CASES:
            all_ch.extend([xxc.base_ch, xxc.exp_ch, xxc.out_ch])
        self.subscribe_channels = all_ch

    def _setup_op_channels(self) -> None:
        for c in OP_BINARY_CASES:
            dtype = sy.DataType.INT64 if c.label == "mod" else sy.DataType.FLOAT64
            create_virtual_channel(self.client, c.a_ch, dtype)
            create_virtual_channel(self.client, c.b_ch, dtype)
            create_virtual_channel(self.client, c.out_ch, dtype)
        create_virtual_channel(self.client, "op_neg_a", sy.DataType.FLOAT64)
        create_virtual_channel(self.client, "op_neg_out", sy.DataType.FLOAT64)
        for c in OP_BINARY_CASES:
            self.subscribe_channels += [c.a_ch, c.b_ch, c.out_ch]
        self.subscribe_channels += ["op_neg_a", "op_neg_out"]

    def _setup_stat_channels(self) -> None:
        for name in STAT_VIRTUAL_INPUTS:
            create_virtual_channel(self.client, name, sy.DataType.FLOAT64)
        for name in STAT_INDEXED_OUTPUTS:
            create_indexed_pair(self.client, name, sy.DataType.FLOAT64)
        self.subscribe_channels += STAT_VIRTUAL_INPUTS + STAT_INDEXED_OUTPUTS

    def _write_many(self, channel: str, values: list[float]) -> None:
        for val in values:
            self.writer.write(channel, val)

    def _write_spaced(
        self,
        channel: str,
        values: list[float],
        dt_ms: int = 20,
    ) -> None:
        for i, val in enumerate(values):
            self.writer.write(channel, val)
            if i < len(values) - 1:
                sy.sleep(dt_ms / 1000.0)

    def verify_sequence_execution(self) -> None:
        self._verify_pow_const_const()
        self._verify_pow_chan_const()
        self._verify_pow_const_chan()
        self._verify_pow_chan_chan()
        self._verify_arithmetic_ops()
        self._verify_stat_basic()
        self._verify_stat_count_window()
        self._verify_stat_count_window_negative()
        self._verify_stat_duration_window()
        self._verify_stat_negative_values()
        self._verify_stat_all_identical()
        self._verify_stat_large_magnitudes()
        self._verify_stat_derivative_constant()
        self._verify_stat_derivative_alternating()

    def _verify_pow_const_const(self) -> None:
        self.log("=== pow(const, const) ===")
        self.writer.write("pow_cc_trigger", 1.0)
        for c in CC_CASES:
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected)

    def _verify_pow_chan_const(self) -> None:
        self.log("=== pow(chan, const) ===")
        for c in XC_CASES:
            assert c.in_ch is not None
            self.log(f"[{c.label}] Writing {c.write_val} to {c.in_ch}")
            self.writer.write(c.in_ch, c.write_val)
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected)

    def _verify_pow_const_chan(self) -> None:
        self.log("=== pow(const, chan) ===")
        for c in CX_CASES:
            assert c.in_ch is not None
            self.log(f"[{c.label}] Writing {c.write_val} to {c.in_ch}")
            self.writer.write(c.in_ch, c.write_val)
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected)

    def _verify_pow_chan_chan(self) -> None:
        self.log("=== pow(chan, chan) ===")
        for c in XX_CASES:
            self.log(f"[{c.label}] Writing exp={c.exp_val} to {c.exp_ch}")
            self.writer.write(c.exp_ch, c.exp_val)
            self.log(f"[{c.label}] Writing base={c.base_val} to {c.base_ch}")
            self.writer.write(c.base_ch, c.base_val)
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected, is_virtual=True)

    def _verify_arithmetic_ops(self) -> None:
        self.log("=== arithmetic operators ===")
        for c in OP_BINARY_CASES:
            self.log(f"[{c.label}] Writing b={c.b_val} to {c.b_ch}")
            self.writer.write(c.b_ch, c.b_val)
            self.log(f"[{c.label}] Writing a={c.a_val} to {c.a_ch}")
            self.writer.write(c.a_ch, c.a_val)
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected, is_virtual=True)

        self.log("[neg] Writing 7.0 to op_neg_a")
        self.writer.write("op_neg_a", 7.0)
        self.log("[neg] Expecting op_neg_out == -7.0")
        self.wait_for_eq("op_neg_out", -7.0, is_virtual=True)

    def _verify_stat_basic(self) -> None:
        self.log("Basic stats: [10, 20, 30]")
        self._write_many("stat_in", [10.0, 20.0, 30.0])

        self.log("[avg] Expecting 20.0")
        self.wait_for_eq("stat_avg_out", 20.0)

        self.log("[min] Expecting 10.0")
        self.wait_for_eq("stat_min_out", 10.0)

        self.log("[max] Expecting 30.0")
        self.wait_for_eq("stat_max_out", 30.0)

        self._write_spaced("stat_deriv_in", [0.0, 1.0])
        self.log("[deriv] Expecting ≈ 50")
        self.wait_for_near("stat_deriv_out", 50.0, tolerance=25.0)

    def _verify_stat_count_window(self) -> None:
        self.log("Count window (count=5): 15 samples, 3 windows")
        self._write_many(
            "stat_count_in",
            [
                10.0,
                20.0,
                30.0,
                40.0,
                50.0,
                100.0,
                200.0,
                300.0,
                400.0,
                500.0,
                1.0,
                2.0,
                3.0,
                4.0,
                5.0,
            ],
        )

        self.wait_for_eq("stat_avg_count_out", 3.0)
        self.wait_for_eq("stat_min_count_out", 1.0)
        self.wait_for_eq("stat_max_count_out", 5.0)

    def _verify_stat_count_window_negative(self) -> None:
        self.log("Count window: all negative [-10,-20,-30,-40,-50]")
        self._write_many(
            "stat_count_in",
            [-10.0, -20.0, -30.0, -40.0, -50.0],
        )

        self.wait_for_eq("stat_avg_count_out", -30.0)
        self.wait_for_eq("stat_min_count_out", -50.0)
        self.wait_for_eq("stat_max_count_out", -10.0)

    def _verify_stat_duration_window(self) -> None:
        self.log("Duration window (500ms): batch, wait 600ms, batch")
        self._write_many("stat_dur_in", [1.0, 2.0, 3.0, 4.0, 5.0])
        self.wait_for_eq("stat_max_dur_out", 5.0)

        sy.sleep(0.6)

        self._write_many("stat_dur_in", [100.0, 200.0, 300.0])

        self.log("[dur_max] Expecting 300.0")
        self.wait_for_eq("stat_max_dur_out", 300.0)

        self.log("[dur_min] Expecting 100.0 (batch 1 expired)")
        self.wait_for_eq("stat_min_dur_out", 100.0)

    def _verify_stat_negative_values(self) -> None:
        self.log("Negative values: [-50, -10, -200, 100, -300, 75]")
        self._write_many(
            "stat_neg_in",
            [-50.0, -10.0, -200.0, 100.0, -300.0, 75.0],
        )

        self.log("[neg_avg] Expecting ≈ -64.17")
        self.wait_for_near("stat_neg_avg_out", -64.17, tolerance=0.01)

        self.log("[neg_min] Expecting -300.0")
        self.wait_for_eq("stat_neg_min_out", -300.0)

        self.log("[neg_max] Expecting 100.0")
        self.wait_for_eq("stat_neg_max_out", 100.0)

        self._write_spaced("stat_deriv_in", [-1.0, 1.0])
        self.log("[neg_deriv] Expecting ≈ 100")
        self.wait_for_near("stat_deriv_out", 100.0, tolerance=50.0)

    def _verify_stat_all_identical(self) -> None:
        self.log("All identical: [42, 42, 42, 42, 42]")
        self._write_many("stat_edge_in", [42.0, 42.0, 42.0, 42.0, 42.0])

        self.wait_for_eq("stat_edge_avg_out", 42.0)
        self.wait_for_eq("stat_edge_min_out", 42.0)
        self.wait_for_eq("stat_edge_max_out", 42.0)

    def _verify_stat_large_magnitudes(self) -> None:
        self.log("Large magnitudes: [1e12, 2e12, 3e12]")
        self._write_many("stat_edge_in", [1e12, 2e12, 3e12])

        self.wait_for_eq("stat_edge_max_out", 3e12)

    def _verify_stat_derivative_constant(self) -> None:
        self.log("Derivative constant: 2 x 5.0 at 20ms")
        self._write_spaced("stat_deriv_in", [5.0, 5.0])

        self.log("[deriv] Expecting rate = 0")
        self.wait_for_eq("stat_deriv_out", 0.0)

    def _verify_stat_derivative_alternating(self) -> None:
        self.log("Derivative positive: [0, 1.0] at 20ms")
        self._write_spaced("stat_deriv_in", [0.0, 1.0])

        self.log("[deriv] Expecting ≈ 50")
        self.wait_for_near("stat_deriv_out", 50.0, tolerance=25.0)

        self.log("Derivative negative: [1.0, 0] at 20ms")
        self._write_spaced("stat_deriv_in", [1.0, 0.0])

        self.log("[deriv] Expecting ≈ -50")
        self.wait_for_near("stat_deriv_out", -50.0, tolerance=25.0)
