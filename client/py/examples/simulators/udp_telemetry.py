#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A telemetry unit that sends binary datagrams over UDP.

The simulator binds ``UDPTelemetrySim.port`` and sends datagrams from it to
``DRIVER_PORT``. Each datagram starts with a type byte:

- Fast (``FAST_TYPE``, 13 bytes, at the simulator's rate): the count as a
  little-endian uint32 at byte 1, ``sine`` as a little-endian int16 at byte 5,
  ``temperature`` as a big-endian int16 at byte 7, and the count times 0.25 as a
  little-endian float32 at byte 9.
- Slow (``SLOW_TYPE``, 4 bytes, at a fifth of the rate): its own count as a
  little-endian uint16 at byte 1, the count modulo 8 in bits 24 to 26, and the count
  modulo 2 in bit 27.
- Unknown (``UNKNOWN_DATAGRAM``, once per second): a type no library message has.

The simulator records every datagram it receives.
"""

import asyncio
import math
import struct

from examples.simulators.bus_sim import BusSim
from synnax import udp

FAST_TYPE = 0x01
SLOW_TYPE = 0x02
DRIVER_PORT = 5031
UNKNOWN_DATAGRAM = b"\x7fUNKNOWN"
SLOW_DIVISOR = 5


def sine(count: int) -> int:
    """Returns the raw sine of the fast datagram with the given count, a full turn
    every 50 datagrams at 2^14 counts per unit."""
    return round(16384 * math.sin(2 * math.pi * count / 50))


def temperature(count: int) -> int:
    """Returns the raw temperature of the fast datagram with the given count, a ramp
    from -200 to 199."""
    return count % 400 - 200


def fast(count: int) -> bytes:
    """Returns the fast datagram with the given count."""
    return (
        struct.pack("<BIh", FAST_TYPE, count, sine(count))
        + struct.pack(">h", temperature(count))
        + struct.pack("<f", count * 0.25)
    )


def slow(count: int) -> bytes:
    """Returns the slow datagram with the given count."""
    flags = count % 8 | (count % 2) << 3
    return struct.pack("<BHB", SLOW_TYPE, count % 65536, flags)


class _Receiver(asyncio.DatagramProtocol):
    def __init__(self, sim: "UDPTelemetrySim") -> None:
        self.sim = sim

    def datagram_received(self, data: bytes, addr: tuple[str | int, ...]) -> None:
        self.sim._record(data)

    def error_received(self, exc: Exception) -> None:
        # The Driver's port refuses datagrams until a read task binds it.
        pass


class UDPTelemetrySim(BusSim[bytes]):
    """UDP telemetry simulator on port 5030, sending to the Driver on port 5031."""

    description = "UDP telemetry simulator on port 5030"
    host = "127.0.0.1"
    port = 5030
    device_name = "UDP Test Telemetry Unit"

    async def _run_server(self) -> None:
        loop = asyncio.get_running_loop()
        transport, _ = await loop.create_datagram_endpoint(
            lambda: _Receiver(self), local_addr=(self.host, self.port)
        )
        driver = (self.host, DRIVER_PORT)

        def send(count: int) -> None:
            transport.sendto(fast(count), driver)
            if count % SLOW_DIVISOR == 0:
                transport.sendto(slow(count // SLOW_DIVISOR), driver)
            if count % int(float(self.rate)) == 0:
                transport.sendto(UNKNOWN_DATAGRAM, driver)

        self._ready.set()
        await self._send_at_rate(send)

    @staticmethod
    def create_device(rack_key: int) -> udp.Device:
        return udp.Device(
            port=DRIVER_PORT,
            remote_host=UDPTelemetrySim.host,
            remote_port=UDPTelemetrySim.port,
            name=UDPTelemetrySim.device_name,
            location=f"{UDPTelemetrySim.host}:{DRIVER_PORT}",
            rack=rack_key,
        )


if __name__ == "__main__":
    asyncio.run(UDPTelemetrySim(verbose=True)._run_server())
