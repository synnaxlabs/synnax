#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI tasks on the C Series modules of the cRIO-9041, run by the Driver built for NI
Linux RT.

Slot | Device | Module
-----|--------|-----------------------------------------------------------------
1    | Mod1   | NI 9205, 32 analog inputs
2    | Mod2   | NI 9263, 4 analog outputs. AO0 and AO1 connect to AI0 and AI1 of the
     |        | NI 9205, and its COM connects to the COM of the NI 9205.
3    | Mod3   | NI 9211, 4 thermocouple inputs. TC0 has a thermocouple in room air.
"""

import synnax as sy
from tests.driver.ni_task import (
    NIAnalogReadTaskCase,
    NIAnalogWriteTaskCase,
    retrieve_present_device,
)
from tests.driver.task import create_channel, create_index, delete_channels
from tests.hitl.crio_case import CRIOCase

# Each level changes both outputs, and no level matches what an earlier test leaves on
# them, so an input can only read a level after the output really changes.
_LEVELS = [(3.0, -1.5), (-2.5, 4.5), (0.5, -0.5)]
# Wide over the few millivolts that the station measures, and far under the volts
# between two levels or two channels.
_LEVEL_TOLERANCE_V = 0.05
_LEVEL_TIMEOUT = 5 * sy.TimeSpan.SECOND
_ROOM_C = (10.0, 45.0)
_TEMPERATURE_SAMPLES = 4
_TEMPERATURE_TIMEOUT = 15 * sy.TimeSpan.SECOND


class CRIOAnalogRead(CRIOCase, NIAnalogReadTaskCase):
    """Reads voltage from four inputs of the NI 9205."""

    task_name = "HITL cRIO Analog Read"
    device_locations = ["Mod1"]
    simulated = False
    SAMPLE_RATE = 1000 * sy.Rate.HZ
    STREAM_RATE = 25 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIChannel]:
        idx = create_index(client, "hitl_crio_ai_time")
        return [
            sy.ni.AIVoltageChannel(
                port=i,
                channel=create_channel(
                    client,
                    name=f"hitl_crio_ai_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                # Each output connects to one input, measured against the shared COM.
                terminal_config="RSE",
                min_val=-10.0,
                max_val=10.0,
            )
            for i in range(4)
        ]


class CRIOAnalogWrite(CRIOCase, NIAnalogWriteTaskCase):
    """Writes voltage to two outputs of the NI 9263."""

    task_name = "HITL cRIO Analog Write"
    device_locations = ["Mod2"]
    simulated = False
    command_values: list[list[float]] | None = [[1, -1], [2, -2]]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AOVoltageChannel]:
        cmd_idx = create_index(client, "hitl_crio_ao_cmd_time")
        state_idx = create_index(client, "hitl_crio_ao_state_time")
        return [
            sy.ni.AOVoltageChannel(
                cmd_channel=create_channel(
                    client,
                    name=f"hitl_crio_ao_cmd_{i}",
                    data_type=sy.DataType.FLOAT64,
                    index=cmd_idx.key,
                ),
                state_channel=create_channel(
                    client,
                    name=f"hitl_crio_ao_state_{i}",
                    data_type=sy.DataType.FLOAT64,
                    index=state_idx.key,
                ),
                port=i,
                min_val=-10,
                max_val=10,
            )
            for i in range(2)
        ]


class CRIOAnalogLoopback(CRIOAnalogRead):
    """Writes a different voltage to AO0 and AO1 of the NI 9263, and checks that AI0 and
    AI1 of the NI 9205 read them. A swap of channels in either task fails the test."""

    task_name = "HITL cRIO Analog Loopback"

    def run(self) -> None:
        assert self.tsk is not None
        inputs = self._channel_keys(self.tsk)[:2]
        output = retrieve_present_device(self.client, "Mod2")
        outputs = CRIOAnalogWrite.create_channels(self.client, {"Mod2": output})
        write = sy.ni.AnalogWriteTask(
            name="HITL cRIO Analog Loopback Write",
            device=output.key,
            state_rate=self.SAMPLE_RATE,
            channels=outputs,
        )
        self.client.tasks.configure(write)
        commands = [ch.cmd_channel for ch in outputs]
        cmd_index = self.client.channels.retrieve(commands[0]).index
        try:
            with (
                self.tsk.run(timeout=self.COMMAND_TIMEOUT),
                write.run(timeout=self.COMMAND_TIMEOUT),
                self.client.open_streamer(inputs) as streamer,
                self.client.open_writer(
                    sy.TimeStamp.now(),
                    [*commands, cmd_index],
                    enable_auto_commit=True,
                ) as writer,
            ):
                for levels in _LEVELS:
                    self.log(f"Testing: AO0 and AO1 at {levels} V")
                    writer.write(
                        {
                            **dict(zip(commands, levels, strict=True)),
                            cmd_index: sy.TimeStamp.now(),
                        }
                    )
                    self._wait_for_levels(streamer, inputs, levels)
        finally:
            # The Core refuses to delete a channel that a task uses.
            self.client.tasks.delete(write.key)
            delete_channels(
                self.client,
                commands + [ch.state_channel for ch in outputs],
            )

    def _wait_for_levels(
        self, streamer: sy.Streamer, inputs: list[int], levels: tuple[float, float]
    ) -> None:
        """Reads streamer until one frame of each input averages to its level. A frame
        that spans a change of level averages between the two, so it never matches."""
        means: list[float] = []
        timer = sy.Timer()
        while timer.elapsed() < _LEVEL_TIMEOUT:
            frame = streamer.read(timeout=_LEVEL_TIMEOUT)
            if frame is None or any(len(frame[k]) == 0 for k in inputs):
                continue
            means = [float(frame[k].to_numpy().mean()) for k in inputs]
            if all(
                abs(mean - level) < _LEVEL_TOLERANCE_V
                for mean, level in zip(means, levels, strict=True)
            ):
                return
        self.fail(f"AI0 and AI1 read {means} V, not {list(levels)} V")


class CRIOThermocoupleRead(CRIOCase, NIAnalogReadTaskCase):
    """Reads the thermocouple on TC0 of the NI 9211, which must read a room temperature.
    An open input reads thousands of degrees."""

    task_name = "HITL cRIO Thermocouple Read"
    device_locations = ["Mod3"]
    simulated = False
    # The NI 9211 converts at most 14 samples per second across its channels.
    SAMPLE_RATE = 2 * sy.Rate.HZ
    STREAM_RATE = 1 * sy.Rate.HZ

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIChannel]:
        idx = create_index(client, "hitl_crio_tc_time")
        return [
            sy.ni.AIThermocoupleChannel(
                port=0,
                channel=create_channel(
                    client,
                    name="hitl_crio_tc_0",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                thermocouple_type="K",
                min_val=0,
                max_val=100,
            )
        ]

    def run(self) -> None:
        assert self.tsk is not None
        key = self._channel_keys(self.tsk)[0]
        temperatures: list[float] = []
        timer = sy.Timer()
        with (
            self.tsk.run(timeout=self.COMMAND_TIMEOUT),
            self.client.open_streamer(key) as streamer,
        ):
            while len(temperatures) < _TEMPERATURE_SAMPLES:
                if timer.elapsed() > _TEMPERATURE_TIMEOUT:
                    self.fail(f"Read {len(temperatures)} temperatures in 15 s")
                frame = streamer.read(timeout=_TEMPERATURE_TIMEOUT)
                if frame is not None:
                    temperatures.extend(frame[key].to_numpy().tolist())
        low, high = _ROOM_C
        if not all(low <= t <= high for t in temperatures):
            self.fail(f"TC0 read {temperatures} °C, not {low} to {high} °C")
        self.log(f"TC0 read {temperatures} °C")
