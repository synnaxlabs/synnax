// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds } from "@synnaxlabs/x";
import { z } from "zod";

import { status } from "@/status/aether";
import { type Factory } from "@/telem/aether/factory";
import {
  type Telem,
  type Spec,
  type BooleanSinkSpec,
  UnarySinkTransformer,
  type BooleanSink,
  UnarySourceTransformer,
  type BooleanSource,
  type BooleanSourceSpec,
  MultiSourceTransformer,
  type StringSourceSpec,
} from "@/telem/aether/telem";

export class TransformerFactory implements Factory {
  type = "transformer";
  create(spec: Spec): Telem | null {
    switch (spec.type) {
      case Setpoint.TYPE:
        return new Setpoint(spec.props);
      case WithinBounds.TYPE:
        return new WithinBounds(spec.props);
      case Mean.TYPE:
        return new Mean(spec.props);
      case BooleanStatus.TYPE:
        return new BooleanStatus(spec.props);
      case StringifyNumber.TYPE:
        return new StringifyNumber(spec.props);
    }
    return null;
  }
}

const setpointProps = z.object({
  truthy: z.number().optional().default(1),
  falsy: z.number().optional().default(0),
});

export type SetpointProps = z.infer<typeof setpointProps>;

export const setpoint = (props: SetpointProps): BooleanSinkSpec => ({
  props,
  type: Setpoint.TYPE,
  variant: "sink",
  valueType: "boolean",
});

export class Setpoint
  extends UnarySinkTransformer<boolean, number, typeof setpointProps>
  implements BooleanSink
{
  static readonly TYPE = "boolean-numeric-converter-sink";
  static readonly propsZ = setpointProps;
  schema = Setpoint.propsZ;

  transform(value: boolean): number {
    return value ? this.props.truthy : this.props.falsy;
  }
}

const withinBoundsProps = z.object({ trueBound: bounds.bounds });

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
    return bounds.contains(this.props.trueBound, value) !== this.curr;
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
  trueVariant: status.variantZ.optional().default("success"),
});

export class BooleanStatus extends UnarySourceTransformer<
  status.Spec,
  boolean,
  typeof booleanStatusProps
> {
  static readonly TYPE = "boolean-status";
  static readonly propsZ = booleanStatusProps;
  schema = BooleanStatus.propsZ;

  protected transform(value: status.Spec): boolean {
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
  precision: z.number().optional().default(2),
  prefix: z.string().optional().default(""),
  suffix: z.string().optional().default(""),
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
    return `${this.props.prefix}${value.toFixed(this.props.precision)}${
      this.props.suffix
    }`;
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
