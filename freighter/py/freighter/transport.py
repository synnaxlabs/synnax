#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import typing
from typing import Protocol, TypeAlias, TypeVar

from pydantic import BaseModel

from freighter.context import Context

Payload: TypeAlias = BaseModel

RS = TypeVar("RS", bound=Payload, contravariant=True)
"""Represents a general response payload.
"""

RQ = TypeVar("RQ", bound=Payload, contravariant=True)
"""Represents a general request payload.
"""

P = TypeVar("P", bound=Payload)
"""Represents a general payload.
"""


class Transport(Protocol):
    """Base class for all transport protocols."""

    def use(self, *middleware: Middleware) -> None:
        """
        Adds middleware(s) to the transport.
        :param middleware: the middleware(s) to add
        """
        ...


class AsyncTransport(Protocol):
    """Base class for all asyncio.py transport protocols."""

    def use(self, *middleware: AsyncMiddleware) -> None:
        """
        Adds middleware(s) to the transport.
        :param middleware: the middleware(s) to add
        """
        ...


Next = typing.Callable[[Context], tuple[Context, Exception | None]]
"""Executes the next middleware in the chain"""

AsyncNext = typing.Callable[
    [Context], typing.Awaitable[tuple[Context, Exception | None]]
]
"""Executes the next middleware in the chain"""

Middleware = typing.Callable[[Context, Next], tuple[Context, Exception | None]]
""""Middleware is a general middleware function that can be used to parse/attach
metadata to a request or alter its behvaior."""

AsyncMiddleware = typing.Callable[
    [Context, AsyncNext], typing.Awaitable[tuple[Context, Exception | None]]
]
"""Middleware is a general middleware function that can be used to parse/attach
metadata to a request or alter its behvaior."""

Finalizer = typing.Callable[[Context], tuple[Context, Exception | None]]
"""Finalizer is a middleware that is executed as the last step in a chain.
It is used to finalize the request and return the response."""

AsyncFinalizer = typing.Callable[
    [Context], typing.Awaitable[tuple[Context, Exception | None]]
]
"""Finalizer is a middleware that is executed as the last step in a chain.
It is used to finalize the request and return the response."""


class MiddlewareCollector:
    """MiddlewareCollector collects and executes middleware in order."""

    _middleware: list[Middleware]

    def __init__(self):
        self._middleware = []

    def use(self, *args: Middleware) -> None:
        """Use implements the Transport protocol."""
        self._middleware.extend(args)

    def exec(
        self,
        ctx: Context,
        finalizer: Finalizer,
    ):
        """Executes the middleware in order, passing metadata to each
        middleware until the end of the chain is reached. It then calls
        the finalizer with the metadata.

        :param ctx: the context to pass to the middleware.
        :param finalizer: the finalizer to call at the end of the chain.
        """
        middleware = self._middleware.copy()

        def __next(ctx_: Context) -> tuple[Context, Exception | None]:
            if len(middleware) == 0:
                return finalizer(ctx_)
            return middleware.pop()(ctx_, __next)

        return __next(ctx)


class AsyncMiddlewareCollector:
    """AsyncMiddlewareCollector collects and executes middleware in order."""

    _middleware: list[AsyncMiddleware]

    def __init__(self):
        self._middleware = []

    def use(self, *args: AsyncMiddleware) -> None:
        """Use implements the Transport protocol."""
        self._middleware.extend(args)

    async def exec(
        self,
        md: Context,
        finalizer: AsyncFinalizer,
    ):
        """Executes the middleware in order, passing metadata to each
        middleware until the end of the chain is reached. It then calls
        the finalizer with the metadata.

        :param md: the metadata to pass to the middleware
        :param finalizer: the finalizer to call at the end of the chain
        """
        middleware = self._middleware.copy()

        async def __next(_md: Context) -> tuple[Context, Exception | None]:
            if len(middleware) == 0:
                return await finalizer(_md)
            return await middleware.pop()(_md, __next)

        return await __next(md)
