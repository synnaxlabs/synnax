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

from freighter import URL


@pytest.fixture
def endpoint() -> URL:
    return URL(host="localhost", port=8080)


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
    # Closing after the client's close message keeps server shutdown fast.
    await ws.recv()


@pytest.fixture(scope="module")
def ca() -> trustme.CA:
    return trustme.CA()


@pytest.fixture(scope="session")
def host() -> str:
    """The address that every test server binds and every client dials. A name such as
    localhost can resolve to another server on the other address family."""
    return "127.0.0.1"


@pytest.fixture
def cert_host(host: str) -> str:
    return host


@pytest.fixture
def tls() -> bool:
    """Whether the http_url and ws_url servers use TLS."""
    return True


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
def http_url(host: str, server_ctx: ssl.SSLContext, tls: bool) -> Iterator[URL]:
    server = HTTPServer((host, 0), _EchoHandler)
    if tls:
        server.socket = server_ctx.wrap_socket(server.socket, server_side=True)
    threading.Thread(target=server.serve_forever, args=(0.01,), daemon=True).start()
    yield URL(host=host, port=server.server_address[1])
    server.shutdown()
    server.server_close()


@pytest.fixture
def ws_url(host: str, server_ctx: ssl.SSLContext, tls: bool) -> Iterator[URL]:
    # The sync websockets server drops some TLS 1.3 handshakes, so the server runs on
    # an event loop in its own thread.
    loop = asyncio.new_event_loop()
    threading.Thread(target=loop.run_forever, daemon=True).start()

    async def start() -> Server:
        return await serve(_open_ack, host, 0, ssl=server_ctx if tls else None)

    server = asyncio.run_coroutine_threadsafe(start(), loop).result()
    yield URL(host=host, port=next(iter(server.sockets)).getsockname()[1])

    async def stop() -> None:
        server.close()
        await server.wait_closed()

    asyncio.run_coroutine_threadsafe(stop(), loop).result()
    loop.call_soon_threadsafe(loop.stop)
