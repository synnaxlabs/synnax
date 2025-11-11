#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json

import pytest

import synnax as sy
from synnax import ValidationError
from synnax.hardware import ni
from synnax.hardware.ni import (
    MAKE,
    AIVoltageChan,
    AnalogReadTask,
    AnalogReadTaskConfig,
    AnalogWriteConfig,
    CounterReadConfig,
    DigitalReadConfig,
    DigitalWriteConfig,
    create_device,
    device_props,
)


@pytest.mark.ni
class TestNITask:
    def test_parse_analog_read_task(self):
        data = {
            "sample_rate": 10,
            "stream_rate": 5,
            "auto_start": True,
            "channels": [
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "k09AWoiyLxN",
                    "type": "ai_voltage",
                    "terminal_config": "Cfg_Default",
                    "channel": 1048582,
                    "port": 0,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "units": "Volts",
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "Ar7iVF6Tjzf",
                    "type": "ai_thermocouple",
                    "channel": 1048583,
                    "port": 1,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "units": "DegC",
                    "thermocouple_type": "J",
                    "cjc_source": "BuiltIn",
                    "cjc_val": 0,
                    "cjc_port": 0,
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "3zRNCIt6H0A",
                    "channel": 1048584,
                    "type": "ai_rtd",
                    "port": 2,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "units": "DegC",
                    "rtd_type": "Pt3750",
                    "resistance_config": "2Wire",
                    "current_excit_source": "Internal",
                    "current_excit_val": 0,
                    "r0": 0,
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "2dYSMHhZHtt",
                    "type": "ai_pressure_bridge_two_point_lin",
                    "channel": 1048585,
                    "port": 3,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "units": "PoundsPerSquareInch",
                    "bridge_config": "FullBridge",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "nominal_bridge_resistance": 0,
                    "electrical_units": "mVoltsPerVolt",
                    "physical_units": "PoundsPerSquareInch",
                    "first_electrical_val": 0,
                    "first_physical_val": 0,
                    "second_electrical_val": 1,
                    "second_physical_val": 1,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "h6aNsbt9iXo",
                    "type": "ai_accel",
                    "channel": 1048586,
                    "port": 4,
                    "units": "g",
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "sensitivity": 0,
                    "sensitivity_units": "mVoltsPerG",
                    "current_excit_source": "Internal",
                    "current_excit_val": 0,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "88xDWnrIF90",
                    "type": "ai_bridge",
                    "units": "mVoltsPerVolt",
                    "channel": 1048587,
                    "port": 5,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "bridge_config": "FullBridge",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "nominal_bridge_resistance": 1,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "9AvgGUrtyvw",
                    "channel": 1048588,
                    "port": 6,
                    "type": "ai_current",
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "units": "Amps",
                    "shunt_resistor_loc": "Default",
                    "ext_shunt_resistor_val": 1,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "fo88BkNspw0",
                    "type": "ai_force_bridge_table",
                    "channel": 1048589,
                    "port": 7,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "units": "Newtons",
                    "bridge_config": "FullBridge",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "nominal_bridge_resistance": 0,
                    "electrical_units": "mVoltsPerVolt",
                    "electrical_vals": [],
                    "physical_units": "Newtons",
                    "physical_vals": [],
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "FN8PbpvQvBt",
                    "type": "ai_force_bridge_two_point_lin",
                    "channel": 1048590,
                    "port": 8,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "units": "Newtons",
                    "bridge_config": "FullBridge",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "nominal_bridge_resistance": 0,
                    "electrical_units": "mVoltsPerVolt",
                    "physical_units": "Newtons",
                    "first_electrical_val": 0,
                    "first_physical_val": 0,
                    "second_electrical_val": 1,
                    "second_physical_val": 1,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "s3KtqxSs6cD",
                    "type": "ai_force_iepe",
                    "channel": 1048591,
                    "port": 9,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "units": "Newtons",
                    "sensitivity": 0,
                    "sensitivity_units": "mVoltsPerVolt",
                    "current_excit_source": "Internal",
                    "current_excit_val": 0,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "VNUF26p0JC2",
                    "type": "ai_microphone",
                    "channel": 1048592,
                    "port": 10,
                    "enabled": True,
                    "terminal_config": "Cfg_Default",
                    "mic_sensitivity": 0,
                    "max_snd_press_level": 0,
                    "current_excit_source": "Internal",
                    "current_excit_val": 0,
                    "units": "Pascals",
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "9b5IgxtRYIx",
                    "type": "ai_pressure_bridge_table",
                    "channel": 1048593,
                    "port": 11,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "units": "PoundsPerSquareInch",
                    "bridge_config": "FullBridge",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "nominal_bridge_resistance": 0,
                    "electrical_units": "mVoltsPerVolt",
                    "electrical_vals": [],
                    "physical_units": "PoundsPerSquareInch",
                    "physical_vals": [],
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "i6dz8FDpPwp",
                    "type": "ai_resistance",
                    "channel": 1048594,
                    "port": 12,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "units": "Ohms",
                    "resistance_config": "2Wire",
                    "current_excit_source": "Internal",
                    "current_excit_val": 0,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "hfADiXS6IMR",
                    "type": "ai_strain_gauge",
                    "channel": 1048595,
                    "port": 13,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "terminal_config": "Cfg_Default",
                    "units": "strain",
                    "strain_config": "full-bridge-I",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "gage_factor": 0,
                    "initial_bridge_voltage": 0,
                    "nominal_gage_resistance": 0,
                    "poisson_ratio": 0,
                    "lead_wire_resistance": 0,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "afSn8BOZ8Nv",
                    "type": "ai_temp_builtin",
                    "channel": 1048596,
                    "port": 14,
                    "enabled": True,
                    "units": "DegC",
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "LuzjMHtLTR9",
                    "type": "ai_torque_bridge_table",
                    "channel": 1048597,
                    "port": 15,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "units": "NewtonMeters",
                    "bridge_config": "FullBridge",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "nominal_bridge_resistance": 0,
                    "electrical_units": "mVoltsPerVolt",
                    "electrical_vals": [],
                    "physical_units": "NewtonMeters",
                    "physical_vals": [],
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "IPJbBSbVf7F",
                    "type": "ai_torque_bridge_two_point_lin",
                    "channel": 1048598,
                    "port": 16,
                    "enabled": True,
                    "min_val": 0,
                    "max_val": 1,
                    "units": "NewtonMeters",
                    "bridge_config": "FullBridge",
                    "voltage_excit_source": "Internal",
                    "voltage_excit_val": 0,
                    "nominal_bridge_resistance": 0,
                    "electrical_units": "mVoltsPerVolt",
                    "physical_units": "NewtonMeters",
                    "first_electrical_val": 0,
                    "first_physical_val": 0,
                    "second_electrical_val": 1,
                    "second_physical_val": 1,
                    "custom_scale": {"type": "none"},
                },
                {
                    "name": "",
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "key": "ZayvFgaGurP",
                    "type": "ai_velocity_iepe",
                    "channel": 1048599,
                    "port": 17,
                    "enabled": True,
                    "terminal_config": "Cfg_Default",
                    "min_val": 0,
                    "max_val": 1,
                    "units": "MetersPerSecond",
                    "sensitivity": 0,
                    "sensitivity_units": "MillivoltsPerMillimeterPerSecond",
                    "current_excit_source": "Internal",
                    "current_excit_val": 0,
                    "custom_scale": {"type": "none"},
                },
            ],
            "data_saving": True,
        }
        AnalogReadTaskConfig.model_validate(data)

    def test_parse_analog_read_task_default_device_none_provided(self):
        with pytest.raises(ValidationError):
            AnalogReadTask(
                sample_rate=10,
                stream_rate=5,
                channels=[
                    AIVoltageChan(
                        key="k09AWoiyLxN",
                        terminal_config="Cfg_Default",
                        channel=1048582,
                        port=0,
                        enabled=True,
                        min_val=0,
                        max_val=1,
                        units="Volts",
                    )
                ],
            )

    def test_parse_analog_read_task_default_device_provided(self):
        AnalogReadTask(
            device="474503CF-49FD-11EF-80E5-91C59E7C9645",
            sample_rate=10,
            stream_rate=5,
            channels=[
                AIVoltageChan(
                    key="k09AWoiyLxN",
                    terminal_config="Cfg_Default",
                    channel=1048582,
                    port=0,
                    enabled=True,
                    min_val=0,
                    max_val=1,
                    units="Volts",
                )
            ],
        )

    def test_parse_analog_write_task(self):
        data = {
            "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
            "state_rate": 10,
            "channels": [
                {
                    "key": "AnalogOut1",
                    "type": "ao_voltage",
                    "enabled": True,
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "cmd_channel": 1048610,
                    "state_channel": 1048611,
                    "port": 0,
                    "min_val": -10.0,
                    "max_val": 10.0,
                    "units": "Volts",
                    "custom_scale": {"type": "none"},
                }
            ],
            "data_saving": True,
            "auto_start": False,
        }
        AnalogWriteConfig.model_validate(data)

    def test_parse_counter_read_task(self):
        data = {
            "sample_rate": 1000,
            "stream_rate": 500,
            "channels": [
                {
                    "key": "CounterFreq1",
                    "type": "ci_frequency",
                    "enabled": True,
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "channel": 1048630,
                    "port": 0,
                    "min_val": 0.0,
                    "max_val": 1000.0,
                    "units": "Hz",
                    "edge": "Rising",
                    "meas_method": "LowFreq1Ctr",
                    "meas_time": 0.001,
                    "divisor": 4,
                    "custom_scale": {"type": "none"},
                }
            ],
            "data_saving": True,
            "auto_start": False,
        }
        CounterReadConfig.model_validate(data)

    def test_parse_counter_read_linear_velocity_task(self):
        data = {
            "sample_rate": 1000,
            "stream_rate": 500,
            "channels": [
                {
                    "key": "LinearVel1",
                    "type": "ci_velocity_linear",
                    "enabled": True,
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "channel": 1048631,
                    "port": 0,
                    "min_val": 0.0,
                    "max_val": 100.0,
                    "units": "MetersPerSecond",
                    "decoding_type": "X4",
                    "dist_per_pulse": 0.001,
                    "terminalA": "",
                    "terminalB": "",
                    "custom_scale": {"type": "none"},
                }
            ],
            "data_saving": True,
            "auto_start": False,
        }
        CounterReadConfig.model_validate(data)

    def test_parse_counter_read_angular_velocity_task(self):
        data = {
            "sample_rate": 1000,
            "stream_rate": 500,
            "channels": [
                {
                    "key": "AngularVel1",
                    "type": "ci_velocity_angular",
                    "enabled": True,
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "channel": 1048632,
                    "port": 1,
                    "min_val": 0.0,
                    "max_val": 10000.0,
                    "units": "RPM",
                    "decoding_type": "X4",
                    "pulses_per_rev": 1024,
                    "terminalA": "",
                    "terminalB": "",
                    "custom_scale": {"type": "none"},
                }
            ],
            "data_saving": True,
            "auto_start": False,
        }
        CounterReadConfig.model_validate(data)

    def test_parse_counter_read_linear_position_task(self):
        data = {
            "sample_rate": 1000,
            "stream_rate": 500,
            "channels": [
                {
                    "key": "LinearPos1",
                    "type": "ci_position_linear",
                    "enabled": True,
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "channel": 1048633,
                    "port": 2,
                    "units": "Meters",
                    "decoding_type": "X4",
                    "dist_per_pulse": 0.001,
                    "initial_pos": 0.0,
                    "z_index_enable": False,
                    "z_index_val": 0.0,
                    "z_index_phase": "AHighBHigh",
                    "terminalA": "",
                    "terminalB": "",
                    "terminalZ": "",
                    "custom_scale": {"type": "none"},
                }
            ],
            "data_saving": True,
            "auto_start": False,
        }
        CounterReadConfig.model_validate(data)

    def test_parse_counter_read_angular_position_task(self):
        data = {
            "sample_rate": 1000,
            "stream_rate": 500,
            "channels": [
                {
                    "key": "AngularPos1",
                    "type": "ci_position_angular",
                    "enabled": True,
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "channel": 1048634,
                    "port": 3,
                    "units": "Degrees",
                    "decoding_type": "X4",
                    "pulses_per_rev": 1024,
                    "initial_angle": 0.0,
                    "z_index_enable": False,
                    "z_index_val": 0.0,
                    "z_index_phase": "AHighBHigh",
                    "terminalA": "",
                    "terminalB": "",
                    "terminalZ": "",
                    "custom_scale": {"type": "none"},
                }
            ],
            "data_saving": True,
            "auto_start": False,
        }
        CounterReadConfig.model_validate(data)

    def test_parse_counter_read_duty_cycle_task(self):
        data = {
            "sample_rate": 1000,
            "stream_rate": 500,
            "channels": [
                {
                    "key": "DutyCycle1",
                    "type": "ci_duty_cycle",
                    "enabled": True,
                    "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
                    "channel": 1048635,
                    "port": 4,
                    "min_val": 0.0,
                    "max_val": 1.0,
                    "activeEdge": "Rising",
                    "custom_scale": {"type": "none"},
                }
            ],
            "data_saving": True,
            "auto_start": False,
        }
        CounterReadConfig.model_validate(data)

    def test_parse_digital_read_task(self):
        data = {
            "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
            "channels": [
                {
                    "key": "5DSZbDQy6a4",
                    "type": "digital_input",
                    "enabled": True,
                    "port": 0,
                    "line": 1,
                    "channel": 1048601,
                }
            ],
            "sample_rate": 50,
            "stream_rate": 25,
            "data_saving": True,
            "auto_start": False,
        }
        DigitalReadConfig.model_validate(data)

    def test_parse_digital_write_task(self):
        data = {
            "device": "474503CF-49FD-11EF-80E5-91C59E7C9645",
            "state_rate": 10,
            "channels": [
                {
                    "key": "Xph3kNL7twt",
                    "type": "digital_output",
                    "enabled": True,
                    "cmd_channel": 1048605,
                    "state_channel": 1048603,
                    "port": 0,
                    "line": 1,
                }
            ],
            "data_saving": True,
            "auto_start": True,
        }
        DigitalWriteConfig.model_validate(data)


