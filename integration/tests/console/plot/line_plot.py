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

        self.test_set_line_thickness(plot)
        self.test_set_line_label(plot, prefix)
        self.test_set_plot_title(plot, prefix)
        self.test_move_channel_between_axes(plot, data_name)
        self.test_live_data(plot)
        self.test_drag_channel_to_canvas(plot)
        self.test_drag_channel_to_toolbar(plot)
        self.test_download_csv(plot, data_name)
        self.test_create_range_from_selection(plot, prefix)

        plot.close()
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

        live_channel = "sy_node_1_metrics_mem_percentage"
        plot.add_channels("Y1", live_channel)
        assert (
            live_channel in plot.data["Y1"]
        ), f"Channel {live_channel} should be on Y1"

    def test_drag_channel_to_canvas(self, plot: Plot) -> None:
        """Test dragging a channel from sidebar onto the plot canvas."""
        self.log("Testing drag channel to canvas")

        channel = "sy_node_1_metrics_cpu_percentage"

        # Debug: Show what channels are visible
        self.console.channels.show_channels()
        visible_channels = self.page.locator("div[id^='channel:']").all()
        self.log(f"Found {len(visible_channels)} channel elements")
        for i, ch in enumerate(visible_channels):  # Log first 10
            try:
                text = ch.inner_text(timeout=1000)
                self.log(f"  Channel {i}: {text}")
            except Exception as e:
                self.log(f"  Channel {i}: (error getting text: {e})")
        self.console.channels.hide_channels()

        plot.drag_channel_to_canvas(channel)
        assert channel in plot.data["Y1"], f"Channel {channel} should be on Y1"

    def test_drag_channel_to_toolbar(self, plot: Plot) -> None:
        """Test dragging a channel from sidebar onto the toolbar data section."""
        self.log("Testing drag channel to toolbar")

        channel = "sy_node_1_metrics_total_size_gb"
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
