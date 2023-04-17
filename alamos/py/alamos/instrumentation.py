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
    Meta: InstrumentationMeta
    L: Logger = NOOP_LOGGER
    T: Tracer = NOOP_TRACER

    def __init__(
        self,
        key: str,
        service_name: str | None = None,
        logger: Logger = NOOP_LOGGER,
        tracer: Tracer = NOOP_TRACER,
    ):
        self.Meta = InstrumentationMeta(
            key=key,
            path=key,
            service_name=service_name,
        )
        self.L = logger
        self.L.meta = self.Meta
        self.T = tracer
        self.T.meta = self.Meta

    def sub(self, key: str) -> Instrumentation:
        meta = self.Meta.child(key)
        ins = Instrumentation(
            key=meta.key,
            logger=self.L.sub(meta),
            tracer=self.T.sub(meta),
        )
        ins.Meta = meta
        return ins


NOOP = Instrumentation("")
"""Noop is instrumentation that does nothing. We highly recommend using this as a
default value for instrumentation fields or function arguments."""


class Traceable(Protocol):
    """A protocol for classes that whose methods can be traced using the trace deecorator"""
    instrumentation: Instrumentation


P = ParamSpec("P")
R = TypeVar("R")


def trace(
    key: str | None = None
) -> Callable[[Callable[Concatenate[Traceable, P], R]], Callable[P, R]]:
    """Trace the given method. The method must be used on a class that implements
    the Traceable protocol and has a non-None instrumentation field.

    :param key:
    :return:
    """

    def decorator(f: Callable[Concatenate[Traceable, P], R]) -> Callable[P, R]:
        def wrapper(self: Traceable, *args: P.args, **kwargs: P.kwargs):
            with self.instrumentation.T.trace(key if key is not None else f.__name__):
                return f(self, *args, **kwargs)

        return wrapper

    return decorator
