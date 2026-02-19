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
from synnax.modbus.types import OutputChan

from tests.driver.modbus_task import ModbusWriteTaskCase


class ModbusWriteCoil(ModbusWriteTaskCase):
    task_name = "Modbus Write Coil"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[OutputChan]:
        return [
            modbus.CoilOutputChan(
                channel=client.channels.create(
                    name="modbus_coil_cmd_0",
                    data_type=sy.DataType.UINT8,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=10,
            ),
            modbus.CoilOutputChan(
                channel=client.channels.create(
                    name="modbus_coil_cmd_1",
                    data_type=sy.DataType.UINT8,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=11,
            ),
            modbus.CoilOutputChan(
                channel=client.channels.create(
                    name="modbus_coil_cmd_2",
                    data_type=sy.DataType.UINT8,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=12,
            ),
        ]


class ModbusWriteHoldingRegister(ModbusWriteTaskCase):
    task_name = "Modbus Write Holding Register"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[OutputChan]:
        return [
            modbus.HoldingRegisterOutputChan(
                channel=client.channels.create(
                    name="modbus_hr_cmd_0",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=10,
                data_type="float32",
            ),
            modbus.HoldingRegisterOutputChan(
                channel=client.channels.create(
                    name="modbus_hr_cmd_1",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=12,
                data_type="float32",
            ),
            modbus.HoldingRegisterOutputChan(
                channel=client.channels.create(
                    name="modbus_hr_cmd_2",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=14,
                data_type="float32",
            ),
        ]


class ModbusWriteMixed(ModbusWriteTaskCase):
    task_name = "Modbus Write Mixed"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[OutputChan]:
        return [
            modbus.CoilOutputChan(
                channel=client.channels.create(
                    name="modbus_mixed_coil_cmd_0",
                    data_type=sy.DataType.UINT8,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=20,
            ),
            modbus.CoilOutputChan(
                channel=client.channels.create(
                    name="modbus_mixed_coil_cmd_1",
                    data_type=sy.DataType.UINT8,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=21,
            ),
            modbus.HoldingRegisterOutputChan(
                channel=client.channels.create(
                    name="modbus_mixed_hr_cmd_0",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=20,
                data_type="float32",
            ),
            modbus.HoldingRegisterOutputChan(
                channel=client.channels.create(
                    name="modbus_mixed_hr_cmd_1",
                    data_type=sy.DataType.FLOAT32,
                    virtual=True,
                    retrieve_if_name_exists=True,
                ).key,
                address=22,
                data_type="float32",
            ),
        ]
