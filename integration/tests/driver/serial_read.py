#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import sys

from examples.serial_sim import SerialSim

import synnax as sy
from synnax import serial
from tests.driver.bus_task import BusReadCase


class SerialRead(BusReadCase[serial.ReadTask]):
    """Reads the temperature and pressure of each line that the serial simulator sends,
    stamped when the line arrives."""

    sim_classes = [SerialSim]
    task_name = "Serial Read"
    channel_prefix = "serial_sim"

    def setup(self) -> None:
        if sys.platform == "win32":
            self.skip("Pseudo-terminals need POSIX")
        super().setup()

    def task(
        self, *, name: str, device: sy.device.Key, entries: list[sy.icd.ReadEntry]
    ) -> serial.ReadTask:
        return serial.ReadTask(name=name, device=device, entries=entries)
