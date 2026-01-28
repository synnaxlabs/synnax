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
from console.schematic.schematic import Schematic
from framework.sim_daq_case import SimDaqTestCase


class SetpointPressUser(SimDaqTestCase, ConsoleCase):
    """
    Test the setpoint symbol. A separate case will
    read the setpoints and determine whether to
    open or close the valves.
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
        schematic = Schematic(self.client, self.console, "setpoint_press_user")

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
                state_channel="press_vlv",
                command_channel="press_vlv",
            )
        )
        press_valve.move(delta_x=-90, delta_y=10)
        vent_valve = schematic.create_symbol(
            Valve(
                label="vent_vlv",
                state_channel="vent_vlv",
                command_channel="vent_vlv",
            )
        )
        vent_valve.move(delta_x=90, delta_y=10)
        setpoint = schematic.create_symbol(
            Setpoint(label="press_setpoint_cmd", channel_name="press_setpoint_cmd")
        )
        setpoint.move(delta_x=0, delta_y=120)

        schematic.set_authority(100)
        # ------------- Test 1: Control Authority --------------
        #
        # SY-3147 Fixes a bug where the schematic is locked out
        # control after it posseses control of a channel, writes,
        # and then another processes with a higher authority
        # takes over.
        #
        # A failure means future commands will not be written.

        self.log("Starting Control Authority Test (1/2)")

        # Assertions 1
        start_flag_val = self.read_tlm("test_flag_cmd")
        press_vlv_state = self.read_tlm("press_vlv_state")
        vent_vlv_state = self.read_tlm("vent_vlv_state")
        assert start_flag_val == 0, "Start flag should be 0 on initial read"
        assert press_vlv_state == 0, "Press valve should be 0 on initial read"
        assert vent_vlv_state == 0, "Vent valve should be 0 on initial read"

        start_cmd.press()  # Set True

        # Take absolute control
        press_valve.toggle_absolute_control()
        vent_valve.toggle_absolute_control()

        press_valve.press()  # Set True
        vent_valve.press()  # Set True

        # Wait for states to propagate
        self.assert_states(press_state=1, vent_state=1, start_flag_state=1)

        press_valve.press()  # Set False
        vent_valve.press()  # Set False

        # Release back to higher authority
        press_valve.toggle_absolute_control()
        vent_valve.toggle_absolute_control()

        # Check we can control something again
        start_cmd.press()  # Set False

        # Wait for states to propagate
        self.assert_states(press_state=0, vent_state=0, start_flag_state=0)

        # ------------- Test 2: Basic Control --------------
        self.log("Starting Basic Control Test (2/2)")
        start_cmd.press()  # Set True

        self.log("Starting test")
        setpoints = [25, 0]
        for target in setpoints:
            self.log(f"Target pressure: {target}")
            setpoint.set_value(target)

            target_reached = False
            while self.should_continue:
                pressure_value = self.get_value("press_pt")
                if pressure_value is not None:
                    delta = abs(pressure_value - target)
                    if delta < 0.5:
                        self.log(f"Target pressure reached: {pressure_value:.2f}")
                        sy.sleep(1)
                        target_reached = True
                        break

            if not target_reached:
                self.console.screenshot("setpoint_press_user_failed")
                self.fail("Exiting on timeout.")
                return

        end_cmd.press()
        self.console.screenshot("setpoint_press_user_passed")

    def assert_states(
        self, press_state: int, vent_state: int, start_flag_state: int
    ) -> None:
        """Wait for valve states to match expected values."""
        while self.should_continue:
            press_vlv_state = self.get_value("press_vlv_state")
            vent_vlv_state = self.get_value("vent_vlv_state")
            start_flag_val = self.read_tlm("test_flag_cmd")  # virtual channel
            if (
                press_vlv_state == press_state
                and vent_vlv_state == vent_state
                and start_flag_val == start_flag_state
            ):
                return
        self.fail(
            f"State mismatch: press={press_vlv_state} (expected {press_state}), "
            f"vent={vent_vlv_state} (expected {vent_state}), "
            f"start_flag={start_flag_val} (expected {start_flag_state})"
        )
