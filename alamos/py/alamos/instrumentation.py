#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Callable, TypeVar, Protocol, Concatenate, ParamSpec

from pydantic import BaseModel

from alamos.logger import Logger
from alamos.tracer import Tracer


class Instrumentation:
    L: Logger
    T: Tracer | None

    def __init__(
        self,
        key: str,
        logger: Logger | None = None,
        tracer: Tracer | None = None,
    ):
        self.L = logger if logger is not None else Logger()
        self.T = tracer if tracer is not None else Tracer()


class InstrumentationMeta(BaseModel):
    """"
    """
    key: str
    path: str
    service_name: str | None = None


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
