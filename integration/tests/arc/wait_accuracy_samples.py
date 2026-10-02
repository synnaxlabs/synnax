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
from tests.arc.timing import Limits, limits

# How many times the sequence holds each wait of a case.
REPEATS = 20


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


class _WaitSamples(ArcCase):
    """A sequence holds each wait of ``profile_ms`` ``REPEATS`` times and writes
    ``ws_cmd`` before and after each one. The case fails if the median error of a wait
    or the spread of its held times is over its limit. The case skips each wait under
    the shortest wait its limits cover.
    """

    #: The waits in ms. The shortest one selects the loop mode of the C++ runtime.
    profile_ms: list[int]

    arc_name_prefix = "ArcWaitAccuracySamples"
    start_cmd_channel = "ws_start_cmd"
    subscribe_channels = ["ws_cmd"]
    start: sy.TimeStamp
    waits_ms: list[int]
    limits: Limits

    def setup(self) -> None:
        self._retrieve_rack()
        assert self.rack is not None
        self.limits = limits(self.rack)
        min_ms = self.limits.min_wait_ms
        profile = [w for w in self.profile_ms if w >= min_ms]
        if not profile:
            self.auto_pass(f"every wait of the case is under {min_ms} ms")
        self.waits_ms = profile * REPEATS
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
        over: list[str] = []
        wide: list[str] = []
        self.log(f"held time in ms, {REPEATS} samples per wait")
        self.log(
            f"{'wait':<6}  {'median':>7}  {'error':>7}"
            f"  {'min':>7}  {'p90':>7}  {'max':>7}  {'spread':>7}"
        )
        for wait_ms in sorted(set(self.waits_ms)):
            held = held_ms[waits_ms == wait_ms]
            median = float(np.median(held))
            error = (median - wait_ms) / wait_ms * 100
            spread = float(held.max() - held.min())
            flag = ""
            if abs(error) > self.limits.max_error_percent:
                flag += "  error over limit"
                over.append(f"{wait_ms} ms ({error:+.1f}%)")
            if spread > self.limits.max_spread_ms:
                flag += "  spread over limit"
                wide.append(f"{wait_ms} ms ({spread:.3f} ms)")
            wait = f"{wait_ms} ms"
            self.log(
                f"{wait:<6}  {median:>7.3f}  {error:>+6.1f}%  {held.min():>7.3f}"
                f"  {np.percentile(held, 90):>7.3f}  {held.max():>7.3f}"
                f"  {spread:>7.3f}{flag}"
            )
        failures: list[str] = []
        if over:
            failures.append(
                f"median error over the {self.limits.max_error_percent:g}% limit: "
                + ", ".join(over)
            )
        if wide:
            failures.append(
                f"spread over the {self.limits.max_spread_ms:g} ms limit: "
                + ", ".join(wide)
            )
        if failures:
            self.fail("; ".join(failures))


class WaitSamplesEventDriven(_WaitSamples):
    profile_ms = [10, 20, 30]


class WaitSamplesHybrid(_WaitSamples):
    profile_ms = [12, 20, 32]


class WaitSamplesRTEvent(_WaitSamples):
    profile_ms = [10, 20, 31]


class WaitSamplesAll(_WaitSamples):
    profile_ms = [30, 20, 10, 5, 1]


class WaitSamplesOnly20(_WaitSamples):
    profile_ms = [20]


class WaitSamplesOnly10(_WaitSamples):
    profile_ms = [10]


class WaitSamplesOnly5(_WaitSamples):
    profile_ms = [5]


class WaitSamplesOnly1(_WaitSamples):
    profile_ms = [1]
