#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A TCP device that sends a reading line at its rate to each client and records the
bytes that its clients send."""

import asyncio
import contextlib
import os
import tempfile
import time
from pathlib import Path
from typing import override
from uuid import uuid4

import synnax as sy
from examples.serial_sim import reading
from examples.simulators.device_sim import DeviceSim
from synnax import tcp


async def run_server(host: str, port: int, rate: float, received: str) -> None:
    """Serve on host and port. Send reading lines at rate to each client, and write the
    bytes that the clients send to the file at received."""
    # Windows opens a file in text mode, which turns each newline into two bytes, unless
    # O_BINARY is set.
    flags = os.O_WRONLY | os.O_CREAT | os.O_TRUNC | getattr(os, "O_BINARY", 0)
    out = os.open(received, flags, 0o600)

    async def serve(reader: asyncio.StreamReader, writer: asyncio.StreamWriter) -> None:
        async def send() -> None:
            start = time.monotonic()
            i = 0
            while True:
                writer.write(reading(i, rate))
                await writer.drain()
                i += 1
                await asyncio.sleep(max(0.0, start + i / rate - time.monotonic()))

        sender = asyncio.create_task(send())
        # A client can reset the connection at any time.
        with contextlib.suppress(ConnectionError):
            while data := await reader.read(4096):
                os.write(out, data)
        sender.cancel()
        with contextlib.suppress(asyncio.CancelledError, ConnectionError):
            await sender
        writer.close()

    server = await asyncio.start_server(serve, host, port)
    async with server:
        await server.serve_forever()


class TCPSim(DeviceSim):
    """A TCP device that sends the line of reading at its rate to each client."""

    description = "TCP device simulator"
    host = "127.0.0.1"
    port = 5030
    received_path = os.path.join(tempfile.gettempdir(), "synnax-tcp-sim.rx")
    device_name = "TCP Sim Device"

    @override
    async def _run_server(self) -> None:
        await run_server(self.host, self.port, float(self.rate), self.received_path)

    @staticmethod
    def received() -> list[str]:
        """Return the lines that the clients have sent, oldest first. A line that has no
        newline yet is left out."""
        return Path(TCPSim.received_path).read_bytes().decode().split("\n")[:-1]

    @override
    def _wait_for_port_closed(self, timeout: sy.TimeSpan) -> None:
        """Wait until the port closes, then remove the file of received bytes."""
        super()._wait_for_port_closed(timeout)
        with contextlib.suppress(FileNotFoundError):
            os.unlink(self.received_path)

    @override
    @staticmethod
    def create_device(client: sy.Synnax, rack_key: int) -> sy.Device:
        return sy.Device(
            key=str(uuid4()),
            rack=rack_key,
            location=f"{TCPSim.host}:{TCPSim.port}",
            make="tcp",
            model="tcp",
            name=TCPSim.device_name,
            configured=True,
            properties=tcp.Properties(host=TCPSim.host, port=TCPSim.port).model_dump(
                mode="json"
            ),
        )
