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

from framework.test_case import TestCase


class SimDaq(TestCase):
    """
    Simulated DAQ for press control sequence
    """

    def setup(self) -> None:
        self.set_manual_timeout(120)
        super().setup()

    def run(self) -> None:
        """
        Run the test case.
        """
        client = self.client

        # Index channel:
        # -------------------
        daq_time_ch = client.channels.create(
            name="daq_time", is_index=True, retrieve_if_name_exists=True
        )

        # Pressure channel:
        # -------------------
        press_pt = client.channels.create(
            name="press_pt",
            index=daq_time_ch.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        # Start test channels:
        # -------------------
        start_test_cmd_time = client.channels.create(
            name="start_test_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        start_test_cmd = client.channels.create(
            name="start_test_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            index=start_test_cmd_time.key,
        )

        start_test_state = client.channels.create(
            name="start_test_state",
            index=daq_time_ch.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        # End test channels:
        # -------------------
        end_test_cmd_time = client.channels.create(
            name="end_test_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        end_test_cmd = client.channels.create(
            name="end_test_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            index=end_test_cmd_time.key,
        )

        end_test_state = client.channels.create(
            name="end_test_state",
            index=daq_time_ch.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        # Pres valve channels:
        # -------------------
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

        # Vent valve channels:
        # -------------------
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
            "start_test_state": 0,
            "end_test_state": 0,
            "press_vlv_state": 0,
            "vent_vlv_state": 0,
            "press_pt": 0,
        }

        loop = sy.Loop(sy.Rate.HZ * 100)

        state = {
            "daq_time": sy.TimeStamp.now(),
            "start_test_state": 0,
            "end_test_state": 0,
            "press_vlv_state": 0,
            "vent_vlv_state": 0,
            "press_pt": 0,
        }

        with client.open_streamer(
            ["start_test_cmd", "end_test_cmd", "press_vlv_cmd", "vent_vlv_cmd"]
        ) as streamer:
            with client.open_writer(
                start=sy.TimeStamp.now(),
                channels=[
                    daq_time_ch.key,
                    "start_test_state",
                    "end_test_state",
                    "press_vlv_state",
                    "vent_vlv_state",
                    "press_pt",
                ],
                name="Simulated DAQ",
            ) as writer:

                def test_active() -> bool:
                    return all([loop.wait(), self.should_continue])

                self.log("Sim DAQ running")
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

                        start_test_cmd = frame.get("start_test_cmd")
                        if len(start_test_cmd) > 0:
                            state["start_test_state"] = start_test_cmd[-1]

                        end_test_cmd = frame.get("end_test_cmd")
                        if len(end_test_cmd) > 0:
                            state["end_test_state"] = end_test_cmd[-1]

                    # Simulate pressure
                    if state["press_vlv_state"] == 1:
                        state["press_pt"] += 0.2

                    elif state["vent_vlv_state"] == 1:
                        state["press_pt"] -= 0.2

                    if state["press_pt"] < 0:
                        state["press_pt"] = 0

                    state["press_pt"] += random.uniform(-0.05, 0.05)
                    state["daq_time"] = sy.TimeStamp.now()
                    writer.write(state)

                    # Check for test end
                    if state["end_test_state"] > 0.9:
                        self.log("Controller has stopped. Ending simulation.")
                        break

        assert state["press_pt"] < 10, "Pressure was left above 10"
        assert state["press_vlv_state"] == 0, "Press valve was left open"
        assert state["vent_vlv_state"] == 0, "Vent valve was left open"
