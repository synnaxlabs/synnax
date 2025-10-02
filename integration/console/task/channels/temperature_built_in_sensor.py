#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal, Optional

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
        temperature_units: Optional[
            Literal[
                "Celsius",
                "Fahrenheit",
                "Kelvin",
                "Rankine",
            ]
        ] = None,
        **kwargs: Any,
    ) -> None:

        # Initialize base analog channel (remaining kwargs passed through)
        super().__init__(
            console=console,
            device=device,
            type="Temperature Built-In Sensor",
            **kwargs,
        )

        # Temperature Built-In Sensor-specific configurations:
        if temperature_units is not None:
            console.click_btn("Temperature Units")
            console.select_from_dropdown(temperature_units)
