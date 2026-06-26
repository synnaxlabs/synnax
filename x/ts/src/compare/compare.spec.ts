// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { compare } from "@/compare";

describe("compare", () => {
  describe("numericStrings", () => {
    interface Spec {
      name: string;
      input: string[];
      expected: string[];
    }
    const SPECS: Spec[] = [
      {
        name: "pure numbers",
        input: ["5", "4", "3", "2", "1"],
        expected: ["1", "2", "3", "4", "5"],
      },
      {
        name: "suffixed numbers with equal prefixes",
        input: ["a5", "a4", "a3", "a2", "a1"],
        expected: ["a1", "a2", "a3", "a4", "a5"],
      },
      {
        name: "suffixed numbers with different prefixes",
        input: ["a1", "b1", "a2", "b2", "a3", "b3"],
        expected: ["a1", "a2", "a3", "b1", "b2", "b3"],
      },
      {
        name: "mixed separators",
        input: ["a2", "a.1", "a-3", "a_4"],
        expected: ["a.1", "a2", "a-3", "a_4"],
      },
    ];
    SPECS.forEach((spec) => {
      it(spec.name, () => {
        expect(spec.input.sort(compare.stringsWithNumbers)).toEqual(spec.expected);
      });
    });
  });

  describe("mapsEqual", () => {
    it("returns true for the same reference", () => {
      const m = new Map<string, number>([["a", 1]]);
      expect(compare.mapsEqual(m, m)).toBe(true);
    });

    it("returns true for content-equal maps with identity-equal values", () => {
      const a = new Map<string, number>([
        ["a", 1],
        ["b", 2],
      ]);
      const b = new Map<string, number>([
        ["a", 1],
        ["b", 2],
      ]);
      expect(compare.mapsEqual(a, b)).toBe(true);
    });

    it("returns true regardless of insertion order", () => {
      const a = new Map<string, number>([
        ["a", 1],
        ["b", 2],
      ]);
      const b = new Map<string, number>([
        ["b", 2],
        ["a", 1],
      ]);
      expect(compare.mapsEqual(a, b)).toBe(true);
    });

    it("returns false when sizes differ", () => {
      const a = new Map<string, number>([["a", 1]]);
      const b = new Map<string, number>([
        ["a", 1],
        ["b", 2],
      ]);
      expect(compare.mapsEqual(a, b)).toBe(false);
    });

    it("returns false when a key is present in one but missing in the other", () => {
      const a = new Map<string, number>([
        ["a", 1],
        ["b", 2],
      ]);
      const b = new Map<string, number>([
        ["a", 1],
        ["c", 2],
      ]);
      expect(compare.mapsEqual(a, b)).toBe(false);
    });

    it("returns false when a value differs by identity", () => {
      const a = new Map<string, { v: number }>([["a", { v: 1 }]]);
      const b = new Map<string, { v: number }>([["a", { v: 1 }]]);
      expect(compare.mapsEqual(a, b)).toBe(false);
    });

    it("returns true when a value is the same object reference", () => {
      const shared = { v: 1 };
      const a = new Map<string, { v: number }>([["a", shared]]);
      const b = new Map<string, { v: number }>([["a", shared]]);
      expect(compare.mapsEqual(a, b)).toBe(true);
    });

    it("treats undefined values as present", () => {
      const a = new Map<string, number | undefined>([["a", undefined]]);
      const b = new Map<string, number | undefined>([["b", undefined]]);
      expect(compare.mapsEqual(a, b)).toBe(false);
    });

    it("returns true for two empty maps", () => {
      expect(compare.mapsEqual(new Map(), new Map())).toBe(true);
    });
  });

  describe("arraysEqual", () => {
    it("returns true for the same reference", () => {
      const a = [1, 2, 3];
      expect(compare.arraysEqual(a, a)).toBe(true);
    });

    it("returns true for content-equal primitive arrays", () => {
      expect(compare.arraysEqual([1, 2, 3], [1, 2, 3])).toBe(true);
    });

    it("returns false when lengths differ", () => {
      expect(compare.arraysEqual([1, 2], [1, 2, 3])).toBe(false);
    });

    it("returns false when an element differs by value", () => {
      expect(compare.arraysEqual([1, 2, 3], [1, 9, 3])).toBe(false);
    });

    it("is order-sensitive", () => {
      expect(compare.arraysEqual([1, 2, 3], [3, 2, 1])).toBe(false);
    });

    it("uses identity equality for object elements", () => {
      const a = { v: 1 };
      const b = { v: 1 };
      expect(compare.arraysEqual([a], [a])).toBe(true);
      expect(compare.arraysEqual([a], [b])).toBe(false);
    });

    it("returns true for two empty arrays", () => {
      expect(compare.arraysEqual([], [])).toBe(true);
    });

    it("works with readonly arrays", () => {
      const a: readonly number[] = [1, 2, 3];
      const b: readonly number[] = [1, 2, 3];
      expect(compare.arraysEqual(a, b)).toBe(true);
    });
  });
});
