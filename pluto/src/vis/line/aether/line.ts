// Copyright 2026 Synnax Labs, Inc.
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
  color,
  DataType,
  type destructor,
  type direction,
  math,
  type MultiSeries,
  scale,
  type Series,
  type SeriesDigest,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { status } from "@/status/aether";
import { telem } from "@/telem/aether";
import { type TickType } from "@/vis/axis/ticks";
import { clip, seriesOverlap, windowBounds } from "@/vis/line/aether/bounds";
import FRAG_SHADER from "@/vis/line/aether/frag.glsl?raw";
import F32_VERT_SHADER from "@/vis/line/aether/vert_f32.glsl?raw";
import HYBRID_VERT_SHADER from "@/vis/line/aether/vert_hybrid.glsl?raw";
import { render } from "@/vis/render";

export const stateZ = z.object({
  x: telem.seriesSourceSpecZ,
  y: telem.seriesSourceSpecZ,
  label: z.string().optional(),
  color: color.colorZ,
  strokeWidth: z.number().default(1),
  downsample: z.number().min(1).max(50).default(1),
  downsampleMode: telem.downsampleModeZ.default("decimate"),
  visible: z.boolean().default(true),
});

const safelyGetDataValue = (
  series: number,
  index: number,
  data: MultiSeries,
): number => {
  if (series === -1 || index === -1 || series >= data.series.length) return NaN;
  return Number(data.series[series].at(index));
};

export type State = z.input<typeof stateZ>;
export type ParsedState = z.infer<typeof stateZ>;

export const DEFAULT_OVERLAP_THRESHOLD = TimeSpan.milliseconds(2);

export interface FindResult {
  key: string;
  position: xy.XY;
  value: xy.XY;
  color: color.Color;
  label?: string;
  units?: string;
  bounds: bounds.Bounds;
  /** Tick type of the x axis the line sits on. Time formats x as a timestamp. */
  xType?: TickType;
  /** Label of the x axis, used as the units of x on a linear axis. */
  xUnits?: string;
}

/** Paired samples of a line inside an x window. */
export interface Samples {
  x: Float64Array;
  y: Float64Array;
}

export interface LineProps {
  /**
   * A box in pixel space representing the region of the display that the line should be
   * rendered in. The root of the pixel coordinate system is the top left of the canvas.
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
  private segmentBuffer?: WebGLBuffer;

  constructor(ctx: render.Context, vertShader: string, fragShader: string) {
    super(ctx, vertShader, fragShader);
    this.translationBufferCache = new Map();
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
    op: DrawOperation,
    instances: number,
    xDataType: DataType,
    yDataType: DataType,
  ): void {
    const { gl } = this.renderCtx;
    const { x, y, downsample, xOffset, yOffset } = op;
    this.bindAttrBuffer("x", x.glBuffer, downsample, xOffset, xDataType);
    this.bindAttrBuffer("y", y.glBuffer, downsample, yOffset, yDataType);
    gl.drawArraysInstanced(gl.LINE_STRIP, 0, vertexCount(op), instances);
  }

  /** Draws one segment per pair of vertices, given as interleaved float32 x and y. */
  segments(vertices: Float32Array, instances: number): void {
    const { gl } = this.renderCtx;
    this.segmentBuffer ??= gl.createBuffer();
    gl.bindBuffer(gl.ARRAY_BUFFER, this.segmentBuffer);
    gl.bufferData(gl.ARRAY_BUFFER, vertices, gl.DYNAMIC_DRAW);
    this.bindAttrBuffer("x", this.segmentBuffer, 2, 0, DataType.FLOAT32);
    this.bindAttrBuffer("y", this.segmentBuffer, 2, 1, DataType.FLOAT32);
    gl.drawArraysInstanced(gl.LINES, 0, vertices.length / 2, instances);
  }

