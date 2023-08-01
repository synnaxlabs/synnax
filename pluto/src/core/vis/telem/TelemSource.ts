// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { GLBufferController, Bounds, Series } from "@synnaxlabs/x";
import { z } from "zod";

import { Color } from "@/core/color";

const transferrable = z.union([
  z.instanceof(ArrayBuffer),
  z.instanceof(OffscreenCanvas),
]);

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export const telemSourceProps = z.object({
  variant: z.string(),
  type: z.string(),
  props: z.any(),
  transfer: z.array(transferrable).optional(),
});

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export type TelemSourceProps = z.infer<typeof telemSourceProps>;

/**
 * Meta data for telemetry source that provides X and Y correlated data.
 */
export const xyTelemSourceProps = telemSourceProps.extend({
  variant: z.literal("xy"),
});

/**
 * Meta data for telemetry source that provides X and Y correlated data.
 */
export type XYTelemSourceProps = z.infer<typeof xyTelemSourceProps>;

/**
 * A telemetry source that provides X and Y correlated data.
 */
export interface XYTelemSource {
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

export const numericTelemSourceProps = telemSourceProps.extend({
  variant: z.literal("numeric"),
});

export type NumericTelemSourceProps = z.infer<typeof numericTelemSourceProps>;

export interface NumericTelemSource {
  value: () => Promise<number>;
  onChange: (f: () => void) => void;
}

export const CrudeColorelemSourceProps = telemSourceProps.extend({
  variant: z.literal("color"),
});

export type CrudeColorelemSourceProps = z.infer<typeof CrudeColorelemSourceProps>;

export interface CrudeColorelemSource {
  value: () => Promise<Color>;
  onChange: (f: () => void) => void;
}

export const booleanTelemSourceProps = telemSourceProps.extend({
  variant: z.literal("boolean"),
});

export type BooleanTelemSinkMeta = z.infer<typeof booleanTelemSourceProps>;

export interface BooleanTelemSource {
  value: () => Promise<boolean>;
  onChange: (f: () => void) => void;
}

export const booleanTelemSinkProps = telemSourceProps.extend({
  variant: z.literal("boolean"),
});

export type BooleanTelemSinkProps = z.infer<typeof booleanTelemSinkProps>;

export interface BooleanTelemSink {
  set: (value: boolean) => void;
}
