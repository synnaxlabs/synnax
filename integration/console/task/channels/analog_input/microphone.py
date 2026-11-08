#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Unpack

import synnax as sy
from console.layout import LayoutClient
from console.task.channels.analog import Analog, Options


class Microphone(Analog):
    """
    Microphone channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        sound_pressure_units (str): "Pascals", "PoundsPerSquareInch", etc.
        microphone_sensitivity (float): Microphone sensitivity value
        max_sound_pressure_level (float): Maximum sound pressure level
        current_excitation_source (str): "Internal", "External", "None"
        current_excitation_value (float): Current excitation value

    Base kwargs from Analog:
        port (int): Physical port number
        terminal_config (str): "Cfg_Default", "Diff", etc.
        min_val (float): Minimum value
        max_val (float): Maximum value
        custom_scale (str): "none", "linear", "map", "table"
    """

    def __init__(
        self,
        layout: LayoutClient,
        name: str,
        device: str,
        *,
        sound_pressure_units: sy.ni.PressureUnits | None = None,
        sensitivity: float | None = None,
        max_sound_pressure_level: float | None = None,
        current_excitation_source: sy.ni.ExcitationSource | None = None,
        current_excitation_value: float | None = None,
        **kwargs: Unpack[Options],
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Microphone",
            **kwargs,
        )

        self._configure_dropdown("Sound pressure units", sound_pressure_units)
        self._configure_input("Microphone sensitivity", sensitivity)
        self._configure_input("Max sound pressure level", max_sound_pressure_level)
        self._configure_dropdown("Current excitation source", current_excitation_source)
        self._configure_input("Current excitation value", current_excitation_value)
