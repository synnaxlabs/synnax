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
}

interface CoveredRange {
  /** The fetched span, held even when the fetch returned no samples. */
  range: TimeRange;
  /** When the fetch completed, for garbage collection staleness. */
  addedAt: TimeStamp;
}

/**
 * A cache for historical channel data that will not be modified after it is written.
 * Fetched and streamed entries are held apart because they measure position in
 * different spaces: a fetched sample carries its committed alignment, the same sample
 * streamed carries a provisional leading one. Insertion and gap computation both assume
 * a single space, so each kind gets its own list.
 */
export class Static {
  private fetched: CacheEntry[] = [];
  private streamed: CacheEntry[] = [];
  // Sorted, non-overlapping spans already fetched, kept apart from the entries so a
  // span that returned no samples still counts as answered and is not refetched.
  private covered: CoveredRange[] = [];
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
   * Writes the given series to the cache, merging written series with any existing
   * series in the cache.
   * @param streamed - Marks the series as live-streamed data. A fetched write evicts
   * every streamed entry whose time range it fully covers.
   */
  write(series: MultiSeries, streamed: boolean = false): void {
    if (series.length === 0) return;
    if (!streamed) this.evictStreamed(series);
    const entries = streamed ? this.streamed : this.fetched;
    series.series.forEach((s) =>
      this.writeOne(this.props.transform.convert(s), entries),
    );
    this.repairIntegrity(series, entries);
  }

  /**
   * Records tr as fetched, merging it into the covered set. Gap computation treats
   * covered spans as answered even when the fetch returned no samples, so an empty
   * span is fetched once per staleness window instead of on every read.
   */
  markFetched(tr: TimeRange): void {
    if (!tr.isValid || tr.span.isZero) return;
    let { start, end } = tr;
    const keep: CoveredRange[] = [];
    for (const c of this.covered)
      if (c.range.end.before(start) || c.range.start.after(end)) keep.push(c);
      else {
        if (c.range.start.before(start)) start = c.range.start;
        if (c.range.end.after(end)) end = c.range.end;
      }
    keep.push({ range: new TimeRange(start, end), addedAt: TimeStamp.now() });
    keep.sort((a, b) => TimeRange.sort(a.range, b.range));
    this.covered = keep;
  }

  // Containment, not overlap: a fetch stamped wider than the data it returned
  // (an uncommitted tail) must not evict streamed samples it did not replace.
  private evictStreamed(written: MultiSeries): void {
    this.streamed = this.streamed.filter(
      (e) =>
        !written.series.some(
          (s) =>
            s.timeRange.start.beforeEq(e.data.timeRange.start) &&
            s.timeRange.end.afterEq(e.data.timeRange.end),
        ),
    );
  }

  /**
   * Executes a 'dirty' read of the cache, retrieving any series in the cache that
   * overlap with the given time range. The series may extend before or after the range.
   * Gaps are computed against fetched entries only. Streamed entries carry provisional
   * leading alignments that cannot pair with fetched data on another channel, so they
   * never claim coverage; the fetch they provoke evicts them.
   * @returns A list of series that overlap with the given time range and a list of
   * gaps, representing the regions of time the fetched series do not cover.
   */
  dirtyRead(tr: TimeRange): DirtyReadResult {
    const overlapping = (entries: CacheEntry[]): Series[] =>
      entries.filter((e) => e.data.timeRange.overlapsWith(tr)).map((e) => e.data);
    const fetched = overlapping(this.fetched);
    const series = [...fetched, ...overlapping(this.streamed)];
    if (fetched.length === 0)
      return { series: new MultiSeries(series), gaps: this.subtractCovered(tr) };
    const gaps: TimeRange[] = [];
    const pushGap = (start: TimeStamp, end: TimeStamp): void => {
      const gap = new TimeRange(start, end);
      if (gap.isValid && !gap.span.isZero) gaps.push(...this.subtractCovered(gap));
    };
    pushGap(tr.start, fetched[0].timeRange.start);
    for (let i = 1; i < fetched.length; i++)
      pushGap(fetched[i - 1].timeRange.end, fetched[i].timeRange.start);
    pushGap(fetched[fetched.length - 1].timeRange.end, tr.end);
    return { series: new MultiSeries(series), gaps };
  }

