#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

from .symbol import Symbol


class Setpoint(Symbol):
    """Schematic setpoint/control symbol"""

    _symbol_group = "General"

    def __init__(self, *, label: str, channel_name: str, symbol_type: str = "Setpoint"):
        """Initialize a setpoint symbol with configuration.

        Args:
            label: Display label for the symbol
            channel_name: Channel name for the setpoint
            symbol_type: The type of symbol (default: "Setpoint")
        """
        super().__init__(label, symbol_type=symbol_type, rotatable=False)
        self.channel_name = channel_name

    def _apply_properties(self) -> None:
        """Apply setpoint configuration after being added to schematic."""
        self.set_properties(channel_name=self.channel_name)

    def set_properties(
        self,
        channel_name: str | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Set Setpoint properties including channel settings."""
        self.click()

        applied_properties: dict[str, Any] = {}
        if channel_name is not None:
            self.set_label(channel_name)

            # Navigate to Properties > Control tab
            self.page.get_by_text("Properties").click()
            self.page.get_by_text("Control").last.click()
            self.set_channel(input_field="Command Channel", channel_name=channel_name)
            applied_properties["channel"] = channel_name

        return applied_properties

    def set_value(self, value: float) -> None:
        self._disable_edit_mode()
        self.notifications.close_all()
        self.click()

        # Fill the input and set the value
        value_input = self.locator.locator("input[type='number'], input").first
        value_input.fill(str(value))
        set_button = self.locator.locator("button").filter(has_text="Set")
        set_button.click(timeout=500)
