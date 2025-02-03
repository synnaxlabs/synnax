// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { UnexpectedError } from "@synnaxlabs/client";
import {
  bounds,
  type box,
  clamp,
  DataType,
  type Destructor,
  type direction,
  type scale,
  type Series,
  type SeriesDigest,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { color } from "@/color/core";
import { telem } from "@/telem/aether";
import FRAG_SHADER from "@/vis/line/aether/frag.glsl?raw";
import F32_VERT_SHADER from "@/vis/line/aether/vert_f32.glsl?raw";
import HYBRID_VERT_SHADER from "@/vis/line/aether/vert_hybrid.glsl?raw";
import { render } from "@/vis/render";

export const stateZ = z.object({
  x: telem.seriesSourceSpecZ,
  y: telem.seriesSourceSpecZ,
  label: z.string().optional(),
  color: color.Color.z,
  strokeWidth: z.number().default(1),
  downsample: z.number().min(1).max(50).optional().default(1),
});

const safelyGetDataValue = (series: number, index: number, data: Series[]): number => {
  if (series === -1 || index === -1 || series >= data.length) return NaN;
  return Number(data[series].at(index));
};

export type State = z.input<typeof stateZ>;
export type ParsedState = z.output<typeof stateZ>;

const DEFAULT_OVERLAP_THRESHOLD = TimeSpan.milliseconds(2);

export interface FindResult {
  // The line key that the point belongs to.
  key: string;
  // The decimal position of the point in the region.
  position: xy.XY;
  // The data value of the point.
  value: xy.XY;
  // The color of the line.
  color: color.Color;
  // The label of the line.
  label?: string;
  // The units of the line.
  units?: string;
}

export const ZERO_FIND_RESULT: FindResult = {
  key: "",
  position: xy.NAN,
  value: xy.NAN,
  color: color.ZERO,
};

export interface LineProps {
  /**
   * A box in pixel space representing the region of the display that the line
   * should be rendered in. The root of the pixel coordinate system is the top
   * left of the canvas.
   */
  region: box.Box;
  /** An XY scale that maps from the data space to decimal space. */
  dataToDecimalScale: scale.XY;
  exposure: number;
}

interface TranslationBufferCacheEntry {
  glBuffer: WebGLBuffer;
  jsBuffer: Float32Array;
}

const dataTypeToGLProgram = (
  gl: WebGL2RenderingContext,
  dataType: DataType,
): number => {
  if (dataType.equals(DataType.UINT8)) return gl.UNSIGNED_BYTE;
  return gl.FLOAT;
};

export class GLProgram extends render.GLProgram {
  private readonly translationBufferCache = new Map<
    string,
    TranslationBufferCacheEntry
  >();

  // Add cached attribute locations
  private readonly attrLocations: Record<string, number> = {};

  constructor(ctx: render.Context, vertShader: string, fragShader: string) {
    super(ctx, vertShader, fragShader);
    this.translationBufferCache = new Map();
    // Cache commonly used attribute locations
    this.attrLocations = {
      x: this.ctx.gl.getAttribLocation(this.prog, "a_x"),
      y: this.ctx.gl.getAttribLocation(this.prog, "a_y"),
      translate: this.ctx.gl.getAttribLocation(this.prog, "a_translate"),
    };
  }

  bindState({ strokeWidth, color }: ParsedState): number {
    this.uniformColor("u_color", color);
    return this.attrStrokeWidth(strokeWidth);
  }

  bindScale(
    dataScaleTransform: scale.XYTransformT,
    regionTransform: scale.XYTransformT,
  ): void {
    const aggregateScale = xy.scale(dataScaleTransform.scale, regionTransform.scale);
    const aggregateOffset = xy.translate(
      xy.scale(regionTransform.scale, dataScaleTransform.offset),
      regionTransform.offset,
    );
    this.uniformXY("u_scale_aggregate", aggregateScale);
    this.uniformXY("u_offset_aggregate", aggregateOffset);
  }

  draw(
    { x, y, count, downsample, xOffset, yOffset }: DrawOperation,
    instances: number,
    xDataType: DataType,
    yDataType: DataType,
  ): void {
    const { gl } = this.ctx;
    this.bindAttrBuffer("x", x.glBuffer, downsample, xOffset, xDataType);
    this.bindAttrBuffer("y", y.glBuffer, downsample, yOffset, yDataType);
    gl.drawArraysInstanced(gl.LINE_STRIP, 0, count / downsample, instances);
  }

  private bindAttrBuffer(
    dir: direction.Crude,
    buffer: WebGLBuffer,
    downsample: number,
    alignment: number = 0,
    dataType: DataType,
  ): void {
    const { gl } = this.ctx;
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const aLoc = gl.getAttribLocation(this.prog, `a_${dir}`);
    const glDataType = dataTypeToGLProgram(gl, dataType);
    const density = dataType.density.valueOf();

    if (dataType.equals(DataType.UINT8))
      // Use gl.vertexAttribIPointer for integer attributes
      gl.vertexAttribIPointer(
        aLoc,
        1,
        glDataType, // e.g., gl.UNSIGNED_BYTE
        density * downsample,
        density * alignment,
      );
    else
      // Use gl.vertexAttribPointer for float attributes
      gl.vertexAttribPointer(
        aLoc,
        1,
        glDataType,
        false,
        density * downsample,
        density * alignment,
      );

    gl.enableVertexAttribArray(aLoc);
  }
  private getAndBindTranslationBuffer(
    strokeWidth: number,
  ): TranslationBufferCacheEntry {
    const { gl } = this.ctx;
    const key = `${this.ctx.aspect}:${strokeWidth}`;
    const existing = this.translationBufferCache.get(key);
    if (existing != null) {
      gl.bindBuffer(gl.ARRAY_BUFFER, existing.glBuffer);
      return existing;
    }
    const buf = gl.createBuffer();
    if (buf == null)
      throw new UnexpectedError("Failed to create buffer from WebGL context");
    const translationBuffer = newTranslationBuffer(this.ctx.aspect, strokeWidth);
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, translationBuffer, gl.DYNAMIC_DRAW);
    const entry = { glBuffer: buf, jsBuffer: translationBuffer };
    this.translationBufferCache.set(key, entry);
    return entry;
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
    const { jsBuffer } = this.getAndBindTranslationBuffer(strokeWidth);
    const loc = gl.getAttribLocation(this.prog, "a_translate");
    gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(loc);
    gl.vertexAttribDivisor(loc, 1);
    return jsBuffer.length / 2;
  }
}

