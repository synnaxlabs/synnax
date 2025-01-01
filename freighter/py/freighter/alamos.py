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
from freighter.transport import AsyncMiddleware, AsyncNext, Middleware, Next


def instrumentation_middleware(instrumentation: Instrumentation) -> Middleware:
    """Adds logs and traces to requests made by the client, and ensures that they are
    propagated to the server.

    :param instrumentation: the instrumentation to use for logging and tracing.
    """

    def _middleware(ctx: Context, next_: Next):
        with instrumentation.T.debug(ctx.target) as span:
            if ctx.role == "client":
                instrumentation.T.propagate(ctx)
            res, exc = next_(ctx)
            span.record_exception(exc)
        _log(ctx, instrumentation, exc)
        return res, exc

    return _middleware


def async_instrumentation_middleware(
    instrumentation: Instrumentation,
) -> AsyncMiddleware:
    """Adds logs and traces to requests made by the client, and ensures that they are
    propagated to the server.

    :param instrumentation: the instrumentation to use for logging and tracing.
    """

    async def _middleware(context: Context, next_: AsyncNext):
        if context.role == "client":
            instrumentation.T.propagate(context)
        with instrumentation.T.trace(context.target, "debug") as span:
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
