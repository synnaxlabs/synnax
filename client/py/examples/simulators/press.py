#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import random

import synnax as sy
from examples.simulators.base import SimDAQ


class PressSimDAQ(SimDAQ):
    """Simulates pressurization system with valves and pressure sensor."""

    description = "Run press simulator standalone"
    end_cmd_channel = "end_test_cmd"

    def _create_channels(self) -> None:
        self._log("Creating channels...")
        client = self.client

        self.daq_time_ch = client.channels.create(
            name="daq_time", is_index=True, retrieve_if_name_exists=True
        )

        self.press_pt = client.channels.create(
            name="press_pt",
            index=self.daq_time_ch.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        self.test_flag_cmd = client.channels.create(
            name="test_flag_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            virtual=True,
        )

        self.end_test_cmd = client.channels.create(
            name="end_test_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            virtual=True,
        )

        client.write(sy.TimeStamp.now(), self.test_flag_cmd.key, [0])
        client.write(sy.TimeStamp.now(), self.end_test_cmd.key, [0])

        self.press_vlv_cmd_time = client.channels.create(
            name="press_vlv_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        self.press_vlv_cmd = client.channels.create(
            name="press_vlv_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            index=self.press_vlv_cmd_time.key,
        )

        self.press_vlv_state = client.channels.create(
            name="press_vlv_state",
            index=self.daq_time_ch.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        self.vent_vlv_cmd_time = client.channels.create(
            name="vent_vlv_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        self.vent_vlv_cmd = client.channels.create(
            name="vent_vlv_cmd",
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            index=self.vent_vlv_cmd_time.key,
        )

        self.vent_vlv_state = client.channels.create(
            name="vent_vlv_state",
            index=self.daq_time_ch.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        self.press_setpoint_cmd = client.channels.create(
            name="press_setpoint_cmd",
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
            virtual=True,
        )

        client.write(sy.TimeStamp.now(), self.press_setpoint_cmd.key, [0.0])
        self._log("Channels created successfully")

    def _run_loop(self) -> None:
        self._log("Starting simulation loop...")
        loop = sy.Loop(sy.Rate.HZ * 100)
        loop_count = 0

        state = {
            "daq_time": sy.TimeStamp.now(),
            "press_vlv_state": 0,
            "vent_vlv_state": 0,
            "press_pt": 0.0,
        }

        with self.client.open_streamer(["press_vlv_cmd", "vent_vlv_cmd"]) as streamer:
            with self.client.open_writer(
                start=sy.TimeStamp.now(),
                channels=[
                    self.daq_time_ch.key,
                    "press_vlv_state",
                    "vent_vlv_state",
                    "press_pt",
                ],
                name="Simulated DAQ",
            ) as writer:
                while self._running and loop.wait():
                    while True:
                        frame = streamer.read(timeout=0)
                        if frame is not None:
                            vent_vlv_cmd = frame.get("vent_vlv_cmd")
                            if len(vent_vlv_cmd) > 0:
                                new_val = vent_vlv_cmd[-1]
                                if hasattr(new_val, "item"):
                                    new_val = new_val.item()
                                if new_val != state["vent_vlv_state"]:
                                    self._log(
                                        f"Vent valve: {state['vent_vlv_state']} -> {new_val}"
                                    )
                                state["vent_vlv_state"] = new_val

                            press_vlv_cmd = frame.get("press_vlv_cmd")
                            if len(press_vlv_cmd) > 0:
                                new_val = press_vlv_cmd[-1]
                                if hasattr(new_val, "item"):
                                    new_val = new_val.item()
                                if new_val != state["press_vlv_state"]:
                                    self._log(
                                        f"Press valve: {state['press_vlv_state']} -> {new_val}"
                                    )
                                state["press_vlv_state"] = new_val
                        else:
                            break

                    if state["press_vlv_state"] == 1:
                        state["press_pt"] += 0.2

                    if state["vent_vlv_state"] == 1:
                        state["press_pt"] -= 0.2

                    if state["press_pt"] < 0:
                        state["press_pt"] = 0

                    state["press_pt"] += random.uniform(-0.05, 0.05)
                    state["daq_time"] = sy.TimeStamp.now()
                    writer.write(state)

                    loop_count += 1
                    if loop_count % 100 == 0:
                        self._log(
                            f"press_pt={state['press_pt']:.2f}, "
                            f"press_vlv={state['press_vlv_state']}, "
                            f"vent_vlv={state['vent_vlv_state']}"
                        )

        self._log("Simulation loop stopped")


if __name__ == "__main__":
    PressSimDAQ.main()
