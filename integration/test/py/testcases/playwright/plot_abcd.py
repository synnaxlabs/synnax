#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from testcases.playwright.plot import Plot


class Plot_Abcd(Plot):
    """
    Simple plot test

    Must be run with the LatencyABC test case.
    """

    def setup(self) -> None:
        super().setup()
        self.configure(loop_rate=0.5, manual_timeout=60)
        
        self.subscribe (["d_ab", "d_bc", "d_cd", "d_da"])
        self.subscribe(["async_a_state", 
            "async_b_state", 
            "async_c_state",
             "async_d_state"
            ])
        

    def run(self) -> None:

        self.wait_for_tlm_init()
        time.sleep(5)

        self.add_Y1(["d_ab", "d_bc", "d_cd", "d_da"])
        self.add_Y2(["async_a_state", 
            "async_b_state", 
            "async_c_state", 
            "async_d_state"
            ])
        self.add_ranges(["30s"])

        self.wait_for_tlm_stale()
        self.save_screenshot()
