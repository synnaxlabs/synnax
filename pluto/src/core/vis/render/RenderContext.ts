// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, Destructor, Scale, XY, XYScale } from "@synnaxlabs/x";

import { Color } from "@/core/color";

export class RenderContext {
  /* The canvas element */
  readonly glCanvas: OffscreenCanvas;
  /** The webgl rendering context extracted from the canvas */
  readonly gl: WebGL2RenderingContext;
  readonly canvas: OffscreenCanvasRenderingContext2D;
  readonly canvasCanvas: OffscreenCanvas;
  /** The region the canvas occupies in pixel space */
  region: Box;
  /** The device pixel ratio of the canvas */
  dpr: number;

  constructor(
    glCanvas: OffscreenCanvas,
    canvasCanvas: OffscreenCanvas,
    region: Box,
    dpr: number
  ) {
    this.canvasCanvas = canvasCanvas;
    this.glCanvas = glCanvas;
    this.region = region;
    const ctx = canvasCanvas.getContext("2d");
    if (ctx == null) throw new Error("Could not get 2D context");
    this.canvas = ctx;
    const gl = glCanvas.getContext("webgl2", { preserveDrawingBuffer: true });
    if (gl == null) throw new Error("Could not get WebGL context");
    this.gl = gl;
    this.dpr = dpr;
    this.resize(region, dpr);
  }

  resize(region: Box, dpr: number): void {
    this.region = region;
    this.dpr = dpr;
    this.glCanvas.width = region.width * dpr;
    this.glCanvas.height = region.height * dpr;
    this.canvasCanvas.width = region.width * dpr;
    this.canvasCanvas.height = region.height * dpr;
    this.canvas.scale(dpr, dpr);
    this.gl.viewport(0, 0, region.width * dpr, region.height * dpr);
  }

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
        .translate(this.region.left + box.left)
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
        .translate(this.region.top + box.top)
        // Rebound the scale to the canvas height.
        .reBound(this.region.height)
        // Rescale the value to clip space.
        .scale(-1, 1)
        // Invert the scale since we read clip space from the bottom.
        .invert(),
    };
  }

  scissor(box: Box): Destructor {
    const scale = this.scaleRegion(box);
    this.gl.enable(this.gl.SCISSOR_TEST);
    this.gl.scissor(scale.x.pos(0), scale.y.pos(0), scale.x.dim(1), scale.y.dim(1));
    return () => this.gl.disable(this.gl.SCISSOR_TEST);
  }

  erase(box: Box, overscan: XY = XY.ZERO): void {
    this.eraseGL(box, overscan);
    this.eraseCanvas(box, overscan);
  }

  private eraseGL(box: Box, overscan: XY = XY.ZERO): void {
    const { gl } = this;
    const os = new Box(
      box.left - overscan.x,
      box.top - overscan.y,
      box.width + overscan.x * 2,
      box.height + overscan.y * 2
    );
    const removeScissor = this.scissor(os);
    gl.clearColor(...Color.zero.rgba1);
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
