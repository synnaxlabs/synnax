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
  /** The span of the block's samples. */
  readonly range: TimeRange;
  /** When the oldest fetch the block answers reads with landed. */
  readonly fetchedAt: TimeStamp;
  /** The bytes of the block's samples. */
  readonly bytes: number;
  /** True while the block must stay cached, as when a subscription serves it. */
  readonly held: boolean;
  /**
   * True when the fetch saw the channel's samples reach the block's coverage end.
   * Appends land after the channel's newest sample, so none lands in a sealed block.
   */
  readonly sealed: boolean;
  /** The number of subscriptions that serve the block's series. */
  pins: number;
}

/** @returns the index of the raw block that holds the timestamp. */
export const rawIndex = (ts: TimeStamp): number => tileIndex(ts, RAW_LEVEL);

/** @returns the store key of the channel's raw block at the index. */
export const rawKey = (ch: channel.Key, index: number): string => `${ch}/raw/${index}`;

const OFFSET_MASK = (1n << 32n) - 1n;

/** @returns the alignment of the sample at the offset of the domain. */
export const alignmentOf = (domain: number | bigint, offset: number): bigint =>
  (BigInt(domain) << 32n) | BigInt(offset);

/** @returns the domain of the alignment: a Core domain, or a raw block index. */
export const domainOf = (alignment: bigint): bigint => alignment >> 32n;

/** @returns the sample offset of the alignment in its domain. */
export const offsetOf = (alignment: bigint): number => Number(alignment & OFFSET_MASK);

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
        alignment: alignmentOf(index, seg.offset + Number(lower - seg.start)),
      });
    });
};

/**
 * @returns the position of the first sample of the timestamp series at or after t,
 * searching from lo.
 */
export const search = (s: Series, t: bigint, lo = 0): number => {
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
 * Splits a fetch's series of one channel at raw block edges, and cuts the time range of
 * each part to its block. A series cut in two copies its samples, so a block never
 * holds the buffer of samples another block holds. Samples with no index sample of the
 * same Core alignment stay out, as when the Core commits a channel before its index.
 * @param timestamps - The fetch's series of the channel's index, in time order.
 * @param series - The fetch's series of the channel, in time order.
 * @returns the part of each raw block that holds index samples, by block index, in
 * block order.
 */
export const split = (timestamps: Series[], series: Series[]): Map<number, Part> => {
  const parts = new Map<number, Part>();
  // edges holds the Core alignment bounds of the index samples of each part, in order.
  const edges: Array<{ part: Part; range: TimeRange; lower: bigint; upper: bigint }> =
    [];
  for (const ts of timestamps)
    for (let lo = 0; lo < ts.length;) {
      const index = rawIndex(new TimeStamp(ts.data[lo]));
      const range = tileRange({ level: RAW_LEVEL, index });
      const hi = search(ts, range.end.valueOf(), lo);
      let part = parts.get(index);
      if (part == null) {
        part = { timestamps: [], series: [] };
        parts.set(index, part);
      }
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

/**
 * One channel's raw samples over one level 16 tile of the grid, in block alignment.
 * Fetched series cover the block from its start. While a join writes the stream into
 * the block, the stream's buffer holds the samples after them.
 */
export class RawBlock implements Block {
  readonly range: TimeRange;
  fetchedAt = TimeStamp.ZERO;
  sealed = false;
  pins = 0;
  /** Fetched samples cover [range.start, coveredEnd). */
  coveredEnd: TimeStamp;
  /** The index samples in [range.start, coveredEnd). */
  coveredCount = 0;
  private _fetched: Series[] = [];
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

  get bytes(): number {
    const live = this.buffer?.leadingBuffer?.byteCapacity.valueOf() ?? 0;
    return sumBytes(this._fetched) + live;
  }

  /** True while a join writes the stream's samples into the block. */
  get joined(): boolean {
    return this.buffer != null;
  }

  get held(): boolean {
    return this.pins > 0 || this.buffer != null || this.pending != null;
  }

  /**
   * @returns true when fetched samples cover the part of tr inside the block. A joined
   * block covers every span, since the stream delivers its samples.
   */
  covers(tr: TimeRange): boolean {
    return (
      this.joined || TimeStamp.min(tr.end, this.range.end).beforeEq(this.coveredEnd)
    );
  }

  /**
   * Moves the coverage to at least end, sets the count of index samples it holds, and
   * seals the block. Call it only when the channel's samples reach end.
   */
  extend(end: TimeStamp, count: number): void {
    if (this.coveredEnd.equals(this.range.start)) this.fetchedAt = this.now();
    this.coveredEnd = TimeStamp.max(this.coveredEnd, end);
    this.coveredCount = count;
    this.sealed = true;
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
    this.sealed = false;
    return true;
  }

  /**
   * Adds the series of a fetch from coveredEnd. Samples the block already holds win.
   * @param series - The fetch's series in block alignment.
   * @param coveredEnd - The end of the span the fetch answers for.
   * @param coveredCount - The index samples in the block before coveredEnd.
   */
  fill(series: Series[], coveredEnd: TimeStamp, coveredCount: number): void {
    if (this.coveredEnd.equals(this.range.start)) this.fetchedAt = this.now();
    series.forEach((s) => this.insert(s));
    this.coveredEnd = coveredEnd;
    this.coveredCount = coveredCount;
  }

  /**
   * @returns the series whose samples overlap tr in time order, the stream's buffer
   * last. The buffer's span ends at its last stamped write, not at its open end.
   */
  seriesIn(tr: TimeRange): Series[] {
    const out = this._fetched.filter((s) => s.timeRange.overlapsWith(tr));
    const live = this.buffer?.leadingBuffer;
    const range = this.buffer?.dataTimeRange;
    if (live != null && range != null && live.length > 0 && range.overlapsWith(tr))
      out.push(live);
    return out;
  }

  /**
   * @returns the piece, starting where the block's newest fetched series ends, or at
   * the block's start, as a read starts a run's samples.
   */
  continued(p: Series): Series {
    const start = this._fetched.at(-1)?.timeRange.end ?? this.range.start;
    return derive(p, { timeRange: new TimeRange(start, p.timeRange.end) });
  }

  /** Ends the newest fetched series at the block's end, as a read cuts a run. */
  cutNewest(): void {
    const newest = this._fetched.at(-1);
    if (newest != null) newest.timeRange.end = this.range.end;
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
