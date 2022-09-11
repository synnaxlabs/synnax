import Canvas from "./Canvas";
import {
  GLSLProgrammer,
  GLSLProgram,
  GLSLShaderGeneratorProps,
  GLSLShaderCode,
} from "./GLSLProgrammer";
import { addXYTuples, multiplyXYTuples, XYTransform } from "./xy";
import { RGBATuple } from "./color";

export type Line = {
  key: string;
  points: Float32Array;
  color: RGBATuple;
  visible: boolean;
  transform: XYTransform;
};

export default class LinePlotter {
  program: GLSLProgram;
  lines: Line[] = [];
  transform: XYTransform;

  constructor(engine: Canvas) {
    this.program = buildLinePlotterProgram(engine.webgl);
    this.transform = { translation: [0, 0], scale: [1, 1] };
  }

  setTransform(transform: XYTransform) {
    this.transform = transform;
  }

  plot() {
    this.program.setActive();
    this.lines.forEach((line) => {
      if (!line.visible) return;
      const scale = multiplyXYTuples(
        line.transform.scale,
        this.transform.scale
      );
      const offset = addXYTuples(
        line.transform.translation,
        this.transform.translation
      );
      this.program.setOffset(offset);
      this.program.setColor(line.color);
      this.program.setScale(scale);
      this.program.draw(line.points);
    });
  }

  addLine(line: Line) {
    this.lines.push(line);
  }

  addLines(lines: Line[]) {
    this.lines.push(...lines);
  }

  removeLines(...keys: string[]) {
    this.lines = this.lines.filter((line) => !keys.includes(line.key));
  }
}

// |||||| GLSL CODE ||||||

const vertCodeGenerator = (
  props: GLSLShaderGeneratorProps
): GLSLShaderCode => ({
  type: props.webgl.VERTEX_SHADER,
  code: `
      attribute vec2 ${props.coordinateKey};
      uniform mat2 ${props.scaleKey};
      uniform vec2 ${props.offsetKey};
      void main(void) {
         vec2 line = vec2(${props.coordinateKey}.x, ${props.coordinateKey}.y);
         gl_Position = vec4(${props.scaleKey}*line + ${props.offsetKey}, 0.0, 1.0);
    }`,
});

const fragCodeGenerator = (
  props: GLSLShaderGeneratorProps
): GLSLShaderCode => ({
  type: props.webgl.FRAGMENT_SHADER,
  code: `
         precision mediump float;
         uniform highp vec4 ${props.colorKey};
         void main(void) {
            gl_FragColor =  ${props.colorKey};
         }`,
});

const buildLinePlotterProgram = (gl: WebGL2RenderingContext): GLSLProgram => {
  const builder = new GLSLProgrammer({ gl: gl });
  builder.registerShader(vertCodeGenerator);
  builder.registerShader(fragCodeGenerator);
  return builder.build();
};
