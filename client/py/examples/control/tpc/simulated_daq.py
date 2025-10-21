#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

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

client = sy.Synnax()

daq_time = client.channels.create(
    name=DAQ_TIME, is_index=True, retrieve_if_name_exists=True
)

for cmd, state in VALVES.items():
    client.channels.create(
        [
            sy.Channel(name=cmd, data_type=sy.DataType.UINT8, virtual=True),
            sy.Channel(name=state, data_type=sy.DataType.UINT8, index=daq_time.key),
        ],
        retrieve_if_name_exists=True,
    )

for sensor in SENSORS:
    s = client.channels.create(
        name=sensor,
        data_type=sy.DataType.FLOAT32,
        index=daq_time.key,
        retrieve_if_name_exists=True,
    )
    print(s.name, s.key)

loop = sy.Loop(sy.Rate.HZ * 3000, precise=True)

DAQ_STATE = {
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

PREV_STATE = DAQ_STATE.copy()


def introduce_randomness(state: dict[str, float]):
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


def clamp_pts(state: dict[str, float]):
    for pt in PTS:
        state[pt] = max(0, state[pt])
    return state


OX_MPV_LAST_OPEN = None
FUEL_MPV_LAST_OPEN = None

with client.open_streamer([cmd for cmd in VALVES.keys()]) as streamer:
    with client.open_writer(
        sy.TimeStamp.now(),
        channels=[*SENSORS, *[state for state in VALVES.values()], DAQ_TIME],
        enable_auto_commit=True,
    ) as writer:
        while loop.wait():
            try:
                while True:
                    frame = streamer.read(0)
                    if frame is None:
                        break
                    for ch in frame.channels:
                        DAQ_STATE[ch] = frame[ch][0]

                if DAQ_STATE[OX_MPV_CMD] == 1 and OX_MPV_LAST_OPEN is None:
                    OX_MPV_LAST_OPEN = sy.TimeStamp.now()
                elif DAQ_STATE[OX_MPV_CMD] == 0:
                    OX_MPV_LAST_OPEN = None

                if DAQ_STATE[FUEL_MPV_CMD] == 1 and FUEL_MPV_LAST_OPEN is None:
                    FUEL_MPV_LAST_OPEN = sy.TimeStamp.now()
                elif DAQ_STATE[FUEL_MPV_CMD] == 0:
                    FUEL_MPV_LAST_OPEN = None

                # If gas booster ISO is open, increase both press tank pts by 1 psi
                if DAQ_STATE[GAS_BOOSTER_ISO_CMD] == 1:
                    DAQ_STATE[PRESS_PT_1] += 1
                    DAQ_STATE[PRESS_PT_2] += 1

                press_pt_1 = DAQ_STATE[PRESS_PT_1]
                if DAQ_STATE[PRESS_ISO_CMD] == 1 and press_pt_1 > 1:
                    if DAQ_STATE[OX_PRESS_CMD] == 1 and DAQ_STATE[OX_PT_1] < press_pt_1:
                        DAQ_STATE[OX_PT_1] += 1
                        DAQ_STATE[OX_PT_2] += 1
                        DAQ_STATE[PRESS_PT_1] -= 1
                        DAQ_STATE[PRESS_PT_2] -= 1
                    if (
                        DAQ_STATE[FUEL_PRESS_CMD] == 1
                        and DAQ_STATE[FUEL_PT_1] < press_pt_1
                    ):
                        DAQ_STATE[FUEL_PT_1] += 1
                        DAQ_STATE[FUEL_PT_2] += 1
                        DAQ_STATE[PRESS_PT_1] -= 1
                        DAQ_STATE[PRESS_PT_2] -= 1

                # If the vents are open
                if DAQ_STATE[OX_VENT_CMD] == 0:
                    DAQ_STATE[OX_PT_1] -= 0.5
                    DAQ_STATE[OX_PT_2] -= 0.5
                if DAQ_STATE[FUEL_VENT_CMD] == 0:
                    DAQ_STATE[FUEL_PT_1] -= 0.5
                    DAQ_STATE[FUEL_PT_2] -= 0.5

                if DAQ_STATE[OX_MPV_CMD] == 1:
                    delta = (
                        0.1 * sy.TimeSpan(sy.TimeStamp.now() - OX_MPV_LAST_OPEN).seconds
                    )
                    DAQ_STATE[OX_PT_1] -= delta
                    DAQ_STATE[OX_PT_2] -= delta

                if DAQ_STATE[FUEL_MPV_CMD] == 1:
                    delta = (
                        0.1
                        * sy.TimeSpan(sy.TimeStamp.now() - FUEL_MPV_LAST_OPEN).seconds
                    )
                    DAQ_STATE[FUEL_PT_1] -= delta
                    DAQ_STATE[FUEL_PT_2] -= delta

                clamped = clamp_pts(DAQ_STATE)
                randomized = introduce_randomness(clamped)
                translated = translate_valves(randomized)
                translated[DAQ_TIME] = sy.TimeStamp.now()
                writer.write(translated)

            except Exception as e:
                print(e)
                raise e
