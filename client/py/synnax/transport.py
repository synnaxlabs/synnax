from freighter import (
    URL,
    AsyncStreamClient,
    HTTPClientFactory,
    JSONEncoder,
    MsgpackEncoder,
    StreamClient,
    SyncStreamClient,
    WebsocketClient,
)


class Transport:
    url: URL
    stream: StreamClient
    stream_async: AsyncStreamClient
    http: HTTPClientFactory

    def __init__(self, url: URL) -> None:
        self.url = url.child("/api/v1/")
        self.stream_async = WebsocketClient(
            base_url=self.url, encoder=MsgpackEncoder(), max_message_size=int(5e6)
        )
        self.stream = SyncStreamClient(self.stream_async)
        self.http = HTTPClientFactory(url=self.url, encoder_decoder=JSONEncoder())
