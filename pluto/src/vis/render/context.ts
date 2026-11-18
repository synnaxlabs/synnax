// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Instrumentation } from "@synnaxlabs/alamos";
import {
  box,
  color,
  type destructor,
  dimensions,
  id,
  type runtime,
  scale,
  xy,
} from "@synnaxlabs/x";

import { type aether } from "@/aether/aether";
import { alamos } from "@/alamos/aether";
import { CSS } from "@/css";
import { status } from "@/status/aether";
import { text } from "@/text/aether";
import { SugaredOffscreenCanvasRenderingContext2D } from "@/vis/draw2d/canvas";
import { clear } from "@/vis/render/clear";
import { Loop } from "@/vis/render/loop";
import { applyOverScan } from "@/vis/render/util";

export type Canvas2DVariant = "upper2d" | "lower2d";
export type CanvasGLVariant = "gl";
export type CanvasVariant = Canvas2DVariant | CanvasGLVariant;

/** A frame of bitmaps, holding only the canvases that changed since the last frame. */
export type Frame = Partial<Record<CanvasVariant, ImageBitmap>>;

const CANVAS_VARIANTS: CanvasVariant[] = ["lower2d", "gl", "upper2d"];
const CANVAS_2D_VARIANTS: Canvas2DVariant[] = ["lower2d", "upper2d"];
const MAX_MSAA_SAMPLES = 4;

const applyDefaultCanvasOpts = <T extends OffscreenCanvasRenderingContext2D>(
  canvas: T,
): T => {
  canvas.textRendering = "optimizeSpeed";
  canvas.imageSmoothingEnabled = false;
  return canvas;
};

const WEBGL_OPTS: WebGLContextAttributes = {
  preserveDrawingBuffer: true,
  powerPreference: "high-performance",
  stencil: false,
  depth: false,
};

// A transfer to a bitmap clears the drawing buffer, so the GL draws go to a back buffer
// that keeps its pixels and resolves the multisampling.
const BITMAP_WEBGL_OPTS: WebGLContextAttributes = {
  ...WEBGL_OPTS,
  preserveDrawingBuffer: false,
  antialias: false,
};

interface BackBuffer {
  framebuffer: WebGLFramebuffer;
  renderbuffer: WebGLRenderbuffer;
}

interface Frames {
  port: MessagePort;
  backBuffer: BackBuffer;
  /** A transfer also clears a 2D canvas, so frames copy each one here to transfer. */
  copy: OffscreenCanvasRenderingContext2D;
  /** The canvases that changed since the last frame. */
  dirty: Set<CanvasVariant>;
  /** True from a post until the page acknowledges it, so frames never queue. */
  inFlight: boolean;
}

/**
 * A hybrid rendering context containing both 2D and WebGL canvases and contexts.
 * Implements several utility methods for correctly scaling the canvas, restricting
 * drawing to a region, and erasing portions of the canvas.
 */
export class Context {
  /* The canvas element used by WebGL. */
  readonly glCanvas: OffscreenCanvas;
  /** The canvas element used by the 2D canvas. */
  readonly upper2dCanvas: OffscreenCanvas;
  /** The canvas element used by the 2D canvas. */
  readonly lower2dCanvas: OffscreenCanvas;
  /** The WebGL rendering context.  */
  readonly gl: WebGL2RenderingContext;
  /** A 2D canvas that sits below the WebGL canvas. */
  lower2d: SugaredOffscreenCanvasRenderingContext2D;
  /** A 2D canvas that sits above the WebGL canvas. */
  upper2d: SugaredOffscreenCanvasRenderingContext2D;
  /** The region the canvas occupies in pixel space */
  region: box.Box;
  /** The device pixel ratio of the canvas */
  dpr: number;
  /** queue render transitions onto the stack */
  readonly loop: Loop;
  /** See the @link{clear.Program} for why this is necessary. */
  private clearProgram?: clear.Program;
  private readonly os: runtime.OS;
  private static readonly CONTEXT_KEY = CSS.B("render-context");
  private readonly instrumentation: Instrumentation;
  private readonly addStatus: status.Adder;
  /** One key for the lost and restored statuses, so the restore replaces the loss. */
  private readonly statusKey = id.create();
  private _generation = 0;
  private readonly restoreHandlers = new Set<() => void>();
  private readonly restore2DHandlers = new Set<() => void>();
  private readonly atlases = new text.AtlasRegistry();
  private frames?: Frames;

