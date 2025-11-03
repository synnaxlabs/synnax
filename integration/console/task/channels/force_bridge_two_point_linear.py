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


class ForceBridgeTwoPointLinear(Analog):
    """
    Force Bridge Two Point Linear channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        force_units (str): "Newtons", "Pounds", "Kilograms", etc.
        bridge_configuration (str): "Full Bridge", "Half Bridge", "Quarter Bridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", "None"
        excitation_value (float): Voltage excitation value
        physical_units (str): "Newtons", "Pounds", "Kilograms", etc.
        electrical_units (str): "mV/V", "V/V"
        physical_value_one (float): Physical value one for scaling
        physical_value_two (float): Physical value two for scaling
        electrical_value_one (float): Electrical value one for scaling
        electrical_value_two (float): Electrical value two for scaling

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Default", "Differential", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
    """

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        force_units: Literal["Newtons", "Pounds", "Kilograms"] | None = None,
        bridge_configuration: (
            Literal["Full Bridge", "Half Bridge", "Quarter Bridge"] | None
        ) = None,
        resistance: float | None = None,
        excitation_source: Literal["Internal", "External", "None"] | None = None,
        excitation_value: float | None = None,
        physical_units: Literal["Newtons", "Pounds", "Kilograms"] | None = None,
        electrical_units: Literal["mV/V", "V/V"] | None = None,
        physical_value_one: float | None = None,
        physical_value_two: float | None = None,
        electrical_value_one: float | None = None,
        electrical_value_two: float | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Force Bridge Two-Point Linear",
            **kwargs,
        )

        # Force Bridge Two Point Linear-specific configurations:
        if force_units is not None:
            console.click_btn("Force Units")
            console.select_from_dropdown(force_units)

        if bridge_configuration is not None:
            console.click_btn("Bridge Configuration")
            console.select_from_dropdown(bridge_configuration)

        if resistance is not None:
            console.fill_input_field("Nominal Bridge Resistance", str(resistance))

        if excitation_source is not None:
            console.click_btn("Voltage Excitation Source")
            console.select_from_dropdown(excitation_source)

        if excitation_value is not None:
            console.fill_input_field("Voltage Excitation Value", str(excitation_value))

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
