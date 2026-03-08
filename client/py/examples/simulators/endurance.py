#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import math
import random

import synnax as sy
from examples.simulators.simdaq import SimDAQ


class EnduranceSimDAQ(SimDAQ):
    """Simulates generator-drive endurance test system with DC load bank."""

    description = "Run endurance test simulator standalone"
    end_cmd_channel = None

    DRIVE_TAU = 5.0
    LOAD_TAU = 2.0
    DRIVE_NOISE = 5.0
    LOAD_NOISE = 0.5
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
            name="drive_speed_fb",
            index=self.daq_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="Load_Current",
            index=self.daq_time.key,
            data_type=sy.DataType.FLOAT32,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="endurance_test_start",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="shutdown",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="endurance_test_state",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="cycle_count_endurance_good",
            data_type=sy.DataType.UINT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="cycle_count_endurance_bad",
            data_type=sy.DataType.UINT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="dc_lb_load_sp",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="dc_lb_enable_sp",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="batt_contactor_sp",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="dc_contactor_sp",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="gen_field_relay_sp",
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="endurance_desired_speed",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="endurance_desired_load",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="drive_speed_sp",
            data_type=sy.DataType.FLOAT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        client.channels.create(
            name="endurance_test_segment_cycle",
            data_type=sy.DataType.UINT32,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        now = sy.TimeStamp.now()
        client.write(
            now,
            {
                self.daq_time.name: [now],
                "drive_speed_fb": [0.0],
                "Load_Current": [0.0],
            },
        )
        self.log("Channels created successfully")

    def _run_loop(self) -> None:
        self.log("Starting simulation loop...")
        loop = sy.Loop(sy.Rate.HZ * self.SIM_RATE)
        loop_count = 0
        dt = 1.0 / self.SIM_RATE

        drive_speed_fb = 0.0
        load_current = 0.0

        cmd = {
            "drive_speed_sp": 0.0,
            "dc_lb_load_sp": 0.0,
            "dc_lb_enable_sp": 0,
            "dc_contactor_sp": 0,
            "gen_field_relay_sp": 0,
            "batt_contactor_sp": 0,
        }

        stream_channels = [
            "drive_speed_sp",
            "dc_lb_load_sp",
            "dc_lb_enable_sp",
            "dc_contactor_sp",
            "gen_field_relay_sp",
            "batt_contactor_sp",
        ]

        with self.client.open_streamer(stream_channels) as streamer:
            with self.client.open_writer(
                start=sy.TimeStamp.now(),
                channels=[self.daq_time.name, "drive_speed_fb", "Load_Current"],
                name="Endurance Sim DAQ",
            ) as writer:
                while self._running and loop.wait():
                    while True:
                        frame = streamer.read(timeout=0)
                        if frame is not None:
                            for ch in stream_channels:
                                data = frame.get(ch)
                                if len(data) > 0:
                                    val = data[-1]
                                    if hasattr(val, "item"):
                                        val = val.item()
                                    cmd[ch] = val
                        else:
                            break

                    drive_active = (
                        cmd["dc_contactor_sp"] == 1
                        and cmd["gen_field_relay_sp"] == 1
                    )
                    load_active = (
                        cmd["dc_lb_enable_sp"] == 1 and cmd["dc_contactor_sp"] == 1
                    )

                    if drive_active:
                        alpha = 1.0 - math.exp(-dt / self.DRIVE_TAU)
                        drive_speed_fb += (cmd["drive_speed_sp"] - drive_speed_fb) * alpha
                    else:
                        alpha = 1.0 - math.exp(-dt / self.DRIVE_TAU)
                        drive_speed_fb += (0.0 - drive_speed_fb) * alpha

                    if load_active:
                        alpha = 1.0 - math.exp(-dt / self.LOAD_TAU)
                        load_current += (cmd["dc_lb_load_sp"] - load_current) * alpha
                    else:
                        alpha = 1.0 - math.exp(-dt / self.LOAD_TAU)
                        load_current += (0.0 - load_current) * alpha

                    noisy_speed = drive_speed_fb + random.gauss(0, self.DRIVE_NOISE)
                    noisy_current = load_current + random.gauss(0, self.LOAD_NOISE)

                    writer.write(
                        {
                            self.daq_time.name: sy.TimeStamp.now(),
                            "drive_speed_fb": noisy_speed,
                            "Load_Current": noisy_current,
                        }
                    )

                    loop_count += 1
                    if loop_count % (self.SIM_RATE * 2) == 0:
                        self.log(
                            f"speed_sp={cmd['drive_speed_sp']:.0f} "
                            f"speed_fb={drive_speed_fb:.1f} "
                            f"load_sp={cmd['dc_lb_load_sp']:.1f} "
                            f"load_fb={load_current:.1f} "
                            f"contactors=[dc={cmd['dc_contactor_sp']} "
                            f"gen={cmd['gen_field_relay_sp']} "
                            f"batt={cmd['batt_contactor_sp']}]"
                        )

        self.log("Simulation loop stopped")


if __name__ == "__main__":
    EnduranceSimDAQ.main()
