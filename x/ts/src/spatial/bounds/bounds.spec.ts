// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { bounds } from "@/spatial/bounds";

// More items than a call takes as arguments.
const MANY = 200_000;

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

    describe("partial bounds", () => {
      it("should default a missing upper to Infinity", () => {
        const bound = bounds.construct({ lower: 0 });
        expect(bound.lower).toEqual(0);
        expect(bound.upper).toEqual(Infinity);
      });
      it("should default a missing lower to -Infinity", () => {
        const bound = bounds.construct({ upper: 10 });
        expect(bound.lower).toEqual(-Infinity);
        expect(bound.upper).toEqual(10);
      });
      it("should default both bounds when given an empty object", () => {
        const bound = bounds.construct({});
        expect(bound.lower).toEqual(-Infinity);
        expect(bound.upper).toEqual(Infinity);
      });
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
    it("should compare reversed bounds as given", () => {
      expect(bounds.equals([1, 2], [2, 1])).toEqual(false);
      expect(bounds.equals([2, 1], [2, 1])).toEqual(true);
    });
    it("should tell INVALID apart from INFINITE", () => {
      expect(bounds.equals(bounds.INVALID, bounds.INVALID)).toEqual(true);
      expect(bounds.equals(bounds.INVALID, bounds.INFINITE)).toEqual(false);
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
    it("should contain no values when the lower is above the upper", () => {
      const b = { lower: 2, upper: 1 };
      expect(bounds.contains(b, 1)).toEqual(false);
      expect(bounds.contains(b, 1.5)).toEqual(false);
      expect(bounds.contains(b, 2)).toEqual(false);
      expect(bounds.contains(b, [1.2, 1.8])).toEqual(false);
    });
    it("should contain nothing in INVALID", () => {
      expect(bounds.contains(bounds.INVALID, 0)).toEqual(false);
      expect(bounds.contains(bounds.INVALID, Infinity)).toEqual(false);
      expect(bounds.contains(bounds.INVALID, -Infinity)).toEqual(false);
      expect(bounds.contains(bounds.INVALID, [1, 2])).toEqual(false);
    });
  });
  describe("span", () => {
    it("should return the span of the bound", () => {
      const b = bounds.construct([1, 2]);
      expect(bounds.span(b)).toEqual(1);
    });
    it("should return a negative span when the lower is above the upper", () => {
      expect(bounds.span({ lower: 2, upper: 1 })).toEqual(-1);
    });
    it("should return a span of -Infinity for INVALID", () => {
      expect(bounds.span(bounds.INVALID)).toEqual(-Infinity);
    });
  });
  describe("union", () => {
    it("should return the bound with the maximum possible span", () => {
      const args: bounds.Crude[] = [
        [1, 2],
        [-1, 1],
      ];
      const bound = bounds.union(args);
      expect(bound.lower).toEqual(-1);
      expect(bound.upper).toEqual(2);
    });
    it("should treat INVALID as the union identity", () => {
      const bound = bounds.union([[-250, -100], bounds.INVALID]);
      expect(bound).toStrictEqual({ lower: -250, upper: -100 });
    });
    it("should preserve a reversed member instead of reordering it", () => {
      const bound = bounds.union([{ lower: 20, upper: 10 }]);
      expect(bound).toStrictEqual({ lower: 20, upper: 10 });
    });
    it("should take more bounds than a call takes arguments", () => {
      const many = Array.from({ length: MANY }, (_, i): bounds.Crude => [i, i + MANY]);
      expect(bounds.union(many)).toStrictEqual({ lower: 0, upper: 2 * MANY - 1 });
    });
  });
  describe("intersection", () => {
    it("should return the bound with the minimum possible span", () => {
      const args: bounds.Crude[] = [
        [1, 2],
        [-1, 1],
      ];
      const bound = bounds.intersection(args);
      expect(bound.lower).toEqual(1);
      expect(bound.upper).toEqual(1);
    });
    it("should treat INVALID as absorbing", () => {
      const bound = bounds.intersection([[-250, -100], bounds.INVALID]);
      expect(bound).toStrictEqual(bounds.INVALID);
    });
    it("should produce reversed bounds for disjoint inputs", () => {
      const bound = bounds.intersection([
        [0, 10],
        [20, 30],
      ]);
      expect(bound).toStrictEqual({ lower: 20, upper: 10 });
    });
    it("should take more bounds than a call takes arguments", () => {
      const many = Array.from({ length: MANY }, (_, i): bounds.Crude => [i, i + MANY]);
      expect(bounds.intersection(many)).toStrictEqual({ lower: MANY - 1, upper: MANY });
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
    it("should return false for INVALID", () => {
      expect(bounds.isFinite(bounds.INVALID)).toEqual(false);
    });
    it("should return true for finite bounds with the lower above the upper", () => {
      expect(bounds.isFinite({ lower: 2, upper: 1 })).toEqual(true);
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
    it("should return false when either lower is above its upper", () => {
      const reversed = { lower: 8, upper: 2 };
      expect(bounds.overlapsWith([0, 10], reversed)).toEqual(false);
      expect(bounds.overlapsWith(reversed, [0, 10])).toEqual(false);
      expect(bounds.overlapsWith(reversed, reversed)).toEqual(false);
    });
    it("should return false for INVALID", () => {
      expect(bounds.overlapsWith(bounds.INVALID, [0, 10])).toEqual(false);
      expect(bounds.overlapsWith(bounds.INFINITE, bounds.INVALID)).toEqual(false);
      expect(bounds.overlapsWith(bounds.INVALID, bounds.INVALID)).toEqual(false);
    });
  });
  describe("clamp", () => {
    it("should project the target onto the closed interval", () => {
      const b = bounds.construct([1, 3]);
      expect(bounds.clamp(b, 0)).toEqual(1);
      expect(bounds.clamp(b, 2)).toEqual(2);
      expect(bounds.clamp(b, 4)).toEqual(3);
    });
    it("should return the upper bound when the target exceeds it", () => {
      expect(bounds.clamp(bounds.DECIMAL, 1)).toEqual(1);
      expect(bounds.clamp(bounds.DECIMAL, 5)).toEqual(1);
    });
    it("should return the lower bound when the target is below it", () => {
      expect(bounds.clamp(bounds.DECIMAL, -5)).toEqual(0);
    });
    it("should read the lower and upper as given when the lower is above the upper", () => {
      const b = { lower: 3, upper: 1 };
      expect(bounds.clamp(b, 0)).toEqual(3);
      expect(bounds.clamp(b, 2)).toEqual(3);
      expect(bounds.clamp(b, 4)).toEqual(1);
    });
    it("should clamp a finite target in INVALID to its lower", () => {
      expect(bounds.clamp(bounds.INVALID, 0)).toEqual(Infinity);
    });
  });
  describe("mean", () => {
    it("should return the mean of the bounds", () => {
      const b = bounds.construct([1, 3]);
      expect(bounds.mean(b)).toEqual(2);
    });
    it("should return the same mean when the lower is above the upper", () => {
      expect(bounds.mean({ lower: 3, upper: 1 })).toEqual(2);
    });
    it("should return NaN for INVALID", () => {
      expect(bounds.mean(bounds.INVALID)).toBeNaN();
    });
  });
});