  /**
   * Creates a render context and sets it on ctx.
   * @param framePort - When set, the context posts each frame to it as a
   * {@link Frame} instead of drawing on canvases shown on the page.
   * @returns null when the canvas gives no WebGL2 context.
   */
  static create(
    ctx: aether.Context,
    glCanvas: OffscreenCanvas,
    lower2dCanvas: OffscreenCanvas,
    upper2dCanvas: OffscreenCanvas,
    os: runtime.OS,
    framePort?: MessagePort,
  ): Context | null {
    const opts = framePort == null ? WEBGL_OPTS : BITMAP_WEBGL_OPTS;
    const gl = glCanvas.getContext("webgl2", opts);
    if (gl == null) return null;
    const instrumentation = alamos.useInstrumentation(ctx, "render_context");
    const render = new Context(
      gl,
      glCanvas,
      lower2dCanvas,
      upper2dCanvas,
      os,
      instrumentation,
      status.useAdder(ctx),
      status.useErrorHandler(ctx),
      framePort,
    );
    ctx.set(Context.CONTEXT_KEY, render);
    return render;
  }

  private constructor(
    gl: WebGL2RenderingContext,
    glCanvas: OffscreenCanvas,
    lower2dCanvas: OffscreenCanvas,
    upper2dCanvas: OffscreenCanvas,
    os: runtime.OS,
    instrumentation: Instrumentation,
    addStatus: status.Adder,
    handleError: status.ErrorHandler,
    framePort?: MessagePort,
  ) {
    this.upper2dCanvas = upper2dCanvas;
    this.lower2dCanvas = lower2dCanvas;
    this.glCanvas = glCanvas;
    this.os = os;
    this.instrumentation = instrumentation;
    this.addStatus = addStatus;
    this.dpr = 1;

    const lowerCtx = this.lower2dCanvas.getContext("2d");
    if (lowerCtx == null) throw new Error("Could not get 2D context");
    this.lower2d = applyDefaultCanvasOpts(
      new SugaredOffscreenCanvasRenderingContext2D(lowerCtx, this.atlases, this.dpr),
    );

    const upperCtx = this.upper2dCanvas.getContext("2d");
    if (upperCtx == null) throw new Error("Could not get 2D context");
    this.upper2d = applyDefaultCanvasOpts(
      new SugaredOffscreenCanvasRenderingContext2D(upperCtx, this.atlases, this.dpr),
    );

    this.gl = gl;
    this.configureGL();
    if (framePort != null) {
      const copy = new OffscreenCanvas(0, 0).getContext("2d");
      if (copy == null) throw new Error("Could not get 2D context");
      const backBuffer = this.createBackBuffer();
      const frames: Frames = {
        port: framePort,
        backBuffer,
        copy,
        dirty: new Set(),
        inFlight: false,
      };
      framePort.onmessage = () => {
        frames.inFlight = false;
      };
      this.frames = frames;
    }

    this.glCanvas.addEventListener("webglcontextlost", (e) => this.lose(e));
    this.glCanvas.addEventListener("webglcontextrestored", () =>
      handleError(() => this.restore(), "Failed to restore WebGL context"),
    );

    this.loop = new Loop({
      afterRender: () => this.gl.flush(),
      present: framePort == null ? undefined : (drawn) => this.present(drawn),
      instrumentation: this.instrumentation,
      handleError,
    });

    this.region = box.ZERO;

    if (this.os === "Windows") this.clearProgram = new clear.Program(this);
  }

  static useOptional(ctx: aether.Context): Context | null {
    return ctx.getOptional<Context>(Context.CONTEXT_KEY);
  }

  static use(ctx: aether.Context): Context {
    return ctx.get<Context>(Context.CONTEXT_KEY);
  }

  update(ctx: aether.Context): void {
    ctx.set(Context.CONTEXT_KEY, this);
  }

  /**
   * Increments each time the WebGL context is restored after a loss. A restore destroys
   * every GL object made in an earlier generation.
   */
  get generation(): number {
    return this._generation;
  }

  /**
   * Calls f each time the WebGL context is restored after a loss. The restore clears
   * every canvas and destroys every GL object made before it.
   * @returns a function that stops the calls.
   */
  onRestore(f: () => void): destructor.Destructor {
    this.restoreHandlers.add(f);
    return () => {
      this.restoreHandlers.delete(f);
    };
  }

  /**
   * Calls f each time a restore rebuilds the 2D canvases, which clears them. The calls
   * happen even when the rest of the restore fails.
   * @returns a function that stops the calls.
   */
  on2DRestore(f: () => void): destructor.Destructor {
    this.restore2DHandlers.add(f);
    return () => {
      this.restore2DHandlers.delete(f);
    };
  }

  private configureGL(): void {
    const { gl } = this;
    gl.disable(gl.DEPTH_TEST);
    gl.disable(gl.STENCIL_TEST);
    gl.disable(gl.CULL_FACE);
    gl.disable(gl.DITHER);
  }

