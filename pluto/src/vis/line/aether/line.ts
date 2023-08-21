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
  Box,
  XYScale,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { color } from "@/color/core";
import FRAG_SHADER from "@/vis/line/aether/frag.glsl?raw";
import VERT_SHADER from "@/vis/line/aether/vert.glsl?raw";
import { render } from "@/vis/render";
import { XYTelemSource, xyTelemSourceSpec } from "@/vis/telem";
import { TelemContext } from "@/vis/telem/TelemContext";

const FLOAT_32_DENSITY = DataType.FLOAT32.density.valueOf();

export const stateZ = z.object({
  telem: xyTelemSourceSpec,
  label: z.string().optional(),
  color: color.Color.z,
  strokeWidth: z.number().default(1),
  downsample: z.number().min(1).max(50).optional().default(1),
});

export type State = z.input<typeof stateZ>;
export type ParsedState = z.output<typeof stateZ>;

export interface FindResult {
  // The line key that the point belongs to.
  key: string;
  // The decimal position of the point in the region.
  position: XY;
  // The data value of the point.
  value: XY;
  // The color of the line.
  color: color.Color;
  // The label of the line.
  label?: string;
  // The units of the line.
  units?: string;
}

export const ZERO_FIND_RESULT: FindResult = {
  key: "",
  position: XY.NAN,
  value: XY.NAN,
  color: color.ZERO,
};

export interface LineProps {
  /**
   * A box in pixel space representing the region of the display that the line
   * should be rendered in. The root of the pixel coordinate system is the top
   * left of the canvas.
   */
  region: Box;
  /** An XY scale that maps from the data space to decimal space. */
  dataToDecimalScale: XYScale;
}

export class Context extends render.GLProgram {
  translationBuffer: WebGLBuffer;

  private static readonly CONTEXT_KEY = "pluto-line-gl-program";

  private constructor(ctx: render.Context) {
    super(ctx, VERT_SHADER, FRAG_SHADER);
    this.translationBuffer = ctx.gl.createBuffer() as WebGLBuffer;
  }

  bindPropsAndState(
    { dataToDecimalScale: scale, region }: LineProps,
    { strokeWidth, color }: ParsedState
  ): number {
    const scaleTransform = xyScaleToTransform(scale);
    const transform = xyScaleToTransform(this.ctx.scaleRegion(region));
    this.uniformXY("u_region_scale", transform.scale);
    this.uniformXY("u_region_offset", transform.offset);
    this.uniformColor("u_color", color);
    this.uniformXY("u_scale", scaleTransform.scale);
    this.uniformXY("u_offset", scaleTransform.offset);
    return this.attrStrokeWidth(strokeWidth);
  }

  draw(x: Series, y: Series, count: number, downsample: number): void {
    const { gl } = this.ctx;
    this.bindAttrBuffer("x", x.glBuffer, downsample);
    this.bindAttrBuffer("y", y.glBuffer, downsample);
    gl.drawArraysInstanced(gl.LINE_STRIP, 0, x.length / downsample, count);
  }

  static create(ctx: aether.Context): Context {
    const renderCtx = render.Context.use(ctx);
    const line = new Context(renderCtx);
    ctx.set(Context.CONTEXT_KEY, line);
    return line;
  }

  static use(ctx: aether.Context): Context {
    return ctx.get<Context>(Context.CONTEXT_KEY);
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

interface InternalState {
  prog: Context;
  telem: XYTelemSource;
  cleanupTelem: Destructor;
  requestRender: () => void;
}

export class Line extends aether.Leaf<typeof stateZ, InternalState> {
  static readonly TYPE = "line";
  schema: typeof stateZ = stateZ;

  afterUpdate(): void {
    const [telem, cleanupTelem] = TelemContext.use<XYTelemSource>(
      this.ctx,
      this.key,
      this.state.telem
    );
    this.internal.telem = telem;
    this.internal.cleanupTelem = cleanupTelem;
    this.internal.prog = Context.use(this.ctx);
    this.internal.requestRender = render.Controller.useRequest(this.ctx);
    this.internal.telem.onChange(() => this.internal.requestRender());
    this.internal.requestRender();
  }

  afterDelete(): void {
    this.internal.cleanupTelem();
    this.internal.requestRender();
  }

  async xBounds(): Promise<Bounds> {
    return await this.internal.telem.xBounds();
  }

  async yBounds(): Promise<Bounds> {
    return await this.internal.telem.yBounds();
  }

  async findByXValue(props: LineProps, target: number): Promise<FindResult> {
    const { telem, prog } = this.internal;
    const data = await telem.x(prog.ctx.gl);
    let [index, series] = [-1, -1];
    data.find((x, i) => {
      const v = x.binarySearch(target);
      const valid = v !== -1 || v !== x.length;
      if (valid) [index, series] = [v, i];
      return valid;
    });
    const value = await this.xyValue(series, index);
    const { key } = this;
    const { color, label } = this.state;
    const position = new XY(
      props.dataToDecimalScale.x.pos(value.x),
      props.dataToDecimalScale.y.pos(value.y)
    );
    return { key, color, label, value, position };
  }

  async render(props: LineProps): Promise<void> {
    const { downsample } = this.state;
    const { telem, prog } = this.internal;
    const { dataToDecimalScale: scale } = props;
    prog.setAsActive();
    const xData = await telem.x(prog.ctx.gl);
    const yData = await telem.y(prog.ctx.gl);
    xData.forEach((x, i) => {
      const y = yData[i];
      if (y == null || x.length === 0 || y.length === 0) {
        return;
      }
      const p = { ...props, dataToDecimalScale: offsetScale(scale, x, y) };
      const count = prog.bindPropsAndState(p, this.state);
      prog.draw(x, y, count, downsample);
    });
  }

  private async xyValue(series: number, index: number): Promise<XY> {
    const { telem, prog } = this.internal;
    const x = await telem.x(prog.ctx.gl);
    const y = await telem.y(prog.ctx.gl);
    return new XY(this.getValue(series, index, x), this.getValue(series, index, y));
  }

  private getValue(series: number, index: number, data: Series[]): number {
    if (series === -1 || index === -1 || series >= data.length) return NaN;
    return Number(data[series].at(index));
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

const offsetScale = (scale: XYScale, x: Series, y: Series): XYScale =>
  scale.translate(
    scale.x.dim(Number(x.sampleOffset)),
    scale.y.dim(Number(y.sampleOffset))
  );
