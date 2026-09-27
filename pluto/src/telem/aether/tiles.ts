// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { framer } from "@synnaxlabs/client";
import { type Aggregation, type TimeRange } from "@synnaxlabs/x";
import { z } from "zod";

/** How many aggregation groups a line draws per pixel column. */
export const detailZ = z.enum(["low", "medium", "high"]);
export type Detail = z.infer<typeof detailZ>;

const GROUPS_PER_COLUMN: Record<Detail, number> = {
  low: 1 / 4,
  medium: 1 / 2,
  high: 1,
};

/** Identifies one tile of the grid: tile index of level. */
export interface Position {
  level: number;
  index: number;
}

/** @returns the smallest level whose tiles span at least half of the range. */
export const level = (tr: TimeRange): number => {
  const half = tr.span.valueOf() / 2n;
  let l = 0;
  while (framer.tileSpan(l).valueOf() < half) l++;
  return l;
};

/** @returns the indexes of every tile of the level that overlaps the range. */
export const indexes = (tr: TimeRange, level: number): number[] => {
  const span = framer.tileSpan(level).valueOf();
  const first = tr.start.valueOf() / span;
  let last = (tr.end.valueOf() + span - 1n) / span;
  if (last <= first) last = first + 1n;
  const out: number[] = [];
  for (let i = first; i < last; i++) out.push(Number(i));
  return out;
};

export interface PointLimitProps {
  level: number;
  /** The range the line shows. */
  view: TimeRange;
  /** The width of the line in pixels. */
  width: number;
  detail: Detail;
  aggregation: Aggregation;
}

/**
 * @returns the point limit of one tile at the level: its share of the line's pixel
 * width, times groups per pixel column, times points per group. The limit rounds up to
 * a power of two, so a resize of a few pixels keeps the same limit.
 */
export const pointLimit = ({
  level,
  view,
  width,
  detail,
  aggregation,
}: PointLimitProps): number => {
  const share =
    (Number(framer.tileSpan(level).valueOf()) / Number(view.span.valueOf())) * width;
  const perGroup = aggregation === "min_max" ? 2 : 1;
  const points = Math.max(share * GROUPS_PER_COLUMN[detail] * perGroup, 1);
  return 2 ** Math.ceil(Math.log2(points));
};

// How many levels above the target a coarse tile may come from.
const MAX_COARSE_LEVELS = 8;

/**
 * Chooses the tiles that draw the given slots of a level, from those that are
 * available. Each slot draws its own tile if available, else both its children, else
 * the finest available ancestor whose slots in view hold nothing else. A slot with none
 * of these stays empty. No two chosen tiles overlap.
 * @param level - The level of the slots.
 * @param slots - Ascending indexes of the slots at the level.
 * @param available - Returns true if the tile at the position can be drawn now.
 * @returns the chosen tiles in time order.
 */
export const choose = (
  level: number,
  slots: number[],
  available: (p: Position) => boolean,
): Position[] => {
  const chosen = new Map<number, Position[]>();
  for (const index of slots) {
    if (available({ level, index })) {
      chosen.set(index, [{ level, index }]);
      continue;
    }
    if (level === 0) continue;
    const children = [2 * index, 2 * index + 1].map((i) => ({
      level: level - 1,
      index: i,
    }));
    if (children.every(available)) chosen.set(index, children);
  }
  for (let up = 1; up <= MAX_COARSE_LEVELS; up++)
    for (const index of slots) {
      if (chosen.has(index)) continue;
      const ancestor = { level: level + up, index: Math.floor(index / 2 ** up) };
      if (!available(ancestor)) continue;
      const covered = slots.filter((s) => Math.floor(s / 2 ** up) === ancestor.index);
      if (covered.some((s) => chosen.has(s))) continue;
      covered.forEach((s, i) => chosen.set(s, i === 0 ? [ancestor] : []));
    }
  return slots.flatMap((s) => chosen.get(s) ?? []);
};
