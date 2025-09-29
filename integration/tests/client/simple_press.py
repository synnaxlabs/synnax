#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from framework.test_case import TestCase


class Simple_Press(TestCase):
    """
    Test a basic press control sequence
    """

    def setup(self) -> None:
        self.set_manual_timeout(30)
        self.subscribe(
            [
                "press_vlv_cmd",
                "vent_vlv_cmd",
                "press_pt",
                "end_test_state",
            ]
        )
        super().setup()

    def run(self) -> None:
        client: sy.Synnax = self.client

        # Define the control channel names
        END_TEST_CMD = "end_test_cmd"
        PRESS_VALVE = "press_vlv_cmd"
        VENT_VALVE = "vent_vlv_cmd"
        PRESSURE = "press_pt"

        with client.control.acquire(
            name="Pressurization Sequence",
            write_authorities=[200],
            write=[PRESS_VALVE, VENT_VALVE, END_TEST_CMD],
            read=[PRESSURE],
        ) as ctrl:

            target_pressure = 20
            ctrl[PRESS_VALVE] = False
            ctrl[VENT_VALVE] = False

            # Pressurize the system
            for i in range(5):
                if self.should_stop:
                    return

                # Open press valve and wait
                ctrl[PRESS_VALVE] = True
                if ctrl.wait_until(
                    (lambda c: c[PRESSURE] > target_pressure),
                    timeout=10 * sy.TimeSpan.SECOND,
                ):
                    self._log_message(
                        f"Target pressure reached: {ctrl[PRESSURE]:.2f} > {target_pressure}"
                    )
                    ctrl[PRESS_VALVE] = False
                    ctrl.sleep(1)
                    target_pressure += 20
                else:
                    self.fail(f"{ctrl[PRESSURE]:.2f} < {target_pressure}")
                    return

            # Depressurize the system
            ctrl[VENT_VALVE] = True
            ctrl.wait_until(
                lambda c: c[PRESSURE] < 5,
                timeout=10 * sy.TimeSpan.SECOND,
            )
            ctrl[VENT_VALVE] = False
            ctrl[END_TEST_CMD] = True
