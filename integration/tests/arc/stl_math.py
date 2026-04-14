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
from tests.arc.arc_case import ArcConsoleCase

ARC_STL_MATH_SOURCE = """
// ─────────────────────── math.pow(const, const) ─────────────────────

func pow_cc_f64() f64 { return math.pow(3.0, 2.0) }
func pow_cc_int() f64 { return math.pow(5, 2) }
pow_cc_trigger -> pow_cc_f64{} -> pow_cc_f64_out
pow_cc_trigger -> pow_cc_int{} -> pow_cc_int_out

// ─────────────────────── math.pow(chan, const) ──────────────────────

func pow_xc_f64(base f64) f64 { return math.pow(base, 2.0) }
func pow_xc_i64(base i64) f64 { return math.pow(base, 2) }
pow_xc_f64_in -> pow_xc_f64{} -> pow_xc_f64_out
pow_xc_i64_in -> pow_xc_i64{} -> pow_xc_i64_out

// ─────────────────────── math.pow(const, chan) ──────────────────────

func pow_cx_f64(exp f64) f64 { return math.pow(2.0, exp) }
func pow_cx_i64(exp i64) f64 { return math.pow(2, exp) }
pow_cx_f64_in -> pow_cx_f64{} -> pow_cx_f64_out
pow_cx_i64_in -> pow_cx_i64{} -> pow_cx_i64_out

// ─────────────────────── math.pow(chan, chan) ────────────────────────

func pow_xx_f64(base f64) { pow_xx_f64_out = math.pow(base, pow_xx_f64_exp) }
func pow_xx_i64(base i64) { pow_xx_i64_out = math.pow(base, pow_xx_i64_exp) }
pow_xx_f64_in -> pow_xx_f64{}
pow_xx_i64_in -> pow_xx_i64{}

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


class StlMath(ArcConsoleCase):
    """Test math.pow() with qualified module syntax.

    Primary axis: input type (const/const, chan/const, const/chan, chan/chan).
    Secondary axis: data type (f64, i32, i64).
    All outputs are f64 — math.pow always returns f64.
    """

    arc_source = ARC_STL_MATH_SOURCE
    arc_name_prefix = "ArcStlMath"
    start_cmd_channel = "start_stl_math_cmd"
    subscribe_channels: list[str] = []

    def setup(self) -> None:
        create_virtual_channel(self.client, "pow_cc_trigger", sy.DataType.FLOAT64)

        # All output channels are f64
        for c in CC_CASES:
            idx = self.client.channels.create(
                name=f"{c.out_ch}_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            self.client.channels.create(
                name=c.out_ch,
                data_type=sy.DataType.FLOAT64,
                index=idx.key,
                retrieve_if_name_exists=True,
            )

        for fc in FLOW_CASES:
            assert fc.in_ch is not None and fc.in_dtype is not None
            create_virtual_channel(self.client, fc.in_ch, fc.in_dtype)
            idx = self.client.channels.create(
                name=f"{fc.out_ch}_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            self.client.channels.create(
                name=fc.out_ch,
                data_type=sy.DataType.FLOAT64,
                index=idx.key,
                retrieve_if_name_exists=True,
            )

        for xxc in XX_CASES:
            create_virtual_channel(self.client, xxc.base_ch, xxc.base_dtype)
            create_virtual_channel(self.client, xxc.exp_ch, xxc.exp_dtype)
            create_virtual_channel(self.client, xxc.out_ch, sy.DataType.FLOAT64)

        all_ch: list[str] = ["pow_cc_trigger"]
        for cc in CC_CASES:
            all_ch.append(cc.out_ch)
        for fc in FLOW_CASES:
            assert fc.in_ch is not None
            all_ch.extend([fc.in_ch, fc.out_ch])
        for xxc in XX_CASES:
            all_ch.extend([xxc.base_ch, xxc.exp_ch, xxc.out_ch])
        self.subscribe_channels = all_ch
        super().setup()

    def _verify_const_const(self) -> None:
        self.log("=== pow(const, const) ===")
        self.writer.write("pow_cc_trigger", 1.0)
        for c in CC_CASES:
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected)

    def _verify_chan_const(self) -> None:
        self.log("=== pow(chan, const) ===")
        for c in XC_CASES:
            assert c.in_ch is not None
            self.log(f"[{c.label}] Writing {c.write_val} to {c.in_ch}")
            self.writer.write(c.in_ch, c.write_val)
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected)

    def _verify_const_chan(self) -> None:
        self.log("=== pow(const, chan) ===")
        for c in CX_CASES:
            assert c.in_ch is not None
            self.log(f"[{c.label}] Writing {c.write_val} to {c.in_ch}")
            self.writer.write(c.in_ch, c.write_val)
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected)

    def _verify_chan_chan(self) -> None:
        self.log("=== pow(chan, chan) ===")
        for c in XX_CASES:
            self.log(f"[{c.label}] Writing exp={c.exp_val} to {c.exp_ch}")
            self.writer.write(c.exp_ch, c.exp_val)
            self.log(f"[{c.label}] Writing base={c.base_val} to {c.base_ch}")
            self.writer.write(c.base_ch, c.base_val)
            self.log(f"[{c.label}] Expecting {c.out_ch} == {c.expected}")
            self.wait_for_eq(c.out_ch, c.expected, is_virtual=True)

    def verify_sequence_execution(self) -> None:
        self._verify_const_const()
        self._verify_chan_const()
        self._verify_const_chan()
        self._verify_chan_chan()
