#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import ssl
import threading
from collections.abc import Iterator
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path

import pytest
import trustme
from websockets.asyncio.server import Server, ServerConnection, serve

from freighter import (
    URL,
    AsyncWebsocketClient,
    HTTPClient,
    Unreachable,
    WebsocketClient,
)
from x.codec import JSONCodec

from .interface import Message, create_registry


class _EchoHandler(BaseHTTPRequestHandler):
    def do_POST(self) -> None:
        body = self.rfile.read(int(self.headers["Content-Length"]))
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


async def _open_ack(ws: ServerConnection) -> None:
    await ws.send(b'{"type": "open", "payload": {"id": null, "message": null}}')
    async for _ in ws:
        pass


@pytest.fixture(scope="module")
def ca() -> trustme.CA:
    return trustme.CA()


@pytest.fixture
def cert_host() -> str:
    return "localhost"


@pytest.fixture
def server_ctx(ca: trustme.CA, cert_host: str) -> ssl.SSLContext:
    ctx = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
    ca.issue_cert(cert_host).configure_cert(ctx)
    return ctx


@pytest.fixture
def trusted_ctx(tmp_path: Path, ca: trustme.CA) -> ssl.SSLContext:
    path = tmp_path / "ca.crt"
    ca.cert_pem.write_to_path(str(path))
    return ssl.create_default_context(cafile=str(path))


@pytest.fixture
def http_url(server_ctx: ssl.SSLContext) -> Iterator[URL]:
    server = HTTPServer(("localhost", 0), _EchoHandler)
    server.socket = server_ctx.wrap_socket(server.socket, server_side=True)
    threading.Thread(target=server.serve_forever, args=(0.01,), daemon=True).start()
    yield URL(host="localhost", port=server.server_address[1])
    server.shutdown()
    server.server_close()


@pytest.fixture
def ws_url(server_ctx: ssl.SSLContext) -> Iterator[URL]:
    # The sync websockets server drops some TLS 1.3 handshakes, so the server runs on
    # an event loop in its own thread.
    loop = asyncio.new_event_loop()
    threading.Thread(target=loop.run_forever, daemon=True).start()

    async def start() -> Server:
        return await serve(_open_ack, "localhost", 0, ssl=server_ctx)

    server = asyncio.run_coroutine_threadsafe(start(), loop).result()
    yield URL(host="localhost", port=next(iter(server.sockets)).getsockname()[1])

    async def stop() -> None:
        server.close()
        await server.wait_closed()

    asyncio.run_coroutine_threadsafe(stop(), loop).result()
    loop.call_soon_threadsafe(loop.stop)


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
    ).with_codec(JSONCodec())
    await (await client.stream("/stream", Message, Message)).close_send()


class TestHTTP:
    """The HTTP client verifies the server's certificate and hostname."""

    def test_untrusted(self, http_url: URL) -> None:
        """Should reject a certificate that no trusted CA signs."""
        with pytest.raises(Unreachable):
            _send(http_url, ssl.create_default_context())

    def test_trusted(self, http_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should accept a certificate that the given CA signs."""
        assert _send(http_url, trusted_ctx) == Message(id=1, message="hi")

    @pytest.mark.parametrize("cert_host", ["example.com"])
    def test_hostname(self, http_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should reject a trusted certificate issued for a different host."""
        with pytest.raises(Unreachable):
            _send(http_url, trusted_ctx)


class TestWebsocket:
    """The WebSocket client verifies the server's certificate and hostname."""

    def test_untrusted(self, ws_url: URL) -> None:
        """Should reject a certificate that no trusted CA signs."""
        with pytest.raises(ssl.SSLCertVerificationError):
            _stream(ws_url, ssl.create_default_context())

    def test_trusted(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should accept a certificate that the given CA signs."""
        _stream(ws_url, trusted_ctx)

    @pytest.mark.parametrize("cert_host", ["example.com"])
    def test_hostname(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should reject a trusted certificate issued for a different host."""
        with pytest.raises(ssl.SSLCertVerificationError):
            _stream(ws_url, trusted_ctx)


@pytest.mark.asyncio
class TestAsyncWebsocket:
    """The async WebSocket client verifies the server's certificate and hostname."""

    async def test_untrusted(self, ws_url: URL) -> None:
        """Should reject a certificate that no trusted CA signs."""
        with pytest.raises(ssl.SSLCertVerificationError):
            await _stream_async(ws_url, ssl.create_default_context())

    async def test_trusted(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should accept a certificate that the given CA signs."""
        await _stream_async(ws_url, trusted_ctx)

    @pytest.mark.parametrize("cert_host", ["example.com"])
    async def test_hostname(self, ws_url: URL, trusted_ctx: ssl.SSLContext) -> None:
        """Should reject a trusted certificate issued for a different host."""
        with pytest.raises(ssl.SSLCertVerificationError):
            await _stream_async(ws_url, trusted_ctx)
