// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { bounds, color, type math, notation } from "@synnaxlabs/x";
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
      case BandColor.TYPE:
        return new BandColor(spec.props);
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

export const withinBoundsProps = z.object({ trueBound: bounds.boundsZ() });

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

  protected transform(value: number): boolean {
    return bounds.contains(this.props.trueBound, value);
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
  math.Numeric,
  string,
  typeof stringifyNumberProps
> {
  static readonly TYPE = "stringify-number";
  static readonly propsZ = stringifyNumberProps;
  schema = StringifyNumber.propsZ;

  protected transform(value: math.Numeric): string {
    if (typeof value === "number" && isNaN(value)) return "";
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
  math.Numeric,
  number,
  typeof rollingAverageProps
> {
  static readonly TYPE = "rolling-average";
  static readonly propsZ = rollingAverageProps;
  schema = rollingAverageProps;
  private readonly window: number[] = [];

  protected transform(value: math.Numeric): number {
    const num = Number(value);
    if (this.props.windowSize < 2 || isNaN(num) || this.window.length === 0) return num;
    return this.window.reduce((a, b) => a + b, 0) / this.window.length;
  }

  // The window advances here because this is the only hook that runs once per arriving
  // sample. Staleness counts arrivals, so every sample must also reach the listener. A
  // NaN sample enters the window like any other, so the average reports it until it
  // slides out.
  protected shouldNotify(value: math.Numeric): boolean {
    if (this.props.windowSize < 2) return true;
    this.window.push(Number(value));
    if (this.window.length > this.props.windowSize) this.window.shift();
    return true;
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

export const bandColorProps = color.scaleZ.extend({
  background: color.colorZ.optional(),
});

export type BandColorProps = z.input<typeof bandColorProps>;

/**
 * Maps a number onto the color of its threshold band. Reads the number from the
 * `source` input and, when a band flashes, a {@link clock} from the `phase` input. A
 * band owns values at or above its threshold and below the next higher threshold.
 * Values below every threshold take the background, or no color when it is absent.
 * While the owning band flashes, odd clock ticks paint the background instead.
 * Listeners hear only changes of color.
 */
export class BandColor extends MultiSourceTransformer<
  math.Numeric | string,
  color.Color,
  typeof bandColorProps
> {
  static readonly TYPE = "band-color";
  static readonly propsZ = bandColorProps;
  schema = BandColor.propsZ;
  private sorted?: color.Band[];
  private notified: color.Color | null = null;

  private get bands(): color.Band[] {
    this.sorted ??= [...this.props.bands].sort((a, b) => a.threshold - b.threshold);
    return this.sorted;
  }

  protected transform({
    source,
    phase = 0,
  }: Record<string, math.Numeric | string>): color.Color {
    const value = source === "" ? NaN : Number(source);
    const background = this.props.background ?? color.ZERO;
    const i = this.bands.findLastIndex(({ threshold }) => threshold <= value);
    if (i === -1) return background;
    const owner = this.bands[i];
    if (owner.flashing && Number(phase) % 2 === 1) return background;
    const next = this.bands.at(i + 1);
    if (!this.props.smooth || next == null) return owner.color;
    return color.fromGradient(
      [
        { key: "owner", position: owner.threshold, color: owner.color },
        { key: "next", position: next.threshold, color: next.color },
      ],
      value,
    );
  }

  protected shouldNotify(): boolean {
    const next = this.value();
    if (this.notified != null && color.equals(this.notified, next)) return false;
    this.notified = next;
    return true;
  }
}

export const bandColor = (props: BandColorProps): ColorSourceSpec => ({
  props,
  type: BandColor.TYPE,
  variant: "source",
  valueType: "color",
});
