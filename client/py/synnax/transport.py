from freighter import (
    URL,
    AsyncStreamClient,
    HTTPClient,
    JSONEncoder,
    MsgpackEncoder,
    StreamClient,
    SyncStreamClient,
    WebsocketClient,
)


class Transport:
    endpoint: URL
    stream: StreamClient
    stream_async: AsyncStreamClient
    http: HTTPClient

    def __init__(self, url: URL) -> None:
        self.endpoint = url.child("/api/v1/")
        self.stream_async = WebsocketClient(
            endpoint=self.endpoint, encoder=MsgpackEncoder(), max_message_size=int(5e6)
        )
        self.stream = SyncStreamClient(self.stream_async)
        self.http = HTTPClient(endpoint=self.endpoint, encoder_decoder=JSONEncoder())
