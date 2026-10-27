// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Aggregation,
  POINTS_PER_GROUP,
  TimeRange,
  TimeSpan,
  type TimeStamp,
} from "@synnaxlabs/x";

/**
 * Identifies one tile of the grid. Tiles at level L span 2^L milliseconds, and tile i
 * of a level starts i spans after the Unix epoch.
 */
export interface Position {
  level: number;
  index: number;
}

/** A tile, optionally cut short at either end. */
export interface Bounds extends Position {
  /** Starts the range after the tile does. */
  start?: TimeStamp;
  /** Ends the range before the tile does. */
  end?: TimeStamp;
}

/** @returns the span of every tile at the given level. */
export const tileSpan = (level: number): TimeSpan =>
  new TimeSpan(TimeSpan.MILLISECOND.valueOf() << BigInt(level));

/** @returns the time range the tile covers, cut to its start and end. */
export const tileRange = ({ level, index, start, end }: Bounds): TimeRange => {
  const span = tileSpan(level).valueOf();
  let lower = BigInt(index) * span;
  let upper = lower + span;
  if (start != null && start.valueOf() > lower) lower = start.valueOf();
  if (end != null && end.valueOf() < upper) upper = end.valueOf();
  return new TimeRange(lower, upper);
};

/** @returns the smallest level whose tiles span at least half of the range. */
export const level = (tr: TimeRange): number => {
  const half = tr.span.valueOf() / 2n;
  let l = 0;
  while (tileSpan(l).valueOf() < half) l++;
  return l;
};

// Returns a / b rounded down, for a positive b.
const floorDiv = (a: bigint, b: bigint): bigint => {
  const q = a / b;
  return a % b < 0n ? q - 1n : q;
};

/** @returns the index of the tile of the level that holds the timestamp. */
export const index = (ts: TimeStamp, level: number): number =>
  Number(floorDiv(ts.valueOf(), tileSpan(level).valueOf()));

/** @returns the indexes of every tile of the level that overlaps the range. */
export const indexes = (tr: TimeRange, level: number): number[] => {
  const span = tileSpan(level).valueOf();
  const first = floorDiv(tr.start.valueOf(), span);
  let last = -floorDiv(-tr.end.valueOf(), span);
  if (last <= first) last = first + 1n;
  const out: number[] = [];
  for (let i = first; i < last; i++) out.push(Number(i));
  return out;
};

export interface PointLimitProps {
  level: number;
  /** The range the line shows. */
  view: TimeRange;
  /** About how many points the line draws across the view. */
  points: number;
  /** The aggregation the line draws. */
  drawn: Aggregation;
  /** The aggregation of the tile. */
  aggregation: Aggregation;
}

/**
 * @returns the point limit of one tile at the level: its share of the line's groups
 * across the view, times the points per group of the tile's aggregation. The limit
 * rounds up to a power of two, so a resize of a few pixels keeps the same limit.
 */
export const pointLimit = ({
  level,
  view,
  points,
  drawn,
  aggregation,
}: PointLimitProps): number => {
  const groups = points / POINTS_PER_GROUP[drawn];
  const share =
    (Number(tileSpan(level).valueOf()) / Number(view.span.valueOf())) * groups;
  return 2 ** Math.ceil(Math.log2(Math.max(share * POINTS_PER_GROUP[aggregation], 1)));
};

/** @returns a key that identifies the position. */
export const positionKey = ({ level, index }: Position): string => `${level}/${index}`;

/**
 * Chooses the tiles that draw the given slots of a level, from the available ones. Each
 * slot draws the first of: its own tile, finer tiles that cover it whole, the finest
 * coarser tile whose slots in view hold nothing else, and finer tiles that cover part
 * of it. A slot with none of these stays empty. No two chosen tiles overlap.
 * @param level - The level of the slots.
 * @param slots - Ascending indexes of the slots at the level.
 * @param available - The tiles that can be drawn now.
 * @returns the chosen tiles in time order.
 */
export const choose = (
  level: number,
  slots: number[],
  available: Position[],
): Position[] => {
  const held = new Set(available.map(positionKey));
  // above holds each position of the level or finer that has an available tile at or
  // below it.
  const above = new Set<string>();
  for (const p of available)
    for (let l = p.level, i = p.index; l <= level; l++, i = Math.floor(i / 2))
      above.add(positionKey({ level: l, index: i }));
  // finer returns the coarsest available tiles at or below p in time order, and
  // whether they cover p whole.
  const finer = (p: Position): [Position[], boolean] => {
    if (held.has(positionKey(p))) return [[p], true];
    if (p.level === 0 || !above.has(positionKey(p))) return [[], false];
    const [first, firstWhole] = finer({ level: p.level - 1, index: 2 * p.index });
    const [second, secondWhole] = finer({ level: p.level - 1, index: 2 * p.index + 1 });
    return [[...first, ...second], firstWhole && secondWhole];
  };
  const chosen = new Map<number, Position[]>();
  const partial = new Map<number, Position[]>();
  for (const index of slots) {
    const [tiles, whole] = finer({ level, index });
    if (whole) chosen.set(index, tiles);
    else if (tiles.length > 0) partial.set(index, tiles);
  }
  const coarser = available
    .filter((p) => p.level > level)
    .sort((a, b) => a.level - b.level);
  for (const ancestor of coarser) {
    const span = 2 ** (ancestor.level - level);
    const covered = slots.filter((s) => Math.floor(s / span) === ancestor.index);
    if (covered.length === 0 || covered.some((s) => chosen.has(s))) continue;
    covered.forEach((s, i) => chosen.set(s, i === 0 ? [ancestor] : []));
  }
  partial.forEach((tiles, index) => {
    if (!chosen.has(index)) chosen.set(index, tiles);
  });
  return slots.flatMap((s) => chosen.get(s) ?? []);
};
