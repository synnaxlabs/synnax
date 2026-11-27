#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.tcp_sim import TCPSim

import synnax as sy
from synnax import tcp
from tests.driver.bus_task import BusReadCase


class TCPRead(BusReadCase[tcp.ReadTask]):
    """Reads the temperature and pressure of each line that the TCP simulator sends,
    stamped when the line arrives."""

    sim_classes = [TCPSim]
    task_name = "TCP Read"
    channel_prefix = "tcp_sim"

    def task(
        self, *, name: str, device: sy.device.Key, entries: list[sy.icd.ReadEntry]
    ) -> tcp.ReadTask:
        return tcp.ReadTask(name=name, device=device, entries=entries)
