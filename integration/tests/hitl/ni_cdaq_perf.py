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

import numpy as np

import synnax as sy
from tests.hitl.ni_cdaq import CDAQDigitalWrite

_COMMANDS = 500
# The first commands are slow while the Core opens its relay to the Driver.
_WARMUP = 20
_STATE_TIMEOUT_S = 5
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
                while True:
                    frame = streamer.read(timeout=_STATE_TIMEOUT_S)
                    if frame is None:
                        self.fail(f"No state {value} in {_STATE_TIMEOUT_S} s")
                    if frame[_STATE].to_numpy()[-1] == value:
                        break
                latencies_ms.append((time.perf_counter() - start) * 1000)
        measured = np.array(latencies_ms[_WARMUP:])
        self.metrics["command_to_state_median_ms"] = float(np.median(measured))
        self.metrics["command_to_state_p99_ms"] = float(np.percentile(measured, 99))
        self.metrics["command_to_state_max_ms"] = float(measured.max())
        self.log(f"command to state in ms: {self.metrics}")
