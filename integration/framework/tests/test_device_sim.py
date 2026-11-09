#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import re
import socket
from collections.abc import Iterator
from multiprocessing.process import BaseProcess

import pytest
from examples.simulators.device_sim import DeviceSim

import synnax as sy


class IdleSim(DeviceSim):
    host = "127.0.0.1"
    device_name = "idle"

    async def _run_server(self) -> None:
        pass


class SilentSim(DeviceSim):
    host = "127.0.0.1"
    device_name = "silent"
    startup_timeout = 500 * sy.TimeSpan.MILLISECOND
    started: BaseProcess | None = None

    async def _run_server(self) -> None:
        await asyncio.sleep(60)

    def _wait_for_ready(self) -> None:
        self.started = self.process
        super()._wait_for_ready()


@pytest.fixture
def free_port() -> int:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.bind(("127.0.0.1", 0))
        port: int = sock.getsockname()[1]
        return port


@pytest.fixture
def listener() -> Iterator[socket.socket]:
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.bind(("127.0.0.1", 0))
    sock.listen()
    yield sock
    sock.close()


class TestDeviceSim:
    """SimulatorCase starts each DeviceSim and relies on it to reject a busy port."""

    def test_should_raise_and_leave_the_holder_when_the_port_is_busy(
        self, listener: socket.socket
    ) -> None:
        sim = IdleSim()
        sim.port = listener.getsockname()[1]
        with pytest.raises(RuntimeError, match="already in use"):
            sim.start()
        assert sim.process is None
        with socket.create_connection(("127.0.0.1", sim.port), timeout=1):
            pass

    def test_should_stop_the_server_when_it_is_not_ready_in_time(
        self, free_port: int
    ) -> None:
        sim = SilentSim()
        sim.port = free_port
        message = f"Server not ready on 127.0.0.1:{free_port} after 500ms"
        with pytest.raises(RuntimeError, match=re.escape(message)):
            sim.start()
        assert sim.process is None
        assert sim.started is not None
        assert not sim.started.is_alive()
