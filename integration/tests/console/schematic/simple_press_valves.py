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

from console.case import ConsoleCase
from console.schematic import Button, Valve
from console.schematic.schematic import Schematic
from framework.sim_daq_case import SimDaqTestCase


class SimplePressValves(SimDaqTestCase, ConsoleCase):
    """
    Test a basic press control sequence using valves and buttons
    """

    sim_daq_class = PressSimDAQ

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
        schematic = self.console.workspace.create_schematic("simple_press_valves")
        schematic.move("left")

        end_test_cmd = schematic.create_symbol(
            Button(label=END_CMD, channel_name=END_CMD)
        )
        end_test_cmd.move(delta_x=0, delta_y=-90)

        press_valve = schematic.create_symbol(
            Valve(
                label="press_vlv",
                state_channel="press_vlv",
                command_channel="press_vlv",
            )
        )
        press_valve.move(delta_x=-200, delta_y=0)

        vent_valve = schematic.create_symbol(
            Valve(
                label="vent_vlv",
                state_channel="vent_vlv",
                command_channel="vent_vlv",
            )
        )
        schematic.connect_symbols(press_valve, "right", vent_valve, "left")

        self.log("Starting test")
        schematic.acquire_control()
        target_Pressure = 20

        for _ in range(2):
            self.log(f"Target pressure: {target_Pressure}")
            press_valve.press()
            self.wait_for_eq("press_vlv_state", 1)
            self.wait_for_eq("vent_vlv_state", 0)
            self.wait_for_ge(PRESSURE, target_Pressure)

            # Configure next cycle
            self.log("Closing press valve")
            press_valve.press()
            self.wait_for_eq("press_vlv_state", 0)
            self.wait_for_eq("vent_vlv_state", 0)
            target_Pressure += 20

        # Safe the system
        self.log("Venting the system")
        vent_valve.press()
        self.wait_for_eq("press_vlv_state", 0)
        self.wait_for_eq("vent_vlv_state", 1)
        self.wait_for_le(PRESSURE, 5)
        self.log("Closing vent valve")
        vent_valve.press()
        self.wait_for_eq("press_vlv_state", 0)
        self.wait_for_eq("vent_vlv_state", 0)
        end_test_cmd.press()
        self.console.screenshot("console_press_control_passed")
