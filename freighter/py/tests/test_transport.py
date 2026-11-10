#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Callable

import pytest

from freighter.context import Context
from freighter.transport import (
    AsyncMiddleware,
    AsyncMiddlewareCollector,
    AsyncNext,
    Middleware,
    MiddlewareCollector,
    Next,
)


def _context() -> Context:
    return Context(protocol="http", target="/test", role="client")


def _recorder(calls: list[str], name: str) -> Middleware:
    def mw(ctx: Context, next: Next) -> Context:
        calls.append(name)
        return next(ctx)

    return mw


def _async_recorder(calls: list[str], name: str) -> AsyncMiddleware:
    async def mw(ctx: Context, next: AsyncNext) -> Context:
        calls.append(name)
        return await next(ctx)

    return mw


def _finalizer(calls: list[str]) -> Callable[[Context], Context]:
    def finalizer(ctx: Context) -> Context:
        calls.append("finalizer")
        return ctx

    return finalizer


class TestMiddlewareCollector:
    def test_registration_order(self) -> None:
        """Should run middleware in registration order, then the finalizer."""
        calls: list[str] = []
        collector = MiddlewareCollector()
        collector.use(_recorder(calls, "a"), _recorder(calls, "b"))
        collector.use(_recorder(calls, "c"))
        collector.exec(_context(), finalizer=_finalizer(calls))
        assert calls == ["a", "b", "c", "finalizer"]

    def test_repeat_exec(self) -> None:
        """Should run the full chain on every call."""
        calls: list[str] = []
        collector = MiddlewareCollector()
        collector.use(_recorder(calls, "a"))
        collector.exec(_context(), finalizer=_finalizer(calls))
        collector.exec(_context(), finalizer=_finalizer(calls))
        assert calls == ["a", "finalizer", "a", "finalizer"]


class TestAsyncMiddlewareCollector:
    @pytest.mark.asyncio
    async def test_registration_order(self) -> None:
        """Should run middleware in registration order, then the finalizer."""
        calls: list[str] = []
        collector = AsyncMiddlewareCollector()
        collector.use(_async_recorder(calls, "a"), _async_recorder(calls, "b"))
        collector.use(_async_recorder(calls, "c"))

        async def finalizer(ctx: Context) -> Context:
            calls.append("finalizer")
            return ctx

        await collector.exec(_context(), finalizer=finalizer)
        assert calls == ["a", "b", "c", "finalizer"]
