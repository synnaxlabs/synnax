#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import platform

import synnax as sy
from console.case import ConsoleCase
from console.task_page import TaskPage
from framework.test_case import TestCase
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


class NIAnalogReadMigration(ReadTaskMigration, NIAnalogReadTaskCase):
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


class NIAnalogReadSetup(ReadTaskMigrationSetup, NIAnalogReadMigration):
    """Create an NI analog read task, run it, and verify sample collection."""


class NIAnalogReadVerify(ReadTaskMigrationVerify, NIAnalogReadMigration):
    """Verify NI analog read task data survived, settings intact, and task still runs."""

    task_type = "ni_analog_read"
    task_class = sy.ni.AnalogReadTask
    channel_prefix = CHANNEL_PREFIX
    num_channels = NUM_CHANNELS


class NIAnalogReadConsoleVerify(ReadTaskConsoleVerify):
    """Verify the NI analog read task configuration renders correctly in the console UI."""

    task_name = TASK_NAME
    expected_channels = [f"{CHANNEL_PREFIX}_{i}" for i in range(NUM_CHANNELS)]
    requires_platform = "windows"


# ── SY-4047: Device properties migration ─────────────────────────


class NIDevicePropertiesSetup(TestCase):
    """Strip device properties after NIAnalogReadSetup to simulate pre-v0.54 data.

    Must run AFTER NIAnalogReadSetup. Removes ``counterInput`` and makes
    ``analogOutput`` partial on the same device used by the NI read task.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="NI sim devices require Windows")
            return

    def run(self) -> None:
        client = self.client
        dev = client.devices.retrieve(location=DEVICE_LOCATION)
        self.log(f"Stripping properties for '{dev.name}' (key={dev.key})")
        self.log(f"  Original keys: {list(dev.properties.keys())}")

        stripped = dict(dev.properties)
        stripped.pop("counterInput", None)
        stripped.pop("counter_input", None)
        for key in ("analogOutput", "analog_output"):
            if key in stripped and isinstance(stripped[key], dict):
                port_key = "portCount" if key == "analogOutput" else "port_count"
                stripped[key] = {port_key: stripped[key].get(port_key, 0)}

        dev.properties = stripped
        client.devices.create(dev)

        dev = client.devices.retrieve(key=dev.key)
        self.log(f"  Stripped keys: {list(dev.properties.keys())}")
        has_counter = (
            "counterInput" in dev.properties or "counter_input" in dev.properties
        )
        assert not has_counter, "counterInput should have been stripped"
        self.log("Device properties stripped successfully")

    def teardown(self) -> None:
        pass


class NIDevicePropertiesConsoleVerify(ConsoleCase):
    """SY-4047: Open the NI task, click Configure, then run via Python client.

    Exercises the buggy code path: ``onConfigure`` →
    ``client.devices.retrieve({ schemas })`` → ``propertiesZ.parse()``.
    On buggy consoles the ZodError prevents Configure from succeeding.
    After Configure, runs the task via Python client to verify data.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="NI sim devices require Windows")
            return
        super().setup()

    def run(self) -> None:
        console = self.console
        client = self.client

        self.log(f"Searching for task '{TASK_NAME}'...")
        task_page = console.workspace.open_from_search(TaskPage, TASK_NAME)
        self.log("Task page opened")

        configure_btn = task_page.page.get_by_role(
            "button", name="Configure", exact=True
        )
        configure_btn.wait_for(state="visible", timeout=5000)
        self.log("Clicking Configure...")
        configure_btn.click(force=True)

        play_btn = task_page.page.locator("button .pluto-icon--play").locator("..")
        play_btn.wait_for(state="visible", timeout=15000)
        self.log("Configure succeeded — play button visible")

        tasks = client.tasks.retrieve(names=[TASK_NAME])
        assert len(tasks) == 1, f"Expected 1 task, got {len(tasks)}"
        raw = tasks[0]
        task = sy.ni.AnalogReadTask(**raw.config)
        task.set_internal(raw)

        channel_keys = [ch.channel for ch in task.config.channels]
        self.log(f"Running task (key={task.key}, channels={channel_keys})...")
        with task.run():
            with client.open_streamer(channel_keys) as streamer:
                frame = streamer.read(timeout=30)
                assert frame is not None, "Task did not produce data"
            self.log("Data received")
            sy.sleep(1)

        self.log("SY-4047 verified — Console Configure + run + data OK")
        console.notifications.close_all()
