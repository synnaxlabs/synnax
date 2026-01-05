// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { type numeric } from "@/numeric";
import { bounds } from "@/spatial/bounds";
import { testutil } from "@/testutil";

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

    it("should accept two arguments", () => {
      const bound = bounds.construct(1, 2);
      expect(bound.lower).toEqual(1);
      expect(bound.upper).toEqual(2);
    });

    it("should consider a single argument as the upper bound", () => {
      const bound = bounds.construct(1);
      expect(bound.lower).toEqual(0);
      expect(bound.upper).toEqual(1);
    });

    describe("makeValid", () => {
      it("should make the bounds valid", () => {
        const bound = bounds.construct<number>(2, 1, { makeValid: true });
        expect(bound.lower).toEqual(1);
        expect(bound.upper).toEqual(2);
      });
      it("should allow for options in a single argument constructor", () => {
        const bound = bounds.construct<number>([2, 1], { makeValid: false });
        expect(bound.lower).toEqual(2);
        expect(bound.upper).toEqual(1);
      });
      it("should not make the bounds valid if makeValid is false", () => {
        const bound = bounds.construct(1, 2, { makeValid: false });
        expect(bound.lower).toEqual(1);
        expect(bound.upper).toEqual(2);
      });
    });
  });
  describe("equals", () => {
    it("should return true if the bounds are equal", () => {
      const a = bounds.construct([1, 2]);
      const b = bounds.construct([1, 2]);
      expect(bounds.equals(a, b)).toEqual(true);
    });
    it("should return true if both bounds are undefined", () => {
      expect(bounds.equals(undefined, undefined)).toEqual(true);
    });
    it("should return false if the bounds are not equal", () => {
      const a = bounds.construct([1, 2]);
      const b = bounds.construct([1, 3]);
      expect(bounds.equals(a, b)).toEqual(false);
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
      test(`should return ${JSON.stringify(expected)} for ${JSON.stringify(b)} and ${target}`, () => {
        expect(bounds.findInsertPosition(b, target)).toEqual(expected);
      });
    });
  });

  describe("traverse", () => {
    interface Spec<T extends numeric.Value> {
      bounds: Array<bounds.Crude<T>>;
      start: T;
      dist: T;
      expected: T;
    }

    const SPECS: Spec<numeric.Value>[] = [
      {
        bounds: [
          [0, 10],
          [10, 20],
        ],
        start: 5,
        dist: 5,
        expected: 10,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 12,
        dist: 3,
        expected: 18,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 12,
        dist: -3,
        expected: 7,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 10,
        dist: -3,
        expected: 7,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 15,
        dist: -3,
        expected: 7,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 16,
        dist: -3,
        expected: 8,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 16,
        dist: 12,
        expected: 20,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 16,
        dist: -20,
        expected: 0,
      },
      {
        bounds: [[620n, 726n]],
        start: 724n,
        dist: -6n,
        expected: 718n,
      },
    ];
    SPECS.forEach(({ bounds: b, start, dist, expected }) => {
      test(`should return ${expected} for ${testutil.toString(b)} and ${start} and ${dist}`, () => {
        expect(bounds.traverse(b, start, dist)).toEqual(expected);
      });
    });
  });

  describe("distance", () => {
    interface Spec<T extends numeric.Value> {
      bounds: Array<bounds.Crude<T>>;
      start: T;
      end: T;
      expected: T;
    }

    const SPECS: Spec<numeric.Value>[] = [
      {
        bounds: [
          [0, 10],
          [10, 20],
        ],
        start: 5,
        end: 15,
        expected: 10,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 12,
        end: 18,
        expected: 3,
      },
      {
        bounds: [
          [0, 10],
          [15, 20],
        ],
        start: 10,
        end: 5,
        expected: 5,
      },
    ];
    SPECS.forEach(({ bounds: b, start, end, expected }) => {
      test(`should return ${expected} for ${JSON.stringify(b)} and ${start} and ${end}`, () => {
        expect(bounds.distance(b, start, end)).toEqual(expected);
      });
    });
  });

  describe("linspace", () => {
    it("should create the correct array", () => {
      const b = bounds.construct([1, 3]);
      const result = bounds.linspace(b);
      expect(result).toEqual([1, 2]);
    });
  });
  describe("clamp", () => {
    it("should clamp the provided target to the bounds", () => {
      const b = bounds.construct([1, 3]);
      expect(bounds.clamp(b, 0)).toEqual(1);
      expect(bounds.clamp(b, 2)).toEqual(2);
      expect(bounds.clamp(b, 4)).toEqual(2);
    });
  });
  describe("mean", () => {
    it("should return the mean of the bounds", () => {
      const b = bounds.construct([1, 3]);
      expect(bounds.mean(b)).toEqual(2);
    });
  });
});
