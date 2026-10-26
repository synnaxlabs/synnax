// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, Series, TimeRange, TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { type Dynamic } from "@/framer/cache/dynamic";
import { index as tileIndex, tileRange } from "@/framer/cache/grid";

/** The grid level of raw blocks: 2^16 milliseconds, about 65.5 seconds. */
export const RAW_LEVEL = 16;

/** A cached part of one channel's samples. */
export interface Block {
  readonly channel: channel.Key;
  /** When the oldest fetch the block answers reads with landed. */
  readonly fetchedAt: TimeStamp;
  /** The bytes of the block's samples. */
  readonly bytes: number;
  /** True while the block must stay cached, as when a caller holds its series. */
  readonly held: boolean;
}

/** @returns the index of the raw block that holds the timestamp. */
export const rawIndex = (ts: TimeStamp): number => tileIndex(ts, RAW_LEVEL);

/** @returns the store key of the channel's raw block at the index. */
export const rawKey = (ch: channel.Key, index: number): string => `${ch}/raw/${index}`;

const derive = (
  s: Series,
  {
    alignment = s.alignment,
    timeRange = s.timeRange,
  }: { alignment?: bigint; timeRange?: TimeRange },
): Series =>
  new Series({
    data: s.data,
    dataType: s.dataType,
    timeRange,
    sampleOffset: s.sampleOffset,
    alignment,
    alignmentMultiple: s.alignmentMultiple,
  });

// A run of index samples with consecutive Core alignments.
interface Segment {
  start: bigint;
  end: bigint;
  offset: number;
}

/**
 * Sets block alignment on a fetch's series of one channel in one block. Block alignment
 * is (index << 32) | offset, where offset counts index samples from the block start.
 * @param index - The raw block the series are in.
 * @param base - The index samples in the block before the fetch's start.
 * @param timestamps - The fetch's series of the channel's index in the block, in time
 * order.
 * @param series - The fetch's series of the channel in the block.
 * @returns the non-empty series, with block alignment.
 * @throws {UnexpectedError} if a series has no index samples with its Core alignment.
 */
export const align = (
  index: number,
  base: number,
  timestamps: Series[],
  series: Series[],
): Series[] => {
  const segments: Segment[] = [];
  let offset = base;
  for (const s of timestamps) {
    const last = segments.at(-1);
    const end = s.alignment + BigInt(s.length);
    if (last?.end === s.alignment) last.end = end;
    else segments.push({ start: s.alignment, end, offset });
    offset += s.length;
  }
  const prefix = BigInt(index) << 32n;
  return series
    .filter((s) => s.length > 0)
    .map((s) => {
      const { lower, upper } = s.alignmentBounds;
      const seg = segments.find((g) => g.start <= lower && upper <= g.end);
      if (seg == null)
        throw new UnexpectedError(
          `no index samples share alignment ${lower} in raw block ${index}`,
        );
      return derive(s, {
        alignment: prefix | BigInt(seg.offset + Number(lower - seg.start)),
      });
    });
};

// Returns the position of the first sample of the timestamp series at or after t, from
// lo.
const search = (s: Series, t: bigint, lo = 0): number => {
  let hi = s.length;
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    if (BigInt(s.data[mid]) < t) lo = mid + 1;
    else hi = mid;
  }
  return lo;
};

/** @returns the number of samples of the timestamp series stamped before ts. */
export const countBefore = (timestamps: Series[], ts: TimeStamp): number =>
  timestamps.reduce((count, s) => count + search(s, ts.valueOf()), 0);

/** The index samples and samples of one channel that a fetch read in one block. */
export interface Part {
  timestamps: Series[];
  series: Series[];
}

const cut = (s: Series, tr: TimeRange): Series =>
  derive(s, {
    timeRange: new TimeRange(
      TimeStamp.max(s.timeRange.start, tr.start),
      TimeStamp.min(s.timeRange.end, tr.end),
    ),
  });

/**
 * Splits a fetch's series of one channel at the edges of the raw blocks first to last,
 * and cuts the time range of each part to its block. A series cut in two copies its
 * samples, so a block never holds the buffer of samples another block holds. Samples
 * with no index sample of the same Core alignment stay out, as when the Core commits a
 * channel before its index.
 * @param first - The first raw block the fetch read.
 * @param last - The last raw block the fetch read.
 * @param timestamps - The fetch's series of the channel's index, in time order.
 * @param series - The fetch's series of the channel, in time order.
 * @returns the part of each block, from first to last.
 */
export const split = (
  first: number,
  last: number,
  timestamps: Series[],
  series: Series[],
): Part[] => {
  const parts: Part[] = Array.from({ length: last - first + 1 }, () => ({
    timestamps: [],
    series: [],
  }));
  // edges holds the Core alignment bounds of the index samples of each part, in order.
  const edges: Array<{ part: Part; range: TimeRange; lower: bigint; upper: bigint }> =
    [];
  for (const ts of timestamps)
    for (let lo = 0; lo < ts.length;) {
      const index = rawIndex(new TimeStamp(ts.data[lo]));
      const range = tileRange({ level: RAW_LEVEL, index });
      const hi = search(ts, range.end.valueOf(), lo);
      const part = parts[index - first];
      part.timestamps.push(cut(ts.slice(lo, hi), range));
      const lower = ts.alignment + BigInt(lo);
      edges.push({ part, range, lower, upper: lower + BigInt(hi - lo) });
      lo = hi;
    }
  let e = 0;
  for (const s of series) {
    const { lower, upper } = s.alignmentBounds;
    while (e < edges.length && edges[e].upper <= lower) e++;
    for (let i = e; i < edges.length && edges[i].lower < upper; i++) {
      const { part, range } = edges[i];
      const from = edges[i].lower > lower ? edges[i].lower : lower;
      const to = edges[i].upper < upper ? edges[i].upper : upper;
      part.series.push(cut(s.slice(Number(from - lower), Number(to - lower)), range));
    }
  }
  return parts;
};