export class Context {
  private static readonly CONTEXT_KEY = "pluto-line-gl-program";
  // Uint8 hybrid program is used for high performance rendering of uint8 data along
  // with float32 timestamp data. It's used as a hot path optimization for common
  // channel such as actuator states.
  private readonly uint8HybridProgram: GLProgram;
  // Float32 program is used for rendering float32 data. It's used for all other
  // channel types.
  private readonly float32Program: GLProgram;

  private constructor(ctx: render.Context) {
    this.uint8HybridProgram = new GLProgram(ctx, HYBRID_VERT_SHADER, FRAG_SHADER);
    this.float32Program = new GLProgram(ctx, F32_VERT_SHADER, FRAG_SHADER);
  }

  get gl(): WebGL2RenderingContext {
    return this.uint8HybridProgram.ctx.gl;
  }

  getProgram(dataType: DataType): GLProgram {
    if (dataType.equals(DataType.UINT8)) return this.uint8HybridProgram;
    return this.float32Program;
  }

  static create(ctx: aether.Context): Context {
    const renderCtx = render.Context.use(ctx);
    const line = new Context(renderCtx);
    ctx.set(Context.CONTEXT_KEY, line);
    return line;
  }

  static use(ctx: aether.Context): Context {
    const glProgram = ctx.get<Context>(Context.CONTEXT_KEY);
    if (glProgram == null) throw new UnexpectedError("GLProgram not found");
    return glProgram;
  }
}

