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

import synnax as sy

ARC_INTERVAL_PRESS_SOURCE = """
func open_press() {
    if (press_pt > 30) {
        press_vlv_cmd = 0
        vent_vlv_cmd = 1
        str_chan = "venting"
    } else if (press_pt < 1) {
        press_vlv_cmd = 1
        vent_vlv_cmd = 0
        str_chan = "pressing"
    }
}

interval{period=50ms} -> open_press{}
"""


class ArcIntervalPress(ArcConsoleCase):
    """Test Arc interval-triggered pressure control with hysteresis.

    This test demonstrates:
    1. Interval-based function execution (every 50ms)
    2. Hysteresis control logic (press < 1, vent > 30)
    3. String channel status output
    4. Multiple pressure cycles (3 complete cycles)

    Unlike stage-based sequences, the interval trigger fires automatically
    when the Arc task starts - no explicit trigger channel needed.
    """

    arc_source = ARC_INTERVAL_PRESS_SOURCE
    arc_name_prefix = "ArcIntervalPress"
    start_cmd_channel = "start_interval_press_cmd"
    end_cmd_channel = "end_test_cmd"
    subscribe_channels = [
        "press_vlv_state",
        "vent_vlv_state",
        "press_pt",
        "str_chan",
        "end_test_cmd",
    ]
    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.client.channels.create(
            name="str_chan",
            data_type=sy.DataType.STRING,
            virtual=True,
            retrieve_if_name_exists=True,
        )
        super().setup()

    def verify_sequence_execution(self) -> None:
        self.log("Verifying initial pressing phase...")
        press_opened = False
        while self.should_continue:
            if self.read_tlm("press_vlv_state") == 1:
                self.log("Press valve opened - pressing mode active")
                press_opened = True
                break
        if not press_opened:
            self.fail("Press valve should open when pressure < 1")
            return

        self.log("Verifying str_chan = 'pressing'...")
        pressing_status = False
        while self.should_continue:
            status = self.read_tlm("str_chan")
            if status is not None and str(status) == "pressing":
                self.log(f"Status confirmed: {status}")
                pressing_status = True
                break
        if not pressing_status:
            self.fail("str_chan should show 'pressing'")
            return

        # Verify 3 complete pressure cycles
        for cycle in range(1, 4):
            self.log(f"[Cycle {cycle}] Waiting for pressure to exceed 27...")
            pressure_exceeded = False
            while self.should_continue:
                press_pt = self.read_tlm("press_pt")
                if press_pt is not None and press_pt > 27:
                    self.log(
                        f"[Cycle {cycle}] Pressure exceeded threshold: {press_pt:.1f}"
                    )
                    pressure_exceeded = True
                    break
            if not pressure_exceeded:
                self.fail(f"[Cycle {cycle}] Pressure should rise above 27")
                return

            self.log(f"[Cycle {cycle}] Verifying transition to venting...")
            venting_mode = False
            while self.should_continue:
                vent_state = self.read_tlm("vent_vlv_state")
                press_state = self.read_tlm("press_vlv_state")
                if vent_state == 1 and press_state == 0:
                    self.log(
                        f"[Cycle {cycle}] Vent valve opened, press valve closed - venting mode"
                    )
                    venting_mode = True
                    break
            if not venting_mode:
                self.fail(
                    f"[Cycle {cycle}] Should transition to venting when pressure > 30"
                )
                return

            self.log(f"[Cycle {cycle}] Waiting for pressure to drop below 4...")
            pressure_dropped = False
            while self.should_continue:
                press_pt = self.read_tlm("press_pt")
                if press_pt is not None and press_pt < 4:
                    self.log(
                        f"[Cycle {cycle}] Pressure dropped below threshold: {press_pt:.1f}"
                    )
                    pressure_dropped = True
                    break
            if not pressure_dropped:
                self.fail(f"[Cycle {cycle}] Pressure should drop below 4")
                return

            self.log(f"[Cycle {cycle}] Verifying return to pressing mode...")
            pressing_mode = False
            while self.should_continue:
                press_state = self.read_tlm("press_vlv_state")
                vent_state = self.read_tlm("vent_vlv_state")
                if press_state == 1 and vent_state == 0:
                    self.log(
                        f"[Cycle {cycle}] Press valve opened, vent valve closed - cycle complete!"
                    )
                    pressing_mode = True
                    break
            if not pressing_mode:
                self.fail(
                    f"[Cycle {cycle}] Should return to pressing when pressure < 1"
                )
                return

        self.log(
            "All 3 cycles completed - interval-triggered hysteresis control verified successfully"
        )
