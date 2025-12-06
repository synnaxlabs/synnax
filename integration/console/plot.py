#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from typing import Any, Literal

import synnax as sy

from .console import Console
from .context_menu import ContextMenu
from .page import ConsolePage

Axis = Literal["Y1", "Y2", "X1"]


class Plot(ConsolePage):
    """Plot page management interface"""

    page_type: str = "Line Plot"
    pluto_label: str = ".pluto-line-plot"

    def __init__(
        self,
        client: sy.Synnax,
        console: Console,
        page_name: str,
    ) -> None:
        """
        Initialize a Plot page.

        Args:
            client: Synnax client instance
            console: Console instance
            page_name: Name for the page
        """
        self.data: dict[str, Any] = {
            "Y1": [],
            "Y2": [],
            "Ranges": [],
            "X1": None,
        }
        super().__init__(client, console, page_name)

    def add_channels(self, axis: Axis, channels: str | list[str]) -> None:
        channels = [channels] if isinstance(channels, str) else channels

        selector = self.page.get_by_text(f"{axis} Select channels")

        selector.click(timeout=5000)

        search_input = self.page.locator("input[placeholder*='Search']")
        for channel in channels:
            search_input.fill(channel)
            self.console.select_from_dropdown(channel)
            self.data[axis].append(channel)

        self.console.ESCAPE

    def add_ranges(
        self, ranges: list[Literal["30s", "1m", "5m", "15m", "30m"]]
    ) -> None:
        """Add time ranges to the plot."""
        self.page.get_by_text("Select ranges").click()

        for range_value in ranges:
            if range_value not in self.data["Ranges"]:
                self.page.get_by_text(range_value, exact=True).click()
                self.data["Ranges"].append(range_value)

        self.console.ESCAPE

    def download_csv(self) -> str:
        """Download the plot as a CSV file."""
        self.console.close_all_notifications()
        csv_button = self.page.locator(".pluto-icon--csv").locator("..")
        with self.page.expect_download() as download_info:
            csv_button.click()
        download = download_info.value
        with open(download.path(), "r") as file:
            return file.read()

    def set_axis(self, axis: Axis, config: dict[str, Any]) -> None:
        """Set axis configuration with the given parameters."""
        self.console.close_all_notifications()
        self.page.get_by_text("Axes").click(timeout=5000)
        self.page.wait_for_selector(".pluto-tabs-selector__btn", timeout=5000)

        self._select_axis_tab(axis)

        for key, value in config.items():
            self._set_axis_property(key, value)

        self.console.ENTER

    def _select_axis_tab(self, axis: Axis) -> None:
        """Select the axis tab in the configuration panel."""
        selectors = [
            f"#{axis.lower()}",
            f"#{axis}",
            f".pluto-tabs-selector__btn:has-text('{axis}')",
        ]

        for selector in selectors:
            locator = self.page.locator(selector)
            if locator.count() > 0:
                locator.click(timeout=5000)
                return

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

    def _set_label_direction(
        self, direction: Literal["horizontal", "vertical"]
    ) -> None:
        """Set label direction button."""

        icon_direction: Literal["arrow-up", "arrow-right"] = (
            "arrow-up" if direction == "vertical" else "arrow-right"
        )
        selector = f"label:has-text('Label Direction') + div button:has([aria-label='pluto-icon--{icon_direction}'])"
        self.page.locator(selector).click(timeout=5000)

    def _set_label_size(self, size: Literal["xs", "s", "m", "l", "xl"]) -> None:
        """Set label size button."""

        selector = f"label:has-text('Label Size') + div button:has-text('{size}')"
        self.page.locator(selector).click(timeout=5000)

    # -------------------------------------------------------------------------
    # Tab-based operations
    # -------------------------------------------------------------------------

    def _get_tab(self):
        """Get the tab locator for this plot."""
        return self.page.locator("div").filter(
            has_text=re.compile(f"^{re.escape(self.page_name)}$")
        ).first

    def rename(self, new_name: str) -> None:
        """Rename the plot by double-clicking the tab name.

        Args:
            new_name: The new name for the plot
        """
        tab = self._get_tab()
        tab.dblclick()
        self.page.wait_for_timeout(100)

        # Find and fill the editable text
        editable = self.page.get_by_text(self.page_name).first
        editable.fill(new_name)
        self.page.keyboard.press("Enter")
        self.page.wait_for_timeout(200)

        # Update the stored page name
        self.page_name = new_name

    def copy_link(self) -> str:
        """Copy link to the plot via tab context menu.

        Returns:
            The copied link from clipboard (empty string if clipboard access fails)
        """
        tab = self._get_tab()
        menu = ContextMenu(self.page)
        menu.open_on(tab)
        menu.click_option("Copy Link")
        self.page.wait_for_timeout(200)

        # Try to get the link from clipboard
        try:
            link = self.page.evaluate("navigator.clipboard.readText()")
            return link
        except Exception:
            # If clipboard access fails, return empty string
            return ""
