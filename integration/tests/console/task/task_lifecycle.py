#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
from dataclasses import dataclass

import synnax as sy
from examples.opcua import OPCUASim

from console.case import ConsoleCase
from console.task_page import TaskPage
from framework.run_with_connection import run_scripts
from framework.utils import assert_link_format, get_random_name
from tests.driver.simulator_case import SimulatorCase

RANGE_NAME = f"Task Lifecycle Range {get_random_name()}"


@dataclass
class TaskDef:
    script: str
    name: str
    type: str
    channels: list[str]


TASKS = [
    TaskDef("read_task", "OPC UA Py - Read Task", "opc_read", ["NS=2;I=8", "NS=2;I=9"]),
    TaskDef(
        "read_task_array",
        "OPC UA Py - Read Task (Array)",
        "opc_read",
        ["NS=2;I=2", "NS=2;I=3"],
    ),
    TaskDef(
        "read_task_boolean",
        "OPC UA Py - Read Task (Boolean)",
        "opc_read",
        ["NS=2;I=13", "NS=2;I=14"],
    ),
    TaskDef(
        "write_task",
        "OPC UA Write Task Example",
        "opc_write",
        ["NS=2;I=18", "NS=2;I=19", "NS=2;I=20"],
    ),
]

READ_TASKS = [t for t in TASKS if "read" in t.type]
TASK_NAMES = [t.name for t in TASKS]


