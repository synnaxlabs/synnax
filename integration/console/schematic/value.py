#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re

from console.schematic.symbol import Properties, Symbol
from x.notation import Notation

_NOTATION_LABELS: dict[Notation, str] = {
    "standard": "Standard",
    "scientific": "Scientific",
    "engineering": "Engineering",
}


class Value(Symbol):
    """Schematic value/telemetry symbol"""

    _symbol_group: str | None = "General"

    def __init__(
        self,
        *,
        label: str,
        channel_name: str,
        notation: Notation | None = None,
        precision: int | None = None,
        averaging_window: int | None = None,
        stale_color: str | None = None,
        stale_timeout: int | None = None,
        symbol_type: str = "Value",
    ):
        """Initialize a value symbol with configuration.

        :param label: Display label for the symbol
        :param channel_name: Channel name for the value display
        :param notation: Number notation format (optional)
        :param precision: Decimal precision (optional)
        :param averaging_window: Averaging window size (optional)
        :param stale_color: Color for stale data (optional)
        :param stale_timeout: Timeout for stale data in milliseconds (optional)
        :param symbol_type: The type of symbol (default: "Value")
        """
        super().__init__(label=label, symbol_type=symbol_type, rotatable=False)
        self.channel_name = channel_name
        self.notation = notation
        self.precision = precision
        self.averaging_window = averaging_window
        self.stale_color = stale_color
        self.stale_timeout = stale_timeout

    def _apply_properties(self) -> None:
        """Apply value configuration after being added to schematic."""
        self.set_properties(
            channel_name=self.channel_name,
            notation=self.notation,
            precision=self.precision,
            averaging_window=self.averaging_window,
            stale_color=self.stale_color,
            stale_timeout=self.stale_timeout,
        )

    def set_properties(
        self,
        *,
        channel_name: str | None = None,
        notation: Notation | None = None,
        precision: int | None = None,
        averaging_window: int | None = None,
        stale_color: str | None = None,
        stale_timeout: int | None = None,
    ) -> None:
        """Set Value symbol properties including channel and telemetry settings."""
        self.click()

        # Always enforce label = channel_name for easy identification
        if channel_name is not None:
            self.set_label(channel_name)

        # Navigate to Properties > Telemetry tab
        self.open_properties_tab()
        self.page.get_by_text("Telemetry").click()

        if channel_name is not None:
            self.set_channel(section="Source", channel_name=channel_name)

        if notation is not None:
            self.layout.click_btn("Notation")
            # Note to self, check if the next line is redundant
            self.page.get_by_text(_NOTATION_LABELS[notation]).click()

        if precision is not None:
            self.layout.fill_input_field("Precision", str(precision))
            self.page.keyboard.press("Enter")

        if averaging_window is not None:
            self.layout.fill_input_field("Averaging window", str(averaging_window))
            self.page.keyboard.press("Enter")

        if stale_color is not None:
            if not re.match(r"^#[0-9A-Fa-f]{6}$", stale_color):
                raise ValueError(
                    "stale_color must be a valid hex color (e.g., #FF5733)"
                )
            self.layout.fill_input_field("Color", stale_color.replace("#", ""))
            self.page.keyboard.press("Enter")

        if stale_timeout is not None:
            self.layout.fill_input_field("Timeout", str(stale_timeout))
            self.page.keyboard.press("Enter")

    def get_properties(self, *, tab: str = "Telemetry") -> Properties:
        """Get the current properties of the symbol"""
        super().get_properties(tab=tab)

        props: Properties = {
            "channel": "",
            "precision": -1,
            "averaging_window": -1,
            "stale_color": "",
            "stale_timeout": -1,
        }

        # Channel Name
        channel_display = (
            self.page.locator('text="Channel"').locator("..").locator("button")
        )
        if channel_display.count() > 0:
            props["channel"] = channel_display.inner_text().strip()

        # Precision
        props["precision"] = int(self.layout.get_input_field("Precision"))

        # Averaging Window
        props["averaging_window"] = int(self.layout.get_input_field("Averaging window"))

        # Staleness Timeout
        props["stale_timeout"] = int(self.layout.get_time_span_field("Timeout").seconds)

        # Notation
        selected = self.layout.get_selected_button(list(_NOTATION_LABELS.values()))
        props["notation"] = next(
            notation
            for notation, label in _NOTATION_LABELS.items()
            if label == selected
        )

        # The staleness color box is empty while the theme picks the color.
        hex_value = self.layout.get_input_field("Color")
        if hex_value:
            props["stale_color"] = f"#{hex_value.upper()}"

        return props
