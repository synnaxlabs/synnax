// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  DataType,
  DEFAULT_POINT_LIMIT,
  fft,
  magnitudeScaleZ,
  MultiSeries,
  sampleRate,
  Series,
  TimeRange,
  windowFunctionZ,
} from "@synnaxlabs/x";
import { z } from "zod";

import {
  MultiSourceTransformer,
  type SeriesSourceSpec,
  type SeriesValue,
} from "@/telem/aether/telem";

export const spectrumPropsZ = z.object({
  window: windowFunctionZ.default("hann"),
  scale: magnitudeScaleZ.default("linear"),
  pointLimit: z.number().default(DEFAULT_POINT_LIMIT),
  /** Which series of the spectrum this transformer emits. */
  output: z.enum(["frequency", "magnitude"]),
});

export type SpectrumProps = z.input<typeof spectrumPropsZ>;

/** Samples of a channel inside a window, paired with its index. */
export interface WindowSamples {
  /** Sample values in index order. */
  y: Float64Array;
  /** Timestamps in nanoseconds, paired with y. */
  t: Float64Array;
}

/**
 * Collects the samples of data whose index timestamp lies inside the window. Data and
 * index are paired by alignment, so both must come from channels sharing an index.
 * @param index - The index series, sorted by alignment.
 * @param data - The data series.
 * @param window - Inclusive timestamp bounds in nanoseconds.
 */
export const windowSamples = (
  index: MultiSeries,
  data: MultiSeries,
  window: bounds.Bounds,
): WindowSamples => {
  const ys: number[] = [];
  const ts: number[] = [];
  for (const ix of index.series) {
    if (ix.length === 0) continue;
    const lo = ix.binarySearch(window.lower);
    let hi = ix.binarySearch(window.upper);
    if (hi < ix.length && Number(ix.at(hi, true)) === window.upper) hi++;
    for (let i = lo; i < hi; i++) {
      const v = data.atAlignment(ix.alignment + BigInt(i) * ix.alignmentMultiple);
      if (v == null) continue;
      ys.push(Number(v));
      ts.push(Number(ix.at(i, true)));
    }
  }
  return { y: Float64Array.from(ys), t: Float64Array.from(ts) };
};

const EMPTY: SeriesValue = [bounds.INVALID, new MultiSeries([])];

interface Computed {
  frequency: SeriesValue;
  magnitude: SeriesValue;
}

/**
 * Plots the amplitude spectrum of a channel over the window its sources serve. Takes
 * two series sources: `data`, the channel's samples, and `index`, its timestamps. The
 * sample rate is taken from the window's first and last timestamp, assuming uniform
 * spacing. While held, the last spectrum is served and new data is ignored.
 */
export class Spectrum extends MultiSourceTransformer<
  SeriesValue,
  SeriesValue,
  typeof spectrumPropsZ
> {
  static readonly TYPE = "spectrum";
  static readonly propsZ = spectrumPropsZ;
  schema = Spectrum.propsZ;
  private signature = "";
  private computed: Computed | null = null;
  private held = false;

  setHold(held: boolean): void {
    this.held = held;
    if (!held) this.signature = "";
  }

  protected transform(values: Record<string, SeriesValue>): SeriesValue {
    const data = values.data;
    const index = values.index;
    if (data == null || index == null) return EMPTY;
    if (this.held && this.computed != null) return this.computed[this.props.output];
    const [indexBounds, ix] = index;
    const signature = `${ix.length}:${data[1].length}:${indexBounds.lower}:${indexBounds.upper}`;
    if (signature !== this.signature || this.computed == null) {
      this.computed = this.compute(ix, data[1], indexBounds);
      this.signature = signature;
    }
    return this.computed[this.props.output];
  }

  private compute(ix: MultiSeries, data: MultiSeries, b: bounds.Bounds): Computed {
    const empty = { frequency: EMPTY, magnitude: EMPTY };
    if (!bounds.isFinite(b) || ix.length === 0) return empty;
    const { y, t } = windowSamples(ix, data, b);
    if (y.length < 2) return empty;
    const rate = sampleRate(y.length, t[0], t[t.length - 1]);
    const { window, scale, pointLimit } = this.props;
    const spectrum = fft(y, rate, { window, scale, pointLimit });
    if (spectrum.frequencies.length === 0) return empty;
    const timeRange = new TimeRange(t[0], t[t.length - 1]);
    const frequency = new Series({
      data: spectrum.frequencies,
      dataType: DataType.FLOAT32,
      timeRange,
      alignment: 0n,
    });
    const magnitude = new Series({
      data: spectrum.magnitudes,
      dataType: DataType.FLOAT32,
      timeRange,
      alignment: 0n,
    });
    return {
      frequency: [{ lower: 0, upper: rate / 2 }, new MultiSeries([frequency])],
      magnitude: [magnitude.bounds, new MultiSeries([magnitude])],
    };
  }
}

export const spectrum = (props: SpectrumProps): SeriesSourceSpec => ({
  props,
  type: Spectrum.TYPE,
  variant: "source",
  valueType: "series",
});
