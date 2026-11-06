#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import dataclasses
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path

from framework.models import SynnaxConnection

_READY_TIMEOUT = 60.0
_STOP_TIMEOUT = 15.0
# A licensed Core logs this once its embedded Driver has started.
_READY_LINE = b"Synnax is running and available at"


@dataclass(frozen=True)
class Core:
    """A Core that the conductor runs tests on, and the log it writes."""

    connection: SynnaxConnection
    log_path: Path

    @property
    def address(self) -> str:
        return f"{self.connection.server_address}:{self.connection.port}"


class Process:
    """A Core that the conductor started."""

    def __init__(self, *, binary: Path, core: Core) -> None:
        self.core = core
        self._data = Path(tempfile.mkdtemp(prefix="synnax-core-"))
        core.log_path.parent.mkdir(parents=True, exist_ok=True)
        self._log = open(core.log_path, "wb")
        # Only the first Core's Driver reads the NI devices.
        args = [str(binary), "start", "-mi", "-l", core.address, "-d", str(self._data)]
        args += ["--disable-integrations", "ni"]
        flags = 0
        if sys.platform == "win32":
            # Windows sends CTRL_BREAK_EVENT only to a new process group.
            flags = subprocess.CREATE_NEW_PROCESS_GROUP
        self._popen = subprocess.Popen(
            args, stdout=self._log, stderr=subprocess.STDOUT, creationflags=flags
        )

    def wait_ready(self) -> None:
        """Blocks until the Core and its Driver are running.

        :raises RuntimeError: If the Core exits or does not start in time.
        """
        deadline = time.monotonic() + _READY_TIMEOUT
        while time.monotonic() < deadline:
            if self._popen.poll() is not None:
                raise RuntimeError(
                    f"Core at {self.core.address} exited with code "
                    f"{self._popen.returncode}. See {self.core.log_path}"
                )
            if _READY_LINE in self.core.log_path.read_bytes():
                return
            time.sleep(0.5)
        raise RuntimeError(
            f"Core at {self.core.address} did not start within {_READY_TIMEOUT}s. "
            f"See {self.core.log_path}"
        )

    def _interrupt(self) -> None:
        if self._popen.poll() is not None:
            return
        if sys.platform == "win32":
            self._popen.send_signal(signal.CTRL_BREAK_EVENT)
        else:
            self._popen.send_signal(signal.SIGINT)

    def _wait_stopped(self) -> None:
        try:
            self._popen.wait(timeout=_STOP_TIMEOUT)
        except subprocess.TimeoutExpired:
            self._popen.kill()
            self._popen.wait()
        self._log.close()
        shutil.rmtree(self._data, ignore_errors=True)


def start(
    *, binary: Path, first: SynnaxConnection, count: int, log_dir: Path
) -> list[Process]:
    """Starts count Cores on the ports after the first Core's port, and waits until
    each runs. Each Core keeps its data in memory.

    :raises RuntimeError: If a port is taken, or a Core does not start.
    """
    connections = [
        dataclasses.replace(first, port=first.port + i) for i in range(1, count + 1)
    ]
    for conn in connections:
        if _listening(conn.server_address, conn.port):
            raise RuntimeError(f"Port {conn.port} is already in use")
    processes: list[Process] = []
    try:
        for i, conn in enumerate(connections, start=1):
            core = Core(connection=conn, log_path=log_dir / f"core-{i}.log")
            processes.append(Process(binary=binary, core=core))
        for p in processes:
            p.wait_ready()
    except BaseException:
        stop(processes)
        raise
    return processes


def stop(processes: list[Process]) -> None:
    """Interrupts each Core so it stops its Driver, then kills a Core that does not
    exit in time."""
    for p in processes:
        p._interrupt()
    for p in processes:
        p._wait_stopped()


def _listening(host: str, port: int) -> bool:
    try:
        with socket.create_connection((host, port), timeout=0.5):
            return True
    except OSError:
        return False
