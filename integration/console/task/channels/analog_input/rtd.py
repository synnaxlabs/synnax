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
        layout: LayoutClient,
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

        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="RTD",
            **kwargs,
        )

        self._configure_dropdown("Temperature Units", temperature_units)
        self._configure_dropdown("RTD Type", rtd_type)
        self._configure_dropdown("Resistance Configuration", resistance_configuration)
        self._configure_dropdown("Current Excitation Source", current_excitation_source)
        self._configure_input("Current Excitation Value", current_excitation_value)
        self._configure_input("R0 Resistance", r0_resistance)
