#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import json
from typing import Any, Literal

import synnax as sy

from framework.utils import get_results_path

from .console import Console
from .page import ConsolePage

Axis = Literal["Y1", "Y2", "X1"]


class Plot(ConsolePage):
    """Plot page management interface"""

    page_type: str = "Line Plot"
    pluto_label: str = ".pluto-line-plot"

    @classmethod
    def open_from_search(cls, client: sy.Synnax, console: Console, name: str) -> Plot:
        """Open an existing plot by searching its name in the command palette.

        Args:
            client: Synnax client instance.
            console: Console instance.
            name: Name of the plot to search for and open.

        Returns:
            Plot instance for the opened plot.
        """
        console.search_palette(name)

        plot_pane = console.page.locator(cls.pluto_label)
        plot_pane.first.wait_for(state="visible", timeout=5000)

        tabs = console.page.locator(".pluto-tabs-selector div").filter(
            has=console.page.locator("[aria-label='pluto-tabs__close']")
        )
        tab_count = tabs.count()
        actual_tab_name = "Line Plot"
        if tab_count > 0:
            last_tab = tabs.nth(tab_count - 1)
            actual_tab_name = last_tab.inner_text().strip()

        plot = cls.__new__(cls)
        plot.client = client
        plot.console = console
        plot.page = console.page
        plot.page_name = actual_tab_name
        plot.pane_locator = plot_pane.first
        plot.data = {"Y1": [], "Y2": [], "Ranges": [], "X1": None}
        return plot

    @classmethod
    def from_open_page(cls, client: sy.Synnax, console: Console, name: str) -> "Plot":
        """Create Plot instance from an already-opened page.

        Overrides parent to initialize the data tracking attribute.
        """
        plot = super().from_open_page(client, console, name)
        plot.data = {"Y1": [], "Y2": [], "Ranges": [], "X1": None}
        return plot

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

        self.page.locator("#data").click(timeout=5000)
        self.page.wait_for_timeout(300)

        axis_label = self.page.locator("label").filter(has_text=axis)
        trigger = axis_label.locator("..").locator(".pluto-dialog__trigger")
        trigger.click(timeout=5000)

        search_input = self.page.locator("input[placeholder*='Search']")
        for channel in channels:
            search_input.fill(channel)
            self.console.select_from_dropdown(channel)
            self.data[axis].append(channel)

        self.console.ESCAPE

    def add_ranges(
        self,
        ranges: list[
            Literal[
                "Rolling 30s", "Rolling 1m", "Rolling 5m", "Rolling 15m", "Rolling 30m"
            ]
        ],
    ) -> None:
        """Add time ranges to the plot."""
        ranges_label = self.page.locator("label").filter(has_text="Ranges")
        trigger = ranges_label.locator("..").locator(".pluto-dialog__trigger")
        trigger.click(timeout=5000)

        for range_value in ranges:
            if range_value not in self.data["Ranges"]:
                self.console.select_from_dropdown(range_value)
                self.data["Ranges"].append(range_value)

        self.console.ESCAPE

    def download_csv(self) -> str:
        """Download the plot data as a CSV file.

        The file is saved to the tests/results directory with the plot name.

        Returns:
            The CSV file contents as a string.
        """
        self.console.notifications.close_all()
        csv_button = self.page.locator(".pluto-icon--csv").locator("..")
        csv_button.click()

        self.page.get_by_text("Download data for").wait_for(
            state="visible", timeout=5000
        )
        download_button = self.page.get_by_role("button", name="Download").last
        download_button.wait_for(state="visible", timeout=5000)

        self.page.evaluate("delete window.showSaveFilePicker")
        self.page.wait_for_function(
            "() => window.showSaveFilePicker === undefined", timeout=5000
        )

        with self.page.expect_download(timeout=20000) as download_info:
            download_button.click()

        download = download_info.value
        save_path = get_results_path(f"{self.page_name}.csv")
        download.save_as(save_path)
        with open(save_path, "r") as f:
            return f.read()

    def set_axis(self, axis: Axis, config: dict[str, Any]) -> None:
        """Set axis configuration with the given parameters."""
        self.console.notifications.close_all()
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

    def copy_link(self) -> str:
        """Copy link to the plot via the toolbar link button.

        Returns:
            The copied link from clipboard (empty string if clipboard access fails)
        """
        self.console.layout.show_visualization_toolbar()
        link_button = self.page.locator(".pluto-icon--link").locator("..")
        link_button.click(timeout=5000)

        try:
            link: str = str(self.page.evaluate("navigator.clipboard.readText()"))
            return link
        except Exception:
            return ""

    def export_json(self) -> dict[str, Any]:
        """Export the plot as a JSON file via the toolbar export button.

        The file is saved to the tests/results directory with the plot name.

        Returns:
            The exported JSON content as a dictionary.
        """
        self.console.layout.show_visualization_toolbar()
        export_button = self.page.locator(".pluto-icon--export").locator("..")
        self.page.evaluate("delete window.showSaveFilePicker")

        with self.page.expect_download(timeout=5000) as download_info:
            export_button.click()

        download = download_info.value
        save_path = get_results_path(f"{self.page_name}.json")
        download.save_as(save_path)
        with open(save_path, "r") as f:
            result: dict[str, Any] = json.load(f)
            return result

    def set_title(self, title: str) -> None:
        """Set the plot title via the Properties tab.

        Note: Setting the title also updates the tab name.

        Args:
            title: The new title for the plot
        """
        self.console.notifications.close_all()
        self.page.locator("#properties").click(timeout=5000)

        title_input = (
            self.page.locator("label:has-text('Title')")
            .locator("..")
            .locator("input[role='textbox']")
        )
        title_input.fill(title, timeout=5000)
        self.console.ENTER
        self.page_name = title

    def set_line_thickness(self, thickness: int) -> None:
        """Set the stroke width for the first line via the Lines tab.

        Args:
            thickness: Stroke width (1-10)
        """
        self.console.notifications.close_all()
        self.page.locator("#lines").click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        if lines_container.count() == 0:
            html = self.page.locator(".console-line-plot-toolbar").inner_html()
            raise RuntimeError(
                f"Lines container not found. Toolbar HTML: {html[:1000]}"
            )

        line_items = lines_container.locator(".pluto-list__item")
        if line_items.count() == 0:
            html = lines_container.inner_html()
            raise RuntimeError(f"No line items. Lines container HTML: {html[:500]}")

        stroke_input = line_items.first.locator("input").nth(1)
        stroke_input.fill(str(thickness), timeout=5000)
        self.console.ENTER

    def set_line_label(self, label: str) -> None:
        """Set the label for the first line via the Lines tab.

        Args:
            label: New label for the line
        """
        self.console.notifications.close_all()
        self.page.locator("#lines").click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        lines_container.wait_for(state="visible", timeout=5000)

        line_item = lines_container.locator(".pluto-list__item").first
        line_item.wait_for(state="visible", timeout=5000)

        label_input = line_item.locator("input").first
        label_input.fill(label, timeout=5000)
        self.console.ENTER

    def get_line_thickness(self) -> int:
        """Get the stroke width for the first line from the Lines tab.

        Returns:
            The current stroke width
        """
        self.console.notifications.close_all()
        self.page.locator("#lines").click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        line_item = lines_container.locator(".pluto-list__item").first
        stroke_input = line_item.locator("input").nth(1)
        return int(stroke_input.input_value())

    def get_line_label(self) -> str:
        """Get the label for the first line from the Lines tab.

        Returns:
            The current line label
        """
        self.console.notifications.close_all()
        self.page.locator("#lines").click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        line_item = lines_container.locator(".pluto-list__item").first
        label_input = line_item.locator("input").first
        return label_input.input_value()

    def drag_channel_to_canvas(self, channel_name: str) -> None:
        """Drag a channel from the sidebar onto the plot canvas.

        Args:
            channel_name: Name of the channel to drag
        """
        self.console.channels.show_channels()

        channel_item = (
            self.page.locator("div[id^='channel:']").filter(has_text=channel_name).first
        )
        channel_item.wait_for(state="visible", timeout=5000)

        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")

        channel_item.drag_to(self.pane_locator)
        self.console.channels.hide_channels()

        self.data["Y1"].append(channel_name)

    def drag_channel_to_toolbar(self, channel_name: str, axis: Axis = "Y1") -> None:
        """Drag a channel from the sidebar onto the toolbar data section.

        Args:
            channel_name: Name of the channel to drag
            axis: Target axis (Y1, Y2, or X1)
        """
        self.console.channels.show_channels()

        channel_item = (
            self.page.locator("div[id^='channel:']").filter(has_text=channel_name).first
        )
        channel_item.wait_for(state="visible", timeout=5000)

        self.page.locator("#data").click(timeout=5000)

        axis_section = self.page.locator(f"label:has-text('{axis}')").locator("..")
        axis_section.wait_for(state="visible", timeout=5000)

        channel_item.drag_to(axis_section)
        self.console.channels.hide_channels()

        self.data[axis].append(channel_name)

    def create_range_from_selection(self, range_name: str) -> None:
        """Create a range by selecting a region on the plot and using the context menu.

        Args:
            range_name: Name for the new range
        """
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")

        box = self.pane_locator.bounding_box()
        if not box:
            raise RuntimeError("Could not get plot bounding box")

        start_x = box["x"] + box["width"] * 0.25
        end_x = box["x"] + box["width"] * 0.75
        center_y = box["y"] + box["height"] / 2

        # Alt+drag to create selection
        self.page.keyboard.down("Alt")
        self.page.mouse.move(start_x, center_y)
        self.page.mouse.down()
        self.page.mouse.move(end_x, center_y, steps=10)
        self.page.mouse.up()
        self.page.keyboard.up("Alt")
        self.page.mouse.click((start_x + end_x) / 2, center_y, button="right")

        self.page.get_by_text("Create range from selection").click(timeout=5000)
        self.page.get_by_role("textbox", name="Name").wait_for(
            state="visible", timeout=5000
        )
        self.page.get_by_role("textbox", name="Name").fill(range_name)
        self.page.get_by_role("button", name="Save to Synnax").click()

    def has_channel(self, axis: Axis, channel_name: str) -> bool:
        """Check if a channel is shown on the specified axis in the toolbar."""
        self.console.layout.get_tab(self.page_name).click()
        self.console.layout.show_visualization_toolbar()
        self.page.locator("#data").click(timeout=5000)
        axis_section = self.page.locator("label").filter(has_text=axis).locator("..")
        result = axis_section.get_by_text(channel_name).count() > 0
        return result
