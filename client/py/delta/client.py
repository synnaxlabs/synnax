from freighter import (
    StreamClient,
    MsgpackEncoderDecoder,
    JSONEncoderDecoder
)

from freighter.ws import StreamClient as WSStreamClient


class Client:
    endpoint: str
    streamClient: StreamClient

    def __init__(self, endpoint: str):
        self.endpoint = endpoint
        self.streamClient = WSStreamClient(
            endpoint=self.endpoint + "/api/v1",
            encoder=MsgpackEncoderDecoder)
