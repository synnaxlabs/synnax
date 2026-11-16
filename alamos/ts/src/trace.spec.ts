// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Span as OtelSpan,
  trace,
  type Tracer as OtelTracer,
} from "@opentelemetry/api";
import { beforeAll, describe, expect, it } from "vitest";

import { instrumentation } from "@/dev";
import { type Instrumentation, Tracer } from "@/index";

// Records the name of each span that ends.
const createTracer = (ended: string[]): OtelTracer => {
  const base = trace.getTracer("test");
  const startSpan = (name: string): OtelSpan => {
    const span = base.startSpan(name);
    span.end = () => ended.push(name);
    return span;
  };
  return {
    startSpan,
    startActiveSpan: <F extends (span: OtelSpan) => unknown>(
      name: string,
      ...args: unknown[]
    ) => (args.at(-1) as F)(startSpan(name)) as ReturnType<F>,
  };
};

describe("Trace", () => {
  let ins: Instrumentation;
  beforeAll(() => {
    ins = instrumentation();
  });
  describe("initialize", () => {
    it("should correctly initialize a tracer", () => {
      const t = new Tracer();
      expect(t).toBeDefined();
    });
  });
  describe("trace", () => {
    it("should start a span with the given key", () => {
      ins.T.prod("test", (span) => {
        expect(span.key).toEqual("test");
      });
    });
    it("should trace an async function correctly", async () => {
      const result = await ins.T.prod("async-test", async () => "test");
      expect(result).toEqual("test");
    });
    it("should end the span of a function after it returns", () => {
      const ended: string[] = [];
      new Tracer(createTracer(ended)).prod("sync", () => {
        expect(ended).toEqual([]);
      });
      expect(ended).toEqual(["sync"]);
    });
    it("should end the span of an async function after it resolves", async () => {
      const ended: string[] = [];
      const done = new Tracer(createTracer(ended)).prod("async", async () => {
        await Promise.resolve();
        expect(ended).toEqual([]);
      });
      await done;
      expect(ended).toEqual(["async"]);
    });
    it("should end the span of a function that throws", () => {
      const ended: string[] = [];
      expect(() =>
        new Tracer(createTracer(ended)).prod("sync", () => {
          throw new Error("failed");
        }),
      ).toThrow("failed");
      expect(ended).toEqual(["sync"]);
    });
    it("should end the span of an async function after it rejects", async () => {
      const ended: string[] = [];
      const done = new Tracer(createTracer(ended)).prod("async", async () => {
        await Promise.resolve();
        expect(ended).toEqual([]);
        throw new Error("failed");
      });
      await expect(done).rejects.toThrow("failed");
      expect(ended).toEqual(["async"]);
    });
  });
});
