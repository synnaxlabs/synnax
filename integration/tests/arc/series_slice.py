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
from framework.utils import create_virtual_channels
from tests.arc.arc import ArcCase

ARC_SERIES_SLICE_SOURCE = """
func slice_values() {
    // ──── slice type ────
    data := [1.0, 2.5, 3.7, 4.2]
    sub := data[1:3]
    typed series f64 := data[0:2]
    slice_first_out = sub[0]
    slice_typed_out = typed[1]
    sum f64 := 0.0
    for v := sub {
        sum = sum + v
    }
    slice_sum_out = sum

    // ──── slice bounds ────
    // Each output has the first element, the last element, and the length as digits.
    n := [1.0, 2.0, 3.0, 4.0, 5.0]
    s1 := n[3:]
    slice_open_end_out = s1[0] * 100.0 + s1[-1] * 10.0 + f64(len(s1))
    s2 := n[:2]
    slice_open_start_out = s2[0] * 100.0 + s2[-1] * 10.0 + f64(len(s2))
    s3 := n[0:-3]
    slice_neg_end_out = s3[0] * 100.0 + s3[-1] * 10.0 + f64(len(s3))
    s4 := n[1:-1]
    slice_inner_out = s4[0] * 100.0 + s4[-1] * 10.0 + f64(len(s4))
    s5 := n[-3:-1]
    slice_neg_both_out = s5[0] * 100.0 + s5[-1] * 10.0 + f64(len(s5))
    s6 := n[:-2]
    slice_open_neg_end_out = s6[0] * 100.0 + s6[-1] * 10.0 + f64(len(s6))
    s7 := n[-1:]
    slice_neg_start_out = s7[0] * 100.0 + s7[-1] * 10.0 + f64(len(s7))
    s8 := n[-10:2]
    slice_before_start_out = s8[0] * 100.0 + s8[-1] * 10.0 + f64(len(s8))
    s9 := n[3:10]
    slice_past_end_out = s9[0] * 100.0 + s9[-1] * 10.0 + f64(len(s9))
    s10 := n[0:-10]
    slice_before_series_out = f64(len(s10))
    s11 := n[-1:-3]
    slice_crossed_out = f64(len(s11))

    // ──── slice copy ────
    parent := [1.0, 2.0, 3.0, 4.0]
    part := parent[1:3]
    part[0] = 9.0
    slice_copy_part_out = part[0]
    slice_copy_parent_out = parent[1]
    parent[2] = 7.0
    slice_copy_after_out = part[1]
}

start_series_slice_cmd -> slice_values()
"""


@dataclass
class Bound:
    channel: str
    expr: str
    expected: float


TYPE_OUTPUTS = ["slice_first_out", "slice_typed_out", "slice_sum_out"]

BOUNDS = [
    Bound("slice_open_end_out", "n[3:]", 452),
    Bound("slice_open_start_out", "n[:2]", 122),
    Bound("slice_neg_end_out", "n[0:-3]", 122),
    Bound("slice_inner_out", "n[1:-1]", 243),
    Bound("slice_neg_both_out", "n[-3:-1]", 342),
    Bound("slice_open_neg_end_out", "n[:-2]", 133),
    Bound("slice_neg_start_out", "n[-1:]", 551),
    Bound("slice_before_start_out", "n[-10:2]", 122),
    Bound("slice_past_end_out", "n[3:10]", 452),
    Bound("slice_before_series_out", "len(n[0:-10])", 0),
    Bound("slice_crossed_out", "len(n[-1:-3])", 0),
]

COPY_OUTPUTS = ["slice_copy_part_out", "slice_copy_parent_out", "slice_copy_after_out"]

OUTPUTS = TYPE_OUTPUTS + [b.channel for b in BOUNDS] + COPY_OUTPUTS


class SeriesSlice(ArcCase):
    """Test that a slice of a series is a series with its own data, and that its bounds
    count from the end when negative and stop at the ends of the series."""

    arc_source = ARC_SERIES_SLICE_SOURCE
    arc_name_prefix = "ArcSeriesSlice"
    start_cmd_channel = "start_series_slice_cmd"
    subscribe_channels = OUTPUTS

    def setup(self) -> None:
        create_virtual_channels(
            self.client, [(name, sy.DataType.FLOAT64) for name in OUTPUTS]
        )
        super().setup()

    def verify_sequence_execution(self) -> None:
        self._verify_slice_type()
        self._verify_slice_bounds()
        self._verify_slice_copy()

    def _verify_slice_type(self) -> None:
        self.log("[untyped] Waiting for sub[0] == 2.5")
        self.wait_for_eq("slice_first_out", 2.5)

        self.log("[typed] Waiting for typed[1] == 2.5")
        self.wait_for_eq("slice_typed_out", 2.5)

        self.log("[iterate] Waiting for the sum of sub == 6.2")
        self.wait_for_eq("slice_sum_out", 6.2)

    def _verify_slice_bounds(self) -> None:
        for bound in BOUNDS:
            self.log(f"[{bound.expr}] Waiting for {bound.expected}")
            self.wait_for_eq(bound.channel, bound.expected)

    def _verify_slice_copy(self) -> None:
        self.log("[write to slice] Waiting for part[0] == 9.0")
        self.wait_for_eq("slice_copy_part_out", 9.0)

        self.log("[write to slice] Waiting for parent[1] == 2.0")
        self.wait_for_eq("slice_copy_parent_out", 2.0)

        self.log("[write to parent] Waiting for part[1] == 3.0")
        self.wait_for_eq("slice_copy_after_out", 3.0)
