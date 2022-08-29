from freighter import Endpoint, StreamClient, AsyncStreamClient, UnaryClient
from freighter import (
    ws,
    sync,
    http,
    encoder,
)


class Transport:
    endpoint: Endpoint
    stream: StreamClient
    stream_async: AsyncStreamClient
    http: http.Client

    def __init__(self, host: str, port: int) -> None:
        self.endpoint = Endpoint(host, port).child("/api/v1/")
        self.stream_async = ws.Client(endpoint=self.endpoint, encoder=encoder.Msgpack())
        self.stream = sync.StreamClient(self.stream_async)
        self.http = http.Client(endpoint=self.endpoint, encoder_decoder=encoder.JSON())
