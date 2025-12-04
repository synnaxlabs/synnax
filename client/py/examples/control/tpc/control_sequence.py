#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import dataclasses
import time

import numpy as np
from scipy.signal import find_peaks

import synnax as sy

client = sy.Synnax()

from examples.control.tpc.common import (
    GAS_BOOSTER_ISO_CMD,
    OX_MPV_CMD,
    OX_PRESS_CMD,
    OX_PRESS_STATE,
    OX_PT_1,
    OX_VENT_CMD,
    PRESS_ISO_CMD,
    PRESS_PT_1,
)


@dataclasses.dataclass
class TPCParameters:
    l_stand_press_target: int
    scuba_press_target: int
    press_1_step: int
    press_2_step: int
    press_step_delay: float
    tpc_upper_bound: int
    tpc_lower_bound: int


TPC_CMD = OX_PRESS_CMD
TPC_CMD_ACK = OX_PRESS_STATE
MPV_CMD = OX_MPV_CMD
SUPPLY_CMD = GAS_BOOSTER_ISO_CMD
VENT_CMD = OX_VENT_CMD
PRESS_TANK_PT = PRESS_PT_1
FUEL_TANK_PT = OX_PT_1
START_SIM_CMD = "start_sim_cmd"

sim_cmd_time = client.channels.create(
    name=f"{START_SIM_CMD}_time",
    data_type=sy.DataType.TIMESTAMP,
    is_index=True,
    retrieve_if_name_exists=True,
)

sim_cmd = client.channels.create(
    name=START_SIM_CMD,
    data_type=sy.DataType.UINT8,
    index=sim_cmd_time.key,
    retrieve_if_name_exists=True,
)

AUTO_LOGS = "auto_logs"

auto_logs = client.channels.create(
    name=AUTO_LOGS,
    data_type=sy.DataType.STRING,
    virtual=True,
    retrieve_if_name_exists=True,
)


def start_sim_cmd(aut: sy.Controller):
    return sim_cmd.key in aut.state and aut[START_SIM_CMD] == 1


def log(aut: sy.Controller, msg: str):
    aut.set(
        AUTO_LOGS,
        f"TPC  {sy.TimeStamp.now().datetime().strftime('%H:%M:%S.%f')}  {msg}",
    )


