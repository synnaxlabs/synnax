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


class NICounterWriteForms(ConsoleCase):
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
        device_name = "CO_E203"

        self.log("Creating NI Counter Write Task")
        console.ni_co.new()

        # Check simple functionality
        console.ni_co.set_parameters(
            task_name="CO_Test_task",
            state_update_rate=10,
            data_saving=True,
            auto_start=False,
        )

        self.create_test_rack(rack_name, device_name)
        self.verify_pulse_output_inputs(device_name)

        # Assert the set values with form state
        ch_names = console.ni_co.channels_by_name.copy()
        random.shuffle(ch_names)
        total = len(ch_names)
        self.log(f"Asserting {total} channel forms in random order")
        for ch in ch_names:
            console.ni_co.assert_channel(ch)

    def create_test_rack(self, rack_name: str, device_name: str) -> None:
        rack = self.client.hardware.racks.create(name=rack_name)
        self.client.hardware.devices.create(
            [
                sy.Device(
                    key=f"130227d9-03bb-47e4-b370-0d590add1bc5",
                    rack=rack.key,
                    name=device_name,
                    make="NI",
                    model="NI 9229",
                    location=device_name,
                    identifier=f"{device_name}Mod1",
                )
            ]
        )
        sy.sleep(1)

    def verify_pulse_output_inputs(self, device_name: str) -> None:
        """Validate Pulse Output inputs"""
        self.log("Configuring channels of type Pulse Output")
        console = self.console
        type = "Pulse Output"

        # Test with different idle state
        console.ni_co.add_channel(
            name="PulseOutput_1",
            type=type,
            device=device_name,
            port=2,
            idle_state="High",
        )

        # Test with all parameters
        console.ni_co.add_channel(
            name="PulseOutput_2",
            type=type,
            device=device_name,
            port=3,
            initial_delay=1,
            high_time=0.05,
            low_time=0.05,
            idle_state="Low",
        )
