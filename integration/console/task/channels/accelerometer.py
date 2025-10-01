#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Optional, Literal

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


class Accelerometer(Analog):

    """
    Accelerometer channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        sensitivity (float): Accelerometer sensitivity value
        units (str): "mV/g" or "V/g"
        excitation_source (str): "Internal", "External", or "None"
        excitation_value (float): Current excitation value

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
        device: str,
        sensitivity: float = None,
        units: Optional[Literal[
            "mV/g",
            "V/g",
        ]]=None,
        excitation_source: Optional[Literal[
        "internal",
        "External",
        "None"
        ]] = None,
        current_excitation_value: Optional[float]= None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,

            device=device,
            type="Accelerometer",
            **kwargs,
        )

        # Accelerometer-specific configurations:
        if sensitivity is not None:
            console.fill_input_field("Sensitivity", str(sensitivity))
        if units is not None:
            console.select_from_dropdown(units)
        if excitation_source is not None:
            console.click_btn("Current Excitation Source")
            console.select_from_dropdown(excitation_source)
        if current_excitation_value is not None:
            console.fill_input_field(
                "Current Excitation Value", str(current_excitation_value)
            )
