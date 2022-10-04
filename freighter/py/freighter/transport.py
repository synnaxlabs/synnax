from __future__ import annotations

import typing
from typing import TypeAlias, TypeVar, Protocol

from .metadata import MetaData

from pydantic import BaseModel

Payload: TypeAlias = BaseModel

RS = TypeVar("RS", bound=Payload, covariant=True)
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

    def use(self, *middleware: Middleware) -> None:
        """
        Adds middleware(s) to the transport.
        :param middleware: the middleware(s) to add
        """
        ...


Next = typing.Callable[[MetaData], Exception | None]
"""Executes the next middleware in the chain"""

AsyncNext = typing.Callable[[MetaData], typing.Awaitable[Exception | None]]
"""Executes the next middleware in the chain"""


Middleware = typing.Callable[[MetaData, Next], Exception | None]
""""Middleware is a general middleware function that can be used to parse/attach
metadata to a request or alter its behvaior."""

AsyncMiddleware = typing.Callable[
    [MetaData, AsyncNext], typing.Awaitable[Exception | None]
]
"""Middleware is a general middleware function that can be used to parse/attach
metadata to a request or alter its behvaior."""

Finalizer = typing.Callable[[MetaData], Exception | None]
"""Finalizer is a middleware that is executed as the last step in a chain.
It is used to finalize the request and return the response."""

AsyncFinalizer = typing.Callable[[MetaData], typing.Awaitable[Exception | None]]
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
        md: MetaData,
        finalizer: Finalizer,
    ):
        """Executes the middleware in order, passing metadata to each
        middleware until the end of the chain is reached. It then calls
        the finalizer with the metadata.

        :param md: the metadata to pass to the middleware
        :param finalizer: the finalizer to call at the end of the chain
        """
        middleware = self._middleware.copy()

        def _next(_md: MetaData) -> Exception | None:
            if len(middleware) == 0:
                return finalizer(_md)
            return middleware.pop()(_md, _next)

        return _next(md)


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
        md: MetaData,
        finalizer: AsyncFinalizer,
    ):
        """Executes the middleware in order, passing metadata to each
        middleware until the end of the chain is reached. It then calls
        the finalizer with the metadata.

        :param md: the metadata to pass to the middleware
        :param finalizer: the finalizer to call at the end of the chain
        """
        middleware = self._middleware.copy()

        async def _next(_md: MetaData) -> Exception | None:
            if len(middleware) == 0:
                return await finalizer(_md)
            return await middleware.pop()(_md, _next)

        return await _next(md)
