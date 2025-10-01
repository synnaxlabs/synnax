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


class No_Device(ConsoleCase):
    """
    Verify status message/level when attempting to
    configure and run a task with selected devices
    that are not present.
    """

    def run(self) -> None:
        """
        Test Opening and closing pages
        """

        self._log_message("Creating NI Analog Read Task Page")
        self.console.task.new()

        rack_name = f"TestRack_{random.randint(100, 999)}"
        self.create_rack(rack_name)
        self.initial_assertion()
        self.configure_without_channels()
        self.nominal_configuration(rack_name)


    def create_rack(self, rack_name: str) -> None:
        self._log_message(f"Creating {rack_name} and devices")

        client = self.client
        rack = client.hardware.racks.create(name=rack_name)
        client.hardware.devices.create(
            [
                sy.Device(
                    key="a0e37b26-5401-413e-8e65-c7ad9d9afd70",
                    rack=rack.key,
                    name="USB-6000",
                    make="NI",
                    model="USB-6000",
                    location="dev3",
                    identifier="dev3",
                ),
            ]
        )

    def initial_assertion(self) -> None:
        """ Initial assertion of task status """
        console = self.console

        status = console.task.status()
        msg = status['msg']
        level = status['level']

        level_expected = 'disabled'
        msg_expected = 'Task has not been configured'

        assert level_expected == level, \
            f"Task status level <{level}> should be <{level_expected}>"
        assert msg_expected == msg, \
            f"Task status msg <{msg}> should be <{msg_expected}>"

    def configure_without_channels(self) -> None:
        """ Configure without defining channels """
        console = self.console
        console.task.configure()

        # Assert error notification
        notifications = (self.console.check_for_notifications())
        msg = notifications[0]["message"]
        msg_expected = "Failed to update Task"
        assert msg_expected == msg, \
            f"Notification msg is <{msg}>, should be <{msg_expected}>"
        self.console.close_all_notifications()

        # Assert Task error status
        status = console.task.status()  
        level = status['level']
        msg = status['msg']
        level_expected = 'error'
        msg_expected = 'Failed to update Task'
        assert level_expected == level, \
            f"Task status level <{level}> should be <{level_expected}>"
        assert msg_expected == msg, \
            f"Task status msg <{msg}> should be <{msg_expected}>"

    def nominal_configuration(self, rack_name: str) -> None:
        """ Nominal configuration of task """
        console = self.console

        # Add channels
        console.task.add_channel(name="new_channel", type="Voltage", device="USB-6000", dev_name="usb_6000")
        console.task.add_channel(name="hello", type="Accelerometer", device="USB-6000", dev_name="usb_6000")
        console.task.add_channel(name="goodbye", type="Bridge", device="USB-6000", dev_name="usb_6000")

        self._log_message("Configuring task")
        console.task.configure()
        self._log_message("Running task")
        console.task.run()

        # Status assertions
        status = console.task.status()
        level = status['level']
        msg = status['msg']

        while level == 'loading' and self.should_continue:
            sy.sleep(0.1)
            status = console.task.status()
            level = status['level']
            msg = status['msg']

        level_expected = 'warning'
        msg_expected = f"{rack_name} is not running"

        assert msg_expected in msg, \
            f"<{msg}> should be <{msg_expected}>"
        assert level_expected == level, \
            f"<{level}> should be <{level_expected}>"
