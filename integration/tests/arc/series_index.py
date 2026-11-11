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

ARC_SERIES_INDEX_SOURCE = """
func read_at() {i i32} f64 {
    data := [1.5, 2.5, 3.5]
    return data[i]
}

func write_at() {i i32} f64 {
    data := [1.5, 2.5, 3.5]
    data[i] = 9.5
    return data[0] + data[1] + data[2]
}

func flag_at() {i i32} u8 {
    mask series bool := [false, true, true]
    hit := mask[i]
    if hit {
        return 1
    }
    return 2
}

func read_wide() {i i64} f64 {
    data := [1.5, 2.5, 3.5]
    return data[i]
}

// An unsigned counter that goes below zero wraps to the maximum of its type.
func read_u8() {i u8} f64 {
    data := [1.5, 2.5, 3.5]
    return data[i - 1]
}

func read_u32() {i u32} f64 {
    data := [1.5, 2.5, 3.5]
    return data[i - 1]
}

func read_u64() {i u64} f64 {
    data := [1.5, 2.5, 3.5]
    return data[i - 1]
}

series_read_idx -> read_at() -> series_read_out
series_u8_idx -> read_u8() -> series_u8_out
series_u32_idx -> read_u32() -> series_u32_out
series_u64_idx -> read_u64() -> series_u64_out
series_wide_idx -> read_wide() -> series_wide_out
series_write_idx -> write_at() -> series_write_out
series_flag_idx -> flag_at() -> series_flag_out
"""


@dataclass
class IndexCase:
    label: str
    in_ch: str
    out_ch: str
    index: int
    # None for an index outside the series.
    expected: float | None
    # The index the task reports, when it is not the value written.
    reported: int | None = None


# Each series holds 3 values. Consecutive expectations on one output differ, so each
# wait proves a new write landed. Each out-of-range index is unique, so its status
# traces to one case. The out-of-range cases come last because a Driver stops the task
# on the first one.
CASES = [
    IndexCase("ReadFirst", "series_read_idx", "series_read_out", 0, 1.5),
    IndexCase("ReadLast", "series_read_idx", "series_read_out", -1, 3.5),
    IndexCase("ReadSecondToLast", "series_read_idx", "series_read_out", -2, 2.5),
    IndexCase("ReadNegativeFirst", "series_read_idx", "series_read_out", -3, 1.5),
    IndexCase("ReadMiddle", "series_read_idx", "series_read_out", 1, 2.5),
    IndexCase("WriteFirst", "series_write_idx", "series_write_out", 0, 15.5),
    IndexCase("WriteLast", "series_write_idx", "series_write_out", -1, 13.5),
    IndexCase("WriteMiddle", "series_write_idx", "series_write_out", 1, 14.5),
    IndexCase("WriteNegativeFirst", "series_write_idx", "series_write_out", -3, 15.5),
    IndexCase("FlagFirst", "series_flag_idx", "series_flag_out", 0, 2),
    IndexCase("FlagLast", "series_flag_idx", "series_flag_out", -1, 1),
    IndexCase("FlagNegativeFirst", "series_flag_idx", "series_flag_out", -3, 2),
    IndexCase("FlagMiddle", "series_flag_idx", "series_flag_out", 1, 1),
    IndexCase("WideFirst", "series_wide_idx", "series_wide_out", 0, 1.5),
    IndexCase("WideLast", "series_wide_idx", "series_wide_out", -1, 3.5),
    IndexCase("U8First", "series_u8_idx", "series_u8_out", 1, 1.5),
    IndexCase("U8Last", "series_u8_idx", "series_u8_out", 3, 3.5),
    IndexCase("U32First", "series_u32_idx", "series_u32_out", 1, 1.5),
    IndexCase("U32Last", "series_u32_idx", "series_u32_out", 3, 3.5),
    IndexCase("U64First", "series_u64_idx", "series_u64_out", 1, 1.5),
    IndexCase("U64Last", "series_u64_idx", "series_u64_out", 3, 3.5),
    IndexCase("ReadPastEnd", "series_read_idx", "series_read_out", 3, None),
    IndexCase("ReadBeforeStart", "series_read_idx", "series_read_out", -4, None),
    IndexCase("WritePastEnd", "series_write_idx", "series_write_out", 4, None),
    IndexCase("WriteBeforeStart", "series_write_idx", "series_write_out", -5, None),
    IndexCase("FlagPastEnd", "series_flag_idx", "series_flag_out", 5, None),
    IndexCase("FlagBeforeStart", "series_flag_idx", "series_flag_out", -6, None),
    IndexCase("WidePastEnd", "series_wide_idx", "series_wide_out", 6, None),
    IndexCase("WideAboveI32", "series_wide_idx", "series_wide_out", 2**32 + 1, None),
    IndexCase("WideBelowI32", "series_wide_idx", "series_wide_out", -(2**32) - 1, None),
    IndexCase("U8BelowZero", "series_u8_idx", "series_u8_out", 0, None, 2**8 - 1),
    IndexCase("U32BelowZero", "series_u32_idx", "series_u32_out", 0, None, 2**32 - 1),
    IndexCase("U64BelowZero", "series_u64_idx", "series_u64_out", 0, None, 2**64 - 1),
]

