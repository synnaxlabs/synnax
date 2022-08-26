import time

import freighter.errors
from freighter import sync
from freighter.ws import AsyncWSClient
from freighter.encoder import MsgpackEncoderDecoder
from freighter.endpoint import Endpoint
from .interface import Message, Error

message_factory = lambda: Message(None, None)

endpoint = Endpoint("", "localhost", 8080)


class TestWS:
    async def test_basic_exchange(self):
        client = AsyncWSClient[Message, Message](
            encoder=MsgpackEncoderDecoder, endpoint=endpoint
        )
        stream = await client.stream("/echo", message_factory)
        for i in range(10):
            await stream.send(Message(i, "hello"))
            msg, err = await stream.receive()
            assert err is None
            assert msg.id == i + 1
            assert msg.message == "hello"
        await stream.close_send()
        msg, err = await stream.receive()
        assert err is not None
        assert msg is None

    async def test_receive_message_after_close(self):
        client = AsyncWSClient[Message, Message](
            encoder=MsgpackEncoderDecoder, endpoint=endpoint
        )
        stream = await client.stream("/sendMessageAfterClientClose", message_factory)
        await stream.close_send()
        msg, err = await stream.receive()
        assert err is None
        assert msg.id == 0
        assert msg.message == "Close Acknowledged"
        msg, err = await stream.receive()
        assert err is not None
        assert msg is None

    async def test_receive_error(self):
        client = AsyncWSClient[Message, Message](
            encoder=MsgpackEncoderDecoder, endpoint=endpoint
        )
        stream = await client.stream("/receiveAndExitWithErr", message_factory)
        await stream.send(Message(id=1, message="hello"))
        msg, err = await stream.receive()
        assert msg is None
        assert isinstance(err, Error)
        assert err.code == 1
        assert err.message == "unexpected error"


class TestSyncWebsocket:
    def test_basic_exchange(self):
        client = sync.StreamClient(AsyncWSClient(
            encoder=MsgpackEncoderDecoder,
            endpoint=endpoint
        ))
        stream = client.stream("/echo", message_factory)
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

    def test_repeated_receive(self):
        client = sync.StreamClient(AsyncWSClient(
            encoder=MsgpackEncoderDecoder,
            endpoint=endpoint
        ))
        stream = client.stream("/respondWithTenMessages", message_factory)
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
