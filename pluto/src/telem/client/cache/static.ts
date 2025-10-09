// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import {
  bounds,
  MultiSeries,
  type Series,
  Size,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

import { convertSeriesToSupportedGL } from "@/telem/aether/convertSeries";

export interface DirtyReadResult {
  series: MultiSeries;
  gaps: TimeRange[];
}

export interface CacheGCMetrics {
  purgedSeries: number;
  purgedBytes: Size;
}

export const zeroCacheGCMetrics = (): CacheGCMetrics => ({
  purgedSeries: 0,
  purgedBytes: Size.bytes(0),
});

/** Props for the @link Static cache. */
export interface StaticProps {
  /** Used for logging */
  instrumentation?: alamos.Instrumentation;
  /**
   * Sets the amount of time that a cache entry must be in the cache before it can
   * be marked as stale and subject to garbage collection.
   * @default TimeSpan.seconds(20) */
  staleEntryThreshold?: TimeSpan;
}

export const DEFAULT_STATIC_PROPS: Required<StaticProps> = {
  instrumentation: alamos.NOOP,
  staleEntryThreshold: TimeSpan.seconds(20),
};

interface CacheEntry {
  data: Series;
  addedAt: TimeStamp;
}

/**
 * A cache for historical channel data that will not be modified after it is written.
 */
export class Static {
  private data: CacheEntry[] = [];
  private readonly props: Required<StaticProps>;

  constructor(props: StaticProps) {
    this.props = { ...DEFAULT_STATIC_PROPS, ...props };
  }

  /**
   * Writes the given series to the cache, merging written series with any
   * existing series in the cache.
   */
  write(series: MultiSeries): void {
    if (series.length === 0) return;
    series.series.forEach((s) => this.writeOne(convertSeriesToSupportedGL(s)));
    this.checkIntegrity(series);
  }

  /**
   * Executes a 'dirty' read of the cache, retrieving any series in the cache that overlap
   * with the given time range. Note that these series may have data that is before or
   * after the given time range.
   *
   * @param tr - The time range to read from the cache.
   * @returns A list of series that overlap with the given time range and a list of gaps,
   * representing the missing regions of time between the series and before and after
   * the first and last series.
   */
  dirtyRead(tr: TimeRange): DirtyReadResult {
    const series = this.data
      .filter(({ data }) => data.timeRange.overlapsWith(tr))
      .map(({ data }) => data);
    if (series.length === 0) return { series: new MultiSeries([]), gaps: [tr] };
    const gaps = series
      .map((s, i) => {
        if (i === 0) return TimeRange.ZERO;
        return new TimeRange(series[i - 1].timeRange.end, s.timeRange.start);
      })
      .filter((t) => !t.span.isZero && t.isValid);
    const leadingGap = new TimeRange(tr.start, series[0].timeRange.start);
    const trailingGap = new TimeRange(series[series.length - 1].timeRange.end, tr.end);
    if (leadingGap.isValid && !leadingGap.span.isZero) gaps.unshift(leadingGap);
    if (trailingGap.isValid && !trailingGap.span.isZero) gaps.push(trailingGap);
    return { series: new MultiSeries(series), gaps };
  }

  /**
   * Garbage collects the cache, removing any stale entries.
   *
   * @returns metrics about the garbage collection.
   */
  gc(): CacheGCMetrics {
    const { staleEntryThreshold } = this.props;
    const res = zeroCacheGCMetrics();
    const newData = this.data.filter((s) => {
      // Keep entries that have a ref count that is greater than 0 or were just read.
      const shouldKeep =
        s.data.refCount > 0 || TimeStamp.since(s.addedAt).lessThan(staleEntryThreshold);
      if (!shouldKeep) res.purgedBytes = res.purgedBytes.add(s.data.byteCapacity);
      return shouldKeep;
    });
    res.purgedSeries = this.data.length - newData.length;
    this.data = newData;
    return res;
  }

  /**
   * Closes the cache, freeing all of its resources.
   */
  close(): void {
    this.data = [];
  }

  private writeOne(series: Series): void {
    const {
      instrumentation: { L },
    } = this.props;
    if (series.length === 0) return;
    const insertionPlan = bounds.buildInsertionPlan(
      this.data.map((s) => s.data.alignmentBounds),
      series.alignmentBounds,
    );
    if (insertionPlan === null)
      return L.debug("Found no viable insertion plan", {
        inserting: series.digest,
        cacheContents: this.data.map((s) => s.data.digest),
      });
    const { removeBefore, removeAfter, insertInto, deleteInBetween } = insertionPlan;
    series = series.slice(removeBefore, series.data.length - removeAfter);
    // This means we executed a redundant read.
    if (series.length === 0) return;
    this.data.splice(insertInto, deleteInBetween, {
      data: series,
      addedAt: TimeStamp.now(),
    });
  }

  private checkIntegrity(write: MultiSeries): void {
    const {
      instrumentation: { L },
    } = this.props;
    const allBounds = this.data.map((s) => s.data.alignmentBounds);
    const invalid = allBounds.some((b, i) =>
      allBounds.some((b2, j) => {
        if (i === j) return false;
        const ok = bounds.overlapsWith(b, b2);
        return ok;
      }),
    );
    if (invalid) {
      L.debug("Cache is in an invalid state - bounds overlap!", () => ({
        write: write.series.map((s) => s.digest),
        cacheContents: this.data.map((s) => s.data.digest),
      }));
      throw new Error("Invalid state");
    }
  }
}
