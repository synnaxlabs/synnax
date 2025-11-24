#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from typing import Any

from .symbol import Symbol


class Value(Symbol):
    """Schematic value/telemetry symbol"""

    channel_name: str
    notation: str
    precision: int
    averaging_window: int
    stale_color: str
    stale_timeout: int

    def edit_properties(
        self,
        channel_name: str | None = None,
        *,
        notation: str | None = None,
        precision: int | None = None,
        averaging_window: int | None = None,
        stale_color: str | None = None,
        stale_timeout: int | None = None,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Edit Value symbol properties including channel and telemetry settings."""
        self._click_symbol()

        applied_properties: dict[str, Any] = {}

        # Always enforce label = channel_name for easy identification
        if channel_name is not None:
            self.set_label(channel_name)

        # Navigate to Properties > Telemetry tab
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Telemetry").click()

        if channel_name is not None:
            self.set_channel("Input Channel", channel_name)
            applied_properties["channel"] = channel_name

        if notation is not None:
            self.console.click_btn("Notation")
            # Note to self, check if the next line is redundant
            self.page.get_by_text(notation).click()
            applied_properties["notation"] = notation

        if precision is not None:
            self.console.fill_input_field("Precision", str(precision))
            self.page.keyboard.press("Enter")
            applied_properties["precision"] = precision

        if averaging_window is not None:
            self.console.fill_input_field("Averaging Window", str(averaging_window))
            self.page.keyboard.press("Enter")
            applied_properties["averaging_window"] = averaging_window

        if stale_color is not None:
            if not re.match(r"^#[0-9A-Fa-f]{6}$", stale_color):
                raise ValueError(
                    "stale_color must be a valid hex color (e.g., #FF5733)"
                )
            self.console.click_btn("Color")
            self.console.fill_input_field("Hex", stale_color.replace("#", ""))
            self.page.keyboard.press("Enter")
            self.page.keyboard.press("Escape")
            applied_properties["stale_color"] = stale_color

        if stale_timeout is not None:
            self.console.fill_input_field("Stale Timeout", str(stale_timeout))
            self.page.keyboard.press("Enter")

            applied_properties["stale_timeout"] = stale_timeout

        return applied_properties

    def get_properties(self, tab: str = "Telemetry") -> dict[str, Any]:
        """Get the current properties of the symbol"""
        console = self.console
        super().get_properties(tab=tab)

        props: dict[str, Any] = {
            "channel": "",
            "notation": "",
            "precision": -1,
            "averaging_window": -1,
            "stale_color": "",
            "stale_timeout": -1,
        }

        # Channel Name
        channel_display = (
            self.page.locator("text=Input Channel").locator("..").locator("button")
        )
        if channel_display.count() > 0:
            props["channel"] = channel_display.inner_text().strip()

        # Precision
        props["precision"] = int(console.get_input_field("Precision"))

        # Averaging Window
        props["averaging_window"] = int(console.get_input_field("Averaging Window"))

        # Staleness Timeout
        props["stale_timeout"] = int(console.get_input_field("Stale Timeout"))

        # Notation
        notation_options = ["Scientific", "Engineering", "Standard"]

        notation = console.get_selected_button(notation_options)
        props["notation"] = notation.lower()

        # Staleness Color - get hex value from color picker
        console.click_btn("Color")
        hex_value = console.get_input_field("Hex")
        if hex_value:
            props["stale_color"] = (
                f"#{hex_value}" if not hex_value.startswith("#") else hex_value
            )
        # Close color picker
        self.page.keyboard.press("Escape")

        return props
