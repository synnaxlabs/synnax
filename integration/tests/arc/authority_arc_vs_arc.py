#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators import PressSimDAQ

import synnax as sy
from framework.utils import create_virtual_channel
from tests.arc.arc_case import ArcConsoleCase

ARC_LOW_PRIORITY_SOURCE = """
authority 100

start_low_cmd => main

sequence main {
    stage active {
        0 -> press_vlv_cmd
        wait{100ms} => active
    }
}
"""

ARC_HIGH_PRIORITY_SOURCE = """
authority 200

start_high_cmd => main

sequence main {
    stage active {
        1 -> press_vlv_cmd
        wait{100ms} => active_hold
    }
    stage active_hold {
        1 -> press_vlv_cmd
        wait{5s} => done
    }
    stage done {
        set_authority{0}
    }
}
"""


class AuthorityArcVsArc(ArcConsoleCase):
    """Test that a higher-authority Arc program wins over a lower one,
    and the lower program resumes when the higher one stops writing."""

    arc_source = ARC_LOW_PRIORITY_SOURCE
    arc_name_prefix = "ArcLow"
    start_cmd_channel = "start_low_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["press_vlv_state", "press_pt", "end_test_cmd"]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        create_virtual_channel(self.client, "start_high_cmd", sy.DataType.UINT8)
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Waiting for Arc A to control (press_vlv_state == 0)...")
        self.wait_for_eq("press_vlv_state", 0)

        self.log("Loading Arc B (high priority)")
        self.load_arc(ARC_HIGH_PRIORITY_SOURCE, "ArcHigh", trigger="start_high_cmd")

        self.log("Waiting for Arc B to override (press_vlv_state == 1)...")
        self.wait_for_eq("press_vlv_state", 1)

        self.log("Waiting for Arc B done stage, Arc A should resume...")
        self.wait_for_eq("press_vlv_state", 0, timeout=10)
