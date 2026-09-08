#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from console.case import ConsoleCase
from console.plot import Plot
from x import random_name


class CalculatedChannels(ConsoleCase):
    """Test calculated channel editing, plotting, and error handling."""

    suffix: str
    calc_x2: str
    calc_x6: str
    calc_editable: str

    def setup(self) -> None:
        super().setup()
        self.suffix = random_name()
        self.calc_x2 = f"calc_x2_{self.suffix}"
        self.calc_x6 = f"calc_x6_{self.suffix}"
        self.calc_editable = f"calc_edit_{self.suffix}"

        error = self.console.channels.create_calculated(
            name=self.calc_x2,
            expression=f"return {self._ch_uptime} * 2",
        )
        assert error is None, f"Failed to create {self.calc_x2}: {error}"

        error = self.console.channels.create_calculated(
            name=self.calc_x6,
            expression=f"return {self.calc_x2} * 3",
        )
        assert error is None, f"Failed to create {self.calc_x6}: {error}"

        error = self.console.channels.create_calculated(
            name=self.calc_editable,
            expression=f"return {self._ch_uptime} * 2",
        )
        assert error is None, f"Failed to create {self.calc_editable}: {error}"

    def teardown(self) -> None:
        with self._try_to("delete channels"):
            self.console.channels.delete(
                [self.calc_x6, self.calc_x2, self.calc_editable]
            )
        super().teardown()

    def run(self) -> None:
        self.test_edit_calculated_channel()
        self.test_plot_calculated_channel()
        self.test_erroneous_calculated_channel()

    def test_edit_calculated_channel(self) -> None:
        """Test editing a calculated channel's calculation via context menu."""
        self.log("Testing edit calculated channel")

        console = self.console
        client = self.client

        updated_multiplier = 30
        updated_expr = f"return {self._ch_uptime} * {updated_multiplier}"

        console.channels.edit_calculated(self.calc_editable, updated_expr)
        for _ in range(5):
            sy.sleep(0.5)
            frame = client.read_latest([self.calc_editable, self._ch_uptime], n=1)
            if len(frame[self._ch_uptime]) > 0:
                break
        uptime_val = int(frame[self._ch_uptime][-1])
        calc_val = int(frame[self.calc_editable][-1])
        expected_val = uptime_val * updated_multiplier
        assert expected_val == calc_val, f"expected {expected_val}, got {calc_val}"

    def test_plot_calculated_channel(self) -> None:
        """Test plotting a nested calculated channel (calc channel referencing another calc channel)."""
        self.log("Testing plot nested calculated channel")

        plot = self.console.pages.create(Plot, f"Nested Calc Plot {self.suffix}")
        self._cleanup_pages.append(plot.page_name)
        plot.add_channels("Y1", [self._ch_uptime, self.calc_x2, self.calc_x6])
        csv_content = plot.download_csv()

        assert self.calc_x2 in csv_content, f"CSV should contain {self.calc_x2}"
        assert self.calc_x6 in csv_content, f"CSV should contain {self.calc_x6}"

        lines = csv_content.strip().split("\n")
        header = lines[0].split(",")
        src_idx = header.index(self._ch_uptime)
        calc_x2_idx = header.index(self.calc_x2)
        calc_x6_idx = header.index(self.calc_x6)

        for line in lines[1:]:
            values = line.split(",")
            src_val = int(values[src_idx])
            calc_x2_val = int(values[calc_x2_idx])
            calc_x6_val = int(values[calc_x6_idx])

            expected_x2 = src_val * 2
            expected_x6 = src_val * 2 * 3
            assert calc_x2_val == expected_x2, (
                f"calc_x2 mismatch: {src_val} * 2 = {expected_x2}, got {calc_x2_val}"
            )
            assert calc_x6_val == expected_x6, (
                f"calc_x6 mismatch: {src_val} * 6 = {expected_x6}, got {calc_x6_val}"
            )

        plot.close()

    def test_erroneous_calculated_channel(self) -> None:
        """Test that erroneous calculated channel expressions are handled gracefully."""
        console = self.console
        console.notifications.close_all()

        self.log("Testing erroneous calculated channel (nonexistent channel)")
        calc_name = f"calc_err_{self.suffix}"
        bad_ch_expression = "return nonexistent_channel_xyz * 3"

        error = console.channels.create_calculated(
            name=calc_name, expression=bad_ch_expression
        )

        assert error is not None, "Expected error for nonexistent channel"
        assert "Failed to update calculated channel" in error, (
            f"Error should mention failure: {error}"
        )
        assert "undefined symbol" in error, (
            f"Error should mention undefined symbol: {error}"
        )
        assert "nonexistent_channel_xyz" in error, (
            f"Error should mention nonexistent channel: {error}"
        )
