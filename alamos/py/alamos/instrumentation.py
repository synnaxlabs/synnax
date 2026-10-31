#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from __future__ import annotations

import copy
import functools
import inspect
from collections.abc import Callable, Coroutine
from typing import Concatenate, ParamSpec, Protocol, TypeVar, overload

from alamos.environment import Environment
from alamos.log import NOOP_LOGGER, Logger
from alamos.meta import InstrumentationMeta
from alamos.trace import NOOP_TRACER, Tracer


class Instrumentation:
    """
    Instrumentation is alamos's core data type, and represents a collection of
    instrumentation tools: a logger and a tracer.

    Instrumentation is specifically designed for dependency injection into different
    services within your application. We recommend creating a single Instrumentation
    instance per application and passing it around to different services.

    Instrumentation is also focused on making noop functionality as easy as possible.
    For that, we've provided the NOOP constant, which we recommend using as the default
    value in function or constructor arguments. This allows you to inject preconfigured
    instrumentation when you want to, and simply default to noop when you don't.

    Instrumentation is organized as a hierarchy, where the child method is used to
    create instrumentation that extends the key of its parent. This allows for your logs
    and traces to match the architecture of your application. For example,
    instrumentation that tracks low-level db requests could be created with the key
    child("db") and instrumentation that tracks api requests could be created with the
    key child("api"). See the child method for more details.
    """

    Meta: InstrumentationMeta
    """Metadata about the instrumentation."""
    L: Logger
    """The logger for this instrumentation."""
    T: Tracer
    """The tracer for this instrumentation."""

    def __init__(
        self,
        *,
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
        self.L = logger.child_(self.Meta)
        self.T = tracer.child_(self.Meta)

    def child(self, key: str) -> Instrumentation:
        """
        Creates a child of this instrumentation with the given key.

        :param key: The key to set on the child. If the parent's path is "parent" and
            the provided key is "child", the child's path will be "parent.child". We
            recommend keeping this key unique within the children of the parent, as name
            conflicts may cause unexpected behavior.
        :returns: A new child Instrumentation.
        """
        ins = copy.copy(self)
        ins.Meta = self.Meta.child_(key)
        ins.L = self.L.child_(ins.Meta)
        ins.T = self.T.child_(ins.Meta)
        return ins


NOOP = Instrumentation(key="")
"""
Noop is instrumentation that does nothing. We highly recommend using this as a default
value for instrumentation fields or function arguments.
"""


class Traceable(Protocol):
    """
    A protocol for classes whose methods can be traced using the trace decorator.
    """

    instrumentation: Instrumentation


P = ParamSpec("P")
R = TypeVar("R")
T = TypeVar("T", bound=Traceable)


class _Decorator(Protocol):
    @overload
    def __call__(
        self, f: Callable[Concatenate[T, P], Coroutine[object, object, R]], /
    ) -> Callable[Concatenate[T, P], Coroutine[object, object, R]]: ...

    @overload
    def __call__(
        self, f: Callable[Concatenate[T, P], R], /
    ) -> Callable[Concatenate[T, P], R]: ...


def trace(env: Environment, *, key: str | None = None) -> _Decorator:
    """
    Trace the given method on the class. This method can only be used on a class that
    implements the Traceable protocol. Supports both sync and async methods. An
    exception raised by the method is recorded on the span and then re-raised.

    :param key: The key of the span. Defaults to the name of the method.
    :param env: The environment to run this span under. If the Tracer's environment
        filter reject the env, a no-op span is returned.
    """

    @overload
    def decorator(
        f: Callable[Concatenate[T, P], Coroutine[object, object, R]],
    ) -> Callable[Concatenate[T, P], Coroutine[object, object, R]]: ...

    @overload
    def decorator(
        f: Callable[Concatenate[T, P], R],
    ) -> Callable[Concatenate[T, P], R]: ...

    def decorator(
        f: Callable[Concatenate[T, P], object],
    ) -> Callable[Concatenate[T, P], object]:
        span_key = f.__name__ if key is None else key
        if inspect.iscoroutinefunction(f):

            @functools.wraps(f)
            async def async_wrapper(
                self: T, /, *args: P.args, **kwargs: P.kwargs
            ) -> object:
                with self.instrumentation.T.trace(span_key, env=env) as span:
                    try:
                        return await f(self, *args, **kwargs)
                    except Exception as e:
                        span.record_exception(e)
                        raise

            return async_wrapper

        @functools.wraps(f)
        def wrapper(self: T, /, *args: P.args, **kwargs: P.kwargs) -> object:
            with self.instrumentation.T.trace(span_key, env=env) as span:
                try:
                    return f(self, *args, **kwargs)
                except Exception as e:
                    span.record_exception(e)
                    raise

        return wrapper

    return decorator
