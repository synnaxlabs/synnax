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
        console: "Console",
        device: str,
        temperature_units: Optional[Literal[
            "Celsius",
            "Fahrenheit",
            "Kelvin",
            "Rankine",
        ]] = None,
        thermocouple_type: Optional[Literal[
            "B",
            "E",
            "J",
            "K",
            "N",
            "R",
            "S",
            "T",
        ]] = None,
        cjc_source: Optional[Literal[
            "Built In",
            "Constant Value",
            "Channel",
        ]] = None,
        cjc_value: Optional[float] = None,
        cjc_port: Optional[int] = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            device=device,
            type="Thermocouple",
            **kwargs,
        )

        # Thermocouple-specific configurations:
        if temperature_units is not None:
            console.click_btn("Temperature Units")
            console.select_from_dropdown(temperature_units)

        if thermocouple_type is not None:
            console.click_btn("Thermocouple Type")
            console.select_from_dropdown(thermocouple_type)

        if cjc_source is not None:
            console.click_btn("CJC Source")
            console.select_from_dropdown(cjc_source)

        if cjc_value is not None and cjc_source == "Constant Value":
            console.fill_input_field("CJC Value", str(cjc_value))

        if cjc_port is not None and cjc_source == "Channel":
            console.fill_input_field("CJC Port", str(cjc_port))
