#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""An engine controller that sends CAN frames on the SocketCAN interface vcan0 (Linux
only).

- Fast (``FAST_ID``, standard, 8 bytes, at the simulator's rate): the count as a
  little-endian uint16 in bits 0 to 15, ``sine`` as a little-endian int16 in bits 16
  to 31, ``temperature`` as a little-endian 12-bit signed value in bits 32 to 43, the
  count modulo 16 in bits 44 to 47, and ``speed`` as a big-endian uint16 in bytes 6
  and 7.
- Slow (``SLOW_ID``, extended, 8 bytes, at a fifth of the rate): its own count as a
  little-endian uint32 in bytes 0 to 3 and the count times 0.25 as a little-endian
  float32 in bytes 4 to 7.
- Unknown (``UNKNOWN_ID``, once per second): an identifier no library message has.

The simulator records every frame it receives from other sockets on the bus.
"""

import asyncio
import math
import os
import struct
import sys
from typing import NamedTuple

import can

from examples.simulators.bus_sim import BusSim
from synnax import can as sy_can

CHANNEL = "vcan0"
FAST_ID = 0x100
SLOW_ID = 0x18FF1234
UNKNOWN_ID = 0x7FF
SLOW_DIVISOR = 5


class Frame(NamedTuple):
    """A CAN frame the simulator received."""

    id: int
    extended: bool
    data: bytes


def available() -> bool:
    """Returns True when this host has the vcan0 interface."""
    return sys.platform == "linux" and os.path.isdir(f"/sys/class/net/{CHANNEL}")


def sine(count: int) -> int:
    """Returns the raw sine of the fast frame with the given count, a full turn every
    50 frames at 2^14 counts per unit."""
    return round(16384 * math.sin(2 * math.pi * count / 50))


def temperature(count: int) -> int:
    """Returns the raw 12-bit temperature of the fast frame with the given count, a
    ramp from -200 to 199."""
    return count % 400 - 200


def speed(count: int) -> int:
    """Returns the raw speed of the fast frame with the given count."""
    return count * 3 % 65536


def fast(count: int) -> bytes:
    """Returns the payload of the fast frame with the given count."""
    packed = temperature(count) & 0xFFF | (count % 16) << 12
    return struct.pack("<HhH", count % 65536, sine(count), packed) + struct.pack(
        ">H", speed(count)
    )


def slow(count: int) -> bytes:
    """Returns the payload of the slow frame with the given count."""
    return struct.pack("<If", count, count * 0.25)


class CANFrameSim(BusSim[Frame]):
    """CAN simulator on the SocketCAN interface vcan0."""

    description = "CAN frame simulator on vcan0"
    host = "127.0.0.1"
    # No TCP port: the Driver reaches this simulator through vcan0.
    port = 0
    device_name = "CAN Test Controller"

    async def _run_server(self) -> None:
        bus = can.Bus(interface="socketcan", channel=CHANNEL)

        def receive() -> None:
            while (msg := bus.recv(timeout=0)) is not None:
                self._record(
                    Frame(msg.arbitration_id, msg.is_extended_id, bytes(msg.data))
                )

        def send(count: int) -> None:
            bus.send(can.Message(arbitration_id=FAST_ID, data=fast(count)))
            if count % SLOW_DIVISOR == 0:
                bus.send(
                    can.Message(
                        arbitration_id=SLOW_ID,
                        is_extended_id=True,
                        data=slow(count // SLOW_DIVISOR),
                    )
                )
            if count % int(float(self.rate)) == 0:
                bus.send(can.Message(arbitration_id=UNKNOWN_ID, data=b"\xff" * 8))

        asyncio.get_running_loop().add_reader(bus.fileno(), receive)
        self._ready.set()
        await self._send_at_rate(send)

    @staticmethod
    def create_device(rack_key: int) -> sy_can.Device:
        return sy_can.Device(
            backend="socketcan",
            channel=CHANNEL,
            name=CANFrameSim.device_name,
            location=CHANNEL,
            rack=rack_key,
        )


if __name__ == "__main__":
    asyncio.run(CANFrameSim(verbose=True)._run_server())
