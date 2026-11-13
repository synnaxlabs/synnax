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
from tests.arc.arc import ArcCase

ARC_FOR_LOOP_SOURCE = """
func sum_i32() {n i32} i32 {
    sum i32 := 0
    for i := range(1, n + 1) {
        sum = sum + i
    }
    return sum
}

func sum_u8() {n u8} u8 {
    sum u8 := 0
    for i := range(u8(1), n + 1) {
        sum = sum + i
    }
    return sum
}

func sum_u32() {n u32} u32 {
    sum u32 := 0
    for i := range(u32(1), n + 1) {
        sum = sum + i
    }
    return sum
}

func sum_step() {n i64} i64 {
    sum i64 := 0
    for i := range(0, n, 2) {
        sum = sum + i
    }
    return sum
}

func accumulate_until(limit i32) {n i32} i32 {
    sum i32 := 0
    for i := range(1, n + 1) {
        sum = sum + i
        if sum > limit {
            break
        }
    }
    return sum
}

func sum_excluding(skip i32) {n i32} i32 {
    sum i32 := 0
    for i := range(1, n + 1) {
        if i == skip {
            continue
        }
        sum = sum + i
    }
    return sum
}

func countdown() {n i64} i64 {
    sum i64 := 0
    for i := range(n, 0, -1) {
        sum = sum + i
    }
    return sum
}

func drain_until(limit i32) {start i32} i32 {
    sum i32 := 0
    for i := range(start, 0, -1) {
        sum = sum + i
        if sum > limit {
            break
        }
    }
    return sum
}

func triangular() {n u32} u32 {
    sum u32 := 0
    for i := range(u32(1), n + 1) {
        for j := range(i) {
            sum = sum + 1
        }
    }
    return sum
}

func nest5() {n i64} i64 {
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

func factorial() {n i64} i64 {
    result i64 := 1
    for i := range(i64(1), n + 1) {
        result = result * i
    }
    return result
}

func sum_factorials() {n i64} i64 {
    total i64 := 0
    for i := range(i64(1), n + 1) {
        total = total + factorial(i)
    }
    return total
}

func range_i16() {n i16} i16 {
    sum i16 := 0
    for i := range(i16(1), n + 1) {
        sum = sum + i
    }
    return sum
}

func empty_range() {n i32} i32 {
    sum i32 := 99
    for i := range(0) {
        sum = sum + 1
    }
    for i := range(i32(10), i32(5)) {
        sum = sum + 1
    }
    for i := range(i32(0), i32(10), i32( -1)) {
        sum = sum + 1
    }
    return sum
}

func while_countdown() {n i32} i32 {
    sum i32 := 0
    for n > 0 {
        sum = sum + n
        n = n - 1
    }
    return sum
}

func next_power_of_two() {n i32} i32 {
    val i32 := 1
    for {
        if val >= n {
            break
        }
        val = val * 2
    }
    return val
}

func loop_channel_write() {
    count f32 := 0.0
    for i := range(5) {
        count = count + 1.0
        loop_counter = count
    }
}

func sum_all() {
    data := [1.0, 2.5, 3.7, 4.2]
    sum f64 := 0.0
    for x := data {
        sum = sum + x
    }
    series_sum_all = sum
}

func find_peak() {
    data := [3.1, 7.5, 2.0, 9.8, 1.4]
    peak f64 := 0.0
    peak_idx i32 := 0
    for i, x := data {
        if x > peak {
            peak = x
            peak_idx = i
        }
    }
    series_peak_val = peak
    series_peak_idx = peak_idx
}

func index_last() {
    data series f64 := [1.0, 2.5, 3.7, 4.2]
    series_index_last = data[len(data) - 1]
}

func index_sum() {
    data series f64 := [1.0, 2.5, 3.7, 4.2]
    sum f64 := 0.0
    for i := range(len(data)) {
        sum = sum + data[i]
    }
    series_index_sum = sum
}

func index_write() {
    data series f64 := [1.0, 2.5, 3.7, 4.2]
    data[len(data) - 1] = 9.0
    data[len(data) - 2] += 1.0
    series_index_write = data[2] + data[3]
}

func slice_tail() {
    data series f64 := [1.0, 2.5, 3.7, 4.2]
    tail series f64 := data[len(data) - 2:len(data)]
    series_slice_tail = tail[0]
}


// A stepped loop over a narrow type must end when the step passes the limit of the
// type.
func step_u8() {n u8} i64 {
    count i64 := 0
    for i := range(u8(0), u8(255), n) {
        count = count + 1
    }
    return count
}

func step_u16() {n u16} i64 {
    count i64 := 0
    for i := range(u16(0), u16(65535), n) {
        count = count + 1
    }
    return count
}

func step_i8() {n i8} i64 {
    count i64 := 0
    for i := range(i8(0), i8(127), n) {
        count = count + 1
    }
    return count
}

func step_down_i8() {n i8} i64 {
    count i64 := 0
    for i := range(i8(0), i8(-125), n) {
        count = count + 1
    }
    return count
}

func step_i16() {n i16} i64 {
    count i64 := 0
    for i := range(i16(0), i16(32767), n) {
        count = count + 1
    }
    return count
}

func step_down_i16() {n i16} i64 {
    count i64 := 0
    for i := range(i16(0), i16(-32000), n) {
        count = count + 1
    }
    return count
}

func step_sum_u8() {n u8} i64 {
    total i64 := 0
    for i := range(u8(0), u8(255), n) {
        total = total + i64(i)
    }
    return total
}

// The same loop over a 32-bit or 64-bit type. Each one runs 3 passes.
func step_i32() {n i32} i64 {
    count i64 := 0
    for i := range(i32(0), i32(2147483647), n) {
        count = count + 1
    }
    return count
}

func step_down_i32() {n i32} i64 {
    count i64 := 0
    for i := range(i32(0), i32(-2147483647), n) {
        count = count + 1
    }
    return count
}

func step_u32() {n u32} i64 {
    count i64 := 0
    for i := range(u32(0), u32(4294967295), n) {
        count = count + 1
    }
    return count
}

func step_i64() {n i64} i64 {
    count i64 := 0
    for i := range(0, 9223372036854775807, n) {
        count = count + 1
    }
    return count
}

func step_down_i64() {n i64} i64 {
    count i64 := 0
    for i := range(0, -9223372036854775807, n) {
        count = count + 1
    }
    return count
}

func step_u64() {n u64} i64 {
    count i64 := 0
    limit u64 := 0
    limit = limit - 1
    for i := range(u64(0), limit, n) {
        count = count + 1
    }
    return count
}

for_in_i32 -> sum_i32() -> for_out_i32
for_in_u8 -> sum_u8() -> for_out_u8
for_in_u32 -> sum_u32() -> for_out_u32
for_in_step -> sum_step() -> for_out_step
for_in_break -> accumulate_until(limit=10) -> for_out_break
for_in_skip -> sum_excluding(skip=3) -> for_out_skip
for_in_countdown -> countdown() -> for_out_countdown
for_in_drain -> drain_until(limit=12) -> for_out_drain
for_in_tri -> triangular() -> for_out_tri
for_in_nest5 -> nest5() -> for_out_nest5
for_in_rec -> sum_factorials() -> for_out_rec
for_in_i16 -> range_i16() -> for_out_i16
for_in_empty -> empty_range() -> for_out_empty
for_in_while -> while_countdown() -> for_out_while
for_in_pow2 -> next_power_of_two() -> for_out_pow2
for_in_step_u8 -> step_u8() -> for_out_step_u8
for_in_step_u16 -> step_u16() -> for_out_step_u16
for_in_step_i8 -> step_i8() -> for_out_step_i8
for_in_step_down_i8 -> step_down_i8() -> for_out_step_down_i8
for_in_step_i16 -> step_i16() -> for_out_step_i16
for_in_step_down_i16 -> step_down_i16() -> for_out_step_down_i16
for_in_step_sum_u8 -> step_sum_u8() -> for_out_step_sum_u8
for_in_step_i32 -> step_i32() -> for_out_step_i32
for_in_step_down_i32 -> step_down_i32() -> for_out_step_down_i32
for_in_step_u32 -> step_u32() -> for_out_step_u32
for_in_step_i64 -> step_i64() -> for_out_step_i64
for_in_step_down_i64 -> step_down_i64() -> for_out_step_down_i64
for_in_step_u64 -> step_u64() -> for_out_step_u64
interval(period=100ms) -> loop_channel_write()
interval(period=100ms) -> sum_all()
interval(period=100ms) -> find_peak()
interval(period=100ms) -> index_last()
interval(period=100ms) -> index_sum()
interval(period=100ms) -> index_write()
interval(period=100ms) -> slice_tail()
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
    ForLoopCase(
        "i32", "for_in_i32", sy.DataType.INT32, "for_out_i32", sy.DataType.INT32, 5, 15
    ),
    ForLoopCase(
        "u8", "for_in_u8", sy.DataType.UINT8, "for_out_u8", sy.DataType.UINT8, 5, 15
    ),
    ForLoopCase(
        "u32",
        "for_in_u32",
        sy.DataType.UINT32,
        "for_out_u32",
        sy.DataType.UINT32,
        5,
        15,
    ),
    # Positive step (range with 3 args)
    ForLoopCase(
        "step",
        "for_in_step",
        sy.DataType.INT64,
        "for_out_step",
        sy.DataType.INT64,
        10,
        20,
    ),
    # Break: accumulate 1+2+...until sum > 10 → 1+2+3+4+5=15
    ForLoopCase(
        "break",
        "for_in_break",
        sy.DataType.INT32,
        "for_out_break",
        sy.DataType.INT32,
        100,
        15,
    ),
    # Continue: sum 1..5 skipping 3 → 1+2+4+5=12
    ForLoopCase(
        "continue",
        "for_in_skip",
        sy.DataType.INT32,
        "for_out_skip",
        sy.DataType.INT32,
        5,
        12,
    ),
    # Negative range: countdown 5→1 → 5+4+3+2+1=15
    ForLoopCase(
        "countdown",
        "for_in_countdown",
        sy.DataType.INT64,
        "for_out_countdown",
        sy.DataType.INT64,
        5,
        15,
    ),
    # Break + negative range: countdown from 10, stop when sum > 12 → 10+9=19
    ForLoopCase(
        "drain_break",
        "for_in_drain",
        sy.DataType.INT32,
        "for_out_drain",
        sy.DataType.INT32,
        10,
        19,
    ),
    # Nested: triangular number T(4) = 1+2+3+4 = 10
    ForLoopCase(
        "nested",
        "for_in_tri",
        sy.DataType.UINT32,
        "for_out_tri",
        sy.DataType.UINT32,
        4,
        10,
    ),
    # 5-deep nesting: n^5 = 3^5 = 243
    ForLoopCase(
        "nest5",
        "for_in_nest5",
        sy.DataType.INT64,
        "for_out_nest5",
        sy.DataType.INT64,
        3,
        243,
    ),
    # Recursion inside a for loop: sum of factorials 1!+2!+3!+4!+5! = 153
    ForLoopCase(
        "recurse_in_loop",
        "for_in_rec",
        sy.DataType.INT64,
        "for_out_rec",
        sy.DataType.INT64,
        5,
        153,
    ),
    # Pure i16 range: sum(1..6)=15
    ForLoopCase(
        "range_i16",
        "for_in_i16",
        sy.DataType.INT16,
        "for_out_i16",
        sy.DataType.INT16,
        5,
        15,
    ),
    # Empty ranges: none of the 3 loops execute, sum stays 99
    ForLoopCase(
        "empty_range",
        "for_in_empty",
        sy.DataType.INT32,
        "for_out_empty",
        sy.DataType.INT32,
        0,
        99,
    ),
    # Conditional (while-style) loop: sum n+(n-1)+...+1 → 5+4+3+2+1=15
    ForLoopCase(
        "while_loop",
        "for_in_while",
        sy.DataType.INT32,
        "for_out_while",
        sy.DataType.INT32,
        5,
        15,
    ),
    # Infinite loop + break: next power of 2 >= 5 → 8
    ForLoopCase(
        "infinite_break",
        "for_in_pow2",
        sy.DataType.INT32,
        "for_out_pow2",
        sy.DataType.INT32,
        5,
        8,
    ),
    # Stepped loops whose step passes the limit of the loop variable type. Each loop
    # must end there and not wrap around to run forever. One case covers each integer
    # type, with a negative step too for the signed types.
    ForLoopCase(
        "step_u8",
        "for_in_step_u8",
        sy.DataType.UINT8,
        "for_out_step_u8",
        sy.DataType.INT64,
        2,
        128,
    ),
    ForLoopCase(
        "step_u16",
        "for_in_step_u16",
        sy.DataType.UINT16,
        "for_out_step_u16",
        sy.DataType.INT64,
        30000,
        3,
    ),
    ForLoopCase(
        "step_i8",
        "for_in_step_i8",
        sy.DataType.INT8,
        "for_out_step_i8",
        sy.DataType.INT64,
        10,
        13,
    ),
    ForLoopCase(
        "step_down_i8",
        "for_in_step_down_i8",
        sy.DataType.INT8,
        "for_out_step_down_i8",
        sy.DataType.INT64,
        -10,
        13,
    ),
    ForLoopCase(
        "step_i16",
        "for_in_step_i16",
        sy.DataType.INT16,
        "for_out_step_i16",
        sy.DataType.INT64,
        10000,
        4,
    ),
    ForLoopCase(
        "step_down_i16",
        "for_in_step_down_i16",
        sy.DataType.INT16,
        "for_out_step_down_i16",
        sy.DataType.INT64,
        -10000,
        4,
    ),
    # The loop variable keeps its value up to the last pass, 0 + 2 + ... + 254
    ForLoopCase(
        "step_sum_u8",
        "for_in_step_sum_u8",
        sy.DataType.UINT8,
        "for_out_step_sum_u8",
        sy.DataType.INT64,
        2,
        16256,
    ),
    ForLoopCase(
        "step_i32",
        "for_in_step_i32",
        sy.DataType.INT32,
        "for_out_step_i32",
        sy.DataType.INT64,
        1000000000,
        3,
    ),
    ForLoopCase(
        "step_down_i32",
        "for_in_step_down_i32",
        sy.DataType.INT32,
        "for_out_step_down_i32",
        sy.DataType.INT64,
        -1000000000,
        3,
    ),
    ForLoopCase(
        "step_u32",
        "for_in_step_u32",
        sy.DataType.UINT32,
        "for_out_step_u32",
        sy.DataType.INT64,
        2000000000,
        3,
    ),
    ForLoopCase(
        "step_i64",
        "for_in_step_i64",
        sy.DataType.INT64,
        "for_out_step_i64",
        sy.DataType.INT64,
        4000000000000000000,
        3,
    ),
    ForLoopCase(
        "step_down_i64",
        "for_in_step_down_i64",
        sy.DataType.INT64,
        "for_out_step_down_i64",
        sy.DataType.INT64,
        -4000000000000000000,
        3,
    ),
    ForLoopCase(
        "step_u64",
        "for_in_step_u64",
        sy.DataType.UINT64,
        "for_out_step_u64",
        sy.DataType.INT64,
        8000000000000000000,
        3,
    ),
]

SERIES_VIRTUAL = [
    "loop_counter",
    "series_sum_all",
    "series_peak_val",
    "series_peak_idx",
    "series_index_last",
    "series_index_sum",
    "series_index_write",
    "series_slice_tail",
]

ALL_CHANNELS = list(SERIES_VIRTUAL)
for c in CASES:
    ALL_CHANNELS.extend([c.in_ch, c.out_ch])


class ForLoops(ArcCase):
    """Test for-loop range() with type inference, break, continue, and
    negative ranges. Each case writes to an input channel, the Arc function
    computes a result using a for loop, and the test checks the output."""

    arc_source = ARC_FOR_LOOP_SOURCE
    arc_name_prefix = "ArcForLoops"
    start_cmd_channel = "start_for_loop_cmd"
    subscribe_channels = ALL_CHANNELS

    def setup(self) -> None:
        virtual_channels = [
            ("loop_counter", sy.DataType.FLOAT32),
            ("series_sum_all", sy.DataType.FLOAT64),
            ("series_peak_val", sy.DataType.FLOAT64),
            ("series_peak_idx", sy.DataType.INT32),
            ("series_index_last", sy.DataType.FLOAT64),
            ("series_index_sum", sy.DataType.FLOAT64),
            ("series_index_write", sy.DataType.FLOAT64),
            ("series_slice_tail", sy.DataType.FLOAT64),
        ]
        for name, dtype in virtual_channels:
            create_virtual_channel(self.client, name, data_type=dtype)
        for case in CASES:
            create_virtual_channel(self.client, case.in_ch, data_type=case.in_dtype)
            create_indexed_pair(self.client, case.out_ch, data_type=case.out_dtype)
        super().setup()

    def verify_sequence_execution(self) -> None:
        for case in CASES:
            self.log(f"[{case.label}] Writing {case.write_val} to {case.in_ch}")
            self.writer.write(case.in_ch, case.write_val)

            self.log(f"[{case.label}] Waiting for {case.out_ch} == {case.expected}")
            self.wait_for_eq(case.out_ch, case.expected)

        self.log("[loop_write] Verifying channel write inside loop body")
        self.wait_for_near("loop_counter", 5.0, tolerance=0.01)

        self.log("[sum_all] Waiting for series sum == 11.4")
        self.wait_for_near("series_sum_all", 11.4, tolerance=0.01)

        self.log("[find_peak] Waiting for peak_idx == 3, peak_val == 9.8")
        self.wait_for_eq("series_peak_idx", 3)
        self.wait_for_near("series_peak_val", 9.8, tolerance=0.01)

        self.log("[index_last] Waiting for data[len(data) - 1] == 4.2")
        self.wait_for_near("series_index_last", 4.2, tolerance=0.01)

        self.log("[index_sum] Waiting for the range index sum == 11.4")
        self.wait_for_near("series_index_sum", 11.4, tolerance=0.01)

        self.log("[index_write] Waiting for the indexed writes == 13.7")
        self.wait_for_near("series_index_write", 13.7, tolerance=0.01)

        self.log("[slice_tail] Waiting for the slice start == 3.7")
        self.wait_for_near("series_slice_tail", 3.7, tolerance=0.01)
