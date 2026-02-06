#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal

from console.layout import LayoutClient
from console.task.channels.analog import Analog


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
        layout: LayoutClient,
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

        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Microphone",
            **kwargs,
        )

        self._configure_dropdown("Sound Pressure Units", sound_pressure_units)
        self._configure_input("Microphone Sensitivity", sensitivity)
        self._configure_input("Max Sound Pressure Level", max_sound_pressure_level)
        self._configure_dropdown("Current Excitation Source", current_excitation_source)
        self._configure_input("Current Excitation Value", current_excitation_value)
