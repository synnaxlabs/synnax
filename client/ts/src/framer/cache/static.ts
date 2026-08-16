// Copyright 2026 Synnax Labs, Inc.
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

import { IDENTITY_TRANSFORM, type Transform } from "@/framer/cache/transform";

export interface DirtyReadResult {
  series: MultiSeries;
  gaps: TimeRange[];
}

export interface GCMetrics {
  purgedSeries: number;
  purgedBytes: Size;
}

export const zeroGCMetrics = (): GCMetrics => ({
  purgedSeries: 0,
  purgedBytes: Size.bytes(0),
});

/** Props for the {@link Static} cache. */
export interface StaticProps {
  /** Used for logging */
  instrumentation?: alamos.Instrumentation;
  /** Applied to every series before it enters the cache. Defaults to identity. */
  transform?: Transform;
  /**
   * Sets the amount of time that a cache entry must be in the cache before it can
   * be marked as stale and subject to garbage collection.
   * @default TimeSpan.seconds(20) */
  staleEntryThreshold?: TimeSpan;
}

export const DEFAULT_STATIC_PROPS: Required<StaticProps> = {
  instrumentation: alamos.NOOP,
  transform: IDENTITY_TRANSFORM,
  staleEntryThreshold: TimeSpan.seconds(20),
};

interface CacheEntry {
  /** The cached samples, after the transform and any insertion trimming. */
  data: Series;
  /** When the entry entered the cache, for garbage collection staleness. */
  addedAt: TimeStamp;
  /** True when the entry came from the live stream instead of a fetch. */
  streamed: boolean;
}

/**
 * A cache for historical channel data that will not be modified after it is written.
 */
export class Static {
  private data: CacheEntry[] = [];
  private readonly props: Required<StaticProps>;

  constructor(props: StaticProps) {
    const {
      instrumentation = DEFAULT_STATIC_PROPS.instrumentation,
      transform = DEFAULT_STATIC_PROPS.transform,
      staleEntryThreshold = DEFAULT_STATIC_PROPS.staleEntryThreshold,
    } = props;
    this.props = { instrumentation, transform, staleEntryThreshold };
  }

  /**
   * Writes the given series to the cache, merging written series with any
   * existing series in the cache.
   *
   * @param series - The series to write.
   * @param streamed - Marks the series as live-streamed data. A fetched write
   * evicts every streamed entry whose time range it fully covers.
   */
  write(series: MultiSeries, streamed: boolean = false): void {
    if (series.length === 0) return;
    if (!streamed) this.evictStreamed(series);
    series.series.forEach((s) =>
      this.writeOne(this.props.transform.convert(s), streamed),
    );
    this.repairIntegrity(series);
  }

  // Containment, not overlap: a fetch stamped wider than the data it returned
  // (an uncommitted tail) must not evict streamed samples it did not replace.
  private evictStreamed(written: MultiSeries): void {
    this.data = this.data.filter(
      (e) =>
        !e.streamed ||
        !written.series.some(
          (s) =>
            s.timeRange.start.beforeEq(e.data.timeRange.start) &&
            s.timeRange.end.afterEq(e.data.timeRange.end),
        ),
    );
  }

  /**
   * Executes a 'dirty' read of the cache, retrieving any series in the cache that
   * overlap with the given time range. The series may extend before or after the
   * range.
   *
   * Gaps are computed against fetched entries only. Streamed entries carry
   * provisional leading alignments that cannot pair with fetched data on another
   * channel, so they never claim coverage; the fetch they provoke evicts them.
   *
   * @param tr - The time range to read from the cache.
   * @returns A list of series that overlap with the given time range and a list of
   * gaps, representing the regions of time the fetched series do not cover.
   */
  dirtyRead(tr: TimeRange): DirtyReadResult {
    const series: Series[] = [];
    const fetched: Series[] = [];
    for (const { data, streamed } of this.data)
      if (data.timeRange.overlapsWith(tr)) {
        series.push(data);
        if (!streamed) fetched.push(data);
      }
    if (fetched.length === 0) return { series: new MultiSeries(series), gaps: [tr] };
    const gaps: TimeRange[] = [];
    const pushGap = (start: TimeStamp, end: TimeStamp): void => {
      const gap = new TimeRange(start, end);
      if (gap.isValid && !gap.span.isZero) gaps.push(gap);
    };
    pushGap(tr.start, fetched[0].timeRange.start);
    for (let i = 1; i < fetched.length; i++)
      pushGap(fetched[i - 1].timeRange.end, fetched[i].timeRange.start);
    pushGap(fetched[fetched.length - 1].timeRange.end, tr.end);
    return { series: new MultiSeries(series), gaps };
  }

  /**
   * Garbage collects the cache, removing any stale entries.
   *
   * @returns metrics about the garbage collection.
   */
  gc(): GCMetrics {
    const { staleEntryThreshold } = this.props;
    const res = zeroGCMetrics();
    const newData = this.data.filter((s) => {
      const shouldKeep =
        s.data.refCount > 0 || TimeStamp.since(s.addedAt).lessThan(staleEntryThreshold);
      if (!shouldKeep) res.purgedBytes = res.purgedBytes.add(s.data.byteCapacity);
      return shouldKeep;
    });
    res.purgedSeries = this.data.length - newData.length;
    this.data = newData;
    return res;
  }

  /** Closes the cache, freeing all of its resources. */
  close(): void {
    this.data = [];
  }

  private writeOne(series: Series, streamed: boolean): void {
    const {
      instrumentation: { L },
    } = this.props;
    if (series.length === 0) return;
    const insertionPlan = bounds.buildInsertionPlan(
      this.data.map((s) => s.data.alignmentBounds),
      series.alignmentBounds,
    );
    if (insertionPlan === null)
      return L.debug("Found no viable insertion plan", () => ({
        inserting: series.digest,
        cacheContents: this.data.map((s) => s.data.digest),
      }));
    const { removeBefore, removeAfter, insertInto, deleteInBetween } = insertionPlan;
    series = series.slice(removeBefore, series.length - removeAfter);
    if (series.length === 0) return;
    this.data.splice(insertInto, deleteInBetween, {
      data: series,
      addedAt: TimeStamp.now(),
      streamed,
    });
  }

  private repairIntegrity(write: MultiSeries): void {
    const {
      instrumentation: { L },
    } = this.props;
    // Entries are ordered and non-overlapping, so neighbor comparison suffices. An
    // overlap is an insertion bug: evict both entries and let a refetch heal the gap.
    for (let i = 1; i < this.data.length; i++) {
      if (
        !bounds.overlapsWith(
          this.data[i - 1].data.alignmentBounds,
          this.data[i].data.alignmentBounds,
        )
      )
        continue;
      const [prev, curr] = this.data.splice(i - 1, 2);
      L.error("cache integrity violation - evicting overlapping entries", () => ({
        write: write.series.map((s) => s.digest),
        evicted: [prev.data.digest, curr.data.digest],
      }));
      i = 0;
    }
  }
}
