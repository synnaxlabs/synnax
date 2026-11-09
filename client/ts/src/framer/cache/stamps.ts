// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type bounds, type Series, TimeStamp } from "@synnaxlabs/x";

const INITIAL_SIZE = 16;

/**
 * The stamps of the last writes to a channel with no index, one record per write. The
 * Core stamps a write's samples 1ns apart. At capacity the oldest write is forgotten.
 */
export class Stamps {
  private starts: BigUint64Array;
  private lengths: Uint32Array;
  private stamps: BigInt64Array;
  private head = 0;
  private length = 0;

  constructor(private readonly capacity: number) {
    const size = Math.min(capacity, INITIAL_SIZE);
    this.starts = new BigUint64Array(size);
    this.lengths = new Uint32Array(size);
    this.stamps = new BigInt64Array(size);
  }

  /**
   * Records the write the series carries. A series with no samples or no time range
   * records nothing. A write that starts at or before a recorded one replaces the
   * records from there on.
   */
  record(series: Series): void {
    const { length, alignment: lower } = series;
    if (length === 0 || series.timeRange.isZero) return;
    while (this.length > 0 && this.startAt(this.length - 1) >= lower) this.length--;
    if (this.length === this.capacity) {
      this.head = (this.head + 1) % this.capacity;
      this.length--;
    } else if (this.length === this.size) this.grow();
    const i = this.slot(this.length);
    this.starts[i] = lower;
    this.lengths[i] = length;
    this.stamps[i] = series.timeRange.start.valueOf();
    this.length++;
  }

  /** @returns the stamp of the sample at alignment, else null. */
  at(alignment: bigint): TimeStamp | null {
    let lo = 0;
    let hi = this.length;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;
      if (this.startAt(mid) <= alignment) lo = mid + 1;
      else hi = mid;
    }
    if (lo === 0) return null;
    const i = this.slot(lo - 1);
    const offset = alignment - this.starts[i];
    if (offset >= BigInt(this.lengths[i])) return null;
    return new TimeStamp(this.stamps[i] + offset);
  }

  // The head only moves once the ring reaches capacity, so it is 0 here.
  private grow(): void {
    const size = Math.min(this.size * 2, this.capacity);
    const starts = new BigUint64Array(size);
    const lengths = new Uint32Array(size);
    const stamps = new BigInt64Array(size);
    starts.set(this.starts);
    lengths.set(this.lengths);
    stamps.set(this.stamps);
    this.starts = starts;
    this.lengths = lengths;
    this.stamps = stamps;
  }

  private get size(): number {
    return this.starts.length;
  }

  private slot(i: number): number {
    return (this.head + i) % this.size;
  }

  private startAt(i: number): bigint {
    return this.starts[this.slot(i)];
  }
}

type Span = bounds.Bounds<bigint>;

const spanOf = (s: Series): Span => s.alignmentBounds;

// Returns the position of the first of the sorted, disjoint items that ends after
// alignment.
const firstEndingAfter = <T>(
  items: T[],
  spanOfItem: (item: T) => Span,
  alignment: bigint,
): number => {
  let lo = 0;
  let hi = items.length;
  while (lo < hi) {
    const mid = (lo + hi) >>> 1;
    if (spanOfItem(items[mid]).upper <= alignment) lo = mid + 1;
    else hi = mid;
  }
  return lo;
};

// Returns the spans sorted, with the ones that touch or overlap joined.
const merge = (spans: Span[]): Span[] => {
  const sorted = [...spans].sort((a, b) => (a.lower < b.lower ? -1 : 1));
  const merged: Span[] = [];
  for (const { lower, upper } of sorted) {
    const last = merged.at(-1);
    if (last == null || lower > last.upper) merged.push({ lower, upper });
    else if (upper > last.upper) last.upper = upper;
  }
  return merged;
};

// Returns true when the span overlaps one of the sorted, disjoint spans.
const overlaps = (spans: Span[], span: Span): boolean => {
  const first = spans[firstEndingAfter(spans, (s) => s, span.lower)];
  return first != null && first.lower < span.upper;
};

/**
 * Holds the index samples that time a data channel's samples. A data sample shares its
 * alignment with the index sample written with it.
 */
export class IndexStamps {
  // series is sorted by alignment, and no two series overlap.
  private series: Series[] = [];

  /**
   * Keeps each index series that overlaps one of the data spans, replacing the series
   * held over its alignments.
   */
  add(index: Series[], data: Span[]): void {
    const spans = merge(data);
    for (const s of index) {
      const span = spanOf(s);
      if (s.length === 0 || !overlaps(spans, span)) continue;
      const from = firstEndingAfter(this.series, spanOf, span.lower);
      let to = from;
      while (to < this.series.length && spanOf(this.series[to]).lower < span.upper)
        to++;
      this.series.splice(from, to - from, s);
    }
  }

  /** @returns the stamp of the index sample at alignment, else null. */
  at(alignment: bigint): TimeStamp | null {
    const s = this.series[firstEndingAfter(this.series, spanOf, alignment)];
    const value = s?.atAlignment(alignment);
    return value == null ? null : new TimeStamp(value as bigint);
  }

  /** Drops every index series that overlaps none of the data spans. */
  retain(data: Span[]): void {
    const spans = merge(data);
    this.series = this.series.filter((s) => overlaps(spans, spanOf(s)));
  }
}
