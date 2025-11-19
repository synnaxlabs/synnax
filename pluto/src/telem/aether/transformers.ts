// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";
import {
  bounds,
  color,
  id,
  MultiSeries,
  notation,
  scale,
  Series,
  status,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type Factory } from "@/telem/aether/factory";
import {
  type BooleanSink,
  type BooleanSinkSpec,
  type BooleanSource,
  type BooleanSourceSpec,
  type ColorSourceSpec,
  MultiSourceTransformer,
  type NumberSourceSpec,
  type SeriesSourceSpec,
  type Spec,
  type StringSourceSpec,
  type Telem,
  UnarySinkTransformer,
  UnarySourceTransformer,
} from "@/telem/aether/telem";

export class TransformerFactory implements Factory {
  type = "transformer";
  create(spec: Spec): Telem | null {
    switch (spec.type) {
      case SetPoint.TYPE:
        return new SetPoint(spec.props);
      case WithinBounds.TYPE:
        return new WithinBounds(spec.props);
      case Mean.TYPE:
        return new Mean(spec.props);
      case BooleanStatus.TYPE:
        return new BooleanStatus(spec.props);
      case StringifyNumber.TYPE:
        return new StringifyNumber(spec.props);
      case RollingAverage.TYPE:
        return new RollingAverage(spec.props);
      case ColorGradient.TYPE:
        return new ColorGradient(spec.props);
      case ScaleNumber.TYPE:
        return new ScaleNumber(spec.props);
    }
    return null;
  }
}

const setpointProps = z.object({
  truthy: z.number().default(1),
  falsy: z.number().default(0),
});

export type SetpointProps = z.infer<typeof setpointProps>;

export const setpoint = (props: SetpointProps): BooleanSinkSpec => ({
  props,
  type: SetPoint.TYPE,
  variant: "sink",
  valueType: "boolean",
});

export class SetPoint
  extends UnarySinkTransformer<boolean, number, typeof setpointProps>
  implements BooleanSink
{
  static readonly TYPE = "boolean-numeric-converter-sink";
  static readonly propsZ = setpointProps;
  schema = SetPoint.propsZ;

  transform(...values: boolean[]): number[] {
    return values.map((value) => (value ? this.props.truthy : this.props.falsy));
  }
}

export const withinBoundsProps = z.object({ trueBound: bounds.bounds });

export type WithinBoundsProps = z.infer<typeof withinBoundsProps>;

export const withinBounds = (props: WithinBoundsProps): BooleanSourceSpec => ({
  props,
  type: WithinBounds.TYPE,
  variant: "source",
  valueType: "boolean",
});

export class WithinBounds
  extends UnarySourceTransformer<number, boolean, typeof withinBoundsProps>
  implements BooleanSource
{
  static readonly TYPE = "boolean-source";
  static readonly propsZ = withinBoundsProps;
  schema = WithinBounds.propsZ;
  curr: boolean | null = null;

  protected shouldNotify(value: number): boolean {
    const shouldNotify = bounds.contains(this.props.trueBound, value) !== this.curr;
    this.curr = bounds.contains(this.props.trueBound, value);
    return shouldNotify;
  }

  protected transform(value: number): boolean {
    this.curr = bounds.contains(this.props.trueBound, value);
    return this.curr;
  }
}

const meanProps = z.object({});

export class Mean extends MultiSourceTransformer<number, number, typeof meanProps> {
  static readonly TYPE = "mean";
  static readonly propsZ = meanProps;
  schema = Mean.propsZ;

  protected transform(values: Record<string, number>): number {
    return (
      Object.values(values).reduce((a, b) => a + b, 0) / Object.keys(values).length
    );
  }
}

export const mean = (props: z.input<typeof meanProps>): BooleanSourceSpec => ({
  props,
  type: Mean.TYPE,
  variant: "source",
  valueType: "boolean",
});

export const booleanStatusProps = z.object({
  trueVariant: status.variantZ.default("success"),
});

