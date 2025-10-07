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


class NIAnalogReadForms(ConsoleCase):
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

        # Mode is used so that we can break this test into smaller
        # chunks to run concurrently. The split is arbitrary. However,
        # there is a balance between small (and fast) chunks and
        # initializing multiple, resource-intensive playwright instances

        mode = self.name[-1]  # A, B, C, D

        # Talks to NI MAX sim devices
        rack_name = f"TestRack_{random.randint(100, 999)}"
        device_name = f"{mode}_E103"
        sy.sleep(5)
        self.log("Creating NI Analog Read Task")
        console.ni_ai.new()

        # Check simple functionality
        console.ni_ai.set_parameters(
            task_name="Test_task",
            sample_rate=100,
            stream_rate=20,
            data_saving=True,
            auto_start=False,
        )

        self.create_test_rack(rack_name, device_name, mode)

        if mode == "a":
            self.verify_voltage_inputs(device_name)
            self.verify_accel_inputs(device_name)
            self.verify_bridge_inputs(device_name)
        if mode == "b":
            self.verify_current_inputs(device_name)
            self.verify_force_bridge_table_inputs(device_name)
            self.verify_force_bridge_two_point_linear_inputs(device_name)
        if mode == "c":
            self.verify_force_iepe_inputs(device_name)
            self.verify_microphone_inputs(device_name)
            self.verify_pressure_bridge_table_inputs(device_name)
        if mode == "d":
            self.verify_pressure_bridge_two_point_linear_inputs(device_name)
            self.verify_resistance_inputs(device_name)
            self.verify_rtd_inputs(device_name)
        if mode == "e":
            self.verify_strain_gauge_inputs(device_name)
            self.verify_temperature_built_in_sensor_inputs(device_name)
            self.verify_thermocouple_inputs(device_name)
        if mode == "f":
            self.verify_torque_bridge_table_inputs(device_name)
            self.verify_torque_bridge_two_point_linear_inputs(device_name)
            self.verify_velocity_iepe_inputs(device_name)

        # Assert the set values with form state
        ch_names = console.ni_ai.channels_by_name.copy()
        random.shuffle(ch_names)
        total = len(ch_names)
        self.log(f"Asserting {total} channel forms in random order")
        for ch in ch_names:
            console.ni_ai.assert_channel(ch)

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

    def verify_voltage_inputs(self, device_name: str) -> None:
        """Validate voltage inputs"""
        self.log("Configuring channels of type Voltage")
        console = self.console
        type = "Voltage"

        console.ni_ai.add_channel(
            name="v0",
            type=type,
            device=device_name,
            terminal_config="Default",
        )
        console.ni_ai.add_channel(
            name="v1",
            type=type,
            device=device_name,
            terminal_config="Differential",
            min_val=-0.1,
            max_val=6.5,
        )
        console.ni_ai.add_channel(
            name="v2",
            type=type,
            device=device_name,
            terminal_config="Pseudo-Differential",
            min_val=-10,
            max_val=10,
        )
        console.ni_ai.add_channel(
            name="v3",
            type=type,
            device=device_name,
            terminal_config="Referenced Single Ended",
        )
        console.ni_ai.add_channel(
            name="v4",
            type=type,
            device=device_name,
            terminal_config="Non-Referenced Single Ended",
        )

    def verify_accel_inputs(self, device_name: str) -> None:
        """Validate accel inputs"""
        self.log("Configuring channels of type Accelerometer")
        console = self.console
        type = "Accelerometer"

        console.ni_ai.add_channel(
            name="Accel_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="Accel_2",
            type=type,
            device=device_name,
            sensitivity=0.25,
            units="mV/g",
            excitation_source="Internal",
            current_excitation_value=0.1,
        )
        console.ni_ai.add_channel(
            name="Accel_3",
            type=type,
            device=device_name,
            units="V/g",
            excitation_source="External",
        )
        console.ni_ai.add_channel(
            name="Accel_4",
            type=type,
            device=device_name,
            excitation_source="None",
        )

    def verify_bridge_inputs(self, device_name: str) -> None:
        """Validate Bridge inputs"""
        self.log("Configuring channels of type Bridge")
        console = self.console
        type = "Bridge"

        console.ni_ai.add_channel(
            name="Bridge_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="Bridge_2",
            type=type,
            device=device_name,
            units="mV/V",
            configuration="Full Bridge",
            resistance=0.1,
            excitation_source="Internal",
            excitation_value=0.2,
        )
        console.ni_ai.add_channel(
            name="Bridge_3",
            type=type,
            device=device_name,
            units="V/V",
            configuration="Half Bridge",
            excitation_source="External",
        )
        console.ni_ai.add_channel(
            name="Bridge_4",
            type=type,
            device=device_name,
            configuration="Quarter Bridge",
            excitation_source="None",
        )

    def verify_current_inputs(self, device_name: str) -> None:
        """Validate Bridge inputs"""
        self.log("Configuring channels of type Current")
        console = self.console
        type = "Current"

        console.ni_ai.add_channel(
            name="Current_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="Current_2",
            type=type,
            device=device_name,
            shunt_resistor="Default",
            resistance=0.1,
        )
        console.ni_ai.add_channel(
            name="Current_3", type=type, device=device_name, shunt_resistor="Internal"
        )
        console.ni_ai.add_channel(
            name="Current_4",
            type=type,
            device=device_name,
            shunt_resistor="External",
        )

    def verify_force_bridge_table_inputs(self, device_name: str) -> None:
        """Validate Force Bridge Table inputs"""
        self.log("Configuring channels of type Force Bridge Table")
        console = self.console
        type = "Force Bridge Table"

        console.ni_ai.add_channel(
            name="ForceBridge_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="ForceBridge_2",
            type=type,
            device=device_name,
            force_units="Newtons",
            bridge_configuration="Full Bridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Newtons",
            electrical_units="mV/V",
        )
        console.ni_ai.add_channel(
            name="ForceBridge_3",
            type=type,
            device=device_name,
            force_units="Pounds",
            bridge_configuration="Half Bridge",
            excitation_source="External",
            physical_units="Pounds",
            electrical_units="V/V",
        ),
        console.ni_ai.add_channel(
            name="ForceBridge_3",
            type=type,
            device=device_name,
            force_units="Kilograms",
            physical_units="Kilograms",
        )

    def verify_force_bridge_two_point_linear_inputs(self, device_name: str) -> None:
        """Validate Force Bridge Two Point Linear inputs"""
        self.log("Configuring channels of type Force Bridge Two-Point Linear")
        console = self.console
        type = "Force Bridge Two-Point Linear"

        console.ni_ai.add_channel(
            name="ForceBridge2Pt_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="ForceBridge2Pt_2",
            type=type,
            device=device_name,
            force_units="Newtons",
            bridge_configuration="Full Bridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Newtons",
            electrical_units="mV/V",
            physical_value_one=0.0,
            physical_value_two=100.0,
            electrical_value_one=0.0,
            electrical_value_two=2.5,
        )
        console.ni_ai.add_channel(
            name="ForceBridge2Pt_3",
            type=type,
            device=device_name,
            force_units="Pounds",
            bridge_configuration="Half Bridge",
            excitation_source="External",
            physical_units="Pounds",
            electrical_units="V/V",
        )
        console.ni_ai.add_channel(
            name="ForceBridge2Pt_4",
            type=type,
            device=device_name,
            force_units="Kilograms",
            bridge_configuration="Quarter Bridge",
            excitation_source="None",
            physical_units="Kilograms",
        )

    def verify_force_iepe_inputs(self, device_name: str) -> None:
        """Validate Force IEPE inputs"""
        self.log("Configuring channels of type Force IEPE")
        console = self.console
        type = "Force IEPE"

        console.ni_ai.add_channel(
            name="ForceIEPE_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="ForceIEPE_2",
            type=type,
            device=device_name,
            force_units="Newtons",
            sensitivity=2.5,
            sensitivity_units="mV/N",
            current_excitation_source="Internal",
            current_excitation_value=4.0,
        )
        console.ni_ai.add_channel(
            name="ForceIEPE_3",
            type=type,
            device=device_name,
            force_units="Pounds",
            sensitivity_units="mV/lb",
            current_excitation_source="External",
        )
        console.ni_ai.add_channel(
            name="ForceIEPE_4",
            type=type,
            device=device_name,
            sensitivity_units="mV/N",
            current_excitation_source="None",
        )

    def verify_microphone_inputs(self, device_name: str) -> None:
        """Validate Microphone inputs"""
        self.log("Configuring channels of type Microphone")
        console = self.console
        type = "Microphone"

        console.ni_ai.add_channel(
            name="Microphone_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="Microphone_2",
            type=type,
            device=device_name,
            sound_pressure_units="Pascals",
            sensitivity=2.5,
            max_sound_pressure_level=120.0,
            current_excitation_source="Internal",
            current_excitation_value=4.0,
        )
        console.ni_ai.add_channel(
            name="Microphone_3",
            type=type,
            device=device_name,
            current_excitation_source="External",
        )
        console.ni_ai.add_channel(
            name="Microphone_4",
            type=type,
            device=device_name,
            current_excitation_source="None",
        )

    def verify_pressure_bridge_table_inputs(self, device_name: str) -> None:
        """Validate Pressure Bridge Table inputs"""
        self.log("Configuring channels of type Pressure Bridge Table")
        console = self.console
        type = "Pressure Bridge Table"

        console.ni_ai.add_channel(
            name="PressureBridge_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="PressureBridge_2",
            type=type,
            device=device_name,
            pressure_units="Pascals",
            bridge_configuration="Full Bridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Pascals",
            electrical_units="mV/V",
        )
        console.ni_ai.add_channel(
            name="PressureBridge_3",
            type=type,
            device=device_name,
            pressure_units="PSI",
            bridge_configuration="Half Bridge",
            excitation_source="External",
            physical_units="PSI",
            electrical_units="V/V",
        )
        console.ni_ai.add_channel(
            name="PressureBridge_4",
            type=type,
            device=device_name,
            bridge_configuration="Quarter Bridge",
            excitation_source="None",
            electrical_units="mV/V",
        )

    def verify_pressure_bridge_two_point_linear_inputs(self, device_name: str) -> None:
        """Validate Pressure Bridge Two-Point Linear inputs"""
        self.log("Configuring channels of type Pressure Bridge Two-Point Linear")
        console = self.console
        type = "Pressure Bridge Two-Point Linear"

        console.ni_ai.add_channel(
            name="PressureBridge2Pt_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="PressureBridge2Pt_2",
            type=type,
            device=device_name,
            pressure_units="Pascals",
            bridge_configuration="Full Bridge",
            resistance=350.0,
            excitation_source="Internal",
            excitation_value=5.0,
            physical_units="Pascals",
            electrical_units="mV/V",
            physical_value_one=0.0,
            physical_value_two=100.0,
            electrical_value_one=0.0,
            electrical_value_two=2.5,
        )
        console.ni_ai.add_channel(
            name="PressureBridge2Pt_3",
            type=type,
            device=device_name,
            pressure_units="PSI",
            bridge_configuration="Half Bridge",
            excitation_source="External",
            physical_units="PSI",
            electrical_units="V/V",
        )
        console.ni_ai.add_channel(
            name="PressureBridge2Pt_4",
            type=type,
            device=device_name,
            bridge_configuration="Quarter Bridge",
            excitation_source="None",
        )

    def verify_resistance_inputs(self, device_name: str) -> None:
        """Validate Resistance inputs"""
        self.log("Configuring channels of type Resistance")
        console = self.console
        type = "Resistance"

        console.ni_ai.add_channel(
            name="Resistance_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="Resistance_2",
            type=type,
            device=device_name,
            resistance_configuration="2-Wire",
            current_excitation_source="Internal",
            current_excitation_value=1.0,
        )
        console.ni_ai.add_channel(
            name="Resistance_3",
            type=type,
            device=device_name,
            resistance_configuration="3-Wire",
            current_excitation_source="External",
        )
        console.ni_ai.add_channel(
            name="Resistance_4",
            type=type,
            device=device_name,
            resistance_configuration="4-Wire",
            current_excitation_source="None",
        )

    def verify_rtd_inputs(self, device_name: str) -> None:
        """Validate RTD inputs"""
        self.log("Configuring channels of type RTD")
        console = self.console
        type = "RTD"

        console.ni_ai.add_channel(
            name="RTD_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="RTD_2",
            type=type,
            device=device_name,
            temperature_units="Celsius",
            rtd_type="Pt3750",
            resistance_configuration="2-Wire",
            current_excitation_source="Internal",
            current_excitation_value=1.0,
            r0_resistance=100.0,
        )
        console.ni_ai.add_channel(
            name="RTD_3",
            type=type,
            device=device_name,
            temperature_units="Fahrenheit",
            rtd_type="Pt3851",
            resistance_configuration="3-Wire",
            current_excitation_source="External",
        )
        console.ni_ai.add_channel(
            name="RTD_4",
            type=type,
            device=device_name,
            temperature_units="Kelvin",
            rtd_type="Pt3928",
            resistance_configuration="4-Wire",
            current_excitation_source="None",
        )

    def verify_strain_gauge_inputs(self, device_name: str) -> None:
        """Validate Strain Gauge inputs"""
        self.log("Configuring channels of type Strain Gauge")
        console = self.console
        type = "Strain Gauge"

        console.ni_ai.add_channel(
            name="StrainGauge_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="StrainGauge_2",
            type=type,
            device=device_name,
            strain_configuration="Full Bridge I",
            excitation_source="Internal",
            excitation_value=2.5,
            gage_factor=2.0,
            initial_bridge_voltage=0.0,
            nominal_gage_resistance=120.0,
            poisson_ratio=0.3,
            lead_wire_resistance=0.0,
        )
        console.ni_ai.add_channel(
            name="StrainGauge_3",
            type=type,
            device=device_name,
            strain_configuration="Half Bridge I",
            excitation_source="External",
        )
        console.ni_ai.add_channel(
            name="StrainGauge_4",
            type=type,
            device=device_name,
            strain_configuration="Full Bridge III",
            excitation_source="None",
        )
        console.ni_ai.add_channel(
            name="StrainGauge_5",
            type=type,
            device=device_name,
            strain_configuration="Quarter Bridge I",
        )

    def verify_temperature_built_in_sensor_inputs(self, device_name: str) -> None:
        """Validate Temperature Built-In Sensor inputs"""
        self.log("Configuring channels of type Temperature Built-In Sensor")
        console = self.console
        type = "Temperature Built-In Sensor"

        console.ni_ai.add_channel(
            name="TempBuiltIn_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="TempBuiltIn_2",
            type=type,
            device=device_name,
            temperature_units="Celsius",
        )
        console.ni_ai.add_channel(
            name="TempBuiltIn_3",
            type=type,
            device=device_name,
            temperature_units="Fahrenheit",
        )
        console.ni_ai.add_channel(
            name="TempBuiltIn_4",
            type=type,
            device=device_name,
            temperature_units="Kelvin",
        )
        console.ni_ai.add_channel(
            name="TempBuiltIn_5",
            type=type,
            device=device_name,
            temperature_units="Rankine",
        )

    def verify_thermocouple_inputs(self, device_name: str) -> None:
        """Validate Thermocouple inputs"""
        self.log("Configuring channels of type Thermocouple")
        console = self.console
        type = "Thermocouple"

        console.ni_ai.add_channel(
            name="Thermocouple_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="Thermocouple_2",
            type=type,
            device=device_name,
            temperature_units="Celsius",
            thermocouple_type="J",
            cjc_source="Built In",
        )
        console.ni_ai.add_channel(
            name="Thermocouple_3",
            type=type,
            device=device_name,
            temperature_units="Fahrenheit",
            thermocouple_type="K",
            cjc_source="Constant Value",
            cjc_value=25.0,
        )
        console.ni_ai.add_channel(
            name="Thermocouple_4",
            type=type,
            device=device_name,
            temperature_units="Rankine",
            thermocouple_type="E",
        )

    def verify_torque_bridge_table_inputs(self, device_name: str) -> None:
        """Validate Torque Bridge Table inputs"""
        self.log("Configuring channels of type Torque Bridge Table")
        console = self.console
        type = "Torque Bridge Table"

        console.ni_ai.add_channel(
            name="TorqueBridgeTable_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="TorqueBridgeTable_2",
            type=type,
            device=device_name,
            torque_units="Newton Meters",
            bridge_configuration="Full Bridge",
            nominal_bridge_resistance=350.0,
            voltage_excitation_source="Internal",
            voltage_excitation_value=5.0,
            physical_units="Newton Meters",
        )
        console.ni_ai.add_channel(
            name="TorqueBridgeTable_3",
            type=type,
            device=device_name,
            torque_units="Foot Pounds",
            bridge_configuration="Half Bridge",
            voltage_excitation_source="External",
            physical_units="Foot Pounds",
            electrical_units="V/V",
        )
        console.ni_ai.add_channel(
            name="TorqueBridgeTable_4",
            type=type,
            device=device_name,
            torque_units="Inch Ounces",
            bridge_configuration="Quarter Bridge",
            voltage_excitation_source="None",
            physical_units="Inch Ounces",
            electrical_units="mV/V",
        )

    def verify_torque_bridge_two_point_linear_inputs(self, device_name: str) -> None:
        """Validate Torque Bridge Two-Point Linear inputs"""
        self.log("Configuring channels of type Torque Bridge Two-Point Linear")
        console = self.console
        type = "Torque Bridge Two-Point Linear"

        console.ni_ai.add_channel(
            name="TorqueBridge2Pt_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="TorqueBridge2Pt_2",
            type=type,
            device=device_name,
            torque_units="Newton Meters",
            bridge_configuration="Full Bridge",
            nominal_bridge_resistance=350.0,
            voltage_excitation_source="Internal",
            voltage_excitation_value=5.0,
            physical_units="Newton Meters",
            electrical_units="mV/V",
            physical_value_one=0.0,
            physical_value_two=100.0,
            electrical_value_one=0.0,
            electrical_value_two=2.5,
        )
        console.ni_ai.add_channel(
            name="TorqueBridge2Pt_3",
            type=type,
            device=device_name,
            torque_units="Foot Pounds",
            bridge_configuration="Half Bridge",
            voltage_excitation_source="External",
            physical_units="Foot Pounds",
            electrical_units="V/V",
        )
        console.ni_ai.add_channel(
            name="TorqueBridge2Pt_4",
            type=type,
            device=device_name,
            torque_units="Inch Ounces",
            bridge_configuration="Quarter Bridge",
            voltage_excitation_source="None",
            physical_units="Inch Ounces",
            electrical_units="mV/V",
        )

    def verify_velocity_iepe_inputs(self, device_name: str) -> None:
        """Validate Velocity IEPE inputs"""
        self.log("Configuring channels of type Velocity IEPE")
        console = self.console
        type = "Velocity IEPE"

        console.ni_ai.add_channel(
            name="VelocityIEPE_1",
            type=type,
            device=device_name,
        )
        console.ni_ai.add_channel(
            name="VelocityIEPE_2",
            type=type,
            device=device_name,
            velocity_units="m/s",
            sensitivity=10.0,
            sensitivity_units="mV/mm/s",
            current_excitation_source="Internal",
            current_excitation_value=4.0,
        )
        console.ni_ai.add_channel(
            name="VelocityIEPE_3",
            type=type,
            device=device_name,
            velocity_units="in/s",
            sensitivity_units="mV/in/s",
        )
        console.ni_ai.add_channel(
            name="VelocityIEPE_4",
            type=type,
            device=device_name,
            velocity_units="m/s",
            sensitivity_units="mV/mm/s",
            current_excitation_source="None",
        )
