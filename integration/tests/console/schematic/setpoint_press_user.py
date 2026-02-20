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
from console.schematic import Button, Setpoint, Valve
from tests.driver.sim_daq_case import SimDaqCase


class SetpointPressUser(SimDaqCase, ConsoleCase):
    """
    Test the setpoint symbol. A separate case will read the setpoints and determine
    whether to open or close the valves.
    """

    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.set_manual_timeout(60)
        self.subscribe(
            [
                "test_flag_cmd",  # virtual channel
                "press_vlv_state",
                "vent_vlv_state",
                "end_test_cmd",
                "press_setpoint_cmd",
                "press_pt",
            ]
        )
        super().setup()

    def run(self) -> None:
        self.log("Creating schematic symbols")
        schematic = self.console.workspace.create_schematic("setpoint_press_user")

        start_cmd = schematic.create_symbol(
            Valve(
                label="test_flag_cmd",
                state_channel="test_flag_cmd",
                command_channel="test_flag_cmd",
            )
        )
        start_cmd.move(delta_x=-90, delta_y=-100)
        end_cmd = schematic.create_symbol(
            Button(label="end_test_cmd", channel_name="end_test_cmd", mode="Fire")
        )
        end_cmd.move(delta_x=90, delta_y=-100)
        press_valve = schematic.create_symbol(
            Valve(
                label="press_vlv",
                state_channel="press_vlv_state",
                command_channel="press_vlv_cmd",
            )
        )
        press_valve.move(delta_x=-90, delta_y=10)
        vent_valve = schematic.create_symbol(
            Valve(
                label="vent_vlv",
                state_channel="vent_vlv_state",
                command_channel="vent_vlv_cmd",
            )
        )
        vent_valve.move(delta_x=90, delta_y=10)
        setpoint = schematic.create_symbol(
            Setpoint(label="press_setpoint_cmd", channel_name="press_setpoint_cmd")
        )
        setpoint.move(delta_x=0, delta_y=120)

        schematic.set_authority(100)

        self.log("Starting Control Authority Test (1/2)")
        self.wait_for_eq("test_flag_cmd", 0, is_virtual=True)
        self.wait_for_eq("press_vlv_state", 0)
        self.wait_for_eq("vent_vlv_state", 0)

        start_cmd.press()

        press_valve.toggle_absolute_control()
        vent_valve.toggle_absolute_control()

        press_valve.press()
        vent_valve.press()

        self.wait_for_eq("press_vlv_state", 1)
        self.wait_for_eq("vent_vlv_state", 1)
        self.wait_for_eq("test_flag_cmd", 1, is_virtual=True)

        press_valve.press()
        vent_valve.press()

        press_valve.toggle_absolute_control()
        vent_valve.toggle_absolute_control()

        start_cmd.press()

        self.wait_for_eq("press_vlv_state", 0)
        self.wait_for_eq("vent_vlv_state", 0)
        self.wait_for_eq("test_flag_cmd", 0, is_virtual=True)

        self.log("Starting Basic Control Test (2/2)")
        start_cmd.press()
        self.wait_for_eq("test_flag_cmd", 1, is_virtual=True)

        self.log("Starting test")
        setpoints = [25, 0]
        for target in setpoints:
            self.log(f"Target pressure: {target}")
            setpoint.set_value(target)
            self.wait_for_eq("press_setpoint_cmd", target, is_virtual=True)
            self.wait_for_near("press_pt", target, tolerance=3.0)
            self.log("Target pressure reached")
            sy.sleep(1)

        end_cmd.press()
        self.console.screenshot("setpoint_press_user_passed")
