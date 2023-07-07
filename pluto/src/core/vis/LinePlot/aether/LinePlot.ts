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

import { LookupResult } from "../../Line/core";
import { RenderPriority } from "../../render/RenderQueue";

import { AetherComposite } from "@/core/aether/worker";
import { CSS } from "@/core/css";
import { AetherXAxis } from "@/core/vis/LinePlot/aether/XAxis";
import { RenderController, RenderContext } from "@/core/vis/render";

const linePlotState = z.object({
  plot: Box.z,
  container: Box.z,
  viewport: Box.z,
  clearOverscan: z.union([z.number(), XY.z]).optional().default(10),
  error: z.string().optional(),
});

interface Derived {
  ctx: RenderContext;
}

export class AetherLinePlot extends AetherComposite<
  typeof linePlotState,
  Derived,
  AetherXAxis
> {
  static readonly TYPE: string = CSS.B("LinePlot");

  static readonly z = linePlotState;
  schema = AetherLinePlot.z;

  derive(): Derived {
    return { ctx: RenderContext.use(this.ctx) };
  }

  afterUpdate(): void {
    RenderController.control(this.ctx, () => this.requestRender(this.region, "low"));
    this.requestRender(this.prevRegion, "high");
  }

  handleDelete(): void {
    const { ctx } = this.derived;
    ctx.erase(this.region, this.clearOverScan);
  }

  private get plottingRegion(): Box {
    return new Box(this.state.plot);
  }

  private get region(): Box {
    return new Box(this.state.container);
  }

  private get prevRegion(): Box {
    return new Box(this.prevState.container);
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

  async lookupX(x: number): Promise<LookupResult[]> {
    return (
      await Promise.all(
        this.childrenOfType(AetherXAxis.TYPE).flatMap(
          async (xAxis) =>
            await xAxis.lookupX(
              {
                plottingRegion: this.plottingRegion,
                viewport: this.viewport,
              },
              x
            )
        )
      )
    ).flat();
  }

  private async render(box: Box): Promise<void> {
    const { ctx } = this.derived;
    ctx.erase(box, this.clearOverScan);
    const removeGlScissor = ctx.scissorGL(this.plottingRegion);
    const removeCanvasScissor = ctx.scissorCanvas(this.region);
    try {
      await Promise.all(
        this.childrenOfType(AetherXAxis.TYPE).map(
          async (xAxis) =>
            await xAxis.render({
              plottingRegion: this.plottingRegion,
              viewport: this.viewport,
            })
        )
      );
    } catch (e) {
      this.setState((p) => ({ ...p, error: (e as Error).message }));
    } finally {
      removeGlScissor();
      removeCanvasScissor();
    }
  }

  requestRender(erase: Box, priority: RenderPriority): void {
    const { ctx } = this.derived;
    ctx.queue.push(this.key, async () => await this.render(erase), priority);
  }
}
