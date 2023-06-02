// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, test, it } from "vitest";

import {
  Bound,
  Dimensions,
  LooseBoundT,
  LooseDimensionsT,
  LooseXYT,
  XY,
} from "@/spatial";

describe.only("Spatial Core", () => {
  describe("XY", () => {
    describe("construction", () => {
      [
        ["from object", { x: 1, y: 2 }],
        ["from couple", [1, 2]],
        ["from dimensions", { width: 1, height: 2 }],
        ["from signed dimensions", { signedWidth: 1, signedHeight: 2 }],
      ].forEach(([name, arg]) => {
        test(name as string, () => {
          const xy = new XY(arg as LooseXYT);
          expect(xy.x).toEqual(1);
          expect(xy.y).toEqual(2);
        });
      });
    });
  });
  test("translateX", () => {
    let xy = new XY([1, 2]);
    xy = xy.translateX(5);
    expect(xy.x).toEqual(6);
    expect(xy.y).toEqual(2);
  });
  test("translateY", () => {
    let xy = new XY([1, 2]);
    xy = xy.translateY(5);
    expect(xy.x).toEqual(1);
    expect(xy.y).toEqual(7);
  });
  test("translate", () => {
    let xy = new XY([1, 2]);
    xy = xy.translate([5, 5]);
    expect(xy.x).toEqual(6);
    expect(xy.y).toEqual(7);
  });
  describe("equals", () => {
    const TESTS: Array<[LooseXYT, LooseXYT, boolean]> = [
      [[1, 1], { x: 1, y: 1 }, true],
      [[1, 1], [1, 1], true],
      [{ x: 1, y: 12 }, { x: 1, y: 1 }, false],
      [{ x: 1, y: 12 }, { width: 1, height: 12 }, true],
      [{ x: 1, y: 12 }, { width: 12, height: 1 }, false],
      [{ x: 1, y: 12 }, { signedWidth: 1, signedHeight: 12 }, true],
    ];
    TESTS.forEach(([one, two, expected], i) => {
      test(`equals ${i}`, () => {
        const xy = new XY(one);
        expect(xy.equals(new XY(two))).toEqual(expected);
      });
    });
  });
  test("couple", () => {
    const xy = new XY([1, 2]);
    expect(xy.couple).toEqual([1, 2]);
  });

  describe("Dimensions", () => {
    describe("construction", () => {
      [
        ["from object", { width: 1, height: 2 }],
        ["from couple", [1, 2]],
        ["from dimensions", { width: 1, height: 2 }],
        ["from signed dimensions", { signedWidth: 1, signedHeight: 2 }],
        ["from XY", { x: 1, y: 2 }],
      ].forEach(([name, arg]) => {
        test(name as string, () => {
          const xy = new Dimensions(arg as LooseDimensionsT);
          expect(xy.width).toEqual(1);
          expect(xy.height).toEqual(2);
        });
      });
    });
    test("couple", () => {
      const xy = new Dimensions([1, 2]);
      expect(xy.couple).toEqual([1, 2]);
    });
    describe("equals", () => {
      const TESTS: Array<[LooseDimensionsT, LooseDimensionsT, boolean]> = [
        [[1, 1], { width: 1, height: 1 }, true],
        [[1, 1], [1, 1], true],
        [{ width: 1, height: 12 }, { width: 1, height: 1 }, false],
        [{ width: 1, height: 12 }, { width: 12, height: 1 }, false],
        [{ width: 1, height: 12 }, { signedWidth: 1, signedHeight: 12 }, true],
      ];
      TESTS.forEach(([one, two, expected], i) => {
        test(`equals ${i}`, () => {
          const xy = new Dimensions(one);
          expect(xy.equals(new Dimensions(two))).toEqual(expected);
        });
      });
    });
  });

  describe("Bound", () => {
    describe("construction", () => {
      [
        ["from couple", [1, 2]],
        ["from bounds", { lower: 1, upper: 2 }],
      ].forEach(([name, arg]) => {
        test(name as string, () => {
          const bound = new Bound(arg as LooseBoundT);
          expect(bound.lower).toEqual(1);
          expect(bound.upper).toEqual(2);
        });
      });
      it("should consider a single argument as the upper bound", () => {
        const bound = new Bound(1);
        expect(bound.lower).toEqual(0);
        expect(bound.upper).toEqual(1);
      });
    });
    describe("contains", () => {
      it("should return true if the value is within the bounds", () => {
        const bound = new Bound([1, 2]);
        expect(bound.contains(1.5)).toEqual(true);
      });
      it("should return false if the value is outside the bounds", () => {
        const bound = new Bound([1, 2]);
        expect(bound.contains(2.5)).toEqual(false);
      });
      it("should return true if the value is equal to the lower bound", () => {
        const bound = new Bound([1, 2]);
        expect(bound.contains(1)).toEqual(true);
      });
      it("should return false if the value is equal to the upper bound", () => {
        const bound = new Bound([1, 2]);
        expect(bound.contains(2)).toEqual(false);
      });
    });
    describe("span", () => {
      it("should return the span of the bound", () => {
        const bound = new Bound([1, 2]);
        expect(bound.span).toEqual(1);
      });
    });
    test("isZero", () => {
      const bound = new Bound([0, 0]);
      expect(bound.isZero).toEqual(true);
    });
    test("spanIsZero", () => {
      const bound = new Bound([1, 1]);
      expect(bound.spanIsZero).toEqual(true);
    });
    describe("max", () => {
      it("should return the bound with the maximum possible span", () => {
        const bounds: LooseBoundT[] = [
          [1, 2],
          [-1, 1],
        ];
        const bound = Bound.max(bounds);
        expect(bound.lower).toEqual(-1);
        expect(bound.upper).toEqual(2);
      });
    });
    describe("min", () => {
      it("should return the bound with the minimum possible span", () => {
        const bounds: LooseBoundT[] = [
          [1, 2],
          [-1, 1],
        ];
        const bound = Bound.min(bounds);
        expect(bound.lower).toEqual(1);
        expect(bound.upper).toEqual(1);
      });
    });
  });
});
