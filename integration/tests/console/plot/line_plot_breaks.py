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
from x import random_name

# Each range holds a minute of samples, and a day apart the gap between them collapses
# into a break.
RANGE_LENGTH = sy.TimeSpan.MINUTE
RANGE_GAP = sy.TimeSpan.DAY
# The rolling window that every new plot shows.
NEW_PLOT_RANGE = "30s"


class LinePlotBreaks(ConsoleCase):
    """Test the breaks between the ranges of a line plot."""

    suffix: str
    index_name: str
    data_name: str
    ranges: list[sy.Range]

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()
        self.index_name = f"breaks_idx_{self.suffix}"
        self.data_name = f"breaks_data_{self.suffix}"
        index = self.client.channels.create(name=self.index_name, is_index=True)
        self.client.channels.create(
            name=self.data_name, data_type=sy.DataType.FLOAT32, index=index.key
        )
        first = sy.TimeStamp.now() - RANGE_GAP * 3
        starts = [first, first + RANGE_GAP]
        self.ranges = []
        for i, start in enumerate(starts):
            self._write_minute(start)
            self.ranges.append(
                self.client.ranges.create(
                    name=f"Breaks {i} {self.suffix}",
                    time_range=sy.TimeRange(start, start + RANGE_LENGTH),
                )
            )

    def teardown(self) -> None:
        try:
            self.client.ranges.delete([r.key for r in self.ranges])
            self.client.channels.delete([self.data_name, self.index_name])
        finally:
            super().teardown()

    def _write_minute(self, start: sy.TimeStamp) -> None:
        """Write a minute of samples from start."""
        count = 60
        end = start + RANGE_LENGTH
        timestamps = np.linspace(int(start), int(end), count, dtype=np.int64)
        values = np.sin(np.linspace(0, 4 * np.pi, count)).astype(np.float32)
        self.client.write(start, {self.index_name: timestamps, self.data_name: values})

    def run(self) -> None:
        plot = self.console.pages.create(Plot, f"Line Plot Breaks {self.suffix}")
        self._cleanup_pages.append(plot.page_name)
        plot.add_channels("y1", self.data_name)
        # On a stopped channel, the rolling window ends at the newest sample, which
        # would keep a break after the test removes a range.
        plot.remove_range(NEW_PLOT_RANGE)
        plot.add_ranges([r.name for r in self.ranges])

        self.test_breaks_shown(plot)
        self.test_open_and_close_all_breaks(plot)
        self.test_open_breaks_after_reload(plot)
        self.test_hide_and_show_breaks(plot)
        self.test_no_toggle_for_one_range(plot)

    def test_breaks_shown(self, plot: Plot) -> None:
        """Test that two ranges far apart show a break and the toggle."""
        self.log("Testing breaks between ranges far apart")
        plot.expect_breaks_shown()
        plot.expect_menu_option("Open all breaks")

    def test_open_and_close_all_breaks(self, plot: Plot) -> None:
        """Test that the plot menu opens and closes every break."""
        self.log("Testing open and close all breaks")
        plot.expect_menu_option("Close all breaks", offered=False)

        plot.select_menu_option("Open all breaks")
        plot.expect_menu_option("Close all breaks")
        plot.expect_menu_option("Open all breaks", offered=False)

        plot.select_menu_option("Close all breaks")
        plot.expect_menu_option("Open all breaks")
        plot.expect_menu_option("Close all breaks", offered=False)

    def test_open_breaks_after_reload(self, plot: Plot) -> None:
        """Test that the breaks a user opens stay open after a reload."""
        self.log("Testing open breaks after a reload")
        plot.select_menu_option("Open all breaks")
        plot.expect_menu_option("Close all breaks")

        plot.reload_console()
        plot.expect_menu_option("Close all breaks")
        plot.expect_menu_option("Open all breaks", offered=False)

        plot.select_menu_option("Close all breaks")
        plot.expect_menu_option("Open all breaks")

    def test_hide_and_show_breaks(self, plot: Plot) -> None:
        """Test that the toggle hides the breaks and shows them again."""
        self.log("Testing hide and show breaks")
        plot.toggle_breaks()
        plot.expect_breaks_shown(shown=False)
        plot.expect_menu_option("Open all breaks", offered=False)

        plot.toggle_breaks()
        plot.expect_breaks_shown()
        plot.expect_menu_option("Open all breaks")

    def test_no_toggle_for_one_range(self, plot: Plot) -> None:
        """Test that a plot of one range shows no breaks toggle."""
        self.log("Testing no breaks toggle for one range")
        plot.remove_range(self.ranges[1].name)
        plot.expect_no_breaks_toggle()
