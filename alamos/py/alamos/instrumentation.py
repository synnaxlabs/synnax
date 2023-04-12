#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Callable, TypeVar, Protocol

from alamos.logger import Logger
from alamos.tracer import Tracer


class Instrumentation:
    L: Logger
    T: Tracer

    def __init__(self, logger: Logger, tracer: Tracer):
        self.L = logger
        self.T = tracer


class Traceable(Protocol):
    instrumentation: Instrumentation


A = TypeVar("A")
R = TypeVar("R")


def trace(
    key: str | None = None
) -> Callable[[Callable[[Traceable, A], R]], Callable[[Traceable, A], R]]:
    def decorator(f: Callable[[Traceable, A], R]) -> Callable[[Traceable, A], R]:
        def wrapper(self, *args, **kwargs):
            with self.instrumentation.T.trace(key if key is not None else f.__name__):
                return f(self, *args, **kwargs)

        return wrapper

    return decorator
