#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from contextlib import contextmanager
from typing import Callable

from opentelemetry.propagators.textmap import CarrierT, Setter, Getter
from opentelemetry.propagators.textmap import TextMapPropagator
from opentelemetry.sdk.trace import Tracer as OtelTracer

from alamos.noop import noop as noopd, Noop


class Tracer:
    noop: bool = True
    _otel_tracer: OtelTracer
    _otel_propagator: TextMapPropagator

    def _(self) -> Noop:
        ...

    def __init__(
        self,
        otel_tracer: OtelTracer | None = None,
        otel_propagator: TextMapPropagator | None = None,
        noop: bool = True,
    ):
        self.noop = noop
        self._otel_tracer = otel_tracer
        self._otel_propagator = otel_propagator

    @noopd
    @contextmanager
    def trace(self, key: str):
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
        self._otel_propagator.extract(carrier, getter=_Getter(getter, keys))


class _Setter(Setter):
    f: Callable[[CarrierT, str, str], None]

    def __init__(self, f: Callable[[CarrierT, str, str], None]):
        self.f = f

    def set(self, carrier: CarrierT, key: str, value: str) -> None:
        self.f(carrier, key, value)


class _Getter(Getter):
    getter: Callable[[CarrierT, str], str]
    keys: Callable[[CarrierT], list[str]]

    def __init__(
        self,
        getter: Callable[[CarrierT, str], str],
        keys: Callable[[CarrierT], list[str]],
    ):
        self.getter = getter
        self.keys = keys

    def get(self, carrier: CarrierT, key: str) -> str:
        return self.getter(carrier, key)

    def keys(self, carrier: CarrierT) -> list[str]:
        return self.keys(carrier)
