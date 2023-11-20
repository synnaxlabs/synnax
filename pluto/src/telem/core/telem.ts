// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { observe, type bounds, type Series, type UnknownRecord } from "@synnaxlabs/x";
import { deep } from "@synnaxlabs/x";
import { z } from "zod";

import { type color } from "@/color/core";
import { type status } from "@/status/aether";
import { prettyParse } from "@/util/zod";

const transferrable = z.union([z.instanceof(ArrayBuffer)]);

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export const specZ = z.object({
  type: z.string(),
  variant: z.string(),
  props: z.any(),
  transfer: z.array(transferrable).optional(),
});

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export type Spec = z.infer<typeof specZ>;

export const xySourceSpecZ = specZ.extend({
  variant: z.literal("xy-source"),
});

export type XYSourceSpec = z.infer<typeof xySourceSpecZ>;

export interface Telem {
  cleanup?: () => void;
}

export interface Source<V> extends Telem, observe.Observable<void> {
  value: () => Promise<V>;
}

export interface Sink<V> extends Telem {
  set: (value: V) => Promise<void>;
}

export type SeriesSource = Source<[bounds.Bounds, Series[]]>;
export const seriesSourceSpecZ = specZ.extend({
  variant: z.literal("series-source"),
});
export type SeriesSourceSpec = z.infer<typeof seriesSourceSpecZ>;

export type BooleanSource = Source<boolean>;
export const booleanSourceSpecZ = specZ.extend({
  variant: z.literal("boolean-source"),
});
export type BooleanSourceSpec = z.infer<typeof booleanSourceSpecZ>;

export type BooleanSink = Sink<boolean>;
export const booleanSinkSpecZ = specZ.extend({
  variant: z.literal("boolean-sink"),
});
export type BooleanSinkSpec = z.infer<typeof booleanSinkSpecZ>;

export type NumericSource = Source<number>;
export const numericSourceSpecZ = specZ.extend({
  variant: z.literal("numeric-source"),
});
export type NumericSourceSpec = z.infer<typeof numericSourceSpecZ>;

export type NumericSink = Sink<number>;
export const numericSinkSpecZ = specZ.extend({
  variant: z.literal("numeric-sink"),
});
export type NumericSinkSpec = z.infer<typeof numericSinkSpecZ>;

export type ColorSource = Source<color.Color>;
export const colorSourceSpecZ = specZ.extend({
  variant: z.literal("color-source"),
});
export type ColorSourceSpec = z.infer<typeof colorSourceSpecZ>;

export type StatusSource = Source<status.Spec>;
export const statusSourceSpecZ = specZ.extend({
  variant: z.literal("status-source"),
});
export type StatusSourceSpec = z.infer<typeof statusSourceSpecZ>;

export class Base<P extends z.ZodTypeAny> extends observe.Observer<void> {
  readonly props: z.output<P>;
  schema: P | undefined = undefined;

  constructor(props: unknown) {
    super();
    this.props = prettyParse(this._schema, props);
  }

  private get _schema(): P {
    if (this.schema == null)
      throw new ValidationError(
        `[BaseTelem] - expected subclass to define props schema, but none was found.
    Make sure to define a property 'schema' on the class.`,
      );
    return this.schema;
  }
}

export class AbstractSource<P extends z.ZodTypeAny> extends Base<P> {}
