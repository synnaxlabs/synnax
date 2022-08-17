from freighter import (
    StreamClient,
    MsgpackEncoderDecoder,
    JSONEncoderDecoder
)

from freighter.ws import StreamClient as WSStreamClient
from delta.segment.writer import Writer


class Client:
    endpoint: str
    streamClient: StreamClient

    def __init__(self, endpoint: str):
        self.endpoint = endpoint
        self.streamClient = WSStreamClient(
            endpoint=self.endpoint + "/api/v1",
            encoder=MsgpackEncoderDecoder)

    async def write(self, keys: list[str]) -> Writer:
        w = Writer(self.streamClient)
        await w.open(keys)
        return w
