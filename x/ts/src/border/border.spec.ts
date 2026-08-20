// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { border } from "@/border";

describe("border", () => {
  describe("construct", () => {
    describe("scalar number form", () => {
      it("should expand a scalar to identical XY values for every corner", () => {
        expect(border.construct(8)).toEqual({
          topLeft: { x: 8, y: 8 },
          topRight: { x: 8, y: 8 },
          bottomLeft: { x: 8, y: 8 },
          bottomRight: { x: 8, y: 8 },
        });
      });

      it("should preserve a scalar of zero across every corner", () => {
        const r = border.construct(0);
        expect(r.topLeft).toEqual({ x: 0, y: 0 });
        expect(r.topRight).toEqual({ x: 0, y: 0 });
        expect(r.bottomLeft).toEqual({ x: 0, y: 0 });
        expect(r.bottomRight).toEqual({ x: 0, y: 0 });
      });
    });

    describe("directional {x, y} form", () => {
      it("should apply the same XY to every corner", () => {
        const r = border.construct({ x: 25, y: 50 });
        expect(r.topLeft).toEqual({ x: 25, y: 50 });
        expect(r.topRight).toEqual({ x: 25, y: 50 });
        expect(r.bottomLeft).toEqual({ x: 25, y: 50 });
        expect(r.bottomRight).toEqual({ x: 25, y: 50 });
      });

      it("should not flip x and y", () => {
        const r = border.construct({ x: 1, y: 99 });
        expect(r.topLeft.x).toBe(1);
        expect(r.topLeft.y).toBe(99);
      });
    });

    describe("per-corner number form", () => {
      it("should expand each corner number into a symmetric XY pair", () => {
        expect(
          border.construct({
            topLeft: 1,
            topRight: 2,
            bottomLeft: 3,
            bottomRight: 4,
          }),
        ).toEqual({
          topLeft: { x: 1, y: 1 },
          topRight: { x: 2, y: 2 },
          bottomLeft: { x: 3, y: 3 },
          bottomRight: { x: 4, y: 4 },
        });
      });
    });

    describe("per-corner XY form", () => {
      it("should pass through per-corner XY values without modification", () => {
        const radius = {
          topLeft: { x: 1, y: 2 },
          topRight: { x: 3, y: 4 },
          bottomLeft: { x: 5, y: 6 },
          bottomRight: { x: 7, y: 8 },
        };
        expect(border.construct(radius)).toEqual(radius);
      });
    });
  });

  describe("crudeZ", () => {
    it("should accept every supported input form", () => {
      expect(() => border.crudeZ.parse(5)).not.toThrow();
      expect(() => border.crudeZ.parse({ x: 1, y: 2 })).not.toThrow();
      expect(() =>
        border.crudeZ.parse({
          topLeft: 1,
          topRight: 2,
          bottomLeft: 3,
          bottomRight: 4,
        }),
      ).not.toThrow();
      expect(() =>
        border.crudeZ.parse({
          topLeft: { x: 1, y: 1 },
          topRight: { x: 2, y: 2 },
          bottomLeft: { x: 3, y: 3 },
          bottomRight: { x: 4, y: 4 },
        }),
      ).not.toThrow();
    });

    it("should reject inputs missing required corners", () => {
      expect(() =>
        border.crudeZ.parse({ topLeft: 1, topRight: 2, bottomLeft: 3 }),
      ).toThrow();
    });

    it("should reject string inputs", () => {
      expect(() => border.crudeZ.parse("8px")).toThrow();
    });
  });

  describe("radiusZ", () => {
    it("should reject the shorthand forms crudeZ accepts", () => {
      expect(() => border.radiusZ.parse(5)).toThrow();
      expect(() =>
        border.radiusZ.parse({
          topLeft: 1,
          topRight: 2,
          bottomLeft: 3,
          bottomRight: 4,
        }),
      ).toThrow();
    });
  });
});
