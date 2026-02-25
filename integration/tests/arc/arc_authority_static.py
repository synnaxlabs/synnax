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

ARC_STATIC_AUTHORITY_SOURCE = """
authority 200

start_seq_cmd => main

sequence main {
    stage press {
        1 -> press_vlv_cmd,
        press_pt > 30 => vent
    }

    stage vent {
        0 -> press_vlv_cmd,
        1 -> vent_vlv_cmd,
        press_pt < 5 => complete
    }

    stage complete {
        0 -> vent_vlv_cmd
    }
}
"""


class ArcStaticAuthority(ArcConsoleCase):
    """Test that static authority declaration (authority 200) works for
    normal uncontested operation. The sequence should complete normally
    with non-absolute authority."""

    arc_source = ARC_STATIC_AUTHORITY_SOURCE
    arc_name_prefix = "ArcStaticAuthority"
    start_cmd_channel = "start_seq_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "press_vlv_state",
        "vent_vlv_state",
        "press_pt",
        "end_test_cmd",
    ]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        self.log("Verifying press stage - valve opens (authority 200)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Press valve opened at authority 200")
                break
        else:
            self.fail("Press valve should open at authority 200")
            return

        self.log("Verifying vent stage - vent valve opens...")
        while self.should_continue:
            if self.read_tlm("vent_vlv_state") == 1:
                self.log("Vent stage reached - vent valve opened")
                break
        else:
            self.fail("Vent valve should open")
            return

        self.log("Verifying complete stage - vent valve closes...")
        while self.should_continue:
            if self.read_tlm("vent_vlv_state") == 0:
                self.log("Complete stage reached - sequence finished")
                break
        else:
            self.fail("Vent valve should close in complete stage")
            return
