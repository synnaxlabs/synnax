import { describe, test, expect } from "vitest";

import { Box } from "./box";

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
  test("translation", () => {
    const b1 = new Box({ x: 5, y: 0 }, { x: 15, y: 10 });
    const b2 = b1.translate({ x: 5, y: 5 });
    expect(b2.topLeft).toEqual({ x: 10, y: 5 });
    expect(b2.topRight).toEqual({ x: 20, y: 5 });
    expect(b2.bottomLeft).toEqual({ x: 10, y: 15 });
    expect(b2.bottomRight).toEqual({ x: 20, y: 15 });
    expect(b2.width).toEqual(10);
    expect(b2.height).toEqual(10);
  });
  test("resize", () => {
    const b1 = new Box({ x: 5, y: 0 }, { x: 15, y: 10 });
    const b2 = b1.resize({ x: 5, y: 5 });
    expect(b2.topLeft).toEqual({ x: 5, y: 0 });
    expect(b2.topRight).toEqual({ x: 10, y: 0 });
    expect(b2.bottomLeft).toEqual({ x: 5, y: 5 });
    expect(b2.bottomRight).toEqual({ x: 10, y: 5 });
    expect(b2.width).toEqual(5);
    expect(b2.height).toEqual(5);
  });
  test("scale", () => {
    const b1 = new Box({ x: 5, y: 0 }, { x: 15, y: 10 });
    const b2 = b1.scaleDims({ x: 2, y: 2 });
    expect(b2.topLeft).toEqual({ x: 5, y: 0 });
    expect(b2.topRight).toEqual({ x: 25, y: 0 });
    expect(b2.bottomLeft).toEqual({ x: 5, y: 20 });
    expect(b2.bottomRight).toEqual({ x: 25, y: 20 });
    expect(b2.width).toEqual(20);
    expect(b2.height).toEqual(20);
  });
  test("toDecimal", () => {
    const parent = new Box({ x: 10, y: 10 }, { x: 110, y: 110 });
    const child = new Box({ x: 15, y: 10 }, { x: 30, y: 25 });
    const decimal = child.toDecimal(parent);
    expect(decimal.width).toBeCloseTo(0.15);
    expect(decimal.height).toBeCloseTo(0.15);
    expect(decimal.top).toBeCloseTo(0);
    expect(decimal.left).toBeCloseTo(0.05);
  });
});
