#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""LabJack task tests on the T4 of the HITL station, on USB.

Port         | Wiring
-------------|--------------------------------------------------
DAC0         | AIN0
AIN1 to AIN3 | Nothing. A floating input reads about 1.4 V.
FIO4 to FIO7 | Nothing. A pull-up holds a floating line at 1.
"""

from abc import abstractmethod
from collections.abc import Callable

import numpy as np

import synnax as sy
from tests.driver.task import (
    ReadTaskCase,
    WriteTaskCase,
    create_channel,
    create_index,
    delete_channels,
)

SERIAL = "440022190"
"""The serial number of the T4, which the Driver uses as the key of its device."""

# The Driver scans USB for LabJacks every few seconds.
_PRESENT_TIMEOUT = 30 * sy.TimeSpan.SECOND
# Each level differs from the levels that an earlier test leaves on DAC0, and from the
# voltage of a floating input, so AIN0 reads a level only after DAC0 really changes.
_LEVELS = [4.5, 0.5, 2.5]
# Wide over the 20 mV that the station measures, and far under the volts between levels.
_LEVEL_TOLERANCE_V = 0.05
_LEVEL_TIMEOUT = 5 * sy.TimeSpan.SECOND


def retrieve_t4(client: sy.Synnax) -> sy.Device:
    """Returns the device of the T4 once the Driver reports it as present.

    :raises TimeoutError: If the Driver does not report the T4 within 30 seconds.
    """

    def present() -> sy.Device | None:
        device = client.devices.retrieve(
            key=SERIAL, include_status=True, ignore_not_found=True
        )
        if device is None or device.status is None:
            return None
        return device if device.status.variant == "success" else None

    device = sy.poll(present, timeout=_PRESENT_TIMEOUT, interval=0.5)
    if device is None:
        raise TimeoutError(f"The Driver did not report the T4 {SERIAL} as present")
    return device


class _T4Read(ReadTaskCase[sy.labjack.ReadTask]):
    """Reads the channels of create_channels from the T4."""

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.labjack.ReadChannel]: ...

    def retrieve_device(self) -> sy.Device:
        return retrieve_t4(self.client)

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.labjack.ReadTask:
        return sy.labjack.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            channels=self.create_channels(self.client),
        )

    def _channel_keys(self, task: sy.labjack.ReadTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]


class _T4Write(WriteTaskCase[sy.labjack.WriteTask]):
    """Writes the channels of create_channels to the T4."""

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.labjack.WriteChannel]: ...

    def retrieve_device(self) -> sy.Device:
        return retrieve_t4(self.client)

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.labjack.WriteTask:
        return sy.labjack.WriteTask(
            name=task_name,
            device=device.key,
            state_rate=sample_rate,
            channels=self.create_channels(self.client),
        )

    def _channel_keys(self, task: sy.labjack.WriteTask) -> list[int]:
        return [ch.cmd_channel for ch in task.config.channels]

    def _state_keys(self, task: sy.labjack.WriteTask) -> list[int]:
        return [ch.state_channel for ch in task.config.channels]


class LabJackAnalogRead(_T4Read):
    """Streams the four high-voltage inputs of the T4."""

    task_name = "HITL LabJack Analog Read"
    SAMPLE_RATE = 1000 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.labjack.ReadChannel]:
        idx = create_index(client, "hitl_labjack_ai_time")
        return [
            sy.labjack.AnalogReadChannel(
                port=f"AIN{i}",
                channel=create_channel(
                    client,
                    name=f"hitl_labjack_ai_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                range=10.0,
            )
            for i in range(4)
        ]


class LabJackAnalogLoopback(LabJackAnalogRead):
    """Writes levels to DAC0, and checks that AIN0 reads each one and that no other
    input does. A swap or a repeat of channels in either task fails the test."""

    task_name = "HITL LabJack Analog Loopback"

    def run(self) -> None:
        assert self.tsk is not None
        inputs = self._channel_keys(self.tsk)
        cmd_idx = create_index(self.client, "hitl_labjack_loopback_cmd_time")
        state_idx = create_index(self.client, "hitl_labjack_loopback_state_time")
        output = sy.labjack.AnalogWriteChannel(
            port="DAC0",
            cmd_channel=create_channel(
                self.client,
                name="hitl_labjack_loopback_cmd",
                data_type=sy.DataType.FLOAT64,
                index=cmd_idx.key,
            ),
            state_channel=create_channel(
                self.client,
                name="hitl_labjack_loopback_state",
                data_type=sy.DataType.FLOAT64,
                index=state_idx.key,
            ),
        )
        write = sy.labjack.WriteTask(
            name="HITL LabJack Analog Loopback Write",
            device=SERIAL,
            channels=[output],
        )
        self.client.tasks.configure(write)
        try:
            with (
                self.tsk.run(timeout=self.COMMAND_TIMEOUT),
                write.run(timeout=self.COMMAND_TIMEOUT),
                self.client.open_streamer(inputs) as streamer,
                self.client.open_writer(
                    sy.TimeStamp.now(),
                    [output.cmd_channel, cmd_idx.key],
                    enable_auto_commit=True,
                ) as writer,
            ):

                def command(level: float) -> None:
                    writer.write(
                        {output.cmd_channel: level, cmd_idx.key: sy.TimeStamp.now()}
                    )

                self.drive(streamer, inputs, command)
        finally:
            # The Core refuses to delete a channel that a task uses.
            self.client.tasks.delete(write.key)
            delete_channels(self.client, [output.cmd_channel, output.state_channel])

    def drive(
        self,
        streamer: sy.Streamer,
        inputs: list[int],
        command: Callable[[float], None],
    ) -> None:
        """Runs with both tasks started. streamer streams inputs, AIN0 to AIN3, and
        command sets DAC0 to a level in volts."""
        for level in _LEVELS:
            self.log(f"Testing: DAC0 at {level} V")
            command(level)
            self._wait_for_level(streamer, inputs, level)

    def _wait_for_level(
        self, streamer: sy.Streamer, inputs: list[int], level: float
    ) -> None:
        """Reads streamer until one frame of AIN0 averages to level. A frame that spans
        a change of level averages between the two, so it never matches."""
        means: list[float] = []
        timer = sy.Timer()
        while timer.elapsed() < _LEVEL_TIMEOUT:
            frame = streamer.read(timeout=_LEVEL_TIMEOUT)
            if frame is None or any(len(frame[k]) == 0 for k in inputs):
                continue
            means = [float(frame[k].to_numpy().mean()) for k in inputs]
            if abs(means[0] - level) >= _LEVEL_TOLERANCE_V:
                continue
            if any(abs(mean - level) < _LEVEL_TOLERANCE_V for mean in means[1:]):
                self.fail(f"AIN0 to AIN3 read {means} V, but only AIN0 reads DAC0")
            return
        self.fail(f"AIN0 to AIN3 read {means} V, but AIN0 must read {level} V")


class LabJackAnalogWrite(_T4Write):
    """Writes voltage to both outputs of the T4."""

    task_name = "HITL LabJack Analog Write"
    command_values: list[list[float]] | None = [[1.0, 2.0], [3.0, 4.0]]

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.labjack.WriteChannel]:
        cmd_idx = create_index(client, "hitl_labjack_ao_cmd_time")
        state_idx = create_index(client, "hitl_labjack_ao_state_time")
        return [
            sy.labjack.AnalogWriteChannel(
                port=f"DAC{i}",
                cmd_channel=create_channel(
                    client,
                    name=f"hitl_labjack_ao_cmd_{i}",
                    data_type=sy.DataType.FLOAT64,
                    index=cmd_idx.key,
                ),
                state_channel=create_channel(
                    client,
                    name=f"hitl_labjack_ao_state_{i}",
                    data_type=sy.DataType.FLOAT64,
                    index=state_idx.key,
                ),
            )
            for i in range(2)
        ]


class LabJackDigitalRead(_T4Read):
    """Streams two floating digital inputs of the T4, which must read 1."""

    task_name = "HITL LabJack Digital Read"
    SAMPLE_RATE = 1000 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.labjack.ReadChannel]:
        idx = create_index(client, "hitl_labjack_di_time")
        return [
            sy.labjack.DigitalReadChannel(
                port=f"FIO{i}",
                channel=create_channel(
                    client,
                    name=f"hitl_labjack_di_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
            )
            for i in (6, 7)
        ]

    def run(self) -> None:
        start = sy.TimeStamp.now()
        super().run()
        assert self.tsk is not None
        keys = self._channel_keys(self.tsk)
        frame = self.client.read(sy.TimeRange(start, sy.TimeStamp.MAX), keys)
        for ch in self.client.channels.retrieve(keys):
            values = np.unique(frame[ch.key].to_numpy()).tolist()
            if values != [1]:
                self.fail(f"{ch.name} read {values}, not only 1")


class LabJackDigitalWrite(_T4Write):
    """Writes two digital outputs of the T4."""

    task_name = "HITL LabJack Digital Write"
    command_values: list[list[float]] | None = [[1, 0], [0, 1]]

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.labjack.WriteChannel]:
        cmd_idx = create_index(client, "hitl_labjack_do_cmd_time")
        state_idx = create_index(client, "hitl_labjack_do_state_time")
        return [
            sy.labjack.DigitalWriteChannel(
                port=f"FIO{i}",
                cmd_channel=create_channel(
                    client,
                    name=f"hitl_labjack_do_cmd_{i}",
                    data_type=sy.DataType.UINT8,
                    index=cmd_idx.key,
                ),
                state_channel=create_channel(
                    client,
                    name=f"hitl_labjack_do_state_{i}",
                    data_type=sy.DataType.UINT8,
                    index=state_idx.key,
                ),
            )
            for i in (4, 5)
        ]
