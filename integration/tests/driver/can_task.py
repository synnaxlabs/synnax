#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Base class for CAN task cases."""

from examples.simulators import can_frames
from examples.simulators.can_frames import CANFrameSim

from tests.driver.bus_task import BusCase


class CANCase(BusCase):
    """Runs a CAN task against the CAN simulator on vcan0."""

    sim_classes = [CANFrameSim]

    def unsupported(self) -> str | None:
        if not can_frames.available():
            return (
                f"The CAN simulator needs the SocketCAN interface {can_frames.CHANNEL}"
            )
        return None
