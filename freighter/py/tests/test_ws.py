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
import time
from collections.abc import Iterator
from uuid import uuid4

import pytest
from pydantic import BaseModel
from websockets.protocol import State
from websockets.sync.server import ServerConnection, serve

import freighter.exceptions
from freighter.context import Context
from freighter.http import HTTPClient
from freighter.transport import AsyncNext, Empty, Next, P
from freighter.url import URL
from freighter.websocket import (
    AsyncWebsocketClient,
    ConnectionClosedError,
    WebsocketClient,
)
from freighter.websocket import Message as WebsocketMessage
from x.codec import Codec, JSONCodec, MessagePackCodec

from .interface import Error, Message, create_registry


class MyVerySpecialCustomCodec(Codec):
    """A custom codec that uses JSON for encoding and decoding."""

    base = JSONCodec()

    def content_type(self) -> str:
        return "application/json"

    def encode(self, data: BaseModel) -> bytes:
        if isinstance(data, WebsocketMessage):
            data.payload = Message(id=4200, message="the key to the universe")
        return self.base.encode(data)

    def decode(self, data: bytes, pld_t: type[P]) -> P:
        return self.base.decode(data, pld_t)


@pytest.fixture
def async_client(endpoint: URL) -> AsyncWebsocketClient:
    ws_endpoint = endpoint.child("stream")
    return AsyncWebsocketClient(
        encoder=MessagePackCodec(), registry=create_registry(), base_url=ws_endpoint
    )


@pytest.fixture
def sync_client(endpoint: URL) -> WebsocketClient:
    ws_endpoint = endpoint.child("stream")
    return WebsocketClient(
        encoder=MessagePackCodec(), registry=create_registry(), base_url=ws_endpoint
    )


@pytest.fixture
def unary_client(endpoint: URL) -> HTTPClient:
    return HTTPClient(
        url=endpoint.child("unary"), codec=JSONCodec(), registry=create_registry()
    )


@pytest.fixture
def errorless_close_endpoint() -> Iterator[URL]:
    """Serves a stream that acks the open, then closes without an error."""
    codec = JSONCodec()

    def handler(ws: ServerConnection) -> None:
        ws.send(codec.encode(WebsocketMessage[Empty](type="open", payload=Empty())))
        ws.send(codec.encode(WebsocketMessage[Empty](type="close", payload=Empty())))
        for _ in ws:
            pass

    with serve(handler, "localhost", 0) as server:
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        yield URL(host="localhost", port=server.socket.getsockname()[1])
        server.shutdown()
        thread.join()


class TestServerCloseWithoutError:
    @pytest.mark.asyncio
    async def test_async(self, errorless_close_endpoint: URL) -> None:
        """Should raise EOF when the server closes without an error."""
        client = AsyncWebsocketClient(
            encoder=JSONCodec(),
            registry=create_registry(),
            base_url=errorless_close_endpoint,
        )
        stream = await client.stream("/", Empty, Empty)
        with pytest.raises(freighter.EOF):
            await stream.receive()
        with pytest.raises(freighter.EOF):
            await stream.receive()

    def test_sync(self, errorless_close_endpoint: URL) -> None:
        """Should raise EOF when the server closes without an error."""
        client = WebsocketClient(
            encoder=JSONCodec(),
            registry=create_registry(),
            base_url=errorless_close_endpoint,
        )
        stream = client.stream("/", Empty, Empty)
        with pytest.raises(freighter.EOF):
            stream.receive()
        with pytest.raises(freighter.EOF):
            stream.receive()


@pytest.fixture
def server_close_endpoint() -> Iterator[tuple[URL, threading.Event]]:
    """Serves a stream that acks the open, then closes the connection. Sets the event
    once the close handshake completes."""
    codec = JSONCodec()
    closed = threading.Event()

    def handler(ws: ServerConnection) -> None:
        ws.send(codec.encode(WebsocketMessage[Empty](type="open", payload=Empty())))
        ws.close()
        closed.set()

    with serve(handler, "localhost", 0) as server:
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        yield URL(host="localhost", port=server.socket.getsockname()[1]), closed
        server.shutdown()
        thread.join()


