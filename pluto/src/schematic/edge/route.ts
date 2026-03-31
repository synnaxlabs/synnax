// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type location, type xy } from "@synnaxlabs/x";

import { type diagram } from "@/vis/diagram/aether";

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

/** Returns the perpendicular direction axis for a given direction. */
const perpendicular = (dir: Dir): "x" | "y" => (isHorizontal(dir) ? "y" : "x");

/** Returns the primary axis for a given direction. */
const primaryAxis = (dir: Dir): "x" | "y" => (isHorizontal(dir) ? "x" : "y");

/**
 * Returns true when the target stump tip lies "behind" the source in its
 * departure direction, meaning the naive route would cross back through the
 * source node body.
 */
export const needsEscape = (stumpTip: xy.XY, dir: Dir, targetTip: xy.XY): boolean => {
  const axis = primaryAxis(dir);
  const delta = targetTip[axis] - stumpTip[axis];
  switch (dir) {
    case "right":
    case "bottom":
      return delta < 0;
    case "left":
    case "top":
      return delta > 0;
  }
};

/**
 * When a route would clip back through a node, compute escape points that route
 * around the node box on the perpendicular axis.
 *
 * Returns the escaped tip position, the new effective direction for the backbone,
 * and the intermediate points to add to the path.
 */
export const computeEscape = (
  stumpTip: xy.XY,
  dir: Dir,
  nodeBox: box.Box,
  targetTip: xy.XY,
  targetDir: Dir,
  targetBox: box.Box | undefined,
  stumpLength: number,
): { escapeTip: xy.XY; escapeDir: Dir; points: xy.XY[] } => {
  const perpAxis = perpendicular(dir);

  // Decide which edge of the node box to escape toward on the perpendicular axis.
  // Default: the edge closer to the target on the perpendicular axis.
  const targetPerpCoord = targetTip[perpAxis];
  const boxTop = perpAxis === "y" ? box.top(nodeBox) : box.left(nodeBox);
  const boxBottom = perpAxis === "y" ? box.bottom(nodeBox) : box.right(nodeBox);

  const distToTop = Math.abs(targetPerpCoord - boxTop);
  const distToBottom = Math.abs(targetPerpCoord - boxBottom);
  let escapeToward: "top" | "bottom" | "left" | "right" =
    perpAxis === "y"
      ? distToTop <= distToBottom
        ? "top"
        : "bottom"
      : distToTop <= distToBottom
        ? "left"
        : "right";

  // Special case: if opposing directions and target box is close on the
  // perpendicular axis, flip to the other edge to avoid clipping through the
  // target box.
  if (targetBox != null && dir === opposite(targetDir)) {
    const targetBoxEdge =
      perpAxis === "y"
        ? escapeToward === "top"
          ? box.bottom(targetBox)
          : box.top(targetBox)
        : escapeToward === "left"
          ? box.right(targetBox)
          : box.left(targetBox);
    const nodeBoxEdge =
      perpAxis === "y"
        ? escapeToward === "top"
          ? box.top(nodeBox)
          : box.bottom(nodeBox)
        : escapeToward === "left"
          ? box.left(nodeBox)
          : box.right(nodeBox);
    if (Math.abs(nodeBoxEdge - targetBoxEdge) < stumpLength) {
      escapeToward =
        perpAxis === "y"
          ? escapeToward === "top"
            ? "bottom"
            : "top"
          : escapeToward === "left"
            ? "right"
            : "left";
    }
  }

  // Compute the escape coordinate: node box edge + margin in the escape direction.
  const nodeEdge =
    perpAxis === "y"
      ? escapeToward === "top"
        ? box.top(nodeBox)
        : box.bottom(nodeBox)
      : escapeToward === "left"
        ? box.left(nodeBox)
        : box.right(nodeBox);
  const sign =
    escapeToward === "top" || escapeToward === "left" ? -stumpLength : stumpLength;
  const escapeCoord = nodeEdge + sign;

  // Build escape points: move perpendicular to clear the box.
  const escapeMid: xy.XY =
    perpAxis === "y"
      ? { x: stumpTip.x, y: escapeCoord }
      : { x: escapeCoord, y: stumpTip.y };

  return {
    escapeTip: escapeMid,
    escapeDir: escapeToward as Dir,
    points: [escapeMid],
  };
};

