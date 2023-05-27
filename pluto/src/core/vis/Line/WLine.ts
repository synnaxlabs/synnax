import { Direction, Transform, Bound } from "@synnaxlabs/x";

import { WComponent, WorkerMessage } from "../worker/worker";

import { hexToRGBA } from "@/core/color";
import { errorUnsupported, Program } from "@/core/vis/gl";
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

export interface WLineProps {
  key: string;
  telem: XYTelemMeta | DynamicXYTelemMeta;
  color: string;
  strokeWidth: number;
}

export interface WLineContext {
  aspect: number;
  viewport: Transform;
  transform: Transform;
}

const ANGLE_INSTANCED_ARRAYS_FEATURE = "ANGLE_instanced_arrays";

export class WLineProgram extends Program {
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

  constructor(gl: WebGL2RenderingContext) {
    super(gl, VERT_SHADER, FRAG_SHADER);
    const ext = gl.getExtension(ANGLE_INSTANCED_ARRAYS_FEATURE);
    if (ext == null) throw errorUnsupported(ANGLE_INSTANCED_ARRAYS_FEATURE);
    this.extension = ext;
    this.translationBuffer = gl.createBuffer() as WebGLBuffer;
  }

  bindAttrBuffer(dir: Direction, buffer: WebGLBuffer): void {
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, buffer);
    const n = this.gl.getAttribLocation(this.gl, WLineProgram.VARS[dir]);
    this.gl.vertexAttribPointer(n, 1, this.gl.FLOAT, false, 0, 0);
    this.gl.enableVertexAttribArray(n);
  }

  /**
   * We apply stroke width by drawing the line multiple times, each time with a slight
   * transformation. This is done as simply as possible. We draw the "centered" line
   * and then four more lines: one to the left, one to the right, one above, and one
   * below. This is done by using the `ANGLE_instanced_arrays` extension. We can repeat
   * this process to make the line thicker.
   */
  attrStrokeWidth(aspect: number, strokeWidth: number): number {
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, this.translationBuffer);
    const translationBuffer = newTranslationBuffer(aspect, strokeWidth);
    this.gl.bufferData(this.gl.ARRAY_BUFFER, translationBuffer, this.gl.DYNAMIC_DRAW);

    const loc = this.gl.getAttribLocation(this.program, WLineProgram.VARS.translate);
    this.gl.vertexAttribPointer(loc, 2, this.gl.FLOAT, false, 0, 0);
    this.gl.enableVertexAttribArray(loc);
    this.extension.vertexAttribDivisorANGLE(loc, 1);

    const numInstances = translationBuffer.length / 2;
    return numInstances;
  }
}

export class WLine implements WComponent {
  props: WLineProps;
  prog: WLineProgram;
  requestRender: () => void;
  telemProv: TelemProvider;
  telem: XYTelem | DynamicXYTelem;

  static readonly TYPE = "line";

  constructor(
    initialProps: WLineProps,
    program: WLineProgram,
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

  setProps(props: WLineProps): void {
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

  async render(ctx: WLineContext): Promise<void> {
    const xData = await this.telem.x();
    const yData = await this.telem.y();

    this.prog.uniformXY(WLineProgram.VARS.viewport.scale, ctx.viewport.scale);
    this.prog.uniformXY(WLineProgram.VARS.viewport.offset, ctx.viewport.offset);
    this.prog.uniformColor(WLineProgram.VARS.color, hexToRGBA(this.props.color));
    this.prog.uniformXY(WLineProgram.VARS.scale, ctx.transform.scale);
    this.prog.uniformXY(WLineProgram.VARS.offset, ctx.transform.offset);

    const numInstances = this.prog.attrStrokeWidth(ctx.aspect, this.props.strokeWidth);
    xData.forEach((x, i) => {
      const y = yData[i];
      this.prog.bindAttrBuffer("x", x.buffer);
      this.prog.bindAttrBuffer("y", y.buffer);
      this.prog.extension.drawArraysInstancedANGLE(
        this.prog.gl.LINE_STRIP,
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
