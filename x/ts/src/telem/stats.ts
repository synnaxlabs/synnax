// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** @returns the smallest sample, or NaN when there are none. */
export const min = (y: ArrayLike<number>): number => {
  let v = Infinity;
  for (let i = 0; i < y.length; i++) if (y[i] < v) v = y[i];
  return y.length === 0 ? NaN : v;
};

/** @returns the largest sample, or NaN when there are none. */
export const max = (y: ArrayLike<number>): number => {
  let v = -Infinity;
  for (let i = 0; i < y.length; i++) if (y[i] > v) v = y[i];
  return y.length === 0 ? NaN : v;
};

/** @returns the difference between the largest and smallest sample. */
export const peakToPeak = (y: ArrayLike<number>): number => max(y) - min(y);

/** @returns the arithmetic mean, or NaN when there are no samples. */
export const mean = (y: ArrayLike<number>): number => {
  if (y.length === 0) return NaN;
  let sum = 0;
  for (let i = 0; i < y.length; i++) sum += y[i];
  return sum / y.length;
};

/** @returns the root mean square, or NaN when there are no samples. */
export const rms = (y: ArrayLike<number>): number => {
  if (y.length === 0) return NaN;
  let sum = 0;
  for (let i = 0; i < y.length; i++) sum += y[i] * y[i];
  return Math.sqrt(sum / y.length);
};

// Linear interpolation of the x at which y crosses level between samples i and i + 1.
const crossingAt = (
  x: ArrayLike<number>,
  y: ArrayLike<number>,
  i: number,
  level: number,
): number => {
  const dy = y[i + 1] - y[i];
  if (dy === 0) return x[i];
  return x[i] + ((level - y[i]) / dy) * (x[i + 1] - x[i]);
};

/**
 * Estimates the fundamental frequency from the rising crossings of the mean.
 * @param x - Sample positions, in the unit whose inverse the result takes (seconds
 * give Hz).
 * @param y - Samples, paired with x.
 * @returns the frequency, or NaN when fewer than two rising crossings exist.
 */
export const frequency = (x: ArrayLike<number>, y: ArrayLike<number>): number => {
  const n = Math.min(x.length, y.length);
  if (n < 3) return NaN;
  const level = mean(y);
  let first = NaN;
  let last = NaN;
  let count = 0;
  for (let i = 0; i < n - 1; i++) {
    if (!(y[i] < level && y[i + 1] >= level)) continue;
    const at = crossingAt(x, y, i, level);
    if (count === 0) first = at;
    last = at;
    count++;
  }
  if (count < 2 || last <= first) return NaN;
  return (count - 1) / (last - first);
};

/**
 * Measures the 10% to 90% rise time of the first rising transition.
 * @param x - Sample positions.
 * @param y - Samples, paired with x.
 * @returns the rise time in x units, or NaN when no full transition exists.
 */
export const riseTime = (x: ArrayLike<number>, y: ArrayLike<number>): number => {
  const n = Math.min(x.length, y.length);
  if (n < 2) return NaN;
  const lo = min(y);
  const span = max(y) - lo;
  if (span === 0) return NaN;
  const low = lo + 0.1 * span;
  const high = lo + 0.9 * span;
  let start = NaN;
  for (let i = 0; i < n - 1; i++) {
    if (Number.isNaN(start)) {
      if (y[i] < low && y[i + 1] >= low) start = crossingAt(x, y, i, low);
      continue;
    }
    if (y[i + 1] < low) start = NaN;
    else if (y[i] < high && y[i + 1] >= high) return crossingAt(x, y, i, high) - start;
  }
  return NaN;
};
