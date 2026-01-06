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
  color,
  DataType,
  MultiSeries,
  Rate,
  Series,
  TimeRange,
  typedArrayZ,
} from "@synnaxlabs/x";
import { z } from "zod";

import { type Factory } from "@/telem/aether/factory";
import {
  AbstractSource,
  type ColorSource,
  type ColorSourceSpec,
  type NumberSource,
  type NumberSourceSpec,
  type SeriesSource,
  type SeriesSourceSpec,
  type Spec,
  type StringSourceSpec,
  type Telem,
} from "@/telem/aether/telem";

export class StaticFactory implements Factory {
  type = "static";
  create(spec: Spec): Telem | null {
    switch (spec.type) {
      case FixedSeries.TYPE:
        return new FixedSeries(spec.props);
      case IterativeSeries.TYPE:
        return new IterativeSeries(spec.props);
      case FixedNumber.TYPE:
        return new FixedNumber(spec.props);
      case FixedString.TYPE:
        return new FixedString(spec.props);
      case FixedColorSource.TYPE:
        return new FixedColorSource(spec.props);
      default:
        return null;
    }
  }
}

export const fixedSeriesPropsZ = z.object({
  data: z.array(typedArrayZ),
  offsets: z.array(z.number()).default([]),
});

export type FixedArrayProps = z.input<typeof fixedSeriesPropsZ>;

class FixedSeries extends AbstractSource<typeof fixedSeriesPropsZ> {
  data: Series[];
  schema = fixedSeriesPropsZ;

  static readonly TYPE = "static-series";

  constructor(props: unknown) {
    super(props);
    this.data = this.props.data.map(
      (x, i) =>
        new Series({
          data: x,
          dataType: DataType.FLOAT32,
          timeRange: TimeRange.ZERO,
          sampleOffset: this.props.offsets[i] ?? 0,
        }),
    );
  }

  value(): [bounds.Bounds, Series[]] {
    const b = bounds.max(this.data.map((x) => x.bounds));
    return [b, this.data];
  }
}

export const iterativeSeriesPropsZ = fixedSeriesPropsZ.extend({
  rate: Rate.z,
  yOffset: z.number().default(0),
  scroll: z.number().default(0),
  startPosition: z.number().default(0),
  scrollBounds: z.boolean().default(false),
});

export type IterativeArrayProps = z.input<typeof iterativeSeriesPropsZ>;

export class IterativeSeries
  extends AbstractSource<typeof iterativeSeriesPropsZ>
  implements SeriesSource
{
  static readonly TYPE = "iterative-series";
  schema = iterativeSeriesPropsZ;

  position: number;
  interval?: number;
  data: Series[];

  constructor(props: unknown) {
    super(props);
    this.position = this.props.startPosition;
    this.start(this.props.rate);
    this.data = this.props.data.map(
      (x, i) =>
        new Series({
          data: x,
          dataType: DataType.FLOAT32,
          timeRange: TimeRange.ZERO,
          sampleOffset: this.props.offsets[i] ?? 0,
        }),
    );
  }

  value(): [bounds.Bounds, MultiSeries] {
    const d = this.data.map((x) => x.slice(0, this.position));
    if (this.props.scrollBounds) {
      const lower =
        d[0].data[
          this.position - this.props.scroll < 0 ? 0 : this.position - this.props.scroll
        ];
      const upper = d[0].data[this.position - 1];
      const b = {
        lower: Number(lower),
        upper: Number(upper),
      };
      return [b, new MultiSeries(d)];
    }
    const b = bounds.max(d.map((x) => x.bounds));
    return [b, new MultiSeries(d)];
  }

  start(rate: Rate): void {
    if (this.interval != null) clearInterval(this.interval);
    this.interval = setInterval(() => {
      this.notify?.();
      this.position++;
    }, rate.period.milliseconds) as unknown as number;
  }

  cleanup(): void {
    clearInterval(this.interval);
    this.interval = undefined;
  }
}

export const fixedNumberPropsZ = z.number();

export type FixedNumberProps = z.infer<typeof fixedNumberPropsZ>;

export class FixedNumber
  extends AbstractSource<typeof fixedNumberPropsZ>
  implements NumberSource
{
  static readonly TYPE = "static-numeric";
  schema = fixedNumberPropsZ;

  value(): number {
    return this.props;
  }
}

export const fixedStringPropsZ = z.string();

export type FixedStringProps = z.infer<typeof fixedStringPropsZ>;

export class FixedString extends AbstractSource<typeof fixedStringPropsZ> {
  static readonly TYPE = "static-string";
  schema = fixedStringPropsZ;

  value(): string {
    return this.props;
  }
}

export const fixedColorSourcePropsZ = color.crudeZ;

export type FixedColorSourceProps = z.infer<typeof fixedColorSourcePropsZ>;

export class FixedColorSource
  extends AbstractSource<typeof fixedColorSourcePropsZ>
  implements ColorSource
{
  static readonly TYPE = "static-color";
  schema = fixedColorSourcePropsZ;

  value(): color.Color {
    return color.construct(this.props);
  }
}

export const fixedArray = (props: FixedArrayProps): SeriesSourceSpec => ({
  type: FixedSeries.TYPE,
  props,
  variant: "source",
  valueType: "series",
});

export const iterativeArray = (props: IterativeArrayProps): SeriesSourceSpec => ({
  type: IterativeSeries.TYPE,
  props,
  variant: "source",
  valueType: "series",
});

export const fixedNumber = (value: number): NumberSourceSpec => ({
  type: FixedNumber.TYPE,
  props: value,
  variant: "source",
  valueType: "number",
});

export const fixedString = (value: string): StringSourceSpec => ({
  type: FixedString.TYPE,
  props: value,
  variant: "source",
  valueType: "string",
});

export const fixedColor = (color: color.Crude): ColorSourceSpec => ({
  type: FixedColorSource.TYPE,
  props: color,
  variant: "source",
  valueType: "color",
});
