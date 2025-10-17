#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Optional

if TYPE_CHECKING:
    from console.console import Console


class Counter:
    """Base class for counter read channel types in NI tasks."""

    name: str
    console: "Console"
    device: str
    form_values: dict[str, str]

    def __init__(
        self,
        console: "Console",
        name: str,
        device: str,
        type: str,
        port: Optional[int] = None,
        min_val: Optional[float] = None,
        max_val: Optional[float] = None,
    ) -> None:
        """
        Initialize counter read channel with common configuration.

        Args:
            console: Console automation interface
            name: Channel name
            device: Device identifier
            type: Channel type (e.g., "Edge Count", "Frequency")
            port: Physical port number
            min_val: Minimum value
            max_val: Maximum value
        """
        self.console = console
        self.device = device
        self.name = name

        values = {}

        # Configure channel type
        console.click_btn("Channel Type")
        console.select_from_dropdown(type)
        values["Channel Type"] = type

        # Get device (set by task.add_channel)
        values["Device"] = console.get_dropdown_value("Device")

        # Optional configurations
        if port is not None:
            console.fill_input_field("Port", str(port))
            values["Port"] = str(port)
        else:
            values["Port"] = console.get_input_field("Port")

        # Min/Max values (not all counter types have these)
        if min_val is not None:
            console.fill_input_field("Minimum Value", str(min_val))
            values["Minimum Value"] = str(min_val)
        elif self.has_min_max():
            values["Minimum Value"] = console.get_input_field("Minimum Value")

        if max_val is not None:
            console.fill_input_field("Maximum Value", str(max_val))
            values["Maximum Value"] = str(max_val)
        elif self.has_min_max():
            values["Maximum Value"] = console.get_input_field("Maximum Value")

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

    def has_min_max(self) -> bool:
        """Check if this channel type has min/max value fields."""
        try:
            return (
                self.console.page.locator("text=Minimum Value")
                .locator("..")
                .locator("input")
                .first.count()
                > 0
            )
        except:
            return False
