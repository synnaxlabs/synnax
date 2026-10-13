// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const WINDOW_FUNCTIONS = ["hann", "rectangular", "flat_top"] as const;
export const windowFunctionZ = z.enum(WINDOW_FUNCTIONS);
export type WindowFunction = z.infer<typeof windowFunctionZ>;

export const MAGNITUDE_SCALES = ["linear", "decibel"] as const;
export const magnitudeScaleZ = z.enum(MAGNITUDE_SCALES);
export type MagnitudeScale = z.infer<typeof magnitudeScaleZ>;

export interface FFTOptions {
  /** Window function applied to each block. Defaults to hann. */
  window?: WindowFunction;
  /** Scale of the magnitudes. Defaults to linear. */
  scale?: MagnitudeScale;
  /** Most samples in one block. Defaults to 65536. */
  pointLimit?: number;
}

export interface Spectrum {
  /** Bin center frequencies in Hz, from 0 to the Nyquist frequency. */
  frequencies: Float32Array<ArrayBuffer>;
  /** Amplitude per bin in the input's units, or in dB relative to one unit. */
  magnitudes: Float32Array<ArrayBuffer>;
  /** Blocks averaged into the result. */
  blocks: number;
  /** Samples per block before padding. */
  blockLength: number;
  /** Transform length each block was padded to. */
  transformLength: number;
}

export const DEFAULT_POINT_LIMIT = 65536;

/** Decibel reading of a zero magnitude, in place of -Infinity. */
export const DECIBEL_FLOOR = -200;

const EMPTY: Spectrum = {
  frequencies: new Float32Array(0),
  magnitudes: new Float32Array(0),
  blocks: 0,
  blockLength: 0,
  transformLength: 0,
};

const FLAT_TOP = [0.21557895, 0.41663158, 0.277263158, 0.083578947, 0.006947368];

const windowCoefficients = (fn: WindowFunction, n: number): Float64Array => {
  const w = new Float64Array(n);
  if (fn === "rectangular") return w.fill(1);
  const d = Math.max(n - 1, 1);
  for (let i = 0; i < n; i++) {
    const t = (2 * Math.PI * i) / d;
    if (fn === "hann") w[i] = 0.5 * (1 - Math.cos(t));
    else
      w[i] =
        FLAT_TOP[0] -
        FLAT_TOP[1] * Math.cos(t) +
        FLAT_TOP[2] * Math.cos(2 * t) -
        FLAT_TOP[3] * Math.cos(3 * t) +
        FLAT_TOP[4] * Math.cos(4 * t);
  }
  return w;
};

const nextPowerOfTwo = (n: number): number => 2 ** Math.ceil(Math.log2(n));

// In-place iterative radix-2 transform. re and im must have a power of two length.
const transform = (re: Float64Array, im: Float64Array): void => {
  const n = re.length;
  for (let i = 1, j = 0; i < n; i++) {
    let bit = n >> 1;
    for (; (j & bit) !== 0; bit >>= 1) j ^= bit;
    j ^= bit;
    if (i < j) {
      [re[i], re[j]] = [re[j], re[i]];
      [im[i], im[j]] = [im[j], im[i]];
    }
  }
  for (let len = 2; len <= n; len <<= 1) {
    const angle = (-2 * Math.PI) / len;
    const wRe = Math.cos(angle);
    const wIm = Math.sin(angle);
    for (let i = 0; i < n; i += len) {
      let cRe = 1;
      let cIm = 0;
      const half = len >> 1;
      for (let k = 0; k < half; k++) {
        const a = i + k;
        const b = a + half;
        const tRe = re[b] * cRe - im[b] * cIm;
        const tIm = re[b] * cIm + im[b] * cRe;
        re[b] = re[a] - tRe;
        im[b] = im[a] - tIm;
        re[a] += tRe;
        im[a] += tIm;
        const nRe = cRe * wRe - cIm * wIm;
        cIm = cRe * wIm + cIm * wRe;
        cRe = nRe;
      }
    }
  }
};

/**
 * @returns the sample rate in Hz of n samples spanning the given times, assuming
 * uniform spacing.
 * @param n - The sample count.
 * @param first - The timestamp of the first sample in nanoseconds.
 * @param last - The timestamp of the last sample in nanoseconds.
 */
export const sampleRate = (n: number, first: number, last: number): number => {
  if (n < 2 || last <= first) return NaN;
  return ((n - 1) * 1e9) / (last - first);
};

/**
 * Computes the amplitude spectrum of uniformly sampled data. The samples are cut into
 * blocks of min(length, pointLimit) samples, the trailing partial block is dropped,
 * each block is windowed and zero-padded to a power of two, and the bin magnitudes
 * are averaged over the blocks. A sine of amplitude A reads A at its bin. Fewer than
 * two samples or a non-finite rate give an empty spectrum.
 * @param samples - The samples, in uniform time order.
 * @param rate - The sample rate in Hz.
 * @param options - Window, scale, and point limit.
 */
export const fft = (
  samples: ArrayLike<number>,
  rate: number,
  { window = "hann", scale = "linear", pointLimit = DEFAULT_POINT_LIMIT }: FFTOptions = {},
): Spectrum => {
  const total = samples.length;
  if (total < 2 || !Number.isFinite(rate) || rate <= 0) return EMPTY;
  const n = Math.min(total, Math.max(2, Math.floor(pointLimit)));
  const N = nextPowerOfTwo(n);
  const blocks = Math.floor(total / n);
  const w = windowCoefficients(window, n);
  let gain = 0;
  for (let i = 0; i < n; i++) gain += w[i];
  gain /= n;
  const bins = N / 2 + 1;
  const acc = new Float64Array(bins);
  const re = new Float64Array(N);
  const im = new Float64Array(N);
  for (let b = 0; b < blocks; b++) {
    const start = b * n;
    re.fill(0);
    im.fill(0);
    for (let i = 0; i < n; i++) re[i] = samples[start + i] * w[i];
    transform(re, im);
    for (let k = 0; k < bins; k++) acc[k] += Math.hypot(re[k], im[k]);
  }
  const norm = 1 / (blocks * n * gain);
  const frequencies = new Float32Array(bins);
  const magnitudes = new Float32Array(bins);
  for (let k = 0; k < bins; k++) {
    frequencies[k] = (k * rate) / N;
    let m = acc[k] * norm;
    if (k !== 0 && k !== N / 2) m *= 2;
    if (scale === "decibel") m = m > 0 ? Math.max(20 * Math.log10(m), DECIBEL_FLOOR) : DECIBEL_FLOOR;
    magnitudes[k] = m;
  }
  return { frequencies, magnitudes, blocks, blockLength: n, transformLength: N };
};
