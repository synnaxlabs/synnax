#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.layout import LayoutClient
from console.task.channels.utils import is_numeric_string


class Counter:
    """Base class for counter read channel types in NI tasks."""

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
        min_val: float | None = None,
        max_val: float | None = None,
    ) -> None:
        """
        Initialize counter read channel with common configuration.

        Args:
            layout: LayoutClient for UI operations
            name: Channel name
            device: Device identifier
            chan_type: Channel type (e.g., "Edge Count", "Frequency")
            port: Physical port number
            min_val: Minimum value
            max_val: Maximum value
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

        # Min/Max values (not all counter types have these)
        if min_val is not None:
            layout.fill_input_field("Minimum Value", str(min_val))
            values["Minimum Value"] = str(min_val)
        elif self.has_min_max():
            values["Minimum Value"] = layout.get_input_field("Minimum Value")

        if max_val is not None:
            layout.fill_input_field("Maximum Value", str(max_val))
            values["Maximum Value"] = str(max_val)
        elif self.has_min_max():
            values["Maximum Value"] = layout.get_input_field("Maximum Value")

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

    def has_min_max(self) -> bool:
        """Check if this channel type has min/max value fields."""
        try:
            count: int = (
                self.layout.page.locator("text=Minimum Value")
                .locator("..")
                .locator("input")
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
            self.layout.press_key("Escape")
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
