// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { box } from "@/spatial/box";
import { location } from "@/spatial/location";
import { type xy } from "@/spatial/xy";

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
    test("from first coordinate and raw params", () => {
      const b = box.construct({ x: 0, y: 0 }, 10, 10);
      expect(box.topLeft(b)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b)).toEqual({ x: 10, y: 0 });
      expect(box.bottomLeft(b)).toEqual({ x: 0, y: 10 });
      expect(box.bottomRight(b)).toEqual({ x: 10, y: 10 });
    });
    test("from first coordinate and raw params 2", () => {
      const b = box.construct({ x: 0, y: 0 }, undefined, 10, 10);
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
        location.CENTER_LEFT,
        location.CENTER_RIGHT,
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
  describe("constructWithAlternateRoot", () => {
    it("should construct a box with the given root", () => {
      const b = box.constructWithAlternateRoot(
        10,
        10,
        10,
        10,
        location.BOTTOM_RIGHT,
        location.TOP_LEFT,
      );
      expect(box.topLeft(b)).toEqual({ x: 0, y: 0 });
      const b2 = box.constructWithAlternateRoot(
        10,
        10,
        10,
        10,
        location.TOP_LEFT,
        location.TOP_LEFT,
      );
      expect(box.topLeft(b2)).toEqual({ x: 10, y: 10 });
      const b3 = box.constructWithAlternateRoot(
        10,
        10,
        10,
        10,
        location.BOTTOM_LEFT,
        location.TOP_LEFT,
      );
      expect(box.topLeft(b3)).toEqual({ x: 10, y: 0 });
    });
  });
  describe("resize", () => {
    it("should resize the x dimension of the box", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.resize(b, "x", 20);
      expect(box.height(b2)).toBe(10);
      expect(box.width(b2)).toBe(20);
      expect(box.topLeft(b2)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b2)).toEqual({ x: 20, y: 0 });
      expect(box.bottomLeft(b2)).toEqual({ x: 0, y: 10 });
      expect(box.bottomRight(b2)).toEqual({ x: 20, y: 10 });
    });
    it("should resize the y dimension of the box", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.resize(b, "y", 20);
      expect(box.height(b2)).toBe(20);
      expect(box.width(b2)).toBe(10);
      expect(box.topLeft(b2)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b2)).toEqual({ x: 10, y: 0 });
      expect(box.bottomLeft(b2)).toEqual({ x: 0, y: 20 });
      expect(box.bottomRight(b2)).toEqual({ x: 10, y: 20 });
    });
    it("should resize both dimensions of the box", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.resize(b, { width: 20, height: 20 });
      expect(box.height(b2)).toBe(20);
      expect(box.width(b2)).toBe(20);
      expect(box.topLeft(b2)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b2)).toEqual({ x: 20, y: 0 });
      expect(box.bottomLeft(b2)).toEqual({ x: 0, y: 20 });
      expect(box.bottomRight(b2)).toEqual({ x: 20, y: 20 });
    });
  });
  describe("translate", () => {
    it("should translate the box by the given coordinate", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.translate(b, { x: 10, y: 10 });
      expect(box.topLeft(b2)).toEqual({ x: 10, y: 10 });
      expect(box.topRight(b2)).toEqual({ x: 20, y: 10 });
      expect(box.bottomLeft(b2)).toEqual({ x: 10, y: 20 });
      expect(box.bottomRight(b2)).toEqual({ x: 20, y: 20 });
    });
    it("should translate the box by the given x amount", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.translate(b, "x", 10);
      expect(box.topLeft(b2)).toEqual({ x: 10, y: 0 });
      expect(box.topRight(b2)).toEqual({ x: 20, y: 0 });
      expect(box.bottomLeft(b2)).toEqual({ x: 10, y: 10 });
      expect(box.bottomRight(b2)).toEqual({ x: 20, y: 10 });
    });
    it("should translate the box by the given y amount", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.translate(b, "y", 10);
      expect(box.topLeft(b2)).toEqual({ x: 0, y: 10 });
      expect(box.topRight(b2)).toEqual({ x: 10, y: 10 });
      expect(box.bottomLeft(b2)).toEqual({ x: 0, y: 20 });
      expect(box.bottomRight(b2)).toEqual({ x: 10, y: 20 });
    });
  });
  describe("truncate", () => {
    it("should truncte the precision of the coordinates", () => {
      const b = box.construct(
        { x: 0.123456, y: 0.123456 },
        { x: 10.123456, y: 10.123456 },
      );
      const b2 = box.truncate(b, 2);
      expect(box.topLeft(b2)).toEqual({ x: 0.12, y: 0.12 });
      expect(box.topRight(b2)).toEqual({ x: 10.12, y: 0.12 });
      expect(box.bottomLeft(b2)).toEqual({ x: 0.12, y: 10.12 });
      expect(box.bottomRight(b2)).toEqual({ x: 10.12, y: 10.12 });
    });
  });
  describe("intersection", () => {
    it("should return the intersection of two boxes", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.construct(5, 5, 15, 15);
      const b3 = box.intersection(b, b2);
      expect(box.topLeft(b3)).toEqual({ x: 5, y: 5 });
      expect(box.topRight(b3)).toEqual({ x: 10, y: 5 });
      expect(box.bottomLeft(b3)).toEqual({ x: 5, y: 10 });
      expect(box.bottomRight(b3)).toEqual({ x: 10, y: 10 });
    });
    it("should return a zero box if there is no intersection", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.construct(15, 15, 20, 20);
      const b3 = box.intersection(b, b2);
      expect(box.topLeft(b3)).toEqual({ x: 0, y: 0 });
      expect(box.topRight(b3)).toEqual({ x: 0, y: 0 });
      expect(box.bottomLeft(b3)).toEqual({ x: 0, y: 0 });
      expect(box.bottomRight(b3)).toEqual({ x: 0, y: 0 });
    });
  });
  describe("area", () => {
    it("should return the area of the box", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.area(b)).toBe(100);
    });
  });
  describe("areaIsZero", () => {
    it("should return true if the area is zero", () => {
      const b = box.construct(0, 0, 0, 0);
      expect(box.areaIsZero(b)).toBe(true);
    });
    it("should return false if the area is not zero", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.areaIsZero(b)).toBe(false);
    });
  });
  describe("isBox", () => {
    it("should return true if the value is a box", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.isBox(b)).toBe(true);
    });
    it("should return false if the value is not a box", () => {
      expect(box.isBox({})).toBe(false);
    });
  });
  describe("yBounds", () => {
    it("should return the y bounds of the box", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.yBounds(b)).toEqual({ lower: 0, upper: 10 });
    });
  });
  describe("xBounds", () => {
    it("should return the x bounds of the box", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.xBounds(b)).toEqual({ lower: 0, upper: 10 });
    });
  });
  describe("contains", () => {
    describe("inclusive of border", () => {
      it("should return true if the box completely contains the other box", () => {
        const b = box.construct(0, 0, 20, 20);
        const b2 = box.construct(5, 5, 15, 15);
        expect(box.contains(b, b2)).toBe(true);
      });
      it("should return true if the box completely contains the other box", () => {
        const b = box.construct(0, 0, 20, 20);
        const b2 = box.construct(5, 5, 14, 14);
        expect(box.contains(b, b2)).toBe(true);
      });
      it("should return false if the box does not completely contain the other box", () => {
        const b = box.construct(0, 0, 10, 10);
        const b2 = box.construct(5, 5, 15, 15);
        expect(box.contains(b, b2)).toBe(false);
      });
      it("should return true if the two boxes are equal", () => {
        const b = box.construct(0, 0, 10, 10);
        expect(box.contains(b, b)).toBe(true);
      });
      it("should return true if the box contains the point", () => {
        const b = box.construct(0, 0, 10, 10);
        const p = { x: 5, y: 5 };
        expect(box.contains(b, p)).toBe(true);
      });
      it("should return false if the box does not contain the point", () => {
        const b = box.construct(0, 0, 10, 10);
        const p = { x: 15, y: 15 };
        expect(box.contains(b, p)).toBe(false);
      });
      it("should return true if the point is on the border", () => {
        const b = box.construct(0, 0, 10, 10);
        const p = { x: 10, y: 10 };
        expect(box.contains(b, p)).toBe(true);
      });
    });
    describe("exclusive of border", () => {
      it("should return false if the box completely contains the other box", () => {
        const b = box.construct(0, 0, 20, 20);
        const b2 = box.construct(5, 5, 15, 15);
        expect(box.contains(b, b2, false)).toBe(false);
      });
      it("should return true if the box completely contains the other box", () => {
        const b = box.construct(0, 0, 20, 20);
        const b2 = box.construct(5, 5, 14, 14);
        expect(box.contains(b, b2, false)).toBe(true);
      });
      it("should return false if the box does not completely contain the other box", () => {
        const b = box.construct(0, 0, 10, 10);
        const b2 = box.construct(5, 5, 15, 15);
        expect(box.contains(b, b2, false)).toBe(false);
      });
      it("should return false if the two boxes are equal", () => {
        const b = box.construct(0, 0, 10, 10);
        expect(box.contains(b, b, false)).toBe(false);
      });
      it("should return false if the box contains the point", () => {
        const b = box.construct(0, 0, 10, 10);
        const p = { x: 5, y: 5 };
        expect(box.contains(b, p, false)).toBe(true);
      });
      it("should return false if the box does not contain the point", () => {
        const b = box.construct(0, 0, 10, 10);
        const p = { x: 15, y: 15 };
        expect(box.contains(b, p, false)).toBe(false);
      });
      it("should return false if the point is on the border", () => {
        const b = box.construct(0, 0, 10, 10);
        const p = { x: 10, y: 10 };
        expect(box.contains(b, p, false)).toBe(false);
      });
    });
  });
  describe("css", () => {
    it("should return the box as an object of css properties", () => {
      const b = box.construct(0, 0, 10, 10);
      expect(box.css(b)).toEqual({
        left: 0,
        top: 0,
        width: 10,
        height: 10,
      });
    });
  });
  describe("copy", () => {
    it("should return a copy of the box", () => {
      const b = box.construct(0, 0, 10, 10);
      const b2 = box.copy(b);
      expect(box.equals(b, b2)).toBe(true);
    });
  });
  describe("edgePoints", () => {
    interface Spec {
      box: box.Box;
      loc: location.Location;
      expected: [xy.XY, xy.XY];
    }

    const SPECS: Spec[] = [
      {
        box: box.construct(0, 0, 10, 10),
        loc: "top",
        expected: [
          { x: 0, y: 0 },
          { x: 10, y: 0 },
        ],
      },
      {
        box: box.construct(0, 0, 10, 10),
        loc: "bottom",
        expected: [
          { x: 0, y: 10 },
          { x: 10, y: 10 },
        ],
      },
      {
        box: box.construct(0, 0, 10, 10),
        loc: "left",
        expected: [
          { x: 0, y: 0 },
          { x: 0, y: 10 },
        ],
      },
      {
        box: box.construct(0, 0, 10, 10),
        loc: "right",
        expected: [
          { x: 10, y: 0 },
          { x: 10, y: 10 },
        ],
      },
    ];
    SPECS.forEach(({ box: b, loc, expected }) => {
      test(`edgePoints-${loc}`, () => {
        expect(box.edgePoints(b, loc)).toEqual(expected);
      });
    });
  });
});
