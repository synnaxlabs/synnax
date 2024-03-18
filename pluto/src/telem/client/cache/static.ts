// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { TimeRange, type Series, bounds, TimeStamp, TimeSpan } from "@synnaxlabs/x";

import { convertSeriesFloat32 } from "@/telem/aether/convertSeries";

export interface DirtyReadResult {
  series: Series[];
  gaps: TimeRange[];
}

interface CacheEntry {
  data: Series;
  addedAt: TimeStamp;
}

export class Static {
  private data: CacheEntry[] = [];
  private readonly ins: alamos.Instrumentation;

  constructor(ins: alamos.Instrumentation = alamos.NOOP) {
    this.ins = ins;
  }

  write(series: Series[]): void {
    if (series.length === 0) return;
    series.forEach((s) => this.writeOne(convertSeriesFloat32(s)));
    this.checkIntegrity(series);
  }

  private writeOne(series: Series): void {
    if (series.length === 0) return;
    const insertionPlan = bounds.buildInsertionPlan(
      this.data.map((s) => s.data.alignmentBounds),
      series.alignmentBounds,
    );
    if (insertionPlan === null) {
      this.ins.L.debug("Found no viable insertion plan", {
        inserting: series.digest,
        cacheContents: this.data.map((s) => s.data.digest),
      });
      return;
    }
    const { removeBefore, removeAfter, insertInto, deleteInBetween } = insertionPlan;
    series = series.slice(removeBefore, series.data.length - removeAfter);
    // This means we executed a redundant read.
    if (series.length === 0) return;
    this.data.splice(insertInto, deleteInBetween, {
      data: series,
      addedAt: TimeStamp.now(),
    });
  }

  private checkIntegrity(write: Series[]): void {
    const allBounds = this.data.map((s) => s.data.alignmentBounds);
    const invalid = allBounds.some((b, i) => {
      return allBounds.some((b2, j) => {
        if (i === j) return false;
        const ok = bounds.overlapsWith(b, b2);
        return ok;
      });
    });
    if (invalid) {
      this.ins.L.debug("Cache is in an invalid state - bounds overlap!", () => ({
        write: write.map((s) => s.digest),
        cacheContents: this.data.map((s) => s.data.digest),
      }));
      throw new Error("Invalid state");
    }
  }

  dirtyRead(tr: TimeRange): DirtyReadResult {
    const series = this.data
      .filter(({ data }) => data.timeRange.overlapsWith(tr))
      .map(({ data }) => data);
    if (series.length === 0) return { series: [], gaps: [tr] };
    const gaps = series
      .map((s, i) => {
        if (i === 0) return TimeRange.ZERO;
        return new TimeRange(series[i - 1].timeRange.end, s.timeRange.start);
      })
      .filter((t) => !t.isZero && t.isValid);
    const leadingGap = new TimeRange(tr.start, series[0].timeRange.start);
    const trailingGap = new TimeRange(series[series.length - 1].timeRange.end, tr.end);
    if (leadingGap.isValid && !leadingGap.isZero) gaps.unshift(leadingGap);
    if (trailingGap.isValid && !trailingGap.isZero) gaps.push(trailingGap);
    return { series, gaps };
  }

  garbageCollect(): void {
    this.data = this.data.filter(
      (s) =>
        s.data.refCount === 0 &&
        TimeStamp.since(s.addedAt).greaterThan(TimeSpan.seconds(5)),
    );
  }

  close(): void {
    this.data = [];
  }
}
