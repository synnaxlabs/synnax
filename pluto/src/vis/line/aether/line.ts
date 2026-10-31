// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import { status, UnexpectedError } from "@synnaxlabs/client";
import {
  type Aggregation,
  bounds,
  box,
  color,
  DataType,
  type destructor,
  type direction,
  type MultiSeries,
  POINTS_PER_GROUP,
  scale,
  type Series,
  type SeriesDigest,
  TimeSpan,
  xy,
} from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { telem } from "@/telem/aether";
import { forEachPair, windowBounds } from "@/vis/line/aether/bounds";
import {
  bridgeVertices,
  buildDrawOperations,
  DEFAULT_OVERLAP_THRESHOLD,
  type DrawOperation,
  finiteRuns,
} from "@/vis/line/aether/draw";
import FRAG_SHADER from "@/vis/line/aether/frag.glsl?raw";
import { groupSize, Reducer, type View } from "@/vis/line/aether/reduce";
import F32_VERT_SHADER from "@/vis/line/aether/vert_f32.glsl?raw";
import HYBRID_VERT_SHADER from "@/vis/line/aether/vert_hybrid.glsl?raw";
import { problem } from "@/vis/problem/aether";
import { render } from "@/vis/render";

export const stateZ = z.object({
  x: telem.seriesSourceSpecZ,
  y: telem.seriesSourceSpecZ,
  label: z.string().optional(),
  color: color.colorZ,
  strokeWidth: z.number().default(1),
  visible: z.boolean().default(true),
  /** The number of aggregation groups the line draws per pixel column. */
  groupsPerColumn: z
    .number()
    .positive()
    .default(1 / 2),
  /** True when the line reports no loading while it has no data. */
  loadingHidden: z.boolean().default(false),
});

// A zoom waits this long for its span to hold before the line's sources read for it.
const ZOOM_SETTLE = TimeSpan.milliseconds(150);

// The line reduces data by this when its y source reports no aggregation.
const DEFAULT_AGGREGATION: Aggregation = "min_max";

type Result = telem.Result<[bounds.Bounds, MultiSeries]>;

const isLoading = ({ status }: Result): boolean => status.variant === "loading";

type LoadingState = "none" | "loading" | "fetching";

const loadingState = (
  x: Result,
  y: Result,
  loaded: boolean,
  hidden: boolean,
): LoadingState => {
  if (!isLoading(x) && !isLoading(y)) return "none";
  if (loaded || (x.data[1].length > 0 && y.data[1].length > 0)) return "fetching";
  return hidden ? "none" : "loading";
};

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

export interface FindResult {
  key: string;
  position: xy.XY;
  value: xy.XY;
  color: color.Color;
  label?: string;
  units?: string;
  bounds: bounds.Bounds;
}

/** Bounds of a line's data, and whether a source still loads the data. */
export interface DataBounds {
  bounds: bounds.Bounds;
  loading: boolean;
}

export interface LineProps {
  /**
   * A box in pixel space representing the region of the display that the line should be
   * rendered in. The root of the pixel coordinate system is the top left of the canvas.
   */
  region: box.Box;
  /** An XY scale that maps from the data space to decimal space. */
  dataToDecimalScale: scale.XY;
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

