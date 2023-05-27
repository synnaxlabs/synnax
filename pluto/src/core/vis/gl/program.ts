// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY } from "@synnaxlabs/x";

import { RGBATuple } from "@/core/color";
import {
  errorCompile,
  ERROR_BAD_SHADER,
  ERROR_NOT_COMPILED,
} from "@/core/vis/gl/errors";

/** A compiler that compiles a static shader program. */
export class Program {
  private readonly vertShader: string;
  private readonly fragShader: string;
  private readonly _program: WebGLProgram | null;
  readonly gl: WebGL2RenderingContext;
  compiled: boolean;

  constructor(gl: WebGL2RenderingContext, vertShader: string, fragShader: string) {
    this.gl = gl;
    this._program = null;
    this.vertShader = vertShader;
    this.fragShader = fragShader;
    this.compiled = false;
  }

  get program(): WebGLProgram {
    this.validateCompiled();
    return this._program as WebGLProgram;
  }

  private compile(): void {
    this.compiled = true;
    this.compileShader(this.vertShader, this.gl.VERTEX_SHADER);
    this.compileShader(this.fragShader, this.gl.FRAGMENT_SHADER);
    this.gl.linkProgram(this._program as WebGLProgram);
  }

  private compileShader(shader: string, type: number): void {
    const vs = this.gl.createShader(type);
    if (vs == null) throw ERROR_BAD_SHADER;
    this.gl.shaderSource(vs, shader);
    this.gl.compileShader(vs);
    const compiled = this.gl.getShaderParameter(vs, gl.COMPILE_STATUS) as GLboolean;
    if (!compiled) {
      const error = this.gl.getShaderInfoLog(vs);
      this.gl.deleteShader(vs);
      throw errorCompile(error ?? "unknown");
    }
    this.gl.attachShader(this.program, vs);
  }

  private validateCompiled(): void {
    if (!this.compiled) throw ERROR_NOT_COMPILED;
  }

  use(gl: WebGLRenderingContext): void {
    gl.useProgram(this.program);
  }

  uniformXY(name: string, value: XY): void {
    this.gl.uniform2fv(this.getUniformLoc(name), [value.x, value.y]);
  }

  uniformColor(name: string, value: RGBATuple): void {
    this.gl.uniform4fv(this.getUniformLoc(name), value);
  }

  private getUniformLoc(name: string): WebGLUniformLocation {
    const loc = this.gl.getUniformLocation(this.program, name);
    if (loc == null) throw new Error(`unexpected missing uniform ${name}`);
    return loc;
  }
}
