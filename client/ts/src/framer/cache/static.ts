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
  type DataType,
  MultiSeries,
  Series,
  Size,
  TimeRange,
  TimeSpan,
  TimeStamp,
} from "@synnaxlabs/x";

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
  /**
   * Sets the amount of time that a cache entry must be in the cache before it can
   * be marked as stale and subject to garbage collection.
   * @default TimeSpan.seconds(20) */
  staleEntryThreshold?: TimeSpan;
  /**
   * Sets how long a fetched span counts as answered before it is refetched. Bounds
   * the refetch rate of spans that stay empty, as every refetch opens a short-lived
   * iterator connection.
   * @default TimeSpan.minutes(10) */
  staleCoverageThreshold?: TimeSpan;
}

export const DEFAULT_STATIC_PROPS: Required<StaticProps> = {
  instrumentation: alamos.NOOP,
  staleEntryThreshold: TimeSpan.seconds(20),
  staleCoverageThreshold: TimeSpan.minutes(10),
};

interface CacheEntry {
  /** The cached samples, after any insertion trimming. */
  data: Series;
  /** When the entry entered the cache, for garbage collection staleness. */
  addedAt: TimeStamp;
}

interface CoveredRange {
  /** The fetched span, held even when the fetch returned no samples. */
  range: TimeRange;
  /** When the fetch completed. Coverage answers gaps until it goes stale. */
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
      staleEntryThreshold = DEFAULT_STATIC_PROPS.staleEntryThreshold,
      staleCoverageThreshold = DEFAULT_STATIC_PROPS.staleCoverageThreshold,
    } = props;
    this.props = {
      instrumentation,
      staleEntryThreshold,
      staleCoverageThreshold,
    };
  }

  /**
   * Writes the given series to the cache, merging written series with any existing
   * series in the cache.
   * @param streamed - Marks the series as live-streamed data. A fetched write evicts
   * every streamed entry whose time range it fully covers.
   */
  write(series: MultiSeries, streamed: boolean = false): void {
    if (series.length === 0) return;
    this.retainDataType(series.series[0].dataType);
    if (!streamed) this.evictStreamed(series);
    const entries = streamed ? this.streamed : this.fetched;
    series.series.forEach((s) => this.writeOne(s, entries));
    this.repairIntegrity(series, entries);
  }

  /**
   * Drops every entry and coverage record when the cache holds a data type other than
   * dt. A channel's data type can change, and series of two types cannot be read
   * together.
   */
  retainDataType(dt: DataType): void {
    const held = this.dataType;
    if (held == null || held.equals(dt)) return;
    this.fetched = [];
    this.streamed = [];
    this.covered = [];
  }

  /**
   * Records tr as fetched, merging it into the covered set. A covered span counts
   * as answered even when empty, so it is refetched once it goes stale, not per read.
   */
  markFetched(tr: TimeRange): void {
    if (!tr.isValid || tr.span.isZero) return;
    let { start, end } = tr;
    // Live merges keep the oldest stamp so a rolling read cannot refresh its own
    // coverage forever. Stale records drop, so a refetch restarts their clock.
    let addedAt = TimeStamp.now();
    const keep: CoveredRange[] = [];
    for (const c of this.covered) {
      if (!this.isLive(c)) continue;
      if (c.range.end.before(start) || c.range.start.after(end)) keep.push(c);
      else {
        if (c.range.start.before(start)) start = c.range.start;
        if (c.range.end.after(end)) end = c.range.end;
        if (c.addedAt.before(addedAt)) addedAt = c.addedAt;
      }
    }
    keep.push({ range: new TimeRange(start, end), addedAt });
    keep.sort((a, b) => TimeRange.sort(a.range, b.range));
    this.covered = keep;
  }

  // Coverage answers gaps only until it goes stale. gc merely prunes dead records.
  private isLive(c: CoveredRange): boolean {
    return TimeStamp.since(c.addedAt).lessThan(this.props.staleCoverageThreshold);
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
   * never claim coverage; the fetch they provoke evicts them. A streamed entry that
   * overlaps a fetched one is left out: consumers order series by alignment, so the
   * two forms of one span would join backwards in time.
   * @returns A list of series that overlap with the given time range and a list of
   * gaps, representing the regions of time the fetched series do not cover.
   */
  dirtyRead(tr: TimeRange): DirtyReadResult {
    const overlapping = (entries: CacheEntry[]): Series[] =>
      entries.filter((e) => e.data.timeRange.overlapsWith(tr)).map((e) => e.data);
    const fetched = overlapping(this.fetched);
    const streamed = overlapping(this.streamed).filter(
      (s) => !this.overlapsFetched(s.timeRange),
    );
    const series = [...fetched, ...streamed];
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

  /** @returns the data type of the held series, or null when the cache is empty. */
  get dataType(): DataType | null {
    return (this.fetched[0]?.data ?? this.streamed[0]?.data)?.dataType ?? null;
  }

  /** @returns true when a fetched entry overlaps the given time range. */
  overlapsFetched(tr: TimeRange): boolean {
    // Fetched entries are sorted and disjoint in time, so only the first one ending
    // after tr starts can overlap it.
    let lo = 0;
    let hi = this.fetched.length;
    while (lo < hi) {
      const mid = (lo + hi) >>> 1;
      if (this.fetched[mid].data.timeRange.end.beforeEq(tr.start)) lo = mid + 1;
      else hi = mid;
    }
    return lo < this.fetched.length && this.fetched[lo].data.timeRange.overlapsWith(tr);
  }

  // Removes the covered portions of gap, returning the still-unanswered remainder.
  private subtractCovered(gap: TimeRange): TimeRange[] {
    let remaining = [gap];
    for (const c of this.covered) {
      if (!this.isLive(c)) continue;
      const { range } = c;
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

  // Removes tr from the covered set, splitting any record it falls inside.
  private uncover(tr: TimeRange): void {
    this.covered = this.covered.flatMap((c) => {
      if (!c.range.overlapsWith(tr)) return [c];
      return [
        new TimeRange(c.range.start, tr.start),
        new TimeRange(tr.end, c.range.end),
      ]
        .filter((r) => r.isValid && !r.span.isZero)
        .map((range) => ({ range, addedAt: c.addedAt }));
    });
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
    const fetched = collect(this.fetched);
    // Purged samples take their coverage with them, or reads would skip their span.
    if (fetched.length < this.fetched.length) {
      const kept = new Set(fetched);
      this.fetched.forEach((e) => {
        if (!kept.has(e)) this.uncover(e.data.timeRange);
      });
    }
    this.fetched = fetched;
    this.streamed = collect(this.streamed);
    // The client cannot tell "no data yet" from "no data ever" (backfill is legal),
    // so stale coverage re-opens an empty span to one cheap fetch per staleness
    // window instead of one per read (space-heater mode). Series purging above is not
    // affected.
    this.covered = this.covered.filter((c) => this.isLive(c));
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
    const entry = { data: series, addedAt: TimeStamp.now() };
    entries.splice(insertInto, deleteInBetween, entry);
    if (removeBefore === 0 && removeAfter === 0) return;
    // A slice keeps the untrimmed time range, which would hide the trimmed span from
    // gap computation once the neighbor it was trimmed against is gone.
    const { start, end } = series.timeRange;
    entry.data = new Series({
      data: series.data,
      dataType: series.dataType,
      timeRange: new TimeRange(
        removeBefore > 0 ? entries[insertInto - 1].data.timeRange.end : start,
        removeAfter > 0 ? entries[insertInto + 1].data.timeRange.start : end,
      ),
      sampleOffset: series.sampleOffset,
      alignment: series.alignment,
      alignmentMultiple: series.alignmentMultiple,
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
