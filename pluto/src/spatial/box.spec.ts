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
});
