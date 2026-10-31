#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Modbus units that share one connection."""

import sys
from collections.abc import Callable

from examples.modbus import ModbusGatewaySim, ModbusSerialSim
from pydantic import BaseModel, ValidationError
from pymodbus.client import ModbusTcpClient

import synnax as sy
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    assert_streamed_values,
    cleanup_task,
    create_channel,
    create_index,
    run_and_expect_rejection,
    send_and_verify_commands,
)
from x.telem import to_float

DEMOTED = (
    "Failed to read data: unit 3 timed out 3 times in a row, so its requests fail "
    "for 10s"
)


class Stats(BaseModel):
    """The request counters of a unit in its device status."""

    requests: int
    retries: int
    timeouts: int
    exceptions: int
    last_response: int
    max_response: int


class ModbusUnits(SimulatorCase):
    """Reads units 1 and 2 of a gateway at once, then adds a unit that never answers.
    The units share one connection, the Driver demotes the silent unit, the other
    units keep reading, and each device status counts the requests of its unit."""

    sim_classes = [ModbusGatewaySim]
    units_sim: type[ModbusGatewaySim] | type[ModbusSerialSim] = ModbusGatewaySim
    prefix = "modbus_unit"

    def setup(self) -> None:
        self.tasks: list[sy.modbus.ReadTask | sy.modbus.WriteTask] = []
        super().setup()
        rack = self.client.racks.retrieve(name=self.options.driver_rack)
        unit_1 = self.client.devices.retrieve(name=self.units_sim.device_name)
        unit_2, unit_3 = self.create_test_devices(
            [
                self.units_sim.create_device(rack.key, unit_id=2),
                self.units_sim.create_device(rack.key, unit_id=3),
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
            self.test_stats()

    def create_task(
        self, unit: int, addresses: tuple[int, ...] = (0, 1)
    ) -> sy.modbus.ReadTask:
        """Configure a task that reads holding registers of a unit."""
        idx = create_index(self.client, f"{self.prefix}_{unit}_{addresses[0]}_time")
        task = sy.modbus.ReadTask(
            name=f"{self.prefix} {unit}",
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
                for address in addresses
            ],
        )
        self.client.tasks.configure(task)
        self.tasks.append(task)
        return task

    def channel_name(self, unit: int, address: int) -> str:
        return f"{self.prefix}_{unit}_hr_{address}"

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
            assert_streamed_values(self.client, streamer, expected=expected)

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
        self.log("Testing: A silent unit is demoted, and the others keep reading")
        silent = self.create_task(3)
        with self.client.open_streamer(["sy_status_set"]) as statuses:
            with silent.run():
                self.wait_for_warning(statuses, silent.key, DEMOTED)
                expected = self.expected_values((1, 2))
                with self.client.open_streamer(list(expected)) as streamer:
                    assert_streamed_values(self.client, streamer, expected=expected)

    def test_stats(self) -> None:
        """Unit 1 answers every request, unit 3 times out, and unit 2 answers a read
        past its registers with an exception."""
        self.log("Testing: Device statuses count the requests of each unit")
        answered = self.wait_for_stats(1, lambda s: s.requests > 0)
        assert answered.timeouts == 0 and answered.exceptions == 0, answered
        assert 0 < answered.last_response <= answered.max_response, answered
        silent = self.wait_for_stats(3, lambda s: s.timeouts >= 3)
        assert silent.exceptions == 0, silent
        rejected = self.create_task(2, addresses=(50,))
        message = run_and_expect_rejection(self.client, rejected)
        expected = (
            f"Failed to read data: {self.channel_name(2, 50)} (holding registers 50): "
            "Illegal data address"
        )
        assert message == expected, f"Rejection should be {expected!r}, got {message!r}"
        self.wait_for_stats(2, lambda s: s.exceptions > 0)

    def wait_for_stats(self, unit: int, done: Callable[[Stats], bool]) -> Stats:
        """Wait for the device status of a unit to report counters that satisfy
        ``done``. The Driver reports them each time it scans the device."""
        stats = None
        timer = sy.Timer()
        while timer.elapsed() < 15 * sy.TimeSpan.SECOND:
            dev = self.client.devices.retrieve(
                key=self.units[unit].key, include_status=True
            )
            details = None if dev.status is None else dev.status.details
            data = None if details is None else details.data
            if data is not None:
                stats = Stats.model_validate(data["stats"])
                if done(stats):
                    return stats
            sy.sleep(0.5)
        raise AssertionError(f"Unit {unit} counters did not arrive, got {stats}")

    @staticmethod
    def wait_for_warning(
        statuses: sy.Streamer,
        task_key: sy.task.Key,
        prefix: str,
        timeout: sy.TimeSpan = 15 * sy.TimeSpan.SECOND,
    ) -> None:
        """Wait for a warning or error status of a task whose message and description
        start with ``prefix``."""
        last = None
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
                    last = f"{status.message}: {status.description}"
                    if last.startswith(prefix):
                        return
        raise AssertionError(
            f"Task {task_key} should warn {prefix!r} within {timeout}, got {last!r}"
        )


class ModbusSerialUnits(ModbusUnits):
    """The same units on one serial line, and a write to unit 2. The line is a
    pseudo-terminal, so the case runs only on POSIX."""

    sim_classes = [ModbusSerialSim]
    units_sim: type[ModbusGatewaySim] | type[ModbusSerialSim] = ModbusSerialSim
    prefix = "modbus_serial_unit"

    def setup(self) -> None:
        if sys.platform == "win32":
            self.skip("Pseudo-terminals need POSIX")
        super().setup()

    def run(self) -> None:
        super().run()
        self.test_write()

    def test_one_connection(self) -> None:
        """The Driver locks a serial port, so a second connection would fail its
        reads. The reads above already show that the units share one connection."""

    def test_write(self) -> None:
        """A command to unit 2 reaches it: a read task on the register sees it."""
        self.log("Testing: Write to a unit on the serial line")
        idx = create_index(self.client, f"{self.prefix}_cmd_time")
        cmd = create_channel(
            self.client,
            name=f"{self.prefix}_cmd",
            data_type=sy.DataType.UINT16,
            index=idx.key,
        )
        write = sy.modbus.WriteTask(
            name=f"{self.prefix} write",
            device=self.units[2].key,
            channels=[
                sy.modbus.HoldingRegisterWriteChannel(
                    channel=cmd, address=5, data_type=sy.DataType.UINT16
                )
            ],
        )
        self.client.tasks.configure(write)
        self.tasks.append(write)
        state_idx = create_index(self.client, f"{self.prefix}_state_time")
        state = create_channel(
            self.client,
            name=f"{self.prefix}_state",
            data_type=sy.DataType.UINT16,
            index=state_idx.key,
        )
        read = sy.modbus.ReadTask(
            name=f"{self.prefix} state",
            device=self.units[2].key,
            sample_rate=10 * sy.Rate.HZ,
            stream_rate=5 * sy.Rate.HZ,
            channels=[
                sy.modbus.HoldingRegisterReadChannel(
                    channel=state, address=5, data_type=sy.DataType.UINT16
                )
            ],
        )
        self.client.tasks.configure(read)
        self.tasks.append(read)
        with write.run(), read.run():
            send_and_verify_commands(
                self.client,
                cmd_keys=[cmd],
                writer_name=f"{self.prefix}_writer",
                task_name=write.name,
                task_key=write.key,
                command_values=[[1234], [4321]],
            )
            self.wait_for_value(state, 4321)

    def wait_for_value(self, key: int, expected: float) -> None:
        """Wait until a channel streams ``expected``."""
        last = None
        timer = sy.Timer()
        with self.client.open_streamer([key]) as streamer:
            while timer.elapsed() < 5 * sy.TimeSpan.SECOND:
                frame = streamer.read(timeout=1)
                if frame is None or key not in frame or len(frame[key]) == 0:
                    continue
                last = to_float(frame[key][-1])
                if last == expected:
                    return
        raise AssertionError(f"Channel {key} should stream {expected}, got {last}")
