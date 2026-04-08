#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from dataclasses import dataclass

from examples.simulators import PressSimDAQ

import synnax as sy
from tests.arc.arc_case import ArcConsoleCase

ARC_FOR_LOOP_SOURCE = """
func sum_i32(n i32) i32 {
    sum i32 := 0
    for i := range(1, n + 1) {
        sum = sum + i
    }
    return sum
}

func sum_i64(n i64) i64 {
    sum i64 := 0
    for i := range(i64(1), n + 1) {
        sum = sum + i
    }
    return sum
}

func sum_u8(n u8) u8 {
    sum u8 := 0
    for i := range(u8(1), n + 1) {
        sum = sum + i
    }
    return sum
}

func sum_u32(n u32) u32 {
    sum u32 := 0
    for i := range(u32(1), n + 1) {
        sum = sum + i
    }
    return sum
}

func sum_step(n i32) i32 {
    sum i32 := 0
    for i := range(0, n, 2) {
        sum = sum + i
    }
    return sum
}

for_in_i32 -> sum_i32{} -> for_out_i32
for_in_i64 -> sum_i64{} -> for_out_i64
for_in_u8 -> sum_u8{} -> for_out_u8
for_in_u32 -> sum_u32{} -> for_out_u32
for_in_step -> sum_step{} -> for_out_step
"""


@dataclass
class ForLoopCase:
    label: str
    in_ch: str
    in_dtype: sy.DataType
    out_ch: str
    out_dtype: sy.DataType
    write_val: int
    expected: int


CASES = [
    ForLoopCase("i32", "for_in_i32", sy.DataType.INT32,
                "for_out_i32", sy.DataType.INT32, 5, 15),
    ForLoopCase("i64", "for_in_i64", sy.DataType.INT64,
                "for_out_i64", sy.DataType.INT64, 5, 15),
    ForLoopCase("u8", "for_in_u8", sy.DataType.UINT8,
                "for_out_u8", sy.DataType.UINT8, 5, 15),
    ForLoopCase("u32", "for_in_u32", sy.DataType.UINT32,
                "for_out_u32", sy.DataType.UINT32, 5, 15),
    ForLoopCase("step", "for_in_step", sy.DataType.INT32,
                "for_out_step", sy.DataType.INT32, 10, 20),
]

ALL_CHANNELS = []
for c in CASES:
    ALL_CHANNELS.extend([c.in_ch, c.out_ch])


class ForLoops(ArcConsoleCase):
    """Test for-loop range() with different integer types.

    Verifies the range type inference fix: range() arguments now infer their
    type from concrete arguments rather than defaulting to i64.

    Each case writes a value to an input channel, the Arc function computes
    sum(1..n) using a for loop with the appropriate type, and the test
    checks the output channel for the expected result.
    """

    arc_source = ARC_FOR_LOOP_SOURCE
    arc_name_prefix = "ArcForLoops"
    start_cmd_channel = "start_for_loop_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ALL_CHANNELS
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        for case in CASES:
            self.client.channels.create(
                name=case.in_ch,
                data_type=case.in_dtype,
                virtual=True,
                retrieve_if_name_exists=True,
            )
            idx = self.client.channels.create(
                name=f"{case.out_ch}_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            self.client.channels.create(
                name=case.out_ch,
                data_type=case.out_dtype,
                index=idx.key,
                retrieve_if_name_exists=True,
            )
        super().setup()

    def verify_sequence_execution(self) -> None:
        for case in CASES:
            self.log(f"[{case.label}] Writing {case.write_val} to {case.in_ch}")
            with self.client.open_writer(
                sy.TimeStamp.now(), case.in_ch
            ) as w:
                w.write(case.in_ch, case.write_val)

            self.log(
                f"[{case.label}] Waiting for {case.out_ch} == {case.expected}"
            )
            self.wait_for_eq(case.out_ch, case.expected)
