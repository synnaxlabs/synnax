// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type location, xy } from "@synnaxlabs/x";

type Dir = location.Outer;

const STUMP_LENGTH = 20;

const extend = (p: xy.XY, dir: Dir, dist: number): xy.XY => {
  switch (dir) {
    case "right":
      return { x: p.x + dist, y: p.y };
    case "left":
      return { x: p.x - dist, y: p.y };
    case "bottom":
      return { x: p.x, y: p.y + dist };
    case "top":
      return { x: p.x, y: p.y - dist };
  }
};

const isHorizontal = (dir: Dir): boolean => dir === "left" || dir === "right";

const opposite = (dir: Dir): Dir => {
  switch (dir) {
    case "left":
      return "right";
    case "right":
      return "left";
    case "top":
      return "bottom";
    case "bottom":
      return "top";
  }
};

/**
 * Compute the backbone points between two stump endpoints. The stump endpoints
 * are already extended outward from their ports by the stump length.
 *
 * Returns only the intermediate points (not A or B themselves).
 */
const computeBackbone = (a: xy.XY, b: xy.XY, srcDir: Dir, tgtDir: Dir): xy.XY[] => {
  const srcH = isHorizontal(srcDir);
  const tgtH = isHorizontal(tgtDir);

  if (srcH && tgtH) {
    if (srcDir === opposite(tgtDir)) {
      // Opposing horizontal (e.g. RIGHT -> LEFT): S-route or U-route
      const goingRight = srcDir === "right";
      const clearance = goingRight ? a.x < b.x : a.x > b.x;
      if (clearance) {
        const midX = (a.x + b.x) / 2;
        return [
          { x: midX, y: a.y },
          { x: midX, y: b.y },
        ];
      }
      // U-route: pick vertical escape
      const midY = (a.y + b.y) / 2;
      return [
        { x: a.x, y: midY },
        { x: b.x, y: midY },
      ];
    }
    // Same direction (both RIGHT or both LEFT)
    const midY = (a.y + b.y) / 2;
    return [
      { x: a.x, y: midY },
      { x: b.x, y: midY },
    ];
  }

  if (!srcH && !tgtH) {
    if (srcDir === opposite(tgtDir)) {
      // Opposing vertical (e.g. BOTTOM -> TOP)
      const goingDown = srcDir === "bottom";
      const clearance = goingDown ? a.y < b.y : a.y > b.y;
      if (clearance) {
        const midY = (a.y + b.y) / 2;
        return [
          { x: a.x, y: midY },
          { x: b.x, y: midY },
        ];
      }
      // U-route
      const midX = (a.x + b.x) / 2;
      return [
        { x: midX, y: a.y },
        { x: midX, y: b.y },
      ];
    }
    // Same direction (both TOP or both BOTTOM)
    const midX = (a.x + b.x) / 2;
    return [
      { x: midX, y: a.y },
      { x: midX, y: b.y },
    ];
  }

  // One horizontal, one vertical: single bend
  if (srcH) return [{ x: b.x, y: a.y }];
  return [{ x: a.x, y: b.y }];
};

/** Remove collinear intermediate points from an orthogonal path. */
const simplify = (points: xy.XY[]): xy.XY[] => {
  if (points.length <= 2) return points;
  const result = [points[0]];
  for (let i = 1; i < points.length - 1; i++) {
    const prev = result[result.length - 1];
    const curr = points[i];
    const next = points[i + 1];
    const sameX = prev.x === curr.x && curr.x === next.x;
    const sameY = prev.y === curr.y && curr.y === next.y;
    if (!sameX && !sameY) result.push(curr);
  }
  result.push(points[points.length - 1]);
  return result;
};

/** Infer the best departure direction from a point toward a target. */
const inferDirection = (from: xy.XY, to: xy.XY): Dir => {
  const dx = to.x - from.x;
  const dy = to.y - from.y;
  if (Math.abs(dx) >= Math.abs(dy)) return dx >= 0 ? "right" : "left";
  return dy >= 0 ? "bottom" : "top";
};

export interface RouteProps {
  /** Position of the source handle. */
  source: xy.XY;
  /** Direction the source handle faces. */
  sourceDir: Dir;
  /** Position of the target handle. */
  target: xy.XY;
  /** Direction the target handle faces. */
  targetDir: Dir;
  /** Optional user-placed intermediate waypoints. */
  waypoints?: xy.XY[];
  /** Stump length in pixels. Default 20. */
  stumpLength?: number;
}

/**
 * Compute an orthogonal route between two handle positions.
 *
 * Returns an array of XY points representing the path. The first point is the
 * source position and the last is the target position. All intermediate segments
 * are axis-aligned (horizontal or vertical).
 *
 * When waypoints are provided, the route passes through each waypoint in order.
 * The routing algorithm runs segment-by-segment between consecutive points.
 */
export const route = ({
  source,
  sourceDir,
  target,
  targetDir,
  waypoints = [],
  stumpLength = STUMP_LENGTH,
}: RouteProps): xy.XY[] => {
  if (waypoints.length === 0) return routeDirect(source, sourceDir, target, targetDir, stumpLength);

  const allPoints: xy.XY[] = [];

  // Source -> first waypoint
  const wp0Dir = opposite(inferDirection(waypoints[0], source));
  const seg0 = routeDirect(source, sourceDir, waypoints[0], wp0Dir, stumpLength);
  allPoints.push(...seg0.slice(0, -1));

  // Waypoint -> waypoint
  for (let i = 0; i < waypoints.length - 1; i++) {
    const fromDir = inferDirection(waypoints[i], waypoints[i + 1]);
    const toDir = opposite(inferDirection(waypoints[i + 1], waypoints[i]));
    const seg = routeDirect(waypoints[i], fromDir, waypoints[i + 1], toDir, stumpLength);
    allPoints.push(...seg.slice(0, -1));
  }

  // Last waypoint -> target
  const lastWp = waypoints[waypoints.length - 1];
  const lastWpDir = inferDirection(lastWp, target);
  const segN = routeDirect(lastWp, lastWpDir, target, targetDir, stumpLength);
  allPoints.push(...segN);

  return simplify(allPoints);
};

/** Route directly between two points with no intermediate waypoints. */
const routeDirect = (
  source: xy.XY,
  sourceDir: Dir,
  target: xy.XY,
  targetDir: Dir,
  stumpLength: number,
): xy.XY[] => {
  const a = extend(source, sourceDir, stumpLength);
  const b = extend(target, targetDir, stumpLength);
  const backbone = computeBackbone(a, b, sourceDir, targetDir);
  return simplify([source, a, ...backbone, b, target]);
};
