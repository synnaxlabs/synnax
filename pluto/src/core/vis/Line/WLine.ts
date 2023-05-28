import {
  Direction,
  Bound,
  Box,
  Scale,
  XYScale,
  XYTransform,
  xyScaleToTransform,
} from "@synnaxlabs/x";

import { hexToRGBA } from "@/core/color";
import { GLProgram, errorUnsupported } from "@/core/vis/render";
import {
  DynamicXYTelem,
  DynamicXYTelemMeta,
  TelemProvider,
  XYTelem,
  XYTelemMeta,
} from "@/core/vis/telem";
// eslint-disable-next-line import/no-unresolved
import FRAG_SHADER from "@/vis/core/Line/frag.glsl?raw";
// eslint-disable-next-line import/no-unresolved
import VERT_SHADER from "@/vis/core/Line/vert.glsl?raw";
import { RenderContext } from "../render/RenderContext";

export interface LineProps {
  /** A unique key identifying the line within the worker DOM */
  key: string;
  /** The telemetry to read from */
  telem: XYTelemMeta | DynamicXYTelemMeta;
  /** A hex color string to color the line */
  color: string;
  /** The stroke width of the line in pixels */
  strokeWidth: number;
}

export interface LineContext {
  /**
   * A box in pixel space representing the region of the display that the line
   * should be rendered in. The root of the pixel coordinate system is the top
   * left of the canvas.
   */
  region: Box;
  /**
   * An XY scale that maps from the data space to decimal space rooted in the
   * bottom of the region.
   */
  scale: XYScale;
}

const ANGLE_INSTANCED_ARRAYS_FEATURE = "ANGLE_instanced_arrays";

/**
 * A factory for creating webgl rendered lines.
 */
export class LineFactory {
  private readonly program: GLLineProgram;
  private readonly telem: TelemProvider;

  /**
   * @param ctx - The webgl rendering context to use.
   * @param telem - A function that returns the telemetry provider.
   */
  constructor(ctx: RenderContext, telem: TelemProvider) {
    this.program = new GLLineProgram(ctx);
    this.telem = telem;
  }

  /**
   * Creates a new line.
   * @param props - The properties of the line.
   * @param requestRender - A function that allows the line to request that its parent re-render it.
   */
  new(props: LineProps, requestRender: () => void): GLLine {
    return new GLLine(props, this.program, requestRender, this.telem);
  }
}

class GLLineProgram extends GLProgram {
  extension: ANGLE_instanced_arrays;
  translationBuffer: WebGLBuffer;

  static readonly VARS = {
    viewport: {
      scale: "u_scale_viewport",
      offset: "u_offset_viewport",
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

  constructor(ctx: RenderContext) {
    super(ctx, VERT_SHADER, FRAG_SHADER);
    const ext = ctx.gl.getExtension(ANGLE_INSTANCED_ARRAYS_FEATURE);
    if (ext == null) throw errorUnsupported(ANGLE_INSTANCED_ARRAYS_FEATURE);
    this.extension = ext;
    this.translationBuffer = ctx.gl.createBuffer() as WebGLBuffer;
  }

  bindAttrBuffer(dir: Direction, buffer: WebGLBuffer): void {
    const gl = this.ctx.gl;
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    const n = gl.getAttribLocation(gl, GLLineProgram.VARS[dir]);
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
  attrStrokeWidth(aspect: number, strokeWidth: number): number {
    const gl = this.ctx.gl;
    gl.bindBuffer(gl.ARRAY_BUFFER, this.translationBuffer);
    const translationBuffer = newTranslationBuffer(aspect, strokeWidth);
    gl.bufferData(gl.ARRAY_BUFFER, translationBuffer, gl.DYNAMIC_DRAW);

    const loc = gl.getAttribLocation(this.prog, GLLineProgram.VARS.translate);
    gl.vertexAttribPointer(loc, 2, gl.FLOAT, false, 0, 0);
    gl.enableVertexAttribArray(loc);
    this.extension.vertexAttribDivisorANGLE(loc, 1);

    const numInstances = translationBuffer.length / 2;
    return numInstances;
  }
}

export class GLLine {
  props: LineProps;
  prog: GLLineProgram;
  requestRender: () => void;
  telemProv: TelemProvider;
  telem: XYTelem | DynamicXYTelem;

  static readonly TYPE = "line";

  constructor(
    initialProps: LineProps,
    program: GLLineProgram,
    requestRender: () => void,
    telemProv: TelemProvider
  ) {
    this.props = initialProps;
    this.prog = program;
    this.requestRender = requestRender;
    this.telemProv = telemProv;
    this.telem = this.telemProv<XYTelem>(this.props.key);
  }

  get key(): string {
    return this.props.key;
  }

  setProps(props: LineProps): void {
    if (props.telem.key !== this.props.telem.key)
      this.telem = this.telemProv<XYTelem>(props.telem.key);
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
    const xData = await this.telem.x();
    const yData = await this.telem.y();
    const regionTransform = xyScaleToTransform(this.prog.ctx.scaleRegion(ctx.region));
    const scaleTransform = xyScaleToTransform(ctx.scale);

    this.prog.uniformXY(GLLineProgram.VARS.viewport.scale, regionTransform.scale);
    this.prog.uniformXY(GLLineProgram.VARS.viewport.offset, regionTransform.offset);
    this.prog.uniformColor(GLLineProgram.VARS.color, hexToRGBA(this.props.color));
    this.prog.uniformXY(GLLineProgram.VARS.scale, scaleTransform.scale);
    this.prog.uniformXY(GLLineProgram.VARS.offset, scaleTransform.offset);

    const numInstances = this.prog.attrStrokeWidth(
      this.prog.ctx.aspect,
      this.props.strokeWidth
    );
    xData.forEach((x, i) => {
      const y = yData[i];
      this.prog.bindAttrBuffer("x", x.buffer);
      this.prog.bindAttrBuffer("y", y.buffer);
      this.prog.extension.drawArraysInstancedANGLE(
        this.prog.ctx.gl.LINE_STRIP,
        0,
        x.length,
        numInstances
      );
    });
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
