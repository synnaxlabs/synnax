#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import Callable, TypeVar, Protocol, Concatenate, ParamSpec

from alamos.log import Logger, NOOP_LOGGER
from alamos.meta import InstrumentationMeta
from alamos.trace import Tracer, NOOP_TRACER


class Instrumentation:
    meta: InstrumentationMeta
    L: Logger = NOOP_LOGGER
    T: Tracer = NOOP_TRACER

    def __init__(
        self,
        key: str,
        service_name: str | None = None,
        logger: Logger = NOOP_LOGGER,
        tracer: Tracer = NOOP_TRACER,
    ):
        self.meta = InstrumentationMeta(
            key=key,
            path=key,
            service_name=service_name,
        )
        self.L = logger
        self.L.meta = self.meta
        self.T = tracer
        self.T.meta = self.meta

    def sub(self, key: str) -> Instrumentation:
        meta = self.meta.sub(key)
        ins = Instrumentation(
            key=meta.key,
            logger=self.L.sub(meta),
            tracer=self.T.sub(meta),
        )
        ins.meta = meta
        return ins


NOOP = Instrumentation("")


class Traceable(Protocol):
    instrumentation: Instrumentation


P = ParamSpec("P")
R = TypeVar("R")


def trace(
    key: str | None = None
) -> Callable[
    [Callable[Concatenate[Traceable, P], R]], Callable[Concatenate[Traceable, P], R]]:
    """Trace the given method. The method must be used on a class that implements
    the Traceable protocol and has a non-None instrumentation field.

    :param key:
    :return:
    """

    def decorator(f: Callable[Concatenate[Traceable, P], R]) -> Callable[
        Concatenate[Traceable, P], R]:
        def wrapper(self: Traceable, *args: P.args, **kwargs: P.kwargs):
            with self.instrumentation.T.trace(key if key is not None else f.__name__):
                return f(self, *args, **kwargs)

        return wrapper

    return decorator