def execute_auto(params: TPCParameters, wait_for_confirm: bool = False) -> sy.Range:
    def run_tpc(auto: sy.Controller):
        pressure = auto[FUEL_TANK_PT]
        one_open = auto[TPC_CMD_ACK]
        if pressure > params.tpc_upper_bound:
            if one_open:
                auto[TPC_CMD] = False
                log(auto, "TPC Valve Closed")
        elif pressure < params.tpc_lower_bound:
            if not one_open:
                auto[TPC_CMD] = True
                log(auto, "TPC Valve Open")
        return pressure < 15

    with client.control.acquire(
        "Autosequence",
        write=[TPC_CMD, MPV_CMD, SUPPLY_CMD, VENT_CMD, PRESS_ISO_CMD, AUTO_LOGS],
        read=[TPC_CMD_ACK, PRESS_TANK_PT, FUEL_TANK_PT, START_SIM_CMD],
        write_authorities=[250],
    ) as ctrl:
        if wait_for_confirm:
            print("Waiting for confirmation to start test")
            log(ctrl, "Waiting for confirmation to start test")
            ctrl.wait_until(start_sim_cmd)
        try:
            parent_rng = client.ranges.create(
                name="TPC Test",
                time_range=sy.TimeRange(sy.TimeStamp.now(), sy.TimeStamp.now()),
            )
            log(ctrl, "Starting TPC Test. Setting initial system state.")
            ctrl.set(
                {
                    TPC_CMD: 0,
                    MPV_CMD: 0,
                    SUPPLY_CMD: 0,
                    VENT_CMD: 1,
                }
            )

            ctrl.sleep(2)

            log(ctrl, f"Pressing SCUBA and L-Stand to 50 PSI")

            # Pressurize l-stand and scuba to 50 PSI
            # Open TPC Valve
            ctrl[TPC_CMD] = True
            ctrl[PRESS_ISO_CMD] = True

            dual_press_start = sy.TimeStamp.now()

            curr_target = params.press_1_step
            while True:
                log(ctrl, f"Pressing L-Stand to {curr_target} PSI")
                ctrl[SUPPLY_CMD] = True
                ctrl.wait_until(lambda c: c[FUEL_TANK_PT] > curr_target)
                ctrl[SUPPLY_CMD] = False
                curr_target += params.press_1_step
                curr_target = min(curr_target, params.l_stand_press_target)
                if ctrl[FUEL_TANK_PT] > params.l_stand_press_target:
                    break
                log(
                    ctrl,
                    f"Holding at {curr_target} PSI for {params.press_step_delay} seconds",
                )
                ctrl.sleep(params.press_step_delay)

            dual_press_end = sy.TimeStamp.now()
            parent_rng.create_child_range(
                name=f"Setup",
                time_range=sy.TimeRange(dual_press_start, dual_press_end),
                color="#D81E5B",
            )

            press_tank_start = sy.TimeStamp.now()

            log(ctrl, "L-Stand Pressurized. Waiting for five seconds")
            ctrl.sleep(params.press_step_delay)
            # ISO off TESCOM and press scuba with ISO
            ctrl[TPC_CMD] = False
            ctrl[PRESS_ISO_CMD] = False
            ctrl[SUPPLY_CMD] = False

            curr_target = params.l_stand_press_target + params.press_2_step
            while True:
                log(ctrl, f"Pressing Press Tank to {curr_target} PSI")
                ctrl[SUPPLY_CMD] = True
                ctrl.wait_until(lambda c: c[PRESS_TANK_PT] > curr_target)
                ctrl[SUPPLY_CMD] = False
                curr_target += params.press_2_step
                curr_target = min(curr_target, params.scuba_press_target)
                if ctrl[PRESS_TANK_PT] > params.scuba_press_target:
                    break
                log(
                    ctrl,
                    f"Holding at {curr_target} PSI for {params.press_step_delay} seconds",
                )
                ctrl.sleep(params.press_step_delay)

            log(ctrl, "Pressurized. Waiting for five seconds")
            ctrl.sleep(2)

            press_tank_end = sy.TimeStamp.now()
            parent_rng.create_child_range(
                name=f"Pressurization",
                time_range=sy.TimeRange(press_tank_start, press_tank_end),
                color="#1E90FF",
            )

            start = sy.TimeStamp.now()

            log(ctrl, "Opening MPV")
            ctrl[PRESS_ISO_CMD] = True
            ctrl[MPV_CMD] = True
            ctrl.wait_until(lambda c: run_tpc(c))
            log(ctrl, "Test complete. Safeing System")

            rng = parent_rng.create_child_range(
                name=f"Test",
                time_range=sy.TimeRange(start, sy.TimeStamp.now()),
                color="#bada55",
            )
            rng.meta_data.set(
                {
                    "l_stand_press_target": f"{params.l_stand_press_target} PSI",
                    "scuba_press_target": f"{params.scuba_press_target} PSI",
                    "press_1_step": f"{params.press_1_step} PSI",
                    "press_2_step": f"{params.press_2_step} PSI",
                    "press_step_delay": f"{params.press_step_delay} seconds",
                    "tpc_upper_bound": f"{params.tpc_upper_bound} PSI",
                    "tpc_lower_bound": f"{params.tpc_lower_bound} PSI",
                }
            )

            ctrl.set(
                {
                    TPC_CMD: 1,
                    SUPPLY_CMD: 0,
                    # Open vent
                    VENT_CMD: 0,
                    MPV_CMD: 0,
                }
            )

            return rng

        except KeyboardInterrupt:
            log(ctrl, "Test interrupted. Safeing System")
            ctrl.set(
                {
                    TPC_CMD: 1,
                    SUPPLY_CMD: 0,
                    VENT_CMD: 0,
                    MPV_CMD: 1,
                }
            )


def perform_analysis(params: TPCParameters, rng: sy.Range) -> TPCParameters:
    print("Performing analysis on the test results. Starting with a 5 second sleep")
    time.sleep(5)
    fuel_pt = rng[FUEL_TANK_PT].to_numpy()
    peaks, _ = find_peaks(fuel_pt, height=params.tpc_upper_bound)
    avg_diff = np.mean(fuel_pt[peaks] - params.tpc_upper_bound)
    rng.meta_data.set("overshoot_avg", f"{avg_diff} PSI")
    tpc_upper_bound = params.tpc_upper_bound - avg_diff
    return TPCParameters(
        l_stand_press_target=params.l_stand_press_target,
        scuba_press_target=params.scuba_press_target,
        press_1_step=params.press_1_step,
        press_2_step=params.press_2_step,
        press_step_delay=params.press_step_delay,
        tpc_upper_bound=tpc_upper_bound,
        tpc_lower_bound=params.tpc_lower_bound,
    )


if __name__ == "__main__":
    initial_params = TPCParameters(
        l_stand_press_target=65,
        scuba_press_target=275,
        press_1_step=20,
        press_2_step=50,
        press_step_delay=1,
        tpc_upper_bound=50,
        tpc_lower_bound=45,
    )
    res = execute_auto(initial_params, wait_for_confirm=True)
    next_params = perform_analysis(initial_params, res)
    res = execute_auto(next_params)
    next_params.tpc_upper_bound = initial_params.tpc_upper_bound
    perform_analysis(next_params, res)
