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

func sum_step(n i64) i64 {
    sum i64 := 0
    for i := range(0, n, 2) {
        sum = sum + i
    }
    return sum
}

func accumulate_until{limit i32}(n i32) i32 {
    sum i32 := 0
    for i := range(1, n + 1) {
        sum = sum + i
        if sum > limit {
            break
        }
    }
    return sum
}

func sum_excluding{skip i32}(n i32) i32 {
    sum i32 := 0
    for i := range(1, n + 1) {
        if i == skip {
            continue
        }
        sum = sum + i
    }
    return sum
}

func countdown(n i64) i64 {
    sum i64 := 0
    for i := range(n, 0, -1) {
        sum = sum + i
    }
    return sum
}

func countdown_by_two(n i32) i32 {
    sum i32 := 0
    for i := range(n, 0, -2) {
        sum = sum + i
    }
    return sum
}

func drain_until{limit i32}(start i32) i32 {
    sum i32 := 0
    for i := range(start, 0, -1) {
        sum = sum + i
        if sum > limit {
            break
        }
    }
    return sum
}

func triangular(n u32) u32 {
    sum u32 := 0
    for i := range(u32(1), n + 1) {
        for j := range(i) {
            sum = sum + 1
        }
    }
    return sum
}

func inner_break_sums(n i32) i64 {
    total i64 := 0
    for i := range(1, n + 1) {
        for j := range(1, 100) {
            if j > i64(i) {
                break
            }
            total = total + j
        }
    }
    return total
}

func off_diagonal(n u32) u32 {
    count u32 := 0
    for i := range(n) {
        for j := range(n) {
            if i == j {
                continue
            }
            count = count + 1
        }
    }
    return count
}

func nest4(n i32) i32 {
    sum i32 := 0
    for a := range(n) {
        for b := range(n) {
            for c := range(n) {
                for d := range(n) {
                    sum = sum + 1
                }
            }
        }
    }
    return sum
}

func nest5(n i64) i64 {
    sum i64 := 0
    for a := range(n) {
        for b := range(n) {
            for c := range(n) {
                for d := range(n) {
                    for e := range(n) {
                        sum = sum + 1
                    }
                }
            }
        }
    }
    return sum
}

func factorial(n i64) i64 {
    result i64 := 1
    for i := range(i64(1), n + 1) {
        result = result * i
    }
    return result
}

func sum_factorials(n i64) i64 {
    total i64 := 0
    for i := range(i64(1), n + 1) {
        total = total + factorial(i)
    }
    return total
}

func loop_channel_write() {
    count f32 := 0.0
    for i := range(5) {
        count = count + 1.0
        loop_counter = count
    }
}

