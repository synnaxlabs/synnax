import time
import numpy as np

import synnax as sy


from common import (
    SENSORS,
    VALVES,
    PTS,
    DAQ_TIME,
    OX_VENT_CMD,
    OX_TPC_CMD,
    OX_MPV_CMD,
    OX_PRESS_CMD,
    FUEL_VENT_CMD,
    FUEL_TPC_CMD,
    FUEL_MPV_CMD,
    FUEL_PRESS_CMD,
    PRESS_ISO_CMD,
    GAS_BOOSTER_ISO_CMD,
    OX_TC_1,
    OX_TC_2,
    OX_PT_1,
    OX_PT_2,
    FUEL_TC_1,
    FUEL_TC_2,
    FUEL_PT_1,
    FUEL_PT_2,
    PRES_TC_1,
    PRES_TC_2,
    PRESS_PT_1,
    PRESS_PT_2,
    SUPPLY_PT,
    PNEUMATICS_PT,
)

client = sy.Synnax()

daq_time = client.channels.create(
    name=DAQ_TIME,
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

for cmd, state in VALVES.items():
    CMD_TIME = f"{cmd}_time"
    cmd_time = client.channels.create(
        name=CMD_TIME,
        data_type=sy.DataType.TIMESTAMP,
        is_index=True,
        retrieve_if_name_exists=True,
    )
    client.channels.create(
        [
            sy.Channel(name=cmd, data_type=sy.DataType.UINT8, index=cmd_time.key),
            sy.Channel(name=state, data_type=sy.DataType.UINT8, index=daq_time.key),
        ],
        retrieve_if_name_exists=True,
    )

for sensor in SENSORS:
    client.channels.create(
        name=sensor,
        data_type=sy.DataType.FLOAT32,
        index=daq_time.key,
        retrieve_if_name_exists=True,
    )

rate = (sy.Rate.HZ * 30).period.seconds

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
    for a in SENSORS:
        state[a] = (
            state[a]
            + np.random.uniform(-0.1, 0.1)
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
    ) as w:
        i = 0
        while True:
            try:
                time.sleep(rate)
                while True:
                    f = streamer.read(0)
                    if f is None:
                        break
                    for k in f.channels:
                        DAQ_STATE[k] = f[k][0]

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

                # If the ox vent is open
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
                ok = w.write(translated)

            except Exception as e:
                print(e)
                raise e
