// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { map } from "@/map";

describe("map", () => {
  describe("getOrInsert", () => {
    it("should return the existing value without calling create", () => {
      const m = new Map<string, number>([["a", 42]]);
      const create = vi.fn(() => 100);
      expect(map.getOrInsert(m, "a", create)).toEqual(42);
      expect(create).not.toHaveBeenCalled();
      expect(m.size).toEqual(1);
    });

    it("should set and return the created value when the key is missing", () => {
      const m = new Map<string, number[]>();
      const created = map.getOrInsert(m, "a", () => []);
      created.push(1);
      expect(m.get("a")).toBe(created);
      expect(m.get("a")).toEqual([1]);
    });

    it("should call create only once for repeated inserts", () => {
      const m = new Map<string, number>();
      const create = vi.fn(() => 7);
      map.getOrInsert(m, "a", create);
      map.getOrInsert(m, "a", create);
      expect(create).toHaveBeenCalledTimes(1);
    });

    it("should replace a stored undefined", () => {
      const m = new Map<string, number | undefined>([["a", undefined]]);
      expect(map.getOrInsert(m, "a", () => 3)).toEqual(3);
      expect(m.get("a")).toEqual(3);
    });

    it("should keep stored falsy values", () => {
      const m = new Map<string, number | string | boolean | null>([
        ["zero", 0],
        ["empty", ""],
        ["false", false],
        ["null", null],
      ]);
      expect(map.getOrInsert(m, "zero", () => 1)).toEqual(0);
      expect(map.getOrInsert(m, "empty", () => "x")).toEqual("");
      expect(map.getOrInsert(m, "false", () => true)).toEqual(false);
      expect(map.getOrInsert(m, "null", () => 1)).toBeNull();
    });
  });

  describe("increment", () => {
    it("should start a missing key at zero", () => {
      const m = new Map<string, number>();
      expect(map.increment(m, "a")).toEqual(1);
      expect(m.get("a")).toEqual(1);
    });

    it("should add to an existing count", () => {
      const m = new Map<string, number>([["a", 2]]);
      expect(map.increment(m, "a")).toEqual(3);
    });

    it("should add the given amount", () => {
      const m = new Map<string, number>([["a", 2]]);
      expect(map.increment(m, "a", 5)).toEqual(7);
      expect(map.increment(m, "b", -2)).toEqual(-2);
    });
  });

  describe("equal", () => {
    it("should be true for the same reference", () => {
      const m = new Map([["a", 1]]);
      expect(map.equal(m, m)).toBe(true);
    });

    it("should be true for two empty maps", () => {
      expect(map.equal(new Map(), new Map())).toBe(true);
    });

    it("should be true for the same entries in any insertion order", () => {
      const a = new Map([
        ["a", 1],
        ["b", 2],
      ]);
      const b = new Map([
        ["b", 2],
        ["a", 1],
      ]);
      expect(map.equal(a, b)).toBe(true);
    });

    it("should be false when the sizes differ", () => {
      const b = new Map([
        ["a", 1],
        ["b", 2],
      ]);
      expect(map.equal(new Map([["a", 1]]), b)).toBe(false);
    });

    it("should be false when a key is missing", () => {
      const a = new Map([
        ["a", 1],
        ["b", 2],
      ]);
      const b = new Map([
        ["a", 1],
        ["c", 2],
      ]);
      expect(map.equal(a, b)).toBe(false);
    });

    it("should be false when a value differs", () => {
      expect(map.equal(new Map([["a", 1]]), new Map([["a", 2]]))).toBe(false);
    });

    it("should compare values by reference", () => {
      const shared = { v: 1 };
      expect(map.equal(new Map([["a", shared]]), new Map([["a", shared]]))).toBe(true);
      expect(map.equal(new Map([["a", { v: 1 }]]), new Map([["a", { v: 1 }]]))).toBe(
        false,
      );
    });

    it("should tell a key with an undefined value from a missing key", () => {
      const a = new Map<string, number | undefined>([["a", undefined]]);
      const b = new Map<string, number | undefined>([["b", undefined]]);
      expect(map.equal(a, b)).toBe(false);
    });
  });
});
