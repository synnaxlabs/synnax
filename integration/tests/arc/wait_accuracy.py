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
from tests.arc.timing import Limits, limits, runtime

# The waits in ms the sequence holds in order. The case skips each wait under the
# minimum of the runtime.
WAITS_MS = [30, 20, 10, 5, 1]


def create_source(waits_ms: list[int]) -> str:
    """Returns a sequence that writes a count to ``wa_cmd`` before and after each wait.

    :param waits_ms: The waits the sequence holds, in order.
    """
    steps = ["    1 -> wa_cmd"]
    for i, wait_ms in enumerate(waits_ms):
        steps.append(f"    time.wait{{{wait_ms}ms}}")
        steps.append(f"    {i + 2} -> wa_cmd")
    body = ",\n".join(steps)
    return f"""
import time
authority 200

sequence main {{
{body}
}}

wa_start_cmd => main
"""


class WaitAccuracy(ArcCase):
    """A sequence writes ``wa_cmd`` before and after each wait of ``WAITS_MS`` that the
    runtime supports. The time between two writes is the wait the runtime held, read
    from the timestamps the runtime stamped.
    """

    arc_name_prefix = "ArcWaitAccuracy"
    start_cmd_channel = "wa_start_cmd"
    subscribe_channels = ["wa_cmd"]
    start: sy.TimeStamp
    waits_ms: list[int]
    limits: Limits

    def setup(self) -> None:
        self._retrieve_rack()
        assert self.rack is not None
        self.limits = limits(self.rack)
        min_ms = self.limits.min_wait_ms
        self.waits_ms = [w for w in WAITS_MS if w >= min_ms]
        skipped = ", ".join(f"{w} ms" for w in WAITS_MS if w < min_ms) or "none"
        self.log(f"{runtime(self.rack)} minimum wait: {min_ms} ms, skipped: {skipped}")
        self.arc_source = create_source(self.waits_ms)
        create_indexed_pair(self.client, "wa_cmd", sy.DataType.UINT8)
        self.start = sy.TimeStamp.now()
        super().setup()

    def verify_sequence_execution(self) -> None:
        writes = list(range(1, len(self.waits_ms) + 2))
        # Program init
        self.wait_for_eq("wa_cmd", writes[0])
        # Program done
        self.wait_for_eq("wa_cmd", writes[-1])
        frame = self.client.read(
            sy.TimeRange(self.start, sy.TimeStamp.now()), ["wa_cmd_time", "wa_cmd"]
        )
        times = frame["wa_cmd_time"].to_numpy().astype(np.int64)
        values = frame["wa_cmd"].to_numpy().tolist()
        if values != writes:
            self.fail(f"wa_cmd holds {values}, expected {writes}")
            return
        held_ms = np.diff(times) / float(sy.TimeSpan.MILLISECOND)
        over: list[str] = []
        self.log(f"{'wait':<6}  {'held':>10}  {'error':>7}")
        for held, wait_ms in zip(held_ms.tolist(), self.waits_ms):
            error = (held - wait_ms) / wait_ms * 100
            # One sample can be off by the spread on top of the median error.
            limit = (
                self.limits.max_error_percent
                + self.limits.max_spread_ms / wait_ms * 100
            )
            flag = ""
            if abs(error) > limit:
                flag = f"  over the {limit:.1f}% limit"
                over.append(f"{wait_ms} ms ({error:+.1f}%, limit {limit:.1f}%)")
            wait = f"{wait_ms} ms"
            self.log(f"{wait:<6}  {held:>7.3f} ms  {error:>+6.1f}%{flag}")
        if over:
            self.fail("over the limit: " + ", ".join(over))
