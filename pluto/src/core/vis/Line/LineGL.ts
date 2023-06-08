// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, xyScaleToTransform, LazyArray, DirectionT } from "@synnaxlabs/x";

import { BobComponentFactory, AetherLeaf } from "@/core/aether/worker";
import {
  LineComponent,
  LineState,
  LineContext,
  lineState,
  ParsedLineState,
} from "@/core/vis/Line/core";
import FRAG_SHADER from "@/core/vis/Line/frag.glsl?raw";
import VERT_SHADER from "@/core/vis/Line/vert.glsl?raw";
import { GLProgram, RenderContext } from "@/core/vis/render";
import { DynamicXYTelemSource, XYTelemSource } from "@/core/vis/telem";
import { TelemProvider } from "@/core/vis/telem/TelemService";

/**
 * A factory for creating webgl rendered lines.
 */
export class LineFactory implements BobComponentFactory<LineComponent> {
  private readonly program: LineGLProgram;
  private readonly telem: TelemProvider;
  requestRender: () => void;

  /**
   * @param prog - The webgl rendering context to use.
   * @param telem - A function that returns the telemetry provider.
   */
  constructor(prog: LineGLProgram, telem: TelemProvider, requestRender: () => void) {
    this.program = prog;
    this.telem = telem;
    this.requestRender = requestRender;
  }

  create(type: string, key: string, props: any): LineComponent {
    if (type !== LineGL.TYPE)
      throw new Error(
        `[LineFactory.create] - Expected type ${LineGL.TYPE} but got ${type}`
      );
    return new LineGL(key, props, this.program, this.requestRender, this.telem);
  }
}

export class LineGLProgram extends GLProgram {
  translationBuffer: WebGLBuffer;

  constructor(ctx: RenderContext) {
    super(ctx, VERT_SHADER, FRAG_SHADER);
    this.translationBuffer = ctx.gl.createBuffer() as WebGLBuffer;
  }

  bindPropsAndContext(ctx: LineContext, props: ParsedLineState): number {
    const scaleTransform = xyScaleToTransform(ctx.scale);
    const transform = xyScaleToTransform(this.ctx.scaleRegion(ctx.region));
    this.uniformXY("u_region_scale", transform.scale);
    this.uniformXY("u_region_offset", transform.offset);
    this.uniformColor("u_color", props.color);
    this.uniformXY("u_scale", scaleTransform.scale);
    this.uniformXY("u_offset", scaleTransform.offset);
    return this.attrStrokeWidth(props.strokeWidth);
  }

  draw(x: LazyArray, y: LazyArray, count: number): void {
    this.bindAttrBuffer("x", x.glBuffer);
    this.bindAttrBuffer("y", y.glBuffer);
    this.ctx.gl.drawArraysInstanced(this.ctx.gl.LINE_STRIP, 0, x.length, count);
  }

  private bindAttrBuffer(dir: DirectionT, buffer: WebGLBuffer): void {
    const { gl } = this.ctx;
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const n = gl.getAttribLocation(this.prog, `a_${dir}`);
    gl.vertexAttribPointer(n, 1, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(n);
  }

  /**
   * We apply stroke width by drawing the line multiple times, each time with a slight
   * transformation. This is done as simply as possible. We draw the "centered" line
   * and then four more lines: one to the left, one to the right, one above, and one
   * below. We can repeat this process an arbitrary number of times to make the line
   * thicker. As we increase the stroke width, we also increase the cost of drawing the
   * line.
   */
  private attrStrokeWidth(strokeWidth: number): number {
    const { gl } = this.ctx;
    gl.bindBuffer(gl.ARRAY_BUFFER, this.translationBuffer);
    const translationBuffer = newTranslationBuffer(this.ctx.aspect, strokeWidth);
    gl.bufferData(gl.ARRAY_BUFFER, translationBuffer, gl.DYNAMIC_DRAW);
    const loc = gl.getAttribLocation(this.prog, "a_translate");
    gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(loc);
    this.ctx.gl.vertexAttribDivisor(loc, 1);
    return translationBuffer.length / 2;
  }
}

export class LineGL
  extends AetherLeaf<LineState, ParsedLineState>
  implements LineComponent
{
  prog: LineGLProgram;
  requestRender: () => void;
  telemProv: TelemProvider;
  telem: XYTelemSource | DynamicXYTelemSource;

  static readonly TYPE = "line";

  constructor(
    key: string,
    props: LineState,
    program: LineGLProgram,
    requestRender: () => void,
    telemProv: TelemProvider
  ) {
    super(key, LineGL.TYPE, props, lineState);
    this.prog = program;
    this.requestRender = requestRender;
    this.telemProv = telemProv;
    this.telem = this.telemProv.get(props.telem.key);
    this.setStateHook(() => this.requestRender());
    if ("onChange" in this.telem) this.telem.onChange(() => this.requestRender());
  }

  async xBound(): Promise<Bounds> {
    return await this.telem.xBound();
  }

  async yBound(): Promise<Bounds> {
    return await this.telem.yBound();
  }

  async render(ctx: LineContext): Promise<void> {
    this.prog.setAsActive();
    const xData = await this.telem.x(this.prog.ctx.gl);
    const yData = await this.telem.y(this.prog.ctx.gl);
    const count = this.prog.bindPropsAndContext(ctx, this.state);
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
