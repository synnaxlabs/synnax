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
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";

import { instrumentation } from "@/dev";
import { type Instrumentation, Tracer } from "@/index";

// Records the name of each span that ends and each duration a span sets.
const createTracer = (ended: string[], durations: number[] = []): OtelTracer => {
  const base = trace.getTracer("test");
  const startSpan = (name: string): OtelSpan => {
    const span = base.startSpan(name);
    span.end = () => ended.push(name);
    span.setAttribute = (key, value) => {
      if (key === "duration") durations.push(value as number);
      return span;
    };
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
  describe("duration", () => {
    afterEach(() => {
      vi.restoreAllMocks();
    });
    it("should time overlapping spans with the same key separately", () => {
      const now = vi.spyOn(performance, "now");
      const durations: number[] = [];
      const tracer = new Tracer(createTracer([], durations));
      now.mockReturnValue(0);
      const endFirst = tracer.prod("span");
      now.mockReturnValue(10);
      const endSecond = tracer.prod("span");
      now.mockReturnValue(15);
      endFirst();
      now.mockReturnValue(40);
      endSecond();
      expect(durations).toEqual([15, 30]);
    });
    it("should time nested calls of a function with the same key separately", () => {
      const now = vi.spyOn(performance, "now");
      const durations: number[] = [];
      const tracer = new Tracer(createTracer([], durations));
      now.mockReturnValue(0);
      tracer.prod("span", () => {
        now.mockReturnValue(5);
        tracer.prod("span", () => now.mockReturnValue(7));
        now.mockReturnValue(20);
      });
      expect(durations).toEqual([2, 20]);
    });
  });
});
