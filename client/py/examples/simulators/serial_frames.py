#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A serial sensor that streams binary frames over a pseudo-terminal (macOS and
Linux).

The simulator links ``SerialFrameSim.path`` to the terminal side of a pty pair, which
the Driver opens as a serial port. Every frame starts with the sync bytes ``SYNC``,
then a one-byte length that counts the bytes after it, checksum included, and ends
with a little-endian CRC-16/MODBUS over every byte before it. Byte 3 is the frame
type:

- Telemetry (``TELEMETRY_TYPE``, 12 bytes, at the simulator's rate): the count as a
  little-endian uint16 at byte 4, ``pressure`` as a little-endian int16 at byte 6,
  and ``flow`` as a big-endian uint16 at byte 8.
- Status (``STATUS_TYPE``, 8 bytes, at a fifth of the rate): its own count modulo 256
  as a uint8 at byte 4 and ``trim`` of that count as an int8 at byte 5.

The simulator records every valid frame the Driver sends, checksum included, and
echoes it back. It stops streaming while more than ``BACKLOG`` bytes wait unread, so
the stream holds only whole frames before the Driver opens the port.
"""

import asyncio
import fcntl
import math
import os
import struct
import termios
import tty

import synnax as sy
from examples.simulators.bus_sim import BusSim
from synnax import serial

SYNC = b"\xaa\x55"
HEADER_SIZE = 3
TELEMETRY_TYPE = 0x01
STATUS_TYPE = 0x02
STATUS_DIVISOR = 5
BACKLOG = 512


def crc16_modbus(data: bytes) -> int:
    """Returns the CRC-16/MODBUS checksum of data."""
    crc = 0xFFFF
    for byte in data:
        crc ^= byte
        for _ in range(8):
            crc = (crc >> 1) ^ 0xA001 if crc & 1 else crc >> 1
    return crc


def frame(body: bytes) -> bytes:
    """Returns the wire frame that carries body, the bytes after the length field."""
    head = SYNC + bytes([len(body) + 2])
    return head + body + struct.pack("<H", crc16_modbus(head + body))


def pressure(count: int) -> int:
    """Returns the raw pressure of the telemetry frame with the given count."""
    return round(2000 * math.sin(2 * math.pi * count / 40))


def flow(count: int) -> int:
    """Returns the raw flow of the telemetry frame with the given count."""
    return count * 7 % 65536


def trim(count: int) -> int:
    """Returns the raw trim of the status frame with the given count byte, from 0 down
    to -49."""
    return -(count % 50)


def telemetry(count: int) -> bytes:
    """Returns the wire frame of the telemetry frame with the given count."""
    return frame(
        struct.pack("<BHh", TELEMETRY_TYPE, count % 65536, pressure(count))
        + struct.pack(">H", flow(count))
    )


def status(count: int) -> bytes:
    """Returns the wire frame of the status frame with the given count."""
    count %= 256
    return frame(struct.pack("<BBb", STATUS_TYPE, count, trim(count)))


def _pending(fd: int) -> int:
    raw = fcntl.ioctl(fd, termios.FIONREAD, b"\0\0\0\0")
    return int(struct.unpack("i", raw)[0])


class _Deframer:
    """Splits the bytes the Driver sends into frames with a valid checksum."""

    def __init__(self) -> None:
        self.buf = bytearray()

    def write(self, chunk: bytes) -> list[bytes]:
        self.buf += chunk
        frames: list[bytes] = []
        while True:
            start = self.buf.find(SYNC)
            if start < 0:
                del self.buf[: max(0, len(self.buf) - len(SYNC) + 1)]
                return frames
            del self.buf[:start]
            if len(self.buf) < HEADER_SIZE:
                return frames
            size = HEADER_SIZE + self.buf[2]
            if size < HEADER_SIZE + 2:
                del self.buf[:1]
                continue
            if len(self.buf) < size:
                return frames
            candidate = bytes(self.buf[:size])
            (checksum,) = struct.unpack("<H", candidate[-2:])
            if crc16_modbus(candidate[:-2]) == checksum:
                frames.append(candidate)
                del self.buf[:size]
            else:
                del self.buf[:1]


class SerialFrameSim(BusSim[bytes]):
    """Serial sensor simulator on a pty linked at /tmp/synnax_serial_sim."""

    description = "Serial frame simulator on a pty at /tmp/synnax_serial_sim"
    host = "127.0.0.1"
    # No TCP port: the Driver reaches this simulator through path.
    port = 0
    path = "/tmp/synnax_serial_sim"
    device_name = "Serial Test Sensor"

    def start(self) -> None:
        self._unlink()
        super().start()

    def stop(self, timeout: sy.TimeSpan = 5 * sy.TimeSpan.SECOND) -> None:
        super().stop(timeout)
        self._unlink()

    def _unlink(self) -> None:
        if os.path.islink(self.path):
            os.unlink(self.path)

    async def _run_server(self) -> None:
        controller, terminal = os.openpty()
        # Raw mode keeps the terminal from echoing streamed frames back before the
        # Driver configures the port. Holding the terminal open keeps reads of the
        # controller from failing while the Driver has the port closed.
        tty.setraw(terminal)
        os.set_blocking(controller, False)
        link = f"{self.path}.{os.getpid()}"
        os.symlink(os.ttyname(terminal), link)
        os.replace(link, self.path)

        deframer = _Deframer()

        def send(data: bytes) -> None:
            if _pending(terminal) + len(data) > BACKLOG:
                return
            try:
                os.write(controller, data)
            except BlockingIOError:
                pass

        def receive() -> None:
            try:
                chunk = os.read(controller, 4096)
            except BlockingIOError:
                return
            for f in deframer.write(chunk):
                self._record(f)
                send(f)

        loop = asyncio.get_running_loop()
        loop.add_reader(controller, receive)
        self._ready.set()
        period = 1 / float(self.rate)
        start = loop.time()
        count = 0
        while True:
            send(telemetry(count))
            if count % STATUS_DIVISOR == 0:
                send(status(count // STATUS_DIVISOR))
            count += 1
            await asyncio.sleep(max(0.0, start + count * period - loop.time()))

    @staticmethod
    def create_device(rack_key: int) -> serial.Device:
        return serial.Device(
            port=SerialFrameSim.path,
            baud_rate=115200,
            name=SerialFrameSim.device_name,
            location=SerialFrameSim.path,
            rack=rack_key,
        )


if __name__ == "__main__":
    asyncio.run(SerialFrameSim(verbose=True)._run_server())
