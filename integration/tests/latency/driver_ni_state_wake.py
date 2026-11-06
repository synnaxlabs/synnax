#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import platform
import random
import time
from collections.abc import Callable

import numpy as np

import synnax as sy
from tests.latency.latency import Latency

STATE_RATE = 10 * sy.Rate.HZ
PAIRS = 200
# The gap between the two commands of a pair. It sweeps the window where the state
# loop is writing the state of the first command.
MAX_GAP_NS = 300_000
# A state update the state loop missed waits out the rest of its 100 ms period.
SLOW_MS = 50.0


def _both_equal(a: int, b: int, value: int) -> Callable[[sy.Controller], bool]:
    """Returns a condition that holds once channels a and b both equal value."""
    return lambda c: bool(c[a] == value and c[b] == value)


class DriverNIStateWake(Latency):
    """Writes two NI DO lines a few hundred microseconds apart at a 10 Hz state rate,
    and fails if the state of the second line takes more than half a state period.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.skip("Windows DAQmx drivers required")
        super().setup()

    def run(self) -> None:
        client = self.client
        # Sim device must be set up in NI MAX
        dev = client.devices.retrieve(location="SYMod1")
        # Each command gets its own index, so one command can be written alone.
        cmd_times = [
            client.channels.create(
                name=f"dsw_{line}_cmd_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            for line in range(2)
        ]
        state_time = client.channels.create(
            name="dsw_state_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        cmds: list[sy.Channel] = []
        states: list[sy.Channel] = []
        for line in range(2):
            cmds.append(
                client.channels.create(
                    name=f"dsw_{line}_cmd",
                    data_type=sy.DataType.UINT8,
                    index=cmd_times[line].key,
                    retrieve_if_name_exists=True,
                )
            )
            states.append(
                client.channels.create(
                    name=f"dsw_{line}_state",
                    data_type=sy.DataType.UINT8,
                    index=state_time.key,
                    retrieve_if_name_exists=True,
                )
            )
        tsk = client.tasks.configure(
            sy.ni.DigitalWriteTask(
                name="NI State Wake",
                device=dev.key,
                state_rate=STATE_RATE,
                channels=[
                    sy.ni.DOChannel(
                        cmd_channel=cmds[line].key,
                        state_channel=states[line].key,
                        port=0,
                        line=line,
                    )
                    for line in range(2)
                ],
            )
        )
        first, second = cmds[0].key, cmds[1].key
        first_state, second_state = states[0].key, states[1].key
        rng = random.Random(42)
        latencies_ms: list[float] = []
        with tsk.run():
            with client.control.acquire(
                name="NI State Wake Test",
                read=[first_state, second_state],
                write=[first, second],
                write_authorities=sy.Authority.ABSOLUTE,
            ) as ctrl:
                value = 0
                for _ in range(PAIRS):
                    value = 1 - value
                    ctrl[first] = value
                    until = time.perf_counter_ns() + rng.randrange(MAX_GAP_NS)
                    while time.perf_counter_ns() < until:
                        pass
                    ctrl[second] = value
                    sent = sy.TimeStamp.now()
                    ctrl.wait_until(
                        _both_equal(first_state, second_state, value), timeout=2
                    )
                    latencies_ms.append(sy.TimeStamp.since(sent).microseconds / 1000)
                ctrl[first] = 0
                ctrl[second] = 0
        times = np.array(latencies_ms)
        slow = times[times > SLOW_MS]
        self.log(
            f"{len(times)} pairs: median {np.median(times):.2f} ms, p99 "
            f"{np.percentile(times, 99):.2f} ms, max {times.max():.2f} ms"
        )
        if len(slow) > 0:
            listed = " ".join(f"{t:.2f}" for t in slow)
            self.fail(
                f"{len(slow)} of {len(times)} second-line states over {SLOW_MS} ms: "
                f"{listed}"
            )
