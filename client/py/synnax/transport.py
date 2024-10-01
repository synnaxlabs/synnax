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
    JSONCodec,
    Middleware,
    MsgPackCodec,
    StreamClient,
    UnaryClient,
    WebsocketClient,
    AsyncWebsocketClient,
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
        ws_args = {
            "base_url": self.url,
            "encoder": MsgPackCodec(),
            "max_message_size": int(Size.MB * 5),
            "secure": secure,
            "open_timeout": open_timeout.seconds,
            "close_timeout": read_timeout.seconds,
        }
        self.stream = WebsocketClient(**ws_args)
        # We need to update these here because the websocket client doesn't support
        # the same arguments as the async websocket client.
        ws_args["ping_interval"] = keep_alive.seconds
        ws_args["ping_timeout"] = 180
        self.stream_async = AsyncWebsocketClient(**ws_args)
        self.unary = HTTPClient(
            url=self.url,
            codec=JSONCodec(),
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
