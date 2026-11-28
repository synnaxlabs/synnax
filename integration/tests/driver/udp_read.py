#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from examples.udp_sim import UDPSim

import synnax as sy
from synnax import udp
from tests.driver.bus_task import BusReadCase


class UDPRead(BusReadCase[udp.ReadTask]):
    """Reads the temperature and pressure of each datagram that the UDP simulator sends,
    stamped when the datagram arrives."""

    sim_classes = [UDPSim]
    task_name = "UDP Read"
    channel_prefix = "udp_sim"

    def task(
        self, *, name: str, device: sy.device.Key, entries: list[sy.icd.ReadEntry]
    ) -> udp.ReadTask:
        return udp.ReadTask(name=name, device=device, entries=entries)
