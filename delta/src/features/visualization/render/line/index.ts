// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ComponentSizeTypographyLevels } from "@synnaxlabs/pluto/dist/core/Typography/types";

import { errorUnsupported } from "../../errors";
import { RGBATuple, XY } from "../../types/spatial";
import { StaticCompiler } from "../compiler";
import { RenderingContext } from "../renderer";

import fragShader from "./frag.glsl?raw";
import vertShader from "./vert.glsl?raw";

import { Range } from "@/features/workspace";
import { unique } from "@/util/unique";

const shaderVars = {
  x: "a_x",
  y: "a_y",
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
  strokeWidth: number;
  xKey: string;
  yKey: string;
}

const THICKNESS_DIVISOR = 12000;
const ANGLE_INSTANCED_ARRAYS_FEATURE = "ANGLE_instanced_arrays";

export interface LineRenderRequest {
  range: Range;
  lines: Line[];
}

export class LineRenderer extends StaticCompiler {
  private translationBuffer: WebGLBuffer | null;
  private _extension: ANGLE_instanced_arrays | null;

  readonly type: string = "line";

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

  async render(ctx: RenderingContext, req: LineRenderRequest): Promise<void> {
    this.use(ctx.gl);
    const { gl } = ctx;

    const { range, lines } = req;

    const keys = unique(lines.flatMap((line) => [line.xKey, line.yKey]));

    const f = await ctx.client.getFrame({ range, keys });

    req.lines.forEach((line) => {
      this.applyOffset(ctx, line);
      this.applyColor(ctx, line);
      const instances = this.applyThickness(ctx, line);
      const xEntry = f.find((e) => e.key === line.xKey);
      const yEntry = f.find((e) => e.key === line.yKey);
      if (xEntry == null || yEntry == null) {
        console.warn(`missing x or y array for line ${line.xKey} ${line.yKey}`);
        return;
      }
      if (xEntry.arrays.length !== yEntry.arrays.length) {
        console.warn(
          `x and y arrays are not the same length for line ${line.xKey} ${line.yKey}`
        );
        return;
      }

      yEntry.arrays.forEach((y, i) => {
        const xArray = xEntry.arrays[i];
        const [xMax, xMin] = [Math.max(...xArray.data), Math.min(...xArray.data)] as [
          number,
          number
        ];
        const [yMax, yMin] = [Math.max(...y.data), Math.min(...y.data)] as [
          number,
          number
        ];
        line.scale.x = 1 / (xMax - xMin);
        line.scale.y = 1 / (yMax - yMin);
        line.offset.y = -yMin * line.scale.y;
        line.offset.x = -xMin * line.scale.x;
        const xBuffer = xEntry.glBuffers[i];
        this.bindBuffer(gl, "x", xBuffer);
        const yBuffer = yEntry.glBuffers[i];
        this.bindBuffer(gl, "y", yBuffer);

        this.applyScale(ctx, line);

        console.log(xArray.data);

        this.extension.drawArraysInstancedANGLE(gl.LINE_STRIP, 0, y.length, instances);
      });
    });
  }

  private bindBuffer(
    gl: WebGLRenderingContext,
    dim: "x" | "y",
    buffer: WebGLBuffer
  ): void {
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const n = gl.getAttribLocation(this.program, shaderVars[dim]);
    gl.vertexAttribPointer(n, 1, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(n);
  }

  private applyThickness(ctx: RenderingContext, req: Line): number {
    const { gl, aspect } = ctx;
    const { strokeWidth } = req;

    gl.bindBuffer(gl.ARRAY_BUFFER, this.translationBuffer);
    console.log(strokeWidth);
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

  private applyScale(ctx: RenderingContext, line: Line): void {
    const { program: prog } = this;
    const { gl, rootScaleClip } = ctx;
    const { scale } = line;
    const rootScale = gl.getUniformLocation(prog, shaderVars.rootScale);
    gl.uniform2fv(rootScale, [rootScaleClip.x, rootScaleClip.y]);
    const s2 = gl.getUniformLocation(prog, shaderVars.scale);
    gl.uniform2fv(s2, [scale.x, scale.y]);
  }

  private applyOffset(ctx: RenderingContext, line: Line): void {
    const { program: prog } = this;
    const { gl, rootOffsetClip } = ctx;
    const { offset } = line;
    const o1 = gl.getUniformLocation(prog, shaderVars.offsetRoot);
    gl.uniform2fv(o1, [rootOffsetClip.x, rootOffsetClip.y]);
    const o2 = gl.getUniformLocation(prog, shaderVars.offset);
    gl.uniform2fv(o2, [offset.x, offset.y]);
  }

  private applyColor(ctx: RenderingContext, line: Line): void {
    const { program: prog } = this;
    const { gl } = ctx;
    const { color } = line;
    const colorLoc = gl.getUniformLocation(prog, shaderVars.color);
    gl.uniform4fv(colorLoc, color);
  }
}

const newTranslationBuffer = (aspect: number, strokeWidth: number): Float32Array =>
  copyBuffer(newDirectionBuffer(aspect), Math.ceil(strokeWidth) - 1).map(
    (v, i) => Math.floor(i / DIRECTION_COUNT) * (1 / (THICKNESS_DIVISOR * aspect)) * v
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
