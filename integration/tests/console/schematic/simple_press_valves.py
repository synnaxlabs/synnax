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


class SimplePressValves(ConsoleCase):
    """
    Test a basic press control sequence using valves and buttons
    """

    def setup(self) -> None:
        self.set_manual_timeout(90)
        self.subscribe(
            [
                "end_test_cmd",
                "end_test_state",
                "press_vlv_cmd",
                "press_vlv_state",
                "vent_vlv_cmd",
                "vent_vlv_state",
                "press_pt",
            ]
        )
        sy.sleep(1)
        super().setup()

    def run(self) -> None:
        console = self.console

        # Define the control channel names
        END_CMD = "end_test_cmd"
        PRESSURE = "press_pt"

        self._log_message("Creating plot page")
        console.plot.new()
        console.plot.add_Y(
            "Y1",
            ["press_vlv_state", "vent_vlv_state"],
        )
        console.plot.add_Y("Y2", ["press_pt"])
        console.plot.add_ranges(["30s"])

        self._log_message("Creating schematic symbols")
        console.schematic.new()
        console.schematic.move("left")

        end_test_cmd = console.schematic.create_button(END_CMD)
        end_test_cmd.move(0, -90)

        press_valve = console.schematic.create_valve("press_vlv")
        press_valve.move(-200, 0)

        vent_valve = console.schematic.create_valve("vent_vlv")
        console.schematic.connect_symbols(press_valve, "right", vent_valve, "left")

        self._log_message("Starting test")
        target_Pressure = 20

        for i in range(3):
            self._log_message(f"Target pressure: {target_Pressure}")
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
            self._log_message("Closing press valve")
            press_valve.press()
            self.assert_states(press_state=0, vent_state=0)
            target_Pressure += 20

        # Safe the system
        self._log_message("Venting the system")
        vent_valve.press()
        self.assert_states(press_state=0, vent_state=1)
        while self.should_continue:
            pressure_value = self.get_value(PRESSURE)
            if pressure_value is not None and pressure_value < 5:
                self._log_message("Closing vent valve")
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
