#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.simulators import PressSimDAQ

from tests.arc.arc_case import ArcConsoleCase

ARC_EMERGENCY_SOURCE = """
authority 100

start_seq_cmd => main

sequence main {
    stage normal {
        1 -> press_vlv_cmd,
        press_pt > 50 => emergency
    }
    stage emergency {
        set_authority{value=255},
        0 -> press_vlv_cmd,
        1 -> vent_vlv_cmd,
        press_pt < 5 => safed
    }
    stage safed {
        0 -> press_vlv_cmd,
        0 -> vent_vlv_cmd
    }
}
"""


class ArcAuthorityEmergency(ArcConsoleCase):
    """Test that Arc can escalate authority with set_authority{value=255}
    to reclaim control from a higher-authority external writer during an
    emergency condition."""

    arc_source = ARC_EMERGENCY_SOURCE
    arc_name_prefix = "ArcEmergency"
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
        self._override_writer = None
        super().setup()
        self.set_manual_timeout(120)

    def verify_sequence_execution(self) -> None:
        try:
            self._verify()
        finally:
            if self._override_writer is not None:
                try:
                    self._override_writer.close()
                except Exception:
                    pass
                self._override_writer = None

    def _verify(self) -> None:
        # Phase 1: Arc opens valve at authority 100, pressure rises
        self.log("Phase 1: Waiting for Arc to open press valve (authority 100)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Phase 1: Press valve opened - pressure rising")
                break
        else:
            self.fail("Phase 1: Arc should open press valve")
            return

        # Phase 2: External writer at authority 200 keeps valve forced open
        # Both Arc (100) and Python (200) write 1, so pressure keeps rising.
        # The Python writer prevents Arc from closing the valve when it tries
        # to in the emergency stage - UNLESS Arc escalates past 200.
        self.log("Phase 2: Opening Python writer at authority 200 on press_vlv_cmd...")
        self._override_writer = self.client.open_writer(
            sy.TimeStamp.now(),
            ["press_vlv_cmd_time", "press_vlv_cmd"],
            200,
        )
        self._override_writer.write(
            {
                "press_vlv_cmd_time": sy.TimeStamp.now(),
                "press_vlv_cmd": 1,
            }
        )
        self.log("Phase 2: Python writer holding authority 200 with valve open")

        # Phase 3: Pressure rises past 50 -> Arc enters emergency stage
        # set_authority{value=255} fires first (flushed before writes),
        # then 0 -> press_vlv_cmd succeeds because 255 > 200.
        self.log("Phase 3: Waiting for emergency escalation (press_vlv_state == 0)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 0:
                self.log("Phase 3: Arc escalated to 255 - press valve closed")
                break
        else:
            self.fail("Phase 3: Arc should escalate and close press valve")
            return

        self.log("Phase 3: Verifying vent valve opened...")
        while self.should_continue:
            if self.read_tlm("vent_vlv_state") == 1:
                self.log("Phase 3: Vent valve opened - depressurizing")
                break
        else:
            self.fail("Phase 3: Vent valve should open in emergency stage")
            return

        # Phase 4: Close Python writer, wait for safe state
        self.log("Phase 4: Closing Python writer...")
        self._override_writer.close()
        self._override_writer = None

        self.log("Phase 4: Waiting for safed stage (vent_vlv_state == 0)...")
        while self.should_continue:
            if self.read_tlm("vent_vlv_state") == 0:
                self.log("Phase 4: System safed - vent valve closed")
                break
        else:
            self.fail("Phase 4: Vent valve should close in safed stage")
            return

    def teardown(self) -> None:
        if self._override_writer is not None:
            try:
                self._override_writer.close()
            except Exception:
                pass
            self._override_writer = None
        super().teardown()
