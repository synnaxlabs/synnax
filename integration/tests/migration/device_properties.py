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

These tests use the same sim NI device/task created by ``NIAnalogReadSetup``
(in ``task_ni.py``). The setup phase strips the device properties after task
creation. The verify phase opens the task in the Console and clicks Configure.

Sequence:
    1. ``NIAnalogReadSetup`` runs first (creates task on sim NI hardware).
    2. ``DevicePropertiesSetup`` strips the device properties to simulate old data.
    3. Core is upgraded to the target version.
    4. ``DevicePropertiesConsoleVerify`` opens the task and clicks Configure.
       On buggy v0.54: ZodError notification. On fixed: succeeds.
"""

import platform

import synnax as sy
from console.case import ConsoleCase
from console.task_page import TaskPage
from framework.test_case import TestCase

# Must match task_ni.py constants — we reuse the same task.
TASK_NAME = "mig_ni_analog_read"
DEVICE_LOCATION = "E101Mod4"


class DevicePropertiesSetup(TestCase):
    """Strip device properties on the NI sim device to simulate pre-v0.54 data.

    Must run AFTER ``NIAnalogReadSetup`` which creates the task and device.
    Removes ``counterInput`` and makes ``analogOutput`` partial.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="NI sim devices require Windows")
            return

    def run(self) -> None:
        client = self.client

        dev = client.devices.retrieve(location=DEVICE_LOCATION)
        self.log(f"Device '{dev.name}' (key={dev.key})")
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

    def run(self) -> None:
        console = self.console

        self.log(f"Searching for task '{TASK_NAME}'...")
        task_page = console.workspace.open_from_search(TaskPage, TASK_NAME)
        self.log("Task page opened")

        self.log("Clicking Configure...")
        task_page.configure()
        self.log("Configure button clicked")

        notifications = console.notifications.check(timeout=3 * sy.TimeSpan.SECOND)
        errors = [n for n in notifications if n.get("type") == "error"]

        if errors:
            self.log(f"Error notifications: {errors}")
        if notifications:
            self.log(f"All notifications: {notifications}")

        # Log browser console errors containing Zod-related keywords
        for log_line in self._browser_logs:
            if any(kw in log_line.lower() for kw in ("error", "zod", "invalid")):
                self.log(f"  {log_line}")

        if self.expect_success:
            assert len(errors) == 0, (
                f"Expected configure to succeed, but got error(s): "
                f"{[e.get('message', '') for e in errors]}"
            )
            self.log("Configure succeeded — device properties migration working")
        else:
            assert len(errors) > 0, (
                "Expected configure to fail with ZodError, but no errors appeared. "
                f"Browser errors: {[l for l in self._browser_logs if 'error' in l.lower()]}"
            )
            self.log(f"Configure failed as expected: {errors[0].get('message', '')}")

        console.notifications.close_all()


class DevicePropertiesConsoleBugVerify(DevicePropertiesConsoleVerify):
    """Run on the BUGGY console — expects the ZodError."""

    expect_success = False


class DevicePropertiesConsoleFixVerify(DevicePropertiesConsoleVerify):
    """Run on the FIXED console — expects configure to succeed."""

    expect_success = True
