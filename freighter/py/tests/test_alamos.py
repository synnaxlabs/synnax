#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from collections.abc import Iterator
from contextlib import contextmanager

import pytest
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.trace import get_current_span
from opentelemetry.trace.propagation.tracecontext import TraceContextTextMapPropagator

from alamos import Instrumentation, Tracer
from freighter.alamos import (
    async_instrumentation_middleware,
    instrumentation_middleware,
)
from freighter.context import Context, Role
from freighter.transport import AsyncMiddlewareCollector, MiddlewareCollector

provider = TracerProvider()


@pytest.fixture
def instrumentation() -> Instrumentation:
    return Instrumentation(
        key="test",
        tracer=Tracer(
            otel_provider=provider, otel_propagator=TraceContextTextMapPropagator()
        ),
    )


@contextmanager
def outer_span() -> Iterator[int]:
    """Starts a span that the middleware span becomes a child of."""
    with provider.get_tracer("outer").start_as_current_span("outer") as span:
        yield span.get_span_context().span_id


class Capture:
    """Records the propagated span ID and the active span ID at the finalizer."""

    propagated: str | None = None
    active: int = 0

    def __call__(self, ctx: Context) -> Context:
        self.propagated = ctx.get("traceparent")
        self.active = get_current_span().get_span_context().span_id
        return ctx

    async def async_call(self, ctx: Context) -> Context:
        return self(ctx)

    def propagated_span_id(self) -> int:
        assert self.propagated is not None
        return int(self.propagated.split("-")[2], 16)


def run_sync(instrumentation: Instrumentation, role: Role) -> Capture:
    collector = MiddlewareCollector()
    collector.use(instrumentation_middleware(instrumentation))
    capture = Capture()
    collector.exec(
        Context(protocol="websocket", target="/echo", role=role), finalizer=capture
    )
    return capture


async def run_async(instrumentation: Instrumentation, role: Role) -> Capture:
    collector = AsyncMiddlewareCollector()
    collector.use(async_instrumentation_middleware(instrumentation))
    capture = Capture()
    await collector.exec(
        Context(protocol="websocket", target="/echo", role=role),
        finalizer=capture.async_call,
    )
    return capture


class TestInstrumentationMiddleware:
    def test_sync_propagates_middleware_span(
        self, instrumentation: Instrumentation
    ) -> None:
        """Should propagate the span the middleware starts, not its parent."""
        with outer_span() as outer:
            capture = run_sync(instrumentation, "client")
        assert capture.propagated_span_id() == capture.active
        assert capture.active != outer

    @pytest.mark.asyncio
    async def test_async_propagates_middleware_span(
        self, instrumentation: Instrumentation
    ) -> None:
        """Should propagate the span the middleware starts, not its parent."""
        with outer_span() as outer:
            capture = await run_async(instrumentation, "client")
        assert capture.propagated_span_id() == capture.active
        assert capture.active != outer

    def test_sync_propagates_without_parent(
        self, instrumentation: Instrumentation
    ) -> None:
        """Should propagate when no span is active before the request."""
        assert run_sync(instrumentation, "client").propagated is not None

    @pytest.mark.asyncio
    async def test_async_propagates_without_parent(
        self, instrumentation: Instrumentation
    ) -> None:
        """Should propagate when no span is active before the request."""
        assert (await run_async(instrumentation, "client")).propagated is not None

    def test_sync_server_does_not_propagate(
        self, instrumentation: Instrumentation
    ) -> None:
        """Should not propagate on the server side."""
        assert run_sync(instrumentation, "server").propagated is None

    @pytest.mark.asyncio
    async def test_async_server_does_not_propagate(
        self, instrumentation: Instrumentation
    ) -> None:
        """Should not propagate on the server side."""
        assert (await run_async(instrumentation, "server")).propagated is None
