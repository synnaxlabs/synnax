// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY } from "@synnaxlabs/x";

import { Compiler } from "@/vis/gl/compiler";
import { GLRendererRegistry } from "@/vis/gl/registry";
import { ScissoredGLRenderer, ScissoredRenderRequest } from "@/vis/gl/scissor";
import { RenderingUnits } from "@/vis/gl/types";

/**
 * A renderer for a specific type of entity. A renderer should not maintain any internal

 * state relating to specific entities, but should instead rely on the request properties
 * to determine how to render it.
 */
export interface GLRenderer<R> extends Compiler {
  /** Type is a unique type for the renderer. */
  type: string;
  /** Renders the given entity under the RenderingContext.  */
  render: (ctx: GLRenderContext, req: R) => void;
}

export class GLRenderContext {
  readonly gl: WebGL2RenderingContext;
  readonly registry: GLRendererRegistry;
  readonly canvas: OffscreenCanvas;
  canvasBox: Box;
  dpr: number;

  constructor(
    canvas: OffscreenCanvas,
    registry: GLRendererRegistry,
    canvasBox: Box,
    dpr: number
  ) {
    this.canvas = canvas;
    const gl = canvas.getContext("webgl2", { preserveDrawingBuffer: true });
    if (gl == null) throw new Error("Could not get WebGL context");
    this.gl = gl;
    this.registry = registry;
    this.registry.compile(this.gl);
    this.canvasBox = canvasBox;
    this.dpr = dpr;
  }

  updateCanvasDims(box: Box, dpr: number): void {
    this.canvasBox = box;
    this.dpr = dpr;
  }

  get aspect(): number {
    const b = this.canvasBox;
    return b.width / b.height;
  }

  scale(box: Box): XY {
    const c = this.canvasBox;
    return { x: box.width / c.width, y: box.height / c.height };
  }

  private offsetPx(box: Box): XY {
    const c = this.canvasBox;
    const relLeft = box.left - c.left;
    const bottom = c.bottom - box.bottom;
    return { y: bottom, x: relLeft };
  }

  offset(box: Box, units: RenderingUnits = "decimal"): XY {
    const dims = this.offsetPx(box);
    if (units === "decimal") return this.normalizeDims(dims);
    return dims;
  }

  private normalizeDims(dims: XY): XY {
    return {
      x: (dims.x * this.dpr) / this.canvas.width,
      y: (dims.y * this.dpr) / this.canvas.height,
    };
  }

  scissor<R extends ScissoredRenderRequest>(
    wrap: GLRenderer<R>,
    overscan?: XY
  ): ScissoredGLRenderer<R> {
    return new ScissoredGLRenderer<R>(wrap, overscan);
  }
}
