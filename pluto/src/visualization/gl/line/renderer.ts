// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StaticCompiler } from "../compiler";
import { errorUnsupported } from "../errors";
import { GLContext, GLRenderer } from "../renderer";
import { ScissoredRenderRequest } from "../scissor";

// eslint-disable-next-line import/no-unresolved
import fragShader from "./frag.glsl?raw";
// eslint-disable-next-line import/no-unresolved
import vertShader from "./vert.glsl?raw";

import { RGBATuple } from "@/color";
import { Transform, XY } from "@/spatial";

const shaderVars = {
  scissor: {
    scale: "u_scale_scissor",
    offset: "u_offset_scissor",
  },
  scale: "u_scale",
  offset: "u_offset",
  color: "u_color",
  aspect: "u_aspect",
  mod: "a_mod",
  translate: "a_translate",
  x: "a_x",
  y: "a_y",
};

/** A line requested for rendering. */
export interface GLLine {
  /** The offset of the line in decimal. */
  offset: XY;
  /** The scale of the line. */
  scale: XY;
  /** The color for the line. */
  color: RGBATuple;
  /** The stroke width for the line */
  strokeWidth: number;
  /**
   * The number of points in the line. Generally, the lengths of the x and y buffers
   * should be equal to this number.
   */
  length: number;
  /**
   * The buffer containing the X coordinates for the line. This buffer should be managed
   * by the same WebGL canvas as provided in the rendering context.
   */
  x: WebGLBuffer;
  /**
   * The buffer containing the Y coordinates for the line. This buffer should be managed
   * by the same WebGL canvas as provided in the rendering context.
   */
  y: WebGLBuffer;
}

/** Just makes sure that the lines we draw to make stuff thick are really close together. */
const THICKNESS_DIVISOR = 12000;
const ANGLE_INSTANCED_ARRAYS_FEATURE = "ANGLE_instanced_arrays";

export interface LineRenderRequest extends ScissoredRenderRequest {
  lines: GLLine[];
}

export const LINE_RENDERER_TYPE = "line";

/* Draws lines with variable stroke width onto the canvas. */
export class GLLineRenderer
  extends StaticCompiler
  implements GLRenderer<LineRenderRequest>
{
  private translationBuffer: WebGLBuffer | null;
  private _extension: ANGLE_instanced_arrays | null;

  type: string = LINE_RENDERER_TYPE;

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

  render(ctx: GLContext, req: LineRenderRequest): void {
    ctx.refreshCanvas();
    const { gl } = ctx;
    this.use(gl);
    if (req.scissor != null) this.applyScissor(ctx, req.scissor);
    req.lines.forEach((l) => {
      this.bindAttrBuffer(gl, "x", l.x);
      this.bindAttrBuffer(gl, "y", l.y);
      this.uniformColor(gl, shaderVars.color, l.color);
      this.uniformXY(gl, shaderVars.scale, l.scale);
      this.uniformXY(gl, shaderVars.offset, l.offset);
      const numInstances = this.attrStrokeWidth(ctx, l.strokeWidth);
      this.extension.drawArraysInstancedANGLE(gl.LINE_STRIP, 0, l.length, numInstances);
    });
  }

  private bindAttrBuffer(
    gl: WebGLRenderingContext,
    dim: "x" | "y",
    buffer: WebGLBuffer
  ): void {
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const n = gl.getAttribLocation(this.program, shaderVars[dim]);
    gl.vertexAttribPointer(n, 1, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(n);
  }

  /**
   * We apply stroke width by drawing the line multiple times, each time with a slight
   * transformation. This is done as simply as possible. We draw the "centered" line
   * and then four more lines: one to the left, one to the right, one above, and one
   * below. This is done by using the `ANGLE_instanced_arrays` extension. We can repeat
   * this process to make the line thicker.
   */
  private attrStrokeWidth(ctx: GLContext, strokeWidth: number): number {
    const { gl, aspect } = ctx;

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

  private applyScissor(ctx: GLContext, scissor: Transform): void {
    this.uniformXY(ctx.gl, shaderVars.scissor.scale, scissor.scale);
    this.uniformXY(ctx.gl, shaderVars.scissor.offset, scissor.offset);
  }
}

const newTranslationBuffer = (aspect: number, strokeWidth: number): Float32Array => {
  if (strokeWidth <= 1) return new Float32Array([0, 0]);
  return copyBuffer(newDirectionBuffer(aspect), Math.ceil(strokeWidth) - 1).map(
    (v, i) => Math.floor(i / DIRECTION_COUNT) * (1 / (THICKNESS_DIVISOR * aspect)) * v
  );
};

const DIRECTION_COUNT = 5;

const newDirectionBuffer = (aspect: number): Float32Array =>
  // prettier-ignore
  new Float32Array([
    0, 0, // center
    0, aspect,  // top
    0, -aspect,  // bottom
    1, 0, // right
    -1, 0, // left
  ]);

const copyBuffer = (buf: Float32Array, times: number): Float32Array => {
  const newBuf = new Float32Array(buf.length * times);
  for (let i = 0; i < times; i++) newBuf.set(buf, i * buf.length);
  return newBuf;
};
