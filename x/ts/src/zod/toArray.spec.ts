// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { toArray } from "@/zod/toArray";

describe("toArray", () => {
  describe("with string schema", () => {
    const schema = toArray(z.string());

    it("should accept an array of strings", () => {
      const result = schema.parse(["hello", "world"]);
      expect(result).toEqual(["hello", "world"]);
    });

    it("should convert a single string to an array", () => {
      const result = schema.parse("hello");
      expect(result).toEqual(["hello"]);
    });

    it("should accept an empty array", () => {
      const result = schema.parse([]);
      expect(result).toEqual([]);
    });

    it("should throw for invalid types in array", () => {
      expect(() => schema.parse([123, 456])).toThrow(z.ZodError);
    });

    it("should throw for invalid single value", () => {
      expect(() => schema.parse(123)).toThrow(z.ZodError);
    });
  });

  describe("with number schema", () => {
    const schema = toArray(z.number());

    it("should accept an array of numbers", () => {
      const result = schema.parse([1, 2, 3]);
      expect(result).toEqual([1, 2, 3]);
    });

    it("should convert a single number to an array", () => {
      const result = schema.parse(42);
      expect(result).toEqual([42]);
    });

    it("should handle zero", () => {
      const result = schema.parse(0);
      expect(result).toEqual([0]);
    });

    it("should handle negative numbers", () => {
      const result = schema.parse(-5);
      expect(result).toEqual([-5]);
    });
  });

  describe("with object schema", () => {
    const objSchema = z.object({
      name: z.string(),
      age: z.number(),
    });
    const schema = toArray(objSchema);

    it("should accept an array of objects", () => {
      const input = [
        { name: "Alice", age: 30 },
        { name: "Bob", age: 25 },
      ];
      const result = schema.parse(input);
      expect(result).toEqual(input);
    });

    it("should convert a single object to an array", () => {
      const input = { name: "Charlie", age: 35 };
      const result = schema.parse(input);
      expect(result).toEqual([input]);
    });

    it("should validate object properties", () => {
      expect(() => schema.parse([{ name: "Dave", age: "invalid" }])).toThrow(
        z.ZodError,
      );
    });
  });

  describe("with optional schema", () => {
    const schema = toArray(z.string().optional());

    it("should handle undefined in array", () => {
      const result = schema.parse([undefined, "hello", undefined]);
      expect(result).toEqual([undefined, "hello", undefined]);
    });

    it("should convert undefined to array with undefined", () => {
      const result = schema.parse(undefined);
      expect(result).toEqual([undefined]);
    });
  });

  describe("with union schema", () => {
    const schema = toArray(z.union([z.string(), z.number()]));

    it("should accept mixed types in array", () => {
      const result = schema.parse(["hello", 42, "world", 123]);
      expect(result).toEqual(["hello", 42, "world", 123]);
    });

    it("should convert single string to array", () => {
      const result = schema.parse("test");
      expect(result).toEqual(["test"]);
    });

    it("should convert single number to array", () => {
      const result = schema.parse(999);
      expect(result).toEqual([999]);
    });

    it("should throw for invalid type", () => {
      expect(() => schema.parse(true)).toThrow(z.ZodError);
    });
  });

  describe("with nullable schema", () => {
    const schema = toArray(z.string().nullable());

    it("should handle null in array", () => {
      const result = schema.parse([null, "hello", null]);
      expect(result).toEqual([null, "hello", null]);
    });

    it("should convert null to array with null", () => {
      const result = schema.parse(null);
      expect(result).toEqual([null]);
    });
  });

  describe("with transformed schema", () => {
    const baseSchema = z.string().transform((val) => val.toUpperCase());
    const schema = toArray(baseSchema);

    it("should apply transformation to array elements", () => {
      const result = schema.parse(["hello", "world"]);
      expect(result).toEqual(["HELLO", "WORLD"]);
    });

    it("should apply transformation to single value", () => {
      const result = schema.parse("hello");
      expect(result).toEqual(["HELLO"]);
    });
  });

  describe("edge cases", () => {
    it("should handle deeply nested arrays", () => {
      const schema = toArray(toArray(z.number()));
      const result = schema.parse([
        [1, 2],
        [3, 4],
      ]);
      expect(result).toEqual([
        [1, 2],
        [3, 4],
      ]);
    });

    it("should convert single value to nested array", () => {
      const schema = toArray(toArray(z.number()));
      const result = schema.parse(5);
      expect(result).toEqual([[5]]);
    });

    it("should return equal array when input is already an array", () => {
      const schema = toArray(z.string());
      const input = ["test"];
      const result = schema.parse(input);
      expect(result).toEqual(input);
    });
  });
});
