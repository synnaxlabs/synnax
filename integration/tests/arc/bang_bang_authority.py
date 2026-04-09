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
from tests.arc.arc_case import ArcConsoleCase

ARC_BANG_BANG_SOURCE = """
authority (press_vlv_cmd 210 vent_vlv_cmd 210)

func high_bang{
    sensor chan f32,
    set_point f32,
    lower_deadband f32,
    upper_deadband f32,
    abort_threshold f32
}() u8 {
    state u8 $= 0
    if sensor > (set_point + upper_deadband) {
        state = 0
    } else if sensor < (set_point - lower_deadband) {
        state = 1
    }
    if sensor > abort_threshold {
        state = 0
    }
    return state
}

func low_bang{
    sensor chan f32,
    set_point f32,
    lower_deadband f32,
    upper_deadband f32,
    abort_threshold f32
}() u8 {
    state u8 $= 0
    if sensor < (set_point - lower_deadband) {
        state = 1
    } else if sensor > (set_point + upper_deadband) {
        state = 0
    }
    if sensor > abort_threshold {
        state = 0
    }
    return state
}

sequence bang_bang_controller {
    stage start {
        set_authority{value=220, channel=press_vlv_cmd},
        set_authority{value=220, channel=vent_vlv_cmd},
        interval{200ms} -> high_bang{
            sensor=press_pt,
            set_point=50,
            lower_deadband=5,
            upper_deadband=5,
            abort_threshold=100
        } -> press_vlv_cmd,
        interval{200ms} -> low_bang{
            sensor=press_pt,
            set_point=10,
            lower_deadband=5,
            upper_deadband=5,
            abort_threshold=100
        } -> vent_vlv_cmd,
        bb_stop_cmd => stop
    }
    stage stop {
        0 -> press_vlv_cmd,
        0 -> vent_vlv_cmd,
        wait{250ms} => yield
    }
    stage yield {
        set_authority{value=0, channel=press_vlv_cmd},
        set_authority{value=0, channel=vent_vlv_cmd},
        bb_start_cmd => start
    }
}

bb_start_cmd => bang_bang_controller
"""


class BangBangAuthority(ArcConsoleCase):
    """Test that a bang-bang controller with per-channel set_authority correctly
    releases and reclaims authority on both channels symmetrically.

    Verifies:
    1. Stale bb_start_cmd does NOT cause re-entry on yield activation.
    2. A fresh bb_start_cmd DOES trigger re-entry from yield.
    3. Both channels are symmetrically reclaimed at authority 220 after re-entry.

    Replicates the customer-reported pattern where one valve channel would
    release authority but the other would reclaim it."""

    arc_source = ARC_BANG_BANG_SOURCE
    arc_name_prefix = "ArcBangBang"
    start_cmd_channel = "bb_start_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = ["press_vlv_state", "vent_vlv_state", "press_pt"]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self._press_writer: sy.Writer | None = None
        self._vent_writer: sy.Writer | None = None
        self.client.channels.create(
            name="bb_stop_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        super().setup()
        self.set_manual_timeout(60)

    def verify_sequence_execution(self) -> None:
        try:
            self._verify()
        finally:
            for w in (self._press_writer, self._vent_writer):
                if w is not None:
                    try:
                        w.close()
                    except Exception:
                        pass
            self._press_writer = None
            self._vent_writer = None

    def _verify(self) -> None:
        # Phase 1: Verify bang-bang controller is running.
        # Starting pressure is 0, so high_bang opens the press valve.
        self.log("Phase 1: Waiting for bang-bang to drive press valve open...")
        self.wait_for_eq("press_vlv_state", 1)
        self.log("Bang-bang controller is active")

        # Phase 2: Trigger stop → wait for stop stage to zero the valve.
        self.log("Phase 2: Triggering stop...")
        self.writer.write("bb_stop_cmd", 1)
        self.wait_for_eq("press_vlv_state", 0)

        # Phase 3: Assert sequence remains in yield — the pre-existing bb_start_cmd
        # value must be ignored on yield activation (SY-3870 fix).
        # Wait beyond the stop wait{250ms} to ensure yield is active, then assert
        # the valve has not returned to 1 (which would indicate spurious re-entry).
        self.log("Phase 3: Asserting no re-entry from stale start signal...")
        sy.sleep(0.4)
        self.wait_for_eq("press_vlv_state", 0, timeout=0)
        self.log("Sequence correctly remains in yield")

        # Phase 4: Send a fresh start command and verify re-entry on both channels.
        self.log("Phase 4: Sending fresh start command and verifying re-entry...")
        self.writer.write("bb_start_cmd", 1)
        self.wait_for_eq("press_vlv_state", 1)
        self.log("Bang-bang re-entered start stage")

        # Phase 5: Verify BOTH channels reclaimed authority after re-entry.
        # Open external writers at authority 50 — they should NOT be able to
        # take control since ARC reclaimed at 220 on both channels.
        self.log("Phase 5: Verifying press_vlv_cmd reclaimed by ARC...")
        self._press_writer = self.client.open_writer(
            sy.TimeStamp.now(),
            ["press_vlv_cmd_time", "press_vlv_cmd"],
            50,
            err_on_unauthorized=False,
        )
        self._press_writer.write(
            {
                "press_vlv_cmd_time": sy.TimeStamp.now(),
                "press_vlv_cmd": 0,
            }
        )
        self.wait_for_eq("press_vlv_state", 1, timeout=5)
        self.log("press_vlv_cmd correctly reclaimed by ARC")

        self.log("Verifying vent_vlv_cmd reclaimed by ARC...")
        self._vent_writer = self.client.open_writer(
            sy.TimeStamp.now(),
            ["vent_vlv_cmd_time", "vent_vlv_cmd"],
            50,
            err_on_unauthorized=False,
        )
        self._vent_writer.write(
            {
                "vent_vlv_cmd_time": sy.TimeStamp.now(),
                "vent_vlv_cmd": 0,
            }
        )
        press_state = self.get_value("press_vlv_state")
        vent_state = self.get_value("vent_vlv_state")
        self.log(
            f"After external write attempt: press_vlv_state={press_state}, "
            f"vent_vlv_state={vent_state}"
        )
        self.wait_for_eq("press_vlv_state", 1, timeout=5)
        self.log("Both channels symmetrically reclaimed by ARC after re-entry")

    def teardown(self) -> None:
        for w in (self._press_writer, self._vent_writer):
            if w is not None:
                try:
                    w.close()
                except Exception:
                    pass
        self._press_writer = None
        self._vent_writer = None
        super().teardown()
