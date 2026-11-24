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
import sys
import threading
from collections.abc import Iterator
from datetime import timedelta
from http.server import BaseHTTPRequestHandler, HTTPServer
from pathlib import Path
from typing import override

import pytest
import trustme
from websockets.asyncio import server as async_server
from websockets.frames import Frame, Opcode
from websockets.protocol import Event
from websockets.sync.server import ServerConnection, serve

from alamos import NOOP
from freighter import URL, Empty, Unreachable
from freighter.websocket import Message
from synnax.exceptions import create_registry
from synnax.telem import CrudeTimeSpan, TimeSpan
from synnax.transport import Transport
from x.codec import JSONCodec


@pytest.fixture
def pinged_endpoint() -> Iterator[tuple[URL, threading.Event]]:
    """Serves a stream that acks the open, then holds the connection. Sets the event
    once the client sends a ping."""
    codec = JSONCodec()
    pinged = threading.Event()

    class Connection(ServerConnection):
        @override
        def process_event(self, event: Event) -> None:
            if isinstance(event, Frame) and event.opcode is Opcode.PING:
                pinged.set()
            super().process_event(event)

    def handler(ws: ServerConnection) -> None:
        ws.send(codec.encode(Message[Empty](type="open", payload=Empty())))
        ws.recv()

    with serve(handler, "localhost", 0, create_connection=Connection) as server:
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        yield URL(host="localhost", port=server.socket.getsockname()[1]), pinged
        server.shutdown()
        thread.join()


class _EchoHandler(BaseHTTPRequestHandler):
    def do_POST(self) -> None:
        body = self.rfile.read(int(self.headers["Content-Length"]))
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)


async def _open_ack(ws: async_server.ServerConnection) -> None:
    await ws.send(JSONCodec().encode(Message[Empty](type="open", payload=Empty())))
    await ws.recv()


@pytest.fixture(scope="module")
def ca() -> trustme.CA:
    return trustme.CA()


@pytest.fixture
def ca_cert(tmp_path: Path, ca: trustme.CA) -> str:
    path = tmp_path / "ca.crt"
    ca.cert_pem.write_to_path(str(path))
    return str(path)


@pytest.fixture
def server_ctx(ca: trustme.CA) -> ssl.SSLContext:
    ctx = ssl.create_default_context(ssl.Purpose.CLIENT_AUTH)
    ca.issue_cert("localhost").configure_cert(ctx)
    return ctx


@pytest.fixture
def tls_endpoints(server_ctx: ssl.SSLContext) -> Iterator[tuple[URL, URL]]:
    """Serves HTTP and WebSocket endpoints with a certificate that the ca fixture
    signs. Yields the HTTP URL, then the WebSocket URL."""
    http = HTTPServer(("localhost", 0), _EchoHandler)
    http.socket = server_ctx.wrap_socket(http.socket, server_side=True)
    threading.Thread(target=http.serve_forever, args=(0.01,), daemon=True).start()
    # The sync websockets server drops some TLS 1.3 handshakes.
    loop = asyncio.new_event_loop()
    threading.Thread(target=loop.run_forever, daemon=True).start()

    async def start() -> async_server.Server:
        return await async_server.serve(_open_ack, "localhost", 0, ssl=server_ctx)

    ws = asyncio.run_coroutine_threadsafe(start(), loop).result()
    yield (
        URL(host="localhost", port=http.server_address[1]),
        URL(host="localhost", port=next(iter(ws.sockets)).getsockname()[1]),
    )

    async def stop() -> None:
        ws.close()
        await ws.wait_closed()

    asyncio.run_coroutine_threadsafe(stop(), loop).result()
    loop.call_soon_threadsafe(loop.stop)
    http.shutdown()
    http.server_close()


def _transport(url: URL, ca_cert: str | None = None) -> Transport:
    return Transport(
        url=url,
        instrumentation=NOOP,
        registry=create_registry(),
        secure=True,
        ca_cert=ca_cert,
    )


