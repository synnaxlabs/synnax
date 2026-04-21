#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from tests.driver.opcua_task import OPCUAReadTaskCase
from tests.driver.task import create_channel, create_index
from tests.migration.task import ReadTaskConsoleVerify, ReadTaskMigrationVerify

TASK_NAME = "mig_opc_read"
IDX_NAME = "mig_opc_idx"
CHANNEL_PREFIX = "mig_opc_float"
NUM_CHANNELS = 2


class OPCUAReadVerify(ReadTaskMigrationVerify, OPCUAReadTaskCase):
    """Verify OPC UA task config survived and task can still run."""

    task_name = TASK_NAME
    task_type = "opc_read"
    task_class = sy.opcua.ReadTask
    channel_prefix = CHANNEL_PREFIX
    num_channels = NUM_CHANNELS
    pre_start_sleep = 5

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, IDX_NAME)
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"{CHANNEL_PREFIX}_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type="float32",
            )
            for i in range(NUM_CHANNELS)
        ]


class OPCUAReadConsoleVerify(ReadTaskConsoleVerify):
    """Verify the OPC UA read task configuration renders correctly in the console UI."""

    task_name = TASK_NAME
    expected_channels = [f"{CHANNEL_PREFIX}_{i}" for i in range(NUM_CHANNELS)] + [
        f"NS=2;I={8 + i}" for i in range(NUM_CHANNELS)
    ]
    expected_sample_rate = "50"
    expected_stream_rate = "10"
