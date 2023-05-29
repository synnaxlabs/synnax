// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Direction, Bound, xyScaleToTransform, LazyArray } from "@synnaxlabs/x";

import { hexToRGBA } from "@/core/color";
import { LineRenderer, LineProps, LineContext } from "@/core/vis/Line/core";
import { GLProgram, errorUnsupported, RenderContext } from "@/core/vis/render";
import { DynamicXYTelemSource, XYTelemSource } from "@/core/vis/telem";
import { TelemProvider } from "@/core/vis/telem/TelemService";
// eslint-disable-next-line import/no-unresolved
import FRAG_SHADER from "@/vis/core/Line/frag.glsl?raw";
// eslint-disable-next-line import/no-unresolved
import VERT_SHADER from "@/vis/core/Line/vert.glsl?raw";

const ANGLE_INSTANCED_ARRAYS_FEATURE = "ANGLE_instanced_arrays";

/**
 * A factory for creating webgl rendered lines.
 */
export class LineFactory {
  private readonly program: LineGLProgram;
  private readonly telem: TelemProvider;

  /**
   * @param ctx - The webgl rendering context to use.
   * @param telem - A function that returns the telemetry provider.
   */
  constructor(ctx: RenderContext, telem: TelemProvider) {
    this.program = new LineGLProgram(ctx);
    this.telem = telem;
  }

  /**
   * Creates a new line.
   * @param props - The properties of the line.
   * @param requestRender - A function that allows the line to request that its parent re-render it.
   */
  new(props: LineProps, requestRender: () => void): LineRenderer {
    return new LineGL(props, this.program, requestRender, this.telem);
  }
}

class LineGLProgram extends GLProgram {
  extension: ANGLE_instanced_arrays;
  translationBuffer: WebGLBuffer;

  constructor(ctx: RenderContext) {
    super(ctx, VERT_SHADER, FRAG_SHADER);
    const ext = ctx.gl.getExtension(ANGLE_INSTANCED_ARRAYS_FEATURE);
    if (ext == null) throw errorUnsupported(ANGLE_INSTANCED_ARRAYS_FEATURE);
    this.extension = ext;
    this.translationBuffer = ctx.gl.createBuffer() as WebGLBuffer;
  }

  bindPropsAndContext(ctx: LineContext, props: LineProps): number {
    const regionTransform = xyScaleToTransform(this.ctx.scaleRegion(ctx.region));
    const scaleTransform = xyScaleToTransform(ctx.scale);
    this.uniformXY("u_region_scale", regionTransform.scale);
    this.uniformXY("u_region_offset", regionTransform.offset);
    this.uniformColor("u_color", hexToRGBA(props.color));
    this.uniformXY("u_scale", scaleTransform.scale);
    this.uniformXY("u_offset", scaleTransform.offset);
    return this.attrStrokeWidth(props.strokeWidth);
  }

  draw(x: LazyArray, y: LazyArray, count: number): void {
    this.bindAttrBuffer("x", x.buffer);
    this.bindAttrBuffer("y", y.buffer);
    this.extension.drawArraysInstancedANGLE(this.ctx.gl.LINE_STRIP, 0, x.length, count);
  }

  private bindAttrBuffer(dir: Direction, buffer: WebGLBuffer): void {
    const { gl } = this.ctx;
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const n = gl.getAttribLocation(gl, `a_${dir}`);
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
  private attrStrokeWidth(strokeWidth: number): number {
    const { gl } = this.ctx;
    gl.bindBuffer(gl.ARRAY_BUFFER, this.translationBuffer);
    const translationBuffer = newTranslationBuffer(this.ctx.aspect, strokeWidth);
    gl.bufferData(gl.ARRAY_BUFFER, translationBuffer, gl.DYNAMIC_DRAW);
    const loc = gl.getAttribLocation(this.prog, "a_translate");
    gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(loc);
    this.extension.vertexAttribDivisorANGLE(loc, 1);
    return translationBuffer.length / 2;
  }
}

export class LineGL implements LineRenderer {
  props: LineProps;
  prog: LineGLProgram;
  requestRender: () => void;
  telemProv: TelemProvider;
  telem: XYTelemSource | DynamicXYTelemSource;

  static readonly TYPE = "line";

  constructor(
    props: LineProps,
    program: LineGLProgram,
    requestRender: () => void,
    telemProv: TelemProvider
  ) {
    this.props = props;
    this.prog = program;
    this.requestRender = requestRender;
    this.telemProv = telemProv;
    this.telem = this.telemProv.get(props.telem.key);
  }

  get key(): string {
    return this.props.key;
  }

  setProps(props: LineProps): void {
    if (props.telem.key !== this.props.telem.key)
      this.telem = this.telemProv.get(props.telem.key);
    this.props = props;
    if ("onChange" in this.telem) this.telem.onChange(() => this.requestRender());
    this.requestRender();
  }

  async xBound(): Promise<Bound> {
    return await this.telem.xBound();
  }

  async yBound(): Promise<Bound> {
    return await this.telem.yBound();
  }

  async render(ctx: LineContext): Promise<void> {
    this.prog.setAsActive();
    const xData = await this.telem.x();
    const yData = await this.telem.y();
    const count = this.prog.bindPropsAndContext(ctx, this.props);
    xData.forEach((x, i) => this.prog.draw(x, yData[i], count));
  }
}

/** Just makes sure that the lines we draw to make stuff thick are really close together. */
const THICKNESS_DIVISOR = 12000;

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
