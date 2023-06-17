// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Bounds, xyScaleToTransform, Series, CrudeDirection } from "@synnaxlabs/x";

import { AetherContext, AetherLeaf, Update } from "@/core/aether/worker";
import {
  LineComponent,
  LineProps,
  ParsedLineState,
  lineState,
} from "@/core/vis/Line/core";
import FRAG_SHADER from "@/core/vis/Line/frag.glsl?raw";
import VERT_SHADER from "@/core/vis/Line/vert.glsl?raw";
import {
  RenderController,
  RequestRender,
  GLProgram,
  RenderContext,
} from "@/core/vis/render";
import { XYTelemSource } from "@/core/vis/telem";
import { TelemContext } from "@/core/vis/telem/TelemService";

export class LineGLProgramContext extends GLProgram {
  translationBuffer: WebGLBuffer;

  private static readonly CONTEXT_KEY = "pluto-line-gl-program";

  private constructor(ctx: RenderContext) {
    super(ctx, VERT_SHADER, FRAG_SHADER);
    this.translationBuffer = ctx.gl.createBuffer() as WebGLBuffer;
  }

  bindPropsAndContext(ctx: LineProps, state: ParsedLineState): number {
    const scaleTransform = xyScaleToTransform(ctx.scale);
    const transform = xyScaleToTransform(this.ctx.scaleRegion(ctx.region));
    this.uniformXY("u_region_scale", transform.scale);
    this.uniformXY("u_region_offset", transform.offset);
    this.uniformColor("u_color", state.color);
    this.uniformXY("u_scale", scaleTransform.scale);
    this.uniformXY("u_offset", scaleTransform.offset);
    return this.attrStrokeWidth(state.strokeWidth);
  }

  draw(x: Series, y: Series, count: number): void {
    this.bindAttrBuffer("x", x.glBuffer);
    this.bindAttrBuffer("y", y.glBuffer);
    this.ctx.gl.drawArraysInstanced(this.ctx.gl.LINE_STRIP, 0, x.length, count);
  }

  static create(ctx: AetherContext): LineGLProgramContext {
    const render = RenderContext.use(ctx);
    const line = new LineGLProgramContext(render);
    ctx.set(LineGLProgramContext.CONTEXT_KEY, line);
    return line;
  }

  static use(ctx: AetherContext): LineGLProgramContext {
    return ctx.get<LineGLProgramContext>(LineGLProgramContext.CONTEXT_KEY);
  }

  private bindAttrBuffer(dir: CrudeDirection, buffer: WebGLBuffer): void {
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

export class LineGL extends AetherLeaf<typeof lineState> implements LineComponent {
  prog: LineGLProgramContext;
  telem: XYTelemSource;
  requestRender: RequestRender;

  static readonly TYPE = "line";

  constructor(change: Update) {
    super(change, lineState);
    this.prog = LineGLProgramContext.use(change.ctx);
    this.telem = TelemContext.use<XYTelemSource>(change.ctx, this.state.telem.key);
    this.requestRender = RenderController.useRequest(change.ctx);
    this.handleUpdate(change.ctx);
    this.onUpdate((ctx) => this.handleUpdate(ctx));
    this.onDelete(() => this.cleanup());
  }

  private handleUpdate(ctx: AetherContext): void {
    this.prog = LineGLProgramContext.use(ctx);
    this.telem = TelemContext.use<XYTelemSource>(ctx, this.state.telem.key);
    this.requestRender = RenderController.useRequest(ctx);
    this.telem.onChange(() => this.requestRender());
    this.requestRender();
  }

  async xBounds(): Promise<Bounds> {
    return await this.telem.xBounds();
  }

  async yBounds(): Promise<Bounds> {
    return await this.telem.yBounds();
  }

  async render(props: LineProps): Promise<void> {
    this.prog.setAsActive();
    const xData = await this.telem.x(this.prog.ctx.gl);
    const yData = await this.telem.y(this.prog.ctx.gl);
    xData.forEach((x, i) => {
      const y = yData[i];
      const count = this.prog.bindPropsAndContext(
        {
          ...props,
          scale: {
            x: props.scale.x.translate(props.scale.x.dim(Number(x.sampleOffset))),
            y: props.scale.y.translate(props.scale.x.dim(Number(y.sampleOffset))),
          },
        },
        this.state
      );
      this.prog.draw(x, y, count);
    });
  }

  cleanup(): void {
    this.telem.release(this.prog.ctx.gl);
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
