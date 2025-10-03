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


class ForceBridgeTable(Analog):
    """
    Force Bridge Table channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        force_units (str): "Newtons", "Pounds", "Kilograms", etc.
        bridge_configuration (str): "Full Bridge", "Half Bridge", "Quarter Bridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", "None"
        excitation_value (float): Voltage excitation value
        physical_units (str): "Newtons", "Pounds", "Kilograms", etc.
        electrical_units (str): "mV/V", "V/V"
        custom_scale (str): "None", "Linear", "Map", "Table"

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
        force_units: Optional[
            Literal[
                "Newtons",
                "Pounds",
                "Kilograms",
            ]
        ] = None,
        bridge_configuration: Optional[
            Literal[
                "Full Bridge",
                "Half Bridge",
                "Quarter Bridge",
            ]
        ] = None,
        resistance: Optional[float] = None,
        excitation_source: Optional[
            Literal[
                "Internal",
                "External",
                "None",
            ]
        ] = None,
        excitation_value: Optional[float] = None,
        physical_units: Optional[
            Literal[
                "Newtons",
                "Pounds",
                "Kilograms",
            ]
        ] = None,
        electrical_units: Optional[
            Literal[
                "mV/V",
                "V/V",
            ]
        ] = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            type="Force Bridge Table",
            **kwargs,
        )

        # Force Bridge Table-specific configurations:
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
