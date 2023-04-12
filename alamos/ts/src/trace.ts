// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Span as OtelSpan, Tracer as OtelTracer, TextMapPropagator } from "@opentelemetry/api"

export class Tracer {
    readonly tracer: OtelTracer
    readonly propagator: TextMapPropagator

    constructor(tracer: OtelTracer, propagator: TextMapPropagator) {
        this.tracer = tracer
        this.propagator = propagator
    }

    trace<F extends (span: Span) => unknown>(key: string, func: F): ReturnType<F> {
        return this.tracer.startActiveSpan(key, (otelSpan) => {
            const span = new Span(key, otelSpan);
            const result = func(span);
            span.end();
            return result as ReturnType<F>;
        });
    }
}

export class Span {
    key: string
    private readonly span: OtelSpan

    constructor(key: string, span: OtelSpan) {
        this.key = key
        this.span = span
    }

    end() {
        this.span.end()
    }
}
