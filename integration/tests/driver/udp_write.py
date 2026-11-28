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
from tests.driver.bus_task import BusWriteCase


class UDPWrite(BusWriteCase[udp.WriteTask]):
    """Sends a setpoint datagram to the UDP simulator for each command."""

    sim_classes = [UDPSim]
    task_name = "UDP Write"
    channel_prefix = "udp_sim"

    def task(
        self, *, name: str, device: sy.device.Key, targets: list[sy.icd.WriteTarget]
    ) -> udp.WriteTask:
        return udp.WriteTask(name=name, device=device, targets=targets)

    def received(self) -> list[str]:
        return UDPSim.received()
