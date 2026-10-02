// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError } from "@synnaxlabs/client";

import FRAG_SHADER from "@/vis/render/clear/frag.glsl?raw";
import VERT_SHADER from "@/vis/render/clear/vert.glsl?raw";
import { type Context } from "@/vis/render/context";
import { GLProgram } from "@/vis/render/GLProgram";

const POSITIONS = new Float32Array([0, 0, 0, 0, 0, 0]);

/**
 * You may be wondering, why does this program that draws a colorless, zero-footprint
 * triangle exist? It turns out that the WeBGL implementation on windows doesn't
 * actually clear a scissored region of the screen when you call `gl.clear`, you
 * actually need to make a draw call to replace what's currently in the framebuffer.
 * This is a workaround that does exactly that.
 */
export class Program extends GLProgram {
  private readonly vao: WebGLVertexArrayObject;

  constructor(ctx: Context) {
    super(ctx, VERT_SHADER, FRAG_SHADER);
    const { gl } = ctx;
    const buffer = gl.createBuffer();
    if (buffer == null) throw new UnexpectedError(`webgl: failed to create buffer`);
    const vao = gl.createVertexArray();
    if (vao == null)
      throw new UnexpectedError(`webgl: failed to create vertex array object`);
    this.vao = vao;
    gl.bindVertexArray(this.vao);
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.bufferData(gl.ARRAY_BUFFER, POSITIONS, gl.STATIC_DRAW);
    const positionAttr = gl.getAttribLocation(this.prog, "a_position");
    gl.enableVertexAttribArray(positionAttr);
    gl.vertexAttribPointer(positionAttr, 2, gl.FLOAT, false, 0, 0);
    gl.bindVertexArray(null);
  }

  exec(): void {
    const { gl } = this.renderCtx;
    gl.bindVertexArray(this.vao);
    this.setAsActive();
    gl.drawArrays(gl.TRIANGLES, 0, 3);
    // Line programs set attributes on the default vertex array.
    gl.bindVertexArray(null);
  }
}
