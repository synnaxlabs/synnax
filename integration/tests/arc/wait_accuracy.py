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

# TODO Add a case with a variable wait duration to quantify its lag on Windows.
# TODO Add error variance check

# TODO needs more pressure, samples, etc
ARC_WAIT_ACCURACY_SOURCE = """
import time
authority 200

sequence main {
    
    1 -> wa_cmd,
    time.wait{30ms},
    2 -> wa_cmd,
    time.wait{20ms},
    3 -> wa_cmd,
    time.wait{10ms},
    4 -> wa_cmd,
    time.wait{5ms},
    5 -> wa_cmd,
    time.wait{1ms},
    6 -> wa_cmd
}

wa_start_cmd => main
"""

# The values the sequence writes to wa_cmd in order.
WRITES = [1, 2, 3, 4, 5, 6]
# The wait in ms between each write and the next.
WAITS_MS = [30, 20, 10, 5, 1]
MAX_ERROR_PERCENT = 10.0


class WaitAccuracy(ArcCase):
    """A sequence writes ``wa_cmd`` before and after 30, 20, 10, 5, and 1 ms waits. The
    time between two writes is the wait the runtime held, read from the timestamps
    the runtime stamped.
    """

    arc_source = ARC_WAIT_ACCURACY_SOURCE
    arc_name_prefix = "ArcWaitAccuracy"
    start_cmd_channel = "wa_start_cmd"
    subscribe_channels = ["wa_cmd"]
    start: sy.TimeStamp

    def setup(self) -> None:
        create_indexed_pair(self.client, "wa_cmd", sy.DataType.UINT8)
        self.start = sy.TimeStamp.now()
        super().setup()

    def verify_sequence_execution(self) -> None:
        # Program init
        self.wait_for_eq("wa_cmd", WRITES[0])
        # Program done
        self.wait_for_eq("wa_cmd", WRITES[-1])
        frame = self.client.read(
            sy.TimeRange(self.start, sy.TimeStamp.now()), ["wa_cmd_time", "wa_cmd"]
        )
        times = frame["wa_cmd_time"].to_numpy().astype(np.int64)
        values = frame["wa_cmd"].to_numpy().tolist()
        if values != WRITES:
            self.fail(f"wa_cmd holds {values}, expected {WRITES}")
            return
        held_ms = np.diff(times) / float(sy.TimeSpan.MILLISECOND)
        over: list[str] = []
        self.log(f"{'wait':<6}  {'held':>10}  {'error':>7}")
        for held, wait_ms in zip(held_ms.tolist(), WAITS_MS):
            error = (held - wait_ms) / wait_ms * 100
            flag = ""
            if abs(error) > MAX_ERROR_PERCENT:
                flag = "  over limit"
                over.append(f"{wait_ms} ms ({error:+.1f}%)")
            wait = f"{wait_ms} ms"
            self.log(f"{wait:<6}  {held:>7.3f} ms  {error:>+6.1f}%{flag}")
        if over:
            self.fail(f"over the {MAX_ERROR_PERCENT:g}% limit: " + ", ".join(over))
