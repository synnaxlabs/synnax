// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  dimensionsOnResize,
  inlineSizeOnResize,
  radiusOnResize,
  sideLengthOnResize,
} from "@/schematic/symbol/Symbols";

describe("onResize helpers", () => {
  describe("dimensionsOnResize", () => {
    it("should wrap dimensions in a dimensions key", () => {
      expect(dimensionsOnResize({ width: 400, height: 300 })).toEqual({
        dimensions: { width: 400, height: 300 },
      });
    });

    it("should handle zero dimensions", () => {
      expect(dimensionsOnResize({ width: 0, height: 0 })).toEqual({
        dimensions: { width: 0, height: 0 },
      });
    });

    it("should handle large dimensions", () => {
      expect(dimensionsOnResize({ width: 2000, height: 2000 })).toEqual({
        dimensions: { width: 2000, height: 2000 },
      });
    });

    it("should handle negative dimensions", () => {
      expect(dimensionsOnResize({ width: -10, height: -20 })).toEqual({
        dimensions: { width: -10, height: -20 },
      });
    });
  });

  describe("sideLengthOnResize", () => {
    it("should compute sideLength as half the width", () => {
      expect(sideLengthOnResize({ width: 100, height: 50 })).toEqual({
        sideLength: 50,
      });
    });

    it("should handle zero width", () => {
      expect(sideLengthOnResize({ width: 0, height: 100 })).toEqual({
        sideLength: 0,
      });
    });

    it("should handle odd width values", () => {
      expect(sideLengthOnResize({ width: 99, height: 50 })).toEqual({
        sideLength: 49.5,
      });
    });

    it("should handle negative width", () => {
      expect(sideLengthOnResize({ width: -100, height: 50 })).toEqual({
        sideLength: -50,
      });
    });
  });

  describe("radiusOnResize", () => {
    it("should compute radius as half the width", () => {
      expect(radiusOnResize({ width: 80, height: 80 })).toEqual({ radius: 40 });
    });

    it("should handle zero width", () => {
      expect(radiusOnResize({ width: 0, height: 0 })).toEqual({ radius: 0 });
    });

    it("should handle negative width", () => {
      expect(radiusOnResize({ width: -60, height: 80 })).toEqual({ radius: -30 });
    });
  });

  describe("inlineSizeOnResize", () => {
    it("should pass width through as inlineSize", () => {
      expect(inlineSizeOnResize({ width: 200, height: 100 })).toEqual({
        inlineSize: 200,
      });
    });

    it("should ignore height", () => {
      const a = inlineSizeOnResize({ width: 200, height: 50 });
      const b = inlineSizeOnResize({ width: 200, height: 300 });
      expect(a).toEqual(b);
    });

    it("should handle zero width", () => {
      expect(inlineSizeOnResize({ width: 0, height: 100 })).toEqual({
        inlineSize: 0,
      });
    });

    it("should handle negative width", () => {
      expect(inlineSizeOnResize({ width: -50, height: 100 })).toEqual({
        inlineSize: -50,
      });
    });
  });
});