  private bindAttrBuffer(
    dir: direction.Crude,
    buffer: WebGLBuffer,
    downsample: number,
    alignment: number = 0,
    dataType: DataType,
  ): void {
    const { gl } = this.renderCtx;
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const aLoc = gl.getAttribLocation(this.prog, `a_${dir}`);
    const glDataType = dataTypeToGLProgram(gl, dataType);
    const density = dataType.density.valueOf();

    if (dataType.equals(DataType.UINT8))
      gl.vertexAttribIPointer(
        aLoc,
        1,
        glDataType, // e.g., gl.UNSIGNED_BYTE
        density * downsample,
        density * alignment,
      );
    else
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
    const { gl } = this.renderCtx;
    const key = `${this.renderCtx.aspect}:${strokeWidth}`;
    const existing = this.translationBufferCache.get(key);
    if (existing != null) {
      gl.bindBuffer(gl.ARRAY_BUFFER, existing.glBuffer);
      return existing;
    }
    const buf = gl.createBuffer();
    if (buf == null)
      throw new UnexpectedError("Failed to create buffer from WebGL context");
    const translationBuffer = newTranslationBuffer(this.renderCtx.aspect, strokeWidth);
    gl.bindBuffer(gl.ARRAY_BUFFER, buf);
    gl.bufferData(gl.ARRAY_BUFFER, translationBuffer, gl.DYNAMIC_DRAW);
    const entry = { glBuffer: buf, jsBuffer: translationBuffer };
    this.translationBufferCache.set(key, entry);
    return entry;
  }

  /**
   * We apply stroke width by drawing the line multiple times, each time with a slight
   * transformation. This is done as simply as possible. We draw the "centered" line and
   * then four more lines: one to the left, one to the right, one above, and one below.
   * We can repeat this process an arbitrary number of times to make the line thicker.
   * As we increase the stroke width, we also increase the cost of drawing the line.
   */
  private attrStrokeWidth(strokeWidth: number): number {
    const { gl } = this.renderCtx;
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
    return this.uint8HybridProgram.renderCtx.gl;
  }

  getProgram(dataType: DataType): GLProgram {
    if (dataType.equals(DataType.UINT8)) return this.uint8HybridProgram;
    return this.float32Program;
  }

  static create(ctx: aether.Context, renderCtx: render.Context): Context {
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
  lineCtx: Context;
  xTelem: telem.SeriesSource;
  stopListeningXTelem?: destructor.Destructor;
  yTelem: telem.SeriesSource;
  stopListeningYTelem?: destructor.Destructor;
  requestRender: render.Requestor;

  xDownsampler: telem.SeriesDownsampler;
  yDownsampler: telem.SeriesDownsampler;
}

export class Line extends aether.Leaf<typeof stateZ, InternalState> {
  static readonly TYPE = "line";
  schema: typeof stateZ = stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const createOptions: telem.CreateOptions = {
      onStatusChange: status.useAdder(ctx),
    };
    i.xTelem = telem.useSource(ctx, this.state.x, i.xTelem, createOptions);
    i.yTelem = telem.useSource(ctx, this.state.y, i.yTelem, createOptions);
    i.instrumentation = alamos.useInstrumentation(ctx, "line");
    i.lineCtx = Context.use(ctx);
    i.requestRender = render.useRequestor(ctx);
    i.stopListeningXTelem?.();
    i.stopListeningYTelem?.();
    i.stopListeningXTelem = i.xTelem.onChange(() => i.requestRender("data"));
    i.stopListeningYTelem = i.yTelem.onChange(() => i.requestRender("data"));
    const held = telem.useHold(ctx);
    i.xTelem.setHold?.(held);
    i.yTelem.setHold?.(held);
    i.requestRender("layout");
    if (
      i.xDownsampler?.props.mode !== this.state.downsampleMode ||
      i.xDownsampler?.props.windowSize !== this.state.downsample
    ) {
      i.xDownsampler = new telem.SeriesDownsampler({
        mode: this.state.downsampleMode,
        windowSize: this.state.downsample,
      });
      i.yDownsampler = new telem.SeriesDownsampler({
        mode: this.state.downsampleMode,
        windowSize: this.state.downsample,
      });
    }
  }

  afterDelete(): void {
    const { internal: i } = this;
    i.xTelem.cleanup?.();
    i.yTelem.cleanup?.();
    i.requestRender("layout");
  }

  get loading(): boolean {
    const { xTelem, yTelem } = this.internal;
    return (xTelem.loading?.() ?? false) || (yTelem.loading?.() ?? false);
  }

  xBounds(): bounds.Bounds {
    return this.internal.xTelem.value()[0];
  }

