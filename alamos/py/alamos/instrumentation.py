#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

from typing import (
    Callable,
    TypeVar,
    Protocol,
    Concatenate,
    ParamSpec,
)

from alamos.environment import Environment
from alamos.log import Logger, NOOP_LOGGER
from alamos.meta import InstrumentationMeta
from alamos.trace import Tracer, NOOP_TRACER


class Instrumentation:
    """Instrumentation is alamos' core data type, and represents a collection of
    instrumentation tools: a logger and a tracer.

    Instrumentation is specifically designed for dependency injection into different
    services within your application. We recommend creating a single Instrumentation
    instance per application and passing it around to different services.

    Instrumentation is also focused on making noop functionality as easy as possible.
    For that, we've provided the NOOP constant, which we recommend using as the default
    value in function or constructor arguments. This allows you to inject preconfigured
    instrumentation when you want to, and simply default to noop when you don't.

    Instrumentation is organized as a hierarchy, where the child method is used to create
    instrumentation that extends the key of its parent. This allows for your logs and
    traces to match the architecture of your application. For example, instrumentation
    that tracks low-level db requests could be created with the key child("db") and
    instrumentation that tracks api requests could be created with the key child("api").
    See the child method for more details.
    """

    Meta: InstrumentationMeta
    """Metadata bout the instrumentation."""
    L: Logger
    """The logger for this instrumentation."""
    T: Tracer
    """The tracer for this instrumentation."""

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
        self.T._meta = self.Meta

    def child(self, key: str) -> Instrumentation:
        """Creates a child of this instrumentation with the given key.

        :param key: The key to set on the child. If the parent's path is "parent" and the
        provided key is "child", the child's path will be "parent.child". We recommend
        keeping this key unique within the children of the parent, as name conflicts
        may cause unexpected behavior.
        :returns: A new child Instrumentation.
        """
        meta = self.Meta.child_(key)
        ins = Instrumentation(
            key=meta.key,
            logger=self.L.child_(meta),
            tracer=self.T.child_(meta),
        )
        ins.Meta = meta
        return ins


NOOP = Instrumentation("")
"""Noop is instrumentation that does nothing. We highly recommend using this as a
default value for instrumentation fields or function arguments."""


class Traceable(Protocol):
    """A protocol for classes whose methods can be traced using the trace
    decorator"""

    instrumentation: Instrumentation


P = ParamSpec("P")
R = TypeVar("R")
T = TypeVar("T", bound=Traceable)


def trace(
    env: Environment, key: str | None = None
) -> Callable[[Callable[Concatenate[T, P], R]], Callable[Concatenate[T, P], R]]:
    """Trace the given method on the class. This method can only be used on a class that
    implements the Traceable protocol.

    :param key: The key of the span.
    :param env: The environment to run this span under. If the Tracer's environment
    filter rejects the env, a no-op span is returned.
    """

    def decorator(f: Callable[Concatenate[T, P], R]) -> Callable[Concatenate[T, P], R]:
        def wrapper(self: T, *args: P.args, **kwargs: P.kwargs):
            _key = f.__name__ if key is None else key
            with self.instrumentation.T.trace(_key, env):
                return f(self, *args, **kwargs)

        return wrapper

    return decorator
