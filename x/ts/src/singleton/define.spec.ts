// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { define } from "@/singleton/define";

describe("define", () => {
  it("should create and return a singleton instance", () => {
    const key = "test-singleton-1";
    let constructorCallCount = 0;

    const createInstance = () => {
      constructorCallCount++;
      return { value: "singleton-value" };
    };

    const getInstance = define(key, createInstance);
    const instance1 = getInstance();

    expect(constructorCallCount).toBe(1);
    expect(instance1).toEqual({ value: "singleton-value" });

    const instance2 = getInstance();
    expect(constructorCallCount).toBe(1);
    expect(instance2).toBe(instance1); // Same instance reference
  });

  it("should maintain separate singletons for different keys", () => {
    const key1 = "test-singleton-2";
    const key2 = "test-singleton-3";

    const getInstance1 = define(key1, () => ({ id: 1 }));
    const getInstance2 = define(key2, () => ({ id: 2 }));

    const instance1 = getInstance1();
    const instance2 = getInstance2();

    expect(instance1).toEqual({ id: 1 });
    expect(instance2).toEqual({ id: 2 });
    expect(instance1).not.toBe(instance2);
  });

  it("should retrieve the same singleton when defining with the same key multiple times", () => {
    const key = "test-singleton-4";
    let constructorCallCount = 0;

    const createInstance1 = () => {
      constructorCallCount++;
      return { id: constructorCallCount };
    };

    const getInstance1 = define(key, createInstance1);
    const firstInstance = getInstance1();
    expect(firstInstance).toEqual({ id: 1 });
    expect(constructorCallCount).toBe(1);

    // Second definition with the same key but different factory
    const createInstance2 = () => {
      constructorCallCount++;
      return { id: 999 }; // Different value
    };

    const getInstance2 = define(key, createInstance2);
    const secondInstance = getInstance2();

    expect(secondInstance).toBe(firstInstance);
    expect(secondInstance).toEqual({ id: 1 });
    expect(constructorCallCount).toBe(1); // Still 1, not 2
  });

  it("should handle object instances as singleton values", () => {
    class TestClass {
      public value: number;
      constructor(value: number) {
        this.value = value;
      }
    }

    const key = "test-singleton-5";
    const getInstance = define(key, () => new TestClass(42));

    const instance = getInstance();
    expect(instance).toBeInstanceOf(TestClass);
    expect(instance.value).toBe(42);
  });
});
