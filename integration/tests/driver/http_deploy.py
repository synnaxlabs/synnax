#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Deploy-on-start integration tests.

Verifies that task configs deploy on the start command: a start picks up the
latest saved config, metadata-only saves never restart a running instance, and
a running instance survives its row moving to another rack.
"""

from examples.http_sim import HTTPSim

import synnax as sy
from synnax import http
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import cleanup_task, create_channel, create_index
from x import random_name


class HTTPDeployOnStart(SimulatorCase):
    """Verify deploy-on-start semantics against the driver.

    Tests (run sequentially):
        1. Start deploys latest config — a config saved between starts is
           picked up, reported via a changed status config hash, and an
           unchanged config never rebuilds the instance.
        2. Rename does not restart — a metadata-only save leaves the running
           instance streaming with its config hash untouched.
        3. Rack move keeps instance running — moving the row to another rack
           leaves the live instance streaming, and stop still reaches it.
    """

    sim_classes = [HTTPSim]

    def setup(self) -> None:
        super().setup()
        self.device = self.client.devices.retrieve(name=self.device_name)

    def run(self) -> None:
        self.test_start_deploys_latest_config()
        self.test_rename_does_not_restart()
        self.test_rack_move_keeps_instance_running()

    def _make_task(self, name: str, ch_name: str, rate: float) -> sy.http.ReadTask:
        idx = create_index(self.client, f"{ch_name}_idx")
        return sy.http.ReadTask(
            name=name,
            device=self.device.key,
            rate=rate,
            data_saving=True,
            endpoints=[
                http.ReadEndpoint(
                    path="/api/v1/data",
                    method="GET",
                    fields=[
                        http.ReadField(
                            pointer="/temperature",
                            channel=create_channel(
                                self.client,
                                name=ch_name,
                                data_type=sy.DataType.FLOAT64,
                                index=idx.key,
                            ),
                            data_type="float64",
                        ),
                    ],
                ),
            ],
        )

    def _start(self, task: sy.Task) -> sy.task.Status:
        """Start the task and return the driver's acknowledgment status."""
        status = task._internal.execute_command_sync("start", timeout=15)
        if status.variant == "error":
            self.fail(f"start failed: {status.message}")
        return status

    def _channel_keys(self, task: sy.Task) -> list[int]:
        return [
            field.channel
            for ep in task.config.endpoints
            for field in ep.fields
            if field.channel != 0
        ]

    def _assert_streaming(self, task: sy.Task, label: str) -> None:
        with self.client.open_streamer(self._channel_keys(task)) as streamer:
            frame = streamer.read(timeout=10)
            assert frame is not None, f"task is not streaming data {label}"

    def test_start_deploys_latest_config(self) -> None:
        """Save a new rate between starts and verify the deploy picks it up."""
        self.log("Testing: Start deploys the latest saved config")
        task = self._make_task("HTTP Deploy Sync Test", "http_deploy_sync_ch", 10)
        try:
            self.client.tasks.configure(task)
            first = self._start(task)
            assert first.details is not None and first.details.running, (
                "task not running after start"
            )
            assert first.details.config_hash != "", "status is missing a config hash"
            self.log(f"  Deployed with hash {first.details.config_hash}")

            task.config.rate = 20
            self.client.tasks.configure(task)
            second = self._start(task)
            assert second.details is not None
            assert second.details.config_hash != first.details.config_hash, (
                "config hash unchanged after redeploying a changed config"
            )
            self.log(f"  Redeployed with hash {second.details.config_hash}")

            third = self._start(task)
            assert third.details is not None
            assert third.details.config_hash == second.details.config_hash, (
                "unchanged config was rebuilt on start"
            )
            self.log("  Unchanged config kept the same instance")
            task.stop()
        finally:
            cleanup_task(self.client, task)

    def test_rename_does_not_restart(self) -> None:
        """Rename a running task and verify the instance is untouched."""
        self.log("Testing: Rename does not restart the running task")
        task = self._make_task("HTTP Deploy Rename Test", "http_deploy_rename_ch", 10)
        new_name = "HTTP Deploy Rename Test (Renamed)"
        try:
            self.client.tasks.configure(task)
            started = self._start(task)
            assert started.details is not None
            hash_before = started.details.config_hash

            task._internal.name = new_name
            self.client.tasks.configure(task)
            retrieved = self.client.tasks.retrieve(task.key)
            assert retrieved.name == new_name, (
                f"rename did not persist: {retrieved.name}"
            )

            self._assert_streaming(task, "after rename")
            self.log("  Instance kept streaming through the rename")

            restarted = self._start(task)
            assert restarted.details is not None
            assert restarted.details.config_hash == hash_before, (
                "metadata-only save rebuilt the instance on the next start"
            )
            self.log("  Next start reused the same instance")
            task.stop()
        finally:
            cleanup_task(self.client, task)

    def test_rack_move_keeps_instance_running(self) -> None:
        """Move the row to another rack and verify the instance keeps running."""
        self.log("Testing: Rack move leaves the live instance running")
        task = self._make_task("HTTP Deploy Move Test", "http_deploy_move_ch", 10)
        ghost = self.client.racks.create(name=f"deploy_ghost_{random_name()}")
        try:
            self.client.tasks.configure(task)
            self._start(task)

            task._internal.rack = ghost.key
            self.client.tasks.configure(task)
            self._assert_streaming(task, "after rack move")
            self.log("  Instance kept streaming after the rack move")

            # Non-start commands route to whichever driver holds the live
            # instance, even when the row points at another rack.
            task.stop()
            self.log("  Stop reached the live instance")
        finally:
            cleanup_task(self.client, task)
