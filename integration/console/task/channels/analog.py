#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Literal

if TYPE_CHECKING:
    from console.console import Console


class Analog:
    """Base class for analog channel types in NI tasks."""

    name: str
    console: "Console"
    device: str
    form_values: dict[str, str | bool]

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        chan_type: str,
        port: int | None = None,
        terminal_config: (
            Literal[
                "Default",
                "Differential",
                "Pseudo-Differential",
                "Referenced Single Ended",
                "Non-Referenced Single Ended",
            ]
            | None
        ) = None,
        min_val: float | None = None,
        max_val: float | None = None,
        custom_scale: Literal["None", "Linear", "Map", "Table"] | None = None,
    ) -> None:
        """
        Initialize analog channel with common configuration.

        Args:
            console: Console automation interface
            name: Channel name
            device: Device identifier
            chan_type: Channel type (e.g., "Voltage", "Accelerometer")
            port: Physical port number
            terminal_config: "Default", "Differential", "Pseudo-Differential",
                           "Referenced Single Ended", "Non-Referenced Single Ended"
            min_val: Minimum value
            max_val: Maximum value
            custom_scale: "None", "Linear", "Map", "Table"
        """
        self.console = console
        self.device = device
        self.name = name

        values: dict[str, str | bool] = {}

        # Configure channel type
        console.click_btn("Channel Type")
        console.select_from_dropdown(chan_type)
        values["Channel Type"] = chan_type

        # Get device (set by task.add_channel)
        values["Device"] = console.get_dropdown_value("Device")

        # Optional configurations
        if port is not None:
            console.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = console.get_input_field("Port")

        if terminal_config is not None:
            console.click_btn("Terminal Configuration")
            console.select_from_dropdown(terminal_config)
            values["Terminal Configuration"] = terminal_config
        elif self.has_terminal_config():
            values["Terminal Configuration"] = console.get_dropdown_value(
                "Terminal Configuration"
            )

        if min_val is not None:
            console.fill_input_field("Minimum Value", str(min_val))
            values["Minimum Value"] = str(min_val)
        elif chan_type != "Microphone":
            values["Minimum Value"] = console.get_input_field("Minimum Value")

        if max_val is not None:
            console.fill_input_field("Maximum Value", str(max_val))
            values["Maximum Value"] = str(max_val)
        elif chan_type != "Microphone":
            values["Maximum Value"] = console.get_input_field("Maximum Value")

        if custom_scale is not None:
            console.click_btn("Custom Scaling")
            console.select_from_dropdown(custom_scale)
            values["Custom Scaling"] = custom_scale
        elif chan_type != "RTD":
            values["Custom Scaling"] = console.get_dropdown_value("Custom Scaling")

        self.form_values = values

    def assert_form(self) -> None:
        """Assert that form values match expected values."""
        for key, expected_value in self.form_values.items():
            actual_value: str | bool
            if isinstance(expected_value, bool):
                actual_value = self.console.get_toggle(key)
            elif self._is_numeric_string(expected_value):
                actual_value = self.console.get_input_field(key)
            else:
                actual_value = self.console.get_dropdown_value(key)

            assert (
                actual_value == expected_value
            ), f"Channel {self.name} Form value '{key}' - Expected: {expected_value} - Actual: {actual_value}"

    @staticmethod
    def _is_numeric_string(value: str | bool) -> bool:
        """Check if a string represents a numeric value."""
        if not isinstance(value, str):
            return False
        # Remove leading/trailing whitespace and check if it's a valid number
        value = value.strip()
        if not value:
            return False
        # Handle negative numbers and decimals
        if value.startswith("-"):
            value = value[1:]
        # Split on decimal point
        parts = value.split(".")
        if len(parts) > 2:
            return False
        # Check all parts are digits
        return all(part.isdigit() for part in parts if part)

    def has_terminal_config(self) -> bool:
        try:
            return (
                self.console.page.locator("text=Terminal Configuration")
                .locator("..")
                .locator("button")
                .first.count()
                > 0
            )
        except Exception:
            return False
