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
        self.wait_for_eq("press_vlv_state", 1)
        self.log("Press valve opened - pressing mode active")

        self.log("Verifying str_chan = 'pressing'...")
        self.wait_for_eq("str_chan", "pressing", is_virtual=True)
        self.log("Status confirmed: pressing")

        # Verify 3 complete pressure cycles
        for cycle in range(1, 4):
            self.log(f"[Cycle {cycle}] Waiting for pressure to exceed 27...")
            self.wait_for_gt("press_pt", 27)
            self.log(f"[Cycle {cycle}] Pressure exceeded threshold")

            self.log(f"[Cycle {cycle}] Verifying transition to venting...")
            self.wait_for_eq("vent_vlv_state", 1)
            self.wait_for_eq("press_vlv_state", 0)
            self.log(f"[Cycle {cycle}] Venting mode")

            self.log(f"[Cycle {cycle}] Waiting for pressure to drop below 4...")
            self.wait_for_lt("press_pt", 4)
            self.log(f"[Cycle {cycle}] Pressure dropped")

            self.log(f"[Cycle {cycle}] Verifying return to pressing mode...")
            self.wait_for_eq("press_vlv_state", 1)
            self.wait_for_eq("vent_vlv_state", 0)
            self.log(f"[Cycle {cycle}] Cycle complete!")

        self.log(
            "All 3 cycles completed - interval-triggered hysteresis control verified successfully"
        )
