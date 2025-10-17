#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Literal, Optional

if TYPE_CHECKING:
    from console.console import Console


class PulseOutput:
    """
    Pulse Output channel type for NI counter write tasks.

    Kwargs:
        port (int): Physical port number
        initial_delay (float): Initial delay in seconds
        high_time (float): High time in seconds
        low_time (float): Low time in seconds
        units (str): Scaled units (default: "Seconds")
        idle_state (str): "Low" or "High"
    """

    name: str
    console: "Console"
    device: str
    form_values: dict[str, str]

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        port: Optional[int] = None,
        initial_delay: Optional[float] = None,
        high_time: Optional[float] = None,
        low_time: Optional[float] = None,
        units: Optional[str] = None,
        idle_state: Optional[Literal["Low", "High"]] = None,
        **kwargs: Any,
    ) -> None:
        """Initialize pulse output channel with configuration."""
        self.console = console
        self.device = device
        self.name = name

        values = {}

        # Configure channel type
        console.click_btn("Channel Type")
        console.select_from_dropdown("Pulse Output")
        values["Channel Type"] = "Pulse Output"

        # Get device (set by task.add_channel)
        values["Device"] = console.get_dropdown_value("Device")

        # Port
        if port is not None:
            console.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = console.get_input_field("Port")

        # Initial Delay
        if initial_delay is not None:
            console.fill_input_field("Initial Delay", str(initial_delay))
            values["Initial Delay"] = str(initial_delay)
        else:
            values["Initial Delay"] = console.get_input_field("Initial Delay")

        # High Time
        if high_time is not None:
            console.fill_input_field("High Time", str(high_time))
            values["High Time"] = str(high_time)
        else:
            values["High Time"] = console.get_input_field("High Time")

        # Low Time
        if low_time is not None:
            console.fill_input_field("Low Time", str(low_time))
            values["Low Time"] = str(low_time)
        else:
            values["Low Time"] = console.get_input_field("Low Time")

        # Scaled Units
        if units is not None:
            console.click_btn("Scaled Units")
            console.select_from_dropdown(units)
            values["Scaled Units"] = units
        else:
            values["Scaled Units"] = console.get_dropdown_value("Scaled Units")

        # Idle State
        if idle_state is not None:
            console.click_btn("Idle State")
            console.select_from_dropdown(idle_state)
            values["Idle State"] = idle_state
        else:
            values["Idle State"] = console.get_dropdown_value("Idle State")

        self.form_values = values

    def assert_form(self) -> None:
        """Assert that form values match expected values."""
        for key, expected_value in self.form_values.items():
            try:
                actual_value = self.console.get_input_field(key)
            except:
                actual_value = self.console.get_dropdown_value(key)

            assert (
                actual_value == expected_value
            ), f"Channel {self.name} Form value '{key}' - Expected: {expected_value} - Actual: {actual_value}"
