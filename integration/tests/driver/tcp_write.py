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
from tests.driver.bus_task import BusWriteCase


class TCPWrite(BusWriteCase[tcp.WriteTask]):
    """Sends a setpoint line to the TCP simulator for each command."""

    sim_classes = [TCPSim]
    task_name = "TCP Write"
    channel_prefix = "tcp_sim"

    def task(
        self, *, name: str, device: sy.device.Key, targets: list[sy.icd.WriteTarget]
    ) -> tcp.WriteTask:
        return tcp.WriteTask(name=name, device=device, targets=targets)

    def received(self) -> list[str]:
        return TCPSim.received()
