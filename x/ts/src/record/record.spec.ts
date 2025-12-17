// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import z from "zod";

import { record } from "@/record";

describe("record", () => {
  describe("types", () => {
    it("should allow string and number keys", () => {
      // Test Key type
      const stringKey: record.Key = "test";
      const numberKey: record.Key = 123;

      expect(typeof stringKey).toBe("string");
      expect(typeof numberKey).toBe("number");
    });

    it("should allow unknown values", () => {
      // Test Unknown type
      const unknown: record.Unknown = {
        string: "value",
        number: 42,
        boolean: true,
        object: { nested: "value" },
        array: [1, 2, 3],
        null: null,
        undefined,
      };

      expect(unknown.string).toBe("value");
      expect(unknown.number).toBe(42);
      expect(unknown.boolean).toBe(true);
    });

    it("should support Keyed interface", () => {
      // Test Keyed interface
      const keyedString: record.Keyed<string> = { key: "test" };
      const keyedNumber: record.Keyed<number> = { key: 123 };

      expect(keyedString.key).toBe("test");
      expect(keyedNumber.key).toBe(123);
    });

    it("should support KeyedNamed interface", () => {
      // Test KeyedNamed interface
      const keyedNamed: record.KeyedNamed = { key: "test", name: "Test Item" };
      const keyedNamedNumber: record.KeyedNamed<number> = {
        key: 123,
        name: "Number Item",
      };

      expect(keyedNamed.key).toBe("test");
      expect(keyedNamed.name).toBe("Test Item");
      expect(keyedNamedNumber.key).toBe(123);
      expect(keyedNamedNumber.name).toBe("Number Item");
    });
  });
  describe("keyZ", () => {
    it("should accept string keys", () => {
      const stringKey = "test";
      expect(record.keyZ.parse(stringKey)).toBe(stringKey);
    });
    it("should accept number keys", () => {
      const numberKey = 123;
      expect(record.keyZ.parse(numberKey)).toBe(numberKey);
    });
    it("should reject symbol keys", () => {
      const symbolKey = Symbol("test");
      expect(() => record.keyZ.parse(symbolKey)).toThrowError(z.ZodError);
    });
    it("should reject undefined keys", () => {
      const undefinedKey = undefined;
      expect(() => record.keyZ.parse(undefinedKey)).toThrowError(z.ZodError);
    });
    it("should reject null keys", () => {
      const nullKey = null;
      expect(() => record.keyZ.parse(nullKey)).toThrowError(z.ZodError);
    });
  });
  describe("unknownZ", () => {
    it("should validate valid records", () => {
      const validRecord = {
        string: "value",
        number: 42,
        symbol: Symbol("test"),
        nested: { key: "value" },
        2: [1, 2, 3],
        function: () => {},
      };
      expect(record.unknownZ.parse(validRecord)).toEqual(validRecord);
    });
    it("should reject symbol keys", () => {
      const invalidRecord = { [Symbol("test")]: "value" };
      expect(() => record.unknownZ.parse(invalidRecord)).toThrowError(z.ZodError);
    });

    it("should accept empty objects", () => {
      const emptyRecord = {};
      expect(record.unknownZ.parse(emptyRecord)).toEqual(emptyRecord);
    });

    it("should accept null and undefined values", () => {
      const recordWithNulls = {
        null: null,
        undefined,
        string: "value",
      };

      expect(record.unknownZ.parse(recordWithNulls)).toEqual(recordWithNulls);
    });
  });

  describe("entries", () => {
    it("should return entries for a simple object", () => {
      const obj = { a: 1, b: "test", c: true };
      const result = record.entries(obj);

      expect(result).toEqual([
        ["a", 1],
        ["b", "test"],
        ["c", true],
      ]);
    });

    it("should return entries for an object with number keys", () => {
      const obj = { 1: "one", 2: "two", 3: "three" };
      const result = record.entries(obj);

      expect(result).toEqual([
        ["1", "one"],
        ["2", "two"],
        ["3", "three"],
      ]);
    });

    it("should return empty array for empty object", () => {
      const obj = {};
      const result = record.entries(obj);

      expect(result).toEqual([]);
    });

    it("should preserve order of entries", () => {
      const obj = { first: 1, second: 2, third: 3 };
      const result = record.entries(obj);

      expect(result[0]).toEqual(["first", 1]);
      expect(result[1]).toEqual(["second", 2]);
      expect(result[2]).toEqual(["third", 3]);
    });

    it("should handle objects with null and undefined values", () => {
      const obj = { a: null, b: undefined, c: "value" };
      const result = record.entries(obj);

      expect(result).toEqual([
        ["a", null],
        ["b", undefined],
        ["c", "value"],
      ]);
    });
  });

  describe("map", () => {
    it("should map values using the provided function", () => {
      const obj = { a: 1, b: 2, c: 3 };
      const result = record.map(obj, (value) => value * 2);

      expect(result).toEqual({ a: 2, b: 4, c: 6 });
    });

    it("should pass both value and key to the mapping function", () => {
      const obj = { a: 1, b: 2, c: 3 };
      const result = record.map(obj, (value, key) => `${key}:${value}`);

      expect(result).toEqual({ a: "a:1", b: "b:2", c: "c:3" });
    });

    it("should handle different return types", () => {
      const obj = { a: 1, b: 2, c: 3 };
      const result = record.map(obj, (value) => value > 2);

      expect(result).toEqual({ a: false, b: false, c: true });
    });

    it("should return empty object for empty input", () => {
      const obj = {};
      const result = record.map(obj, (value) => value);

      expect(result).toEqual({});
    });

    it("should handle objects with mixed value types", () => {
      const obj = { a: 1, b: "test", c: true, d: null };
      const result = record.map(obj, (value) => typeof value);

      expect(result).toEqual({
        a: "number",
        b: "string",
        c: "boolean",
        d: "object",
      });
    });

    it("should preserve key types", () => {
      const obj = { 1: "one", 2: "two" };
      const result = record.map(obj, (value) => value.toUpperCase());

      expect(result).toEqual({ 1: "ONE", 2: "TWO" });
    });
  });

  describe("purgeUndefined", () => {
    it("should remove undefined values", () => {
      const obj = { a: 1, b: undefined, c: "test", d: undefined };
      const result = record.purgeUndefined(obj);

      expect(result).toEqual({ a: 1, c: "test" });
    });

    it("should not remove null values", () => {
      const obj = { a: 1, b: null, c: "test", d: null };
      const result = record.purgeUndefined(obj);

      expect(result).toEqual({ a: 1, b: null, c: "test", d: null });
    });

    it("should preserve other falsy values", () => {
      const obj = { a: 0, b: false, c: "", d: undefined, e: null };
      const result = record.purgeUndefined(obj);

      expect(result).toEqual({ a: 0, b: false, c: "", e: null });
    });

    it("should return empty object when all values are undefined", () => {
      const obj = { a: undefined, b: undefined, c: undefined };
      const result = record.purgeUndefined(obj);

      expect(result).toEqual({});
    });

    it("should return same object when no null/undefined values", () => {
      const obj = { a: 1, b: "test", c: true, d: 0, e: false };
      const result = record.purgeUndefined(obj);

      expect(result).toEqual(obj);
    });

    it("should handle empty object", () => {
      const obj = {};
      const result = record.purgeUndefined(obj);

      expect(result).toEqual({});
    });

    it("should not purge undefined values in nested objects and arrays", () => {
      const obj = {
        a: 1,
        b: undefined,
        c: { nested: "value" },
        d: null,
        e: [1, 2, 3],
      };
      const result = record.purgeUndefined(obj);

      expect(result).toEqual({
        a: 1,
        c: { nested: "value" },
        e: [1, 2, 3],
        d: null,
      });
    });

    it("should create a copy of the object", () => {
      const obj = { a: 1, b: "test" };
      const result = record.purgeUndefined(obj);
      expect(result).not.toBe(obj);
    });
  });

  describe("integration", () => {
    it("should work together: entries -> map -> deleteUndefined", () => {
      const original = { a: 1, b: undefined, c: 3, d: null };

      // Get entries
      const entries = record.entries(original);
      expect(entries).toEqual([
        ["a", 1],
        ["b", undefined],
        ["c", 3],
        ["d", null],
      ]);

      // Map to double values (keeping undefined/null)
      const mapped = record.map(original, (value) =>
        typeof value === "number" ? value * 2 : value,
      );
      expect(mapped).toEqual({ a: 2, b: undefined, c: 6, d: null });

      // Remove undefined/null
      const cleaned = record.purgeUndefined(mapped);
      expect(cleaned).toEqual({ a: 2, c: 6, d: null });
    });

    it("should not purge undefined values in complex nested structures", () => {
      const complex = {
        id: 1,
        name: "test",
        metadata: undefined,
        tags: null,
        config: { enabled: true, timeout: undefined },
        items: [1, 2, 3],
      };

      const result = record.purgeUndefined(complex);
      expect(result).toEqual({
        id: 1,
        name: "test",
        config: { enabled: true, timeout: undefined },
        items: [1, 2, 3],
        tags: null,
      });
    });
  });

  describe("omit", () => {
    type Object = { [key: string]: number };
    it("should return the object if no keys are provided", () =>
      expect(record.omit({ a: 1, b: 2, c: 3 })).toEqual({ a: 1, b: 2, c: 3 }));

    it("should return the object for a single key", () =>
      expect(record.omit({ a: 1, b: 2, c: 3 }, "a")).toEqual({ b: 2, c: 3 }));

    it("should return the object for multiple keys", () =>
      expect(record.omit({ a: 1, b: 2, c: 3 }, "a", "b")).toEqual({ c: 3 }));

    it("should not mutate the original object", () => {
      const obj = { a: 1, b: 2, c: 3 };
      const result = record.omit(obj, "a", "b");
      expect(result).toEqual({ c: 3 });
      expect(obj).toEqual({ a: 1, b: 2, c: 3 });
    });

    it("should be a no-op if the keys are not present", () => {
      const obj: Object = { a: 1, b: 2, c: 3 };
      const result = record.omit(obj, "d", "e");
      expect(result).toEqual({ a: 1, b: 2, c: 3 });
    });

    it("should handle empty objects", () => {
      const obj: Object = {};
      const result = record.omit(obj, "a", "b", "c");
      expect(result).toEqual({});
    });
  });
});
