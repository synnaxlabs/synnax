// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import * as bounds from "@/spatial/bounds/bounds";

describe("Bounds", () => {
  describe("construction", () => {
    type T = [string, bounds.Crude];
    const TESTS: T[] = [
      ["from couple", [1, 2]],
      ["from bounds", { lower: 1, upper: 2 }],
      ["from invalid bounds", { upper: 1, lower: 2 }],
    ];
    TESTS.forEach(([name, arg]) => {
      test(name, () => {
        const bound = bounds.construct(arg);
        expect(bound.lower).toEqual(1);
        expect(bound.upper).toEqual(2);
      });
    });
    it("should consider a single argument as the upper bound", () => {
      const bound = bounds.construct(1);
      expect(bound.lower).toEqual(0);
      expect(bound.upper).toEqual(1);
    });
  });
  describe("contains", () => {
    it("should return true if the value is within the bounds", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 1.5)).toEqual(true);
    });
    it("should return false if the value is outside the bounds", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 2.5)).toEqual(false);
    });
    it("should return true if the value is equal to the lower bound", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 1)).toEqual(true);
    });
    it("should return false if the value is equal to the upper bound", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.contains(b, 2)).toEqual(false);
    });
    it("should return true if a bound contains another boun", () => {
      const b1 = bounds.construct([1, 3]);
      const b2 = bounds.construct([2, 3]);
      expect(bounds.contains(b1, b2)).toEqual(true);
      expect(bounds.contains(b2, b1)).toEqual(false);
    });
    it("should return true if two bounds are equal", () => {
      const b1 = bounds.construct([1, 3]);
      const b2 = bounds.construct([1, 3]);
      expect(bounds.contains(b1, b2)).toEqual(true);
      expect(bounds.contains(b2, b1)).toEqual(true);
    });
  });
  describe("span", () => {
    it("should return the span of the bound", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.span(b)).toEqual(1);
    });
  });
  test("isZero", () => {
    const b = bounds.construct([0, 0]);
    expect(bounds.isZero(b)).toEqual(true);
  });
  test("spanIsZero", () => {
    const b = bounds.construct([1, 1]);
    expect(bounds.spanIsZero(b)).toEqual(true);
  });
  describe("max", () => {
    it("should return the bound with the maximum possible span", () => {
      const args: bounds.Crude[] = [
        [1, 2],
        [-1, 1],
      ];
      const bound = bounds.max(args);
      expect(bound.lower).toEqual(-1);
      expect(bound.upper).toEqual(2);
    });
  });
  describe("min", () => {
    it("should return the bound with the minimum possible span", () => {
      const args: bounds.Crude[] = [
        [1, 2],
        [-1, 1],
      ];
      const bound = bounds.min(args);
      expect(bound.lower).toEqual(1);
      expect(bound.upper).toEqual(1);
    });
  });
  describe("isFinite", () => {
    it("should return false if either bound is infinite", () => {
      const b = bounds.construct([1, Infinity]);
      expect(bounds.isFinite(b)).toEqual(false);
    });
    it("should return true if both bounds are finite", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.isFinite(b)).toEqual(true);
    });
  });
  describe("overlapsWith", () => {
    it("should return false if the bounds are adjacent", () => {
      const a = bounds.construct([1, 2]);
      const b = bounds.construct([2, 3]);
      expect(bounds.overlapsWith(a, b)).toEqual(false);
    });
    it("should return false if the bounds are adjacent", () => {
      const a = bounds.construct([2, 3]);
      const b = bounds.construct([1, 2]);
      expect(bounds.overlapsWith(a, b)).toEqual(false);
    });
    it("should return true if the bounds overlap", () => {
      const a = bounds.construct([1, 2]);
      const b = bounds.construct([1.5, 3]);
      expect(bounds.overlapsWith(a, b)).toEqual(true);
    });
    it("should return false if the bounds are disjoint", () => {
      const a = bounds.construct([1, 2]);
      const b = bounds.construct([3, 4]);
      expect(bounds.overlapsWith(a, b)).toEqual(false);
    });
    it("should return true if the bounds are equal", () => {
      const a = bounds.construct([1, 2]);
      const b = bounds.construct([1, 2]);
      expect(bounds.overlapsWith(a, b)).toEqual(true);
    });
  });
  describe("findInsertPosition", () => {
    const SPECS: Array<[bounds.Crude[], number, { index: number; position: number }]> =
      [
        [[[1, 3]], 2, { index: 0, position: 1 }],
        [[[1, 3]], 3, { index: 1, position: 0 }],
        [[[1, 3]], 4, { index: 1, position: 0 }],
        [[[1, 3]], 4, { index: 1, position: 0 }],
        [
          [
            [1, 3],
            [4, 6],
          ],
          5,
          { index: 1, position: 1 },
        ],
        [
          [
            [1, 2],
            [3, 4],
          ],
          3,
          { index: 1, position: 0 },
        ],
        [
          [
            [1, 3],
            [3, 4],
          ],
          3,
          { index: 1, position: 0 },
        ],
        [
          [
            [1, 2],
            [3, 5],
            [7, 10],
          ],
          6,
          { index: 2, position: 0 },
        ],
        [
          [
            [1, 2],
            [3, 5],
            [7, 10],
          ],
          2,
          { index: 1, position: 0 },
        ],
        [
          [
            [1, 2],
            [3, 5],
            [7, 10],
          ],
          3,
          { index: 1, position: 0 },
        ],
        [
          [
            [1, 2],
            [3, 5],
            [7, 10],
          ],
          4,
          { index: 1, position: 1 },
        ],
        [[[3, 7]], 1, { index: 0, position: 0 }],
        [[[3, 7]], 8, { index: 1, position: 0 }],
      ];
    SPECS.forEach(([b, target, expected]) => {
      test(`should return ${expected.index}:${expected.position} for ${target}`, () => {
        expect(bounds.findInsertPosition(b, target)).toEqual(expected);
      });
    });
  });
  describe("insert", () => {
    describe("formal cases", () => {
      test("insert before adjacent upper", () => {
        const b: bounds.Crude[] = [
          [2, 3],
          [3, 4],
          [5, 6],
        ];
        const v: bounds.Crude = [1, 2];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 1, upper: 2 },
          { lower: 2, upper: 3 },
          { lower: 3, upper: 4 },
          { lower: 5, upper: 6 },
        ]);
      });
      test("insert in-between with no overlap", () => {
        const b: bounds.Crude[] = [
          [2, 3],
          [3, 4],
          [7, 8],
        ];
        const v: bounds.Crude = [5, 6];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 2, upper: 3 },
          { lower: 3, upper: 4 },
          { lower: 5, upper: 6 },
          { lower: 7, upper: 8 },
        ]);
      });
      test("insert in-between adjacent lower and upper", () => {
        const b: bounds.Crude[] = [
          [2, 3],
          [3, 4],
          [5, 6],
        ];
        const v: bounds.Crude = [4, 5];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 2, upper: 3 },
          { lower: 3, upper: 4 },
          { lower: 4, upper: 5 },
          { lower: 5, upper: 6 },
        ]);
      });
      test("insert in-between adjacent lower not upper", () => {
        const b: bounds.Crude[] = [
          [2, 3],
          [3, 4],
          [7, 8],
        ];
        const v: bounds.Crude = [4, 6];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 2, upper: 3 },
          { lower: 3, upper: 4 },
          { lower: 4, upper: 6 },
          { lower: 7, upper: 8 },
        ]);
      });
      test("insert in-between overlap lower adjacent upper", () => {
        const b: bounds.Crude[] = [
          [2, 4],
          [5, 6],
          [7, 8],
        ];
        const v: bounds.Crude = [3, 5];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 2, upper: 4 },
          { lower: 4, upper: 5 },
          { lower: 5, upper: 6 },
          { lower: 7, upper: 8 },
        ]);
      });
      test("insert in-between adjacent lower contain 1 adjacent upper", () => {
        const b: bounds.Crude[] = [
          [2, 4],
          [5, 6],
          [7, 8],
        ];
        const v: bounds.Crude = [4, 7];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 2, upper: 4 },
          { lower: 4, upper: 7 },
          { lower: 7, upper: 8 },
        ]);
      });
      test("insert in-between overlap lower consume 1 adjacent upper", () => {
        const b: bounds.Crude[] = [
          [2, 4],
          [5, 6],
          [7, 8],
        ];
        const v: bounds.Crude = [3, 7];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 2, upper: 4 },
          { lower: 4, upper: 7 },
          { lower: 7, upper: 8 },
        ]);
      });
      test("insert in-between replace 1", () => {
        const b: bounds.Crude[] = [
          [2, 4],
          [5, 6],
          [7, 8],
        ];
        const v: bounds.Crude = [5, 6];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 2, upper: 4 },
          { lower: 5, upper: 6 },
          { lower: 7, upper: 8 },
        ]);
      });
      test("insert before overlap first", () => {
        const b: bounds.Crude[] = [[3, 7]];
        const v: bounds.Crude = [1, 4];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 1, upper: 3 },
          { lower: 3, upper: 7 },
        ]);
      });
      test("insert before no adjacent", () => {
        const b: bounds.Crude[] = [[3, 7]];
        const v: bounds.Crude = [1, 2];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 1, upper: 2 },
          { lower: 3, upper: 7 },
        ]);
      });
      test("insert after no adjacent", () => {
        const b: bounds.Crude[] = [[3, 7]];
        const v: bounds.Crude = [8, 9];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 3, upper: 7 },
          { lower: 8, upper: 9 },
        ]);
      });
      test("insert after adjacent upper", () => {
        const b: bounds.Crude[] = [[3, 7]];
        const v: bounds.Crude = [7, 9];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 3, upper: 7 },
          { lower: 7, upper: 9 },
        ]);
      });
      test("insert after overlap last", () => {
        const b: bounds.Crude[] = [[3, 7]];
        const v: bounds.Crude = [5, 9];
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 3, upper: 7 },
          { lower: 7, upper: 9 },
        ]);
      });
    });
    describe("regressions", () => {
      it("should insert a value into a sorted array of bounds", () => {
        const b: bounds.Crude[] = [
          { lower: 0, upper: 47040 },
          { lower: 47040, upper: 47240 },
          { lower: 47240, upper: 47280 },
          { lower: 47280, upper: 47320 },
          { lower: 47320, upper: 47400 },
        ];
        const v: bounds.Crude = { lower: 47066, upper: 47066 + 354 };
        const result = bounds.insert(b, v);
        expect(result).toEqual([
          { lower: 0, upper: 47040 },
          { lower: 47040, upper: 47240 },
          { lower: 47240, upper: 47066 + 354 },
        ]);
      });
      it("should insert a value into a sorted array of bounds", () => {});
    });
  });
});
