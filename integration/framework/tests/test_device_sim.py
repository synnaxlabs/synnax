#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
from collections.abc import Iterator

import pytest
from examples.simulators.device_sim import DeviceSim


class IdleSim(DeviceSim):
    host = "127.0.0.1"
    device_name = "idle"

    async def _run_server(self) -> None:
        pass


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
