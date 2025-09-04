#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
import re
from zipfile._path import InitializedState
from testcases.playwright.plot import Plot


class Plot_Simple(Plot):
    """
    Simple plot test
    """

    def setup(self) -> None:
        super().setup()
        self.configure(loop_rate=0.5, manual_timeout=60)
        
        self.sub_Y1(["d_ab", "d_bc", "d_cd", "d_da"])
        self.sub_Y2(["t_a", "t_b", "t_c", "t_d"])
        

    def run(self) -> None:

        #self.add_Y1([f"{self.name}_uptime"])


        


        self.wait_for_tlm_init()

        time.sleep(2)
        self.add_Y1(["d_ab", "d_bc", "d_cd", "d_da"])
        self.add_Y2(["t_a", "t_b", "t_c", "t_d"])
        self.add_ranges(["30s"])
        time.sleep(10)
        self.wait_for_tlm_none()

        
        self.save_screenshot()
