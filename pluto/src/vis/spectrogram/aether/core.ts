// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fft } from "@synnaxlabs/x";
import { z } from "zod";

import { type colorMapZ, getLUT } from "@/vis/spectrogram/aether/colormap";

export const coreStateZ = z.object({
  sampleRate: z.number().default(48000),
  fftSize: z.number().default(2048),
  windowFunction: fft.windowFunctionZ.default("hann"),
  overlap: z.number().min(0).max(0.95).default(0.5),
  colorMap: z
    .enum(["viridis", "inferno", "magma", "plasma", "jet", "grayscale"] satisfies [
      z.infer<typeof colorMapZ>,
      ...Array<z.infer<typeof colorMapZ>>,
    ])
    .default("viridis"),
  dbMin: z.number().default(-100),
  dbMax: z.number().default(0),
  freqMin: z.number().default(0),
  freqMax: z.number().default(0),
});

export type CoreState = z.infer<typeof coreStateZ>;

export interface CoreContext {
  spectrogramCanvas: OffscreenCanvas;
  spectrogramCtx: OffscreenCanvasRenderingContext2D;
  spectrogramWidth: number;
}

export class Core {
  private processor: fft.Processor;
  private columnImageData: ImageData;
  private processedSamples: number = 0;
  halfFFT: number;

  private canvas: OffscreenCanvas;
  private ctx: OffscreenCanvasRenderingContext2D;
  width: number;

  constructor(
    private state: CoreState,
    width: number,
  ) {
    this.width = Math.max(1, width);
    this.processor = new fft.Processor({
      size: state.fftSize,
      windowFunction: state.windowFunction,
      overlap: state.overlap,
    });
    this.halfFFT = state.fftSize / 2;
    this.canvas = new OffscreenCanvas(this.width, this.halfFFT);
    const ctx = this.canvas.getContext("2d");
    if (ctx == null) throw new Error("Could not get spectrogram 2d context");
    this.ctx = ctx;
    this.columnImageData = new ImageData(1, this.halfFFT);
  }

  get hopSize(): number {
    return this.processor.hopSize;
  }

  get offscreenCanvas(): OffscreenCanvas {
    return this.canvas;
  }

  get offscreenCtx(): OffscreenCanvasRenderingContext2D {
    return this.ctx;
  }

  updateState(state: CoreState, width: number): boolean {
    width = Math.max(1, Math.round(width));
    const sizeChanged =
      state.fftSize !== this.state.fftSize ||
      state.windowFunction !== this.state.windowFunction ||
      state.overlap !== this.state.overlap;
    if (sizeChanged) {
      this.processor = new fft.Processor({
        size: state.fftSize,
        windowFunction: state.windowFunction,
        overlap: state.overlap,
      });
      this.halfFFT = state.fftSize / 2;
      this.processedSamples = 0;
    }
    const halfFFT = state.fftSize / 2;
    if (width !== this.width || halfFFT !== this.columnImageData.height) {
      const oldCanvas = this.canvas;
      const oldWidth = this.width;
      this.width = width;
      this.canvas = new OffscreenCanvas(width, halfFFT);
      const ctx = this.canvas.getContext("2d");
      if (ctx == null) throw new Error("Could not get spectrogram 2d context");
      this.ctx = ctx;
      if (oldCanvas != null && !sizeChanged) {
        const copyW = Math.min(width, oldWidth);
        ctx.drawImage(
          oldCanvas,
          oldWidth - copyW,
          0,
          copyW,
          oldCanvas.height,
          width - copyW,
          0,
          copyW,
          halfFFT,
        );
      }
      this.columnImageData = new ImageData(1, halfFFT);
    }
    this.state = state;
    return sizeChanged;
  }

  processTelemetry(multiSeries: { series: Array<{ data: ArrayBufferView }> }): void {
    const { colorMap, dbMin, dbMax, sampleRate, fftSize } = this.state;
    const freqMax = this.state.freqMax > 0 ? this.state.freqMax : sampleRate / 2;
    const halfFFT = fftSize / 2;
    const binFreqStep = sampleRate / fftSize;
    const minBin = Math.max(0, Math.floor(this.state.freqMin / binFreqStep));
    const maxBin = Math.min(halfFFT, Math.ceil(freqMax / binFreqStep));
    const invDbRange = 1 / (dbMax - dbMin);
    const lut = getLUT(colorMap);

    let totalSamples = 0;
    for (const series of multiSeries.series) {
      const d = series.data;
      if (d instanceof Float32Array || d instanceof Float64Array)
        totalSamples += d.length;
    }

    let skip = Math.max(0, this.processedSamples);

    for (const series of multiSeries.series) {
      const { data } = series;
      if (!(data instanceof Float32Array) && !(data instanceof Float64Array)) continue;
      const samples = data instanceof Float64Array ? new Float32Array(data) : data;

      if (skip >= samples.length) {
        skip -= samples.length;
        continue;
      }
      const input = skip > 0 ? samples.subarray(skip) : samples;
      skip = 0;

      for (const magnitudes of this.processor.feed(input)) {
        const imgData = this.columnImageData.data;
        for (let bin = minBin; bin < maxBin; bin++) {
          const normalized = Math.max(
            0,
            Math.min(1, (magnitudes[bin] - dbMin) * invDbRange),
          );
          const lutIdx =
            Math.max(0, Math.min(255, (normalized * 255 + 0.5) | 0)) << 2;
          const offset = (halfFFT - 1 - bin) << 2;
          imgData[offset] = lut[lutIdx];
          imgData[offset + 1] = lut[lutIdx + 1];
          imgData[offset + 2] = lut[lutIdx + 2];
          imgData[offset + 3] = 255;
        }
        this.ctx.drawImage(this.canvas, -1, 0);
        this.ctx.putImageData(this.columnImageData, this.width - 1, 0);
      }
    }

    this.processedSamples = totalSamples;
  }
}
