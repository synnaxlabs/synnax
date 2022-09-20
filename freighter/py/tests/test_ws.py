import pytest

import freighter.errors
from freighter.encoder import Msgpack
from freighter.sync import SyncStreamClient
from freighter.url import URL
from freighter.websocket import WebsocketClient

from .interface import Error, Message


@pytest.fixture
def async_client(endpoint: URL) -> WebsocketClient:
    ws_endpoint = endpoint.child("ws")
    return WebsocketClient(encoder=Msgpack(), endpoint=ws_endpoint)


@pytest.fixture
def sync_client(async_client: WebsocketClient) -> SyncStreamClient:
    return SyncStreamClient(async_client)


class TestWS:
    async def test_basic_exchange(self, async_client: WebsocketClient):
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

    async def test_receive_message_after_close(self, async_client: WebsocketClient):
        """Should receive a message and EOF error after the server closes the connection."""
        stream = await async_client.stream(
            "/sendMessageAfterClientClose", Message, Message
        )
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


class TestSyncWebsocket:
    def test_basic_exchange(self, sync_client: SyncStreamClient):
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

    def test_repeated_receive(self, sync_client: SyncStreamClient):
        """Should receive ten messages from the server."""
        stream = sync_client.stream("/respondWithTenMessages", Message, Message)
        c = 0
        while True:
            msg, err = stream.receive()
            if isinstance(err, freighter.errors.EOF):
                break
            c += 1
            assert err is None
            assert msg.message == "hello"
        stream.close_send()
        assert c == 10
