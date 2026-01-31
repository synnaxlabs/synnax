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


class TemperatureBuiltInSensor(Analog):
    """
    Temperature Built-In Sensor channel type for NI analog read tasks.

    Supported kwargs (in addition to Analog base kwargs):
        temperature_units (str): "Celsius", "Fahrenheit", "Kelvin", "Rankine"

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
        **kwargs: Any,
    ) -> None:

        # Does not call super()

        self.layout = layout
        layout = self.layout
        self.name = name
        self.device = device

        values: dict[str, str | bool] = {}

        # Configure channel type
        layout.click_btn("Channel Type")
        layout.select_from_dropdown("Temperature Built-In Sensor")
        values["Channel Type"] = "Temperature Built-In Sensor"

        # Get device (set by task.add_channel)
        values["Device"] = layout.get_dropdown_value("Device")

        # Optional configurations
        if port is not None:
            layout.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = layout.get_input_field("Port")

        # Temperature Built-In Sensor-specific configurations:
        if temperature_units is not None:
            layout.click_btn("Temperature Units")
            layout.select_from_dropdown(temperature_units)

        self.form_values = values
