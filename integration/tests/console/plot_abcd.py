#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from console.case import ConsoleCase


class PlotAbcd(ConsoleCase):
    """
    Simple plot test

    Must be run with the LatencyABC test case.
    """

    def setup(self) -> None:
        super().setup()
        self.configure(loop_rate=0.5, manual_timeout=30)

        self.subscribe(["d_ab", "d_bc", "d_cd", "d_da"])
        self.subscribe(
            ["async_a_state", "async_b_state", "async_c_state", "async_d_state"]
        )

    def run(self) -> None:
        console = self.console
        console.plot.new()

        console.plot.add_Y("Y1", ["d_ab", "d_bc", "d_cd", "d_da"])
        console.plot.add_Y("Y2", ["t_a", "t_b", "t_c", "t_d"])
        console.plot.add_ranges(["30s"])
        console.plot.set_axis(
            "Y1",
            {
                "Lower Bound": -0.005,
                "Upper Bound": 0.08,
                "Tick Spacing": 50,
            },
        )
        console.plot.set_axis(
            "Y2",
            {
                "Tick Spacing": 100,
            },
        )
        self.wait_for_tlm_stale()

        # Reset after auto adjust
        console.plot.set_axis(
            "Y1",
            {
                "Lower Bound": -0.05,
                "Upper Bound": 0.08,
                "Tick Spacing": 50,
            },
        )
        console.plot.set_axis(
            "Y2",
            {
                "Tick Spacing": 100,
            },
        )
        console.plot.screenshot()
