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
  math,
  type NumberArray,
  Series,
  TimeRange,
} from "@synnaxlabs/x";

import { type DrawOperation, touching } from "@/vis/line/aether/draw";

interface Points {
  xs: number[];
  ys: number[];
}

// The series a pair draws. Its span moves as a live series closes, so it never caches.
type Drawn = Omit<DrawOperation, "span">;

const SAMPLE_INDEX_MASK = 0xffffffffn;

const isFullResolution = ({ x, y }: DrawOperation): boolean =>
  x.alignmentMultiple === 1n && y.alignmentMultiple === 1n;

const startOf = ({ x, xOffset }: DrawOperation): bigint =>
  x.alignment + BigInt(xOffset);

const endOf = (op: DrawOperation): bigint => startOf(op) + BigInt(op.count);

/** The span of x a line shows, and its width in pixels. */
export interface View {
  bounds: bounds.Bounds;
  width: number;
}

/**
 * @returns the samples per group that draws the full-resolution ops at groupsPerColumn
 * groups per pixel column of the view: their samples per x unit times the x units per
 * pixel, rounded up to a power of two. Returns 1, meaning no reduction, when that is
 * below 2 or a span is zero or not finite.
 */
export const groupSize = (
  ops: DrawOperation[],
  view: View,
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

// The x or y data of one op in a chain, and the shift that moves its values onto the
// GL offset of the chain's first op.
interface Axis {
  data: NumberArray;
  offset: number;
  shift: number;
}

// One op of a chain. A chain is an op followed by the ops whose samples continue it.
interface Link {
  x: Axis;
  y: Axis;
  count: number;
}

const axis = (s: Series, offset: number, first: Series): Axis => ({
  data: s.glData,
  offset,
  shift: Number(math.sub(s.glOffset, first.glOffset)),
});

const link = (op: DrawOperation, first: DrawOperation): Link => ({
  x: axis(op.x, op.xOffset, first.x),
  y: axis(op.y, op.yOffset, first.y),
  count: op.count,
});

// Returns the mean of chain samples [lo, hi) of a group of size samples on one axis, as
// the Core computes it. A float mean sums each sample over size, then scales a partial
// group by size over its count. An integer mean rounds down. It sums the quotients and
// remainders of the samples over the count apart, so no sum passes 2^53.
const mean = (
  chain: Link[],
  key: "x" | "y",
  lo: number,
  hi: number,
  size: number,
  float: boolean,
): number => {
  const n = hi - lo;
  let q = 0;
  let r = 0;
  let base = 0;
  for (const { [key]: a, count } of chain) {
    if (base >= hi) break;
    const to = a.offset + Math.min(hi - base, count);
    for (let i = a.offset + Math.max(lo - base, 0); i < to; i++) {
      const v = a.data[i] + a.shift;
      if (float) {
        q += v / size;
        continue;
      }
      const d = Math.floor(v / n);
      q += d;
      r += v - d * n;
      if (r >= n) {
        q++;
        r -= n;
      }
    }
    base += count;
  }
  return float && n !== size ? q * (size / n) : q;
};

// The reduced form of one full-resolution op. The op draws each group that starts in
// it, reading past its end into the ops that continue it. With no op before it, it also
// draws the group it starts inside. Complete groups reduce once. Only a trailing
// partial group reduces again.
class Pair {
  private readonly x: Series;
  private readonly y: Series;
  private readonly start: bigint;
  // The op's first sample index within its group.
  private readonly first: number;
  // The op index where the first group the op draws starts. It is negative when that
  // group starts before the op.
  private origin: number;
  // The op index where the next group starts.
  private next: number;
  // The op index where the last complete group ends.
  private reach = 0;
  private complete: Points = { xs: [], ys: [] };
  private count = 0;
  private available = 0;
  private drawn: Drawn | null = null;

  constructor(
    op: DrawOperation,
    private readonly aggregation: Aggregation,
    private readonly groupSize: number,
    preceded: boolean,
  ) {
    this.x = op.x;
    this.y = op.y;
    this.start = startOf(op);
    this.first = Number((this.start & SAMPLE_INDEX_MASK) % BigInt(groupSize));
    this.origin = this.originOf(preceded);
    this.next = this.origin;
  }

  /**
   * @param chain - The op, then the ops that continue it, up to groupSize samples past
   * its end where they exist.
   * @param preceded - Whether an op that the op continues ends where it starts.
   */
  update(chain: DrawOperation[], preceded: boolean): Drawn {
    const { groupSize } = this;
    const { count } = chain[0];
    const origin = this.originOf(preceded);
    const groups = Math.ceil((count - origin) / groupSize);
    // Samples past the last group the op draws do not change its points.
    const end = groups > 0 ? origin + groups * groupSize : count;
    let available = 0;
    for (const op of chain) available += op.count;
    available = Math.min(available, end);
    if (
      this.drawn != null &&
      origin === this.origin &&
      count === this.count &&
      available === this.available
    )
      return this.drawn;
    if (origin !== this.origin || available < this.reach) this.reset(origin);
    const links = chain.map((op) => link(op, chain[0]));
    while (this.next < count && this.next + groupSize <= available) {
      this.reduceGroup(
        links,
        Math.max(this.next, 0),
        this.next + groupSize,
        this.complete,
      );
      this.next += groupSize;
      this.reach = this.next;
    }
    const partial: Points = { xs: [], ys: [] };
    if (this.next < count)
      this.reduceGroup(links, Math.max(this.next, 0), available, partial);
    this.release();
    const x = this.series(this.x, this.complete.xs, partial.xs);
    const y = this.series(this.y, this.complete.ys, partial.ys);
    this.count = count;
    this.available = available;
    this.drawn = { x, y, xOffset: 0, yOffset: 0, count: x.length };
    return this.drawn;
  }

  release(): void {
    this.drawn?.x.release();
    this.drawn?.y.release();
    this.drawn = null;
  }

  private originOf(preceded: boolean): number {
    const { first, groupSize } = this;
    return preceded ? (groupSize - first) % groupSize : -first;
  }

  private reset(origin: number): void {
    this.origin = origin;
    this.next = origin;
    this.reach = 0;
    this.complete = { xs: [], ys: [] };
  }

  // Appends the points the aggregation keeps from chain samples [lo, hi), where lo lies
  // in the first op. min_max and decimate pick sample indices from y and take x at the
  // same indices.
  private reduceGroup(chain: Link[], lo: number, hi: number, out: Points): void {
    switch (this.aggregation) {
      case "min_max": {
        let minK = 0;
        let maxK = 0;
        let minX = NaN;
        let minY = NaN;
        let maxX = NaN;
        let maxY = NaN;
        let k = 0;
        let base = 0;
        for (const { x, y, count } of chain) {
          if (base >= hi) break;
          const to = Math.min(hi - base, count);
          for (let i = Math.max(lo - base, 0); i < to; i++, k++) {
            const v = y.data[y.offset + i] + y.shift;
            if (k === 0 || v < minY) {
              minK = k;
              minX = x.data[x.offset + i] + x.shift;
              minY = v;
            }
            if (k === 0 || v > maxY) {
              maxK = k;
              maxX = x.data[x.offset + i] + x.shift;
              maxY = v;
            }
          }
          base += count;
        }
        if (minK <= maxK) {
          out.xs.push(minX, maxX);
          out.ys.push(minY, maxY);
        } else {
          out.xs.push(maxX, minX);
          out.ys.push(maxY, minY);
        }
        break;
      }
      case "average":
        out.xs.push(mean(chain, "x", lo, hi, this.groupSize, this.x.dataType.isFloat));
        out.ys.push(mean(chain, "y", lo, hi, this.groupSize, this.y.dataType.isFloat));
        break;
      case "decimate": {
        const { x, y } = chain[0];
        out.xs.push(x.data[x.offset + lo]);
        out.ys.push(y.data[y.offset + lo]);
        break;
      }
    }
  }

  private series(source: Series, complete: number[], partial: number[]): Series {
    const { glDataType } = source;
    const data = new glDataType.Array(complete.length + partial.length) as NumberArray;
    data.set(complete);
    data.set(partial, complete.length);
    const { aggregation, groupSize } = this;
    const s = new Series({
      data,
      dataType: glDataType,
      timeRange: source.timeRange,
      sampleOffset: source.glOffset,
      alignment: this.start + BigInt(this.origin),
      alignmentMultiple: BigInt(aggregation === "min_max" ? groupSize / 2 : groupSize),
    });
    s.acquire();
    return s;
  }
}

// Returns the op, then the ops that continue it, until they hold groupSize samples past
// its end. The chain stops where the ops stop touching in time, since block alignment
// goes on across a restart.
const chainOf = (
  op: DrawOperation,
  after: Map<bigint, DrawOperation>,
  groupSize: number,
): DrawOperation[] => {
  const chain = [op];
  let samples = op.count;
  let last = op;
  let next = after.get(endOf(last));
  while (next != null && touching(last, next) && samples < op.count + groupSize) {
    chain.push(next);
    samples += next.count;
    last = next;
    next = after.get(endOf(last));
  }
  return chain;
};

/**
 * Reduces the full-resolution ops of a line with the rules of the Core's reduced reads.
 * Groups of groupSize samples sit at sample index multiples of groupSize within each
 * alignment domain. The x and y series of an op reduce as a pair: min_max keeps the
 * samples where y is lowest and highest in a group, in time order, decimate keeps the
 * first sample of a group, and average keeps the mean of x and the mean of y. A group
 * only partly inside the ops reduces the samples it holds.
 *
 * Ops that continue one another in alignment and touch in time reduce as one run of
 * samples. The op a group starts in draws it, and the first op of a run also draws the
 * group it starts inside. An op that draws no group gives its span to the op before it,
 * so the two ops around it still touch.
 *
 * A reduced pair starts at the alignment of its first group and steps groupSize / 2 per
 * point for min_max, or groupSize otherwise. It holds the GL data type, and keeps the
 * time range and GL offset, of its source series. Pairs are cached until they leave the
 * ops, or until the aggregation or group size changes.
 */
export class Reducer {
  private aggregation?: Aggregation;
  private groupSize = 1;
  private readonly pairs = new Map<Series, Map<Series, Pair>>();

  /**
   * @param groupSize - The samples per group. It must be even for min_max. An average
   * of an integer series rounds down, as the Core does.
   * @returns the ops with each full-resolution op replaced by its reduced pair, or left
   * out when it draws no group. Ops of reduced series stay the same, and so does every
   * op when groupSize is below 2. The pairs of ops not in the list are released.
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
    if (groupSize < 2) return ops;
    const after = new Map<bigint, DrawOperation>();
    const before = new Map<bigint, DrawOperation>();
    for (const op of ops) {
      if (!isFullResolution(op)) continue;
      if (!after.has(startOf(op))) after.set(startOf(op), op);
      before.set(endOf(op), op);
    }
    const drawn = new Set<Pair>();
    const out: DrawOperation[] = [];
    for (const op of ops) {
      if (!isFullResolution(op)) {
        out.push(op);
        continue;
      }
      const prev = before.get(startOf(op));
      const preceded = prev != null && touching(prev, op);
      let byY = this.pairs.get(op.x);
      if (byY == null) {
        byY = new Map();
        this.pairs.set(op.x, byY);
      }
      let pair = byY.get(op.y);
      if (pair == null) {
        pair = new Pair(op, aggregation, groupSize, preceded);
        byY.set(op.y, pair);
      }
      drawn.add(pair);
      const reduced = pair.update(chainOf(op, after, groupSize), preceded);
      const last = out.at(-1);
      if (reduced.count > 0) out.push({ ...reduced, span: op.span });
      else if (last != null && touching(last, op))
        out[out.length - 1] = {
          ...last,
          span: new TimeRange(last.span.start, op.span.end),
        };
    }
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
