from freighter import (
    MsgpackEncoderDecoder,
    Endpoint,
    StreamClient,
    AsyncStreamClient,
    UnaryClient
)
from freighter.ws import Client as WSClient
from freighter.sync import StreamClient as SyncStreamClient


class Transport:
    stream: StreamClient
    stream_async: AsyncStreamClient
    create: UnaryClient
    retrieve: UnaryClient

    def __init__(self, endpoint: Endpoint) -> None:
        self.stream_async = WSClient(
            endpoint=endpoint,
            encoder=MsgpackEncoderDecoder(),
        )
        self.stream = SyncStreamClient(self.stream_async)



