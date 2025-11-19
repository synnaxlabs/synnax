// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { shallow } from "@/shallow";

describe("shallow", () => {
  describe("copy", () => {
    it("should shallow copy plain objects", () => {
      const obj = { a: 1, b: 2 };
      const copied = shallow.copy(obj);
      expect(copied).toEqual(obj);
      expect(copied).not.toBe(obj);
    });

    it("should shallow copy arrays", () => {
      const arr = [1, 2, 3];
      const copied = shallow.copy(arr);
      expect(copied).toEqual(arr);
      expect(copied).not.toBe(arr);
    });

    it("should not deep copy nested objects", () => {
      const obj = { a: 1, nested: { b: 2 } };
      const copied = shallow.copy(obj);
      expect(copied.nested).toBe(obj.nested);
      copied.nested.b = 999;
      expect(obj.nested.b).toBe(999);
    });

    it("should not deep copy nested arrays", () => {
      const arr = [1, [2, 3]];
      const copied = shallow.copy(arr);
      expect(copied[1]).toBe(arr[1]);
      (copied[1] as number[])[0] = 999;
      expect((arr[1] as number[])[0]).toBe(999);
    });

    it("should return primitives as-is", () => {
      expect(shallow.copy(42)).toBe(42);
      expect(shallow.copy("string")).toBe("string");
      expect(shallow.copy(true)).toBe(true);
      expect(shallow.copy(false)).toBe(false);
    });

    it("should return null as-is", () => {
      expect(shallow.copy(null)).toBe(null);
    });

    it("should return undefined as-is", () => {
      expect(shallow.copy(undefined)).toBe(undefined);
    });

    it("should copy arrays with objects", () => {
      const arr = [{ a: 1 }, { b: 2 }];
      const copied = shallow.copy(arr);
      expect(copied).not.toBe(arr);
      expect(copied[0]).toBe(arr[0]);
      expect(copied[1]).toBe(arr[1]);
    });

    it("should copy objects with various types", () => {
      const obj = {
        num: 42,
        str: "test",
        bool: true,
        arr: [1, 2],
        nested: { x: 1 },
      };
      const copied = shallow.copy(obj);
      expect(copied).toEqual(obj);
      expect(copied).not.toBe(obj);
      expect(copied.arr).toBe(obj.arr);
      expect(copied.nested).toBe(obj.nested);
    });

    it("should copy empty objects", () => {
      const obj = {};
      const copied = shallow.copy(obj);
      expect(copied).toEqual({});
      expect(copied).not.toBe(obj);
    });

    it("should copy empty arrays", () => {
      const arr: unknown[] = [];
      const copied = shallow.copy(arr);
      expect(copied).toEqual([]);
      expect(copied).not.toBe(arr);
    });

    it("should handle class instances", () => {
      class TestClass {
        value = 42;
      }
      const instance = new TestClass();
      const copied = shallow.copy(instance);
      expect(copied).not.toBe(instance);
      expect(copied.value).toBe(42);
    });

    it("should spread Date objects into plain objects", () => {
      const date = new Date("2025-01-01");
      const copied = shallow.copy(date);
      expect(copied).not.toBe(date);
      expect(copied).toEqual({});
    });

    it("should handle objects with symbol keys", () => {
      const sym = Symbol("test");
      const obj = { [sym]: "value", regular: 1 };
      const copied = shallow.copy(obj);
      expect(copied[sym]).toBe("value");
      expect(copied.regular).toBe(1);
    });

    it("should lose prototype when spreading objects", () => {
      class Custom {
        x = 1;
      }
      const obj = new Custom();
      const copied = shallow.copy(obj);
      expect(copied).not.toBeInstanceOf(Custom);
      expect(copied.x).toBe(1);
    });

    it("should fill holes in sparse arrays", () => {
      const arr = [1, undefined, 3];
      const copied = shallow.copy(arr);
      expect(copied).not.toBe(arr);
      expect(copied).toEqual([1, undefined, 3]);
      expect(1 in copied).toBe(true);
    });
  });
});