@pytest.mark.ni
class TestNIDeviceHelpers:
    """Tests for NI device helper functions (create_device, device_props)."""

    def test_device_props_creates_correct_structure(self):
        """Test that device_props creates the expected properties dict."""
        props = device_props(identifier="test_device_01")

        assert isinstance(props, dict)
        assert "identifier" in props
        assert props["identifier"] == "test_device_01"

    def test_device_props_with_different_identifiers(self):
        """Test device_props with various identifier formats."""
        test_cases = [
            "ni_9205",
            "cDAQ1/dev_mod1",
            "USB-6008",
            "my_custom_id",
        ]

        for identifier in test_cases:
            props = device_props(identifier=identifier)
            assert props["identifier"] == identifier

    def test_create_device_sets_make(self, client: sy.Synnax):
        """Test that create_device automatically sets make to 'NI'."""
        rack = client.hardware.racks.retrieve_embedded_rack()

        device = create_device(
            client=client,
            name="Test NI Device",
            model="NI 9205",
            location="cDAQ1/dev_mod1",
            rack=rack.key,
            properties=json.dumps(device_props(identifier="dev_mod1")),
        )

        assert device.make == MAKE
        assert device.make == "NI"
        assert device.name == "Test NI Device"
        assert device.model == "NI 9205"
        assert device.location == "cDAQ1/dev_mod1"

    def test_create_device_auto_generates_key(self, client: sy.Synnax):
        """Test that create_device auto-generates a UUID key if not provided."""
        rack = client.hardware.racks.retrieve_embedded_rack()

        device = create_device(
            client=client,
            name="Test NI Device Auto Key",
            model="NI 9205",
            location="cDAQ1/dev_mod2",
            rack=rack.key,
            properties=json.dumps(device_props(identifier="dev_mod2")),
        )

        assert device.key is not None
        assert len(device.key) > 0
        # UUID format check (basic)
        assert "-" in device.key or len(device.key) > 10

    def test_create_device_with_explicit_key(self, client: sy.Synnax):
        """Test that create_device accepts an explicit key."""
        rack = client.hardware.racks.retrieve_embedded_rack()
        explicit_key = "my-explicit-ni-key"

        device = create_device(
            client=client,
            key=explicit_key,
            name="Test NI Device Explicit Key",
            model="NI 9205",
            location="cDAQ1/dev_mod3",
            rack=rack.key,
            properties=json.dumps(device_props(identifier="dev_mod3")),
        )

        assert device.key == explicit_key

    def test_create_device_properties_parsing(self, client: sy.Synnax):
        """Test that device properties are correctly stored and retrieved."""
        rack = client.hardware.racks.retrieve_embedded_rack()
        test_identifier = "test_ni_module_01"

        device = create_device(
            client=client,
            name="Test NI Props",
            model="NI 9205",
            location="cDAQ1/dev_mod4",
            rack=rack.key,
            properties=json.dumps(device_props(identifier=test_identifier)),
        )

        # Retrieve and parse properties
        props = json.loads(device.properties)
        assert "identifier" in props
        assert props["identifier"] == test_identifier

    def test_create_device_using_ni_module_directly(self, client: sy.Synnax):
        """Test that ni.create_device works when imported via module."""
        rack = client.hardware.racks.retrieve_embedded_rack()

        device = ni.create_device(
            client=client,
            name="Test via ni module",
            model="NI 9205",
            location="cDAQ1/dev_mod5",
            rack=rack.key,
            properties=json.dumps(ni.device_props(identifier="dev_mod5")),
        )

        assert device.make == "NI"
        assert device.name == "Test via ni module"

    def test_device_props_returns_dict_not_json(self):
        """Test that device_props returns a dict, not a JSON string."""
        props = device_props(identifier="test_id")

        # Should be a dict, not a string
        assert isinstance(props, dict)
        assert not isinstance(props, str)

        # Should be JSON-serializable
        json_str = json.dumps(props)
        assert isinstance(json_str, str)
        assert "identifier" in json_str
