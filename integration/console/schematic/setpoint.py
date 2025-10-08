#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Dict, Optional

from .symbol import Symbol


class Setpoint(Symbol):
    """Schematic setpoint/control symbol"""

    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
        """Edit Setpoint properties including channel settings."""
        self._click_symbol()

        applied_properties: Dict[str, Any] = {}
        if channel_name is not None:
            self.set_label(channel_name)

            # Navigate to Properties > Control tab
            self.page.get_by_text("Properties").click()
            self.page.get_by_text("Control").last.click()
            self.set_channel("Command Channel", channel_name)
            applied_properties["channel"] = channel_name

        return applied_properties

    def set_value(self, value: float) -> None:
        self._disable_edit_mode()
        self._click_symbol()

        # Fill the input and set the value
        value_input = self.symbol.locator("input[type='number'], input").first
        value_input.fill(str(value))
        set_button = self.symbol.locator("button").filter(has_text="Set")
        set_button.click()
