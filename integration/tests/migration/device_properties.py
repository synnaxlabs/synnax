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

Usage:
    1. Run ``DevicePropertiesSetup`` on any version with a driver (creates task,
       then strips device properties to simulate pre-v0.54 data).
    2. Upgrade to the target console version.
    3. Run ``DevicePropertiesConsoleVerify`` — on the buggy v0.54 console this
       triggers the ZodError; on the fixed console it succeeds.
"""

import platform

import synnax as sy
from console.case import ConsoleCase
from console.task_page import TaskPage
from framework.test_case import TestCase
from tests.driver.task import create_channel, create_index

TASK_NAME = "mig_sy4047_ni_read"
IDX_NAME = "mig_sy4047_idx"
CHANNEL_PREFIX = "mig_sy4047_voltage"
NUM_CHANNELS = 2
DEVICE_LOCATION = "E101Mod4"  # NI 9205


class DevicePropertiesSetup(TestCase):
    """Create an NI analog read task, then strip the device properties.

    After the task is configured via the driver, this overwrites the device
    properties on the server to remove ``counterInput`` and make
    ``analogOutput`` partial — matching what a pre-SY-3060 device would store.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="NI tasks require Windows")
            return

    def run(self) -> None:
        client = self.client

        # Resolve the NI device (registered by the driver's scan task).
        dev = client.devices.retrieve(location=DEVICE_LOCATION)
        self.log(f"Using device '{dev.name}' (key={dev.key})")

        # Create channels.
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

        # Create and configure the task via the driver.
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

        # Now strip the device properties to simulate old stored data.
        dev = client.devices.retrieve(key=dev.key)
        self.log(f"Original property keys: {list(dev.properties.keys())}")

        stripped = dict(dev.properties)
        # Remove counterInput (added in SY-3060, absent in pre-v0.39 data).
        stripped.pop("counterInput", None)
        stripped.pop("counter_input", None)
        # Make analogOutput partial (only portCount, missing stateIndex/channels).
        for key in ("analogOutput", "analog_output"):
            if key in stripped:
                port_key = "portCount" if key == "analogOutput" else "port_count"
                stripped[key] = {port_key: stripped[key].get(port_key, 0)}

        dev.properties = stripped
        client.devices.create(dev)

        # Verify the strip took effect.
        dev = client.devices.retrieve(key=dev.key)
        self.log(f"Stripped property keys: {list(dev.properties.keys())}")
        assert "counterInput" not in dev.properties, "counterInput should be stripped"
        assert "counter_input" not in dev.properties, "counter_input should be stripped"

    def teardown(self) -> None:
        # Don't delete — the task must survive for the verify phase.
        pass


class DevicePropertiesConsoleVerify(ConsoleCase):
    """Open the task in the Console and click Configure.

    On the buggy v0.54 console, this triggers a ZodError notification because
    ``propertiesZ`` rejects incomplete device properties during retrieval.
    On the fixed console, defaults fill in missing fields and it succeeds.

    Set ``expect_success = False`` to assert the bug is present,
    or ``expect_success = True`` (default) to assert the fix works.
    """

    expect_success: bool = True

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="NI tasks require Windows")
            return
        super().setup()

    def run(self) -> None:
        console = self.console

        task_page = console.workspace.open_from_search(TaskPage, TASK_NAME)

        # Click Configure — triggers onConfigure → client.devices.retrieve
        # with Device.SCHEMAS → propertiesZ validation on incomplete data.
        task_page.configure()

        notifications = console.notifications.check(timeout=3 * sy.TimeSpan.SECOND)
        errors = [n for n in notifications if n.get("type") == "error"]

        if self.expect_success:
            assert len(errors) == 0, (
                f"Expected configure to succeed, but got error(s): "
                f"{[e.get('message', '') for e in errors]}"
            )
            self.log("Configure succeeded — device properties migration working")
        else:
            assert len(errors) > 0, (
                "Expected configure to fail with ZodError, but no errors appeared"
            )
            self.log(f"Configure failed as expected: {errors[0].get('message', '')}")

        console.notifications.close_all()


class DevicePropertiesConsoleBugVerify(DevicePropertiesConsoleVerify):
    """Run on the BUGGY console — expects the ZodError."""

    expect_success = False


class DevicePropertiesConsoleFixVerify(DevicePropertiesConsoleVerify):
    """Run on the FIXED console — expects configure to succeed."""

    expect_success = True
