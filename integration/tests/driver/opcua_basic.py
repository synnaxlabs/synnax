#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy
from synnax.hardware import opcua

from tests.driver.devices import Simulator
from tests.driver.driver import Driver


class OPCUABasic(Driver):
    """
    Test OPC UA Basic.
    """

    # Class variables defining the OPC UA configuration
    simulator = Simulator.OPCUA

    def run(self) -> None:
        """
        Execute the basic OPC UA test sequence.
        """
        self.log("Starting OPC UA Basic test sequence")
        # Example Test logic will go here
        sy.sleep(5)

        self.log("OPC UA Basic test sequence complete")

