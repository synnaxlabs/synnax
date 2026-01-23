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

from framework.test_case import TestCase


class ThermalSimDaq(TestCase):
    """Simulated thermal system for Arc thermal monitor test."""

    AMBIENT_TEMP = 25.0
    HEAT_RATE = 0.5
    COOL_RATE = 0.2
    NOISE = 0.1

    def setup(self) -> None:
        self.set_manual_timeout(120)
        super().setup()

    def run(self) -> None:
        client = self.client

        daq_time = client.channels.create(
            name="thermal_daq_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="temp_sensor",
            index=daq_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="heater_state",
            index=daq_time.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        end_test_cmd = client.channels.create(
            name="end_thermal_test_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        heater_cmd_time = client.channels.create(
            name="heater_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="heater_cmd",
            data_type=sy.DataType.UINT8,
            index=heater_cmd_time.key,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="force_overheat_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        alarm_time = client.channels.create(
            name="alarm_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="alarm_active",
            data_type=sy.DataType.UINT8,
            index=alarm_time.key,
            retrieve_if_name_exists=True,
        )

        client.write(sy.TimeStamp.now(), end_test_cmd.key, [0])

        loop = sy.Loop(sy.Rate.HZ * 100)

        state = {
            "thermal_daq_time": sy.TimeStamp.now(),
            "temp_sensor": self.AMBIENT_TEMP,
            "heater_state": 0,
        }

        with client.open_streamer(
            ["end_thermal_test_cmd", "heater_cmd", "force_overheat_cmd"]
        ) as streamer:
            with client.open_writer(
                start=sy.TimeStamp.now(),
                channels=[daq_time.key, "temp_sensor", "heater_state"],
                name="Thermal Sim DAQ",
            ) as writer:

                def test_active() -> bool:
                    return all([loop.wait(), self.should_continue])

                self.log("Thermal Sim DAQ running at 100Hz")
                self.log(f"Initial temp: {state['temp_sensor']}")
                end_test_flag = 0
                force_overheat = False
                log_counter = 0
                frames_received = 0

                while test_active():
                    frame = streamer.read(timeout=0)

                    if frame is not None:
                        frames_received += 1
                        heater_cmd_data = frame.get("heater_cmd")
                        if len(heater_cmd_data) > 0:
                            new_state = heater_cmd_data[-1].item()
                            if new_state != state["heater_state"]:
                                self.log(
                                    f"Heater cmd: {state['heater_state']} -> {new_state}"
                                )
                            state["heater_state"] = new_state

                        end_cmd_data = frame.get("end_thermal_test_cmd")
                        if len(end_cmd_data) > 0:
                            end_test_flag = end_cmd_data[-1].item()

                        overheat_cmd_data = frame.get("force_overheat_cmd")
                        if len(overheat_cmd_data) > 0:
                            new_overheat = overheat_cmd_data[-1].item() == 1
                            if new_overheat and not force_overheat:
                                self.log(
                                    "FORCE OVERHEAT: Ignoring heater commands, heating continuously"
                                )
                            force_overheat = new_overheat

                    if force_overheat or state["heater_state"] == 1:
                        state["temp_sensor"] += self.HEAT_RATE
                    else:
                        if state["temp_sensor"] > self.AMBIENT_TEMP:
                            state["temp_sensor"] -= self.COOL_RATE
                        if state["temp_sensor"] < self.AMBIENT_TEMP:
                            state["temp_sensor"] = self.AMBIENT_TEMP

                    state["temp_sensor"] += random.uniform(-self.NOISE, self.NOISE)
                    state["thermal_daq_time"] = sy.TimeStamp.now()
                    writer.write(state)

                    log_counter += 1
                    if log_counter % 500 == 0:
                        self.log(
                            f"heater={state['heater_state']} temp={state['temp_sensor']:.1f} frames={frames_received}"
                        )

                    if end_test_flag != 0:
                        self.log("Test ended. Stopping thermal simulation.")
                        break

        if state["heater_state"] != 0:
            self.log("WARNING: Heater was left on (test stopped mid-cycle)")
