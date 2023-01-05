// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errorUnsupported } from "../../errors";
import { RGBATuple, XY } from "../../types/spatial";
import { StaticCompiler } from "../compiler";
import { RenderingContext } from "../renderer";

import fragShader from "./frag.glsl?raw";
import vertShader from "./vert.glsl?raw";

const shaderVars = {
  rootScale: "u_scale_root",
  scale: "u_scale",
  offsetRoot: "u_offset_root",
  offset: "u_offset",
  color: "u_color",
  aspect: "u_aspect",
  mod: "a_mod",
  translate: "a_translate",
};

export interface Line {
  offset: XY;
  scale: XY;
  color: RGBATuple;
  x: Float32Array;
  y: Float32Array;
  strokeWidth: number;
}

const THICKNESS_DIVISOR = 3000;
const ANGLE_INSTANCED_ARRAYS_FEATURE = "ANGLE_instanced_arrays";

export class LineRenderer extends StaticCompiler {
  private translationBuffer: WebGLBuffer | null;
  private _extension: ANGLE_instanced_arrays | null;

  constructor() {
    super(vertShader, fragShader);
    this.translationBuffer = null;
    this._extension = null;
  }

  get extension(): ANGLE_instanced_arrays {
    this.validateCompiled();
    return this._extension as ANGLE_instanced_arrays;
  }

  compile(gl: WebGLRenderingContext): void {
    super.compile(gl);
    this._extension = gl.getExtension(ANGLE_INSTANCED_ARRAYS_FEATURE);
    if (this._extension == null) throw errorUnsupported(ANGLE_INSTANCED_ARRAYS_FEATURE);
    this.translationBuffer = gl.createBuffer() as WebGLBuffer;
  }

  render(ctx: RenderingContext): void {
    this.use(ctx.gl);

    this.applyScale(line);
    this.applyOffset(line);

    this.applyColor(line);

    const instances = this.applyThickness(ctx, line);
    this._extension.drawArraysInstancedANGLE(
      gl.LINE_STRIP,
      0,
      line.x.length - 1,
      instances
    );
  }

  private applyThickness(ctx: RenderingContext, req: Line): number {
    const { gl, aspect } = ctx;
    const { strokeWidth } = req;

    gl.bindBuffer(gl.ARRAY_BUFFER, this.translationBuffer);
    const translationBuffer = newTranslationBuffer(aspect, strokeWidth);
    gl.bufferData(gl.ARRAY_BUFFER, translationBuffer, gl.STATIC_DRAW);

    const loc = gl.getAttribLocation(this.program, shaderVars.translate);
    gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(loc);
    this.extension.vertexAttribDivisorANGLE(loc, 1);

    // divide by 2 because each instance has 2 floats.
    const numInstances = translationBuffer.length / 2;
    return numInstances;
  }

  private bindDim(ctx: RenderingContext, dim: "x" | "y", req: Line): void {
    const { gl } = ctx;
    const n = gl.getAttribLocation(this.program, dim);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.buffers[dim]);
    gl.bufferData(gl.ARRAY_BUFFER, req[dim], gl.STATIC_DRAW);
    gl.vertexAttribPointer(n, 1, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(n);
  }

  private applyScale(req: Line): void {
    const { gl, program: prog } = this;
    const rootScale = gl.getUniformLocation(prog, shaderVars.rootScale);
    gl.uniform2fv(rootScale, [req.rootScale.x, req.rootScale.y]);
    const s2 = gl.getUniformLocation(prog, shaderVars.scale);
    this.gl.uniform2fv(s2, [req.scale.x, req.scale.y]);
  }

  private applyOffset(req: Line): void {
    const { gl, program: prog } = this;
    const o1 = gl.getUniformLocation(prog, shaderVars.offsetRoot);
    gl.uniform2fv(o1, [req.rootOffset.x, req.rootOffset.y]);
    const o2 = gl.getUniformLocation(prog, shaderVars.offset);
    gl.uniform2fv(o2, [req.offset.x, req.offset.y]);
  }

  private applyColor(req: Line): void {
    const { gl, program: prog } = this;
    const color = gl.getUniformLocation(prog, shaderVars.color);
    gl.uniform4fv(color, req.color);
  }

  destroy(): void {
    const { gl, program: prog, buffers } = this;
    gl.deleteProgram(prog);
    gl.deleteBuffer(buffers.x);
    gl.deleteBuffer(buffers.y);
  }
}

const newTranslationBuffer = (aspect: number, strokeWidth: number): Float32Array =>
  copyBuffer(newDirectionBuffer(aspect), Math.ceil(strokeWidth)).map(
    (v, i) => Math.floor(i / DIRECTION_COUNT) + (1 / (THICKNESS_DIVISOR * aspect)) * v
  );

const DIRECTION_COUNT = 5;

const newDirectionBuffer = (aspect: number): Float32Array =>
  // prettier-ignore
  new Float32Array([
    0, 0, // center
    0, 1 * aspect,  // top
    0, -1 * aspect,  // bottom
    1, 0, // right
    -1, 0, // left
  ]);

const copyBuffer = (buf: Float32Array, times: number): Float32Array => {
  const newBuf = new Float32Array(buf.length * times);
  for (let i = 0; i < times; i++) newBuf.set(buf, i * buf.length);
  return newBuf;
};

export class BufferControl {
  private readonly gl: WebGLRenderingContext;
  private readonly entries: Record<string, BufferControlEntry> = {};

  constructor(gl: WebGLRenderingContext) {
    this.gl = gl;
  }

  set(key: string, data: Float32Array): void {
    this.delete(key);
    this.entries[key] = new BufferControlEntry(data);
  }

  delete(key: string): void {
    const existing = this.entries[key];
    if (existing == null) return;
    existing.delete(this.gl);
    // eslint-disable-next-line @typescript-eslint/no-dynamic-delete
    delete this.entries[key];
  }

  use(key: string): void {
    this.entries[key].use(this.gl);
  }

  createBuffer(data: Float32Array): BufferControlEntry {
    return new BufferControlEntry(data);
  }
}

export class BufferControlEntry {
  private readonly data: Float32Array;
  private gl: WebGLBuffer | null;
  private buffered: boolean = false;

  constructor(data: Float32Array) {
    this.gl = null;
    this.data = data;
  }

  use(gl: WebGLRenderingContext): void {
    this.maybeBuffer(gl);
    gl.bindBuffer(gl.ARRAY_BUFFER, this.gl);
  }

  delete(gl: WebGLRenderingContext): void {
    gl.deleteBuffer(this.gl);
  }

  private maybeBuffer(gl: WebGLRenderingContext): void {
    if (this.buffered) return;
    this.gl = gl.createBuffer() as WebGLBuffer;
    gl.bufferData(gl.ARRAY_BUFFER, this.data, gl.STATIC_DRAW);
    this.buffered = true;
  }
}
