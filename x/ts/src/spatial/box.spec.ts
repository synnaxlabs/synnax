// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, test, expect, it } from "vitest";

import { Box, CrudeLocation, CrudeXY, XYLocation, positionInCenter } from "@/spatial";

describe("Box", () => {
  describe("construction", () => {
    test("from dom rect", () => {
      const b = new Box({ left: 0, top: 0, right: 10, bottom: 10 });
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from two points", () => {
      const b = new Box({ x: 0, y: 0 }, { x: 10, y: 10 });
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from point and dimensions", () => {
      const b = new Box({ x: 0, y: 0 }, { width: 10, height: 10 });
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from point and width and height", () => {
      const b = new Box({ x: 0, y: 0 }, 10, 10);
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
    test("from raw params", () => {
      const b = new Box(0, 0, 10, 10);
      expect(b.topLeft).toEqual({ x: 0, y: 0 });
      expect(b.topRight).toEqual({ x: 10, y: 0 });
      expect(b.bottomLeft).toEqual({ x: 0, y: 10 });
      expect(b.bottomRight).toEqual({ x: 10, y: 10 });
    });
  });
  describe("zod schema", () => {
    const CASES: Array<[string, unknown]> = [
      [
        "raw string root",
        {
          root: { x: "left", y: "top" },
          one: { x: 0, y: 0 },
          two: { x: 10, y: 10 },
        },
      ],
      [
        "string instance root",
        {
          root: { x: String("left"), y: String("top") },
          one: { x: 0, y: 0 },
          two: { x: 10, y: 10 },
        },
      ],
    ];
    CASES.forEach(([title, value]) => {
      it(`should parse ${title}`, () => {
        expect(() => Box.z.parse(value)).not.toThrow();
      });
    });
  });
  describe("properties", () => {
    const b = new Box(20, 30, 40, 50);
    describe("loc", () => {
      const v: CrudeLocation[] = ["left", "right", "top", "bottom"];
      const expected: number[] = [20, 60, 30, 80];
      v.forEach((v, i) => {
        test(`loc-${v}`, () => {
          expect(b.loc(v)).toEqual(expected[i]);
        });
      });
    });
    describe("xyLoc", () => {
      const v: XYLocation[] = [
        XYLocation.BOTTOM_CENTER,
        XYLocation.LEFT_CENTER,
        XYLocation.RIGHT_CENTER,
        XYLocation.TOP_CENTER,
        XYLocation.BOTTOM_LEFT,
        XYLocation.BOTTOM_RIGHT,
        XYLocation.TOP_LEFT,
        XYLocation.TOP_RIGHT,
      ];
      const expected: CrudeXY[] = [
        { x: 40, y: 80 },
        { x: 20, y: 55 },
        { x: 60, y: 55 },
        { x: 40, y: 30 },
        { x: 20, y: 80 },
        { x: 60, y: 80 },
        { x: 20, y: 30 },
        { x: 60, y: 30 },
      ];
      v.forEach((v, i) => {
        test(`xyLoc-${v.toString()}`, () => {
          expect(b.xyLoc(v)).toEqual(expected[i]);
        });
      });
    });
  });
  describe("equality", () => {
    it("should be equal to itself", () => {
      const b = new Box(0, 0, 10, 10);
      expect(b.equals(b)).toBe(true);
    });
    it("should be equal to a box with the same values", () => {
      const b = new Box(0, 0, 10, 10);
      const b2 = new Box(0, 0, 10, 10);
      expect(b.equals(b2)).toBe(true);
    });
  });
  describe("positionInCenterOf", () => {
    it("should position the box in the center of the other box", () => {
      let b = new Box(0, 0, 10, 10);
      const b2 = new Box(0, 0, 20, 20);
      b = positionInCenter(b, b2);
      expect(b.topLeft).toEqual({ x: 5, y: 5 });
      expect(b.topRight).toEqual({ x: 15, y: 5 });
      expect(b.bottomLeft).toEqual({ x: 5, y: 15 });
      expect(b.bottomRight).toEqual({ x: 15, y: 15 });
    });
  });
});
