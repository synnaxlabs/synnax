#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any

import pytest
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export.in_memory_span_exporter import (
    InMemorySpanExporter,
)
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

from alamos import Instrumentation, Tracer, trace
from alamos.instrumentation import Traceable


class TestTrace:
    def test_initialize(self, provider: TracerProvider) -> None:
        """
        Should initialize the tracer.
        """
        tracer = Tracer(
            otel_provider=provider, otel_propagator=TraceContextTextMapPropagator()
        )
        assert not tracer.noop

    def test_trace(
        self, instrumentation: Instrumentation, exporter: InMemorySpanExporter
    ) -> None:
        """Should export a span with the instrumentation path as its key."""
        with instrumentation.T.prod("test") as span:
            assert span.key == "dev.test"
        assert len(exporter.get_finished_spans()) == 1

    def test_trace_decorator(
        self, instrumentation: Instrumentation, exporter: InMemorySpanExporter
    ) -> None:
        """Should export a span for the decorated method."""

        class Foo:
            instrumentation: Instrumentation

            def _(self) -> Traceable:
                return self

            def __init__(self, ins: Instrumentation):
                self.instrumentation = ins

            @trace("prod")
            def decorated(self) -> str:
                return "hello"

        assert Foo(instrumentation).decorated() == "hello"
        assert len(exporter.get_finished_spans()) == 1


class TestPropagate:
    def test_propagate_depropagate(
        self, instrumentation: Instrumentation, capsys: pytest.CaptureFixture[str]
    ) -> None:
        """Should inject the span context into the carrier without writing to
        stdout."""

        class Carrier(dict[Any, Any]):
            def set(self, key: Any, value: Any) -> None:
                self[key] = value

        carrier = Carrier()

        with instrumentation.T.prod("test"):
            instrumentation.T.propagate(carrier)

        assert "traceparent" in carrier
        assert capsys.readouterr().out == ""
