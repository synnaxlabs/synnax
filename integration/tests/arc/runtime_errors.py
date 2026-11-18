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

ARC_RUNTIME_ERRORS_SOURCE = """
func divide() {d i32} i32 {
    return 100 / d
}

// A slice that starts at 2 holds one value, so the addition has two lengths.
func add_slice() {n i32} f64 {
    a := [1.0, 2.0, 3.0]
    b := [10.0, 20.0]
    hi := n + 2
    c := a[n:hi] + b
    return c[0]
}

rt_err_div_in -> divide() -> rt_err_div_out
rt_err_len_in -> add_slice() -> rt_err_len_out
"""


@dataclass
class ErrorCase:
    label: str
    in_ch: str
    out_ch: str
    good: int
    good_out: float
    bad: int
    # Text of the status that the bad input gives.
    description: str
    recovered: int
    recovered_out: float


CASES = [
    ErrorCase(
        "DivideByZero",
        "rt_err_div_in",
        "rt_err_div_out",
        5,
        20,
        0,
        "integer divide by zero",
        4,
        25,
    ),
    ErrorCase(
        "SeriesLengthMismatch",
        "rt_err_len_in",
        "rt_err_len_out",
        0,
        11,
        2,
        "series length mismatch in addition",
        1,
        12,
    ),
]


class RuntimeErrors(ArcCase):
    """A node that fails gives no output for that cycle, and the task reports it. The
    task keeps running, and the node gives output again on the next good input."""

    arc_source = ARC_RUNTIME_ERRORS_SOURCE
    arc_name_prefix = "ArcRuntimeErrors"
    start_cmd_channel = "start_runtime_errors_cmd"
    subscribe_channels = ["rt_err_div_out", "rt_err_len_out"]
    collect_task_status = True

    def setup(self) -> None:
        for ch in ("rt_err_div_in", "rt_err_len_in"):
            create_virtual_channel(self.client, ch, data_type=sy.DataType.INT32)
        create_indexed_pair(self.client, "rt_err_div_out", data_type=sy.DataType.INT32)
        create_indexed_pair(self.client, "rt_err_len_out")
        super().setup()

    def verify_sequence_execution(self) -> None:
        key = self.task_key(self.arc_name)
        for case in CASES:
            self.log(f"[{case.label}] Writing {case.good} to {case.in_ch}")
            self.writer.write(case.in_ch, case.good)
            self.wait_for_eq(case.out_ch, case.good_out)
            self.log(f"[{case.label}] Writing {case.bad} to {case.in_ch}")
            self.writer.write(case.in_ch, case.bad)
            if not self.wait_for_task_status(key, case.description):
                self.fail(f"[{case.label}] no status surfaced for {case.description}")
            actual = self.read_tlm(case.out_ch)
            if actual != case.good_out:
                self.fail(
                    f"[{case.label}] {case.out_ch} changed from {case.good_out} "
                    f"to {actual}"
                )
            if not self.task_is_running(key):
                self.fail(f"[{case.label}] the task stopped")
            self.log(f"[{case.label}] Writing {case.recovered} to {case.in_ch}")
            self.writer.write(case.in_ch, case.recovered)
            self.wait_for_eq(case.out_ch, case.recovered_out)
