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
sys.path.append(os.path.join(os.path.dirname(__file__), '../..'))
from framework.TestCase import TestCase

import synnax as sy

class Bench_Latency_Report(TestCase):
    """
    Check if the test case is connected to the synnax server.
    """
        

    def setup(self) -> None:
        """
        Setup the test case.
        """

        # Just make sure to call super() last!
        super().setup()

    def run(self) -> None:
        """
        Run the test case.
        """

        # Stuff goes here
        print(f"{self.name} > report is Running")
        time.sleep(3)
        print(f"{self.name} > report is Done")

        # You might NOT need to override
        # ... but then what are you testing?
        super().run()

    def teardown(self) -> None:
        """
        Teardown the test case.
        """

        # Always call super() last
        super().teardown()