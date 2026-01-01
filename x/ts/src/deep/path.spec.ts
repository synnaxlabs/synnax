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

describe("path utilities", () => {
  describe("transformPath", () => {
    it("should transform a path", () => {
      expect(deep.transformPath("a.b.c", (part) => part.toUpperCase())).toEqual(
        "A.B.C",
      );
    });
    it("should inject additional parts into the path", () => {
      expect(deep.transformPath("a.b.c", (p) => [p, "d"])).toEqual("a.d.b.d.c.d");
    });
    it("should remove parts from the path", () => {
      expect(deep.transformPath("a.b.c", (p, i) => (i === 1 ? undefined : p))).toEqual(
        "a.c",
      );
    });
  });

  describe("pathsMatch", () => {
    it("should return true if two paths are equal", () => {
      expect(deep.pathsMatch("a.b.c", "a.b.c")).toEqual(true);
    });
    it("should return true if the pattern is a prefix of the path", () => {
      expect(deep.pathsMatch("a.b.c", "a.b")).toEqual(true);
    });
    it("should return true if the pattern has a wildcard", () => {
      expect(deep.pathsMatch("a.b.c", "a.*.c")).toEqual(true);
    });
    it("should return true for an empty pattern", () => {
      expect(deep.pathsMatch("a.b.c", "")).toEqual(true);
    });
    it("should return false if pattern is longer than path", () => {
      expect(deep.pathsMatch("a.b", "a.b.c")).toEqual(false);
    });
    it("should return false if paths don't match", () => {
      expect(deep.pathsMatch("a.b.c", "a.b.d")).toEqual(false);
    });
  });

  describe("resolvePath", () => {
    it("should resolve a path", () => {
      expect(deep.resolvePath("a.b.c", { a: { b: { c: 1 } } })).toEqual("a.b.c");
    });

    it("should resolve a path with a keyed record", () => {
      expect(deep.resolvePath("a.b.c", { a: { b: { c: 1 } } })).toEqual("a.b.c");
    });

    it("should resolve a path with a record in an array", () => {
      expect(deep.resolvePath("a.b.c", { a: { b: [{ c: 1 }] } })).toEqual("a.b.c");
    });

    it("should resolve a path with a keyed record in an array with a key", () => {
      expect(deep.resolvePath("a.b.0.d", { a: { b: [{ key: "c", d: 1 }] } })).toEqual(
        "a.b.c.d",
      );
    });

    it("should not modify a path that has a keyed record accessed by key", () => {
      expect(deep.resolvePath("a.b.c.d", { a: { b: [{ key: "c", c: 1 }] } })).toEqual(
        "a.b.c.d",
      );
    });
  });

  describe("element", () => {
    it("should get element at index", () => {
      expect(deep.element("a.b.c", 1)).toEqual("b");
    });

    it("should handle negative index", () => {
      expect(deep.element("a.b.c", -1)).toEqual("c");
      expect(deep.element("a.b.c", -2)).toEqual("b");
    });

    it("should get first element", () => {
      expect(deep.element("a.b.c", 0)).toEqual("a");
    });
  });

  describe("getIndex", () => {
    it("should return index for numeric string", () => {
      expect(deep.getIndex("42")).toEqual(42);
    });

    it("should return null for non-numeric string", () => {
      expect(deep.getIndex("abc")).toBeNull();
    });

    it("should return null for mixed string", () => {
      expect(deep.getIndex("1a")).toBeNull();
    });

    it("should handle zero", () => {
      expect(deep.getIndex("0")).toEqual(0);
    });
  });

  describe("defaultGetter", () => {
    it("should get property from object", () => {
      const obj = { a: 1, b: 2 };
      expect(deep.defaultGetter(obj, "a")).toEqual(1);
    });

    it("should get index from array", () => {
      const arr = [1, 2, 3];
      expect(deep.defaultGetter(arr as any, "1")).toEqual(2);
    });

    it("should find keyed item in array", () => {
      const arr = [
        { key: "item1", value: 1 },
        { key: "item2", value: 2 },
      ];
      expect(deep.defaultGetter(arr as any, "item2")).toEqual({
        key: "item2",
        value: 2,
      });
    });

    it("should return undefined for non-existent key", () => {
      const obj = { a: 1 };
      expect(deep.defaultGetter(obj, "b")).toBeUndefined();
    });

    it("should handle empty array", () => {
      const arr: any[] = [];
      expect(deep.defaultGetter(arr as any, "0")).toBeUndefined();
    });
  });

  describe("findBestKey", () => {
    it("should find single part key", () => {
      const obj = { a: { b: 1 } };
      const result = deep.findBestKey(obj, ["a", "b"]);
      expect(result).toEqual(["a", 1]);
    });

    it("should find multi-part key with period", () => {
      const obj = { "a.b": { c: 1 } };
      const result = deep.findBestKey(obj, ["a.b", "c"]);
      expect(result).toEqual(["a.b", 1]);
    });

    it("should return null if no key found", () => {
      const obj = { a: 1 };
      const result = deep.findBestKey(obj, ["b", "c"]);
      expect(result).toBeNull();
    });

    it("should prefer shorter key", () => {
      const obj = { a: { b: 1 }, "a.b": 2 };
      const result = deep.findBestKey(obj, ["a", "b"]);
      expect(result).toEqual(["a", 1]);
    });
  });
});
