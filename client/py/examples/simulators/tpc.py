#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""
Simulates a multi-system rocket engine with oxidizer (OX) and fuel tanks,
pressure systems, and multiple valves and sensors.
"""

import numpy as np

import synnax as sy
from examples.control.tpc.common import (
    DAQ_TIME,
    FUEL_MPV_CMD,
    FUEL_PRESS_CMD,
    FUEL_PT_1,
    FUEL_PT_2,
    FUEL_TC_1,
    FUEL_TC_2,
    FUEL_TPC_CMD,
    FUEL_VENT_CMD,
    GAS_BOOSTER_ISO_CMD,
    OX_MPV_CMD,
    OX_PRESS_CMD,
    OX_PT_1,
    OX_PT_2,
    OX_TC_1,
    OX_TC_2,
    OX_TPC_CMD,
    OX_VENT_CMD,
    PNEUMATICS_PT,
    PRES_TC_1,
    PRES_TC_2,
    PRESS_ISO_CMD,
    PRESS_PT_1,
    PRESS_PT_2,
    PTS,
    SENSORS,
    SUPPLY_PT,
    VALVES,
)
from examples.simulators.base import SimDAQ


class TPCSimDAQ(SimDAQ):
    """Simulates multi-system rocket engine with OX/FUEL tanks."""

    description = "Run TPC simulator standalone"
    end_cmd_channel = "end_tpc_test_cmd"

    def _create_channels(self) -> None:
        self._log("Creating channels...")
        client = self.client

        self.daq_time = client.channels.create(
            name=DAQ_TIME, is_index=True, retrieve_if_name_exists=True
        )

        for cmd, state in VALVES.items():
            client.channels.create(
                [
                    sy.Channel(name=cmd, data_type=sy.DataType.UINT8, virtual=True),
                    sy.Channel(
                        name=state, data_type=sy.DataType.UINT8, index=self.daq_time.key
                    ),
                ],
                retrieve_if_name_exists=True,
            )

        for sensor in SENSORS:
            client.channels.create(
                name=sensor,
                data_type=sy.DataType.FLOAT32,
                index=self.daq_time.key,
                retrieve_if_name_exists=True,
            )

        client.channels.create(
            name=self.end_cmd_channel,
            data_type=sy.DataType.UINT8,
            virtual=True,
            retrieve_if_name_exists=True,
        )

        self._log("Channels created successfully")

    def _run_loop(self) -> None:
        self._log("Starting simulation loop...")
        loop = sy.Loop(sy.Rate.HZ * 50, precise=True)
        loop_count = 0

        daq_state = {
            OX_VENT_CMD: 0,
            OX_TPC_CMD: 0,
            OX_MPV_CMD: 0,
            OX_PRESS_CMD: 0,
            FUEL_VENT_CMD: 0,
            FUEL_TPC_CMD: 0,
            FUEL_MPV_CMD: 0,
            FUEL_PRESS_CMD: 0,
            PRESS_ISO_CMD: 0,
            GAS_BOOSTER_ISO_CMD: 0,
            OX_TC_1: -190,
            OX_TC_2: -190,
            OX_PT_1: 1.75,
            OX_PT_2: 1,
            FUEL_TC_1: -190,
            FUEL_TC_2: -190,
            FUEL_PT_1: 2.25,
            FUEL_PT_2: 3.25,
            PRES_TC_1: -190,
            PRES_TC_2: -190,
            PRESS_PT_1: 0,
            PRESS_PT_2: 1.3,
            SUPPLY_PT: 4000,
            PNEUMATICS_PT: 300,
        }

        ox_mpv_last_open = None
        fuel_mpv_last_open = None

        def introduce_randomness(state: dict[str, float]) -> dict[str, float]:
            now = sy.TimeStamp.now()
            for sensor in SENSORS:
                state[sensor] += (
                    np.random.uniform(-0.1, 0.1)
                    + np.sin(now / 1e9 + np.random.uniform(-0.1, 300)) * 0.1
                )
            return state

        def translate_valves(prev_state: dict[str, float]) -> dict[str, float]:
            next_state = dict()
            for sensor in SENSORS:
                next_state[sensor] = prev_state[sensor]
            for cmd, state in VALVES.items():
                next_state[state] = prev_state[cmd]
            return next_state

        def clamp_pts(state: dict[str, float]) -> dict[str, float]:
            for pt in PTS:
                state[pt] = max(0, state[pt])
            return state

        with self.client.open_streamer([cmd for cmd in VALVES.keys()]) as streamer:
            with self.client.open_writer(
                sy.TimeStamp.now(),
                channels=[*SENSORS, *[state for state in VALVES.values()], DAQ_TIME],
                name="TPC Sim DAQ",
            ) as writer:
                while self._running and loop.wait():
                    # Drain all pending frames
                    while True:
                        frame = streamer.read(0)
                        if frame is None:
                            break
                        for ch in frame.channels:
                            daq_state[ch] = frame[ch][0]

                    # Track MPV open times
                    if daq_state[OX_MPV_CMD] == 1 and ox_mpv_last_open is None:
                        ox_mpv_last_open = sy.TimeStamp.now()
                    elif daq_state[OX_MPV_CMD] == 0:
                        ox_mpv_last_open = None

                    if daq_state[FUEL_MPV_CMD] == 1 and fuel_mpv_last_open is None:
                        fuel_mpv_last_open = sy.TimeStamp.now()
                    elif daq_state[FUEL_MPV_CMD] == 0:
                        fuel_mpv_last_open = None

                    # Gas booster increases press tank pressure
                    if daq_state[GAS_BOOSTER_ISO_CMD] == 1:
                        daq_state[PRESS_PT_1] += 1
                        daq_state[PRESS_PT_2] += 1

                    # Pressurization logic
                    press_pt_1 = daq_state[PRESS_PT_1]
                    if daq_state[PRESS_ISO_CMD] == 1 and press_pt_1 > 1:
                        if (
                            daq_state[OX_PRESS_CMD] == 1
                            and daq_state[OX_PT_1] < press_pt_1
                        ):
                            daq_state[OX_PT_1] += 1
                            daq_state[OX_PT_2] += 1
                            daq_state[PRESS_PT_1] -= 1
                            daq_state[PRESS_PT_2] -= 1
                        if (
                            daq_state[FUEL_PRESS_CMD] == 1
                            and daq_state[FUEL_PT_1] < press_pt_1
                        ):
                            daq_state[FUEL_PT_1] += 1
                            daq_state[FUEL_PT_2] += 1
                            daq_state[PRESS_PT_1] -= 1
                            daq_state[PRESS_PT_2] -= 1

                    # Vent logic (note: inverted - 0 means venting)
                    if daq_state[OX_VENT_CMD] == 0:
                        daq_state[OX_PT_1] -= 0.5
                        daq_state[OX_PT_2] -= 0.5
                    if daq_state[FUEL_VENT_CMD] == 0:
                        daq_state[FUEL_PT_1] -= 0.5
                        daq_state[FUEL_PT_2] -= 0.5

                    # MPV consumption
                    if daq_state[OX_MPV_CMD] == 1 and ox_mpv_last_open is not None:
                        delta = (
                            0.1
                            * sy.TimeSpan(
                                sy.TimeStamp.now() - ox_mpv_last_open
                            ).seconds
                        )
                        daq_state[OX_PT_1] -= delta
                        daq_state[OX_PT_2] -= delta

                    if daq_state[FUEL_MPV_CMD] == 1 and fuel_mpv_last_open is not None:
                        delta = (
                            0.1
                            * sy.TimeSpan(
                                sy.TimeStamp.now() - fuel_mpv_last_open
                            ).seconds
                        )
                        daq_state[FUEL_PT_1] -= delta
                        daq_state[FUEL_PT_2] -= delta

                    clamped = clamp_pts(daq_state)
                    randomized = introduce_randomness(clamped)
                    translated = translate_valves(randomized)
                    translated[DAQ_TIME] = sy.TimeStamp.now()
                    writer.write(translated)

                    loop_count += 1
                    if loop_count % 200 == 0:
                        self._log(
                            f"OX_PT_1={daq_state[OX_PT_1]:.2f}, "
                            f"FUEL_PT_1={daq_state[FUEL_PT_1]:.2f}, "
                            f"PRESS_PT_1={daq_state[PRESS_PT_1]:.2f}"
                        )

        self._log("Simulation loop stopped")


if __name__ == "__main__":
    TPCSimDAQ.main()
