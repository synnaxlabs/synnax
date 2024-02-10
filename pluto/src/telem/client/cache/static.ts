// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  TimeRange,
  type Series,
  type Destructor,
  TimeStamp,
  bounds,
} from "@synnaxlabs/x";
import { Mutex } from "async-mutex";

import { convertSeriesFloat32 } from "@/telem/aether/convertSeries";

export interface DirtyReadResult {
  series: Series[];
  gaps: TimeRange[];
}

export interface DirtyReadForWriteResult {
  series: Series[];
  gaps: TimeRange[];
  done: Destructor;
}

class StativV2 {
  private readonly mu = new Mutex();
  private readonly series: Series[];

  write(series: Series[]): void {}

  private writeOne(series: Series): void {
    // Find the first series whose alignment bounds overlap with or are less than
    // the bounds of the series we're writing.
    const i = this.series.findIndex(
      (s) =>
        bounds.overlapsWith(s.alignmentBounds, series.alignmentBounds) ||
        s.alignmentBounds.upper <= series.alignmentBounds.upper,
    );
  }
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
    if (series.length === 0) return; // This is a condition where we can
    const boundedTR = new TimeRange(
      series[0].timeRange.start,
      series[series.length - 1].timeRange.end,
    );
    if (
      this.entries.some(
        (s) => s.timeRange.overlapsWith(boundedTR) && !boundedTR.contains(s.timeRange),
      )
    )
      return;

    // Check if this write completely contains any cached reads. If so, we should remove
    // those reads and replace them with the fresher data.
    this.spliceOutContainedReads(boundedTR);

    // Convert all series to float32 so they can be used by WebGL.
    series = series.map((s) => convertSeriesFloat32(s));
    const read = new CachedRead(boundedTR, series);
    // Figure out where we need to insert the cached read.
    const i = this.getInsertionIndex(boundedTR);
    // If we're not putting the read at the end of the list, we need to set the gap
    // between the read we're inserting and the next read.
    if (i !== this.entries.length)
      read.gap = new TimeRange(this.entries[i].timeRange.end, boundedTR.end);
    // Similar story for the previous read.
    if (i !== 0) {
      const prev = this.entries[i - 1];
      prev.gap = new TimeRange(prev.timeRange.end, boundedTR.start);
    }
    this.entries.splice(i, 0, read);
  }

  private spliceOutContainedReads(tr: TimeRange): void {
    let found = false;
    this.entries.forEach((r, i) => {
      if (r.timeRange.contains(tr)) {
        this.entries.splice(i, 1);
        found = true;
      }
    });
    if (found) this.spliceOutContainedReads(tr);
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
    // Find any reads we've already executed that overlap with the requested time range.
    const reads = this.entries.filter((r) => r.timeRange.overlapsWith(tr));
    // This means we need to read everything from the database.
    if (reads.length === 0) return { series: [], gaps: [tr] };
    // These are
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
  // This tracks the time range that the read was executed for i.e. the time range
  // provided by the caller.
  timeRange: TimeRange;
  // The timestamp at which this read was executed.
  execTime: TimeStamp;
  // The data that was available from the database at the time of the read.
  data: Series[];
  // The gap between this read and the next read.
  gap: TimeRange;

  constructor(timeRange: TimeRange, data: Series[]) {
    this.timeRange = timeRange;
    this.data = data;
    this.gap = TimeRange.ZERO;
    this.execTime = TimeStamp.now();
  }

  hasInternalGaps(): boolean {
    const internalGaps = this.data.map((d, i) => {
      if (i === 0) return TimeRange.ZERO;
      return new TimeRange(this.data[i - 1].timeRange.end, d.timeRange.start);
    });
    return internalGaps.some((g) => !g.isZero);
  }
}
