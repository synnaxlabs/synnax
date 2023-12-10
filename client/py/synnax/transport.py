#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation
from freighter import (
    URL,
    AsyncMiddleware,
    AsyncStreamClient,
    HTTPClient,
    JSONEncoder,
    Middleware,
    MsgpackEncoder,
    StreamClient,
    SyncStreamClient,
    UnaryClient,
    WebsocketClient,
    async_instrumentation_middleware,
    instrumentation_middleware,
)
from urllib3 import Retry, Timeout

from synnax.telem import Size, TimeSpan


class Transport:
    url: URL
    stream: StreamClient
    stream_async: AsyncStreamClient
    unary: UnaryClient
    secure: bool

    def __init__(
        self,
        url: URL,
        instrumentation: Instrumentation,
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
            max_message_size=int(Size.MB * 5),
            secure=secure,
            open_timeout=open_timeout.seconds,
            ping_interval=keep_alive.seconds,
            close_timeout=read_timeout.seconds,
            ping_timeout=180,
        )
        self.stream = SyncStreamClient(self.stream_async)
        self.unary = HTTPClient(
            url=self.url,
            encoder_decoder=JSONEncoder(),
            secure=secure,
            timeout=Timeout(connect=open_timeout.seconds, read=read_timeout.seconds),
            retries=Retry(total=max_retries),
        )
        self.use(instrumentation_middleware(instrumentation))
        self.use_async(async_instrumentation_middleware(instrumentation))

    def use(self, *middleware: Middleware):
        self.unary.use(*middleware)
        self.stream.use(*middleware)

    def use_async(self, *middleware: AsyncMiddleware):
        self.stream_async.use(*middleware)
