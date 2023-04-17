#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from contextlib import contextmanager
from typing import Callable, Protocol, Iterator

from opentelemetry.propagators.textmap import CarrierT, Setter, Getter
from opentelemetry.propagators.textmap import TextMapPropagator
from opentelemetry.sdk.trace import (
    TracerProvider as OtelTraceProvider,
    Tracer as OtelTracer,
    Span as OtelSpan,
)

from alamos.meta import InstrumentationMeta
from alamos.noop import noop as noopd, Noop


class Span(Protocol):
    otel: OtelSpan

    def record_exception(self, exc: Exception) -> None:
        ...


class _Span:
    otel: OtelSpan

    def _(self) -> Span:
        return self

    def record_exception(self, exc: Exception) -> None:
        self.otel.record_exception(exc)


class NoopSpan:
    def _(self) -> Span:
        return self

    def record_exception(self, exc: Exception) -> None:
        ...


NOOP_SPAN = NoopSpan()


class Tracer:
    noop: bool = True
    meta: InstrumentationMeta
    _otel_provider: OtelTraceProvider
    _otel_propagator: TextMapPropagator
    __otel_tracer: OtelTracer | None

    def _(self) -> Noop:
        ...

    def __init__(
        self,
        otel_provider: OtelTraceProvider | None = None,
        otel_propagator: TextMapPropagator | None = None,
    ):
        self.noop = otel_provider is None and otel_propagator is None
        self._otel_provider = otel_provider
        self._otel_propagator = otel_propagator
        self.__otel_tracer = None

    @property
    def _otel_tracer(self) -> OtelTracer:
        if self.__otel_tracer is None:
            self.__otel_tracer = self._otel_provider.get_tracer(self.meta.key)
        return self.__otel_tracer

    @contextmanager
    def trace(self, key: str) -> Iterator[Span]:
        with self._otel_tracer.start_as_current_span(key) as span:
            yield span

    @noopd
    def propagate(
        self,
        carrier: CarrierT,
        setter: Callable[[CarrierT, str, str], None],
    ):
        self._otel_propagator.inject(carrier, setter=_Setter(setter))

    @noopd
    def depropagate(
        self,
        carrier: CarrierT,
        getter: Callable[[CarrierT, str], str],
        keys: Callable[[CarrierT], list[str]],
    ):
        return self._otel_propagator.extract(carrier, getter=_Getter(getter, keys))

    def sub(self, meta: InstrumentationMeta) -> "Tracer":
        t = Tracer(
            otel_provider=self._otel_provider,
            otel_propagator=self._otel_propagator,
        )
        t.meta = meta
        return t


NOOP_TRACER = Tracer()


class _Setter(Setter):
    f: Callable[[CarrierT, str, str], None]

    def __init__(self, f: Callable[[CarrierT, str, str], None]):
        self.f = f

    def set(self, carrier: CarrierT, key: str, value: str) -> None:
        self.f(carrier, key, value)


class _Getter(Getter):
    getter: Callable[[CarrierT, str], str | None]
    keys: Callable[[CarrierT], list[str]]

    def __init__(
        self,
        getter: Callable[[CarrierT, str], str],
        keys: Callable[[CarrierT], list[str]],
    ):
        self.getter = getter
        self.keys = keys

    def get(self, carrier: CarrierT, key: str) -> str | None:
        return self.getter(carrier, key)

    def keys(self, carrier: CarrierT) -> list[str]:
        return self.keys(carrier)
