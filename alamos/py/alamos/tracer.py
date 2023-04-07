from typing import Callable, Protocol, TypeVar, Concatenate
from contextlib import contextmanager
from opentelemetry import trace

from opentelemetry.sdk.trace import Tracer as OtelTracer


class Tracer:
    otel: OtelTracer

    def __init__(self):
        self.otel = trace.get_tracer(__name__)

    @contextmanager
    def trace(self, key: str):
        with self.otel.start_as_current_span(key) as span:
            yield span


class Traceable(Protocol):
    t: Tracer


A = TypeVar("A")
R = TypeVar("R")


def trace(
    key: str | None,
) -> Callable[[Callable[[Traceable, A], R]], Callable[[Traceable, A], R]]:
    def decorator(f: Callable[[Traceable, A], R]) -> Callable[[Traceable, A], R]:
        def wrapper(self, *args, **kwargs):
            with self.t.trace(key if key is not None else f.__name__):
                return f(self, *args, **kwargs)

        return wrapper

    return decorator
