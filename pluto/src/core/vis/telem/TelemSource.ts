import { GLBufferControl, Bound, LazyArray, ZERO_BOUND } from "@synnaxlabs/x";

/**
 * Metadata about a telemetry source. This metadata can be thought of as a pointer
 * to the underlying telemetry source, and is intended for use as a main thread proxy
 * to the telemetry source on the worker thread.
 */
export interface TelemSourceMeta {
  /** Key is an application-wide unique identifier for the telemetry source. */
  key: string;
  /** Type defines the type of telemetry source. */
  type: string;
}

/**
 * A telemetry source that provides X and Y correlated data.
 */
export interface XYTelemSourceMeta {
  key: string;
  type: "xy";
}

export interface XYTelemSource extends TelemSourceMeta {
  x: (gl?: GLBufferControl) => Promise<LazyArray[]>;
  y: (gl?: GLBufferControl) => Promise<LazyArray[]>;
  xBound: () => Promise<Bound>;
  yBound: () => Promise<Bound>;
}

export interface DynamicXYTelemSourceMeta {
  key: string;
  type: "dynamic-xy";
}

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
