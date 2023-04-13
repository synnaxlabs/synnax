#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from alamos import Instrumentation
from freighter.context import Context, Location
from freighter.transport import (
    Middleware,
    Next,
    AsyncMiddleware,
    AsyncNext,
)


def _setter(carrier: Context, key: str, value: str) -> None:
    carrier.set(key, value)


def _getter(carrier: Context, key: str) -> str:
    return carrier.get(key)


def _keys(carrier: Context) -> list[str]:
    return carrier.keys()


def _core(context: Context, instrumentation: Instrumentation):
    if context.location == Location.CLIENT:
        instrumentation.T.propagate(context, _setter)


def instrumentation_middleware(
    instrumentation: Instrumentation,
) -> Middleware:
    def middleware(context: Context, next_: Next):
        _core(context, instrumentation)
        return next_(context)

    return middleware


def async_instrumentation_middleware(
    instrumentation: Instrumentation,
) -> AsyncMiddleware:
    async def middleware(context: Context, next_: AsyncNext):
        _core(context, instrumentation)
        return await next_(context)

    return middleware
