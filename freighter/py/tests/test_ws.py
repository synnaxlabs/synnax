#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
import time

import pytest

import freighter.exceptions
from freighter.context import Context
from freighter.codec import MsgPackCodec, JSONCodec
from freighter.transport import AsyncNext, Next
from freighter.url import URL
from freighter.websocket import AsyncWebsocketClient, WebsocketClient, ConnectionClosedError
from freighter.http import HTTPClient

from .interface import Error, Message
from uuid import uuid4


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
class TestWS:
    async def test_basic_exchange(self, async_client: AsyncWebsocketClient):
        """Should exchange ten echo messages that increment the ID."""
        stream = await async_client.stream("/echo", Message, Message)
        for i in range(10):
            await stream.send(Message(id=i, message="hello"))
            msg, err = await stream.receive()
            assert err is None
            assert msg.id == i + 1
            assert msg.message == "hello"
        await stream.close_send()
        msg, err = await stream.receive()
        assert err is not None

    async def test_receive_message_after_close(
        self, async_client: AsyncWebsocketClient
    ):
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
        assert msg.id == 0
        assert msg.message == "Close Acknowledged"
        msg, err = await stream.receive()
        assert isinstance(err, freighter.EOF)

    async def test_receive_error(self, async_client):
        """Should correctly decode a custom error from the server."""
        stream = await async_client.stream("/receiveAndExitWithErr", Message, Message)
        await stream.send(Message(id=1, message="hello"))
        msg, err = await stream.receive()
        assert isinstance(err, Error)
        assert err.code == 1
        assert err.message == "unexpected error"
        await stream.close_send()

    async def test_middleware(self, async_client):
        dct = {"called": False}

        async def mw(md: Context, next: AsyncNext) -> Exception | None:
            md.params["Test"] = "test"
            dct["called"] = True
            return await next(md)

        async_client.use(mw)
        stream = await async_client.stream("/middlewareCheck", Message, Message)
        await stream.close_send()
        _, err = await stream.receive()
        assert isinstance(err, freighter.EOF)
        assert dct["called"]

    async def test_server_timeout(self, async_client, unary_client):
        """Should correctly timeout if the server exceeds a write deadline"""
        stream = await async_client.stream("/slamMessages", Message, Message)
        msg_str = str(uuid4())
        await stream.send(Message(id=1, message=msg_str))
        time.sleep(5)
        res, err = unary_client.send(
            "/slamMessagesTimeoutCheck",
            Message(id=1, message=msg_str),
            Message
        )
        assert err is None
        assert res.message == "timeout"
        with pytest.raises(ConnectionClosedError):
            while True:
                _, err = await stream.receive()
                if isinstance(err, freighter.EOF):
                    break



class TestSyncWebsocket:
    def test_basic_exchange(self, sync_client: WebsocketClient):
        stream = sync_client.stream("/echo", Message, Message)
        for i in range(10):
            err = stream.send(Message(id=i, message="hello"))
            assert err is None
            msg, err = stream.receive()
            assert err is None
            assert msg.id == i + 1
            assert msg.message == "hello"
        stream.close_send()
        msg, err = stream.receive()
        assert msg is None
        assert err is not None

    def test_repeated_receive(self, sync_client: WebsocketClient):
        """Should receive ten messages from the server."""
        stream = sync_client.stream("/respondWithTenMessages", Message, Message)
        c = 0
        while True:
            msg, err = stream.receive()
            if isinstance(err, freighter.EOF):
                break
            c += 1
            assert err is None
            assert msg.message == "hello"
        stream.close_send()
        assert c == 10
        _, err = stream.receive()

    def test_middleware(self, sync_client: WebsocketClient):
        """Should receive ten messages from the server."""
        dct = {"called": False}

        def mw(md: Context, next: Next) -> Exception | None:
            md.params["Test"] = "test"
            dct["called"] = True
            return next(md)

        sync_client.use(mw)
        stream = sync_client.stream("/middlewareCheck", Message, Message)
        stream.close_send()
        _, err = stream.receive()
        assert isinstance(err, freighter.EOF)
        assert dct["called"]

    def test_client_timeout(self, sync_client: WebsocketClient):
        """Should correctly timeout if the server exceeds a write deadline"""
        stream = sync_client.stream("/echo", Message, Message)
        with pytest.raises(TimeoutError):
            stream.receive(timeout=0.1)
        stream.close_send()
        while True:
            _, err = stream.receive()
            if isinstance(err, freighter.EOF):
                break
