#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""OPC UA invalid configuration integration tests.

Each test saves a task with an invalid setting and verifies the driver rejects
it with an error status when the task is started.
"""

from examples.opcua import OPCUASim

import synnax as sy
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    assert_start_rejected,
    cleanup_task,
    create_channel,
    create_index,
    run_and_expect_rejection,
)


class OPCUAInvalidConfig(SimulatorCase):
    """Verify the driver rejects invalid OPC UA task configurations.

    Tests (run sequentially):
        1. Nonexistent device — device key that doesn't exist.
        2. No enabled channels — all channels disabled.
        3. Invalid rates — sample rate less than stream rate.
        4. Nonexistent channel key — Synnax channel that doesn't exist.
        5. Invalid node ID — node that doesn't exist on the server.
        6. Duplicate channel — two tasks using the same channel.
    """

    sim_classes = [OPCUASim]

    def setup(self) -> None:
        super().setup()
        self.device = self.client.devices.retrieve(name=self.device_name)

    def run(self) -> None:
        self.test_nonexistent_device()
        self.test_no_enabled_channels()
        self.test_invalid_rates()
        self.test_nonexistent_channel_key()
        self.test_invalid_node_id()
        self.test_duplicate_channel()

    def test_nonexistent_device(self) -> None:
        """Configure a read task with a device key that doesn't exist."""
        self.log("Testing: Nonexistent device key")
        idx = create_index(self.client, "opcua_inv_dev_idx")
        task = sy.opcua.ReadTask(
            name="OPCUA Invalid Device Test",
            device="nonexistent_device_key_12345",
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            data_saving=True,
            channels=[
                sy.opcua.ReadChannel(
                    channel=create_channel(
                        self.client,
                        name="opcua_inv_dev_ch",
                        data_type=sy.DataType.FLOAT32,
                        index=idx.key,
                    ),
                    node_id="NS=2;I=8",
                    data_type="float32",
                ),
            ],
        )
        self._assert_deploy_fails(task, "nonexistent device")

    def test_no_enabled_channels(self) -> None:
        """Configure a read task with all channels disabled."""
        self.log("Testing: No enabled channels")
        idx = create_index(self.client, "opcua_inv_no_ch_idx")
        task = sy.opcua.ReadTask(
            name="OPCUA No Enabled Channels Test",
            device=self.device.key,
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            data_saving=True,
            channels=[
                sy.opcua.ReadChannel(
                    channel=create_channel(
                        self.client,
                        name="opcua_inv_no_ch",
                        data_type=sy.DataType.FLOAT32,
                        index=idx.key,
                    ),
                    node_id="NS=2;I=8",
                    data_type="float32",
                    enabled=False,
                ),
            ],
        )
        self._assert_deploy_fails(task, "no enabled channels")

    def test_invalid_rates(self) -> None:
        """Configure a read task with sample rate less than stream rate."""
        self.log("Testing: Invalid rates (sample_rate < stream_rate)")
        idx = create_index(self.client, "opcua_inv_rate_idx")
        task = sy.opcua.ReadTask(
            name="OPCUA Invalid Rate Test",
            device=self.device.key,
            sample_rate=10 * sy.Rate.HZ,
            stream_rate=100 * sy.Rate.HZ,
            data_saving=True,
            channels=[
                sy.opcua.ReadChannel(
                    channel=create_channel(
                        self.client,
                        name="opcua_inv_rate_ch",
                        data_type=sy.DataType.FLOAT32,
                        index=idx.key,
                    ),
                    node_id="NS=2;I=8",
                    data_type="float32",
                ),
            ],
        )
        self._assert_deploy_fails(task, "invalid rates")

    def test_nonexistent_channel_key(self) -> None:
        """Configure a read task with a Synnax channel key that doesn't exist."""
        self.log("Testing: Nonexistent Synnax channel key")
        task = sy.opcua.ReadTask(
            name="OPCUA Invalid Channel Key Test",
            device=self.device.key,
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            data_saving=True,
            channels=[
                sy.opcua.ReadChannel(
                    channel=999999999,
                    node_id="NS=2;I=8",
                    data_type="float32",
                ),
            ],
        )
        self._assert_deploy_fails(task, "nonexistent channel key")

    def test_invalid_node_id(self) -> None:
        """Start a read task with a node ID that doesn't exist on the server."""
        self.log("Testing: Invalid node ID (runtime)")
        idx = create_index(self.client, "opcua_inv_node_idx")
        task = sy.opcua.ReadTask(
            name="OPCUA Invalid Node ID Test",
            device=self.device.key,
            sample_rate=100 * sy.Rate.HZ,
            stream_rate=25 * sy.Rate.HZ,
            data_saving=True,
            channels=[
                sy.opcua.ReadChannel(
                    channel=create_channel(
                        self.client,
                        name="opcua_inv_node_ch",
                        data_type=sy.DataType.FLOAT32,
                        index=idx.key,
                    ),
                    node_id="NS=99;I=99999",
                    data_type="float32",
                ),
            ],
        )
        self._assert_deploy_fails(task, "invalid node ID")

    def test_duplicate_channel(self) -> None:
        """Configure and run two tasks that use the same channel."""
        self.log("Testing: Duplicate channel (two tasks on same channel)")
        idx = create_index(self.client, "opcua_dup_ch_idx")
        shared_ch_key = create_channel(
            self.client,
            name="opcua_dup_ch",
            data_type=sy.DataType.FLOAT32,
            index=idx.key,
        )

        def _make_task(name: str) -> sy.opcua.ReadTask:
            return sy.opcua.ReadTask(
                name=name,
                device=self.device.key,
                sample_rate=100 * sy.Rate.HZ,
                stream_rate=25 * sy.Rate.HZ,
                data_saving=True,
                channels=[
                    sy.opcua.ReadChannel(
                        channel=shared_ch_key,
                        node_id="NS=2;I=8",
                        data_type="float32",
                    ),
                ],
            )

        task_a = _make_task("OPCUA Dup Channel Task A")
        task_b = _make_task("OPCUA Dup Channel Task B")
        self.client.tasks.configure(task_a)
        self.log("  Task A configured")

        rejected = False
        try:
            with task_a.run():
                self.log("  Task A running")
                self.client.tasks.configure(task_b)
                self.log("  Task B configured (attempting run)")
                message = run_and_expect_rejection(self.client, task_b)
                if message is not None:
                    self.log(f"  Correctly rejected on run: {message}")
                    rejected = True
        finally:
            cleanup_task(self.client, task_a)
            cleanup_task(self.client, task_b)

        if not rejected:
            self.fail(
                "Driver did not reject second task using the "
                "same channel — both tasks ran simultaneously"
            )

    def _assert_deploy_fails(self, task: sy.Task, label: str) -> None:
        """Save a task and assert the driver rejects it on start."""
        message = assert_start_rejected(self.client, task, label)
        self.log(f"  Correctly rejected ({label}): {message}")
