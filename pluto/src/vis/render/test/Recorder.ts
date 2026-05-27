// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type destructor, scale, xy } from "@synnaxlabs/x";

import { type render } from "@/vis/render";

/** A single recorded canvas method call or property assignment. */
export interface Call {
  op: string;
  args: unknown[];
}

const PASSTHROUGH: Record<string, unknown> = {
  measureText: { width: 8 },
  getImageData: { data: new Uint8ClampedArray() },
};

const buildCanvasRecorder = (
  width = 800,
  height = 600,
): { ctx: unknown; calls: Call[] } => {
  const calls: Call[] = [];
  const state: Record<string, unknown> = {
    canvas: { width, height },
  };
  const handler: ProxyHandler<Record<string, unknown>> = {
    get(target, prop) {
      if (typeof prop !== "string") return undefined;
      if (prop in target) return target[prop];
      const fn = (...args: unknown[]): unknown => {
        calls.push({ op: prop, args });
        return PASSTHROUGH[prop];
      };
      target[prop] = fn;
      return fn;
    },
    set(target, prop, value) {
      if (typeof prop !== "string") return false;
      calls.push({ op: `set:${prop}`, args: [value] });
      target[prop] = value;
      return true;
    },
  };
  return { ctx: new Proxy(state, handler), calls };
};

/** Recording analog of `SugaredOffscreenCanvasRenderingContext2D`. Every method and
 * property access is captured in {@link calls} in encounter order. */
export interface RecordingCanvas2D {
  readonly ctx: unknown;
  readonly calls: Call[];
  scissor(region: box.Box, overScan?: xy.XY): destructor.Destructor;
}

/** Recording analog of `WebGL2RenderingContext`. Smoke-level: every call is captured
 * but no real GL state is maintained. Sufficient for "did Line attempt to render?"
 * assertions; not sufficient for pixel-level WebGL testing. */
export interface RecordingGL {
  readonly ctx: unknown;
  readonly calls: Call[];
}

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
 * Drop into {@link aetherTest.mount}'s `renderContext` option to make a component
 * under test see this recorder as its `render.Context`. Recorded data is available on
 * the recorder for assertion after the test drives whatever lifecycle steps it cares
 * about. */
export class Recorder {
  readonly upper2d: RecordingCanvas2D;
  readonly lower2d: RecordingCanvas2D;
  readonly gl: RecordingGL;

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
    const upper = buildCanvasRecorder();
    const lower = buildCanvasRecorder();
    const gl = buildCanvasRecorder();
    this.upper2d = {
      ctx: upper.ctx,
      calls: upper.calls,
      scissor: (region, overScan = xy.ZERO) => {
        upper.calls.push({ op: "scissor", args: [region, overScan] });
        return () => {};
      },
    };
    this.lower2d = {
      ctx: lower.ctx,
      calls: lower.calls,
      scissor: (region, overScan = xy.ZERO) => {
        lower.calls.push({ op: "scissor", args: [region, overScan] });
        return () => {};
      },
    };
    this.gl = { ctx: gl.ctx, calls: gl.calls };
  }

  /** Resets every recording (calls, scissor/erase/loop) and the recorded region/dpr.
   * Use between phases of a test when you only care about calls made since `clear`. */
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
