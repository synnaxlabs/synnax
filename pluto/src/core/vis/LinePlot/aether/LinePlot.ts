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
import { LookupResult } from "@/core/vis/Line/core";
import { gridPositionMeta } from "@/core/vis/LinePlot/aether/grid";
import { AetherXAxis } from "@/core/vis/LinePlot/aether/XAxis";
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

interface Derived {
  ctx: RenderContext;
}

export class AetherLinePlot extends AetherComposite<
  typeof linePlotState,
  Derived,
  AetherXAxis | AetherTooltip
> {
  static readonly TYPE: string = CSS.B("LinePlot");

  static readonly z = linePlotState;
  schema = AetherLinePlot.z;

  derive(): Derived {
    return { ctx: RenderContext.use(this.ctx) };
  }

  afterUpdate(): void {
    RenderController.control(this.ctx, () => this.requestRender("low"));
    this.requestRender("high");
  }

  afterDelete(): void {
    this.requestRender("high");
  }

  private get plottingRegion(): Box {
    return new Box(this.state.plot);
  }

  private get clearRegion(): Box {
    if (this.deleted) return this.state.container;
    return this.prevState.container;
  }

  private get region(): Box {
    return this.state.container;
  }

  private get viewport(): Box {
    return this.state.viewport;
  }

  private get clearOverScan(): XY {
    return new XY(
      typeof this.state.clearOverscan === "number"
        ? { x: this.state.clearOverscan, y: this.state.clearOverscan }
        : this.state.clearOverscan
    );
  }

  async lookupX(x: number): Promise<LookupResult[]> {
    return (
      await Promise.all(
        this.childrenOfType<AetherXAxis>(AetherXAxis.TYPE).flatMap(
          async (xAxis) =>
            await xAxis.lookupX(
              {
                plottingRegion: this.plottingRegion,
                viewport: this.viewport,
                region: this.region,
                grid: this.state.grid,
              },
              x
            )
        )
      )
    ).flat();
  }

  private async render(): Promise<RenderCleanup> {
    if (this.deleted) return async () => {};
    const { ctx } = this.derived;
    const removeGlScissor = ctx.scissorGL(this.plottingRegion);
    const removeCanvasScissor = ctx.scissorCanvas(this.region);

    try {
      await Promise.all(
        this.childrenOfType<AetherXAxis>(AetherXAxis.TYPE).map(
          async (xAxis, i) =>
            await xAxis.render({
              plottingRegion: this.plottingRegion,
              viewport: this.viewport,
              region: this.region,
              grid: this.state.grid,
            })
        )
      );

      console.log(
        this.prevState.container,
        this.state.container,
        this.prevState.container.equals(this.state.container)
      );
      if (this.prevState.container.equals(this.state.container))
        await Promise.all(
          this.childrenOfType<AetherTooltip>(AetherTooltip.TYPE).map(
            async (tooltip) =>
              await tooltip.render({
                lookupX: this.lookupX.bind(this),
                region: this.plottingRegion,
              })
          )
        );
    } catch (e) {
      this.setState((p) => ({ ...p, error: (e as Error).message }));
      throw e;
    } finally {
      removeGlScissor();
      removeCanvasScissor();
    }
    return async () => ctx.erase(new Box(this.clearRegion), this.clearOverScan);
  }

  requestRender(priority: RenderPriority): void {
    const { ctx } = this.derived;
    ctx.queue.push({
      key: `${this.type}-${this.key}`,
      render: this.render.bind(this),
      priority,
    });
  }
}
