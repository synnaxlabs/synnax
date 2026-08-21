#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, TypeVar

import synnax as sy
from console.case import ConsoleCase
from console.task_page import TaskPage
from tests.driver.simulator_case import SimulatorCase
from tests.driver.task import (
    assert_sample_rate,
    collect_samples,
    delete_channels,
    send_and_verify_commands,
)
from x import random_name

PageT = TypeVar("PageT", bound=TaskPage)
REJECTED_LEVELS = ("error", "warning")


def _keys_in(value: Any) -> set[int]:
    """Collect every integer nested in a device property subtree."""
    if isinstance(value, bool):
        return set()
    if isinstance(value, int):
        return {value}
    if isinstance(value, dict):
        return set().union(*(_keys_in(v) for v in value.values()))
    if isinstance(value, list):
        return set().union(*(_keys_in(v) for v in value))
    return set()


class ConsoleTaskCase(SimulatorCase, ConsoleCase):
    """ConsoleCase against a simulated device. Tracks the tasks a test creates
    through the Console and deletes them, plus the channels deploy created, in
    teardown. The steps and assertions shared by the protocol cases take the page
    they act on."""

    STREAM_RATE: sy.Rate = 10 * sy.Rate.HZ
    TASK_DURATION: sy.TimeSpan = 1 * sy.TimeSpan.SECOND
    task_names: list[str]

    def setup(self) -> None:
        self.task_names = []
        super().setup()

    def teardown(self) -> None:
        try:
            with self._try_to("delete tasks and their channels"):
                self._delete_tasks_and_channels()
        finally:
            super().teardown()

    def create_page(self, page_class: type[PageT], prefix: str) -> PageT:
        """Create a task page with a unique name and track it for teardown."""
        name = f"{prefix} {random_name()}"
        self.task_names.append(name)
        return self.console.pages.create(page_class, name)

    def retrieve_task(self, name: str) -> sy.Task:
        """Retrieve a task created through the Console by its exact name."""
        tasks = [t for t in self.client.tasks.retrieve(names=[name]) if t.name == name]
        assert len(tasks) == 1, f"Expected one task named '{name}', got {len(tasks)}"
        return tasks[0]

    @staticmethod
    def channel_keys(task: sy.Task) -> list[int]:
        """Return the Synnax channel keys a read or write task config points at."""
        keys: list[int] = []
        for ch in task.config.get("channels", []):
            key = ch.get("channel", ch.get("cmd_channel", ch.get("cmdChannel")))
            if key:
                keys.append(int(key))
        return keys

    def set_rates(self, page: TaskPage) -> None:
        """Set the case's sample and stream rates on the form."""
        page.set_parameters(
            sample_rate=int(self.SAMPLE_RATE), stream_rate=int(self.STREAM_RATE)
        )

    def test_set_rates(self, page: TaskPage) -> None:
        """Set the sample and stream rates and read them back."""
        self.log("Testing: Set sample and stream rate")
        self.set_rates(page)
        layout = self.console.layout
        rates = (("Sample rate", self.SAMPLE_RATE), ("Stream rate", self.STREAM_RATE))
        for label, rate in rates:
            shown = layout.get_input_field(label)
            assert shown == str(int(rate)), (
                f"{label} should be {int(rate)}, got '{shown}'"
            )

    def test_deploy(self, page: TaskPage) -> None:
        """Deploy from the form; the toolbar shows the task running."""
        self.log("Testing: Deploy task")
        page.deploy()
        self.console.tasks.wait_for_state(page.page_name, "running")

    def stop(self, page: TaskPage) -> None:
        """Stop from the form and wait for the toolbar to show the task stopped."""
        page.stop()
        self.console.tasks.wait_for_state(page.page_name, "stopped")

    def test_stop(self, page: TaskPage) -> None:
        self.log("Testing: Stop task")
        self.stop(page)

    def test_reopen_config(self, page: TaskPage, expected: list[str]) -> None:
        """Close every tab, reopen the task from the toolbar, and check its channels."""
        self.log("Testing: Reopen task configuration")
        self.console.close_all_tabs()
        self.console.tasks.open_config(page.page_name)
        page.verify_config(expected)

    def assert_sample_count(self, page: TaskPage, keys: list[int]) -> None:
        """Hold the running task, stop it from the form, then count the samples on
        ``keys``, as the driver read task tests do."""
        start = collect_samples(self.client, keys, self.TASK_DURATION)
        self.stop(page)
        assert_sample_rate(
            self.client,
            channel_keys=keys,
            start=start,
            sample_rate=float(self.SAMPLE_RATE),
            duration=self.TASK_DURATION,
            task_name=page.page_name,
        )

    def send_commands(self, page: TaskPage, task: sy.Task) -> None:
        """Write two rounds of commands to the task's channels and check they stream
        back without the task reporting an error."""
        send_and_verify_commands(
            self.client,
            cmd_keys=self.channel_keys(task),
            writer_name=f"{page.page_name}_test_writer",
            task_name=page.page_name,
            task_key=task.key,
        )

    def assert_not_deployed(self, page: TaskPage, label: str) -> None:
        status = page.status()
        assert status["level"] == "disabled", (
            f"Task should not be deployed ({label}), "
            f"but the status is '{status['level']}': '{status['msg']}'"
        )

    def assert_device_required(self, page: TaskPage, label: str) -> None:
        """Deploy without a device; the form blocks it at the ``label`` field."""
        page.deploy(expect=None)
        message = page.field_help_text(label)
        assert message == "Device is required", (
            f"Device field should report it is required, got '{message}'"
        )
        self.assert_not_deployed(page, "no device")

    def assert_rates_rejected(self, page: TaskPage) -> None:
        """Deploy with a stream rate above the sample rate; the form blocks it."""
        page.set_parameters(sample_rate=10, stream_rate=100)
        page.deploy(expect=None)
        message = page.field_help_text("Stream rate")
        assert "less than or equal to the sample rate" in message, (
            f"Stream rate field should report the rate conflict, got '{message}'"
        )
        self.assert_not_deployed(page, "invalid rates")

    def assert_driver_rejects(self, page: TaskPage, label: str) -> None:
        """Deploy; the Driver rejects the task with an error or warning status."""
        page.deploy(expect=None)
        status = page.wait_for_status_level(REJECTED_LEVELS)
        self.log(f"  Correctly rejected ({label}): {status['msg']}")

    def _delete_tasks_and_channels(self) -> None:
        keys: set[int] = set()
        for name in self.task_names:
            try:
                tasks = self.client.tasks.retrieve(names=[name])
            except sy.NotFoundError:
                continue
            for task in tasks:
                keys.update(self.channel_keys(task))
                self.client.tasks.delete(task.key)
        # Deploy records every channel it creates on the device, including ones a
        # later config edit no longer references.
        try:
            device = self.client.devices.retrieve(name=self.sim_classes[0].device_name)
            for group in ("read", "write"):
                keys.update(_keys_in(device.properties.get(group, {})))
        except sy.NotFoundError:
            pass
        delete_channels(self.client, list(keys))
