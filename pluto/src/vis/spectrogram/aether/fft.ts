// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import FFT from "fft.js";
import { z } from "zod";

export const windowFunctionZ = z.enum(["hann", "blackmanHarris"]);
export type WindowFunction = z.infer<typeof windowFunctionZ>;

const generateHannWindow = (size: number): Float32Array => {
  const w = new Float32Array(size);
  for (let i = 0; i < size; i++)
    w[i] = 0.5 * (1 - Math.cos((2 * Math.PI * i) / (size - 1)));
  return w;
};

const generateBlackmanHarrisWindow = (size: number): Float32Array => {
  const w = new Float32Array(size);
  const a0 = 0.35875,
    a1 = 0.48829,
    a2 = 0.14128,
    a3 = 0.01168;
  for (let i = 0; i < size; i++) {
    const t = (2 * Math.PI * i) / (size - 1);
    w[i] = a0 - a1 * Math.cos(t) + a2 * Math.cos(2 * t) - a3 * Math.cos(3 * t);
  }
  return w;
};

export class FFTProcessor {
  private fft: FFT;
  private window: Float32Array;
  private readonly fftSize: number;
  private readonly output: Float32Array;
  private readonly complexOutput: Float32Array;
  private readonly windowed: Float32Array;
  private readonly normDb: number;

  constructor(fftSize: number, windowFunction: WindowFunction) {
    this.fftSize = fftSize;
    this.fft = new FFT(fftSize);
    this.window =
      windowFunction === "hann"
        ? generateHannWindow(fftSize)
        : generateBlackmanHarrisWindow(fftSize);
    this.output = new Float32Array(fftSize / 2);
    this.complexOutput = new Float32Array(fftSize * 2);
    this.windowed = new Float32Array(fftSize);
    this.normDb = 20 * Math.log10(fftSize);
  }

  process(samples: Float32Array): Float32Array {
    const w = this.windowed;
    for (let i = 0; i < this.fftSize; i++) w[i] = samples[i] * this.window[i];
    this.fft.realTransform(this.complexOutput, w);
    const halfSize = this.fftSize / 2;
    const normDb = this.normDb;
    const co = this.complexOutput;
    for (let i = 0; i < halfSize; i++) {
      const re = co[2 * i];
      const im = co[2 * i + 1];
      const powerSq = re * re + im * im;
      this.output[i] = 10 * Math.log10(Math.max(powerSq, 1e-20)) - normDb;
    }
    return this.output;
  }
}
