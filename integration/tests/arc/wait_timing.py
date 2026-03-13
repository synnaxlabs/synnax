#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time

import synnax as sy
from examples.simulators import PressSimDAQ

from tests.arc.arc_case import ArcConsoleCase

ARC_WAIT_TIMING_SOURCE = """
authority 200

sequence main {
    stage stage1 {
        1 -> toggle_cmd,
        wait{duration=3s} => next,
    }
    stage stage2 {
        0 -> toggle_cmd,
    }
}

start_cmd => main
"""

# The wait{duration=3s} should cause a ~3 second delay between stage1 and stage2.
# With reasonable overhead, we allow up to 4 seconds. Anything beyond that indicates
# a bug in the runtime's interval or wait handling.
MAX_WAIT_DURATION = 4.0
MIN_WAIT_DURATION = 2.5


class ArcWaitTiming(ArcConsoleCase):
    """Test that wait{duration=3s} stage transitions complete in ~3 seconds.

    Regression test for a bug where wait durations took 5-7 seconds instead of 3.
    The test creates a two-stage sequence where stage1 writes 1 to toggle_cmd and
    waits 3 seconds before transitioning to stage2, which writes 0 to toggle_cmd.
    We measure the wall-clock time between observing toggle_cmd=1 and toggle_cmd=0
    and assert it falls within [2.5, 4.0] seconds.
    """

    arc_source = ARC_WAIT_TIMING_SOURCE
    arc_name_prefix = "ArcWaitTiming"
    start_cmd_channel = "start_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "toggle_cmd",
        "end_test_cmd",
    ]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        toggle_cmd_time = self.client.channels.create(
            name="toggle_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        self.client.channels.create(
            name="toggle_cmd",
            data_type=sy.DataType.UINT8,
            index=toggle_cmd_time.key,
            retrieve_if_name_exists=True,
        )
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for toggle_cmd=1 (stage1 entered)...")
        self.wait_for_eq("toggle_cmd", 1, timeout=5.0)
        stage1_time = time.monotonic()
        self.log("toggle_cmd=1 observed, starting 3s wait measurement")

        self.log("Waiting for toggle_cmd=0 (stage2 entered)...")
        self.wait_for_eq("toggle_cmd", 0, timeout=8.0)
        stage2_time = time.monotonic()

        wait_duration = stage2_time - stage1_time
        self.log(f"Wait duration: {wait_duration:.2f}s (expected ~3.0s)")

        if wait_duration > MAX_WAIT_DURATION:
            self.fail(
                f"wait{{duration=3s}} took {wait_duration:.2f}s, "
                f"exceeding maximum of {MAX_WAIT_DURATION}s. "
                f"Expected ~3s."
            )
            return

        if wait_duration < MIN_WAIT_DURATION:
            self.fail(
                f"wait{{duration=3s}} took {wait_duration:.2f}s, "
                f"below minimum of {MIN_WAIT_DURATION}s. "
                f"Expected ~3s."
            )
            return

        self.log(
            f"Wait timing verified: {wait_duration:.2f}s is within "
            f"[{MIN_WAIT_DURATION}, {MAX_WAIT_DURATION}]s"
        )
