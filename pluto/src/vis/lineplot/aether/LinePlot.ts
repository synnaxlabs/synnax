// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { status } from "@/status/aether";
import { type FindResult } from "@/vis/line/aether/line";
import { calculatePlotBox, gridPositionSpecZ } from "@/vis/lineplot/aether/grid";
import { XAxis } from "@/vis/lineplot/aether/XAxis";
import { YAxis } from "@/vis/lineplot/aether/YAxis";
import { measure } from "@/vis/measure/aether";
import { render } from "@/vis/render";
import { tooltip } from "@/vis/tooltip/aether";

export const linePlotStateZ = z.object({
  container: box.box,
  viewport: box.box,
  clearOverscan: z.union([z.number(), xy.xy]).optional().default(10),
  hold: z.boolean().optional().default(false),
  grid: z.array(gridPositionSpecZ),
});

interface InternalState {
  aggregate: status.Aggregate;
  render: render.Context;
}

type Children = XAxis | tooltip.Tooltip | measure.Measure;

export class LinePlot extends aether.Composite<
  typeof linePlotStateZ,
  InternalState,
  Children
> {
  static readonly TYPE: string = "LinePlot";
  private readonly eraser: render.Eraser = new render.Eraser();

  schema = linePlotStateZ;

  afterUpdate(): void {
    this.internal.aggregate = status.useAggregate(this.ctx);
    this.internal.render = render.Context.use(this.ctx);
    render.Controller.control(this.ctx, (r) => this.requestRender("low", r));
    this.requestRender("high", render.REASON_LAYOUT);
  }

  afterDelete(): void {
    this.internal.render = render.Context.use(this.ctx);
    this.requestRender("high", render.REASON_LAYOUT);
  }

  async findByXDecimal(x: number): Promise<FindResult[]> {
    const props = { ...this.state, plot: this.calculatePlot() };
    const p = this.axes.flatMap(async (xAxis) => await xAxis.findByXDecimal(props, x));
    return (await Promise.all(p)).flat();
  }

  async findByXValue(x: number): Promise<FindResult[]> {
    const props = { ...this.state, plot: this.calculatePlot() };
    const p = this.axes.flatMap(async (a) => await a.findByXValue(props, x));
    return (await Promise.all(p)).flat();
  }

  private get axes(): readonly XAxis[] {
    return this.childrenOfType<XAxis>(XAxis.TYPE);
  }

  private get tooltips(): readonly tooltip.Tooltip[] {
    return this.childrenOfType<tooltip.Tooltip>(tooltip.Tooltip.TYPE);
  }

  private get measures(): readonly measure.Measure[] {
    return this.childrenOfType<measure.Measure>(measure.Measure.TYPE);
  }

  private async renderAxes(
    plot: box.Box,
    canvases: render.CanvasVariant[],
  ): Promise<void> {
    const p = { ...this.state, plot, canvases };
    await Promise.all(this.axes.map(async (xAxis) => await xAxis.render(p)));
  }

  private async renderTooltips(
    region: box.Box,
    canvases: render.CanvasVariant[],
  ): Promise<void> {
    const p = { findByXDecimal: this.findByXDecimal.bind(this), region, canvases };
    await Promise.all(this.tooltips.map(async (t) => await t.render(p)));
  }

  private async renderMeasures(
    region: box.Box,
    canvases: render.CanvasVariant[],
  ): Promise<void> {
    const p = {
      findByXDecimal: this.findByXDecimal.bind(this),
      findByXValue: this.findByXValue.bind(this),
      region,
      canvases,
    };
    await Promise.all(this.measures.map(async (m) => await m.render(p)));
  }

  private calculatePlot(): box.Box {
    return calculatePlotBox(this.state.grid, this.state.container);
  }

  private async render(canvases: render.CanvasVariant[]): Promise<render.Cleanup> {
    if (this.deleted) return async () => {};
    const plot = this.calculatePlot();
    const { render: ctx } = this.internal;
    const os = xy.construct(this.state.clearOverscan);
    const removeCanvasScissor = ctx.scissor(
      this.state.container,
      os,
      canvases.filter((c) => c !== "gl"),
    );
    const removeGLScissor = ctx.scissor(
      plot,
      xy.ZERO,
      canvases.filter((c) => c === "gl"),
    );
    try {
      await this.renderAxes(plot, canvases);
      await this.renderTooltips(plot, canvases);
      await this.renderMeasures(plot, canvases);
    } catch (e) {
      this.internal.aggregate({ variant: "error", message: (e as Error).message });
    } finally {
      removeCanvasScissor();
      removeGLScissor();
    }
    return async ({ canvases }) => {
      this.eraser.erase(
        ctx,
        this.state.container,
        this.prevState.container,
        xy.construct(this.state.clearOverscan),
        canvases,
      );
    };
  }

  requestRender(priority: render.Priority, reason: string): void {
    const { render: ctx } = this.internal;
    let canvases: render.CanvasVariant[] = ["upper2d", "lower2d", "gl"];
    // Optimization for tooltips, measures and other utilities. In this case, we only
    // need to render the upper2d canvas.
    if (reason === render.REASON_TOOL) canvases = ["upper2d"];
    ctx.queue.push({
      key: `${this.type}-${this.key}`,
      render: async () => await this.render(canvases),
      priority,
      canvases,
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [LinePlot.TYPE]: LinePlot,
  [XAxis.TYPE]: XAxis,
  [YAxis.TYPE]: YAxis,
};