export class BooleanStatus extends UnarySourceTransformer<
  status.Status,
  boolean,
  typeof booleanStatusProps
> {
  static readonly TYPE = "boolean-status";
  static readonly propsZ = booleanStatusProps;
  schema = BooleanStatus.propsZ;

  protected transform(value: status.Status): boolean {
    return value.variant === this.props.trueVariant;
  }
}

export const booleanStatus = (
  props: z.input<typeof booleanStatusProps>,
): BooleanSourceSpec => ({
  props,
  type: BooleanStatus.TYPE,
  variant: "source",
  valueType: "boolean",
});

export const stringifyNumberProps = z.object({
  precision: z.number().default(2),
  prefix: z.string().default(""),
  suffix: z.string().default(""),
  notation: notation.notationZ.default("standard"),
});

export class StringifyNumber extends UnarySourceTransformer<
  number,
  string,
  typeof stringifyNumberProps
> {
  static readonly TYPE = "stringify-number";
  static readonly propsZ = stringifyNumberProps;
  schema = StringifyNumber.propsZ;

  protected transform(value: number): string {
    if (isNaN(value)) return "";
    const { precision, prefix, suffix, notation: pNotation } = this.props;
    return `${prefix}${notation.stringifyNumber(value, precision, pNotation)}${suffix}`;
  }
}

export const stringifyNumber = (
  props: z.input<typeof stringifyNumberProps>,
): StringSourceSpec => ({
  props,
  type: StringifyNumber.TYPE,
  variant: "source",
  valueType: "string",
});

export const rollingAverageProps = z.object({
  windowSize: z.number().default(5),
});

export class RollingAverage extends UnarySourceTransformer<
  number,
  number,
  typeof rollingAverageProps
> {
  static readonly TYPE = "rolling-average";
  static readonly propsZ = meanProps;
  schema = rollingAverageProps;
  private values: number[] = [];

  protected transform(value: number): number {
    if (this.props.windowSize < 2 || isNaN(value)) return value;
    return this.values.reduce((a, b) => a + b, 0) / this.values.length;
  }

  protected shouldNotify(value: number): boolean {
    if (this.props.windowSize < 2) return true;
    if (this.values.length > this.props.windowSize) this.values = [];
    this.values.push(value);
    return this.values.length === this.props.windowSize;
  }
}

export const rollingAverage = (
  props: z.input<typeof rollingAverageProps>,
): NumberSourceSpec => ({
  props,
  type: RollingAverage.TYPE,
  variant: "source",
  valueType: "number",
});

export const colorGradientProps = z.object({
  gradient: color.gradientZ,
});

export class ColorGradient extends UnarySourceTransformer<
  number,
  color.Color,
  typeof colorGradientProps
> {
  static readonly TYPE = "color-gradient";
  static readonly propsZ = colorGradientProps;
  schema = ColorGradient.propsZ;

  protected transform(value: number): color.Color {
    return color.fromGradient(this.props.gradient, value);
  }
}

export const colorGradient = (
  props: z.input<typeof colorGradientProps>,
): ColorSourceSpec => ({
  props,
  type: ColorGradient.TYPE,
  variant: "source",
  valueType: "color",
});

export const scaleNumberProps = z.object({
  scale: scale.transform,
});

export class ScaleNumber extends UnarySourceTransformer<
  number,
  number,
  typeof scaleNumberProps
> {
  static readonly TYPE = "scale-number";
  static readonly propsZ = scaleNumberProps;
  schema = ScaleNumber.propsZ;

  protected transform(value: number): number {
    if (isNaN(value)) return value;
    const { offset, scale } = this.props.scale;
    return value * scale + offset;
  }
}

export const scaleNumber = (
  props: z.input<typeof scaleNumberProps>,
): NumberSourceSpec => ({
  props,
  type: ScaleNumber.TYPE,
  variant: "source",
  valueType: "number",
});

export const downsampleModeZ = z.enum(["average", "decimate"]);

export type DownsampleMode = z.infer<typeof downsampleModeZ>;