  // Removes the covered portions of gap, returning the still-unanswered remainder.
  private subtractCovered(gap: TimeRange): TimeRange[] {
    let remaining = [gap];
    for (const { range } of this.covered) {
      const next: TimeRange[] = [];
      for (const r of remaining) {
        if (!range.overlapsWith(r)) {
          next.push(r);
          continue;
        }
        const before = new TimeRange(r.start, range.start);
        const after = new TimeRange(range.end, r.end);
        if (before.isValid && !before.span.isZero) next.push(before);
        if (after.isValid && !after.span.isZero) next.push(after);
      }
      remaining = next;
      if (remaining.length === 0) break;
    }
    return remaining;
  }

  /**
   * Garbage collects the cache, removing any stale entries.
   * @returns metrics about the garbage collection.
   */
  gc(): GCMetrics {
    const { staleEntryThreshold } = this.props;
    const res = zeroGCMetrics();
    const collect = (entries: CacheEntry[]): CacheEntry[] =>
      entries.filter((s) => {
        const shouldKeep =
          s.data.refCount > 0 ||
          TimeStamp.since(s.addedAt).lessThan(staleEntryThreshold);
        if (!shouldKeep) {
          res.purgedBytes = res.purgedBytes.add(s.data.byteCapacity);
          res.purgedSeries++;
        }
        return shouldKeep;
      });
    this.fetched = collect(this.fetched);
    this.streamed = collect(this.streamed);
    // Expiring coverage re-opens the span to one fetch per staleness window, so data
    // committed late (backfill, an uncommitted tail) still gets picked up.
    this.covered = this.covered.filter((c) =>
      TimeStamp.since(c.addedAt).lessThan(staleEntryThreshold),
    );
    return res;
  }

  /** Closes the cache, freeing all of its resources. */
  close(): void {
    this.fetched = [];
    this.streamed = [];
    this.covered = [];
  }

  private writeOne(series: Series, entries: CacheEntry[]): void {
    const {
      instrumentation: { L },
    } = this.props;
    if (series.length === 0) return;
    const insertionPlan = bounds.buildInsertionPlan(
      entries.map((s) => s.data.alignmentBounds),
      series.alignmentBounds,
    );
    if (insertionPlan === null)
      return L.debug("Found no viable insertion plan", () => ({
        inserting: series.digest,
        cacheContents: entries.map((s) => s.data.digest),
      }));
    const { removeBefore, removeAfter, insertInto, deleteInBetween } = insertionPlan;
    series = series.slice(removeBefore, series.length - removeAfter);
    if (series.length === 0) return;
    entries.splice(insertInto, deleteInBetween, {
      data: series,
      addedAt: TimeStamp.now(),
    });
  }

  private repairIntegrity(write: MultiSeries, entries: CacheEntry[]): void {
    const {
      instrumentation: { L },
    } = this.props;
    // Entries are ordered and non-overlapping, so neighbor comparison suffices. An
    // overlap is an insertion bug: evict both entries and let a refetch heal the gap.
    for (let i = 1; i < entries.length; i++) {
      if (
        !bounds.overlapsWith(
          entries[i - 1].data.alignmentBounds,
          entries[i].data.alignmentBounds,
        )
      )
        continue;
      const [prev, curr] = entries.splice(i - 1, 2);
      L.error("cache integrity violation - evicting overlapping entries", () => ({
        write: write.series.map((s) => s.digest),
        evicted: [prev.data.digest, curr.data.digest],
      }));
      i = 0;
    }
  }
}
