#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import threading
from collections.abc import Iterator

import pytest
from websockets.frames import Frame, Opcode
from websockets.protocol import Event
from websockets.sync.server import ServerConnection, serve

from alamos import NOOP
from freighter import URL, Empty
from freighter.websocket import Message
from synnax.exceptions import create_registry
from synnax.telem import TimeSpan
from synnax.transport import Transport
from x.codec import JSONCodec


@pytest.fixture
def pinged_endpoint() -> Iterator[tuple[URL, threading.Event]]:
    """Serves a stream that acks the open, then holds the connection. Sets the event
    once the client sends a ping."""
    codec = JSONCodec()
    pinged = threading.Event()

    class Connection(ServerConnection):
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
    def test_sync(self, pinged_endpoint: tuple[URL, threading.Event]) -> None:
        """Should ping the Core at the keep-alive interval on a sync stream."""
        url, pinged = pinged_endpoint
        transport = Transport(
            url=url,
            instrumentation=NOOP,
            registry=create_registry(),
            keep_alive=TimeSpan.MILLISECOND * 50,
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