export const downsampleModeProps = z.object({
  mode: downsampleModeZ,
  windowSize: z.number().default(5),
});

export type DownsampleModeProps = z.infer<typeof downsampleModeProps>;

export const downsampleMode = (props: DownsampleModeProps): NumberSourceSpec => ({
  props,
  type: SeriesDownsampler.TYPE,
  variant: "source",
  valueType: "number",
});

interface DownsampleFunction {
  (source: Series, downsampled: Series, windowSize: number): void;
}

const decimate: DownsampleFunction = (source, downsampled, windowSize) => {
  const startIdx = downsampled.length * windowSize;

  for (let i = startIdx; i < source.length; i += windowSize) {
    const sample = source.sub(i, i + 1);
    if (sample !== undefined) downsampled.write(sample);
  }
};

const average: DownsampleFunction = (source, downsampled, windowSize) => {
  const startIdx = downsampled.length * windowSize;

  for (let i = startIdx; i < source.length; i += windowSize) {
    if (i + windowSize > source.length) break;
    const endIdx = Math.min(i + windowSize, source.length);
    let sum = 0;
    let count = 0;

    for (let j = i; j < endIdx; j++) {
      const val = source.at(j);
      if (val !== undefined && typeof val === "number") {
        sum += val;
        count++;
      }
    }

    if (count > 0)
      downsampled.write(
        new Series({
          data: [sum / count],
          dataType: source.dataType,
        }),
      );
  }
};

const DOWNSAMPLE_FUNCTIONS: Record<DownsampleMode, DownsampleFunction> = {
  decimate,
  average,
};

export class SeriesDownsampler {
  static readonly TYPE = "series-downsampler";
  private _downsample: DownsampleFunction | null = null;
  private readonly cache: MultiSeries = new MultiSeries();
  readonly props: DownsampleModeProps;

  constructor(props: DownsampleModeProps) {
    this.props = props;
  }

  private downsample(source: MultiSeries): DownsampleFunction {
    if (this._downsample == null)
      if (source.series[0].sampleOffset !== 0) this._downsample = decimate;
      else this._downsample = DOWNSAMPLE_FUNCTIONS[this.props.mode];
    return this._downsample;
  }

  transform(source: MultiSeries): MultiSeries {
    if (this.props.mode === "decimate" || this.props.windowSize <= 1) return source;
    if (source.series.length === 0) return this.cache;

    // Step 1: Evict Removed Series from Cache. We know we have an old entry if
    // the key of the first series in the source is not equal to the key of the
    // first series in the cache.
    while (
      this.cache.series.length > 0 &&
      !this.cache.series[0].key.startsWith(source.series[0].key)
    )
      this.cache.series.shift();

    // Step 2: Update the downsampled series in the cache.
    source.series.forEach((ser, i) => {
      let downsampledSeries = this.cache.series.at(i);
      // Step 2A: If the series is not in the cache, allocate a new series.
      if (downsampledSeries == null) {
        const capacity = Math.ceil(ser.capacity / this.props.windowSize);
        downsampledSeries = Series.alloc({
          key: ser.key + id.create(),
          dataType: ser.dataType,
          capacity,
          alignment: ser.alignment,
          alignmentMultiple: BigInt(this.props.windowSize),
          sampleOffset: ser.sampleOffset,
          timeRange: ser.timeRange,
        });
        this.cache.push(downsampledSeries);
      } else if (!downsampledSeries.key.startsWith(ser.key))
        throw new UnexpectedError(
          `[SeriesDownsampler] - expected series with key ${ser.key} to be in cache, but found ${downsampledSeries.key}`,
        );
      this.downsample(source)(ser, downsampledSeries, this.props.windowSize);
    });
    return this.cache;
  }
}

export const seriesDownsampler = (
  props: z.input<typeof downsampleModeProps>,
): SeriesSourceSpec => ({
  props,
  type: SeriesDownsampler.TYPE,
  variant: "source",
  valueType: "series",
});
