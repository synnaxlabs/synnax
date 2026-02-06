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

ARC_CONTINUOUS_PRESS_SOURCE = """
authority 200

start_seq_cmd => main

sequence main {
    stage pressurize {
        1 -> press_vlv_cmd,
        wait{duration=100ms} => pressurize
    }
}
"""


class ArcAuthorityPythonOverride(ArcConsoleCase):
    """Test that an external Python writer at higher authority overrides Arc,
    and Arc automatically resumes when the Python writer is closed."""

    arc_source = ARC_CONTINUOUS_PRESS_SOURCE
    arc_name_prefix = "ArcAuthPyOverride"
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
        self._override_writer: sy.Writer | None = None
        super().setup()
        self.set_manual_timeout(60)

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
        # Phase 1: Arc in control at authority 200
        self.log("Phase 1: Verifying Arc controls valve at authority 200...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Phase 1: Press valve opened by Arc")
                break
        else:
            self.fail("Phase 1: Press valve should open")
            return

        # Phase 2: Python writer overrides at authority 255
        self.log("Phase 2: Opening Python writer at authority 255...")
        self._override_writer = self.client.open_writer(
            sy.TimeStamp.now(),
            ["press_vlv_cmd_time", "press_vlv_cmd"],
            255,
        )
        self._override_writer.write(
            {
                "press_vlv_cmd_time": sy.TimeStamp.now(),
                "press_vlv_cmd": 0,
            }
        )

        self.log("Phase 2: Waiting for valve to close (Python override)...")
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 0:
                self.log("Phase 2: Press valve closed by Python override")
                break
        else:
            self.fail("Phase 2: Press valve should close from Python override")
            return

        # Phase 3: Close Python writer, Arc resumes
        self.log("Phase 3: Closing Python writer, Arc should resume...")
        self._override_writer.close()
        self._override_writer = None

        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Phase 3: Press valve reopened by Arc (auto-resume)")
                break
        else:
            self.fail("Phase 3: Arc should resume control after Python writer closes")
            return

    def teardown(self) -> None:
        if self._override_writer is not None:
            try:
                self._override_writer.close()
            except Exception:
                pass
            self._override_writer = None
        super().teardown()
