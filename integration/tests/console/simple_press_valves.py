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


class Simple_Press_Valves(ConsoleCase):
    """
    Test a basic press control sequence using valves and buttons
    """

    # TODO BEFORE MERGE
    # Update subscribe to leverage wait_on()
    # # Wait for setpoint to be defined by console
    # Example: ctrl.wait_until_defined([PRESS_SETPOINT_CMD])

    def setup(self) -> None:
        sy.sleep(2)
        self.subscribe(
            [
                "start_test_state",
                "end_test_state",
                "press_vlv_cmd",
                "vent_vlv_cmd",
                "press_pt",
            ]
        )
        self.set_manual_timeout(45)
        super().setup()

    def run(self) -> None:
        console = self.console

        if not self.wait_for_tlm_init():
            self.fail()
            return

        # Define the control channel names
        START_CMD = "start_test_cmd"
        END_CMD = "end_test_cmd"
        PRESS_VALVE = "press_vlv_cmd"
        VENT_VALVE = "vent_vlv_cmd"
        PRESSURE = "press_pt"

        self._log_message("Creating plot page")
        console.plot.new()
        console.plot.add_Y(
            "Y1",
            ["start_test_cmd", "end_test_cmd", "start_test_state", "end_test_state"],
        )
        console.plot.add_Y("Y2", ["press_pt"])
        console.plot.add_ranges(["30s"])

        self._log_message("Creating schematic symbols")
        console.schematic.new()
        console.schematic.move("left")

        start_test_cmd = console.schematic.create_button(START_CMD, mode="Momentary")
        start_test_cmd.move(-200, -90)

        end_test_cmd = console.schematic.create_button(END_CMD, mode="Fire")
        end_test_cmd.move(0, -90)

        press_valve = console.schematic.create_valve("press_vlv")
        press_valve.move(-200, 0)

        vent_valve = console.schematic.create_valve("vent_vlv")
        console.schematic.connect_symbols(press_valve, "right", vent_valve, "left")

        self._log_message("Starting test")
        start_test_cmd.press()
        target_Pressure = 20

        for i in range(3):
            self._log_message(f"Target pressure: {target_Pressure}")
            press_valve.press()
            while self.should_continue:
                pressure_value = self.get_value(PRESSURE)
                if pressure_value is not None and pressure_value > target_Pressure:
                    break
                elif self.should_stop:
                    self.fail("Exiting on timeout.")
                    return

            # Configure next cycle
            press_valve.press()
            sy.sleep(1)
            target_Pressure += 20

        # Safe the system
        self._log_message("Venting the system")
        vent_valve.press()
        while self.should_continue:
            pressure_value = self.get_value(PRESSURE)
            if pressure_value is not None and pressure_value < 5:
                vent_valve.press()
                sy.sleep(0.2)
                end_test_cmd.press()
                self.console.screenshot("console_press_control_passed")
                return

        self.console.screenshot("console_press_control_failed")
        self.fail("Exited without venting")
