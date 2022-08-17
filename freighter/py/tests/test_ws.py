from freighter.ws import StreamClient
from freighter.encoder import MsgpackEncoderDecoder, JSONEncoderDecoder
from freighter.errors import EOF
from dataclasses import dataclass
import asyncio


@dataclass
class SampleRequest:
    id: int
    message: str


@dataclass
class SampleResponse:
    id: int
    message: str


class TestWS:
    async def test_ws(self):
        client = StreamClient[SampleRequest, SampleResponse](JSONEncoderDecoder)
        stream = await client.stream("ws://localhost:8080/")
        await stream.send(SampleRequest(1, "hello"))
        await stream.close_send()
        while True:
            err = await stream.receive(SampleResponse(1, "hello"))
            if isinstance(err, EOF):
                break
            if err is not None:
                raise err


