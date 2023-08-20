// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { GLBufferController, Bounds, Series } from "@synnaxlabs/x";
import { z } from "zod";

import { Color } from "@/color";

const transferrable = z.union([
  z.instanceof(ArrayBuffer),
  z.instanceof(OffscreenCanvas),
]);

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export const telemSpec = z.object({
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
export type TelemSpec = z.infer<typeof telemSpec>;

export const xyTelemSourceSpec = telemSpec.extend({
  variant: z.literal("xy-source"),
});

export type XYTelemSourceSpec = z.infer<typeof xyTelemSourceSpec>;

export interface Telem {
  setProps: (props: any) => void;
  cleanup: () => void;
  invalidate: () => void;
}

/**
 * A telemetry source that provides X and Y correlated data.
 */
export interface XYTelemSource extends Telem {
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
  xBounds: () => Promise<Bounds>;
  /**
   * @returns the maximum possible bound of the Y axis data. This is useful for
   * automatically scaling the Y axis of a plot.
   */
  yBounds: () => Promise<Bounds>;
  /**
   * Binds the provided callback to the source, and calls the callback whenever
   * x or y data changes.
   *
   * @param f - The callback to bind to the source.
   */
  onChange: (f: () => void) => void;
}

export const numericTelemSourceSpec = telemSpec.extend({
  variant: z.literal("numeric-source"),
});

export type NumericTelemSourceSpec = z.infer<typeof numericTelemSourceSpec>;

export interface NumericTelemSource extends Telem {
  value: () => Promise<number>;
  onChange: (f: () => void) => void;
}

export const colorTelemSourceSpec = telemSpec.extend({
  variant: z.literal("color-source"),
});

export type ColorTelemSourceSpec = z.infer<typeof colorTelemSourceSpec>;

export interface ColorTelemSource extends Telem {
  value: () => Promise<Color>;
  onChange: (f: () => void) => void;
}

export const booleanTelemSourceSpec = telemSpec.extend({
  variant: z.literal("boolean-source"),
});

export type BooleanTelemSourceSpec = z.infer<typeof booleanTelemSourceSpec>;

export interface BooleanTelemSource extends Telem {
  value: () => Promise<boolean>;
  onChange: (f: () => void) => void;
}

export const booleanTelemSinkSpec = telemSpec.extend({
  variant: z.literal("boolean-sink"),
});

export interface BooleanTelemSink extends Telem {
  set: (value: boolean) => Promise<void>;
}

export type BooleanTelemSinkSpec = z.infer<typeof booleanTelemSinkSpec>;

export interface NumericTelemSink extends Telem {
  set: (value: number) => Promise<void>;
}

export const numericTelemSinkSpec = telemSpec.extend({
  variant: z.literal("numeric-sink"),
});

export type NumericTelemSinkSpec = z.infer<typeof numericTelemSinkSpec>;