  private lose(event: Event): void {
    // The browser restores only a context whose loss event was canceled.
    event.preventDefault();
    this.loop.glLost = true;
    this.addStatus({
      key: this.statusKey,
      variant: "error",
      message: "WebGL context lost",
      description: "Plots draw again once the browser restores the context.",
    });
  }

  private restore(): void {
    this.restore2D();
    this._generation++;
    this.configureGL();
    if (this.frames != null) this.frames.backBuffer = this.createBackBuffer();
    this.markDirty(["gl"]);
    if (this.os === "Windows") this.clearProgram = new clear.Program(this);
    this.restoreHandlers.forEach((f) => f());
    this.loop.glLost = false;
    this.addStatus({
      key: this.statusKey,
      variant: "success",
      message: "WebGL context restored",
    });
  }

  // WebKit fires no event for 2D canvases when its GPU process dies. Their pixels and
  // draw state are gone while getTransform still reports the old values. Only a new
  // bitmap repairs them, and WebKit keeps the bitmap unless the size really changes.
  private restore2D(): void {
    this.atlases.clear();
    this.upper2dCanvas.width = 0;
    this.lower2dCanvas.width = 0;
    if (this.frames != null) this.frames.copy.canvas.width = 0;
    this.reset2D();
    this.markDirty(CANVAS_2D_VARIANTS);
    this.restore2DHandlers.forEach((f) => f());
  }

  private reset2D(): void {
    this.resizeCanvas(this.upper2dCanvas);
    this.resizeCanvas(this.lower2dCanvas);
    [this.lower2d, this.upper2d].forEach((canvas) => {
      // A size change clears a canvas's state, but not the state its wrapper caches.
      canvas.reset();
      canvas.scale(this.dpr, this.dpr);
    });
  }

  /**
   * Resizes the canvas to the given region and device pixel ratio. Ensuring
   * that all drawing operations and viewports are scaled correctly.
   */
  resize(region: box.Box, dpr: number): void {
    if (box.equals(this.region, region) && this.dpr === dpr) return;
    this.region = region;
    this.dpr = dpr;
    this.resizeCanvas(this.glCanvas);
    this.reset2D();
    this.gl.viewport(0, 0, box.width(region) * dpr, box.height(region) * dpr);
    if (this.frames != null) this.sizeBackBuffer(this.frames.backBuffer);
    this.markDirty(CANVAS_VARIANTS);
  }

  private markDirty(canvases: CanvasVariant[]): void {
    canvases.forEach((c) => this.frames?.dirty.add(c));
  }

  private createBackBuffer(): BackBuffer {
    const { gl } = this;
    const backBuffer = {
      framebuffer: gl.createFramebuffer(),
      renderbuffer: gl.createRenderbuffer(),
    };
    gl.bindFramebuffer(gl.FRAMEBUFFER, backBuffer.framebuffer);
    this.sizeBackBuffer(backBuffer);
    gl.framebufferRenderbuffer(
      gl.FRAMEBUFFER,
      gl.COLOR_ATTACHMENT0,
      gl.RENDERBUFFER,
      backBuffer.renderbuffer,
    );
    return backBuffer;
  }

  /** Sizes the back buffer to the GL canvas, which clears it. */
  private sizeBackBuffer({ renderbuffer }: BackBuffer): void {
    const { gl, glCanvas } = this;
    gl.bindRenderbuffer(gl.RENDERBUFFER, renderbuffer);
    gl.renderbufferStorageMultisample(
      gl.RENDERBUFFER,
      Math.min(MAX_MSAA_SAMPLES, gl.getParameter(gl.MAX_SAMPLES)),
      gl.RGBA8,
      Math.max(glCanvas.width, 1),
      Math.max(glCanvas.height, 1),
    );
  }

