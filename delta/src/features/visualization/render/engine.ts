import { Box, calculateBottom } from "./box";
import { Line, LineRenderer, XY } from "./line";

export interface RenderRequest {
  box: Box;
  lines: Array<Omit<Line, "rootScale" | "rootOffset">>;
}

const DPR = window.devicePixelRatio ?? 1;
const CLEAR_OVERSCAN_Y = 2 * 12 + 24;
const CLEAR_OVERSCAN_X = 2 * 12 + 1;

export class RenderingEngine {
  private readonly canvas: HTMLCanvasElement;
  private readonly gl: WebGLRenderingContext;
  private readonly line: LineRenderer;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    this.gl = this.canvas.getContext("webgl", {
      preserveDrawingBuffer: true,
    }) as WebGLRenderingContext;
    this.line = new LineRenderer(this.gl);
  }

  render(req: RenderRequest): void {
    this.refreshCanvas();
    this.clear(req.box);
    const scale = this.rootScale(req.box);
    const offset = this.rootOffset(req.box);
    req.lines.forEach((l) => this.renderLine(l, scale, offset));
    this.gl.disable(this.gl.SCISSOR_TEST);
  }

  clear(box: Box): void {
    this.gl.enable(this.gl.SCISSOR_TEST);
    const { x, y } = this.relativeOffset(box);
    this.gl.scissor(
      x * DPR,
      y * DPR,
      (box.width + CLEAR_OVERSCAN_X) * DPR,
      (box.height + CLEAR_OVERSCAN_Y) * DPR
    );
    this.gl.clearColor(0, 0, 0, 0);
    this.gl.clear(this.gl.COLOR_BUFFER_BIT);
  }

  private relativeOffset(box: Box): XY {
    const rect = this.canvas.getBoundingClientRect();
    const relLeft = box.left - rect.left;
    const bot = calculateBottom(rect, box);
    return { y: bot, x: relLeft };
  }

  private renderLine(
    line: Omit<Line, "rootScale" | "rootOffset">,
    rootScale: XY,
    rootOffset: XY
  ): void {
    this.line.render({
      ...line,
      rootScale,
      rootOffset,
    });
  }

  private rootScale(box: Box): XY {
    const rect = this.canvas.getBoundingClientRect();
    const x = box.width / rect.width;
    const y = box.height / rect.height;
    return { x, y };
  }

  private rootOffset(box: Box): XY {
    const rect = this.canvas.getBoundingClientRect();
    const { x, y } = this.relativeOffset(box);
    return { x: x / rect.width, y: y / rect.height };
  }

  private refreshCanvas(): void {
    this.maybeResetDisplaySize();
    this.gl.viewport(0, 0, this.canvas.width, this.canvas.height);
  }

  private maybeResetDisplaySize(): void {
    const { canvas } = this;
    const { clientWidth: dw, clientHeight: dh, width: w, height: h } = canvas;
    const needResize = w !== dw || h !== dh;
    if (needResize) [canvas.width, canvas.height] = [dw * DPR, dh * DPR];
  }
}
