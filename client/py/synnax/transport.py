#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import ssl
from collections.abc import Sequence

from pydantic import BaseModel, SerializeAsAny
from urllib3 import Retry, Timeout

from alamos import Instrumentation
from freighter import (
    URL,
    AsyncMiddleware,
    AsyncWebsocketClient,
    Empty,
    FileTransport,
    HTTPClient,
    Middleware,
    UnaryClient,
    WebsocketClient,
    async_instrumentation_middleware,
    instrumentation_middleware,
)
from synnax.telem import CrudeTimeSpan, Size, TimeSpan
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
        open_timeout: CrudeTimeSpan = TimeSpan.SECOND * 5,
        read_timeout: CrudeTimeSpan = TimeSpan.SECOND * 5,
        keep_alive: CrudeTimeSpan = TimeSpan.SECOND * 30,
        max_retries: int = 3,
        ca_cert: str | None = None,
    ) -> None:
        """
        :param registry: Decodes the errors that the Core returns.
        :param secure: Whether to use TLS. The client then verifies the Core's
            certificate against ca_cert, or the system trust roots when ca_cert is None.
        :param open_timeout: The time to wait for a connection. A bare number is
            seconds.
        :param read_timeout: The time to wait for a response. A bare number is seconds.
        :param keep_alive: The time between keep alive pings. A bare number is seconds.
        :param ca_cert: The path to a PEM file of CA certificates that sign the Core's
            certificate.
        :raises ValueError: If ca_cert is set and secure is False.
        """
        if ca_cert is not None and not secure:
            raise ValueError("ca_cert requires secure=True")
        open_timeout = TimeSpan.from_seconds(open_timeout)
        read_timeout = TimeSpan.from_seconds(read_timeout)
        keep_alive = TimeSpan.from_seconds(keep_alive)
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


class _BatchEntry(BaseModel):
    endpoint: str
    payload: SerializeAsAny[BaseModel]


class _BatchRequest(BaseModel):
    requests: list[_BatchEntry]


def send_batch(
    client: UnaryClient, endpoint: str, payloads: Sequence[BaseModel]
) -> None:
    """Sends one request to endpoint for each payload, in order, in one transaction.
    The first failure fails the batch, and nothing commits.

    :param client: The client to send the batch with.
    :param endpoint: The endpoint of each request, such as ``/user/dispatch``.
    :param payloads: The payload of each request.
    :raises ValidationError: If the batch does not accept the endpoint, or if a
        request is invalid.
    """
    send_requests(client, [(endpoint, p) for p in payloads])


def send_requests(
    client: UnaryClient, requests: Sequence[tuple[str, BaseModel]]
) -> None:
    """Sends each request to its endpoint, in order, in one transaction. The first
    failure fails the batch, and nothing commits.

    :param client: The client to send the batch with.
    :param requests: The endpoint and payload of each request.
    :raises ValidationError: If the batch does not accept an endpoint, or if a request
        is invalid.
    """
    entries = [_BatchEntry(endpoint=e, payload=p) for e, p in requests]
    client.send("/batch", _BatchRequest(requests=entries), Empty)
