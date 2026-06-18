// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type xy } from "@synnaxlabs/x";

const RADIUS = 6;
/// @brief radius of the semicircle a hop arcs over a crossed edge.
const HOP_RADIUS = 5;
/// @brief radius of the quadratic fillet blending each foot of a hop into the straight
/// run, so the joints are rounded rather than meeting at a sharp right angle. The arc and
/// the two fillets together rise HOP_RADIUS + FOOT above the line, matching the footprint
/// of a plain semicircular hop.
const FOOT = 2;
/// @brief half the length a hop occupies along its run: the arc plus a fillet on each
/// side.
const HOP_SPAN = HOP_RADIUS + FOOT;
/// @brief the largest perpendicular gap (in flow units) allowed between a jump point and
/// the run it is snapped onto. Generous because jump points are derived from a polyline
/// reconstructed from the flow store, which can sit a few pixels off the drawn line.
const MAX_SNAP = 16;

const lerp = (a: xy.XY, b: xy.XY, t: number): xy.XY => ({
  x: a.x * (1 - t) + b.x * t,
  y: a.y * (1 - t) + b.y * t,
});

interface Run {
  a: xy.XY;
  from: xy.XY;
  to: xy.XY;
}

/// @brief emits the straight run from `from` to `to`, arcing a hop over each scalar
/// offset along it. The hop is a semicircle lifted clear of the line, joined to the run
/// by a quadratic fillet at each foot so every junction is tangent-continuous. Hops
/// bulge up on horizontal runs and left on vertical runs, regardless of the direction
/// the run is drawn. With no offsets this emits a single lineto, identical to the
/// un-jumped path.
const straight = (from: xy.XY, to: xy.XY, offsets: number[]): string => {
  const tail = `L${to.x},${to.y}`;
  if (offsets.length === 0) return tail;
  const dx = to.x - from.x;
  const dy = to.y - from.y;
  const len = Math.hypot(dx, dy);
  if (len === 0) return tail;
  const ux = dx / len;
  const uy = dy / len;
  const horizontal = Math.abs(ux) >= Math.abs(uy);
  // Bulge horizontal runs up and vertical runs left.
  const nx = horizontal ? 0 : -1;
  const ny = horizontal ? -1 : 0;
  // Sweep that arcs toward that side given the run's drawn direction (derived from the
  // SVG arc-center rules for each of the four axis-aligned orientations).
  const sweep = ux - uy > 0 ? 1 : 0;
  let path = "";
  let last = -Infinity;
  for (const s of [...offsets].sort((p, q) => p - q)) {
    if (s - last < 2 * HOP_SPAN) continue;
    last = s;
    const cx = from.x + s * ux;
    const cy = from.y + s * uy;
    const at = (alongU: number, alongN: number): string =>
      `${cx + alongU * ux + alongN * nx},${cy + alongU * uy + alongN * ny}`;
    path +=
      `L${at(-HOP_SPAN, 0)}Q${at(-HOP_RADIUS, 0)} ${at(-HOP_RADIUS, FOOT)}` +
      `A${HOP_RADIUS},${HOP_RADIUS} 0 0 ${sweep} ${at(HOP_RADIUS, FOOT)}` +
      `Q${at(HOP_RADIUS, 0)} ${at(HOP_SPAN, 0)}`;
  }
  return `${path}${tail}`;
};

/// @brief returns, for each run, the scalar offsets at which a jump should hop. Each jump
/// is snapped to the single nearest run whose interior its projection falls within, so a
/// jump point that sits a few pixels off the drawn line still lands on the correct run.
const assignJumps = (runs: Run[], jumps: xy.XY[]): number[][] => {
  const byRun: number[][] = runs.map(() => []);
  for (const j of jumps) {
    let best = -1;
    let bestPerp = MAX_SNAP;
    let bestS = 0;
    runs.forEach(({ from, to }, i) => {
      const dx = to.x - from.x;
      const dy = to.y - from.y;
      const len = Math.hypot(dx, dy);
      if (len === 0) return;
      const ux = dx / len;
      const uy = dy / len;
      const s = (j.x - from.x) * ux + (j.y - from.y) * uy;
      if (s <= HOP_SPAN || s >= len - HOP_SPAN) return;
      const perp = Math.abs((j.x - from.x) * -uy + (j.y - from.y) * ux);
      if (perp < bestPerp) {
        bestPerp = perp;
        best = i;
        bestS = s;
      }
    });
    if (best >= 0) byRun[best].push(bestS);
  }
  return byRun;
};

/// @brief builds an SVG path string through coords with rounded corners. When jumps are
/// provided, a hop is arced wherever a jump point falls on a run, used to lift an edge
/// over another it crosses. Passing no jumps reproduces the plain rounded path exactly.
export const calcPath = (coords: xy.XY[], jumps: xy.XY[] = []): string => {
  if (coords.length === 0) return "";
  if (coords.length === 1) return `M${coords[0].x},${coords[0].y}`;
  const last = coords.length - 2;
  const runs: Run[] = [];
  for (let i = 0; i <= last; i++) {
    const a = coords[i];
    const b = coords[i + 1];
    const t = Math.min(RADIUS / Math.hypot(b.x - a.x, b.y - a.y), 0.5);
    runs.push({
      a,
      from: i === 0 ? a : lerp(a, b, t),
      to: i === last ? b : lerp(a, b, 1 - t),
    });
  }
  const offsets = assignJumps(runs, jumps);
  let path = "";
  runs.forEach(({ a, from, to }, i) => {
    if (i === 0) path += `M${a.x},${a.y}`;
    else path += `Q${a.x},${a.y} ${from.x},${from.y}`;
    path += straight(from, to, offsets[i]);
  });
  return path;
};