  draw(op: DrawOperation, instances: number): void {
    const { gl } = this.renderCtx;
    const { x, y, xOffset, yOffset } = op;
    this.bindAttrBuffer("x", x.glBuffer, 1, xOffset, x.glDataType);
    this.bindAttrBuffer("y", y.glBuffer, 1, yOffset, y.glDataType);
    for (const [first, count] of finiteRuns(op))
      gl.drawArraysInstanced(gl.LINE_STRIP, first, count, instances);
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

  /**
   * Binds buffer to the attribute of dir, reading one value every stride values from
   * the value at alignment.
   */
  private bindAttrBuffer(
    dir: direction.Crude,
    buffer: WebGLBuffer,
    stride: number,
    alignment: number,
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
        density * stride,
        density * alignment,
      );
    else
      gl.vertexAttribPointer(
        aLoc,
        1,
        glDataType,
        false,
        density * stride,
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

interface Programs {
  // Uint8 hybrid program is used for high performance rendering of uint8 data along
  // with float32 timestamp data. It's used as a hot path optimization for common
  // channel such as actuator states.
  uint8Hybrid: GLProgram;
  // Float32 program is used for rendering float32 data. It's used for all other
  // channel types.
  float32: GLProgram;
}

const compile = (ctx: render.Context): Programs => ({
  uint8Hybrid: new GLProgram(ctx, HYBRID_VERT_SHADER, FRAG_SHADER),
  float32: new GLProgram(ctx, F32_VERT_SHADER, FRAG_SHADER),
});

export class Context {
  private static readonly CONTEXT_KEY = "pluto-line-gl-program";
  readonly renderCtx: render.Context;
  private programs: Programs;

  private constructor(renderCtx: render.Context) {
    this.renderCtx = renderCtx;
    this.programs = compile(renderCtx);
    renderCtx.onRestore(() => {
      this.programs = compile(renderCtx);
    });
  }

  getProgram(dataType: DataType): GLProgram {
    if (dataType.equals(DataType.UINT8)) return this.programs.uint8Hybrid;
    return this.programs.float32;
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
  reducer: Reducer;
  // zoom is the log2 span of the last bounds drawn, and settleTimer runs until that
  // zoom has held for ZOOM_SETTLE.
  zoom?: number;
  settleTimer?: ReturnType<typeof setTimeout>;
  // aggregation, loadingState, and problems come from the data the line last drew.
  aggregation?: Aggregation;
  loadingState?: LoadingState;
  problems?: problem.Problem[];
  // loaded is true once both sources leave loading, until either source changes.
  loaded?: boolean;
}

export class Line extends aether.Leaf<typeof stateZ, InternalState> {
  static readonly TYPE = "line";
  schema: typeof stateZ = stateZ;

  afterUpdate(ctx: aether.Context): void {
    const { internal: i } = this;
    const { xTelem, yTelem } = i;
    i.xTelem = telem.useSource(ctx, this.state.x, i.xTelem);
    i.yTelem = telem.useSource(ctx, this.state.y, i.yTelem);
    if (i.xTelem !== xTelem || i.yTelem !== yTelem) i.loaded = false;
    i.instrumentation = alamos.useInstrumentation(ctx, "line");
    i.lineCtx = Context.use(ctx);
    i.requestRender = render.useRequestor(ctx);
    i.reducer ??= new Reducer();
    i.stopListeningXTelem?.();
    i.stopListeningYTelem?.();
    i.stopListeningXTelem = i.xTelem.onChange(() => i.requestRender("data"));
    i.stopListeningYTelem = i.yTelem.onChange(() => i.requestRender("data"));
    i.requestRender("layout");
  }

  afterDelete(): void {
    const { internal: i } = this;
    clearTimeout(i.settleTimer);
    i.xTelem.cleanup?.();
    i.yTelem.cleanup?.();
    i.reducer.release();
    i.requestRender("layout");
  }

  /** True while the first load of a source runs and the line drew no data for it. */
  get loading(): boolean {
    return this.internal.loadingState === "loading";
  }

  /** True while a source loads after its first load, or while the line drew data. */
  get fetching(): boolean {
    return this.internal.loadingState === "fetching";
  }

  /** The problems of the data the line last drew, named by its label. */
  get problems(): problem.Problem[] {
    const { problems = [] } = this.internal;
    return problems;
  }

  xBounds(): DataBounds {
    const x = this.internal.xTelem.value();
    return { bounds: x.data[0], loading: isLoading(x) };
  }

  /**
   * @param xWindow - the visible x range. Bounds cover only samples whose x value
   * falls inside it; when the window is non-finite or clips out every sample, the
   * source's full bounds are used instead.
   * @returns the y bounds of this line's samples inside the window.
   */
  yBounds(xWindow: bounds.Bounds): DataBounds {
    const { xTelem, yTelem } = this.internal;
    const valueProps = { extremesKept: true };
    const y = yTelem.value(valueProps);
    const [b, yData] = y.data;
    if (!bounds.isFinite(xWindow)) return { bounds: b, loading: isLoading(y) };
    const x = xTelem.value(valueProps);
    return {
      bounds: windowBounds(x.data[1], yData, xWindow, DEFAULT_OVERLAP_THRESHOLD, b),
      loading: isLoading(x) || isLoading(y),
    };
  }

  findByXValue(props: LineProps, target: number): FindResult {
    const { xTelem, yTelem } = this.internal;
    const valueProps = this.read(view(props));
    const [, xData] = xTelem.value(valueProps).data;
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
    const [, yData] = yTelem.value(valueProps).data;
    const alignment = xSeries.alignment + BigInt(index) * xSeries.alignmentMultiple;
    let ySeries: Series | undefined;
    forEachPair(
      [xSeries],
      yData.series,
      DEFAULT_OVERLAP_THRESHOLD,
      (_, y, start, end) => {
        if (alignment >= start && alignment < end) ySeries ??= y;
      },
    );
    if (ySeries == null) return result;
    result.value.x = safelyGetDataValue(series, index, xData);

    result.value.y = Number(ySeries.atAlignment(alignment, true));

    result.bounds = { ...ySeries.bounds };

    result.position = {
      x: props.dataToDecimalScale.x.pos(result.value.x),
      y: props.dataToDecimalScale.y.pos(result.value.y),
    };
    return result;
  }

  // read returns the value props of the view. A zoom gives the sources points only once
  // its span holds, so a fast zoom does not read the spans it passes through. A pan
  // reads at once.
  private read({ bounds: b, width }: View): telem.ValueProps {
    const { internal: i } = this;
    const zoom = Math.ceil(Math.log2(bounds.span(b)));
    if (zoom !== i.zoom) {
      i.zoom = zoom;
      clearTimeout(i.settleTimer);
      i.settleTimer = setTimeout(() => {
        i.settleTimer = undefined;
        i.requestRender("data");
      }, ZOOM_SETTLE.milliseconds);
    }
    if (i.settleTimer != null) return { bounds: b };
    const perGroup = POINTS_PER_GROUP[i.aggregation ?? DEFAULT_AGGREGATION];
    return { bounds: b, points: width * this.state.groupsPerColumn * perGroup };
  }

  render(props: LineProps): void {
    const { internal: i } = this;
    if (this.deleted) return;
    if (!this.state.visible) {
      i.loadingState = undefined;
      i.problems = [];
      return;
    }
    const { xTelem, yTelem, lineCtx: ctx, reducer } = i;
    const { groupsPerColumn, loadingHidden, label = "" } = this.state;
    const { dataToDecimalScale } = props;
    const v = view(props);
    const valueProps = this.read(v);
    const x = xTelem.value(valueProps);
    const y = yTelem.value(valueProps);
    i.loadingState = loadingState(x, y, i.loaded === true, loadingHidden);
    i.problems = problem.fromStatus(status.worst(x.status, y.status)!, label);
    if (!isLoading(x) && !isLoading(y)) i.loaded = true;
    // Full-resolution data reduces with the aggregation of the source's reduced reads.
    i.aggregation = y.aggregation ?? DEFAULT_AGGREGATION;
    const full = buildDrawOperations(x.data[1], y.data[1], DEFAULT_OVERLAP_THRESHOLD);
    const size = groupSize(full, v, groupsPerColumn);
    const ops = reducer.reduce(full, i.aggregation, size);
    if (ops.length === 0) return;
    ops.forEach(({ x, y }) => {
      x.updateGLBuffer(ctx.renderCtx);
      y.updateGLBuffer(ctx.renderCtx);
    });
    const prog = ctx.getProgram(ops[0].y.glDataType);
    this.internal.instrumentation.L.debug("render", () => ({
      key: this.key,
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
      prog.draw(op, instances);
    });
    clearProg();
    const bridges = bridgeVertices(ops, dataToDecimalScale);
    if (bridges.length === 0) return;
    const bridgeProg = ctx.getProgram(DataType.FLOAT32);
    const clearBridgeProg = bridgeProg.setAsActive();
    const bridgeInstances = bridgeProg.bindState(this.state);
    bridgeProg.bindScale(scale.XY.IDENTITY.transform, regionTransform);
    bridgeProg.segments(bridges, bridgeInstances);
    clearBridgeProg();
  }
}

// view returns the part of the x range that the region shows.
const view = ({ region, dataToDecimalScale }: LineProps): View => {
  const decimalToData = dataToDecimalScale.x.reverse();
  return {
    bounds: { lower: decimalToData.pos(0), upper: decimalToData.pos(1) },
    width: box.width(region),
  };
};

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
    scale.x.dim(Number(op.x.glOffset)),
    scale.y.dim(Number(op.y.glOffset)),
  );

export const REGISTRY: aether.ComponentRegistry = { [Line.TYPE]: Line };

interface DrawOperationDigest extends Omit<DrawOperation, "x" | "y"> {
  x: SeriesDigest;
  y: SeriesDigest;
}

const digests = (ops: DrawOperation[]): DrawOperationDigest[] =>
  ops.map((op) => ({ ...op, x: op.x.digest, y: op.y.digest }));
