// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, describe, expect, it, vi } from "vitest";

import { TEST_SOURCE_TYPE, TestFactory } from "@/telem/aether/test/factory";
import {
  booleanSourceSpec,
  numberSourceSpec,
  source,
  stringSourceSpec,
  TestSource,
} from "@/telem/aether/test/source";

describe("TestSource", () => {
  const sources: TestSource<unknown>[] = [];

  afterEach(() => {
    sources.forEach((s) => s.cleanup());
    sources.length = 0;
  });

  const createSource = <V>(initialValue: V): TestSource<V> => {
    const s = source(initialValue);
    sources.push(s);
    return s;
  };

  describe("constructor and id", () => {
    it("should have static TYPE equal to TEST_SOURCE_TYPE", () => {
      expect(TestSource.TYPE).toBe(TEST_SOURCE_TYPE);
    });

    it("should generate unique id for each instance", () => {
      const s1 = createSource(0);
      const s2 = createSource(0);

      expect(s1.id).toBeDefined();
      expect(s2.id).toBeDefined();
      expect(s1.id).not.toBe(s2.id);
    });

    it("should auto-register on construction", () => {
      const s = createSource(42);
      const factory = new TestFactory();

      const result = factory.create({
        type: "test-source",
        valueType: "any",
        variant: "source",
        props: { testId: s.id },
      });

      expect(result).toBe(s);
    });
  });

  describe("value and setValue", () => {
    it("should return initial value", () => {
      const s = createSource("initial");
      expect(s.value()).toBe("initial");
    });

    it("should update value with setValue", () => {
      const s = createSource(10);
      s.setValue(20);
      expect(s.value()).toBe(20);
    });

    it("should trigger observer notification on setValue", () => {
      const s = createSource(false);
      const listener = vi.fn();
      s.onChange(listener);

      s.setValue(true);

      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should work with different value types", () => {
      const boolSource = createSource(true);
      const numSource = createSource(123);
      const strSource = createSource("hello");

      expect(boolSource.value()).toBe(true);
      expect(numSource.value()).toBe(123);
      expect(strSource.value()).toBe("hello");

      boolSource.setValue(false);
      numSource.setValue(456);
      strSource.setValue("world");

      expect(boolSource.value()).toBe(false);
      expect(numSource.value()).toBe(456);
      expect(strSource.value()).toBe("world");
    });
  });

  describe("cleanup", () => {
    it("should unregister from factory on cleanup", () => {
      const s = source(0);
      const factory = new TestFactory();

      expect(
        factory.create({
          type: "test-source",
          valueType: "any",
          variant: "source",
          props: { testId: s.id },
        }),
      ).toBe(s);

      s.cleanup();

      expect(
        factory.create({
          type: "test-source",
          valueType: "any",
          variant: "source",
          props: { testId: s.id },
        }),
      ).toBeNull();
    });
  });

  describe("source factory function", () => {
    it("should create TestSource with initial value", () => {
      const s = createSource(99);
      expect(s).toBeInstanceOf(TestSource);
      expect(s.value()).toBe(99);
    });
  });

  describe("spec creators", () => {
    it("booleanSourceSpec should create correct spec", () => {
      const s = createSource(true);
      const spec = booleanSourceSpec(s);

      expect(spec.type).toBe(TEST_SOURCE_TYPE);
      expect(spec.variant).toBe("source");
      expect(spec.valueType).toBe("boolean");
      expect(spec.props.testId).toBe(s.id);
    });

    it("numberSourceSpec should create correct spec", () => {
      const s = createSource(42);
      const spec = numberSourceSpec(s);

      expect(spec.type).toBe(TEST_SOURCE_TYPE);
      expect(spec.variant).toBe("source");
      expect(spec.valueType).toBe("number");
      expect(spec.props.testId).toBe(s.id);
    });

    it("stringSourceSpec should create correct spec", () => {
      const s = createSource("test");
      const spec = stringSourceSpec(s);

      expect(spec.type).toBe(TEST_SOURCE_TYPE);
      expect(spec.variant).toBe("source");
      expect(spec.valueType).toBe("string");
      expect(spec.props.testId).toBe(s.id);
    });

    it("spec should allow factory to retrieve source", () => {
      const s = createSource(true);
      const spec = booleanSourceSpec(s);
      const factory = new TestFactory();

      const result = factory.create(spec);

      expect(result).toBe(s);
    });
  });
});
