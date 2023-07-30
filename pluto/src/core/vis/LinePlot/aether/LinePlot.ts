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

import { AetherComposite } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { FindResult } from "@/core/vis/Line/aether";
import { gridPositionMeta } from "@/core/vis/LinePlot/aether/grid";
import { AetherXAxis } from "@/core/vis/LinePlot/aether/XAxis";
import { AetherMeasure } from "@/core/vis/Measure/aether";
import {
  RenderController,
  RenderContext,
  RenderCleanup,
  RenderPriority,
} from "@/core/vis/render";
import { AetherTooltip } from "@/core/vis/Tooltip/aether";

const linePlotState = z.object({
  plot: Box.z,
  container: Box.z,
  viewport: Box.z,
  clearOverscan: z.union([z.number(), XY.z]).optional().default(10),
  error: z.string().optional(),
  grid: z.array(gridPositionMeta),
});

interface InternalState {
  render: RenderContext;
}

type Children = AetherXAxis | AetherTooltip | AetherMeasure;

export class AetherLinePlot extends AetherComposite<
  typeof linePlotState,
  InternalState,
  Children
> {
  static readonly TYPE: string = CSS.B("LinePlot");

  static readonly z = linePlotState;
  schema = AetherLinePlot.z;

  afterUpdate(): void {
    this.internal.render = RenderContext.use(this.ctx);
    RenderController.control(this.ctx, () => this.requestRender("low"));
    this.requestRender("high");
  }

  afterDelete(): void {
    this.internal.render = RenderContext.use(this.ctx);
    this.requestRender("high");
  }

  private get clearRegion(): Box {
    if (this.deleted) return this.state.container;
    return this.prevState.container;
  }

  async findByXDecimal(x: number): Promise<FindResult[]> {
    const p = this.axes.flatMap(
      async (xAxis) => await xAxis.findByXDecimal(this.state, x)
    );
    return (await Promise.all(p)).flat();
  }

  async findByXValue(x: number): Promise<FindResult[]> {
    const p = this.axes.flatMap(
      async (xAxis) => await xAxis.findByXValue(this.state, x)
    );
    return (await Promise.all(p)).flat();
  }

  private get axes(): readonly AetherXAxis[] {
    return this.childrenOfType<AetherXAxis>(AetherXAxis.TYPE);
  }

  private get tooltips(): readonly AetherTooltip[] {
    return this.childrenOfType<AetherTooltip>(AetherTooltip.TYPE);
  }

  private get measures(): readonly AetherMeasure[] {
    return this.childrenOfType<AetherMeasure>(AetherMeasure.TYPE);
  }

  private async renderAxes(): Promise<void> {
    await Promise.all(this.axes.map(async (xAxis) => await xAxis.render(this.state)));
  }

  private async renderTooltips(): Promise<void> {
    const tooltipProps = {
      findByXDecimal: this.findByXDecimal.bind(this),
      region: this.state.plot,
    };
    await Promise.all(
      this.tooltips.map(async (tooltip) => await tooltip.render(tooltipProps))
    );
  }

  private async renderMeasures(): Promise<void> {
    const measureProps = {
      findByXDecimal: this.findByXDecimal.bind(this),
      findByXValue: this.findByXValue.bind(this),
      region: this.state.plot,
    };
    await Promise.all(
      this.measures.map(async (measure) => await measure.render(measureProps))
    );
  }

  private async render(): Promise<RenderCleanup> {
    if (this.deleted) return async () => {};
    const { render: ctx } = this.internal;
    const removeGlScissor = ctx.scissorGL(this.state.plot);
    const removeCanvasScissor = ctx.scissorCanvas(this.state.container);
    try {
      await this.renderAxes();
      await this.renderTooltips();
      await this.renderMeasures();
      this.clearError();
    } catch (e) {
      this.setError(e as Error);
    } finally {
      removeGlScissor();
      removeCanvasScissor();
    }
    return async () =>
      ctx.erase(new Box(this.clearRegion), new XY(this.state.clearOverscan));
  }

  private setError(error: Error): void {
    this.setState((p) => ({ ...p, error: error.message }));
  }

  private clearError(): void {
    if (this.state.error == null) return;
    this.setState((p) => ({ ...p, error: undefined }));
  }

  requestRender(priority: RenderPriority): void {
    const { render: ctx } = this.internal;
    ctx.queue.push({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority,
    });
  }
}
