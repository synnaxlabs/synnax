#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Performance of the Driver with the real cDAQ-9185 (SY2) of the HITL station."""

import time
from functools import partial
from operator import eq

import synnax as sy
from tests.hitl import latency
from tests.hitl.ni_cdaq import CDAQDigitalWrite

_COMMANDS = 500
# The first commands are slow while the Core opens its relay to the Driver.
_WARMUP = 20
_STATE_TIMEOUT_S = 5
# About three times what the station measures.
_MAX_MEDIAN_MS = 6
_MAX_P99_MS = 10
_CMD = "hitl_cdaq_do_cmd_0"
_STATE = "hitl_cdaq_do_state_0"


class CDAQDigitalWriteLatency(CDAQDigitalWrite):
    """Measures the time from a command of this machine's client to the state that the
    Driver sends once it writes the output of the NI 9375."""

    task_name = "HITL cDAQ Digital Write Latency"

    def run(self) -> None:
        if self.tsk is None:
            self.fail("Task not configured")
        latencies_ms: list[float] = []
        with (
            self.tsk.run(),
            self.client.open_streamer(_STATE) as streamer,
            self.client.open_writer(
                sy.TimeStamp.now(),
                [_CMD, "hitl_cdaq_do_cmd_time"],
                enable_auto_commit=True,
            ) as writer,
        ):
            for i in range(_WARMUP + _COMMANDS):
                value = 1 - i % 2
                start = time.perf_counter()
                writer.write({_CMD: value, "hitl_cdaq_do_cmd_time": sy.TimeStamp.now()})
                if not latency.wait_for(
                    streamer, _STATE, partial(eq, value), timeout_s=_STATE_TIMEOUT_S
                ):
                    self.fail(f"No state {value} in {_STATE_TIMEOUT_S} s")
                latencies_ms.append((time.perf_counter() - start) * 1000)
        latency.check(
            self,
            "command_to_state",
            latencies_ms[_WARMUP:],
            max_median_ms=_MAX_MEDIAN_MS,
            max_p99_ms=_MAX_P99_MS,
        )
