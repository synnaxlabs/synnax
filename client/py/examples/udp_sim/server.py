#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A UDP device that sends a reading datagram at its rate to its peer and records the
datagrams that it receives."""

import asyncio
import contextlib
import os
import socket
import tempfile
import time
from pathlib import Path
from typing import override
from uuid import uuid4

import synnax as sy
from examples.serial_sim import reading
from examples.simulators.device_sim import DeviceSim
from synnax import udp


async def run_server(
    local: tuple[str, int], peer: tuple[str, int], rate: float, received: str
) -> None:
    """Bind local, send reading datagrams at rate to peer, and write the datagrams that
    it receives to the file at received. A datagram that no socket gets is dropped."""
    sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
    sock.bind(local)
    sock.setblocking(False)
    # Windows opens a file in text mode, which turns each newline into two bytes, unless
    # O_BINARY is set.
    flags = os.O_WRONLY | os.O_CREAT | os.O_TRUNC | getattr(os, "O_BINARY", 0)
    out = os.open(received, flags, 0o644)
    start = time.monotonic()
    i = 0
    while True:
        # Windows reports a datagram that reached no socket as a reset on the next call.
        # The client binds its port only while its task runs.
        with contextlib.suppress(ConnectionResetError):
            sock.sendto(reading(i, rate), peer)
        with contextlib.suppress(BlockingIOError):
            while True:
                with contextlib.suppress(ConnectionResetError):
                    os.write(out, sock.recv(65535))
        i += 1
        await asyncio.sleep(max(0.0, start + i / rate - time.monotonic()))


class UDPSim(DeviceSim):
    """A UDP device on port that sends the datagram of reading at its rate to
    client_port, the port that the device of the client binds."""

    description = "UDP device simulator"
    host = "127.0.0.1"
    port = 5031
    client_port = 5032
    received_path = os.path.join(tempfile.gettempdir(), "synnax-udp-sim.rx")
    device_name = "UDP Sim Device"

    @override
    async def _run_server(self) -> None:
        await run_server(
            (self.host, self.port),
            (self.host, self.client_port),
            float(self.rate),
            self.received_path,
        )

    @staticmethod
    def received() -> list[str]:
        """Return the lines that the client has sent, oldest first. A line that has no
        newline yet is left out."""
        return Path(UDPSim.received_path).read_bytes().decode().split("\n")[:-1]

    def _port_free(self) -> bool:
        """Return true when no socket holds port."""
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
            try:
                s.bind((self.host, self.port))
            except OSError:
                return False
        return True

    @override
    def _require_port_free(self) -> None:
        if not self._port_free():
            raise RuntimeError(f"UDP port {self.port} is already in use")

    @override
    def _wait_for_ready(self) -> None:
        """Poll until the server holds its port."""
        assert self.process is not None, "start() must run before _wait_for_ready()"
        timer = sy.Timer()
        while timer.elapsed() < self.startup_timeout:
            if not self.process.is_alive():
                raise RuntimeError(
                    f"Server process died during startup "
                    f"(exit code: {self.process.exitcode})"
                )
            if not self._port_free():
                return
            sy.sleep(0.1)
        raise RuntimeError(
            f"UDP simulator not on port {self.port} after {self.startup_timeout}"
        )

    @override
    def _wait_for_port_closed(self, timeout: sy.TimeSpan) -> None:
        """Poll until the port is free, then remove the file of received bytes."""
        timer = sy.Timer()
        while not self._port_free():
            if timer.elapsed() >= timeout:
                raise RuntimeError(f"UDP port {self.port} still held after {timeout}")
            sy.sleep(0.1)
        with contextlib.suppress(FileNotFoundError):
            os.unlink(self.received_path)

    @override
    @staticmethod
    def create_device(client: sy.Synnax, rack_key: int) -> sy.Device:
        return sy.Device(
            key=str(uuid4()),
            rack=rack_key,
            location=f"{UDPSim.host}:{UDPSim.port}",
            make="udp",
            model="udp",
            name=UDPSim.device_name,
            configured=True,
            properties=udp.Properties(
                local_address=UDPSim.host,
                local_port=UDPSim.client_port,
                remote=udp.Remote(host=UDPSim.host, port=UDPSim.port),
            ).model_dump(mode="json"),
        )
