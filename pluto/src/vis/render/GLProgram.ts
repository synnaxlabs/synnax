// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { color, type destructor, xy } from "@synnaxlabs/x";

import { type Context } from "@/vis/render/context";

const errorCompile = (msg: string): Error =>
  new Error(`failed to compile webgl program: ${msg}`);

const ERROR_BAD_SHADER = new Error("null shader encountered");

/**
 * A general purpose compiler and utility container for workign with WebGL programs.
 */
export class GLProgram {
  /** The render context used by this program. */
  readonly renderCtx: Context;
  /** The underlying webgl program. */
  readonly prog: WebGLProgram;
  /** The code for the vertex shader. */
  private readonly vertShader: string;
  /** The code for the fragment shader. */
  private readonly fragShader: string;
  uniformLocCache = new Map<string, WebGLUniformLocation>();

  /**
   * @constructor compiles the given vertex and fragment shaders under the given
   * render context into a program.
   *
   * @param ctx - The render context to use.
   * @param vertShader - The vertex shader code.
   * @param fragShader - The fragment shader code.
   */
  constructor(ctx: Context, vertShader: string, fragShader: string) {
    this.renderCtx = ctx;
    const prog = ctx.gl.createProgram();
    if (prog == null) throw errorCompile("failed to create program");
    this.prog = prog;
    this.vertShader = vertShader;
    this.fragShader = fragShader;
    this.compile();
  }

  /** Sets the current program as the active program used by the context. */
  setAsActive(): destructor.Destructor {
    this.renderCtx.gl.useProgram(this.prog);
    return (): void => this.renderCtx.gl.useProgram(null);
  }

  /**
   * Sets a uniform XY coordinate value.
   *
   * @param name - The name of the uniform.
   * @param value - The value to set.
   */
  uniformXY(name: string, value: xy.Crude): void {
    this.renderCtx.gl.uniform2fv(this.getUniformLoc(name), xy.couple(value));
  }

  /**
   * Sets a uniform color value.
   *
   * @param name - The name of the uniform.
   * @param value - The value to set.
   */
  uniformColor(name: string, value: color.Color): void {
    this.renderCtx.gl.uniform4fv(this.getUniformLoc(name), color.rgba1(value));
  }

  private getUniformLoc(name: string): WebGLUniformLocation {
    const v = this.uniformLocCache.get(name);
    if (v != null) return v;
    const loc = this.renderCtx.gl.getUniformLocation(this.prog, name);
    if (loc == null) throw new Error(`unexpected missing uniform ${name}`);
    this.uniformLocCache.set(name, loc);
    return loc;
  }

  private compile(): void {
    const gl = this.renderCtx.gl;
    this.compileShader(this.vertShader, gl.VERTEX_SHADER);
    this.compileShader(this.fragShader, gl.FRAGMENT_SHADER);
    gl.linkProgram(this.prog);
  }

  private compileShader(shader: string, type: number): void {
    const gl = this.renderCtx.gl;
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
}
