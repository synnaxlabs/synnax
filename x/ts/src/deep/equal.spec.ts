// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { deep } from "@/deep";
import { TimeStamp } from "@/telem";

interface TestRecord {
  a: number;
  b: {
    c?: number;
    d?: number;
  };
}

describe("deepEqual", () => {
  describe("deepPartialEqual", () => {
    it("should return true for overlapping objects", () => {
      const base: TestRecord = { a: 1, b: { c: 2, d: 3 } };
      const partial: TestRecord = { a: 1, b: { c: 2 } };
      expect(deep.partialEqual(base, partial)).toBe(true);
    });
    it("should return false for non-overlapping objects", () => {
      const base = { a: 1, b: { c: 2, d: 3 } };
      const partial = { a: 1, b: { c: 2, d: 4 } };
      expect(deep.partialEqual(base, partial)).toBe(false);
    });
  });
  describe("deepEqual", () => {
    it("should return true for equal objects", () => {
      const a = { a: 1, b: { c: 2, d: 3 } };
      const b = { a: 1, b: { c: 2, d: 3 } };
      expect(deep.equal(a, b)).toBe(true);
    });
    it("should return false for non-equal objects", () => {
      const a = { a: 1, b: { c: 2, d: 3 } };
      const b = { a: 1, b: { c: 2, d: 4 } };
      expect(deep.equal(a, b)).toBe(false);
    });
    it("should return false for partial objects", () => {
      const a = { a: 1, b: { c: 2, d: 3 } };
      const b = { a: 1, b: { c: 2 } };
      expect(deep.equal(a, b)).toBe(false);
    });
    it("should return true for primitive arrays that are equal", () => {
      const a = [1, 2, 3];
      const b = [1, 2, 3];
      expect(deep.equal(a, b)).toBe(true);
    });
    it("should true for object arrays that are equal", () => {
      const a = [{ a: 1 }, { b: 2 }];
      const b = [{ a: 1 }, { b: 2 }];
      expect(deep.equal(a, b)).toBe(true);
    });
    it("should return true for nested object arrays that are equal", () => {
      const a = [{ a: 1 }, { b: 2, c: [{ d: 3 }] }];
      const b = [{ a: 1 }, { b: 2, c: [{ d: 3 }] }];
      expect(deep.equal(a, b)).toBe(true);
    });
    it("shold return false for nested object arrays that are not equal", () => {
      const a = [{ a: 1 }, { b: 2, c: [{ d: 3 }] }];
      const b = [{ a: 1 }, { b: 2, c: [{ d: 4 }] }];
      expect(deep.equal(a, b)).toBe(false);
    });
    it("should return false for primitive arrays that are not equal", () => {
      const a = [1, 2, 3];
      const b = [1, 2, 4];
      expect(deep.equal(a, b)).toBe(false);
    });
    it("should return false for object arrays that are not equal", () => {
      const a = [{ a: 1 }, { b: 2 }];
      const b = [{ a: 1 }, { b: 2, c: 4 }];
      expect(deep.equal(a, b)).toBe(false);
    });
    it("should return false for objects with different keys set to undefined", () => {
      expect(deep.equal({ a: undefined }, { b: undefined })).toBe(false);
    });
  });

  describe("dates", () => {
    it("should return true for dates at the same time", () => {
      expect(deep.equal(new Date(5), new Date(5))).toBe(true);
    });

    it("should return false for dates at different times", () => {
      expect(deep.equal(new Date(5), new Date(6))).toBe(false);
    });

    it("should compare nested dates by time", () => {
      expect(deep.equal({ at: new Date(5) }, { at: new Date(6) })).toBe(false);
      expect(deep.equal({ at: new Date(5) }, { at: new Date(5) })).toBe(true);
    });

    it("should return false for a date and a plain object", () => {
      expect(deep.equal<unknown>(new Date(5), {})).toBe(false);
      expect(deep.equal<unknown>({}, new Date(5))).toBe(false);
    });
  });

  describe("maps", () => {
    it("should return true for maps with equal entries", () => {
      const a = new Map([["a", { x: 1 }]]);
      const b = new Map([["a", { x: 1 }]]);
      expect(deep.equal(a, b)).toBe(true);
    });

    it("should compare map values all the way down", () => {
      const a = new Map([["a", { x: [1, 2] }]]);
      const b = new Map([["a", { x: [1, 3] }]]);
      expect(deep.equal(a, b)).toBe(false);
    });

    it("should return false for maps with different keys", () => {
      expect(deep.equal(new Map([["a", 1]]), new Map([["b", 1]]))).toBe(false);
    });

    it("should return false for maps of different sizes", () => {
      const a = new Map([["a", 1]]);
      const b = new Map([
        ["a", 1],
        ["b", 2],
      ]);
      expect(deep.equal(a, b)).toBe(false);
      expect(deep.equal(b, a)).toBe(false);
    });

    it("should return false for a map and a plain object", () => {
      expect(deep.equal<unknown>(new Map(), {})).toBe(false);
    });
  });

  describe("sets", () => {
    it("should return true for sets with the same members", () => {
      expect(deep.equal(new Set([1, 2]), new Set([2, 1]))).toBe(true);
    });

    it("should return false for sets with different members", () => {
      expect(deep.equal(new Set([1, 2]), new Set([1, 3]))).toBe(false);
    });

    it("should return false for sets of different sizes", () => {
      expect(deep.equal(new Set([1]), new Set([1, 2]))).toBe(false);
    });

    it("should compare object members by identity", () => {
      expect(deep.equal(new Set([{ a: 1 }]), new Set([{ a: 1 }]))).toBe(false);
    });

    it("should return false for a set and a map", () => {
      expect(deep.equal<unknown>(new Set(), new Map())).toBe(false);
    });
  });

  describe("equals methods", () => {
    it("should let a value with equals decide", () => {
      expect(deep.equal(new TimeStamp(1), new TimeStamp(1))).toBe(true);
      expect(deep.equal(new TimeStamp(1), new TimeStamp(2))).toBe(false);
    });

    it("should give the same answer in either argument order", () => {
      const decides = { equals: () => true };
      expect(deep.equal<unknown>(decides, { other: 1 })).toBe(true);
      expect(deep.equal<unknown>({ other: 1 }, decides)).toBe(true);
    });

    it("should use equals on nested values", () => {
      expect(deep.equal({ at: new TimeStamp(1) }, { at: new TimeStamp(1) })).toBe(true);
      expect(deep.equal({ at: new TimeStamp(1) }, { at: new TimeStamp(2) })).toBe(
        false,
      );
    });

    it("should ignore an equals field that is not a function", () => {
      expect(deep.equal({ equals: 1 }, { equals: 1 })).toBe(true);
      expect(deep.equal({ equals: 1 }, { equals: 2 })).toBe(false);
    });
  });

  describe("other objects", () => {
    it("should compare errors by their own enumerable fields", () => {
      expect(deep.equal(new Error("a"), new Error("b"))).toBe(true);
      const withCode = Object.assign(new Error("a"), { code: 1 });
      expect(deep.equal(withCode, Object.assign(new Error("a"), { code: 2 }))).toBe(
        false,
      );
    });

    it("should compare typed arrays by their elements", () => {
      expect(deep.equal(new Uint8Array([1, 2]), new Uint8Array([1, 2]))).toBe(true);
      expect(deep.equal(new Uint8Array([1, 2]), new Uint8Array([1, 3]))).toBe(false);
    });

    it("should compare class instances by their own fields", () => {
      class Point {
        constructor(
          readonly x: number,
          readonly y: number,
        ) {}
      }
      expect(deep.equal(new Point(1, 2), new Point(1, 2))).toBe(true);
      expect(deep.equal(new Point(1, 2), new Point(1, 3))).toBe(false);
    });

    it("should not throw on objects with other prototypes", () => {
      expect(() => deep.equal<unknown>(new Error("a"), /a/)).not.toThrow();
    });
  });

  describe("partialEqual on whole values", () => {
    it("should compare dates by time", () => {
      expect(deep.partialEqual({ at: new Date(5) }, { at: new Date(5) })).toBe(true);
      expect(deep.partialEqual({ at: new Date(5) }, { at: new Date(6) })).toBe(false);
    });

    it("should compare maps by their entries", () => {
      const base = { m: new Map([["a", 1]]) };
      expect(deep.partialEqual(base, { m: new Map([["a", 1]]) })).toBe(true);
      expect(deep.partialEqual(base, { m: new Map([["a", 2]]) })).toBe(false);
    });

    it("should compare sets by their members", () => {
      const base = { s: new Set([1]) };
      expect(deep.partialEqual(base, { s: new Set([1]) })).toBe(true);
      expect(deep.partialEqual(base, { s: new Set([2]) })).toBe(false);
    });

    it("should let a value with equals decide", () => {
      const base = { at: new TimeStamp(1) };
      expect(deep.partialEqual(base, { at: new TimeStamp(1) })).toBe(true);
      expect(deep.partialEqual(base, { at: new TimeStamp(2) })).toBe(false);
    });

    it("should still ignore fields absent from the partial", () => {
      const base = { at: new Date(5), name: "a" };
      expect(deep.partialEqual(base, { at: new Date(5) })).toBe(true);
    });
  });
});
