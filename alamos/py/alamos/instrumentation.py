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
