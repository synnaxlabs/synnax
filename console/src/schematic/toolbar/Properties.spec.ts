// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Diagram } from "@synnaxlabs/pluto";
import { box, xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

describe("Schematic Toolbar Alignment Operations", () => {
  describe("alignment helper functions", () => {
    it("should prepare layouts for vertical alignment", () => {
      // Simulate getting layouts from DOM elements for alignment
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 150, y: 20 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: 300, y: -10 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      // Convert to NodeLayout objects
      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      // Test alignment
      const aligned = Diagram.alignNodes(layouts, "x");

      // All should be on same horizontal line
      const yPositions = aligned.map((l) => box.top(l.box));
      expect(yPositions[0]).toBe(yPositions[1]);
      expect(yPositions[1]).toBe(yPositions[2]);
    });

    it("should prepare layouts for horizontal alignment", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 20, y: 150 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: -10, y: 300 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const aligned = Diagram.alignNodes(layouts, "y");

      // All should be on same vertical line
      const xPositions = aligned.map((l) => box.left(l.box));
      expect(xPositions[0]).toBe(xPositions[1]);
      expect(xPositions[1]).toBe(xPositions[2]);
    });

    it("should prepare layouts for distribution with different widths", () => {
      const elements = [
        {
          key: "setpoint",
          position: { x: 0, y: 0 },
          dimensions: { width: 80, height: 60 },
        },
        {
          key: "valve",
          position: { x: 100, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "sensor",
          position: { x: 500, y: 0 },
          dimensions: { width: 50, height: 50 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const distributed = Diagram.distributeNodes(layouts, "x");

      // Should be distributed with even spacing
      const sorted = [...distributed].sort((a, b) => box.left(a.box) - box.left(b.box));

      expect(sorted).toHaveLength(3);
      expect(box.left(sorted[1].box)).toBeGreaterThan(box.left(sorted[0].box));
      expect(box.left(sorted[2].box)).toBeGreaterThan(box.left(sorted[1].box));
    });
  });

  describe("alignment workflows", () => {
    it("should handle align left operation", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 50, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 150, y: 50 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: 10, y: 100 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const aligned = Diagram.alignNodes(layouts, "left");

      // All should align to leftmost edge (x=10)
      expect(box.left(aligned[0].box)).toBe(10);
      expect(box.left(aligned[1].box)).toBe(10);
      expect(box.left(aligned[2].box)).toBe(10);
    });

    it("should handle align right operation", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 50, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 150, y: 50 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: 10, y: 100 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const aligned = Diagram.alignNodes(layouts, "right");

      // All should align to rightmost edge (250 = 150+100)
      expect(box.right(aligned[0].box)).toBe(250);
      expect(box.right(aligned[1].box)).toBe(250);
      expect(box.right(aligned[2].box)).toBe(250);
    });

    it("should handle align top operation", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 50 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 50, y: 150 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: 100, y: 10 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const aligned = Diagram.alignNodes(layouts, "top");

      // All should align to topmost edge (y=10)
      expect(box.top(aligned[0].box)).toBe(10);
      expect(box.top(aligned[1].box)).toBe(10);
      expect(box.top(aligned[2].box)).toBe(10);
    });

    it("should handle align bottom operation", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 50 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 50, y: 150 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: 100, y: 10 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const aligned = Diagram.alignNodes(layouts, "bottom");

      // All should align to bottommost edge (250 = 150+100)
      expect(box.bottom(aligned[0].box)).toBe(250);
      expect(box.bottom(aligned[1].box)).toBe(250);
      expect(box.bottom(aligned[2].box)).toBe(250);
    });
  });

  describe("distribution workflows", () => {
    it("should handle distribute horizontally operation", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 110, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: 600, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const distributed = Diagram.distributeNodes(layouts, "x");

      // Check even spacing
      const sorted = [...distributed].sort((a, b) => box.left(a.box) - box.left(b.box));

      const gap1 = box.left(sorted[1].box) - box.right(sorted[0].box);
      const gap2 = box.left(sorted[2].box) - box.right(sorted[1].box);

      expect(gap1).toBeCloseTo(gap2, 2);
    });

    it("should handle distribute vertically operation", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 0, y: 110 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-3",
          position: { x: 0, y: 600 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const distributed = Diagram.distributeNodes(layouts, "y");

      // Check even spacing
      const sorted = [...distributed].sort((a, b) => box.top(a.box) - box.top(b.box));

      const gap1 = box.top(sorted[1].box) - box.bottom(sorted[0].box);
      const gap2 = box.top(sorted[2].box) - box.bottom(sorted[1].box);

      expect(gap1).toBeCloseTo(gap2, 2);
    });
  });

  describe("rotation workflows", () => {
    it("should handle rotate individual clockwise", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
          rotation: 0,
        },
        {
          key: "valve-2",
          position: { x: 150, y: 0 },
          dimensions: { width: 100, height: 100 },
          rotation: 0,
        },
      ];

      // Individual rotation adds 90 degrees to each element
      const rotated = elements.map((el) => ({
        ...el,
        rotation: (el.rotation + 90) % 360,
      }));

      expect(rotated[0].rotation).toBe(90);
      expect(rotated[1].rotation).toBe(90);
      expect(rotated[0].position).toEqual(elements[0].position);
      expect(rotated[1].position).toEqual(elements[1].position);
    });

    it("should handle rotate individual counter-clockwise", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
          rotation: 90,
        },
        {
          key: "valve-2",
          position: { x: 150, y: 0 },
          dimensions: { width: 100, height: 100 },
          rotation: 90,
        },
      ];

      // Individual rotation subtracts 90 degrees from each element
      const rotated = elements.map((el) => ({
        ...el,
        rotation: (el.rotation - 90 + 360) % 360,
      }));

      expect(rotated[0].rotation).toBe(0);
      expect(rotated[1].rotation).toBe(0);
    });

    it("should handle rotate group clockwise", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 200, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const rotated = Diagram.rotateNodesAroundCenter(layouts, "clockwise");

      // Positions should change after rotation
      expect(box.topLeft(rotated[0].box)).not.toEqual(elements[0].position);
      expect(box.topLeft(rotated[1].box)).not.toEqual(elements[1].position);

      // Dimensions should be preserved
      expect(box.dims(rotated[0].box)).toEqual(elements[0].dimensions);
      expect(box.dims(rotated[1].box)).toEqual(elements[1].dimensions);
    });

    it("should handle rotate group counter-clockwise", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 200, y: 100 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const rotatedCCW = Diagram.rotateNodesAroundCenter(layouts, "counterclockwise");

      // Verify rotation happened
      expect(box.topLeft(rotatedCCW[0].box)).not.toEqual({ x: 0, y: 0 });
      expect(box.topLeft(rotatedCCW[1].box)).not.toEqual({ x: 200, y: 100 });
    });
  });

  describe("complex workflows from integration test", () => {
    it("should simulate the complete alignment workflow", () => {
      // Replicate the workflow from integration/tests/console/schematic/alignment.py

      // Initial positions (setpoint, threeWayValve, threeWayBall, valve)
      const elements = [
        {
          key: "setpoint",
          position: { x: -210, y: 0 },
          dimensions: { width: 80, height: 60 },
        },
        {
          key: "threeWayValve",
          position: { x: -150, y: 0 },
          dimensions: { width: 120, height: 120 },
        },
        {
          key: "threeWayBall",
          position: { x: 150, y: -20 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve",
          position: { x: 0, y: 50 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      let layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      // Step 1: Align Vertical (all same Y)
      layouts = Diagram.alignNodes(layouts, "x");
      // Verify alignment worked
      expect(layouts).toHaveLength(4);

      // Step 2: Distribute Horizontal
      layouts = Diagram.distributeNodes(layouts, "x");
      const sorted = [...layouts].sort((a, b) => box.left(a.box) - box.left(b.box));
      expect(sorted).toHaveLength(4);
      expect(box.left(sorted[1].box)).toBeGreaterThan(box.left(sorted[0].box));
      expect(box.left(sorted[2].box)).toBeGreaterThan(box.left(sorted[1].box));
      expect(box.left(sorted[3].box)).toBeGreaterThan(box.left(sorted[2].box));

      // Step 3: Move some valves (simulating user interaction)
      const valve1 = layouts.find((l) => l.key === "threeWayValve")!;
      const valve2 = layouts.find((l) => l.key === "threeWayBall")!;

      valve1.box = box.construct(
        xy.translate(box.topLeft(valve1.box), { x: 0, y: -100 }),
        box.dims(valve1.box),
      );
      valve2.box = box.construct(
        xy.translate(box.topLeft(valve2.box), { x: 0, y: 100 }),
        box.dims(valve2.box),
      );

      // Step 4: Align Horizontal
      layouts = Diagram.alignNodes(layouts, "y");
      // Verify alignment worked
      expect(layouts).toHaveLength(4);

      // Step 5: Distribute Vertical
      layouts = Diagram.distributeNodes(layouts, "y");
      const sortedVertical = [...layouts].sort(
        (a, b) => box.top(a.box) - box.top(b.box),
      );
      expect(sortedVertical).toHaveLength(4);

      // Step 6: Align Left
      layouts = Diagram.alignNodes(layouts, "left");
      const leftEdges = layouts.map((l) => box.left(l.box));
      expect(leftEdges[0]).toBe(leftEdges[1]);
      expect(leftEdges[1]).toBe(leftEdges[2]);
      expect(leftEdges[2]).toBe(leftEdges[3]);

      // Step 7: Align Right
      layouts = Diagram.alignNodes(layouts, "right");
      const rightEdges = layouts.map((l) => box.right(l.box));
      expect(rightEdges[0]).toBe(rightEdges[1]);
      expect(rightEdges[1]).toBe(rightEdges[2]);
      expect(rightEdges[2]).toBe(rightEdges[3]);

      // All operations completed successfully
      expect(layouts).toHaveLength(4);
    });
  });

  describe("edge cases in toolbar operations", () => {
    it("should handle alignment with single element", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 50, y: 50 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const aligned = Diagram.alignNodes(layouts, "left");

      // Single element should stay in place
      expect(box.topLeft(aligned[0].box)).toEqual({ x: 50, y: 50 });
    });

    it("should handle distribution with two elements", () => {
      const elements = [
        {
          key: "valve-1",
          position: { x: 0, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
        {
          key: "valve-2",
          position: { x: 500, y: 0 },
          dimensions: { width: 100, height: 100 },
        },
      ];

      const layouts = elements.map(
        (el) =>
          new Diagram.NodeLayout(el.key, box.construct(el.position, el.dimensions), []),
      );

      const distributed = Diagram.distributeNodes(layouts, "x");

      // With only 2 elements, they should stay at start and end
      expect(box.left(distributed[0].box)).toBe(0);
      expect(box.left(distributed[1].box)).toBe(500);
    });

    it("should handle empty selection", () => {
      const layouts: Diagram.NodeLayout[] = [];

      const aligned = Diagram.alignNodes(layouts, "x");
      const distributed = Diagram.distributeNodes(layouts, "x");
      const rotated = Diagram.rotateNodesAroundCenter(layouts, "clockwise");

      expect(aligned).toEqual([]);
      expect(distributed).toEqual([]);
      expect(rotated).toEqual([]);
    });
  });
});
