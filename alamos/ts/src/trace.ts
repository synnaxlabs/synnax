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