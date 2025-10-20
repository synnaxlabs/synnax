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


class ControlAuthority(TestCase):
    """
    Open multiple processes at different control authourities.
    Make them take authority over non-virtual channels ("real" hardware control)
    and command them all at the same time to force conflicts.
    """

    def setup(self) -> None:

        '''
        Note: sim_daq.py creates end_test_state command so
        we don't have to create it in this matrix test.
        '''
        self.set_manual_timeout(300)

        self.control_authority = self.params.get("control_authority", -1)
        self.log(f"Running with control authority: {self.control_authority}")

        self.subscribe(["press_vlv_cmd", "vent_vlv_cmd"])

        super().setup()

    def run(self) -> None:
        client = self.client

        ctrl_valves = ["press_vlv_cmd", "vent_vlv_cmd"]
        read_chans = ["end_test_cmd", "test_flag_cmd"]
        
        with client.control.acquire(
            name = f"Auth {self.control_authority}",
            write = ctrl_valves,
            read = read_chans,
            write_authorities = 1,
        ) as ctrl:
            loop = sy.Loop(sy.Rate.HZ * 100)

            ctrl.wait_until(lambda c: c.get("test_flag_cmd", False) == True)
            ctrl.set_authority(self.control_authority)
            test_flag_prev = True

            while loop.wait() and self.should_continue:

                test_flag = ctrl.get("test_flag_cmd")
                end_test = ctrl.get("end_test_cmd")

                if test_flag != test_flag_prev:
                    if test_flag:
                        ctrl.set_authority(self.control_authority)
                    else:
                        ctrl.set_authority(0)
                    test_flag_prev = test_flag

                if end_test == True:
                    self.log("End signal received")
                    break
