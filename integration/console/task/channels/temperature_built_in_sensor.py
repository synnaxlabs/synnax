#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Literal

from console.task.channels.analog import Analog

if TYPE_CHECKING:
    from console.console import Console


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
        console: "Console",
        device: str,
        port: int | None = None,
        temperature_units: (
            Literal["Celsius", "Fahrenheit", "Kelvin", "Rankine"] | None
        ) = None,
    ) -> None:

        # Does not call super()

        self.console = console
        self.device = device

        values = {}

        # Configure channel type
        console.click_btn("Channel Type")
        console.select_from_dropdown("Temperature Built-In Sensor")
        values["Channel Type"] = "Temperature Built-In Sensor"

        # Get device (set by task.add_channel)
        values["Device"] = console.get_dropdown_value("Device")

        # Optional configurations
        if port is not None:
            console.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = console.get_input_field("Port")

        # Temperature Built-In Sensor-specific configurations:
        if temperature_units is not None:
            console.click_btn("Temperature Units")
            console.select_from_dropdown(temperature_units)

        self.form_values = values
