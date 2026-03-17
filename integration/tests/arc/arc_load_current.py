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
        load_current > 50 => wait{duration=2s} => next,
    }
    stage last {
        0 -> flag,
    }
}
"""

WAIT_DURATION = 2.0
TIMING_TOLERANCE = 1.5


class ArcLoadCurrent(ArcConsoleCase):
    """Test condition-gated wait timer with stage transition.

    Verifies:
    1. Stage entry writes flag=1 immediately on sequence start.
    2. The wait timer does not begin until load_current exceeds 50.
    3. After the 2s wait elapses, the sequence transitions to the last stage
       and writes flag=0.
    4. The elapsed time between condition firing and stage transition is
       approximately equal to the wait duration.
    """

    arc_source = ARC_LOAD_CURRENT_SOURCE
    arc_name_prefix = "ArcLoadCurrent"
    start_cmd_channel = "start_load_current_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["load_current", "flag"]
    sim_daq_class = LoadCurrentSimDAQ

    def setup(self) -> None:
        super().setup()
        self.set_manual_timeout(30)

    def verify_sequence_execution(self) -> None:
        self.log("Phase 1: Waiting for flag == 1 (stage first entered)...")
        self.wait_for_eq("flag", 1, is_virtual=True)
        self.log("flag is 1, stage first is active")

        self.log("Phase 2: Waiting for load_current > 50 (wait timer starts)...")
        self.wait_for_gt("load_current", 50, timeout=10 * sy.TimeSpan.SECOND)
        timer = sy.Timer()
        self.log("load_current crossed 50, wait timer should now be running")

        self.log("Phase 3: Asserting flag is still 1 (wait has not elapsed yet)...")
        sy.sleep(500 * sy.TimeSpan.MILLISECOND)
        self.wait_for_eq("flag", 1, timeout=0, is_virtual=True)
        self.log("flag remains 1 during wait period")

        self.log("Phase 4: Waiting for flag == 0 (stage last entered after 2s wait)...")
        self.wait_for_eq("flag", 0, timeout=10 * sy.TimeSpan.SECOND, is_virtual=True)

        elapsed_secs = timer.elapsed() / sy.TimeSpan.SECOND
        self.log(
            f"Wait elapsed: {elapsed_secs:.2f}s "
            f"(expected ~{WAIT_DURATION}s, tolerance {TIMING_TOLERANCE}s)"
        )

        assert elapsed_secs >= WAIT_DURATION - TIMING_TOLERANCE, (
            f"Wait fired too early: {elapsed_secs:.2f}s < "
            f"{WAIT_DURATION - TIMING_TOLERANCE:.1f}s"
        )
        assert elapsed_secs <= WAIT_DURATION + TIMING_TOLERANCE, (
            f"Wait took too long: {elapsed_secs:.2f}s > "
            f"{WAIT_DURATION + TIMING_TOLERANCE:.1f}s"
        )

        self.log("Condition-gated wait transition verified with correct timing.")
