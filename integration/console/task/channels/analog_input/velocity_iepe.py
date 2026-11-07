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


class VelocityIEPE(Analog):
    """
    Velocity IEPE channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        velocity_units (str): "MetersPerSecond", "InchesPerSecond"
        sensitivity (float): Velocity sensitivity value
        sensitivity_units (str): "MillivoltsPerMillimeterPerSecond" or
            "MilliVoltsPerInchPerSecond"
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
        velocity_units: sy.ni.VelocityUnits | None = None,
        sensitivity: float | None = None,
        sensitivity_units: sy.ni.VelocitySensitivityUnits | None = None,
        current_excitation_source: sy.ni.ExcitationSource | None = None,
        current_excitation_value: float | None = None,
        **kwargs: Unpack[Options],
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Velocity IEPE",
            **kwargs,
        )

        self._configure_dropdown("Velocity units", velocity_units)
        self._configure_input("Sensitivity", sensitivity)

        self._configure_symbol_dropdown("mV/", sensitivity_units)

        self._configure_dropdown("Current excitation source", current_excitation_source)
        self._configure_input("Current excitation value", current_excitation_value)
