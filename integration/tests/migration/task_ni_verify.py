#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from tests.driver.ni_task import NIAnalogReadTaskCase
from tests.driver.task import create_channel, create_index
from tests.migration.task_ni_setup import (
    CHANNEL_PREFIX,
    DEVICE_LOCATION,
    IDX_NAME,
    NUM_CHANNELS,
    TASK_NAME,
)
from tests.migration.task_verify import ReadTaskConsoleVerify, ReadTaskMigrationVerify


class NIAnalogReadVerify(ReadTaskMigrationVerify, NIAnalogReadTaskCase):
    """Verify NI analog read task config survived and task can still run."""

    task_name = TASK_NAME
    task_type = "ni_analog_read"
    task_class = sy.ni.AnalogReadTask
    channel_prefix = CHANNEL_PREFIX
    num_channels = NUM_CHANNELS
    device_locations = [DEVICE_LOCATION]

    @staticmethod
    def create_channels(
        client: sy.Synnax, devices: dict[str, sy.Device]
    ) -> list[sy.ni.AIVoltageChan]:
        idx = create_index(client, IDX_NAME)
        return [
            sy.ni.AIVoltageChan(
                port=i,
                channel=create_channel(
                    client,
                    name=f"{CHANNEL_PREFIX}_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                terminal_config="Cfg_Default",
                min_val=-10.0,
                max_val=10.0,
            )
            for i in range(NUM_CHANNELS)
        ]


class NIAnalogReadConsoleVerify(ReadTaskConsoleVerify):
    """Verify the NI analog read task configuration renders correctly in the console."""

    task_name = TASK_NAME
    expected_channels = [f"{CHANNEL_PREFIX}_{i}" for i in range(NUM_CHANNELS)]
    requires_platform = "windows"
