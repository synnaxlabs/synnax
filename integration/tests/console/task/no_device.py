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
        console = self.console
        client = self.client

        rack_name = f"TestRack_{random.randint(100, 999)}"
        self._log_message(f"Creating {rack_name} and devices")
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

        self._log_message("Creating NI Analog Read Task")
        console.task.new()
        # Check simple functionality
        console.task.set_parameters(
            task_name="Test_task",
            sample_rate=100,
            stream_rate=20,
            data_saving=True,
            auto_start=False,
        )

        # Add channels
        console.task.add_channel(name="new_channel", type="Voltage", device="USB-6000", dev_name="usb_6000")
        console.task.add_channel(name="hello", type="Accelerometer", device="USB-6000", dev_name="usb_6000")
        console.task.add_channel(name="goodbye", type="Bridge", device="USB-6000", dev_name="usb_6000")

        # Status Assertions
        status = console.task.status()
        msg = status['msg']
        level = status['level']

        level_expected = 'disabled'
        msg_expected = 'Task has not been configured'

        assert level_expected == level, \
            f"<{level}> should be <{level_expected}>"
        assert msg_expected == msg, \
            f"<{msg}> should be <{msg_expected}>"

        self._log_message("Configuring task")
        console.task.configure()
        self._log_message("Running task")
        console.task.run()

        # Status assertions
        status = console.task.status()
        level = status['level']

        while level == 'loading' and self.should_continue:
            sy.sleep(0.1)
            status = console.task.status()
            level = status['level']
            msg = status['msg']

        level_expected = 'warning'
        msg_expected = f"{rack_name} is not running"

        assert level_expected == level, \
            f"<{level}> should be <{level_expected}>"
        assert msg_expected in msg, \
            f"<{msg}> should be <{msg_expected}>"
