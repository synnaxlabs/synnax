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
       Configure, then runs the task and verifies data is produced.
"""

import platform

import synnax as sy
from console.case import ConsoleCase
from console.task_page import TaskPage
from framework.test_case import TestCase
from framework.utils import get_results_path
from tests.driver.task import create_channel, create_index

TASK_NAME = "mig_sy4047_ni_read"
IDX_NAME = "mig_sy4047_idx"
CHANNEL_PREFIX = "mig_sy4047_voltage"
NUM_CHANNELS = 2
DEVICE_LOCATION = "E101Mod4"  # NI 9205 sim


class DevicePropertiesSetup(TestCase):
    """Create an NI analog read task on sim hardware, then strip device properties.

    Creates the task via ``tasks.configure`` (driver ACK) so it's fully valid,
    then overwrites the device properties to remove ``counterInput`` and make
    ``analogOutput`` partial — simulating what a pre-v0.54 device would store.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="NI sim devices require Windows")
            return

    def run(self) -> None:
        client = self.client

        # Resolve the sim NI device (registered by the driver's scan task).
        dev = client.devices.retrieve(location=DEVICE_LOCATION)
        self.log(f"Device '{dev.name}' (key={dev.key})")

        # Create channels and task.
        idx = create_index(client, IDX_NAME)
        channels = [
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
        task = sy.ni.AnalogReadTask(
            name=TASK_NAME,
            device=dev.key,
            sample_rate=50 * sy.Rate.HZ,
            stream_rate=10 * sy.Rate.HZ,
            data_saving=True,
            channels=channels,
        )
        client.tasks.configure(task)
        self.log(f"Task '{TASK_NAME}' configured (key={task.key})")

        # Verify it runs.
        with task.run():
            sy.sleep(1)
        self.log("Task ran successfully")

        # Strip device properties to simulate old stored data.
        dev = client.devices.retrieve(key=dev.key)
        self.log(f"  Original property keys: {list(dev.properties.keys())}")

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
        self.log(f"  Stripped property keys: {list(dev.properties.keys())}")

        has_counter = (
            "counterInput" in dev.properties or "counter_input" in dev.properties
        )
        assert not has_counter, "counterInput should have been stripped"
        self.log("Device properties stripped successfully")

    def teardown(self) -> None:
        pass


class DevicePropertiesConsoleVerify(ConsoleCase):
    """Open the NI task in the Console, click Configure, run, and verify data.

    On the buggy v0.54 console, clicking Configure triggers a ZodError because
    ``propertiesZ`` rejects incomplete device properties during
    ``client.devices.retrieve``. On the fixed console, Configure succeeds,
    the task can be run, and data channels contain samples.
    """

    requires_platform: str | None = "windows"

    def _screenshot(self, label: str) -> None:
        name = type(self).__name__
        path = get_results_path(f"{name}_{label}.png")
        try:
            self.page.screenshot(path=path)
            self.log(f"Screenshot saved: {path}")
        except Exception as e:
            self.log(f"Screenshot failed: {e}")

    def setup(self) -> None:
        if (
            self.requires_platform is not None
            and platform.system().lower() != self.requires_platform
        ):
            self.auto_pass(
                msg=f"Requires {self.requires_platform}, "
                f"running on {platform.system().lower()}"
            )
            return
        super().setup()

    def run(self) -> None:
        console = self.console
        client = self.client

        self.log(f"Searching for task '{TASK_NAME}'...")
        task_page = console.workspace.open_from_search(TaskPage, TASK_NAME)
        self.log("Task page opened")

        self._screenshot("before_configure")

        # Click Configure.
        configure_btn = task_page.page.get_by_role(
            "button", name="Configure", exact=True
        )
        configure_btn.wait_for(state="visible", timeout=5000)
        self.log("Clicking Configure...")
        configure_btn.click(force=True)

        # Wait for the play button — indicates configure succeeded.
        play_btn = task_page.page.locator("button .pluto-icon--play").locator("..")
        play_btn.wait_for(state="visible", timeout=15000)
        self.log("Play button appeared — configure succeeded")

        self._screenshot("after_configure")

        # Close notifications that may overlay the play button.
        console.notifications.close_all()

        # Run the task via the Console play button.
        self.log("Running task...")
        play_btn.dispatch_event("click")
        sy.sleep(2)

        self._screenshot("after_run")

        # Stop the task.
        stop_btn = task_page.page.locator("button .pluto-icon--pause").locator("..")
        stop_btn.dispatch_event("click")
        self.log("Task stopped")

        # Verify data was produced via the Python client.
        for i in range(NUM_CHANNELS):
            ch = client.channels.retrieve(f"{CHANNEL_PREFIX}_{i}")
            data = ch.read(sy.TimeRange(sy.TimeStamp.MIN, sy.TimeStamp.now()))
            assert len(data) > 0, f"Channel '{ch.name}' has no data after configure+run"
            self.log(f"Channel '{ch.name}' has {len(data)} samples")

        self.log("Device properties migration verified — configure + run + data OK")

        console.notifications.close_all()
