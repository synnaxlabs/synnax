#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from collections.abc import Awaitable, Callable
from typing import Protocol, TypeVar

from pydantic import BaseModel

from freighter.context import Context


class Empty(BaseModel):
    """Empty represents an empty payload."""

    pass


RQ = TypeVar("RQ", bound=BaseModel, contravariant=True)
"""Represents a general request payload."""

RS = TypeVar("RS", bound=BaseModel, covariant=True)
"""Represents a general response payload."""


P = TypeVar("P", bound=BaseModel)
"""Represents a general payload."""


class Transport(Protocol):
    """Base class for all transport protocols."""

    def use(self, *middleware: Middleware) -> None:
        """
        Adds middleware(s) to the transport.
        :param middleware: the middleware(s) to add
        """
        ...


class AsyncTransport(Protocol):
    """Base class for all asyncio-based transport protocols."""

    def use(self, *middleware: AsyncMiddleware) -> None:
        """
        Adds middleware(s) to the transport.
        :param middleware: the middleware(s) to add
        """
        ...


Next = Callable[[Context], Context]
"""Executes the next middleware in the chain, returning the response context. Raises if
the request fails."""

AsyncNext = Callable[[Context], Awaitable[Context]]
"""Executes the next middleware in the chain, returning the response context. Raises if
the request fails."""

Middleware = Callable[[Context, Next], Context]
"""
Middleware is a general middleware function that can be used to parse or attach metadata
to a request or alter its behavior. It returns the response context and raises if the
request fails.
"""

AsyncMiddleware = Callable[[Context, AsyncNext], Awaitable[Context]]
"""
AsyncMiddleware is a general middleware function that can be used to parse or attach
metadata to a request or alter its behavior. It returns the response context and raises
if the request fails.
"""

Finalizer = Callable[[Context], Context]
"""
Finalizer is a middleware that is executed as the last step in a chain. It is used to
finalize the request and return the response context, raising if the request fails.
"""

AsyncFinalizer = Callable[[Context], Awaitable[Context]]
"""
AsyncFinalizer is a middleware that is executed as the last step in a chain. It is used
to finalize the request and return the response context, raising if the request fails.
"""


class MiddlewareCollector:
    """MiddlewareCollector collects and executes middleware in order."""

    _middleware: list[Middleware]

    def __init__(self) -> None:
        self._middleware = []

    def use(self, *args: Middleware) -> None:
        """Use implements the Transport protocol."""
        self._middleware.extend(args)

    def exec(self, ctx: Context, finalizer: Finalizer) -> Context:
        """
        Executes the middleware in order, passing metadata to each middleware until the
        end of the chain is reached. It then calls the finalizer with the metadata.

        :param ctx: the context to pass to the middleware.
        :param finalizer: the finalizer to call at the end of the chain.
        :returns: the response context.
        :raises Exception: if any middleware or the finalizer fails.
        """
        middleware = self._middleware.copy()

        def _next(ctx_: Context) -> Context:
            if len(middleware) == 0:
                return finalizer(ctx_)
            return middleware.pop()(ctx_, _next)

        return _next(ctx)


class AsyncMiddlewareCollector:
    """AsyncMiddlewareCollector collects and executes middleware in order."""

    _middleware: list[AsyncMiddleware]

    def __init__(self) -> None:
        self._middleware = []

    def use(self, *args: AsyncMiddleware) -> None:
        """Use implements the Transport protocol."""
        self._middleware.extend(args)

    async def exec(self, md: Context, finalizer: AsyncFinalizer) -> Context:
        """
        Executes the middleware in order, passing metadata to each middleware until the
        end of the chain is reached. It then calls the finalizer with the metadata.

        :param md: the metadata to pass to the middleware
        :param finalizer: the finalizer to call at the end of the chain
        :returns: the response context.
        :raises Exception: if any middleware or the finalizer fails.
        """
        middleware = self._middleware.copy()

        async def _next(_md: Context) -> Context:
            if len(middleware) == 0:
                return await finalizer(_md)
            return await middleware.pop()(_md, _next)

        return await _next(md)
