#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


from test.console.plot import Plot


class Plot_Abcd(Plot):
    """
    Simple plot test

    Must be run with the LatencyABC test case.
    """

    def setup(self) -> None:
        super().setup()
        self.configure(loop_rate=0.5, manual_timeout=60)

        self.subscribe(["d_ab", "d_bc", "d_cd", "d_da"])
        self.subscribe(
            ["async_a_state", "async_b_state", "async_c_state", "async_d_state"]
        )

    def run(self) -> None:

        self.wait_for_tlm_init()

        self.add_Y("Y1", ["d_ab", "d_bc", "d_cd", "d_da"])
        self.add_Y("Y2", ["t_a", "t_b", "t_c", "t_d"])
        self.add_ranges(["30s"])
        self.set_Y1_axis(
            {
                "Lower Bound": -0.005,
                "Upper Bound": 0.08,
                "Tick Spacing": 50,
            }
        )
        self.set_Y2_axis(
            {
                "Tick Spacing": 100,
            }
        )
        self.wait_for_tlm_stale()

        # Reset after auto adjust
        self.set_Y1_axis(
            {
                "Lower Bound": -0.05,
                "Upper Bound": 0.08,
                "Tick Spacing": 50,
            }
        )
        self.set_Y2_axis(
            {
                "Tick Spacing": 100,
            }
        )
        self.save_screenshot()