  /**
   * @param xWindow - the visible x range. Bounds cover only samples whose x value
   * falls inside it; when the window is non-finite or clips out every sample, the
   * source's full bounds are used instead.
   * @returns the y bounds of this line's samples inside the window.
   */
  yBounds(xWindow: bounds.Bounds): bounds.Bounds {
    const { xTelem, yTelem } = this.internal;
    const [b, yData] = yTelem.value();
    if (!bounds.isFinite(xWindow)) return b;
    const [, xData] = xTelem.value();
    return windowBounds(xData, yData, xWindow, DEFAULT_OVERLAP_THRESHOLD, b);
  }

  /** @returns the paired x and y samples whose x value lies inside the window. */
  samplesIn(xWindow: bounds.Bounds): Samples {
    const { xTelem, yTelem } = this.internal;
    const [, xData] = xTelem.value();
    const [, yData] = yTelem.value();
    const xs: number[] = [];
    const ys: number[] = [];
    for (const x of xData.series)
      for (const y of yData.series) {
        const range = clip(x, y, xWindow, DEFAULT_OVERLAP_THRESHOLD);
        if (range == null) continue;
        const offset = Number((y.alignment - x.alignment) / x.alignmentMultiple);
        for (let j = range[0]; j < range[1]; j++) {
          xs.push(Number(x.at(j + offset, true)));
          ys.push(Number(y.at(j, true)));
        }
      }
    return { x: Float64Array.from(xs), y: Float64Array.from(ys) };
  }

