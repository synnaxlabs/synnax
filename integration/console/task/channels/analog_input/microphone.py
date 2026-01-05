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


class Microphone(Analog):
    """
    Microphone channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        sound_pressure_units (str): "Pascals", "PSI", etc.
        microphone_sensitivity (float): Microphone sensitivity value
        max_sound_pressure_level (float): Maximum sound pressure level
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
        sound_pressure_units: Literal["Pascals"] | None = None,
        sensitivity: float | None = None,
        max_sound_pressure_level: float | None = None,
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
            chan_type="Microphone",
            **kwargs,
        )

        # Microphone-specific configurations:
        if sound_pressure_units is not None:
            console.click_btn("Sound Pressure Units")
            console.select_from_dropdown(sound_pressure_units)

        if sensitivity is not None:
            console.fill_input_field("Microphone Sensitivity", str(sensitivity))

        if max_sound_pressure_level is not None:
            console.fill_input_field(
                "Max Sound Pressure Level", str(max_sound_pressure_level)
            )

        if current_excitation_source is not None:
            console.click_btn("Current Excitation Source")
            console.select_from_dropdown(current_excitation_source)

        if current_excitation_value is not None:
            console.fill_input_field(
                "Current Excitation Value", str(current_excitation_value)
            )
