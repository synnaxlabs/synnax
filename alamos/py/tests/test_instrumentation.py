#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import asyncio
import inspect

import pytest
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export.in_memory_span_exporter import (
    InMemorySpanExporter,
)
from opentelemetry.trace import StatusCode

from alamos import Instrumentation, Tracer, trace
from alamos.log import NOOP_LOGGER
from alamos.trace import NOOP_TRACER, _NoopSpan


class TestChild:
    def test_nested_path(self, provider: TracerProvider) -> None:
        """Should extend the full parent path for nested children."""
        ins = Instrumentation(key="root", tracer=Tracer(otel_provider=provider))
        child = ins.child("a").child("b")
        assert child.Meta.path == "root.a.b"
        assert child.L.meta.path == "root.a.b"
        with child.T.prod("span") as span:
            assert span.key == "root.a.b.span"

    def test_filter(self, provider: TracerProvider) -> None:
        """Should keep the environment filter of the parent tracer."""
        tracer = Tracer(otel_provider=provider, filter_=lambda env: env == "prod")
        child = Instrumentation(key="root", tracer=tracer).child("a")
        with child.T.debug("span") as span:
            assert isinstance(span, _NoopSpan)


class Service:
    def __init__(self, instrumentation: Instrumentation):
        self.instrumentation = instrumentation

    @trace("prod")
    def sync_method(self, value: int) -> int:
        """Returns the value."""
        return value

    @trace("prod", key="custom")
    async def async_method(self, value: int) -> int:
        """Returns the value."""
        await asyncio.sleep(0)
        return value

    @trace("prod")
    def sync_fail(self) -> None:
        raise ValueError("sync")

    @trace("prod")
    async def async_fail(self) -> None:
        raise ValueError("async")


class TestTrace:
    def test_span_name(
        self, provider: TracerProvider, exporter: InMemorySpanExporter
    ) -> None:
        """Should name the span with the full instrumentation path."""
        ins = Instrumentation(key="root", tracer=Tracer(otel_provider=provider))
        with ins.child("a").T.prod("span"):
            pass
        (span,) = exporter.get_finished_spans()
        assert span.name == "root.a.span"

    def test_noop_span_key(self) -> None:
        """Should give the noop span an empty key."""
        with Instrumentation(key="root").T.prod("span") as span:
            assert span.key == ""


class TestTraceDecorator:
    @pytest.fixture
    def service(self, provider: TracerProvider) -> Service:
        return Service(
            Instrumentation(key="root", tracer=Tracer(otel_provider=provider))
        )

    def test_sync(self, service: Service, exporter: InMemorySpanExporter) -> None:
        """Should trace a sync method under its name and return its value."""
        assert service.sync_method(1) == 1
        (span,) = exporter.get_finished_spans()
        assert span.name == "root.sync_method"

    def test_async(self, service: Service, exporter: InMemorySpanExporter) -> None:
        """Should trace an async method under the given key and return its value."""
        assert asyncio.run(service.async_method(2)) == 2
        (span,) = exporter.get_finished_spans()
        assert span.name == "root.custom"

    def test_wraps(self) -> None:
        """Should keep the name and docstring of the decorated method."""
        assert Service.sync_method.__name__ == "sync_method"
        assert Service.async_method.__doc__ == "Returns the value."
        assert inspect.iscoroutinefunction(Service.async_method)

    def test_sync_exception(
        self, service: Service, exporter: InMemorySpanExporter
    ) -> None:
        """Should record an exception from a sync method and re-raise it."""
        with pytest.raises(ValueError, match="sync"):
            service.sync_fail()
        (span,) = exporter.get_finished_spans()
        assert span.status.status_code == StatusCode.ERROR
        assert [e.name for e in span.events] == ["exception"]

    def test_async_exception(
        self, service: Service, exporter: InMemorySpanExporter
    ) -> None:
        """Should record an exception from an async method and re-raise it."""
        with pytest.raises(ValueError, match="async"):
            asyncio.run(service.async_fail())
        (span,) = exporter.get_finished_spans()
        assert span.status.status_code == StatusCode.ERROR
        assert [e.name for e in span.events] == ["exception"]


class TestConstruction:
    def test_noop_not_mutated(self) -> None:
        """Should not change the shared noop logger and tracer."""
        Instrumentation(key="root")
        assert not hasattr(NOOP_LOGGER, "meta")
        assert not hasattr(NOOP_TRACER, "_meta")

    def test_shared_tracer(
        self, provider: TracerProvider, exporter: InMemorySpanExporter
    ) -> None:
        """Should name each otel tracer by its own instrumentation key."""
        tracer = Tracer(otel_provider=provider)
        first = Instrumentation(key="first", tracer=tracer)
        Instrumentation(key="second", tracer=tracer)
        with first.T.prod("span"):
            pass
        (span,) = exporter.get_finished_spans()
        assert span.instrumentation_scope is not None
        assert span.instrumentation_scope.name == "first"
