// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type destructor, type dimensions, scale, xy } from "@synnaxlabs/x";

import { type render } from "@/vis/render";

/** A single recorded canvas method call or property assignment. */
export interface Call {
  op: string;
  args: unknown[];
}

/** Return values for the canvas methods that callers read a result from. Everything
 * else records and returns undefined. */
const PASSTHROUGH: Record<string, unknown> = {
  measureText: { width: 8 },
  getImageData: { data: new Uint8ClampedArray() },
};

/** Style/state properties seeded so reads before a write return canvas-shaped defaults
 * instead of a recording stub. Mirrors the defaults a real 2D context exposes. */
const STYLE_DEFAULTS: Record<string, unknown> = {
  fillStyle: "",
  strokeStyle: "",
  lineWidth: 1,
  lineCap: "butt",
  lineJoin: "miter",
  lineDashOffset: 0,
  globalAlpha: 1,
  font: "",
  textAlign: "start",
  textBaseline: "alphabetic",
};

/** Recording analog of `SugaredOffscreenCanvasRenderingContext2D`. It is itself the
 * drawing surface — every method call and property assignment is captured in
 * {@link calls} in encounter order — so it can stand in directly wherever production
 * reads `render.Context.lower2d` / `upper2d` (e.g. constructing a `Draw2D`). */
export interface RecordingCanvas {
  /** Recorded method calls and property sets, in encounter order. */
  calls: Call[];
  /** Restrict drawing to a region; records the call and returns a no-op destructor. */
  scissor(region: box.Box, overScan?: xy.XY): destructor.Destructor;
  /** Records the call and returns this same recording surface, so chained draw calls
   * land on the same {@link calls} list (production returns a scaled sub-context). */
  applyScale(scale: scale.XY): RecordingCanvas;
  /** Records the call and returns a deterministic size derived from the label length, so
   * width-dependent layout logic under test gets stable, varied inputs. */
  textDimensions(label: string, options?: unknown): dimensions.Dimensions;
  [op: string]: unknown;
}

const buildCanvas = (width = 800, height = 600): RecordingCanvas => {
  const calls: Call[] = [];
  const target: Record<string, unknown> = {
    canvas: { width, height },
    calls,
    scissor: (region: box.Box, overScan: xy.XY = xy.ZERO) => {
      calls.push({ op: "scissor", args: [region, overScan] });
      return () => {};
    },
    ...STYLE_DEFAULTS,
  };
  const handler: ProxyHandler<Record<string, unknown>> = {
    get(t, prop) {
      if (typeof prop !== "string") return undefined;
      if (prop in t) return t[prop];
      const fn = (...args: unknown[]): unknown => {
        calls.push({ op: prop, args });
        return PASSTHROUGH[prop];
      };
      t[prop] = fn;
      return fn;
    },
    set(t, prop, value) {
      if (typeof prop !== "string") return false;
      calls.push({ op: `set:${prop}`, args: [value] });
      t[prop] = value;
      return true;
    },
  };
  const proxy = new Proxy(target, handler) as unknown as RecordingCanvas;
  target.applyScale = (s: scale.XY): RecordingCanvas => {
    calls.push({ op: "applyScale", args: [s] });
    return proxy;
  };
  target.textDimensions = (label: string): dimensions.Dimensions => {
    calls.push({ op: "textDimensions", args: [label] });
    return { width: label.length * 8, height: 12 };
  };
  return proxy;
};

/** A single `scissor()` call recorded on the recorder. */
export interface ScissorCall {
  region: box.Box;
  overScan: xy.XY;
  canvases: render.CanvasVariant[];
}

/** A single `erase()` call recorded on the recorder. */
export interface EraseCall {
  region: box.Box;
  overScan: xy.Crude;
  canvases: render.CanvasVariant[];
}

/** A single `loop.set()` call recorded on the recorder. */
export interface LoopCall {
  args: unknown[];
}

/** Duck-typed render context that records every component-visible call.
 *
 * Pass as the `render` option to `renderAether` (or `render`) to make a component under
 * test see this recorder as its `render.Context`. The 2D canvases are themselves
 * recording drawing surfaces, so components that draw (via `Draw2D`) work unchanged;
 * recorded data is available on the recorder for assertion afterward. */
export class Recorder {
  readonly upper2d: RecordingCanvas;
  readonly lower2d: RecordingCanvas;
  readonly gl: RecordingCanvas;

  /** Calls to `Recorder.scissor` in encounter order. */
  readonly scissorCalls: ScissorCall[] = [];
  /** Calls to `Recorder.erase` in encounter order. */
  readonly eraseCalls: EraseCall[] = [];
  /** Calls to `Recorder.loop.set` in encounter order. */
  readonly loopCalls: LoopCall[] = [];

  region: box.Box = box.ZERO;
  dpr: number = 1;

  readonly loop = {
    set: (...args: unknown[]): void => {
      this.loopCalls.push({ args });
    },
  };

  constructor() {
    this.upper2d = buildCanvas();
    this.lower2d = buildCanvas();
    this.gl = buildCanvas();
  }

  /** Resets every recording — the per-canvas calls and the scissor/erase/loop lists.
   * Region and dpr are left as-is. Use between phases of a test when you only care
   * about calls made since `clear`. */
  clear(): void {
    this.upper2d.calls.length = 0;
    this.lower2d.calls.length = 0;
    this.gl.calls.length = 0;
    this.scissorCalls.length = 0;
    this.eraseCalls.length = 0;
    this.loopCalls.length = 0;
  }

  scissor(
    region: box.Box,
    overScan: xy.XY = xy.ZERO,
    canvases: render.CanvasVariant[] = ["upper2d", "lower2d", "gl"],
  ): destructor.Destructor {
    this.scissorCalls.push({ region, overScan, canvases });
    return () => {};
  }

  erase(
    region: box.Box,
    overScan: xy.Crude = xy.ZERO,
    ...canvases: render.CanvasVariant[]
  ): void {
    const cs: render.CanvasVariant[] =
      canvases.length === 0 ? ["upper2d", "lower2d", "gl"] : canvases;
    this.eraseCalls.push({ region, overScan, canvases: cs });
  }

  resize(region: box.Box, dpr: number): void {
    this.region = region;
    this.dpr = dpr;
  }

  get aspect(): number {
    return box.aspect(this.region);
  }

  scaleRegion(_b: box.Box): scale.XY {
    return new scale.XY(scale.Scale.IDENTITY, scale.Scale.IDENTITY);
  }
}
