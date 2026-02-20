#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax import modbus

from tests.driver.modbus_task import ModbusReadTaskCase
from tests.driver.task import create_channel, create_index


class ModbusReadInputRegister(ModbusReadTaskCase):
    task_name = "Modbus Read Input Register"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.BaseChan]:
        idx = create_index(client, "input_register_index")
        return [
            sy.modbus.InputRegisterChan(
                channel=create_channel(
                    client,
                    name=f"input_register_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
                data_type="uint8",
            )
            for i in range(2)
        ]


class ModbusReadHoldingRegister(ModbusReadTaskCase):
    task_name = "Modbus Read Holding Register"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.BaseChan]:
        idx = create_index(client, "holding_register_index")
        return [
            sy.modbus.HoldingRegisterInputChan(
                channel=create_channel(
                    client,
                    name=f"holding_register_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
                data_type="uint8",
            )
            for i in range(2)
        ]


class ModbusReadDiscreteInput(ModbusReadTaskCase):
    task_name = "Modbus Read Discrete Input"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.BaseChan]:
        idx = create_index(client, "discrete_input_index")
        return [
            sy.modbus.DiscreteInputChan(
                channel=create_channel(
                    client,
                    name=f"discrete_input_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
            )
            for i in range(2)
        ]


class ModbusReadCoil(ModbusReadTaskCase):
    task_name = "Modbus Read Coil"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.BaseChan]:
        idx = create_index(client, "coil_input_index")
        return [
            sy.modbus.CoilInputChan(
                channel=create_channel(
                    client,
                    name=f"coil_input_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
            )
            for i in range(2)
        ]


class ModbusReadMixed(ModbusReadTaskCase):
    task_name = "Modbus Read Mixed"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.BaseChan]:
        idx = create_index(client, "modbus_mixed_index")
        return [
            sy.modbus.InputRegisterChan(
                channel=create_channel(
                    client,
                    name=f"input_register_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
                data_type="uint8",
            )
            for i in range(2)
        ] + [
            sy.modbus.DiscreteInputChan(
                channel=create_channel(
                    client,
                    name=f"discrete_input_{i}",
                    data_type=sy.DataType.UINT8,
                    index=idx.key,
                ),
                address=i,
            )
            for i in range(2)
        ]
