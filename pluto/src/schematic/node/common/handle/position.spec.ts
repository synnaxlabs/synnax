// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { location } from "@synnaxlabs/x/location";
import { Position as RFPosition } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { Handle } from "@/schematic/node/common/handle";

const ORIENTATIONS: location.Outer[] = ["left", "right", "top", "bottom"];
const POSITIONS: location.Outer[] = ["left", "right", "top", "bottom"];
const RF_POSITIONS: RFPosition[] = [
  RFPosition.Left,
  RFPosition.Right,
  RFPosition.Top,
  RFPosition.Bottom,
];

describe("Handle.smart", () => {
  describe("identity orientation", () => {
    it("should pass through every position when orientation is left", () => {
      expect(Handle.smart("left", "left")).toBe(RFPosition.Left);
      expect(Handle.smart("right", "left")).toBe(RFPosition.Right);
      expect(Handle.smart("top", "left")).toBe(RFPosition.Top);
      expect(Handle.smart("bottom", "left")).toBe(RFPosition.Bottom);
    });
  });

  describe("180-degree rotations", () => {
    it("should flip horizontal positions when orientation is right", () => {
      expect(Handle.smart("left", "right")).toBe(RFPosition.Right);
      expect(Handle.smart("right", "right")).toBe(RFPosition.Left);
    });

    it("should flip vertical positions when orientation is right", () => {
      expect(Handle.smart("top", "right")).toBe(RFPosition.Bottom);
      expect(Handle.smart("bottom", "right")).toBe(RFPosition.Top);
    });
  });

  describe("90-degree rotations", () => {
    it("should rotate clockwise when orientation is top", () => {
      expect(Handle.smart("left", "top")).toBe(RFPosition.Bottom);
      expect(Handle.smart("right", "top")).toBe(RFPosition.Top);
      expect(Handle.smart("top", "top")).toBe(RFPosition.Left);
      expect(Handle.smart("bottom", "top")).toBe(RFPosition.Right);
    });

    it("should rotate counter-clockwise when orientation is bottom", () => {
      expect(Handle.smart("left", "bottom")).toBe(RFPosition.Top);
      expect(Handle.smart("right", "bottom")).toBe(RFPosition.Bottom);
      expect(Handle.smart("top", "bottom")).toBe(RFPosition.Right);
      expect(Handle.smart("bottom", "bottom")).toBe(RFPosition.Left);
    });
  });

  describe("structural invariants", () => {
    it("should always return a valid xyflow Position for every combination", () => {
      for (const orientation of ORIENTATIONS)
        for (const position of POSITIONS) {
          const out = Handle.smart(position, orientation);
          expect(RF_POSITIONS).toContain(out);
        }
    });

    it("should be a bijection on positions for any fixed orientation", () => {
      for (const orientation of ORIENTATIONS) {
        const outputs = POSITIONS.map((p) => Handle.smart(p, orientation));
        expect(new Set(outputs).size).toBe(POSITIONS.length);
      }
    });

    it("should be a bijection on orientations for any fixed position", () => {
      for (const position of POSITIONS) {
        const outputs = ORIENTATIONS.map((o) => Handle.smart(position, o));
        expect(new Set(outputs).size).toBe(ORIENTATIONS.length);
      }
    });
  });
});

describe("Handle.swap", () => {
  it("should flip every cardinal direction by default", () => {
    expect(Handle.swap(RFPosition.Left)).toBe(RFPosition.Right);
    expect(Handle.swap(RFPosition.Right)).toBe(RFPosition.Left);
    expect(Handle.swap(RFPosition.Top)).toBe(RFPosition.Bottom);
    expect(Handle.swap(RFPosition.Bottom)).toBe(RFPosition.Top);
  });

  it("should be involutive (swap twice yields the original)", () => {
    for (const p of RF_POSITIONS) expect(Handle.swap(Handle.swap(p))).toBe(p);
  });

  it("should be a no-op when bypass is true", () => {
    for (const p of RF_POSITIONS) expect(Handle.swap(p, true)).toBe(p);
  });

  it("should flip when bypass is explicitly false", () => {
    expect(Handle.swap(RFPosition.Left, false)).toBe(RFPosition.Right);
  });
});

describe("Handle.adjust", () => {
  describe("prevent flag", () => {
    it("should pass through coordinates unchanged regardless of orientation", () => {
      for (const orientation of ORIENTATIONS) {
        const out = Handle.adjust(30, 70, orientation, true);
        expect(out).toEqual({ top: 30, left: 70 });
      }
    });
  });

  describe("orientation transforms", () => {
    it("should pass through coordinates when orientation is left", () => {
      expect(Handle.adjust(30, 70, "left")).toEqual({ top: 30, left: 70 });
    });

    it("should reflect through (50, 50) when orientation is right", () => {
      expect(Handle.adjust(30, 70, "right")).toEqual({ top: 70, left: 30 });
    });

    it("should rotate so left edge becomes the top edge when orientation is top", () => {
      // top-orientation maps (top, left) -> (100 - left, top), placing a handle
      // that was "30% from top, 70% from left" onto "30% from top, 30% from left"
      expect(Handle.adjust(30, 70, "top")).toEqual({ top: 30, left: 30 });
    });

    it("should rotate so left edge becomes the bottom edge when orientation is bottom", () => {
      expect(Handle.adjust(30, 70, "bottom")).toEqual({ top: 70, left: 70 });
    });
  });

  describe("structural invariants", () => {
    it("should map (50, 50) to itself for every orientation", () => {
      for (const orientation of ORIENTATIONS)
        expect(Handle.adjust(50, 50, orientation)).toEqual({ top: 50, left: 50 });
    });

    it("should keep all outputs in the same numeric range as inputs", () => {
      const samples = [
        [0, 0],
        [0, 100],
        [100, 0],
        [100, 100],
        [25, 75],
        [10, 90],
      ] as const;
      for (const orientation of ORIENTATIONS)
        for (const [t, l] of samples) {
          const { top, left } = Handle.adjust(t, l, orientation);
          expect(top).toBeGreaterThanOrEqual(0);
          expect(top).toBeLessThanOrEqual(100);
          expect(left).toBeGreaterThanOrEqual(0);
          expect(left).toBeLessThanOrEqual(100);
        }
    });

    it("should be involutive on left and right orientations", () => {
      for (const orientation of ["left", "right"] as const) {
        const once = Handle.adjust(20, 80, orientation);
        const twice = Handle.adjust(once.top, once.left, orientation);
        expect(twice).toEqual({ top: 20, left: 80 });
      }
    });

    it("should compose top with top into the right orientation transform", () => {
      // Two 90-degree rotations equal a 180-degree reflection, which is the
      // right orientation transform.
      const once = Handle.adjust(20, 80, "top");
      const twice = Handle.adjust(once.top, once.left, "top");
      expect(twice).toEqual(Handle.adjust(20, 80, "right"));
    });

    it("should make top and bottom inverse operations", () => {
      // bottom(top(t, l)) = (t, l) since they are 90-degree rotations in
      // opposite directions.
      const samples: Array<[number, number]> = [
        [0, 0],
        [25, 75],
        [40, 90],
      ];
      for (const [t, l] of samples) {
        const rotated = Handle.adjust(t, l, "top");
        const back = Handle.adjust(rotated.top, rotated.left, "bottom");
        expect(back).toEqual({ top: t, left: l });
      }
    });
  });
});
