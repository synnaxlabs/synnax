// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type MultiSeries, type Series, type TimeSpan } from "@synnaxlabs/x";

export const seriesOverlap = (
  x: Series,
  ys: Series,
  overlapThreshold: TimeSpan,
): boolean => {
  if (x.alignmentMultiple !== ys.alignmentMultiple) {
    console.warn(
      "encountered two series with different alignment multiples in draw operations",
      { x: x.digest, y: ys.digest },
    );
    return false;
  }
  // If the time ranges of the x and y series overlap, we meet the first condition
  // for drawing them together. Dynamic buffering can sometimes lead to very slight,
  // unintended overlaps, so we only consider them overlapping if they overlap by a
  // certain threshold.
  const timeRangesOverlap = x.timeRange.overlapsWith(ys.timeRange, overlapThreshold);
  // If the 'indexes' of the x and y series overlap, we meet the second condition
  // for drawing them together.
  const alignmentsOverlap = bounds.overlapsWith(x.alignmentBounds, ys.alignmentBounds);
  return timeRangesOverlap && alignmentsOverlap;
};

const BLOCK_SIZE = 4096;

interface BlockSummary {
  mins: number[];
  maxs: number[];
}

/**
 * @returns the y index range [lo, hi) of samples whose paired x value falls inside
 * the window, or null when the pair shares no samples there.
 */
const clip = (
  x: Series,
  y: Series,
  xWindow: bounds.Bounds,
  overlapThreshold: TimeSpan,
): [number, number] | null => {
  if (y.dataType.isVariable || !seriesOverlap(x, y, overlapThreshold)) return null;
  const mult = x.alignmentMultiple;
  const start = x.alignment > y.alignment ? x.alignment : y.alignment;
  const xUpper = x.alignmentBounds.upper;
  const yUpper = y.alignmentBounds.upper;
  const end = xUpper < yUpper ? xUpper : yUpper;
  if (end <= start) return null;
  const xLo = x.binarySearch(xWindow.lower);
  let xHi = x.binarySearch(xWindow.upper);
  if (xHi < x.length && Number(x.at(xHi, true)) === xWindow.upper) xHi++;
  const winStart = x.alignment + BigInt(xLo) * mult;
  const winEnd = x.alignment + BigInt(xHi) * mult;
  const s = winStart > start ? winStart : start;
  const e = winEnd < end ? winEnd : end;
  if (e <= s) return null;
  return [Number((s - y.alignment) / mult), Number((e - y.alignment) / mult)];
};

/**
 * Computes the min/max of y samples whose paired x value falls inside a window.
 * Summaries are cached per completed block of samples; series are append-only, so a
 * cached block never invalidates and the still-filling tail is always scanned raw.
 */
export class BoundsCache {
  private readonly summaries = new WeakMap<Series, BlockSummary>();
  private readonly blockSize: number;

  constructor(blockSize: number = BLOCK_SIZE) {
    this.blockSize = blockSize;
  }

  /**
   * @param xData - the x (timestamp) series.
   * @param yData - the y series, paired with x by time range and alignment overlap.
   * @param xWindow - the x range to bound over.
   * @param overlapThreshold - minimum time range overlap for an x/y pair to count.
   * @returns the bounds of y samples inside the window. Non-finite when none are.
   */
  value(
    xData: MultiSeries,
    yData: MultiSeries,
    xWindow: bounds.Bounds,
    overlapThreshold: TimeSpan,
  ): bounds.Bounds {
    let lower = Infinity;
    let upper = -Infinity;
    for (const x of xData.series)
      for (const y of yData.series) {
        const range = clip(x, y, xWindow, overlapThreshold);
        if (range == null) continue;
        const b = this.query(y, range[0], range[1]);
        if (b.lower < lower) lower = b.lower;
        if (b.upper > upper) upper = b.upper;
      }
    return { lower, upper };
  }

  private query(series: Series, lo: number, hi: number): bounds.Bounds {
    const { blockSize } = this;
    let summary = this.summaries.get(series);
    if (summary == null) {
      summary = { mins: [], maxs: [] };
      this.summaries.set(series, summary);
    }
    const { data } = series;
    const complete = Math.floor(series.length / blockSize);
    for (let b = summary.mins.length; b < complete; b++) {
      let min = Infinity;
      let max = -Infinity;
      const end = (b + 1) * blockSize;
      for (let i = b * blockSize; i < end; i++) {
        const v = Number(data[i]);
        if (v < min) min = v;
        if (v > max) max = v;
      }
      summary.mins.push(min);
      summary.maxs.push(max);
    }
    let lower = Infinity;
    let upper = -Infinity;
    let i = lo;
    while (i < hi) {
      const block = Math.floor(i / blockSize);
      const blockEnd = (block + 1) * blockSize;
      if (i === block * blockSize && blockEnd <= hi && block < summary.mins.length) {
        if (summary.mins[block] < lower) lower = summary.mins[block];
        if (summary.maxs[block] > upper) upper = summary.maxs[block];
        i = blockEnd;
        continue;
      }
      const end = Math.min(hi, blockEnd);
      for (; i < end; i++) {
        const v = Number(data[i]);
        if (v < lower) lower = v;
        if (v > upper) upper = v;
      }
    }
    const offset = Number(series.sampleOffset);
    return { lower: lower + offset, upper: upper + offset };
  }
}
