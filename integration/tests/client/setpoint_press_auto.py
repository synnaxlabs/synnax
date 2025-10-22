#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from framework.test_case import TestCase


class Setpoint_Press_Auto(TestCase):
    """
    Reads a setpoint and opens or closes a valve based on the value.
    """

    def setup(self) -> None:

        self.set_manual_timeout(120)
        self.control_authority = self.params.get("control_authority", -1)

        self.subscribe(
            [
                "press_vlv_cmd",
                "vent_vlv_cmd",
                "press_pt",
                "end_test_cmd",
                "test_flag_cmd",
            ]
        )
        super().setup()

    def run(self) -> None:
        client: sy.Synnax = self.client

        # Create Press Setpoint channel for console to write to:
        # -------------------

        press_setpoint_cmd = client.channels.create(
            name="press_setpoint_cmd",
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
            virtual=True,
        )

        ctrl_valves = ["press_vlv_cmd", "vent_vlv_cmd"]
        read_chans = ["press_pt", "press_setpoint_cmd", "test_flag_cmd", "end_test_cmd"]
        with client.control.acquire(
            name="Pressurization Automation",
            write_authorities=1,  # Start low
            write=ctrl_valves,
            read=read_chans,
        ) as ctrl:
            # Define loop and stop conditions
            loop = sy.Loop(sy.Rate.HZ * 100)

            def test_active() -> bool:
                return all([loop.wait(), self.should_continue])

            ctrl.wait_until(lambda c: c.get("test_flag_cmd", False) == True)
            ctrl.set_authority(self.control_authority)  # Set high

            # Initialize valves to closed
            ctrl.set({"press_vlv_cmd": 0, "vent_vlv_cmd": 0})
            if not ctrl.wait_until_defined(
                ["press_pt", "press_setpoint_cmd"], timeout=60
            ):
                self.fail("Timeout (60s) for press_pt and press_setpoint_cmd")
                return

            self.log("Starting pressurization logic")
            mode = "hold"
            setpoint_prev = None

            while test_active():
                end_test_cmd = ctrl.get("end_test_cmd", False)
                setpoint = ctrl["press_setpoint_cmd"]
                pressure = ctrl["press_pt"]

                # Update on a new value
                if setpoint != setpoint_prev:
                    setpoint_prev = setpoint
                    self.log(f"Setpoint changed to {setpoint:.2f}")

                if mode == "hold":
                    if pressure - setpoint > 2:
                        self.log("Venting")
                        mode = "vent"
                        ctrl["vent_vlv_cmd"] = True
                    elif setpoint - pressure > 2:
                        self.log("Pressing")
                        mode = "press"
                        ctrl["press_vlv_cmd"] = True

                elif mode == "press" and pressure > setpoint:
                    self.log("Holding")
                    mode = "hold"
                    ctrl["press_vlv_cmd"] = False

                elif mode == "vent" and pressure < setpoint:
                    self.log("Holding")
                    mode = "hold"
                    ctrl["vent_vlv_cmd"] = 0

                # Check for test end
                if end_test_cmd == True:
                    self.log("End signal received")
                    ctrl["press_vlv_cmd"] = False
                    ctrl["vent_vlv_cmd"] = False
                    return

        self.fail("Test failed on timeout")
