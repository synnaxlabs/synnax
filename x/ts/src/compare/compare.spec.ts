// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";

import { compare } from "@/compare";

describe("numbers", () => {
  it("should order numbers in ascending order", () => {
    expect(compare.numbers(1, 2)).toBeLessThan(0);
    expect(compare.numbers(2, 1)).toBeGreaterThan(0);
    expect(compare.numbers(-3, -3)).toBe(0);
  });

  it("should sort an array with duplicates and negative numbers", () => {
    expect([3, -1, 2, 3, 0].sort(compare.numbers)).toEqual([-1, 0, 2, 3, 3]);
  });

  it("should sort an empty array", () => {
    expect([].sort(compare.numbers)).toEqual([]);
  });

  it("should order infinities at the ends", () => {
    expect([1, Infinity, -Infinity].sort(compare.numbers)).toEqual([
      -Infinity,
      1,
      Infinity,
    ]);
  });

  it("should return NaN when either value is NaN", () => {
    expect(compare.numbers(NaN, 1)).toBeNaN();
    expect(compare.numbers(1, NaN)).toBeNaN();
    expect(compare.numbers(NaN, NaN)).toBeNaN();
  });
});

describe("bigints", () => {
  it("should return -1, 0, or 1", () => {
    expect(compare.bigints(1n, 2n)).toBe(-1);
    expect(compare.bigints(2n, 2n)).toBe(0);
    expect(compare.bigints(3n, 2n)).toBe(1);
  });

  it("should order values that a number cannot hold exactly", () => {
    const big = 2n ** 64n;
    expect(compare.bigints(big, big + 1n)).toBe(-1);
    expect(compare.bigints(big + 1n, big)).toBe(1);
  });

  it("should sort an array with duplicates and negative values", () => {
    expect([3n, -1n, 2n, 3n].sort(compare.bigints)).toEqual([-1n, 2n, 3n, 3n]);
  });
});

describe("by", () => {
  interface Item {
    name: string;
    count: number;
  }

  it("should order values by their key with text by default", () => {
    const items: Item[] = [
      { name: "b", count: 1 },
      { name: "A", count: 2 },
      { name: "c", count: 3 },
    ];
    expect(items.sort(compare.by((i) => i.name)).map((i) => i.name)).toEqual([
      "A",
      "b",
      "c",
    ]);
  });

  it("should order values by their key with the given comparator", () => {
    const items: Item[] = [
      { name: "a", count: 10 },
      { name: "b", count: 2 },
      { name: "c", count: 2 },
    ];
    expect(
      items.sort(compare.by((i) => i.count, compare.numbers)).map((i) => i.name),
    ).toEqual(["b", "c", "a"]);
  });

  it("should return 0 for values with equal keys", () => {
    const cmp = compare.by((i: Item) => i.name);
    expect(cmp({ name: "a", count: 1 }, { name: "a", count: 2 })).toBe(0);
  });
});

describe("reverse", () => {
  it("should invert the order of the comparator", () => {
    const cmp = compare.reverse(compare.numbers);
    expect(cmp(1, 2)).toBeGreaterThan(0);
    expect(cmp(2, 1)).toBeLessThan(0);
    expect(cmp(1, 1)).toBe(0);
  });

  it("should sort in descending order", () => {
    expect([1, 3, 2, 3].sort(compare.reverse(compare.numbers))).toEqual([3, 3, 2, 1]);
  });

  it("should restore the order when applied twice", () => {
    const cmp = compare.reverse(compare.reverse(compare.text));
    expect(["b", "a", "c"].sort(cmp)).toEqual(["a", "b", "c"]);
  });
});

describe("chain", () => {
  interface Item {
    group: string;
    rank: number;
  }
  const byGroup = compare.by((i: Item) => i.group);
  const byRank = compare.by((i: Item) => i.rank, compare.numbers);

  it("should return the first non-zero result", () => {
    const cmp = compare.chain(byGroup, byRank);
    expect(cmp({ group: "a", rank: 2 }, { group: "b", rank: 1 })).toBe(-1);
    expect(cmp({ group: "a", rank: 2 }, { group: "a", rank: 1 })).toBeGreaterThan(0);
  });

  it("should not call a later comparator once an earlier one decides", () => {
    const later = vi.fn<compare.Comparator<Item>>(() => 0);
    compare.chain(byGroup, later)({ group: "a", rank: 1 }, { group: "b", rank: 1 });
    expect(later).not.toHaveBeenCalled();
  });

  it("should return 0 when every comparator returns 0", () => {
    const cmp = compare.chain(byGroup, byRank);
    expect(cmp({ group: "a", rank: 1 }, { group: "a", rank: 1 })).toBe(0);
  });

  it("should return 0 when given no comparators", () => {
    expect(compare.chain<number>()(1, 2)).toBe(0);
  });

  it("should sort by each comparator in turn", () => {
    const items: Item[] = [
      { group: "b", rank: 1 },
      { group: "a", rank: 2 },
      { group: "a", rank: 1 },
    ];
    expect(items.sort(compare.chain(byGroup, compare.reverse(byRank)))).toEqual([
      { group: "a", rank: 2 },
      { group: "a", rank: 1 },
      { group: "b", rank: 1 },
    ]);
  });
});
