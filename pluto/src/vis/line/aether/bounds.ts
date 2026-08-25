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
 * @param xData - the x (timestamp) series.
 * @param yData - the y series, paired with x by time range and alignment overlap.
 * @param xWindow - the x range to bound over.
 * @param overlapThreshold - minimum time range overlap for an x/y pair to count.
 * @param fallback - returned when no paired sample falls inside the window.
 * @returns the bounds of y samples whose paired x value falls inside the window.
 */
export const windowBounds = (
  xData: MultiSeries,
  yData: MultiSeries,
  xWindow: bounds.Bounds,
  overlapThreshold: TimeSpan,
  fallback: bounds.Bounds,
): bounds.Bounds => {
  let lower = Infinity;
  let upper = -Infinity;
  for (const x of xData.series)
    for (const y of yData.series) {
      const range = clip(x, y, xWindow, overlapThreshold);
      if (range == null) continue;
      const b = y.boundsFor(range[0], range[1]);
      if (b.lower < lower) lower = b.lower;
      if (b.upper > upper) upper = b.upper;
    }
  const result = { lower, upper };
  if (!bounds.isFinite(result)) return fallback;
  return result;
};
