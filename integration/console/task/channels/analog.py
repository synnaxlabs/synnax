#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Literal

from console.layout import LayoutClient
from console.task.channels.utils import is_numeric_string


class Analog:
    """Base class for analog channel types in NI tasks."""

    name: str
    layout: LayoutClient
    device: str
    form_values: dict[str, str | bool]

    def __init__(
        self,
        layout: LayoutClient,
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
            layout: LayoutClient for UI operations
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
        self.layout = layout
        self.device = device
        self.name = name

        values: dict[str, str | bool] = {}

        # Configure channel type
        layout.click_btn("Channel Type")
        layout.select_from_dropdown(chan_type)
        values["Channel Type"] = chan_type

        # Get device (set by task.add_channel)
        values["Device"] = layout.get_dropdown_value("Device")

        # Optional configurations
        if port is not None:
            layout.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = layout.get_input_field("Port")

        if terminal_config is not None:
            layout.click_btn("Terminal Configuration")
            layout.select_from_dropdown(terminal_config)
            values["Terminal Configuration"] = terminal_config
        elif self.has_terminal_config():
            values["Terminal Configuration"] = layout.get_dropdown_value(
                "Terminal Configuration"
            )

        no_min_max_types = ("Microphone", "Temperature Built-In Sensor", "Thermocouple")
        if min_val is not None:
            layout.fill_input_field("Minimum Value", str(min_val))
            values["Minimum Value"] = str(min_val)
        elif chan_type not in no_min_max_types:
            values["Minimum Value"] = layout.get_input_field("Minimum Value")

        if max_val is not None:
            layout.fill_input_field("Maximum Value", str(max_val))
            values["Maximum Value"] = str(max_val)
        elif chan_type not in no_min_max_types:
            values["Maximum Value"] = layout.get_input_field("Maximum Value")

        no_custom_scale_types = ("RTD", "Temperature Built-In Sensor", "Thermocouple")
        if custom_scale is not None:
            layout.click_btn("Custom Scaling")
            layout.select_from_dropdown(custom_scale)
            values["Custom Scaling"] = custom_scale
        elif chan_type not in no_custom_scale_types:
            values["Custom Scaling"] = layout.get_dropdown_value("Custom Scaling")

        self.form_values = values

    def assert_form(self) -> None:
        """Assert that form values match expected values."""
        for key, expected_value in self.form_values.items():
            actual_value: str | bool
            if isinstance(expected_value, bool):
                actual_value = self.layout.get_toggle(key)
            elif is_numeric_string(expected_value):
                actual_value = self.layout.get_input_field(key)
            else:
                actual_value = self.layout.get_dropdown_value(key)

            assert (
                actual_value == expected_value
            ), f"Channel {self.name} Form value '{key}' - Expected: {expected_value} - Actual: {actual_value}"

    def has_terminal_config(self) -> bool:
        try:
            count: int = (
                self.layout.page.locator("text=Terminal Configuration")
                .locator("..")
                .locator("button")
                .first.count()
            )
            return count > 0
        except Exception:
            return False

    def _configure_dropdown(
        self,
        label: str,
        value: str | None,
        *,
        track: bool = True,
    ) -> None:
        """Configure a dropdown field.

        Args:
            label: The UI label for the dropdown
            value: The value to select, or None to read current value
            track: Whether to track the value in form_values
        """
        if value is not None:
            self.layout.click_btn(label)
            self.layout.select_from_dropdown(value)
            if track:
                self.form_values[label] = value
        elif track:
            self.form_values[label] = self.layout.get_dropdown_value(label)

    def _configure_input(
        self,
        label: str,
        value: str | float | int | None,
        *,
        track: bool = True,
    ) -> None:
        """Configure an input field.

        Args:
            label: The UI label for the input field
            value: The value to set, or None to read current value
            track: Whether to track the value in form_values
        """
        if value is not None:
            self.layout.fill_input_field(label, str(value))
            # Blur the input to trigger UI normalization (e.g., "4.0" -> "4")
            self.layout.press_key("Tab")
            if track:
                self.form_values[label] = self.layout.get_input_field(label)
        elif track:
            self.form_values[label] = self.layout.get_input_field(label)

    def _configure_toggle(
        self,
        label: str,
        value: bool | None,
        *,
        track: bool = True,
    ) -> None:
        """Configure a toggle/checkbox field.

        Args:
            label: The UI label for the toggle
            value: The desired state, or None to read current value
            track: Whether to track the value in form_values
        """
        if value is not None:
            current = self.layout.get_toggle(label)
            if current != value:
                self.layout.click_checkbox(label)
            if track:
                self.form_values[label] = value
        elif track:
            self.form_values[label] = self.layout.get_toggle(label)
