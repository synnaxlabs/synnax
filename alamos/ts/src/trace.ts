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
} from "@opentelemetry/api";

import { Environment, EnvironmentFilter, envThresholdFilter } from "@/environment";

export type Carrier = Record<string, string>;
export type SpanF = (span: Span) => unknown;

/**
 * Tracer wraps an opentelemetry tracer to provide an opinionated intreface 
 * for tracing within the Synnax stack.
 */
export class Tracer {
  private readonly noop: boolean;
  private readonly tracer: OtelTracer;
  private readonly propagator: TextMapPropagator;
  private readonly filter: EnvironmentFilter;

  constructor(
    tracer: OtelTracer,
    propagator: TextMapPropagator,
    filter: EnvironmentFilter = envThresholdFilter("debug"),
    noop: boolean = false
  ) {
    this.tracer = tracer;
    this.propagator = propagator;
    this.filter = filter;
    this.noop = noop;
  }

  /**
  * Stars a new span with the given key and environment. If a span already
  * exists in the current context, it will be used as the parent span.
  *
  * @param key - The name of the span.
  * @param env - The environment to run the span under.
  * @param  f -  The function to run under the span.
  * @returns A span that tracks program execution. If the Tracer's environment
  * rejects the provided span or the Tracer is noop, a NoopSpan is returned.
  */
  trace<F extends SpanF>(key: string, env: Environment, f: F): ReturnType<F> {
    if (this.noop || !this.filter(env)) f(new NoopSpan(key));
    return this.tracer.startActiveSpan(key, (otelSpan) => {
      const span = new _Span(key, otelSpan);
      const result = f(span);
      otelSpan.end();
      return result as ReturnType<F>;
    });
  }

  /**
  * Starts a new span in the debug environment. If a span already exists in the 
  * current context, it will be used as the parent span.
  *
  * @param key - The name of the span.
  * @param f -  The function to run under the span.
  * @returns A span that tracks program execution. If the Tracer's environment
  * rejects the 'debug' environment or the Tracer is noop, a NoopSpan is returned.
  */
  debug<F extends SpanF>(key: string, f: F): ReturnType<F> {
    return this.trace(key, "debug", f);
  }

  /**
  * Starts a new span in the bench environment. If a span already exists in the 
  * current context, it will be used as the parent span.
  *
  * @param key - The name of the span.
  * @param f -  The function to run under the span.
  * @returns A span that tracks program execution. If the Tracer's environment
  * rejects the 'bench' environment or the Tracer is noop, a NoopSpan is returned.
  */
  bench<F extends SpanF>(key: string, f: F): ReturnType<F> {
    return this.trace(key, "bench", f);
  }

  /**
  * Starts a new span in the prod environment. If a span already exists in the 
  * current context, it will be used as the parent span.
  *
  * @param key - The name of the span.
  * @param f -  The function to run under the span.
  * @returns A span that tracks program execution. If the Tracer's environment
  * rejects the 'prod' environment or the Tracer is noop, a NoopSpan is returned.
  */
  prod<F extends SpanF>(key: string, f: F): ReturnType<F> {
    return this.trace(key, "prod", f);
  }

  /**
  * Injects meta-data about the current trace into the provided carrier. This 
  * meta-data can be paresed on teh other side of a network or IPC request to
  * allow the trace to proapgate across services.
  * 
  * @param carrier - The carrier to inject the meta-data into.
  */
  propagate(carrier: Carrier): void {
    if (this.noop) return;
    const ctx = context.active();
    this.propagator.inject(ctx, carrier, {
      set: (carrier, key, value) => {
        carrier[key] = value;
      },
    });
  }
}

export interface Span {
  key: string;
  recordException: (error: Error) => void;
}

export class _Span implements Span {
  key: string;
  private readonly otel: OtelSpan;

  constructor(key: string, span: OtelSpan) {
    this.key = key;
    this.otel = span;
  }

  recordException(error: Error): void {
    this.otel.recordException(error);
  }
}

export class NoopSpan implements Span {
  key: string;

  constructor(key: string) {
    this.key = key;
  }

  recordException(_: Error): void { }
}
