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
from tests.driver.task import create_channel, create_index


class ModbusWriteCoil(ModbusWriteTaskCase):
    task_name = "Modbus Write Coil"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[OutputChan]:
        idx = create_index(client, "modbus_coil_cmd_time")
        return [
            modbus.CoilOutputChan(
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
    def create_channels(client: sy.Synnax) -> list[OutputChan]:
        idx = create_index(client, "modbus_hr_cmd_time")
        return [
            modbus.HoldingRegisterOutputChan(
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
    def create_channels(client: sy.Synnax) -> list[OutputChan]:
        idx = create_index(client, "modbus_mixed_cmd_time")
        return [
            modbus.CoilOutputChan(
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
            modbus.HoldingRegisterOutputChan(
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
