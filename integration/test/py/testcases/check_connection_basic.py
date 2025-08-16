#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
import os
import time

# Set up the path before importing framework modules
sys.path.append(os.path.join(os.path.dirname(__file__), '..'))
from framework.TestCase import TestCase, SynnaxConnection

import synnax as sy

class CheckConnectionBasic(TestCase):
    """
    Check if the test case is connected to the synnax server.
    """
    def __init__(self, SynnaxConnection: SynnaxConnection):

        # Always call the parent class constructor first
        # This will initialize the index channel and baseline tlm
        super().__init__(SynnaxConnection=SynnaxConnection)
        self.Expected_Timeout = 5

        # You can then add your own tlm channels here:
        self.add_channel(name="is_connected", data_type=sy.DataType.UINT8, initial_value=2)
    

    def run(self) -> None:
        """
        Run the test case.
        """

        # Stuff goes here
        time.sleep(10)



        # You might NOT need to call super() here
        super().run()

        

    def teardown(self) -> None:
        """
        Teardown the test case.
        """

        # Stuff goes here

        # Always call super() last
        super().teardown()