#  Copyright 2023 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from contextlib import contextmanager

from opentelemetry import trace as otel_trace
from opentelemetry.propagators.textmap import TextMapPropagator
from opentelemetry.sdk.trace import Tracer as OtelTracer


class Tracer:
    otel: OtelTracer
    propagator: TextMapPropagator

    def __init__(self):
        self.otel = otel_trace.get_tracer(__name__)

    @contextmanager
    def trace(self, key: str):
        with self.otel.start_as_current_span(key) as span:
            yield span
