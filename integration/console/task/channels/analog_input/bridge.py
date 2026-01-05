#  Copyright 2026 Synnax Labs, Inc.
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


class Bridge(Analog):
    """
    Bridge channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        units (str): "mV/V" or "V/V"
        configuration (str): "FullBridge", "HalfBridge", or "QuarterBridge"
        resistance (float): Nominal bridge resistance
        excitation_source (str): "Internal", "External", or "None"
        excitation_value (float): Voltage excitation value

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
        units: Literal["mV/V", "V/V"] | None = None,
        configuration: (
            Literal["Full Bridge", "Half Bridge", "Quarter Bridge"] | None
        ) = None,
        resistance: float | None = None,
        excitation_source: Literal["Internal", "External", "None"] | None = None,
        excitation_value: float | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Bridge",
            **kwargs,
        )

        # Bridge-specific configurations:
        if units is not None:
            console.click_btn("Electrical Units")
            console.select_from_dropdown(units)

        if configuration is not None:
            console.click_btn("Bridge Configuration")
            console.select_from_dropdown(configuration)

        if resistance is not None:
            console.fill_input_field("Nominal Bridge Resistance", str(resistance))

        if excitation_source is not None:
            console.click_btn("Voltage Excitation Source")
            console.select_from_dropdown(excitation_source)

        if excitation_value is not None:
            console.fill_input_field("Voltage Excitation Value", str(excitation_value))
