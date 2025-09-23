#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import TYPE_CHECKING, Any, Dict, List, Union

from playwright.sync_api import Page

from .console_page import ConsolePage

if TYPE_CHECKING:
    from ..console import Console


class Plot(ConsolePage):
    """Plot page management interface"""

    def __init__(self, page: Page, console: "Console") -> None:
        super().__init__(page, console)
        self.page_type = "Line Plot"
        self.pluto_label = ".pluto-line-plot"

        self.data: Dict[str, Any] = {
            "Y1": [],
            "Y2": [],
            "Ranges": [],
            "X1": None,
        }

    def add_Y(self, axis: str, channel_ids: Union[str, List[str]]) -> None:
        if axis not in ("Y1", "Y2"):
            raise ValueError(f"Invalid axis: {axis}. Must be 'Y1' or 'Y2'")

        selector = self.page.get_by_text(f"{axis} Select Channels", exact=True)
        selector.click(timeout=5000)

        self.page.get_by_text("Retrieving Channels").wait_for(state="hidden")
        channels = [channel_ids] if isinstance(channel_ids, str) else channel_ids

        # Add each channel
        for channel in channels:
            self.console._select_from_dropdown_item(
                channel, "input[placeholder*='Search']"
            )
            self.data[axis].append(channel)

        self.console.ESCAPE

    def add_ranges(self, ranges: List[str]) -> None:
        """Add time ranges to the plot."""
        valid_ranges = {"30s", "1m", "5m", "15m", "30m"}
        self.page.get_by_text("Select Ranges").click()

        for range_value in ranges:
            if range_value in valid_ranges and range_value not in self.data["Ranges"]:
                self.page.get_by_text(range_value, exact=True).click()
                self.data["Ranges"].append(range_value)

        self.console.ESCAPE

    def set_X1_axis(self, config: Dict[str, Any]) -> None:
        """Set X1 axis configuration."""
        self.set_axis("X1", config)

    def set_Y1_axis(self, config: Dict[str, Any]) -> None:
        """Set Y1 axis configuration."""
        self.set_axis("Y1", config)

    def set_Y2_axis(self, config: Dict[str, Any]) -> None:
        """Set Y2 axis configuration."""
        self.set_axis("Y2", config)

    def set_axis(self, axis: str, config: Dict[str, Any]) -> None:
        """Set axis configuration with the given parameters."""
        self.page.get_by_text("Axes").click(timeout=5000)
        self.page.wait_for_selector(".pluto-tabs-selector__btn", timeout=5000)

        self._select_axis_tab(axis)

        for key, value in config.items():
            self._set_axis_property(key, value)

        self.console.ENTER

    def _select_axis_tab(self, axis: str) -> None:
        """Select the axis tab in the configuration panel."""
        selectors = [
            f"#{axis.lower()}",
            f"#{axis}",
            f".pluto-tabs-selector__btn:has-text('{axis}')",
        ]

        for selector in selectors:
            try:
                locator = self.page.locator(selector)
                if locator.count() > 0:
                    locator.click(timeout=5000)
                    return
            except Exception:
                continue

        raise RuntimeError(f"Could not find axis tab: {axis}")

    def _set_axis_property(self, key: str, value: Any) -> None:
        """Set a single axis property."""
        try:
            if key in {"Lower Bound", "Upper Bound", "Tick Spacing", "Label"}:
                self._set_input_field(key, value)
            elif key == "Label Direction":
                self._set_label_direction(value)
            elif key == "Label Size":
                self._set_label_size(value)
            else:
                self.page.locator(key).fill(str(value), timeout=5000)
        except Exception as e:
            raise ValueError(f'Failed to set axis property "{key}" to "{value}": {e}')

    def _set_input_field(self, key: str, value: Any) -> None:
        """Set an input field value."""
        selectors = [
            f"label:has-text('{key}') + div input",
            f"label:has-text('{key}') input",
            f"input[aria-label*='{key}']",
            f"input[placeholder*='{key}']",
        ]

        for selector in selectors:
            try:
                input_field = self.page.locator(selector)
                if input_field.count() > 0:
                    input_field.clear(timeout=5000)
                    input_field.fill(str(value), timeout=5000)
                    return
            except Exception:
                continue

        raise RuntimeError(f"Could not find input field for {key}")

    def _set_label_direction(self, value: Any) -> None:
        """Set label direction button."""
        direction = "arrow-up" if str(value).lower() == "up" else "arrow-right"
        selector = f"label:has-text('Label Direction') + div button:has([aria-label='pluto-icon--{direction}'])"
        self.page.locator(selector).click(timeout=5000)

    def _set_label_size(self, value: Any) -> None:
        """Set label size button."""
        size_mapping = {"xs": "XS", "s": "S", "m": "M", "l": "L", "xl": "XL"}
        button_text = size_mapping.get(str(value).lower(), str(value).upper())
        selector = (
            f"label:has-text('Label Size') + div button:has-text('{button_text}')"
        )
        self.page.locator(selector).click(timeout=5000)
