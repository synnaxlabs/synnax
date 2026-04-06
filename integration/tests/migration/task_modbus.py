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

from tests.driver.modbus_task import ModbusReadTaskCase
from tests.driver.task import create_channel, create_index

TASK_NAME = "mig_modbus_read"
IDX_NAME = "mig_modbus_idx"
CHANNEL_PREFIX = "mig_modbus_reg"
NUM_CHANNELS = 2


class TaskModbusMigration(ModbusReadTaskCase):
    """Base class defining the migration test contract for Modbus tasks.

    Subclasses must implement each test method — setup creates the state,
    verify checks it after migration.
    """

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

    def run(self) -> None:
        self.test_task_config()
        self.test_data()

    def teardown(self) -> None:
        """Stop task and sims without deleting — must survive across phases."""
        if self.tsk is not None:
            self.tsk.stop()
            self.log(f"Task '{self.task_name}' stopped")
        for sim in self.sims.values():
            if sim is not None:
                sim.stop()
        self.sims = {}
        self.sim = None

    @abstractmethod
    def test_task_config(self) -> None: ...

    @abstractmethod
    def test_data(self) -> None: ...


class TaskModbusSetup(TaskModbusMigration):
    """Create a Modbus read task, run it, and verify sample collection."""

    def test_task_config(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

    def test_data(self) -> None:
        self.test_start_and_stop()


class TaskModbusVerify(TaskModbusMigration):
    """Verify Modbus task data survived, settings intact, and task still runs."""

    def create(self, **kwargs: Any) -> sy.modbus.ReadTask:
        """Retrieve the existing task instead of creating a new one."""
        tasks = self.client.tasks.retrieve(names=[TASK_NAME])
        assert (
            len(tasks) == 1
        ), f"Expected exactly 1 task named '{TASK_NAME}', got {len(tasks)}"
        raw = tasks[0]
        typed = sy.modbus.ReadTask(**raw.config)
        typed.set_internal(raw)
        return typed

    def test_task_config(self) -> None:
        self.log("Testing: Task config survived migration")
        self.test_task_exists()
        assert self.tsk is not None
        assert (
            self.tsk._internal.type == "modbus_read"
        ), f"Expected type 'modbus_read', got '{self.tsk._internal.type}'"
        assert self.tsk.config.data_saving is True, "data_saving should be True"
        assert len(self.tsk.config.channels) == NUM_CHANNELS, (
            f"Expected {NUM_CHANNELS} channels, " f"got {len(self.tsk.config.channels)}"
        )

    def test_data(self) -> None:
        self.log("Testing: Data survived migration")
        for i in range(NUM_CHANNELS):
            ch = self.client.channels.retrieve(f"{CHANNEL_PREFIX}_{i}")
            data = ch.read(sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now()))
            assert len(data) > 0, f"Channel '{ch.name}' has no data after migration"

        sy.sleep(2)
        self.test_start_and_stop()
