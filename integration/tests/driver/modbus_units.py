#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Modbus units behind one gateway."""

import re

from examples.modbus import ModbusGatewaySim
from pydantic import ValidationError
from pymodbus.client import ModbusTcpClient

import synnax as sy
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    assert_streamed_values,
    cleanup_task,
    create_channel,
    create_index,
)


class ModbusUnits(SimulatorCase):
    """Reads units 1 and 2 of a gateway at once, then adds a unit that never answers.
    The units share one connection, the silent unit reports a warning, and the other
    units keep reading."""

    sim_classes = [ModbusGatewaySim]

    def setup(self) -> None:
        self.tasks: list[sy.modbus.ReadTask] = []
        super().setup()
        rack = self.client.racks.retrieve(name=self.options.driver_rack)
        unit_1 = self.client.devices.retrieve(name=ModbusGatewaySim.device_name)
        unit_2, unit_3 = self.create_test_devices(
            [
                ModbusGatewaySim.create_device(rack.key, unit_id=2),
                ModbusGatewaySim.create_device(rack.key, unit_id=3),
            ]
        )
        self.units = {1: unit_1, 2: unit_2, 3: unit_3}

    def teardown(self) -> None:
        try:
            for task in self.tasks:
                cleanup_task(self.client, task)
        finally:
            super().teardown()

    def run(self) -> None:
        reading = [self.create_task(1), self.create_task(2)]
        with reading[0].run(), reading[1].run():
            self.test_units_read_their_values()
            self.test_one_connection()
            self.test_silent_unit()

    def create_task(self, unit: int) -> sy.modbus.ReadTask:
        """Configure a task that reads holding registers 0 and 1 of a unit."""
        idx = create_index(self.client, f"modbus_unit_{unit}_time")
        task = sy.modbus.ReadTask(
            name=f"Modbus Unit {unit}",
            device=self.units[unit].key,
            sample_rate=10 * sy.Rate.HZ,
            stream_rate=5 * sy.Rate.HZ,
            channels=[
                sy.modbus.HoldingRegisterReadChannel(
                    channel=create_channel(
                        self.client,
                        name=self.channel_name(unit, address),
                        data_type=sy.DataType.UINT16,
                        index=idx.key,
                    ),
                    address=address,
                    data_type=sy.DataType.UINT16,
                )
                for address in (0, 1)
            ],
        )
        self.client.tasks.configure(task)
        self.tasks.append(task)
        return task

    @staticmethod
    def channel_name(unit: int, address: int) -> str:
        return f"modbus_unit_{unit}_hr_{address}"

    def expected_values(self, units: tuple[int, ...]) -> dict[int, float]:
        """Holding register n of unit u holds 100 * u + n."""
        expected: dict[int, float] = {}
        for unit in units:
            for address in (0, 1):
                ch = self.client.channels.retrieve(self.channel_name(unit, address))
                expected[ch.key] = 100 * unit + address
        return expected

    def test_units_read_their_values(self) -> None:
        self.log("Testing: Each unit reads its own values")
        expected = self.expected_values((1, 2))
        with self.client.open_streamer(list(expected)) as streamer:
            assert_streamed_values(self.client, streamer, expected)

    def test_one_connection(self) -> None:
        """The gateway counts its open connections: the Driver's and this check's."""
        self.log("Testing: Units share one connection")
        with ModbusTcpClient(ModbusGatewaySim.host, port=ModbusGatewaySim.port) as sim:
            for _ in range(5):
                res = sim.read_input_registers(0, count=1, device_id=1)
                assert not res.isError(), res
                assert res.registers == [2], (
                    f"The gateway should have 2 connections, got {res.registers[0]}"
                )
                sy.sleep(0.2)

    def test_silent_unit(self) -> None:
        self.log("Testing: A silent unit warns, and the other units keep reading")
        silent = self.create_task(3)
        with self.client.open_streamer(["sy_status_set"]) as statuses:
            with silent.run():
                warning = self.wait_for_warning(statuses, silent.key)
                expected = self.expected_values((1, 2))
                with self.client.open_streamer(list(expected)) as streamer:
                    assert_streamed_values(self.client, streamer, expected)
        # The timeout text is the platform's: "Operation timed out" on macOS.
        pattern = r"Failed to read data: (Operation|Connection) timed out"
        assert re.match(pattern, warning), f"{warning!r} should match {pattern!r}"

    @staticmethod
    def wait_for_warning(
        statuses: sy.Streamer,
        task_key: sy.task.Key,
        timeout: sy.TimeSpan = 15 * sy.TimeSpan.SECOND,
    ) -> str:
        """Return the message and description of the first warning or error status
        of a task."""
        timer = sy.Timer()
        while timer.elapsed() < timeout:
            frame = statuses.read(timeout=timeout)
            if frame is None or "sy_status_set" not in frame:
                continue
            for raw in frame["sy_status_set"]:
                try:
                    status = sy.task.Status.model_validate(raw)
                except ValidationError:
                    continue
                if status.details is None or status.details.task != task_key:
                    continue
                if status.variant in ("warning", "error"):
                    return f"{status.message}: {status.description}"
        raise AssertionError(f"Task {task_key} did not warn within {timeout}")
