#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from urllib3 import Timeout, Retry

from freighter import (
    URL,
    AsyncMiddleware,
    AsyncStreamClient,
    HTTPClientPool,
    JSONEncoder,
    Middleware,
    MsgpackEncoder,
    StreamClient,
    SyncStreamClient,
    WebsocketClient,
)

from synnax.telem import TimeSpan


class Transport:
    url: URL
    stream: StreamClient
    stream_async: AsyncStreamClient
    http: HTTPClientPool
    secure: bool

    def __init__(
        self,
        url: URL,
        secure: bool = False,
        open_timeout: TimeSpan = TimeSpan.SECOND * 5,
        read_timeout: TimeSpan = TimeSpan.SECOND * 5,
        keep_alive: TimeSpan = TimeSpan.SECOND * 30,
        max_retries: int = 3,
    ) -> None:
        self.url = url.child("/api/v1/")
        self.stream_async = WebsocketClient(
            base_url=self.url,
            encoder=MsgpackEncoder(),
            max_message_size=int(5e6),
            secure=secure,
            open_timeout=open_timeout.seconds(),
            ping_interval=keep_alive.seconds(),
            close_timeout=read_timeout.seconds(),
            ping_timeout=read_timeout.seconds(),
        )
        self.stream = SyncStreamClient(self.stream_async)
        self.http = HTTPClientPool(
            url=self.url,
            encoder_decoder=JSONEncoder(),
            secure=secure,
            timeout=Timeout(
                connect=open_timeout.seconds(), read=read_timeout.seconds()
            ),
            retries=Retry(total=max_retries),
        )

    def use(self, *middleware: Middleware):
        self.http.use(*middleware)
        self.stream.use(*middleware)

    def use_async(self, *middleware: AsyncMiddleware):
        self.stream_async.use(*middleware)
