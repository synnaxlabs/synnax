// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, test, expect, it } from "vitest";

import * as box from "@/spatial/box";
import * as location from "@/spatial/location";
import type * as xy from "@/spatial/xy";

describe("Box", () => {
  describe("construction", () => {
    test("from dom rect", () => {
      const b = box.construct({ left: 0, top: 0, right: 10, bottom: 10 });
      expect(box.topLeft(b)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b)).toEqual({ x: 10, y: 0 });
      expect(box.bottomLeft(b)).toEqual({ x: 0, y: 10 });
      expect(box.bottomRight(b)).toEqual({ x: 10, y: 10 });
    });
    test("from two points", () => {
      const b = box.construct({ x: 0, y: 0 }, { x: 10, y: 10 });
      expect(box.topLeft(b)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b)).toEqual({ x: 10, y: 0 });
      expect(box.bottomLeft(b)).toEqual({ x: 0, y: 10 });
      expect(box.bottomRight(b)).toEqual({ x: 10, y: 10 });
    });
    test("from point and dimensions", () => {
      const b = box.construct({ x: 0, y: 0 }, { width: 10, height: 10 });
      expect(box.topLeft(b)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b)).toEqual({ x: 10, y: 0 });
      expect(box.bottomLeft(b)).toEqual({ x: 0, y: 10 });
      expect(box.bottomRight(b)).toEqual({ x: 10, y: 10 });
    });
    test("from point and width and height", () => {
      const b = box.construct({ x: 0, y: 0 }, 10, 10);
      expect(box.topLeft(b)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b)).toEqual({ x: 10, y: 0 });
      expect(box.bottomLeft(b)).toEqual({ x: 0, y: 10 });
      expect(box.bottomRight(b)).toEqual({ x: 10, y: 10 });
    });
    test("from raw params", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.topLeft(b)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b)).toEqual({ x: 10, y: 0 });
      expect(box.bottomLeft(b)).toEqual({ x: 0, y: 10 });
      expect(box.bottomRight(b)).toEqual({ x: 10, y: 10 });
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
        expect(() => box.box.parse(value)).not.toThrow();
      });
    });
  });
  describe("properties", () => {
    const b = box.construct(20, 30, 40, 50);
    describe("loc", () => {
      const v: location.Location[] = ["left", "right", "top", "bottom"];
      const expected: number[] = [20, 60, 30, 80];
      v.forEach((v, i) => {
        test(`loc-${v}`, () => expect(box.loc(b, v)).toEqual(expected[i]));
      });
    });
    describe("xyLoc", () => {
      const v: location.XY[] = [
        location.BOTTOM_CENTER,
        location.LEFT_CENTER,
        location.RIGHT_CENTER,
        location.TOP_CENTER,
        location.BOTTOM_LEFT,
        location.BOTTOM_RIGHT,
        location.TOP_LEFT,
        location.TOP_RIGHT,
      ];
      const expected: xy.Crude[] = [
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
        test(`xyLoc-${location.xyToString(v)}`, () =>
          expect(box.xyLoc(b, v)).toEqual(expected[i]));
      });
    });
  });
  describe("equality", () => {
    it("should be equal to itself", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.equals(b, b)).toBe(true);
    });
    it("should be equal to a box with the same values", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.construct(0, 0, 10, 10);
      expect(box.equals(b, b2)).toBe(true);
    });
  });
  describe("positionInCenterOf", () => {
    it("should position the box in the center of the other box", () => {
      let b = box.construct(0, 0, 10, 10);
      const b2 = box.construct(0, 0, 20, 20);
      b = box.positionInCenter(b, b2);
      expect(box.topLeft(b)).toEqual({ x: 5, y: 5 });
      expect(box.topRight(b)).toEqual({ x: 15, y: 5 });
      expect(box.bottomLeft(b)).toEqual({ x: 5, y: 15 });
      expect(box.bottomRight(b)).toEqual({ x: 15, y: 15 });
    });
  });
});
