from freighter import MsgpackEncoderDecoder, Endpoint
from freighter.ws import Client as WSClient
from freighter.sync import StreamClient

from arya.segment.writer import (
    WriterRequest,
    WriterResponse,
    NumpyWriter,
    Core
)


class Client:
    endpoint: Endpoint

    def write(self, keys: list[str]) -> NumpyWriter:
        transport = StreamClient[WriterRequest, WriterResponse](
            WSClient[WriterRequest, WriterResponse](
                encoder=MsgpackEncoderDecoder(),
                endpoint=self.endpoint,
            ),
        )
        core = Core(transport=transport)
        npw = NumpyWriter(core=core)
