#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, Literal

import synnax as sy

from framework.utils import get_results_path

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
        self.page.evaluate("delete window.showSaveFilePicker")

        with self.page.expect_download(timeout=5000) as download_info:
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
            link: str = str(self.page.evaluate("navigator.clipboard.readText()"))
            return link
        except Exception:
            # If clipboard access fails, return empty string
            return ""

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

    @classmethod
    def open_from_channel(
        cls, client: sy.Synnax, console: Console, channel_name: str
    ) -> "Plot":
        """Open a channel's plot by double-clicking it in the sidebar.

        Args:
            client: Synnax client instance
            console: Console instance
            channel_name: The name of the channel to open

        Returns:
            Plot instance for the opened plot
        """
        console.channels.show_channels()
        item = console.channels._find_channel_item(channel_name)
        if item is None:
            raise ValueError(f"Channel {channel_name} not found")
        item.dblclick()

        line_plot = console.page.locator(cls.pluto_label)
        line_plot.first.wait_for(state="visible", timeout=5000)

        console.channels.hide_channels()

        console.layout.show_visualization_toolbar()
        page_name = console.layout.get_visualization_toolbar_title()

        plot = cls.__new__(cls)
        plot.client = client
        plot.console = console
        plot.page = console.page
        plot.page_name = page_name
        plot.data = {"Y1": [channel_name], "Y2": [], "Ranges": [], "X1": None}
        plot.pane_locator = line_plot.first
        return plot

    @classmethod
    def open_from_search(
        cls, client: sy.Synnax, console: Console, channel_name: str
    ) -> "Plot":
        """Open a channel plot by searching its name in the command palette.

        Args:
            client: Synnax client instance
            console: Console instance
            channel_name: The name of the channel to search for and open

        Returns:
            Plot instance for the opened plot
        """
        console.search_palette(channel_name)

        line_plot = console.page.locator(cls.pluto_label)
        line_plot.first.wait_for(state="visible", timeout=5000)

        console.layout.show_visualization_toolbar()
        page_name = console.layout.get_visualization_toolbar_title()

        plot = cls.__new__(cls)
        plot.client = client
        plot.console = console
        plot.page = console.page
        plot.page_name = page_name
        plot.data = {"Y1": [channel_name], "Y2": [], "Ranges": [], "X1": None}
        plot.pane_locator = line_plot.first
        return plot
