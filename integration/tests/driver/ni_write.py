#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""NI write task integration tests."""

from pydantic import ValidationError

import synnax as sy
from tests.driver.ni_task import NIAnalogWriteTaskCase, NIDigitalWriteTaskCase
from tests.driver.task import (
    create_channel,
    create_index,
)


def _do_channels(
    client: sy.Synnax, devices: dict[str, sy.Device]
) -> list[sy.ni.DOChan]:
    """Create two digital output channels on port 0, lines 0 and 1."""
    cmd_idx = create_index(client, "ni_do_cmd_time")
    state_idx = create_index(client, "ni_do_state_time")
    return [
        sy.ni.DOChan(
            cmd_channel=create_channel(
                client,
                name=f"ni_do_cmd_{i}",
                data_type=sy.DataType.UINT8,
                index=cmd_idx.key,
            ),
            state_channel=create_channel(
                client,
                name=f"ni_do_state_{i}",
                data_type=sy.DataType.UINT8,
                index=state_idx.key,
            ),
            port=0,
            line=i,
        )
        for i in range(2)
    ]


class NIDigitalWrite(NIDigitalWriteTaskCase):
    """Write valid digital output (0/1) on NI device port 0, lines 0 and 1."""

    task_name = "NI Digital Write"
    device_locations = ["SYMod1"]
    command_values = [[1, 0], [0, 1]]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.DOChan]:
        return _do_channels(client, devices)


def _assert_driver_rejects_value(
    client: sy.Synnax,
    task_key: int,
    *,
    cmd_keys: list[int],
    value: float,
    writer_name: str,
    timeout: sy.TimeSpan = 10 * sy.TimeSpan.SECOND,
) -> None:
    """Write a value to all cmd channels and assert the driver emits a warning or error.

    Opens the status streamer before writing to avoid missing events on slow runners.
    """

    channels = client.channels.retrieve(cmd_keys)
    index_keys = list({ch.index for ch in channels if ch.index != 0})

    with client.open_streamer(["sy_status_set"]) as streamer:
        writer = client.open_writer(
            start=sy.TimeStamp.now(),
            channels=cmd_keys + index_keys,
            name=writer_name,
            enable_auto_commit=True,
        )
        try:
            writer.write(
                {
                    **{k: value for k in cmd_keys},
                    **{k: sy.TimeStamp.now() for k in index_keys},
                }
            )
        finally:
            writer.close()

        timer = sy.Timer()
        while timer.elapsed() < timeout:
            frame = streamer.read(timeout=timeout)
            if frame is None:
                break
            if "sy_status_set" not in frame:
                continue
            for raw in frame["sy_status_set"]:
                try:
                    status = sy.task.Status.model_validate(raw)
                except ValidationError:
                    continue
                if status.details is None or status.details.task != task_key:
                    continue
                if status.variant in ("warning", "error"):
                    return

    raise AssertionError(f"Driver did not report an error for value {value}")