for_in_i32 -> sum_i32{} -> for_out_i32
for_in_i64 -> sum_i64{} -> for_out_i64
for_in_u8 -> sum_u8{} -> for_out_u8
for_in_u32 -> sum_u32{} -> for_out_u32
for_in_step -> sum_step{} -> for_out_step
for_in_break -> accumulate_until{limit=10} -> for_out_break
for_in_skip -> sum_excluding{skip=3} -> for_out_skip
for_in_countdown -> countdown{} -> for_out_countdown
for_in_cd_two -> countdown_by_two{} -> for_out_cd_two
for_in_drain -> drain_until{limit=12} -> for_out_drain
for_in_tri -> triangular{} -> for_out_tri
for_in_inner_brk -> inner_break_sums{} -> for_out_inner_brk
for_in_offdiag -> off_diagonal{} -> for_out_offdiag
for_in_nest4 -> nest4{} -> for_out_nest4
for_in_nest5 -> nest5{} -> for_out_nest5
for_in_rec -> sum_factorials{} -> for_out_rec
interval{period=100ms} -> loop_channel_write{}
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
    # Type inference across integer widths
    ForLoopCase("i32", "for_in_i32", sy.DataType.INT32,
                "for_out_i32", sy.DataType.INT32, 5, 15),
    ForLoopCase("i64", "for_in_i64", sy.DataType.INT64,
                "for_out_i64", sy.DataType.INT64, 5, 15),
    ForLoopCase("u8", "for_in_u8", sy.DataType.UINT8,
                "for_out_u8", sy.DataType.UINT8, 5, 15),
    ForLoopCase("u32", "for_in_u32", sy.DataType.UINT32,
                "for_out_u32", sy.DataType.UINT32, 5, 15),
    # Positive step (range with 3 args)
    ForLoopCase("step", "for_in_step", sy.DataType.INT64,
                "for_out_step", sy.DataType.INT64, 10, 20),
    # Break: accumulate 1+2+...until sum > 10 → 1+2+3+4+5=15
    ForLoopCase("break", "for_in_break", sy.DataType.INT32,
                "for_out_break", sy.DataType.INT32, 100, 15),
    # Continue: sum 1..5 skipping 3 → 1+2+4+5=12
    ForLoopCase("continue", "for_in_skip", sy.DataType.INT32,
                "for_out_skip", sy.DataType.INT32, 5, 12),
    # Negative range: countdown 5→1 → 5+4+3+2+1=15
    ForLoopCase("countdown", "for_in_countdown", sy.DataType.INT64,
                "for_out_countdown", sy.DataType.INT64, 5, 15),
    # Negative step=-2: 10,8,6,4,2 → 30
    ForLoopCase("countdown_by_2", "for_in_cd_two", sy.DataType.INT32,
                "for_out_cd_two", sy.DataType.INT32, 10, 30),
    # Break + negative range: countdown from 10, stop when sum > 12 → 10+9=19
    ForLoopCase("drain_break", "for_in_drain", sy.DataType.INT32,
                "for_out_drain", sy.DataType.INT32, 10, 19),
    # Nested: triangular number T(4) = 1+2+3+4 = 10
    ForLoopCase("nested", "for_in_tri", sy.DataType.UINT32,
                "for_out_tri", sy.DataType.UINT32, 4, 10),
    # Nested + inner break: sum(1..i) for each i in 1..4 → 1+3+6+10=20
    ForLoopCase("inner_break", "for_in_inner_brk", sy.DataType.INT32,
                "for_out_inner_brk", sy.DataType.INT64, 4, 20),
    # Nested + continue: off-diagonal count in 4x4 grid → 16-4=12
    ForLoopCase("nested_continue", "for_in_offdiag", sy.DataType.UINT32,
                "for_out_offdiag", sy.DataType.UINT32, 4, 12),
    # 4-deep nesting: n^4 = 3^4 = 81
    ForLoopCase("nest4", "for_in_nest4", sy.DataType.INT32,
                "for_out_nest4", sy.DataType.INT32, 3, 81),
    # 5-deep nesting: n^5 = 3^5 = 243
    ForLoopCase("nest5", "for_in_nest5", sy.DataType.INT64,
                "for_out_nest5", sy.DataType.INT64, 3, 243),
    # Recursion inside a for loop: sum of factorials 1!+2!+3!+4!+5! = 153
    ForLoopCase("recurse_in_loop", "for_in_rec", sy.DataType.INT64,
                "for_out_rec", sy.DataType.INT64, 5, 153),
]

ALL_CHANNELS = ["loop_counter"]
for c in CASES:
    ALL_CHANNELS.extend([c.in_ch, c.out_ch])


class ForLoops(ArcConsoleCase):
    """Test for-loop range() with type inference, break, continue, and
    negative ranges. Each case writes to an input channel, the Arc function
    computes a result using a for loop, and the test checks the output."""

    arc_source = ARC_FOR_LOOP_SOURCE
    arc_name_prefix = "ArcForLoops"
    start_cmd_channel = "start_for_loop_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ALL_CHANNELS
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.client.channels.create(
            name="loop_counter",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )
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

        self.log("[loop_write] Verifying channel write inside loop body")
        self.wait_for_near(
            "loop_counter", 5.0, tolerance=0.01, is_virtual=True
        )
