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

        console.task.new()
        #console.task.move("left")

        console.task.set_parameters(
            task_name="Test_task",
            sample_rate=100,
            stream_rate=20,
            data_saving=True,
            auto_start=False,
        )

        console.task.add_channel(name="new_channel", type="Voltage", device="USB-6000", dev_name="usb_6000")
        console.task.add_channel(name="hello", type="Accelerometer", device="E103")
        console.task.add_channel(name="goodbye", type="Bridge", device="E103")

        console.task.configure()
        self._log_message("selecting configure")
        sy.sleep(5)

        #console.task.run()
        sy.sleep(15)
