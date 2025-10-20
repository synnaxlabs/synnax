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
    Reads a setpoint and opens or closes a valve based on the value.
    """

    def setup(self) -> None:

        '''
        Note: sim_daq.py creates end_test_state command so
        we don't have to create it in this matrix test.
        '''
        self.set_manual_timeout(300)

        # Get control authority from test parameters
        self.control_authority = self.params.get("control_authority", 0)
        self.log(f"Running with control authority: {self.control_authority}")

        # Create unique channel names based on control authority
        self.ctrl_index_name = f"ctrl_index_{self.control_authority}"
        self.ctrl_chan_name = f"ctrl_chan_{self.control_authority}"

        self.subscribe(
            [
                "press_pt",
                "end_test_state",
                "test_flag_state",
            ]
        )
        super().setup()

    def run(self) -> None:
        client: sy.Synnax = self.client

        # Create index channel for command
        ctrl_index_ch = client.channels.create(
            name=self.ctrl_index_name,
            is_index=True,
            data_type=sy.DataType.TIMESTAMP,
            retrieve_if_name_exists=True,
        )
        # Create command channel
        client.channels.create(
            name=self.ctrl_chan_name,
            data_type=sy.DataType.UINT8,
            retrieve_if_name_exists=True,
            index=ctrl_index_ch.key,
        )

        with client.control.acquire(
            name=f"Authority {self.control_authority}",
            write_authorities=[self.control_authority],
            write=[
                self.ctrl_chan_name,
            ],
            read=["press_pt", "end_test_state", "test_flag_state"],
        ) as ctrl:
            loop = sy.Loop(sy.Rate.HZ * 500)

            def test_active() -> bool:
                return all([loop.wait(), self.should_continue])

            if not ctrl.wait_until_defined(
                # If press_pt is live, then the test is active
                ["press_pt"], timeout=15
            ):
                self.fail("Timeout (15s) for press_pt")
                return

            test_flag_state_prev = None
            while test_active():
                end_test_state = ctrl["end_test_state"]
                test_flag_state = ctrl["test_flag_state"]

                if test_flag_state != test_flag_state_prev:
                    if test_flag_state > 0.9:
                        ctrl.set_authority(self.control_authority)
                        ctrl[self.ctrl_chan_name] = 1
                    else:
                        ctrl[self.ctrl_chan_name] = 0
                        ctrl.set_authority(0)
                test_flag_state_prev = test_flag_state

                # Check for test end
                if end_test_state > 0.9:
                    self.log("End signal received")
                    ctrl[self.ctrl_chan_name] = 0
                    return

