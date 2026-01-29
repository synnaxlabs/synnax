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


class Resistance(Analog):
    """
    Resistance channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        resistance_configuration (str): "2-Wire", "3-Wire", "4-Wire"
        current_excitation_source (str): "Internal", "External", "None"
        current_excitation_value (float): Current excitation value

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
        resistance_configuration: Literal["2-Wire", "3-Wire", "4-Wire"] | None = None,
        current_excitation_source: (
            Literal["Internal", "External", "None"] | None
        ) = None,
        current_excitation_value: float | None = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            name=name,
            device=device,
            chan_type="Resistance",
            **kwargs,
        )
        layout = self.layout

        # Resistance-specific configurations:
        if resistance_configuration is not None:
            layout.click_btn("Resistance Configuration")
            layout.select_from_dropdown(resistance_configuration)

        if current_excitation_source is not None:
            layout.click_btn("Current Excitation Source")
            layout.select_from_dropdown(current_excitation_source)

        if current_excitation_value is not None:
            layout.fill_input_field(
                "Current Excitation Value", str(current_excitation_value)
            )
