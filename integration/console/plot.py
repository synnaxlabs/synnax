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
from console.notifications import click_past
from console.page import ConsolePage, PageType
from console.range.toolbar import Toolbar as RangeToolbar

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

    def add_ranges(self, ranges: list[str]) -> None:
        """Add ranges to the plot from its range picker.

        :param ranges: The names of saved ranges, or rolling windows such as
            "Rolling 30s".
        """
        self._range_picker().click(timeout=5000)
        for range_value in ranges:
            if range_value not in self.data["Ranges"]:
                self.layout.select_from_dropdown(range_value)
                self.data["Ranges"].append(range_value)

        self.layout.press_escape()

    def remove_range(self, range_name: str) -> None:
        """Remove a range from the plot with the remove button of its tag.

        :param range_name: The name of the range.
        """
        remove_btn = self.get_range_tag(range_name).get_by_role(
            "button", name=f"Remove {range_name}"
        )
        # The button takes the pointer only while its tag is hovered.
        remove_btn.dispatch_event("click")
        remove_btn.wait_for(state="detached", timeout=5000)
        # A new plot shows a rolling window that this wrapper did not add.
        if range_name in self.data["Ranges"]:
            self.data["Ranges"].remove(range_name)

    def download_csv(self) -> str:
        """Download the plot data as a CSV file.

        The file is saved to the tests/results directory with the plot name.

        :returns: The CSV file contents as a string.
        """
        self.layout.show_visualization_toolbar()
        click_past(self.page.get_by_role("button", name="Download as CSV", exact=True))

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
        click_past(self.page.get_by_role("tab", name="Properties", exact=True))

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
        click_past(self.page.get_by_role("tab", name="Lines", exact=True))

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
        click_past(self.page.get_by_role("tab", name="Lines", exact=True))

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
        click_past(self.page.get_by_role("tab", name="Lines", exact=True))

        lines_container = self.page.locator(".console-line-plot__toolbar-lines")
        line_item = lines_container.locator(".lyra-list__item").first
        stroke_input = line_item.locator("input").nth(1)
        return int(stroke_input.input_value())

    def get_line_labels(self) -> list[str]:
        """Get labels for all lines from the Lines tab.

        :returns: List of line label strings.
        """
        self.layout.show_visualization_toolbar()
        click_past(self.page.get_by_role("tab", name="Lines", exact=True))

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

    def _range_picker(self) -> Locator:
        """The range picker in the Data tab of the visualization toolbar."""
        self.layout.show_visualization_toolbar()
        self.page.get_by_role("tab", name="Data", exact=True).click(timeout=5000)
        label = self.page.locator("label").filter(has_text="Ranges")
        return label.locator("..").locator(".lyra-dialog__trigger")

    def drag_range_to_toolbar(self, range_name: str, *, ranges: RangeToolbar) -> None:
        """Drag a range from the ranges toolbar onto the plot's range picker.

        :param range_name: The name of a favorite range.
        :param ranges: The ranges toolbar to drag the range from.
        """
        self.notifications.close_all()
        picker = self._range_picker()
        ranges.show()
        ranges.expand("Favorites")
        line = ranges.get_item(range_name, group="Favorites")
        line.wait_for(state="visible", timeout=5000)
        self.layout.drag(line, picker)
        self.data["Ranges"].append(range_name)

    def get_range_tag(self, range_name: str) -> Locator:
        """The tag of a range in the plot's range picker.

        :param range_name: The name of the range.
        """
        return (
            self._range_picker()
            .locator(".lyra-tag")
            .filter(has_text=re.compile(f"^{re.escape(range_name)}$"))
        )

    def select_region(self) -> dict[str, float]:
        """Select the middle half of the plot with an Alt drag.

        :returns: The center of the selection, in page pixels.
        """
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")
        box = self.pane_locator.bounding_box()
        if not box:
            raise RuntimeError("Could not get plot bounding box")
        start_x = box["x"] + box["width"] * 0.25
        end_x = box["x"] + box["width"] * 0.75
        center_y = box["y"] + box["height"] / 2
        self.page.keyboard.down("Alt")
        self.page.mouse.move(start_x, center_y)
        self.page.mouse.down()
        self.page.mouse.move(end_x, center_y, steps=10)
        self.page.mouse.up()
        self.page.keyboard.up("Alt")
        return {"x": (start_x + end_x) / 2, "y": center_y}

    def create_range_from_selection(self, range_name: str) -> None:
        """Create a range by selecting a region on the plot and using the context menu.

        :param range_name: Name for the new range
        """
        center = self.select_region()
        self.page.mouse.click(center["x"], center["y"], button="right")

        self.page.get_by_text("Create range from selection").click(timeout=5000)
        self.page.get_by_role("textbox", name="Name").wait_for(
            state="visible", timeout=5000
        )
        self.page.get_by_role("textbox", name="Name").fill(range_name)
        self.page.get_by_role("button", name="Create", exact=True).click()

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

    def get_breaks_toggle(self) -> Locator:
        """Return the button that hides and shows the breaks between ranges.

        The button only exists while the plot has a break, shown or hidden.
        """
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")
        return self.pane_locator.locator(
            ".console-controls button:has(.lyra-icon--split-x)"
        )

    def toggle_breaks(self) -> None:
        """Click the button that hides and shows the breaks."""
        self.get_breaks_toggle().click(timeout=5000)

    def expect_breaks_shown(self, *, shown: bool = True, timeout: int = 5000) -> None:
        """Assert the breaks toggle shows, and that it is on when shown is True."""
        pattern = re.compile(r"lyra--selected")
        toggle = expect(self.get_breaks_toggle())
        if shown:
            toggle.to_have_class(pattern, timeout=timeout)
        else:
            toggle.not_to_have_class(pattern, timeout=timeout)

    def expect_no_breaks_toggle(self, *, timeout: int = 5000) -> None:
        """Assert the plot shows no breaks toggle."""
        expect(self.get_breaks_toggle()).to_have_count(0, timeout=timeout)

    def _open_menu(self) -> None:
        """Open the plot menu over the left quarter of the plot, clear of the seam
        between two ranges of equal length."""
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")
        box = self.pane_locator.bounding_box()
        if box is None:
            raise RuntimeError("Could not get plot bounding box")
        self.layout.ctx_menu.open_on(
            self.pane_locator,
            position={"x": box["width"] * 0.25, "y": box["height"] * 0.6},
        )

    def expect_menu_option(
        self, option: str, *, offered: bool = True, timeout: int = 5000
    ) -> None:
        """Open the plot menu and assert whether it offers an option."""
        self._open_menu()
        menu = self.page.get_by_role("menu").first
        items = menu.get_by_role("menuitem").filter(has_text=option)
        expect(items).to_have_count(1 if offered else 0, timeout=timeout)
        self.layout.ctx_menu.close()

    def select_menu_option(self, option: str) -> None:
        """Open the plot menu and click an option."""
        self._open_menu()
        self.layout.ctx_menu.click_option(option)

    def reload_console(self) -> None:
        """Reload the Console from the plot menu and wait for the plot to show."""
        with self.page.expect_navigation(wait_until="load", timeout=30000):
            self.select_menu_option("Reload Console")
        if not self.pane_locator:
            raise RuntimeError("Plot pane locator not available")
        self.pane_locator.wait_for(state="visible", timeout=10000)

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
