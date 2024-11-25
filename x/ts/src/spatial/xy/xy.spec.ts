// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import * as xy from "@/spatial/xy/xy";

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
      expect(xy.equals(p, [1.1, 1.1], 0.15)).toBe(true);
    });
  });
  test("couple", () => {
    const p = xy.construct([1, 2]);
    expect(xy.couple(p)).toEqual([1, 2]);
  });
  describe("isNan", () => {
    it("should return true if x or y is NaN", () => {
      expect(xy.isNan(xy.construct([1, NaN]))).toBe(true);
      expect(xy.isNan(xy.construct([NaN, 1]))).toBe(true);
    });
  });
  describe("isFinite", () => {
    it("should return true if x or y is finite", () => {
      expect(xy.isFinite(xy.construct([1, 2]))).toBe(true);
      expect(xy.isFinite(xy.construct([Infinity, 2]))).toBe(false);
      expect(xy.isFinite(xy.construct([1, Infinity]))).toBe(false);
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
});
