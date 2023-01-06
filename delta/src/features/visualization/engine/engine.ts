// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RendererRegistry } from "../render/registry";
import { RenderingContext } from "../render/renderer";
import { ScissoredRenderer } from "../render/scissored/scissor";
import { TelemetryClient } from "../telem/client";
import { calculateBottomOffset, Box, XY, CSSBox } from "../types/spatial";

export interface RenderRequest {
  box: Box;
  renderer: string;
  request: any;
}

const DPR = window.devicePixelRatio ?? 1;

export class RenderingEngine {
  private readonly canvas: HTMLCanvasElement;
  private readonly gl: WebGLRenderingContext;
  private readonly registry: RendererRegistry;
  private readonly client: TelemetryClient;

  constructor(
    canvas: HTMLCanvasElement,
    gl: WebGLRenderingContext,
    registry: RendererRegistry,
    client: TelemetryClient
  ) {
    this.canvas = canvas;
    this.gl = gl;
    this.registry = registry;
    this.client = client;
    this.registry.compile(gl);
  }

  async render(req: RenderRequest): Promise<void> {
    this.refreshCanvas();

    const [offset, clipOffset] = this.rootOffset(req.box);

    const ctx: RenderingContext = {
      gl: this.gl,
      rootScaleClip: this.rootScale(req.box),
      rootOffsetClip: clipOffset,
      rootOffsetPx: offset,
      client: this.client,
      dpr: DPR,
      aspect: this.canvas.width / this.canvas.height,
    };

    const scissored = new ScissoredRenderer(
      this.registry.get(req.renderer),
      req.box,
      true,
      {
        x: 24,
        y: 48,
      }
    );

    await scissored.render(ctx, req.request);
  }

  private relativeOffset(box: Box): XY {
    const canvas = this.canvasBox();
    const relLeft = box.left - canvas.left;
    const bot = calculateBottomOffset(canvas, box);
    return { y: bot, x: relLeft };
  }

  private rootScale(box: Box): XY {
    const rect = this.canvasBox();
    const x = box.width / rect.width;
    const y = box.height / rect.height;
    return { x, y };
  }

  private rootOffset(box: Box): [XY, XY] {
    const { width, height } = this.canvasBox();
    const { x, y } = this.relativeOffset(box);
    return [
      { x, y },
      { x: x / width, y: y / height },
    ];
  }

  private canvasBox(): Box {
    return CSSBox.fromDomRect(this.canvas.getBoundingClientRect());
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
