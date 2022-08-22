from freighter.ws import StreamClient
from freighter.encoder import MsgpackEncoderDecoder
from .interface import Message, Error


class TestWebsocket:
    async def test_basic_exchange(self):
        client = StreamClient[Message, Message](
            encoder=MsgpackEncoderDecoder,
            endpoint="localhost:8080"
        )
        stream = await client.stream("/echo")
        for i in range(10):
            await stream.send(Message(i, "hello"))
            msg = Message(None, None)
            err = await stream.receive(msg)
            assert err is None
            assert msg.id == i + 1
            assert msg.message == "hello"
        await stream.close_send()
        err = await stream.receive(Message(None, None))
        assert err is not None

    async def test_receive_message_after_close(self):
        client = StreamClient[Message, Message](
            encoder=MsgpackEncoderDecoder,
            endpoint="localhost:8080"
        )
        stream = await client.stream("/sendMessageAfterClientClose")
        await stream.close_send()
        msg = Message(None, None)
        err = await stream.receive(msg)
        assert err is None
        assert msg.id == 0
        assert msg.message == "Close Acknowledged"
        err = await stream.receive(Message(None, None))
        assert err is not None

    async def test_receive_error(self):
        client = StreamClient[Message, Message](
            encoder=MsgpackEncoderDecoder,
            endpoint="localhost:8080"
        )
        stream = await client.stream("/receiveAndExitWithErr")
        await stream.send(Message(id=1,message="hello"))
        msg = Message(None, None)
        err = await stream.receive(msg)
        assert isinstance(err, Error)
        assert err.code == 1
        assert err.message == "unexpected error"
