// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { TEST_SINK_TYPE, TestFactory } from "@/telem/aether/test/factory";
import {
  booleanSinkSpec,
  numberSinkSpec,
  sink,
  stringSinkSpec,
  TestSink,
} from "@/telem/aether/test/sink";

describe("TestSink", () => {
  const sinks: TestSink<unknown>[] = [];

  afterEach(() => {
    sinks.forEach((s) => s.cleanup());
    sinks.length = 0;
  });

  const createSink = <V>(): TestSink<V> => {
    const s = sink<V>();
    sinks.push(s);
    return s;
  };

  describe("constructor and id", () => {
    it("should have static TYPE equal to TEST_SINK_TYPE", () => {
      expect(TestSink.TYPE).toBe(TEST_SINK_TYPE);
    });

    it("should generate unique id for each instance", () => {
      const s1 = createSink<number>();
      const s2 = createSink<number>();

      expect(s1.id).toBeDefined();
      expect(s2.id).toBeDefined();
      expect(s1.id).not.toBe(s2.id);
    });

    it("should auto-register on construction", () => {
      const s = createSink<number>();
      const factory = new TestFactory();

      const result = factory.create({
        type: "test-sink",
        valueType: "any",
        variant: "sink",
        props: { testId: s.id },
      });

      expect(result).toBe(s);
    });

    it("should start with empty values array", () => {
      const s = createSink<number>();
      expect(s.values).toEqual([]);
    });
  });

  describe("set", () => {
    it("should append single value", () => {
      const s = createSink<number>();
      s.set(42);
      expect(s.values).toEqual([42]);
    });

    it("should append multiple values in single call", () => {
      const s = createSink<boolean>();
      s.set(true, false, true);
      expect(s.values).toEqual([true, false, true]);
    });

    it("should accumulate values across multiple calls", () => {
      const s = createSink<string>();
      s.set("a");
      s.set("b");
      s.set("c");
      expect(s.values).toEqual(["a", "b", "c"]);
    });

    it("should trigger observer notification", () => {
      const s = createSink<number>();
      const listener = vi.fn();
      s.onChange(listener);

      s.set(1);

      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should trigger notification for each call", () => {
      const s = createSink<number>();
      const listener = vi.fn();
      s.onChange(listener);

      s.set(1);
      s.set(2);
      s.set(3);

      expect(listener).toHaveBeenCalledTimes(3);
    });
  });

  describe("lastValue", () => {
    it("should return undefined when empty", () => {
      const s = createSink<number>();
      expect(s.lastValue).toBeUndefined();
    });

    it("should return most recent value", () => {
      const s = createSink<number>();
      s.set(1);
      expect(s.lastValue).toBe(1);

      s.set(2);
      expect(s.lastValue).toBe(2);

      s.set(3, 4, 5);
      expect(s.lastValue).toBe(5);
    });
  });

  describe("clear", () => {
    it("should empty the values array", () => {
      const s = createSink<number>();
      s.set(1, 2, 3);
      expect(s.values).toHaveLength(3);

      s.clear();

      expect(s.values).toEqual([]);
      expect(s.lastValue).toBeUndefined();
    });
  });

  describe("cleanup", () => {
    it("should unregister from factory", () => {
      const s = sink<number>();
      const factory = new TestFactory();

      expect(
        factory.create({
          type: "test-sink",
          valueType: "any",
          variant: "sink",
          props: { testId: s.id },
        }),
      ).toBe(s);

      s.cleanup();

      expect(
        factory.create({
          type: "test-sink",
          valueType: "any",
          variant: "sink",
          props: { testId: s.id },
        }),
      ).toBeNull();
    });
  });

  describe("sink factory function", () => {
    it("should create TestSink instance", () => {
      const s = createSink<boolean>();
      expect(s).toBeInstanceOf(TestSink);
    });
  });

  describe("spec creators", () => {
    it("booleanSinkSpec should create correct spec", () => {
      const s = createSink<boolean>();
      const spec = booleanSinkSpec(s);

      expect(spec.type).toBe(TEST_SINK_TYPE);
      expect(spec.variant).toBe("sink");
      expect(spec.valueType).toBe("boolean");
      expect(spec.props.testId).toBe(s.id);
    });

    it("numberSinkSpec should create correct spec", () => {
      const s = createSink<number>();
      const spec = numberSinkSpec(s);

      expect(spec.type).toBe(TEST_SINK_TYPE);
      expect(spec.variant).toBe("sink");
      expect(spec.valueType).toBe("number");
      expect(spec.props.testId).toBe(s.id);
    });

    it("stringSinkSpec should create correct spec", () => {
      const s = createSink<string>();
      const spec = stringSinkSpec(s);

      expect(spec.type).toBe(TEST_SINK_TYPE);
      expect(spec.variant).toBe("sink");
      expect(spec.valueType).toBe("string");
      expect(spec.props.testId).toBe(s.id);
    });

    it("spec should allow factory to retrieve sink", () => {
      const s = createSink<boolean>();
      const spec = booleanSinkSpec(s);
      const factory = new TestFactory();

      const result = factory.create(spec);

      expect(result).toBe(s);
    });
  });
});
