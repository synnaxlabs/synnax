/* eslint-disable import/no-unresolved */
import { mat4 } from "gl-matrix";

import fragShader from "./frag.glsl?raw";
import vertShader from "./vert.glsl?raw";

const shaderVars = {
  rootScale: "u_scale_root",
  scale: "u_scale",
  offsetRoot: "u_offset_root",
  offset: "u_offset",
  color: "u_color",
};

export interface XY {
  x: number;
  y: number;
}

export type RGBATuple = [number, number, number, number];

export interface Line {
  rootOffset: XY;
  offset: XY;
  rootScale: XY;
  scale: XY;
  color: RGBATuple;
  x: Float32Array;
  y: Float32Array;
}

export class LineRenderer {
  private readonly gl: WebGLRenderingContext;
  private readonly prog: WebGLProgram;
  private readonly buffers: {
    x: WebGLBuffer;
    y: WebGLBuffer;
  };

  private readonly ext: ANGLE_instanced_arrays;

  constructor(gl: WebGLRenderingContext) {
    this.gl = gl;
    this.ext = gl.getExtension("ANGLE_instanced_arrays") as ANGLE_instanced_arrays;
    if (this.ext == null) {
      throw new Error("ANGLE_instanced_arrays not supported");
    }
    this.prog = gl.createProgram() as WebGLProgram;
    this.compileAndLink();
    this.buffers = {
      x: gl.createBuffer() as WebGLBuffer,
      y: gl.createBuffer() as WebGLBuffer,
    };

    this.gl = gl;
  }

  render(line: Line): void {
    const { gl, prog } = this;
    gl.useProgram(prog);
    this.applyScale(line);
    this.applyOffset(line);
    this.applyColor(line);
    this.bindDim("x", line);
    this.bindDim("y", line);

    const aspect = gl.canvas.width / gl.canvas.height;
    const aspectLoc = gl.getUniformLocation(prog, "aspect");
    gl.uniform1f(aspectLoc, aspect);

    const instances = 5;

    const transGlBuffer = gl.createBuffer() as WebGLBuffer;
    const transBuffer = Float32Array.from({ length: instances }, (_, i) => {
      return (Math.floor(i / 5) + 1) / (3000 * aspect);
    });
    gl.bindBuffer(gl.ARRAY_BUFFER, transGlBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, transBuffer, gl.STATIC_DRAW);
    const transLoc = gl.getAttribLocation(prog, "translate");
    gl.vertexAttribPointer(transLoc, 1, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(transLoc);
    this.ext.vertexAttribDivisorANGLE(transLoc, 1);

    const modGlBuffer = gl.createBuffer() as WebGLBuffer;
    const modBuffer = Float32Array.from({ length: instances }, (_, i) => i % 5);
    gl.bindBuffer(gl.ARRAY_BUFFER, modGlBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, modBuffer, gl.STATIC_DRAW);
    const modLoc = gl.getAttribLocation(prog, "a_mod");
    gl.vertexAttribPointer(modLoc, 1, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(modLoc);
    this.ext.vertexAttribDivisorANGLE(modLoc, 1);

    this.ext.drawArraysInstancedANGLE(gl.LINE_STRIP, 0, line.x.length - 1, instances);
  }

  private compileAndLink(): void {
    const { gl } = this;
    const vs = gl.createShader(gl.VERTEX_SHADER) as WebGLShader;
    gl.shaderSource(vs, vertShader);
    gl.compileShader(vs);
    gl.attachShader(this.prog, vs);
    const fs = gl.createShader(gl.FRAGMENT_SHADER) as WebGLShader;
    gl.shaderSource(fs, fragShader);
    gl.compileShader(fs);
    gl.attachShader(this.prog, fs);
    gl.linkProgram(this.prog);
  }

  private bindDim(dim: "x" | "y", req: Line): void {
    const { gl } = this;
    const n = gl.getAttribLocation(this.prog, dim);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers[dim]);
    gl.bufferData(gl.ARRAY_BUFFER, req[dim], gl.STATIC_DRAW);
    gl.vertexAttribPointer(n, 1, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(n);
  }

  private applyScale(req: Line): void {
    const { gl, prog } = this;
    const rootScale = gl.getUniformLocation(prog, shaderVars.rootScale);
    gl.uniform2fv(rootScale, [req.rootScale.x, req.rootScale.y]);
    const s2 = gl.getUniformLocation(prog, shaderVars.scale);
    this.gl.uniform2fv(s2, [req.scale.x, req.scale.y]);
  }

  private applyOffset(req: Line): void {
    const { gl, prog } = this;
    const o1 = gl.getUniformLocation(prog, shaderVars.offsetRoot);
    gl.uniform2fv(o1, [req.rootOffset.x, req.rootOffset.y]);
    const o2 = gl.getUniformLocation(prog, shaderVars.offset);
    gl.uniform2fv(o2, [req.offset.x, req.offset.y]);
  }

  private applyColor(req: Line): void {
    const { gl, prog } = this;
    const color = gl.getUniformLocation(prog, shaderVars.color);
    gl.uniform4fv(color, req.color);
  }

  destroy(): void {
    const { gl, prog, buffers } = this;
    gl.deleteProgram(prog);
    gl.deleteBuffer(buffers.x);
    gl.deleteBuffer(buffers.y);
  }
}
