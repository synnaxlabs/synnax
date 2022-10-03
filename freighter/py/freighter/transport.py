from __future__ import annotations

import typing
from typing import TypeAlias, TypeVar, Protocol

from .metadata import MetaData

from pydantic import BaseModel

Payload: TypeAlias = BaseModel

# Represents the inbound payload for a freighter.
RS = TypeVar("RS", bound=Payload, covariant=True)
# Represents the outbound payload for a freighter.
RQ = TypeVar("RQ", bound=Payload, contravariant=True)
# Represents any payload.
P = TypeVar("P", bound=Payload)


class Transport(Protocol):
    """Base class for all transport protocols."""

    def use(self, *args: list[Middleware]) -> None:
        """
        Adds middleware(s) to the transport.
        :param args: the middleware(s) to add
        """
        ...


class AsyncTransport(Protocol):
    """Base class for all asyncio.py transport protocols."""

    def use(self, *args: Middleware) -> None:
        """
        Adds middleware(s) to the transport.
        :param args: the middleware(s) to add
        """
        ...


Next = typing.Callable[[MetaData], Exception | None]
AsyncNext = typing.Callable[[MetaData], typing.Awaitable[Exception | None]]
Middleware = typing.Callable[[MetaData, Next], Exception | None]
AsyncMiddleware = typing.Callable[[MetaData, AsyncNext], typing.Awaitable[Exception | None]]
Finalizer = typing.Callable[[MetaData], Exception | None]
AsyncFinalizer = typing.Callable[[MetaData], typing.Awaitable[Exception | None]]


class MiddlewareCollector:
    _middleware: list[Middleware]

    def __init__(self):
        self._middleware = []

    def use(self, *args: Middleware) -> None:
        self._middleware.extend(args)

    def exec(
            self,
            md: MetaData,
            finalizer: Finalizer,
    ):
        middleware = self._middleware.copy()

        def _next(_md: MetaData) -> Exception | None:
            if len(middleware) == 0:
                return finalizer(_md)
            return middleware.pop()(_md, _next)

        return _next(md)


class AsyncMiddlewareCollector:
    _middleware: list[AsyncMiddleware]

    def __init__(self):
        self._middleware = []

    def use(self, *args: AsyncMiddleware) -> None:
        self._middleware.extend(args)

    async def exec(
            self,
            md: MetaData,
            finalizer: AsyncFinalizer,
    ):
        middleware = self._middleware.copy()

        async def _next(_md: MetaData) -> Exception | None:
            if len(middleware) == 0:
                return await finalizer(_md)
            return await middleware.pop()(_md, _next)

        return await _next(md)
