#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from examples.simulators import PressSimDAQ

from framework.test_case import TestCase


class SimplePress(TestCase):
    """
    Test a basic press control sequence
    """

    sim_daq_class = PressSimDAQ

    def setup(self) -> None:
        self.set_manual_timeout(30)
        self.subscribe(
            [
                "end_test_cmd",
                "press_vlv_cmd",
                "press_vlv_state",
                "vent_vlv_cmd",
                "vent_vlv_state",
                "press_pt",
            ]
        )
        super().setup()

    def run(self) -> None:
        client: sy.Synnax = self.client

        # Define the control channel names
        END_TEST_CMD = "end_test_cmd"
        PRESS_VALVE_CMD = "press_vlv_cmd"
        PRESS_VALVE_STATE = "press_vlv_state"
        VENT_VALVE_CMD = "vent_vlv_cmd"
        VENT_VALVE_STATE = "vent_vlv_state"
        PRESSURE = "press_pt"

        with client.control.acquire(
            name="Pressurization Sequence",
            write_authorities=[200],
            write=[PRESS_VALVE_CMD, VENT_VALVE_CMD, END_TEST_CMD],
            read=[PRESSURE, PRESS_VALVE_STATE, VENT_VALVE_STATE],
        ) as ctrl:
            # Wait for initial state to be received before proceeding
            ctrl.wait_until_defined(
                [PRESSURE, PRESS_VALVE_STATE, VENT_VALVE_STATE],
                timeout=5 * sy.TimeSpan.SECOND,
            )

            target_pressure = 30
            ctrl[PRESS_VALVE_CMD] = False
            ctrl[VENT_VALVE_CMD] = False

            # Pressurize the system
            for i in range(4):
                if self.should_stop:
                    return
                # Open press valve and wait
                ctrl[PRESS_VALVE_CMD] = 1
                ctrl.wait_until(
                    lambda c: c[PRESS_VALVE_STATE] and not c[VENT_VALVE_STATE],
                    timeout=3 * sy.TimeSpan.SECOND,
                )
                if ctrl.wait_until(
                    lambda c: c[PRESSURE] > target_pressure,
                    timeout=10 * sy.TimeSpan.SECOND,
                ):
                    self.log(
                        f"Target pressure reached: {ctrl[PRESSURE]:.2f} > {target_pressure}"
                    )
                    ctrl[PRESS_VALVE_CMD] = 0
                    ctrl.wait_until(
                        lambda c: not c[PRESS_VALVE_STATE] and not c[VENT_VALVE_STATE],
                        timeout=3 * sy.TimeSpan.SECOND,
                    )
                    target_pressure += 30
                    sy.sleep(1)  # Give "Bad Actor" time to run
                else:
                    self.fail(f"{ctrl[PRESSURE]:.2f} < {target_pressure}")
                    return

            # Depressurize the system
            ctrl[VENT_VALVE_CMD] = 1
            ctrl.wait_until(
                lambda c: not c[PRESS_VALVE_STATE] and c[VENT_VALVE_STATE],
                timeout=3 * sy.TimeSpan.SECOND,
            )
            ctrl.wait_until(
                lambda c: c[PRESSURE] < 5,
                timeout=10 * sy.TimeSpan.SECOND,
            )
            ctrl[VENT_VALVE_CMD] = 0
            ctrl.wait_until(
                lambda c: not c[PRESS_VALVE_STATE] and not c[VENT_VALVE_STATE],
                timeout=3 * sy.TimeSpan.SECOND,
            )
            ctrl[END_TEST_CMD] = 1
