#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import synnax as sy

from tests.driver.devices import Simulator
from tests.driver.driver import Driver


class ModbusBasic(Driver):
    """
    Test Modbus TCP Basic.
    """

    simulator = Simulator.MODBUS

    def run(self) -> None:
        """
        Execute the basic Modbus test sequence.
        """
        self.log("Starting Modbus Basic test sequence")

        # Example Test logic will go here
        sy.sleep(5)

        self.log("Modbus Basic test sequence complete")