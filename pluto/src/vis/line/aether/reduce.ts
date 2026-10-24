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
  bounds,
  DataType,
  Series,
  type TypedArray,
} from "@synnaxlabs/x";

import { type telem } from "@/telem/aether";
import { type DrawOperation } from "@/vis/line/aether/line";

type NumberArray = Exclude<TypedArray, BigInt64Array | BigUint64Array>;

interface Points {
  xs: number[];
  ys: number[];
}

const SAMPLE_INDEX_MASK = 0xffffffffn;

const isFullResolution = ({ x, y }: DrawOperation): boolean =>
  x.alignmentMultiple === 1n && y.alignmentMultiple === 1n;

const isFloat32 = (s: Series): boolean => s.dataType.equals(DataType.FLOAT32);

/**
 * @returns the samples per group that draws the full-resolution ops at groupsPerColumn
 * groups per pixel column of the view: their samples per x unit times the x units per
 * pixel, rounded up to a power of two. Returns 1, meaning no reduction, when that is
 * below 2 or a span is zero or not finite.
 */
export const groupSize = (
  ops: DrawOperation[],
  view: telem.View,
  groupsPerColumn: number,
): number => {
  let count = 0;
  let lower = Infinity;
  let upper = -Infinity;
  for (const op of ops) {
    if (!isFullResolution(op)) continue;
    count += op.count;
    const b = op.x.boundsFor(op.xOffset, op.xOffset + op.count);
    lower = Math.min(lower, b.lower);
    upper = Math.max(upper, b.upper);
  }
  const samplesPerPixel =
    (count / (upper - lower)) * (bounds.span(view.bounds) / view.width);
  const size = 2 ** Math.ceil(Math.log2(samplesPerPixel / groupsPerColumn));
  return Number.isFinite(size) && size >= 2 ? size : 1;
};

// Returns the mean of the samples in [lo, hi) as the Core computes it. Summing offsets
// from the minimum keeps the precision of the samples. A UINT8 store truncates the mean
// as the Core does.
const mean = (data: NumberArray, lo: number, hi: number, float32: boolean): number => {
  let min = data[lo];
  for (let i = lo + 1; i < hi; i++) if (data[i] < min) min = data[i];
  let sum = 0;
  for (let i = lo; i < hi; i++)
    sum += float32 ? Math.fround(data[i] - min) : data[i] - min;
  const offset = sum / (hi - lo);
  return min + (float32 ? Math.fround(offset) : offset);
};

// The reduced form of one full-resolution op. Complete groups are reduced once, and
// only the trailing partial group is reduced again as the op grows.
class Pair {
  private readonly x: Series;
  private readonly y: Series;
  private readonly xOffset: number;
  private readonly yOffset: number;
  private readonly alignment: bigint;
  private readonly complete: Points = { xs: [], ys: [] };
  // The op index where the next group starts. It is negative while the first group
  // starts before the op.
  private next: number;
  private count = 0;
  private drawn: DrawOperation | null = null;

  constructor(
    op: DrawOperation,
    private readonly aggregation: Aggregation,
    private readonly groupSize: number,
  ) {
    this.x = op.x;
    this.y = op.y;
    this.xOffset = op.xOffset;
    this.yOffset = op.yOffset;
    const start = op.x.alignment + BigInt(op.xOffset);
    const first = Number((start & SAMPLE_INDEX_MASK) % BigInt(groupSize));
    this.alignment = start - BigInt(first);
    this.next = -first;
  }

  update(count: number): DrawOperation {
    if (this.drawn != null && count === this.count) return this.drawn;
    const { groupSize } = this;
    const xd = this.x.data as NumberArray;
    const yd = this.y.data as NumberArray;
    for (; this.next + groupSize <= count; this.next += groupSize)
      this.reduceGroup(
        xd,
        yd,
        Math.max(this.next, 0),
        this.next + groupSize,
        this.complete,
      );
    const partial: Points = { xs: [], ys: [] };
    if (this.next < count)
      this.reduceGroup(xd, yd, Math.max(this.next, 0), count, partial);
    this.release();
    const x = this.series(this.x, this.complete.xs, partial.xs);
    const y = this.series(this.y, this.complete.ys, partial.ys);
    this.count = count;
    this.drawn = { x, y, xOffset: 0, yOffset: 0, count: x.length };
    return this.drawn;
  }

