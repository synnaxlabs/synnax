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
    3. ``DevicePropertiesConsoleVerify`` opens the task and clicks Configure.
       On buggy v0.54: ZodError notification. On fixed: succeeds.
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
        if "analogOutput" in dev.properties:
            self.log(f"  analogOutput: {dev.properties['analogOutput']}")
        elif "analog_output" in dev.properties:
            self.log(f"  analog_output: {dev.properties['analog_output']}")

        has_counter = (
            "counterInput" in dev.properties or "counter_input" in dev.properties
        )
        assert not has_counter, "counterInput should have been stripped"
        self.log("Device properties stripped successfully")

    def teardown(self) -> None:
        pass


class DevicePropertiesConsoleVerify(ConsoleCase):
    """Open the NI task in the Console and click Configure.

    On the buggy v0.54 console, clicking Configure triggers a ZodError because
    ``propertiesZ`` rejects incomplete device properties during
    ``client.devices.retrieve``.

    Set ``expect_success`` to control the assertion:
      - True (default): assert Configure succeeds (for the fixed console).
      - False: assert Configure produces an error (for the buggy console).
    """

    expect_success: bool = True
    requires_platform: str | None = "windows"

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

    def _screenshot(self, label: str) -> None:
        name = type(self).__name__
        path = get_results_path(f"{name}_{label}.png")
        try:
            self.page.screenshot(path=path)
            self.log(f"Screenshot saved: {path}")
        except Exception as e:
            self.log(f"Screenshot failed: {e}")

    def run(self) -> None:
        console = self.console

        self.log(f"Searching for task '{TASK_NAME}'...")
        task_page = console.workspace.open_from_search(TaskPage, TASK_NAME)
        self.log("Task page opened")

        # Click Configure directly — don't use task_page.configure() because
        # it waits for the button to hide, which times out on error.
        configure_btn = task_page.page.get_by_role(
            "button", name="Configure", exact=True
        )
        configure_btn.wait_for(state="visible", timeout=5000)
        self.log("Configure button visible, clicking...")
        self._screenshot("before_configure")
        configure_btn.click(force=True)

        # Wait for either: button hides (success) or error notification appears.
        try:
            # Race: whichever comes first — button hidden or error notification.
            task_page.page.wait_for_selector(
                ".pluto-notification:has(svg[color*='error']), "
                ".pluto-notification:has-text('Failed')",
                state="visible",
                timeout=10000,
            )
            got_error = True
            self.log("Error notification appeared")
        except Exception:
            got_error = False

        self._screenshot("after_configure")

        # Also check if the button went hidden (success case).
        button_hidden = not configure_btn.is_visible()
        if button_hidden:
            self.log("Configure button hidden (success)")

        # Gather notification details for direct proof.
        notifications = console.notifications.check(timeout=2 * sy.TimeSpan.SECOND)
        errors = [n for n in notifications if n.get("type") == "error"]

        for n in notifications:
            self.log(f"Notification: type={n.get('type')} message={n.get('message')}")
            if n.get("description"):
                self.log(f"  Description: {n['description'][:500]}")

        # Log browser console errors for additional proof.
        for log_line in self._browser_logs:
            if any(kw in log_line.lower() for kw in ("error", "zod", "invalid")):
                self.log(f"  {log_line}")

        if self.expect_success:
            assert not got_error and len(errors) == 0, (
                f"Expected configure to succeed, but got error(s): "
                f"{[e.get('message', '') for e in errors]}"
            )
            self.log("Configure succeeded — device properties migration working")
        else:
            assert got_error or len(errors) > 0, (
                "Expected configure to fail with error, but no errors appeared. "
                f"Button hidden: {button_hidden}. "
                f"Browser errors: {[l for l in self._browser_logs if 'error' in l.lower()]}"
            )
            error_msg = errors[0].get("message", "") if errors else "no notification"
            error_desc = errors[0].get("description", "")[:200] if errors else ""
            self.log(f"Configure failed as expected: {error_msg}")
            if error_desc:
                self.log(f"  Error description: {error_desc}")

        console.notifications.close_all()


class DevicePropertiesConsoleBugVerify(DevicePropertiesConsoleVerify):
    """Run on the BUGGY console — expects the ZodError."""

    expect_success = False


class DevicePropertiesConsoleFixVerify(DevicePropertiesConsoleVerify):
    """Run on the FIXED console — expects configure to succeed."""

    expect_success = True
