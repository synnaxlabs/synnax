#  Copyright 2024 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
import synnax as sy
from synnax.control.controller import Controller

# TPC Control Bound
BOUND = 5  # PSI
TPC_UPPER_BOUND = 50  # PSI
TPC_LOWER_BOUND = TPC_UPPER_BOUND - BOUND

L_STAND_PRESS_TARGET = 65
SCUBA_PRESS_TARGET = 275  # PSI

PRESS_1_STEP = 10  # PSI
PRESS_2_STEP = 50  # PSI

PRESS_STEP_DELAY = (1 * sy.TimeSpan.SECOND).seconds  # Seconds

client = sy.Synnax(
    host="localhost", port=9090, username="synnax", password="seldon", secure=False
)


def run_tpc(auto: Controller):
    pressure = auto[FUEL_TANK_PT]
    one_open = auto[TPC_CMD_ACK]

    if pressure > TPC_UPPER_BOUND:
        if one_open:
            auto[TPC_CMD] = False
    elif pressure < TPC_LOWER_BOUND:
        auto[TPC_CMD] = True

    return pressure < 15


TPC_CMD = "tpc_vlv_cmd"
TPC_CMD_ACK = "tpc_vlv_ack"
MPV_CMD = "mpv_vlv_cmd"
PRESS_ISO_CMD = "press_iso_cmd"
VENT_CMD = "vent_cmd"
PRESS_TANK_PT = "press_tank_pt"
FUEL_TANK_PT = "fuel_tank_pt"

try:
    with client.control.acquire(
        "Autosequence",
        write=[TPC_CMD, MPV_CMD, PRESS_ISO_CMD, VENT_CMD],
        read=[TPC_CMD_ACK, PRESS_TANK_PT, FUEL_TANK_PT],
        write_authorities=[250],
    ) as auto:
        try:
            print("Starting TPC Test. Setting initial system state.")
            auto.set(
                {
                    TPC_CMD: 0,
                    MPV_CMD: 0,
                    PRESS_ISO_CMD: 0,
                    VENT_CMD: 1,
                }
            )

            time.sleep(2)

            print(f"Pressing SCUBA and L-Stand to 50 PSI")

            # Pressurize l-stand and scuba to 50 PSI
            # Open TPC Valve
            auto[TPC_CMD] = True

            dual_press_start = sy.TimeStamp.now()

            curr_target = PRESS_1_STEP
            while True:
                print(f"Pressing L-Stand to {curr_target} PSI")
                auto[PRESS_ISO_CMD] = True
                auto.wait_until(lambda c: c[FUEL_TANK_PT] > curr_target)
                auto[PRESS_ISO_CMD] = False
                curr_target += PRESS_1_STEP
                curr_target = min(curr_target, L_STAND_PRESS_TARGET)
                if auto[FUEL_TANK_PT] > L_STAND_PRESS_TARGET:
                    break
                print("Taking a nap")
                time.sleep(PRESS_STEP_DELAY)

            dual_press_end = sy.TimeStamp.now()
            client.ranges.create(
                name=f"{dual_press_start.__str__()[11:16]} Dual Press Sequence",
                time_range=sy.TimeRange(dual_press_start, dual_press_end),
                # a nice red
                color="#D81E5B",
            )

            press_tank_start = sy.TimeStamp.now()

            print("Pressurized. Waiting for five seconds")
            time.sleep(PRESS_STEP_DELAY)
            # ISO off TESCOM and press scuba with ISO
            auto[TPC_CMD] = False

            curr_target = L_STAND_PRESS_TARGET + PRESS_2_STEP
            while True:
                auto[PRESS_ISO_CMD] = True
                auto.wait_until(lambda c: c[PRESS_TANK_PT] > curr_target)
                auto[PRESS_ISO_CMD] = False
                curr_target += PRESS_2_STEP
                curr_target = min(curr_target, SCUBA_PRESS_TARGET)
                if auto[PRESS_TANK_PT] > SCUBA_PRESS_TARGET:
                    break
                print("Taking a nap")
                time.sleep(PRESS_STEP_DELAY)

            print("Pressurized. Waiting for five seconds")
            time.sleep(2)

            press_tank_end = sy.TimeStamp.now()
            client.ranges.create(
                name=f"{press_tank_start.__str__()[11:16]} Press Tank Pressurization",
                time_range=sy.TimeRange(press_tank_start, press_tank_end),
                # a nice blue
                color="#1E90FF",
            )

            start = sy.TimeStamp.now()

            print("Opening MPV")
            auto[MPV_CMD] = 1
            auto.wait_until(lambda c: run_tpc(c))
            print("Test complete. Safeing System")

            rng = client.ranges.create(
                name=f"{start.__str__()[11:16]} Bang Bang Sim",
                time_range=sy.TimeRange(start, sy.TimeStamp.now()),
                color="#bada55",
            )

            auto.set(
                {
                    TPC_CMD: 1,
                    PRESS_ISO_CMD: 0,
                    # Open vent
                    VENT_CMD: 0,
                    MPV_CMD: 0,
                }
            )
        except KeyboardInterrupt:
            print("Test interrupted. Safeing System")
            auto.set(
                {
                    TPC_CMD: 1,
                    PRESS_ISO_CMD: 0,
                    VENT_CMD: 0,
                    MPV_CMD: 1,
                }
            )
finally:
    time.sleep(100)
