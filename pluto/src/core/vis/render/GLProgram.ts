// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY } from "@synnaxlabs/x";

import { RenderContext } from "./RenderContext";

import { RGBATuple } from "@/core/color";
import { errorCompile, ERROR_BAD_SHADER } from "@/core/vis/render/errors";

export class GLProgram {
  readonly ctx: RenderContext;
  readonly prog: WebGLProgram;
  private readonly vertShader: string;
  private readonly fragShader: string;

  constructor(ctx: RenderContext, vertShader: string, fragShader: string) {
    this.ctx = ctx;
    const prog = ctx.gl.createProgram();
    if (prog == null) throw errorCompile("failed to create program");
    this.prog = prog;
    this.vertShader = vertShader;
    this.fragShader = fragShader;
    this.compile();
  }

  private compile(): void {
    const gl = this.ctx.gl;
    this.compileShader(this.vertShader, gl.VERTEX_SHADER);
    this.compileShader(this.fragShader, gl.FRAGMENT_SHADER);
    gl.linkProgram(this.prog);
  }

  private compileShader(shader: string, type: number): void {
    const gl = this.ctx.gl;
    const vs = gl.createShader(type);
    if (vs == null) throw ERROR_BAD_SHADER;
    gl.shaderSource(vs, shader);
    gl.compileShader(vs);
    const compiled = gl.getShaderParameter(vs, gl.COMPILE_STATUS) as GLboolean;
    if (!compiled) {
      const error = gl.getShaderInfoLog(vs);
      gl.deleteShader(vs);
      throw errorCompile(error ?? "unknown");
    }
    gl.attachShader(this.prog, vs);
  }

  setAsActive(): void {
    this.ctx.gl.useProgram(this.prog);
  }

  uniformXY(name: string, value: XY): void {
    this.ctx.gl.uniform2fv(this.getUniformLoc(name), [value.x, value.y]);
  }

  uniformColor(name: string, value: RGBATuple): void {
    this.ctx.gl.uniform4fv(this.getUniformLoc(name), value);
  }

  private getUniformLoc(name: string): WebGLUniformLocation {
    const loc = this.ctx.gl.getUniformLocation(this.prog, name);
    if (loc == null) throw new Error(`unexpected missing uniform ${name}`);
    return loc;
  }
}
