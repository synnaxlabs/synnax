#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from opentelemetry.propagate import get_global_textmap
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import (
    BatchSpanProcessor,
    ConsoleSpanExporter,
)

from alamos import Tracer
from alamos.dev import instrumentation

provider = TracerProvider()
processor = BatchSpanProcessor(ConsoleSpanExporter())
provider.add_span_processor(processor)


class TestTrace:
    def test_initialize(self):
        """
        Should initialize the tracer.
        """
        tracer = Tracer(
            otel_provider=provider,
            otel_propagator=get_global_textmap()
        )
        assert tracer is not None

    def test_trace(self):
        """
        Should not raise an exception.
        """
        ins = instrumentation()
        with ins.T.trace("test"):
            pass


class TestPropagate:
    def test_propagate_depropagate(self):
        """Should correctly inject the span context into the carrier.
        """
        ins = instrumentation()
        carrier = dict()

        def setter(carrier, key, value):
            carrier[key] = value

        with ins.T.trace("test") as span:
            ins.T.propagate(carrier, setter)

        assert len(carrier) == 1

        def getter(carrier, key):
            return carrier.get(key, None)

        def keys(carrier):
            return list(carrier.keys())

        span_context = ins.T.depropagate(carrier, getter, keys)
        assert span_context is not None
