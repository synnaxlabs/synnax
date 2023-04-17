#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation
from freighter.context import Context, Role
from freighter.transport import (
    Middleware,
    Next,
    AsyncMiddleware,
    AsyncNext,
)


def middleware(instrumentation: Instrumentation) -> Middleware:
    """Adds logs and traces to requests made by the client, and ensures that they are
    propagated to the server.

    :param instrumentation: the instrumentation to use for logging and tracing.
    """

    def _middleware(context: Context, next_: Next):
        if context.role == Role.CLIENT:
            instrumentation.T.propagate(context, _setter)
        with instrumentation.T.trace(context.target) as span:
            res, exc = next_(context)
            span.record_exception(exc)
        _log(context, instrumentation, exc)
        return res, exc

    return _middleware


def async_middleware(instrumentation: Instrumentation) -> AsyncMiddleware:
    """Adds logs and traces to requests made by the client, and ensures that they are
    propagated to the server.

    :param instrumentation: the instrumentation to use for logging and tracing.
    """

    async def _middleware(context: Context, next_: AsyncNext):
        if context.role == Role.CLIENT:
            instrumentation.T.propagate(context, _setter)
        with instrumentation.T.trace(context.target) as span:
            res, exc = await next_(context)
            span.record_exception(exc)
        _log(context, instrumentation, exc)
        return res, exc

    return _middleware


def _log(
    context: Context,
    instrumentation: Instrumentation,
    exc: Exception = None,
):
    if exc:
        instrumentation.L.error(f"{context.target} {exc}")
    else:
        instrumentation.L.debug(f"{context.target}")


def _setter(carrier: Context, key: str, value: str) -> None:
    carrier.set(key, value)


def _getter(carrier: Context, key: str) -> str:
    return carrier.get(key)


def _keys(carrier: Context) -> list[str]:
    return carrier.keys()
