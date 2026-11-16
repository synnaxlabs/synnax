// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";

import { type Path } from "@/schematic/edge/common/path";

/// @brief an edge's polyline, widths, and render order; higher order paints on top.
export interface Polyline {
  key: string;
  points: xy.XY[];
  order: number;
  /// @brief the width of each line the edge draws.
  strokeWidth: number;
  /// @brief the full width the edge draws, including any parallel lines.
  width: number;
}

/// @brief gap between a hop's arc and the crossed line.
const HOP_CLEARANCE = 3;

/// @brief clearance beyond the hop radius that a crossing needs from the ends of the
/// hop-bearing segment, so the full arc fits clear of that segment's corners.
const WINNER_MARGIN = 9;
/// @brief clearance from the ends of the crossed segment, only enough to skip a true
/// T-junction so hops persist almost to where the crossed line ends.
const CROSS_BUFFER = 2;

/// @brief off-axis tolerance for treating a segment as axis-aligned.
const AXIS_EPSILON = 0.5;

interface Span {
  key: string;
  order: number;
  strokeWidth: number;
  width: number;
  cross: number;
  lo: number;
  hi: number;
}

const pushSpans = (
  { key, points, order, strokeWidth, width }: Polyline,
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
        strokeWidth,
        width,
        cross: a.y,
        lo: Math.min(a.x, b.x),
        hi: Math.max(a.x, b.x),
      });
    else if (dx <= AXIS_EPSILON && dy > AXIS_EPSILON)
      vertical.push({
        key,
        order,
        strokeWidth,
        width,
        cross: a.x,
        lo: Math.min(a.y, b.y),
        hi: Math.max(a.y, b.y),
      });
  }
};

/// @brief returns the hop points per edge: where a horizontal run of one edge crosses a
/// vertical run of another, assigned to the edge on top. Crossings near a segment's
/// ends are dropped; collinear overlaps and self-crossings are ignored by construction.
export const findCrossings = (polylines: Polyline[]): Map<string, Path.Crossing[]> => {
  const horizontal: Span[] = [];
  const vertical: Span[] = [];
  for (const p of polylines) pushSpans(p, horizontal, vertical);
  const result = new Map<string, Path.Crossing[]>();
  for (const h of horizontal)
    for (const v of vertical) {
      if (h.key === v.key) continue;
      const x = v.cross;
      const y = h.cross;
      const hWins = h.order >= v.order;
      const [top, crossed] = hWins ? [h, v] : [v, h];
      // The top edge hops on each of its own lines, so only its stroke counts. The
      // hop must clear every line of the crossed edge.
      const radius = (top.strokeWidth + crossed.width) / 2 + HOP_CLEARANCE;
      // The hop is drawn along the winner's segment, so only it needs the wide
      // clearance; the crossed segment just needs to clear a T-junction. An inner
      // parallel line turns its corners sooner, by the line's offset from the core.
      const winnerBuf = radius + WINNER_MARGIN + (top.width - top.strokeWidth) / 2;
      const hBuf = hWins ? winnerBuf : CROSS_BUFFER;
      const vBuf = hWins ? CROSS_BUFFER : winnerBuf;
      if (x <= h.lo + hBuf || x >= h.hi - hBuf) continue;
      if (y <= v.lo + vBuf || y >= v.hi - vBuf) continue;
      const existing = result.get(top.key);
      if (existing == null) result.set(top.key, [{ x, y, radius }]);
      else existing.push({ x, y, radius });
    }
  return result;
};
