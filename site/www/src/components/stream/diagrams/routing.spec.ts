// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { manhattanMidpoint, manhattanPath } from "@/components/stream/diagrams/routing";

describe("routing", () => {
  describe("manhattanPath", () => {
    it("should draw a straight line between points at the same height", () => {
      expect(manhattanPath(0, 50, 100, 50)).toBe("M0,50 L100,50");
    });

    it("should treat a drop under one unit as the same height", () => {
      expect(manhattanPath(0, 50, 100, 50.5)).toBe("M0,50 L100,50.5");
    });

    it("should turn down at the horizontal midpoint with rounded corners", () => {
      expect(manhattanPath(0, 0, 100, 100)).toBe(
        "M0,0 L42,0 Q50,0 50,8 L50,92 Q50,100 58,100 L100,100",
      );
    });

    it("should turn up when the end is above the start", () => {
      expect(manhattanPath(0, 100, 100, 0)).toBe(
        "M0,100 L42,100 Q50,100 50,92 L50,8 Q50,0 58,0 L100,0",
      );
    });

    it("should take the corner radius it is given", () => {
      expect(manhattanPath(0, 0, 100, 100, 4)).toBe(
        "M0,0 L46,0 Q50,0 50,4 L50,96 Q50,100 54,100 L100,100",
      );
    });

    it("should shrink the corners to half of a short drop", () => {
      expect(manhattanPath(0, 0, 100, 6)).toBe(
        "M0,0 L47,0 Q50,0 50,3 L50,3 Q50,6 53,6 L100,6",
      );
    });

    it("should shrink the corners to half of the run to the midpoint", () => {
      expect(manhattanPath(0, 0, 12, 100)).toBe(
        "M0,0 L3,0 Q6,0 6,3 L6,97 Q6,100 9,100 L12,100",
      );
    });
  });

  describe("manhattanMidpoint", () => {
    it("should return the point halfway between the ends", () => {
      expect(manhattanMidpoint(0, 10, 100, 50)).toEqual({ x: 50, y: 30 });
    });
  });
});