@pytest.mark.asyncio
class TestAsyncServerCloseConnection:
    async def test_receive(
        self, server_close_endpoint: tuple[URL, threading.Event]
    ) -> None:
        """Should raise EOF when the server closes the connection."""
        url, _ = server_close_endpoint
        client = AsyncWebsocketClient(
            encoder=JSONCodec(), registry=create_registry(), base_url=url
        )
        stream = await client.stream("/", Empty, Empty)
        with pytest.raises(freighter.EOF):
            await stream.receive()

    async def test_close_send(
        self, server_close_endpoint: tuple[URL, threading.Event]
    ) -> None:
        """Should raise EOF when closing the send side after the server closes the
        connection."""
        url, closed = server_close_endpoint
        client = AsyncWebsocketClient(
            encoder=JSONCodec(), registry=create_registry(), base_url=url
        )
        stream = await client.stream("/", Empty, Empty)
        assert await asyncio.to_thread(closed.wait, 2)
        with pytest.raises(freighter.EOF):
            await stream.close_send()


@pytest.fixture
def failed_open_endpoint() -> Iterator[tuple[URL, threading.Event]]:
    """Serves a stream that answers the open with a data message instead of an ack.
    Sets the event once the client closes the connection."""
    codec = JSONCodec()
    closed = threading.Event()

    def handler(ws: ServerConnection) -> None:
        ws.send(codec.encode(WebsocketMessage[Empty](type="data", payload=Empty())))
        for _ in ws:
            pass
        closed.set()

    with serve(handler, "localhost", 0) as server:
        thread = threading.Thread(target=server.serve_forever)
        thread.start()
        yield URL(host="localhost", port=server.socket.getsockname()[1]), closed
        server.shutdown()
        thread.join()


class TestFailedOpen:
    @pytest.mark.asyncio
    async def test_async(
        self, failed_open_endpoint: tuple[URL, threading.Event]
    ) -> None:
        """Should close the connection when the server does not ack the open."""
        url, closed = failed_open_endpoint
        client = AsyncWebsocketClient(
            encoder=JSONCodec(), registry=create_registry(), base_url=url
        )
        with pytest.raises(RuntimeError):
            await client.stream("/", Empty, Empty)
        assert await asyncio.to_thread(closed.wait, 2)

    def test_sync(self, failed_open_endpoint: tuple[URL, threading.Event]) -> None:
        """Should close the connection when the server does not ack the open."""
        url, closed = failed_open_endpoint
        client = WebsocketClient(
            encoder=JSONCodec(), registry=create_registry(), base_url=url
        )
        with pytest.raises(RuntimeError):
            client.stream("/", Empty, Empty)
        assert closed.wait(2)


class TestWithCodec:
    @pytest.mark.asyncio
    async def test_async_max_message_size(self, errorless_close_endpoint: URL) -> None:
        """Should keep the maximum message size of the original client."""
        client = AsyncWebsocketClient(
            encoder=JSONCodec(),
            registry=create_registry(),
            base_url=errorless_close_endpoint,
            max_message_size=8,
        ).with_codec(JSONCodec())
        with pytest.raises(ConnectionClosedError):
            await client.stream("/", Empty, Empty)

    def test_sync_max_message_size(self, errorless_close_endpoint: URL) -> None:
        """Should keep the maximum message size of the original client."""
        client = WebsocketClient(
            encoder=JSONCodec(),
            registry=create_registry(),
            base_url=errorless_close_endpoint,
            max_message_size=8,
        ).with_codec(JSONCodec())
        with pytest.raises(ConnectionClosedError):
            client.stream("/", Empty, Empty)


