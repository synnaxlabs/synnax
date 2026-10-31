#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
import threading
from collections.abc import Iterator

import pytest

from framework.proxy import SeverableProxy


def _echo(conn: socket.socket) -> None:
    with conn:
        while data := conn.recv(1024):
            conn.sendall(data)


def _serve(listener: socket.socket) -> None:
    while True:
        try:
            conn, _ = listener.accept()
        except OSError:
            return
        threading.Thread(target=_echo, args=(conn,), daemon=True).start()


@pytest.fixture
def proxy() -> Iterator[SeverableProxy]:
    listener = socket.create_server(("127.0.0.1", 0))
    threading.Thread(target=_serve, args=(listener,), daemon=True).start()
    p = SeverableProxy(target_host="127.0.0.1", target_port=listener.getsockname()[1])
    yield p
    p.close()
    listener.close()


def _connect(proxy: SeverableProxy) -> socket.socket:
    return socket.create_connection(("127.0.0.1", proxy.port), timeout=5)


def _round_trip(conn: socket.socket, data: bytes) -> bytes:
    conn.sendall(data)
    return conn.recv(1024)


class TestSeverableProxy:
    """The proxy forwards to its target until severed, and again once restored."""

    def test_should_forward_to_the_target(self, proxy: SeverableProxy) -> None:
        with _connect(proxy) as conn:
            assert _round_trip(conn, b"ping") == b"ping"

    def test_should_drop_live_links_and_refuse_new_ones_when_severed(
        self, proxy: SeverableProxy
    ) -> None:
        with _connect(proxy) as conn:
            assert _round_trip(conn, b"ping") == b"ping"
            proxy.sever()
            try:
                assert conn.recv(1024) == b""
            except ConnectionResetError:
                pass
        with pytest.raises(ConnectionRefusedError):
            _connect(proxy)

    def test_should_forward_again_on_the_same_port_when_restored(
        self, proxy: SeverableProxy
    ) -> None:
        port = proxy.port
        proxy.sever()
        proxy.restore()
        assert proxy.port == port
        with _connect(proxy) as conn:
            assert _round_trip(conn, b"ping") == b"ping"
