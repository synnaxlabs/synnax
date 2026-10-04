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
from examples.modbus.server import SLAVE_ID
from pymodbus.client import ModbusTcpClient

import synnax as sy
from synnax import modbus
from tests.driver.modbus_task import ModbusWriteTaskCase
from tests.driver.task import create_channel, create_index


class ModbusWriteCoil(ModbusWriteTaskCase):
    task_name = "Modbus Write Coil"

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
                data_type="float32",
            )
            for i in range(3)
        ]


class ModbusWriteMixed(ModbusWriteTaskCase):
    task_name = "Modbus Write Mixed"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.WriteChannel]:
        idx = create_index(client, "modbus_mixed_cmd_time")
        return [
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
        ] + [
            modbus.HoldingRegisterWriteChannel(
                channel=create_channel(
                    client,
                    name=f"modbus_mixed_hr_cmd_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                address=20 + i * 2,
                data_type="float32",
            )
            for i in range(2)
        ]


class ModbusWriteWordOrder(ModbusWriteTaskCase):
    """Writes 1.5 (0x3FC00000) on a device that swaps words. The first register pair
    follows the device, and the second overrides it."""

    task_name = "Modbus Write Word Order"
    command_values = [[1.5, 1.5], [1.5, 1.5]]

    def create(self, *, device: sy.Device, **kwargs: Any) -> sy.modbus.WriteTask:
        device.properties["connection"]["swap_words"] = True
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
                data_type="float32",
                words_swapped=words_swapped,
            )
            for i, words_swapped in enumerate([None, False])
        ]

    def run(self) -> None:
        super().run()
        with ModbusTcpClient(ModbusSim.host, port=ModbusSim.port) as sim:
            res = sim.read_holding_registers(40, count=4, device_id=SLAVE_ID)
        assert res.registers == [0x3FC0, 0x0000, 0x0000, 0x3FC0], res.registers