@pytest.mark.ws
@pytest.mark.asyncio
class TestAsyncWebsocket:
    async def test_basic_exchange(self, async_client: AsyncWebsocketClient) -> None:
        """Should exchange ten echo messages that increment the ID."""
        stream = await async_client.stream("/echo", Message, Message)
        for i in range(10):
            await stream.send(Message(id=i, message="hello"))
            msg = await stream.receive()
            assert msg.id == i + 1
            assert msg.message == "hello"
        await stream.close_send()
        with pytest.raises(freighter.EOF):
            await stream.receive()

    async def test_receive_message_after_close(
        self, async_client: AsyncWebsocketClient
    ) -> None:
        """Should receive a message and EOF error after the server closes the
        connection."""
        stream = await async_client.stream(
            "/sendMessageAfterClientClose", Message, Message
        )
        await stream.close_send()
        # calling should be idempotent
        await stream.close_send()
        msg = await stream.receive()
        assert msg.id == 0
        assert msg.message == "Close Acknowledged"
        with pytest.raises(freighter.EOF):
            await stream.receive()

    async def test_receive_error(self, async_client: AsyncWebsocketClient) -> None:
        """Should correctly decode a custom error from the server."""
        stream = await async_client.stream("/receiveAndExitWithErr", Message, Message)
        await stream.send(Message(id=1, message="hello"))
        with pytest.raises(Error) as exc_info:
            await stream.receive()
        assert exc_info.value.code == 1
        assert exc_info.value.message == "unexpected error"
        await stream.close_send()

    async def test_middleware(self, async_client: AsyncWebsocketClient) -> None:
        dct = {"called": False}

        async def mw(md: Context, next: AsyncNext) -> Context:
            md.params["Test"] = "test"
            dct["called"] = True
            return await next(md)

        async_client.use(mw)
        stream = await async_client.stream("/middlewareCheck", Message, Message)
        await stream.close_send()
        with pytest.raises(freighter.EOF):
            await stream.receive()
        assert dct["called"]

    async def test_server_timeout(
        self, async_client: AsyncWebsocketClient, unary_client: HTTPClient
    ) -> None:
        """Should correctly timeout if the server exceeds a write deadline"""
        stream = await async_client.stream("/slamMessages", Message, Message)
        msg_str = str(uuid4())
        await stream.send(Message(id=1, message=msg_str))
        # The server times out only after the socket buffers fill, which takes a few
        # seconds on a slow runner. The blocking sleep keeps the client from reading.
        deadline = time.monotonic() + 30
        while True:
            res = unary_client.send(
                "/slamMessagesTimeoutCheck", Message(id=1, message=msg_str), Message
            )
            if res.message == "timeout" or time.monotonic() > deadline:
                break
            time.sleep(0.25)
        assert res.message == "timeout"
        with pytest.raises(ConnectionClosedError):
            while True:
                await stream.receive()

    async def test_with_custom_codec(self, async_client: AsyncWebsocketClient) -> None:
        """Should correctly use a custom codec for the websocket client."""
        async_client = async_client.with_codec(MyVerySpecialCustomCodec())
        stream = await async_client.stream("/echo", Message, Message)
        for _ in range(1):
            await stream.send(Message(id=12, message="what we send here is ignored"))
            msg = await stream.receive()
            assert msg.id == 4201
            assert msg.message == "the key to the universe"


