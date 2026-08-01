#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

from console.task.analog_write import AnalogWrite
from tests.console.task.rack_case import RackCase


class NIAnalogWriteForms(RackCase):
    """
    Test the input selection for each channel type. Not running the tasks here.
    Only verify that each input type (dropdown/int/float) can be
    appropriately selected. Tasks are not configured/run.
    """

    def run(self) -> None:
        """
        Test Opening and closing pages
        """
        console = self.console

        # Talks to NI MAX sim devices
        rack_name = f"TestRack_{random.randint(100, 999)}"
        device_name = "E203"

        self.log("Creating NI Analog Write Task")
        ni_ao = console.project.create_task("NI Analog Write Task", "AO_Test_task")

        # Check simple functionality
        ni_ao.set_parameters(
            task_name="AO_Test_task",
            state_update_rate=10,
            data_saving=True,
            auto_start=False,
        )

        self.create_test_ni_rack(
            rack_name, device_name, "130227d7-02cc-4733-b370-0d590add1bc4"
        )
        self.verify_voltage_inputs(ni_ao, device_name)
        self.verify_current_inputs(ni_ao, device_name)

        # Assert the set values with form state
        ch_names = ni_ao.channels_by_name.copy()
        random.shuffle(ch_names)
        total = len(ch_names)
        self.log(f"Asserting {total} channel forms in random order")
        for ch in ch_names:
            ni_ao.assert_channel(ch)

    def verify_voltage_inputs(self, ni_ao: AnalogWrite, device_name: str) -> None:
        """Validate voltage inputs"""
        self.log("Configuring channels of type Voltage")
        ni_ao.add_channel(
            name="v0",
            chan_type="Voltage",
            device=device_name,
        )
        ni_ao.add_channel(
            name="v1",
            chan_type="Voltage",
            device=device_name,
            min_val=-0.1,
            max_val=6.5,
        )

    def verify_current_inputs(self, ni_ao: AnalogWrite, device_name: str) -> None:
        """Validate Bridge inputs"""
        self.log("Configuring channels of type Current")
        ni_ao.add_channel(
            name="Current_1",
            chan_type="Current",
            device=device_name,
        )
        ni_ao.add_channel(
            name="Current_2",
            chan_type="Current",
            device=device_name,
            min_val=-0.1,
            max_val=6.5,
        )
