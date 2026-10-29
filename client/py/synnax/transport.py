#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl

from urllib3 import Retry, Timeout

from alamos import Instrumentation
from freighter import (
    URL,
    AsyncMiddleware,
    AsyncWebsocketClient,
    FileTransport,
    HTTPClient,
    Middleware,
    UnaryClient,
    WebsocketClient,
    async_instrumentation_middleware,
    instrumentation_middleware,
)
from synnax.telem import Size, TimeSpan
from x.codec import JSONCodec
from x.exceptions import Registry


class Transport:
    url: URL
    stream: WebsocketClient
    stream_async: AsyncWebsocketClient
    unary: UnaryClient
    file_transport: FileTransport

    def __init__(
        self,
        *,
        url: URL,
        instrumentation: Instrumentation,
        registry: Registry,
        secure: bool = False,
        open_timeout: TimeSpan = TimeSpan.SECOND * 5,
        read_timeout: TimeSpan = TimeSpan.SECOND * 5,
        keep_alive: TimeSpan = TimeSpan.SECOND * 30,
        max_retries: int = 3,
        ca_cert: str | None = None,
    ) -> None:
        """
        :param registry: Decodes the errors that the Core returns.
        :param secure: Whether to use TLS. The client then verifies the Core's
            certificate against ca_cert, or the system trust roots when ca_cert is None.
        :param ca_cert: The path to a PEM file of CA certificates that sign the Core's
            certificate.
        :raises ValueError: If ca_cert is set and secure is False.
        """
        if ca_cert is not None and not secure:
            raise ValueError("ca_cert requires secure=True")
        ssl_context = ssl.create_default_context(cafile=ca_cert) if secure else None
        self.url = url.child("/api/v1/")
        codec = JSONCodec()
        max_message_size = int(Size.MB * 5)
        ws_options = {
            "open_timeout": open_timeout.seconds,
            "close_timeout": read_timeout.seconds,
            "ping_interval": keep_alive.seconds,
            "ping_timeout": 180,
        }
        self.stream = WebsocketClient(
            encoder=codec,
            registry=registry,
            base_url=self.url,
            max_message_size=max_message_size,
            ssl_context=ssl_context,
            **ws_options,
        )
        self.stream_async = AsyncWebsocketClient(
            encoder=codec,
            registry=registry,
            base_url=self.url,
            max_message_size=max_message_size,
            ssl_context=ssl_context,
            **ws_options,
        )
        http = HTTPClient(
            url=self.url,
            codec=codec,
            registry=registry,
            ssl_context=ssl_context,
            timeout=Timeout(connect=open_timeout.seconds, read=read_timeout.seconds),
            retries=Retry(total=max_retries),
        )
        self.unary = http
        self.file_transport = http
        self.use(instrumentation_middleware(instrumentation))
        self.use_async(async_instrumentation_middleware(instrumentation))

    def use(self, *middleware: Middleware) -> None:
        self.unary.use(*middleware)
        self.stream.use(*middleware)

    def use_async(self, *middleware: AsyncMiddleware) -> None:
        self.stream_async.use(*middleware)
