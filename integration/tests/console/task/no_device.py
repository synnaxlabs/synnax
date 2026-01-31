#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import platform
import random

import synnax as sy

from console.case import ConsoleCase
from console.task.analog_read import AnalogRead


class NoDevice(ConsoleCase):
    """
    Verify status message/level when attempting to
    configure and run a task with selected devices
    that are not present.
    """

    def setup(self) -> None:
        if platform.system().lower() != "windows":
            self.auto_pass(msg="Windows DAQmx drivers required")
        super().setup()

    def run(self) -> None:
        """
        Test Opening and closing pages
        """

        self.log("Creating NI Analog Read Task Page")
        rand_int = random.randint(100, 999)
        ni_ai = self.console.workspace.create_task(
            "NI Analog Read Task",
            f"USB-6000_{rand_int}",
        )

        rack_name = f"TestRack_{rand_int}"
        dev_name = f"USB-6000_{rand_int}"
        self.create_rack(rack_name, dev_name)
        self.initial_assertion(ni_ai)
        self.configure_without_channels(ni_ai)
        self.nominal_configuration(ni_ai, rack_name, dev_name)

    def create_rack(self, rack_name: str, dev_name: str) -> None:
        self.log(f"Creating {rack_name} and devices")

        client = self.client
        rack = client.racks.create(name=rack_name)
        client.devices.create(
            [
                sy.Device(
                    key="a0e37b26-5401-413e-8e65-c7ad9d9afd70",
                    rack=rack.key,
                    name=dev_name,
                    make="NI",
                    model=dev_name,
                    location="dev3",
                    identifier="dev3",
                ),
            ]
        )

    def initial_assertion(self, ni_ai: AnalogRead) -> None:
        """Initial assertion of task status"""
        status = ni_ai.status()
        msg = status["msg"]
        level = status["level"]

        level_expected = "disabled"
        msg_expected = "Task has not been configured"

        assert (
            level_expected == level
        ), f"Task status level <{level}> should be <{level_expected}>"
        assert (
            msg_expected == msg
        ), f"Task status msg <{msg}> should be <{msg_expected}>"

    def configure_without_channels(self, ni_ai: AnalogRead) -> None:
        """Configure without defining channels"""
        ni_ai.configure()

        # Assert error notification
        notifications = self.console.layout.notifications.check(timeout=5)
        msg = notifications[0]["message"]
        msg_expected = "Failed to update Task"
        assert (
            msg_expected == msg
        ), f"Notification msg is <{msg}>, should be <{msg_expected}>"

        # Assert Task error status
        status = ni_ai.status()
        level = status["level"]
        msg = status["msg"]
        level_expected = "error"
        msg_expected = "Failed to update Task"
        assert (
            level_expected == level
        ), f"Task status level <{level}> should be <{level_expected}>"
        assert (
            msg_expected == msg
        ), f"Task status msg <{msg}> should be <{msg_expected}>"

    def nominal_configuration(
        self, ni_ai: AnalogRead, rack_name: str, dev_name: str
    ) -> None:
        """Nominal configuration of task"""
        # Add channel
        ni_ai.add_channel(
            name="new_channel",
            chan_type="Voltage",
            device=dev_name,
            dev_name="usb_6000",
        )

        self.log("Configuring task")
        ni_ai.configure()
        self.log("Running task")
        ni_ai.run()

        # Status assertions
        status = ni_ai.status()
        level = status["level"]
        msg = status["msg"]

        while level == "loading" and self.should_continue:
            sy.sleep(0.1)
            status = ni_ai.status()
            level = status["level"]
            msg = status["msg"]

        level_expected = "warning"
        msg_expected = f"{rack_name} is not running"

        assert msg_expected in msg, f"<{msg}> should be <{msg_expected}>"
        assert level_expected == level, f"<{level}> should be <{level_expected}>"
