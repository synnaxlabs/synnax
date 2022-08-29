import pytest

import freighter.errors
from freighter import sync
from freighter import ws
from freighter.encoder import MsgpackEncoderDecoder
from freighter.endpoint import Endpoint
from .interface import Message, Error, message_factory


@pytest.fixture
def async_client(endpoint: Endpoint) -> ws.Client:
    ws_endpoint = endpoint.child("ws", protocol="ws")
    return ws.Client(encoder=MsgpackEncoderDecoder(), endpoint=ws_endpoint)


@pytest.fixture
def sync_client(async_client: ws.Client) -> sync.StreamClient:
    return sync.StreamClient(async_client)


class TestWS:

    async def test_basic_exchange(self, async_client: ws.Client):
        stream = await async_client.stream("/echo", Message, message_factory)
        for i in range(10):
            await stream.send(Message(i, "hello"))
            msg, err = await stream.receive()
            assert err is None
            assert msg.id == i + 1
            assert msg.message == "hello"
        await stream.close_send()
        msg, err = await stream.receive()
        assert err is not None

    async def test_receive_message_after_close(self, async_client: ws.Client):
        stream = await async_client.stream("/sendMessageAfterClientClose", Message,
                                           message_factory)
        await stream.close_send()
        msg, err = await stream.receive()
        assert err is None
        assert msg.id == 0
        assert msg.message == "Close Acknowledged"
        msg, err = await stream.receive()
        assert err is not None

    async def test_receive_error(self, async_client):
        stream = await async_client.stream("/receiveAndExitWithErr", Message,
                                           message_factory)
        await stream.send(Message(id=1, message="hello"))
        msg, err = await stream.receive()
        assert isinstance(err, Error)
        assert err.code == 1
        assert err.message == "unexpected error"


class TestSyncWebsocket:
    def test_basic_exchange(self, sync_client: sync.StreamClient):
        stream = sync_client.stream("/echo", Message, message_factory)
        for i in range(10):
            err = stream.send(Message(i, "hello"))
            assert err is None
            msg, err = stream.receive()
            assert err is None
            assert msg.id == i + 1
            assert msg.message == "hello"
        stream.close_send()
        msg, err = stream.receive()
        assert msg is None
        assert err is not None

    def test_repeated_receive(self, sync_client: sync.StreamClient):
        stream = sync_client.stream("/respondWithTenMessages", Message, message_factory)
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
