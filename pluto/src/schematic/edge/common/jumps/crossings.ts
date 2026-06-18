// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";

/// @brief an edge's polyline and render order; higher order paints on top.
export interface Polyline {
  key: string;
  points: xy.XY[];
  order: number;
}

/// @brief clearance a crossing needs from the ends of the hop-bearing segment, so the arc
/// fits clear of that segment's corners.
const WINNER_BUFFER = 14;
/// @brief clearance from the ends of the crossed segment, only enough to skip a true
/// T-junction so hops persist almost to where the crossed line ends.
const CROSS_BUFFER = 2;

/// @brief off-axis tolerance for treating a segment as axis-aligned.
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

/// @brief returns the hop points per edge: where a horizontal run of one edge crosses a
/// vertical run of another, assigned to the edge on top. Crossings near a segment's ends
/// are dropped; collinear overlaps and self-crossings are ignored by construction.
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
      // The hop is drawn along the winner's segment, so only it needs the wide clearance;
      // the crossed segment just needs to clear a T-junction.
      const hWins = h.order >= v.order;
      const hBuf = hWins ? WINNER_BUFFER : CROSS_BUFFER;
      const vBuf = hWins ? CROSS_BUFFER : WINNER_BUFFER;
      if (x <= h.lo + hBuf || x >= h.hi - hBuf) continue;
      if (y <= v.lo + vBuf || y >= v.hi - vBuf) continue;
      const winner = hWins ? h.key : v.key;
      const existing = result.get(winner);
      if (existing == null) result.set(winner, [{ x, y }]);
      else existing.push({ x, y });
    }
  return result;
};
