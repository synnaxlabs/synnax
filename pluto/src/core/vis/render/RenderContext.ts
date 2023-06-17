// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Destructor, Scale, XY, XYScale } from "@synnaxlabs/x";

import { RenderQueue } from "./RenderQueue";

import { AetherContext } from "@/core/aether/worker";
import { Color } from "@/core/color";

/**
 * A hybrid rendering context containing both 2D and WebGL canvases and contexts.
 * Implements several utility methods for correctly scaling the canvas, restricting
 * drawing to a region, and erasing portions of the canvas.
 */
export class RenderContext {
  /* The canvas element used by WebGL. */
  readonly glCanvas: OffscreenCanvas;

  /** The canvas element used by the 2D canvas. */
  readonly canvasCanvas: OffscreenCanvas;
  /** The webgl rendering context extracted from the canvas */

  /** The WebGL rendering context.  */
  readonly gl: WebGL2RenderingContext;

  /** The 2D canvas rendering context. */
  readonly canvas: OffscreenCanvasRenderingContext2D;

  /** The region the canvas occupies in pixel space */
  region: Box;

  /** The device pixel ratio of the canvas */
  dpr: number;

  /** queue render transitions onto the stack */
  readonly queue: RenderQueue;

  private static readonly CONTEXT_KEY = "pluto-render";

  static create(
    ctx: AetherContext,
    glCanvas: OffscreenCanvas,
    canvasCanvas: OffscreenCanvas
  ): RenderContext {
    const render = new RenderContext(glCanvas, canvasCanvas);
    ctx.set("render", render);
    return render;
  }

  private constructor(glCanvas: OffscreenCanvas, canvasCanvas: OffscreenCanvas) {
    this.canvasCanvas = canvasCanvas;
    this.glCanvas = glCanvas;
    this.queue = new RenderQueue();

    const ctx = canvasCanvas.getContext("2d");
    if (ctx == null) throw new Error("Could not get 2D context");
    this.canvas = ctx;

    const gl = glCanvas.getContext("webgl2", { preserveDrawingBuffer: true });
    if (gl == null) throw new Error("Could not get WebGL context");
    this.gl = gl;

    this.region = Box.ZERO;
    this.dpr = 1;
  }

  static useOptional(ctx: AetherContext): RenderContext | null {
    return ctx.getOptional<RenderContext>(RenderContext.CONTEXT_KEY);
  }

  static use(ctx: AetherContext): RenderContext {
    return ctx.get<RenderContext>(RenderContext.CONTEXT_KEY);
  }

  /**
   * Resizes the canvas to the given region and device pixel ratio. Ensuring
   * that all drawing operations and viewports are scaled correctly.
   */
  resize(region: Box, dpr: number): void {
    this.region = region;
    this.dpr = dpr;
    this.glCanvas.width = region.width * this.dpr;
    this.glCanvas.height = region.height * this.dpr;
    this.canvasCanvas.width = region.width * this.dpr;
    this.canvasCanvas.height = region.height * this.dpr;
    this.canvas.scale(this.dpr, this.dpr);
    this.gl.viewport(0, 0, region.width * dpr, region.height * dpr);
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
    return {
      // Accept a value in decimal.
      x: Scale.scale(0, 1)
        // Turn it to pixels relative to the child width.
        .scale(box.width)
        // Translate the value to the left based on the parent and childs position.
        .translate(box.left)
        // Rebound the scale to the canvas width.
        .reBound(this.region.width)
        // Rescale the value to clip space.
        .scale(-1, 1),
      // Accept a value in decimal.
      y: Scale.scale(0, 1)
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
    };
  }

  /**
   * Scissors the WebGL rendering canvas to the given region.
   *
   * @param region - The region to scissor to. The region should be in
   * pixel space relative to the screen.
   * @returns a destructor that must be called to remove the scissor.
   */
  scissorGL(region: Box): Destructor {
    this.gl.enable(this.gl.SCISSOR_TEST);
    this.gl.scissor(
      (region.left - this.region.left) * this.dpr,
      (this.region.bottom - region.bottom) * this.dpr,
      region.width * this.dpr,
      region.height * this.dpr
    );
    return () => this.gl.disable(this.gl.SCISSOR_TEST);
  }

  /**
   * Erases the given portion of both the 2D and the WebGL canvases.
   *
   * @param region - The region to erase. The region should be in
   * pixel space relative to the screen.
   * @param overscan - The amount to overscan (in pixels) the region by.
   * This is useful for ensuring that the edges of the canvas are cleared
   * when the canvas is scaled.
   */
  erase(region: Box, overscan: XY = XY.ZERO): void {
    this.eraseGL(region, overscan);
    this.eraseCanvas(region, overscan);
  }

  private eraseGL(box: Box, overscan: XY = XY.ZERO): void {
    const { gl } = this;
    const os = new Box(
      (box.left - overscan.x) * this.dpr,
      (box.top - overscan.y) * this.dpr,
      (box.width + overscan.x * 2) * this.dpr,
      (box.height + overscan.y * 2) * this.dpr
    );
    const removeScissor = this.scissorGL(os);
    gl.clearColor(...Color.ZERO.rgba1);
    gl.clear(gl.COLOR_BUFFER_BIT);
    removeScissor();
  }

  private eraseCanvas(box: Box, overscan: XY = XY.ZERO): void {
    const { canvas } = this;
    const os = new Box(
      box.left - overscan.x,
      box.top - overscan.y,
      box.width + overscan.x * 2,
      box.height + overscan.y * 2
    );
    canvas.clearRect(os.left, os.top, os.width, os.height);
  }
}

export type RequestRender = () => void;

export class RenderController {
  f: RequestRender;

  static readonly CONTEXT_KEY = "pluto-vis-renderer";

  private constructor(f: RequestRender) {
    this.f = f;
  }

  static create(ctx: AetherContext, f: RequestRender): void {
    ctx.set(RenderController.CONTEXT_KEY, new RenderController(f));
  }

  static useRequest(ctx: AetherContext): () => void {
    return ctx.get<RenderController>(RenderController.CONTEXT_KEY).f;
  }

  static requestRender(ctx: AetherContext): void {
    this.useRequest(ctx)();
  }
}