/**
 * Clamp a coordinate value so it doesn't fall inside any obstacle box on the
 * given axis. If the value is inside a box, push it to the nearest edge + margin.
 * `crossMin`/`crossMax` define the range on the perpendicular axis that the
 * segment using this coordinate will span. Only boxes that overlap on the
 * cross-axis are considered.
 */
const clampCoord = (
  value: number,
  axis: "x" | "y",
  crossMin: number,
  crossMax: number,
  obstacles: box.Box[],
  margin: number,
): number => {
  for (const ob of obstacles) {
    const obMin = axis === "x" ? box.left(ob) : box.top(ob);
    const obMax = axis === "x" ? box.right(ob) : box.bottom(ob);
    const obCrossMin = axis === "x" ? box.top(ob) : box.left(ob);
    const obCrossMax = axis === "x" ? box.bottom(ob) : box.right(ob);
    if (value <= obMin || value >= obMax) continue;
    if (crossMax <= obCrossMin || crossMin >= obCrossMax) continue;
    const distToMin = value - obMin;
    const distToMax = obMax - value;
    if (distToMin <= distToMax) value = obMin - margin;
    else value = obMax + margin;
  }
  return value;
};

/**
 * Compute the backbone points between two stump endpoints. The stump endpoints
 * are already extended outward from their ports by the stump length.
 *
 * Returns only the intermediate points (not A or B themselves).
 *
 * When obstacles are provided, intermediate coordinates are clamped to avoid
 * routing through any obstacle box.
 */
