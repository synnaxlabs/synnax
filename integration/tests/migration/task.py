#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Shared base classes for read-task migration tests.

Each protocol file (task_opc.py, task_modbus.py, task_ni.py) inherits from these
alongside its protocol-specific ReadTaskCase and only needs to define:

    - Module constants (TASK_NAME, IDX_NAME, CHANNEL_PREFIX, NUM_CHANNELS)
    - ``create_channels`` (protocol-specific channel types)
    - Class attributes: ``task_type``, ``task_class``, ``channel_prefix``,
      ``num_channels``, and optionally ``pre_start_sleep``.
"""

from abc import abstractmethod

import synnax as sy

from tests.driver.task import ReadTaskCase


class ReadTaskMigration(ReadTaskCase):
    """Base migration contract for read tasks.

    Provides ``run()``, ``teardown()``, and the abstract test methods.
    """

    def run(self) -> None:
        self.test_task_config()
        self.test_data()

    def teardown(self) -> None:
        """Stop task and sims without deleting — must survive across phases."""
        if self.tsk is not None:
            self.tsk.stop()
            self.log(f"Task '{self.task_name}' stopped")
        # OPC UA and Modbus inherit SimulatorCase which provides sims.
        if hasattr(self, "sims"):
            for sim in getattr(self, "sims").values():
                if sim is not None:
                    sim.stop()

    @abstractmethod
    def test_task_config(self) -> None: ...

    @abstractmethod
    def test_data(self) -> None: ...


class ReadTaskMigrationSetup(ReadTaskMigration):
    """Setup phase: create the task, verify it exists, collect samples."""

    def test_task_config(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()

    def test_data(self) -> None:
        self.test_start_and_stop()


class ReadTaskMigrationVerify(ReadTaskMigration):
    """Verify phase: retrieve existing task, assert config/data, re-run.

    Subclasses must set:
        task_type:       Task type string (e.g. ``"opc_read"``).
        task_class:      Typed task class (e.g. ``sy.opcua.ReadTask``).
        channel_prefix:  Channel name prefix (e.g. ``"mig_opc_float"``).
        num_channels:    Number of data channels created in setup.
        pre_start_sleep: Seconds to sleep before re-running start/stop (default 0).
    """

    task_type: str
    task_class: type
    channel_prefix: str
    num_channels: int
    pre_start_sleep: float = 0

    def create(
        self,
        *,
        device: sy.Device,
        task_name: str,
        sample_rate: sy.Rate,
        stream_rate: sy.Rate,
    ) -> sy.Task:
        """Retrieve the existing task instead of creating a new one."""
        tasks = self.client.tasks.retrieve(names=[self.task_name])
        assert (
            len(tasks) == 1
        ), f"Expected exactly 1 task named '{self.task_name}', got {len(tasks)}"
        raw = tasks[0]
        typed = self.task_class(**raw.config)
        typed.set_internal(raw)
        return typed

    def test_task_config(self) -> None:
        self.log("Testing: Task config survived migration")
        self.test_task_exists()
        assert self.tsk is not None
        assert (
            self.tsk._internal.type == self.task_type
        ), f"Expected type '{self.task_type}', got '{self.tsk._internal.type}'"
        assert self.tsk.config.data_saving is True, "data_saving should be True"
        assert len(self.tsk.config.channels) == self.num_channels, (
            f"Expected {self.num_channels} channels, "
            f"got {len(self.tsk.config.channels)}"
        )

    def test_data(self) -> None:
        self.log("Testing: Data survived migration")
        for i in range(self.num_channels):
            ch = self.client.channels.retrieve(f"{self.channel_prefix}_{i}")
            data = ch.read(sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now()))
            assert len(data) > 0, f"Channel '{ch.name}' has no data after migration"

        if self.pre_start_sleep:
            sy.sleep(self.pre_start_sleep)
        self.test_start_and_stop()
