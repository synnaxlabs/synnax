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
from examples.simulators.simdaq import SimDAQ


class ThermalSimDAQ(SimDAQ):
    """Simulates thermal system with heater and temperature sensor."""

    description = "Run thermal simulator standalone"
    end_cmd_channel: str | None = "end_thermal_test_cmd"

    AMBIENT_TEMP = 25.0
    HEAT_RATE = 1.0
    COOL_RATE = 0.4
    NOISE = 0.1

    def _create_channels(self) -> None:
        self.log("Creating channels...")
        client = self.client

        self.daq_time = client.channels.create(
            name="thermal_daq_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="temp_sensor",
            index=self.daq_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="heater_state",
            index=self.daq_time.key,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
        )

        self.end_test_cmd = client.channels.create(
            name="end_thermal_test_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        self.heater_cmd_time = client.channels.create(
            name="heater_cmd_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="heater_cmd",
            data_type=sy.DataType.UINT8,
            index=self.heater_cmd_time.key,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="force_overheat_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        self.alarm_time = client.channels.create(
            name="alarm_time",
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="alarm_active",
            data_type=sy.DataType.UINT8,
            index=self.alarm_time.key,
            retrieve_if_name_exists=True,
        )

        client.write(sy.TimeStamp.now(), self.end_test_cmd.key, [0])

        now = sy.TimeStamp.now()
        client.write(
            now,
            {
                self.daq_time.name: [now],
                "temp_sensor": [self.AMBIENT_TEMP],
                "heater_state": [0],
            },
        )
        self.log("Channels created successfully")

    def _run_loop(self) -> None:
        self.log("Starting simulation loop...")
        loop = sy.Loop(sy.Rate.HZ * 50)
        loop_count = 0

        temp_sensor = self.AMBIENT_TEMP
        heater_state = 0

        with self.client.open_streamer(
            ["heater_cmd", "force_overheat_cmd"]
        ) as streamer:
            with self.client.open_writer(
                start=sy.TimeStamp.now(),
                channels=[self.daq_time.name, "temp_sensor", "heater_state"],
                name="Thermal Sim DAQ",
            ) as writer:
                force_overheat = False

                while self._running and loop.wait():
                    while True:
                        frame = streamer.read(timeout=0)
                        if frame is not None:
                            heater_cmd_data = frame["heater_cmd"]
                            if len(heater_cmd_data) > 0:
                                new_val = int(heater_cmd_data.to_numpy()[-1])
                                if new_val != heater_state:
                                    self.log(f"Heater: {heater_state} -> {new_val}")
                                heater_state = new_val

                            overheat_cmd_data = frame["force_overheat_cmd"]
                            if len(overheat_cmd_data) > 0:
                                new_overheat = bool(
                                    overheat_cmd_data.to_numpy()[-1] == 1
                                )
                                if new_overheat != force_overheat:
                                    self.log(
                                        f"Force overheat: {force_overheat} -> "
                                        f"{new_overheat}"
                                    )
                                force_overheat = new_overheat
                        else:
                            break

                    if force_overheat or heater_state == 1:
                        temp_sensor += self.HEAT_RATE
                    else:
                        if temp_sensor > self.AMBIENT_TEMP:
                            temp_sensor -= self.COOL_RATE
                        if temp_sensor < self.AMBIENT_TEMP:
                            temp_sensor = self.AMBIENT_TEMP

                    temp_sensor += random.uniform(-self.NOISE, self.NOISE)
                    writer.write(
                        {
                            "thermal_daq_time": sy.TimeStamp.now(),
                            "temp_sensor": temp_sensor,
                            "heater_state": heater_state,
                        }
                    )

                    loop_count += 1
                    if loop_count % 100 == 0:
                        self.log(
                            f"temp={temp_sensor:.2f}°C, "
                            f"heater={heater_state}, "
                            f"overheat={force_overheat}"
                        )

        self.log("Simulation loop stopped")


if __name__ == "__main__":
    ThermalSimDAQ.main()
