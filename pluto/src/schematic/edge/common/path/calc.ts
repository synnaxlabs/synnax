// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { xy } from "@synnaxlabs/x";

const RADIUS = 6;
/// @brief radius of a hop's semicircle.
const HOP_RADIUS = 5;
/// @brief fillet radius rounding each foot of a hop into the run; arc plus fillets rise
/// HOP_RADIUS + FOOT, matching a plain semicircle's footprint.
const FOOT = 2;
/// @brief half a hop's length along the run (arc plus a fillet each side).
const HOP_SPAN = HOP_RADIUS + FOOT;
/// @brief largest perpendicular gap allowed when snapping a jump onto a run; generous
/// since reconstructed jump points can sit a few pixels off the drawn line.
const MAX_SNAP = 16;

interface Run {
  a: xy.XY;
  from: xy.XY;
  to: xy.XY;
}

/// @brief emits the run from `from` to `to`, arcing a tangent-continuous hop at each
/// offset. Hops bulge up on horizontal runs, left on vertical. No offsets yields a plain
/// lineto.
const straight = (from: xy.XY, to: xy.XY, offsets: number[]): string => {
  const tail = `L${to.x},${to.y}`;
  if (offsets.length === 0 || xy.equals(from, to)) return tail;
  const u = xy.normalize(xy.sub(to, from));
  const horizontal = Math.abs(u.x) >= Math.abs(u.y);
  // Bulge horizontal runs up, vertical runs left.
  const nx = horizontal ? 0 : -1;
  const ny = horizontal ? -1 : 0;
  // Sweep toward that side for the run's drawn direction (per SVG arc-center rules).
  const sweep = u.x - u.y > 0 ? 1 : 0;
  let path = "";
  let last = -Infinity;
  for (const s of [...offsets].sort((p, q) => p - q)) {
    if (s - last < 2 * HOP_SPAN) continue;
    last = s;
    const cx = from.x + s * u.x;
    const cy = from.y + s * u.y;
    const at = (alongU: number, alongN: number): string =>
      `${cx + alongU * u.x + alongN * nx},${cy + alongU * u.y + alongN * ny}`;
    path +=
      `L${at(-HOP_SPAN, 0)}Q${at(-HOP_RADIUS, 0)} ${at(-HOP_RADIUS, FOOT)}` +
      `A${HOP_RADIUS},${HOP_RADIUS} 0 0 ${sweep} ${at(HOP_RADIUS, FOOT)}` +
      `Q${at(HOP_RADIUS, 0)} ${at(HOP_SPAN, 0)}`;
  }
  return `${path}${tail}`;
};

/// @brief per-run offsets at which to hop. Each jump snaps to the nearest run it projects
/// onto, so a slightly-off jump point still lands on the right run.
const assignJumps = (runs: Run[], jumps: xy.XY[]): number[][] => {
  const byRun: number[][] = runs.map(() => []);
  for (const j of jumps) {
    let best = -1;
    let bestPerp = MAX_SNAP;
    let bestS = 0;
    runs.forEach(({ from, to }, i) => {
      const len = xy.distance(from, to);
      if (len === 0) return;
      const u = xy.scale(xy.sub(to, from), 1 / len);
      const d = xy.sub(j, from);
      const s = xy.dot(d, u);
      if (s <= HOP_SPAN || s >= len - HOP_SPAN) return;
      const perp = Math.abs(xy.dot(d, xy.normal(u)));
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

/// @brief builds a rounded-corner SVG path through coords, arcing a hop wherever a jump
/// falls on a run. No jumps reproduces the plain rounded path exactly.
export const rounded = (coords: xy.XY[], jumps: xy.XY[] = []): string => {
  if (coords.length === 0) return "";
  if (coords.length === 1) return `M${coords[0].x},${coords[0].y}`;
  const last = coords.length - 2;
  const runs: Run[] = [];
  for (let i = 0; i <= last; i++) {
    const a = coords[i];
    const b = coords[i + 1];
    const t = Math.min(RADIUS / xy.distance(a, b), 0.5);
    runs.push({
      a,
      from: i === 0 ? a : xy.lerp(a, b, t),
      to: i === last ? b : xy.lerp(a, b, 1 - t),
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
