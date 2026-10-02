#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np

import synnax as sy
from framework.utils import create_indexed_pair
from tests.arc.arc import ArcCase

# The waits of each profile in ms. Their GCD selects the loop mode of the C++ runtime.
PROFILES: dict[str, list[int]] = {
    "event_driven": [10, 20, 30],
    "hybrid": [12, 20, 32],
    "rt_event": [10, 20, 31],
}
# How many times the sequence holds each wait of a profile.
REPEATS = 20
MAX_ERROR_PERCENT = 10.0


def create_source(waits_ms: list[int]) -> str:
    """Returns a sequence that writes a count to ``ws_cmd`` before and after each wait.

    :param waits_ms: The waits the sequence holds, in order.
    """
    steps = ["    1 -> ws_cmd"]
    for i, wait_ms in enumerate(waits_ms):
        steps.append(f"    time.wait{{{wait_ms}ms}}")
        steps.append(f"    {i + 2} -> ws_cmd")
    body = ",\n".join(steps)
    return f"""
import time
authority 200

sequence main {{
{body}
}}

ws_start_cmd => main
"""


class WaitAccuracySamples(ArcCase):
    """A sequence holds each wait of a profile ``REPEATS`` times and writes ``ws_cmd``
    before and after each one. The case logs every held time and fails if the median
    of a wait is over the limit.
    """

    arc_name_prefix = "ArcWaitAccuracySamples"
    start_cmd_channel = "ws_start_cmd"
    subscribe_channels = ["ws_cmd"]
    start: sy.TimeStamp
    waits_ms: list[int]

    def setup(self) -> None:
        self.waits_ms = PROFILES[self.params["profile"]] * REPEATS
        self.arc_source = create_source(self.waits_ms)
        create_indexed_pair(self.client, "ws_cmd", sy.DataType.UINT8)
        self.start = sy.TimeStamp.now()
        super().setup()

    def verify_sequence_execution(self) -> None:
        writes = list(range(1, len(self.waits_ms) + 2))
        self.wait_for_eq("ws_cmd", writes[-1], timeout=30 * sy.TimeSpan.SECOND)
        frame = self.client.read(
            sy.TimeRange(self.start, sy.TimeStamp.now()), ["ws_cmd_time", "ws_cmd"]
        )
        times = frame["ws_cmd_time"].to_numpy().astype(np.int64)
        values = frame["ws_cmd"].to_numpy().tolist()
        if values != writes:
            self.fail(f"ws_cmd holds {values}, expected {writes}")
            return
        held_ms = np.diff(times) / float(sy.TimeSpan.MILLISECOND)
        waits_ms = np.array(self.waits_ms)
        failures: list[str] = []
        for wait_ms in sorted(set(self.waits_ms)):
            held = held_ms[waits_ms == wait_ms]
            median = float(np.median(held))
            error = (median - wait_ms) / wait_ms * 100
            report = (
                f"wait{{{wait_ms}ms}} held {median:.3f} ms median, error {error:+.1f}%,"
                f" min {held.min():.3f}, p90 {np.percentile(held, 90):.3f},"
                f" max {held.max():.3f}"
            )
            self.log(report)
            samples = " ".join(f"{h:.3f}" for h in held.tolist())
            self.log(f"wait{{{wait_ms}ms}} samples: {samples}")
            if abs(error) > MAX_ERROR_PERCENT:
                failures.append(report)
        if failures:
            self.fail(f"over the {MAX_ERROR_PERCENT}% limit: " + "; ".join(failures))
