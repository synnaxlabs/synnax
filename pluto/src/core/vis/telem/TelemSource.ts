import { GLBufferController, Bounds, LazyArray } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export const telemSourceMeta = z.object({
  key: z.string(),
  variant: z.string(),
});

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export type TelemSourceMeta = z.infer<typeof telemSourceMeta>;

/**
 * Meta data for telemetry source that provides X and Y correlated data.
 */
export const xyTelemSourceMeta = telemSourceMeta.extend({
  variant: z.literal("xy"),
});

/**
 * Meta data for telemetry source that provides X and Y correlated data.
 */
export type XYTelemSourceMeta = z.infer<typeof xyTelemSourceMeta>;

/**
 * A telemetry source that provides X and Y correlated data.
 */
export interface XYTelemSource extends TelemSourceMeta {
  /**
   * Resolves data for the X axis.
   *
   * @param gl - The GLBufferController to use for buffering the data into
   * the GPU. Data can be cached by the source and only updated when it changes.
   * The GLBufferController identity does not change throughought the lifetime of the
   * source, and it remains attached to the same rendering context.
   *
   * @returns - lazy arrays that areexpected to have the same topology as the Y axis
   * data i.e. the same number of arrays and the same length for each array.
   */
  x: (gl: GLBufferController) => Promise<LazyArray[]>;
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
  y: (gl: GLBufferController) => Promise<LazyArray[]>;
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
  /**
   *
   */
  release: (gl: GLBufferController) => void;
}
