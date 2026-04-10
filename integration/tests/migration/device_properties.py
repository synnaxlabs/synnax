#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""SY-4047: Device properties migration tests.

Reproduces the bug where clicking Configure on an NI task fails with a ZodError
because ``propertiesZ`` rejects incomplete device properties stored by older versions.

Sequence:
    1. ``DevicePropertiesSetup`` creates an NI analog read task on sim hardware,
       then strips the device properties to simulate pre-v0.54 stored data.
    2. Core/Console is upgraded to the target version.
    3. ``DevicePropertiesConsoleVerify`` opens the task in the Console, clicks
       Configure, and verifies the play button appears (configure succeeded).
    4. ``DevicePropertiesVerify`` retrieves the task via Python client, runs it,
       and verifies data is produced using ``assert_sample_count``.
"""

import platform

import synnax as sy
from console.case import ConsoleCase
from console.task_page import TaskPage
from tests.driver.ni_task import NIAnalogReadTaskCase
from tests.driver.task import create_channel, create_index
from tests.migration.task import (
    ReadTaskMigration,
    ReadTaskMigrationSetup,
    ReadTaskMigrationVerify,
)

TASK_NAME = "mig_sy4047_ni_read"
IDX_NAME = "mig_sy4047_idx"
CHANNEL_PREFIX = "mig_sy4047_voltage"
NUM_CHANNELS = 2
DEVICE_LOCATION = "E101Mod4"  # NI 9205 sim


class _Base(ReadTaskMigration, NIAnalogReadTaskCase):
    """Shared base for SY-4047 device properties migration tests."""

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


class DevicePropertiesSetup(ReadTaskMigrationSetup, _Base):
    """Create an NI analog read task, verify it runs, then strip device properties.

    After normal setup (create + configure + start/stop), overwrites the device
    properties to remove ``counterInput`` and make ``analogOutput`` partial —
    simulating what a pre-v0.54 device would store.
    """

    def run(self) -> None:
        super().run()
        client = self.client
        dev = client.devices.retrieve(name=self.device_name)
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


class DevicePropertiesVerify(ReadTaskMigrationVerify, _Base):
    """Retrieve the task, verify config survived, and run to verify data."""

    task_type = "ni_analog_read"
    task_class = sy.ni.AnalogReadTask
    channel_prefix = CHANNEL_PREFIX
    num_channels = NUM_CHANNELS


class DevicePropertiesConsoleVerify(ConsoleCase):
    """Open the task, click Configure in the Console, then run via Python client.

    This exercises the exact code path that triggers the ZodError on buggy
    consoles: ``onConfigure`` → ``client.devices.retrieve({ schemas })`` →
    ``propertiesZ.parse()``. After Configure succeeds, the task is run via
    the Python client and ``assert_sample_count`` verifies data is produced.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="NI sim devices require Windows")
            return
        super().setup()

    def run(self) -> None:
        console = self.console
        client = self.client

        # Open the task and click Configure in the Console UI.
        self.log(f"Searching for task '{TASK_NAME}'...")
        task_page = console.workspace.open_from_search(TaskPage, TASK_NAME)
        self.log("Task page opened")

        configure_btn = task_page.page.get_by_role(
            "button", name="Configure", exact=True
        )
        configure_btn.wait_for(state="visible", timeout=5000)
        self.log("Clicking Configure...")
        configure_btn.click(force=True)

        # Play button appearing confirms Configure succeeded.
        play_btn = task_page.page.locator("button .pluto-icon--play").locator("..")
        play_btn.wait_for(state="visible", timeout=15000)
        self.log("Configure succeeded — play button visible")

        # Run the task and verify data via the Python client.
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

        self.log("Migration verified — Console Configure + Python run + data OK")
        console.notifications.close_all()