  release(): void {
    this.drawn?.x.release();
    this.drawn?.y.release();
    this.drawn = null;
  }

  // Appends the points the aggregation keeps from op samples [lo, hi). min_max and
  // decimate pick sample indices from y and take x at the same indices.
  private reduceGroup(
    xd: NumberArray,
    yd: NumberArray,
    lo: number,
    hi: number,
    out: Points,
  ): void {
    const { xOffset, yOffset } = this;
    switch (this.aggregation) {
      case "min_max": {
        let minI = lo;
        let maxI = lo;
        for (let i = lo + 1; i < hi; i++) {
          const v = yd[yOffset + i];
          if (v < yd[yOffset + minI]) minI = i;
          if (v > yd[yOffset + maxI]) maxI = i;
        }
        const first = Math.min(minI, maxI);
        const last = Math.max(minI, maxI);
        out.xs.push(xd[xOffset + first], xd[xOffset + last]);
        out.ys.push(yd[yOffset + first], yd[yOffset + last]);
        break;
      }
      case "average":
        out.xs.push(mean(xd, xOffset + lo, xOffset + hi, isFloat32(this.x)));
        out.ys.push(mean(yd, yOffset + lo, yOffset + hi, isFloat32(this.y)));
        break;
      case "decimate":
        out.xs.push(xd[xOffset + lo]);
        out.ys.push(yd[yOffset + lo]);
        break;
    }
  }

  private series(source: Series, complete: number[], partial: number[]): Series {
    const data = new source.dataType.Array(
      complete.length + partial.length,
    ) as NumberArray;
    data.set(complete);
    data.set(partial, complete.length);
    const { aggregation, groupSize } = this;
    const s = new Series({
      data,
      dataType: source.dataType,
      timeRange: source.timeRange,
      sampleOffset: source.sampleOffset,
      alignment: this.alignment,
      alignmentMultiple: BigInt(aggregation === "min_max" ? groupSize / 2 : groupSize),
    });
    s.acquire();
    return s;
  }
}

/**
 * Reduces the full-resolution ops of a line with the rules the Core uses for tiles.
 * Groups of groupSize samples sit at sample index multiples of groupSize within each
 * alignment domain. The x and y series of an op reduce as a pair: min_max keeps the
 * samples where y is lowest and highest in a group, in time order, decimate keeps the
 * first sample of a group, and average keeps the mean of x and the mean of y. A group
 * only partly inside an op reduces the samples it holds.
 *
 * A reduced pair starts at the alignment of its first group and steps groupSize / 2 per
 * point for min_max, or groupSize otherwise. It keeps the time range and sample offset
 * of its source series. Pairs are cached until they leave the ops, or until the
 * aggregation or group size changes.
 */
export class Reducer {
  private aggregation?: Aggregation;
  private groupSize = 1;
  private readonly pairs = new Map<Series, Map<Series, Pair>>();

  /**
   * @param groupSize - The samples per group. It must be even for min_max.
   * @returns the ops with each full-resolution op replaced by its reduced pair. Ops of
   * reduced series stay the same, and so does every op when groupSize is below 2. The
   * pairs of ops not in the list are released.
   */
  reduce(
    ops: DrawOperation[],
    aggregation: Aggregation,
    groupSize: number,
  ): DrawOperation[] {
    if (aggregation !== this.aggregation || groupSize !== this.groupSize) {
      this.release();
      this.aggregation = aggregation;
      this.groupSize = groupSize;
    }
    const drawn = new Set<Pair>();
    const out = ops.map((op) => {
      if (groupSize < 2 || !isFullResolution(op)) return op;
      let byY = this.pairs.get(op.x);
      if (byY == null) {
        byY = new Map();
        this.pairs.set(op.x, byY);
      }
      let pair = byY.get(op.y);
      if (pair == null) {
        pair = new Pair(op, aggregation, groupSize);
        byY.set(op.y, pair);
      }
      drawn.add(pair);
      return pair.update(op.count);
    });
    for (const [x, byY] of this.pairs) {
      for (const [y, pair] of byY)
        if (!drawn.has(pair)) {
          pair.release();
          byY.delete(y);
        }
      if (byY.size === 0) this.pairs.delete(x);
    }
    return out;
  }

  /** Releases every reduced series, GPU buffers included. */
  release(): void {
    this.pairs.forEach((byY) => byY.forEach((pair) => pair.release()));
    this.pairs.clear();
  }
}
