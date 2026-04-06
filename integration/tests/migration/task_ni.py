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
from tests.migration.task import (
    ReadTaskConsoleVerify,
    ReadTaskMigration,
    ReadTaskMigrationSetup,
    ReadTaskMigrationVerify,
)

TASK_NAME = "mig_ni_analog_read"
IDX_NAME = "mig_ni_idx"
CHANNEL_PREFIX = "mig_ni_voltage"
NUM_CHANNELS = 2
DEVICE_LOCATION = "E101Mod4"  # NI 9205


class TaskNIMigration(ReadTaskMigration, NIAnalogReadTaskCase):
    """NI analog read task migration base."""

    task_name = TASK_NAME
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


class TaskNISetup(ReadTaskMigrationSetup, TaskNIMigration):
    """Create an NI analog read task, run it, and verify sample collection."""


class TaskNIVerify(ReadTaskMigrationVerify, TaskNIMigration):
    """Verify NI analog read task data survived, settings intact, and task still runs."""

    task_type = "ni_analog_read"
    task_class = sy.ni.AnalogReadTask
    channel_prefix = CHANNEL_PREFIX
    num_channels = NUM_CHANNELS


class TaskNIConsoleVerify(ReadTaskConsoleVerify):
    """Verify the NI analog read task configuration renders correctly in the console UI."""

    task_name = TASK_NAME
    expected_channels = [f"{CHANNEL_PREFIX}_{i}" for i in range(NUM_CHANNELS)]
    requires_platform = "windows"
