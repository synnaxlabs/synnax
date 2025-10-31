#  Copyright 2025 Synnax Labs, Inc.
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

        # Does not call super()

        self.console = console
        self.device = device
        self.name = name

        values = {}

        # Configure channel type
        console.click_btn("Channel Type")
        console.select_from_dropdown("Thermocouple")
        values["Channel Type"] = "Thermocouple"

        # Get device (set by task.add_channel)
        values["Device"] = console.get_dropdown_value("Device")

        # Optional configurations
        if port is not None:
            console.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = console.get_input_field("Port")

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

        self.form_values = values
