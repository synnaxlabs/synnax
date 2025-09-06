#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import sys
import time

import synnax as sy

from framework.test_case import SynnaxConnection, TestCase


class CheckConnectionBasic(TestCase):
    """
    Check if the test case is connected to the synnax server.
    """

    def setup(self) -> None:
        """
        Setup the test case.
        """

        # You can then add your own tlm channels here:
        self.add_channel(
            name="is_connected", data_type=sy.DataType.UINT32, initial_value=1
        )

        # Or explcitiely change the time out
        self.Expected_Timeout = 6

        # Or change it via test parameters
        self.Expected_Timeout = self.params.get("timeout", -1)

        # Just make sure to call super() last!
        super().setup()

    def run(self) -> None:
        """
        Run the test case.
        """

        # Stuff goes here
        wait_time = self.params.get("wait_time", 0)
        time.sleep(wait_time)

        # Or induce a failure
        if self.params.get("fail_test", False):
            raise Exception("Injected failure")

        # You might NOT need to override
        # ... but then what are you testing?
        super().run()

    def teardown(self) -> None:
        """
        Teardown the test case.
        """

        # Always call super() last
        super().teardown()
