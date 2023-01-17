import { Compiler } from "./compiler";
import { GLRendererRegistry } from "./registry";
import { ScissoredGLRenderer, ScissoredRenderRequest } from "./scissor";

import { Box, calculateBottomOffset, XY } from "@/spatial";

import { RenderingUnits } from "./types";

/**
 * A renderer for a specific type of entity. A renderer should not maintain any internal

 * state relating to specific entities, but should instead rely on the request properties
 * to determine how to render it.
 */
export interface GLRenderer<R> extends Compiler {
  /** Type is a unique type for the renderer. */
  type: string;
  /** Renders the given entity under the RenderingContext.  */
  render: (ctx: GLContext, req: R) => void;
}

export class GLContext {
  readonly gl: WebGLRenderingContext;
  readonly registry: GLRendererRegistry;
  private readonly canvas: HTMLCanvasElement;

  constructor(canvas: HTMLCanvasElement, registry: GLRendererRegistry) {
    this.canvas = canvas;
    const gl = canvas.getContext("webgl", { preserveDrawingBuffer: true });
    if (gl == null) throw new Error("Could not get WebGL context");
    this.gl = gl;
    this.registry = registry;
    this.registry.compile(this.gl);
  }

  get aspect(): number {
    const b = this.canvasBox;
    return b.width / b.height;
  }

  get dpr(): number {
    return window.devicePixelRatio ?? 1;
  }

  scale(box: Box): XY {
    const c = this.canvasBox;
    return { x: box.width / c.width, y: box.height / c.height };
  }

  private offsetPx(box: Box): XY {
    const c = this.canvasBox;
    const relLeft = box.left - c.left;
    const bot = calculateBottomOffset(c, box);
    return { y: bot, x: relLeft };
  }

  offset(box: Box, units: RenderingUnits = "decimal"): XY {
    const dims = this.offsetPx(box);
    if (units === "decimal") return this.normalizeDims(dims);
    return dims;
  }

  private normalizeDims(dims: XY): XY {
    return {
      x: (dims.x * this.dpr) / this.canvas.width,
      y: (dims.y * this.dpr) / this.canvas.height,
    };
  }

  scissor<R extends ScissoredRenderRequest>(
    wrap: GLRenderer<R>,
    overscan?: XY
  ): ScissoredGLRenderer<R> {
    return new ScissoredGLRenderer<R>(wrap, overscan);
  }

  private get canvasBox(): Box {
    return new Box(this.canvas.getBoundingClientRect());
  }

  refreshCanvas(): void {
    if (this.maybeResetDisplaySize())
      this.gl.viewport(0, 0, this.canvas.width, this.canvas.height);
  }

  private maybeResetDisplaySize(): boolean {
    const { canvas } = this;
    const { clientWidth: dw, clientHeight: dh, width: w, height: h } = canvas;
    const needResize = w !== dw || h !== dh;
    if (needResize) [canvas.width, canvas.height] = [dw * this.dpr, dh * this.dpr];
    return needResize;
  }
}
