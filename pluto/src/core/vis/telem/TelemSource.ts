import { GLBufferControl, Bound, LazyArray, ZERO_BOUND } from "@synnaxlabs/x";
import { z } from "zod";

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

export const xyTelemSourceMeta = z.object({
  type: z.literal("xy"),
  key: z.string(),
});

/**
 * A telemetry source that provides X and Y correlated data.
 */
export type XYTelemSourceMeta = z.infer<typeof xyTelemSourceMeta>;

export interface XYTelemSource extends TelemSourceMeta {
  x: (gl?: GLBufferControl) => Promise<LazyArray[]>;
  y: (gl?: GLBufferControl) => Promise<LazyArray[]>;
  xBound: () => Promise<Bound>;
  yBound: () => Promise<Bound>;
}

export const dynamicXYTelemSourceMeta = z.object({
  type: z.literal("dynamic-xy"),
  key: z.string(),
});

export type DynamicXYTelemSourceMeta = z.infer<typeof dynamicXYTelemSourceMeta>;

export interface DynamicXYTelemSource extends XYTelemSource {
  onChange: (f: () => void) => void;
}

class EmptyStaticXYTelem implements XYTelemSource {
  key = "empty";
  type = "empty";

  async x(): Promise<LazyArray[]> {
    return [];
  }

  async y(): Promise<LazyArray[]> {
    return [];
  }

  async xBound(): Promise<Bound> {
    return ZERO_BOUND;
  }

  async yBound(): Promise<Bound> {
    return ZERO_BOUND;
  }
}

export const ZERO_XY_TELEM = new EmptyStaticXYTelem();
