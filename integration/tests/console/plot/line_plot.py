#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import uuid

import numpy as np
import synnax as sy

from console.case import ConsoleCase
from console.plot import Plot


class LinePlot(ConsoleCase):
    """Test line plot operations."""

    def run(self) -> None:
        """Run all line plot tests."""
        prefix = str(uuid.uuid4())[:6]
        index_name, data_name = self._setup_channels(prefix)

        plot = Plot(self.client, self.console, f"Line Plot Test {prefix}")
        plot.add_channels("Y1", data_name)

        # General
        self.test_set_line_thickness(plot)
        self.test_set_line_label(plot, prefix)
        self.test_set_plot_title(plot, prefix)
        self.test_move_channel_between_axes(plot, data_name)
        self.test_live_data(plot)
        self.test_drag_channel_to_canvas(plot)
        self.test_drag_channel_to_toolbar(plot)
        self.test_download_csv(plot, data_name)
        self.test_create_range_from_selection(plot, prefix)
        self.test_export_json(plot, data_name)

        plot_link = self.test_copy_link(plot)
        plot_name = plot.page_name
        plot.close()

        # Resources Toolbar
        self.test_open_plot_from_resources(plot_name, plot_link)
        self.test_drag_plot_onto_mosaic(plot_name, plot_link)

        # Resources Toolbar > Context Menu
        self.test_ctx_rename_plot()
        self.test_ctx_delete_plot()
        self.test_ctx_delete_multiple_plots()
        self.test_ctx_export_json()
        self.test_ctx_copy_link()

        self.test_open_plot_by_name(plot_name, plot_link)

        self.client.channels.delete([data_name, index_name])

    def _setup_channels(self, prefix: str) -> tuple[str, str]:
        """Create and populate test channels."""
        index_name = f"test_idx_{prefix}"
        data_name = f"test_data_{prefix}"

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

    def test_set_line_label(self, plot: Plot, prefix: str) -> None:
        """Test relabeling a line via Lines tab."""
        self.log("Testing set line label")

        new_label = f"Custom Label {prefix}"
        plot.set_line_label(new_label)
        value = plot.get_line_label()
        assert value == new_label, f"Expected label '{new_label}', got '{value}'"

    def test_set_plot_title(self, plot: Plot, prefix: str) -> None:
        """Test setting the plot title via Properties tab."""
        self.log("Testing set plot title")

        new_title = f"Custom Plot Title {prefix}"
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
        plot.drag_channel_to_canvas(channel)
        assert channel in plot.data["Y1"], f"Channel {channel} should be on Y1"

    def test_drag_channel_to_toolbar(self, plot: Plot) -> None:
        """Test dragging a channel from sidebar onto the toolbar data section."""
        self.log("Testing drag channel to toolbar")

        channel = "line_plot_time"
        plot.drag_channel_to_toolbar(channel, "Y2")
        assert channel in plot.data["Y2"], f"Channel {channel} should be on Y2"

    def test_download_csv(self, plot: Plot, data_name: str) -> None:
        """Test downloading the plot data as a CSV file."""
        self.log("Testing download CSV")

        csv_content = plot.download_csv()

        assert csv_content, "CSV content should not be empty"
        assert data_name in csv_content, f"CSV should contain channel {data_name}"

        lines = csv_content.strip().split("\n")
        assert len(lines) > 1, "CSV should have header and data rows"

    def test_create_range_from_selection(self, plot: Plot, prefix: str) -> None:
        """Test creating a range from a plot selection."""
        self.log("Testing create range from selection")

        range_name = f"Test Range {prefix}"
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

        assert link.startswith(
            "synnax://"
        ), f"Link should start with synnax://, got: {link}"
        parts = link.replace("synnax://", "").split("/")
        assert len(parts) == 4, f"Link should have 4 path parts, got: {parts}"
        assert parts[0] == "cluster", f"First part should be 'cluster', got: {parts[0]}"
        assert (
            len(parts[1]) == 36
        ), f"Cluster ID should be 36 chars (UUID), got: {parts[1]}"
        assert (
            parts[2] == "lineplot"
        ), f"Third part should be 'lineplot', got: {parts[2]}"
        assert (
            len(parts[3]) == 36
        ), f"Plot ID should be 36 chars (UUID), got: {parts[3]}"

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

        plot = Plot.open_from_toolbar(self.client, self.console, plot_name)

        assert plot.pane_locator is not None, "Plot pane should be visible"
        assert plot.pane_locator.is_visible(), "Plot pane should be visible"

        opened_link = plot.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened plot link should match: expected {expected_link}, got {opened_link}"

        plot.close()

    def test_drag_plot_onto_mosaic(self, plot_name: str, expected_link: str) -> None:
        """Test dragging a plot from the resources toolbar onto the mosaic."""
        self.log("Testing drag plot onto mosaic")

        plot = Plot.open_from_drag(self.client, self.console, plot_name)

        assert plot.pane_locator is not None, "Plot pane should be visible"
        assert plot.pane_locator.is_visible(), "Plot pane should be visible"

        opened_link = plot.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened plot link should match: expected {expected_link}, got {opened_link}"

        plot.close()

    def test_ctx_rename_plot(self) -> None:
        """Test renaming a plot via context menu in the workspace resources toolbar."""
        self.log("Testing rename plot via context menu")

        prefix = str(uuid.uuid4())[:6]
        plot = Plot(self.client, self.console, f"Rename Test {prefix}")
        original_name = plot.page_name
        plot.close()

        new_name = f"Renamed Plot {prefix}"
        self.console.workspace.rename_page(original_name, new_name)

        assert self.console.workspace.page_exists(
            new_name
        ), f"Renamed plot '{new_name}' should exist"
        assert not self.console.workspace.page_exists(
            original_name, timeout=1000
        ), f"Original plot '{original_name}' should not exist"

        self.console.workspace.delete_page(new_name)

    def test_ctx_delete_plot(self) -> None:
        """Test deleting a plot via context menu in the workspace resources toolbar."""
        self.log("Testing delete plot via context menu")

        prefix = str(uuid.uuid4())[:6]
        plot = Plot(self.client, self.console, f"Delete Test {prefix}")
        plot_name = plot.page_name
        plot.close()

        assert self.console.workspace.page_exists(
            plot_name
        ), f"Plot '{plot_name}' should exist before deletion"

        self.console.workspace.delete_page(plot_name)

        assert not self.console.workspace.page_exists(
            plot_name, timeout=1000
        ), f"Plot '{plot_name}' should not exist after deletion"

    def test_ctx_delete_multiple_plots(self) -> None:
        """Test deleting multiple plots via multi-select and context menu."""
        self.log("Testing delete multiple plots via context menu")

        prefix = str(uuid.uuid4())[:6]
        plot_names = []

        for i in range(3):
            plot = Plot(self.client, self.console, f"Multi Delete {prefix} {i}")
            plot_names.append(plot.page_name)
            plot.close()

        for name in plot_names:
            assert self.console.workspace.page_exists(
                name
            ), f"Plot '{name}' should exist before deletion"

        self.console.workspace.delete_pages(plot_names)

        for name in plot_names:
            assert not self.console.workspace.page_exists(
                name, timeout=1000
            ), f"Plot '{name}' should not exist after deletion"

    def test_ctx_export_json(self) -> None:
        """Test exporting a plot as JSON via context menu."""
        self.log("Testing export plot via context menu")

        prefix = str(uuid.uuid4())[:6]
        plot = Plot(self.client, self.console, f"Export Test {prefix}")
        plot_name = plot.page_name
        plot.close()

        exported = self.console.workspace.export_page(plot_name)

        assert "key" in exported, "Exported JSON should contain 'key'"
        assert len(exported["key"]) == 36, "Plot key should be a UUID"
        assert "channels" in exported, "Exported JSON should contain 'channels'"

        self.console.workspace.delete_page(plot_name)

    def test_ctx_copy_link(self) -> None:
        """Test copying a link to a plot via context menu."""
        self.log("Testing copy link via context menu")

        prefix = str(uuid.uuid4())[:6]
        plot = Plot(self.client, self.console, f"Copy Link Test {prefix}")
        plot_name = plot.page_name
        expected_link = plot.copy_link()
        plot.close()

        link = self.console.workspace.copy_page_link(plot_name)

        assert (
            link == expected_link
        ), f"Context menu link should match: expected {expected_link}, got {link}"

        self.console.workspace.delete_page(plot_name)

    def test_open_plot_by_name(self, plot_name: str, expected_link: str) -> None:
        """Test opening an existing plot by searching its name in the command palette."""
        self.log("Testing open plot by name via command palette")

        plot = Plot.open_by_name(self.client, self.console, plot_name)

        assert plot.pane_locator is not None, "Plot pane should be visible"
        assert plot.pane_locator.is_visible(), "Plot pane should be visible"

        opened_link = plot.copy_link()
        assert (
            opened_link == expected_link
        ), f"Opened plot link should match: expected {expected_link}, got {opened_link}"

        plot.close()
