#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.simulators import PressSimDAQ

from tests.arc.arc_case import ArcConsoleCase

ARC_SEQUENCE_SOURCE = """
start_seq_cmd => main

sequence main {
    stage press {
        1 -> press_vlv_cmd,
        press_pt > 30 => maintain
    }

    stage maintain {
        0 -> press_vlv_cmd,
        wait{duration=1s} => vent
    }

    stage vent {
        1 -> vent_vlv_cmd,
        press_pt < 5 => complete
    }

    stage complete {
        0 -> vent_vlv_cmd
    }
}
"""


class ArcPressSequence(ArcConsoleCase):
    """Test Arc pressurization sequence execution via Console UI."""

    arc_source = ARC_SEQUENCE_SOURCE
    arc_name_prefix = "ArcPressSequence"
    start_cmd_channel = "start_seq_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "press_vlv_state",
        "vent_vlv_state",
        "press_pt",
        "end_test_cmd",
    ]
    sim_daq_class = PressSimDAQ

    def verify_sequence_execution(self) -> None:
        self.log("Verifying press stage - valve opens...")
        self.wait_for_eq("press_vlv_state", 1)
        self.log("Press valve opened")

        self.log("Verifying maintain stage - press valve closes...")
        self.wait_for_eq("press_vlv_state", 0)
        self.log("Maintain stage reached - press valve closed")

        self.log("Verifying vent stage - vent valve opens...")
        self.wait_for_eq("vent_vlv_state", 1)
        self.log("Vent stage reached - vent valve opened")

        self.log("Verifying complete stage - vent valve closes...")
        self.wait_for_eq("vent_vlv_state", 0)
        self.log("Complete stage reached - sequence finished!")
