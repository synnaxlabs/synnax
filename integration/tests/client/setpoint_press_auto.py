#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from framework.test_case import TestCase


class SetpointPressAuto(TestCase):
    """
    Reads a setpoint and opens or closes a valve based on the value.
    """

    def setup(self) -> None:
        self.set_manual_timeout(60)
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

        # ------------- Test 1: Control Authority --------------
        # SY-3147: Pair this test with setpoint_press_user.py
        # to ensure that the schematic control authority is working.
        # Use read-only streaming for Test 1 to avoid opening any writer
        # that could conflict with the schematic's writes.
        with client.open_streamer(["test_flag_cmd"]) as streamer:
            # Wait for test_flag_cmd == 1 (user starts Test 1)
            while self.should_continue:
                frame = streamer.read(timeout=0.1)
                if frame is not None and "test_flag_cmd" in frame:
                    if frame["test_flag_cmd"][-1] == 1:
                        break
            # Wait for test_flag_cmd == 0 (user completes Test 1)
            while self.should_continue:
                frame = streamer.read(timeout=0.1)
                if frame is not None and "test_flag_cmd" in frame:
                    if frame["test_flag_cmd"][-1] == 0:
                        break

        # ------------- Test 2: Basic Control --------------
        # setpoint_press_user will control setpoint via the console schematic.
        # Only open the writer after Test 1 completes to avoid timestamp conflicts.
        with client.control.acquire(
            name="Pressurization Automation",
            write_authorities=self.control_authority,
            write=ctrl_valves,
            read=read_chans,
        ) as ctrl:
            # Define loop and stop conditions
            loop = sy.Loop(sy.Rate.HZ * 100)

            def test_active() -> bool:
                return all([loop.wait(), self.should_continue])

            ctrl.wait_until(lambda c: c.get("test_flag_cmd", 0) == 1, timeout=3)

            # Initialize valves to closed
            ctrl.set({"press_vlv_cmd": 0, "vent_vlv_cmd": 0})
            if not ctrl.wait_until_defined(
                ["press_pt", "press_setpoint_cmd"], timeout=10
            ):
                self.fail("Timeout (60s) for press_pt and press_setpoint_cmd")
                return

            self.log("Starting pressurization logic")
            mode = "hold"
            setpoint_prev = None

            while test_active():
                end_test_cmd = ctrl.get("end_test_cmd", 0)
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
                        ctrl["vent_vlv_cmd"] = 1
                    elif setpoint - pressure > 2:
                        self.log("Pressing")
                        mode = "press"
                        ctrl["press_vlv_cmd"] = 1

                elif mode == "press" and pressure > setpoint:
                    self.log("Holding")
                    mode = "hold"
                    ctrl["press_vlv_cmd"] = 0

                elif mode == "vent" and pressure < setpoint:
                    self.log("Holding")
                    mode = "hold"
                    ctrl["vent_vlv_cmd"] = 0

                # Check for test end
                if end_test_cmd == 1:
                    self.log("End signal received")
                    ctrl["press_vlv_cmd"] = 0
                    ctrl["vent_vlv_cmd"] = 0
                    return

        self.fail("Test failed on timeout")
