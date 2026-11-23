#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import socket
import ssl
import threading
from collections.abc import Iterator
from socketserver import StreamRequestHandler, ThreadingTCPServer
from typing import override
from urllib.parse import urlsplit

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
    requests: list[tuple[str, str | None]]
    """The target and the Proxy-Authorization header of each connection."""

    def __init__(self) -> None:
        super().__init__(("localhost", 0), _Handler)
        self.requests = []

    @property
    def address(self) -> str:
        return f"localhost:{self.server_address[1]}"


class _Handler(StreamRequestHandler):
    """Tunnels a CONNECT request or forwards a plain one, and records it."""

    @override
    def handle(self) -> None:
        assert isinstance(self.server, _Proxy)
        head = [self.rfile.readline()]
        while head[-1] not in (b"\r\n", b""):
            head.append(self.rfile.readline())
        method, target, _ = head[0].decode().split(" ", 2)
        headers = {
            k.strip().lower(): v.strip()
            for k, v in (line.decode().split(":", 1) for line in head[1:-1])
        }
        address = target if method == "CONNECT" else urlsplit(target).netloc
        self.server.requests.append((address, headers.get("proxy-authorization")))
        host, port = address.rsplit(":", 1)
        with socket.create_connection((host, int(port))) as upstream:
            if method == "CONNECT":
                self.wfile.write(b"HTTP/1.1 200 Connection established\r\n\r\n")
            else:
                body = self.rfile.read(int(headers.get("content-length", "0")))
                upstream.sendall(b"".join(head) + body)
            back = threading.Thread(target=_pipe, args=(upstream, self.connection))
            back.start()
            _pipe(self.connection, upstream)
            back.join()


@pytest.fixture
def proxy(monkeypatch: pytest.MonkeyPatch, tls: bool) -> Iterator[_Proxy]:
    for name in _PROXY_VARIABLES:
        monkeypatch.delenv(name, raising=False)
    server = _Proxy()
    threading.Thread(target=server.serve_forever, args=(0.01,), daemon=True).start()
    monkeypatch.setenv(
        "HTTPS_PROXY" if tls else "HTTP_PROXY", f"http://{server.address}"
    )
    yield server
    server.shutdown()
    server.server_close()


@pytest.fixture
def client_ctx(tls: bool, trusted_ctx: ssl.SSLContext) -> ssl.SSLContext | None:
    return trusted_ctx if tls else None


def _send(url: URL, ctx: ssl.SSLContext | None) -> Message:
    client = HTTPClient(
        url=url, codec=JSONCodec(), registry=create_registry(), ssl_context=ctx
    )
    return client.send("/echo", Message(id=1, message="hi"), Message)


def _stream(url: URL, ctx: ssl.SSLContext | None) -> None:
    client = WebsocketClient(
        encoder=JSONCodec(), registry=create_registry(), base_url=url, ssl_context=ctx
    )
    client.stream("/stream", Message, Message).close_send()


async def _stream_async(url: URL, ctx: ssl.SSLContext | None) -> None:
    client = AsyncWebsocketClient(
        encoder=JSONCodec(), registry=create_registry(), base_url=url, ssl_context=ctx
    )
    await (await client.stream("/stream", Message, Message)).close_send()


def _target(url: URL) -> str:
    return f"{url.host}:{url.port}"


@pytest.mark.parametrize("tls", [True, False])
class TestProxy:
    """The clients connect through the proxy that the environment names."""

    def test_unary(
        self, proxy: _Proxy, http_url: URL, client_ctx: ssl.SSLContext | None
    ) -> None:
        """Should send a unary request through the proxy."""
        assert _send(http_url, client_ctx) == Message(id=1, message="hi")
        assert proxy.requests == [(_target(http_url), None)]

    def test_stream(
        self, proxy: _Proxy, ws_url: URL, client_ctx: ssl.SSLContext | None
    ) -> None:
        """Should open a stream through the proxy."""
        _stream(ws_url, client_ctx)
        assert proxy.requests == [(_target(ws_url), None)]

    @pytest.mark.asyncio
    async def test_stream_async(
        self, proxy: _Proxy, ws_url: URL, client_ctx: ssl.SSLContext | None
    ) -> None:
        """Should open an async stream through the proxy."""
        await _stream_async(ws_url, client_ctx)
        assert proxy.requests == [(_target(ws_url), None)]

    @pytest.mark.asyncio
    async def test_no_proxy(
        self,
        proxy: _Proxy,
        http_url: URL,
        ws_url: URL,
        client_ctx: ssl.SSLContext | None,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should bypass the proxy for a host that NO_PROXY names."""
        monkeypatch.setenv("NO_PROXY", "localhost")
        assert _send(http_url, client_ctx) == Message(id=1, message="hi")
        _stream(ws_url, client_ctx)
        await _stream_async(ws_url, client_ctx)
        assert proxy.requests == []

    def test_credentials(
        self,
        proxy: _Proxy,
        http_url: URL,
        ws_url: URL,
        client_ctx: ssl.SSLContext | None,
        tls: bool,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should send the decoded credentials in the proxy URL to the proxy."""
        name = "HTTPS_PROXY" if tls else "HTTP_PROXY"
        monkeypatch.setenv(name, f"http://user:p%40ss@{proxy.address}")
        _send(http_url, client_ctx)
        _stream(ws_url, client_ctx)
        auth = f"Basic {base64.b64encode(b'user:p@ss').decode()}"
        assert proxy.requests == [(_target(http_url), auth), (_target(ws_url), auth)]

    def test_no_scheme(
        self,
        proxy: _Proxy,
        http_url: URL,
        ws_url: URL,
        client_ctx: ssl.SSLContext | None,
        tls: bool,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should treat a proxy with no scheme as an http proxy."""
        name = "HTTPS_PROXY" if tls else "HTTP_PROXY"
        monkeypatch.setenv(name, proxy.address)
        _send(http_url, client_ctx)
        _stream(ws_url, client_ctx)
        assert proxy.requests == [(_target(http_url), None), (_target(ws_url), None)]

    def test_unsupported_scheme(
        self,
        proxy: _Proxy,
        client_ctx: ssl.SSLContext | None,
        tls: bool,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should reject a SOCKS proxy when the clients are created."""
        name = "HTTPS_PROXY" if tls else "HTTP_PROXY"
        monkeypatch.setenv(name, f"socks5h://{proxy.address}")
        url = URL(host="example.com", port=9090)
        with pytest.raises(ValueError, match=f"{name} has scheme 'socks5h'"):
            HTTPClient(
                url=url,
                codec=JSONCodec(),
                registry=create_registry(),
                ssl_context=client_ctx,
            )
        with pytest.raises(ValueError, match=f"{name} has scheme 'socks5h'"):
            WebsocketClient(
                encoder=JSONCodec(),
                registry=create_registry(),
                base_url=url,
                ssl_context=client_ctx,
            )
