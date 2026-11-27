#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""A serial device on a pseudo-terminal that sends a reading line at its rate."""

import asyncio
import contextlib
import math
import os
import sys
import tempfile
import time
from typing import override
from uuid import uuid4

import synnax as sy
from examples.simulators.device_sim import DeviceSim
from synnax import serial


def reading(i: int, rate: float) -> bytes:
    """Return the line of reading i at rate: a temperature and a pressure with three
    decimals, such as b"21.500,101.300\\n"."""
    t = i / rate
    return f"{20 + 5 * math.sin(t):.3f},{100 + 2 * math.cos(t):.3f}\n".encode()


async def run_server(link: str, rate: float) -> None:
    """Send reading lines at rate on a pseudo-terminal, and point link at the terminal
    that a client opens. A line that the terminal has no room for is dropped."""
    if sys.platform != "win32":
        # tty imports termios, which Windows does not have.
        import tty

        master, slave = os.openpty()
        # Holding the client end keeps the terminal up while no client has it open. Raw
        # mode stops the echo of each line back to the master.
        tty.setraw(slave)
        os.set_blocking(master, False)
        with contextlib.suppress(FileNotFoundError):
            os.unlink(link)
        os.symlink(os.ttyname(slave), link)
        start = time.monotonic()
        i = 0
        while True:
            with contextlib.suppress(BlockingIOError):
                os.write(master, reading(i, rate))
            i += 1
            await asyncio.sleep(max(0.0, start + i / rate - time.monotonic()))
    else:
        raise NotImplementedError("Windows has no pseudo-terminals")


class SerialSim(DeviceSim):
    """A serial device on a pseudo-terminal that sends the line of reading at its rate.
    A client opens serial_port, a link to the terminal. POSIX only."""

    description = "Serial device simulator on a pseudo-terminal"
    serial_port = os.path.join(tempfile.gettempdir(), "synnax-serial-sim")
    device_name = "Serial Sim Device"

    @override
    async def _run_server(self) -> None:
        await run_server(self.serial_port, float(self.rate))

    @override
    def _require_port_free(self) -> None:
        """The server replaces a link left by an earlier run."""

    @override
    def _wait_for_ready(self) -> None:
        """Poll until the link exists."""
        assert self.process is not None, "start() must run before _wait_for_ready()"
        timer = sy.Timer()
        while timer.elapsed() < self.startup_timeout:
            if not self.process.is_alive():
                raise RuntimeError(
                    f"Server process died during startup "
                    f"(exit code: {self.process.exitcode})"
                )
            if os.path.exists(self.serial_port):
                return
            sy.sleep(0.1)
        raise RuntimeError(
            f"Serial simulator link {self.serial_port} missing after "
            f"{self.startup_timeout}"
        )

    @override
    def _wait_for_port_closed(self, timeout: sy.TimeSpan) -> None:
        """The terminal closes with the server. Remove the link it leaves."""
        with contextlib.suppress(FileNotFoundError):
            os.unlink(self.serial_port)

    @override
    @staticmethod
    def create_device(client: sy.Synnax, rack_key: int) -> sy.Device:
        return sy.Device(
            key=str(uuid4()),
            rack=rack_key,
            location=SerialSim.serial_port,
            make="serial",
            model="serial",
            name=SerialSim.device_name,
            configured=True,
            properties=serial.Properties(port=SerialSim.serial_port).model_dump(
                mode="json"
            ),
        )
