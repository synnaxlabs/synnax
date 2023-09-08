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

import { aether } from "@/aether/aether";
import { CSS } from "@/css";
import { status } from "@/status/aether";
import { type FindResult } from "@/vis/line/aether/line";
import { calculatePlotBox, gridPositionSpecZ } from "@/vis/lineplot/aether/grid";
import { XAxis } from "@/vis/lineplot/aether/XAxis";
import { YAxis } from "@/vis/lineplot/aether/YAxis";
import { measure } from "@/vis/measure/aether";
import { render } from "@/vis/render";
import { tooltip } from "@/vis/tooltip/aether";

export const linePlotStateZ = z.object({
  container: Box.z,
  viewport: Box.z,
  clearOverscan: z.union([z.number(), XY.z]).optional().default(10),
  error: z.string().optional(),
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
  static readonly TYPE: string = CSS.B("LinePlot");
  readonly eraser: render.Eraser = new render.Eraser();

  schema = linePlotStateZ;

  afterUpdate(): void {
    this.internal.aggregate = status.useAggregate(this.ctx);
    this.internal.render = render.Context.use(this.ctx);
    render.Controller.control(this.ctx, () => this.requestRender("low"));
    this.requestRender("high");
  }

  afterDelete(): void {
    this.internal.render = render.Context.use(this.ctx);
    this.requestRender("high");
  }

  private get clearRegion(): Box {
    if (this.deleted) return this.state.container;
    return this.prevState.container;
  }

  async findByXDecimal(x: number): Promise<FindResult[]> {
    const props = { ...this.state, plot: this.calculatePlot() };
    const p = this.axes.flatMap(async (xAxis) => await xAxis.findByXDecimal(props, x));
    return (await Promise.all(p)).flat();
  }

  async findByXValue(x: number): Promise<FindResult[]> {
    const props = { ...this.state, plot: this.calculatePlot() };
    const p = this.axes.flatMap(async (xAxis) => await xAxis.findByXValue(props, x));
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

  private async renderAxes(plot: Box): Promise<void> {
    const p = { ...this.state, plot };
    await Promise.all(this.axes.map(async (xAxis) => await xAxis.render(p)));
  }

  private async renderTooltips(plot: Box): Promise<void> {
    const tooltipProps = {
      findByXDecimal: this.findByXDecimal.bind(this),
      region: plot,
    };
    await Promise.all(
      this.tooltips.map(async (tooltip) => await tooltip.render(tooltipProps)),
    );
  }

  private async renderMeasures(region: Box): Promise<void> {
    const measureProps = {
      findByXDecimal: this.findByXDecimal.bind(this),
      findByXValue: this.findByXValue.bind(this),
      region,
    };
    await Promise.all(
      this.measures.map(async (measure) => await measure.render(measureProps)),
    );
  }

  private calculatePlot(): Box {
    return calculatePlotBox(this.state.grid, this.state.container);
  }

  private async render(): Promise<render.Cleanup> {
    if (this.deleted) return async () => {};
    const plot = this.calculatePlot();
    const { render: ctx } = this.internal;
    const removeGlScissor = ctx.scissorGL(plot);
    const removeCanvasScissor = ctx.scissorCanvas(
      this.state.container,
      new XY(this.state.clearOverscan).scale(0.5),
    );
    try {
      await this.renderAxes(plot);
      await this.renderTooltips(plot);
      await this.renderMeasures(plot);
      this.clearError();
    } catch (e) {
      this.internal.aggregate({
        variant: "error",
        message: (e as Error).message,
      });
    } finally {
      removeGlScissor();
      removeCanvasScissor();
    }
    return async () => {
      this.eraser.erase(
        this.internal.render,
        this.state.container,
        this.prevState.container,
        new XY(this.state.clearOverscan),
      );
    };
  }

  private setError(error: Error): void {
    this.setState((p) => ({ ...p, error: error.message }));
  }

  private clearError(): void {
    if (this.state.error == null) return;
    this.setState((p) => ({ ...p, error: undefined }));
  }

  requestRender(priority: render.Priority): void {
    const { render: ctx } = this.internal;
    ctx.queue.push({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority,
    });
  }
}

export const REGISTRY: aether.ComponentRegistry = {
  [LinePlot.TYPE]: LinePlot,
  [XAxis.TYPE]: XAxis,
  [YAxis.TYPE]: YAxis,
};
