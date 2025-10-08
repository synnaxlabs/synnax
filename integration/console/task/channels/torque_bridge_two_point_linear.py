#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal, Optional

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


class TorqueBridgeTwoPointLinear(Analog):
    """
    Torque Bridge Two-Point Linear channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        torque_units (str): "Newton Meters", "Inch Ounces", "Foot Pounds"
        bridge_configuration (str): "Full Bridge", "Half Bridge", "Quarter Bridge"
        nominal_bridge_resistance (float): Bridge resistance value
        voltage_excitation_source (str): "Internal", "External", "None"
        voltage_excitation_value (float): Excitation voltage value
        physical_units (str): "Newton Meters", "Inch Ounces", "Foot Pounds"
        electrical_units (str): "mV/V", "V/V"
        physical_value_one (float): First physical calibration point
        physical_value_two (float): Second physical calibration point
        electrical_value_one (float): First electrical calibration point
        electrical_value_two (float): Second electrical calibration point

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
        custom_scale (str): "None", "Linear", "Map", "Table"
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        torque_units: Optional[
            Literal[
                "Newton Meters",
                "Inch Ounces",
                "Foot Pounds",
            ]
        ] = None,
        bridge_configuration: Optional[
            Literal[
                "Full Bridge",
                "Half Bridge",
                "Quarter Bridge",
            ]
        ] = None,
        nominal_bridge_resistance: Optional[float] = None,
        voltage_excitation_source: Optional[
            Literal[
                "Internal",
                "External",
                "None",
            ]
        ] = None,
        voltage_excitation_value: Optional[float] = None,
        physical_units: Optional[
            Literal[
                "Newton Meters",
                "Inch Ounces",
                "Foot Pounds",
            ]
        ] = None,
        electrical_units: Optional[
            Literal[
                "mV/V",
                "V/V",
            ]
        ] = None,
        physical_value_one: Optional[float] = None,
        physical_value_two: Optional[float] = None,
        electrical_value_one: Optional[float] = None,
        electrical_value_two: Optional[float] = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(
            console=console,
            name=name,
            device=device,
            type="Torque Bridge Two-Point Linear",
            **kwargs,
        )

        if torque_units is not None:
            console.click_btn("Torque Units")
            console.select_from_dropdown(torque_units)

        if bridge_configuration is not None:
            console.click_btn("Bridge Configuration")
            console.select_from_dropdown(bridge_configuration)

        if nominal_bridge_resistance is not None:
            console.fill_input_field(
                "Nominal Bridge Resistance", str(nominal_bridge_resistance)
            )

        if voltage_excitation_source is not None:
            console.click_btn("Voltage Excitation Source")
            console.select_from_dropdown(voltage_excitation_source)

        if voltage_excitation_value is not None:
            console.fill_input_field(
                "Voltage Excitation Value", str(voltage_excitation_value)
            )

        if physical_units is not None:
            console.click_btn("Physical Units")
            console.select_from_dropdown(physical_units)

        if electrical_units is not None:
            console.click_btn("Electrical Units")
            console.select_from_dropdown(electrical_units)

        if physical_value_one is not None:
            console.fill_input_field("Physical Value One", str(physical_value_one))

        if physical_value_two is not None:
            console.fill_input_field("Physical Value Two", str(physical_value_two))

        if electrical_value_one is not None:
            console.fill_input_field("Electrical Value One", str(electrical_value_one))

        if electrical_value_two is not None:
            console.fill_input_field("Electrical Value Two", str(electrical_value_two))
