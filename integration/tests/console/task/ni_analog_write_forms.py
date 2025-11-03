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


class NIAnalogWriteForms(ConsoleCase):
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
        sy.sleep(5)
        self.log("Creating NI Analog Write Task")
        console.ni_ao.new()

        # Check simple functionality
        console.ni_ao.set_parameters(
            task_name="AO_Test_task",
            state_update_rate=10,
            data_saving=True,
            auto_start=False,
        )

        self.create_test_rack(rack_name, device_name)
        self.verify_voltage_inputs(device_name)
        self.verify_current_inputs(device_name)

        # Assert the set values with form state
        ch_names = console.ni_ao.channels_by_name.copy()
        random.shuffle(ch_names)
        total = len(ch_names)
        self.log(f"Asserting {total} channel forms in random order")
        for ch in ch_names:
            console.ni_ao.assert_channel(ch)

    def create_test_rack(self, rack_name: str, device_name: str) -> None:
        rack = self.client.hardware.racks.create(name=rack_name)
        self.client.hardware.devices.create(
            [
                sy.Device(
                    key=f"130227d7-02cc-4733-b370-0d590add1bc4",
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

    def verify_voltage_inputs(self, device_name: str) -> None:
        """Validate voltage inputs"""
        self.log("Configuring channels of type Voltage")
        console = self.console

        console.ni_ao.add_channel(
            name="v0",
            chan_type="Voltage",
            device=device_name,
        )
        console.ni_ao.add_channel(
            name="v1",
            chan_type="Voltage",
            device=device_name,
            min_val=-0.1,
            max_val=6.5,
        )

    def verify_current_inputs(self, device_name: str) -> None:
        """Validate Bridge inputs"""
        self.log("Configuring channels of type Current")
        console = self.console

        console.ni_ao.add_channel(
            name="Current_1",
            chan_type="Current",
            device=device_name,
        )
        console.ni_ao.add_channel(
            name="Current_2",
            chan_type="Current",
            device=device_name,
            min_val=-0.1,
            max_val=6.5,
        )
