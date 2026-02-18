#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import numpy as np
import synnax as sy

from console.case import ConsoleCase
from console.plot import Plot
from framework.utils import assert_link_format, get_random_name


class LinePlot(ConsoleCase):
    """Test line plot operations."""

    suffix: str
    _shared_plot_name: str | None
    ctx_plot_name: str | None
    ctx_plot_link: str

    def setup(self) -> None:
        super().setup()
        self.suffix = get_random_name()
        self._shared_plot_name = None
        self.ctx_plot_name = None

        ctx_plot = self.console.workspace.create_plot(
            f"Context Menu Test {self.suffix}"
        )
        self.ctx_plot_name = ctx_plot.page_name
        self.ctx_plot_link = ctx_plot.copy_link()
        ctx_plot.close()

    def teardown(self) -> None:
        if self._shared_plot_name is not None:
            self._cleanup_pages.append(self._shared_plot_name)
        if self.ctx_plot_name is not None:
            self._cleanup_pages.append(self.ctx_plot_name)
        super().teardown()

    def run(self) -> None:
        """Run all line plot tests."""

        suffix = get_random_name()
        index_name, data_name = self._setup_channels(suffix)

        plot = self.console.workspace.create_plot(f"Line Plot Test {suffix}")
        self._shared_plot_name = plot.page_name
        plot.add_channels("Y1", data_name)

        # General
        self.test_set_line_thickness(plot)
        self.test_set_line_label(plot, suffix)
        self.test_set_plot_title(plot, suffix)
        # Track the renamed plot title
        self._shared_plot_name = plot.page_name
        self.test_move_channel_between_axes(plot, data_name)
        self.test_live_data(plot)
        self.test_drag_channel_to_canvas(plot)
        self.test_drag_channel_to_toolbar(plot)
        self.test_download_csv(plot, data_name)
        self.test_create_range_from_selection(plot, suffix)
        self.test_export_json(plot, data_name)

        plot_link = self.test_copy_link(plot)
        plot_name = plot.page_name
        self._shared_plot_name = plot_name
        plot.close()

        # Resources Toolbar
        self.test_open_plot_from_resources(plot_name, plot_link)
        self.test_drag_plot_onto_mosaic(plot_name, plot_link)

        # Resources Toolbar > Context Menu
        self.test_ctx_operations()
        self.test_ctx_delete_multiple_plots()
        self.test_ctx_delete_plot()

        self.test_open_plot_by_name(plot_name, plot_link)

        # Delete shared plot after all tests that reference it
        self.console.workspace.delete_page(plot_name)
        self._shared_plot_name = None

        self.client.channels.delete([data_name, index_name])

    def _setup_channels(self, suffix: str) -> tuple[str, str]:
        """Create and populate test channels."""
        index_name = f"test_idx_{suffix}"
        data_name = f"test_data_{suffix}"

        index_ch = self.client.channels.create(
            name=index_name,
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name=data_name,
            data_type=sy.DataType.FLOAT32,
            index=index_ch.key,
            retrieve_if_name_exists=True,
        )

        num_samples = 100
        now = sy.TimeStamp.now()
        start_time = now - sy.TimeSpan.SECOND * 30

        timestamps = np.linspace(int(start_time), int(now), num_samples, dtype=np.int64)
        values = np.sin(np.linspace(0, 4 * np.pi, num_samples)).astype(np.float32)

        self.client.write(start_time, {index_name: timestamps, data_name: values})

        return index_name, data_name

    def test_move_channel_between_axes(self, plot: Plot, data_name: str) -> None:
        """Test adding a channel to Y2 axis."""
        self.log("Testing move channel between axes")

        plot.add_channels("Y2", data_name)

        assert data_name in plot.data["Y1"], f"Channel {data_name} should be on Y1"
        assert data_name in plot.data["Y2"], f"Channel {data_name} should be on Y2"

    def test_set_line_thickness(self, plot: Plot) -> None:
        """Test adjusting line thickness via Lines tab."""
        self.log("Testing set line thickness")

        plot.set_line_thickness(5)
        value = plot.get_line_thickness()
        assert value == 5, f"Expected stroke width 5, got {value}"

    def test_set_line_label(self, plot: Plot, suffix: str) -> None:
        """Test relabeling a line via Lines tab."""
        self.log("Testing set line label")

        new_label = f"Custom Label {suffix}"
        plot.set_line_label(new_label)
        labels = plot.get_line_labels()
        assert (
            labels[0] == new_label
        ), f"Expected label '{new_label}', got '{labels[0]}'"

    def test_set_plot_title(self, plot: Plot, suffix: str) -> None:
        """Test setting the plot title via Properties tab."""
        self.log("Testing set plot title")

        new_title = f"Custom Plot Title {suffix}"
        plot.set_title(new_title)
        value = plot.get_title()
        assert value == new_title, f"Expected title '{new_title}', got '{value}'"

    def test_live_data(self, plot: Plot) -> None:
        """Test plotting live data with a rolling time range."""
        self.log("Testing live data")

        live_channel = "line_plot_uptime"
        plot.add_channels("Y1", live_channel)
        assert (
            live_channel in plot.data["Y1"]
        ), f"Channel {live_channel} should be on Y1"

    def test_drag_channel_to_canvas(self, plot: Plot) -> None:
        """Test dragging a channel from sidebar onto the plot canvas."""
        self.log("Testing drag channel to canvas")

        channel = "line_plot_state"
        plot.drag_channel_to_canvas(channel, self.console.channels)
        assert channel in plot.data["Y1"], f"Channel {channel} should be on Y1"

    def test_drag_channel_to_toolbar(self, plot: Plot) -> None:
        """Test dragging a channel from sidebar onto the toolbar data section."""
        self.log("Testing drag channel to toolbar")

        channel = "line_plot_time"
        plot.drag_channel_to_toolbar(channel, self.console.channels, "Y2")
        assert channel in plot.data["Y2"], f"Channel {channel} should be on Y2"

    def test_download_csv(self, plot: Plot, data_name: str) -> None:
        """Test downloading the plot data as a CSV file."""
        self.log("Testing download CSV")

        csv_content = plot.download_csv()

        assert csv_content, "CSV content should not be empty"
        assert data_name in csv_content, f"CSV should contain channel {data_name}"

        lines = csv_content.strip().split("\n")
        assert len(lines) > 1, "CSV should have header and data rows"

    def test_create_range_from_selection(self, plot: Plot, suffix: str) -> None:
        """Test creating a range from a plot selection."""
        self.log("Testing create range from selection")

        range_name = f"Test Range {suffix}"
        plot.create_range_from_selection(range_name)

        created_range = self.client.ranges.retrieve(name=range_name)
        assert (
            created_range.name == range_name
        ), f"Range name mismatch: {created_range.name}"

    def test_copy_link(self, plot: Plot) -> str:
        """Test copying a link to the line plot via toolbar button.

        Returns:
            The full link to the plot.
        """
        self.log("Testing copy link to line plot")

        link = plot.copy_link()

        assert_link_format(link, "lineplot")

        return link

    def test_export_json(self, plot: Plot, data_name: str) -> None:
        """Test exporting the plot as a JSON file via toolbar button."""
        self.log("Testing export plot as JSON")

        exported = plot.export_json()

        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Plot key should be a UUID"
        assert "channels" in exported, "Exported JSON should contain 'channels'"
        assert "y1" in exported["channels"], "Channels should include 'y1' axis"

        data_channel = self.client.channels.retrieve(data_name)
        y1_channels = exported["channels"]["y1"]
        assert (
            data_channel.key in y1_channels
        ), f"Y1 should contain channel key {data_channel.key}"

    def test_open_plot_from_resources(self, plot_name: str, expected_link: str) -> None:
        """Test opening a plot by double-clicking it in the workspace resources toolbar."""
        self.log("Testing open plot from resources toolbar")

        plot = self.console.workspace.open_plot(plot_name)

        assert plot.pane_locator is not None, "Plot pane should be visible"
        assert plot.pane_locator.is_visible(), "Plot pane should be visible"

        opened_link = plot.copy_link()
        # Verify link is the same between resources and visualization toolbars
        assert (
            opened_link == expected_link
        ), f"Opened plot link should match: expected {expected_link}, got {opened_link}"

        plot.close()

    def test_drag_plot_onto_mosaic(self, plot_name: str, expected_link: str) -> None:
        """Test dragging a plot from the resources toolbar onto the mosaic."""
        self.log("Testing drag plot onto mosaic")

        plot = self.console.workspace.drag_plot_to_mosaic(plot_name)

        assert plot.pane_locator is not None, "Plot pane should be visible"
        assert plot.pane_locator.is_visible(), "Plot pane should be visible"

        opened_link = plot.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened plot link should match: expected {expected_link}, got {opened_link}"

        plot.close()

    def test_ctx_operations(self) -> None:
        """Test non-destructive context menu operations using shared plot."""
        assert self.ctx_plot_name is not None

        self.log("Testing copy link via context menu")
        link = self.console.workspace.copy_page_link(self.ctx_plot_name)
        assert (
            link == self.ctx_plot_link
        ), f"Context menu link should match: expected {self.ctx_plot_link}, got {link}"

        self.log("Testing export plot via context menu")
        exported = self.console.workspace.export_page(self.ctx_plot_name)
        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Plot key should be a UUID"
        assert "channels" in exported, "Exported JSON should contain 'channels'"

        self.log("Testing rename plot via context menu")
        new_name = f"Renamed Plot {self.suffix}"
        self.console.workspace.rename_page(self.ctx_plot_name, new_name)
        assert self.console.workspace.page_exists(
            new_name
        ), f"Renamed plot '{new_name}' should exist"
        self.ctx_plot_name = new_name

    def test_ctx_delete_plot(self) -> None:
        """Test deleting a plot via context menu in the workspace resources toolbar."""
        self.log("Testing delete plot via context menu")
        assert self.ctx_plot_name is not None
        self.console.workspace.delete_page(self.ctx_plot_name)
        self.ctx_plot_name = None

    def test_ctx_delete_multiple_plots(self) -> None:
        """Test deleting multiple plots via multi-select and context menu."""
        self.log("Testing delete multiple plots via context menu")

        suffix = get_random_name()
        plot_names = []

        for i in range(3):
            plot = self.console.workspace.create_plot(f"Multi Delete {suffix} {i}")
            plot_names.append(plot.page_name)
            self._cleanup_pages.append(plot.page_name)
            plot.close()

        self.console.workspace.delete_pages(plot_names)
        for name in plot_names:
            self._cleanup_pages.remove(name)

    def test_open_plot_by_name(self, plot_name: str, expected_link: str) -> None:
        """Test opening an existing plot by searching its name in the command palette."""
        self.log("Testing open plot by name via command palette")

        plot = self.console.workspace.open_from_search(Plot, plot_name)

        assert plot.pane_locator is not None, "Plot pane should be visible"
        assert plot.pane_locator.is_visible(), "Plot pane should be visible"

        opened_link = plot.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened plot link should match: expected {expected_link}, got {opened_link}"

        plot.close()