const computeBackbone = (
  a: xy.XY,
  b: xy.XY,
  srcDir: Dir,
  tgtDir: Dir,
  obstacles: box.Box[],
  margin: number,
): xy.XY[] => {
  const srcH = isHorizontal(srcDir);
  const tgtH = isHorizontal(tgtDir);

  if (srcH && tgtH) {
    if (srcDir === opposite(tgtDir)) {
      const goingRight = srcDir === "right";
      const clearance = goingRight ? a.x < b.x : a.x > b.x;
      if (clearance) {
        const yMin = Math.min(a.y, b.y);
        const yMax = Math.max(a.y, b.y);
        const midX = clampCoord((a.x + b.x) / 2, "x", yMin, yMax, obstacles, margin);
        return [
          { x: midX, y: a.y },
          { x: midX, y: b.y },
        ];
      }
      const xMin = Math.min(a.x, b.x);
      const xMax = Math.max(a.x, b.x);
      const midY = clampCoord((a.y + b.y) / 2, "y", xMin, xMax, obstacles, margin);
      return [
        { x: a.x, y: midY },
        { x: b.x, y: midY },
      ];
    }
    const xMin = Math.min(a.x, b.x);
    const xMax = Math.max(a.x, b.x);
    const midY = clampCoord((a.y + b.y) / 2, "y", xMin, xMax, obstacles, margin);
    return [
      { x: a.x, y: midY },
      { x: b.x, y: midY },
    ];
  }

  if (!srcH && !tgtH) {
    if (srcDir === opposite(tgtDir)) {
      const goingDown = srcDir === "bottom";
      const clearance = goingDown ? a.y < b.y : a.y > b.y;
      if (clearance) {
        const xMin = Math.min(a.x, b.x);
        const xMax = Math.max(a.x, b.x);
        const midY = clampCoord((a.y + b.y) / 2, "y", xMin, xMax, obstacles, margin);
        return [
          { x: a.x, y: midY },
          { x: b.x, y: midY },
        ];
      }
      const yMin = Math.min(a.y, b.y);
      const yMax = Math.max(a.y, b.y);
      const midX = clampCoord((a.x + b.x) / 2, "x", yMin, yMax, obstacles, margin);
      return [
        { x: midX, y: a.y },
        { x: midX, y: b.y },
      ];
    }
    const yMin = Math.min(a.y, b.y);
    const yMax = Math.max(a.y, b.y);
    const midX = clampCoord((a.x + b.x) / 2, "x", yMin, yMax, obstacles, margin);
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

/**
 * Create an L-shaped bend between two points. Returns the bend point(s)
 * connecting `from` to `to` with orthogonal segments. If already axis-aligned,
 * returns nothing (simplify will handle it).
 */
const lBend = (from: xy.XY, to: xy.XY): xy.XY[] => {
  if (from.x === to.x || from.y === to.y) return [to];
  return [{ x: to.x, y: from.y }, to];
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
  source: diagram.EdgeEndpoint;
  /** Position of the target handle. */
  target: diagram.EdgeEndpoint;
  /** Optional user-placed intermediate waypoints. */
  waypoints?: xy.XY[];
  /** Stump length in pixels. Default 20. */
  stumpLength?: number;
  /** Bounding box of the source node (for avoidance). */
  sourceBox?: box.Box;
  /** Bounding box of the target node (for avoidance). */
  targetBox?: box.Box;
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
 *
 * When sourceBox/targetBox are provided, the router adds escape segments to
 * avoid clipping through node bodies.
 */
export const route = ({
  source,
  target,
  waypoints = [],
  stumpLength = STUMP_LENGTH,
  sourceBox,
  targetBox,
}: RouteProps): xy.XY[] => {
  if (waypoints.length === 0)
    return routeDirect(
      source.position,
      source.orientation,
      target.position,
      target.orientation,
      stumpLength,
      sourceBox,
      targetBox,
    );

  // Build the full chain: source handle → waypoints → target handle.
  // Only source and target get stumps/escape. Waypoints are simple bend
  // points connected with L-shaped orthogonal segments.
  const a = extend(source.position, source.orientation, stumpLength);
  const lastWp = waypoints[waypoints.length - 1];
  const b = extend(target.position, target.orientation, stumpLength);

  const allPoints: xy.XY[] = [source.position, a];

  // Apply source escape if first waypoint is behind source
  let effectiveA = a;
  if (sourceBox != null && needsEscape(a, source.orientation, waypoints[0])) {
    const esc = computeEscape(
      a,
      source.orientation,
      sourceBox,
      waypoints[0],
      inferDirection(waypoints[0], a),
      undefined,
      stumpLength,
    );
    allPoints.push(...esc.points);
    effectiveA = esc.escapeTip;
  }

  // Source stump/escape tip → first waypoint: L-bend for orthogonal transition
  allPoints.push(...lBend(effectiveA, waypoints[0]));

  // Middle waypoints: connect each pair. If consecutive waypoints are axis-aligned,
  // they stitch directly. Otherwise, insert an L-bend.
  for (let i = 1; i < waypoints.length; i++)
    allPoints.push(...lBend(waypoints[i - 1], waypoints[i]));

  // Last waypoint → target stump tip: L-bend for orthogonal transition
  allPoints.push(...lBend(lastWp, b));
  allPoints.push(target.position);

  return simplify(allPoints);
};

/** Route directly between two points with no intermediate waypoints. */
const routeDirect = (
  source: xy.XY,
  sourceDir: Dir,
  target: xy.XY,
  targetDir: Dir,
  stumpLength: number,
  sourceBox?: box.Box,
  targetBox?: box.Box,
): xy.XY[] => {
  const a = extend(source, sourceDir, stumpLength);
  const b = extend(target, targetDir, stumpLength);

  let srcEscapePoints: xy.XY[] = [];
  let tgtEscapePoints: xy.XY[] = [];
  let effectiveA = a;
  let effectiveSrcDir = sourceDir;
  let effectiveB = b;
  let effectiveTgtDir = targetDir;

  if (sourceBox != null && needsEscape(a, sourceDir, b)) {
    const esc = computeEscape(
      a,
      sourceDir,
      sourceBox,
      b,
      targetDir,
      targetBox,
      stumpLength,
    );
    srcEscapePoints = esc.points;
    effectiveA = esc.escapeTip;
    effectiveSrcDir = esc.escapeDir;
  }

  if (targetBox != null && needsEscape(b, targetDir, effectiveA)) {
    const esc = computeEscape(
      b,
      targetDir,
      targetBox,
      effectiveA,
      effectiveSrcDir,
      sourceBox,
      stumpLength,
    );
    tgtEscapePoints = esc.points;
    effectiveB = esc.escapeTip;
    effectiveTgtDir = esc.escapeDir;
  }

  const obstacles: box.Box[] = [];
  if (sourceBox != null) obstacles.push(sourceBox);
  if (targetBox != null) obstacles.push(targetBox);

  const backbone = computeBackbone(
    effectiveA,
    effectiveB,
    effectiveSrcDir,
    effectiveTgtDir,
    obstacles,
    stumpLength,
  );
  return simplify([
    source,
    a,
    ...srcEscapePoints,
    ...backbone,
    ...tgtEscapePoints.reverse(),
    b,
    target,
  ]);
};
