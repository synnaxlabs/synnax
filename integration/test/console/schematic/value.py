#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from typing import Any, Dict, Optional

from playwright.sync_api import Page

from .schematic_symbol import SchematicSymbol


class Value(SchematicSymbol):
    """Schematic value/telemetry symbol"""

    channel_name: str
    notation: str
    precision: int
    averaging_window: int
    stale_color: str
    stale_timeout: int

    def __init__(self, page: Page, symbol_id: str, channel_name: str):
        super().__init__(page, symbol_id, channel_name)

    def edit_properties(
        self,
        channel_name: Optional[str] = None,
        *,
        notation: Optional[str] = None,
        precision: Optional[int] = None,
        averaging_window: Optional[int] = None,
        stale_color: Optional[str] = None,
        stale_timeout: Optional[int] = None,
        **kwargs: Any,
    ) -> Dict[str, Any]:
        """Edit Value symbol properties including channel and telemetry settings."""
        print("Editing value properties")
        self._click_symbol()
        print("Symbol clicked")
        # Always enforce label = channel_name for easy identification
        if channel_name is not None:
            self.set_label(channel_name)

        # Navigate to Properties > Telemetry tab
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Telemetry").click()

        if channel_name is not None:
            self.set_channel("Input Channel", channel_name)

        if notation is not None:
            notation_button = (
                self.page.locator("text=Notation").locator("..").locator("button").first
            )
            notation_button.click()
            self.page.get_by_text(notation).click()

        if precision is not None:
            precision_input = (
                self.page.locator("text=Precision").locator("..").locator("input")
            )
            precision_input.fill(str(precision))
            precision_input.press("Enter")

        if averaging_window is not None:
            averaging_window_input = (
                self.page.locator("text=Averaging Window")
                .locator("..")
                .locator("input")
            )
            averaging_window_input.fill(str(averaging_window))
            averaging_window_input.press("Enter")

        if stale_color is not None:
            if not re.match(r"^#[0-9A-Fa-f]{6}$", stale_color):
                raise ValueError(
                    "stale_color must be a valid hex color (e.g., #FF5733)"
                )

            color_button = (
                self.page.locator("text=Color").locator("..").locator("button")
            )
            color_button.click()
            hex_input = self.page.locator("text=Hex").locator("..").locator("input")
            hex_input.fill(stale_color.replace("#", ""))
            hex_input.press("Enter")
            self.page.keyboard.press("Escape")

        if stale_timeout is not None:
            stale_timeout_input = (
                self.page.locator("text=Stale Timeout").locator("..").locator("input")
            )
            stale_timeout_input.fill(str(stale_timeout))
            stale_timeout_input.press("Enter")

        applied_properties: Dict[str, Any] = {}
        if channel_name is not None:
            applied_properties["channel"] = channel_name
        if notation is not None:
            applied_properties["notation"] = notation
        if precision is not None:
            applied_properties["precision"] = precision
        if averaging_window is not None:
            applied_properties["averaging_window"] = averaging_window
        if stale_color is not None:
            applied_properties["stale_color"] = stale_color
        if stale_timeout is not None:
            applied_properties["stale_timeout"] = stale_timeout

        return applied_properties

    def get_properties(self) -> Dict[str, Any]:
        """Get the current properties of the symbol"""
        self._click_symbol()
        self.page.get_by_text("Properties").click()
        self.page.get_by_text("Telemetry").click()

        props: Dict[str, Any] = {
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
        precision_input = (
            self.page.locator("text=Precision").locator("..").locator("input")
        )
        props["precision"] = int(precision_input.input_value())

        # Averaging Window
        avg_input = (
            self.page.locator("text=Averaging Window").locator("..").locator("input")
        )
        props["averaging_window"] = int(avg_input.input_value())

        # Staleness Timeout
        timeout_input = (
            self.page.locator("text=Stale Timeout").locator("..").locator("input")
        )
        props["stale_timeout"] = int(timeout_input.input_value())

        # Notation
        notation_options = ["Scientific", "Engineering", "Standard"]
        for option in notation_options:
            try:
                button = self.page.get_by_text(option).first
                if button.count() > 0:
                    class_name = button.get_attribute("class") or ""
                    if "pluto-btn--filled" in class_name:
                        props["notation"] = str(option).lower()
                        break
            except:
                continue

        # Staleness Color - get hex value from color picker
        color_button = self.page.locator("text=Color").locator("..").locator("button")
        color_button.click()
        hex_input = self.page.locator("text=Hex").locator("..").locator("input")
        if hex_input.count() > 0:
            hex_value = hex_input.input_value()
            if hex_value:
                props["stale_color"] = (
                    f"#{hex_value}" if not hex_value.startswith("#") else hex_value
                )
        # Close color picker
        self.page.keyboard.press("Escape")

        return props
