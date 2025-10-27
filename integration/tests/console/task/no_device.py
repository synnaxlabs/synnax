#  Copyright 2025 Synnax Labs, Inc.
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


class NoDevice(ConsoleCase):
    """
    Verify status message/level when attempting to
    configure and run a task with selected devices
    that are not present.
    """

    def run(self) -> None:
        """
        Test Opening and closing pages
        """

        self.log("Creating NI Analog Read Task Page")
        self.console.ni_ai.new()

        rand_int = random.randint(100, 999)
        rack_name = f"TestRack_{rand_int}"
        dev_name = f"USB-6000_{rand_int}"
        self.create_rack(rack_name, dev_name)
        self.initial_assertion()
        self.configure_without_channels()
        self.nominal_configuration(rack_name, dev_name)

    def create_rack(self, rack_name: str, dev_name: str) -> None:
        self.log(f"Creating {rack_name} and devices")

        client = self.client
        rack = client.hardware.racks.create(name=rack_name)
        client.hardware.devices.create(
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

    def initial_assertion(self) -> None:
        """Initial assertion of task status"""
        console = self.console

        status = console.ni_ai.status()
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

    def configure_without_channels(self) -> None:
        """Configure without defining channels"""
        console = self.console
        console.ni_ai.configure()

        # Assert error notification
        sy.sleep(1)
        notifications = self.console.check_for_notifications()
        msg = notifications[0]["message"]
        msg_expected = "Failed to update Task"
        assert (
            msg_expected == msg
        ), f"Notification msg is <{msg}>, should be <{msg_expected}>"
        self.console.close_all_notifications()

        # Assert Task error status
        status = console.ni_ai.status()
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

    def nominal_configuration(self, rack_name: str, dev_name: str) -> None:
        """Nominal configuration of task"""
        console = self.console

        # Add channel
        console.ni_ai.add_channel(
            name="new_channel", type="Voltage", device=dev_name, dev_name="usb_6000"
        )

        self.log("Configuring task")
        console.ni_ai.configure()
        self.log("Running task")
        console.ni_ai.run()

        # Status assertions
        status = console.ni_ai.status()
        level = status["level"]
        msg = status["msg"]

        while level == "loading" and self.should_continue:
            sy.sleep(0.1)
            status = console.ni_ai.status()
            level = status["level"]
            msg = status["msg"]

        level_expected = "warning"
        msg_expected = f"{rack_name} is not running"

        assert msg_expected in msg, f"<{msg}> should be <{msg_expected}>"
        assert level_expected == level, f"<{level}> should be <{level_expected}>"
