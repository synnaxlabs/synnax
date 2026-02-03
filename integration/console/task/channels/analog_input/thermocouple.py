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


class Thermocouple(Analog):
    """
    Thermocouple channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        temperature_units (str): "Celsius", "Fahrenheit", "Kelvin", "Rankine"
        thermocouple_type (str): "B", "E", "J", "K", "N", "R", "S", "T"
        cjc_source (str): "Built In", "Constant Value", "Channel"
        cjc_value (float): CJC value when using "Constant Value" source
        cjc_port (int): CJC port when using "Channel" source

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
        port: int | None = None,
        temperature_units: (
            Literal["Celsius", "Fahrenheit", "Kelvin", "Rankine"] | None
        ) = None,
        thermocouple_type: (
            Literal["B", "E", "J", "K", "N", "R", "S", "T"] | None
        ) = None,
        cjc_source: Literal["Built In", "Constant Value", "Channel"] | None = None,
        cjc_value: float | None = None,
        cjc_port: int | None = None,
        **kwargs: Any,
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Thermocouple",
            port=port,
            **kwargs,
        )

        self._configure_dropdown("Temperature Units", temperature_units)
        self._configure_dropdown("Thermocouple Type", thermocouple_type)
        self._configure_dropdown("CJC Source", cjc_source)

        if cjc_source == "Constant Value":
            self._configure_input("CJC Value", cjc_value)
        elif cjc_source == "Channel":
            self._configure_input("CJC Port", cjc_port)
