// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import {
  box,
  color,
  type destructor,
  dimensions,
  type runtime,
  scale,
  xy,
} from "@synnaxlabs/x";

import { type aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { CSS } from "@/css";
import { status } from "@/status/aether";
import { SugaredOffscreenCanvasRenderingContext2D } from "@/vis/draw2d/canvas";
import { clear } from "@/vis/render/clear";
import { Loop } from "@/vis/render/loop";
import { applyOverScan } from "@/vis/render/util";
import { text } from "@/vis/text";

export type Canvas2DVariant = "upper2d" | "lower2d";
export type CanvasGLVariant = "gl";
export type CanvasVariant = Canvas2DVariant | CanvasGLVariant;

const applyDefaultCanvasOpts = <T extends OffscreenCanvasRenderingContext2D>(
  canvas: T,
): T => {
  canvas.textRendering = "optimizeSpeed";
  canvas.imageSmoothingEnabled = false;
  return canvas;
};

/**
 * A hybrid rendering context containing both 2D and WebGL canvases and contexts.
 * Implements several utility methods for correctly scaling the canvas, restricting
 * drawing to a region, and erasing portions of the canvas.
 */
export class Context {
  /* The canvas element used by WebGL. */
  readonly glCanvas: OffscreenCanvas;
  /** The canvas element used by the 2D canvas. */
  readonly upper2dCanvas: OffscreenCanvas;
  /** The canvas element used by the 2D canvas. */
  readonly lower2dCanvas: OffscreenCanvas;
  /** The WebGL rendering context.  */
  readonly gl: WebGL2RenderingContext;
  /** A 2D canvas that sits below the WebGL canvas. */
  lower2d: SugaredOffscreenCanvasRenderingContext2D;
  /** A 2D canvas that sits above the WebGL canvas. */
  upper2d: SugaredOffscreenCanvasRenderingContext2D;
  /** The region the canvas occupies in pixel space */
  region: box.Box;
  /** The device pixel ratio of the canvas */
  dpr: number;
  /** queue render transitions onto the stack */
  readonly loop: Loop;
  /** See the @link{clear.Program} for why this is necessary. */
  private readonly clearProgram?: clear.Program;
  private readonly os: runtime.OS;
  private static readonly CONTEXT_KEY = CSS.B("render-context");
  private readonly instrumentation: Instrumentation;

  static create(
    ctx: aether.Context,
    glCanvas: OffscreenCanvas,
    lower2dCanvas: OffscreenCanvas,
    upper2dCanvas: OffscreenCanvas,
    os: runtime.OS,
  ): Context {
    const instrumentation = alamos.useInstrumentation(ctx, "render_context");
    const render = new Context(
      glCanvas,
      lower2dCanvas,
      upper2dCanvas,
      os,
      instrumentation,
      status.useAdder(ctx),
      status.useErrorHandler(ctx),
    );
    ctx.set(Context.CONTEXT_KEY, render);
    return render;
  }

  private constructor(
    glCanvas: OffscreenCanvas,
    lower2dCanvas: OffscreenCanvas,
    upper2dCanvas: OffscreenCanvas,
    os: runtime.OS,
    instrumentation: Instrumentation,
    addStatus: status.Adder,
    handleError: status.ErrorHandler,
  ) {
    this.upper2dCanvas = upper2dCanvas;
    this.lower2dCanvas = lower2dCanvas;
    this.glCanvas = glCanvas;
    this.os = os;
    this.instrumentation = instrumentation;
    this.dpr = 1;

    const lowerCtx = this.lower2dCanvas.getContext("2d");
    if (lowerCtx == null) throw new Error("Could not get 2D context");
    const atlas = new text.AtlasRegistry();
    this.lower2d = applyDefaultCanvasOpts(
      new SugaredOffscreenCanvasRenderingContext2D(lowerCtx, atlas, this.dpr),
    );

    const upperCtx = this.upper2dCanvas.getContext("2d");
    if (upperCtx == null) throw new Error("Could not get 2D context");
    this.upper2d = applyDefaultCanvasOpts(
      new SugaredOffscreenCanvasRenderingContext2D(upperCtx, atlas, this.dpr),
    );

    const webGlOpts: WebGLContextAttributes = {
      preserveDrawingBuffer: true,
      powerPreference: "high-performance",
      stencil: false,
      depth: false,
    };
    const gl = this.glCanvas.getContext("webgl2", webGlOpts);
    if (gl == null) throw new Error("Could not get WebGL context");
    gl.disable(gl.DEPTH_TEST);
    gl.disable(gl.STENCIL_TEST);
    gl.disable(gl.CULL_FACE);
    gl.disable(gl.DITHER);
    this.gl = gl;

    this.glCanvas.oncontextlost = () =>
      addStatus({
        variant: "error",
        message: "WebGL context lost",
      });

    const afterRender = () => {
      this.gl.flush();
      this.gl.finish();
    };

    this.loop = new Loop({
      afterRender,
      instrumentation: this.instrumentation,
      handleError,
    });

    this.region = box.ZERO;

    if (this.os === "Windows") this.clearProgram = new clear.Program(this);
  }

  static useOptional(ctx: aether.Context): Context | null {
    return ctx.getOptional<Context>(Context.CONTEXT_KEY);
  }

  static use(ctx: aether.Context): Context {
    return ctx.get<Context>(Context.CONTEXT_KEY);
  }

  update(ctx: aether.Context): void {
    ctx.set(Context.CONTEXT_KEY, this);
  }

  /**
   * Resizes the canvas to the given region and device pixel ratio. Ensuring
   * that all drawing operations and viewports are scaled correctly.
   */
  resize(region: box.Box, dpr: number): void {
    if (box.equals(this.region, region) && this.dpr === dpr) return;
    this.region = region;
    this.dpr = dpr;
    this.resizeCanvas(this.glCanvas);
    this.resizeCanvas(this.upper2dCanvas);
    this.resizeCanvas(this.lower2dCanvas);
    this.lower2d.scale(this.dpr, this.dpr);
    this.upper2d.scale(this.dpr, this.dpr);
    this.gl.viewport(0, 0, box.width(region) * dpr, box.height(region) * dpr);
  }

  private resizeCanvas(canvas: OffscreenCanvas): void {
    canvas.width = box.width(this.region) * this.dpr;
    canvas.height = box.height(this.region) * this.dpr;
  }

  /** @returns the aspect ratio of the canvas. */
  get aspect(): number {
    return box.aspect(this.region);
  }

  /**
   * Takes the given box in PIXEL space and produces a transform
   * in CLIP space representing the sub-region represented by the box
   * in the canvas.
   */
  scaleRegion(b: box.Box): scale.XY {
    return new scale.XY(
      // Accept a value in decimal.
      scale.Scale.scale<number>(0, 1)
        // Turn it to pixels relative to the child width.
        .scale(box.width(b))
        // Translate the value to the left based on the parent and child's position.
        .translate(box.left(b))
        // Rebound the scale to the canvas width.
        .reBound(box.width(this.region))
        // Rescale the value to clip space.
        .scale(-1, 1),
      // Accept a value in decimal.
      scale.Scale.scale<number>(0, 1)
        // Turn it to pixels relative to the child height.
        .scale(box.height(b))
        // Invert the scale since we read pixels from the top.
        .invert()
        // Translate the value to the top based on the parent and child's position.
        .translate(box.top(b))
        // Rebound the scale to the canvas height.
        .reBound(box.height(this.region))
        // Rescale the value to clip space.
        .scale(-1, 1)
        // Invert the scale since we read clip space from the bottom.
        .invert(),
    );
  }

  scissor(
    region: box.Box,
    overScan: xy.XY = xy.ZERO,
    canvases: CanvasVariant[],
  ): destructor.Destructor {
    const destructor: destructor.Destructor[] = [];
    if (canvases.includes("upper2d"))
      destructor.push(this.upper2d.scissor(region, overScan));
    if (canvases.includes("lower2d"))
      destructor.push(this.lower2d.scissor(region, overScan));
    if (canvases.includes("gl")) destructor.push(this.scissorGL(region, overScan));
    return () => destructor.forEach((d) => d());
  }

  private scissorGL(region: box.Box, overscan: xy.XY = xy.ZERO): destructor.Destructor {
    this.gl.enable(this.gl.SCISSOR_TEST);
    region = applyOverScan(region, overscan);
    this.gl.scissor(
      box.left(region) * this.dpr,
      (box.height(this.region) - box.bottom(region)) * this.dpr,
      box.width(region) * this.dpr,
      box.height(region) * this.dpr,
    );
    return () => this.gl.disable(this.gl.SCISSOR_TEST);
  }

  erase(
    region: box.Box,
    overscan: xy.Crude = xy.ZERO,
    ...canvases: CanvasVariant[]
  ): void {
    overscan = xy.construct(overscan);
    if (canvases.length === 0) canvases = ["upper2d", "lower2d", "gl"];
    if (canvases.includes("upper2d")) this.eraseCanvas(this.upper2d, region, overscan);
    if (canvases.includes("lower2d")) this.eraseCanvas(this.lower2d, region, overscan);
    if (canvases.includes("gl")) this.eraseGL(region, overscan);
  }

  private eraseGL(box: box.Box, overscan: xy.XY = xy.ZERO): void {
    const { gl } = this;
    const removeScissor = this.scissorGL(applyOverScan(box, overscan));
    gl.clearColor(...color.rgba1(color.ZERO));
    gl.clear(gl.COLOR_BUFFER_BIT);
    // See the documentation for the clear program for why this is necessary.
    if (this.os === "Windows") this.clearProgram?.exec();
    removeScissor();
  }

  private eraseCanvas(
    c: OffscreenCanvasRenderingContext2D,
    b: box.Box,
    overscan: xy.XY = xy.ZERO,
  ): void {
    const os = applyOverScan(b, overscan);
    c.clearRect(...xy.couple(box.topLeft(os)), ...dimensions.couple(box.dims(os)));
  }
}
