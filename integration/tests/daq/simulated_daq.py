#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import synnax as sy

from framework.test_case import STATUS, TestCase


class Simulated_DAQ(TestCase):
    """
    Simulated DAQ for press control sequence
    """

    def setup(self) -> None:

        self.set_manual_timeout(30)
        super().setup()

    def run(self) -> None:
        """
        Run the test case.
        """
        # self.wait_for_tlm_init()
        client = self.client

        daq_time_ch = client.channels.create(
            name="daq_time", is_index=True, retrieve_if_name_exists=True
        )

        press_pt = client.channels.create(
            name="press_pt",
            index=daq_time_ch.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        press_vlv_cmd_time = client.channels.create(
            name="press_vlv_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        press_vlv_cmd = client.channels.create(
            name="press_vlv_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            index=press_vlv_cmd_time.key,
        )

        press_vlv_state = client.channels.create(
            name="press_vlv_state",
            index=daq_time_ch.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        vent_vlv_cmd_time = client.channels.create(
            name="vent_vlv_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        vent_vlv_cmd = client.channels.create(
            name="vent_vlv_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            index=vent_vlv_cmd_time.key,
        )

        vent_vlv_state = client.channels.create(
            name="vent_vlv_state",
            index=daq_time_ch.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        loop = sy.Loop(sy.Rate.HZ * 100)

        state = {
            "daq_time": sy.TimeStamp.now(),
            "press_vlv_state": 0,
            "vent_vlv_state": 0,
            "press_pt": 0,
        }

        loop = sy.Loop(sy.Rate.HZ * 100)

        state = {
            "daq_time": sy.TimeStamp.now(),
            "press_vlv_state": 0,
            "vent_vlv_state": 0,
            "press_pt": 0,
        }

        sy.sleep(1)
        with client.open_streamer(["press_vlv_cmd", "vent_vlv_cmd"]) as streamer:
            with client.open_writer(
                start=sy.TimeStamp.now(),
                channels=[
                    daq_time_ch.key,
                    "press_vlv_state",
                    "vent_vlv_state",
                    "press_pt",
                ],
                name="Simulated DAQ",
                enable_auto_commit=True,
            ) as writer:

                def test_active() -> bool:
                    return all([loop.wait(), self.should_continue])

                watch_for_end = False
                while test_active():
                    # Read incoming commands
                    frame = streamer.read(timeout=0)
                    if frame is not None:
                        vent_vlv_cmd = frame.get("vent_vlv_cmd")
                        if len(vent_vlv_cmd) > 0:
                            state["vent_vlv_state"] = vent_vlv_cmd[-1]

                        press_vlv_cmd = frame.get("press_vlv_cmd")
                        if len(press_vlv_cmd) > 0:
                            state["press_vlv_state"] = press_vlv_cmd[-1]

                    # Simulate pressure
                    if state["press_vlv_state"] == 1:
                        state["press_pt"] += 0.2

                    elif state["vent_vlv_state"] == 1:
                        if not watch_for_end:
                            watch_for_end = True
                        state["press_pt"] -= 0.2

                    if state["press_pt"] < 0:
                        state["press_pt"] = 0

                    state["press_pt"] += random.uniform(-0.1, 0.1)
                    state["daq_time"] = sy.TimeStamp.now()
                    writer.write(state)

                    value = self.get_value("simple_press_control_state")
                    if (
                        watch_for_end
                        and value is not None
                        and value >= STATUS.PASSED.value
                    ):
                        self._log_message("Controller has stopped. Ending simulation.")
                        break

        if state["press_pt"] > 10:
            self.fail("Pressure was left above 10")
        if state["press_vlv_state"] == 1:
            self.fail("Press valve was left open")
        if state["vent_vlv_state"] == 1:
            self.fail("Vent valve was left open")
