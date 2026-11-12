#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

from console.task.analog_read import AnalogRead
from console.task.channels import analog_input
from tests.console.task.ni_rack_case import NIRackCase


class NIAnalogReadForms(NIRackCase):
    """
    Test the input selection for each channel type. Not running the tasks here.
    Only verify that each input type (dropdown/int/float) can be
    appropriately selected. Tasks are not configured/run.

    Randomly selects ~1/8 of all channel types to test each run.
    """

    def run(self) -> None:
        """Test channel type form inputs with random subset selection."""
        console = self.console
        suffix = random.randint(100, 999)
        rack_name = f"TestRack_{suffix}"
        device_name = f"AI_E103_{suffix}"

        self.log("Creating NI Analog Read Task")
        ni_ai = console.pages.create(AnalogRead, "Test_AI_task")

        ni_ai.set_parameters(
            task_name="Test_task",
            sample_rate=100,
            stream_rate=20,
            auto_start=False,
        )

        self.create_test_ni_rack(rack_name, device_name=device_name)

        # All available channel type verifiers
        all_verifiers = [
            self.verify_voltage_inputs,
            self.verify_accel_inputs,
            self.verify_bridge_inputs,
            self.verify_current_inputs,
            self.verify_force_bridge_table_inputs,
            self.verify_force_bridge_two_point_linear_inputs,
            self.verify_force_iepe_inputs,
            self.verify_microphone_inputs,
            self.verify_pressure_bridge_table_inputs,
            self.verify_pressure_bridge_two_point_linear_inputs,
            self.verify_resistance_inputs,
            self.verify_rtd_inputs,
            self.verify_strain_gauge_inputs,
            self.verify_temperature_built_in_sensor_inputs,
            self.verify_thermocouple_inputs,
            self.verify_torque_bridge_table_inputs,
            self.verify_torque_bridge_two_point_linear_inputs,
            self.verify_velocity_iepe_inputs,
        ]

        # Select random 1/8 of verifiers
        sample_size = max(1, len(all_verifiers) // 8)
        selected = random.sample(all_verifiers, sample_size)

        self.log(f"Testing {len(selected)}/{len(all_verifiers)} channel types")
        for verifier in selected:
            verifier(ni_ai, device_name=device_name)

        # Assert the set values with form state
        ch_names = ni_ai.channels_by_name.copy()
        random.shuffle(ch_names)
        self.log(f"Asserting {len(ch_names)} channel forms in random order")
        for ch in ch_names:
            ni_ai.assert_channel(ch)

    def verify_voltage_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate voltage inputs"""
        self.log("Configuring channels of type Voltage")
        channel_type = analog_input.Voltage

        ni_ai.add_channel(
            name="v0",
            chan_type=channel_type,
            device=device_name,
            terminal_config="Cfg_Default",
        )
        ni_ai.add_channel(
            name="v1",
            chan_type=channel_type,
            device=device_name,
            terminal_config="Diff",
            min_val=-0.1,
            max_val=6.5,
        )
        ni_ai.add_channel(
            name="v2",
            chan_type=channel_type,
            device=device_name,
            terminal_config="PseudoDiff",
            min_val=-10,
            max_val=10,
        )
        ni_ai.add_channel(
            name="v3",
            chan_type=channel_type,
            device=device_name,
            terminal_config="RSE",
        )
        ni_ai.add_channel(
            name="v4",
            chan_type=channel_type,
            device=device_name,
            terminal_config="NRSE",
        )

    def verify_accel_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate accel inputs"""
        self.log("Configuring channels of type Accelerometer")
        channel_type = analog_input.Accelerometer

        ni_ai.add_channel(
            name="Accel_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="Accel_2",
            chan_type=channel_type,
            device=device_name,
            sensitivity=0.25,
            units="mVoltsPerG",
            excitation_source="Internal",
            current_excitation_value=0.1,
        )
        ni_ai.add_channel(
            name="Accel_3",
            chan_type=channel_type,
            device=device_name,
            units="VoltsPerG",
            excitation_source="External",
        )
        ni_ai.add_channel(
            name="Accel_4",
            chan_type=channel_type,
            device=device_name,
            excitation_source="None",
        )

    def verify_bridge_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate Bridge inputs"""
        self.log("Configuring channels of type Bridge")
        channel_type = analog_input.Bridge

        ni_ai.add_channel(
            name="Bridge_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="Bridge_2",
            chan_type=channel_type,
            device=device_name,
            units="mVoltsPerVolt",
            configuration="FullBridge",
            resistance=0.1,
            excitation_source="Internal",
            excitation_value=0.2,
        )
        ni_ai.add_channel(
            name="Bridge_3",
            chan_type=channel_type,
            device=device_name,
            units="VoltsPerVolt",
            configuration="HalfBridge",
            excitation_source="External",
        )
        ni_ai.add_channel(
            name="Bridge_4",
            chan_type=channel_type,
            device=device_name,
            configuration="QuarterBridge",
            excitation_source="None",
        )

    def verify_current_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate Bridge inputs"""
        self.log("Configuring channels of type Current")
        channel_type = analog_input.Current

        ni_ai.add_channel(
            name="Current_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="Current_2",
            chan_type=channel_type,
            device=device_name,
            shunt_resistor="Default",
            resistance=0.1,
        )
        ni_ai.add_channel(
            name="Current_3",
            chan_type=channel_type,
            device=device_name,
            shunt_resistor="Internal",
        )
        ni_ai.add_channel(
            name="Current_4",
            chan_type=channel_type,
            device=device_name,
            shunt_resistor="External",
        )

    def verify_force_bridge_table_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Force Bridge Table inputs"""
        self.log("Configuring channels of type Force Bridge Table")
        channel_type = analog_input.ForceBridgeTable

        ni_ai.add_channel(
            name="ForceBridge_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="ForceBridge_2",
            chan_type=channel_type,
            device=device_name,
            force_units="Newtons",
            bridge_configuration="FullBridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Newtons",
            electrical_units="mVoltsPerVolt",
        )
        ni_ai.add_channel(
            name="ForceBridge_3",
            chan_type=channel_type,
            device=device_name,
            force_units="Pounds",
            bridge_configuration="HalfBridge",
            excitation_source="External",
            physical_units="Pounds",
            electrical_units="VoltsPerVolt",
        )
        ni_ai.add_channel(
            name="ForceBridge_3",
            chan_type=channel_type,
            device=device_name,
            force_units="KilogramForce",
            physical_units="KilogramForce",
        )

    def verify_force_bridge_two_point_linear_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Force Bridge Two Point Linear inputs"""
        self.log("Configuring channels of type Force Bridge Two-Point Linear")
        channel_type = analog_input.ForceBridgeTwoPointLinear

        ni_ai.add_channel(
            name="ForceBridge2Pt_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="ForceBridge2Pt_2",
            chan_type=channel_type,
            device=device_name,
            force_units="Newtons",
            bridge_configuration="FullBridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Newtons",
            electrical_units="mVoltsPerVolt",
            physical_value_one=0.0,
            physical_value_two=100.0,
            electrical_value_one=0.0,
            electrical_value_two=2.5,
        )
        ni_ai.add_channel(
            name="ForceBridge2Pt_3",
            chan_type=channel_type,
            device=device_name,
            force_units="Pounds",
            bridge_configuration="HalfBridge",
            excitation_source="External",
            physical_units="Pounds",
            electrical_units="VoltsPerVolt",
        )
        ni_ai.add_channel(
            name="ForceBridge2Pt_4",
            chan_type=channel_type,
            device=device_name,
            force_units="KilogramForce",
            bridge_configuration="QuarterBridge",
            excitation_source="None",
            physical_units="KilogramForce",
        )

    def verify_force_iepe_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate Force IEPE inputs"""
        self.log("Configuring channels of type Force IEPE")
        channel_type = analog_input.ForceIEPE

        ni_ai.add_channel(
            name="ForceIEPE_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="ForceIEPE_2",
            chan_type=channel_type,
            device=device_name,
            force_units="Newtons",
            sensitivity=2.5,
            sensitivity_units="mVoltsPerNewton",
            current_excitation_source="Internal",
            current_excitation_value=4.0,
        )
        ni_ai.add_channel(
            name="ForceIEPE_3",
            chan_type=channel_type,
            device=device_name,
            force_units="Pounds",
            sensitivity_units="mVoltsPerPound",
            current_excitation_source="External",
        )
        ni_ai.add_channel(
            name="ForceIEPE_4",
            chan_type=channel_type,
            device=device_name,
            sensitivity_units="mVoltsPerNewton",
            current_excitation_source="None",
        )

    def verify_microphone_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate Microphone inputs"""
        self.log("Configuring channels of type Microphone")
        channel_type = analog_input.Microphone

        ni_ai.add_channel(
            name="Microphone_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="Microphone_2",
            chan_type=channel_type,
            device=device_name,
            sound_pressure_units="Pascals",
            sensitivity=2.5,
            max_sound_pressure_level=120.0,
            current_excitation_source="Internal",
            current_excitation_value=4.0,
        )
        ni_ai.add_channel(
            name="Microphone_3",
            chan_type=channel_type,
            device=device_name,
            current_excitation_source="External",
        )
        ni_ai.add_channel(
            name="Microphone_4",
            chan_type=channel_type,
            device=device_name,
            current_excitation_source="None",
        )

    def verify_pressure_bridge_table_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Pressure Bridge Table inputs"""
        self.log("Configuring channels of type Pressure Bridge Table")
        channel_type = analog_input.PressureBridgeTable

        ni_ai.add_channel(
            name="PressureBridge_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="PressureBridge_2",
            chan_type=channel_type,
            device=device_name,
            pressure_units="Pascals",
            bridge_configuration="FullBridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Pascals",
            electrical_units="mVoltsPerVolt",
        )
        ni_ai.add_channel(
            name="PressureBridge_3",
            chan_type=channel_type,
            device=device_name,
            pressure_units="PoundsPerSquareInch",
            bridge_configuration="HalfBridge",
            excitation_source="External",
            physical_units="PoundsPerSquareInch",
            electrical_units="VoltsPerVolt",
        )
        ni_ai.add_channel(
            name="PressureBridge_4",
            chan_type=channel_type,
            device=device_name,
            bridge_configuration="QuarterBridge",
            excitation_source="None",
            electrical_units="mVoltsPerVolt",
        )

    def verify_pressure_bridge_two_point_linear_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Pressure Bridge Two-Point Linear inputs"""
        self.log("Configuring channels of type Pressure Bridge Two-Point Linear")
        channel_type = analog_input.PressureBridgeTwoPointLinear

        ni_ai.add_channel(
            name="PressureBridge2Pt_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="PressureBridge2Pt_2",
            chan_type=channel_type,
            device=device_name,
            pressure_units="Pascals",
            bridge_configuration="FullBridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Pascals",
            electrical_units="mVoltsPerVolt",
            physical_value_one=0.0,
            physical_value_two=100.0,
            electrical_value_one=0.0,
            electrical_value_two=2.5,
        )
        ni_ai.add_channel(
            name="PressureBridge2Pt_3",
            chan_type=channel_type,
            device=device_name,
            pressure_units="PoundsPerSquareInch",
            bridge_configuration="HalfBridge",
            excitation_source="External",
            physical_units="PoundsPerSquareInch",
            electrical_units="VoltsPerVolt",
        )
        ni_ai.add_channel(
            name="PressureBridge2Pt_4",
            chan_type=channel_type,
            device=device_name,
            bridge_configuration="QuarterBridge",
            excitation_source="None",
        )

    def verify_resistance_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate Resistance inputs"""
        self.log("Configuring channels of type Resistance")
        channel_type = analog_input.Resistance

        ni_ai.add_channel(
            name="Resistance_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="Resistance_2",
            chan_type=channel_type,
            device=device_name,
            resistance_configuration="2Wire",
            current_excitation_source="Internal",
            current_excitation_value=1.0,
        )
        ni_ai.add_channel(
            name="Resistance_3",
            chan_type=channel_type,
            device=device_name,
            resistance_configuration="3Wire",
            current_excitation_source="External",
        )
        ni_ai.add_channel(
            name="Resistance_4",
            chan_type=channel_type,
            device=device_name,
            resistance_configuration="4Wire",
            current_excitation_source="None",
        )

    def verify_rtd_inputs(self, ni_ai: AnalogRead, *, device_name: str) -> None:
        """Validate RTD inputs"""
        self.log("Configuring channels of type RTD")
        channel_type = analog_input.RTD

        ni_ai.add_channel(
            name="RTD_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="RTD_2",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegC",
            rtd_type="Pt3750",
            resistance_configuration="2Wire",
            current_excitation_source="Internal",
            current_excitation_value=1.0,
            r0_resistance=100.0,
        )
        ni_ai.add_channel(
            name="RTD_3",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegF",
            rtd_type="Pt3851",
            resistance_configuration="3Wire",
            current_excitation_source="External",
        )
        ni_ai.add_channel(
            name="RTD_4",
            chan_type=channel_type,
            device=device_name,
            temperature_units="Kelvins",
            rtd_type="Pt3928",
            resistance_configuration="4Wire",
            current_excitation_source="None",
        )

    def verify_strain_gauge_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Strain Gauge inputs"""
        self.log("Configuring channels of type Strain Gauge")
        channel_type = analog_input.StrainGauge

        ni_ai.add_channel(
            name="StrainGauge_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="StrainGauge_2",
            chan_type=channel_type,
            device=device_name,
            strain_configuration="FullBridgeI",
            excitation_source="Internal",
            excitation_value=2.5,
            gage_factor=2.0,
            initial_bridge_voltage=0.0,
            nominal_gage_resistance=120.0,
            poisson_ratio=0.3,
            lead_wire_resistance=0.0,
        )
        ni_ai.add_channel(
            name="StrainGauge_3",
            chan_type=channel_type,
            device=device_name,
            strain_configuration="HalfBridgeI",
            excitation_source="External",
        )
        ni_ai.add_channel(
            name="StrainGauge_4",
            chan_type=channel_type,
            device=device_name,
            strain_configuration="FullBridgeIII",
            excitation_source="None",
        )
        ni_ai.add_channel(
            name="StrainGauge_5",
            chan_type=channel_type,
            device=device_name,
            strain_configuration="QuarterBridgeI",
        )

    def verify_temperature_built_in_sensor_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Temperature Built-In Sensor inputs"""
        self.log("Configuring channels of type Temperature Built-In Sensor")
        channel_type = analog_input.TemperatureBuiltInSensor

        ni_ai.add_channel(
            name="TempBuiltIn_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="TempBuiltIn_2",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegC",
        )
        ni_ai.add_channel(
            name="TempBuiltIn_3",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegF",
        )
        ni_ai.add_channel(
            name="TempBuiltIn_4",
            chan_type=channel_type,
            device=device_name,
            temperature_units="Kelvins",
        )
        ni_ai.add_channel(
            name="TempBuiltIn_5",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegR",
        )

    def verify_thermocouple_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Thermocouple inputs"""
        self.log("Configuring channels of type Thermocouple")
        channel_type = analog_input.Thermocouple

        ni_ai.add_channel(
            name="Thermocouple_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="Thermocouple_2",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegC",
            thermocouple_type="J",
            cjc_source="built_in",
        )
        ni_ai.add_channel(
            name="Thermocouple_3",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegF",
            thermocouple_type="K",
            cjc_source="const_val",
            cjc_value=25.0,
        )
        ni_ai.add_channel(
            name="Thermocouple_4",
            chan_type=channel_type,
            device=device_name,
            temperature_units="DegR",
            thermocouple_type="E",
        )

    def verify_torque_bridge_table_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Torque Bridge Table inputs"""
        self.log("Configuring channels of type Torque Bridge Table")
        channel_type = analog_input.TorqueBridgeTable

        ni_ai.add_channel(
            name="TorqueBridgeTable_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="TorqueBridgeTable_2",
            chan_type=channel_type,
            device=device_name,
            torque_units="NewtonMeters",
            bridge_configuration="FullBridge",
            nominal_bridge_resistance=350.0,
            voltage_excitation_source="Internal",
            voltage_excitation_value=5.0,
            physical_units="NewtonMeters",
        )
        ni_ai.add_channel(
            name="TorqueBridgeTable_3",
            chan_type=channel_type,
            device=device_name,
            torque_units="FootPounds",
            bridge_configuration="HalfBridge",
            voltage_excitation_source="External",
            physical_units="FootPounds",
            electrical_units="VoltsPerVolt",
        )
        ni_ai.add_channel(
            name="TorqueBridgeTable_4",
            chan_type=channel_type,
            device=device_name,
            torque_units="InchOunces",
            bridge_configuration="QuarterBridge",
            voltage_excitation_source="None",
            physical_units="InchOunces",
            electrical_units="mVoltsPerVolt",
        )

    def verify_torque_bridge_two_point_linear_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Torque Bridge Two-Point Linear inputs"""
        self.log("Configuring channels of type Torque Bridge Two-Point Linear")
        channel_type = analog_input.TorqueBridgeTwoPointLinear

        ni_ai.add_channel(
            name="TorqueBridge2Pt_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="TorqueBridge2Pt_2",
            chan_type=channel_type,
            device=device_name,
            torque_units="NewtonMeters",
            bridge_configuration="FullBridge",
            nominal_bridge_resistance=350.0,
            voltage_excitation_source="Internal",
            voltage_excitation_value=5.0,
            physical_units="NewtonMeters",
            electrical_units="mVoltsPerVolt",
            physical_value_one=0.0,
            physical_value_two=100.0,
            electrical_value_one=0.0,
            electrical_value_two=2.5,
        )
        ni_ai.add_channel(
            name="TorqueBridge2Pt_3",
            chan_type=channel_type,
            device=device_name,
            torque_units="FootPounds",
            bridge_configuration="HalfBridge",
            voltage_excitation_source="External",
            physical_units="FootPounds",
            electrical_units="VoltsPerVolt",
        )
        ni_ai.add_channel(
            name="TorqueBridge2Pt_4",
            chan_type=channel_type,
            device=device_name,
            torque_units="InchOunces",
            bridge_configuration="QuarterBridge",
            voltage_excitation_source="None",
            physical_units="InchOunces",
            electrical_units="mVoltsPerVolt",
        )

    def verify_velocity_iepe_inputs(
        self, ni_ai: AnalogRead, *, device_name: str
    ) -> None:
        """Validate Velocity IEPE inputs"""
        self.log("Configuring channels of type Velocity IEPE")
        channel_type = analog_input.VelocityIEPE

        ni_ai.add_channel(
            name="VelocityIEPE_1",
            chan_type=channel_type,
            device=device_name,
        )
        ni_ai.add_channel(
            name="VelocityIEPE_2",
            chan_type=channel_type,
            device=device_name,
            velocity_units="MetersPerSecond",
            sensitivity=10.0,
            sensitivity_units="MillivoltsPerMillimeterPerSecond",
            current_excitation_source="Internal",
            current_excitation_value=4.0,
        )
        ni_ai.add_channel(
            name="VelocityIEPE_3",
            chan_type=channel_type,
            device=device_name,
            velocity_units="InchesPerSecond",
            sensitivity_units="MilliVoltsPerInchPerSecond",
        )
        ni_ai.add_channel(
            name="VelocityIEPE_4",
            chan_type=channel_type,
            device=device_name,
            velocity_units="MetersPerSecond",
            sensitivity_units="MillivoltsPerMillimeterPerSecond",
            current_excitation_source="None",
        )
