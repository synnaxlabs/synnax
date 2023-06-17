// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, Series } from "@synnaxlabs/x";

/**
 * A cache for channel data that only accepts pre-written arrays i.e. it performs
 * no allocatio, buffering, or modification of new arrays.
 */
export class StaticCache {
  private readonly entries: CachedRead[];

  constructor() {
    this.entries = [];
  }

  /**
   * @returns the total time range of all entries in the cache.
   */
  get extent(): TimeRange {
    if (this.entries.length === 0) return TimeRange.ZERO;
    const first = this.entries[0].timeRange;
    const last = this.entries[this.entries.length - 1].timeRange;
    return new TimeRange(first.start, last.end);
  }

  /**
   * @returns a list of all gaps between cache reads.
   */
  get gaps(): TimeRange[] {
    return this.entries.map((r) => r.gap);
  }

  write(tr: TimeRange, entries: Series[]): void {
    const read = new CachedRead(tr, entries);
    const i = this.getInsertionIndex(tr);
    if (i !== this.entries.length) {
      read.gap = new TimeRange(this.entries[i].timeRange.end, tr.end);
    }
    if (i !== 0) {
      const prev = this.entries[i - 1];
      prev.gap = new TimeRange(prev.timeRange.end, tr.start);
    }
    this.entries.splice(i, 0, new CachedRead(tr, entries));
  }

  private getInsertionIndex(tr: TimeRange): number {
    let i = 0;
    while (i < this.entries.length && this.entries[i].timeRange.start < tr.start) i++;
    return i;
  }

  dirtyRead(tr: TimeRange): [Series[], TimeRange[]] {
    const reads = this.entries.filter((r) => r.timeRange.overlapsWith(tr));
    if (reads.length === 0) return [[], [tr]];
    const gaps = reads
      .map((r) => r.gap)
      .filter((t, i) => i !== reads.length - 1 && !t.isZero);
    const leadingGap = new TimeRange(tr.start, reads[0].timeRange.start);
    const trailingGap = new TimeRange(reads[reads.length - 1].timeRange.end, tr.end);
    if (leadingGap.isValid && !leadingGap.isZero) gaps.unshift(leadingGap);
    if (trailingGap.isValid && !trailingGap.isZero) gaps.push(trailingGap);
    return [
      reads.flatMap((r) => r.data).filter((d) => d.timeRange.overlapsWith(tr)),
      gaps,
    ];
  }
}

class CachedRead {
  timeRange: TimeRange;
  data: Series[];
  gap: TimeRange;

  constructor(timeRange: TimeRange, data: Series[]) {
    this.timeRange = timeRange;
    this.data = data;
    this.gap = TimeRange.ZERO;
  }
}
