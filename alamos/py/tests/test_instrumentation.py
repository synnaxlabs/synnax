#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import pytest
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import SimpleSpanProcessor
from opentelemetry.sdk.trace.export.in_memory_span_exporter import (
    InMemorySpanExporter,
)

from alamos import Instrumentation, Tracer
from alamos.log import NOOP_LOGGER
from alamos.trace import NOOP_TRACER, _NoopSpan


@pytest.fixture
def exporter() -> InMemorySpanExporter:
    return InMemorySpanExporter()


@pytest.fixture
def provider(exporter: InMemorySpanExporter) -> TracerProvider:
    provider = TracerProvider()
    provider.add_span_processor(SimpleSpanProcessor(exporter))
    return provider


class TestChild:
    def test_nested_path(self, provider: TracerProvider) -> None:
        """Should extend the full parent path for nested children."""
        ins = Instrumentation("root", tracer=Tracer(otel_provider=provider))
        child = ins.child("a").child("b")
        assert child.Meta.path == "root.a.b"
        assert child.L.meta.path == "root.a.b"
        with child.T.prod("span") as span:
            assert span.key == "root.a.b.span"

    def test_filter(self, provider: TracerProvider) -> None:
        """Should keep the environment filter of the parent tracer."""
        tracer = Tracer(otel_provider=provider, filter_=lambda env: env == "prod")
        child = Instrumentation("root", tracer=tracer).child("a")
        with child.T.debug("span") as span:
            assert isinstance(span, _NoopSpan)


class TestConstruction:
    def test_noop_not_mutated(self) -> None:
        """Should not change the shared noop logger and tracer."""
        Instrumentation("root")
        assert not hasattr(NOOP_LOGGER, "meta")
        assert not hasattr(NOOP_TRACER, "_meta")

    def test_shared_tracer(
        self, provider: TracerProvider, exporter: InMemorySpanExporter
    ) -> None:
        """Should name each otel tracer by its own instrumentation key."""
        tracer = Tracer(otel_provider=provider)
        first = Instrumentation("first", tracer=tracer)
        Instrumentation("second", tracer=tracer)
        with first.T.prod("span"):
            pass
        (span,) = exporter.get_finished_spans()
        assert span.instrumentation_scope is not None
        assert span.instrumentation_scope.name == "first"
