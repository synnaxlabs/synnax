// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, type Destructor, Scale, XY, XYScale } from "@synnaxlabs/x";

import { type aether } from "@/aether/aether";
import { color } from "@/color/core";
import { CSS } from "@/css";
import { SugaredOffscreenCanvasRenderingContext2D } from "@/vis/draw2d/canvas";
import { Queue } from "@/vis/render/queue";

export type CanvasVariant = "upper2d" | "lower2d" | "gl";

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
  readonly lower2d: SugaredOffscreenCanvasRenderingContext2D;

  /** A 2D canvas that sits above the WebGL canvas. */
  readonly upper2d: SugaredOffscreenCanvasRenderingContext2D;

  /** The region the canvas occupies in pixel space */
  region: Box;

  /** The device pixel ratio of the canvas */
  dpr: number;

  /** queue render transitions onto the stack */
  readonly queue: Queue;

  private static readonly CONTEXT_KEY = CSS.B("render-context");

  static create(
    ctx: aether.Context,
    glCanvas: OffscreenCanvas,
    lower2dCanvas: OffscreenCanvas,
    upper2dCanvas: OffscreenCanvas,
  ): Context {
    const render = new Context(glCanvas, lower2dCanvas, upper2dCanvas);
    ctx.set(Context.CONTEXT_KEY, render);
    return render;
  }

  private constructor(
    glCanvas: OffscreenCanvas,
    lower2dCanvas: OffscreenCanvas,
    upper2dCanvas: OffscreenCanvas,
  ) {
    this.upper2dCanvas = upper2dCanvas;
    this.lower2dCanvas = lower2dCanvas;
    this.glCanvas = glCanvas;
    this.queue = new Queue();

    const lowerCtx = this.lower2dCanvas.getContext("2d");
    if (lowerCtx == null) throw new Error("Could not get 2D context");
    this.lower2d = new SugaredOffscreenCanvasRenderingContext2D(lowerCtx);

    const upperCtx = this.upper2dCanvas.getContext("2d");
    if (upperCtx == null) throw new Error("Could not get 2D context");
    this.upper2d = new SugaredOffscreenCanvasRenderingContext2D(upperCtx);

    const gl = this.glCanvas.getContext("webgl2", {
      preserveDrawingBuffer: true,
      antialias: true,
    });
    if (gl == null) throw new Error("Could not get WebGL context");
    this.gl = gl;

    this.region = Box.ZERO;
    this.dpr = 1;
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
  resize(region: Box, dpr: number): void {
    if (this.region.equals(region) && this.dpr === dpr) return;
    this.region = region;
    this.dpr = dpr;
    this.resizeCanvas(this.glCanvas);
    this.resizeCanvas(this.upper2dCanvas);
    this.resizeCanvas(this.lower2dCanvas);
    this.lower2d.scale(this.dpr, this.dpr);
    this.upper2d.scale(this.dpr, this.dpr);
    this.gl.viewport(0, 0, region.width * dpr, region.height * dpr);
  }

  private resizeCanvas(canvas: OffscreenCanvas): void {
    canvas.width = this.region.width * this.dpr;
    canvas.height = this.region.height * this.dpr;
  }

  /** @returns the aspect ratio of the canvas. */
  get aspect(): number {
    return this.region.width / this.region.height;
  }

  /**
   * Takes the given box in PIXEL space and produces a transform
   * in CLIP space representing the sub-region represented by the box
   * in the canvas.
   */
  scaleRegion(box: Box): XYScale {
    return new XYScale(
      // Accept a value in decimal.
      Scale.scale(0, 1)
        // Turn it to pixels relative to the child width.
        .scale(box.width)
        // Translate the value to the left based on the parent and childs position.
        .translate(box.left)
        // Rebound the scale to the canvas width.
        .reBound(this.region.width)
        // Rescale the value to clip space.
        .scale(-1, 1),
      // Accept a value in decimal.
      Scale.scale(0, 1)
        // Turn it to pixels relative to the child height.
        .scale(box.height)
        // Invert the scale since we read pixels from the top.
        .invert()
        // Translate the value to the top based on the parent and childs position.
        .translate(box.top)
        // Rebound the scale to the canvas height.
        .reBound(this.region.height)
        // Rescale the value to clip space.
        .scale(-1, 1)
        // Invert the scale since we read clip space from the bottom.
        .invert(),
    );
  }

  scissor(region: Box, overscan: XY = XY.ZERO, canvases: CanvasVariant[]): Destructor {
    const destructor: Destructor[] = [];
    if (canvases.includes("upper2d"))
      destructor.push(this.scissorCanvas(this.upper2d, region, overscan));
    if (canvases.includes("lower2d"))
      destructor.push(this.scissorCanvas(this.lower2d, region, overscan));
    if (canvases.includes("gl")) destructor.push(this.scissorGL(region, overscan));
    return () => destructor.forEach((d) => d());
  }

  private scissorCanvas(
    c: OffscreenCanvasRenderingContext2D,
    region: Box,
    overscan: XY = XY.ZERO,
  ): Destructor {
    const p = new Path2D();
    region = applyOverscan(region, overscan);
    p.rect(...region.topLeft.couple, ...region.dims.couple);
    c.save();
    c.clip(p);
    return () => c.restore();
  }

  private scissorGL(region: Box, overscan: XY = XY.ZERO): Destructor {
    this.gl.enable(this.gl.SCISSOR_TEST);
    region = applyOverscan(region, overscan);
    this.gl.scissor(
      (region.left - this.region.left) * this.dpr,
      (this.region.bottom - region.bottom) * this.dpr,
      region.width * this.dpr,
      region.height * this.dpr,
    );
    return () => this.gl.disable(this.gl.SCISSOR_TEST);
  }

  erase(region: Box, overscan: XY = XY.ZERO, ...canvases: CanvasVariant[]): void {
    if (canvases.length === 0) canvases = ["upper2d", "lower2d", "gl"];
    if (canvases.includes("upper2d")) this.eraseCanvas(this.upper2d, region, overscan);
    if (canvases.includes("lower2d")) this.eraseCanvas(this.lower2d, region, overscan);
    if (canvases.includes("gl")) this.eraseGL(region, overscan);
  }

  private eraseGL(box: Box, overscan: XY = XY.ZERO): void {
    const { gl } = this;
    const removeScissor = this.scissorGL(applyOverscan(box, overscan));
    gl.clearColor(...color.ZERO.rgba1);
    gl.clear(gl.COLOR_BUFFER_BIT);
    removeScissor();
  }

  private eraseCanvas(
    c: OffscreenCanvasRenderingContext2D,
    box: Box,
    overscan: XY = XY.ZERO,
  ): void {
    const os = applyOverscan(box, overscan);
    c.clearRect(...os.topLeft.couple, ...os.dims.couple);
  }
}

const applyOverscan = (box: Box, overscan: XY): Box =>
  new Box(
    box.left - overscan.x,
    box.top - overscan.y,
    box.width + overscan.x * 2,
    box.height + overscan.y * 2,
  );
