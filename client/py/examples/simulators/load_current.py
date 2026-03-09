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


class LoadCurrentSimDAQ(SimDAQ):
    """Simulates a load current sensor that ramps from 0 upward."""

    description = "Run load current simulator standalone"
    end_cmd_channel = "end_test_cmd"

    RAMP_RATE = 10.0
    NOISE = 0.3
    SIM_RATE = 50

    def _create_channels(self) -> None:
        self.log("Creating channels...")
        client = self.client

        self.daq_time = client.channels.create(
            name="daq_time",
            is_index=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="Load_Current",
            index=self.daq_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="end_test_cmd",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="flag",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        now = sy.TimeStamp.now()
        client.write(
            now,
            {
                self.daq_time.name: [now],
                "Load_Current": [0.0],
            },
        )
        self.log("Channels created successfully")

    def _run_loop(self) -> None:
        self.log("Starting simulation loop...")
        loop = sy.Loop(sy.Rate.HZ * self.SIM_RATE)
        loop_count = 0
        dt = 1.0 / self.SIM_RATE
        load_current = 0.0

        with self.client.open_writer(
            start=sy.TimeStamp.now(),
            channels=[self.daq_time.name, "Load_Current"],
            name="Load Current Sim DAQ",
        ) as writer:
            while self._running and loop.wait():
                load_current += self.RAMP_RATE * dt
                noisy_current = load_current + random.gauss(0, self.NOISE)

                writer.write(
                    {
                        self.daq_time.name: sy.TimeStamp.now(),
                        "Load_Current": noisy_current,
                    }
                )

                loop_count += 1
                if loop_count % (self.SIM_RATE * 2) == 0:
                    self.log(f"Load_Current={load_current:.1f}")

        self.log("Simulation loop stopped")


if __name__ == "__main__":
    LoadCurrentSimDAQ.main()
