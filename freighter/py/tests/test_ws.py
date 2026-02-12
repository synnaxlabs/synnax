#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import time
from uuid import uuid4

import pytest

import freighter.exceptions
from freighter.codec import Codec, JSONCodec, MsgPackCodec
from freighter.context import Context
from freighter.http import HTTPClient
from freighter.transport import AsyncNext, Next, P
from freighter.url import URL
from freighter.websocket import (
    AsyncWebsocketClient,
    ConnectionClosedError,
)
from freighter.websocket import Message as WebsocketMessage
from freighter.websocket import WebsocketClient
from pydantic import BaseModel

from .interface import Error, Message


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
    return AsyncWebsocketClient(encoder=MsgPackCodec(), base_url=ws_endpoint)


@pytest.fixture
def sync_client(endpoint: URL) -> WebsocketClient:
    ws_endpoint = endpoint.child("stream")
    return WebsocketClient(encoder=MsgPackCodec(), base_url=ws_endpoint)


@pytest.fixture
def unary_client(endpoint: URL) -> HTTPClient:
    http_endpoint = endpoint.child("unary")
    return HTTPClient(http_endpoint, JSONCodec())


@pytest.mark.ws
@pytest.mark.asyncio
class TestAsyncWebsocket:
    async def test_basic_exchange(self, async_client: AsyncWebsocketClient) -> None:
        """Should exchange ten echo messages that increment the ID."""
        stream = await async_client.stream("/echo", Message, Message)
        for i in range(10):
            await stream.send(Message(id=i, message="hello"))
            msg, err = await stream.receive()
            assert err is None
            assert msg is not None
            assert msg.id == i + 1
            assert msg.message == "hello"
        await stream.close_send()
        msg, err = await stream.receive()
        assert err is not None

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
        msg, err = await stream.receive()
        assert err is None
        assert msg is not None
        assert msg.id == 0
        assert msg.message == "Close Acknowledged"
        msg, err = await stream.receive()
        assert isinstance(err, freighter.EOF)

    async def test_receive_error(self, async_client: AsyncWebsocketClient) -> None:
        """Should correctly decode a custom error from the server."""
        stream = await async_client.stream("/receiveAndExitWithErr", Message, Message)
        await stream.send(Message(id=1, message="hello"))
        msg, err = await stream.receive()
        assert isinstance(err, Error)
        assert err.code == 1
        assert err.message == "unexpected error"
        await stream.close_send()

    async def test_middleware(self, async_client: AsyncWebsocketClient) -> None:
        dct = {"called": False}

        async def mw(md: Context, next: AsyncNext) -> tuple[Context, Exception | None]:
            md.params["Test"] = "test"
            dct["called"] = True
            return await next(md)

        async_client.use(mw)
        stream = await async_client.stream("/middlewareCheck", Message, Message)
        await stream.close_send()
        _, err = await stream.receive()
        assert isinstance(err, freighter.EOF)
        assert dct["called"]

    async def test_server_timeout(
        self, async_client: AsyncWebsocketClient, unary_client: HTTPClient
    ) -> None:
        """Should correctly timeout if the server exceeds a write deadline"""
        stream = await async_client.stream("/slamMessages", Message, Message)
        msg_str = str(uuid4())
        await stream.send(Message(id=1, message=msg_str))
        time.sleep(2)
        res, err = unary_client.send(
            "/slamMessagesTimeoutCheck", Message(id=1, message=msg_str), Message
        )
        assert err is None
        assert res is not None
        assert res.message == "timeout"
        with pytest.raises(ConnectionClosedError):
            while True:
                _, err = await stream.receive()
                if isinstance(err, freighter.EOF):
                    break

    async def test_with_custom_codec(self, async_client: AsyncWebsocketClient) -> None:
        """Should correctly use a custom codec for the websocket client."""
        async_client = async_client.with_codec(MyVerySpecialCustomCodec())
        stream = await async_client.stream("/echo", Message, Message)
        for i in range(1):
            err = await stream.send(
                Message(id=12, message="what we send here is ignored")
            )
            assert err is None
            msg, err = await stream.receive()
            assert err is None
            assert msg is not None
            assert msg.id == 4201
            assert msg.message == "the key to the universe"


