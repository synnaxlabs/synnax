#  Copyright 2025 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import platform
import synnax as sy


from console.case import ConsoleCase

from console.task.task import Task


class Ni_Channel_Types(ConsoleCase):
    """
    Test creating and closing pages
    """

    def setup(self) -> None:
        if platform.system() != "Windows":
           #self.auto_pass(msg="Requires DAQmx drivers")
           print('autopass goes here')
        super().setup()

    def run(self) -> None:
        """
        Test Opening and closing pages
        """
        console = self.console
        self._log_message("Creating NI Analog Read Task page")

        task = Task(console, type="NI Analog Read Task")
        task.add_channel(name="new_channel", type="Voltage", device="E103")
        task.add_channel(name="hello", type="Accelerometer", device="E103")
        task.add_channel(name="goodbye", type="Bridge", device="E103")


