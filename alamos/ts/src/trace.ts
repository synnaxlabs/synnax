// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type AttributeValue,
  context,
  propagation,
  type Span as OtelSpan,
  SpanStatusCode,
  type Tracer as OtelTracer,
} from "@opentelemetry/api";
import { type destructor } from "@synnaxlabs/x";

import {
  type Environment,
  type EnvironmentFilter,
  envThresholdFilter,
} from "@/environment";
import { Meta } from "@/meta";

/** Carrier is an entitty that can carry trace metadata across process bounds */
export type Carrier = Record<string, string>;

/** Function that executes under the given span */
export type SpanF = (span: Span) => unknown;

/**
 * Tracer wraps an opentelemetry tracer to provide an opinionated intreface
 * for tracing within the Synnax stack.
 */
export class Tracer {
  private meta: Meta = Meta.NOOP;
  private readonly otelTracer?: OtelTracer;
  private readonly filter: EnvironmentFilter;

  constructor(
    tracer?: OtelTracer,
    filter: EnvironmentFilter = envThresholdFilter("debug"),
  ) {
    this.otelTracer = tracer;
    this.filter = filter;
  }

  child(meta: Meta): Tracer {
    const t = new Tracer(this.otelTracer, this.filter);
    t.meta = meta;
    return t;
  }

  debug(key: string): destructor.Destructor;
  debug<F extends SpanF>(key: string, f: F): ReturnType<F>;

  /**
   * Starts a new span in the debug environment. If a span already exists in the
   * current context, it will be used as the parent span.
   *
   * @param key - The name of the span.
   * @param f - The function to run under the span.
   * @returns A span that tracks program execution. If the Tracer's environment
   * rejects the 'debug' environment or the Tracer is noop, a NoopSpan is returned.
   */
  debug<F extends SpanF>(key: string, f?: F): ReturnType<F> | destructor.Destructor {
    return this.trace(key, "debug", f);
  }

  bench(key: string): destructor.Destructor;
  bench<F extends SpanF>(key: string, f: F): ReturnType<F>;

  /**
   * Starts a new span in the bench environment. If a span already exists in the
   * current context, it will be used as the parent span.
   *
   * @param key - The name of the span.
   * @param f - The function to run under the span.
   * @returns A span that tracks program execution. If the Tracer's environment
   * rejects the 'bench' environment or the Tracer is noop, a NoopSpan is returned.
   */
  bench<F extends SpanF>(key: string, f?: F): ReturnType<F> | destructor.Destructor {
    return this.trace(key, "bench", f);
  }

  prod(key: string): destructor.Destructor;
  prod<F extends SpanF>(key: string, f: F): ReturnType<F>;

  /**
   * Starts a new span in the prod environment. If a span already exists in the
   * current context, it will be used as the parent span.
   *
   * @param key - The name of the span.
   * @param f - The function to run under the span.
   * @returns A span that tracks program execution. If the Tracer's environment
   * rejects the 'prod' environment or the Tracer is noop, a NoopSpan is returned.
   */
  prod<F extends SpanF>(key: string, f?: F): ReturnType<F> | destructor.Destructor {
    return this.trace(key, "prod", f);
  }

  trace(key: string, env: Environment): destructor.Destructor;
  trace<F extends SpanF>(key: string, env: Environment, f: F): ReturnType<F>;
  trace<F extends SpanF>(
    key: string,
    env: Environment,
    f?: F,
  ): ReturnType<F> | destructor.Destructor;

  /**
   * Stars a new span with the given key and environment. If a span already
   * exists in the current context, it will be used as the parent span.
   *
   * @param key - The name of the span.
   * @param env - The environment to run the span under.
   * @param f - The function to run under the span.
   * @returns A span that tracks program execution. If the Tracer's environment
   * rejects the provided span or the Tracer is noop, a NoopSpan is returned.
   */
  trace<F extends SpanF>(
    key: string,
    env: Environment,
    f?: F,
  ): ReturnType<F> | destructor.Destructor {
    const skip = this.meta.noop || !this.filter(env) || this.otelTracer == null;
    if (f == null) {
      if (skip) return () => {};
      const span = new _Span(key, this.otelTracer.startSpan(key));
      span.start();
      return () => span.end();
    }

    if (skip) return f(new NoopSpan(key)) as ReturnType<F>;
    return this.otelTracer.startActiveSpan(key, (otelSpan) => {
      const span = new _Span(key, otelSpan);
      const result = f(span);
      return result as ReturnType<F>;
    });
  }

  /**
   * Injects metadata about the current trace into the provided carrier. This metadata
   * can be parsed on the other side of a network or IPC request to allow the trace to
   * propagate across services.
   *
   * @param carrier - The carrier to inject the metadata into.
   */
  propagate(carrier: Carrier): void {
    if (this.meta.noop) return;
    const ctx = context.active();
    propagation.inject(ctx, carrier, {
      set: (carrier, key, value) => {
        carrier[key] = value;
      },
    });
  }

  /** Tracer implementation that does nothing */
  static readonly NOOP = new Tracer();
}

/** A span in a trace that can be used to track function execution */
export interface Span {
  /**
   * The key identifying the span. This is the name of the key
   * passed into the tracing method combined with the path of the
   * instrumentation that started the span. For example, take the
   * instrumentation titled 'synnax' and call to trace with 'test.
   * The span key would be 'synnax.test'.
   */
  key: string;
  /**
   * If the error is not null, records the error in the span and sets
   * its status to error.
   */
  recordError: (error?: Error | null) => void;
  /**
   * Sets the given key-value pair as an attribute on the span.
   */
  set: (key: string, value: AttributeValue) => void;
}

export class _Span implements Span {
  key: string;
  private readonly otel: OtelSpan;

  constructor(key: string, span: OtelSpan) {
    this.key = key;
    this.otel = span;
  }

  set(key: string, value: AttributeValue): void {
    this.otel.setAttribute(key, value);
  }

  recordError(error?: Error | null): void {
    if (error == null) return;
    this.otel.recordException(error);
    this.otel.setStatus({ code: SpanStatusCode.ERROR });
  }

  start(): void {
    performance.mark(`alamos.trace.start.${this.key}`);
  }

  end(): void {
    try {
      performance.mark(`alamos.trace.end.${this.key}`);
      const duration = performance.measure(
        `alamos.trace.duration.${this.key}`,
        `alamos.trace.start.${this.key}`,
        `alamos.trace.end.${this.key}`,
      );
      this.set("duration", duration.duration);
      this.otel.end();
    } catch (e) {
      console.error(e);
    }
  }
}

/** Span implementation that does nothing */
export class NoopSpan implements Span {
  key: string;

  constructor(key: string) {
    this.key = key;
  }

  set(): void {}

  recordError(_?: Error | null): void {}
}