class NIDigitalWriteInvalidData(NIDigitalWriteTaskCase):
    """Write invalid digital data (42) and verify the driver reports an error."""

    task_name = "NI Digital Write Invalid"
    device_locations = ["SYMod1"]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.DOChan]:
        return _do_channels(client, devices)

    def run(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Send invalid digital values (42)")
        with self.tsk.run():
            _assert_driver_rejects_value(
                self.client,
                self.tsk.key,
                cmd_keys=self._channel_keys(self.tsk),
                value=42.0,
                writer_name=f"{self.task_name}_test_writer",
            )
        self.log("Driver correctly rejected invalid digital data")


def _ao_voltage_channels(
    client: sy.Synnax, devices: dict[str, sy.Device]
) -> list[sy.ni.AOVoltageChan]:
    """Create three voltage output channels on E102Mod4 (NI 9263).

    Port 0: MapScale (-10V..+10V → 0%..100%) — user writes percent
    Port 1: No scale — user writes volts (-10 to +10)
    Port 2: LinScale (slope=5 %/V, intercept=50) — user writes percent
    """
    cmd_idx = create_index(client, "ni_ao_volt_cmd_time")
    state_idx = create_index(client, "ni_ao_volt_state_time")
    map_scale = sy.ni.MapScale(
        pre_scaled_min=-10,
        pre_scaled_max=10,
        scaled_min=0,
        scaled_max=100,
        pre_scaled_units="Volts",
    )
    # slope=5: each volt = 5%, y_intercept=50: 0V = 50%
    lin_scale = sy.ni.LinScale(
        slope=5,
        y_intercept=50,
        pre_scaled_units="Volts",
        scaled_units="Volts",
    )
    map_scaled = sy.ni.AOVoltageChan(
        cmd_channel=create_channel(
            client,
            name="ni_ao_volt_cmd_0",
            data_type=sy.DataType.FLOAT64,
            index=cmd_idx.key,
        ),
        state_channel=create_channel(
            client,
            name="ni_ao_volt_state_0",
            data_type=sy.DataType.FLOAT64,
            index=state_idx.key,
        ),
        port=0,
        min_val=0,
        max_val=100,
        custom_scale=map_scale,
    )
    nominal = sy.ni.AOVoltageChan(
        cmd_channel=create_channel(
            client,
            name="ni_ao_volt_cmd_1",
            data_type=sy.DataType.FLOAT64,
            index=cmd_idx.key,
        ),
        state_channel=create_channel(
            client,
            name="ni_ao_volt_state_1",
            data_type=sy.DataType.FLOAT64,
            index=state_idx.key,
        ),
        port=1,
        min_val=-10,
        max_val=10,
    )
    lin_scaled = sy.ni.AOVoltageChan(
        cmd_channel=create_channel(
            client,
            name="ni_ao_volt_cmd_2",
            data_type=sy.DataType.FLOAT64,
            index=cmd_idx.key,
        ),
        state_channel=create_channel(
            client,
            name="ni_ao_volt_state_2",
            data_type=sy.DataType.FLOAT64,
            index=state_idx.key,
        ),
        port=2,
        min_val=0,
        max_val=100,
        custom_scale=lin_scale,
    )
    return [map_scaled, nominal, lin_scaled]


def _ao_current_channels(
    client: sy.Synnax, devices: dict[str, sy.Device]
) -> list[sy.ni.AOCurrentChan]:
    """Create three current output channels on E102Mod5 (NI 9265).

    Port 0: MapScale (4mA..20mA → 0%..100%) — user writes percent
    Port 1: No scale — user writes amps (0.004 to 0.020)
    Port 2: LinScale (slope=6250 %/A, intercept=-25) — user writes percent
    """
    cmd_idx = create_index(client, "ni_ao_curr_cmd_time")
    state_idx = create_index(client, "ni_ao_curr_state_time")
    map_scale = sy.ni.MapScale(
        pre_scaled_min=0.004,
        pre_scaled_max=0.020,
        scaled_min=0,
        scaled_max=100,
        pre_scaled_units="Amps",
    )
    # slope=6250: each amp = 6250%, y_intercept=-25: 0.004A = 0%
    lin_scale = sy.ni.LinScale(
        slope=6250,
        y_intercept=-25,
        pre_scaled_units="Amps",
        scaled_units="Amps",
    )
    map_scaled = sy.ni.AOCurrentChan(
        cmd_channel=create_channel(
            client,
            name="ni_ao_curr_cmd_0",
            data_type=sy.DataType.FLOAT64,
            index=cmd_idx.key,
        ),
        state_channel=create_channel(
            client,
            name="ni_ao_curr_state_0",
            data_type=sy.DataType.FLOAT64,
            index=state_idx.key,
        ),
        port=0,
        min_val=0,
        max_val=100,
        custom_scale=map_scale,
    )
    nominal = sy.ni.AOCurrentChan(
        cmd_channel=create_channel(
            client,
            name="ni_ao_curr_cmd_1",
            data_type=sy.DataType.FLOAT64,
            index=cmd_idx.key,
        ),
        state_channel=create_channel(
            client,
            name="ni_ao_curr_state_1",
            data_type=sy.DataType.FLOAT64,
            index=state_idx.key,
        ),
        port=1,
        min_val=0.004,
        max_val=0.020,
    )
    lin_scaled = sy.ni.AOCurrentChan(
        cmd_channel=create_channel(
            client,
            name="ni_ao_curr_cmd_2",
            data_type=sy.DataType.FLOAT64,
            index=cmd_idx.key,
        ),
        state_channel=create_channel(
            client,
            name="ni_ao_curr_state_2",
            data_type=sy.DataType.FLOAT64,
            index=state_idx.key,
        ),
        port=2,
        min_val=0,
        max_val=100,
        custom_scale=lin_scale,
    )
    return [map_scaled, nominal, lin_scaled]


class _NIAnalogWriteVoltageBase(NIAnalogWriteTaskCase):
    """Shared channel setup for voltage output tests on E102Mod4 (NI 9263)."""

    device_locations = ["E102Mod4"]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AOVoltageChan]:
        return _ao_voltage_channels(client, devices)


class _NIAnalogWriteCurrentBase(NIAnalogWriteTaskCase):
    """Shared channel setup for current output tests on E102Mod5 (NI 9265)."""

    device_locations = ["E102Mod5"]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AOCurrentChan]:
        return _ao_current_channels(client, devices)


