#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from abc import abstractmethod
from typing import Any

import synnax as sy

from tests.driver.ni_task import NIAnalogReadTaskCase
from tests.driver.task import create_channel, create_index

TASK_NAME = "mig_ni_analog_read"
IDX_NAME = "mig_ni_idx"
CHANNEL_PREFIX = "mig_ni_voltage"
NUM_CHANNELS = 2
DEVICE_LOCATION = "E101Mod1"  # NI 9229


class TaskNIMigration(NIAnalogReadTaskCase):
    """Base class defining the migration test contract for NI analog read tasks.

    Subclasses must implement each test method — setup creates the state,
    verify checks it after migration.
    """

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

    def run(self) -> None:
        self.test_task_config()
        self.test_data()

    def teardown(self) -> None:
        """Stop task without deleting — must survive across phases."""
        if self.tsk is not None:
            self.tsk.stop()
            self.log(f"Task '{self.task_name}' stopped")

    @abstractmethod
    def test_task_config(self) -> None: ...

    @abstractmethod
    def test_data(self) -> None: ...


class TaskNISetup(TaskNIMigration):
    """Create an NI analog read task, run it, and verify sample collection."""

    def test_task_config(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

    def test_data(self) -> None:
        self.test_start_and_stop()


class TaskNIVerify(TaskNIMigration):
    """Verify NI analog read task data survived, settings intact, and task still runs."""

    def create(self, **kwargs: Any) -> sy.ni.AnalogReadTask:
        """Retrieve the existing task instead of creating a new one."""
        tasks = self.client.tasks.retrieve(names=[TASK_NAME])
        assert (
            len(tasks) == 1
        ), f"Expected exactly 1 task named '{TASK_NAME}', got {len(tasks)}"
        raw = tasks[0]
        typed = sy.ni.AnalogReadTask(**raw.config)
        typed.set_internal(raw)
        return typed

    def test_task_config(self) -> None:
        self.log("Testing: Task config survived migration")
        self.test_task_exists()
        assert self.tsk is not None
        assert (
            self.tsk._internal.type == "ni_analog_read"
        ), f"Expected type 'ni_analog_read', got '{self.tsk._internal.type}'"
        assert self.tsk.config.data_saving is True, "data_saving should be True"
        assert len(self.tsk.config.channels) == NUM_CHANNELS, (
            f"Expected {NUM_CHANNELS} channels, "
            f"got {len(self.tsk.config.channels)}"
        )

    def test_data(self) -> None:
        self.log("Testing: Data survived migration")
        for i in range(NUM_CHANNELS):
            ch = self.client.channels.retrieve(f"{CHANNEL_PREFIX}_{i}")
            data = ch.read(sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now()))
            assert len(data) > 0, f"Channel '{ch.name}' has no data after migration"

        self.test_start_and_stop()
