#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import multiprocessing
import os
import signal
import socket
import sys
from abc import abstractmethod
from multiprocessing.process import BaseProcess

import synnax as sy
from examples.simulators.base import Simulator
from synnax.device import Device as SynnaxDevice


class DeviceSim(Simulator):
    """Base class for protocol device simulators.

    These simulators expose network endpoints (Modbus TCP, OPC UA, etc.)
    but do NOT interact with Synnax. They run async servers in a separate
    process to avoid event loop conflicts.

    Subclasses must implement:
    - _run_server(): async coroutine that starts and runs the server

    Class attributes:
    - host: network host the server binds to
    - port: TCP port the server listens on (used for readiness check)
    - startup_timeout: max seconds to wait for the server to accept connections
    - device_name: name used for Synnax device registration
    """

    host: str
    port: int
    startup_timeout: float = 10.0
    device_name: str

    def __init__(self, verbose: bool = False):
        super().__init__(verbose=verbose)
        self._process: BaseProcess | None = None

    def start(self) -> None:
        """Start the device server in a subprocess.

        Blocks until the server is accepting TCP connections on its port,
        or raises RuntimeError if the process dies or the timeout expires.
        """
        self._process = multiprocessing.Process(
            target=self._subprocess_entry, daemon=True
        )
        self._process.start()
        self._running = True
        self._log(f"Server started with PID: {self._process.pid}")
        self._wait_for_ready()

    def _wait_for_ready(self) -> None:
        """Poll until the server is accepting connections on its port."""
        timeout = self.startup_timeout * sy.TimeSpan.SECOND
        timer = sy.Timer()
        while timer.elapsed() < timeout:
            if not self._process.is_alive():
                raise RuntimeError(
                    f"Server process died during startup "
                    f"(exit code: {self._process.exitcode})"
                )
            try:
                with socket.create_connection((self.host, self.port), timeout=0.5):
                    self._log(f"Server ready on {self.host}:{self.port}")
                    return
            except OSError:
                sy.sleep(0.1)
        raise RuntimeError(
            f"Server not ready on {self.host}:{self.port} "
            f"after {self.startup_timeout}s"
        )

    def stop(self, timeout: float = 5.0) -> None:
        """Terminate the server subprocess."""
        self._running = False
        if self._process is None:
            return
        if not self._process.is_alive():
            self._process = None
            return
        try:
            self._process.terminate()
            self._process.join(timeout=timeout)
            if self._process.is_alive():
                self._process.kill()
                self._process.join(timeout=2)
        except Exception as e:
            self._log(f"Error terminating server: {e}")
        finally:
            self._process = None
            sy.sleep(1)

    @property
    def process(self) -> BaseProcess | None:
        return self._process

    def _subprocess_entry(self) -> None:
        """Entry point for the subprocess."""
        devnull_fd = os.open(os.devnull, os.O_WRONLY)
        os.dup2(devnull_fd, 1)
        os.dup2(devnull_fd, 2)
        os.close(devnull_fd)
        signal.signal(signal.SIGINT, signal.SIG_DFL)
        if sys.platform != "win32":
            signal.signal(signal.SIGTERM, signal.SIG_DFL)
        asyncio.run(self._run_server())

    @abstractmethod
    async def _run_server(self) -> None:
        """The async server coroutine. Must run indefinitely until killed."""
        ...

    @staticmethod
    @abstractmethod
    def create_device(rack_key: int) -> SynnaxDevice:
        """Create a Synnax device instance for this simulator."""
        ...