class NIAnalogWriteVoltage(_NIAnalogWriteVoltageBase):
    """Write voltage on NI 9263 (E102Mod4).

    Port 0: MapScale (%) | Port 1: nominal (V) | Port 2: LinScale (%)
    """

    task_name = "NI Analog Write Voltage"
    # port 0 (map): 25%=-5V, 75%=5V | port 1: -5V, 5V | port 2 (lin): 25%=-5V, 75%=5V
    command_values = [[25, -5, 25], [75, 5, 75]]


class NIAnalogWriteCurrent(_NIAnalogWriteCurrentBase):
    """Write current on NI 9265 (E102Mod5).

    Port 0: MapScale (%) | Port 1: nominal (A) | Port 2: LinScale (%)
    """

    task_name = "NI Analog Write Current"
    # port 0 (map): 25%=8mA, 75%=16mA | port 1: 0.008A, 0.016A | port 2 (lin): 25%, 75%
    command_values = [[25, 0.008, 25], [75, 0.016, 75]]


def _send_oob_and_assert_state_clamped(
    client: sy.Synnax,
    tsk: sy.Task,
    *,
    oob_values: list[float],
    task_name: str,
    timeout: sy.TimeSpan = 10 * sy.TimeSpan.SECOND,
) -> None:
    """Send out-of-bounds values and verify state channels reflect clamped output.

    NI-DAQmx silently clamps analog values to hardware limits. This helper
    writes OOB command values, then reads the state channels and asserts
    the actual state does NOT match the commanded OOB values.
    """
    cmd_keys = [ch.cmd_channel for ch in tsk.config.channels]
    state_keys = [ch.state_channel for ch in tsk.config.channels]

    channels = client.channels.retrieve(cmd_keys)
    index_keys = list({ch.index for ch in channels if ch.index != 0})
    all_writer_keys = cmd_keys + index_keys

    with client.open_streamer(state_keys) as streamer:
        writer = client.open_writer(
            start=sy.TimeStamp.now(),
            channels=all_writer_keys,
            name=f"{task_name}_oob_writer",
            enable_auto_commit=True,
        )
        try:
            cmd_frame = {key: float(v) for key, v in zip(cmd_keys, oob_values)}
            cmd_frame.update({k: sy.TimeStamp.now() for k in index_keys})
            writer.write(cmd_frame)

            received: dict[int, float] = {}
            timer = sy.Timer()
            while len(received) < len(state_keys):
                if timer.elapsed() > timeout:
                    missing = set(state_keys) - set(received.keys())
                    raise AssertionError(
                        f"{task_name}: Timeout waiting for state values. "
                        f"Missing keys: {missing}"
                    )
                frame = streamer.read(timeout=timeout)
                if frame is None:
                    continue
                for key in state_keys:
                    if key in frame and len(frame[key]) > 0:
                        received[key] = float(frame[key][-1])

            for state_key, cmd_key, oob_val in zip(state_keys, cmd_keys, oob_values):
                state_val = received[state_key]
                if state_val == oob_val:
                    cmd_ch = client.channels.retrieve(cmd_key)
                    raise AssertionError(
                        f"{task_name}: Channel '{cmd_ch.name}' state "
                        f"matches OOB command {oob_val} — expected "
                        f"clamped value"
                    )
        finally:
            writer.close()


class _NIAnalogWriteOOBBase(NIAnalogWriteTaskCase):
    """Base for OOB analog write tests — sends out-of-bounds values and verifies clamping."""

    _oob_values: list[float]

    def run(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Send OOB values and verify state clamping")
        with self.tsk.run():
            _send_oob_and_assert_state_clamped(
                self.client,
                self.tsk,
                oob_values=self._oob_values,
                task_name=self.task_name,
            )
        self.log("State channels correctly show clamped values")


class NIAnalogWriteVoltageOOB(_NIAnalogWriteOOBBase, _NIAnalogWriteVoltageBase):
    """Send out-of-bounds voltage and verify state is clamped by hardware."""

    task_name = "NI Analog Write Voltage OOB"
    command_values = [[25, -5, 25], [75, 5, 75]]
    # port 0 (map): 200% OOB, port 1: 15V OOB, port 2 (lin): 200% OOB
    _oob_values = [200, 15, 101]


class NIAnalogWriteCurrentOOB(_NIAnalogWriteOOBBase, _NIAnalogWriteCurrentBase):
    """Send out-of-bounds current and verify state is clamped by hardware."""

    task_name = "NI Analog Write Current OOB"
    command_values = [[25, 0.008, 25], [75, 0.016, 75]]
    # port 0 (map): 200% OOB, port 1: 0.025A OOB, port 2 (lin): 200% OOB
    _oob_values = [200, 0.025, 101]
