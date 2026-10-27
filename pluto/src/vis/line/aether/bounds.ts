// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, type MultiSeries, type Series, type TimeSpan } from "@synnaxlabs/x";

// Dynamic buffering can lead to very slight, unintended overlaps, so time ranges only
// pair when they overlap by a threshold. Equal time ranges always pair, as series from
// one write can span less than the threshold.
const timesOverlap = (x: Series, y: Series, overlapThreshold: TimeSpan): boolean =>
  x.timeRange.equals(y.timeRange) ||
  x.timeRange.overlapsWith(y.timeRange, overlapThreshold);

// Returns the series in alignment order, copying them only when they are out of order.
const byAlignment = (series: Series[]): Series[] => {
  for (let i = 1; i < series.length; i++)
    if (series[i].alignment < series[i - 1].alignment)
      return series.toSorted((a, b) =>
        a.alignment < b.alignment ? -1 : a.alignment > b.alignment ? 1 : 0,
      );
  return series;
};

/**
 * Calls f with each x and y series that pair, along with the alignments [start, end)
 * they share. Series pair when they share alignments, step through them equally, and
 * their time ranges overlap by overlapThreshold.
 *
 * Pairs come in alignment order: x by x, then y by y, when no list holds two series
 * that overlap in alignment. The cost is linear in the series and the pairs.
 */
export const forEachPair = (
  xs: Series[],
  ys: Series[],
  overlapThreshold: TimeSpan,
  f: (x: Series, y: Series, start: bigint, end: bigint) => void,
): void => {
  const pair = (x: Series, y: Series): void => {
    const start = x.alignment > y.alignment ? x.alignment : y.alignment;
    const xUpper = x.alignmentBounds.upper;
    const yUpper = y.alignmentBounds.upper;
    const end = xUpper < yUpper ? xUpper : yUpper;
    if (
      end > start &&
      x.alignmentMultiple === y.alignmentMultiple &&
      timesOverlap(x, y, overlapThreshold)
    )
      f(x, y, start, end);
  };
  xs = byAlignment(xs);
  ys = byAlignment(ys);
  let i = 0;
  let j = 0;
  // A forward scan: the series that starts first pairs with each series of the other
  // list that starts before it ends, so overlaps inside a list still pair.
  while (i < xs.length && j < ys.length)
    if (xs[i].alignment <= ys[j].alignment) {
      const x = xs[i++];
      const { upper } = x.alignmentBounds;
      for (let k = j; k < ys.length && ys[k].alignment < upper; k++) pair(x, ys[k]);
    } else {
      const y = ys[j++];
      const { upper } = y.alignmentBounds;
      for (let k = i; k < xs.length && xs[k].alignment < upper; k++) pair(xs[k], y);
    }
};

/**
 * @returns the y index range [lo, hi) of the shared alignments [start, end) whose x
 * value falls inside the window, or null when there is none.
 */
const clip = (
  x: Series,
  y: Series,
  xWindow: bounds.Bounds,
  start: bigint,
  end: bigint,
): [number, number] | null => {
  if (y.dataType.isVariable) return null;
  const mult = x.alignmentMultiple;
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
 * @param yData - the y series, paired with x by {@link forEachPair}.
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
  forEachPair(xData.series, yData.series, overlapThreshold, (x, y, start, end) => {
    const range = clip(x, y, xWindow, start, end);
    if (range == null) return;
    const b = y.boundsFor(range[0], range[1]);
    if (b.lower < lower) lower = b.lower;
    if (b.upper > upper) upper = b.upper;
  });
  const result = { lower, upper };
  if (!bounds.isFinite(result)) return fallback;
  return result;
};
