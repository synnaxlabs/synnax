// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { instance } from "@/instance";

describe("createMatcher", () => {
  class TestClass implements instance.Discriminated {
    discriminator = "test";
    constructor(public value: string) {}
  }

  const isTestClass = instance.createMatcher("test", TestClass);

  it("should return true for instances of the class", () => {
    const instance = new TestClass("value");
    expect(isTestClass(instance)).toBe(true);
  });

  it("should return true for objects with matching discriminator", () => {
    const obj = { discriminator: "test", value: "value" };
    expect(isTestClass(obj)).toBe(true);
  });

  it("should return false for null", () => {
    expect(isTestClass(null)).toBe(false);
  });

  it("should return false for undefined", () => {
    expect(isTestClass(undefined)).toBe(false);
  });

  it("should return false for non-objects", () => {
    expect(isTestClass("string")).toBe(false);
    expect(isTestClass(123)).toBe(false);
    expect(isTestClass(true)).toBe(false);
  });

  it("should return false for objects with different discriminator", () => {
    const obj = { discriminator: "other", value: "value" };
    expect(isTestClass(obj)).toBe(false);
  });

  it("should return false for objects without discriminator", () => {
    const obj = { value: "value" };
    expect(isTestClass(obj)).toBe(false);
  });

  it("should work with multiple class instances", () => {
    class ClassA implements instance.Discriminated {
      discriminator = "a";
      constructor(public value: string) {}
    }

    class ClassB implements instance.Discriminated {
      discriminator = "b";
      constructor(public value: string) {}
    }

    const isClassA = instance.createMatcher("a", ClassA);
    const isClassB = instance.createMatcher("b", ClassB);

    const instanceA = new ClassA("value");
    const instanceB = new ClassB("value");

    expect(isClassA(instanceA)).toBe(true);
    expect(isClassA(instanceB)).toBe(false);
    expect(isClassB(instanceA)).toBe(false);
    expect(isClassB(instanceB)).toBe(true);
  });
});
