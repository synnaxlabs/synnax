#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import synnax as sy
from console.case import ConsoleCase
from console.task.analog_read import AnalogRead
from console.task.channels import analog_input
from framework.hardware_case import HardwareCase


class NoDevice(HardwareCase, ConsoleCase):
    """
    Verify status message/level when attempting to
    configure and run a task with selected devices
    that are not present.
    """

    def run(self) -> None:
        """
        Test Opening and closing pages
        """

        rand_int = random.randint(100, 999)
        rack_name = f"TestRack_{rand_int}"
        dev_name = f"USB-6000_{rand_int}"
        # A new task takes the only NI device if it is configured. This device keeps
        # one that a concurrent test configured from being the only one.
        self.create_rack(rack_name, dev_name=dev_name)

        self.log("Creating NI Analog Read Task Page")
        ni_ai = self.console.pages.create(AnalogRead, f"USB-6000_{rand_int}")
        self.initial_assertion(ni_ai)
        self.configure_without_device(ni_ai)
        self.nominal_configuration(ni_ai, rack_name=rack_name, dev_name=dev_name)

    def create_rack(self, rack_name: str, *, dev_name: str) -> None:
        self.log(f"Creating {rack_name} and devices")

        rack = self.create_test_rack(rack_name)
        self.create_test_devices(
            [
                sy.ni.Device(
                    rack=rack.key,
                    name=dev_name,
                    model=dev_name,
                    location="dev3",
                    identifier="dev3",
                ),
            ]
        )

    def initial_assertion(self, ni_ai: AnalogRead) -> None:
        """Initial assertion of task status"""
        status = ni_ai.status()
        variant_expected = "disabled"
        message_expected = "Task not deployed"

        assert variant_expected == status.variant, (
            f"Task status variant <{status.variant}> should be <{variant_expected}>"
        )
        assert message_expected == status.message, (
            f"Task status message <{status.message}> should be <{message_expected}>"
        )

    def configure_without_device(self, ni_ai: AnalogRead) -> None:
        """Deploy the first channel of a new task, which has no device; the form
        blocks it at the device field."""
        ni_ai.deploy(expect=None)

        msg_expected = "Device is required"
        msg = ni_ai.field_help_text("Device")
        assert msg_expected == msg, f"Device error <{msg}> should be <{msg_expected}>"

        # Deploy rejects the config before it writes or starts anything, so the task
        # keeps the status it had.
        self.initial_assertion(ni_ai)

    def nominal_configuration(
        self, ni_ai: AnalogRead, *, rack_name: str, dev_name: str
    ) -> None:
        """Nominal configuration of task"""
        # Add channel
        ni_ai.add_channel(
            name="new_channel",
            chan_type=analog_input.Voltage,
            device=dev_name,
            dev_name="usb_6000",
        )

        self.log("Deploying task")
        ni_ai.deploy(expect=None)

        # No Driver answers a start command on this rack, so the Console stands the
        # rack's own problem in for a wait that would never end. The Core stamps a
        # "status unknown" placeholder while the task saves, so the wait keys on the
        # message rather than on the variant.
        ni_ai.wait_for_status("Driver not running")

        variant = ni_ai.status().variant
        variant_expected = "warning"
        assert variant_expected == variant, (
            f"<{variant}> should be <{variant_expected}>"
        )
