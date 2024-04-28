import dataclasses
import time
import synnax as sy
from synnax.control.controller import Controller
import numpy as np
from scipy.signal import find_peaks

BOUND = 5  # PSI
TPC_UPPER_BOUND = 50  # PSI
TPC_LOWER_BOUND = TPC_UPPER_BOUND - BOUND
L_STAND_PRESS_TARGET = 65
SCUBA_PRESS_TARGET = 275  # PSI
PRESS_1_STEP = 20  # PSI
PRESS_2_STEP = 50  # PSI
PRESS_STEP_DELAY = (1 * sy.TimeSpan.SECOND).seconds  # Seconds


@dataclasses.dataclass
class TPCParameters:
    l_stand_press_target: int
    scuba_press_target: int
    press_1_step: int
    press_2_step: int
    press_step_delay: float
    tpc_upper_bound: int
    tpc_lower_bound: int


client = sy.Synnax(
    host="localhost", port=9090, username="synnax", password="seldon", secure=False
)

TPC_CMD = "tpc_vlv_cmd"
TPC_CMD_ACK = "tpc_vlv_ack"
MPV_CMD = "mpv_vlv_cmd"
PRESS_ISO_CMD = "press_iso_cmd"
VENT_CMD = "vent_cmd"
PRESS_TANK_PT = "press_tank_pt"
FUEL_TANK_PT = "fuel_tank_pt"


def execute_auto(params: TPCParameters):
    def run_tpc(auto: Controller):
        pressure = auto[FUEL_TANK_PT]
        one_open = auto[TPC_CMD_ACK]
        if pressure > params.tpc_upper_bound:
            if one_open:
                auto[TPC_CMD] = False
        elif pressure < params.tpc_lower_bound:
            auto[TPC_CMD] = True
        return pressure < 15

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

            curr_target = params.press_1_step
            while True:
                print(f"Pressing L-Stand to {curr_target} PSI")
                auto[PRESS_ISO_CMD] = True
                auto.wait_until(lambda c: c[FUEL_TANK_PT] > curr_target)
                auto[PRESS_ISO_CMD] = False
                curr_target += params.press_1_step
                curr_target = min(curr_target, params.l_stand_press_target)
                if auto[FUEL_TANK_PT] > params.l_stand_press_target:
                    break
                print("Taking a nap")
                time.sleep(params.press_step_delay)

            dual_press_end = sy.TimeStamp.now()
            client.ranges.create(
                name=f"{dual_press_start.__str__()[11:16]} Dual Press Sequence",
                time_range=sy.TimeRange(dual_press_start, dual_press_end),
                # a nice red
                color="#D81E5B"
            )

            press_tank_start = sy.TimeStamp.now()

            print("Pressurized. Waiting for five seconds")
            time.sleep(params.press_step_delay)
            # ISO off TESCOM and press scuba with ISO
            auto[TPC_CMD] = False

            curr_target = params.l_stand_press_target + params.press_2_step
            while True:
                auto[PRESS_ISO_CMD] = True
                auto.wait_until(lambda c: c[PRESS_TANK_PT] > curr_target)
                auto[PRESS_ISO_CMD] = False
                curr_target += params.press_2_step
                curr_target = min(curr_target, params.scuba_press_target)
                if auto[PRESS_TANK_PT] > params.scuba_press_target:
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
                color="#1E90FF"
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

            return rng

        except KeyboardInterrupt:
            print("Test interrupted. Safeing System")
            auto.set({
                TPC_CMD: 1,
                PRESS_ISO_CMD: 0,
                VENT_CMD: 0,
                MPV_CMD: 1,
            })


def perform_analysis(params: TPCParameters, rng: sy.Range) -> TPCParameters:
    print("Performing analysis on the test results. Starting with a 5 second sleep")
    time.sleep(5)
    fuel_pt = rng[FUEL_TANK_PT].to_numpy()
    print(fuel_pt)
    peaks, _ = find_peaks(fuel_pt, height=params.tpc_upper_bound)
    print(f"Found {len(peaks)} peaks")
    # get the average amount the peaks are off by
    avg_diff = np.mean(fuel_pt[peaks] - params.tpc_upper_bound)
    print(f"Average difference: {avg_diff}")
    # subtract the average difference from the target
    params.tpc_upper_bound -= avg_diff
    return params


if __name__ == "__main__":
    initial_params = TPCParameters(
        L_STAND_PRESS_TARGET,
        SCUBA_PRESS_TARGET,
        PRESS_1_STEP,
        PRESS_2_STEP,
        PRESS_STEP_DELAY,
        TPC_UPPER_BOUND,
        TPC_LOWER_BOUND,
    )
    print("HERLLO")
    res = execute_auto(initial_params)
    next_params = perform_analysis(initial_params, res)
    execute_auto(next_params)
