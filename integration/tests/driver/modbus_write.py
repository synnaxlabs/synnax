#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from examples.modbus import ModbusSim

import synnax as sy
from synnax import modbus
from tests.driver.modbus_task import (
    ModbusWriteTaskCase,
    assert_sim_coils,
    assert_sim_registers,
    float32_words,
    set_sim_coil,
    set_sim_register,
)
from tests.driver.task import create_channel, create_index


class ModbusWriteCoil(ModbusWriteTaskCase):
    task_name = "Modbus Write Coil"
    command_values: list[list[float]] | None = [[1, 0, 1], [0, 1, 0]]

    def assert_written(self) -> None:
        assert_sim_coils(10, [False, True, False])

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_coil_cmd_time")
        return [
            modbus.CoilWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_coil_cmd_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=10 + i,
            )
            for i in range(3)
        ]


class ModbusWriteHoldingRegister(ModbusWriteTaskCase):
    task_name = "Modbus Write Holding Register"
    command_values: list[list[float]] | None = [[1.5, 2.5, 3.5], [100, 101, 102]]

    def assert_written(self) -> None:
        words = [w for v in (100, 101, 102) for w in float32_words(v)]
        assert_sim_registers(10, words)

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_hr_cmd_time")
        return [
            modbus.HoldingRegisterWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_hr_cmd_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                address=10 + i * 2,
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(3)
        ]


class ModbusWriteMixed(ModbusWriteTaskCase):
    task_name = "Modbus Write Mixed"
    command_values: list[list[float]] | None = [[1, 0, 1.5, 2.5], [0, 1, 100, 101]]

    def assert_written(self) -> None:
        assert_sim_coils(20, [False, True])
        assert_sim_registers(20, [*float32_words(100), *float32_words(101)])

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_mixed_cmd_time")
        coils: list[sy.modbus.WriteChannel] = [
            modbus.CoilWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_mixed_coil_cmd_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=20 + i,
            )
            for i in range(2)
        ]
        registers: list[sy.modbus.WriteChannel] = [
            modbus.HoldingRegisterWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_mixed_hr_cmd_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                address=20 + i * 2,
                data_type=sy.DataType.FLOAT32,
            )
            for i in range(2)
        ]
        return coils + registers


class ModbusWriteWordOrder(ModbusWriteTaskCase):
    """Writes 1.5 (0x3FC00000) on a device that swaps words. The first register pair
    follows the device, and the second overrides it."""

    task_name = "Modbus Write Word Order"
    command_values: list[list[float]] | None = [[1.5, 1.5], [1.5, 1.5]]

    def create(self, *, device: sy.Device, **kwargs: Any) -> sy.modbus.WriteTask:
        sy.device.ensure_object(device.properties, "connection")["swap_words"] = True
        self.client.devices.create(device)
        return super().create(device=device, **kwargs)

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_order_cmd_time")
        return [
            modbus.HoldingRegisterWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_order_cmd_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                address=40 + i * 2,
                data_type=sy.DataType.FLOAT32,
                words_swapped=words_swapped,
            )
            for i, words_swapped in enumerate([None, False])
        ]

    def assert_written(self) -> None:
        assert_sim_registers(40, [0x3FC0, 0x0000, 0x0000, 0x3FC0])


class ModbusWriteGap(ModbusWriteTaskCase):
    """Writes channels on both sides of an address that no channel covers, while
    another client changes that address. The address keeps the other client's value."""

    task_name = "Modbus Write Gap"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_gap_cmd_time")
        registers: list[sy.modbus.WriteChannel] = [
            modbus.HoldingRegisterWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_gap_hr_cmd_{address}",
                    data_type=sy.DataType.UINT16,
                    index=idx.key,
                ),
                address=address,
                data_type=sy.DataType.UINT16,
            )
            for address in (120, 122)
        ]
        coils: list[sy.modbus.WriteChannel] = [
            modbus.CoilWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_gap_coil_cmd_{address}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=address,
            )
            for address in (110, 112)
        ]
        return registers + coils

    def run(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Write around addresses another client changes")
        with self.tsk.run():
            self.send_commands([[1, 2, 1, 0], [3, 4, 0, 1]])
            set_sim_register(121, 7)
            set_sim_coil(111, False)
            self.send_commands([[5, 6, 1, 0], [5, 6, 1, 0]])
            assert_sim_registers(120, [5, 7, 6])
            assert_sim_coils(110, [True, False, False])


class ModbusWriteCoilFloat(ModbusWriteTaskCase):
    """Drives a coil from a float32 command channel."""

    task_name = "Modbus Write Coil Float"
    command_values: list[list[float]] | None = [[0.0], [1.0]]

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_coil_float_cmd_time")
        return [
            modbus.CoilWriteChannel(
                channel=create_channel(
                    client,
                    name="modbus_coil_float_cmd",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                address=130,
            )
        ]

    def assert_written(self) -> None:
        assert_sim_coils(130, [True])


# Holds 0b10001100. The bit cases write its bits 0 and 3.
BIT_REGISTER = 140


class ModbusWriteRegisterBits(ModbusWriteTaskCase):
    """Writes two bits of one register in each command, while another client changes a
    third bit between commands. The register keeps the other client's change."""

    task_name = "Modbus Write Register Bits"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_bit_cmd_time")
        return [
            modbus.HoldingRegisterBitWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_bit_cmd_{bit}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=BIT_REGISTER,
                bit=bit,
            )
            for bit in (0, 3)
        ]

    def run(self) -> None:
        assert self.tsk is not None
        self.log("Testing: Write register bits around a bit another client changes")
        with self.tsk.run():
            self.send_commands([[1, 0], [1, 0]])
            assert_sim_registers(BIT_REGISTER, [0b10000101])
            set_sim_register(BIT_REGISTER, 0b00000101)
            self.send_commands([[0, 1], [0, 1]])
            assert_sim_registers(BIT_REGISTER, [0b00001100])


class ModbusWriteRegisterBitsMasked(ModbusWriteRegisterBits):
    """The same writes to a server that takes function code 22, which changes bits
    without a read."""

    task_name = "Modbus Write Register Bits Masked"

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.modbus.WriteTask:
        [masked] = self.create_test_devices(
            [
                modbus.Device(
                    host=ModbusSim.host,
                    port=ModbusSim.port,
                    name="Modbus Mask Write Server",
                    location=f"{ModbusSim.host}:{ModbusSim.port}",
                    rack=device.rack,
                    mask_write_supported=True,
                )
            ]
        )
        return super().create(
            device=masked,
            task_name=task_name,
            sample_rate=sample_rate,
            stream_rate=stream_rate,
        )
