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
from abc import abstractmethod
from collections.abc import Callable

import numpy as np

import synnax as sy
from tests.latency.latency import Latency

PAIRS = 1000
# The gap between the two commands of a pair. It sweeps the window where the state
# loop is writing the state of the first command.
MAX_GAP_NS = 300_000


def _both_equal(a: int, b: int, value: int) -> Callable[[sy.Controller], bool]:
    """Returns a condition that holds once channels a and b both equal value."""
    return lambda c: bool(c[a] == value and c[b] == value)


class DriverStateWake(Latency):
    """Writes two output commands a few hundred microseconds apart, and fails if the
    state of the second takes more than half a state period. A subclass creates the
    write task for its driver.
    """

    state_rate: sy.Rate = 10 * sy.Rate.HZ

    @abstractmethod
    def create_task(
        self, cmds: list[sy.Channel], states: list[sy.Channel]
    ) -> sy.task.StarterStopperMixin:
        """Configures a write task that maps each command channel to the state channel
        at the same position, at this case's state rate.
        """

    def run(self) -> None:
        client = self.client
        cmds: list[sy.Channel] = []
        states: list[sy.Channel] = []
        state_time = client.channels.create(
            name="state_wake_state_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        for i in range(2):
            # Each command has its own index, so one command can be written alone.
            cmd_time = client.channels.create(
                name=f"state_wake_{i}_cmd_time",
                is_index=True,
                data_type=sy.DataType.TIMESTAMP,
                retrieve_if_name_exists=True,
            )
            cmds.append(
                client.channels.create(
                    name=f"state_wake_{i}_cmd",
                    data_type=sy.DataType.UINT8,
                    index=cmd_time.key,
                    retrieve_if_name_exists=True,
                )
            )
            states.append(
                client.channels.create(
                    name=f"state_wake_{i}_state",
                    data_type=sy.DataType.UINT8,
                    index=state_time.key,
                    retrieve_if_name_exists=True,
                )
            )
        tsk = self.create_task(cmds, states)
        first, second = cmds[0].key, cmds[1].key
        first_state, second_state = states[0].key, states[1].key
        slow_ms = self.state_rate.period.milliseconds / 2
        timeout = 2 * self.state_rate.period.seconds + 1
        rng = random.Random(42)
        latencies_ms: list[float] = []
        with tsk.run():
            with client.control.acquire(
                name=f"{type(self).__name__} Test",
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
                        _both_equal(first_state, second_state, value), timeout=timeout
                    )
                    latencies_ms.append(sy.TimeStamp.since(sent).milliseconds)
                ctrl[first] = 0
                ctrl[second] = 0
        times = np.array(latencies_ms)
        slow = times[times > slow_ms]
        self.log(
            f"{len(times)} pairs at {float(self.state_rate):g} Hz: median "
            f"{np.median(times):.2f} ms, p99 {np.percentile(times, 99):.2f} ms, max "
            f"{times.max():.2f} ms"
        )
        if len(slow) > 0:
            listed = " ".join(f"{t:.2f}" for t in slow)
            self.fail(
                f"{len(slow)} of {len(times)} second-line states over {slow_ms:g} ms: "
                f"{listed}"
            )


class DriverNIStateWake(DriverStateWake):
    """Runs the state wake check on two NI DO lines."""

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.skip("Windows DAQmx drivers required")
        super().setup()

    def create_task(
        self, cmds: list[sy.Channel], states: list[sy.Channel]
    ) -> sy.ni.DigitalWriteTask:
        # Sim device must be set up in NI MAX
        dev = self.client.devices.retrieve(location="SYMod1", one=True)
        return self.client.tasks.configure(
            sy.ni.DigitalWriteTask(
                name=type(self).__name__,
                device=dev.key,
                state_rate=self.state_rate,
                channels=[
                    sy.ni.DOChannel(
                        cmd_channel=cmd.key,
                        state_channel=state.key,
                        port=0,
                        line=line,
                    )
                    for line, (cmd, state) in enumerate(zip(cmds, states, strict=True))
                ],
            )
        )


class DriverNIStateWake1Hz(DriverNIStateWake):
    state_rate = 1 * sy.Rate.HZ


class DriverNIStateWake100Hz(DriverNIStateWake):
    state_rate = 100 * sy.Rate.HZ
