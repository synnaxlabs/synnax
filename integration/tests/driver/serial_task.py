#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Base class for serial task cases."""

import sys

from examples.simulators import serial_frames
from examples.simulators.serial_frames import SerialFrameSim

from synnax import bus
from tests.driver.bus_task import BusCase


class SerialCase(BusCase):
    """Runs a serial task against the serial simulator's pseudo-terminal."""

    sim_classes = [SerialFrameSim]

    @staticmethod
    def framing() -> bus.SyncFraming:
        """Returns the framing of the simulator's frames."""
        return bus.SyncFraming(
            sync=serial_frames.SYNC.hex(),
            length_offset=2,
            length_size=1,
            checksum=bus.CHECKSUM_CRC16_MODBUS,
            checksum_byte_order="little_endian",
        )

    def unsupported(self) -> str | None:
        if sys.platform == "win32":
            return "The serial simulator needs a pseudo-terminal (macOS or Linux)"
        return None
