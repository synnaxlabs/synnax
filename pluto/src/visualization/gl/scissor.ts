// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { GLRenderer, GLContext } from "./renderer";

import { RGBATuple } from "@/color";
import { Box, XY, Transform, ZERO_XY } from "@/spatial";

const CLEAR_COLOR: RGBATuple = [0, 0, 0, 0];

export interface ScissoredRenderRequest {
  box: Box;
  scissor?: Transform;
}

export class ScissoredGLRenderer<R extends ScissoredRenderRequest>
  implements GLRenderer<R>
{
  readonly wrapped: GLRenderer<R>;
  private readonly overscan: XY;

  constructor(wrapped: GLRenderer<R>, overscan: XY = ZERO_XY) {
    this.overscan = overscan;
    this.wrapped = wrapped;
  }

  get type(): string {
    return this.wrapped.type;
  }

  compile(gl: WebGLRenderingContext): void {
    this.wrapped.compile(gl);
  }

  render(ctx: GLContext, req: R): void {
    ctx.refreshCanvas();
    ctx.gl.enable(ctx.gl.SCISSOR_TEST);
    this.scissor(ctx, req.box);
    req.scissor = this.calculateTransform(ctx, req.box);
    this.wrapped.render(ctx, req);
    ctx.gl.disable(ctx.gl.SCISSOR_TEST);
  }

  private calculateTransform(ctx: GLContext, box: Box): { offset: XY; scale: XY } {
    return { scale: ctx.scale(box), offset: ctx.offset(box, "decimal") };
  }

  clear(ctx: GLContext, box: Box): void {
    ctx.gl.enable(ctx.gl.SCISSOR_TEST);
    this.scissor(ctx, box);
    ctx.gl.clearColor(...CLEAR_COLOR);
    ctx.gl.clear(ctx.gl.COLOR_BUFFER_BIT);
    ctx.gl.disable(ctx.gl.SCISSOR_TEST);
  }

  private scissor(ctx: GLContext, box: Box): void {
    const { x, y } = ctx.offset(box, "px");
    const { width, height } = box;
    const { x: ox, y: oy } = this.overscan;
    ctx.gl.scissor(
      (x - ox / 2) * ctx.dpr,
      (y - ox / 2) * ctx.dpr,
      (width + ox) * ctx.dpr,
      (height + oy) * ctx.dpr
    );
  }
}
