// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Series, TimeStamp } from "@synnaxlabs/x";

/**
 * The stamps of the last writes to a channel with no index, one record per write. Every
 * sample of a write shares the stamp the Core gave the write. At capacity the oldest
 * write is forgotten.
 */
export class Stamps {
  private readonly starts: BigUint64Array;
  private readonly ends: BigUint64Array;
  private readonly stamps: BigInt64Array;
  private head = 0;
  private length = 0;

  constructor(capacity: number) {
    this.starts = new BigUint64Array(capacity);
    this.ends = new BigUint64Array(capacity);
    this.stamps = new BigInt64Array(capacity);
  }

  /**
   * Records the write the series carries. A series with no samples or no time range
   * records nothing. A write that starts at or before a recorded one replaces the
   * records from there on.
   */
  record(series: Series): void {
    if (series.length === 0 || series.timeRange.isZero) return;
    const { lower, upper } = series.alignmentBounds;
    while (this.length > 0 && this.startAt(this.length - 1) >= lower) this.length--;
    const { capacity } = this;
    if (this.length === capacity) {
      this.head = (this.head + 1) % capacity;
      this.length--;
    }
    const i = this.slot(this.length);
    this.starts[i] = lower;
    this.ends[i] = upper;
    this.stamps[i] = series.timeRange.start.valueOf();
    this.length++;
  }

  /**
   * Records a fetched write unless a recorded write already reaches its start. A fetch
   * can land behind newer live writes, which keep their records.
   */
  recordFetched(series: Series): void {
    const { lower } = series.alignmentBounds;
    if (this.length > 0 && this.endAt(this.length - 1) > lower) return;
    this.record(series);
  }

  /** @returns the stamp of the write holding the sample at alignment, else null. */
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
    if (alignment >= this.ends[i]) return null;
    return new TimeStamp(this.stamps[i]);
  }

  private get capacity(): number {
    return this.starts.length;
  }

  private slot(i: number): number {
    return (this.head + i) % this.capacity;
  }

  private startAt(i: number): bigint {
    return this.starts[this.slot(i)];
  }

  private endAt(i: number): bigint {
    return this.ends[this.slot(i)];
  }
}
