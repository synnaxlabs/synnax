import { GLBufferControl, Bound, LazyArray } from "@synnaxlabs/x";
import { z } from "zod";

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export const telemSourceMeta = z.object({
  key: z.string(),
  type: z.string(),
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
export const xyTelemSourceMeta = z.object({
  type: z.literal("xy"),
  key: z.string(),
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
   * @param gl - The GLBufferControl to use for buffering the data into
   * the GPU. Data can be cached by the source and only updated when it changes.
   * The GLBufferControl identity does not change throughought the lifetime of the
   * source, and it remains attached to the same rendering context.
   *
   * @returns - lazy arrays that areexpected to have the same topology as the Y axis
   * data i.e. the same number of arrays and the same length for each array.
   */
  x: (gl: GLBufferControl) => Promise<LazyArray[]>;
  /**
   * Resolves data for the Y axis.
   *
   * @param gl - The GLBufferControl to use for buffering the data into
   * the GPU. Data can be cached by the source and only updated when it changes.
   * The GLBufferControl identity does not change throughought the lifetime of the
   * source, and it remains attached to the same rendering context.
   *
   * @returns - lazy arrays that are expected to have the same topology as the X axis
   * data i.e. the same number of arrays and the same length for each array.
   */
  y: (gl: GLBufferControl) => Promise<LazyArray[]>;
  /**
   * @returns the maximum possible bound of the X axis data. This is useful for
   * automatically scaling the X axis of a plot.
   */
  xBound: () => Promise<Bound>;
  /**
   * @returns the maximum possible bound of the Y axis data. This is useful for
   * automatically scaling the Y axis of a plot.
   */
  yBound: () => Promise<Bound>;
}

/**
 * Metadata for an extension of xyTelemSource that allows for the source to call a
 * provided callback when the data changes i.e. request a re-render.
 */
export const dynamicXYTelemSourceMeta = z.object({
  type: z.literal("dynamic-xy"),
  key: z.string(),
});

/**
 * Metadata for an extension of xyTelemSource that allows for the source to call a provided
 * callback when the data changes i.e. request a re-render.
 */
export type DynamicXYTelemSourceMeta = z.infer<typeof dynamicXYTelemSourceMeta>;

/**
 * An extension of xyTelemSource that allows for the source to call a provided
 * callback when the data changes i.e. request a re-render.
 */
export interface DynamicXYTelemSource extends XYTelemSource {
  /**
   * Binds the provided callback to the source, and calls the callback whenever
   * x or y data changes.
   *
   * @param f - The callback to bind to the source.
   */
  onChange: (f: () => void) => void;
}
