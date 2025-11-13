#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


class TorqueBridgeTable(Analog):
    """
    Torque Bridge Table channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        torque_units (str): "Newton Meters", "Inch Ounces", "Foot Pounds"
        bridge_configuration (str): "Full Bridge", "Half Bridge", "Quarter Bridge"
        nominal_bridge_resistance (float): Bridge resistance value
        voltage_excitation_source (str): "Internal", "External", "None"
        voltage_excitation_value (float): Excitation voltage value
        physical_units (str): "Newton Meters", "Inch Ounces", "Foot Pounds"
        electrical_units (str): "mV/V", "V/V"

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
        torque_units: (
            Literal["Newton Meters", "Inch Ounces", "Foot Pounds"] | None
        ) = None,
        bridge_configuration: (
            Literal["Full Bridge", "Half Bridge", "Quarter Bridge"] | None
        ) = None,
        nominal_bridge_resistance: float | None = None,
        voltage_excitation_source: (
            Literal["Internal", "External", "None"] | None
        ) = None,
        voltage_excitation_value: float | None = None,
        physical_units: (
            Literal["Newton Meters", "Inch Ounces", "Foot Pounds"] | None
        ) = None,
        electrical_units: Literal["mV/V", "V/V"] | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Torque Bridge Table",
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
