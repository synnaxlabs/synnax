// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { type bounds, type GLBufferController, type Series } from "@synnaxlabs/x";
import { z } from "zod";

import { type color } from "@/color/core";

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
  setProps: (props: any) => void;
  cleanup: () => void;
  invalidate: () => void;
}

/**
 * A telemetry source that provides X and Y correlated data.
 */
export interface XYSource extends Telem {
  /**
   * Resolves data for the X axis.
   *
   * @param gl - The GLBufferController to use for buffering the data into
   * the GPU. Data can be cached by the source and only updated when it changes.
   * The GLBufferController identity does not change throughought the lifetime of the
   * source, and it remains attached to the same rendering context.
   *
   * @returns - series expected to have the same topology as the Y axis
   * data i.e. the same number of arrays and the same length for each array.
   */
  x: (gl: GLBufferController) => Promise<Series[]>;
  /**
   * Resolves data for the Y axis.
   *
   * @param gl - The GLBufferController to use for buffering the data into
   * the GPU. Data can be cached by the source and only updated when it changes.
   * The GLBufferController identity does not change throughought the lifetime of the
   * source, and it remains attached to the same rendering context.
   *
   * @returns - lazy arrays that are expected to have the same topology as the X axis
   * data i.e. the same number of arrays and the same length for each array.
   */
  y: (gl: GLBufferController) => Promise<Series[]>;
  /**
   * @returns the maximum possible bound of the X axis data. This is useful for
   * automatically scaling the X axis of a plot.
   */
  xBounds: () => Promise<bounds.Bounds>;
  /**
   * @returns the maximum possible bound of the Y axis data. This is useful for
   * automatically scaling the Y axis of a plot.
   */
  yBounds: () => Promise<bounds.Bounds>;
  /**
   * Binds the provided callback to the source, and calls the callback whenever
   * x or y data changes.
   *
   * @param f - The callback to bind to the source.
   */
  onChange: (f: () => void) => void;
}

export const numericSourceSpecZ = specZ.extend({
  variant: z.literal("numeric-source"),
});

export type NumericSourceSpec = z.infer<typeof numericSourceSpecZ>;

export interface NumericSource extends Telem {
  value: () => Promise<number>;
  onChange: (f: () => void) => void;
}

export const colorSourceSpecZ = specZ.extend({
  variant: z.literal("color-source"),
});

export type ColorSourceSpec = z.infer<typeof colorSourceSpecZ>;

export interface ColorSource extends Telem {
  value: () => Promise<color.Color>;
  onChange: (f: () => void) => void;
}

export const booleanSourceSpecZ = specZ.extend({
  variant: z.literal("boolean-source"),
});

export type BooleanSourceSpec = z.infer<typeof booleanSourceSpecZ>;

export interface BooleanSource extends Telem {
  value: () => Promise<boolean>;
  onChange: (f: () => void) => void;
}

export const booleanSinkSpecZ = specZ.extend({
  variant: z.literal("boolean-sink"),
});

export interface BooleanSink extends Telem {
  set: (value: boolean) => Promise<void>;
}

export type BooleanSinkSpec = z.infer<typeof booleanSinkSpecZ>;

export interface NumericSink extends Telem {
  set: (value: number) => Promise<void>;
}

export const numericSinkSpecZ = specZ.extend({
  variant: z.literal("numeric-sink"),
});

export type NumericSinkSpec = z.infer<typeof numericSinkSpecZ>;
