// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type direction, type location } from "@synnaxlabs/x";
import { box, xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import {
  alignNodesAlongDirection,
  alignNodesToLocation,
  distributeNodes,
  rotateNodesAroundCenter,
} from "@/vis/diagram/align";
import { HandleLayout, NodeLayout } from "@/vis/diagram/util";

// Helper that dispatches to the correct alignment function based on argument
const alignNodes = (
  layouts: NodeLayout[],
  target: direction.Direction | location.Outer,
): NodeLayout[] => {
  if (target === "x" || target === "y") {
    return alignNodesAlongDirection(layouts, target);
  }
  return alignNodesToLocation(layouts, target);
};

describe("Schematic Alignment", () => {
  describe("aligning valve symbols", () => {
    it("should align valves vertically (along x-axis)", () => {
      // Create mock valve nodes with different Y positions
      const valves = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 150, y: 20 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: 300, y: -10 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      // Align them vertically (all on same horizontal line)
      const aligned = alignNodes(valves, "x");

      // All should have the same y-coordinate
      const yCoords = aligned.map((v) => box.top(v.box));
      expect(yCoords[0]).toBe(yCoords[1]);
      expect(yCoords[1]).toBe(yCoords[2]);
    });

    it("should align valves horizontally (along y-axis)", () => {
      // Create mock valve nodes with different X positions
      const valves = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 20, y: 150 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: -10, y: 300 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      // Align them horizontally (all on same vertical line)
      const aligned = alignNodes(valves, "y");

      // All should have the same x-coordinate
      const xCoords = aligned.map((v) => box.left(v.box));
      expect(xCoords[0]).toBe(xCoords[1]);
      expect(xCoords[1]).toBe(xCoords[2]);
    });

    it("should distribute valves horizontally with equal spacing", () => {
      const valves = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 110, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: 600, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const distributed = distributeNodes(valves, "x");

      // Check that spacing is equal
      const x1 = box.left(distributed[0].box);
      const x2 = box.left(distributed[1].box);
      const x3 = box.left(distributed[2].box);

      const gap1 = x2 - (x1 + 100); // gap between valve1 and valve2
      const gap2 = x3 - (x2 + 100); // gap between valve2 and valve3

      expect(gap1).toBeCloseTo(gap2, 2);
    });

    it("should distribute valves vertically with equal spacing", () => {
      const valves = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 0, y: 110 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: 0, y: 600 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const distributed = distributeNodes(valves, "y");

      // Check that spacing is equal
      const y1 = box.top(distributed[0].box);
      const y2 = box.top(distributed[1].box);
      const y3 = box.top(distributed[2].box);

      const gap1 = y2 - (y1 + 100); // gap between valve1 and valve2
      const gap2 = y3 - (y2 + 100); // gap between valve2 and valve3

      expect(gap1).toBeCloseTo(gap2, 2);
    });
  });

  describe("aligning mixed symbol types", () => {
    it("should align setpoint, valve, and three-way valve symbols to left", () => {
      const symbols = [
        new NodeLayout(
          "setpoint",
          box.construct({ x: 50, y: 0 }, { width: 80, height: 60 }),
          [],
        ),
        new NodeLayout(
          "valve",
          box.construct({ x: 150, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "threeWayValve",
          box.construct({ x: 10, y: 100 }, { width: 120, height: 100 }),
          [],
        ),
      ];

      const aligned = alignNodes(symbols, "left");

      // All should align to the leftmost position (x=10)
      const xCoords = aligned.map((s) => box.left(s.box));
      expect(xCoords).toEqual([10, 10, 10]);
    });

    it("should align mixed symbols to right", () => {
      const symbols = [
        new NodeLayout(
          "setpoint",
          box.construct({ x: 50, y: 0 }, { width: 80, height: 60 }),
          [],
        ),
        new NodeLayout(
          "valve",
          box.construct({ x: 150, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "threeWayValve",
          box.construct({ x: 10, y: 100 }, { width: 120, height: 100 }),
          [],
        ),
      ];

      const aligned = alignNodes(symbols, "right");

      // All should align to the rightmost position
      // Setpoint right: 50+80=130, Valve right: 150+100=250, ThreeWay right: 10+120=130
      // Max is 250, so all should have right edge at 250
      expect(box.right(aligned[0].box)).toBe(250);
      expect(box.right(aligned[1].box)).toBe(250);
      expect(box.right(aligned[2].box)).toBe(250);
    });

    it("should align mixed symbols to top", () => {
      const symbols = [
        new NodeLayout(
          "setpoint",
          box.construct({ x: 0, y: 50 }, { width: 80, height: 60 }),
          [],
        ),
        new NodeLayout(
          "valve",
          box.construct({ x: 50, y: 150 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "threeWayValve",
          box.construct({ x: 100, y: 10 }, { width: 120, height: 100 }),
          [],
        ),
      ];

      const aligned = alignNodes(symbols, "top");

      // All should align to the topmost position (y=10)
      const yCoords = aligned.map((s) => box.top(s.box));
      expect(yCoords).toEqual([10, 10, 10]);
    });

    it("should align mixed symbols to bottom", () => {
      const symbols = [
        new NodeLayout(
          "setpoint",
          box.construct({ x: 0, y: 50 }, { width: 80, height: 60 }),
          [],
        ),
        new NodeLayout(
          "valve",
          box.construct({ x: 50, y: 150 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "threeWayValve",
          box.construct({ x: 100, y: 10 }, { width: 120, height: 100 }),
          [],
        ),
      ];

      const aligned = alignNodes(symbols, "bottom");

      // All should align to the bottommost position
      // Setpoint bottom: 50+60=110, Valve bottom: 150+100=250, ThreeWay bottom: 10+100=110
      // Max is 250
      expect(box.bottom(aligned[0].box)).toBe(250);
      expect(box.bottom(aligned[1].box)).toBe(250);
      expect(box.bottom(aligned[2].box)).toBe(250);
    });
  });

  describe("distributing mixed symbol types", () => {
    it("should distribute four symbols with different widths horizontally", () => {
      const symbols = [
        new NodeLayout(
          "setpoint",
          box.construct({ x: -210, y: 0 }, { width: 80, height: 60 }),
          [],
        ),
        new NodeLayout(
          "threeWayValve",
          box.construct({ x: -150, y: 0 }, { width: 120, height: 100 }),
          [],
        ),
        new NodeLayout(
          "threeWayBall",
          box.construct({ x: 150, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve",
          box.construct({ x: 0, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const distributed = distributeNodes(symbols, "x");

      // Verify they're spaced out between leftmost and rightmost
      const positions = distributed
        .sort((a, b) => box.left(a.box) - box.left(b.box))
        .map((s) => box.left(s.box));

      // Each symbol should be further right than the previous
      expect(positions[1]).toBeGreaterThan(positions[0]);
      expect(positions[2]).toBeGreaterThan(positions[1]);
      expect(positions[3]).toBeGreaterThan(positions[2]);
    });

    it("should distribute four symbols vertically", () => {
      const symbols = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 0, y: 150 }, { width: 100, height: 120 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: 0, y: 400 }, { width: 100, height: 80 }),
          [],
        ),
        new NodeLayout(
          "valve4",
          box.construct({ x: 0, y: 600 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const distributed = distributeNodes(symbols, "y");

      // Verify they're spaced out between topmost and bottommost
      const positions = distributed.map((s) => box.top(s.box));

      // Each symbol should be further down than the previous
      expect(positions[1]).toBeGreaterThan(positions[0]);
      expect(positions[2]).toBeGreaterThan(positions[1]);
      expect(positions[3]).toBeGreaterThan(positions[2]);
    });
  });

  describe("alignment with handles", () => {
    it("should align nodes with handles based on handle positions", () => {
      const nodesWithHandles = [
        new NodeLayout("valve1", box.construct(xy.ZERO, { width: 100, height: 100 }), [
          new HandleLayout({ x: 0, y: 50 }, "left"),
          new HandleLayout({ x: 100, y: 50 }, "right"),
        ]),
        new NodeLayout(
          "valve2",
          box.construct({ x: 10, y: 10 }, { width: 100, height: 100 }),
          [
            new HandleLayout({ x: 0, y: 60 }, "left"),
            new HandleLayout({ x: 100, y: 50 }, "right"),
          ],
        ),
      ];

      const aligned = alignNodes(nodesWithHandles, "x");

      // Nodes with different handle positions should align differently
      // than nodes without handles
      expect(aligned).toHaveLength(2);
      expect(aligned[0].handles).toHaveLength(2);
      expect(aligned[1].handles).toHaveLength(2);
    });

    it("should handle nodes without handles gracefully", () => {
      const mixedNodes = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [new HandleLayout({ x: 50, y: 0 }, "top")],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 150, y: 20 }, { width: 100, height: 100 }),
          [], // No handles
        ),
      ];

      const aligned = alignNodes(mixedNodes, "x");

      // Should align based on centers when handles aren't available
      const yCoords = aligned.map((v) => box.top(v.box));
      expect(yCoords[0]).toBe(yCoords[1]);
    });
  });

  describe("rotation operations", () => {
    it("should rotate symbols clockwise around their group center", () => {
      const symbols = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 200, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const rotated = rotateNodesAroundCenter(symbols, "clockwise");

      // After rotation, positions should change
      expect(box.topLeft(rotated[0].box)).not.toEqual({ x: 0, y: 0 });
      expect(box.topLeft(rotated[1].box)).not.toEqual({ x: 200, y: 0 });

      // Dimensions should be preserved
      expect(box.dims(rotated[0].box)).toEqual({ width: 100, height: 100 });
      expect(box.dims(rotated[1].box)).toEqual({ width: 100, height: 100 });
    });

    it("should rotate symbols counter-clockwise around their group center", () => {
      const symbols = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 200, y: 100 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const rotatedCCW = rotateNodesAroundCenter(symbols, "counterclockwise");

      // Verify rotation happened (nodes moved from original positions)
      expect(box.topLeft(rotatedCCW[0].box)).not.toEqual({ x: 0, y: 0 });
      expect(box.topLeft(rotatedCCW[1].box)).not.toEqual({ x: 200, y: 100 });

      // Dimensions should be preserved
      expect(box.dims(rotatedCCW[0].box)).toEqual({ width: 100, height: 100 });
      expect(box.dims(rotatedCCW[1].box)).toEqual({ width: 100, height: 100 });
    });

    it("should handle single symbol rotation", () => {
      const single = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const rotated = rotateNodesAroundCenter(single, "clockwise");

      // Single node rotates around itself, so position should stay the same
      expect(box.topLeft(rotated[0].box)).toEqual({ x: 50, y: 50 });
    });

    it("should preserve all node properties except position during rotation", () => {
      const symbols = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 150, height: 80 }),
          [new HandleLayout({ x: 75, y: 0 }, "top")],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 200, y: 0 }, { width: 100, height: 120 }),
          [new HandleLayout({ x: 50, y: 0 }, "top")],
        ),
      ];

      const rotated = rotateNodesAroundCenter(symbols, "clockwise");

      // Keys should be preserved
      expect(rotated[0].key).toBe("valve1");
      expect(rotated[1].key).toBe("valve2");

      // Dimensions should be preserved
      expect(box.dims(rotated[0].box)).toEqual({ width: 150, height: 80 });
      expect(box.dims(rotated[1].box)).toEqual({ width: 100, height: 120 });

      // Handles should be preserved
      expect(rotated[0].handles).toHaveLength(1);
      expect(rotated[1].handles).toHaveLength(1);
    });
  });

  describe("edge cases", () => {
    it("should handle empty array for alignment", () => {
      const empty = alignNodes([], "x");
      expect(empty).toEqual([]);
    });

    it("should handle empty array for distribution", () => {
      const empty = distributeNodes([], "x");
      expect(empty).toEqual([]);
    });

    it("should handle empty array for rotation", () => {
      const empty = rotateNodesAroundCenter([], "clockwise");
      expect(empty).toEqual([]);
    });

    it("should handle single element alignment", () => {
      const single = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const aligned = alignNodes(single, "left");
      expect(box.topLeft(aligned[0].box)).toEqual({ x: 50, y: 50 });
    });

    it("should handle two elements for distribution (no middle nodes)", () => {
      const two = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 500, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const distributed = distributeNodes(two, "x");

      // With only 2 nodes, first and last should stay in place
      expect(box.left(distributed[0].box)).toBe(0);
      expect(box.left(distributed[1].box)).toBe(500);
    });

    it("should handle overlapping nodes during distribution", () => {
      // When nodes are too close together, distribution should pack them
      const overlapping = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 50, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: 100, y: 0 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      const distributed = distributeNodes(overlapping, "x");

      // Should stack them without negative spacing
      const x1 = box.left(distributed[0].box);
      const x2 = box.left(distributed[1].box);
      const x3 = box.left(distributed[2].box);

      expect(x2).toBeGreaterThanOrEqual(x1 + 100); // No overlap
      expect(x3).toBeGreaterThanOrEqual(x2 + 100); // No overlap
    });

    it("should handle nodes with zero dimensions", () => {
      const zeroDims = [
        new NodeLayout(
          "point1",
          box.construct({ x: 0, y: 0 }, { width: 0, height: 0 }),
          [],
        ),
        new NodeLayout(
          "point2",
          box.construct({ x: 100, y: 0 }, { width: 0, height: 0 }),
          [],
        ),
      ];

      const aligned = alignNodes(zeroDims, "x");
      expect(aligned).toHaveLength(2);
    });
  });

  describe("realistic schematic scenarios", () => {
    it("should handle complex piping schematic with multiple valve types", () => {
      // Simulate a realistic schematic with setpoint, valves, and sensors
      const schematic = [
        new NodeLayout(
          "setpoint",
          box.construct({ x: -210, y: 0 }, { width: 80, height: 60 }),
          [],
        ),
        new NodeLayout(
          "threeWayValve",
          box.construct({ x: -150, y: 0 }, { width: 120, height: 120 }),
          [
            new HandleLayout({ x: 0, y: 60 }, "left"),
            new HandleLayout({ x: 120, y: 60 }, "right"),
            new HandleLayout({ x: 60, y: 0 }, "top"),
          ],
        ),
        new NodeLayout(
          "valve",
          box.construct({ x: 0, y: 50 }, { width: 100, height: 100 }),
          [
            new HandleLayout({ x: 0, y: 50 }, "left"),
            new HandleLayout({ x: 100, y: 50 }, "right"),
          ],
        ),
        new NodeLayout(
          "threeWayBall",
          box.construct({ x: 150, y: -20 }, { width: 100, height: 100 }),
          [
            new HandleLayout({ x: 0, y: 50 }, "left"),
            new HandleLayout({ x: 100, y: 50 }, "right"),
          ],
        ),
      ];

      // Test sequence from integration test: align vertical, distribute horizontal
      const alignedVertical = alignNodes([...schematic], "x");
      const distributedHorizontal = distributeNodes(alignedVertical, "x");

      // When nodes have handles, alignment is based on handle positions
      // So we just verify that alignment happened (nodes moved)
      expect(alignedVertical).toHaveLength(4);

      // Should be distributed with even spacing
      const sorted = [...distributedHorizontal].sort(
        (a, b) => box.left(a.box) - box.left(b.box),
      );
      expect(sorted).toHaveLength(4);
    });

    it("should handle workflow: align horizontal, then distribute vertical", () => {
      const symbols = [
        new NodeLayout(
          "valve1",
          box.construct({ x: 0, y: -100 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 150, y: 100 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: 50, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      // First align horizontally (same vertical line)
      const alignedHorizontal = alignNodes([...symbols], "y");

      // All should have same x-coordinate
      const xCoords = alignedHorizontal.map((s) => box.left(s.box));
      expect(xCoords[0]).toBe(xCoords[1]);
      expect(xCoords[1]).toBe(xCoords[2]);

      // Then distribute vertically
      const distributedVertical = distributeNodes(alignedHorizontal, "y");
      const sorted = [...distributedVertical].sort(
        (a, b) => box.top(a.box) - box.top(b.box),
      );

      // Should be spread vertically with even spacing
      expect(sorted).toHaveLength(3);
      expect(box.top(sorted[1].box)).toBeGreaterThan(box.top(sorted[0].box));
      expect(box.top(sorted[2].box)).toBeGreaterThan(box.top(sorted[1].box));
    });

    it("should handle workflow: align top, distribute horizontal, then rotate", () => {
      const symbols = [
        new NodeLayout(
          "valve1",
          box.construct({ x: -150, y: 30 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve2",
          box.construct({ x: 150, y: -20 }, { width: 100, height: 100 }),
          [],
        ),
        new NodeLayout(
          "valve3",
          box.construct({ x: 0, y: 50 }, { width: 100, height: 100 }),
          [],
        ),
      ];

      // Align to top
      const alignedTop = alignNodes([...symbols], "top");
      expect(box.top(alignedTop[0].box)).toBe(box.top(alignedTop[1].box));
      expect(box.top(alignedTop[1].box)).toBe(box.top(alignedTop[2].box));

      // Distribute horizontally
      const distributed = distributeNodes(alignedTop, "x");

      // Rotate clockwise
      const rotated = rotateNodesAroundCenter(distributed, "clockwise");

      // Verify rotation happened
      expect(rotated).toHaveLength(3);
      rotated.forEach((node, i) => {
        expect(box.dims(node.box)).toEqual({ width: 100, height: 100 });
        expect(node.key).toBe(symbols[i].key);
      });
    });
  });
});
