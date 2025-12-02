// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { direction } from "@/spatial/direction";

describe("Direction", () => {
  describe("construction", () => {
    type T = [string, direction.Crude];
    const TESTS: T[] = [
      ["from location", "top"],
      ["from literal", "y"],
    ];
    TESTS.forEach(([name, arg]) =>
      test(name, () => expect(direction.construct(arg)).toEqual("y")),
    );
  });

  describe("isX", () => {
    const TESTS: [direction.Crude, boolean][] = [
      ["x", true],
      ["y", false],
      ["left", true],
      ["right", true],
      ["top", false],
      ["bottom", false],
      ["center", false],
    ];
    TESTS.forEach(([arg, expected]) => {
      it(`should return ${expected} for ${arg}`, () => {
        expect(direction.isX(arg)).toBe(expected);
      });
    });
  });

  describe("isY", () => {
    const TESTS: [direction.Crude, boolean][] = [
      ["x", false],
      ["y", true],
      ["left", false],
      ["right", false],
      ["top", true],
      ["bottom", true],
      ["center", false],
    ];
    TESTS.forEach(([arg, expected]) => {
      it(`should return ${expected} for ${arg}`, () => {
        expect(direction.isY(arg)).toBe(expected);
      });
    });
  });

  describe("rotate", () => {
    it("should rotate a point 90 degrees clockwise around the origin", () => {
      const point = { x: 1, y: 0 };
      const center = { x: 0, y: 0 };
      const result = direction.rotate(point, center, "clockwise");
      expect(result.x).toBeCloseTo(0);
      expect(result.y).toBeCloseTo(1);
    });

    it("should rotate a point 90 degrees counterclockwise around the origin", () => {
      const point = { x: 1, y: 0 };
      const center = { x: 0, y: 0 };
      const result = direction.rotate(point, center, "counterclockwise");
      expect(result.x).toBeCloseTo(0);
      expect(result.y).toBeCloseTo(-1);
    });

    it("should rotate a point around a non-origin center", () => {
      const point = { x: 2, y: 1 };
      const center = { x: 1, y: 1 };
      const result = direction.rotate(point, center, "clockwise");
      expect(result.x).toBeCloseTo(1);
      expect(result.y).toBeCloseTo(2);
    });

    it("should return the same point when rotating around itself", () => {
      const point = { x: 5, y: 5 };
      const result = direction.rotate(point, point, "clockwise");
      expect(result.x).toBeCloseTo(5);
      expect(result.y).toBeCloseTo(5);
    });
  });
});
