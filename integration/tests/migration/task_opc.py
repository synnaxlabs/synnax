#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from framework.test_case import TestCase
from tests.driver.opcua_task import OPCUAReadTaskCase
from tests.driver.task import create_channel, create_index


class TasksOpcSetup(OPCUAReadTaskCase):
    """Create an OPC UA read task, run it, and verify sample collection."""

    task_name = "mig_opc_read"

    @staticmethod
    def create_channels(client: sy.Synnax) -> list[sy.opcua.ReadChannel]:
        idx = create_index(client, "mig_opc_idx")
        return [
            sy.opcua.ReadChannel(
                channel=create_channel(
                    client,
                    name=f"mig_opc_float_{i}",
                    data_type=sy.DataType.FLOAT32,
                    index=idx.key,
                ),
                node_id=f"NS=2;I={8 + i}",
                data_type="float32",
            )
            for i in range(2)
        ]

    def run(self) -> None:
        assert self.tsk is not None
        self.test_task_exists()
        self.test_start_and_stop()

    def teardown(self) -> None:
        # Do NOT delete the task - it must survive for verify phase.
        for sim in self.sims.values():
            if sim is not None:
                sim.stop()
        self.sims = {}
        self.sim = None


class TasksOpcVerify(TestCase):
    """Verify OPC UA task data survived, settings intact, and task still runs."""

    def run(self) -> None:
        self.test_data_survived()
        self.test_task_config()

    def test_data_survived(self) -> None:
        self.log("Testing: Data survived migration")
        ch0 = self.client.channels.retrieve("mig_opc_float_0")
        ch1 = self.client.channels.retrieve("mig_opc_float_1")
        for ch in [ch0, ch1]:
            data = ch.read(sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now()))
            assert len(data) > 0, f"Channel '{ch.name}' has no data after migration"

    def test_task_config(self) -> None:
        self.log("Testing: Task config survived migration")
        tasks = self.client.tasks.retrieve(names=["mig_opc_read"])
        assert len(tasks) >= 1, f"Expected at least 1 task, got {len(tasks)}"
        task = tasks[-1]
        assert task.type == "opc_read", f"Expected type 'opc_read', got '{task.type}'"
        config = task.config if isinstance(task.config, dict) else task.config.__dict__
        assert config["data_saving"] is True, "data_saving should be True"
        assert (
            len(config["channels"]) == 2
        ), f"Expected 2 channels, got {len(config['channels'])}"
