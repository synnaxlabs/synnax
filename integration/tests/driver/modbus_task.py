#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Modbus-specific task test cases."""

import struct
from abc import abstractmethod
from collections.abc import Callable

from examples.modbus import ModbusSim
from examples.modbus.server import SLAVE_ID
from pymodbus.client import ModbusTcpClient

import synnax as sy
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import ReadTaskCase, WriteTaskCase, send_and_verify_commands


def float32_words(value: float) -> list[int]:
    """Return the register words of a float32, low word first as the Driver writes by
    default."""
    high, low = struct.unpack(">HH", struct.pack(">f", value))
    return [low, high]


def _wait_for_sim[T](
    read: Callable[[ModbusTcpClient], list[T]], expected: list[T], what: str
) -> None:
    """Assert that ``read`` returns ``expected`` within two seconds. The Synnax echo
    of a command can arrive before the Driver writes it to the device."""
    timer = sy.Timer()
    with ModbusTcpClient(ModbusSim.host, port=ModbusSim.port) as sim:
        while (got := read(sim)) != expected:
            if timer.elapsed() > 2 * sy.TimeSpan.SECOND:
                raise AssertionError(f"{what} should be {expected}, got {got}")
            sy.sleep(0.05)


def assert_sim_registers(address: int, expected: list[int]) -> None:
    """Assert the simulator's holding registers from ``address`` become ``expected``."""

    def read(sim: ModbusTcpClient) -> list[int]:
        res = sim.read_holding_registers(
            address, count=len(expected), device_id=SLAVE_ID
        )
        assert not res.isError(), res
        return list(res.registers)

    _wait_for_sim(read, expected, f"Holding registers from {address}")


def set_sim_register(address: int, value: int) -> None:
    """Write one holding register on the simulator, as another client would."""
    with ModbusTcpClient(ModbusSim.host, port=ModbusSim.port) as sim:
        res = sim.write_register(address, value, device_id=SLAVE_ID)
    assert not res.isError(), res


def set_sim_coil(address: int, value: bool) -> None:
    """Write one coil on the simulator, as another client would."""
    with ModbusTcpClient(ModbusSim.host, port=ModbusSim.port) as sim:
        res = sim.write_coil(address, value, device_id=SLAVE_ID)
    assert not res.isError(), res


def assert_sim_coils(address: int, expected: list[bool]) -> None:
    """Assert the simulator's coils from ``address`` become ``expected``."""

    def read(sim: ModbusTcpClient) -> list[bool]:
        res = sim.read_coils(address, count=len(expected), device_id=SLAVE_ID)
        assert not res.isError(), res
        return list(res.bits[: len(expected)])

    _wait_for_sim(read, expected, f"Coils from {address}")


class ModbusReadTaskCase(SimulatorCase, ReadTaskCase[sy.modbus.ReadTask]):
    """Base class for Modbus TCP read task tests."""

    sim_classes = [ModbusSim]
    SAMPLE_RATE = 50 * sy.Rate.HZ

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.ReadChannel]: ...

    def _channel_keys(self, task: sy.modbus.ReadTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.modbus.ReadTask:
        """Create a Modbus read task."""
        channels = self.create_channels(self.client)

        return sy.modbus.ReadTask(
            name=task_name,
            device=device.key,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
            channels=channels,
        )


class ModbusWriteTaskCase(SimulatorCase, WriteTaskCase[sy.modbus.WriteTask]):
    """Base class for Modbus TCP write task tests."""

    sim_classes = [ModbusSim]

    @staticmethod
    @abstractmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]: ...

    def _channel_keys(self, task: sy.modbus.WriteTask) -> list[int]:
        return [ch.channel for ch in task.config.channels]

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.modbus.WriteTask:
        """Create a Modbus write task."""
        channels = self.create_channels(self.client)
        return sy.modbus.WriteTask(
            name=task_name,
            device=device.key,
            channels=channels,
        )

    def send_commands(self, values: list[list[float]]) -> None:
        """Send two rounds of commands to the running task, one value per channel in
        each."""
        assert self.tsk is not None
        send_and_verify_commands(
            self.client,
            cmd_keys=self._channel_keys(self.tsk),
            writer_name=f"{self.task_name}_test_writer",
            task_name=self.tsk.name,
            task_key=self.tsk.key,
            command_values=values,
        )
