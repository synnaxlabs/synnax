import { XYTuple } from "./xy";
import { RGBATuple } from "./color";

const glslScaleKey = "u_scale";
const glslOffsetKey = "u_offset";
const glslColorKey = "u_color";
const glslCoordinatesKey = "u_coordinates";

export type GLSLShaderGeneratorProps = {
  webgl: WebGL2RenderingContext;
  scaleKey: string;
  offsetKey: string;
  colorKey: string;
  coordinateKey: string;
};

export type GLSLShaderCode = {
  type: number;
  code: string;
};

export type GLSLProgramBuilderProps = {
  gl: WebGL2RenderingContext;
};

export class GLSLProgrammer {
  gl: WebGL2RenderingContext;
  program: WebGLProgram;

  constructor(props: GLSLProgramBuilderProps) {
    this.gl = props.gl;
    this.program = props.gl.createProgram() as WebGLProgram;
  }

  registerShader(
    generator: (props: GLSLShaderGeneratorProps) => GLSLShaderCode
  ) {
    const { type, code } = generator({
      webgl: this.gl,
      coordinateKey: glslCoordinatesKey,
      scaleKey: glslScaleKey,
      offsetKey: glslOffsetKey,
      colorKey: glslColorKey,
    });
    const shader = this.gl.createShader(type) as WebGLShader;
    this.gl.shaderSource(shader, code);
    this.gl.compileShader(shader);
    this.gl.attachShader(this.program, shader);
  }

  build(): GLSLProgram {
    this.gl.linkProgram(this.program);
    return new GLSLProgram(this.gl, this.program);
  }
}

export class GLSLProgram {
  gl: WebGL2RenderingContext;
  program: WebGLProgram;
  buffer: Float32Array | null;

  constructor(webgl: WebGL2RenderingContext, program: WebGLProgram) {
    this.gl = webgl;
    this.program = program;
    this.constructCoordinateBuffer();
    this.buffer = null;
  }

  setActive() {
    this.gl.useProgram(this.program);
  }

  setScale(scale: XYTuple) {
    const uScale = this.gl.getUniformLocation(this.program, glslScaleKey);
    this.gl.uniformMatrix2fv(
      uScale,
      false,
      new Float32Array([scale[0], 0, 0, scale[1]])
    );
  }

  setOffset(offset: XYTuple) {
    const uOffset = this.gl.getUniformLocation(this.program, glslOffsetKey);
    this.gl.uniform2fv(uOffset, new Float32Array(offset));
  }

  setColor(color: RGBATuple) {
    const uColor = this.gl.getUniformLocation(this.program, glslColorKey);
    this.gl.uniform4fv(uColor, new Float32Array(color));
  }

  draw(points: Float32Array) {
    const bufferNow = performance.now();
    if (!this.buffer) {
      this.gl.bufferData(
        this.gl.ARRAY_BUFFER,
        points as ArrayBuffer,
        this.gl.DYNAMIC_DRAW
      );
      this.buffer = points;
    }
    const bufferTime = performance.now() - bufferNow;
    const drawNow = performance.now();
    this.gl.drawArrays(this.gl.LINE_STRIP, 0, points.length / 2);
    const drawTime = performance.now() - drawNow;
  }

  constructCoordinateBuffer() {
    const buf = this.gl.createBuffer();
    this.gl.bindBuffer(this.gl.ARRAY_BUFFER, buf);
    const aCoordinate = this.gl.getAttribLocation(
      this.program,
      glslCoordinatesKey
    );
    this.gl.vertexAttribPointer(aCoordinate, 2, this.gl.FLOAT, false, 0, 0);
    this.gl.enableVertexAttribArray(aCoordinate);
  }
}
