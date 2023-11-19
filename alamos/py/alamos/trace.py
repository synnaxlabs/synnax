#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from contextlib import contextmanager
from typing import Protocol, Iterator

from opentelemetry.propagators.textmap import Setter
from opentelemetry.propagators.textmap import TextMapPropagator
from opentelemetry.sdk.trace import (
    TracerProvider as OtelTraceProvider,
    Tracer as OtelTracer,
    Span as OtelSpan,
    StatusCode,
)

from alamos.environment import env_threshold_filter, EnvironmentFilter, Environment
from alamos.meta import InstrumentationMeta
from alamos.noop import noop as noopd, Noop


class Carrier(Protocol):
    def set(self, key: str, value: str) -> None:
        ...


class Span(Protocol):
    """A protocol for a Span that is part of a trace."""

    key: str
    """The key identifying the span. This is the name of the key passed into 'trace'
    combined with the path of the instrumentation that started the span. For example,
    take instrumentation titled 'synnax' and a call to trace with 'test'. The span key
    would be 'synnax.test'
    """

    def record_exception(self, exc: Exception | None) -> None:
        """If exception is not none, records it on the span and sets the span's status
        to error.

        :param exc: An optionally defined exception to record.
        """
        ...


class _Span:
    """Base implementation of the Span protocol"""

    otel: OtelSpan
    key: str

    def __init__(self, otel: OtelSpan, key: str):
        self.otel = otel
        self.key = key

    def _(self) -> Span:
        return self

    def record_exception(self, exc: Exception | None) -> None:
        if exc is None:
            return
        self.otel.record_exception(exc)
        self.otel.set_status(StatusCode.ERROR)


class NoopSpan:
    """Span that does nothing."""

    key: str

    def _(self) -> Span:
        return self

    def record_exception(self, exc: Exception | None) -> None:
        ...


_NOOP_SPAN = NoopSpan()


class Tracer:
    """Tracer wraps an open-telemetry tracer to provide an opinionated interface for
    tracing within the Synnax stack.
    """

    noop: bool
    _meta: InstrumentationMeta
    _filter: EnvironmentFilter
    _otel_provider: OtelTraceProvider
    _otel_propagator: TextMapPropagator
    __otel_tracer: OtelTracer | None

    def _(self) -> Noop:
        return self

    def __init__(
        self,
        otel_provider: OtelTraceProvider | None = None,
        otel_propagator: TextMapPropagator | None = None,
        filter_: EnvironmentFilter = env_threshold_filter("debug"),
    ):
        self.noop = otel_provider is None and otel_propagator is None
        self._filter = filter_ or self._filter
        self._otel_provider = otel_provider
        self._otel_propagator = otel_propagator
        self.__otel_tracer = None

    @property
    def _otel_tracer(self) -> OtelTracer:
        if self.__otel_tracer is None:
            self.__otel_tracer = self._otel_provider.get_tracer(self._meta.key)
        return self.__otel_tracer

    @contextmanager
    def trace(self, key: str, env: Environment) -> Iterator[Span]:
        """Starts a new span with the given key and environment. If
        a span already exists on the current context, the new span is made as its child.

        :param key: The key of the span.
        :param env: The environment to run the span under.
        :return: A span that tracks program execution. If the Tracer's environment filter
        rejects the provided env or the Tracer is noop, a no-op span is provided.
        """
        if self.noop or not self._filter(env):
            yield _NOOP_SPAN
            return
        with self._otel_tracer.start_as_current_span(key) as span:
            yield _Span(otel=span, key=self._meta.extend_path_(key))

    @contextmanager
    def debug(self, key: str) -> Iterator[Span]:
        """Starts a new span at the 'debug' level. If a span already exists on the current
        context, the new span is made as its child.

        :param key: The key of the span.
        :return: A span that tracks program execution. If the Tracer's environment filter
        rejects the 'debug' env or the Tracer is noop, a no-op span is provided.
        """
        with self.trace(key, "debug") as span:
            yield span

    @contextmanager
    def bench(self, key: str) -> Iterator[Span]:
        """Starts a new span at the 'debug' level. If a span already exists on the current
        context, the new span is made as its child.

        :param key: The key of the span.
        :return: A span that tracks program execution. If the Tracer's environment filter
        rejects the 'bench' env or the Tracer is noop, a no-op span is provided.
        """
        with self.trace(key, "bench") as span:
            yield span

    @contextmanager
    def prod(self, key: str) -> Iterator[Span]:
        """Starts a new span at the 'debug' level. If a span already exists on the current
        context, the new span is made as its child.

        :param key: The key of the span.
        :return: A span that tracks program execution. If the Tracer's environment filter
        rejects the 'prod' env or the Tracer is noop, a no-op span is provided.
        """
        with self.trace(key, "prod") as span:
            yield span

    class _Setter(Setter[Carrier]):
        @staticmethod
        def set(carrier: Carrier, key: str, value: str) -> None:
            print("setting", key, value)
            carrier.set(key, value)

    @noopd
    def propagate(self, carrier: Carrier) -> None:
        """Injects metadata about the current trace into the provided carrier.
        This metadata can be parsed on the other side of a network or IPC request using
        allowing the trace to propagate across services.

        :param carrier: The carrier to set the trace metadata on.
        """
        self._otel_propagator.inject(carrier, setter=Tracer._Setter)

    def child_(self, meta: InstrumentationMeta) -> "Tracer":
        t = Tracer(
            otel_provider=self._otel_provider,
            otel_propagator=self._otel_propagator,
        )
        t._meta = meta
        return t


NOOP_TRACER = Tracer()
"""A tracer that does nothing."""
