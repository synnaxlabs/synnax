// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { location } from "@/spatial/location";
import { xy } from "@/spatial/xy";

describe("XY", () => {
  describe("construction", () => {
    [
      ["from object", { x: 1, y: 2 }],
      ["from couple", [1, 2]],
      ["from dimensions", { width: 1, height: 2 }],
      ["from signed dimensions", { signedWidth: 1, signedHeight: 2 }],
      ["from client dimensions", { clientX: 1, clientY: 2 }],
    ].forEach(([name, arg]) => {
      test(name as string, () => {
        const p = xy.construct(arg as xy.Crude);
        expect(p.x).toEqual(1);
        expect(p.y).toEqual(2);
      });
    });
    it("should set the x coordinate if the direction is x", () => {
      const p = xy.construct("x", 1);
      expect(p.x).toEqual(1);
      expect(p.y).toEqual(0);
    });
    it("should set the y coordinate if the direction is y", () => {
      const p = xy.construct("y", 1);
      expect(p.x).toEqual(0);
      expect(p.y).toEqual(1);
    });
  });
  test("translateX", () => {
    let p = xy.construct([1, 2]);
    p = xy.translateX(p, 5);
    expect(p.x).toEqual(6);
    expect(p.y).toEqual(2);
  });
  test("translateY", () => {
    let p = xy.construct([1, 2]);
    p = xy.translateY(p, 5);
    expect(p.x).toEqual(1);
    expect(p.y).toEqual(7);
  });
  test("translate", () => {
    let p = xy.construct([1, 2]);
    p = xy.translate(p, [5, 5]);
    expect(p.x).toEqual(6);
    expect(p.y).toEqual(7);
  });

  test("translate multiple", () => {
    let p = xy.construct([1, 2]);
    p = xy.translate(p, [5, 5], [2, 2]);
    expect(p.x).toEqual(8);
    expect(p.y).toEqual(9);
  });

  describe("translate with Direction", () => {
    it("should translate in the x direction", () => {
      const p = xy.construct([1, 2]);
      const result = xy.translate(p, "x", 5);
      expect(result.x).toEqual(6);
      expect(result.y).toEqual(2);
    });

    it("should translate in the y direction", () => {
      const p = xy.construct([1, 2]);
      const result = xy.translate(p, "y", 3);
      expect(result.x).toEqual(1);
      expect(result.y).toEqual(5);
    });

    it("should translate negative values in x direction", () => {
      const p = xy.construct([10, 20]);
      const result = xy.translate(p, "x", -7);
      expect(result.x).toEqual(3);
      expect(result.y).toEqual(20);
    });

    it("should translate negative values in y direction", () => {
      const p = xy.construct([10, 20]);
      const result = xy.translate(p, "y", -15);
      expect(result.x).toEqual(10);
      expect(result.y).toEqual(5);
    });

    it("should work with different input formats", () => {
      const couple = xy.translate([5, 5], "x", 10);
      expect(couple).toEqual({ x: 15, y: 5 });

      const dims = xy.translate({ width: 3, height: 4 }, "y", 6);
      expect(dims).toEqual({ x: 3, y: 10 });
    });
  });

  describe("equals", () => {
    const TESTS: Array<[xy.Crude, xy.Crude, boolean]> = [
      [[1, 1], { x: 1, y: 1 }, true],
      [[1, 1], [1, 1], true],
      [{ x: 1, y: 12 }, { x: 1, y: 1 }, false],
      [{ x: 1, y: 12 }, { width: 1, height: 12 }, true],
      [{ x: 1, y: 12 }, { width: 12, height: 1 }, false],
      [{ x: 1, y: 12 }, { signedWidth: 1, signedHeight: 12 }, true],
    ];
    TESTS.forEach(([one, two, expected], i) => {
      test(`equals ${i}`, () => {
        const p = xy.construct(one);
        expect(xy.equals(p, two)).toEqual(expected);
      });
    });
    it("should retrun true if the two points are within the given threshold", () => {
      const p = xy.construct([1, 1]);
      expect(xy.equals(p, [1.1, 1.1], 0.15)).toBeTruthy();
    });
  });
  test("couple", () => {
    const p = xy.construct([1, 2]);
    expect(xy.couple(p)).toEqual([1, 2]);
  });
  describe("isNan", () => {
    it("should return true if x or y is NaN", () => {
      expect(xy.isNan(xy.construct([1, NaN]))).toBeTruthy();
      expect(xy.isNan(xy.construct([NaN, 1]))).toBeTruthy();
    });
  });
  describe("isFinite", () => {
    it("should return true if x or y is finite", () => {
      expect(xy.isFinite(xy.construct([1, 2]))).toBeTruthy();
      expect(xy.isFinite(xy.construct([Infinity, 2]))).toBeFalsy();
      expect(xy.isFinite(xy.construct([1, Infinity]))).toBeFalsy();
    });
  });
  describe("distance", () => {
    it("should return the distance between two points", () => {
      expect(xy.distance([1, 1], [1, 1])).toBe(0);
      expect(xy.distance([1, 1], [1, 2])).toBe(1);
      expect(xy.distance([1, 1], [2, 2])).toBe(Math.sqrt(2));
    });
  });
  describe("css", () => {
    it("should return the css properties of the point", () => {
      expect(xy.css([1, 2])).toEqual({ left: 1, top: 2 });
    });
  });
  describe("truncate", () => {
    it("should truncate the point to the given precision", () => {
      expect(xy.truncate([1.12345, 2.12345], 2)).toEqual({ x: 1.12, y: 2.12 });
    });
  });
  describe("scale", () => {
    it("should scale the point by the given factor", () => {
      expect(xy.scale([1, 2], 2)).toEqual({ x: 2, y: 4 });
    });
  });
  describe("sub", () => {
    it("should subtract the second point from the first point", () => {
      expect(xy.sub([1, 2], [2, 1])).toEqual({ x: -1, y: 1 });
    });
  });

  describe("calculateMiters", () => {
    it("should calculate the miters of the given points", () => {
      const points: xy.XY[] = [
        { x: 0, y: 0 },
        { x: 0, y: 1 },
        { x: 1, y: 1 },
      ];
      const miters = xy.calculateMiters(points, 1);
      expect(miters).toEqual([
        { x: -1, y: 0 },
        { x: -1, y: 1 },
        { x: -0, y: 1 },
      ]);
    });
  });

  describe("swap", () => {
    it("should swap x and y coordinates", () => {
      expect(xy.swap([1, 2])).toEqual({ x: 2, y: 1 });
      expect(xy.swap({ x: 3, y: 4 })).toEqual({ x: 4, y: 3 });
      expect(xy.swap({ width: 5, height: 6 })).toEqual({ x: 6, y: 5 });
    });
  });

  describe("translate with location.XY", () => {
    it("should translate with top-left location", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.TOP_LEFT, [5, 5]);
      expect(result).toEqual({ x: 5, y: 5 });
    });

    it("should translate with top-right location", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.TOP_RIGHT, [5, 5]);
      expect(result).toEqual({ x: 15, y: 5 });
    });

    it("should translate with bottom-left location", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.BOTTOM_LEFT, [5, 5]);
      expect(result).toEqual({ x: 5, y: 15 });
    });

    it("should translate with bottom-right location", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.BOTTOM_RIGHT, [5, 5]);
      expect(result).toEqual({ x: 15, y: 15 });
    });

    it("should handle center x locations", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.TOP_CENTER, [5, 5]);
      expect(result).toEqual({ x: 10, y: 5 });

      const result2 = xy.translate(p, location.BOTTOM_CENTER, [5, 5]);
      expect(result2).toEqual({ x: 10, y: 15 });
    });

    it("should handle center y locations", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.CENTER_LEFT, [5, 5]);
      expect(result).toEqual({ x: 5, y: 10 });

      const result2 = xy.translate(p, location.CENTER_RIGHT, [5, 5]);
      expect(result2).toEqual({ x: 15, y: 10 });
    });

    it("should handle center-center location", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.CENTER, [5, 5]);
      expect(result).toEqual({ x: 10, y: 10 });
    });

    it("should work with different coordinate input formats", () => {
      const result1 = xy.translate([10, 10], { x: "left", y: "top" }, [5, 5]);
      expect(result1).toEqual({ x: 5, y: 5 });

      const result2 = xy.translate(
        { width: 10, height: 10 },
        { x: "right", y: "bottom" },
        { width: 5, height: 5 },
      );
      expect(result2).toEqual({ x: 15, y: 15 });
    });

    it("should handle negative translations correctly", () => {
      const p = xy.construct([10, 10]);
      const result = xy.translate(p, location.TOP_LEFT, [-5, -5]);
      expect(result).toEqual({ x: 15, y: 15 });

      const result2 = xy.translate(p, location.BOTTOM_RIGHT, [-5, -5]);
      expect(result2).toEqual({ x: 5, y: 5 });
    });

    it("should work with custom location objects", () => {
      const p = xy.construct([10, 10]);
      const customLocation: location.XY = { x: "left", y: "bottom" };
      const result = xy.translate(p, customLocation, [3, 7]);
      expect(result).toEqual({ x: 7, y: 17 });
    });
  });
});