async def _connect(endpoints: tuple[URL, URL], ca_cert: str | None = None) -> None:
    http_url, ws_url = endpoints
    _transport(http_url, ca_cert).unary.send("/", Empty(), Empty)
    _transport(ws_url, ca_cert).stream.stream("/", Empty, Empty).close_send()
    stream = await _transport(ws_url, ca_cert).stream_async.stream("/", Empty, Empty)
    await stream.close_send()


class TestTransport:
    def test_ca_cert_insecure(self) -> None:
        """Should reject a CA certificate for an insecure connection."""
        with pytest.raises(ValueError, match="secure"):
            Transport(
                url=URL(host="localhost", port=9090),
                instrumentation=NOOP,
                registry=create_registry(),
                ca_cert="/certs/ca.crt",
            )


class TestKeepAlive:
    @pytest.mark.parametrize(
        "keep_alive",
        [0.05, TimeSpan.MILLISECOND * 50, timedelta(milliseconds=50)],
        ids=["seconds", "time_span", "timedelta"],
    )
    def test_sync(
        self,
        pinged_endpoint: tuple[URL, threading.Event],
        keep_alive: CrudeTimeSpan,
    ) -> None:
        """Should ping the Core at the keep-alive interval on a sync stream."""
        url, pinged = pinged_endpoint
        transport = Transport(
            url=url,
            instrumentation=NOOP,
            registry=create_registry(),
            keep_alive=keep_alive,
        )
        stream = transport.stream.stream("/", Empty, Empty)
        assert pinged.wait(2)
        stream.close_send()

    @pytest.mark.asyncio
    async def test_async(self, pinged_endpoint: tuple[URL, threading.Event]) -> None:
        """Should ping the Core at the keep-alive interval on an async stream."""
        url, pinged = pinged_endpoint
        transport = Transport(
            url=url,
            instrumentation=NOOP,
            registry=create_registry(),
            keep_alive=TimeSpan.MILLISECOND * 50,
        )
        stream = await transport.stream_async.stream("/", Empty, Empty)
        assert await asyncio.to_thread(pinged.wait, 2)
        await stream.close_send()


@pytest.mark.asyncio
class TestTLS:
    async def test_ca_cert(self, tls_endpoints: tuple[URL, URL], ca_cert: str) -> None:
        """Should trust a Core certificate that ca_cert signs."""
        await _connect(tls_endpoints, ca_cert)

    async def test_untrusted(self, tls_endpoints: tuple[URL, URL]) -> None:
        """Should reject a Core certificate that the system store does not trust."""
        http_url, ws_url = tls_endpoints
        with pytest.raises(Unreachable):
            _transport(http_url).unary.send("/", Empty(), Empty)
        with pytest.raises(ssl.SSLCertVerificationError):
            _transport(ws_url).stream.stream("/", Empty, Empty)
        with pytest.raises(ssl.SSLCertVerificationError):
            await _transport(ws_url).stream_async.stream("/", Empty, Empty)

    @pytest.mark.skipif(
        sys.platform != "linux", reason="The system store is OpenSSL only on Linux"
    )
    async def test_system_store(
        self,
        tls_endpoints: tuple[URL, URL],
        ca_cert: str,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should trust a Core certificate that the system store trusts."""
        monkeypatch.setenv("SSL_CERT_FILE", ca_cert)
        await _connect(tls_endpoints)

    @pytest.mark.skipif(
        sys.platform == "linux", reason="The system store is OpenSSL on Linux"
    )
    async def test_openssl_file(
        self,
        tls_endpoints: tuple[URL, URL],
        ca_cert: str,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        """Should ignore the certificate file of Python's OpenSSL outside Linux."""
        monkeypatch.setenv("SSL_CERT_FILE", ca_cert)
        with pytest.raises(Unreachable):
            _transport(tls_endpoints[0]).unary.send("/", Empty(), Empty)