  findByXValue(props: LineProps, target: number): FindResult {
    const { xTelem, yTelem } = this.internal;
    let [, xData] = xTelem.value();
    xData = this.internal.xDownsampler.transform(xData);
    let [index, series] = [-1, -1];
    xData.series.find((x, i) => {
      const v = x.binarySearch(target);
      // The returned value gives us the insert position, so anything that is not
      // a valid index is not a valid value.
      const valid = v >= 0 && v < x.length;
      if (valid) [index, series] = [v, i];
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
      bounds: { lower: 0, upper: 0 },
    };

    if (index === -1 || series === -1 || !this.state.visible) return result;

    const xSeries = xData.series[series];
    result.value.x = safelyGetDataValue(series, index, xData);
    let [, yData] = yTelem.value();
    yData = this.internal.yDownsampler.transform(yData);
    const ySeries = yData.series.find((ys) =>
      bounds.contains(ys.alignmentBounds, xSeries.alignment + BigInt(index)),
    );
    if (ySeries == null) return result;

    const op = buildDrawOperation(
      xSeries,
      ySeries,
      props.exposure,
      this.state.downsample,
      this.state.downsampleMode,
      DEFAULT_OVERLAP_THRESHOLD,
    );
    if (op != null) {
      index = nearestVertex(op, index);
      result.value.x = safelyGetDataValue(series, index, xData);
    }

    const alignmentDiff = Number(ySeries.alignment - xSeries.alignment);
    result.value.y = Number(ySeries.at(index - alignmentDiff));

    result.bounds = { ...ySeries.bounds };

    result.position = {
      x: props.dataToDecimalScale.x.pos(result.value.x),
      y: props.dataToDecimalScale.y.pos(result.value.y),
    };
    return result;
  }

  render(props: LineProps): void {
    if (this.deleted || !this.state.visible) return;
    const { downsample } = this.state;
    const { xTelem, yTelem, lineCtx: ctx, xDownsampler, yDownsampler } = this.internal;

    const { dataToDecimalScale, exposure } = props;
    let [[, xData], [, yData]] = [xTelem.value(), yTelem.value()];
    xData = xDownsampler.transform(xData);
    yData = yDownsampler.transform(yData);
    xData.updateGLBuffer(ctx.gl);
    yData.updateGLBuffer(ctx.gl);
    if (xData.length === 0 || yData.length === 0) return;
    const prog = ctx.getProgram(yData.dataType);
    const ops = buildDrawOperations(
      xData,
      yData,
      exposure,
      downsample,
      this.state.downsampleMode,
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
    const regionTransform = prog.renderCtx.scaleRegion(props.region).transform;
    ops.forEach((op) => {
      const scaleTransform = offsetScale(dataToDecimalScale, op).transform;
      prog.bindScale(scaleTransform, regionTransform);
      prog.draw(op, instances, xData.dataType, yData.dataType);
    });
    clearProg();
    if (ops.length < 2) return;
    const bridgeProg = ctx.getProgram(DataType.FLOAT32);
    const clearBridgeProg = bridgeProg.setAsActive();
    const bridgeInstances = bridgeProg.bindState(this.state);
    bridgeProg.bindScale(scale.XY.IDENTITY.transform, regionTransform);
    bridgeProg.segments(bridgeVertices(ops, dataToDecimalScale), bridgeInstances);
    clearBridgeProg();
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
  xSeries: MultiSeries,
  ySeries: MultiSeries,
  exposure: number,
  userSpecifiedDownSampling: number,
  downsampleMode: telem.DownsampleMode,
  overlapThreshold: TimeSpan,
): DrawOperation[] => {
  if (xSeries.series.length === 0 || ySeries.series.length === 0) return [];
  const ops: DrawOperation[] = [];
  xSeries.series.forEach((x) =>
    ySeries.series.forEach((y) => {
      const op = buildDrawOperation(
        x,
        y,
        exposure,
        userSpecifiedDownSampling,
        downsampleMode,
        overlapThreshold,
      );
      if (op != null) ops.push(op);
    }),
  );
  return ops;
};

const buildDrawOperation = (
  x: Series,
  y: Series,
  exposure: number,
  userSpecifiedDownSampling: number,
  downsampleMode: telem.DownsampleMode,
  overlapThreshold: TimeSpan,
): DrawOperation | null => {
  if (!seriesOverlap(x, y, overlapThreshold)) return null;
  let xAlignmentOffset = 0n;
  let yAlignmentOffset = 0n;
  // This means that the x series starts before the y series.
  if (x.alignment < y.alignment) xAlignmentOffset = y.alignment - x.alignment;
  // This means that the y series starts before the x series.
  else if (y.alignment < x.alignment) yAlignmentOffset = x.alignment - y.alignment;
  // The total number of alignment steps that are common to the two series.
  const alignmentCount = math.min(
    bounds.span(x.alignmentBounds) - xAlignmentOffset,
    bounds.span(y.alignmentBounds) - yAlignmentOffset,
  );
  if (alignmentCount === 0n) return null;
  let downsample = bounds.clamp(
    {
      lower: userSpecifiedDownSampling,
      upper: 51,
    },
    Math.round(exposure * 4 * Number(alignmentCount)),
  );
  if (downsampleMode !== "decimate") downsample = 1;
  const count = Number(alignmentCount / x.alignmentMultiple);
  const xOffset = Number(xAlignmentOffset / x.alignmentMultiple);
  const yOffset = Number(yAlignmentOffset / y.alignmentMultiple);
  return { x, y, xOffset, yOffset, count, downsample };
};

/** @returns the number of vertices in the strip op draws. */
const vertexCount = (op: DrawOperation): number => Math.floor(op.count / op.downsample);

/** @returns the position in the strip of the last vertex op draws. */
const lastVertex = (op: DrawOperation): number => Math.max(vertexCount(op) - 1, 0);

/** @returns the x index of the vertex op draws nearest to the given x index. */
export const nearestVertex = (op: DrawOperation, index: number): number => {
  const vertex = Math.round((index - op.xOffset) / op.downsample);
  const clamped = bounds.clamp({ lower: 0, upper: lastVertex(op) }, vertex);
  return op.xOffset + clamped * op.downsample;
};

/** @returns interleaved decimal x and y of the segments joining each op to the next. */
export const bridgeVertices = (ops: DrawOperation[], s: scale.XY): Float32Array => {
  const vertices = new Float32Array((ops.length - 1) * 4);
  for (let i = 1; i < ops.length; i++) {
    const a = ops[i - 1];
    const b = ops[i];
    const last = lastVertex(a) * a.downsample;
    const j = (i - 1) * 4;
    vertices[j] = s.x.pos(Number(a.x.at(a.xOffset + last, true)));
    vertices[j + 1] = s.y.pos(Number(a.y.at(a.yOffset + last, true)));
    vertices[j + 2] = s.x.pos(Number(b.x.at(b.xOffset, true)));
    vertices[j + 3] = s.y.pos(Number(b.y.at(b.yOffset, true)));
  }
  return vertices;
};

const digests = (ops: DrawOperation[]): DrawOperationDigest[] =>
  ops.map((op) => ({ ...op, x: op.x.digest, y: op.y.digest }));
