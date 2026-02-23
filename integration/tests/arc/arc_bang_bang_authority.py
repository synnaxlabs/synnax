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

ARC_BANG_BANG_SOURCE = """
authority (press_vlv_cmd 210 vent_vlv_cmd 210)

func high_bang{
    sensor chan f32,
    set_point f32,
    lower_deadband f32,
    upper_deadband f32,
    abort_threshold f32
}() u8 {
    state $= 0
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
    state $= 0
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
        interval{period=200ms} -> high_bang{
            sensor=press_pt,
            set_point=50,
            lower_deadband=5,
            upper_deadband=5,
            abort_threshold=100
        } -> press_vlv_cmd,
        interval{period=200ms} -> low_bang{
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
        wait{duration=250ms} => yield
    }
    stage yield {
        set_authority{value=0, channel=press_vlv_cmd},
        set_authority{value=0, channel=vent_vlv_cmd},
        bb_start_cmd => start
    }
}

bb_start_cmd => bang_bang_controller
"""


class ArcBangBangAuthority(ArcConsoleCase):
    """Test that a bang-bang controller with per-channel set_authority correctly
    releases authority on both channels after transitioning through
    start -> stop -> yield.

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

        # Phase 2: Clear the start signal so yield doesn't re-enter start,
        # then trigger the stop transition.
        self.log("Phase 2: Clearing start signal...")
        with self.client.open_writer(sy.TimeStamp.now(), "bb_start_cmd") as w:
            w.write("bb_start_cmd", 0)
        time.sleep(0.5)

        self.log("Triggering stop...")
        with self.client.open_writer(sy.TimeStamp.now(), "bb_stop_cmd") as w:
            w.write("bb_stop_cmd", 1)

        # Wait for stop stage (writes 0 to both, waits 250ms) then yield.
        self.log("Waiting for stop -> yield transition...")
        time.sleep(1.5)

        # Phase 3: Verify authority released on BOTH channels by opening
        # external writers at authority 50. If the Arc released to 0, these
        # should take control. If one channel still holds 220, its write fails.
        self.log("Phase 3: Verifying authority released on press_vlv_cmd...")
        self._press_writer = self.client.open_writer(
            sy.TimeStamp.now(),
            ["press_vlv_cmd_time", "press_vlv_cmd"],
            50,
        )
        self._press_writer.write(
            {
                "press_vlv_cmd_time": sy.TimeStamp.now(),
                "press_vlv_cmd": 1,
            }
        )
        self.wait_for_eq("press_vlv_state", 1, timeout=5)
        self.log("press_vlv_cmd authority released OK")

        self.log("Verifying authority released on vent_vlv_cmd...")
        self._vent_writer = self.client.open_writer(
            sy.TimeStamp.now(),
            ["vent_vlv_cmd_time", "vent_vlv_cmd"],
            50,
        )
        self._vent_writer.write(
            {
                "vent_vlv_cmd_time": sy.TimeStamp.now(),
                "vent_vlv_cmd": 1,
            }
        )
        self.wait_for_eq("vent_vlv_state", 1, timeout=5)
        self.log("vent_vlv_cmd authority released OK")

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
