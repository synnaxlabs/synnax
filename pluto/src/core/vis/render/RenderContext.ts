// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ZERO_COLOR } from "@/core/color";
import { Box, Scale, XY, XYScale, ZERO_XY } from "@synnaxlabs/x";

export class RenderContext {
  /* The canvas element */
  readonly canvas: OffscreenCanvas;
  /** The webgl rendering context extracted from the canvas */
  readonly gl: WebGL2RenderingContext;
  /** The region the canvas occupies in pixel space */
  region: Box;
  /** The device pixel ratio of the canvas */
  dpr: number;

  constructor(canvas: OffscreenCanvas, region: Box, dpr: number) {
    this.canvas = canvas;
    this.region = region;
    const gl = canvas.getContext("webgl2", { preserveDrawingBuffer: true });
    if (gl == null) throw new Error("Could not get WebGL context");
    this.gl = gl;
    this.dpr = dpr;
  }

  updateCanvasRegion(region: Box, dpr: number): void {
    this.region = region;
    this.dpr = dpr;
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

  erase(box: Box, overscan: XY = ZERO_XY): void {
    const { gl } = this;
    const os = new Box(
      box.left - overscan.x,
      box.top - overscan.y,
      box.width + overscan.x * 2,
      box.height + overscan.y * 2
    );
    const scale = this.scaleRegion(os);
    gl.enable(gl.SCISSOR_TEST);
    gl.scissor(scale.x.pos(0), scale.y.pos(0), scale.x.dim(1), scale.y.dim(1));
    gl.clearColor(...ZERO_COLOR);
    gl.clear(gl.COLOR_BUFFER_BIT);
    gl.disable(gl.SCISSOR_TEST);
  }
}