interface InternalState {
  instrumentation: Instrumentation;
  ctx: Context;
  xTelem: telem.SeriesSource;
  stopListeningXTelem?: Destructor;
  yTelem: telem.SeriesSource;
  stopListeningYTelem?: Destructor;
  requestRender: render.RequestF;
}

export class Line extends aether.Leaf<typeof stateZ, InternalState> {
  static readonly TYPE = "line";
  schema: typeof stateZ = stateZ;

  async afterUpdate(): Promise<void> {
    if (this.deleted) return;
    const { internal: i } = this;
    i.xTelem = await telem.useSource(this.ctx, this.state.x, i.xTelem);
    i.yTelem = await telem.useSource(this.ctx, this.state.y, i.yTelem);
    i.instrumentation = alamos.useInstrumentation(this.ctx, "line");
    i.ctx = Context.use(this.ctx);
    i.requestRender = render.Controller.useRequest(this.ctx);
    i.stopListeningXTelem?.();
    i.stopListeningYTelem?.();
    i.stopListeningXTelem = i.xTelem.onChange(() =>
      i.requestRender(render.REASON_DATA),
    );
    i.stopListeningYTelem = i.yTelem.onChange(() =>
      i.requestRender(render.REASON_DATA),
    );
    i.requestRender(render.REASON_LAYOUT);
  }

  async afterDelete(): Promise<void> {
    const { internal: i } = this;
    await i.xTelem.cleanup?.();
    await i.yTelem.cleanup?.();
    i.requestRender(render.REASON_LAYOUT);
  }

  async xBounds(): Promise<bounds.Bounds> {
    return (await this.internal.xTelem.value())[0];
  }

  async yBounds(): Promise<bounds.Bounds> {
    return (await this.internal.yTelem.value())[0];
  }

  async findByXValue(props: LineProps, target: number): Promise<FindResult> {
    const { xTelem, yTelem } = this.internal;
    const [, xData] = await xTelem.value();
    let [index, series] = [-1, -1];
    xData.find((x, i) => {
      const v = x.binarySearch(target);
      // The returned value gives us the insert position, so anything that is not
      // a valid index is not a valid value.
      const valid = v >= 0 && v < x.length;
      if (valid) [index, series] = [v, i];
      // We can stop the search if we have found a valid value.
      return valid;
    });
    const { key } = this;
    const { color, label } = this.state;
    const result = {
      key,
      color,
      label,
      position: { x: 0, y: 0 },
      value: { x: NaN, y: NaN },
    };

    if (index === -1 || series === -1) return result;

    const xSeries = xData[series];
    result.value.x = safelyGetDataValue(series, index, xData);
    const [, yData] = await yTelem.value();
    const ySeries = yData.find((ys) =>
      bounds.contains(ys.alignmentBounds, xSeries.alignment + BigInt(index)),
    );
    if (ySeries == null) return result;

    const alignmentDiff = Number(ySeries.alignment - xSeries.alignment);
    result.value.y = Number(ySeries.at(index - alignmentDiff));

    result.position = {
      x: props.dataToDecimalScale.x.pos(result.value.x),
      y: props.dataToDecimalScale.y.pos(result.value.y),
    };
    return result;
  }

  async render(props: LineProps): Promise<void> {
    if (this.deleted) return;
    const { downsample } = this.state;
    const { xTelem, yTelem, ctx } = this.internal;

    const { dataToDecimalScale, exposure } = props;
    const [[, xData], [, yData]] = await Promise.all([xTelem.value(), yTelem.value()]);
    xData.forEach((x) => x.updateGLBuffer(ctx.gl));
    yData.forEach((y) => y.updateGLBuffer(ctx.gl));
    if (xData.length === 0 || yData.length === 0) return;
    const prog = ctx.getProgram(yData[0].dataType);
    const ops = buildDrawOperations(
      xData,
      yData,
      exposure,
      downsample,
      DEFAULT_OVERLAP_THRESHOLD,
    );
    this.internal.instrumentation.L.debug("render", () => ({
      key: this.key,
      downsample,
      scale: dataToDecimalScale.transform,
      props: props.region,
      ops: digests(ops),
    }));
    const clearProg = prog.setAsActive();
    const instances = prog.bindState(this.state);
    const regionTransform = prog.ctx.scaleRegion(props.region).transform;
    ops.forEach((op) => {
      const scaleTransform = offsetScale(dataToDecimalScale, op).transform;
      prog.bindScale(scaleTransform, regionTransform);
      prog.draw(op, instances, xData[0].dataType, yData[0].dataType);
    });
    clearProg();
  }
}

