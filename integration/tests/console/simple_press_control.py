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


class Simple_Press_Control(ConsoleCase):
    """
    Control the pressure of "simulated_daq" test case
    """

    def setup(self) -> None:
        sy.sleep(2)
        self.subscribe(["press_vlv_cmd", "vent_vlv_cmd", "press_pt"])
        self.set_manual_timeout(30)
        super().setup()

    def run(self) -> None:
        console = self.console

        if not self.wait_for_tlm_init():
            self.fail()
            return

        # Define the control channel names
        PRESS_VALVE = "press_vlv_cmd"
        VENT_VALVE = "vent_vlv_cmd"
        PRESSURE = "press_pt"

        self._log_message("Creating plot page")
        console.plot.new()
        console.plot.add_Y("Y1", ["press_vlv_state", "vent_vlv_state"])
        console.plot.add_Y("Y2", ["press_pt"])
        console.plot.add_ranges(["30s"])

        self._log_message("Creating schematic symbols")
        console.schematic.new()
        console.schematic.move("left")
        press_valve = console.schematic.create_setpoint(PRESS_VALVE)
        press_valve.move(-200, 0)

        press_pt = console.schematic.create_value(PRESSURE)
        press_pt.move(0, -3)

        vent_valve = console.schematic.create_setpoint(VENT_VALVE)
        vent_valve.move(200, 0)

        console.schematic.connect_symbols(press_valve, "right", press_pt, "left")
        console.schematic.connect_symbols(press_pt, "right", vent_valve, "left")

        target_Pressure = 20
        press_valve.set_value(0)
        vent_valve.set_value(0)

        for i in range(3):
            self._log_message(f"Target pressure: {target_Pressure}")
            if self.should_stop:
                return

            # Press and wait
            press_valve.set_value(1.0)
            while self.should_continue:
                pressure_value = self.get_value(PRESSURE)
                if pressure_value is not None and pressure_value > target_Pressure:
                    break
            if self.should_stop:
                return

            press_valve.set_value(0)
            sy.sleep(1)
            target_Pressure += 20

        # Depressurize the system
        self._log_message("Venting the system")
        vent_valve.set_value(1)
        while self.should_continue:
            pressure_value = self.get_value(PRESSURE)
            if pressure_value is not None and pressure_value < 5:
                vent_valve.set_value(0)
                self._log_message("System vented")
                self.console.screenshot("console_press_control_passed")
                return

        self.console.screenshot("console_press_control_failed")
        self.fail("Exited without venting")
