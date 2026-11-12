#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import re
from typing import Literal

from playwright.sync_api import Locator, expect

import synnax as sy
from console.channels import ChannelClient
from console.layout import LayoutClient
from console.page import ConsolePage, PageType

Axis = Literal["x1", "y1", "y2"]
"""The keys of `sy.lineplot.AxisKey` that the Console toolbar offers."""

# The middle of the range strip's first lane: 4 px of padding, then a 20 px lane.
_FIRST_LANE_CENTER = 14


class Plot(ConsolePage):
    """Plot page management interface"""

    page_type: PageType = "lineplot"
    pluto_label: str = ".pluto-line-plot"

    def __init__(
        self,
        *,
        layout: LayoutClient,
        client: sy.Synnax,
        page_name: str,
        pane_locator: Locator,
    ) -> None:
        """Initialize a Plot page wrapper (see ConsolePage.__init__ for details)."""
        self.data: dict[str, list[str]] = {"y1": [], "y2": [], "x1": [], "Ranges": []}
        super().__init__(
            layout=layout, client=client, page_name=page_name, pane_locator=pane_locator
        )

    def add_channels(self, axis: Axis, channels: str | list[str]) -> None:
        channels = [channels] if isinstance(channels, str) else channels

        data_tab = self.page.get_by_role("tab", name="Data", exact=True)
        data_tab.click(timeout=5000)

        axis_label = self.page.locator("label").filter(has_text=axis.upper())
        axis_label.wait_for(state="visible", timeout=5000)
        trigger = axis_label.locator("..").locator(".lyra-dialog__trigger")
        trigger.click(timeout=5000)

        search_input = self.page.locator("input[placeholder*='Search']")
        for channel in channels:
            search_input.fill(channel)
            self.layout.select_from_dropdown(channel)
            self.data[axis].append(channel)

        self.layout.press_escape()

    def add_ranges(
        self,
        ranges: list[
            Literal[
                "Rolling 30s",
                "Rolling 1min",
                "Rolling 5min",
                "Rolling 15min",
                "Rolling 30min",
            ]
        ],
    ) -> None:
        """Add time ranges to the plot."""
        ranges_label = self.page.locator("label").filter(has_text="Ranges")
        trigger = ranges_label.locator("..").locator(".lyra-dialog__trigger")
        trigger.click(timeout=5000)

        for range_value in ranges:
            if range_value not in self.data["Ranges"]:
                self.layout.select_from_dropdown(range_value)
                self.data["Ranges"].append(range_value)

        self.layout.press_escape()

    def download_csv(self) -> str:
        """Download the plot data as a CSV file.

        The file is saved to the tests/results directory with the plot name.

        :returns: The CSV file contents as a string.
        """
        self.notifications.close_all()
        self.layout.show_visualization_toolbar()
        self.page.get_by_role("button", name="Download as CSV", exact=True).click()

        self.page.get_by_text("Download data for").wait_for(
            state="visible", timeout=5000
        )
        download_button = self.page.get_by_role("button", name="Download").last
        download_button.wait_for(state="visible", timeout=5000)

        with self.page.expect_download(timeout=20000) as download_info:
            download_button.click()

        download = download_info.value
        save_path = str(self.layout.results_dir / f"{self.page_name}.csv")
        download.save_as(save_path)
        with open(save_path, encoding="utf-8") as f:
            return f.read()

    def set_title(self, title: str) -> None:
        """Set the plot title via the Properties tab.

        Note: Setting the title also updates the tab name.

        :param title: The new title for the plot
        """
        self.notifications.close_all()
        self.page.get_by_role("tab", name="Properties", exact=True).click(timeout=5000)

        title_input = (
            self.page.locator("label:has-text('Title')")
            .locator("..")
            .locator("input[role='textbox']")
        )
        title_input.fill(title, timeout=5000)
        self.layout.press_enter()
        self.page_name = title

    def set_line_thickness(self, thickness: int) -> None:
        """Set the stroke width for the first line via the Lines tab.

        :param thickness: Stroke width (1-10)
        """
        self.notifications.close_all()
        self.page.get_by_role("tab", name="Lines", exact=True).click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        if lines_container.count() == 0:
            html = self.page.locator(".console-line-plot-toolbar").inner_html()
            raise RuntimeError(
                f"Lines container not found. Toolbar HTML: {html[:1000]}"
            )

        line_items = lines_container.locator(".lyra-list__item")
        if line_items.count() == 0:
            html = lines_container.inner_html()
            raise RuntimeError(f"No line items. Lines container HTML: {html[:500]}")

        stroke_input = line_items.first.locator("input").nth(1)
        stroke_input.fill(str(thickness), timeout=5000)
        self.layout.press_enter()

    def set_line_label(self, label: str) -> None:
        """Set the label for the first line via the Lines tab.

        :param label: New label for the line
        """
        self.notifications.close_all()
        self.page.get_by_role("tab", name="Lines", exact=True).click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        lines_container.wait_for(state="visible", timeout=5000)

        line_item = lines_container.locator(".lyra-list__item").first
        line_item.wait_for(state="visible", timeout=5000)

        label_input = line_item.locator("input").first
        label_input.fill(label, timeout=5000)
        self.layout.press_enter()

    def get_line_thickness(self) -> int:
        """Get the stroke width for the first line from the Lines tab.

        :returns: The current stroke width
        """
        self.notifications.close_all()
        self.page.get_by_role("tab", name="Lines", exact=True).click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        line_item = lines_container.locator(".lyra-list__item").first
        stroke_input = line_item.locator("input").nth(1)
        return int(stroke_input.input_value())

    def get_line_labels(self) -> list[str]:
        """Get labels for all lines from the Lines tab.

        :returns: List of line label strings.
        """
        self.notifications.close_all()
        self.layout.show_visualization_toolbar()
        self.page.get_by_role("tab", name="Lines", exact=True).click(timeout=5000)

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        line_items = lines_container.locator(".lyra-list__item")
        count = line_items.count()
        return [
            line_items.nth(i).locator("input").first.input_value() for i in range(count)
        ]

    def drag_channel_to_canvas(
        self, channel_name: str, *, channels: ChannelClient
    ) -> None:
        """Drag a channel from the sidebar onto the plot canvas.

        :param channel_name: Name of the channel to drag
        :param channels: ChannelClient for showing/hiding channels sidebar
        """
        channels.show_channels()

        # The channel tree is windowed, so the row must be scrolled into the
        # DOM before it can be dragged.
        channel_item = channels.tree.wait_for_name("channel:", channel_name)

        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")

        self.layout.drag(channel_item, self.pane_locator)
        channels.hide_channels()

        self.data["y1"].append(channel_name)

    def drag_channel_to_toolbar(
        self,
        channel_name: str,
        *,
        channels: ChannelClient,
        axis: Axis = "y1",
    ) -> None:
        """Drag a channel from the sidebar onto the toolbar data section.

        :param channel_name: Name of the channel to drag
        :param channels: ChannelClient for showing/hiding channels sidebar
        :param axis: Target axis
        """
        self.notifications.close_all()
        channels.show_channels()

        # The channel tree is windowed, so the row must be scrolled into the
        # DOM before it can be dragged.
        channel_item = channels.tree.wait_for_name("channel:", channel_name)

        self.layout.show_visualization_toolbar()
        data_tab = self.page.get_by_role("tab", name="Data", exact=True)
        data_tab.click(timeout=5000)

        # The axis row lays out as cells of the tab's grid and has no box of its own.
        axis_field = self.page.locator(f"label:has-text('{axis.upper()}') + *")
        axis_field.wait_for(state="visible", timeout=5000)

        self.layout.drag(channel_item, axis_field)
        channels.hide_channels()

        self.data[axis].append(channel_name)

    def create_range_from_selection(self, range_name: str) -> None:
        """Create a range by selecting a region on the plot and using the context menu.

        :param range_name: Name for the new range
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
        self.page.get_by_role("button", name="Save to Core").click()

    def get_annotation_toggle(self) -> Locator:
        """Return the range-annotation visibility toggle button.

        The button only exists in the DOM when at least one range intersects
        the current plot viewport.
        """
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")
        return self.pane_locator.locator(
            ".console-controls button:has(.lyra-icon--range)"
        )

    def wait_for_annotation_toggle(self, *, timeout: int = 5000) -> None:
        """Wait for the annotation visibility toggle to appear."""
        self.get_annotation_toggle().wait_for(state="visible", timeout=timeout)

    def toggle_annotation_visibility(self) -> None:
        """Click the annotation visibility toggle."""
        self.get_annotation_toggle().click(timeout=5000)

    def expect_annotations_visible(
        self, *, visible: bool = True, timeout: int = 5000
    ) -> None:
        """Assert the annotation visibility toggle reflects the given state.

        Pluto's Button.Toggle carries lyra--selected when on.
        """
        pattern = re.compile(r"lyra--selected")
        toggle = expect(self.get_annotation_toggle())
        if visible:
            toggle.to_have_class(pattern, timeout=timeout)
        else:
            toggle.not_to_have_class(pattern, timeout=timeout)

    def get_range_strip(self) -> Locator:
        """Return the strip above the plot that shows the plot's ranges."""
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")
        return self.pane_locator.locator(".pluto-lineplot-range-provider")

    def click_range(self, *, button: Literal["left", "right"] = "left") -> None:
        """Click the range in the strip's first lane, at the middle of the plot.

        Waits for the strip to report the range under the cursor before the click,
        since the plot finds that range on its next render.
        """
        strip = self.get_range_strip()
        box = strip.bounding_box()
        if box is None:
            raise RuntimeError("Range strip not visible")
        x = box["x"] + box["width"] / 2
        y = box["y"] + _FIRST_LANE_CENTER
        self.page.mouse.move(x, y)
        expect(strip).to_have_class(re.compile(r"pluto--hovered"), timeout=5000)
        self.page.mouse.click(x, y, button=button)

    def get_pause_toggle(self) -> Locator:
        """Return the button that pauses and resumes live plotting."""
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")
        return self.pane_locator.locator(
            ".console-controls button:has(.lyra-icon--pause), "
            ".console-controls button:has(.lyra-icon--play)"
        )

    def expect_paused(self, *, paused: bool = True, timeout: int = 5000) -> None:
        """Assert the plot is paused, or plotting live when paused is False."""
        pattern = re.compile(r"lyra--selected")
        toggle = expect(self.get_pause_toggle())
        if paused:
            toggle.to_have_class(pattern, timeout=timeout)
        else:
            toggle.not_to_have_class(pattern, timeout=timeout)

    def has_channel(self, axis: Axis, channel_name: str) -> bool:
        """Check if a channel is shown on the specified axis in the toolbar."""
        self.layout.get_tab(self.page_name).click()
        self.layout.show_visualization_toolbar()
        self.page.get_by_role("tab", name="Data", exact=True).click(timeout=5000)
        axis_section = (
            self.page.locator("label").filter(has_text=axis.upper()).locator("..")
        )
        return axis_section.get_by_text(channel_name).count() > 0

    def focus(self) -> None:
        """Bring focus onto the plot pane so keyboard triggers fire on it."""
        if self.pane_locator is not None:
            self.pane_locator.click()

    def undo(self) -> None:
        """Cmd/Ctrl + Z — pop the last entry off the undo stack."""
        self.focus()
        self.layout.press_key("ControlOrMeta+z")
        self.page.wait_for_timeout(300)

    def redo(self) -> None:
        """Cmd/Ctrl + Shift + Z — re-apply the most recently undone entry."""
        self.focus()
        self.layout.press_key("ControlOrMeta+Shift+z")
        self.page.wait_for_timeout(300)
