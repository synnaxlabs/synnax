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


class Setpoint_Press_User(ConsoleCase):
    """
    Test the setpoint symbol. A separate case will
    read the setpoints and determine whether to
    open or close the valves.
    """

    def setup(self) -> None:
        sy.sleep(2)
        self.subscribe(
            [
                "start_test_state",
                "end_test_state",
                "press_setpoint_state",
                "press_pt",
            ]
        )
        self.set_manual_timeout(45)
        super().setup()

    def run(self) -> None:
        console = self.console

        sy.sleep(5)
        if not self.wait_for_tlm_init():
            self.fail()
            return

        # Define the control channel names
        START_CMD = "start_test_cmd"
        END_CMD = "end_test_cmd"
        SETPOINT = "press_setpoint_cmd"
        PRESSURE = "press_pt"

        self._log_message("Creating plot page")
        console.plot.new()
        console.plot.add_Y(
            "Y1",
            ["press_vlv_state", "vent_vlv_state"],
        )
        console.plot.add_Y("Y2", ["press_pt", "press_setpoint_state"])
        console.plot.add_ranges(["30s"])

        self._log_message("Creating schematic symbols")
        console.schematic.new()
        console.schematic.move("left")

        start_cmd = console.schematic.create_button(START_CMD, mode="Momentary")
        start_cmd.move(-200, -90)
        end_cmd = console.schematic.create_button(END_CMD, mode="Fire")
        end_cmd.move(0, -90)

        # For display ONLY
        # press_vlv = console.schematic.create_valve("press_vlv")
        # press_vlv.move(-200, 0)
        # vent_vlv = console.schematic.create_valve("vent_vlv")
        # vent_vlv.move(200, 0)

        # Setpoint control
        setpoint = console.schematic.create_setpoint(SETPOINT)

        self._log_message("Starting test")
        sy.sleep(1)
        start_cmd.press()

        setpoints = [30, 15, 60, 30, 0]
        for target in setpoints:
            self._log_message(f"Target pressure: {target}")
            setpoint.set_value(target)

            while self.should_continue:
                pressure_value = self.get_value(PRESSURE)
                if pressure_value is not None:
                    delta = abs(pressure_value - target)
                    if delta < 0.5:
                        self._log_message(f"Target pressure reached: {pressure_value}.2f")
                        break

                if self.should_stop:
                    self.console.screenshot("setpoint_press_user_failed")
                    self.fail("Exiting on timeout.")
                    return

        end_cmd.press()
        self.console.screenshot("setpoint_press_user_passed")
