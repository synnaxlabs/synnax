// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { XY } from "@synnaxlabs/x";

import { RGBATuple } from "@/color";
import { errorCompile, ERROR_BAD_SHADER, ERROR_NOT_COMPILED } from "@/vis/gl/errors";

export interface Compiler {
  /** Compiles and links the program to the given context.  */
  compile: (gl: WebGLRenderingContext) => void;
}

/** A compiler that compiles a static shader program. */
export class StaticCompiler implements Compiler {
  private readonly vertShader: string;
  private readonly fragShader: string;
  private _program: WebGLProgram | null;
  compiled: boolean;

  constructor(vertShader: string, fragShader: string) {
    this._program = null;
    this.vertShader = vertShader;
    this.fragShader = fragShader;
    this.compiled = false;
  }

  get program(): WebGLProgram {
    this.validateCompiled();
    return this._program as WebGLProgram;
  }

  compile(gl: WebGLRenderingContext): void {
    this._program = gl.createProgram();
    this.compiled = true;
    this.compileShader(gl, this.vertShader, gl.VERTEX_SHADER);
    this.compileShader(gl, this.fragShader, gl.FRAGMENT_SHADER);
    gl.linkProgram(this._program as WebGLProgram);
  }

  compileShader(gl: WebGLRenderingContext, shader: string, type: number): void {
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
    gl.attachShader(this.program, vs);
  }

  validateCompiled(): void {
    if (!this.compiled) throw ERROR_NOT_COMPILED;
  }

  use(gl: WebGLRenderingContext): void {
    gl.useProgram(this.program);
  }

  uniformXY(gl: WebGLRenderingContext, name: string, value: XY): void {
    gl.uniform2fv(this.getUniformLoc(gl, name), [value.x, value.y]);
  }

  uniformColor(gl: WebGLRenderingContext, name: string, value: RGBATuple): void {
    gl.uniform4fv(this.getUniformLoc(gl, name), value);
  }

  private getUniformLoc(gl: WebGLRenderingContext, name: string): WebGLUniformLocation {
    const loc = gl.getUniformLocation(this.program, name);
    if (loc == null) throw new Error(`unexpected missing uniform ${name}`);
    return loc;
  }
}
