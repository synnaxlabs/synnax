#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.simulators import LoadCurrentSimDAQ

from tests.arc.arc_case import ArcConsoleCase

ARC_LOAD_CURRENT_SOURCE = """
start_load_current_cmd => main

sequence main {
    stage first {
        1 -> flag,
        Load_Current > 50 => wait{duration=5s} => next,
    }
    stage last {
        0 -> flag,
    }
}
"""


class ArcLoadCurrent(ArcConsoleCase):
    """Test condition-gated wait timer with stage transition.

    Verifies:
    1. Stage entry writes flag=1 immediately on sequence start.
    2. The wait timer does not begin until Load_Current exceeds 50.
    3. After the 5s wait elapses, the sequence transitions to the last stage
       and writes flag=0.
    """

    arc_source = ARC_LOAD_CURRENT_SOURCE
    arc_name_prefix = "ArcLoadCurrent"
    start_cmd_channel = "start_load_current_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["Load_Current", "flag"]
    sim_daq_class = LoadCurrentSimDAQ

    def setup(self) -> None:
        super().setup()
        self.set_manual_timeout(45)

    def verify_sequence_execution(self) -> None:
        self.log("Phase 1: Waiting for flag == 1 (stage first entered)...")
        self.wait_for_eq("flag", 1, is_virtual=True)
        self.log("flag is 1, stage first is active")

        self.log("Phase 2: Waiting for Load_Current > 50 (wait timer starts)...")
        self.wait_for_gt("Load_Current", 50, timeout=15)
        self.log("Load_Current crossed 50, wait timer should now be running")

        self.log("Phase 3: Asserting flag is still 1 (wait has not elapsed yet)...")
        sy.sleep(1)
        self.wait_for_eq("flag", 1, timeout=0, is_virtual=True)
        self.log("flag remains 1 during wait period")

        self.log("Phase 4: Waiting for flag == 0 (stage last entered after 5s wait)...")
        self.wait_for_eq("flag", 0, timeout=10, is_virtual=True)
        self.log("flag is 0, stage last entered. Condition-gated wait transition verified.")