class TestSyncWebsocket:
    def test_basic_exchange(self, sync_client: WebsocketClient) -> None:
        stream = sync_client.stream("/echo", Message, Message)
        for i in range(10):
            err = stream.send(Message(id=i, message="hello"))
            assert err is None
            msg, err = stream.receive()
            assert err is None
            assert msg is not None
            assert msg.id == i + 1
            assert msg.message == "hello"
        stream.close_send()
        msg, err = stream.receive()
        assert msg is None
        assert err is not None

    def test_repeated_receive(self, sync_client: WebsocketClient) -> None:
        """Should receive ten messages from the server."""
        stream = sync_client.stream("/respondWithTenMessages", Message, Message)
        c = 0
        while True:
            msg, err = stream.receive()
            if isinstance(err, freighter.EOF):
                break
            c += 1
            assert err is None
            assert msg is not None
            assert msg.message == "hello"
        stream.close_send()
        assert c == 10
        _, err = stream.receive()

    def test_middleware(self, sync_client: WebsocketClient) -> None:
        """Should receive ten messages from the server."""
        dct = {"called": False}

        def mw(md: Context, next: Next) -> tuple[Context, Exception | None]:
            md.params["Test"] = "test"
            dct["called"] = True
            return next(md)

        sync_client.use(mw)
        stream = sync_client.stream("/middlewareCheck", Message, Message)
        stream.close_send()
        _, err = stream.receive()
        assert isinstance(err, freighter.EOF)
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
            _, err = stream.receive()
            if isinstance(err, freighter.EOF):
                break

    def test_timeout_0(self, sync_client: WebsocketClient) -> None:
        """Should correctly return a frame if and when available"""
        stream = sync_client.stream("/eventuallyResponseWithMessage", Message, Message)
        stream.send(Message(id=1, message="hello"))
        cycle_count = 0
        sleep = 0.05
        dur = 0.25
        max_cycles = (dur / sleep) + 1
        while True:
            if cycle_count > max_cycles:
                break
            try:
                time.sleep(sleep)
                msg, err = stream.receive(timeout=0)
                assert err is None
                assert msg is not None
                assert msg.id == 1
                break
            except TimeoutError:
                cycle_count += 1
                pass
        assert cycle_count < max_cycles, "test timed out"

    def test_receive_error(self, sync_client: WebsocketClient) -> None:
        """Should correctly decode a custom error from the server."""
        stream = sync_client.stream("/receiveAndExitWithErr", Message, Message)
        err = stream.send(Message(id=1, message="hello"))
        assert err is None
        msg, err = stream.receive()
        assert isinstance(err, Error)
        assert err.code == 1
        assert err.message == "unexpected error"
        stream.close_send()

    def test_exit_immediately_with_err(self, sync_client: WebsocketClient) -> None:
        """Should correctly return the error to the stream"""
        stream = sync_client.stream("/immediatelyExitWithErr", Message, Message)
        for i in range(100):
            stream.send(Message(id=1, message="hello"))
        msg, err = stream.receive()
        assert isinstance(err, Error)
        assert err.code == 1
        assert err.message == "unexpected error"
        stream.close_send()

    def test_with_custom_codec(self, sync_client: WebsocketClient) -> None:
        """Should correctly use a custom codec for the websocket client."""
        sync_client = sync_client.with_codec(MyVerySpecialCustomCodec())
        stream = sync_client.stream("/echo", Message, Message)
        for i in range(1):
            err = stream.send(Message(id=12, message="what we send here is ignored"))
            assert err is None
            msg, err = stream.receive()
            assert err is None
            assert msg is not None
            assert msg.id == 4201
            assert msg.message == "the key to the universe"
