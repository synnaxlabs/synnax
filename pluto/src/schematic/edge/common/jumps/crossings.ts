// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";

/// @brief a single edge's polyline together with the z-order at which it renders.
/// order is the edge's index in the render list; higher values paint on top.
export interface Polyline {
  key: string;
  points: xy.XY[];
  order: number;
}

/// @brief the minimum distance (in flow units) a crossing must sit from either
/// crossed segment's endpoints. Keeps hops clear of the rounded corners at vertices
/// and of the handles where edges legitimately meet.
const END_BUFFER = 14;

/// @brief a segment is treated as axis-aligned only when its off-axis extent is below
/// this threshold, absorbing sub-pixel noise in endpoint coordinates.
const AXIS_EPSILON = 0.5;

interface Span {
  key: string;
  order: number;
  cross: number;
  lo: number;
  hi: number;
}

const pushSpans = (
  { key, points, order }: Polyline,
  horizontal: Span[],
  vertical: Span[],
): void => {
  for (let i = 0; i < points.length - 1; i++) {
    const a = points[i];
    const b = points[i + 1];
    const dx = Math.abs(b.x - a.x);
    const dy = Math.abs(b.y - a.y);
    if (dy <= AXIS_EPSILON && dx > AXIS_EPSILON)
      horizontal.push({
        key,
        order,
        cross: a.y,
        lo: Math.min(a.x, b.x),
        hi: Math.max(a.x, b.x),
      });
    else if (dx <= AXIS_EPSILON && dy > AXIS_EPSILON)
      vertical.push({
        key,
        order,
        cross: a.x,
        lo: Math.min(a.y, b.y),
        hi: Math.max(a.y, b.y),
      });
  }
};

/// @brief finds the points at which a horizontal segment of one edge strictly crosses a
/// vertical segment of another, and assigns each crossing to whichever edge renders on
/// top (the one that should draw the hop). Crossings within END_BUFFER of either
/// segment's endpoints are dropped so hops never collide with corners or shared
/// handles. Collinear overlaps and same-edge self-crossings are ignored by
/// construction. Returns a map from edge key to the hop points on that edge; edges
/// without hops are absent.
export const findCrossings = (polylines: Polyline[]): Map<string, xy.XY[]> => {
  const horizontal: Span[] = [];
  const vertical: Span[] = [];
  for (const p of polylines) pushSpans(p, horizontal, vertical);
  const result = new Map<string, xy.XY[]>();
  for (const h of horizontal)
    for (const v of vertical) {
      if (h.key === v.key) continue;
      const x = v.cross;
      const y = h.cross;
      if (x <= h.lo + END_BUFFER || x >= h.hi - END_BUFFER) continue;
      if (y <= v.lo + END_BUFFER || y >= v.hi - END_BUFFER) continue;
      const winner = h.order >= v.order ? h.key : v.key;
      const existing = result.get(winner);
      if (existing == null) result.set(winner, [{ x, y }]);
      else existing.push({ x, y });
    }
  return result;
};
