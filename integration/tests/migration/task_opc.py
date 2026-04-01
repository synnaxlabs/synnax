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

from tests.driver.opcua_task import OPCUAReadTaskCase
from tests.driver.task import create_channel, create_index

TASK_NAME = "mig_opc_read"
IDX_NAME = "mig_opc_idx"
CHANNEL_PREFIX = "mig_opc_float"
NUM_CHANNELS = 2


class TaskOpcMigration(OPCUAReadTaskCase):
    """Base class defining the migration test contract for OPC UA tasks.

    Subclasses must implement each test method — setup creates the state,
    verify checks it after migration.
    """

    task_name = TASK_NAME

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

    def run(self) -> None:
        self.test_task_config()
        self.test_data()

    def teardown(self) -> None:
        """Stop sims without deleting the task — it must survive across phases."""
        for sim in self.sims.values():
            if sim is not None:
                sim.stop()
        self.sims = {}
        self.sim = None

    @abstractmethod
    def test_task_config(self) -> None: ...

    @abstractmethod
    def test_data(self) -> None: ...


class TaskspcSetup(TaskOpcMigration):
    """Create an OPC UA read task, run it, and verify sample collection."""

    def test_task_config(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

    def test_data(self) -> None:
        self.test_start_and_stop()


class TasksOpVerify(TaskOpcMigration):
    """Verify OPC UA task data survived, settings intact, and task still runs."""

    def create(self, **kwargs: Any) -> sy.Task:
        """Retrieve the existing task instead of creating a new one."""
        tasks = self.client.tasks.retrieve(names=[TASK_NAME])
        assert (
            len(tasks) == 1
        ), f"Expected exactly 1 task named '{TASK_NAME}', got {len(tasks)}"
        return tasks[0]

    def _channel_keys(self, task: sy.Task) -> list[int]:
        return [ch["channel"] for ch in task.config["channels"]]

    def test_task_config(self) -> None:
        self.log("Testing: Task config survived migration")
        assert self.tsk is not None
        assert (
            self.tsk.type == "opc_read"
        ), f"Expected type 'opc_read', got '{self.tsk.type}'"
        assert self.tsk.config["data_saving"] is True, "data_saving should be True"
        assert len(self.tsk.config["channels"]) == NUM_CHANNELS, (
            f"Expected {NUM_CHANNELS} channels, "
            f"got {len(self.tsk.config['channels'])}"
        )

    def test_data(self) -> None:
        self.log("Testing: Data survived migration")
        for i in range(NUM_CHANNELS):
            ch = self.client.channels.retrieve(f"{CHANNEL_PREFIX}_{i}")
            data = ch.read(sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now()))
            assert len(data) > 0, f"Channel '{ch.name}' has no data after migration"

        self.log("Testing: Task still runs after migration")
        self.test_start_and_stop()
