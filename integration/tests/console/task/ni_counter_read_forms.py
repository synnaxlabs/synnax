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
from console.task.counter_read import CounterRead


class NICounterReadForms(ConsoleCase):
    """
    Test the input selection for each channel type. Not running the tasks here.
    Only verify that each input type (dropdown/int/float) can be
    appropriately selected. Tasks are not configured/run.

    Randomly selects ~1/4 of all channel types to test each run.
    """

    def run(self) -> None:
        """Test channel type form inputs with random subset selection."""
        console = self.console
        rack_name = f"TestRack_{random.randint(100, 999)}"
        device_name = "CI_E103"

        self.log("Creating NI Counter Read Task")
        ni_ci = CounterRead(self.client, console, "CI_Test_task")

        ni_ci.set_parameters(
            task_name="CI_Test_task",
            sample_rate=100,
            stream_rate=20,
            data_saving=True,
            auto_start=False,
        )

        self.create_test_rack(rack_name, device_name)

        # All available channel type verifiers
        all_verifiers = [
            self.verify_edge_count_inputs,
            self.verify_frequency_inputs,
            self.verify_period_inputs,
            self.verify_pulse_width_inputs,
            self.verify_semi_period_inputs,
            self.verify_two_edge_sep_inputs,
            self.verify_duty_cycle_inputs,
            self.verify_linear_velocity_inputs,
            self.verify_angular_velocity_inputs,
            self.verify_linear_position_inputs,
            self.verify_angular_position_inputs,
        ]

        # Select random 1/4 of verifiers
        sample_size = max(1, len(all_verifiers) // 4)
        selected = random.sample(all_verifiers, sample_size)

        self.log(f"Testing {len(selected)}/{len(all_verifiers)} channel types")
        for verifier in selected:
            verifier(ni_ci, device_name)

        # Assert the set values with form state
        ch_names = ni_ci.channels_by_name.copy()
        random.shuffle(ch_names)
        self.log(f"Asserting {len(ch_names)} channel forms in random order")
        for ch in ch_names:
            ni_ci.assert_channel(ch)

    def create_test_rack(self, rack_name: str, device_name: str) -> None:
        rack = self.client.racks.create(name=rack_name)
        self.client.devices.create(
            [
                sy.Device(
                    key="230227d9-02aa-47e4-b370-0d590add1bc1",
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

    def verify_edge_count_inputs(self, ni_ci: CounterRead, device_name: str) -> None:
        """Validate Edge Count inputs"""
        self.log("Configuring channels of type Edge Count")
        console = self.console
        channel_type = "Edge Count"

        ni_ci.add_channel(
            name="EdgeCount_1",
            chan_type=channel_type,
            device=device_name,
            active_edge="Rising",
            count_direction="Count Up",
            terminal="PFI0",
            initial_count=10,
        )
        ni_ci.add_channel(
            name="EdgeCount_2",
            chan_type=channel_type,
            device=device_name,
            active_edge="Falling",
            count_direction="Count Down",
        )
        ni_ci.add_channel(
            name="EdgeCount_3",
            chan_type=channel_type,
            device=device_name,
            count_direction="Externally Controlled",
        )

    def verify_frequency_inputs(self, ni_ci: CounterRead, device_name: str) -> None:
        """Validate Frequency inputs"""
        self.log("Configuring channels of type Frequency")
        console = self.console
        channel_type = "Frequency"

        ni_ci.add_channel(
            name="Frequency_1",
            chan_type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1000,
            edge="Rising",
            units="Hz",
            meas_method="One Counter (Low Frequency)",
        )
        ni_ci.add_channel(
            name="Frequency_2",
            chan_type=channel_type,
            device=device_name,
            edge="Falling",
            units="Ticks",
            meas_method="Two Counters (High Frequency)",
        )
        ni_ci.add_channel(
            name="Frequency_3",
            chan_type=channel_type,
            device=device_name,
            meas_method="Two Counters (Large Range)",
        )

    def verify_period_inputs(self, ni_ci: CounterRead, device_name: str) -> None:
        """Validate Period inputs"""
        self.log("Configuring channels of type Period")
        console = self.console
        channel_type = "Period"

        ni_ci.add_channel(
            name="Period_1",
            chan_type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1,
            starting_edge="Rising",
            units="Seconds",
            terminal="PFI1",
            meas_method="One Counter (Low Frequency)",
        )
        ni_ci.add_channel(
            name="Period_2",
            chan_type=channel_type,
            device=device_name,
            starting_edge="Falling",
            units="Ticks",
            meas_method="Two Counters (High Frequency)",
        )
        ni_ci.add_channel(
            name="Period_3",
            chan_type=channel_type,
            device=device_name,
            units="Seconds",
            meas_method="Two Counters (Large Range)",
        )
        ni_ci.add_channel(
            name="Period_4",
            chan_type=channel_type,
            device=device_name,
            meas_method="Dynamic Averaging",
        )

    def verify_pulse_width_inputs(self, ni_ci: CounterRead, device_name: str) -> None:
        """Validate Pulse Width inputs"""
        self.log("Configuring channels of type Pulse Width")
        console = self.console
        channel_type = "Pulse Width"

        ni_ci.add_channel(
            name="PulseWidth_1",
            chan_type=channel_type,
            device=device_name,
            units="Seconds",
        )
        ni_ci.add_channel(
            name="PulseWidth_2",
            chan_type=channel_type,
            device=device_name,
            starting_edge="Rising",
            units="Seconds",
            terminal="PFI2",
        )
        ni_ci.add_channel(
            name="PulseWidth_3",
            chan_type=channel_type,
            device=device_name,
            min_val=0.001,
            max_val=10,
            starting_edge="Falling",
            units="Ticks",
        )

    def verify_semi_period_inputs(self, ni_ci: CounterRead, device_name: str) -> None:
        """Validate Semi Period inputs"""
        self.log("Configuring channels of type Semi Period")
        console = self.console
        channel_type = "Semi Period"

        ni_ci.add_channel(
            name="SemiPeriod_1",
            chan_type=channel_type,
            device=device_name,
            units="Seconds",
        )
        ni_ci.add_channel(
            name="SemiPeriod_2",
            chan_type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1,
            units="Seconds",
        )
        ni_ci.add_channel(
            name="SemiPeriod_3",
            chan_type=channel_type,
            device=device_name,
            units="Ticks",
        )

    def verify_two_edge_sep_inputs(self, ni_ci: CounterRead, device_name: str) -> None:
        """Validate Two Edge Separation inputs"""
        self.log("Configuring channels of type Two Edge Separation")
        console = self.console
        channel_type = "Two Edge Separation"

        ni_ci.add_channel(
            name="TwoEdgeSep_1",
            chan_type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1,
            units="Seconds",
            first_edge="Rising",
            second_edge="Falling",
        )
        ni_ci.add_channel(
            name="TwoEdgeSep_2",
            chan_type=channel_type,
            device=device_name,
            units="Ticks",
            first_edge="Falling",
            second_edge="Rising",
        )

    def verify_linear_velocity_inputs(
        self, ni_ci: CounterRead, device_name: str
    ) -> None:
        """Validate Linear Velocity inputs"""
        self.log("Configuring channels of type Velocity Linear")
        console = self.console
        channel_type = "Velocity Linear"

        ni_ci.add_channel(
            name="LinearVelocity_1",
            chan_type=channel_type,
            device=device_name,
            min_val=0,
            max_val=10,
            units="m/s",
            decoding_type="X4",
            dist_per_pulse=0.001,
        )
        ni_ci.add_channel(
            name="LinearVelocity_2",
            chan_type=channel_type,
            device=device_name,
            units="in/s",
            decoding_type="X1",
            dist_per_pulse=0.01,
        )
        ni_ci.add_channel(
            name="LinearVelocity_3",
            chan_type=channel_type,
            device=device_name,
            units="m/s",
            decoding_type="X2",
            dist_per_pulse=0.005,
        )

    def verify_angular_velocity_inputs(
        self, ni_ci: CounterRead, device_name: str
    ) -> None:
        """Validate Angular Velocity inputs"""
        self.log("Configuring channels of type Velocity Angular")
        console = self.console
        channel_type = "Velocity Angular"

        ni_ci.add_channel(
            name="AngularVelocity_1",
            chan_type=channel_type,
            device=device_name,
            min_val=0,
            max_val=1000,
            units="RPM",
            decoding_type="X4",
            pulses_per_rev=24,
        )
        ni_ci.add_channel(
            name="AngularVelocity_2",
            chan_type=channel_type,
            device=device_name,
            units="Radians/s",
            decoding_type="X2",
            pulses_per_rev=100,
        )
        ni_ci.add_channel(
            name="AngularVelocity_3",
            chan_type=channel_type,
            device=device_name,
            units="Degrees/s",
            decoding_type="Two Pulse",
            pulses_per_rev=360,
        )

    def verify_linear_position_inputs(
        self, ni_ci: CounterRead, device_name: str
    ) -> None:
        """Validate Linear Position inputs"""
        self.log("Configuring channels of type Position Linear")
        console = self.console
        channel_type = "Position Linear"

        ni_ci.add_channel(
            name="LinearPosition_1",
            chan_type=channel_type,
            device=device_name,
            units="Meters",
            decoding_type="X4",
            dist_per_pulse=0.001,
            initial_pos=0,
            z_index_enable=True,
            z_index_val=0,
            z_index_phase="A High B High",
        )
        ni_ci.add_channel(
            name="LinearPosition_2",
            chan_type=channel_type,
            device=device_name,
            units="Inches",
            decoding_type="X2",
            dist_per_pulse=0.01,
            initial_pos=5,
            z_index_enable=False,
        )
        ni_ci.add_channel(
            name="LinearPosition_3",
            chan_type=channel_type,
            device=device_name,
            units="Ticks",
            decoding_type="X1",
            dist_per_pulse=0.005,
            z_index_enable=True,
            z_index_val=10,
            z_index_phase="A Low B Low",
        )

    def verify_angular_position_inputs(
        self, ni_ci: CounterRead, device_name: str
    ) -> None:
        """Validate Angular Position inputs"""
        self.log("Configuring channels of type Position Angular")
        console = self.console
        channel_type = "Position Angular"

        ni_ci.add_channel(
            name="AngularPosition_1",
            chan_type=channel_type,
            device=device_name,
            units="Degrees",
            decoding_type="X4",
            pulses_per_rev=24,
            initial_angle=0,
            z_index_enable=True,
            z_index_val=0,
            z_index_phase="A High B High",
        )
        ni_ci.add_channel(
            name="AngularPosition_2",
            chan_type=channel_type,
            device=device_name,
            units="Radians",
            decoding_type="X2",
            pulses_per_rev=100,
            initial_angle=1.57,
            z_index_enable=False,
        )
        ni_ci.add_channel(
            name="AngularPosition_3",
            chan_type=channel_type,
            device=device_name,
            units="Ticks",
            decoding_type="Two Pulse",
            pulses_per_rev=360,
            z_index_enable=True,
            z_index_val=90,
            z_index_phase="A High B Low",
        )

    def verify_duty_cycle_inputs(self, ni_ci: CounterRead, device_name: str) -> None:
        """Validate Duty Cycle inputs"""
        self.log("Configuring channels of type Duty Cycle")
        console = self.console
        channel_type = "Duty Cycle"

        ni_ci.add_channel(
            name="DutyCycle_1",
            chan_type=channel_type,
            device=device_name,
            min_val=2,
            max_val=10000,
            edge="Rising",
        )
        ni_ci.add_channel(
            name="DutyCycle_2",
            chan_type=channel_type,
            device=device_name,
            min_val=10,
            max_val=5000,
            edge="Falling",
        )
        ni_ci.add_channel(
            name="DutyCycle_3",
            chan_type=channel_type,
            device=device_name,
            min_val=5,
            max_val=20000,
            edge="Rising",
        )
