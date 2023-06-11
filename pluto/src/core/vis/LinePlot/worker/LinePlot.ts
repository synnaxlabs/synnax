// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Box, XY } from "@synnaxlabs/x";
import { z } from "zod";

import { AetherFactory, AtherComposite } from "@/core/aether/worker";
import { LineFactory, LineGLProgram } from "@/core/vis/Line/LineGL";
import { XAxis, XAxisProps, XAxisFactory } from "@/core/vis/LinePlot/worker/XAxis";
import { YAxisFactory } from "@/core/vis/LinePlot/worker/YAxis";
import { RenderContext, RenderQueue } from "@/core/vis/render";
import { TelemProvider } from "@/core/vis/telem/TelemService";

export const linePlotState = z.object({
  plot: Box.z,
  container: Box.z,
  viewport: Box.z,
  clearOverscan: z.union([z.number(), XY.z]).optional().default(0),
});

export type LinePlotState = z.input<typeof linePlotState>;
export type ParsedLinePlotState = z.output<typeof linePlotState>;

export class LinePlotFactory implements AetherFactory<LinePlot> {
  ctx: RenderContext;
  lines: LineGLProgram;
  renderQueue: RenderQueue;
  telem: TelemProvider;

  constructor(ctx: RenderContext, renderQueue: RenderQueue, telem: TelemProvider) {
    this.ctx = ctx;
    this.lines = new LineGLProgram(ctx);
    this.renderQueue = renderQueue;
    this.telem = telem;
  }

  create(type: string, key: string, props: LinePlotState): LinePlot {
    return new LinePlot(this.ctx, key, props, this.lines, this.renderQueue, this.telem);
  }
}

export class LinePlot extends AtherComposite<
  XAxis,
  LinePlotState,
  ParsedLinePlotState
> {
  ctx: RenderContext;
  renderQueue: RenderQueue;

  static readonly TYPE: string = "line-plot";

  constructor(
    ctx: RenderContext,
    key: string,
    state: LinePlotState,
    lines: LineGLProgram,
    renderQueue: RenderQueue,
    telem: TelemProvider
  ) {
    const lineFactory = new LineFactory(lines, telem, () => this.requestRender());
    const yAxisFactory = new YAxisFactory(ctx, lineFactory, () => this.requestRender());
    const xAxisFactory = new XAxisFactory(ctx, yAxisFactory, () =>
      this.requestRender()
    );
    super(LinePlot.TYPE, key, xAxisFactory, linePlotState, state);
    this.ctx = ctx;
    this.renderQueue = renderQueue;
    this.setStateHook(() => this.requestRender());
  }

  private get plottingRegion(): Box {
    return new Box(this.state.plot);
  }

  private get region(): Box {
    return new Box(this.state.container);
  }

  private get viewport(): Box {
    return new Box(this.state.viewport);
  }

  private get clearOverScan(): XY {
    return new XY(
      typeof this.state.clearOverscan === "number"
        ? { x: this.state.clearOverscan, y: this.state.clearOverscan }
        : this.state.clearOverscan
    );
  }

  private erase(): void {
    this.ctx.erase(this.region, this.clearOverScan);
  }

  private async render(): Promise<void> {
    this.erase();
    const removeScissor = this.ctx.scissorGL(this.plottingRegion);
    await Promise.all(
      this.children.map(async (xAxis) => {
        const ctx: XAxisProps = {
          plottingRegion: this.plottingRegion,
          viewport: this.viewport,
        };
        await xAxis.render(ctx);
      })
    );
    removeScissor();
  }

  requestRender(): void {
    this.renderQueue.push(this.key, async () => await this.render());
  }
}
