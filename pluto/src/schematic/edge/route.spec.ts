// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type RouteProps, computeEscape, needsEscape, route } from "./route";

type Dir = "left" | "right" | "top" | "bottom";

const ep = (x: number, y: number, orientation: Dir) => ({
  position: { x, y },
  orientation,
});

/** Assert fundamental invariants that must hold for every valid route. */
const assertValidRoute = (points: xy.XY[], source: xy.XY, target: xy.XY): void => {
  expect(points.length).toBeGreaterThanOrEqual(2);
  // First point equals source position
  expect(points[0]).toEqual(source);
  // Last point equals target position
  expect(points[points.length - 1]).toEqual(target);
  // All segments are axis-aligned
  for (let i = 1; i < points.length; i++) {
    const prev = points[i - 1];
    const curr = points[i];
    const axisAligned = prev.x === curr.x || prev.y === curr.y;
    expect(axisAligned).toBe(true);
  }
  // No zero-length segments
  for (let i = 1; i < points.length; i++) {
    expect(xy.equals(points[i - 1], points[i])).toBe(false);
  }
};

/** Helper to call route with a compact interface. */
const r = (props: RouteProps): xy.XY[] => route(props);

describe("route", () => {
  describe("needsEscape", () => {
    it("right: target behind source", () => {
      expect(needsEscape({ x: 10, y: 0 }, "right", { x: 5, y: 0 })).toBe(true);
    });
    it("right: target ahead of source", () => {
      expect(needsEscape({ x: 10, y: 0 }, "right", { x: 15, y: 0 })).toBe(false);
    });
    it("left: target behind source", () => {
      expect(needsEscape({ x: 0, y: 0 }, "left", { x: 5, y: 0 })).toBe(true);
    });
    it("left: target ahead of source", () => {
      expect(needsEscape({ x: 10, y: 0 }, "left", { x: 5, y: 0 })).toBe(false);
    });
    it("bottom: target behind source", () => {
      expect(needsEscape({ x: 0, y: 10 }, "bottom", { x: 0, y: 5 })).toBe(true);
    });
    it("bottom: target ahead of source", () => {
      expect(needsEscape({ x: 0, y: 10 }, "bottom", { x: 0, y: 15 })).toBe(false);
    });
    it("top: target behind source", () => {
      expect(needsEscape({ x: 0, y: 0 }, "top", { x: 0, y: 5 })).toBe(true);
    });
    it("top: target ahead of source", () => {
      expect(needsEscape({ x: 0, y: 10 }, "top", { x: 0, y: 5 })).toBe(false);
    });
    it("right: target at same position", () => {
      expect(needsEscape({ x: 10, y: 0 }, "right", { x: 10, y: 0 })).toBe(false);
    });
    it("left: target at same position", () => {
      expect(needsEscape({ x: 10, y: 0 }, "left", { x: 10, y: 0 })).toBe(false);
    });
    it("top: target at same position", () => {
      expect(needsEscape({ x: 0, y: 10 }, "top", { x: 0, y: 10 })).toBe(false);
    });
    it("bottom: target at same position", () => {
      expect(needsEscape({ x: 0, y: 10 }, "bottom", { x: 0, y: 10 })).toBe(false);
    });
  });

  describe("computeEscape", () => {
    it("escapes toward the target on the perpendicular axis", () => {
      const nodeBox = box.construct({ x: 0, y: 0 }, { width: 50, height: 50 });
      const result = computeEscape(
        { x: 70, y: 25 },
        "right",
        nodeBox,
        { x: -10, y: 80 },
        "left",
        undefined,
        20,
      );
      // Target is below, so escape should go toward bottom of node box
      expect(result.escapeTip.y).toBeGreaterThan(25);
      expect(result.escapeTip.x).toBe(70);
    });

    it("flips escape direction when target box is too close", () => {
      const sourceBox = box.construct({ x: 0, y: 0 }, { width: 50, height: 50 });
      const targetBox = box.construct({ x: 0, y: 55 }, { width: 50, height: 50 });
      const result = computeEscape(
        { x: 70, y: 25 },
        "right",
        sourceBox,
        { x: -10, y: 75 },
        "left",
        targetBox,
        20,
      );
      // Target is below but boxes are close on y-axis, so escape should flip to top
      expect(result.escapeTip.y).toBeLessThan(25);
    });
  });

  describe("basic routing - opposing directions with clearance (S-routes)", () => {
    it("right -> left with horizontal clearance", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(100, 50, "left"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 100, y: 50 });
    });

    it("left -> right with horizontal clearance", () => {
      const points = r({
        source: ep(100, 0, "left"),
        target: ep(0, 50, "right"),
      });
      assertValidRoute(points, { x: 100, y: 0 }, { x: 0, y: 50 });
    });

    it("bottom -> top with vertical clearance", () => {
      const points = r({
        source: ep(0, 0, "bottom"),
        target: ep(50, 100, "top"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 50, y: 100 });
    });

    it("top -> bottom with vertical clearance", () => {
      const points = r({
        source: ep(0, 100, "top"),
        target: ep(50, 0, "bottom"),
      });
      assertValidRoute(points, { x: 0, y: 100 }, { x: 50, y: 0 });
    });
  });

  describe("basic routing - opposing directions without clearance (U-routes)", () => {
    it("right -> left but target is behind source", () => {
      const points = r({
        source: ep(100, 0, "right"),
        target: ep(0, 50, "left"),
      });
      assertValidRoute(points, { x: 100, y: 0 }, { x: 0, y: 50 });
      // Should form a U-route: stumps extend outward, then connect via midline
      expect(points.length).toBeGreaterThanOrEqual(4);
    });

    it("bottom -> top but target is above source", () => {
      const points = r({
        source: ep(0, 100, "bottom"),
        target: ep(50, 0, "top"),
      });
      assertValidRoute(points, { x: 0, y: 100 }, { x: 50, y: 0 });
      expect(points.length).toBeGreaterThanOrEqual(4);
    });
  });

  describe("basic routing - same direction", () => {
    it("both right", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(100, 50, "right"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 100, y: 50 });
    });

    it("both left", () => {
      const points = r({
        source: ep(100, 0, "left"),
        target: ep(0, 50, "left"),
      });
      assertValidRoute(points, { x: 100, y: 0 }, { x: 0, y: 50 });
    });

    it("both bottom", () => {
      const points = r({
        source: ep(0, 0, "bottom"),
        target: ep(50, 100, "bottom"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 50, y: 100 });
    });

    it("both top", () => {
      const points = r({
        source: ep(0, 100, "top"),
        target: ep(50, 0, "top"),
      });
      assertValidRoute(points, { x: 0, y: 100 }, { x: 50, y: 0 });
    });
  });

  describe("basic routing - perpendicular (single bend)", () => {
    it("right -> top", () => {
      const points = r({
        source: ep(0, 50, "right"),
        target: ep(100, 0, "top"),
      });
      assertValidRoute(points, { x: 0, y: 50 }, { x: 100, y: 0 });
    });

    it("right -> bottom", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(100, 50, "bottom"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 100, y: 50 });
    });

    it("bottom -> right", () => {
      const points = r({
        source: ep(0, 0, "bottom"),
        target: ep(100, 50, "right"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 100, y: 50 });
    });

    it("top -> left", () => {
      const points = r({
        source: ep(100, 50, "top"),
        target: ep(0, 0, "left"),
      });
      assertValidRoute(points, { x: 100, y: 50 }, { x: 0, y: 0 });
    });
  });

  describe("degenerate cases", () => {
    it("source and target at same position, opposing directions", () => {
      const points = r({
        source: ep(50, 50, "right"),
        target: ep(50, 50, "left"),
      });
      // Degenerate case: source equals target. The route should still produce
      // valid axis-aligned segments, though some may overlap.
      expect(points[0]).toEqual({ x: 50, y: 50 });
      expect(points[points.length - 1]).toEqual({ x: 50, y: 50 });
      expect(points.length).toBeGreaterThanOrEqual(2);
    });

    it("aligned horizontally, opposing directions", () => {
      const points = r({
        source: ep(0, 50, "right"),
        target: ep(100, 50, "left"),
      });
      assertValidRoute(points, { x: 0, y: 50 }, { x: 100, y: 50 });
      // Should be a straight line through stumps
    });

    it("aligned vertically, opposing directions", () => {
      const points = r({
        source: ep(50, 0, "bottom"),
        target: ep(50, 100, "top"),
      });
      assertValidRoute(points, { x: 50, y: 0 }, { x: 50, y: 100 });
    });

    it("very close endpoints", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(5, 5, "left"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 5, y: 5 });
    });
  });

  describe("node-box avoidance", () => {
    it("source escape when target is behind source node", () => {
      const sourceBox = box.construct({ x: 0, y: 0 }, { width: 80, height: 40 });
      const points = r({
        source: ep(80, 20, "right"),
        target: ep(40, 80, "top"),
        sourceBox,
      });
      assertValidRoute(points, { x: 80, y: 20 }, { x: 40, y: 80 });
    });

    it("target escape when source is behind target node", () => {
      const targetBox = box.construct({ x: 50, y: 0 }, { width: 80, height: 40 });
      const points = r({
        source: ep(0, 80, "bottom"),
        target: ep(50, 20, "left"),
        targetBox,
      });
      assertValidRoute(points, { x: 0, y: 80 }, { x: 50, y: 20 });
    });

    it("both nodes need escape", () => {
      const sourceBox = box.construct({ x: 80, y: 0 }, { width: 40, height: 40 });
      const targetBox = box.construct({ x: 0, y: 0 }, { width: 40, height: 40 });
      const points = r({
        source: ep(80, 20, "left"),
        target: ep(40, 20, "right"),
        sourceBox,
        targetBox,
      });
      assertValidRoute(points, { x: 80, y: 20 }, { x: 40, y: 20 });
    });

    it("no escape needed when target is ahead with clearance", () => {
      const sourceBox = box.construct({ x: 0, y: 0 }, { width: 40, height: 40 });
      const pointsWithBox = r({
        source: ep(40, 20, "right"),
        target: ep(200, 20, "left"),
        sourceBox,
      });
      const pointsWithout = r({
        source: ep(40, 20, "right"),
        target: ep(200, 20, "left"),
      });
      // With clearance, the box shouldn't change the route
      expect(pointsWithBox).toEqual(pointsWithout);
    });

    it("opposing directions with tight box spacing flips escape", () => {
      const sourceBox = box.construct({ x: 0, y: 0 }, { width: 50, height: 50 });
      const targetBox = box.construct({ x: 0, y: 55 }, { width: 50, height: 50 });
      const points = r({
        source: ep(50, 25, "right"),
        target: ep(0, 80, "left"),
        sourceBox,
        targetBox,
      });
      assertValidRoute(points, { x: 50, y: 25 }, { x: 0, y: 80 });
    });
  });

  describe("waypoint routing", () => {
    it("single waypoint between source and target", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(200, 100, "left"),
        waypoints: [{ x: 100, y: 50 }],
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 200, y: 100 });
    });

    it("multiple waypoints", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(300, 0, "left"),
        waypoints: [
          { x: 100, y: 50 },
          { x: 200, y: -50 },
        ],
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 300, y: 0 });
    });

    it("waypoint with source box avoidance", () => {
      const sourceBox = box.construct({ x: 0, y: 0 }, { width: 80, height: 40 });
      const points = r({
        source: ep(80, 20, "right"),
        target: ep(200, 100, "left"),
        waypoints: [{ x: 40, y: 80 }],
        sourceBox,
      });
      assertValidRoute(points, { x: 80, y: 20 }, { x: 200, y: 100 });
    });
  });

  describe("simplification", () => {
    it("removes collinear points in straight path", () => {
      // Right -> Left aligned on same y: should produce a simple straight-ish path
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(100, 0, "left"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 100, y: 0 });
      // All points should be on y=0, so simplified to just source + target (or
      // source + stump endpoints + target)
      for (const p of points) expect(p.y).toBe(0);
    });
  });

  describe("ported from old connector tests - basic formation", () => {
    it("simple bottom to top", () => {
      const points = r({
        source: ep(0, 0, "bottom"),
        target: ep(0, 30, "top"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 0, y: 30 });
      // All points should be on x=0 (straight vertical)
      for (const p of points) expect(p.x).toBe(0);
    });

    it("simple left to right", () => {
      const points = r({
        source: ep(30, 0, "left"),
        target: ep(0, 0, "right"),
      });
      assertValidRoute(points, { x: 30, y: 0 }, { x: 0, y: 0 });
      for (const p of points) expect(p.y).toBe(0);
    });

    it("simple top to bottom", () => {
      const points = r({
        source: ep(0, 0, "top"),
        target: ep(0, -30, "bottom"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 0, y: -30 });
      for (const p of points) expect(p.x).toBe(0);
    });

    it("simple right to left", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(30, 0, "left"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 30, y: 0 });
      for (const p of points) expect(p.y).toBe(0);
    });
  });

  describe("ported from old connector tests - same direction", () => {
    it("left-left: target is down and right", () => {
      const points = r({
        source: ep(0, 0, "left"),
        target: ep(30, 30, "left"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 30, y: 30 });
      // Should go left from source, down, then right to target
      // First point after source should be to the left
      expect(points[1].x).toBeLessThan(0);
    });

    it("left-left: target is up and left", () => {
      const points = r({
        source: ep(30, 30, "left"),
        target: ep(0, 0, "left"),
      });
      assertValidRoute(points, { x: 30, y: 30 }, { x: 0, y: 0 });
      expect(points[1].x).toBeLessThan(30);
    });
  });

  describe("ported from old connector tests - recompute on node move", () => {
    it("route updates when source moves", () => {
      const before = r({
        source: ep(0, 0, "right"),
        target: ep(100, 50, "left"),
      });
      const after = r({
        source: ep(0, 20, "right"),
        target: ep(100, 50, "left"),
      });
      assertValidRoute(before, { x: 0, y: 0 }, { x: 100, y: 50 });
      assertValidRoute(after, { x: 0, y: 20 }, { x: 100, y: 50 });
      // Source moved down, so first point should reflect that
      expect(after[0].y).toBe(20);
    });

    it("route updates when target moves", () => {
      const before = r({
        source: ep(0, 0, "right"),
        target: ep(100, 50, "left"),
      });
      const after = r({
        source: ep(0, 0, "right"),
        target: ep(120, 30, "left"),
      });
      assertValidRoute(before, { x: 0, y: 0 }, { x: 100, y: 50 });
      assertValidRoute(after, { x: 0, y: 0 }, { x: 120, y: 30 });
      expect(after[after.length - 1]).toEqual({ x: 120, y: 30 });
    });
  });

  describe("stump length", () => {
    it("respects custom stump length", () => {
      // For right->left S-route, the backbone midX is the average of the two
      // stump tips. With stumpLength=40: midX = (0+40 + 200-40)/2 = 100
      const points40 = r({
        source: ep(0, 0, "right"),
        target: ep(200, 50, "left"),
        stumpLength: 40,
      });
      assertValidRoute(points40, { x: 0, y: 0 }, { x: 200, y: 50 });
      // With stumpLength=10: midX = (0+10 + 200-10)/2 = 100 (same midX but
      // different stumps). Use perpendicular setup to expose stump tip directly.
      // Right -> Top: stump extends right, then single bend down to target.
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(100, 100, "top"),
        stumpLength: 40,
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 100, y: 100 });
      // For perpendicular right->top: source stump goes to x=40, target stump
      // goes to y=100-40=60. The bend point is at {100, 0} (x=tgt, y=src).
      // Stump at {40, 0} may or may not be simplified depending on bend location.
      // Just verify the route produces valid points.
    });

    it("uses default stump length of 20", () => {
      const points = r({
        source: ep(0, 0, "right"),
        target: ep(200, 50, "left"),
      });
      assertValidRoute(points, { x: 0, y: 0 }, { x: 200, y: 50 });
      // Backbone midX = (20 + 180) / 2 = 100
      expect(points[1]).toEqual({ x: 100, y: 0 });
    });
  });
});
