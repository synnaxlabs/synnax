// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, calculateBottomOffset, CSSBox, XY, ZERO_XY } from "@synnaxlabs/pluto";

import { TelemetryClient } from "../telem/client";

import { Renderer, RendererRegistry, RenderingContext, RenderingUnits } from "./render";
import { ScissoredRenderer } from "./scissor";

export interface RenderRequest {
  box: Box;
  renderer: string;
  request: any;
}

const DPR = window.devicePixelRatio ?? 1;

export class CanvasRenderingContext implements RenderingContext {
  readonly gl: WebGLRenderingContext;
  readonly client: TelemetryClient;
  readonly registry: RendererRegistry;
  private readonly canvas: HTMLCanvasElement;

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
    this.registry.compile(this.gl);
  }

  get aspect(): number {
    const b = this.canvasBox;
    return b.width / b.height;
  }

  get dpr(): number {
    return DPR;
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
    return { x: dims.x / this.canvas.width, y: dims.y / this.canvas.height };
  }

  private get canvasBox(): Box {
    return new CSSBox(this.canvas.getBoundingClientRect());
  }

  refreshCanvas(): void {
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
