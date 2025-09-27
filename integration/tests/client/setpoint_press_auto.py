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
        sy.sleep(2)
        self.subscribe(
            [
                "press_vlv_cmd",
                "vent_vlv_cmd",
                "press_pt",
                "start_test_state",
                "end_test_state",
            ]
        )
        self.set_manual_timeout(120)
        super().setup()

    def run(self) -> None:
        client: sy.Synnax = self.client

        # Create Press Setpoint channel for console to write to:
        # -------------------

        press_setpoint_time = client.channels.create(
            name="press_setpoint_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        press_setpoint_state = client.channels.create(
            name="press_setpoint_state",
            index=press_setpoint_time.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )
        press_setpoint_cmd_time = client.channels.create(
            name="press_setpoint_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        press_setpoint_cmd = client.channels.create(
            name="press_setpoint_cmd",
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
            index=press_setpoint_cmd_time.key,
        )

        # Wait for sim DAQ channels
        if not self.wait_for_tlm_init():
            self.fail()
            return

        print("Acquiring control")
        with client.control.acquire(
            name="Pressurization Sequence",
            write_authorities=[200],
            write=[
                "press_vlv_cmd",
                "vent_vlv_cmd",
                "press_setpoint_state",
                "press_setpoint_time",
            ],
            read=["press_pt", "press_setpoint_cmd", "end_test_state"],
        ) as ctrl:
            print("Control acquired")
            loop = sy.Loop(sy.Rate.HZ * 100)
            state = {
                "daq_time": sy.TimeStamp.now(),
                "press_setpoint_state": 0,
            }

            def test_active() -> bool:
                return all([loop.wait(), self.should_continue])

            # Initialize valves to closed
            print("Setting initial state")
            ctrl.set(
                {
                    "press_vlv_cmd": 0,
                    "vent_vlv_cmd": 0,
                    "press_setpoint_state": 0,
                    "press_setpoint_time": sy.TimeStamp.now(),
                }
            )

            mode = "hold"
            setpoint_prev = None
            ctrl.wait_until_defined(["press_setpoint_cmd"])
            while test_active():

                setpoint = ctrl["press_setpoint_cmd"]
                pressure = ctrl["press_pt"]
                end_test_state = ctrl["end_test_state"]
                ctrl["press_setpoint_state"] = setpoint

                # Update on a new value
                if setpoint != setpoint_prev:
                    setpoint_prev = setpoint
                import time

                if mode == "hold":
                    if pressure - setpoint > 2:
                        mode = "vent"
                        ctrl["vent_vlv_cmd"] = 1
                    elif setpoint - pressure > 2:
                        mode = "press"
                        ctrl["press_vlv_cmd"] = 1

                elif mode == "press" and pressure > setpoint:
                    mode = "hold"
                    ctrl["press_vlv_cmd"] = 0

                elif mode == "vent" and pressure < setpoint:
                    mode = "hold"
                    ctrl["vent_vlv_cmd"] = 0

                # Check for test end
                if end_test_state > 0.9:
                    print("Test ended")
                    ctrl["press_vlv_cmd"] = 0
                    ctrl["vent_vlv_cmd"] = 0
                    return

        self.fail("Test failed on timeout")
