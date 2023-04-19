// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
    Span as OtelSpan,
    Tracer as OtelTracer,
    TextMapPropagator,
    context,
} from "@opentelemetry/api"

import { Environment, EnvironmentFilter, envThresholdFilter } from "@/environment"

export type Carrier = Record<string, string>;
export type SpanF = (span: Span) => unknown;

export class Tracer {
    private readonly noop: boolean
    private readonly tracer: OtelTracer
    private readonly propagator: TextMapPropagator
    private readonly filter: EnvironmentFilter

    constructor(
        tracer: OtelTracer,
        propagator: TextMapPropagator,
        filter: EnvironmentFilter = envThresholdFilter("debug"),
        noop: boolean = false,
    ) {
        this.tracer = tracer
        this.propagator = propagator
        this.filter = filter
        this.noop = noop
    }

    trace<F extends SpanF>(key: string, env: Environment, func: F): ReturnType<F> {
        if (this.noop || !this.filter(env)) func(new NoopSpan(key));
        return this.tracer.startActiveSpan(key, (otelSpan) => {
            const span = new _Span(key, otelSpan);
            const result = func(span);
            otelSpan.end();
            return result as ReturnType<F>;
        });
    }

    debug(key: string, func: (span: Span) => unknown): ReturnType<typeof func> {
        return this.trace(key, "debug", func);
    }

    bench(key: string, func: (span: Span) => unknown): ReturnType<typeof func> {
        return this.trace(key, "bench", func);
    }

    prod(key: string, func: (span: Span) => unknown): ReturnType<typeof func> {
        return this.trace(key, "prod", func);
    }

    propagate(carrier: Carrier) {
        const ctx = context.active();
        this.propagator.inject(ctx, carrier, {
            set: (carrier, key, value) => {
                carrier[key] = value;
            }
        })
    }
}

export interface Span {
    key: string;
    recordException(error: Error): void;
}

export class _Span implements Span {
    key: string
    private readonly otel: OtelSpan

    constructor(key: string, span: OtelSpan) {
        this.key = key
        this.otel = span
    }

    recordException(error: Error) {
        this.otel.recordException(error)
    }
}

export class NoopSpan implements Span {
    key: string

    constructor(key: string) {
        this.key = key
    }

    recordException(error: Error) { }
}