class TaskLifecycle(SimulatorCase, ConsoleCase):
    """Task Lifecycle Tests"""

    sim_classes = [OPCUASim]
    _cleanup_tasks: list[str]

    def setup_tasks(self) -> None:
        self._cleanup_tasks = list(TASK_NAMES)
        procs = run_scripts(
            self.synnax_connection,
            [f"examples.opcua.{t.script}" for t in TASKS],
        )
        for t in TASKS:
            self.console.tasks.wait_for_task(t.name)
        for proc in procs:
            proc.terminate()

    def teardown(self) -> None:
        self.log("Beginning teardown")
        for name in list(getattr(self, "_cleanup_tasks", [])):
            try:
                tasks = self.client.tasks.retrieve(names=[name])
                if tasks:
                    self.client.tasks.delete([t.key for t in tasks])
            except (sy.NotFoundError, TypeError):
                pass
        try:
            rng = self.client.ranges.retrieve(name=RANGE_NAME)
            self.client.ranges.delete(rng.key)
        except sy.QueryError:
            pass
        super().teardown()

    def run(self) -> None:
        self.setup_tasks()
        self.test_play_pause()
        self.test_data_saving()
        self.test_export_task()
        self.test_open_task_config()
        self.test_open_task_via_search()
        self.test_snapshot_to_active_range()
        self.test_rename_task()
        self.test_delete_task()

    def assert_data_saving(self, name: str, expected: bool) -> None:
        """Verify data saving state via the Python client, with polling."""
        for _ in range(10):
            task = self.client.tasks.retrieve(names=[name])[0]
            actual = task.config.get("dataSaving", task.config.get("data_saving"))
            if actual == expected:
                return
            sy.sleep(0.5)
        assert (
            actual == expected
        ), f"Task '{name}' data_saving should be {expected}, got {actual}"

    def test_play_pause(self) -> None:
        """Stop and start each task (individually and in groups)."""

        self.log("Testing: Individual task stop/start")
        for name in TASK_NAMES:
            self.console.tasks.stop_task(name)

        sy.sleep(0.1)

        for name in TASK_NAMES:
            self.console.tasks.start_task(name)

        self.log("Testing: Group task stop/start")
        # Set one task to stopped (1 stopped, 3 running)
        self.console.tasks.stop_task(TASK_NAMES[2])
        self.console.tasks.stop_tasks(TASK_NAMES)

        sy.sleep(0.1)

        # Set one task to running (1 running, 3 stopped)
        self.console.tasks.start_task(TASK_NAMES[1])
        self.console.tasks.start_tasks(TASK_NAMES)

    def test_data_saving(self) -> None:
        """Enable and disable data saving (individually and in groups)."""

        self.log("Testing: Individual data saving disable/enable")
        read_names = [t.name for t in READ_TASKS]

        for name in read_names:
            self.console.tasks.disable_data_saving(name)
        for name in read_names:
            self.assert_data_saving(name, False)

        sy.sleep(0.1)

        for name in read_names:
            self.console.tasks.enable_data_saving(name)
        for name in read_names:
            self.assert_data_saving(name, True)

        sy.sleep(0.1)

        self.log("Testing: Group data saving disable/enable")

        # Set one read task to data saving disabled (1 disabled, rest enabled)
        self.console.tasks.disable_data_saving(read_names[2])
        self.console.tasks.disable_data_saving_tasks(TASK_NAMES)
        for name in read_names:
            self.assert_data_saving(name, False)

        sy.sleep(0.1)

        # Set one read task to data saving enabled (1 enabled, rest disabled)
        self.console.tasks.enable_data_saving(read_names[1])
        self.console.tasks.enable_data_saving_tasks(TASK_NAMES)
        for name in read_names:
            self.assert_data_saving(name, True)

    def test_export_task(self) -> None:
        """Export a task via context menu and verify the JSON content."""
        t = TASKS[0]
        self.log(f"Testing: Export task '{t.name}'")
        exported = self.console.tasks.export_task(t.name)
        assert "type" in exported, "Exported JSON should contain a 'type' field"
        assert (
            exported["type"] == t.type
        ), f"Exported type should be '{t.type}', got '{exported['type']}'"
        assert "channels" in exported, "Exported JSON should contain 'channels'"
        assert len(exported["channels"]) > 0, "Exported channels should not be empty"

    def test_open_task_config(self) -> None:
        """Open each task config via context menu and verify contents."""
        for t in TASKS:
            self.log(f"Testing: Open config for '{t.name}'")
            toolbar_link = self.console.tasks.copy_link(t.name)
            task = self.client.tasks.retrieve(names=[t.name])[0]
            assert_link_format(toolbar_link, "task", str(task.key))
            self.console.tasks.open_task_config(t.name)
            pane = self.console.page.locator(f".console-task-configure--{t.type}")
            task_page = TaskPage(
                self.console.layout, self.client, t.name, pane_locator=pane
            )
            task_page.verify_config(t.channels)
            page_link = task_page.copy_link()
            assert toolbar_link == page_link, (
                f"Page link should match toolbar link. "
                f"Got '{page_link}', expected '{toolbar_link}'"
            )
        self.console.close_all_tabs()

    def test_open_task_via_search(self) -> None:
        """Open a task configuration via the search palette."""
        name = TASKS[3].name
        self.log(f"Testing: Open task config via search palette for '{name}'")
        task_page = self.console.workspace.open_from_search(TaskPage, name)
        assert (
            task_page.page_name == name
        ), f"Opened page name should be '{name}', got '{task_page.page_name}'"
        self.console.close_all_tabs()

    def test_snapshot_to_active_range(self) -> None:
        """Snapshot tasks to the active range (individual and group)."""
        self.console.ranges.create(RANGE_NAME, persisted=True)
        self.console.ranges.favorite(RANGE_NAME)
        self.console.ranges.set_active(RANGE_NAME)

        self.log("Testing: Snapshot single task to active range")
        self.console.tasks.snapshot_tasks([TASK_NAMES[0]], RANGE_NAME)

        self.log("Testing: Snapshot multiple tasks to active range")
        self.console.tasks.snapshot_tasks(TASK_NAMES[1:], RANGE_NAME)

    def test_rename_task(self) -> None:
        """Rename a task and verify synchronization across UI elements."""
        old_name = TASK_NAMES[0]
        new_name = "Renamed Read Task"
        self._cleanup_tasks.append(new_name)

        self.console.tasks.open_task_config(old_name)

        self.log(f"Testing: Rename task '{old_name}' to '{new_name}'")
        self.console.tasks.rename_task(old_name, new_name)

        self.log("Testing: Verify rename synchronization")
        self.console.layout.get_tab(new_name).wait_for(state="visible", timeout=5000)
        name_value = self.console.layout.get_input_field("Name")
        assert (
            name_value == new_name
        ), f"Task config Name field should show '{new_name}', got '{name_value}'"
        self.console.close_all_tabs()

    def test_delete_task(self) -> None:
        """Delete a single task and multiple tasks via context menu."""
        remaining = TASK_NAMES[1:] + ["Renamed Read Task"]

        self.log("Testing: Stop all tasks before deletion")
        self.console.tasks.stop_tasks(remaining)

        single = remaining[0]
        rest = remaining[1:]

        self.log(f"Testing: Delete single task '{single}'")
        self.console.tasks.delete_task(single)

        self.log("Testing: Delete multiple tasks")
        self.console.tasks.delete_tasks(rest)
