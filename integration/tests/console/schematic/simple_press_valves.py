#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from console.case import ConsoleCase
from console.schematic.schematic import Schematic


class SimplePressValves(ConsoleCase):
    """
    Test a basic press control sequence using valves and buttons
    """

    def setup(self) -> None:
        self.set_manual_timeout(90)
        self.subscribe(
            [
                "end_test_cmd",
                "press_vlv_cmd",
                "press_vlv_state",
                "vent_vlv_cmd",
                "vent_vlv_state",
                "press_pt",
            ]
        )
        super().setup()

    def run(self) -> None:

        # Define the control channel names
        END_CMD = "end_test_cmd"
        PRESSURE = "press_pt"

        self.log("Creating schematic symbols")
        schematic = Schematic(self.client, self.console, "simple_press_valves")
        schematic.move("left")

        end_test_cmd = schematic.create_button(END_CMD)
        end_test_cmd.move(0, -90)

        press_valve = schematic.create_valve("press_vlv")
        press_valve.move(-200, 0)

        vent_valve = schematic.create_valve("vent_vlv")
        schematic.connect_symbols(press_valve, "right", vent_valve, "left")

        self.log("Starting test")
        target_Pressure = 20

        for _ in range(3):
            self.log(f"Target pressure: {target_Pressure}")
            press_valve.press()
            self.assert_states(press_state=1, vent_state=0)
            while self.should_continue:
                pressure_value = self.get_value(PRESSURE)
                if pressure_value is not None and pressure_value > target_Pressure:
                    break
                elif self.should_stop:
                    self.fail("Exiting on timeout.")
                    return

            # Configure next cycle
            self.log("Closing press valve")
            press_valve.press()
            self.assert_states(press_state=0, vent_state=0)
            target_Pressure += 20

        # Safe the system
        self.log("Venting the system")
        vent_valve.press()
        self.assert_states(press_state=0, vent_state=1)
        while self.should_continue:
            pressure_value = self.get_value(PRESSURE)
            if pressure_value is not None and pressure_value < 5:
                self.log("Closing vent valve")
                vent_valve.press()
                self.assert_states(press_state=0, vent_state=0)
                end_test_cmd.press()
                self.console.screenshot("console_press_control_passed")
                return

        self.console.screenshot("console_press_control_failed")
        self.fail("Exited without venting")

    def assert_states(self, press_state: int, vent_state: int) -> None:
        sy.sleep(1)
        press_vlv_state = self.client.read_latest("press_vlv_state")
        vent_vlv_state = self.client.read_latest("vent_vlv_state")
        assert (
            press_vlv_state == press_state
        ), f"Press valve state should be {press_state}"
        assert vent_vlv_state == vent_state, f"Vent valve state should be {vent_state}"
