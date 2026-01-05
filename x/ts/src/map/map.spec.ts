// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { map } from "@/map";

describe("map", () => {
  describe("getOrSetDefault", () => {
    it("should return existing value when key exists", () => {
      const testMap = new Map<string, number>();
      testMap.set("existing", 42);

      const result = map.getOrSetDefault(testMap, "existing", 100);

      expect(result).toEqual(42);
      expect(testMap.get("existing")).toEqual(42);
      expect(testMap.size).toEqual(1);
    });

    it("should set and return default value when key doesn't exist", () => {
      const testMap = new Map<string, number>();

      const result = map.getOrSetDefault(testMap, "new", 100);

      expect(result).toEqual(100);
      expect(testMap.get("new")).toEqual(100);
      expect(testMap.size).toEqual(1);
    });

    it("should replace the value when key exists with undefined value", () => {
      const testMap = new Map<string, number | undefined>();
      testMap.set("undefined", undefined);

      const result = map.getOrSetDefault(testMap, "undefined", 100);

      expect(result).toEqual(100);
      expect(testMap.get("undefined")).toEqual(100);
      expect(testMap.size).toEqual(1);
    });

    it("should return null when key exists with null value", () => {
      const testMap = new Map<string, number | null>();
      testMap.set("null", null);

      const result = map.getOrSetDefault(testMap, "null", 100);

      expect(result).toBeNull();
      expect(testMap.get("null")).toBeNull();
      expect(testMap.size).toEqual(1);
    });

    it("should work with different key types", () => {
      const numberMap = new Map<number, string>();
      const result1 = map.getOrSetDefault(numberMap, 1, "default");
      expect(result1).toEqual("default");
      expect(numberMap.get(1)).toEqual("default");
      const objKey = { id: 1 };
      const objectMap = new Map<object, string>();
      const result2 = map.getOrSetDefault(objectMap, objKey, "object-default");
      expect(result2).toEqual("object-default");
      expect(objectMap.get(objKey)).toEqual("object-default");
    });

    it("should work with different value types", () => {
      const stringMap = new Map<string, string>();
      const result1 = map.getOrSetDefault(stringMap, "key", "default");
      expect(result1).toEqual("default");
      const boolMap = new Map<string, boolean>();
      const result2 = map.getOrSetDefault(boolMap, "key", true);
      expect(result2).toEqual(true);
      const objValue = { name: "test" };
      const objMap = new Map<string, object>();
      const result3 = map.getOrSetDefault(objMap, "key", objValue);
      expect(result3).toEqual(objValue);
      expect(result3).toBe(objValue);
    });

    it("should handle empty maps", () => {
      const emptyMap = new Map<string, number>();

      const result = map.getOrSetDefault(emptyMap, "first", 42);

      expect(result).toEqual(42);
      expect(emptyMap.get("first")).toEqual(42);
      expect(emptyMap.size).toEqual(1);
    });

    it("should handle multiple operations on same map", () => {
      const testMap = new Map<string, number>();
      const result1 = map.getOrSetDefault(testMap, "key1", 10);
      expect(result1).toEqual(10);
      expect(testMap.size).toEqual(1);
      const result2 = map.getOrSetDefault(testMap, "key1", 20);
      expect(result2).toEqual(10); // Original value, not new default
      expect(testMap.size).toEqual(1);
      const result3 = map.getOrSetDefault(testMap, "key2", 30);
      expect(result3).toEqual(30);
      expect(testMap.size).toEqual(2);
    });

    it("should handle falsy values correctly", () => {
      const testMap = new Map<string, number | string | boolean>();
      testMap.set("zero", 0);
      const result1 = map.getOrSetDefault(testMap, "zero", 100);
      expect(result1).toEqual(0);
      testMap.set("empty", "");
      const result2 = map.getOrSetDefault(testMap, "empty", "default");
      expect(result2).toEqual("");
      testMap.set("false", false);
      const result3 = map.getOrSetDefault(testMap, "false", true);
      expect(result3).toEqual(false);
    });

    it("should handle arrays and complex objects", () => {
      const testMap = new Map<string, number[]>();
      const defaultArray = [1, 2, 3];

      const result = map.getOrSetDefault(testMap, "array", defaultArray);

      expect(result).toEqual([1, 2, 3]);
      expect(result).toBe(defaultArray); // Same reference
      expect(testMap.get("array")).toBe(defaultArray);
    });
  });
});
