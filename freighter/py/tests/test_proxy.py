#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
import ssl
import threading
from collections.abc import Iterator
from socketserver import StreamRequestHandler, ThreadingTCPServer
from typing import override

import pytest

from freighter import URL, AsyncWebsocketClient, HTTPClient, WebsocketClient
from x.codec import JSONCodec

from .interface import Message, create_registry

_PROXY_VARIABLES = (
    "HTTP_PROXY",
    "HTTPS_PROXY",
    "ALL_PROXY",
    "NO_PROXY",
    "http_proxy",
    "https_proxy",
    "all_proxy",
    "no_proxy",
)


def _pipe(src: socket.socket, dst: socket.socket) -> None:
    while data := src.recv(65536):
        dst.sendall(data)
    dst.shutdown(socket.SHUT_WR)


class _Proxy(ThreadingTCPServer):
    daemon_threads = True
    targets: list[str]

    def __init__(self) -> None:
        super().__init__(("localhost", 0), _ConnectHandler)
        self.targets = []


class _ConnectHandler(StreamRequestHandler):
    """Tunnels a CONNECT request to its target and records the target."""

    @override
    def handle(self) -> None:
        assert isinstance(self.server, _Proxy)
        method, target, _ = self.rfile.readline().decode().split(" ", 2)
        while self.rfile.readline() not in (b"\r\n", b""):
            pass
        assert method == "CONNECT"
        self.server.targets.append(target)
        host, port = target.rsplit(":", 1)
        with socket.create_connection((host, int(port))) as upstream:
            self.wfile.write(b"HTTP/1.1 200 Connection established\r\n\r\n")
            back = threading.Thread(target=_pipe, args=(upstream, self.connection))
            back.start()
            _pipe(self.connection, upstream)
            back.join()


@pytest.fixture
def proxy(monkeypatch: pytest.MonkeyPatch) -> Iterator[_Proxy]:
    for name in _PROXY_VARIABLES:
        monkeypatch.delenv(name, raising=False)
    server = _Proxy()
    threading.Thread(target=server.serve_forever, args=(0.01,), daemon=True).start()
    monkeypatch.setenv("HTTPS_PROXY", f"http://localhost:{server.server_address[1]}")
    yield server
    server.shutdown()
    server.server_close()


def _send(url: URL, ctx: ssl.SSLContext) -> Message:
    client = HTTPClient(
        url=url, codec=JSONCodec(), registry=create_registry(), ssl_context=ctx
    )
    return client.send("/echo", Message(id=1, message="hi"), Message)


def _stream(url: URL, ctx: ssl.SSLContext) -> None:
    client = WebsocketClient(
        encoder=JSONCodec(), registry=create_registry(), base_url=url, ssl_context=ctx
    )
    client.stream("/stream", Message, Message).close_send()


async def _stream_async(url: URL, ctx: ssl.SSLContext) -> None:
    client = AsyncWebsocketClient(
        encoder=JSONCodec(), registry=create_registry(), base_url=url, ssl_context=ctx
    )
    await (await client.stream("/stream", Message, Message)).close_send()


def _target(url: URL) -> str:
    return f"{url.host}:{url.port}"


class TestProxy:
    """The clients connect through the proxy that the environment names."""

    def test_unary(
        self, proxy: _Proxy, http_url: URL, trusted_ctx: ssl.SSLContext
    ) -> None:
        """Should send a unary request through the proxy and verify the server."""
        assert _send(http_url, trusted_ctx) == Message(id=1, message="hi")
        assert proxy.targets == [_target(http_url)]

    def test_stream(
        self, proxy: _Proxy, ws_url: URL, trusted_ctx: ssl.SSLContext
    ) -> None:
        """Should open a stream through the proxy and verify the server."""
        _stream(ws_url, trusted_ctx)
        assert proxy.targets == [_target(ws_url)]

    @pytest.mark.asyncio
    async def test_stream_async(
        self, proxy: _Proxy, ws_url: URL, trusted_ctx: ssl.SSLContext
    ) -> None:
        """Should open an async stream through the proxy and verify the server."""
        await _stream_async(ws_url, trusted_ctx)
        assert proxy.targets == [_target(ws_url)]

    def test_no_proxy(
        self,
        proxy: _Proxy,
        http_url: URL,
        ws_url: URL,
        trusted_ctx: ssl.SSLContext,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should bypass the proxy for a host that NO_PROXY names."""
        monkeypatch.setenv("NO_PROXY", "localhost")
        assert _send(http_url, trusted_ctx) == Message(id=1, message="hi")
        _stream(ws_url, trusted_ctx)
        assert proxy.targets == []
