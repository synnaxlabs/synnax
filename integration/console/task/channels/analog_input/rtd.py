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


class RTD(Analog):
    """
    RTD channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        temperature_units (str): "DegC", "DegF", "Kelvins", "DegR"
        rtd_type (str): "Pt3750", "Pt3851", "Pt3911", "Pt3916", "Pt3920", "Pt3928"
        resistance_configuration (str): "2Wire", "3Wire", "4Wire"
        current_excitation_source (str): "Internal", "External", "None"
        current_excitation_value (float): Current excitation value
        r0_resistance (float): R0 resistance value

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
        temperature_units: sy.ni.TemperatureUnits | None = None,
        rtd_type: sy.ni.RTDType | None = None,
        resistance_configuration: sy.ni.ResistanceConfig | None = None,
        current_excitation_source: sy.ni.ExcitationSource | None = None,
        current_excitation_value: float | None = None,
        r0_resistance: float | None = None,
        **kwargs: Unpack[Options],
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="RTD",
            **kwargs,
        )

        self._configure_dropdown("Temperature units", temperature_units)
        self._configure_dropdown("RTD type", rtd_type)
        self._configure_dropdown("Resistance configuration", resistance_configuration)
        self._configure_dropdown("Current excitation source", current_excitation_source)
        self._configure_input("Current excitation value", current_excitation_value)
        self._configure_input("R0 resistance", r0_resistance)
