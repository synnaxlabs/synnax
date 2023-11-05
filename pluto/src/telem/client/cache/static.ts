// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, type Series, type Destructor } from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { convertSeriesFloat32 } from "@/telem/core/convertSeries";

export interface DirtyReadResult {
  series: Series[];
  gaps: TimeRange[];
}

export interface DirtyReadForWriteResult {
  series: Series[];
  gaps: TimeRange[];
  done: Destructor;
}

/**
 * A cache for channel data that only accepts pre-written arrays i.e. it performs
 * no allocatio, buffering, or modification of new arrays.
 */
export class Static {
  private readonly mu = new Mutex();
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

  write(tr: TimeRange, series: Series[]): void {
    if (series.length === 0) return;
    if (series.length === 0 || this.entries.some((e) => e.timeRange.overlapsWith(tr)))
      return;
    series = series.map((s) => convertSeriesFloat32(s));
    const read = new CachedRead(tr, series);
    const i = this.getInsertionIndex(tr);
    if (i !== this.entries.length) {
      read.gap = new TimeRange(this.entries[i].timeRange.end, tr.end);
    }
    if (i !== 0) {
      const prev = this.entries[i - 1];
      prev.gap = new TimeRange(prev.timeRange.end, tr.start);
    }
    this.entries.splice(i, 0, new CachedRead(tr, series));
  }

  private getInsertionIndex(tr: TimeRange): number {
    let i = 0;
    while (i < this.entries.length && this.entries[i].timeRange.start < tr.start) i++;
    return i;
  }

  async dirtyReadForWrite(tr: TimeRange): Promise<DirtyReadForWriteResult> {
    const done = await this.mu.acquire();
    return {
      ...this.dirtyRead(tr),
      done,
    };
  }

  dirtyRead(tr: TimeRange): DirtyReadResult {
    const reads = this.entries.filter((r) => r.timeRange.overlapsWith(tr));
    if (reads.length === 0) return { series: [], gaps: [tr] };
    const gaps = reads
      .map((r) => r.gap)
      .filter((t, i) => i !== reads.length - 1 && !t.isZero);
    const leadingGap = new TimeRange(tr.start, reads[0].timeRange.start);
    const trailingGap = new TimeRange(reads[reads.length - 1].timeRange.end, tr.end);
    if (leadingGap.isValid && !leadingGap.isZero) gaps.unshift(leadingGap);
    if (trailingGap.isValid && !trailingGap.isZero) gaps.push(trailingGap);
    return {
      series: reads.flatMap((r) => r.data).filter((d) => d.timeRange.overlapsWith(tr)),
      gaps,
    };
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