const sumBytes = (series: Series[]): number =>
  series.reduce((sum, s) => sum + s.byteCapacity.valueOf(), 0);

/** A streamed series and the span its samples cover. */
interface Streamed {
  series: Series;
  range: TimeRange;
}

/**
 * One channel's raw samples over one level 16 tile of the grid. Fetched series carry
 * block alignment and cover the block from its start. Streamed series keep the stream's
 * alignment and never claim coverage.
 */
export class RawBlock implements Block {
  readonly range: TimeRange;
  fetchedAt = TimeStamp.ZERO;
  /** Fetched samples cover [range.start, coveredEnd). */
  coveredEnd: TimeStamp;
  /** The index samples in [range.start, coveredEnd). */
  coveredCount = 0;
  private _fetched: Series[] = [];
  private _streamed: Series[] = [];
  /** The channel's live buffer while a stream writes into the block. */
  buffer: Dynamic | null = null;
  /** Settles when the fetch that fills the block lands. */
  pending: Promise<void> | null = null;

  constructor(
    readonly channel: channel.Key,
    readonly index: number,
    private readonly now: () => TimeStamp = () => TimeStamp.now(),
  ) {
    this.range = tileRange({ level: RAW_LEVEL, index });
    this.coveredEnd = this.range.start;
  }

  /** Fetched series in alignment order, with no overlap. */
  get fetched(): readonly Series[] {
    return this._fetched;
  }

  /** Buffers the stream filled while the block was open. */
  get streamed(): readonly Series[] {
    return this._streamed;
  }

  get bytes(): number {
    const live = this.buffer?.leadingBuffer?.byteCapacity.valueOf() ?? 0;
    return sumBytes(this._fetched) + sumBytes(this._streamed) + live;
  }

  get held(): boolean {
    return (
      this.buffer != null ||
      this.pending != null ||
      this.fetched.some((s) => s.refCount > 0) ||
      this.streamed.some((s) => s.refCount > 0)
    );
  }

  /** @returns true when fetched samples cover the part of tr inside the block. */
  covers(tr: TimeRange): boolean {
    return TimeStamp.min(tr.end, this.range.end).beforeEq(this.coveredEnd);
  }

  /**
   * Drops the fetched samples, so reads fetch the block again.
   * @returns true if the block held fetched samples or coverage.
   */
  expire(): boolean {
    if (this._fetched.length === 0 && this.coveredEnd.equals(this.range.start))
      return false;
    this._fetched = [];
    this.coveredEnd = this.range.start;
    this.coveredCount = 0;
    return true;
  }

  /** Drops every sample and the coverage, so reads fetch the block from its start. */
  reset(): void {
    this.expire();
    this._streamed = [];
  }

  /** Adds buffers the stream filled. */
  stream(series: Series[]): void {
    this._streamed.push(...series);
  }

  /**
   * Adds the series of a fetch from coveredEnd. Samples the block already holds win.
   * Streamed series that a fetched series holds in time leave the block.
   * @param series - The fetch's series in block alignment.
   * @param coveredEnd - The end of the span the fetch answers for.
   * @param coveredCount - The index samples in the block before coveredEnd.
   */
  fill(series: Series[], coveredEnd: TimeStamp, coveredCount: number): void {
    if (this.coveredEnd.equals(this.range.start)) this.fetchedAt = this.now();
    series.forEach((s) => this.insert(s));
    this.coveredEnd = coveredEnd;
    this.coveredCount = coveredCount;
    this._streamed = this._streamed.filter(
      (s) =>
        !this._fetched.some(
          (f) =>
            f.timeRange.start.beforeEq(s.timeRange.start) &&
            f.timeRange.end.afterEq(s.timeRange.end),
        ),
    );
  }

  /**
   * @returns the streamed series whose samples overlap tr, with the live buffer last.
   * The live buffer's span ends at its last stamped write, not at its open end.
   */
  streamedIn(tr: TimeRange): Streamed[] {
    const out = this._streamed
      .filter((s) => s.timeRange.overlapsWith(tr))
      .map((series) => ({ series, range: series.timeRange }));
    const live = this.buffer?.leadingBuffer;
    const range = this.buffer?.dataTimeRange;
    if (live != null && range != null && live.length > 0 && range.overlapsWith(tr))
      out.push({ series: live, range });
    return out;
  }

  private insert(series: Series): void {
    const plan = bounds.buildInsertionPlan(
      this._fetched.map((s) => s.alignmentBounds),
      series.alignmentBounds,
    );
    if (plan == null) return;
    const { removeBefore, removeAfter, insertInto, deleteInBetween } = plan;
    let s = series.slice(removeBefore, series.length - removeAfter);
    if (s.length === 0) return;
    // A slice keeps the untrimmed time range, which overlaps the neighbors it was
    // trimmed against.
    if (removeBefore > 0 || removeAfter > 0) {
      const { start, end } = s.timeRange;
      s = derive(s, {
        timeRange: new TimeRange(
          removeBefore > 0 ? this._fetched[insertInto - 1].timeRange.end : start,
          removeAfter > 0
            ? this._fetched[insertInto + deleteInBetween].timeRange.start
            : end,
        ),
      });
    }
    this._fetched.splice(insertInto, deleteInBetween, s);
  }
}
