#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from freighter import (
    URL,
    AsyncStreamClient,
    HTTPClientFactory,
    JSONEncoder,
    MsgpackEncoder,
    StreamClient,
    SyncStreamClient,
    WebsocketClient,
    Middleware,
    AsyncMiddleware,
)


class Transport:
    url: URL
    stream: StreamClient
    stream_async: AsyncStreamClient
    http: HTTPClientFactory
    secure: bool

    def __init__(self, url: URL, secure: bool = False) -> None:
        self.url = url.child("/api/v1/")
        self.stream_async = WebsocketClient(
            base_url=self.url,
            encoder=MsgpackEncoder(),
            max_message_size=int(5e6),
            secure=secure,
        )
        self.stream = SyncStreamClient(self.stream_async)
        self.http = HTTPClientFactory(
            url=self.url,
            encoder_decoder=JSONEncoder(),
            secure=secure,
        )

    def use(self, *middleware: Middleware):
        self.http.use(*middleware)
        self.stream.use(*middleware)

    def use_async(self, *middleware: AsyncMiddleware):
        self.http.use(*middleware)
        self.stream_async.use(*middleware)