/** Just makes sure that the lines we draw to make stuff thick are really close together. */
const THICKNESS_DIVISOR = 5000;

const newTranslationBuffer = (aspect: number, strokeWidth: number): Float32Array =>
  replicateBuffer(newDirectionBuffer(aspect), strokeWidth).map(
    (v, i) => Math.floor(i / DIRECTION_COUNT) * (1 / (THICKNESS_DIVISOR * aspect)) * v,
  );

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

const replicateBuffer = (buf: Float32Array, times: number): Float32Array => {
  const newBuf = new Float32Array(buf.length * times);
  for (let i = 0; i < times; i++) newBuf.set(buf, i * buf.length);
  return newBuf;
};

const offsetScale = (scale: scale.XY, op: DrawOperation): scale.XY =>
  scale.translate(
    scale.x.dim(Number(op.x.sampleOffset)),
    scale.y.dim(Number(op.y.sampleOffset)),
  );

export const REGISTRY: aether.ComponentRegistry = { [Line.TYPE]: Line };

export interface DrawOperation {
  x: Series;
  y: Series;
  xOffset: number;
  yOffset: number;
  count: number;
  downsample: number;
}

interface DrawOperationDigest extends Omit<DrawOperation, "x" | "y"> {
  x: SeriesDigest;
  y: SeriesDigest;
}

export const buildDrawOperations = (
  xSeries: Series[],
  ySeries: Series[],
  exposure: number,
  userSpecifiedDownSampling: number,
  overlapThreshold: TimeSpan,
): DrawOperation[] => {
  if (xSeries.length === 0 || ySeries.length === 0) return [];
  const ops: DrawOperation[] = [];
  xSeries.forEach((x) =>
    ySeries.forEach((y) => {
      if (!seriesOverlap(x, y, overlapThreshold)) return;
      let xOffset = 0;
      let yOffset = 0;
      // This means that the x series starts before the y series.
      if (x.alignment < y.alignment) xOffset = Number(y.alignment - x.alignment);
      // This means that the y series starts before the x series.
      else if (y.alignment < x.alignment) yOffset = Number(x.alignment - y.alignment);
      const count = Math.min(x.length - xOffset, y.length - yOffset);
      if (count === 0) return;
      const downsample = clamp(
        Math.round(exposure * 4 * count),
        userSpecifiedDownSampling,
        51,
      );
      ops.push({ x, y, xOffset, yOffset, count, downsample });
    }),
  );
  return ops;
};

const digests = (ops: DrawOperation[]): DrawOperationDigest[] =>
  ops.map((op) => ({ ...op, x: op.x.digest, y: op.y.digest }));

const seriesOverlap = (x: Series, ys: Series, overlapThreshold: TimeSpan): boolean => {
  // This is just a runtime check that both series' have time ranges defined.
  const haveTimeRanges = x._timeRange != null && ys._timeRange != null;
  if (!haveTimeRanges)
    throw new UnexpectedError(
      `Encountered series without time range in buildDrawOperations. X series present: ${x._timeRange != null}, Y series present: ${ys._timeRange != null}`,
    );
  // If the time ranges of the x and y series overlap, we meet the first condition
  // for drawing them together. Dynamic buffering can sometimes lead to very slight,
  // unintended overlaps, so we only consider them overlapping if they overlap by a
  // certain threshold.
  const timeRangesOverlap = x.timeRange.overlapsWith(ys.timeRange, overlapThreshold);
  // If the 'indexes' of the x and y series overlap, we meet the second condition
  // for drawing them together.
  const alignmentsOverlap = bounds.overlapsWith(x.alignmentBounds, ys.alignmentBounds);
  return timeRangesOverlap && alignmentsOverlap;
};
