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


class NICounterReadForms(ConsoleCase):
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

        # Get mode from matrix parameters
        mode = self.params.get("mode")
        if mode is None:
            raise ValueError("Missing required parameter 'mode' from matrix")

        # Talks to NI MAX sim devices
        rack_name = f"TestRack_{random.randint(100, 999)}"
        device_name = f"{mode}_CI103"

        self.log("Creating NI Counter Read Task")
        console.ni_ci.new()

        # Check simple functionality
        console.ni_ci.set_parameters(
            task_name="CI_Test_task",
            sample_rate=100,
            stream_rate=20,
            data_saving=True,
            auto_start=False,
        )

        self.create_test_rack(rack_name, device_name, mode)

        # Verify each counter input channel type by mode
        if mode == "a":
            self.verify_edge_count_inputs(device_name)
            self.verify_frequency_inputs(device_name)
        if mode == "b":
            self.verify_period_inputs(device_name)
            self.verify_pulse_width_inputs(device_name)
        if mode == "c":
            self.verify_semi_period_inputs(device_name)
            self.verify_two_edge_sep_inputs(device_name)

        # Assert the set values with form state
        ch_names = console.ni_ci.channels_by_name.copy()
        random.shuffle(ch_names)
        total = len(ch_names)
        self.log(f"Asserting {total} channel forms in random order")
        for ch in ch_names:
            console.ni_ci.assert_channel(ch)

    def create_test_rack(self, rack_name: str, device_name: str, mode: str) -> None:
        rack = self.client.hardware.racks.create(name=rack_name)
        self.client.hardware.devices.create(
            [
                sy.Device(
                    key=f"130227d9-02aa-47e4-b370-0d590add1bc{mode}",
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

    def verify_edge_count_inputs(self, device_name: str) -> None:
        """Validate Edge Count inputs"""
        self.log("Configuring channels of type Edge Count")
        console = self.console
        channel_type = "Edge Count"

        console.ni_ci.add_channel(
            name="EdgeCount_1",
            type=channel_type,
            device=device_name,
            active_edge="Rising",
            count_direction="Count Up",
            terminal="PFI0",
            initial_count=10,
        )
        console.ni_ci.add_channel(
            name="EdgeCount_2",
            type=channel_type,
            device=device_name,
            active_edge="Falling",
            count_direction="Count Down",
        )
        console.ni_ci.add_channel(
            name="EdgeCount_3",
            type=channel_type,
            device=device_name,
            count_direction="Externally Controlled",
        )

    def verify_frequency_inputs(self, device_name: str) -> None:
        """Validate Frequency inputs"""
        self.log("Configuring channels of type Frequency")
        console = self.console
        channel_type = "Frequency"

        console.ni_ci.add_channel(
            name="Frequency_1",
            type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1000,
            edge="Rising",
            units="Hz",
            meas_method="1 Counter (Low Frequency)",
        )
        console.ni_ci.add_channel(
            name="Frequency_2",
            type=channel_type,
            device=device_name,
            edge="Falling",
            units="Ticks",
            meas_method="2 Counters (High Frequency)",
        )
        console.ni_ci.add_channel(
            name="Frequency_3",
            type=channel_type,
            device=device_name,
            meas_method="2 Counters (Large Range)",
        )

    def verify_period_inputs(self, device_name: str) -> None:
        """Validate Period inputs"""
        self.log("Configuring channels of type Period")
        console = self.console
        channel_type = "Period"

        console.ni_ci.add_channel(
            name="Period_1",
            type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1,
            starting_edge="Rising",
            units="Seconds",
            terminal="PFI1",
            meas_method="1 Counter (Low Frequency)",
        )
        console.ni_ci.add_channel(
            name="Period_2",
            type=channel_type,
            device=device_name,
            starting_edge="Falling",
            units="Ticks",
            meas_method="2 Counters (High Frequency)",
        )
        console.ni_ci.add_channel(
            name="Period_3",
            type=channel_type,
            device=device_name,
            units="Custom",
            meas_method="2 Counters (Large Range)",
        )
        console.ni_ci.add_channel(
            name="Period_4",
            type=channel_type,
            device=device_name,
            meas_method="Dynamic Averaging",
        )

    def verify_pulse_width_inputs(self, device_name: str) -> None:
        """Validate Pulse Width inputs"""
        self.log("Configuring channels of type Pulse Width")
        console = self.console
        channel_type = "Pulse Width"

        console.ni_ci.add_channel(
            name="PulseWidth_1",
            type=channel_type,
            device=device_name,
            units="Custom",
        )
        console.ni_ci.add_channel(
            name="PulseWidth_2",
            type=channel_type,
            device=device_name,
            starting_edge="Rising",
            units="Seconds",
            terminal="PFI2",
        )
        console.ni_ci.add_channel(
            name="PulseWidth_3",
            type=channel_type,
            device=device_name,
            min_val=0.001,
            max_val=10,
            starting_edge="Falling",
            units="Ticks",
        )

    def verify_semi_period_inputs(self, device_name: str) -> None:
        """Validate Semi Period inputs"""
        self.log("Configuring channels of type Semi Period")
        console = self.console
        channel_type = "Semi Period"

        console.ni_ci.add_channel(
            name="SemiPeriod_1",
            type=channel_type,
            device=device_name,
            units="Custom",
        )
        console.ni_ci.add_channel(
            name="SemiPeriod_2",
            type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1,
            units="Seconds",
        )
        console.ni_ci.add_channel(
            name="SemiPeriod_3",
            type=channel_type,
            device=device_name,
            units="Ticks",
        )

    def verify_two_edge_sep_inputs(self, device_name: str) -> None:
        """Validate Two Edge Separation inputs"""
        self.log("Configuring channels of type Two Edge Separation")
        console = self.console
        channel_type = "Two Edge Separation"

        console.ni_ci.add_channel(
            name="TwoEdgeSep_1",
            type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1,
            units="Seconds",
            first_edge="Rising",
            second_edge="Falling",
        )
        console.ni_ci.add_channel(
            name="TwoEdgeSep_2",
            type=channel_type,
            device=device_name,
            units="Ticks",
            first_edge="Falling",
            second_edge="Rising",
        )
