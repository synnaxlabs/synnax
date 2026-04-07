#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from tests.driver.modbus_task import ModbusReadTaskCase
from tests.driver.task import create_channel, create_index
from tests.migration.task import (
    ReadTaskConsoleVerify,
    ReadTaskMigration,
    ReadTaskMigrationSetup,
    ReadTaskMigrationVerify,
)

TASK_NAME = "mig_modbus_read"
IDX_NAME = "mig_modbus_idx"
CHANNEL_PREFIX = "mig_modbus_reg"
NUM_CHANNELS = 2


class ModbusReadMigration(ReadTaskMigration, ModbusReadTaskCase):
    """Modbus read task migration base."""

    task_name = TASK_NAME

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.modbus.BaseChan]:
        idx = create_index(client, IDX_NAME)
        return [
            sy.modbus.HoldingRegisterInputChan(
                channel=create_channel(
                    client,
                    name=f"{CHANNEL_PREFIX}_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                address=i,
                data_type="float32",
            )
            for i in range(NUM_CHANNELS)
        ]


class ModbusReadSetup(ReadTaskMigrationSetup, ModbusReadMigration):
    """Create a Modbus read task, run it, and verify sample collection."""


class ModbusReadVerify(ReadTaskMigrationVerify, ModbusReadMigration):
    """Verify Modbus task data survived, settings intact, and task still runs."""

    task_type = "modbus_read"
    task_class = sy.modbus.ReadTask
    channel_prefix = CHANNEL_PREFIX
    num_channels = NUM_CHANNELS
    pre_start_sleep = 2


class ModbusReadConsoleVerify(ReadTaskConsoleVerify):
    """Verify the Modbus read task configuration renders correctly in the console UI."""

    task_name = TASK_NAME
    expected_channels = [f"{CHANNEL_PREFIX}_{i}" for i in range(NUM_CHANNELS)]
    expected_sample_rate = "50"
    expected_stream_rate = "10"
