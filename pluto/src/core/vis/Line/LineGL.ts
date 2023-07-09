// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Bounds,
  xyScaleToTransform,
  Series,
  CrudeDirection,
  Destructor,
  XY,
  DataType,
} from "@synnaxlabs/x";

import { AetherContext, AetherLeaf } from "@/core/aether/worker";
import {
  LineComponent,
  LineProps,
  LookupResult,
  ParsedLineState,
  lineState,
} from "@/core/vis/Line/core";
import FRAG_SHADER from "@/core/vis/Line/frag.glsl?raw";
import VERT_SHADER from "@/core/vis/Line/vert.glsl?raw";
import { RenderController, GLProgram, RenderContext } from "@/core/vis/render";
import { XYTelemSource } from "@/core/vis/telem";
import { TelemContext } from "@/core/vis/telem/TelemContext";

const FLOAT_32_DENSITY = DataType.FLOAT32.density.valueOf();

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

  draw(x: Series, y: Series, count: number, downsample: number): void {
    this.bindAttrBuffer("x", x.glBuffer, downsample);
    this.bindAttrBuffer("y", y.glBuffer, downsample);
    this.ctx.gl.drawArraysInstanced(
      this.ctx.gl.LINE_STRIP,
      0,
      x.length / downsample,
      count
    );
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

  private bindAttrBuffer(
    dir: CrudeDirection,
    buffer: WebGLBuffer,
    downsample: number
  ): void {
    const { gl } = this.ctx;
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const n = gl.getAttribLocation(this.prog, `a_${dir}`);
    gl.vertexAttribPointer(n, 1, gl.FLOAT, false, FLOAT_32_DENSITY * downsample, 0);
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

interface Derived {
  prog: LineGLProgramContext;
  telem: XYTelemSource;
  cleanupTelem: Destructor;
  requestRender: () => void;
}

export class LineGL
  extends AetherLeaf<typeof lineState, Derived>
  implements LineComponent
{
  static readonly TYPE = "line";
  schema: typeof lineState = lineState;

  derive(): Derived {
    return {
      prog: LineGLProgramContext.use(this.ctx),
      requestRender: RenderController.useRequest(this.ctx),
      ...TelemContext.use<XYTelemSource>(this.ctx, this.key, this.state.telem),
    };
  }

  afterUpdate(): void {
    this.derived.telem.onChange(() => this.derived.requestRender());
    this.derived.requestRender();
  }

  async xBounds(): Promise<Bounds> {
    const { telem } = this.derived;
    return await telem.xBounds();
  }

  async yBounds(): Promise<Bounds> {
    const { telem } = this.derived;
    return await telem.yBounds();
  }

  async searchX(props: LineProps, value: number): Promise<LookupResult> {
    const xData = await this.derived.telem.x(this.derived.prog.ctx.gl);
    let index: number = -1;
    let arr: number = -1;
    xData.forEach((x, i) => {
      const v = x.binarySearch(value);
      if (v !== -1 || v !== x.length) {
        index = v;
        arr = i;
      }
    });
    const xValue = await this.xValue(arr, index);
    const yValue = await this.yValue(arr, index);
    return {
      value: Number(yValue),
      position: new XY(props.scale.x.pos(xValue), props.scale.y.pos(yValue)),
    };
  }

  private async xValue(arr: number, index: number): Promise<number> {
    const { telem, prog } = this.derived;
    const xData = await telem.x(prog.ctx.gl);
    return Number(xData[arr].data[index]);
  }

  private async yValue(arr: number, index: number): Promise<number> {
    const { telem, prog } = this.derived;
    const yData = await telem.y(prog.ctx.gl);
    return Number(yData[arr].data[index]);
  }

  async render(props: LineProps): Promise<void> {
    const { telem, prog } = this.derived;
    prog.setAsActive();
    const xData = await telem.x(prog.ctx.gl);
    const yData = await telem.y(prog.ctx.gl);
    xData.forEach((x, i) => {
      const y = yData[i];
      if (y === undefined) return;
      if (x.length === 0 || y.length === 0) return;
      const count = prog.bindPropsAndContext(
        {
          ...props,
          scale: {
            x: props.scale.x.translate(props.scale.x.dim(Number(x.sampleOffset))),
            y: props.scale.y.translate(props.scale.x.dim(Number(y.sampleOffset))),
          },
        },
        this.state
      );
      prog.draw(x, y, count, this.state.downsample);
    });
  }

  handleDelete(): void {
    const { cleanupTelem } = this.derived;
    cleanupTelem();
    this.derived.requestRender();
  }
}

/** Just makes sure that the lines we draw to make stuff thick are really close together. */
const THICKNESS_DIVISOR = 5000;

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