OUTPUTS = [
    "series_read_out",
    "series_write_out",
    "series_flag_out",
    "series_wide_out",
    "series_u8_out",
    "series_u32_out",
    "series_u64_out",
]


class SeriesIndex(ArcCase):
    """Test series indexes on a read and on a write. A negative index counts from
    the end. An index outside the series gives no output, and the task reports it."""

    arc_source = ARC_SERIES_INDEX_SOURCE
    arc_name_prefix = "ArcSeriesIndex"
    start_cmd_channel = "start_series_index_cmd"
    subscribe_channels = OUTPUTS
    collect_task_status = True

    def setup(self) -> None:
        for ch in ("series_read_idx", "series_write_idx", "series_flag_idx"):
            create_virtual_channel(self.client, ch, data_type=sy.DataType.INT32)
        create_virtual_channel(
            self.client, "series_wide_idx", data_type=sy.DataType.INT64
        )
        create_indexed_pair(self.client, "series_read_out")
        create_indexed_pair(self.client, "series_wide_out")
        for name, data_type in (
            ("u8", sy.DataType.UINT8),
            ("u32", sy.DataType.UINT32),
            ("u64", sy.DataType.UINT64),
        ):
            create_virtual_channel(
                self.client, f"series_{name}_idx", data_type=data_type
            )
            create_indexed_pair(self.client, f"series_{name}_out")
        create_indexed_pair(self.client, "series_write_out")
        create_indexed_pair(self.client, "series_flag_out", data_type=sy.DataType.UINT8)
        super().setup()

    def _expect_out_of_bounds(self, case: IndexCase, last: float | None) -> None:
        """Fail unless the task reported the index and the output kept its last
        value."""
        index = case.index if case.reported is None else case.reported
        description = f"index {index} out of bounds for series of size 3"
        if not -(2**31) <= index < 2**31:
            description = f"index {index} out of bounds for an i32 series index"
        key = self.task_key(self.arc_name)
        if not self.wait_for_task_status(key, description):
            self.fail(f"[{case.label}] no status surfaced for index {index}")
        actual = self.read_tlm(case.out_ch)
        if actual != last:
            self.fail(f"[{case.label}] {case.out_ch} changed from {last} to {actual}")

    def verify_sequence_execution(self) -> None:
        key = self.task_key(self.arc_name)
        last: dict[str, float | None] = dict.fromkeys(OUTPUTS)
        for case in CASES:
            self.log(f"[{case.label}] Writing {case.index} to {case.in_ch}")
            self.writer.write(case.in_ch, case.index)
            if case.expected is not None:
                self.wait_for_eq(case.out_ch, case.expected)
                last[case.out_ch] = case.expected
                continue
            self._expect_out_of_bounds(case, last[case.out_ch])
            # A Driver stops the task on a runtime error. The Core keeps it running.
            if not self.task_is_running(key):
                return
