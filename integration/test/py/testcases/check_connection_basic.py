#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys
from typing import NamedTuple

import synnax as sy
from integration import FILE_NAME
from framework.TestCase import TestCase, SynnaxConnection

class CheckConnectionBasic(TestCase):
    """
    Check if the test case is connected to the synnax server.
    """
    def __init__(self, SynnaxConnection: SynnaxConnection):
        
        # Always call the parent class constructor first
        # This will initialize the index channel and baseline tlm
        super().__init__(SynnaxConnection=SynnaxConnection)

        # You can then add your own tlm channels here
        self.add_channel(name="is_connected", data_type=sy.DataType.BOOL, initial_value=False)

        print(f"\n\nCheckConnectionBasic > {self.tlm}\n\n")
        

    def run(self) -> None:
        """
        Run the test case.
        """
        self.client.connect()
        self.client.disconnect()
        self.client.connect()
        self.client.disconnect()

    def teardown(self) -> None:
        """
        Teardown the test case.
        """
        self.client.disconnect()