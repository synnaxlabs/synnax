#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Performance of the Driver with the EtherCAT bus of the HITL station."""

import time

from tests.hitl import latency
from tests.hitl.ethercat import EtherCATRead

_STARTS = 10
# The station measures about 1.3 s. With ten starts, p99 is about the slowest start, so
# its limit allows one start in which the Driver tries the bus a second time.
_MAX_MEDIAN_MS = 2_000
_MAX_P99_MS = 4_000


class EtherCATStartTime(EtherCATRead):
    """Measures the time from a start command to the Driver's acknowledgment, for
    starts after a stop. Each of these starts scans the bus again."""

    task_name = "HITL EtherCAT Start Time"

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Task not configured")
        # The first start can use the scan that found the slaves.
        self.tsk.start(timeout=self.COMMAND_TIMEOUT)
        self.tsk.stop(timeout=self.COMMAND_TIMEOUT)
        starts_ms: list[float] = []
        for _ in range(_STARTS):
            start = time.perf_counter()
            self.tsk.start(timeout=self.COMMAND_TIMEOUT)
            starts_ms.append((time.perf_counter() - start) * 1000)
            self.tsk.stop(timeout=self.COMMAND_TIMEOUT)
        latency.check(
            self,
            "start_after_stop",
            starts_ms,
            max_median_ms=_MAX_MEDIAN_MS,
            max_p99_ms=_MAX_P99_MS,
        )
