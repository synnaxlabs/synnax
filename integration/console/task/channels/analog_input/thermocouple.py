#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal, Unpack

import synnax as sy
from console.layout import LayoutClient
from console.task.channels.analog import Analog, Options


class Thermocouple(Analog):
    """
    Thermocouple channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        temperature_units (str): "DegC", "DegF", "Kelvins", "DegR"
        thermocouple_type (str): "B", "E", "J", "K", "N", "R", "S", "T"
        cjc_source (str): "built_in", "const_val", "chan"
        cjc_value (float): CJC value when using "const_val" source
        cjc_port (int): CJC port when using "chan" source

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
        thermocouple_type: sy.ni.ThermocoupleType | None = None,
        cjc_source: Literal["built_in", "const_val", "chan"] | None = None,
        cjc_value: float | None = None,
        cjc_port: int | None = None,
        **kwargs: Unpack[Options],
    ) -> None:
        super().__init__(
            layout=layout,
            name=name,
            device=device,
            chan_type="Thermocouple",
            **kwargs,
        )

        self._configure_dropdown("Temperature units", temperature_units)
        self._configure_dropdown("Thermocouple type", thermocouple_type)
        self._configure_dropdown("CJC source", cjc_source)

        if cjc_source == "const_val":
            self._configure_input("CJC value", cjc_value)
        elif cjc_source == "chan":
            self._configure_input("CJC port", cjc_port)
