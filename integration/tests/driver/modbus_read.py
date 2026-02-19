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
from synnax.modbus.types import BaseChan

from tests.driver.modbus_task import ModbusReadTaskCase


class ModbusReadInputRegister(ModbusReadTaskCase):
    task_name = "Modbus Read Input Register"

    def create_channels(self) -> list[BaseChan]:
        index_c = self.client.channels.create(
            name="input_register_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            modbus.InputRegisterChan(
                channel=self.client.channels.create(
                    name="input_register_0",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                address=0,
                data_type="uint8",
            ),
            modbus.InputRegisterChan(
                channel=self.client.channels.create(
                    name="input_register_1",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                address=1,
                data_type="uint8",
            ),
        ]


class ModbusReadHoldingRegister(ModbusReadTaskCase):
    task_name = "Modbus Read Holding Register"

    def create_channels(self) -> list[BaseChan]:
        index_c = self.client.channels.create(
            name="holding_register_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            modbus.HoldingRegisterInputChan(
                channel=self.client.channels.create(
                    name="holding_register_0",
                    index=index_c.key,
                    data_type=sy.DataType.UINT8,
                    retrieve_if_name_exists=True,
                ).key,
                address=0,
                data_type="uint8",
            ),
            modbus.HoldingRegisterInputChan(
                channel=self.client.channels.create(
                    name="holding_register_1",
                    index=index_c.key,
                    data_type=sy.DataType.UINT8,
                    retrieve_if_name_exists=True,
                ).key,
                address=1,
                data_type="uint8",
            ),
        ]


class ModbusReadDiscreteInput(ModbusReadTaskCase):
    task_name = "Modbus Read Discrete Input"

    def create_channels(self) -> list[BaseChan]:
        index_c = self.client.channels.create(
            name="discrete_input_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            modbus.DiscreteInputChan(
                channel=self.client.channels.create(
                    name="discrete_input_0",
                    index=index_c.key,
                    data_type=sy.DataType.UINT8,
                    retrieve_if_name_exists=True,
                ).key,
                address=0,
            ),
            modbus.DiscreteInputChan(
                channel=self.client.channels.create(
                    name="discrete_input_1",
                    index=index_c.key,
                    data_type=sy.DataType.UINT8,
                    retrieve_if_name_exists=True,
                ).key,
                address=1,
            ),
        ]


class ModbusReadCoil(ModbusReadTaskCase):
    task_name = "Modbus Read Coil"

    def create_channels(self) -> list[BaseChan]:
        index_c = self.client.channels.create(
            name="coil_input_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            modbus.CoilInputChan(
                channel=self.client.channels.create(
                    name="coil_input_0",
                    index=index_c.key,
                    data_type=sy.DataType.UINT8,
                    retrieve_if_name_exists=True,
                ).key,
                address=0,
            ),
            modbus.CoilInputChan(
                channel=self.client.channels.create(
                    name="coil_input_1",
                    index=index_c.key,
                    data_type=sy.DataType.UINT8,
                    retrieve_if_name_exists=True,
                ).key,
                address=1,
            ),
        ]


class ModbusReadMixed(ModbusReadTaskCase):
    task_name = "Modbus Read Mixed"

    def create_channels(self) -> list[BaseChan]:
        index_c = self.client.channels.create(
            name="modbus_mixed_index",
            data_type=sy.DataType.TIMESTAMP,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        return [
            modbus.InputRegisterChan(
                channel=self.client.channels.create(
                    name="input_register_0",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                address=0,
                data_type="uint8",
            ),
            modbus.InputRegisterChan(
                channel=self.client.channels.create(
                    name="input_register_1",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                address=1,
                data_type="uint8",
            ),
            modbus.DiscreteInputChan(
                channel=self.client.channels.create(
                    name="discrete_input_0",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                address=0,
            ),
            modbus.DiscreteInputChan(
                channel=self.client.channels.create(
                    name="discrete_input_1",
                    data_type=sy.DataType.UINT8,
                    index=index_c.key,
                    retrieve_if_name_exists=True,
                ).key,
                address=1,
            ),
        ]
