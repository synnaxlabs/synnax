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


class RTD(Analog):
    """
    RTD channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        temperature_units (str): "Celsius", "Fahrenheit", "Kelvin", "Rankine"
        rtd_type (str): "Pt3750", "Pt3851", "Pt3911", "Pt3916", "Pt3920", "Pt3928"
        resistance_configuration (str): "2-Wire", "3-Wire", "4-Wire"
        current_excitation_source (str): "Internal", "External", "None"
        current_excitation_value (float): Current excitation value
        r0_resistance (float): R0 resistance value

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
        temperature_units: (
            Literal["Celsius", "Fahrenheit", "Kelvin", "Rankine"] | None
        ) = None,
        rtd_type: (
            Literal["Pt3750", "Pt3851", "Pt3911", "Pt3916", "Pt3920", "Pt3928"] | None
        ) = None,
        resistance_configuration: Literal["2-Wire", "3-Wire", "4-Wire"] | None = None,
        current_excitation_source: (
            Literal["Internal", "External", "None"] | None
        ) = None,
        current_excitation_value: float | None = None,
        r0_resistance: float | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="RTD",
            **kwargs,
        )

        # RTD-specific configurations:
        if temperature_units is not None:
            console.click_btn("Temperature Units")
            console.select_from_dropdown(temperature_units)

        if rtd_type is not None:
            console.click_btn("RTD Type")
            console.select_from_dropdown(rtd_type)

        if resistance_configuration is not None:
            console.click_btn("Resistance Configuration")
            console.select_from_dropdown(resistance_configuration)

        if current_excitation_source is not None:
            console.click_btn("Current Excitation Source")
            console.select_from_dropdown(current_excitation_source)

        if current_excitation_value is not None:
            console.fill_input_field(
                "Current Excitation Value", str(current_excitation_value)
            )

        if r0_resistance is not None:
            console.fill_input_field("R0 Resistance", str(r0_resistance))
