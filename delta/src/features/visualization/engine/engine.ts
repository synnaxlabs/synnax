// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Renderer } from "../render/renderer";
import { calculateBottomOffset, Box } from "../types/spatial";

export interface RenderRequest {
  box: Box;
  renderer: Renderer;
}

const DPR = window.devicePixelRatio ?? 1;

export class RenderingEngine {
  private readonly canvas: HTMLCanvasElement;
  private readonly gl: WebGLRenderingContext;

  constructor(canvas: HTMLCanvasElement) {
    this.canvas = canvas;
    this.gl = this.canvas.getContext("webgl", {
      preserveDrawingBuffer: true,
    }) as WebGLRenderingContext;
  }

  render(req: RenderRequest): void {
    this.refreshCanvas();
    const scale = this.rootScale(req.box);
    const offset = this.rootOffset(req.box);

    req.lines.forEach((l) => this.renderLine(l, scale, offset));
    this.gl.disable(this.gl.SCISSOR_TEST);
  }

  private relativeOffset(box: CSSBox): XY {
    const rect = this.canvas.getBoundingClientRect();
    const relLeft = box.left - rect.left;
    const bot = calculateBottomOffset(rect, box);
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

  private rootScale(box: CSSBox): XY {
    const rect = this.canvas.getBoundingClientRect();
    const x = box.width / rect.width;
    const y = box.height / rect.height;
    return { x, y };
  }

  private rootOffset(box: CSSBox): XY {
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