@pytest.mark.ws
class TestSyncWebsocket:
    def test_basic_exchange(self, sync_client: WebsocketClient) -> None:
        stream = sync_client.stream("/echo", Message, Message)
        for i in range(10):
            stream.send(Message(id=i, message="hello"))
            msg = stream.receive()
            assert msg.id == i + 1
            assert msg.message == "hello"
        stream.close_send()
        with pytest.raises(freighter.EOF):
            stream.receive()

    def test_repeated_receive(self, sync_client: WebsocketClient) -> None:
        """Should receive ten messages from the server."""
        stream = sync_client.stream("/respondWithTenMessages", Message, Message)
        c = 0
        while True:
            try:
                msg = stream.receive()
            except freighter.EOF:
                break
            c += 1
            assert msg.message == "hello"
        stream.close_send()
        assert c == 10
        with pytest.raises(freighter.EOF):
            stream.receive()

    def test_middleware(self, sync_client: WebsocketClient) -> None:
        """Should receive ten messages from the server."""
        dct = {"called": False}

        def mw(md: Context, next: Next) -> Context:
            md.params["Test"] = "test"
            dct["called"] = True
            return next(md)

        sync_client.use(mw)
        stream = sync_client.stream("/middlewareCheck", Message, Message)
        stream.close_send()
        with pytest.raises(freighter.EOF):
            stream.receive()
        assert dct["called"]

    def test_middleware_error_on_server(self, sync_client: WebsocketClient) -> None:
        """Should correctly decode and throw an error when the server middleware chain
        fails"""
        with pytest.raises(Error):
            sync_client.stream("/middlewareCheck", Message, Message)

    def test_client_timeout(self, sync_client: WebsocketClient) -> None:
        """Should correctly timeout if the server exceeds a write deadline"""
        stream = sync_client.stream("/echo", Message, Message)
        with pytest.raises(TimeoutError):
            stream.receive(timeout=0.1)
        stream.close_send()
        while True:
            try:
                stream.receive()
            except freighter.EOF:
                break

    def test_timeout_0(self, sync_client: WebsocketClient) -> None:
        """Should return immediately when empty, and the frame once it is available"""
        stream = sync_client.stream("/eventuallyResponseWithMessage", Message, Message)
        stream.send(Message(id=1, message="hello"))
        msg: Message | None = None
        # The server holds the answer back, so the early polls must come back empty
        # instead of blocking.
        empty_polls = 0
        for _ in range(100):
            try:
                msg = stream.receive(timeout=0)
                break
            except TimeoutError:
                empty_polls += 1
                time.sleep(0.05)
        assert msg is not None, "the server never answered"
        assert msg.id == 1
        assert empty_polls > 0, "receive(timeout=0) blocked until the answer arrived"

    def test_receive_error(self, sync_client: WebsocketClient) -> None:
        """Should correctly decode a custom error from the server."""
        stream = sync_client.stream("/receiveAndExitWithErr", Message, Message)
        stream.send(Message(id=1, message="hello"))
        with pytest.raises(Error) as exc_info:
            stream.receive()
        assert exc_info.value.code == 1
        assert exc_info.value.message == "unexpected error"
        stream.close_send()

    def test_exit_immediately_with_err(self, sync_client: WebsocketClient) -> None:
        """Should correctly return the error to the stream"""
        stream = sync_client.stream("/immediatelyExitWithErr", Message, Message)
        for _ in range(100):
            try:
                stream.send(Message(id=1, message="hello"))
            except freighter.EOF:
                break
        with pytest.raises(Error) as exc_info:
            stream.receive()
        assert exc_info.value.code == 1
        assert exc_info.value.message == "unexpected error"
        stream.close_send()

    def test_with_custom_codec(self, sync_client: WebsocketClient) -> None:
        """Should correctly use a custom codec for the websocket client."""
        sync_client = sync_client.with_codec(MyVerySpecialCustomCodec())
        stream = sync_client.stream("/echo", Message, Message)
        for _ in range(1):
            stream.send(Message(id=12, message="what we send here is ignored"))
            msg = stream.receive()
            assert msg.id == 4201
            assert msg.message == "the key to the universe"

    def test_connection_closed_after_server_close(
        self, sync_client: WebsocketClient
    ) -> None:
        """Should open the connection on stream and close it once the server closes."""
        stream = sync_client.stream("/echo", Message, Message)
        state = stream._internal.state
        assert state is State.OPEN
        stream.close_send()
        with pytest.raises(freighter.EOF):
            stream.receive()
        state = stream._internal.state
        assert state is State.CLOSED


class TestContext:
    def test_sync_protocol_and_target(self, sync_client: WebsocketClient) -> None:
        """Should give middleware the websocket protocol and the stream target."""
        contexts: list[Context] = []

        def mw(ctx: Context, _: Next) -> Context:
            contexts.append(ctx)
            raise RuntimeError("stop")

        sync_client.use(mw)
        with pytest.raises(RuntimeError, match="stop"):
            sync_client.stream("/echo", Message, Message)
        assert contexts[0].protocol == "websocket"
        assert contexts[0].target == "/echo"

    @pytest.mark.asyncio
    async def test_async_protocol_and_target(
        self, async_client: AsyncWebsocketClient
    ) -> None:
        """Should give middleware the websocket protocol and the stream target."""
        contexts: list[Context] = []

        async def mw(ctx: Context, _: AsyncNext) -> Context:
            contexts.append(ctx)
            raise RuntimeError("stop")

        async_client.use(mw)
        with pytest.raises(RuntimeError, match="stop"):
            await async_client.stream("/echo", Message, Message)
        assert contexts[0].protocol == "websocket"
        assert contexts[0].target == "/echo"
