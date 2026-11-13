// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";

const EPSILON = 1e-6;

const collinear = (a: xy.XY, b: xy.XY, c: xy.XY): boolean =>
  Math.abs((b.x - a.x) * (c.y - b.y) - (b.y - a.y) * (c.x - b.x)) < EPSILON;

const corners = (points: xy.XY[]): xy.XY[] => {
  const out: xy.XY[] = [];
  for (const p of points) {
    while (out.length >= 2 && collinear(out[out.length - 2], out[out.length - 1], p))
      out.pop();
    if (out.length === 0 || !xy.equals(out[out.length - 1], p, EPSILON)) out.push(p);
  }
  return out;
};

const reversed = (from: xy.XY[], to: xy.XY[], i: number): boolean =>
  xy.dot(xy.sub(to[i + 1], to[i]), xy.sub(from[i + 1], from[i])) <= 0;

/**
 * @returns the polyline parallel to points at the signed distance, on the side of
 * `xy.normal` when positive. A run that doubles back draws no spike, and an end run
 * shorter than the distance is cut off so the line does not fold back past its end.
 */
export const offset = (points: xy.XY[], distance: number): xy.XY[] => {
  const base = corners(points);
  if (base.length < 2) return [];
  const miters = xy.calculateMiters(base, distance);
  const out = base.map((p, i) => xy.translate(p, miters[i]));
  while (out.length > 2 && reversed(base, out, 0)) {
    base.shift();
    out.shift();
  }
  while (out.length > 2 && reversed(base, out, out.length - 2)) {
    base.pop();
    out.pop();
  }
  return out;
};