  /** Posts a bitmap of each changed canvas. The page keeps showing the others. */
  private present(drawn: Set<CanvasVariant>): void {
    const { gl, glCanvas, frames } = this;
    if (frames == null) return;
    const { dirty, copy, backBuffer } = frames;
    drawn.forEach((c) => dirty.add(c));
    const { width, height } = glCanvas;
    if (frames.inFlight || dirty.size === 0 || width === 0 || height === 0) return;
    const frame: Frame = {};
    if (dirty.has("gl") && !this.loop.glLost) {
      gl.bindFramebuffer(gl.READ_FRAMEBUFFER, backBuffer.framebuffer);
      gl.bindFramebuffer(gl.DRAW_FRAMEBUFFER, null);
      gl.blitFramebuffer(
        0,
        0,
        width,
        height,
        0,
        0,
        width,
        height,
        gl.COLOR_BUFFER_BIT,
        gl.NEAREST,
      );
      gl.bindFramebuffer(gl.FRAMEBUFFER, backBuffer.framebuffer);
      frame.gl = glCanvas.transferToImageBitmap();
      dirty.delete("gl");
    }
    if (copy.canvas.width !== width || copy.canvas.height !== height) {
      copy.canvas.width = width;
      copy.canvas.height = height;
    }
    copy.globalCompositeOperation = "copy";
    CANVAS_2D_VARIANTS.forEach((c) => {
      if (!dirty.has(c)) return;
      copy.drawImage(c === "lower2d" ? this.lower2dCanvas : this.upper2dCanvas, 0, 0);
      frame[c] = copy.canvas.transferToImageBitmap();
      dirty.delete(c);
    });
    const bitmaps = Object.values(frame);
    if (bitmaps.length === 0) return;
    frames.inFlight = true;
    frames.port.postMessage(frame, bitmaps);
  }

  /** Closes the frame port, after which the context posts no more frames. */
  close(): void {
    this.frames?.port.close();
    this.frames = undefined;
  }

  private resizeCanvas(canvas: OffscreenCanvas): void {
    canvas.width = box.width(this.region) * this.dpr;
    canvas.height = box.height(this.region) * this.dpr;
  }

  /** @returns the aspect ratio of the canvas. */
  get aspect(): number {
    return box.aspect(this.region);
  }

  /**
   * Takes the given box in PIXEL space and produces a transform
   * in CLIP space representing the sub-region represented by the box
   * in the canvas.
   */
  scaleRegion(b: box.Box): scale.XY {
    return new scale.XY(
      scale.Scale.scale<number>(0, 1)
        .scale(box.width(b))
        .translate(box.left(b))
        .reBound(box.width(this.region))
        .scale(-1, 1),
      scale.Scale.scale<number>(0, 1)
        .scale(box.height(b))
        // Invert the scale since we read pixels from the top.
        .invert()
        .translate(box.top(b))
        .reBound(box.height(this.region))
        .scale(-1, 1)
        // Invert the scale since we read clip space from the bottom.
        .invert(),
    );
  }

  scissor(
    region: box.Box,
    overScan: xy.XY = xy.ZERO,
    canvases: CanvasVariant[],
  ): destructor.Destructor {
    const destructor: destructor.Destructor[] = [];
    if (canvases.includes("upper2d"))
      destructor.push(this.upper2d.scissor(region, overScan));
    if (canvases.includes("lower2d"))
      destructor.push(this.lower2d.scissor(region, overScan));
    if (canvases.includes("gl")) destructor.push(this.scissorGL(region, overScan));
    return () => destructor.forEach((d) => d());
  }

  private scissorGL(region: box.Box, overscan: xy.XY = xy.ZERO): destructor.Destructor {
    this.gl.enable(this.gl.SCISSOR_TEST);
    region = applyOverScan(region, overscan);
    this.gl.scissor(
      box.left(region) * this.dpr,
      (box.height(this.region) - box.bottom(region)) * this.dpr,
      box.width(region) * this.dpr,
      box.height(region) * this.dpr,
    );
    return () => this.gl.disable(this.gl.SCISSOR_TEST);
  }

  erase(
    region: box.Box,
    overscan: xy.Crude = xy.ZERO,
    ...canvases: CanvasVariant[]
  ): void {
    overscan = xy.construct(overscan);
    if (canvases.length === 0) canvases = ["upper2d", "lower2d", "gl"];
    if (canvases.includes("upper2d")) this.eraseCanvas(this.upper2d, region, overscan);
    if (canvases.includes("lower2d")) this.eraseCanvas(this.lower2d, region, overscan);
    if (canvases.includes("gl")) this.eraseGL(region, overscan);
    this.markDirty(canvases);
  }

  private eraseGL(box: box.Box, overscan: xy.XY = xy.ZERO): void {
    const { gl } = this;
    const removeScissor = this.scissorGL(applyOverScan(box, overscan));
    gl.clearColor(...color.rgba1(color.TRANSPARENT));
    gl.clear(gl.COLOR_BUFFER_BIT);
    // See the documentation for the clear program for why this is necessary.
    if (this.os === "Windows") this.clearProgram?.exec();
    removeScissor();
  }

  private eraseCanvas(
    c: OffscreenCanvasRenderingContext2D,
    b: box.Box,
    overscan: xy.XY = xy.ZERO,
  ): void {
    const os = applyOverScan(b, overscan);
    c.clearRect(...xy.couple(box.topLeft(os)), ...dimensions.couple(box.dims(os)));
  }
}
