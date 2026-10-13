// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { DECIBEL_FLOOR, fft, sampleRate } from "@/telem";

const sine = (n: number, rate: number, f: number, a = 1, offset = 0): Float32Array => {
  const out = new Float32Array(n);
  for (let i = 0; i < n; i++) out[i] = offset + a * Math.sin((2 * Math.PI * f * i) / rate);
  return out;
};

const peak = (magnitudes: Float32Array): number => {
  let at = 0;
  for (let k = 1; k < magnitudes.length; k++) if (magnitudes[k] > magnitudes[at]) at = k;
  return at;
};

describe("fft", () => {
  it("should return an empty spectrum for fewer than two samples", () => {
    expect(fft([1], 100).frequencies.length).toBe(0);
    expect(fft([], 100).magnitudes.length).toBe(0);
  });

  it("should place a sine at its bin with its amplitude", () => {
    const rate = 1024;
    const spectrum = fft(sine(1024, rate, 64, 3), rate, { window: "rectangular" });
    const at = peak(spectrum.magnitudes);
    expect(spectrum.frequencies[at]).toBeCloseTo(64, 6);
    expect(spectrum.magnitudes[at]).toBeCloseTo(3, 3);
    expect(spectrum.frequencies.at(-1)).toBeCloseTo(rate / 2, 6);
  });

  it("should recover the amplitude under a hann window", () => {
    const rate = 1024;
    const spectrum = fft(sine(1024, rate, 64, 2), rate, { window: "hann" });
    expect(spectrum.magnitudes[peak(spectrum.magnitudes)]).toBeCloseTo(2, 2);
  });

  it("should recover the amplitude under a flat top window", () => {
    const rate = 1024;
    const spectrum = fft(sine(1024, rate, 64, 2), rate, { window: "flat_top" });
    expect(spectrum.magnitudes[peak(spectrum.magnitudes)]).toBeCloseTo(2, 2);
  });

  it("should read a DC offset at the zero bin without doubling", () => {
    const spectrum = fft(new Float32Array(256).fill(5), 256, { window: "rectangular" });
    expect(spectrum.magnitudes[0]).toBeCloseTo(5, 6);
  });

  it("should average blocks limited by the point limit", () => {
    const rate = 1024;
    const spectrum = fft(sine(4096, rate, 64, 1), rate, {
      window: "rectangular",
      pointLimit: 1024,
    });
    expect(spectrum.blocks).toBe(4);
    expect(spectrum.blockLength).toBe(1024);
    expect(spectrum.transformLength).toBe(1024);
    expect(spectrum.magnitudes[peak(spectrum.magnitudes)]).toBeCloseTo(1, 3);
  });

  it("should keep each block within the point limit and pad to a power of two", () => {
    const spectrum = fft(sine(100000, 1000, 10), 1000, { pointLimit: 50000 });
    expect(spectrum.blocks).toBe(2);
    expect(spectrum.blockLength).toBe(50000);
    expect(spectrum.transformLength).toBe(65536);
  });

  it("should zero pad a short window", () => {
    const spectrum = fft(sine(100, 100, 10), 100, { window: "rectangular" });
    expect(spectrum.transformLength).toBe(128);
    expect(spectrum.frequencies.length).toBe(65);
    expect(spectrum.magnitudes[peak(spectrum.magnitudes)]).toBeCloseTo(1, 1);
  });

  it("should draw a two sample window", () => {
    const spectrum = fft([0, 1], 10);
    expect(spectrum.frequencies.length).toBe(2);
  });

  it("should scale to decibels with a floor", () => {
    const rate = 1024;
    const spectrum = fft(sine(1024, rate, 64, 10), rate, {
      window: "rectangular",
      scale: "decibel",
    });
    expect(spectrum.magnitudes[peak(spectrum.magnitudes)]).toBeCloseTo(20, 2);
    const zeros = fft(new Float32Array(64), 64, { scale: "decibel" });
    expect(zeros.magnitudes[3]).toBe(DECIBEL_FLOOR);
  });
});

describe("sampleRate", () => {
  it("should take the rate from the span between the first and last sample", () => {
    expect(sampleRate(1001, 0, 1e9)).toBeCloseTo(1000, 6);
  });

  it("should return NaN for fewer than two samples", () => {
    expect(sampleRate(1, 0, 1e9)).toBeNaN();
  });
});
