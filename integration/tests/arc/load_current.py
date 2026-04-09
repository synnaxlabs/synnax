#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_LOAD_CURRENT_SOURCE = """
func count{c_chan chan u8}() {
    n u8 $= 0
    n = n + 1
    c_chan = n
}

start_load_current_cmd => main

sequence main {
    stage first {
        1 -> flag,
        load_current > 50 => count{c_chan = lc_tick_count},
        load_current > 50 => wait{2s} => next,
    }
    stage last {
        0 -> flag,
    }
}
"""

WAIT_DURATION = 2.0
TIMING_TOLERANCE = 1.5


class LoadCurrent(ArcConsoleCase):
    """Test condition-gated wait timer with stage transition.

    Verifies:
    1. Stage entry writes flag=1 immediately on sequence start.
    2. The wait timer does not begin until load_current exceeds 50.
    3. While load_current > 50, the conditional edge re-fires every tick,
       incrementing lc_tick_count above 1.
    4. After the 2s wait elapses, the sequence transitions to the last stage
       and writes flag=0.
    5. The elapsed time between condition firing and stage transition is
       approximately equal to the wait duration.
    """

    arc_source = ARC_LOAD_CURRENT_SOURCE
    arc_name_prefix = "ArcLoadCurrent"
    start_cmd_channel = "start_load_current_cmd"
    subscribe_channels = ["load_current", "flag", "lc_tick_count"]

    def setup(self) -> None:
        create_virtual_channel(self.client, "load_current")
        create_virtual_channel(self.client, "flag", sy.DataType.UINT8)
        idx = self.client.channels.create(
            name="lc_tick_count_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="lc_tick_count",
            index=idx.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )
        super().setup()
        self.set_manual_timeout(30)

    def verify_sequence_execution(self) -> None:
        self.log("Phase 1: Waiting for flag == 1 (stage first entered)...")
        self.wait_for_eq("flag", 1, is_virtual=True)
        self.log("flag is 1, stage first is active")

        self.log("Phase 2: Writing load_current = 100 to trigger condition...")
        self.writer.write("load_current", 100.0)
        timer = sy.Timer()
        self.log("load_current set to 100, wait timer should now be running")

        self.log("Phase 3: Asserting flag is still 1 (wait has not elapsed yet)...")
        sy.sleep(500 * sy.TimeSpan.MILLISECOND)
        self.wait_for_eq("flag", 1, timeout=0, is_virtual=True)
        self.log("flag remains 1 during wait period")

        self.log("Phase 3b: Verifying conditional edge re-fires while truthy...")
        self.wait_for_gt("lc_tick_count", 1, timeout=5 * sy.TimeSpan.SECOND)
        tick_count = self.get_value("lc_tick_count")
        self.log(f"lc_tick_count={tick_count} (>1 confirms conditional re-fires)")

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
